// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {Test, Vm} from "forge-std/Test.sol";
import {PoRVerifier} from "../src/PoRVerifier.sol";
import {CIDRegistry} from "../src/CIDRegistry.sol";
import {IAccessControl} from "@openzeppelin/contracts/access/IAccessControl.sol";

/// @title PoRVerifierTest
/// @notice Foundry tests for PoRVerifier. Mirrors the structure of
///         CIDRegistry.t.sol and AuditorLog.t.sol. Uses a real CIDRegistry as
///         the source of (merkleRoot, numChunks) so the on-chain proof
///         verification is end-to-end.
contract PoRVerifierTest is Test {
    PoRVerifier internal por;
    CIDRegistry internal registry;

    address internal admin = address(0xA11CE);
    address internal auditor = address(0xAA1D);
    address internal responder = address(0xC0DE);
    address internal stranger = address(0xDEAD);

    bytes32 internal constant BUNDLE_ID = keccak256("lbvr://bundle/abc123");
    bytes32 internal constant POLICY_ID = keccak256("policy://ehds/clinician-read");

    bytes internal constant CID_HOT = bytes("QmHotTierShardDataZeroAAAAAAAAAAAAAAAAAAAAAAAAA");
    bytes internal constant CID_WARM = bytes("bafkwarmTierShardDataOneBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB");
    bytes internal constant CID_COLD = bytes("bafkcoldTierParityZeroCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC");

    // Realistic 96-byte BLS-G2 signature shape (content irrelevant — sig is not
    // verified on-chain). Two distinguishable shapes for double-response tests.
    bytes internal constant BLS_SIG = hex"a0b1c2d3e4f5061728394a5b6c7d8e9f0a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4f5061728394a5b6c7d8e9f"
        hex"0a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4f5061728394a5b6c7d8e9f";

    event ChallengePosted(
        bytes32 indexed challengeId,
        bytes32 indexed bundleId,
        uint32 indexed shardIdx,
        uint32 chunkIdx,
        bytes32 nonce,
        uint64 deadline
    );
    event ChallengeResponded(bytes32 indexed challengeId, bytes32 chunkHash, address indexed responder);
    event VerdictRecorded(bytes32 indexed challengeId, bool success, string reason);
    event ShardMigrationRequired(
        bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 consecutiveFailures
    );

    function setUp() public {
        vm.startPrank(admin);
        registry = new CIDRegistry(admin, 0);
        por = new PoRVerifier(admin, 0, address(registry));
        por.grantRole(por.AUDITOR_ROLE(), auditor);
        por.grantRole(por.RESPONDER_ROLE(), responder);
        vm.stopPrank();
    }

    // --- helpers ---------------------------------------------------------

    struct Bundle {
        bytes32 merkleRoot;
        uint32 numChunks;
        bytes32[] leaves; // leaf hashes (= sha256(chunk_i))
    }

    function _registerBundle(uint32 numChunks) internal returns (Bundle memory b) {
        b = _buildBundle(numChunks);

        CIDRegistry.ShardPlacement[] memory shards = new CIDRegistry.ShardPlacement[](3);
        shards[0] = CIDRegistry.ShardPlacement({cid: CID_HOT, tier: CIDRegistry.TierClass.Hot});
        shards[1] = CIDRegistry.ShardPlacement({cid: CID_WARM, tier: CIDRegistry.TierClass.Warm});
        shards[2] = CIDRegistry.ShardPlacement({cid: CID_COLD, tier: CIDRegistry.TierClass.Cold});

        // Note: msg.sender during this prank is `address(this)`, but ownership
        // is irrelevant for PoRVerifier reads.
        registry.registerBundle(BUNDLE_ID, b.merkleRoot, b.numChunks, shards, POLICY_ID);
    }

    /// @dev Build a Merkle tree using SHA-256 + Bitcoin-style odd-width
    ///      duplication, matching internal/merkle.Build EXACTLY. Leaves are
    ///      sha256(uint256(i)) so each leaf is deterministic and distinct.
    ///      Returns the root, the leaf count, and the leaf hashes (so callers
    ///      can call _proofFor(b, i) to get a per-leaf proof).
    function _buildBundle(uint32 numChunks) internal pure returns (Bundle memory b) {
        require(numChunks > 0, "numChunks=0 not supported in tests");
        b.numChunks = numChunks;
        b.leaves = new bytes32[](numChunks);
        for (uint256 i = 0; i < numChunks; ++i) {
            // sha256 of the chunk bytes; we treat the index as the chunk content
            // for fixture purposes — the real tree hashes the actual chunk.
            b.leaves[i] = sha256(abi.encodePacked(uint256(i)));
        }
        b.merkleRoot = _rootOf(b.leaves);
    }

    function _rootOf(bytes32[] memory leaves) internal pure returns (bytes32) {
        if (leaves.length == 0) return bytes32(0);
        bytes32[] memory cur = leaves;
        while (cur.length > 1) {
            uint256 width = cur.length;
            if (width % 2 == 1) {
                bytes32[] memory padded = new bytes32[](width + 1);
                for (uint256 i = 0; i < width; ++i) padded[i] = cur[i];
                padded[width] = cur[width - 1]; // duplicate last
                cur = padded;
                width += 1;
            }
            bytes32[] memory next = new bytes32[](width / 2);
            for (uint256 i = 0; i < width; i += 2) {
                next[i / 2] = sha256(abi.encodePacked(cur[i], cur[i + 1]));
            }
            cur = next;
        }
        return cur[0];
    }

    /// @dev Produces the sibling path for `chunkIdx` in the same level-by-level
    ///      order that PoRVerifier._verifyMerkleProof consumes. Mirrors
    ///      internal/merkle.Tree.Proof: when the current level has odd width
    ///      and idx is the duplicated tail, the sibling is `cur` itself.
    function _proofFor(Bundle memory b, uint32 chunkIdx) internal pure returns (bytes32[] memory proof) {
        require(chunkIdx < b.numChunks, "chunkIdx oor");
        if (b.numChunks == 1) {
            return new bytes32[](0);
        }

        // Reconstruct each level so we can read the per-level sibling. We
        // store the levels for clarity; depth is ~log2(N), which for
        // realistic N (<= a few thousand) is trivial.
        bytes32[][] memory levels = new bytes32[][](64);
        uint256 levelCount = 0;
        levels[levelCount++] = b.leaves;
        bytes32[] memory cur = b.leaves;
        while (cur.length > 1) {
            uint256 width = cur.length;
            if (width % 2 == 1) {
                bytes32[] memory padded = new bytes32[](width + 1);
                for (uint256 i = 0; i < width; ++i) padded[i] = cur[i];
                padded[width] = cur[width - 1];
                cur = padded;
                width += 1;
            }
            bytes32[] memory next = new bytes32[](width / 2);
            for (uint256 i = 0; i < width; i += 2) {
                next[i / 2] = sha256(abi.encodePacked(cur[i], cur[i + 1]));
            }
            // Note: we store the *unpadded* level in `levels` to match what
            // internal/merkle.Tree.levels holds (the duplication is ephemeral
            // per the package's buildFromLeaves logic — but for proof
            // generation, the unpadded width is what determines whether
            // sibIdx >= len(row) triggers the "sibling is self" branch).
            // Wait: the Go impl appends the duplicated leaf into `cur` before
            // pairing, so levels[lvl] in Go contains the ORIGINAL row width;
            // see merkle.go buildFromLeaves where `t.levels = append(t.levels, leaves)`
            // happens BEFORE the loop. Inside the loop `cur` is mutated with
            // the duplication but `t.levels` is appended with `next` (already
            // halved). So levels[lvl] holds widths: [N, ceil(N/2), ...] in the
            // pre-duplication sense. We replicate that.
            levels[levelCount++] = next;
            cur = next;
        }

        proof = new bytes32[](levelCount - 1);
        uint256 idx = chunkIdx;
        for (uint256 lvl = 0; lvl < levelCount - 1; ++lvl) {
            bytes32[] memory row = levels[lvl];
            uint256 sibIdx = idx ^ 1;
            if (sibIdx >= row.length) {
                // Odd-width tail: sibling is self.
                proof[lvl] = row[idx];
            } else {
                proof[lvl] = row[sibIdx];
            }
            idx /= 2;
        }
    }

    function _post(uint32 shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 win) internal returns (bytes32 id) {
        vm.prank(auditor);
        id = por.postChallenge(BUNDLE_ID, shardIdx, chunkIdx, nonce, win);
    }

    // --- happy path ------------------------------------------------------

    function test_postRespondVerdict_happyPath() public {
        Bundle memory b = _registerBundle(8);
        uint32 chunkIdx = 5;
        bytes32 nonce = keccak256("nonce/1");

        // Post.
        bytes32 id = keccak256(
            abi.encode(BUNDLE_ID, uint32(0), chunkIdx, nonce, uint64(block.timestamp))
        );
        vm.expectEmit(true, true, true, true, address(por));
        emit ChallengePosted(id, BUNDLE_ID, 0, chunkIdx, nonce, uint64(block.timestamp + 1 hours));

        bytes32 returned = _post(0, chunkIdx, nonce, 1 hours);
        assertEq(returned, id, "challengeId determinism");

        PoRVerifier.Challenge memory ch = por.getChallenge(id);
        assertEq(ch.bundleId, BUNDLE_ID);
        assertEq(ch.shardIdx, 0);
        assertEq(ch.chunkIdx, chunkIdx);
        assertEq(ch.auditor, auditor);
        assertEq(ch.deadline, uint64(block.timestamp + 1 hours));

        // Respond.
        bytes32 leaf = b.leaves[chunkIdx];
        bytes32[] memory proof = _proofFor(b, chunkIdx);

        vm.expectEmit(true, true, false, true, address(por));
        emit ChallengeResponded(id, leaf, responder);

        vm.prank(responder);
        por.respondToChallenge(id, leaf, proof, BLS_SIG, b.numChunks);

        PoRVerifier.Response memory r = por.getResponse(id);
        assertEq(r.chunkHash, leaf);
        assertEq(r.responder, responder);
        assertEq(r.blsSig, BLS_SIG);
        assertEq(r.respondedAt, uint64(block.timestamp));

        // Verdict.
        vm.expectEmit(true, false, false, true, address(por));
        emit VerdictRecorded(id, true, "ok");
        vm.prank(auditor);
        por.recordVerdict(id, true, "ok");

        PoRVerifier.Verdict memory v = por.getVerdict(id);
        assertTrue(v.recorded);
        assertTrue(v.success);
        assertEq(v.reason, "ok");
        assertEq(v.auditor, auditor);
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 0), 0);
    }

    function test_respond_singleLeafTree() public {
        Bundle memory b = _registerBundle(1);
        bytes32 nonce = keccak256("nonce/single");
        bytes32 id = _post(1, 0, nonce, 1 hours);

        bytes32[] memory empty = new bytes32[](0);
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], empty, BLS_SIG, 1);

        assertEq(por.getResponse(id).chunkHash, b.leaves[0]);
    }

    function test_respond_oddWidthTree() public {
        // 7-leaf tree exercises the odd-width duplication at level 0 (7 -> 8).
        Bundle memory b = _registerBundle(7);

        for (uint32 i = 0; i < 7; ++i) {
            bytes32 nonce = keccak256(abi.encode("nonce/odd", i));
            bytes32 id = _post(2, i, nonce, 1 hours);
            bytes32[] memory proof = _proofFor(b, i);
            vm.prank(responder);
            por.respondToChallenge(id, b.leaves[i], proof, BLS_SIG, b.numChunks);
            assertEq(por.getResponse(id).chunkHash, b.leaves[i]);
        }
    }

    // --- post: reverts ---------------------------------------------------

    function test_post_revertsOnInvalidShardIdx() public {
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.InvalidShardIdx.selector, uint32(3), uint32(2)));
        vm.prank(auditor);
        por.postChallenge(BUNDLE_ID, 3, 0, keccak256("n"), 1 hours);

        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.InvalidShardIdx.selector, uint32(99), uint32(2)));
        vm.prank(auditor);
        por.postChallenge(BUNDLE_ID, 99, 0, keccak256("n"), 1 hours);
    }

    function test_post_revertsOnZeroResponseWindow() public {
        vm.expectRevert(PoRVerifier.InvalidResponseWindow.selector);
        vm.prank(auditor);
        por.postChallenge(BUNDLE_ID, 0, 0, keccak256("n"), 0);
    }

    function test_post_revertsWithoutAuditorRole() public {
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, por.AUDITOR_ROLE()
            )
        );
        vm.prank(stranger);
        por.postChallenge(BUNDLE_ID, 0, 0, keccak256("n"), 1 hours);
    }

    // --- respond: reverts ------------------------------------------------

    function test_respond_revertsOnMissingChallenge() public {
        bytes32 fake = keccak256("not-a-challenge");
        bytes32[] memory proof = new bytes32[](0);
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ChallengeNotFound.selector, fake));
        vm.prank(responder);
        por.respondToChallenge(fake, keccak256("h"), proof, BLS_SIG, 1);
    }

    function test_respond_revertsAfterDeadline() public {
        Bundle memory b = _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        uint64 deadline = uint64(block.timestamp + 1 hours);

        vm.warp(block.timestamp + 1 hours + 1);
        bytes32[] memory proof = _proofFor(b, 0);

        vm.expectRevert(
            abi.encodeWithSelector(
                PoRVerifier.ChallengeExpired.selector, id, deadline, uint64(block.timestamp)
            )
        );
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
    }

    function test_respond_revertsOnDoubleResponse() public {
        Bundle memory b = _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 0);

        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);

        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ChallengeAlreadyResponded.selector, id));
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
    }

    function test_respond_revertsOnEmptyChunkHash() public {
        _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = new bytes32[](2);
        vm.expectRevert(PoRVerifier.EmptyChunkHash.selector);
        vm.prank(responder);
        por.respondToChallenge(id, bytes32(0), proof, BLS_SIG, 4);
    }

    function test_respond_revertsOnEmptyBLSSig() public {
        Bundle memory b = _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 0);
        vm.expectRevert(PoRVerifier.EmptyBLSSig.selector);
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], proof, bytes(""), b.numChunks);
    }

    function test_respond_revertsOnBadProof() public {
        Bundle memory b = _registerBundle(8);
        bytes32 id = _post(0, 3, keccak256("n"), 1 hours);

        // Corrupt one proof element.
        bytes32[] memory proof = _proofFor(b, 3);
        proof[0] = keccak256("tampered");

        vm.expectRevert(); // Generic — payload includes a computed root we don't predict.
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[3], proof, BLS_SIG, b.numChunks);
    }

    function test_respond_revertsOnWrongLeafHash() public {
        Bundle memory b = _registerBundle(8);
        bytes32 id = _post(0, 3, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 3);

        vm.expectRevert();
        vm.prank(responder);
        por.respondToChallenge(id, keccak256("not-the-real-leaf"), proof, BLS_SIG, b.numChunks);
    }

    function test_respond_revertsOnTotalChunksMismatch() public {
        Bundle memory b = _registerBundle(8);
        bytes32 id = _post(0, 3, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 3);

        // Lying about the tree width should be rejected via the same
        // MerkleProofInvalid path (computed root reported as 0 to signal
        // "rejected before reconstruction").
        vm.expectRevert(
            abi.encodeWithSelector(PoRVerifier.MerkleProofInvalid.selector, b.merkleRoot, bytes32(0))
        );
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[3], proof, BLS_SIG, 7);
    }

    function test_respond_revertsWithoutResponderRole() public {
        Bundle memory b = _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 0);

        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, por.RESPONDER_ROLE()
            )
        );
        vm.prank(stranger);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
    }

    // --- recordVerdict: reverts -----------------------------------------

    function test_recordVerdict_revertsOnMissingChallenge() public {
        bytes32 fake = keccak256("not-a-challenge");
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ChallengeNotFound.selector, fake));
        vm.prank(auditor);
        por.recordVerdict(fake, true, "ok");
    }

    function test_recordVerdict_revertsBeforeDeadlineWithoutResponse() public {
        _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);

        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ResponseRequired.selector, id));
        vm.prank(auditor);
        por.recordVerdict(id, false, "timeout");
    }

    function test_recordVerdict_allowsTimeoutAfterDeadline() public {
        _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);

        // No response landed; advance past the deadline.
        vm.warp(block.timestamp + 1 hours + 1);

        vm.prank(auditor);
        por.recordVerdict(id, false, "timeout");

        PoRVerifier.Verdict memory v = por.getVerdict(id);
        assertTrue(v.recorded);
        assertFalse(v.success);
        assertEq(v.reason, "timeout");
        // Timeout counts as a failure for the consecutive counter.
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 0), 1);
    }

    function test_recordVerdict_revertsOnDoubleVerdict() public {
        Bundle memory b = _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 0);

        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);

        vm.prank(auditor);
        por.recordVerdict(id, true, "ok");

        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.VerdictAlreadyRecorded.selector, id));
        vm.prank(auditor);
        por.recordVerdict(id, false, "bls_invalid");
    }

    function test_recordVerdict_revertsWithoutAuditorRole() public {
        Bundle memory b = _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 0);
        vm.prank(responder);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);

        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, por.AUDITOR_ROLE()
            )
        );
        vm.prank(stranger);
        por.recordVerdict(id, true, "ok");
    }

    // --- views: reverts --------------------------------------------------

    function test_getChallenge_revertsIfMissing() public {
        bytes32 fake = keccak256("not-a-challenge");
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ChallengeNotFound.selector, fake));
        por.getChallenge(fake);
    }

    function test_getResponse_revertsIfMissing() public {
        _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ResponseRequired.selector, id));
        por.getResponse(id);
    }

    function test_getVerdict_revertsIfChallengeMissing() public {
        bytes32 fake = keccak256("not-a-challenge");
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ChallengeNotFound.selector, fake));
        por.getVerdict(fake);
    }

    function test_getVerdict_revertsIfNoVerdict() public {
        _registerBundle(4);
        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        vm.expectRevert(abi.encodeWithSelector(PoRVerifier.ResponseRequired.selector, id));
        por.getVerdict(id);
    }

    // --- failure tracking ------------------------------------------------

    /// @dev 3 consecutive bls_invalid verdicts on the same (bundleId, shardIdx)
    ///      MUST emit ShardMigrationRequired exactly once and reset the counter.
    function test_failures_emitMigrationAfterThree() public {
        Bundle memory b = _registerBundle(4);
        bytes32[] memory proof = _proofFor(b, 0);

        for (uint32 i = 0; i < 3; ++i) {
            bytes32 nonce = keccak256(abi.encode("nonce", i));
            bytes32 id = _post(1, 0, nonce, 1 hours);
            vm.prank(responder);
            por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);

            if (i == 2) {
                // Migration emit on the third failure.
                vm.expectEmit(true, true, false, true, address(por));
                emit ShardMigrationRequired(BUNDLE_ID, 1, 3);
            }
            vm.prank(auditor);
            por.recordVerdict(id, false, "bls_invalid");
        }

        // Counter resets after migration emit.
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 1), 0);
    }

    /// @dev Success in between resets the counter, so a subsequent failure
    ///      starts from 1 (no migration emit).
    function test_failures_successResetsCounter() public {
        Bundle memory b = _registerBundle(4);
        bytes32[] memory proof = _proofFor(b, 0);

        // 2 failures.
        for (uint32 i = 0; i < 2; ++i) {
            bytes32 id = _post(0, 0, keccak256(abi.encode("f", i)), 1 hours);
            vm.prank(responder);
            por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
            vm.prank(auditor);
            por.recordVerdict(id, false, "bls_invalid");
        }
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 0), 2);

        // 1 success — resets to 0.
        {
            bytes32 id = _post(0, 0, keccak256("ok"), 1 hours);
            vm.prank(responder);
            por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
            vm.prank(auditor);
            por.recordVerdict(id, true, "ok");
        }
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 0), 0);

        // 1 failure — counter at 1, no migration.
        {
            bytes32 id = _post(0, 0, keccak256("f-final"), 1 hours);
            vm.prank(responder);
            por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
            vm.recordLogs();
            vm.prank(auditor);
            por.recordVerdict(id, false, "bls_invalid");
            // Confirm no ShardMigrationRequired was emitted in the last call.
            Vm.Log[] memory logs = vm.getRecordedLogs();
            bytes32 sig = keccak256("ShardMigrationRequired(bytes32,uint32,uint32)");
            for (uint256 i = 0; i < logs.length; ++i) {
                assertTrue(logs[i].topics[0] != sig, "unexpected migration emit");
            }
        }
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 0), 1);
    }

    /// @dev Per-(bundleId, shardIdx) accounting: failures on shard 0 do not
    ///      pollute shard 1's counter.
    function test_failures_perShardIsolation() public {
        Bundle memory b = _registerBundle(4);
        bytes32[] memory proof = _proofFor(b, 0);

        for (uint32 i = 0; i < 2; ++i) {
            bytes32 id = _post(0, 0, keccak256(abi.encode("s0", i)), 1 hours);
            vm.prank(responder);
            por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);
            vm.prank(auditor);
            por.recordVerdict(id, false, "bls_invalid");
        }

        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 0), 2);
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 1), 0);
        assertEq(por.getConsecutiveFailures(BUNDLE_ID, 2), 0);
    }

    // --- role admin ------------------------------------------------------

    function test_adminCanGrantAndRevokeAuditorRole() public {
        bytes32 role = por.AUDITOR_ROLE();
        address newAuditor = address(0xBEEF);

        vm.prank(admin);
        por.grantRole(role, newAuditor);
        assertTrue(por.hasRole(role, newAuditor));

        vm.prank(newAuditor);
        por.postChallenge(BUNDLE_ID, 0, 0, keccak256("n"), 1 hours);

        vm.prank(admin);
        por.revokeRole(role, newAuditor);
        assertFalse(por.hasRole(role, newAuditor));

        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, newAuditor, role)
        );
        vm.prank(newAuditor);
        por.postChallenge(BUNDLE_ID, 0, 1, keccak256("n2"), 1 hours);
    }

    function test_adminCanGrantAndRevokeResponderRole() public {
        Bundle memory b = _registerBundle(4);
        bytes32 role = por.RESPONDER_ROLE();
        address newResponder = address(0xBEEF);

        vm.prank(admin);
        por.grantRole(role, newResponder);

        bytes32 id = _post(0, 0, keccak256("n"), 1 hours);
        bytes32[] memory proof = _proofFor(b, 0);

        vm.prank(newResponder);
        por.respondToChallenge(id, b.leaves[0], proof, BLS_SIG, b.numChunks);

        vm.prank(admin);
        por.revokeRole(role, newResponder);
        bytes32 id2 = _post(0, 1, keccak256("n2"), 1 hours);
        bytes32[] memory proof2 = _proofFor(b, 1);

        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, newResponder, role)
        );
        vm.prank(newResponder);
        por.respondToChallenge(id2, b.leaves[1], proof2, BLS_SIG, b.numChunks);
    }

    function test_nonAdminCannotGrantRole() public {
        bytes32 role = por.AUDITOR_ROLE();
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, por.DEFAULT_ADMIN_ROLE()
            )
        );
        vm.prank(stranger);
        por.grantRole(role, stranger);
    }

    // --- constructor -----------------------------------------------------

    function test_constructor_revertsOnZeroRegistry() public {
        vm.expectRevert(PoRVerifier.ZeroRegistry.selector);
        new PoRVerifier(admin, 0, address(0));
    }

    // --- fuzz ------------------------------------------------------------

    /// @dev Same inputs MUST produce the same challengeId; differing any input
    ///      (including the postedAt timestamp) MUST produce a distinct id.
    function testFuzz_postChallenge_idDeterministic(
        bytes32 bundleId,
        uint8 shardSeed,
        uint32 chunkIdx,
        bytes32 nonce,
        uint64 win
    ) public {
        uint32 shardIdx = uint32(shardSeed) % 3;
        win = uint64(bound(uint256(win), 1, 30 days));

        // Deterministic challengeId derivation: must be reproducible from inputs alone.
        uint64 ts1 = uint64(block.timestamp);
        bytes32 expected = keccak256(abi.encode(bundleId, shardIdx, chunkIdx, nonce, ts1));

        vm.prank(auditor);
        bytes32 got1 = por.postChallenge(bundleId, shardIdx, chunkIdx, nonce, win);
        assertEq(got1, expected, "id derivation");

        // Same inputs at a later timestamp produce a different id (postedAt is in the hash).
        vm.warp(block.timestamp + 1);
        vm.prank(auditor);
        bytes32 got2 = por.postChallenge(bundleId, shardIdx, chunkIdx, nonce, win);
        assertTrue(got1 != got2, "postedAt salts the id");
    }

    /// @dev Logs gas consumed by postChallenge across realistic inputs. No
    ///      ceiling asserted (E5 measures the actual cost; CLAUDE.md §3.1
    ///      target is ≤200k gas per submission for the verify path).
    function testFuzz_postChallenge_gasSnapshot(bytes32 bundleId, uint32 chunkIdx, bytes32 nonce) public {
        vm.prank(auditor);
        uint256 gasBefore = gasleft();
        por.postChallenge(bundleId, 0, chunkIdx, nonce, 1 hours);
        uint256 gasUsed = gasBefore - gasleft();

        emit log_named_uint("postChallenge.gasUsed", gasUsed);
    }

    /// @dev Logs gas for respondToChallenge across realistic proof depths. The
    ///      tree depths sampled (8 -> depth 3, 256 -> depth 8, 4096 -> depth 12)
    ///      bracket the realistic range for Synthea bundles
    ///      (P50 ~137 chunks ~ depth 8; max ~6300 chunks ~ depth 13).
    function testFuzz_respondToChallenge_gasSnapshot(uint8 sizeSeed, uint32 chunkSeed) public {
        uint32[3] memory sizes = [uint32(8), uint32(256), uint32(4096)];
        uint32 numChunks = sizes[sizeSeed % 3];
        uint32 chunkIdx = chunkSeed % numChunks;

        Bundle memory b = _buildBundle(numChunks);
        CIDRegistry.ShardPlacement[] memory shards = new CIDRegistry.ShardPlacement[](3);
        shards[0] = CIDRegistry.ShardPlacement({cid: CID_HOT, tier: CIDRegistry.TierClass.Hot});
        shards[1] = CIDRegistry.ShardPlacement({cid: CID_WARM, tier: CIDRegistry.TierClass.Warm});
        shards[2] = CIDRegistry.ShardPlacement({cid: CID_COLD, tier: CIDRegistry.TierClass.Cold});
        bytes32 bid = keccak256(abi.encode("gas-snapshot-bundle", numChunks));
        registry.registerBundle(bid, b.merkleRoot, b.numChunks, shards, POLICY_ID);

        bytes32 nonce = keccak256(abi.encode("nonce/gas", chunkSeed));
        vm.prank(auditor);
        bytes32 id = por.postChallenge(bid, 0, chunkIdx, nonce, 1 hours);

        bytes32[] memory proof = _proofFor(b, chunkIdx);

        vm.prank(responder);
        uint256 gasBefore = gasleft();
        por.respondToChallenge(id, b.leaves[chunkIdx], proof, BLS_SIG, numChunks);
        uint256 gasUsed = gasBefore - gasleft();

        emit log_named_uint("respondToChallenge.numChunks", numChunks);
        emit log_named_uint("respondToChallenge.proofDepth", proof.length);
        emit log_named_uint("respondToChallenge.gasUsed", gasUsed);
    }
}

