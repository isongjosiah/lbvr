// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {Test} from "forge-std/Test.sol";
import {console2} from "forge-std/console2.sol";
import {PoRVerifier} from "../src/PoRVerifier.sol";
import {CIDRegistry} from "../src/CIDRegistry.sol";

/// @title PoRVerifierGasTest
/// @notice E5 verify-side gas measurement (CLAUDE.md §8 Fig 6).
///         Each test_gas_* function emits one or more lines of the form
///
///             E5_GAS,<fn>,<depth>,<gas>
///
///         to forge stdout via console2.log. cmd/bench/e5/gas.go shells
///         out to `forge test --match-contract PoRVerifierGas -vv` and
///         regex-parses these lines into the run JSON. Keep the format
///         stable.
///
///         Gas semantics: gasleft() before/after each external call. The
///         in-memory EVM is at the Cancun fork (matches Polygon zkEVM
///         Cardona). Per-call overhead from gasleft() itself is ~2 gas,
///         negligible relative to the 50k–200k gas range we measure.
contract PoRVerifierGasTest is Test {
    PoRVerifier internal por;
    CIDRegistry internal registry;

    address internal admin = address(0xA11CE);
    address internal auditor = address(0xAA1D);
    address internal responder = address(0xC0DE);

    bytes32 internal constant BUNDLE_ID = keccak256("lbvr://bundle/e5-gas");
    bytes32 internal constant POLICY_ID = keccak256("policy://e5-gas");

    bytes internal constant CID_HOT = bytes("QmHotE5GasShardZeroAAAAAAAAAAAAAAAAAAAAAAAAA");
    bytes internal constant CID_WARM = bytes("bafkwarmE5GasShardOneBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB");
    bytes internal constant CID_COLD = bytes("bafkcoldE5GasParityZeroCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC");

    bytes internal constant BLS_SIG =
        hex"a0b1c2d3e4f5061728394a5b6c7d8e9f0a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4f5061728394a5b6c7d8e9f"
        hex"0a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4f5061728394a5b6c7d8e9f";

    struct Bundle {
        bytes32 merkleRoot;
        uint32 numChunks;
        bytes32[] leaves;
    }

    function setUp() public {
        vm.startPrank(admin);
        registry = new CIDRegistry(admin, 0);
        por = new PoRVerifier(admin, 0, address(registry));
        por.grantRole(por.AUDITOR_ROLE(), auditor);
        por.grantRole(por.RESPONDER_ROLE(), responder);
        vm.stopPrank();
    }

    // --- per-depth full-cycle gas tests ---------------------------------
    //
    // depth → numChunks: 3→8, 5→32, 7→128, 9→512, 11→2048, 13→8192.
    //
    // These dominate the bench's wall time (the depth=13 test builds an
    // 8192-leaf tree in pure Solidity). To skip the heavier ones during
    // local iteration, set FOUNDRY_PROFILE=fast in foundry.toml or invoke
    // with --match-test depth_(3|5|7).

    function test_gas_cycle_depth_3() public {
        _runCycle(3, 8);
    }

    function test_gas_cycle_depth_5() public {
        _runCycle(5, 32);
    }

    function test_gas_cycle_depth_7() public {
        _runCycle(7, 128);
    }

    function test_gas_cycle_depth_9() public {
        _runCycle(9, 512);
    }

    function test_gas_cycle_depth_11() public {
        _runCycle(11, 2048);
    }

    function test_gas_cycle_depth_13() public {
        _runCycle(13, 8192);
    }

    // --- core measurement loop ------------------------------------------

    /// @dev Registers a bundle of `numChunks` leaves, runs the full PoR
    ///      cycle (post → respond → verdict), and emits per-call gas to
    ///      stdout in the format `E5_GAS,<fn>,<depth>,<gas>`.
    function _runCycle(uint32 depth, uint32 numChunks) internal {
        Bundle memory b = _registerBundle(numChunks);
        // Pick a chunk in the middle of the tree so the proof exercises
        // both left and right siblings — the worst-case proof length is
        // depth, regardless of which leaf we pick, but a mid-tree leaf
        // gives the most representative branch shape.
        uint32 chunkIdx = numChunks / 2;
        bytes32 nonce = keccak256(abi.encodePacked("nonce/", depth));
        uint64 win = 1 hours;

        // post -----------------------------------------------------------
        vm.prank(auditor);
        uint256 g0 = gasleft();
        bytes32 id = por.postChallenge(BUNDLE_ID, 0, chunkIdx, nonce, win);
        uint256 g1 = gasleft();
        console2.log("E5_GAS,post,", uint256(depth), ",", g0 - g1);

        // respond --------------------------------------------------------
        bytes32[] memory proof = _proofFor(b, chunkIdx);
        vm.prank(responder);
        g0 = gasleft();
        por.respondToChallenge(id, b.leaves[chunkIdx], proof, BLS_SIG, b.numChunks);
        g1 = gasleft();
        console2.log("E5_GAS,respond,", uint256(depth), ",", g0 - g1);

        // verdict --------------------------------------------------------
        vm.prank(auditor);
        g0 = gasleft();
        por.recordVerdict(id, true, "ok");
        g1 = gasleft();
        console2.log("E5_GAS,verdict,", uint256(depth), ",", g0 - g1);
    }

    // --- helpers (lifted from PoRVerifier.t.sol; intentional duplication
    //     for self-containment of the gas-measurement contract) ----------

    function _registerBundle(uint32 numChunks) internal returns (Bundle memory b) {
        b = _buildBundle(numChunks);

        CIDRegistry.ShardPlacement[] memory shards = new CIDRegistry.ShardPlacement[](3);
        shards[0] = CIDRegistry.ShardPlacement({cid: CID_HOT, tier: CIDRegistry.TierClass.Hot});
        shards[1] = CIDRegistry.ShardPlacement({cid: CID_WARM, tier: CIDRegistry.TierClass.Warm});
        shards[2] = CIDRegistry.ShardPlacement({cid: CID_COLD, tier: CIDRegistry.TierClass.Cold});

        registry.registerBundle(BUNDLE_ID, b.merkleRoot, b.numChunks, shards, POLICY_ID);
    }

    function _buildBundle(uint32 numChunks) internal pure returns (Bundle memory b) {
        require(numChunks > 0, "numChunks=0 not supported");
        b.numChunks = numChunks;
        b.leaves = new bytes32[](numChunks);
        for (uint256 i = 0; i < numChunks; ++i) {
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
                padded[width] = cur[width - 1];
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

    function _proofFor(Bundle memory b, uint32 chunkIdx) internal pure returns (bytes32[] memory proof) {
        require(chunkIdx < b.numChunks, "chunkIdx oor");
        if (b.numChunks == 1) {
            return new bytes32[](0);
        }

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
            levels[levelCount++] = next;
            cur = next;
        }

        proof = new bytes32[](levelCount - 1);
        uint256 idx = chunkIdx;
        for (uint256 lvl = 0; lvl < levelCount - 1; ++lvl) {
            bytes32[] memory row = levels[lvl];
            uint256 sibIdx = idx ^ 1;
            if (sibIdx >= row.length) {
                proof[lvl] = row[idx];
            } else {
                proof[lvl] = row[sibIdx];
            }
            idx /= 2;
        }
    }
}
