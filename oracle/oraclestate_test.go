package oracle

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_AddSubscription(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.AddSubscriptionIfNotAlready(10, "0x", "0x")
	state.IncreaseAllPendingRewards(big.NewInt(100))
	state.ConsolidateBalance(10)
	state.IncreaseAllPendingRewards(big.NewInt(200))
	require.Equal(t, big.NewInt(200), state.Validators[10].PendingRewardsWei)
	require.Equal(t, big.NewInt(100), state.Validators[10].AccumulatedRewardsWei)

	// check that adding again doesnt reset the subscription
	state.AddSubscriptionIfNotAlready(10, "0x", "0x")
	require.Equal(t, big.NewInt(200), state.Validators[10].PendingRewardsWei)
	require.Equal(t, big.NewInt(100), state.Validators[10].AccumulatedRewardsWei)
}

func Test_AddSubscriptionIfNotAlready(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.AddSubscriptionIfNotAlready(uint64(100), "0x3000000000000000000000000000000000000000", "0xkey")
	require.Equal(t, 1, len(state.Validators))
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(0),
		PendingRewardsWei:       big.NewInt(0),
		CollateralWei:           big.NewInt(0),
		WithdrawalAddress:       "0x3000000000000000000000000000000000000000",
		ValidatorIndex:          100,
		ValidatorKey:            "0xkey",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}, state.Validators[100])

	// Modify the validator
	state.Validators[100].AccumulatedRewardsWei = big.NewInt(334545546)
	state.Validators[100].PendingRewardsWei = big.NewInt(87653)

	// If we call it again, it shouldnt be overwritten as its already there
	state.AddSubscriptionIfNotAlready(uint64(100), "0x3000000000000000000000000000000000000000", "0xkey")

	require.Equal(t, big.NewInt(334545546), state.Validators[100].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(87653), state.Validators[100].PendingRewardsWei)
}

func Test_AddDonation(t *testing.T) {
	state := NewOracleState(&config.Config{})
	donations := []Donation{
		Donation{AmountWei: big.NewInt(765432), Block: uint64(100), TxHash: "0x1"},
		Donation{AmountWei: big.NewInt(30023456), Block: uint64(100), TxHash: "0x2"},
	}
	state.HandleDonations(donations)

	require.Equal(t, big.NewInt(765432), state.Donations[0].AmountWei)
	require.Equal(t, uint64(100), state.Donations[0].Block)
	require.Equal(t, "0x1", state.Donations[0].TxHash)

	require.Equal(t, big.NewInt(30023456), state.Donations[1].AmountWei)
	require.Equal(t, uint64(100), state.Donations[1].Block)
	require.Equal(t, "0x2", state.Donations[1].TxHash)
}

func Test_HandleManualSubscriptions_Valid(t *testing.T) {
	// Tests a valid subscription, with enough collateral to a not subscribed validator
	// and sent from the validator's withdrawal address

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	state.HandleManualSubscriptions([]Subscription{sub1})

	require.Equal(t, state.Validators[33], &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(0),
		PendingRewardsWei:       big.NewInt(1000),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          33,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	})
	require.Equal(t, 1, len(state.Validators))
	require.Equal(t, 1, len(state.Subscriptions))
	require.Equal(t, sub1, state.Subscriptions[0])
}

func Test_HandleManualSubscriptions_FromWrongAddress(t *testing.T) {
	// Tests a subscription sent from a wrong address, meaning that it doesnt
	// match the validator's withdrawal address. No subscription is produced

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		Validator: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	// No subscriptions are produced
	state.HandleManualSubscriptions([]Subscription{sub1})
	require.Equal(t, 0, len(state.Validators))
	require.Equal(t, 0, len(state.Subscriptions))
}

func Test_HandleManualSubscriptions_AlreadySubscribed(t *testing.T) {
	// Test a subscription to an already subscribed validator, we return the collateral

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	// Run 3 subscriptions, only one should be added
	state.HandleManualSubscriptions([]Subscription{sub1, sub1, sub1})

	require.Equal(t, state.Validators[33], &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(2000), // Second and third collateral are returned to the user
		PendingRewardsWei:       big.NewInt(1000),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          33,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	})
	require.Equal(t, 1, len(state.Validators))
}

func Test_HandleManualSubscriptions_AlreadySubscribed_WithBalance(t *testing.T) {
	// Test a subscription to an already subscribed validator, that already
	// has some balance. Assert that the existing balance is not touched and the
	// collateral is returned

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	// Validator is subscribed
	state.HandleManualSubscriptions([]Subscription{sub1})

	// And has some rewards
	state.IncreaseValidatorAccumulatedRewards(33, big.NewInt(9000))
	state.IncreaseValidatorPendingRewards(33, big.NewInt(44000))

	// Due to some mistake, the user subscribes again and again
	state.HandleManualSubscriptions([]Subscription{sub1, sub1})

	require.Equal(t, state.Validators[33], &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(9000 + 1000*2),
		PendingRewardsWei:       big.NewInt(44000 + 1000),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          33,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	})
	require.Equal(t, 1, len(state.Validators))
}

func Test_HandleManualSubscriptions_Wrong_BlsCredentials(t *testing.T) {
	// A validator with wrong withdrawal address (bls) tries to subscribe. The validator
	// is nos subscribed and the collateral is given to the pool, since we dont have a way
	// to return it to its owner.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// bls address, not supported
				WithdrawalCredentials: []byte{0, 120, 22, 197, 153, 67, 183, 29, 244, 168, 13, 66, 101, 227, 165, 250, 41, 86, 97, 10, 40, 91, 140, 65, 154, 102, 143, 67, 117, 255, 140, 254},
			},
		},
	}

	state.HandleManualSubscriptions([]Subscription{sub1})
	require.Equal(t, 0, len(state.Validators))
	require.Equal(t, big.NewInt(1000), state.PoolAccumulatedFees)
}

func Test_HandleManualSubscriptions_NonExistent(t *testing.T) {
	// Test a subscription of a non-existent validator. Someone subscribes a validator
	// index that doesnt exist. Nothing happens, and the pool gets this collateral.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: nil,
	}

	state.HandleManualSubscriptions([]Subscription{sub1})
	require.Equal(t, 0, len(state.Validators))
	require.Equal(t, big.NewInt(1000), state.PoolAccumulatedFees)
}

func Test_HandleManualSubscriptions_WrongStateValidator(t *testing.T) {
	// Test a subscription of a validator in a wrong state (eg slashed validator or exited)
	// Nothing happens, and the pool gets this collateral.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	// Was slashed and exited
	sub1 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateExitedSlashed,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		},
	}

	// Its active but its exiting
	sub2 := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            34,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x2}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  34,
			Status: v1.ValidatorStateActiveExiting,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		},
	}

	state.HandleManualSubscriptions([]Subscription{sub1, sub2})

	require.Equal(t, 0, len(state.Validators))
	require.Equal(t, big.NewInt(1000*2), state.PoolAccumulatedFees)
}

func Test_HandleManualSubscriptions_BannedValidator(t *testing.T) {
	// Test a subscription of a banned validator. Check that the validator is not subscribed
	// and its kept in Banned state. Since we track this validator, we return the collateral
	// to the owner in good faith.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	bannedIndex := uint64(300000)
	state.Validators[bannedIndex] = &ValidatorInfo{
		ValidatorStatus:         Banned,
		AccumulatedRewardsWei:   big.NewInt(0),
		PendingRewardsWei:       big.NewInt(0),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          bannedIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}

	sub := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            bannedIndex,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(bannedIndex),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		},
	}
	state.HandleManualSubscriptions([]Subscription{sub})

	// Banned validator stays banned
	require.Equal(t, 1, len(state.Validators))

	// Note that since we track it, we return the collateral as accumulated rewards
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:         Banned,
		AccumulatedRewardsWei:   big.NewInt(1000),
		PendingRewardsWei:       big.NewInt(0),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          bannedIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}, state.Validators[bannedIndex])
}

func Test_SubThenUnsubThenAuto(t *testing.T) {
	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(500000),
		PoolFeesPercent: 0,
	})

	// Subscribe a validator
	valIdx := uint64(9000)
	sub := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            valIdx,
			SubscriptionCollateral: big.NewInt(500000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(valIdx),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{7, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}
	state.HandleManualSubscriptions([]Subscription{sub})

	// Share some rewards with it
	state.Validators[valIdx].PendingRewardsWei = big.NewInt(10000)
	state.Validators[valIdx].AccumulatedRewardsWei = big.NewInt(20000)

	// Unsubscribe it
	unsub := Unsubscription{
		Event: &contract.ContractUnsubscribeValidator{
			ValidatorID: valIdx,
			Sender:      common.Address{3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:         types.Log{TxHash: [32]byte{0x1}},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(valIdx),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{7, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	state.HandleManualUnsubscriptions([]Unsubscription{unsub})

	// Check is no longer subscribed and balances are kept (pending is reset)
	require.Equal(t, big.NewInt(0), state.Validators[valIdx].PendingRewardsWei)
	require.Equal(t, big.NewInt(20000), state.Validators[valIdx].AccumulatedRewardsWei)
	require.Equal(t, NotSubscribed, state.Validators[valIdx].ValidatorStatus)

	// Force automatic subscription
	block1 := Block{
		Slot:              0,
		ValidatorIndex:    valIdx,
		ValidatorKey:      "0x",
		Reward:            big.NewInt(90000000),
		RewardType:        VanilaBlock,
		WithdrawalAddress: "0x0327a30991170f917d7b83def6e44d26577871ed",
	}
	state.HandleCorrectBlockProposal(block1)

	// Pending are 0 because the 90000000 are instantly consolidated
	require.Equal(t, big.NewInt(0), state.Validators[valIdx].PendingRewardsWei)
	// We have the new plus old ones
	require.Equal(t, big.NewInt(20000+90000000), state.Validators[valIdx].AccumulatedRewardsWei)
	require.Equal(t, Active, state.Validators[valIdx].ValidatorStatus)
}

func Test_HandleUnsubscriptions_ValidSubscription(t *testing.T) {
	// Unsubscribe an existing subscribed validator correctly, checking that the event is
	// sent from the withdrawal address of the validator. Check also that when unsubscribing
	// the pending validator rewards are shared among the rest of the validators.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(500000),
	})

	for _, valIdx := range []uint64{6, 9, 10, 15} {
		sub := Subscription{
			Event: &contract.ContractSubscribeValidator{
				ValidatorID:            valIdx,
				SubscriptionCollateral: big.NewInt(500000),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
			Validator: &v1.Validator{
				Index:  phase0.ValidatorIndex(valIdx),
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// byte(valIdx) just to have different key/withdrawal addresses
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					PublicKey:             phase0.BLSPubKey{byte(valIdx), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		}
		state.HandleManualSubscriptions([]Subscription{sub})

		// Simulate some proposals increasing the rewards
		state.IncreaseValidatorAccumulatedRewards(valIdx, big.NewInt(3000))
		state.IncreaseValidatorPendingRewards(valIdx, big.NewInt(300000000000000000-500000))
	}

	require.Equal(t, 4, len(state.Validators))

	// Receive valid unsubscription event for index 6
	unsub := Unsubscription{
		Event: &contract.ContractUnsubscribeValidator{
			ValidatorID: 6,
			// Same as withdrawal credential without the prefix
			Sender: common.Address{byte(6), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(6),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key/withdrawal addresses
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(6), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(6), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}
	state.HandleManualUnsubscriptions([]Unsubscription{unsub})

	require.Equal(t, state.Validators[6], &ValidatorInfo{
		ValidatorStatus:         NotSubscribed,    // Validator is still tracked but not subscribed
		AccumulatedRewardsWei:   big.NewInt(3000), // Accumulated rewards are kept
		PendingRewardsWei:       big.NewInt(0),    // Pending rewards are cleared
		CollateralWei:           big.NewInt(500000),
		WithdrawalAddress:       "0x0627a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          6,
		ValidatorKey:            "0x06aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	})
	require.Equal(t, 4, len(state.Validators))
	require.Equal(t, 1, len(state.Unsubscriptions))
	require.Equal(t, unsub, state.Unsubscriptions[0])

	// The rest get the pending of valIndex=6
	require.Equal(t, state.Validators[9].PendingRewardsWei, big.NewInt(300000000000000000+300000000000000000/3))
	require.Equal(t, state.Validators[10].PendingRewardsWei, big.NewInt(300000000000000000+300000000000000000/3))
	require.Equal(t, state.Validators[15].PendingRewardsWei, big.NewInt(300000000000000000+300000000000000000/3))

	// And accumulated do not change
	require.Equal(t, state.Validators[9].AccumulatedRewardsWei, big.NewInt(3000))
	require.Equal(t, state.Validators[10].AccumulatedRewardsWei, big.NewInt(3000))
	require.Equal(t, state.Validators[15].AccumulatedRewardsWei, big.NewInt(3000))

	// And state of the rest is not changed
	require.Equal(t, state.Validators[9].ValidatorStatus, Active)
	require.Equal(t, state.Validators[10].ValidatorStatus, Active)
	require.Equal(t, state.Validators[15].ValidatorStatus, Active)

	// Unsubscribe all remaining validators
	unsubs := make([]Unsubscription, 0)
	for _, valIdx := range []uint64{ /*6*/ 9, 10, 15} {
		unsub := Unsubscription{
			Event: &contract.ContractUnsubscribeValidator{
				ValidatorID: valIdx,
				// Same as withdrawal credential without the prefix
				Sender: common.Address{byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				Raw:    types.Log{TxHash: [32]byte{0x1}},
			},
			Validator: &v1.Validator{
				Index:  phase0.ValidatorIndex(valIdx),
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// byte(valIdx) just to have different key/withdrawal addresses
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					PublicKey:             phase0.BLSPubKey{byte(valIdx), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		}
		unsubs = append(unsubs, unsub)
	}

	// Unsubscribe all at once
	state.HandleManualUnsubscriptions(unsubs)

	require.Equal(t, 4, len(state.Validators))
	require.Equal(t, state.Validators[6].ValidatorStatus, NotSubscribed)
	require.Equal(t, state.Validators[9].ValidatorStatus, NotSubscribed)
	require.Equal(t, state.Validators[10].ValidatorStatus, NotSubscribed)
	require.Equal(t, state.Validators[15].ValidatorStatus, NotSubscribed)

	require.Equal(t, state.Validators[6].PendingRewardsWei, big.NewInt(0))
	require.Equal(t, state.Validators[9].PendingRewardsWei, big.NewInt(0))
	require.Equal(t, state.Validators[10].PendingRewardsWei, big.NewInt(0))
	require.Equal(t, state.Validators[15].PendingRewardsWei, big.NewInt(0))
}

func Test_HandleUnsubscriptions_NonExistentValidator(t *testing.T) {
	// We receive an unsubscription for a validator that does not exist in the beacon
	// chain. Nothing happens to existing subscribed validators.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	// Simulate subscription of validator 33
	state.Validators[33] = &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(9000 + 1000*2), // Second and third collateral are added to accumulated rewards (returned)
		PendingRewardsWei:       big.NewInt(44000 + 1000),  // First collateral is added to pending (claimable in next block)
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          33,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}

	// Receive event of a validator index that doesnt exist in the beacon chain
	unsub := Unsubscription{
		Event: &contract.ContractUnsubscribeValidator{
			ValidatorID: 900300,
			// Same as withdrawal credential without the prefix
			Sender: common.Address{byte(50), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
		Validator: nil,
	}
	state.HandleManualUnsubscriptions([]Unsubscription{unsub})

	// Check that the existing validator is not affected
	require.Equal(t, 1, len(state.Validators))
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(9000 + 1000*2), // Second and third collateral are added to accumulated rewards (returned)
		PendingRewardsWei:       big.NewInt(44000 + 1000),  // First collateral is added to pending (claimable in next block)
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          33,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}, state.Validators[33])
}

func Test_HandleUnsubscriptions_NotSubscribedValidator(t *testing.T) {
	// We receive an unsubscription for a validator that is not subscribed but exists in
	// the beacon chain. Nothing happens, and no subscriptions are added.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	// Unsubscribe event of a validator index that BUT is not subscribed
	valIdx := uint64(730100)
	unsub := Unsubscription{
		Event: &contract.ContractUnsubscribeValidator{
			ValidatorID: valIdx,
			// Same as withdrawal credential without the prefix
			Sender: common.Address{byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(valIdx),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key/withdrawal addresses
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(valIdx), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}
	state.HandleManualUnsubscriptions([]Unsubscription{unsub})
	require.Equal(t, 0, len(state.Validators))
}

func Test_HandleUnsubscriptions_FromWrongAddress(t *testing.T) {
	// An unsubscription for a subscribed validator is received, but the sender is not the
	// withdrawal address of that validator. Nothing happens to this validator

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	// Simulate subscription of validator 750100
	valIndex := uint64(750100)
	state.Validators[valIndex] = &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(5000000000000000000),
		PendingRewardsWei:       big.NewInt(3000000000000000000),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          valIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}

	unsub := Unsubscription{
		Event: &contract.ContractUnsubscribeValidator{
			ValidatorID: valIndex,
			// Wrong sender address (see WithdrawalCredentials)
			Sender: common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(valIndex),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key/withdrawal addresses
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(valIndex), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}
	state.HandleManualUnsubscriptions([]Unsubscription{unsub})

	// Validator remains intact, since unsubscription event was wrong
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(5000000000000000000),
		PendingRewardsWei:       big.NewInt(3000000000000000000),
		CollateralWei:           big.NewInt(1000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          valIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}, state.Validators[valIndex])
}

func Test_Unsubscribe_AndRejoin(t *testing.T) {
	// A validator subscribes, the unsubscribes and the rejoins. Check that its accumulated balances
	// are kept, and that it can rejoin succesfully.

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(500000),
	})

	// Simulate subscription of validator 750100
	valIndex := uint64(750100)
	state.Validators[valIndex] = &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(0),
		PendingRewardsWei:       big.NewInt(0),
		CollateralWei:           big.NewInt(500000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          valIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}

	// Add some rewards
	state.IncreaseValidatorAccumulatedRewards(valIndex, big.NewInt(1000000000000000000))
	state.IncreaseValidatorPendingRewards(valIndex, big.NewInt(5000000000000000000))

	// Now it unsubscribes ok
	unsub := Unsubscription{
		Event: &contract.ContractUnsubscribeValidator{
			ValidatorID: valIndex,
			// Wrong sender address (see WithdrawalCredentials)
			Sender: common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(valIndex),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key addresses
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(valIndex), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}
	state.HandleManualUnsubscriptions([]Unsubscription{unsub})

	// Unsubscription is ok
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:         NotSubscribed,
		AccumulatedRewardsWei:   big.NewInt(1000000000000000000),
		PendingRewardsWei:       big.NewInt(0),
		CollateralWei:           big.NewInt(500000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          valIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}, state.Validators[valIndex])

	// Now the same validator tries to rejoin
	sub := Subscription{
		Event: &contract.ContractSubscribeValidator{
			ValidatorID:            valIndex,
			SubscriptionCollateral: big.NewInt(500000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		Validator: &v1.Validator{
			Index:  phase0.ValidatorIndex(valIndex),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(valIndex), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}
	state.HandleManualSubscriptions([]Subscription{sub})

	// Its subscribed again with its old accumulated rewards
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:         Active,
		AccumulatedRewardsWei:   big.NewInt(1000000000000000000),
		PendingRewardsWei:       big.NewInt(500000),
		CollateralWei:           big.NewInt(500000),
		WithdrawalAddress:       "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:          valIndex,
		ValidatorKey:            "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
		ValidatorProposedBlocks: []Block{},
		ValidatorMissedBlocks:   []Block{},
		ValidatorWrongFeeBlocks: []Block{},
	}, state.Validators[valIndex])
}

func Test_StoreLatestOnchainState(t *testing.T) {
	state := NewOracleState(&config.Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0xfee0000000000000000000000000000000000000",
	})

	valInfo1 := &ValidatorInfo{
		ValidatorStatus:       Active,
		ValidatorIndex:        1,
		AccumulatedRewardsWei: big.NewInt(1000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		WithdrawalAddress:     "0x1000000000000000000000000000000000000000",
		ValidatorProposedBlocks: []Block{
			{
				Slot:   1,
				Block:  1,
				Reward: big.NewInt(1),
			},
			{
				Slot:   2,
				Block:  2,
				Reward: big.NewInt(2),
			},
		},
	}

	valInfo2 := &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		ValidatorIndex:        2,
		AccumulatedRewardsWei: big.NewInt(2000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		// same withdrawal address as valInfo3
		WithdrawalAddress: "0x2000000000000000000000000000000000000000",
		ValidatorMissedBlocks: []Block{
			{
				Slot:   10,
				Block:  10,
				Reward: big.NewInt(1),
			},
		},
	}

	valInfo3 := &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		ValidatorIndex:        3,
		AccumulatedRewardsWei: big.NewInt(2000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		// same withdrawal address as valInfo2
		WithdrawalAddress: "0x2000000000000000000000000000000000000000",
		ValidatorMissedBlocks: []Block{
			{
				Slot:   50,
				Block:  50,
				Reward: big.NewInt(10000),
			},
		},
	}

	state.Validators[1] = valInfo1
	state.Validators[2] = valInfo2
	state.Validators[3] = valInfo3

	// Function under test
	state.StoreLatestOnchainState()

	// Ensure all validators are present in the state
	require.Equal(t, valInfo1, state.LatestCommitedState.Validators[1])
	require.Equal(t, valInfo2, state.LatestCommitedState.Validators[2])
	require.Equal(t, valInfo3, state.LatestCommitedState.Validators[3])

	// Ensure merkle root matches
	require.Equal(t, "0xd9a1eee574026532cddccbcce6320c0600f370a7c64ce30c5eafc63357449940", state.LatestCommitedState.MerkleRoot)

	// Ensure proofs and leafs are correct
	require.Equal(t, state.LatestCommitedState.Proofs["0xfee0000000000000000000000000000000000000"], []string{"0x8bfb8acff6772a60d6641cb854587bb2b6f2100391fbadff2c34be0b8c20a0cc", "0x27205dd4c642acd1b1352617df2c4f410e20ff3fd6f3e3efddee9cea044921f8"})
	require.Equal(t, state.LatestCommitedState.Proofs["0x1000000000000000000000000000000000000000"], []string{"0xaaf838df9c8d5cec6ed77fcbc2cace945e8f2078eede4a0bb7164818d425f24d", "0x27205dd4c642acd1b1352617df2c4f410e20ff3fd6f3e3efddee9cea044921f8"})
	require.Equal(t, state.LatestCommitedState.Proofs["0x2000000000000000000000000000000000000000"], []string{"0xd643163144dcba353b4d27c50939b3d11133bd3c6916092de059d07353b4cb5f", "0xda53f5dd3e17f66f4a35c9c9d5fd27c094fa4249e2933fb819ac724476dc9ae1"})

	require.Equal(t, state.LatestCommitedState.Leafs["0xfee0000000000000000000000000000000000000"], RawLeaf{"0xfee0000000000000000000000000000000000000", big.NewInt(0)})
	require.Equal(t, state.LatestCommitedState.Leafs["0x1000000000000000000000000000000000000000"], RawLeaf{"0x1000000000000000000000000000000000000000", big.NewInt(1000000000000000000)})
	require.Equal(t, state.LatestCommitedState.Leafs["0x2000000000000000000000000000000000000000"], RawLeaf{"0x2000000000000000000000000000000000000000", big.NewInt(4000000000000000000)})

	// Ensure LatestCommitedState contains a deep copy of the validators and not just a reference
	// This is very important since otherwise they will be modified when the state is modified
	// and we want a frozen snapshot of the state at that moment.

	// Do some changes in validators
	state.Validators[1].ValidatorProposedBlocks[0].Slot = 3
	state.Validators[2].AccumulatedRewardsWei = big.NewInt(22)
	state.Validators[3].PendingRewardsWei = big.NewInt(22)

	// And assert the frozen state is not changes
	require.Equal(t, uint64(1), state.LatestCommitedState.Validators[1].ValidatorProposedBlocks[0].Slot)
	require.Equal(t, big.NewInt(2000000000000000000), state.LatestCommitedState.Validators[2].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(500000), state.LatestCommitedState.Validators[3].PendingRewardsWei)
}

func Test_IncreaseAllPendingRewards_1(t *testing.T) {

	state := NewOracleState(&config.Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	state.AddSubscriptionIfNotAlready(1, "0x", "0x")
	state.AddSubscriptionIfNotAlready(2, "0x", "0x")
	state.AddSubscriptionIfNotAlready(3, "0x", "0x")

	state.IncreaseAllPendingRewards(big.NewInt(10000))

	// Note that in this case even with PoolFeesPercent: 0, the pool gets the remainder
	require.Equal(t, big.NewInt(3333), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(3333), state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(3333), state.Validators[3].PendingRewardsWei)
	require.Equal(t, big.NewInt(1), state.PoolAccumulatedFees)
}

func Test_IncreaseAllPendingRewards_2(t *testing.T) {

	state := NewOracleState(&config.Config{
		PoolFeesPercent: 10,
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	state.AddSubscriptionIfNotAlready(1, "0x", "0x")
	state.AddSubscriptionIfNotAlready(2, "0x", "0x")
	state.AddSubscriptionIfNotAlready(3, "0x", "0x")

	state.IncreaseAllPendingRewards(big.NewInt(10000))

	// Note that in this case even with PoolFeesPercent: 0, the pool gets the remainder
	require.Equal(t, big.NewInt(3000), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(3000), state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(3000), state.Validators[3].PendingRewardsWei)
	require.Equal(t, big.NewInt(1000), state.PoolAccumulatedFees)
}

func Test_IncreaseAllPendingRewards_3(t *testing.T) {

	// Multiple test with different combinations of: fee, reward, validators

	type pendingRewardTest struct {
		FeePercent       int
		Reward           []*big.Int
		AmountValidators int
	}

	tests := []pendingRewardTest{
		// FeePercent |Reward | AmountValidators
		{0, []*big.Int{big.NewInt(100)}, 1},
		{0, []*big.Int{big.NewInt(500)}, 2},
		{0, []*big.Int{big.NewInt(398)}, 3},
		{10, []*big.Int{big.NewInt(0)}, 1},
		{15, []*big.Int{big.NewInt(23033)}, 1},
		{33, []*big.Int{big.NewInt(99999)}, 5},
		{33, []*big.Int{big.NewInt(1)}, 5},
		{33, []*big.Int{big.NewInt(1), big.NewInt(403342)}, 200},
		{12, []*big.Int{big.NewInt(32000000000000), big.NewInt(333333333333), big.NewInt(345676543234567)}, 233},
		{14, []*big.Int{big.NewInt(32000000000000), big.NewInt(333333333333), big.NewInt(345676543234567), big.NewInt(9)}, 99},
	}

	for _, test := range tests {
		state := NewOracleState(&config.Config{
			PoolFeesPercent: test.FeePercent,
			PoolFeesAddress: "0x",
		})
		for i := 0; i < test.AmountValidators; i++ {
			state.AddSubscriptionIfNotAlready(uint64(i), "0x", "0x")
		}

		totalRewards := big.NewInt(0)
		for _, reward := range test.Reward {
			state.IncreaseAllPendingRewards(reward)
			totalRewards.Add(totalRewards, reward)
		}

		totalDistributedRewards := big.NewInt(0)
		totalDistributedRewards.Add(totalDistributedRewards, state.PoolAccumulatedFees)
		for i := 0; i < test.AmountValidators; i++ {
			totalDistributedRewards.Add(totalDistributedRewards, state.Validators[uint64(i)].PendingRewardsWei)
		}

		// Assert that the rewards that were shared, equal the ones that we had
		// kirchhoff law, what comes in = what it goes out!
		require.Equal(t, totalDistributedRewards, totalRewards)
	}
}

func Test_IncreaseValidatorPendingRewards(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[12] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(100),
		AccumulatedRewardsWei: big.NewInt(0),
	}
	state.Validators[200] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(100),
		AccumulatedRewardsWei: big.NewInt(0),
	}

	state.IncreaseValidatorPendingRewards(12, big.NewInt(8765432))
	require.Equal(t, big.NewInt(8765432+100), state.Validators[12].PendingRewardsWei)
	require.Equal(t, big.NewInt(0), state.Validators[12].AccumulatedRewardsWei)

	state.IncreaseValidatorPendingRewards(200, big.NewInt(0))
	require.Equal(t, big.NewInt(100), state.Validators[200].PendingRewardsWei)

	state.IncreaseValidatorPendingRewards(12, big.NewInt(1))
	require.Equal(t, big.NewInt(8765432+100+1), state.Validators[12].PendingRewardsWei)
}

func Test_IncreaseValidatorAccumulatedRewards(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[9999999] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(100),
		AccumulatedRewardsWei: big.NewInt(99999999999999),
	}
	state.IncreaseValidatorAccumulatedRewards(9999999, big.NewInt(87676545432))
	require.Equal(t, big.NewInt(87676545432+99999999999999), state.Validators[9999999].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(100), state.Validators[9999999].PendingRewardsWei)
}

func Test_SendRewardToPool(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.SendRewardToPool(big.NewInt(10456543212340))
	require.Equal(t, big.NewInt(10456543212340), state.PoolAccumulatedFees)

	state.SendRewardToPool(big.NewInt(99999))
	require.Equal(t, big.NewInt(10456543212340+99999), state.PoolAccumulatedFees)
}

func Test_ResetPendingRewards(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[1] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(99999999999999),
		AccumulatedRewardsWei: big.NewInt(99999999999999),
	}
	state.ResetPendingRewards(1)

	require.Equal(t, big.NewInt(0), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(99999999999999), state.Validators[1].AccumulatedRewardsWei)
}

func Test_IncreasePendingRewards(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[12] = &ValidatorInfo{
		WithdrawalAddress: "0xaa",
		ValidatorStatus:   Active,
		PendingRewardsWei: big.NewInt(100),
	}
	totalAmount := big.NewInt(130)

	require.Equal(t, big.NewInt(100), state.Validators[12].PendingRewardsWei)
	state.IncreaseAllPendingRewards(totalAmount)
	require.Equal(t, big.NewInt(230), state.Validators[12].PendingRewardsWei)
}

func Test_IncreasePendingEmptyPool(t *testing.T) {
	// Test a case where a new rewards adds to the pool but no validators are subscribed
	// This can happen when a donation is recived to the pool but no validators are subscribed
	state := NewOracleState(&config.Config{})

	// This prevents division by zero
	state.IncreaseAllPendingRewards(big.NewInt(10000))

	// Pool gets all rewards
	require.Equal(t, big.NewInt(10000), state.PoolAccumulatedFees)
}

func Test_ConsolidateBalance_Eligible(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[10] = &ValidatorInfo{
		AccumulatedRewardsWei: big.NewInt(77),
		PendingRewardsWei:     big.NewInt(23),
	}

	require.Equal(t, big.NewInt(77), state.Validators[10].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(23), state.Validators[10].PendingRewardsWei)

	state.ConsolidateBalance(10)

	require.Equal(t, big.NewInt(100), state.Validators[10].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(0), state.Validators[10].PendingRewardsWei)
}

func Test_StateMachine(t *testing.T) {
	state := NewOracleState(&config.Config{})
	valIndex1 := uint64(1000)
	valIndex2 := uint64(2000)

	type stateTest struct {
		From  ValidatorStatus
		Event Event
		End   ValidatorStatus
	}

	stateMachineTestVector := []stateTest{
		// FromState | Event | EndState
		{Active, ProposalOk, Active},
		{Active, ProposalMissed, YellowCard},
		{Active, ProposalWrongFee, Banned},
		{Active, Unsubscribe, NotSubscribed},

		{YellowCard, ProposalOk, Active},
		{YellowCard, ProposalMissed, RedCard},
		{YellowCard, ProposalWrongFee, Banned},
		{YellowCard, Unsubscribe, NotSubscribed},

		{RedCard, ProposalOk, YellowCard},
		{RedCard, ProposalMissed, RedCard},
		{RedCard, ProposalWrongFee, Banned},
		{RedCard, Unsubscribe, NotSubscribed},

		{NotSubscribed, ManualSubscription, Active},
		{NotSubscribed, AutoSubscription, Active},
	}

	for _, testState := range stateMachineTestVector {
		state.Validators[valIndex1] = &ValidatorInfo{
			ValidatorStatus: testState.From,
		}
		state.Validators[valIndex2] = &ValidatorInfo{
			ValidatorStatus: testState.From,
		}

		state.AdvanceStateMachine(valIndex1, testState.Event)
		state.AdvanceStateMachine(valIndex2, testState.Event)

		require.Equal(t, testState.End, state.Validators[valIndex1].ValidatorStatus)
		require.Equal(t, testState.End, state.Validators[valIndex2].ValidatorStatus)
	}
}

func Test_SaveLoadFromToFile_EmptyState(t *testing.T) {
	state := NewOracleState(&config.Config{
		PoolAddress:     "0x0000000000000000000000000000000000000000",
		PoolFeesAddress: "0x1000000000000000000000000000000000000000",
		Network:         "mainnet",
	})

	state.SaveStateToFile()
	defer os.Remove(filepath.Join(StateFileName, StateFolder))
	defer os.RemoveAll(StateFolder)

	err := state.LoadStateFromFile()
	require.NoError(t, err)
	require.Equal(t, state, state)
}
func Test_SaveLoadFromToFile_PopulatedState(t *testing.T) {

	state := NewOracleState(&config.Config{
		PoolAddress:     "0x0000000000000000000000000000000000000000",
		PoolFeesAddress: "0x1000000000000000000000000000000000000000",
		Network:         "mainnet",
	})

	state.Donations = make([]Donation, 1)

	state.Donations[0] = Donation{
		AmountWei: big.NewInt(1000),
		Block:     1000,
		TxHash:    "0x",
	}

	state.Validators[10] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(1000),
		PendingRewardsWei:     big.NewInt(1000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        10,
		ValidatorKey:          "0xc", // TODO: Fix this, should be uint64
		ValidatorProposedBlocks: []Block{
			Block{
				Reward:     big.NewInt(1000),
				RewardType: VanilaBlock,
				Slot:       1000,
			}, Block{
				Reward:     big.NewInt(12000),
				RewardType: VanilaBlock,
				Slot:       3000,
			}, Block{
				Reward:     big.NewInt(7000),
				RewardType: MevBlock,
				Slot:       6000,
			}},
		ValidatorMissedBlocks: []Block{Block{
			Reward:     big.NewInt(1000),
			RewardType: VanilaBlock,
			Slot:       500,
		}, Block{
			Reward:     big.NewInt(1000),
			RewardType: VanilaBlock,
			Slot:       12000,
		}},
		ValidatorWrongFeeBlocks: []Block{Block{
			Reward:     big.NewInt(1000),
			RewardType: VanilaBlock,
			Slot:       500,
		}, Block{
			Reward:     big.NewInt(1000),
			RewardType: VanilaBlock,
			Slot:       12000,
		}},
	}

	state.Validators[20] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(13000),
		PendingRewardsWei:     big.NewInt(100),
		CollateralWei:         big.NewInt(1000000),
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        20,
		ValidatorKey:          "0xc",
		ValidatorProposedBlocks: []Block{
			Block{
				Reward:     big.NewInt(1000),
				RewardType: VanilaBlock,
				Slot:       1000,
			}, Block{
				Reward:     big.NewInt(12000),
				RewardType: VanilaBlock,
				Slot:       3000,
			}, Block{
				Reward:     big.NewInt(7000),
				RewardType: MevBlock,
				Slot:       6000,
			}},
		ValidatorMissedBlocks: []Block{Block{
			Reward:     big.NewInt(33000),
			RewardType: VanilaBlock,
			Slot:       800,
		}, Block{
			Reward:     big.NewInt(11000),
			RewardType: VanilaBlock,
			Slot:       15000,
		}},
		ValidatorWrongFeeBlocks: []Block{Block{
			Reward:     big.NewInt(14000),
			RewardType: VanilaBlock,
			Slot:       700,
		}, Block{
			Reward:     big.NewInt(18000),
			RewardType: VanilaBlock,
			Slot:       19000,
		}},
	}

	state.Validators[30] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(53000),
		PendingRewardsWei:     big.NewInt(000),
		CollateralWei:         big.NewInt(4000000),
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        30,
		ValidatorKey:          "0xc",
		// Empty Proposed blocks
		ValidatorMissedBlocks: []Block{Block{
			Reward:     big.NewInt(303000),
			RewardType: VanilaBlock,
			Slot:       12200,
		}},
		ValidatorWrongFeeBlocks: []Block{Block{
			Reward:     big.NewInt(15000),
			RewardType: VanilaBlock,
			Slot:       800,
		}, Block{
			Reward:     big.NewInt(189000),
			RewardType: VanilaBlock,
			Slot:       232000,
		}},
	}

	defer os.Remove(filepath.Join(StateFileName, StateFolder))
	defer os.RemoveAll(StateFolder)
	state.SaveStateToFile()

	err := state.LoadStateFromFile()
	require.NoError(t, err)
	require.Equal(t, state, state)
}

func Test_IsValidatorSubscribed(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[10] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(100),
		PendingRewardsWei:     big.NewInt(200),
	}
	state.Validators[20] = &ValidatorInfo{
		ValidatorStatus:       YellowCard,
		AccumulatedRewardsWei: big.NewInt(300),
		PendingRewardsWei:     big.NewInt(300),
	}
	state.Validators[30] = &ValidatorInfo{
		ValidatorStatus:       RedCard,
		AccumulatedRewardsWei: big.NewInt(900),
		PendingRewardsWei:     big.NewInt(100),
	}
	state.Validators[40] = &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		AccumulatedRewardsWei: big.NewInt(50),
		PendingRewardsWei:     big.NewInt(10),
	}
	state.Validators[40] = &ValidatorInfo{
		ValidatorStatus:       Banned,
		AccumulatedRewardsWei: big.NewInt(50),
		PendingRewardsWei:     big.NewInt(10),
	}
	require.Equal(t, true, state.IsSubscribed(10))
	require.Equal(t, true, state.IsSubscribed(20))
	require.Equal(t, true, state.IsSubscribed(30))
	require.Equal(t, false, state.IsSubscribed(40))
	require.Equal(t, false, state.IsSubscribed(50))
}

func Test_BanValidator(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.AddSubscriptionIfNotAlready(1, "0xa", "0xb")
	state.AddSubscriptionIfNotAlready(2, "0xa", "0xb")
	state.AddSubscriptionIfNotAlready(3, "0xa", "0xb")

	// New reward arrives
	state.IncreaseAllPendingRewards(big.NewInt(99))

	// Shared equally among all validators
	require.Equal(t, big.NewInt(33), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(33), state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(33), state.Validators[3].PendingRewardsWei)

	// Ban validator 3
	state.HandleBanValidator(Block{ValidatorIndex: 3})

	// Its pending balance is shared equally among the rest
	require.Equal(t, big.NewInt(49), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(49), state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(0), state.Validators[3].PendingRewardsWei)

	// The pool fee address gets the rounding errors (1 wei, neglectable)
	require.Equal(t, big.NewInt(1), state.PoolAccumulatedFees)
}

func Test_IsBanned(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[1] = &ValidatorInfo{
		ValidatorStatus: Active,
	}
	state.Validators[2] = &ValidatorInfo{
		ValidatorStatus: YellowCard,
	}
	state.Validators[3] = &ValidatorInfo{
		ValidatorStatus: RedCard,
	}
	state.Validators[4] = &ValidatorInfo{
		ValidatorStatus: NotSubscribed,
	}
	state.Validators[5] = &ValidatorInfo{
		ValidatorStatus: Banned,
	}

	require.Equal(t, false, state.IsBanned(1))
	require.Equal(t, false, state.IsBanned(2))
	require.Equal(t, false, state.IsBanned(3))
	require.Equal(t, false, state.IsBanned(4))
	require.Equal(t, true, state.IsBanned(5))
}

// TODO: Add a Test_Handle_Subscriptions_1 happy path to cover the normal flow

// Follows an non happy path with a lot of edge cases and possible misconfigurations
func Test_Handle_Subscriptions_1(t *testing.T) {

	state := NewOracleState(&config.Config{
		CollateralInWei: big.NewInt(1000),
	})

	//Set up 3 new subs (val index 33,34,35), two valid and one invalid (low collateral)
	subs := []Subscription{
		{
			Event: &contract.ContractSubscribeValidator{
				ValidatorID:            33,
				SubscriptionCollateral: big.NewInt(1000),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
			Validator: &v1.Validator{
				Index:  33,
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
					PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		},
		{
			Event: &contract.ContractSubscribeValidator{
				ValidatorID:            34,
				SubscriptionCollateral: big.NewInt(1000),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
			Validator: &v1.Validator{
				Index:  34,
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
					PublicKey: phase0.BLSPubKey{130, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		},
		{
			Event: &contract.ContractSubscribeValidator{
				ValidatorID:            35,
				SubscriptionCollateral: big.NewInt(50),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{150, 39, 165, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
			Validator: &v1.Validator{
				Index:  35,
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 150, 39, 165, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
					PublicKey: phase0.BLSPubKey{131, 170, 2, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		},
	}

	state.HandleManualSubscriptions(subs)

	// 3 validator tried to sub, 2 ok, 1 not enough collateral
	require.Equal(t, 2, len(state.Validators))
	require.Equal(t, 2, len(state.Subscriptions))

	require.Equal(t, subs[0], state.Subscriptions[0])

	//one validator subscribed with wrong collateral --> sent to the pool
	require.Equal(t, big.NewInt(50), state.PoolAccumulatedFees)

	// We keep track of [33 & 34] since subscription was valid
	require.Equal(t, Active, state.Validators[33].ValidatorStatus)
	require.Equal(t, Active, state.Validators[34].ValidatorStatus)

	// Accumulated rewards should be 0
	require.Equal(t, big.NewInt(0), state.Validators[33].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(0), state.Validators[34].AccumulatedRewardsWei)

	// Collateral should be 1000
	require.Equal(t, big.NewInt(1000), state.Validators[33].CollateralWei)
	require.Equal(t, big.NewInt(1000), state.Validators[34].CollateralWei)

	//Set up 2 new subs, both of already subscribed validators one sends configured collateral, the other does not
	subs2 := []Subscription{
		{
			// validator already subscribed sends subscription again with too much collateral
			Event: &contract.ContractSubscribeValidator{
				ValidatorID:            33,
				SubscriptionCollateral: big.NewInt(5000000),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
			Validator: &v1.Validator{
				Index:  33,
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
					PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		},
		{
			// validator already subscribed sends subscription again with correct collateral
			Event: &contract.ContractSubscribeValidator{
				ValidatorID:            34,
				SubscriptionCollateral: big.NewInt(1000),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
			Validator: &v1.Validator{
				Index:  34,
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
					PublicKey: phase0.BLSPubKey{130, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		},
	}

	state.HandleManualSubscriptions(subs2)

	// [33] & [34] should still be active after trying to subscribe again
	require.Equal(t, Active, state.Validators[33].ValidatorStatus)
	require.Equal(t, Active, state.Validators[34].ValidatorStatus)

	// when an already subscribed validator manually subscribes again, we send the collateral to their accumulated rewards
	require.Equal(t, big.NewInt(5000000), state.Validators[33].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(1000), state.Validators[34].AccumulatedRewardsWei)

	// Collateral does not change
	require.Equal(t, big.NewInt(1000), state.Validators[33].CollateralWei)
	require.Equal(t, big.NewInt(1000), state.Validators[34].CollateralWei)

	// Ban validator 34
	state.HandleBanValidator(Block{
		Slot:           uint64(100),
		ValidatorIndex: uint64(34),
	})

	// Validator 34 should be banned
	require.Equal(t, Banned, state.Validators[34].ValidatorStatus)
	// Accumulated does not change
	require.Equal(t, big.NewInt(1000), state.Validators[34].AccumulatedRewardsWei)

	// Accumulated rewards of other validators does not change because banned validator didnt have pending rewards
	require.Equal(t, big.NewInt(5000000), state.Validators[33].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(1000), state.Validators[34].AccumulatedRewardsWei)
}

func Test_Handle_TODO(t *testing.T) {
	/*
		cfg := &config.Config{
			PoolFeesAddress: "0xa",
			PoolFeesPercent: 0,
			CollateralInWei: big.NewInt(1000000),
		}

			state := NewOracleState(cfg)

			// Two subscriptions ok. Third not enough collateral
			subs := []Subscription{
				{
					ValidatorIndex: 10,
					ValidatorKey:   "0xaa",
					Collateral:     big.NewInt(1000000), // Enough
					BlockNumber:    0,
					TxHash:         "0xab",
					WithdrawalAddress: "0xac",
				},
				{
					ValidatorIndex: 20,
					ValidatorKey:   "0xba",
					Collateral:     big.NewInt(1000000), // Enough
					BlockNumber:    0,
					TxHash:         "0xbb",
					WithdrawalAddress: "0xbc",
				},
				{
					ValidatorIndex: 30,
					ValidatorKey:   "0xba",
					Collateral:     big.NewInt(50), // Not enough
					BlockNumber:    0,
					TxHash:         "0xbb",
					WithdrawalAddress: "0xbc",
				},
			}
			state.HandleManualSubscriptions(cfg.CollateralInWei, subs)

			// Block from a subscribed validator (manual)
			block1 := Block{
				Slot:           0,
				ValidatorIndex: 10,
				ValidatorKey:   "0x",
				Reward:         big.NewInt(50000000),
				RewardType:     VanilaBlock,
				WithdrawalAddress: "0ac",
			}
			state.HandleCorrectBlockProposal(block1)

			// Block from a non-subscribed validator (auto)
			block2 := Block{
				Slot:           0,
				ValidatorIndex: 40,
				ValidatorKey:   "0x",
				Reward:         big.NewInt(3333333),
				RewardType:     VanilaBlock,
				WithdrawalAddress: "0ac",
			}
			state.HandleCorrectBlockProposal(block2)

			fmt.Println(state.Validators[10])
			fmt.Println(state.Validators[20])
			fmt.Println(state.Validators[30])

			// Test also
			//or.State.HandleBanValidator(customBlock)
			//or.State.HandleManualUnsubscriptions(newBlockUnsub)
			//or.State.HandleDonations(blockDonations)
			//or.State.HandleMissedBlock(customBlock)
	*/
}

// TODO: Add tests for add subscription and remove subscription
// TODO: Add more tests when spec settled

func Test_CanValidatorSubscribeToPool(t *testing.T) {

	require.Equal(t, CanValidatorSubscribeToPool(&v1.Validator{
		Status: v1.ValidatorStatePendingInitialized,
	}), true)

	require.Equal(t, CanValidatorSubscribeToPool(&v1.Validator{
		Status: v1.ValidatorStatePendingQueued,
	}), true)

	require.Equal(t, CanValidatorSubscribeToPool(&v1.Validator{
		Status: v1.ValidatorStateActiveOngoing,
	}), true)
}
