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

// PoRVerifierChallenge is an auto generated low-level Go binding around an user-defined struct.
type PoRVerifierChallenge struct {
	BundleId [32]byte
	ShardIdx uint32
	ChunkIdx uint32
	Nonce    [32]byte
	PostedAt uint64
	Deadline uint64
	Auditor  common.Address
}

// PoRVerifierResponse is an auto generated low-level Go binding around an user-defined struct.
type PoRVerifierResponse struct {
	ChunkHash   [32]byte
	MerkleProof []byte
	BlsSig      []byte
	RespondedAt uint64
	Responder   common.Address
}

// PoRVerifierVerdict is an auto generated low-level Go binding around an user-defined struct.
type PoRVerifierVerdict struct {
	Recorded  bool
	Success   bool
	Reason    string
	VerdictAt uint64
	Auditor   common.Address
}

// ChainPoRVerifierMetaData contains all meta data concerning the ChainPoRVerifier contract.
var ChainPoRVerifierMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"admin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"adminTransferDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"},{\"name\":\"registryAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"AUDITOR_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"DEFAULT_ADMIN_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MAX_CONSECUTIVE_FAILURES\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RESPONDER_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"acceptDefaultAdminTransfer\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"beginDefaultAdminTransfer\",\"inputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"cancelDefaultAdminTransfer\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"changeDefaultAdminDelay\",\"inputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"defaultAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defaultAdminDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defaultAdminDelayIncreaseWait\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getChallenge\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structPoRVerifier.Challenge\",\"components\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"shardIdx\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"chunkIdx\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"nonce\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"postedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"auditor\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getConsecutiveFailures\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"shardIdx\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getResponse\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structPoRVerifier.Response\",\"components\":[{\"name\":\"chunkHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"merkleProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"blsSig\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"respondedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"responder\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRoleAdmin\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getVerdict\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structPoRVerifier.Verdict\",\"components\":[{\"name\":\"recorded\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"success\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"reason\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"verdictAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"auditor\",\"type\":\"address\",\"internalType\":\"address\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"grantRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"hasRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingDefaultAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingDefaultAdminDelay\",\"inputs\":[],\"outputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"internalType\":\"uint48\"},{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"postChallenge\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"shardIdx\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"chunkIdx\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"nonce\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"responseWindow\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"recordVerdict\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"success\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"reason\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"registry\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractICIDRegistry\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"respondToChallenge\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"chunkHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"merkleProof\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"},{\"name\":\"blsSig\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"totalChunks\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revokeRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"rollbackDefaultAdminDelay\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"supportsInterface\",\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\",\"internalType\":\"bytes4\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ChallengePosted\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"bundleId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"shardIdx\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"chunkIdx\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"},{\"name\":\"nonce\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ChallengeResponded\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"chunkHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"responder\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminDelayChangeCanceled\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminDelayChangeScheduled\",\"inputs\":[{\"name\":\"newDelay\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"},{\"name\":\"effectSchedule\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminTransferCanceled\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DefaultAdminTransferScheduled\",\"inputs\":[{\"name\":\"newAdmin\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"acceptSchedule\",\"type\":\"uint48\",\"indexed\":false,\"internalType\":\"uint48\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleAdminChanged\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"previousAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"newAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleGranted\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleRevoked\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ShardMigrationRequired\",\"inputs\":[{\"name\":\"bundleId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"shardIdx\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"consecutiveFailures\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"VerdictRecorded\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"success\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"},{\"name\":\"reason\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AccessControlBadConfirmation\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AccessControlEnforcedDefaultAdminDelay\",\"inputs\":[{\"name\":\"schedule\",\"type\":\"uint48\",\"internalType\":\"uint48\"}]},{\"type\":\"error\",\"name\":\"AccessControlEnforcedDefaultAdminRules\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AccessControlInvalidDefaultAdmin\",\"inputs\":[{\"name\":\"defaultAdmin\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AccessControlUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"neededRole\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ChallengeAlreadyResponded\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ChallengeExpired\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"nowTs\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"type\":\"error\",\"name\":\"ChallengeNotFound\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"EmptyBLSSig\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EmptyChunkHash\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidResponseWindow\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidShardIdx\",\"inputs\":[{\"name\":\"got\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"max\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]},{\"type\":\"error\",\"name\":\"MerkleProofInvalid\",\"inputs\":[{\"name\":\"expectedRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"computedRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ResponseRequired\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"SafeCastOverflowedUintDowncast\",\"inputs\":[{\"name\":\"bits\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"VerdictAlreadyRecorded\",\"inputs\":[{\"name\":\"challengeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ZeroRegistry\",\"inputs\":[]}]",
}

// ChainPoRVerifierABI is the input ABI used to generate the binding from.
// Deprecated: Use ChainPoRVerifierMetaData.ABI instead.
var ChainPoRVerifierABI = ChainPoRVerifierMetaData.ABI

// ChainPoRVerifier is an auto generated Go binding around an Ethereum contract.
type ChainPoRVerifier struct {
	ChainPoRVerifierCaller     // Read-only binding to the contract
	ChainPoRVerifierTransactor // Write-only binding to the contract
	ChainPoRVerifierFilterer   // Log filterer for contract events
}

// ChainPoRVerifierCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChainPoRVerifierCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainPoRVerifierTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChainPoRVerifierTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainPoRVerifierFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChainPoRVerifierFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChainPoRVerifierSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChainPoRVerifierSession struct {
	Contract     *ChainPoRVerifier // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ChainPoRVerifierCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChainPoRVerifierCallerSession struct {
	Contract *ChainPoRVerifierCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ChainPoRVerifierTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChainPoRVerifierTransactorSession struct {
	Contract     *ChainPoRVerifierTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ChainPoRVerifierRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChainPoRVerifierRaw struct {
	Contract *ChainPoRVerifier // Generic contract binding to access the raw methods on
}

// ChainPoRVerifierCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChainPoRVerifierCallerRaw struct {
	Contract *ChainPoRVerifierCaller // Generic read-only contract binding to access the raw methods on
}

// ChainPoRVerifierTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChainPoRVerifierTransactorRaw struct {
	Contract *ChainPoRVerifierTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChainPoRVerifier creates a new instance of ChainPoRVerifier, bound to a specific deployed contract.
func NewChainPoRVerifier(address common.Address, backend bind.ContractBackend) (*ChainPoRVerifier, error) {
	contract, err := bindChainPoRVerifier(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifier{ChainPoRVerifierCaller: ChainPoRVerifierCaller{contract: contract}, ChainPoRVerifierTransactor: ChainPoRVerifierTransactor{contract: contract}, ChainPoRVerifierFilterer: ChainPoRVerifierFilterer{contract: contract}}, nil
}

// NewChainPoRVerifierCaller creates a new read-only instance of ChainPoRVerifier, bound to a specific deployed contract.
func NewChainPoRVerifierCaller(address common.Address, caller bind.ContractCaller) (*ChainPoRVerifierCaller, error) {
	contract, err := bindChainPoRVerifier(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierCaller{contract: contract}, nil
}

// NewChainPoRVerifierTransactor creates a new write-only instance of ChainPoRVerifier, bound to a specific deployed contract.
func NewChainPoRVerifierTransactor(address common.Address, transactor bind.ContractTransactor) (*ChainPoRVerifierTransactor, error) {
	contract, err := bindChainPoRVerifier(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierTransactor{contract: contract}, nil
}

// NewChainPoRVerifierFilterer creates a new log filterer instance of ChainPoRVerifier, bound to a specific deployed contract.
func NewChainPoRVerifierFilterer(address common.Address, filterer bind.ContractFilterer) (*ChainPoRVerifierFilterer, error) {
	contract, err := bindChainPoRVerifier(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierFilterer{contract: contract}, nil
}

// bindChainPoRVerifier binds a generic wrapper to an already deployed contract.
func bindChainPoRVerifier(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ChainPoRVerifierMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChainPoRVerifier *ChainPoRVerifierRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChainPoRVerifier.Contract.ChainPoRVerifierCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChainPoRVerifier *ChainPoRVerifierRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.ChainPoRVerifierTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChainPoRVerifier *ChainPoRVerifierRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.ChainPoRVerifierTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChainPoRVerifier *ChainPoRVerifierCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChainPoRVerifier.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChainPoRVerifier *ChainPoRVerifierTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChainPoRVerifier *ChainPoRVerifierTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.contract.Transact(opts, method, params...)
}

// AUDITORROLE is a free data retrieval call binding the contract method 0x6e1d616e.
//
// Solidity: function AUDITOR_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) AUDITORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "AUDITOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AUDITORROLE is a free data retrieval call binding the contract method 0x6e1d616e.
//
// Solidity: function AUDITOR_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierSession) AUDITORROLE() ([32]byte, error) {
	return _ChainPoRVerifier.Contract.AUDITORROLE(&_ChainPoRVerifier.CallOpts)
}

// AUDITORROLE is a free data retrieval call binding the contract method 0x6e1d616e.
//
// Solidity: function AUDITOR_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) AUDITORROLE() ([32]byte, error) {
	return _ChainPoRVerifier.Contract.AUDITORROLE(&_ChainPoRVerifier.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ChainPoRVerifier.Contract.DEFAULTADMINROLE(&_ChainPoRVerifier.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ChainPoRVerifier.Contract.DEFAULTADMINROLE(&_ChainPoRVerifier.CallOpts)
}

// MAXCONSECUTIVEFAILURES is a free data retrieval call binding the contract method 0x6375184a.
//
// Solidity: function MAX_CONSECUTIVE_FAILURES() view returns(uint32)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) MAXCONSECUTIVEFAILURES(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "MAX_CONSECUTIVE_FAILURES")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// MAXCONSECUTIVEFAILURES is a free data retrieval call binding the contract method 0x6375184a.
//
// Solidity: function MAX_CONSECUTIVE_FAILURES() view returns(uint32)
func (_ChainPoRVerifier *ChainPoRVerifierSession) MAXCONSECUTIVEFAILURES() (uint32, error) {
	return _ChainPoRVerifier.Contract.MAXCONSECUTIVEFAILURES(&_ChainPoRVerifier.CallOpts)
}

// MAXCONSECUTIVEFAILURES is a free data retrieval call binding the contract method 0x6375184a.
//
// Solidity: function MAX_CONSECUTIVE_FAILURES() view returns(uint32)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) MAXCONSECUTIVEFAILURES() (uint32, error) {
	return _ChainPoRVerifier.Contract.MAXCONSECUTIVEFAILURES(&_ChainPoRVerifier.CallOpts)
}

// RESPONDERROLE is a free data retrieval call binding the contract method 0x514ebb42.
//
// Solidity: function RESPONDER_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) RESPONDERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "RESPONDER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RESPONDERROLE is a free data retrieval call binding the contract method 0x514ebb42.
//
// Solidity: function RESPONDER_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierSession) RESPONDERROLE() ([32]byte, error) {
	return _ChainPoRVerifier.Contract.RESPONDERROLE(&_ChainPoRVerifier.CallOpts)
}

// RESPONDERROLE is a free data retrieval call binding the contract method 0x514ebb42.
//
// Solidity: function RESPONDER_ROLE() view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) RESPONDERROLE() ([32]byte, error) {
	return _ChainPoRVerifier.Contract.RESPONDERROLE(&_ChainPoRVerifier.CallOpts)
}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) DefaultAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "defaultAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierSession) DefaultAdmin() (common.Address, error) {
	return _ChainPoRVerifier.Contract.DefaultAdmin(&_ChainPoRVerifier.CallOpts)
}

// DefaultAdmin is a free data retrieval call binding the contract method 0x84ef8ffc.
//
// Solidity: function defaultAdmin() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) DefaultAdmin() (common.Address, error) {
	return _ChainPoRVerifier.Contract.DefaultAdmin(&_ChainPoRVerifier.CallOpts)
}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) DefaultAdminDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "defaultAdminDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainPoRVerifier *ChainPoRVerifierSession) DefaultAdminDelay() (*big.Int, error) {
	return _ChainPoRVerifier.Contract.DefaultAdminDelay(&_ChainPoRVerifier.CallOpts)
}

// DefaultAdminDelay is a free data retrieval call binding the contract method 0xcc8463c8.
//
// Solidity: function defaultAdminDelay() view returns(uint48)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) DefaultAdminDelay() (*big.Int, error) {
	return _ChainPoRVerifier.Contract.DefaultAdminDelay(&_ChainPoRVerifier.CallOpts)
}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) DefaultAdminDelayIncreaseWait(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "defaultAdminDelayIncreaseWait")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainPoRVerifier *ChainPoRVerifierSession) DefaultAdminDelayIncreaseWait() (*big.Int, error) {
	return _ChainPoRVerifier.Contract.DefaultAdminDelayIncreaseWait(&_ChainPoRVerifier.CallOpts)
}

// DefaultAdminDelayIncreaseWait is a free data retrieval call binding the contract method 0x022d63fb.
//
// Solidity: function defaultAdminDelayIncreaseWait() view returns(uint48)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) DefaultAdminDelayIncreaseWait() (*big.Int, error) {
	return _ChainPoRVerifier.Contract.DefaultAdminDelayIncreaseWait(&_ChainPoRVerifier.CallOpts)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,uint32,uint32,bytes32,uint64,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierCaller) GetChallenge(opts *bind.CallOpts, challengeId [32]byte) (PoRVerifierChallenge, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "getChallenge", challengeId)

	if err != nil {
		return *new(PoRVerifierChallenge), err
	}

	out0 := *abi.ConvertType(out[0], new(PoRVerifierChallenge)).(*PoRVerifierChallenge)

	return out0, err

}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,uint32,uint32,bytes32,uint64,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierSession) GetChallenge(challengeId [32]byte) (PoRVerifierChallenge, error) {
	return _ChainPoRVerifier.Contract.GetChallenge(&_ChainPoRVerifier.CallOpts, challengeId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,uint32,uint32,bytes32,uint64,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) GetChallenge(challengeId [32]byte) (PoRVerifierChallenge, error) {
	return _ChainPoRVerifier.Contract.GetChallenge(&_ChainPoRVerifier.CallOpts, challengeId)
}

// GetConsecutiveFailures is a free data retrieval call binding the contract method 0xc74337c7.
//
// Solidity: function getConsecutiveFailures(bytes32 bundleId, uint32 shardIdx) view returns(uint32)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) GetConsecutiveFailures(opts *bind.CallOpts, bundleId [32]byte, shardIdx uint32) (uint32, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "getConsecutiveFailures", bundleId, shardIdx)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// GetConsecutiveFailures is a free data retrieval call binding the contract method 0xc74337c7.
//
// Solidity: function getConsecutiveFailures(bytes32 bundleId, uint32 shardIdx) view returns(uint32)
func (_ChainPoRVerifier *ChainPoRVerifierSession) GetConsecutiveFailures(bundleId [32]byte, shardIdx uint32) (uint32, error) {
	return _ChainPoRVerifier.Contract.GetConsecutiveFailures(&_ChainPoRVerifier.CallOpts, bundleId, shardIdx)
}

// GetConsecutiveFailures is a free data retrieval call binding the contract method 0xc74337c7.
//
// Solidity: function getConsecutiveFailures(bytes32 bundleId, uint32 shardIdx) view returns(uint32)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) GetConsecutiveFailures(bundleId [32]byte, shardIdx uint32) (uint32, error) {
	return _ChainPoRVerifier.Contract.GetConsecutiveFailures(&_ChainPoRVerifier.CallOpts, bundleId, shardIdx)
}

// GetResponse is a free data retrieval call binding the contract method 0x415a2bc1.
//
// Solidity: function getResponse(bytes32 challengeId) view returns((bytes32,bytes,bytes,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierCaller) GetResponse(opts *bind.CallOpts, challengeId [32]byte) (PoRVerifierResponse, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "getResponse", challengeId)

	if err != nil {
		return *new(PoRVerifierResponse), err
	}

	out0 := *abi.ConvertType(out[0], new(PoRVerifierResponse)).(*PoRVerifierResponse)

	return out0, err

}

// GetResponse is a free data retrieval call binding the contract method 0x415a2bc1.
//
// Solidity: function getResponse(bytes32 challengeId) view returns((bytes32,bytes,bytes,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierSession) GetResponse(challengeId [32]byte) (PoRVerifierResponse, error) {
	return _ChainPoRVerifier.Contract.GetResponse(&_ChainPoRVerifier.CallOpts, challengeId)
}

// GetResponse is a free data retrieval call binding the contract method 0x415a2bc1.
//
// Solidity: function getResponse(bytes32 challengeId) view returns((bytes32,bytes,bytes,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) GetResponse(challengeId [32]byte) (PoRVerifierResponse, error) {
	return _ChainPoRVerifier.Contract.GetResponse(&_ChainPoRVerifier.CallOpts, challengeId)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ChainPoRVerifier.Contract.GetRoleAdmin(&_ChainPoRVerifier.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ChainPoRVerifier.Contract.GetRoleAdmin(&_ChainPoRVerifier.CallOpts, role)
}

// GetVerdict is a free data retrieval call binding the contract method 0x8184717b.
//
// Solidity: function getVerdict(bytes32 challengeId) view returns((bool,bool,string,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierCaller) GetVerdict(opts *bind.CallOpts, challengeId [32]byte) (PoRVerifierVerdict, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "getVerdict", challengeId)

	if err != nil {
		return *new(PoRVerifierVerdict), err
	}

	out0 := *abi.ConvertType(out[0], new(PoRVerifierVerdict)).(*PoRVerifierVerdict)

	return out0, err

}

// GetVerdict is a free data retrieval call binding the contract method 0x8184717b.
//
// Solidity: function getVerdict(bytes32 challengeId) view returns((bool,bool,string,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierSession) GetVerdict(challengeId [32]byte) (PoRVerifierVerdict, error) {
	return _ChainPoRVerifier.Contract.GetVerdict(&_ChainPoRVerifier.CallOpts, challengeId)
}

// GetVerdict is a free data retrieval call binding the contract method 0x8184717b.
//
// Solidity: function getVerdict(bytes32 challengeId) view returns((bool,bool,string,uint64,address))
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) GetVerdict(challengeId [32]byte) (PoRVerifierVerdict, error) {
	return _ChainPoRVerifier.Contract.GetVerdict(&_ChainPoRVerifier.CallOpts, challengeId)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainPoRVerifier *ChainPoRVerifierSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ChainPoRVerifier.Contract.HasRole(&_ChainPoRVerifier.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ChainPoRVerifier.Contract.HasRole(&_ChainPoRVerifier.CallOpts, role, account)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierSession) Owner() (common.Address, error) {
	return _ChainPoRVerifier.Contract.Owner(&_ChainPoRVerifier.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) Owner() (common.Address, error) {
	return _ChainPoRVerifier.Contract.Owner(&_ChainPoRVerifier.CallOpts)
}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) PendingDefaultAdmin(opts *bind.CallOpts) (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "pendingDefaultAdmin")

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
func (_ChainPoRVerifier *ChainPoRVerifierSession) PendingDefaultAdmin() (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	return _ChainPoRVerifier.Contract.PendingDefaultAdmin(&_ChainPoRVerifier.CallOpts)
}

// PendingDefaultAdmin is a free data retrieval call binding the contract method 0xcf6eefb7.
//
// Solidity: function pendingDefaultAdmin() view returns(address newAdmin, uint48 schedule)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) PendingDefaultAdmin() (struct {
	NewAdmin common.Address
	Schedule *big.Int
}, error) {
	return _ChainPoRVerifier.Contract.PendingDefaultAdmin(&_ChainPoRVerifier.CallOpts)
}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) PendingDefaultAdminDelay(opts *bind.CallOpts) (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "pendingDefaultAdminDelay")

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
func (_ChainPoRVerifier *ChainPoRVerifierSession) PendingDefaultAdminDelay() (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	return _ChainPoRVerifier.Contract.PendingDefaultAdminDelay(&_ChainPoRVerifier.CallOpts)
}

// PendingDefaultAdminDelay is a free data retrieval call binding the contract method 0xa1eda53c.
//
// Solidity: function pendingDefaultAdminDelay() view returns(uint48 newDelay, uint48 schedule)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) PendingDefaultAdminDelay() (struct {
	NewDelay *big.Int
	Schedule *big.Int
}, error) {
	return _ChainPoRVerifier.Contract.PendingDefaultAdminDelay(&_ChainPoRVerifier.CallOpts)
}

// Registry is a free data retrieval call binding the contract method 0x7b103999.
//
// Solidity: function registry() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) Registry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "registry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Registry is a free data retrieval call binding the contract method 0x7b103999.
//
// Solidity: function registry() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierSession) Registry() (common.Address, error) {
	return _ChainPoRVerifier.Contract.Registry(&_ChainPoRVerifier.CallOpts)
}

// Registry is a free data retrieval call binding the contract method 0x7b103999.
//
// Solidity: function registry() view returns(address)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) Registry() (common.Address, error) {
	return _ChainPoRVerifier.Contract.Registry(&_ChainPoRVerifier.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainPoRVerifier *ChainPoRVerifierCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _ChainPoRVerifier.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainPoRVerifier *ChainPoRVerifierSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ChainPoRVerifier.Contract.SupportsInterface(&_ChainPoRVerifier.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ChainPoRVerifier *ChainPoRVerifierCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ChainPoRVerifier.Contract.SupportsInterface(&_ChainPoRVerifier.CallOpts, interfaceId)
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) AcceptDefaultAdminTransfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "acceptDefaultAdminTransfer")
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) AcceptDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.AcceptDefaultAdminTransfer(&_ChainPoRVerifier.TransactOpts)
}

// AcceptDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xcefc1429.
//
// Solidity: function acceptDefaultAdminTransfer() returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) AcceptDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.AcceptDefaultAdminTransfer(&_ChainPoRVerifier.TransactOpts)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) BeginDefaultAdminTransfer(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "beginDefaultAdminTransfer", newAdmin)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) BeginDefaultAdminTransfer(newAdmin common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.BeginDefaultAdminTransfer(&_ChainPoRVerifier.TransactOpts, newAdmin)
}

// BeginDefaultAdminTransfer is a paid mutator transaction binding the contract method 0x634e93da.
//
// Solidity: function beginDefaultAdminTransfer(address newAdmin) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) BeginDefaultAdminTransfer(newAdmin common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.BeginDefaultAdminTransfer(&_ChainPoRVerifier.TransactOpts, newAdmin)
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) CancelDefaultAdminTransfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "cancelDefaultAdminTransfer")
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) CancelDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.CancelDefaultAdminTransfer(&_ChainPoRVerifier.TransactOpts)
}

// CancelDefaultAdminTransfer is a paid mutator transaction binding the contract method 0xd602b9fd.
//
// Solidity: function cancelDefaultAdminTransfer() returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) CancelDefaultAdminTransfer() (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.CancelDefaultAdminTransfer(&_ChainPoRVerifier.TransactOpts)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) ChangeDefaultAdminDelay(opts *bind.TransactOpts, newDelay *big.Int) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "changeDefaultAdminDelay", newDelay)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) ChangeDefaultAdminDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.ChangeDefaultAdminDelay(&_ChainPoRVerifier.TransactOpts, newDelay)
}

// ChangeDefaultAdminDelay is a paid mutator transaction binding the contract method 0x649a5ec7.
//
// Solidity: function changeDefaultAdminDelay(uint48 newDelay) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) ChangeDefaultAdminDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.ChangeDefaultAdminDelay(&_ChainPoRVerifier.TransactOpts, newDelay)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.GrantRole(&_ChainPoRVerifier.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.GrantRole(&_ChainPoRVerifier.TransactOpts, role, account)
}

// PostChallenge is a paid mutator transaction binding the contract method 0xbacd2ba0.
//
// Solidity: function postChallenge(bytes32 bundleId, uint32 shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 responseWindow) returns(bytes32 challengeId)
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) PostChallenge(opts *bind.TransactOpts, bundleId [32]byte, shardIdx uint32, chunkIdx uint32, nonce [32]byte, responseWindow uint64) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "postChallenge", bundleId, shardIdx, chunkIdx, nonce, responseWindow)
}

// PostChallenge is a paid mutator transaction binding the contract method 0xbacd2ba0.
//
// Solidity: function postChallenge(bytes32 bundleId, uint32 shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 responseWindow) returns(bytes32 challengeId)
func (_ChainPoRVerifier *ChainPoRVerifierSession) PostChallenge(bundleId [32]byte, shardIdx uint32, chunkIdx uint32, nonce [32]byte, responseWindow uint64) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.PostChallenge(&_ChainPoRVerifier.TransactOpts, bundleId, shardIdx, chunkIdx, nonce, responseWindow)
}

// PostChallenge is a paid mutator transaction binding the contract method 0xbacd2ba0.
//
// Solidity: function postChallenge(bytes32 bundleId, uint32 shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 responseWindow) returns(bytes32 challengeId)
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) PostChallenge(bundleId [32]byte, shardIdx uint32, chunkIdx uint32, nonce [32]byte, responseWindow uint64) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.PostChallenge(&_ChainPoRVerifier.TransactOpts, bundleId, shardIdx, chunkIdx, nonce, responseWindow)
}

// RecordVerdict is a paid mutator transaction binding the contract method 0xcd8391fd.
//
// Solidity: function recordVerdict(bytes32 challengeId, bool success, string reason) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) RecordVerdict(opts *bind.TransactOpts, challengeId [32]byte, success bool, reason string) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "recordVerdict", challengeId, success, reason)
}

// RecordVerdict is a paid mutator transaction binding the contract method 0xcd8391fd.
//
// Solidity: function recordVerdict(bytes32 challengeId, bool success, string reason) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) RecordVerdict(challengeId [32]byte, success bool, reason string) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RecordVerdict(&_ChainPoRVerifier.TransactOpts, challengeId, success, reason)
}

// RecordVerdict is a paid mutator transaction binding the contract method 0xcd8391fd.
//
// Solidity: function recordVerdict(bytes32 challengeId, bool success, string reason) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) RecordVerdict(challengeId [32]byte, success bool, reason string) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RecordVerdict(&_ChainPoRVerifier.TransactOpts, challengeId, success, reason)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RenounceRole(&_ChainPoRVerifier.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RenounceRole(&_ChainPoRVerifier.TransactOpts, role, account)
}

// RespondToChallenge is a paid mutator transaction binding the contract method 0xba314170.
//
// Solidity: function respondToChallenge(bytes32 challengeId, bytes32 chunkHash, bytes32[] merkleProof, bytes blsSig, uint32 totalChunks) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) RespondToChallenge(opts *bind.TransactOpts, challengeId [32]byte, chunkHash [32]byte, merkleProof [][32]byte, blsSig []byte, totalChunks uint32) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "respondToChallenge", challengeId, chunkHash, merkleProof, blsSig, totalChunks)
}

// RespondToChallenge is a paid mutator transaction binding the contract method 0xba314170.
//
// Solidity: function respondToChallenge(bytes32 challengeId, bytes32 chunkHash, bytes32[] merkleProof, bytes blsSig, uint32 totalChunks) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) RespondToChallenge(challengeId [32]byte, chunkHash [32]byte, merkleProof [][32]byte, blsSig []byte, totalChunks uint32) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RespondToChallenge(&_ChainPoRVerifier.TransactOpts, challengeId, chunkHash, merkleProof, blsSig, totalChunks)
}

// RespondToChallenge is a paid mutator transaction binding the contract method 0xba314170.
//
// Solidity: function respondToChallenge(bytes32 challengeId, bytes32 chunkHash, bytes32[] merkleProof, bytes blsSig, uint32 totalChunks) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) RespondToChallenge(challengeId [32]byte, chunkHash [32]byte, merkleProof [][32]byte, blsSig []byte, totalChunks uint32) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RespondToChallenge(&_ChainPoRVerifier.TransactOpts, challengeId, chunkHash, merkleProof, blsSig, totalChunks)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RevokeRole(&_ChainPoRVerifier.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RevokeRole(&_ChainPoRVerifier.TransactOpts, role, account)
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactor) RollbackDefaultAdminDelay(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChainPoRVerifier.contract.Transact(opts, "rollbackDefaultAdminDelay")
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainPoRVerifier *ChainPoRVerifierSession) RollbackDefaultAdminDelay() (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RollbackDefaultAdminDelay(&_ChainPoRVerifier.TransactOpts)
}

// RollbackDefaultAdminDelay is a paid mutator transaction binding the contract method 0x0aa6220b.
//
// Solidity: function rollbackDefaultAdminDelay() returns()
func (_ChainPoRVerifier *ChainPoRVerifierTransactorSession) RollbackDefaultAdminDelay() (*types.Transaction, error) {
	return _ChainPoRVerifier.Contract.RollbackDefaultAdminDelay(&_ChainPoRVerifier.TransactOpts)
}

// ChainPoRVerifierChallengePostedIterator is returned from FilterChallengePosted and is used to iterate over the raw logs and unpacked data for ChallengePosted events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierChallengePostedIterator struct {
	Event *ChainPoRVerifierChallengePosted // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierChallengePostedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierChallengePosted)
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
		it.Event = new(ChainPoRVerifierChallengePosted)
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
func (it *ChainPoRVerifierChallengePostedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierChallengePostedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierChallengePosted represents a ChallengePosted event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierChallengePosted struct {
	ChallengeId [32]byte
	BundleId    [32]byte
	ShardIdx    uint32
	ChunkIdx    uint32
	Nonce       [32]byte
	Deadline    uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChallengePosted is a free log retrieval operation binding the contract event 0xd4f8365b451289f306e893969cadbe39542a3d1ab33b3c83a9819d12f8e20df3.
//
// Solidity: event ChallengePosted(bytes32 indexed challengeId, bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 deadline)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterChallengePosted(opts *bind.FilterOpts, challengeId [][32]byte, bundleId [][32]byte, shardIdx []uint32) (*ChainPoRVerifierChallengePostedIterator, error) {

	var challengeIdRule []interface{}
	for _, challengeIdItem := range challengeId {
		challengeIdRule = append(challengeIdRule, challengeIdItem)
	}
	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var shardIdxRule []interface{}
	for _, shardIdxItem := range shardIdx {
		shardIdxRule = append(shardIdxRule, shardIdxItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "ChallengePosted", challengeIdRule, bundleIdRule, shardIdxRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierChallengePostedIterator{contract: _ChainPoRVerifier.contract, event: "ChallengePosted", logs: logs, sub: sub}, nil
}

// WatchChallengePosted is a free log subscription operation binding the contract event 0xd4f8365b451289f306e893969cadbe39542a3d1ab33b3c83a9819d12f8e20df3.
//
// Solidity: event ChallengePosted(bytes32 indexed challengeId, bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 deadline)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchChallengePosted(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierChallengePosted, challengeId [][32]byte, bundleId [][32]byte, shardIdx []uint32) (event.Subscription, error) {

	var challengeIdRule []interface{}
	for _, challengeIdItem := range challengeId {
		challengeIdRule = append(challengeIdRule, challengeIdItem)
	}
	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var shardIdxRule []interface{}
	for _, shardIdxItem := range shardIdx {
		shardIdxRule = append(shardIdxRule, shardIdxItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "ChallengePosted", challengeIdRule, bundleIdRule, shardIdxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierChallengePosted)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "ChallengePosted", log); err != nil {
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

// ParseChallengePosted is a log parse operation binding the contract event 0xd4f8365b451289f306e893969cadbe39542a3d1ab33b3c83a9819d12f8e20df3.
//
// Solidity: event ChallengePosted(bytes32 indexed challengeId, bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 chunkIdx, bytes32 nonce, uint64 deadline)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseChallengePosted(log types.Log) (*ChainPoRVerifierChallengePosted, error) {
	event := new(ChainPoRVerifierChallengePosted)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "ChallengePosted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierChallengeRespondedIterator is returned from FilterChallengeResponded and is used to iterate over the raw logs and unpacked data for ChallengeResponded events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierChallengeRespondedIterator struct {
	Event *ChainPoRVerifierChallengeResponded // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierChallengeRespondedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierChallengeResponded)
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
		it.Event = new(ChainPoRVerifierChallengeResponded)
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
func (it *ChainPoRVerifierChallengeRespondedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierChallengeRespondedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierChallengeResponded represents a ChallengeResponded event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierChallengeResponded struct {
	ChallengeId [32]byte
	ChunkHash   [32]byte
	Responder   common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChallengeResponded is a free log retrieval operation binding the contract event 0xe8dfad5155da886e6e7de81658419a3bc9296d40aa53f4a6c898bb2a034b5bf2.
//
// Solidity: event ChallengeResponded(bytes32 indexed challengeId, bytes32 chunkHash, address indexed responder)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterChallengeResponded(opts *bind.FilterOpts, challengeId [][32]byte, responder []common.Address) (*ChainPoRVerifierChallengeRespondedIterator, error) {

	var challengeIdRule []interface{}
	for _, challengeIdItem := range challengeId {
		challengeIdRule = append(challengeIdRule, challengeIdItem)
	}

	var responderRule []interface{}
	for _, responderItem := range responder {
		responderRule = append(responderRule, responderItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "ChallengeResponded", challengeIdRule, responderRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierChallengeRespondedIterator{contract: _ChainPoRVerifier.contract, event: "ChallengeResponded", logs: logs, sub: sub}, nil
}

// WatchChallengeResponded is a free log subscription operation binding the contract event 0xe8dfad5155da886e6e7de81658419a3bc9296d40aa53f4a6c898bb2a034b5bf2.
//
// Solidity: event ChallengeResponded(bytes32 indexed challengeId, bytes32 chunkHash, address indexed responder)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchChallengeResponded(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierChallengeResponded, challengeId [][32]byte, responder []common.Address) (event.Subscription, error) {

	var challengeIdRule []interface{}
	for _, challengeIdItem := range challengeId {
		challengeIdRule = append(challengeIdRule, challengeIdItem)
	}

	var responderRule []interface{}
	for _, responderItem := range responder {
		responderRule = append(responderRule, responderItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "ChallengeResponded", challengeIdRule, responderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierChallengeResponded)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "ChallengeResponded", log); err != nil {
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

// ParseChallengeResponded is a log parse operation binding the contract event 0xe8dfad5155da886e6e7de81658419a3bc9296d40aa53f4a6c898bb2a034b5bf2.
//
// Solidity: event ChallengeResponded(bytes32 indexed challengeId, bytes32 chunkHash, address indexed responder)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseChallengeResponded(log types.Log) (*ChainPoRVerifierChallengeResponded, error) {
	event := new(ChainPoRVerifierChallengeResponded)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "ChallengeResponded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator is returned from FilterDefaultAdminDelayChangeCanceled and is used to iterate over the raw logs and unpacked data for DefaultAdminDelayChangeCanceled events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator struct {
	Event *ChainPoRVerifierDefaultAdminDelayChangeCanceled // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierDefaultAdminDelayChangeCanceled)
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
		it.Event = new(ChainPoRVerifierDefaultAdminDelayChangeCanceled)
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
func (it *ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierDefaultAdminDelayChangeCanceled represents a DefaultAdminDelayChangeCanceled event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminDelayChangeCanceled struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminDelayChangeCanceled is a free log retrieval operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterDefaultAdminDelayChangeCanceled(opts *bind.FilterOpts) (*ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator, error) {

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "DefaultAdminDelayChangeCanceled")
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierDefaultAdminDelayChangeCanceledIterator{contract: _ChainPoRVerifier.contract, event: "DefaultAdminDelayChangeCanceled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminDelayChangeCanceled is a free log subscription operation binding the contract event 0x2b1fa2edafe6f7b9e97c1a9e0c3660e645beb2dcaa2d45bdbf9beaf5472e1ec5.
//
// Solidity: event DefaultAdminDelayChangeCanceled()
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchDefaultAdminDelayChangeCanceled(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierDefaultAdminDelayChangeCanceled) (event.Subscription, error) {

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "DefaultAdminDelayChangeCanceled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierDefaultAdminDelayChangeCanceled)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminDelayChangeCanceled", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseDefaultAdminDelayChangeCanceled(log types.Log) (*ChainPoRVerifierDefaultAdminDelayChangeCanceled, error) {
	event := new(ChainPoRVerifierDefaultAdminDelayChangeCanceled)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminDelayChangeCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator is returned from FilterDefaultAdminDelayChangeScheduled and is used to iterate over the raw logs and unpacked data for DefaultAdminDelayChangeScheduled events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator struct {
	Event *ChainPoRVerifierDefaultAdminDelayChangeScheduled // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierDefaultAdminDelayChangeScheduled)
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
		it.Event = new(ChainPoRVerifierDefaultAdminDelayChangeScheduled)
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
func (it *ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierDefaultAdminDelayChangeScheduled represents a DefaultAdminDelayChangeScheduled event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminDelayChangeScheduled struct {
	NewDelay       *big.Int
	EffectSchedule *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminDelayChangeScheduled is a free log retrieval operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterDefaultAdminDelayChangeScheduled(opts *bind.FilterOpts) (*ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator, error) {

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "DefaultAdminDelayChangeScheduled")
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierDefaultAdminDelayChangeScheduledIterator{contract: _ChainPoRVerifier.contract, event: "DefaultAdminDelayChangeScheduled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminDelayChangeScheduled is a free log subscription operation binding the contract event 0xf1038c18cf84a56e432fdbfaf746924b7ea511dfe03a6506a0ceba4888788d9b.
//
// Solidity: event DefaultAdminDelayChangeScheduled(uint48 newDelay, uint48 effectSchedule)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchDefaultAdminDelayChangeScheduled(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierDefaultAdminDelayChangeScheduled) (event.Subscription, error) {

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "DefaultAdminDelayChangeScheduled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierDefaultAdminDelayChangeScheduled)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminDelayChangeScheduled", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseDefaultAdminDelayChangeScheduled(log types.Log) (*ChainPoRVerifierDefaultAdminDelayChangeScheduled, error) {
	event := new(ChainPoRVerifierDefaultAdminDelayChangeScheduled)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminDelayChangeScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierDefaultAdminTransferCanceledIterator is returned from FilterDefaultAdminTransferCanceled and is used to iterate over the raw logs and unpacked data for DefaultAdminTransferCanceled events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminTransferCanceledIterator struct {
	Event *ChainPoRVerifierDefaultAdminTransferCanceled // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierDefaultAdminTransferCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierDefaultAdminTransferCanceled)
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
		it.Event = new(ChainPoRVerifierDefaultAdminTransferCanceled)
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
func (it *ChainPoRVerifierDefaultAdminTransferCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierDefaultAdminTransferCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierDefaultAdminTransferCanceled represents a DefaultAdminTransferCanceled event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminTransferCanceled struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminTransferCanceled is a free log retrieval operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterDefaultAdminTransferCanceled(opts *bind.FilterOpts) (*ChainPoRVerifierDefaultAdminTransferCanceledIterator, error) {

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "DefaultAdminTransferCanceled")
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierDefaultAdminTransferCanceledIterator{contract: _ChainPoRVerifier.contract, event: "DefaultAdminTransferCanceled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminTransferCanceled is a free log subscription operation binding the contract event 0x8886ebfc4259abdbc16601dd8fb5678e54878f47b3c34836cfc51154a9605109.
//
// Solidity: event DefaultAdminTransferCanceled()
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchDefaultAdminTransferCanceled(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierDefaultAdminTransferCanceled) (event.Subscription, error) {

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "DefaultAdminTransferCanceled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierDefaultAdminTransferCanceled)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminTransferCanceled", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseDefaultAdminTransferCanceled(log types.Log) (*ChainPoRVerifierDefaultAdminTransferCanceled, error) {
	event := new(ChainPoRVerifierDefaultAdminTransferCanceled)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminTransferCanceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierDefaultAdminTransferScheduledIterator is returned from FilterDefaultAdminTransferScheduled and is used to iterate over the raw logs and unpacked data for DefaultAdminTransferScheduled events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminTransferScheduledIterator struct {
	Event *ChainPoRVerifierDefaultAdminTransferScheduled // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierDefaultAdminTransferScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierDefaultAdminTransferScheduled)
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
		it.Event = new(ChainPoRVerifierDefaultAdminTransferScheduled)
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
func (it *ChainPoRVerifierDefaultAdminTransferScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierDefaultAdminTransferScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierDefaultAdminTransferScheduled represents a DefaultAdminTransferScheduled event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierDefaultAdminTransferScheduled struct {
	NewAdmin       common.Address
	AcceptSchedule *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDefaultAdminTransferScheduled is a free log retrieval operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterDefaultAdminTransferScheduled(opts *bind.FilterOpts, newAdmin []common.Address) (*ChainPoRVerifierDefaultAdminTransferScheduledIterator, error) {

	var newAdminRule []interface{}
	for _, newAdminItem := range newAdmin {
		newAdminRule = append(newAdminRule, newAdminItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "DefaultAdminTransferScheduled", newAdminRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierDefaultAdminTransferScheduledIterator{contract: _ChainPoRVerifier.contract, event: "DefaultAdminTransferScheduled", logs: logs, sub: sub}, nil
}

// WatchDefaultAdminTransferScheduled is a free log subscription operation binding the contract event 0x3377dc44241e779dd06afab5b788a35ca5f3b778836e2990bdb26a2a4b2e5ed6.
//
// Solidity: event DefaultAdminTransferScheduled(address indexed newAdmin, uint48 acceptSchedule)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchDefaultAdminTransferScheduled(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierDefaultAdminTransferScheduled, newAdmin []common.Address) (event.Subscription, error) {

	var newAdminRule []interface{}
	for _, newAdminItem := range newAdmin {
		newAdminRule = append(newAdminRule, newAdminItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "DefaultAdminTransferScheduled", newAdminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierDefaultAdminTransferScheduled)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminTransferScheduled", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseDefaultAdminTransferScheduled(log types.Log) (*ChainPoRVerifierDefaultAdminTransferScheduled, error) {
	event := new(ChainPoRVerifierDefaultAdminTransferScheduled)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "DefaultAdminTransferScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierRoleAdminChangedIterator struct {
	Event *ChainPoRVerifierRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierRoleAdminChanged)
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
		it.Event = new(ChainPoRVerifierRoleAdminChanged)
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
func (it *ChainPoRVerifierRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierRoleAdminChanged represents a RoleAdminChanged event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*ChainPoRVerifierRoleAdminChangedIterator, error) {

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

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierRoleAdminChangedIterator{contract: _ChainPoRVerifier.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierRoleAdminChanged)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseRoleAdminChanged(log types.Log) (*ChainPoRVerifierRoleAdminChanged, error) {
	event := new(ChainPoRVerifierRoleAdminChanged)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierRoleGrantedIterator struct {
	Event *ChainPoRVerifierRoleGranted // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierRoleGranted)
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
		it.Event = new(ChainPoRVerifierRoleGranted)
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
func (it *ChainPoRVerifierRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierRoleGranted represents a RoleGranted event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ChainPoRVerifierRoleGrantedIterator, error) {

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

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierRoleGrantedIterator{contract: _ChainPoRVerifier.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierRoleGranted)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseRoleGranted(log types.Log) (*ChainPoRVerifierRoleGranted, error) {
	event := new(ChainPoRVerifierRoleGranted)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierRoleRevokedIterator struct {
	Event *ChainPoRVerifierRoleRevoked // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierRoleRevoked)
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
		it.Event = new(ChainPoRVerifierRoleRevoked)
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
func (it *ChainPoRVerifierRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierRoleRevoked represents a RoleRevoked event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ChainPoRVerifierRoleRevokedIterator, error) {

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

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierRoleRevokedIterator{contract: _ChainPoRVerifier.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierRoleRevoked)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseRoleRevoked(log types.Log) (*ChainPoRVerifierRoleRevoked, error) {
	event := new(ChainPoRVerifierRoleRevoked)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierShardMigrationRequiredIterator is returned from FilterShardMigrationRequired and is used to iterate over the raw logs and unpacked data for ShardMigrationRequired events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierShardMigrationRequiredIterator struct {
	Event *ChainPoRVerifierShardMigrationRequired // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierShardMigrationRequiredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierShardMigrationRequired)
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
		it.Event = new(ChainPoRVerifierShardMigrationRequired)
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
func (it *ChainPoRVerifierShardMigrationRequiredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierShardMigrationRequiredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierShardMigrationRequired represents a ShardMigrationRequired event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierShardMigrationRequired struct {
	BundleId            [32]byte
	ShardIdx            uint32
	ConsecutiveFailures uint32
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterShardMigrationRequired is a free log retrieval operation binding the contract event 0x04efdc3e71b4dc8c7b4fad682a18459c1d21bd5a18e625e9241135be4df3199c.
//
// Solidity: event ShardMigrationRequired(bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 consecutiveFailures)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterShardMigrationRequired(opts *bind.FilterOpts, bundleId [][32]byte, shardIdx []uint32) (*ChainPoRVerifierShardMigrationRequiredIterator, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var shardIdxRule []interface{}
	for _, shardIdxItem := range shardIdx {
		shardIdxRule = append(shardIdxRule, shardIdxItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "ShardMigrationRequired", bundleIdRule, shardIdxRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierShardMigrationRequiredIterator{contract: _ChainPoRVerifier.contract, event: "ShardMigrationRequired", logs: logs, sub: sub}, nil
}

// WatchShardMigrationRequired is a free log subscription operation binding the contract event 0x04efdc3e71b4dc8c7b4fad682a18459c1d21bd5a18e625e9241135be4df3199c.
//
// Solidity: event ShardMigrationRequired(bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 consecutiveFailures)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchShardMigrationRequired(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierShardMigrationRequired, bundleId [][32]byte, shardIdx []uint32) (event.Subscription, error) {

	var bundleIdRule []interface{}
	for _, bundleIdItem := range bundleId {
		bundleIdRule = append(bundleIdRule, bundleIdItem)
	}
	var shardIdxRule []interface{}
	for _, shardIdxItem := range shardIdx {
		shardIdxRule = append(shardIdxRule, shardIdxItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "ShardMigrationRequired", bundleIdRule, shardIdxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierShardMigrationRequired)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "ShardMigrationRequired", log); err != nil {
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

// ParseShardMigrationRequired is a log parse operation binding the contract event 0x04efdc3e71b4dc8c7b4fad682a18459c1d21bd5a18e625e9241135be4df3199c.
//
// Solidity: event ShardMigrationRequired(bytes32 indexed bundleId, uint32 indexed shardIdx, uint32 consecutiveFailures)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseShardMigrationRequired(log types.Log) (*ChainPoRVerifierShardMigrationRequired, error) {
	event := new(ChainPoRVerifierShardMigrationRequired)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "ShardMigrationRequired", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChainPoRVerifierVerdictRecordedIterator is returned from FilterVerdictRecorded and is used to iterate over the raw logs and unpacked data for VerdictRecorded events raised by the ChainPoRVerifier contract.
type ChainPoRVerifierVerdictRecordedIterator struct {
	Event *ChainPoRVerifierVerdictRecorded // Event containing the contract specifics and raw log

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
func (it *ChainPoRVerifierVerdictRecordedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChainPoRVerifierVerdictRecorded)
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
		it.Event = new(ChainPoRVerifierVerdictRecorded)
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
func (it *ChainPoRVerifierVerdictRecordedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChainPoRVerifierVerdictRecordedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChainPoRVerifierVerdictRecorded represents a VerdictRecorded event raised by the ChainPoRVerifier contract.
type ChainPoRVerifierVerdictRecorded struct {
	ChallengeId [32]byte
	Success     bool
	Reason      string
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterVerdictRecorded is a free log retrieval operation binding the contract event 0x457c2d49bb177a3347ad31b50be48291c89eedb5e2ed86f6f9f36c15d48f5737.
//
// Solidity: event VerdictRecorded(bytes32 indexed challengeId, bool success, string reason)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) FilterVerdictRecorded(opts *bind.FilterOpts, challengeId [][32]byte) (*ChainPoRVerifierVerdictRecordedIterator, error) {

	var challengeIdRule []interface{}
	for _, challengeIdItem := range challengeId {
		challengeIdRule = append(challengeIdRule, challengeIdItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.FilterLogs(opts, "VerdictRecorded", challengeIdRule)
	if err != nil {
		return nil, err
	}
	return &ChainPoRVerifierVerdictRecordedIterator{contract: _ChainPoRVerifier.contract, event: "VerdictRecorded", logs: logs, sub: sub}, nil
}

// WatchVerdictRecorded is a free log subscription operation binding the contract event 0x457c2d49bb177a3347ad31b50be48291c89eedb5e2ed86f6f9f36c15d48f5737.
//
// Solidity: event VerdictRecorded(bytes32 indexed challengeId, bool success, string reason)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) WatchVerdictRecorded(opts *bind.WatchOpts, sink chan<- *ChainPoRVerifierVerdictRecorded, challengeId [][32]byte) (event.Subscription, error) {

	var challengeIdRule []interface{}
	for _, challengeIdItem := range challengeId {
		challengeIdRule = append(challengeIdRule, challengeIdItem)
	}

	logs, sub, err := _ChainPoRVerifier.contract.WatchLogs(opts, "VerdictRecorded", challengeIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChainPoRVerifierVerdictRecorded)
				if err := _ChainPoRVerifier.contract.UnpackLog(event, "VerdictRecorded", log); err != nil {
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

// ParseVerdictRecorded is a log parse operation binding the contract event 0x457c2d49bb177a3347ad31b50be48291c89eedb5e2ed86f6f9f36c15d48f5737.
//
// Solidity: event VerdictRecorded(bytes32 indexed challengeId, bool success, string reason)
func (_ChainPoRVerifier *ChainPoRVerifierFilterer) ParseVerdictRecorded(log types.Log) (*ChainPoRVerifierVerdictRecorded, error) {
	event := new(ChainPoRVerifierVerdictRecorded)
	if err := _ChainPoRVerifier.contract.UnpackLog(event, "VerdictRecorded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
