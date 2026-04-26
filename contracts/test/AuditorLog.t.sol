// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {Test} from "forge-std/Test.sol";
import {AuditorLog} from "../src/AuditorLog.sol";
import {IAccessControl} from "@openzeppelin/contracts/access/IAccessControl.sol";

contract AuditorLogTest is Test {
    AuditorLog internal auditor;

    address internal admin = address(0xA11CE);
    address internal stranger = address(0xDEAD);
    address internal anchorer = address(0xC0DE);

    bytes32 internal constant BUNDLE_ID = keccak256("lbvr://bundle/abc123");
    bytes32 internal constant RETRIEVAL_ID = keccak256("lbvr://retrieval/xyz789");
    bytes32 internal constant PROV_HASH = keccak256("jcs-canonical-prov-json-v1");

    string internal constant DID_GW1 = "did:lbvr:gw-1";
    string internal constant DID_GW2 = "did:lbvr:gw-2";

    // BLS12-381 G1 compressed = 48 bytes (docs/provenance-spec.md §5.1).
    bytes internal constant PUBKEY_G1 =
        hex"a0b1c2d3e4f5061728394a5b6c7d8e9f0a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4f5061728394a5b6c7d8e9f";
    // Rotation target: distinguishable shape, same length.
    bytes internal constant PUBKEY_G1_ROTATED =
        hex"f9e8d7c6b5a49382716f5e4d3c2b1a09f8e7d6c5b4a39281706f5e4d3c2b1a09f8e7d6c5b4a39281706f5e4d3c2b1a09";
    // Future-version G2 compressed = 96 bytes (sequential 0x00..0x5f for shape only).
    bytes internal constant PUBKEY_G2 = hex"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f"
        hex"303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f";

    event ProvenanceAnchored(
        bytes32 indexed bundleId, bytes32 indexed retrievalId, bytes32 provHash, address indexed anchoredBy
    );
    event GatewayKeyRegistered(bytes32 indexed didHash, bytes publicKey);

    function setUp() public {
        vm.prank(admin);
        auditor = new AuditorLog(admin, 0);
    }

    // --- happy path: anchor ---------------------------------------------

    function test_anchorProvenance_happyPath() public {
        vm.expectEmit(true, true, true, true, address(auditor));
        emit ProvenanceAnchored(BUNDLE_ID, RETRIEVAL_ID, PROV_HASH, admin);

        vm.prank(admin);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, PROV_HASH);

        AuditorLog.ProvenanceAnchor memory a = auditor.getProvenanceAnchor(BUNDLE_ID, RETRIEVAL_ID);
        assertEq(a.provHash, PROV_HASH, "provHash");
        assertEq(a.blockNumber, block.number, "blockNumber");
        assertEq(a.timestamp, block.timestamp, "timestamp");
        assertEq(a.anchoredBy, admin, "anchoredBy");
    }

    // --- anchor reverts --------------------------------------------------

    function test_anchorProvenance_revertsOnDuplicate() public {
        vm.prank(admin);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, PROV_HASH);

        vm.expectRevert(abi.encodeWithSelector(AuditorLog.AlreadyAnchored.selector, BUNDLE_ID, RETRIEVAL_ID));
        vm.prank(admin);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, keccak256("attempted-overwrite"));
    }

    function test_anchorProvenance_revertsOnZeroProvHash() public {
        vm.expectRevert(AuditorLog.EmptyProvHash.selector);
        vm.prank(admin);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, bytes32(0));
    }

    function test_anchorProvenance_revertsWithoutRole() public {
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, auditor.ANCHOR_ROLE()
            )
        );
        vm.prank(stranger);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, PROV_HASH);
    }

    /// @dev Distinct (bundleId, retrievalId) pairs must be independently anchorable
    ///      — i.e. anchoring under one bundle does not pollute another's slot.
    function test_anchorProvenance_distinctPairsCoexist() public {
        bytes32 otherRetrieval = keccak256("lbvr://retrieval/other");
        bytes32 otherBundle = keccak256("lbvr://bundle/other");
        bytes32 otherHash = keccak256("other-prov");

        vm.startPrank(admin);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, PROV_HASH);
        auditor.anchorProvenance(BUNDLE_ID, otherRetrieval, otherHash);
        auditor.anchorProvenance(otherBundle, RETRIEVAL_ID, otherHash);
        vm.stopPrank();

        assertEq(auditor.getProvenanceAnchor(BUNDLE_ID, RETRIEVAL_ID).provHash, PROV_HASH);
        assertEq(auditor.getProvenanceAnchor(BUNDLE_ID, otherRetrieval).provHash, otherHash);
        assertEq(auditor.getProvenanceAnchor(otherBundle, RETRIEVAL_ID).provHash, otherHash);
    }

    // --- read reverts ----------------------------------------------------

    function test_getProvenanceAnchor_revertsIfMissing() public {
        vm.expectRevert(abi.encodeWithSelector(AuditorLog.AnchorNotFound.selector, BUNDLE_ID, RETRIEVAL_ID));
        auditor.getProvenanceAnchor(BUNDLE_ID, RETRIEVAL_ID);
    }

    // --- gateway key registration: happy path ---------------------------

    function test_registerGatewayKey_happyPath() public {
        bytes32 didHash = keccak256(bytes(DID_GW1));

        vm.expectEmit(true, false, false, true, address(auditor));
        emit GatewayKeyRegistered(didHash, PUBKEY_G1);

        vm.prank(admin);
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G1);

        assertEq(auditor.getGatewayKey(DID_GW1), PUBKEY_G1, "stored G1 key");
    }

    function test_getGatewayKey_returnsEmptyForUnknownDID() public view {
        assertEq(auditor.getGatewayKey("did:lbvr:never-registered").length, 0);
    }

    // --- gateway key registration: rotation -----------------------------

    /// @dev Re-registration must overwrite (key rotation is supported per the
    ///      contract's natspec) and must emit GatewayKeyRegistered each time so
    ///      verifiers can replay history.
    function test_registerGatewayKey_rotationOverwritesAndReEmits() public {
        bytes32 didHash = keccak256(bytes(DID_GW1));

        vm.expectEmit(true, false, false, true, address(auditor));
        emit GatewayKeyRegistered(didHash, PUBKEY_G1);
        vm.prank(admin);
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G1);

        vm.expectEmit(true, false, false, true, address(auditor));
        emit GatewayKeyRegistered(didHash, PUBKEY_G1_ROTATED);
        vm.prank(admin);
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G1_ROTATED);

        assertEq(auditor.getGatewayKey(DID_GW1), PUBKEY_G1_ROTATED, "rotated key returned");
    }

    function test_registerGatewayKey_distinctDIDsAreIndependent() public {
        vm.startPrank(admin);
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G1);
        auditor.registerGatewayKey(DID_GW2, PUBKEY_G1_ROTATED);
        vm.stopPrank();

        assertEq(auditor.getGatewayKey(DID_GW1), PUBKEY_G1);
        assertEq(auditor.getGatewayKey(DID_GW2), PUBKEY_G1_ROTATED);
    }

    // --- gateway key registration: reverts ------------------------------

    function test_registerGatewayKey_revertsOnEmptyDID() public {
        vm.expectRevert(AuditorLog.EmptyDID.selector);
        vm.prank(admin);
        auditor.registerGatewayKey("", PUBKEY_G1);
    }

    function test_registerGatewayKey_revertsOnEmptyPublicKey() public {
        vm.expectRevert(AuditorLog.EmptyPublicKey.selector);
        vm.prank(admin);
        auditor.registerGatewayKey(DID_GW1, bytes(""));
    }

    function test_registerGatewayKey_revertsFromNonAdmin() public {
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, auditor.DEFAULT_ADMIN_ROLE()
            )
        );
        vm.prank(stranger);
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G1);
    }

    // --- role admin ------------------------------------------------------

    function test_adminCanGrantAndRevokeAnchorRole() public {
        bytes32 role = auditor.ANCHOR_ROLE();

        vm.prank(admin);
        auditor.grantRole(role, anchorer);
        assertTrue(auditor.hasRole(role, anchorer));

        vm.prank(anchorer);
        auditor.anchorProvenance(BUNDLE_ID, RETRIEVAL_ID, PROV_HASH);
        assertEq(auditor.getProvenanceAnchor(BUNDLE_ID, RETRIEVAL_ID).anchoredBy, anchorer);

        vm.prank(admin);
        auditor.revokeRole(role, anchorer);
        assertFalse(auditor.hasRole(role, anchorer));

        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, anchorer, role)
        );
        vm.prank(anchorer);
        auditor.anchorProvenance(BUNDLE_ID, keccak256("retrieval/post-revoke"), PROV_HASH);
    }

    function test_nonAdminCannotGrantRole() public {
        bytes32 role = auditor.ANCHOR_ROLE();
        vm.expectRevert(
            abi.encodeWithSelector(
                IAccessControl.AccessControlUnauthorizedAccount.selector, stranger, auditor.DEFAULT_ADMIN_ROLE()
            )
        );
        vm.prank(stranger);
        auditor.grantRole(role, stranger);
    }

    // --- fuzz ------------------------------------------------------------

    /// @dev Round-trip arbitrary (bundleId, retrievalId, provHash) triples through
    ///      anchorProvenance + getProvenanceAnchor. Skips provHash == 0 (sentinel)
    ///      because it's covered by the dedicated revert test.
    function testFuzz_anchorProvenance_roundtrip(bytes32 bundleId, bytes32 retrievalId, bytes32 provHash) public {
        vm.assume(provHash != bytes32(0));

        vm.expectEmit(true, true, true, true, address(auditor));
        emit ProvenanceAnchored(bundleId, retrievalId, provHash, admin);

        vm.prank(admin);
        auditor.anchorProvenance(bundleId, retrievalId, provHash);

        AuditorLog.ProvenanceAnchor memory a = auditor.getProvenanceAnchor(bundleId, retrievalId);
        assertEq(a.provHash, provHash);
        assertEq(a.blockNumber, block.number);
        assertEq(a.timestamp, block.timestamp);
        assertEq(a.anchoredBy, admin);
    }

    // --- gas snapshots ---------------------------------------------------

    /// @dev Mirrors CIDRegistry's `testFuzz_registerBundle_gasSnapshot` pattern.
    ///      No ceiling asserted; E5 / E-PROV measure the actual cost
    ///      (docs/provenance-spec.md §6.2 targets ~50k gas). Logs let
    ///      `forge snapshot` and CI gas reports latch onto a live datapoint.
    function testFuzz_anchorProvenance_gasSnapshot(bytes32 bundleId, bytes32 retrievalId, bytes32 provHash) public {
        vm.assume(provHash != bytes32(0));

        vm.prank(admin);
        uint256 gasBefore = gasleft();
        auditor.anchorProvenance(bundleId, retrievalId, provHash);
        uint256 gasUsed = gasBefore - gasleft();

        emit log_named_uint("anchorProvenance.gasUsed", gasUsed);
    }

    /// @dev Gas for registering a 48-byte G1 compressed pubkey (current scheme).
    function test_registerGatewayKey_gasSnapshot_G1() public {
        vm.prank(admin);
        uint256 gasBefore = gasleft();
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G1);
        uint256 gasUsed = gasBefore - gasleft();

        emit log_named_uint("registerGatewayKey.G1.gasUsed", gasUsed);
        emit log_named_uint("registerGatewayKey.G1.keyLen", PUBKEY_G1.length);
    }

    /// @dev Gas for the future 96-byte G2 compressed pubkey shape so we can
    ///      project the journal-extension cost without re-deploying.
    function test_registerGatewayKey_gasSnapshot_G2() public {
        vm.prank(admin);
        uint256 gasBefore = gasleft();
        auditor.registerGatewayKey(DID_GW1, PUBKEY_G2);
        uint256 gasUsed = gasBefore - gasleft();

        emit log_named_uint("registerGatewayKey.G2.gasUsed", gasUsed);
        emit log_named_uint("registerGatewayKey.G2.keyLen", PUBKEY_G2.length);
    }
}
