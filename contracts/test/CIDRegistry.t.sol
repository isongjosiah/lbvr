// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {Test} from "forge-std/Test.sol";
import {CIDRegistry} from "../src/CIDRegistry.sol";
import {IAccessControl} from "@openzeppelin/contracts/access/IAccessControl.sol";

contract CIDRegistryTest is Test {
    CIDRegistry internal registry;

    address internal admin = address(0xA11CE);
    address internal client = address(0xB0B);
    address internal migrator = address(0xC0DE);
    address internal stranger = address(0xDEAD);

    // Realistic-shape CIDv0 / CIDv1 byte strings (46 / ~59 bytes in multibase form).
    bytes internal constant CID_HOT = bytes("QmHotTierShardDataZeroAAAAAAAAAAAAAAAAAAAAAAAAA");
    bytes internal constant CID_WARM = bytes("bafkwarmTierShardDataOneBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB");
    bytes internal constant CID_COLD = bytes("bafkcoldTierParityZeroCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC");

    bytes32 internal constant BUNDLE_ID = keccak256("lbvr://bundle/abc123");
    bytes32 internal constant MERKLE_ROOT = keccak256("merkle-root-v1");
    bytes32 internal constant POLICY_ID = keccak256("policy://ehds/clinician-read");
    // Realistic chunk count for a ~2.2 MB P50 bundle at 16 KiB chunks.
    uint32 internal constant NUM_CHUNKS = 137;

    event BundleRegistered(
        bytes32 indexed bundleId, bytes32 indexed merkleRoot, address indexed owner, bytes32 policyId
    );
    event ShardLayoutUpdated(
        bytes32 indexed bundleId, CIDRegistry.ShardPlacement[] oldShards, CIDRegistry.ShardPlacement[] newShards
    );

    function setUp() public {
        vm.prank(admin);
        registry = new CIDRegistry(admin, 0);
    }

    // --- helpers ---------------------------------------------------------

    function _validLayout() internal pure returns (CIDRegistry.ShardPlacement[] memory shards) {
        shards = new CIDRegistry.ShardPlacement[](3);
        shards[0] = CIDRegistry.ShardPlacement({cid: CID_HOT, tier: CIDRegistry.TierClass.Hot});
        shards[1] = CIDRegistry.ShardPlacement({cid: CID_WARM, tier: CIDRegistry.TierClass.Warm});
        shards[2] = CIDRegistry.ShardPlacement({cid: CID_COLD, tier: CIDRegistry.TierClass.Cold});
    }

    function _layoutOfSize(uint256 n) internal pure returns (CIDRegistry.ShardPlacement[] memory shards) {
        shards = new CIDRegistry.ShardPlacement[](n);
        for (uint256 i = 0; i < n; ++i) {
            shards[i] = CIDRegistry.ShardPlacement({cid: CID_HOT, tier: CIDRegistry.TierClass.Hot});
        }
    }

    // --- happy path ------------------------------------------------------

    function test_registerBundle_happyPath() public {
        CIDRegistry.ShardPlacement[] memory shards = _validLayout();

        vm.expectEmit(true, true, true, true, address(registry));
        emit BundleRegistered(BUNDLE_ID, MERKLE_ROOT, client, POLICY_ID);

        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, shards, POLICY_ID);

        CIDRegistry.BundleRecord memory rec = registry.getRecord(BUNDLE_ID);
        assertEq(rec.merkleRoot, MERKLE_ROOT, "merkleRoot");
        assertEq(rec.owner, client, "owner");
        assertEq(rec.policyId, POLICY_ID, "policyId");
        assertEq(rec.lastMigratedAt, 0, "lastMigratedAt");
        assertEq(rec.registeredAt, uint64(block.timestamp), "registeredAt");
        assertEq(rec.numChunks, NUM_CHUNKS, "numChunks");
        assertEq(rec.shards.length, 3, "shardCount");
        assertEq(rec.shards[0].cid, CID_HOT, "shard0.cid");
        assertEq(uint8(rec.shards[1].tier), uint8(CIDRegistry.TierClass.Warm), "shard1.tier");
        assertEq(uint8(rec.shards[2].tier), uint8(CIDRegistry.TierClass.Cold), "shard2.tier");
    }

    function test_getShardLayout_returnsStoredLayout() public {
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        CIDRegistry.ShardPlacement[] memory got = registry.getShardLayout(BUNDLE_ID);
        assertEq(got.length, 3);
        assertEq(got[0].cid, CID_HOT);
        assertEq(got[1].cid, CID_WARM);
        assertEq(got[2].cid, CID_COLD);
    }

    // --- registration reverts -------------------------------------------

    function test_registerBundle_revertsOnDuplicate() public {
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.BundleAlreadyRegistered.selector, BUNDLE_ID));
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);
    }

    function test_registerBundle_revertsOnZeroShards() public {
        CIDRegistry.ShardPlacement[] memory shards = _layoutOfSize(0);
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.InvalidShardCount.selector, 0, 3));
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, shards, POLICY_ID);
    }

    function test_registerBundle_revertsOnTwoShards() public {
        CIDRegistry.ShardPlacement[] memory shards = _layoutOfSize(2);
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.InvalidShardCount.selector, 2, 3));
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, shards, POLICY_ID);
    }

    function test_registerBundle_revertsOnFourShards() public {
        CIDRegistry.ShardPlacement[] memory shards = _layoutOfSize(4);
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.InvalidShardCount.selector, 4, 3));
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, shards, POLICY_ID);
    }

    function test_registerBundle_revertsOnEmptyCID() public {
        CIDRegistry.ShardPlacement[] memory shards = _validLayout();
        shards[1].cid = bytes("");
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.EmptyShardCID.selector, 1));
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, shards, POLICY_ID);
    }

    function test_registerBundle_revertsOnZeroNumChunks() public {
        vm.expectRevert(CIDRegistry.NumChunksZero.selector);
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, 0, _validLayout(), POLICY_ID);
    }

    // --- read reverts ----------------------------------------------------

    function test_getRecord_revertsIfMissing() public {
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.BundleNotFound.selector, BUNDLE_ID));
        registry.getRecord(BUNDLE_ID);
    }

    function test_getShardLayout_revertsIfMissing() public {
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.BundleNotFound.selector, BUNDLE_ID));
        registry.getShardLayout(BUNDLE_ID);
    }

    // --- updateShardLayout ----------------------------------------------

    function test_updateShardLayout_happyPath() public {
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        CIDRegistry.ShardPlacement[] memory newShards = new CIDRegistry.ShardPlacement[](3);
        newShards[0] = CIDRegistry.ShardPlacement({cid: bytes("QmMigratedHotShard"), tier: CIDRegistry.TierClass.Hot});
        newShards[1] =
            CIDRegistry.ShardPlacement({cid: bytes("bafkMigratedWarmShard"), tier: CIDRegistry.TierClass.Warm});
        newShards[2] =
            CIDRegistry.ShardPlacement({cid: bytes("bafkMigratedColdShard"), tier: CIDRegistry.TierClass.Cold});

        // Move time forward so lastMigratedAt is distinguishable from registeredAt.
        vm.warp(block.timestamp + 30 days);

        // Don't strictly match the data payload (old/new arrays); just confirm the
        // indexed topic (bundleId) and that the event fires from the right contract.
        vm.expectEmit(true, false, false, false, address(registry));
        emit ShardLayoutUpdated(BUNDLE_ID, new CIDRegistry.ShardPlacement[](0), new CIDRegistry.ShardPlacement[](0));

        vm.prank(admin);
        registry.updateShardLayout(BUNDLE_ID, newShards);

        CIDRegistry.BundleRecord memory rec = registry.getRecord(BUNDLE_ID);
        assertEq(rec.shards.length, 3, "shardCount after migration");
        assertEq(rec.shards[0].cid, bytes("QmMigratedHotShard"));
        assertEq(rec.shards[1].cid, bytes("bafkMigratedWarmShard"));
        assertEq(rec.shards[2].cid, bytes("bafkMigratedColdShard"));
        assertEq(rec.lastMigratedAt, uint64(block.timestamp), "lastMigratedAt bumped");
    }

    function test_updateShardLayout_revertsWithoutRole() public {
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        CIDRegistry.ShardPlacement[] memory newShards = _validLayout();
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, registry.MIGRATOR_ROLE()
            )
        );
        vm.prank(stranger);
        registry.updateShardLayout(BUNDLE_ID, newShards);
    }

    function test_updateShardLayout_revertsIfBundleMissing() public {
        CIDRegistry.ShardPlacement[] memory newShards = _validLayout();
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.BundleNotFound.selector, BUNDLE_ID));
        vm.prank(admin);
        registry.updateShardLayout(BUNDLE_ID, newShards);
    }

    function test_updateShardLayout_revertsOnWrongShardCount() public {
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        CIDRegistry.ShardPlacement[] memory newShards = _layoutOfSize(2);
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.InvalidShardCount.selector, 2, 3));
        vm.prank(admin);
        registry.updateShardLayout(BUNDLE_ID, newShards);
    }

    function test_updateShardLayout_revertsOnEmptyCID() public {
        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        CIDRegistry.ShardPlacement[] memory newShards = _validLayout();
        newShards[2].cid = bytes("");
        vm.expectRevert(abi.encodeWithSelector(CIDRegistry.EmptyShardCID.selector, 2));
        vm.prank(admin);
        registry.updateShardLayout(BUNDLE_ID, newShards);
    }

    // --- role admin ------------------------------------------------------

    function test_adminCanGrantAndRevokeMigratorRole() public {
        bytes32 role = registry.MIGRATOR_ROLE();

        vm.prank(admin);
        registry.grantRole(role, migrator);
        assertTrue(registry.hasRole(role, migrator));

        vm.prank(client);
        registry.registerBundle(BUNDLE_ID, MERKLE_ROOT, NUM_CHUNKS, _validLayout(), POLICY_ID);

        // Granted migrator can update.
        vm.prank(migrator);
        registry.updateShardLayout(BUNDLE_ID, _validLayout());

        // Revoke, then confirm migrator is locked out.
        vm.prank(admin);
        registry.revokeRole(role, migrator);
        assertFalse(registry.hasRole(role, migrator));

        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, migrator, role)
        );
        vm.prank(migrator);
        registry.updateShardLayout(BUNDLE_ID, _validLayout());
    }

    function test_nonAdminCannotGrantRole() public {
        bytes32 role = registry.MIGRATOR_ROLE();
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, registry.DEFAULT_ADMIN_ROLE()
            )
        );
        vm.prank(stranger);
        registry.grantRole(role, stranger);
    }

    // --- gas snapshot ----------------------------------------------------

    /// @dev Fuzzes realistic shard CID lengths (32..100 bytes) and logs the gas consumed
    ///      by registerBundle. No ceiling asserted — E5 is where we measure it; this just
    ///      keeps a live datapoint that `forge snapshot` and CI gas reports can latch onto.
    function testFuzz_registerBundle_gasSnapshot(bytes32 seed, uint8 lenSeed) public {
        uint256 cidLen = 32 + (uint256(lenSeed) % 69); // [32, 100]
        bytes memory cid0 = _deterministicBytes(seed, uint8(cidLen));
        bytes memory cid1 = _deterministicBytes(keccak256(abi.encodePacked(seed, uint8(1))), uint8(cidLen));
        bytes memory cid2 = _deterministicBytes(keccak256(abi.encodePacked(seed, uint8(2))), uint8(cidLen));

        CIDRegistry.ShardPlacement[] memory shards = new CIDRegistry.ShardPlacement[](3);
        shards[0] = CIDRegistry.ShardPlacement({cid: cid0, tier: CIDRegistry.TierClass.Hot});
        shards[1] = CIDRegistry.ShardPlacement({cid: cid1, tier: CIDRegistry.TierClass.Warm});
        shards[2] = CIDRegistry.ShardPlacement({cid: cid2, tier: CIDRegistry.TierClass.Cold});

        bytes32 id = keccak256(abi.encodePacked(seed, "gas"));

        vm.pauseGasMetering();
        vm.prank(client);
        vm.resumeGasMetering();
        uint256 gasBefore = gasleft();
        registry.registerBundle(id, MERKLE_ROOT, NUM_CHUNKS, shards, POLICY_ID);
        uint256 gasUsed = gasBefore - gasleft();
        vm.pauseGasMetering();

        emit log_named_uint("registerBundle.gasUsed", gasUsed);
        emit log_named_uint("registerBundle.cidLen", cidLen);
    }

    function testFuzz_registerBundle_roundtrip(bytes32 bundleId, bytes32 merkleRoot, bytes32 policyId, bytes32 seed)
        public
    {
        vm.assume(bundleId != bytes32(0));
        bytes memory cid0 = _deterministicBytes(seed, 46);
        bytes memory cid1 = _deterministicBytes(keccak256(abi.encodePacked(seed, uint8(1))), 59);
        bytes memory cid2 = _deterministicBytes(keccak256(abi.encodePacked(seed, uint8(2))), 59);

        CIDRegistry.ShardPlacement[] memory shards = new CIDRegistry.ShardPlacement[](3);
        shards[0] = CIDRegistry.ShardPlacement({cid: cid0, tier: CIDRegistry.TierClass.Hot});
        shards[1] = CIDRegistry.ShardPlacement({cid: cid1, tier: CIDRegistry.TierClass.Warm});
        shards[2] = CIDRegistry.ShardPlacement({cid: cid2, tier: CIDRegistry.TierClass.Cold});

        vm.prank(client);
        registry.registerBundle(bundleId, merkleRoot, NUM_CHUNKS, shards, policyId);

        CIDRegistry.BundleRecord memory rec = registry.getRecord(bundleId);
        assertEq(rec.merkleRoot, merkleRoot);
        assertEq(rec.policyId, policyId);
        assertEq(rec.owner, client);
        assertEq(rec.shards.length, 3);
        assertEq(rec.shards[0].cid, cid0);
        assertEq(rec.shards[1].cid, cid1);
        assertEq(rec.shards[2].cid, cid2);
    }

    // --- internals -------------------------------------------------------

    function _deterministicBytes(bytes32 seed, uint8 len) internal pure returns (bytes memory out) {
        out = new bytes(len);
        bytes32 h = seed;
        for (uint256 i = 0; i < len; ++i) {
            if (i % 32 == 0 && i > 0) {
                h = keccak256(abi.encodePacked(h));
            }
            out[i] = h[i % 32];
        }
        // Avoid a leading zero byte since the contract only rejects length-0 CIDs,
        // but realistic multibase CIDs never start with 0x00.
        if (len > 0 && out[0] == 0) out[0] = 0x01;
    }
}
