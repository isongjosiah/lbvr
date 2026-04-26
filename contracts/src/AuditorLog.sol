// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {AccessControlDefaultAdminRules} from
    "@openzeppelin/contracts/access/extensions/AccessControlDefaultAdminRules.sol";

/// @title AuditorLog
/// @notice On-chain anchor registry for LBVR-Med PROV-JSON provenance documents
///         (CLAUDE.md §4.6, docs/provenance-spec.md §6) and gateway BLS public-key
///         directory used by the off-chain provenance verifier
///         (docs/provenance-spec.md §5.5, §7).
///
///         Each retrieval emits a JCS-canonicalized PROV-JSON document; its SHA-256
///         hash is anchored here so that any external auditor can detect tampering
///         (EU AI Act Art. 12 / EHDS Art. 44–50 logging requirements).
///
/// @dev    Anchoring is append-only per (bundleId, retrievalId) pair. Re-anchoring
///         the same pair reverts; this prevents a compromised gateway key from
///         retroactively rewriting its own audit trail.
///
/// @dev    Access model: AccessControlDefaultAdminRules (OZ 5.x) mirrors CIDRegistry.
///         A dedicated ANCHOR_ROLE gates `anchorProvenance`; the deployer holds it
///         initially and grants it to gateway addresses (or the future PoRVerifier).
///         See the constructor for the rationale on deviating from the spec's
///         confused-deputy `onlyAuthorizedGateway` modifier.
contract AuditorLog is AccessControlDefaultAdminRules {
    /// @notice Role authorised to anchor PROV-JSON hashes.
    ///         Held by the deployer initially; granted to gateway nodes (and, in the
    ///         journal extension, to a threshold-multisig contract).
    /// @dev    Deviation from docs/provenance-spec.md §6.1: the spec sketches an
    ///         `onlyAuthorizedGateway` modifier that derives the caller's DID from
    ///         `msg.sender` via `keccak256(abi.encodePacked("did:lbvr:", toHex(msg.sender)))`
    ///         and looks the result up in `gatewayKeys`. That couples the DID
    ///         namespace to the eth address format and is a confused-deputy
    ///         invitation: any contract that can be made to call `anchorProvenance`
    ///         on behalf of a registered gateway address would pass the check
    ///         regardless of whether the gateway authorised the anchor. Per the D10
    ///         brief we substitute the OZ AccessControl ANCHOR_ROLE pattern, which
    ///         (a) decouples the DID/key registry from the authorisation surface,
    ///         (b) is consistent with `CIDRegistry.MIGRATOR_ROLE`, and (c) lets us
    ///         grant the role to the future PoRVerifier contract without inventing
    ///         a synthetic DID for it.
    bytes32 public constant ANCHOR_ROLE = keccak256("ANCHOR_ROLE");

    /// @dev One on-chain SSTORE per retrieval (~50k gas target per
    ///      docs/provenance-spec.md §6.2). The full PROV-JSON lives off-chain on
    ///      Pinata; only its canonical-JCS SHA-256 root is anchored here.
    struct ProvenanceAnchor {
        bytes32 provHash;
        uint256 blockNumber;
        uint256 timestamp;
        address anchoredBy;
    }

    /// @dev bundleId -> retrievalId -> anchor. retrievalId is expected to be a
    ///      keccak256 of the off-chain retrieval UUID; the contract treats both
    ///      keys as opaque.
    mapping(bytes32 => mapping(bytes32 => ProvenanceAnchor)) private _anchors;

    /// @dev didHash = keccak256(bytes(did)). Public key is BLS12-381 G1
    ///      compressed (48 bytes) per docs/provenance-spec.md §5.1; stored as
    ///      `bytes` because 48 bytes does not fit in `bytes32` and the journal
    ///      extension may rotate to 96-byte G2 public keys without a schema change.
    mapping(bytes32 => bytes) private _gatewayKeys;

    event ProvenanceAnchored(
        bytes32 indexed bundleId, bytes32 indexed retrievalId, bytes32 provHash, address indexed anchoredBy
    );

    /// @dev `publicKey` is non-indexed because indexed `bytes` is hashed by the EVM
    ///      and clients would lose the actual key value in event logs. didHash is
    ///      indexed so verifiers can filter by gateway DID.
    event GatewayKeyRegistered(bytes32 indexed didHash, bytes publicKey);

    error AlreadyAnchored(bytes32 bundleId, bytes32 retrievalId);
    error AnchorNotFound(bytes32 bundleId, bytes32 retrievalId);
    error EmptyProvHash();
    error EmptyPublicKey();
    error EmptyDID();

    /// @param admin               Initial default admin (held by deployer).
    /// @param adminTransferDelay  Delay (seconds) enforced on default-admin handover.
    constructor(address admin, uint48 adminTransferDelay) AccessControlDefaultAdminRules(adminTransferDelay, admin) {
        _grantRole(ANCHOR_ROLE, admin);
    }

    /// @notice Anchors the canonical SHA-256 hash of a PROV-JSON document.
    /// @dev Append-only per (bundleId, retrievalId); re-anchoring reverts. Rejects
    ///      `provHash == bytes32(0)` because zero is the sentinel value the storage
    ///      layout uses to mean "not anchored".
    function anchorProvenance(bytes32 bundleId, bytes32 retrievalId, bytes32 provHash)
        external
        onlyRole(ANCHOR_ROLE)
    {
        if (provHash == bytes32(0)) revert EmptyProvHash();
        if (_anchors[bundleId][retrievalId].provHash != bytes32(0)) {
            revert AlreadyAnchored(bundleId, retrievalId);
        }

        _anchors[bundleId][retrievalId] = ProvenanceAnchor({
            provHash: provHash,
            blockNumber: block.number,
            timestamp: block.timestamp,
            anchoredBy: msg.sender
        });

        emit ProvenanceAnchored(bundleId, retrievalId, provHash, msg.sender);
    }

    /// @notice Returns the anchor for a given (bundleId, retrievalId).
    /// @dev Reverts with {AnchorNotFound} when no anchor exists, mirroring
    ///      `CIDRegistry.BundleNotFound` so callers see consistent error semantics.
    function getProvenanceAnchor(bytes32 bundleId, bytes32 retrievalId) external view returns (ProvenanceAnchor memory) {
        ProvenanceAnchor storage a = _anchors[bundleId][retrievalId];
        if (a.provHash == bytes32(0)) revert AnchorNotFound(bundleId, retrievalId);
        return a;
    }

    /// @notice Registers (or rotates) a gateway's BLS public key.
    /// @dev Re-registration is permitted and overwrites the prior key to support
    ///      planned rotation. Historical PROV signatures continue to verify against
    ///      the prior key, so off-chain verifiers MUST resolve the key as-of the
    ///      anchor's `blockNumber` by replaying {GatewayKeyRegistered} events
    ///      (TODO(journal): expose an explicit historical lookup API once batched
    ///      anchoring lands per docs/provenance-spec.md §6.2).
    function registerGatewayKey(string calldata did, bytes calldata publicKey) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (bytes(did).length == 0) revert EmptyDID();
        if (publicKey.length == 0) revert EmptyPublicKey();

        bytes32 didHash = keccak256(bytes(did));
        _gatewayKeys[didHash] = publicKey;
        emit GatewayKeyRegistered(didHash, publicKey);
    }

    /// @notice Returns the currently-registered BLS public key for a DID.
    /// @dev Returns an empty `bytes` if the DID has never been registered. Callers
    ///      that require historical lookups (verifying signatures against keys
    ///      valid at an anchor's blockNumber) must replay {GatewayKeyRegistered}
    ///      events; see the rotation note on `registerGatewayKey`.
    function getGatewayKey(string calldata did) external view returns (bytes memory) {
        return _gatewayKeys[keccak256(bytes(did))];
    }
}
