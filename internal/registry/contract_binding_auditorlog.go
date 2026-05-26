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

// AuditorLogProvenanceAnchor is an auto generated low-level Go binding around an user-defined struct.
type AuditorLogProvenanceAnchor struct {
	ProvHash    [32]byte
	BlockNumber *big.Int
	Timestamp   *big.Int
	AnchoredBy  common.Address
}

// ChainAuditorLogMetaData contains all meta data concerning the ChainAuditorLog contract.
var ChainAuditorLogMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"admin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"adminTransferDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"ANCHOR_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"DEFAULT_ADMIN_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"acceptDefaultAdminTransfer\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"anchorProvenance\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"retrievalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"provHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"beginDefaultAdminTransfer\",\"inputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"cancelDefaultAdminTransfer\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"changeDefaultAdminDelay\",\"inputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"defaultAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defaultAdminDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defaultAdminDelayIncreaseWait\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getGatewayKey\",\"inputs\":[{\"name\":\"did\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getProvenanceAnchor\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"retrievalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structAuditorLog.ProvenanceAnchor\",\"components\":[{\"name\":\"provHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"blockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"anchoredBy\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRoleAdmin\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"grantRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"hasRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingDefaultAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingDefaultAdminDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"},{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerGatewayKey\",\"inputs\":[{\"name\":\"did\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"publicKey\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revokeRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"rollbackDefaultAdminDelay\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"supportsInterface\",\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\",\"internalType\":\"bytes4\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"DefaultAdminDelayChangeCanceled\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminDelayChangeScheduled\",\"inputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"},{\"name\":\"effectSchedule\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminTransferCanceled\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminTransferScheduled\",\"inputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"acceptSchedule\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"GatewayKeyRegistered\",\"inputs\":[{\"name\":\"didHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"publicKey\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ProvenanceAnchored\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"retrievalId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"provHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"anchoredBy\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleAdminChanged\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"previousAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"newAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleGranted\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleRevoked\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AccessControlBadConfirmation\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AccessControlEnforcedDefaultAdminDelay\",\"inputs\":[{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}]},{\"type\":\"error\",\"name\":\"AccessControlEnforcedDefaultAdminRules\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AccessControlInvalidDefaultAdmin\",\"inputs\":[{\"name\":\"defaultAdmin\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AccessControlUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"neededRole\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"AlreadyAnchored\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"retrievalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"AnchorNotFound\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"retrievalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"EmptyDID\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EmptyProvHash\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EmptyPublicKey\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeCastOverflowedUintDowncast\",\"inputs\":[{\"name\":\"bits\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]",
}

// ChainAuditorLogABI is the input ABI used to generate the binding from.
// Deprecated: Use ChainAuditorLogMetaData.ABI instead.
var ChainAuditorLogABI = ChainAuditorLogMetaData.ABI

// ChainAuditorLog is an auto generated Go binding around an Ethereum contract.
type ChainAuditorLog struct {
	ChainAuditorLogCaller     // Read-only binding to the contract
	ChainAuditorLogTransactor // Write-only binding to the contract
	ChainAuditorLogFilterer   // Log filterer for contract events
}

// ChainAuditorLogCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChainAuditorLogCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainAuditorLogTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChainAuditorLogTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainAuditorLogFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChainAuditorLogFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainAuditorLogSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChainAuditorLogSession struct {
	Contract     *ChainAuditorLog  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ChainAuditorLogCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChainAuditorLogCallerSession struct {
	Contract *ChainAuditorLogCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ChainAuditorLogTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChainAuditorLogTransactorSession struct {
	Contract     *ChainAuditorLogTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ChainAuditorLogRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChainAuditorLogRaw struct {
	Contract *ChainAuditorLog // Generic contract binding to access the raw methods on
}

// ChainAuditorLogCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChainAuditorLogCallerRaw struct {
	Contract *ChainAuditorLogCaller // Generic read-only contract binding to access the raw methods on
}

// ChainAuditorLogTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChainAuditorLogTransactorRaw struct {
	Contract *ChainAuditorLogTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChainAuditorLog creates a new instance of ChainAuditorLog, bound to a specific deployed contract.
func NewChainAuditorLog(address common.Address, backend bind.ContractBackend) (*ChainAuditorLog, error) {
	contract, err := bindChainAuditorLog(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLog{ChainAuditorLogCaller: ChainAuditorLogCaller{contract: contract}, ChainAuditorLogTransactor: ChainAuditorLogTransactor{contract: contract}, ChainAuditorLogFilterer: ChainAuditorLogFilterer{contract: contract}}, nil
}

// NewChainAuditorLogCaller creates a new read-only instance of ChainAuditorLog, bound to a specific deployed contract.
func NewChainAuditorLogCaller(address common.Address, caller bind.ContractCaller) (*ChainAuditorLogCaller, error) {
	contract, err := bindChainAuditorLog(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogCaller{contract: contract}, nil
}

// NewChainAuditorLogTransactor creates a new write-only instance of ChainAuditorLog, bound to a specific deployed contract.
func NewChainAuditorLogTransactor(address common.Address, transactor bind.ContractTransactor) (*ChainAuditorLogTransactor, error) {
	contract, err := bindChainAuditorLog(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogTransactor{contract: contract}, nil
}

// NewChainAuditorLogFilterer creates a new log filterer instance of ChainAuditorLog, bound to a specific deployed contract.
func NewChainAuditorLogFilterer(address common.Address, filterer bind.ContractFilterer) (*ChainAuditorLogFilterer, error) {
	contract, err := bindChainAuditorLog(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogFilterer{contract: contract}, nil
}

// bindChainAuditorLog binds a generic wrapper to an already deployed contract.
func bindChainAuditorLog(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ChainAuditorLogMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChainAuditorLog *ChainAuditorLogRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChainAuditorLog.Contract.ChainAuditorLogCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChainAuditorLog *ChainAuditorLogRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.ChainAuditorLogTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChainAuditorLog *ChainAuditorLogRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.ChainAuditorLogTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChainAuditorLog *ChainAuditorLogCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChainAuditorLog.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChainAuditorLog *ChainAuditorLogTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChainAuditorLog *ChainAuditorLogTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.contract.Transact(opts, method, params...)
}

// ANCHORROLE is a free data retrieval call binding the contract method 0xfd5f48c5.
//
// Solidity: function ANCHOR_ROLE() view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogCaller) ANCHORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "ANCHOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ANCHORROLE is a free data retrieval call binding the contract method 0xfd5f48c5.
//
// Solidity: function ANCHOR_ROLE() view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogSession) ANCHORROLE() ([32]byte, error) {
	return _ChainAuditorLog.Contract.ANCHORROLE(&_ChainAuditorLog.CallOpts)
}

// ANCHORROLE is a free data retrieval call binding the contract method 0xfd5f48c5.
//
// Solidity: function ANCHOR_ROLE() view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) ANCHORROLE() ([32]byte, error) {
	return _ChainAuditorLog.Contract.ANCHORROLE(&_ChainAuditorLog.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ChainAuditorLog.Contract.DEFAULTADMINROLE(&_ChainAuditorLog.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ChainAuditorLog.Contract.DEFAULTADMINROLE(&_ChainAuditorLog.CallOpts)
}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainAuditorLog *ChainAuditorLogCaller) DefaultAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "defaultAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainAuditorLog *ChainAuditorLogSession) DefaultAdmin() (common.Address, error) {
	return _ChainAuditorLog.Contract.DefaultAdmin(&_ChainAuditorLog.CallOpts)
}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) DefaultAdmin() (common.Address, error) {
	return _ChainAuditorLog.Contract.DefaultAdmin(&_ChainAuditorLog.CallOpts)
}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainAuditorLog *ChainAuditorLogCaller) DefaultAdminDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "defaultAdminDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainAuditorLog *ChainAuditorLogSession) DefaultAdminDelay() (*big.Int, error) {
	return _ChainAuditorLog.Contract.DefaultAdminDelay(&_ChainAuditorLog.CallOpts)
}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) DefaultAdminDelay() (*big.Int, error) {
	return _ChainAuditorLog.Contract.DefaultAdminDelay(&_ChainAuditorLog.CallOpts)
}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainAuditorLog *ChainAuditorLogCaller) DefaultAdminDelayIncreaseWait(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "defaultAdminDelayIncreaseWait")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainAuditorLog *ChainAuditorLogSession) DefaultAdminDelayIncreaseWait() (*big.Int, error) {
	return _ChainAuditorLog.Contract.DefaultAdminDelayIncreaseWait(&_ChainAuditorLog.CallOpts)
}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) DefaultAdminDelayIncreaseWait() (*big.Int, error) {
	return _ChainAuditorLog.Contract.DefaultAdminDelayIncreaseWait(&_ChainAuditorLog.CallOpts)
}

// GetGatewayKey is a free data retrieval call binding the contract method 0xcf750cdc.
//
// Solidity: function getGatewayKey(string did) view returns(bytes)
func (_ChainAuditorLog *ChainAuditorLogCaller) GetGatewayKey(opts *bind.CallOpts, did string) ([]byte, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "getGatewayKey", did)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetGatewayKey is a free data retrieval call binding the contract method 0xcf750cdc.
//
// Solidity: function getGatewayKey(string did) view returns(bytes)
func (_ChainAuditorLog *ChainAuditorLogSession) GetGatewayKey(did string) ([]byte, error) {
	return _ChainAuditorLog.Contract.GetGatewayKey(&_ChainAuditorLog.CallOpts, did)
}

// GetGatewayKey is a free data retrieval call binding the contract method 0xcf750cdc.
//
// Solidity: function getGatewayKey(string did) view returns(bytes)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) GetGatewayKey(did string) ([]byte, error) {
	return _ChainAuditorLog.Contract.GetGatewayKey(&_ChainAuditorLog.CallOpts, did)
}

// GetProvenanceAnchor is a free data retrieval call binding the contract method 0x14fa8922.
//
// Solidity: function getProvenanceAnchor(bytes32 bundleId, bytes32 retrievalId) view returns((bytes32,uint256,uint256,address))
func (_ChainAuditorLog *ChainAuditorLogCaller) GetProvenanceAnchor(opts *bind.CallOpts, bundleId [32]byte, retrievalId [32]byte) (AuditorLogProvenanceAnchor, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "getProvenanceAnchor", bundleId, retrievalId)

	if err != nil {
		return *new(AuditorLogProvenanceAnchor), err
	}

	out0 := *abi.ConvertType(out[0], new(AuditorLogProvenanceAnchor)).(*AuditorLogProvenanceAnchor)

	return out0, err

}

// GetProvenanceAnchor is a free data retrieval call binding the contract method 0x14fa8922.
//
// Solidity: function getProvenanceAnchor(bytes32 bundleId, bytes32 retrievalId) view returns((bytes32,uint256,uint256,address))
func (_ChainAuditorLog *ChainAuditorLogSession) GetProvenanceAnchor(bundleId [32]byte, retrievalId [32]byte) (AuditorLogProvenanceAnchor, error) {
	return _ChainAuditorLog.Contract.GetProvenanceAnchor(&_ChainAuditorLog.CallOpts, bundleId, retrievalId)
}

// GetProvenanceAnchor is a free data retrieval call binding the contract method 0x14fa8922.
//
// Solidity: function getProvenanceAnchor(bytes32 bundleId, bytes32 retrievalId) view returns((bytes32,uint256,uint256,address))
func (_ChainAuditorLog *ChainAuditorLogCallerSession) GetProvenanceAnchor(bundleId [32]byte, retrievalId [32]byte) (AuditorLogProvenanceAnchor, error) {
	return _ChainAuditorLog.Contract.GetProvenanceAnchor(&_ChainAuditorLog.CallOpts, bundleId, retrievalId)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ChainAuditorLog.Contract.GetRoleAdmin(&_ChainAuditorLog.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ChainAuditorLog.Contract.GetRoleAdmin(&_ChainAuditorLog.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainAuditorLog *ChainAuditorLogCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainAuditorLog *ChainAuditorLogSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ChainAuditorLog.Contract.HasRole(&_ChainAuditorLog.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ChainAuditorLog.Contract.HasRole(&_ChainAuditorLog.CallOpts, role, account)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainAuditorLog *ChainAuditorLogCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainAuditorLog *ChainAuditorLogSession) Owner() (common.Address, error) {
	return _ChainAuditorLog.Contract.Owner(&_ChainAuditorLog.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) Owner() (common.Address, error) {
	return _ChainAuditorLog.Contract.Owner(&_ChainAuditorLog.CallOpts)
}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainAuditorLog *ChainAuditorLogCaller) PendingDefaultAdmin(opts *bind.CallOpts) (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "pendingDefaultAdmin")

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
func (_ChainAuditorLog *ChainAuditorLogSession) PendingDefaultAdmin() (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	return _ChainAuditorLog.Contract.PendingDefaultAdmin(&_ChainAuditorLog.CallOpts)
}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) PendingDefaultAdmin() (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	return _ChainAuditorLog.Contract.PendingDefaultAdmin(&_ChainAuditorLog.CallOpts)
}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainAuditorLog *ChainAuditorLogCaller) PendingDefaultAdminDelay(opts *bind.CallOpts) (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "pendingDefaultAdminDelay")

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
func (_ChainAuditorLog *ChainAuditorLogSession) PendingDefaultAdminDelay() (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	return _ChainAuditorLog.Contract.PendingDefaultAdminDelay(&_ChainAuditorLog.CallOpts)
}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) PendingDefaultAdminDelay() (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	return _ChainAuditorLog.Contract.PendingDefaultAdminDelay(&_ChainAuditorLog.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainAuditorLog *ChainAuditorLogCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _ChainAuditorLog.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainAuditorLog *ChainAuditorLogSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ChainAuditorLog.Contract.SupportsInterface(&_ChainAuditorLog.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainAuditorLog *ChainAuditorLogCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ChainAuditorLog.Contract.SupportsInterface(&_ChainAuditorLog.CallOpts, interfaceId)
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) AcceptDefaultAdminTransfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "acceptDefaultAdminTransfer")
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainAuditorLog *ChainAuditorLogSession) AcceptDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.AcceptDefaultAdminTransfer(&_ChainAuditorLog.TransactOpts)
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) AcceptDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.AcceptDefaultAdminTransfer(&_ChainAuditorLog.TransactOpts)
}

// AnchorProvenance is a paid mutator transaction binding the contract method 0x68e911d3.
//
// Solidity: function anchorProvenance(bytes32 bundleId, bytes32 retrievalId, bytes32 provHash) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) AnchorProvenance(opts *bind.TransactOpts, bundleId [32]byte, retrievalId [32]byte, provHash [32]byte) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "anchorProvenance", bundleId, retrievalId, provHash)
}

// AnchorProvenance is a paid mutator transaction binding the contract method 0x68e911d3.
//
// Solidity: function anchorProvenance(bytes32 bundleId, bytes32 retrievalId, bytes32 provHash) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) AnchorProvenance(bundleId [32]byte, retrievalId [32]byte, provHash [32]byte) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.AnchorProvenance(&_ChainAuditorLog.TransactOpts, bundleId, retrievalId, provHash)
}

// AnchorProvenance is a paid mutator transaction binding the contract method 0x68e911d3.
//
// Solidity: function anchorProvenance(bytes32 bundleId, bytes32 retrievalId, bytes32 provHash) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) AnchorProvenance(bundleId [32]byte, retrievalId [32]byte, provHash [32]byte) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.AnchorProvenance(&_ChainAuditorLog.TransactOpts, bundleId, retrievalId, provHash)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) BeginDefaultAdminTransfer(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "beginDefaultAdminTransfer", newAdmin)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) BeginDefaultAdminTransfer(newAdmin common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.BeginDefaultAdminTransfer(&_ChainAuditorLog.TransactOpts, newAdmin)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) BeginDefaultAdminTransfer(newAdmin common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.BeginDefaultAdminTransfer(&_ChainAuditorLog.TransactOpts, newAdmin)
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) CancelDefaultAdminTransfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "cancelDefaultAdminTransfer")
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainAuditorLog *ChainAuditorLogSession) CancelDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.CancelDefaultAdminTransfer(&_ChainAuditorLog.TransactOpts)
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) CancelDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.CancelDefaultAdminTransfer(&_ChainAuditorLog.TransactOpts)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) ChangeDefaultAdminDelay(opts *bind.TransactOpts, newDelay *big.Int) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "changeDefaultAdminDelay", newDelay)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) ChangeDefaultAdminDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.ChangeDefaultAdminDelay(&_ChainAuditorLog.TransactOpts, newDelay)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) ChangeDefaultAdminDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.ChangeDefaultAdminDelay(&_ChainAuditorLog.TransactOpts, newDelay)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.GrantRole(&_ChainAuditorLog.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.GrantRole(&_ChainAuditorLog.TransactOpts, role, account)
}

// RegisterGatewayKey is a paid mutator transaction binding the contract method 0x40f090ca.
//
// Solidity: function registerGatewayKey(string did, bytes publicKey) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) RegisterGatewayKey(opts *bind.TransactOpts, did string, publicKey []byte) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "registerGatewayKey", did, publicKey)
}

// RegisterGatewayKey is a paid mutator transaction binding the contract method 0x40f090ca.
//
// Solidity: function registerGatewayKey(string did, bytes publicKey) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) RegisterGatewayKey(did string, publicKey []byte) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RegisterGatewayKey(&_ChainAuditorLog.TransactOpts, did, publicKey)
}

// RegisterGatewayKey is a paid mutator transaction binding the contract method 0x40f090ca.
//
// Solidity: function registerGatewayKey(string did, bytes publicKey) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) RegisterGatewayKey(did string, publicKey []byte) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RegisterGatewayKey(&_ChainAuditorLog.TransactOpts, did, publicKey)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RenounceRole(&_ChainAuditorLog.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RenounceRole(&_ChainAuditorLog.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RevokeRole(&_ChainAuditorLog.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RevokeRole(&_ChainAuditorLog.TransactOpts, role, account)
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainAuditorLog *ChainAuditorLogTransactor) RollbackDefaultAdminDelay(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainAuditorLog.contract.Transact(opts, "rollbackDefaultAdminDelay")
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainAuditorLog *ChainAuditorLogSession) RollbackDefaultAdminDelay() (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RollbackDefaultAdminDelay(&_ChainAuditorLog.TransactOpts)
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainAuditorLog *ChainAuditorLogTransactorSession) RollbackDefaultAdminDelay() (*types.Transaction, error) {
	return _ChainAuditorLog.Contract.RollbackDefaultAdminDelay(&_ChainAuditorLog.TransactOpts)
}

// ChainAuditorLogDefaultAdminDelayChangeCanceledIterator is returned from FilterDefaultAdminDelayChangeCanceled and is used to iterate over the raw logs and unpacked data for DefaultAdminDelayChangeCanceled events raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminDelayChangeCanceledIterator struct {
	Event *ChainAuditorLogDefaultAdminDelayChangeCanceled // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogDefaultAdminDelayChangeCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogDefaultAdminDelayChangeCanceled)
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
		it.Event = new(ChainAuditorLogDefaultAdminDelayChangeCanceled)
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
func (it *ChainAuditorLogDefaultAdminDelayChangeCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogDefaultAdminDelayChangeCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogDefaultAdminDelayChangeCanceled represents a DefaultAdminDelayChangeCanceled event raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminDelayChangeCanceled struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminDelayChangeCanceled is a free log retrieval operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterDefaultAdminDelayChangeCanceled(opts *bind.FilterOpts) (*ChainAuditorLogDefaultAdminDelayChangeCanceledIterator, error) {

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "DefaultAdminDelayChangeCanceled")
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogDefaultAdminDelayChangeCanceledIterator{contract: _ChainAuditorLog.contract, event: "DefaultAdminDelayChangeCanceled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminDelayChangeCanceled is a free log subscription operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchDefaultAdminDelayChangeCanceled(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogDefaultAdminDelayChangeCanceled) (event.Subscription, error) {

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "DefaultAdminDelayChangeCanceled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogDefaultAdminDelayChangeCanceled)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminDelayChangeCanceled", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseDefaultAdminDelayChangeCanceled(log types.Log) (*ChainAuditorLogDefaultAdminDelayChangeCanceled, error) {
	event := new(ChainAuditorLogDefaultAdminDelayChangeCanceled)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminDelayChangeCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogDefaultAdminDelayChangeScheduledIterator is returned from FilterDefaultAdminDelayChangeScheduled and is used to iterate over the raw logs and unpacked data for DefaultAdminDelayChangeScheduled events raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminDelayChangeScheduledIterator struct {
	Event *ChainAuditorLogDefaultAdminDelayChangeScheduled // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogDefaultAdminDelayChangeScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogDefaultAdminDelayChangeScheduled)
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
		it.Event = new(ChainAuditorLogDefaultAdminDelayChangeScheduled)
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
func (it *ChainAuditorLogDefaultAdminDelayChangeScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogDefaultAdminDelayChangeScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogDefaultAdminDelayChangeScheduled represents a DefaultAdminDelayChangeScheduled event raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminDelayChangeScheduled struct {
	NewDelay       *big.Int
	EffectSchedule *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminDelayChangeScheduled is a free log retrieval operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterDefaultAdminDelayChangeScheduled(opts *bind.FilterOpts) (*ChainAuditorLogDefaultAdminDelayChangeScheduledIterator, error) {

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "DefaultAdminDelayChangeScheduled")
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogDefaultAdminDelayChangeScheduledIterator{contract: _ChainAuditorLog.contract, event: "DefaultAdminDelayChangeScheduled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminDelayChangeScheduled is a free log subscription operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchDefaultAdminDelayChangeScheduled(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogDefaultAdminDelayChangeScheduled) (event.Subscription, error) {

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "DefaultAdminDelayChangeScheduled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogDefaultAdminDelayChangeScheduled)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminDelayChangeScheduled", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseDefaultAdminDelayChangeScheduled(log types.Log) (*ChainAuditorLogDefaultAdminDelayChangeScheduled, error) {
	event := new(ChainAuditorLogDefaultAdminDelayChangeScheduled)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminDelayChangeScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogDefaultAdminTransferCanceledIterator is returned from FilterDefaultAdminTransferCanceled and is used to iterate over the raw logs and unpacked data for DefaultAdminTransferCanceled events raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminTransferCanceledIterator struct {
	Event *ChainAuditorLogDefaultAdminTransferCanceled // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogDefaultAdminTransferCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogDefaultAdminTransferCanceled)
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
		it.Event = new(ChainAuditorLogDefaultAdminTransferCanceled)
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
func (it *ChainAuditorLogDefaultAdminTransferCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogDefaultAdminTransferCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogDefaultAdminTransferCanceled represents a DefaultAdminTransferCanceled event raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminTransferCanceled struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminTransferCanceled is a free log retrieval operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterDefaultAdminTransferCanceled(opts *bind.FilterOpts) (*ChainAuditorLogDefaultAdminTransferCanceledIterator, error) {

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "DefaultAdminTransferCanceled")
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogDefaultAdminTransferCanceledIterator{contract: _ChainAuditorLog.contract, event: "DefaultAdminTransferCanceled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminTransferCanceled is a free log subscription operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchDefaultAdminTransferCanceled(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogDefaultAdminTransferCanceled) (event.Subscription, error) {

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "DefaultAdminTransferCanceled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogDefaultAdminTransferCanceled)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminTransferCanceled", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseDefaultAdminTransferCanceled(log types.Log) (*ChainAuditorLogDefaultAdminTransferCanceled, error) {
	event := new(ChainAuditorLogDefaultAdminTransferCanceled)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminTransferCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogDefaultAdminTransferScheduledIterator is returned from FilterDefaultAdminTransferScheduled and is used to iterate over the raw logs and unpacked data for DefaultAdminTransferScheduled events raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminTransferScheduledIterator struct {
	Event *ChainAuditorLogDefaultAdminTransferScheduled // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogDefaultAdminTransferScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogDefaultAdminTransferScheduled)
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
		it.Event = new(ChainAuditorLogDefaultAdminTransferScheduled)
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
func (it *ChainAuditorLogDefaultAdminTransferScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogDefaultAdminTransferScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogDefaultAdminTransferScheduled represents a DefaultAdminTransferScheduled event raised by the ChainAuditorLog contract.
type ChainAuditorLogDefaultAdminTransferScheduled struct {
	NewAdmin       common.Address
	AcceptSchedule *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminTransferScheduled is a free log retrieval operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterDefaultAdminTransferScheduled(opts *bind.FilterOpts, newAdmin []common.Address) (*ChainAuditorLogDefaultAdminTransferScheduledIterator, error) {

	var newAdminRule []interface{}
	for _, newAdminItem := range newAdmin {
		newAdminRule = append(newAdminRule, newAdminItem)
	}

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "DefaultAdminTransferScheduled", newAdminRule)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogDefaultAdminTransferScheduledIterator{contract: _ChainAuditorLog.contract, event: "DefaultAdminTransferScheduled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminTransferScheduled is a free log subscription operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchDefaultAdminTransferScheduled(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogDefaultAdminTransferScheduled, newAdmin []common.Address) (event.Subscription, error) {

	var newAdminRule []interface{}
	for _, newAdminItem := range newAdmin {
		newAdminRule = append(newAdminRule, newAdminItem)
	}

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "DefaultAdminTransferScheduled", newAdminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogDefaultAdminTransferScheduled)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminTransferScheduled", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseDefaultAdminTransferScheduled(log types.Log) (*ChainAuditorLogDefaultAdminTransferScheduled, error) {
	event := new(ChainAuditorLogDefaultAdminTransferScheduled)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "DefaultAdminTransferScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogGatewayKeyRegisteredIterator is returned from FilterGatewayKeyRegistered and is used to iterate over the raw logs and unpacked data for GatewayKeyRegistered events raised by the ChainAuditorLog contract.
type ChainAuditorLogGatewayKeyRegisteredIterator struct {
	Event *ChainAuditorLogGatewayKeyRegistered // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogGatewayKeyRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogGatewayKeyRegistered)
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
		it.Event = new(ChainAuditorLogGatewayKeyRegistered)
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
func (it *ChainAuditorLogGatewayKeyRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogGatewayKeyRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogGatewayKeyRegistered represents a GatewayKeyRegistered event raised by the ChainAuditorLog contract.
type ChainAuditorLogGatewayKeyRegistered struct {
	DidHash   [32]byte
	PublicKey []byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterGatewayKeyRegistered is a free log retrieval operation binding the contract event 0x4883c3068828fb26fdd1b1d6772faf7ff162f748d81a7a76c69c01129c1a1e3c.
//
// Solidity: event GatewayKeyRegistered(bytes32 indexed didHash, bytes publicKey)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterGatewayKeyRegistered(opts *bind.FilterOpts, didHash [][32]byte) (*ChainAuditorLogGatewayKeyRegisteredIterator, error) {

	var didHashRule []interface{}
	for _, didHashItem := range didHash {
		didHashRule = append(didHashRule, didHashItem)
	}

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "GatewayKeyRegistered", didHashRule)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogGatewayKeyRegisteredIterator{contract: _ChainAuditorLog.contract, event: "GatewayKeyRegistered", logs: logs, sub: sub}, nil
}

// WatchGatewayKeyRegistered is a free log subscription operation binding the contract event 0x4883c3068828fb26fdd1b1d6772faf7ff162f748d81a7a76c69c01129c1a1e3c.
//
// Solidity: event GatewayKeyRegistered(bytes32 indexed didHash, bytes publicKey)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchGatewayKeyRegistered(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogGatewayKeyRegistered, didHash [][32]byte) (event.Subscription, error) {

	var didHashRule []interface{}
	for _, didHashItem := range didHash {
		didHashRule = append(didHashRule, didHashItem)
	}

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "GatewayKeyRegistered", didHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogGatewayKeyRegistered)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "GatewayKeyRegistered", log); err != nil {
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

// ParseGatewayKeyRegistered is a log parse operation binding the contract event 0x4883c3068828fb26fdd1b1d6772faf7ff162f748d81a7a76c69c01129c1a1e3c.
//
// Solidity: event GatewayKeyRegistered(bytes32 indexed didHash, bytes publicKey)
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseGatewayKeyRegistered(log types.Log) (*ChainAuditorLogGatewayKeyRegistered, error) {
	event := new(ChainAuditorLogGatewayKeyRegistered)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "GatewayKeyRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogProvenanceAnchoredIterator is returned from FilterProvenanceAnchored and is used to iterate over the raw logs and unpacked data for ProvenanceAnchored events raised by the ChainAuditorLog contract.
type ChainAuditorLogProvenanceAnchoredIterator struct {
	Event *ChainAuditorLogProvenanceAnchored // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogProvenanceAnchoredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogProvenanceAnchored)
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
		it.Event = new(ChainAuditorLogProvenanceAnchored)
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
func (it *ChainAuditorLogProvenanceAnchoredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogProvenanceAnchoredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogProvenanceAnchored represents a ProvenanceAnchored event raised by the ChainAuditorLog contract.
type ChainAuditorLogProvenanceAnchored struct {
	BundleId    [32]byte
	RetrievalId [32]byte
	ProvHash    [32]byte
	AnchoredBy  common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterProvenanceAnchored is a free log retrieval operation binding the contract event 0xd7d40e0685e79760d1850943ba7cf6ec254edc713c6c5c3b84ccc1db93eb752a.
//
// Solidity: event ProvenanceAnchored(bytes32 indexed bundleId, bytes32 indexed retrievalId, bytes32 provHash, address indexed anchoredBy)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterProvenanceAnchored(opts *bind.FilterOpts, bundleId [][32]byte, retrievalId [][32]byte, anchoredBy []common.Address) (*ChainAuditorLogProvenanceAnchoredIterator, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var retrievalIdRule []interface{}
	for _, retrievalIdItem := range retrievalId {
		retrievalIdRule = append(retrievalIdRule, retrievalIdItem)
	}

	var anchoredByRule []interface{}
	for _, anchoredByItem := range anchoredBy {
		anchoredByRule = append(anchoredByRule, anchoredByItem)
	}

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "ProvenanceAnchored", bundleIdRule, retrievalIdRule, anchoredByRule)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogProvenanceAnchoredIterator{contract: _ChainAuditorLog.contract, event: "ProvenanceAnchored", logs: logs, sub: sub}, nil
}

// WatchProvenanceAnchored is a free log subscription operation binding the contract event 0xd7d40e0685e79760d1850943ba7cf6ec254edc713c6c5c3b84ccc1db93eb752a.
//
// Solidity: event ProvenanceAnchored(bytes32 indexed bundleId, bytes32 indexed retrievalId, bytes32 provHash, address indexed anchoredBy)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchProvenanceAnchored(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogProvenanceAnchored, bundleId [][32]byte, retrievalId [][32]byte, anchoredBy []common.Address) (event.Subscription, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var retrievalIdRule []interface{}
	for _, retrievalIdItem := range retrievalId {
		retrievalIdRule = append(retrievalIdRule, retrievalIdItem)
	}

	var anchoredByRule []interface{}
	for _, anchoredByItem := range anchoredBy {
		anchoredByRule = append(anchoredByRule, anchoredByItem)
	}

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "ProvenanceAnchored", bundleIdRule, retrievalIdRule, anchoredByRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogProvenanceAnchored)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "ProvenanceAnchored", log); err != nil {
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

// ParseProvenanceAnchored is a log parse operation binding the contract event 0xd7d40e0685e79760d1850943ba7cf6ec254edc713c6c5c3b84ccc1db93eb752a.
//
// Solidity: event ProvenanceAnchored(bytes32 indexed bundleId, bytes32 indexed retrievalId, bytes32 provHash, address indexed anchoredBy)
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseProvenanceAnchored(log types.Log) (*ChainAuditorLogProvenanceAnchored, error) {
	event := new(ChainAuditorLogProvenanceAnchored)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "ProvenanceAnchored", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the ChainAuditorLog contract.
type ChainAuditorLogRoleAdminChangedIterator struct {
	Event *ChainAuditorLogRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogRoleAdminChanged)
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
		it.Event = new(ChainAuditorLogRoleAdminChanged)
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
func (it *ChainAuditorLogRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogRoleAdminChanged represents a RoleAdminChanged event raised by the ChainAuditorLog contract.
type ChainAuditorLogRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*ChainAuditorLogRoleAdminChangedIterator, error) {

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

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogRoleAdminChangedIterator{contract: _ChainAuditorLog.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogRoleAdminChanged)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseRoleAdminChanged(log types.Log) (*ChainAuditorLogRoleAdminChanged, error) {
	event := new(ChainAuditorLogRoleAdminChanged)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the ChainAuditorLog contract.
type ChainAuditorLogRoleGrantedIterator struct {
	Event *ChainAuditorLogRoleGranted // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogRoleGranted)
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
		it.Event = new(ChainAuditorLogRoleGranted)
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
func (it *ChainAuditorLogRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogRoleGranted represents a RoleGranted event raised by the ChainAuditorLog contract.
type ChainAuditorLogRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ChainAuditorLogRoleGrantedIterator, error) {

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

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogRoleGrantedIterator{contract: _ChainAuditorLog.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogRoleGranted)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseRoleGranted(log types.Log) (*ChainAuditorLogRoleGranted, error) {
	event := new(ChainAuditorLogRoleGranted)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainAuditorLogRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the ChainAuditorLog contract.
type ChainAuditorLogRoleRevokedIterator struct {
	Event *ChainAuditorLogRoleRevoked // Event containing the contract specifics and raw log

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
func (it *ChainAuditorLogRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainAuditorLogRoleRevoked)
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
		it.Event = new(ChainAuditorLogRoleRevoked)
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
func (it *ChainAuditorLogRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainAuditorLogRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainAuditorLogRoleRevoked represents a RoleRevoked event raised by the ChainAuditorLog contract.
type ChainAuditorLogRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainAuditorLog *ChainAuditorLogFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ChainAuditorLogRoleRevokedIterator, error) {

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

	logs, sub, err := _ChainAuditorLog.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ChainAuditorLogRoleRevokedIterator{contract: _ChainAuditorLog.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainAuditorLog *ChainAuditorLogFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *ChainAuditorLogRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _ChainAuditorLog.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainAuditorLogRoleRevoked)
				if err := _ChainAuditorLog.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_ChainAuditorLog *ChainAuditorLogFilterer) ParseRoleRevoked(log types.Log) (*ChainAuditorLogRoleRevoked, error) {
	event := new(ChainAuditorLogRoleRevoked)
	if err := _ChainAuditorLog.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
