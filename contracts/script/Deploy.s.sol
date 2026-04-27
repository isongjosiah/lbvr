// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import {Script, console2} from "forge-std/Script.sol";
import {CIDRegistry} from "../src/CIDRegistry.sol";
import {AuditorLog} from "../src/AuditorLog.sol";
import {PoRVerifier} from "../src/PoRVerifier.sol";

/// @title Deploy
/// @notice Deployment script for CIDRegistry, AuditorLog, and PoRVerifier on
///         Polygon zkEVM Cardona (chain id 2442). All three contracts share an
///         admin (the deployer) and the same admin-transfer delay so role
///         rotation can be coordinated. Deployment order matters: PoRVerifier
///         takes the CIDRegistry address as a constructor arg (it reads
///         per-bundle Merkle params on every response), so the registry must
///         be deployed first.
///
/// @dev Invocation:
///      forge script contracts/script/Deploy.s.sol:Deploy \
///          --rpc-url $CARDONA_RPC_URL --broadcast --verify \
///          --etherscan-api-key $POLYGONSCAN_API_KEY
///
/// @dev Required env:
///      CARDONA_PRIVATE_KEY  — deployer key (becomes default admin +
///                             MIGRATOR_ROLE on CIDRegistry, ANCHOR_ROLE on
///                             AuditorLog, AUDITOR_ROLE + RESPONDER_ROLE on
///                             PoRVerifier; revoke responder/auditor on the
///                             deployer key before promoting the gateway).
///      CARDONA_RPC_URL      — RPC endpoint (also read via foundry.toml [rpc_endpoints]).
///      POLYGONSCAN_API_KEY  — for --verify.
///
///      Optional:
///      ADMIN_TRANSFER_DELAY — default-admin handover delay (seconds); defaults to 3 days.
contract Deploy is Script {
    /// @dev Three days matches a reasonable testnet rotation window; tighten or relax by
    ///      setting ADMIN_TRANSFER_DELAY before broadcasting.
    uint48 internal constant DEFAULT_ADMIN_TRANSFER_DELAY = 3 days;

    function run() external returns (CIDRegistry registry, AuditorLog auditor, PoRVerifier por) {
        uint256 pk = vm.envUint("CARDONA_PRIVATE_KEY");
        address admin = vm.addr(pk);
        uint48 delay = uint48(vm.envOr("ADMIN_TRANSFER_DELAY", uint256(DEFAULT_ADMIN_TRANSFER_DELAY)));

        vm.startBroadcast(pk);
        registry = new CIDRegistry(admin, delay);
        auditor = new AuditorLog(admin, delay);
        // PoRVerifier reads CIDRegistry on every response; constructor binds the
        // immutable registry address. TODO(D12+): once the auditor daemon is
        // running on a dedicated key, grant CIDRegistry.MIGRATOR_ROLE to the
        // auditor and revoke it from the deployer so ShardMigrationRequired
        // events can be acted on without the deployer key in the loop.
        por = new PoRVerifier(admin, delay, address(registry));
        vm.stopBroadcast();

        console2.log("CIDRegistry deployed at:", address(registry));
        console2.log("AuditorLog  deployed at:", address(auditor));
        console2.log("PoRVerifier deployed at:", address(por));
        console2.log("Admin (CIDRegistry MIGRATOR_ROLE / AuditorLog ANCHOR_ROLE / PoR roles):", admin);
        console2.log("Admin transfer delay (s):", delay);
    }
}
