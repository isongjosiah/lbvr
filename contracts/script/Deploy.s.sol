// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {Script, console2} from "forge-std/Script.sol";
import {CIDRegistry} from "../src/CIDRegistry.sol";
import {AuditorLog} from "../src/AuditorLog.sol";

/// @title Deploy
/// @notice Deployment script for CIDRegistry and AuditorLog on Polygon zkEVM
///         Cardona (chain id 2442). Both contracts share an admin (the deployer)
///         and the same admin-transfer delay so role rotation can be coordinated.
///
/// @dev Invocation:
///      forge script contracts/script/Deploy.s.sol:Deploy \
///          --rpc-url $CARDONA_RPC_URL --broadcast --verify \
///          --etherscan-api-key $POLYGONSCAN_API_KEY
///
/// @dev Required env:
///      CARDONA_PRIVATE_KEY  — deployer key (becomes default admin + MIGRATOR_ROLE
///                             on CIDRegistry and ANCHOR_ROLE on AuditorLog).
///      CARDONA_RPC_URL      — RPC endpoint (also read via foundry.toml [rpc_endpoints]).
///      POLYGONSCAN_API_KEY  — for --verify.
///
///      Optional:
///      ADMIN_TRANSFER_DELAY — default-admin handover delay (seconds); defaults to 3 days.
contract Deploy is Script {
    /// @dev Three days matches a reasonable testnet rotation window; tighten or relax by
    ///      setting ADMIN_TRANSFER_DELAY before broadcasting.
    uint48 internal constant DEFAULT_ADMIN_TRANSFER_DELAY = 3 days;

    function run() external returns (CIDRegistry registry, AuditorLog auditor) {
        uint256 pk = vm.envUint("CARDONA_PRIVATE_KEY");
        address admin = vm.addr(pk);
        uint48 delay = uint48(vm.envOr("ADMIN_TRANSFER_DELAY", uint256(DEFAULT_ADMIN_TRANSFER_DELAY)));

        vm.startBroadcast(pk);
        registry = new CIDRegistry(admin, delay);
        auditor = new AuditorLog(admin, delay);
        vm.stopBroadcast();

        console2.log("CIDRegistry deployed at:", address(registry));
        console2.log("AuditorLog  deployed at:", address(auditor));
        console2.log("Admin (CIDRegistry MIGRATOR_ROLE / AuditorLog ANCHOR_ROLE):", admin);
        console2.log("Admin transfer delay (s):", delay);
    }
}
