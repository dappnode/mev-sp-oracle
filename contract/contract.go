// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

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

// ContractMetaData contains all meta data concerning the Contract contract.
var ContractMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"claimedRewards\",\"type\":\"uint256\"}],\"name\":\"ClaimRewards\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"donationAmount\",\"type\":\"uint256\"}],\"name\":\"Donation\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"}],\"name\":\"SuscribeAddress\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"validatorID\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"}],\"name\":\"SuscribeValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"}],\"name\":\"UnbannValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"validatorID\",\"type\":\"uint32\"}],\"name\":\"UnsuscribeValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"newRewardsRoot\",\"type\":\"bytes32\"}],\"name\":\"UpdateRewardsRoot\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"validatorID\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newPoolRecipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"reactivateSuscription\",\"type\":\"bool\"}],\"name\":\"UpdateSuscription\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"newSuscriptionsRoot\",\"type\":\"bytes32\"}],\"name\":\"UpdateSuscriptionsRoot\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"availableBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"unbanBalance\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"merkleProof\",\"type\":\"bytes32[]\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"claimedBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_suscriptionsRoot\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_oracle\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracle\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rewardsRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32[]\",\"name\":\"validatorID\",\"type\":\"uint32[]\"},{\"internalType\":\"address[]\",\"name\":\"validatorAddress\",\"type\":\"address[]\"},{\"internalType\":\"uint32[]\",\"name\":\"blockStart\",\"type\":\"uint32[]\"}],\"name\":\"suscribeOracle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"validatorID\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"},{\"internalType\":\"bytes32[]\",\"name\":\"merkleProof\",\"type\":\"bytes32[]\"}],\"name\":\"suscribeValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"suscriptionsRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"availableBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"unbanBalance\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"merkleProof\",\"type\":\"bytes32[]\"}],\"name\":\"unbannAccount\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"validatorID\",\"type\":\"uint32\"}],\"name\":\"unsuscribeValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"newRewardsRoot\",\"type\":\"bytes32\"}],\"name\":\"updateRewardsRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"validatorID\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"newPoolRecipient\",\"type\":\"address\"}],\"name\":\"updateSuscription\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"newSuscriptionsRoot\",\"type\":\"bytes32\"}],\"name\":\"updateSuscriptionsRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"name\":\"validatorToSuscription\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"depositAddress\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"blockStart\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"blockEnd\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// ContractABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractMetaData.ABI instead.
var ContractABI = ContractMetaData.ABI

// Contract is an auto generated Go binding around an Ethereum contract.
type Contract struct {
	ContractCaller     // Read-only binding to the contract
	ContractTransactor // Write-only binding to the contract
	ContractFilterer   // Log filterer for contract events
}

// ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractSession struct {
	Contract     *Contract         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractCallerSession struct {
	Contract *ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractTransactorSession struct {
	Contract     *ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractRaw struct {
	Contract *Contract // Generic contract binding to access the raw methods on
}

// ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractCallerRaw struct {
	Contract *ContractCaller // Generic read-only contract binding to access the raw methods on
}

// ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractTransactorRaw struct {
	Contract *ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address common.Address, backend bind.ContractBackend) (*Contract, error) {
	contract, err := bindContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// NewContractCaller creates a new read-only instance of Contract, bound to a specific deployed contract.
func NewContractCaller(address common.Address, caller bind.ContractCaller) (*ContractCaller, error) {
	contract, err := bindContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractCaller{contract: contract}, nil
}

// NewContractTransactor creates a new write-only instance of Contract, bound to a specific deployed contract.
func NewContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractTransactor, error) {
	contract, err := bindContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{contract: contract}, nil
}

// NewContractFilterer creates a new log filterer instance of Contract, bound to a specific deployed contract.
func NewContractFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractFilterer, error) {
	contract, err := bindContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractFilterer{contract: contract}, nil
}

// bindContract binds a generic wrapper to an already deployed contract.
func bindContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transact(opts, method, params...)
}

// ClaimedBalance is a free data retrieval call binding the contract method 0x9886c2a5.
//
// Solidity: function claimedBalance(address ) view returns(uint256)
func (_Contract *ContractCaller) ClaimedBalance(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "claimedBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ClaimedBalance is a free data retrieval call binding the contract method 0x9886c2a5.
//
// Solidity: function claimedBalance(address ) view returns(uint256)
func (_Contract *ContractSession) ClaimedBalance(arg0 common.Address) (*big.Int, error) {
	return _Contract.Contract.ClaimedBalance(&_Contract.CallOpts, arg0)
}

// ClaimedBalance is a free data retrieval call binding the contract method 0x9886c2a5.
//
// Solidity: function claimedBalance(address ) view returns(uint256)
func (_Contract *ContractCallerSession) ClaimedBalance(arg0 common.Address) (*big.Int, error) {
	return _Contract.Contract.ClaimedBalance(&_Contract.CallOpts, arg0)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_Contract *ContractCaller) Oracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "oracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_Contract *ContractSession) Oracle() (common.Address, error) {
	return _Contract.Contract.Oracle(&_Contract.CallOpts)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_Contract *ContractCallerSession) Oracle() (common.Address, error) {
	return _Contract.Contract.Oracle(&_Contract.CallOpts)
}

// RewardsRoot is a free data retrieval call binding the contract method 0x217863b7.
//
// Solidity: function rewardsRoot() view returns(bytes32)
func (_Contract *ContractCaller) RewardsRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "rewardsRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RewardsRoot is a free data retrieval call binding the contract method 0x217863b7.
//
// Solidity: function rewardsRoot() view returns(bytes32)
func (_Contract *ContractSession) RewardsRoot() ([32]byte, error) {
	return _Contract.Contract.RewardsRoot(&_Contract.CallOpts)
}

// RewardsRoot is a free data retrieval call binding the contract method 0x217863b7.
//
// Solidity: function rewardsRoot() view returns(bytes32)
func (_Contract *ContractCallerSession) RewardsRoot() ([32]byte, error) {
	return _Contract.Contract.RewardsRoot(&_Contract.CallOpts)
}

// SuscriptionsRoot is a free data retrieval call binding the contract method 0x6b0e0b11.
//
// Solidity: function suscriptionsRoot() view returns(bytes32)
func (_Contract *ContractCaller) SuscriptionsRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "suscriptionsRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SuscriptionsRoot is a free data retrieval call binding the contract method 0x6b0e0b11.
//
// Solidity: function suscriptionsRoot() view returns(bytes32)
func (_Contract *ContractSession) SuscriptionsRoot() ([32]byte, error) {
	return _Contract.Contract.SuscriptionsRoot(&_Contract.CallOpts)
}

// SuscriptionsRoot is a free data retrieval call binding the contract method 0x6b0e0b11.
//
// Solidity: function suscriptionsRoot() view returns(bytes32)
func (_Contract *ContractCallerSession) SuscriptionsRoot() ([32]byte, error) {
	return _Contract.Contract.SuscriptionsRoot(&_Contract.CallOpts)
}

// ValidatorToSuscription is a free data retrieval call binding the contract method 0xceb8b6ea.
//
// Solidity: function validatorToSuscription(uint32 ) view returns(address depositAddress, uint32 blockStart, uint32 blockEnd, address poolRecipient)
func (_Contract *ContractCaller) ValidatorToSuscription(opts *bind.CallOpts, arg0 uint32) (struct {
	DepositAddress common.Address
	BlockStart     uint32
	BlockEnd       uint32
	PoolRecipient  common.Address
}, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "validatorToSuscription", arg0)

	outstruct := new(struct {
		DepositAddress common.Address
		BlockStart     uint32
		BlockEnd       uint32
		PoolRecipient  common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.DepositAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.BlockStart = *abi.ConvertType(out[1], new(uint32)).(*uint32)
	outstruct.BlockEnd = *abi.ConvertType(out[2], new(uint32)).(*uint32)
	outstruct.PoolRecipient = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// ValidatorToSuscription is a free data retrieval call binding the contract method 0xceb8b6ea.
//
// Solidity: function validatorToSuscription(uint32 ) view returns(address depositAddress, uint32 blockStart, uint32 blockEnd, address poolRecipient)
func (_Contract *ContractSession) ValidatorToSuscription(arg0 uint32) (struct {
	DepositAddress common.Address
	BlockStart     uint32
	BlockEnd       uint32
	PoolRecipient  common.Address
}, error) {
	return _Contract.Contract.ValidatorToSuscription(&_Contract.CallOpts, arg0)
}

// ValidatorToSuscription is a free data retrieval call binding the contract method 0xceb8b6ea.
//
// Solidity: function validatorToSuscription(uint32 ) view returns(address depositAddress, uint32 blockStart, uint32 blockEnd, address poolRecipient)
func (_Contract *ContractCallerSession) ValidatorToSuscription(arg0 uint32) (struct {
	DepositAddress common.Address
	BlockStart     uint32
	BlockEnd       uint32
	PoolRecipient  common.Address
}, error) {
	return _Contract.Contract.ValidatorToSuscription(&_Contract.CallOpts, arg0)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0x85e3b607.
//
// Solidity: function claimRewards(address depositAddress, address poolRecipient, uint256 availableBalance, uint256 unbanBalance, bytes32[] merkleProof) returns()
func (_Contract *ContractTransactor) ClaimRewards(opts *bind.TransactOpts, depositAddress common.Address, poolRecipient common.Address, availableBalance *big.Int, unbanBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "claimRewards", depositAddress, poolRecipient, availableBalance, unbanBalance, merkleProof)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0x85e3b607.
//
// Solidity: function claimRewards(address depositAddress, address poolRecipient, uint256 availableBalance, uint256 unbanBalance, bytes32[] merkleProof) returns()
func (_Contract *ContractSession) ClaimRewards(depositAddress common.Address, poolRecipient common.Address, availableBalance *big.Int, unbanBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.ClaimRewards(&_Contract.TransactOpts, depositAddress, poolRecipient, availableBalance, unbanBalance, merkleProof)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0x85e3b607.
//
// Solidity: function claimRewards(address depositAddress, address poolRecipient, uint256 availableBalance, uint256 unbanBalance, bytes32[] merkleProof) returns()
func (_Contract *ContractTransactorSession) ClaimRewards(depositAddress common.Address, poolRecipient common.Address, availableBalance *big.Int, unbanBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.ClaimRewards(&_Contract.TransactOpts, depositAddress, poolRecipient, availableBalance, unbanBalance, merkleProof)
}

// Initialize is a paid mutator transaction binding the contract method 0x6910e334.
//
// Solidity: function initialize(bytes32 _suscriptionsRoot, address _oracle) returns()
func (_Contract *ContractTransactor) Initialize(opts *bind.TransactOpts, _suscriptionsRoot [32]byte, _oracle common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "initialize", _suscriptionsRoot, _oracle)
}

// Initialize is a paid mutator transaction binding the contract method 0x6910e334.
//
// Solidity: function initialize(bytes32 _suscriptionsRoot, address _oracle) returns()
func (_Contract *ContractSession) Initialize(_suscriptionsRoot [32]byte, _oracle common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, _suscriptionsRoot, _oracle)
}

// Initialize is a paid mutator transaction binding the contract method 0x6910e334.
//
// Solidity: function initialize(bytes32 _suscriptionsRoot, address _oracle) returns()
func (_Contract *ContractTransactorSession) Initialize(_suscriptionsRoot [32]byte, _oracle common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, _suscriptionsRoot, _oracle)
}

// SuscribeOracle is a paid mutator transaction binding the contract method 0x89df9621.
//
// Solidity: function suscribeOracle(uint32[] validatorID, address[] validatorAddress, uint32[] blockStart) returns()
func (_Contract *ContractTransactor) SuscribeOracle(opts *bind.TransactOpts, validatorID []uint32, validatorAddress []common.Address, blockStart []uint32) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "suscribeOracle", validatorID, validatorAddress, blockStart)
}

// SuscribeOracle is a paid mutator transaction binding the contract method 0x89df9621.
//
// Solidity: function suscribeOracle(uint32[] validatorID, address[] validatorAddress, uint32[] blockStart) returns()
func (_Contract *ContractSession) SuscribeOracle(validatorID []uint32, validatorAddress []common.Address, blockStart []uint32) (*types.Transaction, error) {
	return _Contract.Contract.SuscribeOracle(&_Contract.TransactOpts, validatorID, validatorAddress, blockStart)
}

// SuscribeOracle is a paid mutator transaction binding the contract method 0x89df9621.
//
// Solidity: function suscribeOracle(uint32[] validatorID, address[] validatorAddress, uint32[] blockStart) returns()
func (_Contract *ContractTransactorSession) SuscribeOracle(validatorID []uint32, validatorAddress []common.Address, blockStart []uint32) (*types.Transaction, error) {
	return _Contract.Contract.SuscribeOracle(&_Contract.TransactOpts, validatorID, validatorAddress, blockStart)
}

// SuscribeValidator is a paid mutator transaction binding the contract method 0x0f895701.
//
// Solidity: function suscribeValidator(uint32 validatorID, address poolRecipient, bytes32[] merkleProof) returns()
func (_Contract *ContractTransactor) SuscribeValidator(opts *bind.TransactOpts, validatorID uint32, poolRecipient common.Address, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "suscribeValidator", validatorID, poolRecipient, merkleProof)
}

// SuscribeValidator is a paid mutator transaction binding the contract method 0x0f895701.
//
// Solidity: function suscribeValidator(uint32 validatorID, address poolRecipient, bytes32[] merkleProof) returns()
func (_Contract *ContractSession) SuscribeValidator(validatorID uint32, poolRecipient common.Address, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SuscribeValidator(&_Contract.TransactOpts, validatorID, poolRecipient, merkleProof)
}

// SuscribeValidator is a paid mutator transaction binding the contract method 0x0f895701.
//
// Solidity: function suscribeValidator(uint32 validatorID, address poolRecipient, bytes32[] merkleProof) returns()
func (_Contract *ContractTransactorSession) SuscribeValidator(validatorID uint32, poolRecipient common.Address, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SuscribeValidator(&_Contract.TransactOpts, validatorID, poolRecipient, merkleProof)
}

// UnbannAccount is a paid mutator transaction binding the contract method 0x718fb021.
//
// Solidity: function unbannAccount(address depositAddress, address poolRecipient, uint256 availableBalance, uint256 unbanBalance, bytes32[] merkleProof) payable returns()
func (_Contract *ContractTransactor) UnbannAccount(opts *bind.TransactOpts, depositAddress common.Address, poolRecipient common.Address, availableBalance *big.Int, unbanBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "unbannAccount", depositAddress, poolRecipient, availableBalance, unbanBalance, merkleProof)
}

// UnbannAccount is a paid mutator transaction binding the contract method 0x718fb021.
//
// Solidity: function unbannAccount(address depositAddress, address poolRecipient, uint256 availableBalance, uint256 unbanBalance, bytes32[] merkleProof) payable returns()
func (_Contract *ContractSession) UnbannAccount(depositAddress common.Address, poolRecipient common.Address, availableBalance *big.Int, unbanBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.UnbannAccount(&_Contract.TransactOpts, depositAddress, poolRecipient, availableBalance, unbanBalance, merkleProof)
}

// UnbannAccount is a paid mutator transaction binding the contract method 0x718fb021.
//
// Solidity: function unbannAccount(address depositAddress, address poolRecipient, uint256 availableBalance, uint256 unbanBalance, bytes32[] merkleProof) payable returns()
func (_Contract *ContractTransactorSession) UnbannAccount(depositAddress common.Address, poolRecipient common.Address, availableBalance *big.Int, unbanBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.UnbannAccount(&_Contract.TransactOpts, depositAddress, poolRecipient, availableBalance, unbanBalance, merkleProof)
}

// UnsuscribeValidator is a paid mutator transaction binding the contract method 0x2a59ed35.
//
// Solidity: function unsuscribeValidator(uint32 validatorID) returns()
func (_Contract *ContractTransactor) UnsuscribeValidator(opts *bind.TransactOpts, validatorID uint32) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "unsuscribeValidator", validatorID)
}

// UnsuscribeValidator is a paid mutator transaction binding the contract method 0x2a59ed35.
//
// Solidity: function unsuscribeValidator(uint32 validatorID) returns()
func (_Contract *ContractSession) UnsuscribeValidator(validatorID uint32) (*types.Transaction, error) {
	return _Contract.Contract.UnsuscribeValidator(&_Contract.TransactOpts, validatorID)
}

// UnsuscribeValidator is a paid mutator transaction binding the contract method 0x2a59ed35.
//
// Solidity: function unsuscribeValidator(uint32 validatorID) returns()
func (_Contract *ContractTransactorSession) UnsuscribeValidator(validatorID uint32) (*types.Transaction, error) {
	return _Contract.Contract.UnsuscribeValidator(&_Contract.TransactOpts, validatorID)
}

// UpdateRewardsRoot is a paid mutator transaction binding the contract method 0x24b5e01b.
//
// Solidity: function updateRewardsRoot(bytes32 newRewardsRoot) returns()
func (_Contract *ContractTransactor) UpdateRewardsRoot(opts *bind.TransactOpts, newRewardsRoot [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateRewardsRoot", newRewardsRoot)
}

// UpdateRewardsRoot is a paid mutator transaction binding the contract method 0x24b5e01b.
//
// Solidity: function updateRewardsRoot(bytes32 newRewardsRoot) returns()
func (_Contract *ContractSession) UpdateRewardsRoot(newRewardsRoot [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateRewardsRoot(&_Contract.TransactOpts, newRewardsRoot)
}

// UpdateRewardsRoot is a paid mutator transaction binding the contract method 0x24b5e01b.
//
// Solidity: function updateRewardsRoot(bytes32 newRewardsRoot) returns()
func (_Contract *ContractTransactorSession) UpdateRewardsRoot(newRewardsRoot [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateRewardsRoot(&_Contract.TransactOpts, newRewardsRoot)
}

// UpdateSuscription is a paid mutator transaction binding the contract method 0xd54afa37.
//
// Solidity: function updateSuscription(uint32 validatorID, address newPoolRecipient) returns()
func (_Contract *ContractTransactor) UpdateSuscription(opts *bind.TransactOpts, validatorID uint32, newPoolRecipient common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateSuscription", validatorID, newPoolRecipient)
}

// UpdateSuscription is a paid mutator transaction binding the contract method 0xd54afa37.
//
// Solidity: function updateSuscription(uint32 validatorID, address newPoolRecipient) returns()
func (_Contract *ContractSession) UpdateSuscription(validatorID uint32, newPoolRecipient common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateSuscription(&_Contract.TransactOpts, validatorID, newPoolRecipient)
}

// UpdateSuscription is a paid mutator transaction binding the contract method 0xd54afa37.
//
// Solidity: function updateSuscription(uint32 validatorID, address newPoolRecipient) returns()
func (_Contract *ContractTransactorSession) UpdateSuscription(validatorID uint32, newPoolRecipient common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateSuscription(&_Contract.TransactOpts, validatorID, newPoolRecipient)
}

// UpdateSuscriptionsRoot is a paid mutator transaction binding the contract method 0x73a52fe4.
//
// Solidity: function updateSuscriptionsRoot(bytes32 newSuscriptionsRoot) returns()
func (_Contract *ContractTransactor) UpdateSuscriptionsRoot(opts *bind.TransactOpts, newSuscriptionsRoot [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateSuscriptionsRoot", newSuscriptionsRoot)
}

// UpdateSuscriptionsRoot is a paid mutator transaction binding the contract method 0x73a52fe4.
//
// Solidity: function updateSuscriptionsRoot(bytes32 newSuscriptionsRoot) returns()
func (_Contract *ContractSession) UpdateSuscriptionsRoot(newSuscriptionsRoot [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateSuscriptionsRoot(&_Contract.TransactOpts, newSuscriptionsRoot)
}

// UpdateSuscriptionsRoot is a paid mutator transaction binding the contract method 0x73a52fe4.
//
// Solidity: function updateSuscriptionsRoot(bytes32 newSuscriptionsRoot) returns()
func (_Contract *ContractTransactorSession) UpdateSuscriptionsRoot(newSuscriptionsRoot [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateSuscriptionsRoot(&_Contract.TransactOpts, newSuscriptionsRoot)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Contract *ContractTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Contract *ContractSession) Receive() (*types.Transaction, error) {
	return _Contract.Contract.Receive(&_Contract.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Contract *ContractTransactorSession) Receive() (*types.Transaction, error) {
	return _Contract.Contract.Receive(&_Contract.TransactOpts)
}

// ContractClaimRewardsIterator is returned from FilterClaimRewards and is used to iterate over the raw logs and unpacked data for ClaimRewards events raised by the Contract contract.
type ContractClaimRewardsIterator struct {
	Event *ContractClaimRewards // Event containing the contract specifics and raw log

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
func (it *ContractClaimRewardsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractClaimRewards)
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
		it.Event = new(ContractClaimRewards)
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
func (it *ContractClaimRewardsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractClaimRewardsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractClaimRewards represents a ClaimRewards event raised by the Contract contract.
type ContractClaimRewards struct {
	DepositAddress common.Address
	PoolRecipient  common.Address
	ClaimedRewards *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterClaimRewards is a free log retrieval operation binding the contract event 0x9aa05b3d70a9e3e2f004f039648839560576334fb45c81f91b6db03ad9e2efc9.
//
// Solidity: event ClaimRewards(address depositAddress, address poolRecipient, uint256 claimedRewards)
func (_Contract *ContractFilterer) FilterClaimRewards(opts *bind.FilterOpts) (*ContractClaimRewardsIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "ClaimRewards")
	if err != nil {
		return nil, err
	}
	return &ContractClaimRewardsIterator{contract: _Contract.contract, event: "ClaimRewards", logs: logs, sub: sub}, nil
}

// WatchClaimRewards is a free log subscription operation binding the contract event 0x9aa05b3d70a9e3e2f004f039648839560576334fb45c81f91b6db03ad9e2efc9.
//
// Solidity: event ClaimRewards(address depositAddress, address poolRecipient, uint256 claimedRewards)
func (_Contract *ContractFilterer) WatchClaimRewards(opts *bind.WatchOpts, sink chan<- *ContractClaimRewards) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "ClaimRewards")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractClaimRewards)
				if err := _Contract.contract.UnpackLog(event, "ClaimRewards", log); err != nil {
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

// ParseClaimRewards is a log parse operation binding the contract event 0x9aa05b3d70a9e3e2f004f039648839560576334fb45c81f91b6db03ad9e2efc9.
//
// Solidity: event ClaimRewards(address depositAddress, address poolRecipient, uint256 claimedRewards)
func (_Contract *ContractFilterer) ParseClaimRewards(log types.Log) (*ContractClaimRewards, error) {
	event := new(ContractClaimRewards)
	if err := _Contract.contract.UnpackLog(event, "ClaimRewards", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractDonationIterator is returned from FilterDonation and is used to iterate over the raw logs and unpacked data for Donation events raised by the Contract contract.
type ContractDonationIterator struct {
	Event *ContractDonation // Event containing the contract specifics and raw log

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
func (it *ContractDonationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractDonation)
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
		it.Event = new(ContractDonation)
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
func (it *ContractDonationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractDonationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractDonation represents a Donation event raised by the Contract contract.
type ContractDonation struct {
	DonationAmount *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterDonation is a free log retrieval operation binding the contract event 0x4ad6d3cd6f9acbf734536f9812be14aded33cc8c4ae2cfbf3ff9a0a4acb33748.
//
// Solidity: event Donation(uint256 donationAmount)
func (_Contract *ContractFilterer) FilterDonation(opts *bind.FilterOpts) (*ContractDonationIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Donation")
	if err != nil {
		return nil, err
	}
	return &ContractDonationIterator{contract: _Contract.contract, event: "Donation", logs: logs, sub: sub}, nil
}

// WatchDonation is a free log subscription operation binding the contract event 0x4ad6d3cd6f9acbf734536f9812be14aded33cc8c4ae2cfbf3ff9a0a4acb33748.
//
// Solidity: event Donation(uint256 donationAmount)
func (_Contract *ContractFilterer) WatchDonation(opts *bind.WatchOpts, sink chan<- *ContractDonation) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Donation")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractDonation)
				if err := _Contract.contract.UnpackLog(event, "Donation", log); err != nil {
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

// ParseDonation is a log parse operation binding the contract event 0x4ad6d3cd6f9acbf734536f9812be14aded33cc8c4ae2cfbf3ff9a0a4acb33748.
//
// Solidity: event Donation(uint256 donationAmount)
func (_Contract *ContractFilterer) ParseDonation(log types.Log) (*ContractDonation, error) {
	event := new(ContractDonation)
	if err := _Contract.contract.UnpackLog(event, "Donation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Contract contract.
type ContractInitializedIterator struct {
	Event *ContractInitialized // Event containing the contract specifics and raw log

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
func (it *ContractInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractInitialized)
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
		it.Event = new(ContractInitialized)
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
func (it *ContractInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractInitialized represents a Initialized event raised by the Contract contract.
type ContractInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Contract *ContractFilterer) FilterInitialized(opts *bind.FilterOpts) (*ContractInitializedIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ContractInitializedIterator{contract: _Contract.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Contract *ContractFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ContractInitialized) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractInitialized)
				if err := _Contract.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Contract *ContractFilterer) ParseInitialized(log types.Log) (*ContractInitialized, error) {
	event := new(ContractInitialized)
	if err := _Contract.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractSuscribeAddressIterator is returned from FilterSuscribeAddress and is used to iterate over the raw logs and unpacked data for SuscribeAddress events raised by the Contract contract.
type ContractSuscribeAddressIterator struct {
	Event *ContractSuscribeAddress // Event containing the contract specifics and raw log

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
func (it *ContractSuscribeAddressIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractSuscribeAddress)
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
		it.Event = new(ContractSuscribeAddress)
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
func (it *ContractSuscribeAddressIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractSuscribeAddressIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractSuscribeAddress represents a SuscribeAddress event raised by the Contract contract.
type ContractSuscribeAddress struct {
	DepositAddress common.Address
	PoolRecipient  common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterSuscribeAddress is a free log retrieval operation binding the contract event 0xff24fcf46d5d7d9ea0b371f53d23c5ab747001111a868a6c7ae00410fe92a9dd.
//
// Solidity: event SuscribeAddress(address depositAddress, address poolRecipient)
func (_Contract *ContractFilterer) FilterSuscribeAddress(opts *bind.FilterOpts) (*ContractSuscribeAddressIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "SuscribeAddress")
	if err != nil {
		return nil, err
	}
	return &ContractSuscribeAddressIterator{contract: _Contract.contract, event: "SuscribeAddress", logs: logs, sub: sub}, nil
}

// WatchSuscribeAddress is a free log subscription operation binding the contract event 0xff24fcf46d5d7d9ea0b371f53d23c5ab747001111a868a6c7ae00410fe92a9dd.
//
// Solidity: event SuscribeAddress(address depositAddress, address poolRecipient)
func (_Contract *ContractFilterer) WatchSuscribeAddress(opts *bind.WatchOpts, sink chan<- *ContractSuscribeAddress) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "SuscribeAddress")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractSuscribeAddress)
				if err := _Contract.contract.UnpackLog(event, "SuscribeAddress", log); err != nil {
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

// ParseSuscribeAddress is a log parse operation binding the contract event 0xff24fcf46d5d7d9ea0b371f53d23c5ab747001111a868a6c7ae00410fe92a9dd.
//
// Solidity: event SuscribeAddress(address depositAddress, address poolRecipient)
func (_Contract *ContractFilterer) ParseSuscribeAddress(log types.Log) (*ContractSuscribeAddress, error) {
	event := new(ContractSuscribeAddress)
	if err := _Contract.contract.UnpackLog(event, "SuscribeAddress", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractSuscribeValidatorIterator is returned from FilterSuscribeValidator and is used to iterate over the raw logs and unpacked data for SuscribeValidator events raised by the Contract contract.
type ContractSuscribeValidatorIterator struct {
	Event *ContractSuscribeValidator // Event containing the contract specifics and raw log

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
func (it *ContractSuscribeValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractSuscribeValidator)
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
		it.Event = new(ContractSuscribeValidator)
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
func (it *ContractSuscribeValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractSuscribeValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractSuscribeValidator represents a SuscribeValidator event raised by the Contract contract.
type ContractSuscribeValidator struct {
	ValidatorID    uint32
	DepositAddress common.Address
	PoolRecipient  common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterSuscribeValidator is a free log retrieval operation binding the contract event 0x43dbebac36cbacd564dbc3f59c1d698b53c31a3f3902626d9cc890d682fec453.
//
// Solidity: event SuscribeValidator(uint32 validatorID, address depositAddress, address poolRecipient)
func (_Contract *ContractFilterer) FilterSuscribeValidator(opts *bind.FilterOpts) (*ContractSuscribeValidatorIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "SuscribeValidator")
	if err != nil {
		return nil, err
	}
	return &ContractSuscribeValidatorIterator{contract: _Contract.contract, event: "SuscribeValidator", logs: logs, sub: sub}, nil
}

// WatchSuscribeValidator is a free log subscription operation binding the contract event 0x43dbebac36cbacd564dbc3f59c1d698b53c31a3f3902626d9cc890d682fec453.
//
// Solidity: event SuscribeValidator(uint32 validatorID, address depositAddress, address poolRecipient)
func (_Contract *ContractFilterer) WatchSuscribeValidator(opts *bind.WatchOpts, sink chan<- *ContractSuscribeValidator) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "SuscribeValidator")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractSuscribeValidator)
				if err := _Contract.contract.UnpackLog(event, "SuscribeValidator", log); err != nil {
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

// ParseSuscribeValidator is a log parse operation binding the contract event 0x43dbebac36cbacd564dbc3f59c1d698b53c31a3f3902626d9cc890d682fec453.
//
// Solidity: event SuscribeValidator(uint32 validatorID, address depositAddress, address poolRecipient)
func (_Contract *ContractFilterer) ParseSuscribeValidator(log types.Log) (*ContractSuscribeValidator, error) {
	event := new(ContractSuscribeValidator)
	if err := _Contract.contract.UnpackLog(event, "SuscribeValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUnbannValidatorIterator is returned from FilterUnbannValidator and is used to iterate over the raw logs and unpacked data for UnbannValidator events raised by the Contract contract.
type ContractUnbannValidatorIterator struct {
	Event *ContractUnbannValidator // Event containing the contract specifics and raw log

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
func (it *ContractUnbannValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUnbannValidator)
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
		it.Event = new(ContractUnbannValidator)
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
func (it *ContractUnbannValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUnbannValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUnbannValidator represents a UnbannValidator event raised by the Contract contract.
type ContractUnbannValidator struct {
	DepositAddress common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUnbannValidator is a free log retrieval operation binding the contract event 0x56593a92592b232c4c3e1f4b5ff860c04f86a34b2d30d55b5a6fb04b60001974.
//
// Solidity: event UnbannValidator(address depositAddress)
func (_Contract *ContractFilterer) FilterUnbannValidator(opts *bind.FilterOpts) (*ContractUnbannValidatorIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UnbannValidator")
	if err != nil {
		return nil, err
	}
	return &ContractUnbannValidatorIterator{contract: _Contract.contract, event: "UnbannValidator", logs: logs, sub: sub}, nil
}

// WatchUnbannValidator is a free log subscription operation binding the contract event 0x56593a92592b232c4c3e1f4b5ff860c04f86a34b2d30d55b5a6fb04b60001974.
//
// Solidity: event UnbannValidator(address depositAddress)
func (_Contract *ContractFilterer) WatchUnbannValidator(opts *bind.WatchOpts, sink chan<- *ContractUnbannValidator) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UnbannValidator")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUnbannValidator)
				if err := _Contract.contract.UnpackLog(event, "UnbannValidator", log); err != nil {
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

// ParseUnbannValidator is a log parse operation binding the contract event 0x56593a92592b232c4c3e1f4b5ff860c04f86a34b2d30d55b5a6fb04b60001974.
//
// Solidity: event UnbannValidator(address depositAddress)
func (_Contract *ContractFilterer) ParseUnbannValidator(log types.Log) (*ContractUnbannValidator, error) {
	event := new(ContractUnbannValidator)
	if err := _Contract.contract.UnpackLog(event, "UnbannValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUnsuscribeValidatorIterator is returned from FilterUnsuscribeValidator and is used to iterate over the raw logs and unpacked data for UnsuscribeValidator events raised by the Contract contract.
type ContractUnsuscribeValidatorIterator struct {
	Event *ContractUnsuscribeValidator // Event containing the contract specifics and raw log

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
func (it *ContractUnsuscribeValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUnsuscribeValidator)
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
		it.Event = new(ContractUnsuscribeValidator)
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
func (it *ContractUnsuscribeValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUnsuscribeValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUnsuscribeValidator represents a UnsuscribeValidator event raised by the Contract contract.
type ContractUnsuscribeValidator struct {
	ValidatorID uint32
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterUnsuscribeValidator is a free log retrieval operation binding the contract event 0x9230fd8b831c210d12b86495abc019a2639010284d7583fbb378a7cc044d11bc.
//
// Solidity: event UnsuscribeValidator(uint32 validatorID)
func (_Contract *ContractFilterer) FilterUnsuscribeValidator(opts *bind.FilterOpts) (*ContractUnsuscribeValidatorIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UnsuscribeValidator")
	if err != nil {
		return nil, err
	}
	return &ContractUnsuscribeValidatorIterator{contract: _Contract.contract, event: "UnsuscribeValidator", logs: logs, sub: sub}, nil
}

// WatchUnsuscribeValidator is a free log subscription operation binding the contract event 0x9230fd8b831c210d12b86495abc019a2639010284d7583fbb378a7cc044d11bc.
//
// Solidity: event UnsuscribeValidator(uint32 validatorID)
func (_Contract *ContractFilterer) WatchUnsuscribeValidator(opts *bind.WatchOpts, sink chan<- *ContractUnsuscribeValidator) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UnsuscribeValidator")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUnsuscribeValidator)
				if err := _Contract.contract.UnpackLog(event, "UnsuscribeValidator", log); err != nil {
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

// ParseUnsuscribeValidator is a log parse operation binding the contract event 0x9230fd8b831c210d12b86495abc019a2639010284d7583fbb378a7cc044d11bc.
//
// Solidity: event UnsuscribeValidator(uint32 validatorID)
func (_Contract *ContractFilterer) ParseUnsuscribeValidator(log types.Log) (*ContractUnsuscribeValidator, error) {
	event := new(ContractUnsuscribeValidator)
	if err := _Contract.contract.UnpackLog(event, "UnsuscribeValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateRewardsRootIterator is returned from FilterUpdateRewardsRoot and is used to iterate over the raw logs and unpacked data for UpdateRewardsRoot events raised by the Contract contract.
type ContractUpdateRewardsRootIterator struct {
	Event *ContractUpdateRewardsRoot // Event containing the contract specifics and raw log

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
func (it *ContractUpdateRewardsRootIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateRewardsRoot)
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
		it.Event = new(ContractUpdateRewardsRoot)
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
func (it *ContractUpdateRewardsRootIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateRewardsRootIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateRewardsRoot represents a UpdateRewardsRoot event raised by the Contract contract.
type ContractUpdateRewardsRoot struct {
	NewRewardsRoot [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpdateRewardsRoot is a free log retrieval operation binding the contract event 0xe7e71285729271e0243f632019c0033071db647a3b6eec10ec9f14073975f720.
//
// Solidity: event UpdateRewardsRoot(bytes32 newRewardsRoot)
func (_Contract *ContractFilterer) FilterUpdateRewardsRoot(opts *bind.FilterOpts) (*ContractUpdateRewardsRootIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateRewardsRoot")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateRewardsRootIterator{contract: _Contract.contract, event: "UpdateRewardsRoot", logs: logs, sub: sub}, nil
}

// WatchUpdateRewardsRoot is a free log subscription operation binding the contract event 0xe7e71285729271e0243f632019c0033071db647a3b6eec10ec9f14073975f720.
//
// Solidity: event UpdateRewardsRoot(bytes32 newRewardsRoot)
func (_Contract *ContractFilterer) WatchUpdateRewardsRoot(opts *bind.WatchOpts, sink chan<- *ContractUpdateRewardsRoot) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateRewardsRoot")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateRewardsRoot)
				if err := _Contract.contract.UnpackLog(event, "UpdateRewardsRoot", log); err != nil {
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

// ParseUpdateRewardsRoot is a log parse operation binding the contract event 0xe7e71285729271e0243f632019c0033071db647a3b6eec10ec9f14073975f720.
//
// Solidity: event UpdateRewardsRoot(bytes32 newRewardsRoot)
func (_Contract *ContractFilterer) ParseUpdateRewardsRoot(log types.Log) (*ContractUpdateRewardsRoot, error) {
	event := new(ContractUpdateRewardsRoot)
	if err := _Contract.contract.UnpackLog(event, "UpdateRewardsRoot", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateSuscriptionIterator is returned from FilterUpdateSuscription and is used to iterate over the raw logs and unpacked data for UpdateSuscription events raised by the Contract contract.
type ContractUpdateSuscriptionIterator struct {
	Event *ContractUpdateSuscription // Event containing the contract specifics and raw log

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
func (it *ContractUpdateSuscriptionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateSuscription)
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
		it.Event = new(ContractUpdateSuscription)
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
func (it *ContractUpdateSuscriptionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateSuscriptionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateSuscription represents a UpdateSuscription event raised by the Contract contract.
type ContractUpdateSuscription struct {
	ValidatorID           uint32
	NewPoolRecipient      common.Address
	ReactivateSuscription bool
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterUpdateSuscription is a free log retrieval operation binding the contract event 0xeb716c3cbbbc150282a432f6f6e7dc6c88e6e99120871f98e73ee1ed5a33c642.
//
// Solidity: event UpdateSuscription(uint32 validatorID, address newPoolRecipient, bool reactivateSuscription)
func (_Contract *ContractFilterer) FilterUpdateSuscription(opts *bind.FilterOpts) (*ContractUpdateSuscriptionIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateSuscription")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateSuscriptionIterator{contract: _Contract.contract, event: "UpdateSuscription", logs: logs, sub: sub}, nil
}

// WatchUpdateSuscription is a free log subscription operation binding the contract event 0xeb716c3cbbbc150282a432f6f6e7dc6c88e6e99120871f98e73ee1ed5a33c642.
//
// Solidity: event UpdateSuscription(uint32 validatorID, address newPoolRecipient, bool reactivateSuscription)
func (_Contract *ContractFilterer) WatchUpdateSuscription(opts *bind.WatchOpts, sink chan<- *ContractUpdateSuscription) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateSuscription")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateSuscription)
				if err := _Contract.contract.UnpackLog(event, "UpdateSuscription", log); err != nil {
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

// ParseUpdateSuscription is a log parse operation binding the contract event 0xeb716c3cbbbc150282a432f6f6e7dc6c88e6e99120871f98e73ee1ed5a33c642.
//
// Solidity: event UpdateSuscription(uint32 validatorID, address newPoolRecipient, bool reactivateSuscription)
func (_Contract *ContractFilterer) ParseUpdateSuscription(log types.Log) (*ContractUpdateSuscription, error) {
	event := new(ContractUpdateSuscription)
	if err := _Contract.contract.UnpackLog(event, "UpdateSuscription", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateSuscriptionsRootIterator is returned from FilterUpdateSuscriptionsRoot and is used to iterate over the raw logs and unpacked data for UpdateSuscriptionsRoot events raised by the Contract contract.
type ContractUpdateSuscriptionsRootIterator struct {
	Event *ContractUpdateSuscriptionsRoot // Event containing the contract specifics and raw log

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
func (it *ContractUpdateSuscriptionsRootIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateSuscriptionsRoot)
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
		it.Event = new(ContractUpdateSuscriptionsRoot)
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
func (it *ContractUpdateSuscriptionsRootIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateSuscriptionsRootIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateSuscriptionsRoot represents a UpdateSuscriptionsRoot event raised by the Contract contract.
type ContractUpdateSuscriptionsRoot struct {
	NewSuscriptionsRoot [32]byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterUpdateSuscriptionsRoot is a free log retrieval operation binding the contract event 0x0f6d5035f2311661ba34ec6ed900efc679ac72868eaba7df79a217881092f613.
//
// Solidity: event UpdateSuscriptionsRoot(bytes32 newSuscriptionsRoot)
func (_Contract *ContractFilterer) FilterUpdateSuscriptionsRoot(opts *bind.FilterOpts) (*ContractUpdateSuscriptionsRootIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateSuscriptionsRoot")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateSuscriptionsRootIterator{contract: _Contract.contract, event: "UpdateSuscriptionsRoot", logs: logs, sub: sub}, nil
}

// WatchUpdateSuscriptionsRoot is a free log subscription operation binding the contract event 0x0f6d5035f2311661ba34ec6ed900efc679ac72868eaba7df79a217881092f613.
//
// Solidity: event UpdateSuscriptionsRoot(bytes32 newSuscriptionsRoot)
func (_Contract *ContractFilterer) WatchUpdateSuscriptionsRoot(opts *bind.WatchOpts, sink chan<- *ContractUpdateSuscriptionsRoot) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateSuscriptionsRoot")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateSuscriptionsRoot)
				if err := _Contract.contract.UnpackLog(event, "UpdateSuscriptionsRoot", log); err != nil {
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

// ParseUpdateSuscriptionsRoot is a log parse operation binding the contract event 0x0f6d5035f2311661ba34ec6ed900efc679ac72868eaba7df79a217881092f613.
//
// Solidity: event UpdateSuscriptionsRoot(bytes32 newSuscriptionsRoot)
func (_Contract *ContractFilterer) ParseUpdateSuscriptionsRoot(log types.Log) (*ContractUpdateSuscriptionsRoot, error) {
	event := new(ContractUpdateSuscriptionsRoot)
	if err := _Contract.contract.UnpackLog(event, "UpdateSuscriptionsRoot", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
