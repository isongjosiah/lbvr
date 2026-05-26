// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// CIDRegistryBundleRecord is an auto generated low-level Go binding around an user-defined struct.
type CIDRegistryBundleRecord struct {
	MerkleRoot     [32]byte
	NumChunks      uint32
	Shards         []CIDRegistryShardPlacement
	Owner          common.Address
	PolicyId       [32]byte
	RegisteredAt   uint64
	LastMigratedAt uint64
}

// CIDRegistryShardPlacement is an auto generated low-level Go binding around an user-defined struct.
type CIDRegistryShardPlacement struct {
	Cid  []byte
	Tier uint8
}

// ChainCIDRegistryMetaData contains all meta data concerning the ChainCIDRegistry contract.
var ChainCIDRegistryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"admin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"adminTransferDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"DEFAULT_ADMIN_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIGRATOR_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"acceptDefaultAdminTransfer\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"beginDefaultAdminTransfer\",\"inputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"cancelDefaultAdminTransfer\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"changeDefaultAdminDelay\",\"inputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"defaultAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defaultAdminDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defaultAdminDelayIncreaseWait\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRecord\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structCIDRegistry.BundleRecord\",\"components\":[{\"name\":\"merkleRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"numChunks\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"shards\",\"type\":\"tuple[]\",\"internalType\":\"structCIDRegistry.ShardPlacement[]\",\"components\":[{\"name\":\"cid\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"tier\",\"type\":\"uint8\",\"internalType\":\"enumCIDRegistry.TierClass\"}]},{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"policyId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"registeredAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"lastMigratedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRoleAdmin\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getShardLayout\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structCIDRegistry.ShardPlacement[]\",\"components\":[{\"name\":\"cid\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"tier\",\"type\":\"uint8\",\"internalType\":\"enumCIDRegistry.TierClass\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"grantRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"hasRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingDefaultAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingDefaultAdminDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"},{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerBundle\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"merkleRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"numChunks\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"shards\",\"type\":\"tuple[]\",\"internalType\":\"structCIDRegistry.ShardPlacement[]\",\"components\":[{\"name\":\"cid\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"tier\",\"type\":\"uint8\",\"internalType\":\"enumCIDRegistry.TierClass\"}]},{\"name\":\"policyId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revokeRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"rollbackDefaultAdminDelay\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"supportsInterface\",\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\",\"internalType\":\"bytes4\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updateShardLayout\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"newShards\",\"type\":\"tuple[]\",\"internalType\":\"structCIDRegistry.ShardPlacement[]\",\"components\":[{\"name\":\"cid\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"tier\",\"type\":\"uint8\",\"internalType\":\"enumCIDRegistry.TierClass\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"BundleRegistered\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"merkleRoot\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"policyId\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminDelayChangeCanceled\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminDelayChangeScheduled\",\"inputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"},{\"name\":\"effectSchedule\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminTransferCanceled\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminTransferScheduled\",\"inputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"acceptSchedule\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleAdminChanged\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"previousAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"newAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleGranted\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleRevoked\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ShardLayoutUpdated\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"oldShards\",\"type\":\"tuple[]\",\"indexed\":false,\"internalType\":\"structCIDRegistry.ShardPlacement[]\",\"components\":[{\"name\":\"cid\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"tier\",\"type\":\"uint8\",\"internalType\":\"enumCIDRegistry.TierClass\"}]},{\"name\":\"newShards\",\"type\":\"tuple[]\",\"indexed\":false,\"internalType\":\"structCIDRegistry.ShardPlacement[]\",\"components\":[{\"name\":\"cid\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"tier\",\"type\":\"uint8\",\"internalType\":\"enumCIDRegistry.TierClass\"}]}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AccessControlBadConfirmation\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AccessControlEnforcedDefaultAdminDelay\",\"inputs\":[{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}]},{\"type\":\"error\",\"name\":\"AccessControlEnforcedDefaultAdminRules\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AccessControlInvalidDefaultAdmin\",\"inputs\":[{\"name\":\"defaultAdmin\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AccessControlUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"neededRole\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"BundleAlreadyRegistered\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"BundleNotFound\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"EmptyShardCID\",\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"InvalidShardCount\",\"inputs\":[{\"name\":\"got\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"expected\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"NumChunksZero\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeCastOverflowedUintDowncast\",\"inputs\":[{\"name\":\"bits\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]",
}

// ChainCIDRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use ChainCIDRegistryMetaData.ABI instead.
var ChainCIDRegistryABI = ChainCIDRegistryMetaData.ABI

// ChainCIDRegistry is an auto generated Go binding around an Ethereum contract.
type ChainCIDRegistry struct {
	ChainCIDRegistryCaller     // Read-only binding to the contract
	ChainCIDRegistryTransactor // Write-only binding to the contract
	ChainCIDRegistryFilterer   // Log filterer for contract events
}

// ChainCIDRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChainCIDRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainCIDRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChainCIDRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainCIDRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChainCIDRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainCIDRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChainCIDRegistrySession struct {
	Contract     *ChainCIDRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ChainCIDRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChainCIDRegistryCallerSession struct {
	Contract *ChainCIDRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ChainCIDRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChainCIDRegistryTransactorSession struct {
	Contract     *ChainCIDRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ChainCIDRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChainCIDRegistryRaw struct {
	Contract *ChainCIDRegistry // Generic contract binding to access the raw methods on
}

// ChainCIDRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChainCIDRegistryCallerRaw struct {
	Contract *ChainCIDRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// ChainCIDRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChainCIDRegistryTransactorRaw struct {
	Contract *ChainCIDRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChainCIDRegistry creates a new instance of ChainCIDRegistry, bound to a specific deployed contract.
func NewChainCIDRegistry(address common.Address, backend bind.ContractBackend) (*ChainCIDRegistry, error) {
	contract, err := bindChainCIDRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistry{ChainCIDRegistryCaller: ChainCIDRegistryCaller{contract: contract}, ChainCIDRegistryTransactor: ChainCIDRegistryTransactor{contract: contract}, ChainCIDRegistryFilterer: ChainCIDRegistryFilterer{contract: contract}}, nil
}

// NewChainCIDRegistryCaller creates a new read-only instance of ChainCIDRegistry, bound to a specific deployed contract.
func NewChainCIDRegistryCaller(address common.Address, caller bind.ContractCaller) (*ChainCIDRegistryCaller, error) {
	contract, err := bindChainCIDRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryCaller{contract: contract}, nil
}

// NewChainCIDRegistryTransactor creates a new write-only instance of ChainCIDRegistry, bound to a specific deployed contract.
func NewChainCIDRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*ChainCIDRegistryTransactor, error) {
	contract, err := bindChainCIDRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryTransactor{contract: contract}, nil
}

// NewChainCIDRegistryFilterer creates a new log filterer instance of ChainCIDRegistry, bound to a specific deployed contract.
func NewChainCIDRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*ChainCIDRegistryFilterer, error) {
	contract, err := bindChainCIDRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryFilterer{contract: contract}, nil
}

// bindChainCIDRegistry binds a generic wrapper to an already deployed contract.
func bindChainCIDRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ChainCIDRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChainCIDRegistry *ChainCIDRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChainCIDRegistry.Contract.ChainCIDRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChainCIDRegistry *ChainCIDRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.ChainCIDRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChainCIDRegistry *ChainCIDRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.ChainCIDRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChainCIDRegistry *ChainCIDRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChainCIDRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChainCIDRegistry *ChainCIDRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChainCIDRegistry *ChainCIDRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistrySession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ChainCIDRegistry.Contract.DEFAULTADMINROLE(&_ChainCIDRegistry.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ChainCIDRegistry.Contract.DEFAULTADMINROLE(&_ChainCIDRegistry.CallOpts)
}

// MIGRATORROLE is a free data retrieval call binding the contract method 0x6fae2e15.
//
// Solidity: function MIGRATOR_ROLE() view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) MIGRATORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "MIGRATOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MIGRATORROLE is a free data retrieval call binding the contract method 0x6fae2e15.
//
// Solidity: function MIGRATOR_ROLE() view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistrySession) MIGRATORROLE() ([32]byte, error) {
	return _ChainCIDRegistry.Contract.MIGRATORROLE(&_ChainCIDRegistry.CallOpts)
}

// MIGRATORROLE is a free data retrieval call binding the contract method 0x6fae2e15.
//
// Solidity: function MIGRATOR_ROLE() view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) MIGRATORROLE() ([32]byte, error) {
	return _ChainCIDRegistry.Contract.MIGRATORROLE(&_ChainCIDRegistry.CallOpts)
}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) DefaultAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "defaultAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainCIDRegistry *ChainCIDRegistrySession) DefaultAdmin() (common.Address, error) {
	return _ChainCIDRegistry.Contract.DefaultAdmin(&_ChainCIDRegistry.CallOpts)
}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) DefaultAdmin() (common.Address, error) {
	return _ChainCIDRegistry.Contract.DefaultAdmin(&_ChainCIDRegistry.CallOpts)
}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) DefaultAdminDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "defaultAdminDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainCIDRegistry *ChainCIDRegistrySession) DefaultAdminDelay() (*big.Int, error) {
	return _ChainCIDRegistry.Contract.DefaultAdminDelay(&_ChainCIDRegistry.CallOpts)
}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) DefaultAdminDelay() (*big.Int, error) {
	return _ChainCIDRegistry.Contract.DefaultAdminDelay(&_ChainCIDRegistry.CallOpts)
}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) DefaultAdminDelayIncreaseWait(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "defaultAdminDelayIncreaseWait")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainCIDRegistry *ChainCIDRegistrySession) DefaultAdminDelayIncreaseWait() (*big.Int, error) {
	return _ChainCIDRegistry.Contract.DefaultAdminDelayIncreaseWait(&_ChainCIDRegistry.CallOpts)
}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) DefaultAdminDelayIncreaseWait() (*big.Int, error) {
	return _ChainCIDRegistry.Contract.DefaultAdminDelayIncreaseWait(&_ChainCIDRegistry.CallOpts)
}

// GetRecord is a free data retrieval call binding the contract method 0x213681cd.
//
// Solidity: function getRecord(bytes32 bundleId) view returns((bytes32,uint32,(bytes,uint8)[],address,bytes32,uint64,uint64))
func (_ChainCIDRegistry *ChainCIDRegistryCaller) GetRecord(opts *bind.CallOpts, bundleId [32]byte) (CIDRegistryBundleRecord, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "getRecord", bundleId)

	if err != nil {
		return *new(CIDRegistryBundleRecord), err
	}

	out0 := *abi.ConvertType(out[0], new(CIDRegistryBundleRecord)).(*CIDRegistryBundleRecord)

	return out0, err

}

// GetRecord is a free data retrieval call binding the contract method 0x213681cd.
//
// Solidity: function getRecord(bytes32 bundleId) view returns((bytes32,uint32,(bytes,uint8)[],address,bytes32,uint64,uint64))
func (_ChainCIDRegistry *ChainCIDRegistrySession) GetRecord(bundleId [32]byte) (CIDRegistryBundleRecord, error) {
	return _ChainCIDRegistry.Contract.GetRecord(&_ChainCIDRegistry.CallOpts, bundleId)
}

// GetRecord is a free data retrieval call binding the contract method 0x213681cd.
//
// Solidity: function getRecord(bytes32 bundleId) view returns((bytes32,uint32,(bytes,uint8)[],address,bytes32,uint64,uint64))
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) GetRecord(bundleId [32]byte) (CIDRegistryBundleRecord, error) {
	return _ChainCIDRegistry.Contract.GetRecord(&_ChainCIDRegistry.CallOpts, bundleId)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistrySession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ChainCIDRegistry.Contract.GetRoleAdmin(&_ChainCIDRegistry.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ChainCIDRegistry.Contract.GetRoleAdmin(&_ChainCIDRegistry.CallOpts, role)
}

// GetShardLayout is a free data retrieval call binding the contract method 0x036b192c.
//
// Solidity: function getShardLayout(bytes32 bundleId) view returns((bytes,uint8)[])
func (_ChainCIDRegistry *ChainCIDRegistryCaller) GetShardLayout(opts *bind.CallOpts, bundleId [32]byte) ([]CIDRegistryShardPlacement, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "getShardLayout", bundleId)

	if err != nil {
		return *new([]CIDRegistryShardPlacement), err
	}

	out0 := *abi.ConvertType(out[0], new([]CIDRegistryShardPlacement)).(*[]CIDRegistryShardPlacement)

	return out0, err

}

// GetShardLayout is a free data retrieval call binding the contract method 0x036b192c.
//
// Solidity: function getShardLayout(bytes32 bundleId) view returns((bytes,uint8)[])
func (_ChainCIDRegistry *ChainCIDRegistrySession) GetShardLayout(bundleId [32]byte) ([]CIDRegistryShardPlacement, error) {
	return _ChainCIDRegistry.Contract.GetShardLayout(&_ChainCIDRegistry.CallOpts, bundleId)
}

// GetShardLayout is a free data retrieval call binding the contract method 0x036b192c.
//
// Solidity: function getShardLayout(bytes32 bundleId) view returns((bytes,uint8)[])
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) GetShardLayout(bundleId [32]byte) ([]CIDRegistryShardPlacement, error) {
	return _ChainCIDRegistry.Contract.GetShardLayout(&_ChainCIDRegistry.CallOpts, bundleId)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainCIDRegistry *ChainCIDRegistrySession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ChainCIDRegistry.Contract.HasRole(&_ChainCIDRegistry.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ChainCIDRegistry.Contract.HasRole(&_ChainCIDRegistry.CallOpts, role, account)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainCIDRegistry *ChainCIDRegistrySession) Owner() (common.Address, error) {
	return _ChainCIDRegistry.Contract.Owner(&_ChainCIDRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) Owner() (common.Address, error) {
	return _ChainCIDRegistry.Contract.Owner(&_ChainCIDRegistry.CallOpts)
}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) PendingDefaultAdmin(opts *bind.CallOpts) (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "pendingDefaultAdmin")

	outstruct := new(struct {
		NewAdmin common.Address
		Schedule *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.NewAdmin = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Schedule = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainCIDRegistry *ChainCIDRegistrySession) PendingDefaultAdmin() (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	return _ChainCIDRegistry.Contract.PendingDefaultAdmin(&_ChainCIDRegistry.CallOpts)
}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) PendingDefaultAdmin() (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	return _ChainCIDRegistry.Contract.PendingDefaultAdmin(&_ChainCIDRegistry.CallOpts)
}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) PendingDefaultAdminDelay(opts *bind.CallOpts) (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "pendingDefaultAdminDelay")

	outstruct := new(struct {
		NewDelay *big.Int
		Schedule *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.NewDelay = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Schedule = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainCIDRegistry *ChainCIDRegistrySession) PendingDefaultAdminDelay() (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	return _ChainCIDRegistry.Contract.PendingDefaultAdminDelay(&_ChainCIDRegistry.CallOpts)
}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) PendingDefaultAdminDelay() (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	return _ChainCIDRegistry.Contract.PendingDefaultAdminDelay(&_ChainCIDRegistry.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainCIDRegistry *ChainCIDRegistryCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _ChainCIDRegistry.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainCIDRegistry *ChainCIDRegistrySession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ChainCIDRegistry.Contract.SupportsInterface(&_ChainCIDRegistry.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainCIDRegistry *ChainCIDRegistryCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ChainCIDRegistry.Contract.SupportsInterface(&_ChainCIDRegistry.CallOpts, interfaceId)
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) AcceptDefaultAdminTransfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "acceptDefaultAdminTransfer")
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) AcceptDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.AcceptDefaultAdminTransfer(&_ChainCIDRegistry.TransactOpts)
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) AcceptDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.AcceptDefaultAdminTransfer(&_ChainCIDRegistry.TransactOpts)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) BeginDefaultAdminTransfer(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "beginDefaultAdminTransfer", newAdmin)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) BeginDefaultAdminTransfer(newAdmin common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.BeginDefaultAdminTransfer(&_ChainCIDRegistry.TransactOpts, newAdmin)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) BeginDefaultAdminTransfer(newAdmin common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.BeginDefaultAdminTransfer(&_ChainCIDRegistry.TransactOpts, newAdmin)
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) CancelDefaultAdminTransfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "cancelDefaultAdminTransfer")
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) CancelDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.CancelDefaultAdminTransfer(&_ChainCIDRegistry.TransactOpts)
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) CancelDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.CancelDefaultAdminTransfer(&_ChainCIDRegistry.TransactOpts)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) ChangeDefaultAdminDelay(opts *bind.TransactOpts, newDelay *big.Int) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "changeDefaultAdminDelay", newDelay)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) ChangeDefaultAdminDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.ChangeDefaultAdminDelay(&_ChainCIDRegistry.TransactOpts, newDelay)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) ChangeDefaultAdminDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.ChangeDefaultAdminDelay(&_ChainCIDRegistry.TransactOpts, newDelay)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.GrantRole(&_ChainCIDRegistry.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.GrantRole(&_ChainCIDRegistry.TransactOpts, role, account)
}

// RegisterBundle is a paid mutator transaction binding the contract method 0xf2c0d077.
//
// Solidity: function registerBundle(bytes32 bundleId, bytes32 merkleRoot, uint32 numChunks, (bytes,uint8)[] shards, bytes32 policyId) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) RegisterBundle(opts *bind.TransactOpts, bundleId [32]byte, merkleRoot [32]byte, numChunks uint32, shards []CIDRegistryShardPlacement, policyId [32]byte) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "registerBundle", bundleId, merkleRoot, numChunks, shards, policyId)
}

// RegisterBundle is a paid mutator transaction binding the contract method 0xf2c0d077.
//
// Solidity: function registerBundle(bytes32 bundleId, bytes32 merkleRoot, uint32 numChunks, (bytes,uint8)[] shards, bytes32 policyId) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) RegisterBundle(bundleId [32]byte, merkleRoot [32]byte, numChunks uint32, shards []CIDRegistryShardPlacement, policyId [32]byte) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RegisterBundle(&_ChainCIDRegistry.TransactOpts, bundleId, merkleRoot, numChunks, shards, policyId)
}

// RegisterBundle is a paid mutator transaction binding the contract method 0xf2c0d077.
//
// Solidity: function registerBundle(bytes32 bundleId, bytes32 merkleRoot, uint32 numChunks, (bytes,uint8)[] shards, bytes32 policyId) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) RegisterBundle(bundleId [32]byte, merkleRoot [32]byte, numChunks uint32, shards []CIDRegistryShardPlacement, policyId [32]byte) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RegisterBundle(&_ChainCIDRegistry.TransactOpts, bundleId, merkleRoot, numChunks, shards, policyId)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RenounceRole(&_ChainCIDRegistry.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RenounceRole(&_ChainCIDRegistry.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RevokeRole(&_ChainCIDRegistry.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RevokeRole(&_ChainCIDRegistry.TransactOpts, role, account)
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) RollbackDefaultAdminDelay(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "rollbackDefaultAdminDelay")
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) RollbackDefaultAdminDelay() (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RollbackDefaultAdminDelay(&_ChainCIDRegistry.TransactOpts)
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) RollbackDefaultAdminDelay() (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.RollbackDefaultAdminDelay(&_ChainCIDRegistry.TransactOpts)
}

// UpdateShardLayout is a paid mutator transaction binding the contract method 0xab132ad1.
//
// Solidity: function updateShardLayout(bytes32 bundleId, (bytes,uint8)[] newShards) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactor) UpdateShardLayout(opts *bind.TransactOpts, bundleId [32]byte, newShards []CIDRegistryShardPlacement) (*types.Transaction, error) {
	return _ChainCIDRegistry.contract.Transact(opts, "updateShardLayout", bundleId, newShards)
}

// UpdateShardLayout is a paid mutator transaction binding the contract method 0xab132ad1.
//
// Solidity: function updateShardLayout(bytes32 bundleId, (bytes,uint8)[] newShards) returns()
func (_ChainCIDRegistry *ChainCIDRegistrySession) UpdateShardLayout(bundleId [32]byte, newShards []CIDRegistryShardPlacement) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.UpdateShardLayout(&_ChainCIDRegistry.TransactOpts, bundleId, newShards)
}

// UpdateShardLayout is a paid mutator transaction binding the contract method 0xab132ad1.
//
// Solidity: function updateShardLayout(bytes32 bundleId, (bytes,uint8)[] newShards) returns()
func (_ChainCIDRegistry *ChainCIDRegistryTransactorSession) UpdateShardLayout(bundleId [32]byte, newShards []CIDRegistryShardPlacement) (*types.Transaction, error) {
	return _ChainCIDRegistry.Contract.UpdateShardLayout(&_ChainCIDRegistry.TransactOpts, bundleId, newShards)
}

// ChainCIDRegistryBundleRegisteredIterator is returned from FilterBundleRegistered and is used to iterate over the raw logs and unpacked data for BundleRegistered events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryBundleRegisteredIterator struct {
	Event *ChainCIDRegistryBundleRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryBundleRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryBundleRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryBundleRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryBundleRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryBundleRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryBundleRegistered represents a BundleRegistered event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryBundleRegistered struct {
	BundleId   [32]byte
	MerkleRoot [32]byte
	Owner      common.Address
	PolicyId   [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterBundleRegistered is a free log retrieval operation binding the contract event 0x201d27b97a9e53cdd5e730131936f465d0b219e1b8abad5cdae296a34f74209b.
//
// Solidity: event BundleRegistered(bytes32 indexed bundleId, bytes32 indexed merkleRoot, address indexed owner, bytes32 policyId)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterBundleRegistered(opts *bind.FilterOpts, bundleId [][32]byte, merkleRoot [][32]byte, owner []common.Address) (*ChainCIDRegistryBundleRegisteredIterator, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var merkleRootRule []interface{}
	for _, merkleRootItem := range merkleRoot {
		merkleRootRule = append(merkleRootRule, merkleRootItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "BundleRegistered", bundleIdRule, merkleRootRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryBundleRegisteredIterator{contract: _ChainCIDRegistry.contract, event: "BundleRegistered", logs: logs, sub: sub}, nil
}

// WatchBundleRegistered is a free log subscription operation binding the contract event 0x201d27b97a9e53cdd5e730131936f465d0b219e1b8abad5cdae296a34f74209b.
//
// Solidity: event BundleRegistered(bytes32 indexed bundleId, bytes32 indexed merkleRoot, address indexed owner, bytes32 policyId)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchBundleRegistered(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryBundleRegistered, bundleId [][32]byte, merkleRoot [][32]byte, owner []common.Address) (event.Subscription, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var merkleRootRule []interface{}
	for _, merkleRootItem := range merkleRoot {
		merkleRootRule = append(merkleRootRule, merkleRootItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "BundleRegistered", bundleIdRule, merkleRootRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryBundleRegistered)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "BundleRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBundleRegistered is a log parse operation binding the contract event 0x201d27b97a9e53cdd5e730131936f465d0b219e1b8abad5cdae296a34f74209b.
//
// Solidity: event BundleRegistered(bytes32 indexed bundleId, bytes32 indexed merkleRoot, address indexed owner, bytes32 policyId)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseBundleRegistered(log types.Log) (*ChainCIDRegistryBundleRegistered, error) {
	event := new(ChainCIDRegistryBundleRegistered)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "BundleRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator is returned from FilterDefaultAdminDelayChangeCanceled and is used to iterate over the raw logs and unpacked data for DefaultAdminDelayChangeCanceled events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator struct {
	Event *ChainCIDRegistryDefaultAdminDelayChangeCanceled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryDefaultAdminDelayChangeCanceled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryDefaultAdminDelayChangeCanceled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryDefaultAdminDelayChangeCanceled represents a DefaultAdminDelayChangeCanceled event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminDelayChangeCanceled struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminDelayChangeCanceled is a free log retrieval operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterDefaultAdminDelayChangeCanceled(opts *bind.FilterOpts) (*ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator, error) {

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "DefaultAdminDelayChangeCanceled")
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryDefaultAdminDelayChangeCanceledIterator{contract: _ChainCIDRegistry.contract, event: "DefaultAdminDelayChangeCanceled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminDelayChangeCanceled is a free log subscription operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchDefaultAdminDelayChangeCanceled(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryDefaultAdminDelayChangeCanceled) (event.Subscription, error) {

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "DefaultAdminDelayChangeCanceled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryDefaultAdminDelayChangeCanceled)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminDelayChangeCanceled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDefaultAdminDelayChangeCanceled is a log parse operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseDefaultAdminDelayChangeCanceled(log types.Log) (*ChainCIDRegistryDefaultAdminDelayChangeCanceled, error) {
	event := new(ChainCIDRegistryDefaultAdminDelayChangeCanceled)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminDelayChangeCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator is returned from FilterDefaultAdminDelayChangeScheduled and is used to iterate over the raw logs and unpacked data for DefaultAdminDelayChangeScheduled events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator struct {
	Event *ChainCIDRegistryDefaultAdminDelayChangeScheduled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryDefaultAdminDelayChangeScheduled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryDefaultAdminDelayChangeScheduled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryDefaultAdminDelayChangeScheduled represents a DefaultAdminDelayChangeScheduled event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminDelayChangeScheduled struct {
	NewDelay       *big.Int
	EffectSchedule *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminDelayChangeScheduled is a free log retrieval operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterDefaultAdminDelayChangeScheduled(opts *bind.FilterOpts) (*ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator, error) {

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "DefaultAdminDelayChangeScheduled")
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryDefaultAdminDelayChangeScheduledIterator{contract: _ChainCIDRegistry.contract, event: "DefaultAdminDelayChangeScheduled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminDelayChangeScheduled is a free log subscription operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchDefaultAdminDelayChangeScheduled(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryDefaultAdminDelayChangeScheduled) (event.Subscription, error) {

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "DefaultAdminDelayChangeScheduled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryDefaultAdminDelayChangeScheduled)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminDelayChangeScheduled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDefaultAdminDelayChangeScheduled is a log parse operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseDefaultAdminDelayChangeScheduled(log types.Log) (*ChainCIDRegistryDefaultAdminDelayChangeScheduled, error) {
	event := new(ChainCIDRegistryDefaultAdminDelayChangeScheduled)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminDelayChangeScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryDefaultAdminTransferCanceledIterator is returned from FilterDefaultAdminTransferCanceled and is used to iterate over the raw logs and unpacked data for DefaultAdminTransferCanceled events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminTransferCanceledIterator struct {
	Event *ChainCIDRegistryDefaultAdminTransferCanceled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryDefaultAdminTransferCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryDefaultAdminTransferCanceled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryDefaultAdminTransferCanceled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryDefaultAdminTransferCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryDefaultAdminTransferCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryDefaultAdminTransferCanceled represents a DefaultAdminTransferCanceled event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminTransferCanceled struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminTransferCanceled is a free log retrieval operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterDefaultAdminTransferCanceled(opts *bind.FilterOpts) (*ChainCIDRegistryDefaultAdminTransferCanceledIterator, error) {

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "DefaultAdminTransferCanceled")
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryDefaultAdminTransferCanceledIterator{contract: _ChainCIDRegistry.contract, event: "DefaultAdminTransferCanceled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminTransferCanceled is a free log subscription operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchDefaultAdminTransferCanceled(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryDefaultAdminTransferCanceled) (event.Subscription, error) {

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "DefaultAdminTransferCanceled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryDefaultAdminTransferCanceled)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminTransferCanceled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDefaultAdminTransferCanceled is a log parse operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseDefaultAdminTransferCanceled(log types.Log) (*ChainCIDRegistryDefaultAdminTransferCanceled, error) {
	event := new(ChainCIDRegistryDefaultAdminTransferCanceled)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminTransferCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryDefaultAdminTransferScheduledIterator is returned from FilterDefaultAdminTransferScheduled and is used to iterate over the raw logs and unpacked data for DefaultAdminTransferScheduled events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminTransferScheduledIterator struct {
	Event *ChainCIDRegistryDefaultAdminTransferScheduled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryDefaultAdminTransferScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryDefaultAdminTransferScheduled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryDefaultAdminTransferScheduled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryDefaultAdminTransferScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryDefaultAdminTransferScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryDefaultAdminTransferScheduled represents a DefaultAdminTransferScheduled event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryDefaultAdminTransferScheduled struct {
	NewAdmin       common.Address
	AcceptSchedule *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminTransferScheduled is a free log retrieval operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterDefaultAdminTransferScheduled(opts *bind.FilterOpts, newAdmin []common.Address) (*ChainCIDRegistryDefaultAdminTransferScheduledIterator, error) {

	var newAdminRule []interface{}
	for _, newAdminItem := range newAdmin {
		newAdminRule = append(newAdminRule, newAdminItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "DefaultAdminTransferScheduled", newAdminRule)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryDefaultAdminTransferScheduledIterator{contract: _ChainCIDRegistry.contract, event: "DefaultAdminTransferScheduled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminTransferScheduled is a free log subscription operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchDefaultAdminTransferScheduled(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryDefaultAdminTransferScheduled, newAdmin []common.Address) (event.Subscription, error) {

	var newAdminRule []interface{}
	for _, newAdminItem := range newAdmin {
		newAdminRule = append(newAdminRule, newAdminItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "DefaultAdminTransferScheduled", newAdminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryDefaultAdminTransferScheduled)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminTransferScheduled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDefaultAdminTransferScheduled is a log parse operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseDefaultAdminTransferScheduled(log types.Log) (*ChainCIDRegistryDefaultAdminTransferScheduled, error) {
	event := new(ChainCIDRegistryDefaultAdminTransferScheduled)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "DefaultAdminTransferScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryRoleAdminChangedIterator struct {
	Event *ChainCIDRegistryRoleAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryRoleAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryRoleAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryRoleAdminChanged represents a RoleAdminChanged event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*ChainCIDRegistryRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryRoleAdminChangedIterator{contract: _ChainCIDRegistry.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryRoleAdminChanged)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseRoleAdminChanged(log types.Log) (*ChainCIDRegistryRoleAdminChanged, error) {
	event := new(ChainCIDRegistryRoleAdminChanged)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryRoleGrantedIterator struct {
	Event *ChainCIDRegistryRoleGranted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryRoleGranted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryRoleGranted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryRoleGranted represents a RoleGranted event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ChainCIDRegistryRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryRoleGrantedIterator{contract: _ChainCIDRegistry.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryRoleGranted)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "RoleGranted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseRoleGranted(log types.Log) (*ChainCIDRegistryRoleGranted, error) {
	event := new(ChainCIDRegistryRoleGranted)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryRoleRevokedIterator struct {
	Event *ChainCIDRegistryRoleRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryRoleRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryRoleRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryRoleRevoked represents a RoleRevoked event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ChainCIDRegistryRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryRoleRevokedIterator{contract: _ChainCIDRegistry.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryRoleRevoked)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseRoleRevoked(log types.Log) (*ChainCIDRegistryRoleRevoked, error) {
	event := new(ChainCIDRegistryRoleRevoked)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainCIDRegistryShardLayoutUpdatedIterator is returned from FilterShardLayoutUpdated and is used to iterate over the raw logs and unpacked data for ShardLayoutUpdated events raised by the ChainCIDRegistry contract.
type ChainCIDRegistryShardLayoutUpdatedIterator struct {
	Event *ChainCIDRegistryShardLayoutUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChainCIDRegistryShardLayoutUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainCIDRegistryShardLayoutUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChainCIDRegistryShardLayoutUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChainCIDRegistryShardLayoutUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainCIDRegistryShardLayoutUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainCIDRegistryShardLayoutUpdated represents a ShardLayoutUpdated event raised by the ChainCIDRegistry contract.
type ChainCIDRegistryShardLayoutUpdated struct {
	BundleId  [32]byte
	OldShards []CIDRegistryShardPlacement
	NewShards []CIDRegistryShardPlacement
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterShardLayoutUpdated is a free log retrieval operation binding the contract event 0x954ff2e772d5aea6ea580cfdbbb2bb91d01f02bfcc3f2604ed15d72a8935e08a.
//
// Solidity: event ShardLayoutUpdated(bytes32 indexed bundleId, (bytes,uint8)[] oldShards, (bytes,uint8)[] newShards)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) FilterShardLayoutUpdated(opts *bind.FilterOpts, bundleId [][32]byte) (*ChainCIDRegistryShardLayoutUpdatedIterator, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.FilterLogs(opts, "ShardLayoutUpdated", bundleIdRule)
	if err != nil {
		return nil, err
	}
	return &ChainCIDRegistryShardLayoutUpdatedIterator{contract: _ChainCIDRegistry.contract, event: "ShardLayoutUpdated", logs: logs, sub: sub}, nil
}

// WatchShardLayoutUpdated is a free log subscription operation binding the contract event 0x954ff2e772d5aea6ea580cfdbbb2bb91d01f02bfcc3f2604ed15d72a8935e08a.
//
// Solidity: event ShardLayoutUpdated(bytes32 indexed bundleId, (bytes,uint8)[] oldShards, (bytes,uint8)[] newShards)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) WatchShardLayoutUpdated(opts *bind.WatchOpts, sink chan<- *ChainCIDRegistryShardLayoutUpdated, bundleId [][32]byte) (event.Subscription, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}

	logs, sub, err := _ChainCIDRegistry.contract.WatchLogs(opts, "ShardLayoutUpdated", bundleIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainCIDRegistryShardLayoutUpdated)
				if err := _ChainCIDRegistry.contract.UnpackLog(event, "ShardLayoutUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseShardLayoutUpdated is a log parse operation binding the contract event 0x954ff2e772d5aea6ea580cfdbbb2bb91d01f02bfcc3f2604ed15d72a8935e08a.
//
// Solidity: event ShardLayoutUpdated(bytes32 indexed bundleId, (bytes,uint8)[] oldShards, (bytes,uint8)[] newShards)
func (_ChainCIDRegistry *ChainCIDRegistryFilterer) ParseShardLayoutUpdated(log types.Log) (*ChainCIDRegistryShardLayoutUpdated, error) {
	event := new(ChainCIDRegistryShardLayoutUpdated)
	if err := _ChainCIDRegistry.contract.UnpackLog(event, "ShardLayoutUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
