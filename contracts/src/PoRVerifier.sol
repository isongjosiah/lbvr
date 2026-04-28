// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {AccessControlDefaultAdminRules} from
    "@openzeppelin/contracts/access/extensions/AccessControlDefaultAdminRules.sol";

/// @title ICIDRegistry
/// @notice Minimal local interface to read the Merkle parameters of a bundle from
///         the deployed CIDRegistry contract.
/// @dev    The mirror struct below MUST keep the same field layout (in the same
///         order, with the same types) as `CIDRegistry.BundleRecord`. Solidity's
///         external ABI matches by tuple shape, not by type name, so this lets
///         PoRVerifier read the registry without importing the full CIDRegistry
///         source — we only need `merkleRoot` and `numChunks`. If
///         `CIDRegistry.BundleRecord` is ever extended or reordered, this mirror
///         (and its consumers below) MUST be updated in lockstep.
interface ICIDRegistry {
    enum TierClass {
        Hot,
        Warm,
        Cold
    }

    struct ShardPlacement {
        bytes cid;
        TierClass tier;
    }

    struct BundleRecord {
        bytes32 merkleRoot;
        uint32 numChunks;
        ShardPlacement[] shards;
        address owner;
        bytes32 policyId;
        uint64 registeredAt;
        uint64 lastMigratedAt;
    }

    function getRecord(bytes32 bundleId) external view returns (BundleRecord memory);
}

/// @title PoRVerifier
/// @notice On-chain Proof-of-Retrievability protocol for LBVR-Med per CLAUDE.md
///         §4.4. Off-chain auditors post `(bundleId, shardIdx, chunkIdx, nonce)`
///         challenges; storage replicas (or their delegates) respond with the
///         requested chunk hash, a Merkle proof against the bundle's on-chain
///         `merkleRoot`, and a BLS signature. Verdicts (success/failure) are
///         recorded by the auditor after off-chain BLS verification.
///
///         After {MAX_CONSECUTIVE_FAILURES} consecutive `bls_invalid` verdicts on
///         the same `(bundleId, shardIdx)`, the contract emits
///         {ShardMigrationRequired}; the off-chain auditor (which holds
///         `MIGRATOR_ROLE` on `CIDRegistry`) reads the event and rewrites the
///         shard layout. Wiring the migration call directly into this contract
///         requires holding `MIGRATOR_ROLE` on the registry and is journal-scope
///         (CLAUDE.md §4.4 last paragraph).
///
/// @dev    BLS verification is intentionally OFF-CHAIN. Polygon zkEVM lacks a
///         BLS12-381 pairing precompile (EIP-2537 is not enabled on Cardona as
///         of 2026-04), so an on-chain pairing check would burn ~1.5M+ gas via
///         a verifier library — far above the §3.1 "≤200k gas per submission"
///         budget. The contract therefore stores the submitted signature and
///         emits it for off-chain verification; `recordVerdict` is the
///         on-chain auditable hook the auditor uses to commit the verification
///         outcome. TODO(journal): replace with an on-chain BLS precompile path
///         once EIP-2537 ships on Cardona.
///
/// @dev    Access model mirrors {CIDRegistry} and {AuditorLog}: a single
///         `AUDITOR_ROLE` posts challenges and records verdicts; a single
///         `RESPONDER_ROLE` covers any registered storage replica (per-shard
///         responder authorisation is journal-scope). Admin handover uses
///         OZ 5.x `AccessControlDefaultAdminRules`.
contract PoRVerifier is AccessControlDefaultAdminRules {
    /// @notice Posts challenges and records verdicts.
    bytes32 public constant AUDITOR_ROLE = keccak256("AUDITOR_ROLE");
    /// @notice Storage-replica delegates that submit chunk responses.
    bytes32 public constant RESPONDER_ROLE = keccak256("RESPONDER_ROLE");

    /// @dev After this many consecutive `bls_invalid` verdicts on the same
    ///      `(bundleId, shardIdx)`, the contract emits {ShardMigrationRequired}
    ///      and resets the counter. A successful verdict before the threshold
    ///      also resets the counter (see `recordVerdict`).
    uint32 public constant MAX_CONSECUTIVE_FAILURES = 3;

    /// @dev Mirrors `CIDRegistry._SHARD_COUNT`. RS(2,3) -> 3 shards (hot/warm/cold).
    uint32 private constant _SHARD_COUNT = 3;

    /// @dev On-chain anchor of an off-chain PoR challenge.
    struct Challenge {
        bytes32 bundleId;
        uint32 shardIdx; // 0=hot, 1=warm, 2=cold; mirrors CIDRegistry.TierClass
        uint32 chunkIdx;
        bytes32 nonce;
        uint64 postedAt;
        uint64 deadline;
        address auditor;
    }

    /// @dev `merkleProof` and `blsSig` are stored as `bytes` (not `bytes32[]`
    ///      and not fixed `bytes96`) to keep the storage layout simple and
    ///      future-proof: proof depth varies per bundle (~log2(numChunks)) and
    ///      the BLS signature size could change in a journal upgrade. The
    ///      tradeoff is a slightly higher SSTORE cost vs. typed storage, which
    ///      is acceptable because the response path is exercised at PoR
    ///      cadence (~30 days/bundle), not at retrieval cadence.
    struct Response {
        bytes32 chunkHash;
        bytes merkleProof; // ABI-encoded bytes32[]
        bytes blsSig; // 96-byte BLS-G2 signature (length not enforced on-chain)
        uint64 respondedAt;
        address responder;
    }

    /// @dev Verdict is recorded *after* the auditor verifies BLS off-chain.
    ///      `reason` is a free-form string ("ok" / "bls_invalid" / "timeout"
    ///      etc.); the contract only special-cases `bls_invalid` for the
    ///      consecutive-failure counter.
    struct Verdict {
        bool recorded;
        bool success;
        string reason;
        uint64 verdictAt;
        address auditor;
    }

    mapping(bytes32 => Challenge) private _challenges;
    mapping(bytes32 => Response) private _responses;
    mapping(bytes32 => Verdict) private _verdicts;
    /// @dev bundleId -> shardIdx -> consecutive bls_invalid (or other failure)
    ///      verdicts since the last success/migration.
    mapping(bytes32 => mapping(uint32 => uint32)) private _consecutiveFailures;

    ICIDRegistry public immutable registry;

    event ChallengePosted(
        bytes32 indexed challengeId,
        bytes32 indexed bundleId,
        uint32 indexed shardIdx,
        uint32 chunkIdx,
        bytes32 nonce,
        uint64 deadline
    );

    /// @dev `chunkHash`, `responder`, and the (already-indexed) `challengeId`
    ///      are sufficient for off-chain verifiers to reconstruct the BLS
    ///      message; the full proof + sig live in the {Response} record so
    ///      auditors can fetch them via `getResponse`.
    event ChallengeResponded(bytes32 indexed challengeId, bytes32 chunkHash, address indexed responder);

    event VerdictRecorded(bytes32 indexed challengeId, bool success, string reason);

    event ShardMigrationRequired(
        bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 consecutiveFailures
    );

    error InvalidShardIdx(uint32 got, uint32 max);
    error InvalidResponseWindow();
    error ZeroRegistry();
    error ChallengeNotFound(bytes32 challengeId);
    error ChallengeExpired(bytes32 challengeId, uint64 deadline, uint64 nowTs);
    error ChallengeAlreadyResponded(bytes32 challengeId);
    error MerkleProofInvalid(bytes32 expectedRoot, bytes32 computedRoot);
    error EmptyChunkHash();
    error EmptyBLSSig();
    error VerdictAlreadyRecorded(bytes32 challengeId);
    error ResponseRequired(bytes32 challengeId);

    /// @param admin               Initial default admin (held by deployer).
    /// @param adminTransferDelay  Delay (seconds) enforced on default-admin handover.
    /// @param registryAddress     Deployed CIDRegistry contract; must be non-zero.
    constructor(address admin, uint48 adminTransferDelay, address registryAddress)
        AccessControlDefaultAdminRules(adminTransferDelay, admin)
    {
        if (registryAddress == address(0)) revert ZeroRegistry();
        registry = ICIDRegistry(registryAddress);
        _grantRole(AUDITOR_ROLE, admin);
        _grantRole(RESPONDER_ROLE, admin);
    }

    // --- challenge / response / verdict ---------------------------------

    /// @notice Posts a PoR challenge for `(bundleId, shardIdx, chunkIdx, nonce)`.
    /// @dev    Reverts on `shardIdx >= 3` ({InvalidShardIdx}) or
    ///         `responseWindow == 0` ({InvalidResponseWindow}). The contract
    ///         does NOT verify that `bundleId` exists in the registry — the
    ///         existence check happens on response, where the registry must be
    ///         consulted anyway to fetch the Merkle root. This keeps challenge
    ///         posting cheap (auditors batch challenges) and tolerates registry
    ///         deletions that happen between post and response.
    /// @return challengeId Deterministic id derived from
    ///         `keccak256(abi.encode(bundleId, shardIdx, chunkIdx, nonce, postedAt))`.
    function postChallenge(
        bytes32 bundleId,
        uint32 shardIdx,
        uint32 chunkIdx,
        bytes32 nonce,
        uint64 responseWindow
    ) external onlyRole(AUDITOR_ROLE) returns (bytes32 challengeId) {
        if (shardIdx >= _SHARD_COUNT) revert InvalidShardIdx(shardIdx, _SHARD_COUNT - 1);
        if (responseWindow == 0) revert InvalidResponseWindow();

        uint64 postedAt = uint64(block.timestamp);
        uint64 deadline = postedAt + responseWindow;
        challengeId = keccak256(abi.encode(bundleId, shardIdx, chunkIdx, nonce, postedAt));

        _challenges[challengeId] = Challenge({
            bundleId: bundleId,
            shardIdx: shardIdx,
            chunkIdx: chunkIdx,
            nonce: nonce,
            postedAt: postedAt,
            deadline: deadline,
            auditor: msg.sender
        });

        emit ChallengePosted(challengeId, bundleId, shardIdx, chunkIdx, nonce, deadline);
    }

    /// @notice Submits a chunk hash + Merkle proof + BLS signature for an open
    ///         challenge.
    /// @dev    Reverts on missing/expired challenge, double-response, empty
    ///         chunkHash / blsSig, or Merkle proof that does not reconstruct
    ///         the registry's `merkleRoot` for the bundle. BLS verification is
    ///         off-chain (see the contract-level dev block above); the
    ///         signature is stored and emitted, not validated here.
    /// @param  totalChunks Required because internal/merkle's odd-width
    ///         duplication needs the tree width to recover the per-level
    ///         "duplicated tail" semantics; the responder reads it from the
    ///         registry's `numChunks` and forwards it.
    function respondToChallenge(
        bytes32 challengeId,
        bytes32 chunkHash,
        bytes32[] calldata merkleProof,
        bytes calldata blsSig,
        uint32 totalChunks
    ) external onlyRole(RESPONDER_ROLE) {
        Challenge storage ch = _challenges[challengeId];
        if (ch.bundleId == bytes32(0)) revert ChallengeNotFound(challengeId);
        uint64 nowTs = uint64(block.timestamp);
        if (nowTs > ch.deadline) revert ChallengeExpired(challengeId, ch.deadline, nowTs);
        if (_responses[challengeId].respondedAt != 0) revert ChallengeAlreadyResponded(challengeId);
        if (chunkHash == bytes32(0)) revert EmptyChunkHash();
        if (blsSig.length == 0) revert EmptyBLSSig();

        // Bind the proof to the live on-chain merkleRoot. Cross the registry
        // boundary once — getRecord returns the full struct but we only use
        // (merkleRoot, numChunks). totalChunks is supplied by the responder
        // and cross-checked against the registry to prevent CVE-2012-2459-class
        // forgeries (see internal/merkle package doc).
        ICIDRegistry.BundleRecord memory rec = registry.getRecord(ch.bundleId);
        // If the responder claims a different tree width than the registry, the
        // proof would verify against a forged width; reject explicitly via the
        // shared MerkleProofInvalid path so callers see one revert reason for
        // any proof-binding failure.
        bytes32 expectedRoot = rec.merkleRoot;
        if (totalChunks != rec.numChunks) {
            revert MerkleProofInvalid(expectedRoot, bytes32(0));
        }

        (bool ok, bytes32 computed) = _verifyMerkleProof(chunkHash, ch.chunkIdx, merkleProof, totalChunks, expectedRoot);
        if (!ok) revert MerkleProofInvalid(expectedRoot, computed);

        _responses[challengeId] = Response({
            chunkHash: chunkHash,
            merkleProof: abi.encode(merkleProof),
            blsSig: blsSig,
            respondedAt: nowTs,
            responder: msg.sender
        });

        emit ChallengeResponded(challengeId, chunkHash, msg.sender);
    }

    /// @notice Records the auditor's verdict on a challenge (after off-chain
    ///         BLS verification).
    /// @dev    Allowed inputs:
    ///         - `success=true,  reason="ok"` after a successful BLS verify;
    ///         - `success=false, reason="bls_invalid"` after a failed BLS verify;
    ///         - `success=false, reason="timeout"` when the response window
    ///           lapsed without a {ChallengeResponded} (in which case the
    ///           contract has no on-chain response — the auditor is recording
    ///           the absence of one).
    ///         The contract requires either a prior response OR (if no
    ///         response) that the deadline has passed. This blocks the auditor
    ///         from declaring a "timeout" verdict prematurely.
    ///         Any `success=false` verdict increments the consecutive-failure
    ///         counter for `(bundleId, shardIdx)`; success resets it to 0.
    function recordVerdict(bytes32 challengeId, bool success, string calldata reason)
        external
        onlyRole(AUDITOR_ROLE)
    {
        Challenge storage ch = _challenges[challengeId];
        if (ch.bundleId == bytes32(0)) revert ChallengeNotFound(challengeId);
        if (_verdicts[challengeId].recorded) revert VerdictAlreadyRecorded(challengeId);

        // Either a response landed, or the window has fully elapsed (allowing a
        // timeout verdict). We do not constrain `reason` to a fixed enum here —
        // the off-chain auditor logs are the source of truth for the reason
        // string; only `bls_invalid` semantics matter on-chain (counter
        // increment), and even non-`bls_invalid` failures count as failures
        // for the migration heuristic.
        if (_responses[challengeId].respondedAt == 0 && uint64(block.timestamp) <= ch.deadline) {
            revert ResponseRequired(challengeId);
        }

        _verdicts[challengeId] = Verdict({
            recorded: true,
            success: success,
            reason: reason,
            verdictAt: uint64(block.timestamp),
            auditor: msg.sender
        });

        bytes32 bundleId = ch.bundleId;
        uint32 shardIdx = ch.shardIdx;
        if (success) {
            // Reset the failure counter on any successful verdict.
            if (_consecutiveFailures[bundleId][shardIdx] != 0) {
                _consecutiveFailures[bundleId][shardIdx] = 0;
            }
        } else {
            uint32 next = _consecutiveFailures[bundleId][shardIdx] + 1;
            if (next >= MAX_CONSECUTIVE_FAILURES) {
                // Reset BEFORE emitting so external observers (and any
                // re-entrancy via the event listener path) see the counter at
                // its post-migration value of 0. The event carries the
                // pre-reset count for forensic logging.
                _consecutiveFailures[bundleId][shardIdx] = 0;
                emit ShardMigrationRequired(bundleId, shardIdx, next);
            } else {
                _consecutiveFailures[bundleId][shardIdx] = next;
            }
        }

        emit VerdictRecorded(challengeId, success, reason);
    }

    // --- views ----------------------------------------------------------

    /// @notice Returns the challenge record. Reverts with {ChallengeNotFound}.
    function getChallenge(bytes32 challengeId) external view returns (Challenge memory) {
        Challenge storage ch = _challenges[challengeId];
        if (ch.bundleId == bytes32(0)) revert ChallengeNotFound(challengeId);
        return ch;
    }

    /// @notice Returns the response record. Reverts with {ResponseRequired}.
    function getResponse(bytes32 challengeId) external view returns (Response memory) {
        Response storage r = _responses[challengeId];
        if (r.respondedAt == 0) revert ResponseRequired(challengeId);
        return r;
    }

    /// @notice Returns the verdict record. Reverts with {VerdictAlreadyRecorded}'s
    ///         dual — {ChallengeNotFound} when no challenge exists; we reuse
    ///         {ResponseRequired} when a verdict was never recorded so callers
    ///         see one consistent "missing" error per record type.
    function getVerdict(bytes32 challengeId) external view returns (Verdict memory) {
        Verdict storage v = _verdicts[challengeId];
        if (!v.recorded) {
            // A verdict can be missing because the challenge itself doesn't
            // exist OR because no verdict has been recorded yet. Distinguish
            // for the caller.
            if (_challenges[challengeId].bundleId == bytes32(0)) revert ChallengeNotFound(challengeId);
            revert ResponseRequired(challengeId);
        }
        return v;
    }

    /// @notice Returns the current consecutive-failure count for the
    ///         `(bundleId, shardIdx)` pair (0 if never recorded or just reset).
    function getConsecutiveFailures(bytes32 bundleId, uint32 shardIdx) external view returns (uint32) {
        return _consecutiveFailures[bundleId][shardIdx];
    }

    // --- internal -------------------------------------------------------

    /// @dev Verifies a Merkle proof for `(leafHash, leafIdx)` against
    ///      `expectedRoot`, mirroring `internal/merkle.Verify` EXACTLY:
    ///      - Hash function is SHA-256 (NOT keccak256). The off-chain Merkle
    ///        stack (internal/merkle, internal/crypto) is built on SHA-256
    ///        because chunk integrity tags are SHA-256 elsewhere in the
    ///        pipeline; switching either side would force a coordinated
    ///        migration. SHA-256 is an EVM precompile (address 0x02) so the
    ///        gas cost is comparable to keccak.
    ///      - Pairing rule: `cur = sha256(cur || sib)` if `idx % 2 == 0`,
    ///        else `cur = sha256(sib || cur)`.
    ///      - Odd-level duplication: when the current level has odd width and
    ///        `idx` is the last (duplicated) node, the responder's proof MUST
    ///        provide `sib == cur` (which `internal/merkle.Proof` does); we do
    ///        not re-derive sibling-equals-self here because the proof byte
    ///        stream is what the BLS signature commits to.
    ///      - `levelWidth` tracks the post-duplication width so that the index
    ///        bookkeeping stays in sync with how the tree was actually built.
    ///      Returns (ok, computedRoot) so {MerkleProofInvalid} can carry the
    ///      computed value for off-chain debugging.
    function _verifyMerkleProof(
        bytes32 leafHash,
        uint32 leafIdx,
        bytes32[] memory proof,
        uint32 totalLeaves,
        bytes32 expectedRoot
    ) internal pure returns (bool ok, bytes32 computed) {
        if (totalLeaves == 0 || leafIdx >= totalLeaves) {
            return (false, bytes32(0));
        }
        // Single-leaf tree: the leaf hash is the root and the proof must be empty.
        if (totalLeaves == 1) {
            if (proof.length != 0) return (false, leafHash);
            return (leafHash == expectedRoot, leafHash);
        }
        if (proof.length != _treeDepth(totalLeaves)) {
            return (false, bytes32(0));
        }

        bytes32 cur = leafHash;
        uint256 idx = leafIdx;
        uint256 levelWidth = totalLeaves;

        for (uint256 i = 0; i < proof.length; ++i) {
            bytes32 sib = proof[i];
            if (idx % 2 == 0) {
                cur = sha256(abi.encodePacked(cur, sib));
            } else {
                cur = sha256(abi.encodePacked(sib, cur));
            }
            idx /= 2;
            if (levelWidth % 2 == 1) {
                levelWidth += 1;
            }
            levelWidth /= 2;
        }

        return (cur == expectedRoot, cur);
    }

    /// @dev Mirrors `internal/merkle.treeDepth`: counts internal levels above
    ///      the leaves, accounting for odd-width duplication at each level.
    function _treeDepth(uint32 n) private pure returns (uint256 depth) {
        if (n <= 1) return 0;
        uint256 cur = n;
        while (cur > 1) {
            if (cur % 2 == 1) cur += 1;
            cur /= 2;
            depth += 1;
        }
    }
}
