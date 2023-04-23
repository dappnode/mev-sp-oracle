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
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"withdrawalAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rewardAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"claimableBalance\",\"type\":\"uint256\"}],\"name\":\"ClaimRewards\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"donationAmount\",\"type\":\"uint256\"}],\"name\":\"EtherReceived\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"withdrawalAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"poolRecipient\",\"type\":\"address\"}],\"name\":\"SetRewardRecipient\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"subscriptionCollateral\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"validatorID\",\"type\":\"uint64\"}],\"name\":\"SubscribeValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"validatorID\",\"type\":\"uint64\"}],\"name\":\"UnsubscribeValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newOracle\",\"type\":\"address\"}],\"name\":\"UpdateOracle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"newRewardsRoot\",\"type\":\"bytes32\"}],\"name\":\"UpdateRewardsRoot\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newSubscriptionCollateral\",\"type\":\"uint256\"}],\"name\":\"UpdateSubscriptionCollateral\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"withdrawalAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"accumulatedBalance\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"merkleProof\",\"type\":\"bytes32[]\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"claimedBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oracle\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_subscriptionCollateral\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracle\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"rewardRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rewardsRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rewardAddress\",\"type\":\"address\"}],\"name\":\"setRewardRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"validatorID\",\"type\":\"uint64\"}],\"name\":\"subscribeValidator\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"subscriptionCollateral\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"validatorID\",\"type\":\"uint64\"}],\"name\":\"unsubscribeValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newSubscriptionCollateral\",\"type\":\"uint256\"}],\"name\":\"updateCollateral\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOracle\",\"type\":\"address\"}],\"name\":\"updateOracle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"newRewardsRoot\",\"type\":\"bytes32\"}],\"name\":\"updateRewardsRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractSession) Owner() (common.Address, error) {
	return _Contract.Contract.Owner(&_Contract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractCallerSession) Owner() (common.Address, error) {
	return _Contract.Contract.Owner(&_Contract.CallOpts)
}

// RewardRecipient is a free data retrieval call binding the contract method 0xf372c0c9.
//
// Solidity: function rewardRecipient(address ) view returns(address)
func (_Contract *ContractCaller) RewardRecipient(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "rewardRecipient", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RewardRecipient is a free data retrieval call binding the contract method 0xf372c0c9.
//
// Solidity: function rewardRecipient(address ) view returns(address)
func (_Contract *ContractSession) RewardRecipient(arg0 common.Address) (common.Address, error) {
	return _Contract.Contract.RewardRecipient(&_Contract.CallOpts, arg0)
}

// RewardRecipient is a free data retrieval call binding the contract method 0xf372c0c9.
//
// Solidity: function rewardRecipient(address ) view returns(address)
func (_Contract *ContractCallerSession) RewardRecipient(arg0 common.Address) (common.Address, error) {
	return _Contract.Contract.RewardRecipient(&_Contract.CallOpts, arg0)
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

// SubscriptionCollateral is a free data retrieval call binding the contract method 0xf93558e3.
//
// Solidity: function subscriptionCollateral() view returns(uint256)
func (_Contract *ContractCaller) SubscriptionCollateral(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "subscriptionCollateral")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SubscriptionCollateral is a free data retrieval call binding the contract method 0xf93558e3.
//
// Solidity: function subscriptionCollateral() view returns(uint256)
func (_Contract *ContractSession) SubscriptionCollateral() (*big.Int, error) {
	return _Contract.Contract.SubscriptionCollateral(&_Contract.CallOpts)
}

// SubscriptionCollateral is a free data retrieval call binding the contract method 0xf93558e3.
//
// Solidity: function subscriptionCollateral() view returns(uint256)
func (_Contract *ContractCallerSession) SubscriptionCollateral() (*big.Int, error) {
	return _Contract.Contract.SubscriptionCollateral(&_Contract.CallOpts)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xd64bc331.
//
// Solidity: function claimRewards(address withdrawalAddress, uint256 accumulatedBalance, bytes32[] merkleProof) returns()
func (_Contract *ContractTransactor) ClaimRewards(opts *bind.TransactOpts, withdrawalAddress common.Address, accumulatedBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "claimRewards", withdrawalAddress, accumulatedBalance, merkleProof)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xd64bc331.
//
// Solidity: function claimRewards(address withdrawalAddress, uint256 accumulatedBalance, bytes32[] merkleProof) returns()
func (_Contract *ContractSession) ClaimRewards(withdrawalAddress common.Address, accumulatedBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.ClaimRewards(&_Contract.TransactOpts, withdrawalAddress, accumulatedBalance, merkleProof)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xd64bc331.
//
// Solidity: function claimRewards(address withdrawalAddress, uint256 accumulatedBalance, bytes32[] merkleProof) returns()
func (_Contract *ContractTransactorSession) ClaimRewards(withdrawalAddress common.Address, accumulatedBalance *big.Int, merkleProof [][32]byte) (*types.Transaction, error) {
	return _Contract.Contract.ClaimRewards(&_Contract.TransactOpts, withdrawalAddress, accumulatedBalance, merkleProof)
}

// Initialize is a paid mutator transaction binding the contract method 0xcd6dc687.
//
// Solidity: function initialize(address _oracle, uint256 _subscriptionCollateral) returns()
func (_Contract *ContractTransactor) Initialize(opts *bind.TransactOpts, _oracle common.Address, _subscriptionCollateral *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "initialize", _oracle, _subscriptionCollateral)
}

// Initialize is a paid mutator transaction binding the contract method 0xcd6dc687.
//
// Solidity: function initialize(address _oracle, uint256 _subscriptionCollateral) returns()
func (_Contract *ContractSession) Initialize(_oracle common.Address, _subscriptionCollateral *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, _oracle, _subscriptionCollateral)
}

// Initialize is a paid mutator transaction binding the contract method 0xcd6dc687.
//
// Solidity: function initialize(address _oracle, uint256 _subscriptionCollateral) returns()
func (_Contract *ContractTransactorSession) Initialize(_oracle common.Address, _subscriptionCollateral *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, _oracle, _subscriptionCollateral)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractSession) RenounceOwnership() (*types.Transaction, error) {
	return _Contract.Contract.RenounceOwnership(&_Contract.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Contract.Contract.RenounceOwnership(&_Contract.TransactOpts)
}

// SetRewardRecipient is a paid mutator transaction binding the contract method 0xe521136f.
//
// Solidity: function setRewardRecipient(address rewardAddress) returns()
func (_Contract *ContractTransactor) SetRewardRecipient(opts *bind.TransactOpts, rewardAddress common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setRewardRecipient", rewardAddress)
}

// SetRewardRecipient is a paid mutator transaction binding the contract method 0xe521136f.
//
// Solidity: function setRewardRecipient(address rewardAddress) returns()
func (_Contract *ContractSession) SetRewardRecipient(rewardAddress common.Address) (*types.Transaction, error) {
	return _Contract.Contract.SetRewardRecipient(&_Contract.TransactOpts, rewardAddress)
}

// SetRewardRecipient is a paid mutator transaction binding the contract method 0xe521136f.
//
// Solidity: function setRewardRecipient(address rewardAddress) returns()
func (_Contract *ContractTransactorSession) SetRewardRecipient(rewardAddress common.Address) (*types.Transaction, error) {
	return _Contract.Contract.SetRewardRecipient(&_Contract.TransactOpts, rewardAddress)
}

// SubscribeValidator is a paid mutator transaction binding the contract method 0xb9cd552e.
//
// Solidity: function subscribeValidator(uint64 validatorID) payable returns()
func (_Contract *ContractTransactor) SubscribeValidator(opts *bind.TransactOpts, validatorID uint64) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "subscribeValidator", validatorID)
}

// SubscribeValidator is a paid mutator transaction binding the contract method 0xb9cd552e.
//
// Solidity: function subscribeValidator(uint64 validatorID) payable returns()
func (_Contract *ContractSession) SubscribeValidator(validatorID uint64) (*types.Transaction, error) {
	return _Contract.Contract.SubscribeValidator(&_Contract.TransactOpts, validatorID)
}

// SubscribeValidator is a paid mutator transaction binding the contract method 0xb9cd552e.
//
// Solidity: function subscribeValidator(uint64 validatorID) payable returns()
func (_Contract *ContractTransactorSession) SubscribeValidator(validatorID uint64) (*types.Transaction, error) {
	return _Contract.Contract.SubscribeValidator(&_Contract.TransactOpts, validatorID)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.TransferOwnership(&_Contract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.TransferOwnership(&_Contract.TransactOpts, newOwner)
}

// UnsubscribeValidator is a paid mutator transaction binding the contract method 0xc1542c52.
//
// Solidity: function unsubscribeValidator(uint64 validatorID) returns()
func (_Contract *ContractTransactor) UnsubscribeValidator(opts *bind.TransactOpts, validatorID uint64) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "unsubscribeValidator", validatorID)
}

// UnsubscribeValidator is a paid mutator transaction binding the contract method 0xc1542c52.
//
// Solidity: function unsubscribeValidator(uint64 validatorID) returns()
func (_Contract *ContractSession) UnsubscribeValidator(validatorID uint64) (*types.Transaction, error) {
	return _Contract.Contract.UnsubscribeValidator(&_Contract.TransactOpts, validatorID)
}

// UnsubscribeValidator is a paid mutator transaction binding the contract method 0xc1542c52.
//
// Solidity: function unsubscribeValidator(uint64 validatorID) returns()
func (_Contract *ContractTransactorSession) UnsubscribeValidator(validatorID uint64) (*types.Transaction, error) {
	return _Contract.Contract.UnsubscribeValidator(&_Contract.TransactOpts, validatorID)
}

// UpdateCollateral is a paid mutator transaction binding the contract method 0x6721de26.
//
// Solidity: function updateCollateral(uint256 newSubscriptionCollateral) returns()
func (_Contract *ContractTransactor) UpdateCollateral(opts *bind.TransactOpts, newSubscriptionCollateral *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateCollateral", newSubscriptionCollateral)
}

// UpdateCollateral is a paid mutator transaction binding the contract method 0x6721de26.
//
// Solidity: function updateCollateral(uint256 newSubscriptionCollateral) returns()
func (_Contract *ContractSession) UpdateCollateral(newSubscriptionCollateral *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateCollateral(&_Contract.TransactOpts, newSubscriptionCollateral)
}

// UpdateCollateral is a paid mutator transaction binding the contract method 0x6721de26.
//
// Solidity: function updateCollateral(uint256 newSubscriptionCollateral) returns()
func (_Contract *ContractTransactorSession) UpdateCollateral(newSubscriptionCollateral *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateCollateral(&_Contract.TransactOpts, newSubscriptionCollateral)
}

// UpdateOracle is a paid mutator transaction binding the contract method 0x1cb44dfc.
//
// Solidity: function updateOracle(address newOracle) returns()
func (_Contract *ContractTransactor) UpdateOracle(opts *bind.TransactOpts, newOracle common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateOracle", newOracle)
}

// UpdateOracle is a paid mutator transaction binding the contract method 0x1cb44dfc.
//
// Solidity: function updateOracle(address newOracle) returns()
func (_Contract *ContractSession) UpdateOracle(newOracle common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateOracle(&_Contract.TransactOpts, newOracle)
}

// UpdateOracle is a paid mutator transaction binding the contract method 0x1cb44dfc.
//
// Solidity: function updateOracle(address newOracle) returns()
func (_Contract *ContractTransactorSession) UpdateOracle(newOracle common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateOracle(&_Contract.TransactOpts, newOracle)
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

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Contract *ContractTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _Contract.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Contract *ContractSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Contract.Contract.Fallback(&_Contract.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Contract *ContractTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Contract.Contract.Fallback(&_Contract.TransactOpts, calldata)
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
	WithdrawalAddress common.Address
	RewardAddress     common.Address
	ClaimableBalance  *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterClaimRewards is a free log retrieval operation binding the contract event 0x9aa05b3d70a9e3e2f004f039648839560576334fb45c81f91b6db03ad9e2efc9.
//
// Solidity: event ClaimRewards(address withdrawalAddress, address rewardAddress, uint256 claimableBalance)
func (_Contract *ContractFilterer) FilterClaimRewards(opts *bind.FilterOpts) (*ContractClaimRewardsIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "ClaimRewards")
	if err != nil {
		return nil, err
	}
	return &ContractClaimRewardsIterator{contract: _Contract.contract, event: "ClaimRewards", logs: logs, sub: sub}, nil
}

// WatchClaimRewards is a free log subscription operation binding the contract event 0x9aa05b3d70a9e3e2f004f039648839560576334fb45c81f91b6db03ad9e2efc9.
//
// Solidity: event ClaimRewards(address withdrawalAddress, address rewardAddress, uint256 claimableBalance)
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
// Solidity: event ClaimRewards(address withdrawalAddress, address rewardAddress, uint256 claimableBalance)
func (_Contract *ContractFilterer) ParseClaimRewards(log types.Log) (*ContractClaimRewards, error) {
	event := new(ContractClaimRewards)
	if err := _Contract.contract.UnpackLog(event, "ClaimRewards", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractEtherReceivedIterator is returned from FilterEtherReceived and is used to iterate over the raw logs and unpacked data for EtherReceived events raised by the Contract contract.
type ContractEtherReceivedIterator struct {
	Event *ContractEtherReceived // Event containing the contract specifics and raw log

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
func (it *ContractEtherReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractEtherReceived)
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
		it.Event = new(ContractEtherReceived)
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
func (it *ContractEtherReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractEtherReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractEtherReceived represents a EtherReceived event raised by the Contract contract.
type ContractEtherReceived struct {
	Sender         common.Address
	DonationAmount *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterEtherReceived is a free log retrieval operation binding the contract event 0x1e57e3bb474320be3d2c77138f75b7c3941292d647f5f9634e33a8e94e0e069b.
//
// Solidity: event EtherReceived(address sender, uint256 donationAmount)
func (_Contract *ContractFilterer) FilterEtherReceived(opts *bind.FilterOpts) (*ContractEtherReceivedIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "EtherReceived")
	if err != nil {
		return nil, err
	}
	return &ContractEtherReceivedIterator{contract: _Contract.contract, event: "EtherReceived", logs: logs, sub: sub}, nil
}

// WatchEtherReceived is a free log subscription operation binding the contract event 0x1e57e3bb474320be3d2c77138f75b7c3941292d647f5f9634e33a8e94e0e069b.
//
// Solidity: event EtherReceived(address sender, uint256 donationAmount)
func (_Contract *ContractFilterer) WatchEtherReceived(opts *bind.WatchOpts, sink chan<- *ContractEtherReceived) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "EtherReceived")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractEtherReceived)
				if err := _Contract.contract.UnpackLog(event, "EtherReceived", log); err != nil {
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

// ParseEtherReceived is a log parse operation binding the contract event 0x1e57e3bb474320be3d2c77138f75b7c3941292d647f5f9634e33a8e94e0e069b.
//
// Solidity: event EtherReceived(address sender, uint256 donationAmount)
func (_Contract *ContractFilterer) ParseEtherReceived(log types.Log) (*ContractEtherReceived, error) {
	event := new(ContractEtherReceived)
	if err := _Contract.contract.UnpackLog(event, "EtherReceived", log); err != nil {
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

// ContractOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Contract contract.
type ContractOwnershipTransferredIterator struct {
	Event *ContractOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ContractOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractOwnershipTransferred)
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
		it.Event = new(ContractOwnershipTransferred)
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
func (it *ContractOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractOwnershipTransferred represents a OwnershipTransferred event raised by the Contract contract.
type ContractOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ContractOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ContractOwnershipTransferredIterator{contract: _Contract.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ContractOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractOwnershipTransferred)
				if err := _Contract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) ParseOwnershipTransferred(log types.Log) (*ContractOwnershipTransferred, error) {
	event := new(ContractOwnershipTransferred)
	if err := _Contract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractSetRewardRecipientIterator is returned from FilterSetRewardRecipient and is used to iterate over the raw logs and unpacked data for SetRewardRecipient events raised by the Contract contract.
type ContractSetRewardRecipientIterator struct {
	Event *ContractSetRewardRecipient // Event containing the contract specifics and raw log

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
func (it *ContractSetRewardRecipientIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractSetRewardRecipient)
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
		it.Event = new(ContractSetRewardRecipient)
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
func (it *ContractSetRewardRecipientIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractSetRewardRecipientIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractSetRewardRecipient represents a SetRewardRecipient event raised by the Contract contract.
type ContractSetRewardRecipient struct {
	WithdrawalAddress common.Address
	PoolRecipient     common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSetRewardRecipient is a free log retrieval operation binding the contract event 0xc6b66e0e282673c442421e1c6b89458b7631f26f5dcd0b2b216c45831ca1d7d5.
//
// Solidity: event SetRewardRecipient(address withdrawalAddress, address poolRecipient)
func (_Contract *ContractFilterer) FilterSetRewardRecipient(opts *bind.FilterOpts) (*ContractSetRewardRecipientIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "SetRewardRecipient")
	if err != nil {
		return nil, err
	}
	return &ContractSetRewardRecipientIterator{contract: _Contract.contract, event: "SetRewardRecipient", logs: logs, sub: sub}, nil
}

// WatchSetRewardRecipient is a free log subscription operation binding the contract event 0xc6b66e0e282673c442421e1c6b89458b7631f26f5dcd0b2b216c45831ca1d7d5.
//
// Solidity: event SetRewardRecipient(address withdrawalAddress, address poolRecipient)
func (_Contract *ContractFilterer) WatchSetRewardRecipient(opts *bind.WatchOpts, sink chan<- *ContractSetRewardRecipient) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "SetRewardRecipient")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractSetRewardRecipient)
				if err := _Contract.contract.UnpackLog(event, "SetRewardRecipient", log); err != nil {
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

// ParseSetRewardRecipient is a log parse operation binding the contract event 0xc6b66e0e282673c442421e1c6b89458b7631f26f5dcd0b2b216c45831ca1d7d5.
//
// Solidity: event SetRewardRecipient(address withdrawalAddress, address poolRecipient)
func (_Contract *ContractFilterer) ParseSetRewardRecipient(log types.Log) (*ContractSetRewardRecipient, error) {
	event := new(ContractSetRewardRecipient)
	if err := _Contract.contract.UnpackLog(event, "SetRewardRecipient", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractSubscribeValidatorIterator is returned from FilterSubscribeValidator and is used to iterate over the raw logs and unpacked data for SubscribeValidator events raised by the Contract contract.
type ContractSubscribeValidatorIterator struct {
	Event *ContractSubscribeValidator // Event containing the contract specifics and raw log

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
func (it *ContractSubscribeValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractSubscribeValidator)
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
		it.Event = new(ContractSubscribeValidator)
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
func (it *ContractSubscribeValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractSubscribeValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractSubscribeValidator represents a SubscribeValidator event raised by the Contract contract.
type ContractSubscribeValidator struct {
	Sender                 common.Address
	SubscriptionCollateral *big.Int
	ValidatorID            uint64
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterSubscribeValidator is a free log retrieval operation binding the contract event 0x1094f8cfeb6abd0fd67e4ce1e1d3999c0176a4d8c7f8325e3ecddb5a1249fde9.
//
// Solidity: event SubscribeValidator(address sender, uint256 subscriptionCollateral, uint64 validatorID)
func (_Contract *ContractFilterer) FilterSubscribeValidator(opts *bind.FilterOpts) (*ContractSubscribeValidatorIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "SubscribeValidator")
	if err != nil {
		return nil, err
	}
	return &ContractSubscribeValidatorIterator{contract: _Contract.contract, event: "SubscribeValidator", logs: logs, sub: sub}, nil
}

// WatchSubscribeValidator is a free log subscription operation binding the contract event 0x1094f8cfeb6abd0fd67e4ce1e1d3999c0176a4d8c7f8325e3ecddb5a1249fde9.
//
// Solidity: event SubscribeValidator(address sender, uint256 subscriptionCollateral, uint64 validatorID)
func (_Contract *ContractFilterer) WatchSubscribeValidator(opts *bind.WatchOpts, sink chan<- *ContractSubscribeValidator) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "SubscribeValidator")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractSubscribeValidator)
				if err := _Contract.contract.UnpackLog(event, "SubscribeValidator", log); err != nil {
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

// ParseSubscribeValidator is a log parse operation binding the contract event 0x1094f8cfeb6abd0fd67e4ce1e1d3999c0176a4d8c7f8325e3ecddb5a1249fde9.
//
// Solidity: event SubscribeValidator(address sender, uint256 subscriptionCollateral, uint64 validatorID)
func (_Contract *ContractFilterer) ParseSubscribeValidator(log types.Log) (*ContractSubscribeValidator, error) {
	event := new(ContractSubscribeValidator)
	if err := _Contract.contract.UnpackLog(event, "SubscribeValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUnsubscribeValidatorIterator is returned from FilterUnsubscribeValidator and is used to iterate over the raw logs and unpacked data for UnsubscribeValidator events raised by the Contract contract.
type ContractUnsubscribeValidatorIterator struct {
	Event *ContractUnsubscribeValidator // Event containing the contract specifics and raw log

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
func (it *ContractUnsubscribeValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUnsubscribeValidator)
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
		it.Event = new(ContractUnsubscribeValidator)
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
func (it *ContractUnsubscribeValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUnsubscribeValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUnsubscribeValidator represents a UnsubscribeValidator event raised by the Contract contract.
type ContractUnsubscribeValidator struct {
	Sender      common.Address
	ValidatorID uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterUnsubscribeValidator is a free log retrieval operation binding the contract event 0x5a984d13ffecb21f69de43956dcd971abcff8e187fac09a0ce3209562da14e0a.
//
// Solidity: event UnsubscribeValidator(address sender, uint64 validatorID)
func (_Contract *ContractFilterer) FilterUnsubscribeValidator(opts *bind.FilterOpts) (*ContractUnsubscribeValidatorIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UnsubscribeValidator")
	if err != nil {
		return nil, err
	}
	return &ContractUnsubscribeValidatorIterator{contract: _Contract.contract, event: "UnsubscribeValidator", logs: logs, sub: sub}, nil
}

// WatchUnsubscribeValidator is a free log subscription operation binding the contract event 0x5a984d13ffecb21f69de43956dcd971abcff8e187fac09a0ce3209562da14e0a.
//
// Solidity: event UnsubscribeValidator(address sender, uint64 validatorID)
func (_Contract *ContractFilterer) WatchUnsubscribeValidator(opts *bind.WatchOpts, sink chan<- *ContractUnsubscribeValidator) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UnsubscribeValidator")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUnsubscribeValidator)
				if err := _Contract.contract.UnpackLog(event, "UnsubscribeValidator", log); err != nil {
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

// ParseUnsubscribeValidator is a log parse operation binding the contract event 0x5a984d13ffecb21f69de43956dcd971abcff8e187fac09a0ce3209562da14e0a.
//
// Solidity: event UnsubscribeValidator(address sender, uint64 validatorID)
func (_Contract *ContractFilterer) ParseUnsubscribeValidator(log types.Log) (*ContractUnsubscribeValidator, error) {
	event := new(ContractUnsubscribeValidator)
	if err := _Contract.contract.UnpackLog(event, "UnsubscribeValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateOracleIterator is returned from FilterUpdateOracle and is used to iterate over the raw logs and unpacked data for UpdateOracle events raised by the Contract contract.
type ContractUpdateOracleIterator struct {
	Event *ContractUpdateOracle // Event containing the contract specifics and raw log

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
func (it *ContractUpdateOracleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateOracle)
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
		it.Event = new(ContractUpdateOracle)
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
func (it *ContractUpdateOracleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateOracleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateOracle represents a UpdateOracle event raised by the Contract contract.
type ContractUpdateOracle struct {
	NewOracle common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterUpdateOracle is a free log retrieval operation binding the contract event 0x09ad0a3595604db9b7aef0dbd4918cea3642b96bc65ad7c9fb501a1529becd79.
//
// Solidity: event UpdateOracle(address newOracle)
func (_Contract *ContractFilterer) FilterUpdateOracle(opts *bind.FilterOpts) (*ContractUpdateOracleIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateOracle")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateOracleIterator{contract: _Contract.contract, event: "UpdateOracle", logs: logs, sub: sub}, nil
}

// WatchUpdateOracle is a free log subscription operation binding the contract event 0x09ad0a3595604db9b7aef0dbd4918cea3642b96bc65ad7c9fb501a1529becd79.
//
// Solidity: event UpdateOracle(address newOracle)
func (_Contract *ContractFilterer) WatchUpdateOracle(opts *bind.WatchOpts, sink chan<- *ContractUpdateOracle) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateOracle")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateOracle)
				if err := _Contract.contract.UnpackLog(event, "UpdateOracle", log); err != nil {
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

// ParseUpdateOracle is a log parse operation binding the contract event 0x09ad0a3595604db9b7aef0dbd4918cea3642b96bc65ad7c9fb501a1529becd79.
//
// Solidity: event UpdateOracle(address newOracle)
func (_Contract *ContractFilterer) ParseUpdateOracle(log types.Log) (*ContractUpdateOracle, error) {
	event := new(ContractUpdateOracle)
	if err := _Contract.contract.UnpackLog(event, "UpdateOracle", log); err != nil {
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

// ContractUpdateSubscriptionCollateralIterator is returned from FilterUpdateSubscriptionCollateral and is used to iterate over the raw logs and unpacked data for UpdateSubscriptionCollateral events raised by the Contract contract.
type ContractUpdateSubscriptionCollateralIterator struct {
	Event *ContractUpdateSubscriptionCollateral // Event containing the contract specifics and raw log

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
func (it *ContractUpdateSubscriptionCollateralIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateSubscriptionCollateral)
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
		it.Event = new(ContractUpdateSubscriptionCollateral)
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
func (it *ContractUpdateSubscriptionCollateralIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateSubscriptionCollateralIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateSubscriptionCollateral represents a UpdateSubscriptionCollateral event raised by the Contract contract.
type ContractUpdateSubscriptionCollateral struct {
	NewSubscriptionCollateral *big.Int
	Raw                       types.Log // Blockchain specific contextual infos
}

// FilterUpdateSubscriptionCollateral is a free log retrieval operation binding the contract event 0xdb50d3f51ff6294ba829f0d7f6b99b5b606a52807d5106ef44151d9297720217.
//
// Solidity: event UpdateSubscriptionCollateral(uint256 newSubscriptionCollateral)
func (_Contract *ContractFilterer) FilterUpdateSubscriptionCollateral(opts *bind.FilterOpts) (*ContractUpdateSubscriptionCollateralIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateSubscriptionCollateral")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateSubscriptionCollateralIterator{contract: _Contract.contract, event: "UpdateSubscriptionCollateral", logs: logs, sub: sub}, nil
}

// WatchUpdateSubscriptionCollateral is a free log subscription operation binding the contract event 0xdb50d3f51ff6294ba829f0d7f6b99b5b606a52807d5106ef44151d9297720217.
//
// Solidity: event UpdateSubscriptionCollateral(uint256 newSubscriptionCollateral)
func (_Contract *ContractFilterer) WatchUpdateSubscriptionCollateral(opts *bind.WatchOpts, sink chan<- *ContractUpdateSubscriptionCollateral) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateSubscriptionCollateral")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateSubscriptionCollateral)
				if err := _Contract.contract.UnpackLog(event, "UpdateSubscriptionCollateral", log); err != nil {
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

// ParseUpdateSubscriptionCollateral is a log parse operation binding the contract event 0xdb50d3f51ff6294ba829f0d7f6b99b5b606a52807d5106ef44151d9297720217.
//
// Solidity: event UpdateSubscriptionCollateral(uint256 newSubscriptionCollateral)
func (_Contract *ContractFilterer) ParseUpdateSubscriptionCollateral(log types.Log) (*ContractUpdateSubscriptionCollateral, error) {
	event := new(ContractUpdateSubscriptionCollateral)
	if err := _Contract.contract.UnpackLog(event, "UpdateSubscriptionCollateral", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
