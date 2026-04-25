// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {AccessControlDefaultAdminRules} from
    "@openzeppelin/contracts/access/extensions/AccessControlDefaultAdminRules.sol";

/// @title CIDRegistry
/// @notice On-chain registry of LBVR-Med bundle records. Each record binds a bundleId
///         to its Merkle root, erasure-coded shard layout (RS(2,3) -> 3 shards, one per
///         tier), owner, and policy reference. Acts as the authoritative source for
///         `shardLayout` and `tier_history` reads during retrieval (CLAUDE.md §4.3)
///         and is updated by the PoR subsystem on auto-migration (CLAUDE.md §4.4).
///
/// @dev    bundleId is expected to be keccak256(abi.encodePacked(clientId, merkleRoot))
///         computed off-chain. The contract treats it as an opaque unique key and does
///         not verify the derivation — uniqueness is enforced by rejecting re-registration.
///
/// @dev    Access model: AccessControlDefaultAdminRules (OZ 5.x idiom) is used instead of
///         pairing Ownable + AccessControl. It gives us a two-step admin handover with
///         enforced delay (safer for a testnet-rotatable deployer key) and unifies the
///         role surface so the future PoRVerifier contract (TODO(D12)) only needs a role
///         grant rather than an ownership transfer.
contract CIDRegistry is AccessControlDefaultAdminRules {
    /// @notice Role authorised to rewrite shard layouts on PoR-triggered migration.
    ///         Held by the deployer initially; transferred to PoRVerifier in D12.
    bytes32 public constant MIGRATOR_ROLE = keccak256("MIGRATOR_ROLE");

    /// @dev RS(2,3) invariant: 2 data shards + 1 parity shard, one per tier (hot/warm/cold).
    ///      Conference scope — may relax in the journal extension when we evaluate RS(3,5).
    uint256 private constant _SHARD_COUNT = 3;

    enum TierClass {
        Hot, // 0 — Pinata
        Warm, // 1 — Filebase
        Cold // 2 — Arweave / Irys
    }

    /// @dev `cid` is stored as `bytes` (multibase CIDv0/v1, up to ~100 chars) rather
    ///      than bytes32. Storing the raw CID costs roughly 2x the gas of a 32-byte
    ///      hash, but the pinning-service APIs (Pinata, Filebase S3, Irys) need the
    ///      multibase form verbatim to round-trip a GET — truncating to a hash would
    ///      force an off-chain CID table and defeat the point of an on-chain layout.
    struct ShardPlacement {
        bytes cid;
        TierClass tier;
    }

    struct BundleRecord {
        bytes32 merkleRoot;
        /// @dev Bound on-chain so verifiers of off-chain Merkle proofs know the leaf
        ///      count. Without it, Bitcoin-style odd-width duplication in the Merkle
        ///      tree (used by `internal/merkle`) would admit a CVE-2012-2459-class
        ///      second-preimage attack where a proof valid against width=n is also
        ///      valid against a crafted width=n+1 tree. uint32 is room for ~4.3B
        ///      chunks × 16 KiB = 64 TiB per bundle; measured P99 is ~2.4K chunks.
        uint32 numChunks;
        ShardPlacement[] shards;
        address owner;
        bytes32 policyId;
        uint64 registeredAt;
        uint64 lastMigratedAt;
    }

    mapping(bytes32 => BundleRecord) private _bundles;

    event BundleRegistered(
        bytes32 indexed bundleId, bytes32 indexed merkleRoot, address indexed owner, bytes32 policyId
    );

    event ShardLayoutUpdated(bytes32 indexed bundleId, ShardPlacement[] oldShards, ShardPlacement[] newShards);

    error BundleAlreadyRegistered(bytes32 bundleId);
    error BundleNotFound(bytes32 bundleId);
    error InvalidShardCount(uint256 got, uint256 expected);
    error EmptyShardCID(uint256 index);
    error NumChunksZero();

    /// @param admin               Initial default admin (held by deployer).
    /// @param adminTransferDelay  Delay (seconds) enforced on default-admin handover.
    constructor(address admin, uint48 adminTransferDelay) AccessControlDefaultAdminRules(adminTransferDelay, admin) {
        _grantRole(MIGRATOR_ROLE, admin);
    }

    /// @notice Registers a new bundle with its Merkle root, shard layout, and policy id.
    /// @dev Caller becomes the recorded owner. Reverts if bundleId already exists,
    ///      if shard count is not 3, or if any shard CID is empty.
    function registerBundle(
        bytes32 bundleId,
        bytes32 merkleRoot,
        uint32 numChunks,
        ShardPlacement[] calldata shards,
        bytes32 policyId
    ) external {
        if (_bundles[bundleId].owner != address(0)) revert BundleAlreadyRegistered(bundleId);
        if (numChunks == 0) revert NumChunksZero();
        if (shards.length != _SHARD_COUNT) revert InvalidShardCount(shards.length, _SHARD_COUNT);

        BundleRecord storage rec = _bundles[bundleId];
        rec.merkleRoot = merkleRoot;
        rec.numChunks = numChunks;
        rec.owner = msg.sender;
        rec.policyId = policyId;
        rec.registeredAt = uint64(block.timestamp);
        rec.lastMigratedAt = 0;

        for (uint256 i = 0; i < shards.length; ++i) {
            if (shards[i].cid.length == 0) revert EmptyShardCID(i);
            rec.shards.push(shards[i]);
        }

        emit BundleRegistered(bundleId, merkleRoot, msg.sender, policyId);
    }

    /// @notice Returns the current shard layout for a bundle.
    /// @dev Reverts if the bundle is not registered.
    function getShardLayout(bytes32 bundleId) external view returns (ShardPlacement[] memory) {
        BundleRecord storage rec = _bundles[bundleId];
        if (rec.owner == address(0)) revert BundleNotFound(bundleId);
        return rec.shards;
    }

    /// @notice Returns the full bundle record.
    /// @dev Reverts if the bundle is not registered.
    function getRecord(bytes32 bundleId) external view returns (BundleRecord memory) {
        BundleRecord storage rec = _bundles[bundleId];
        if (rec.owner == address(0)) revert BundleNotFound(bundleId);
        return rec;
    }

    /// @notice Rewrites the shard layout after a PoR-driven tier migration.
    /// @dev Only callable by MIGRATOR_ROLE (TODO(D12): grant to PoRVerifier, revoke from
    ///      deployer). Reverts if the bundle does not exist, if the new shard count is
    ///      not 3, or if any new shard CID is empty. Emits {ShardLayoutUpdated} with the
    ///      prior layout for audit trails.
    function updateShardLayout(bytes32 bundleId, ShardPlacement[] calldata newShards)
        external
        onlyRole(MIGRATOR_ROLE)
    {
        BundleRecord storage rec = _bundles[bundleId];
        if (rec.owner == address(0)) revert BundleNotFound(bundleId);
        if (newShards.length != _SHARD_COUNT) revert InvalidShardCount(newShards.length, _SHARD_COUNT);
        for (uint256 i = 0; i < newShards.length; ++i) {
            if (newShards[i].cid.length == 0) revert EmptyShardCID(i);
        }

        // Snapshot old layout into memory for the event before we overwrite storage.
        ShardPlacement[] memory oldShards = rec.shards;

        delete rec.shards;
        for (uint256 i = 0; i < newShards.length; ++i) {
            rec.shards.push(newShards[i]);
        }
        rec.lastMigratedAt = uint64(block.timestamp);

        emit ShardLayoutUpdated(bundleId, oldShards, newShards);
    }
}
