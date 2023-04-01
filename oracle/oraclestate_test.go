package oracle

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
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

func Test_IncreasePendingRewards(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[12] = &ValidatorInfo{
		DepositAddress:    "0xaa",
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
func Test_SaveLoadFromToFile(t *testing.T) {

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
		DepositAddress:        "0xa000000000000000000000000000000000000000",
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
		DepositAddress:        "0xa000000000000000000000000000000000000000",
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
		DepositAddress:        "0xa000000000000000000000000000000000000000",
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

	StateFileName = "test_state.gob"
	defer os.Remove(StateFileName)
	state.SaveStateToFile()

	recovered, err := ReadStateFromFile()
	require.NoError(t, err)
	require.Equal(t, state, recovered)
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
	require.Equal(t, true, state.IsValidatorSubscribed(10))
	require.Equal(t, true, state.IsValidatorSubscribed(20))
	require.Equal(t, true, state.IsValidatorSubscribed(30))
	require.Equal(t, false, state.IsValidatorSubscribed(40))
	require.Equal(t, false, state.IsValidatorSubscribed(50))
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
	cfg := &config.Config{
		PoolFeesAddress: "0xa",
		PoolFeesPercent: 0,
		CollateralInWei: big.NewInt(1000000),
	}
	state := NewOracleState(cfg)

	// Two subscriptions ok. Third not enough collateral
	subs := []Subscription{
		{
			ValidatorIndex: 1,
			ValidatorKey:   "0xaa",
			Collateral:     big.NewInt(1000000), // Enough
			BlockNumber:    0,
			TxHash:         "0xab",
			DepositAddress: "0xac",
		},
		{
			ValidatorIndex: 2,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(1000000), // Enough
			BlockNumber:    0,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
		},
		{
			ValidatorIndex: 3,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(50), // Not enough
			BlockNumber:    0,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
		},
	}
	state.HandleManualSubscriptions(cfg.CollateralInWei, subs)

	require.Equal(t, 3, len(state.Validators))
	require.Equal(t, Active, state.Validators[1].ValidatorStatus)
	require.Equal(t, Active, state.Validators[2].ValidatorStatus)
	// We keep track of [3] since we returned the rewards, but NotSubscribed
	require.Equal(t, NotSubscribed, state.Validators[3].ValidatorStatus)

	require.Equal(t, big.NewInt(0), state.Validators[1].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(0), state.Validators[2].AccumulatedRewardsWei)
	// Rewards were return
	require.Equal(t, big.NewInt(50), state.Validators[3].AccumulatedRewardsWei)

	// Valid subscriptions have the pending correctly updated
	require.Equal(t, big.NewInt(1000000), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(1000000), state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(0), state.Validators[3].PendingRewardsWei)

	// Collateral is updated properly
	require.Equal(t, big.NewInt(1000000), state.Validators[1].CollateralWei)
	require.Equal(t, big.NewInt(1000000), state.Validators[2].CollateralWei)
	// We dont even consider collateral for [3] since it was returned
	require.Equal(t, big.NewInt(0), state.Validators[3].CollateralWei)

	// Already subscribed validators
	subs2 := []Subscription{
		{
			ValidatorIndex: 1,
			ValidatorKey:   "0xaa",
			Collateral:     big.NewInt(5000000), // Too much + already subscribed
			BlockNumber:    5,
			TxHash:         "0xab",
			DepositAddress: "0xac",
		},
		{
			ValidatorIndex: 2,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(5000000), // Too much + already subscribed
			BlockNumber:    5,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
		},
	}

	state.HandleManualSubscriptions(cfg.CollateralInWei, subs2)

	// Still active, nothing changes
	require.Equal(t, Active, state.Validators[1].ValidatorStatus)
	require.Equal(t, Active, state.Validators[2].ValidatorStatus)

	// We return this extra collateral
	require.Equal(t, big.NewInt(5000000), state.Validators[1].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(5000000), state.Validators[2].AccumulatedRewardsWei)

	// Pending still the same
	require.Equal(t, big.NewInt(1000000), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(1000000), state.Validators[2].PendingRewardsWei)

	// Collateral does not change
	require.Equal(t, big.NewInt(1000000), state.Validators[1].CollateralWei)
	require.Equal(t, big.NewInt(1000000), state.Validators[2].CollateralWei)

	// Validator 3 tries to subscribe again, now with actually more collateral
	// than needed
	subs3 := []Subscription{
		{
			ValidatorIndex: 3, // Already tracked, wrongly deposited collateral before
			ValidatorKey:   "0xca",
			Collateral:     big.NewInt(1000070), // More than enough
			BlockNumber:    0,
			TxHash:         "0xcb",
			DepositAddress: "0xcc",
		},
	}
	state.HandleManualSubscriptions(cfg.CollateralInWei, subs3)

	// Boilerplate asserts to ensure nothing changed

	// Still active, nothing changes
	require.Equal(t, Active, state.Validators[1].ValidatorStatus)
	require.Equal(t, Active, state.Validators[2].ValidatorStatus)

	// We return this extra collateral
	require.Equal(t, big.NewInt(5000000), state.Validators[1].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(5000000), state.Validators[2].AccumulatedRewardsWei)

	// Pending still the same
	require.Equal(t, big.NewInt(1000000), state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(1000000), state.Validators[2].PendingRewardsWei)

	// Collateral does not change
	require.Equal(t, big.NewInt(1000000), state.Validators[1].CollateralWei)
	require.Equal(t, big.NewInt(1000000), state.Validators[2].CollateralWei)

	// Validator [3] asserts
	require.Equal(t, Active, state.Validators[3].ValidatorStatus)
	require.Equal(t, big.NewInt(50), state.Validators[3].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(1000070), state.Validators[3].PendingRewardsWei)
	require.Equal(t, big.NewInt(1000070), state.Validators[3].CollateralWei)

	// Ban validator 3
	state.HandleBanValidator(Block{
		Slot:           uint64(100),
		ValidatorIndex: uint64(3),
		ValidatorKey:   "0xca",
	})

	require.Equal(t, Banned, state.Validators[3].ValidatorStatus)
	// Pending rewards are reset to zero
	require.Equal(t, big.NewInt(0), state.Validators[3].PendingRewardsWei)
	// Accumulated do not change
	require.Equal(t, big.NewInt(50), state.Validators[3].AccumulatedRewardsWei)

	// Banned validator tries to add more collateral
	subs4 := []Subscription{
		{
			ValidatorIndex: 3, // Already tracked, wrongly deposited collateral before
			ValidatorKey:   "0xca",
			Collateral:     big.NewInt(2500000), // More than enough
			BlockNumber:    0,
			TxHash:         "0xcb",
			DepositAddress: "0xcc",
		},
	}
	state.HandleManualSubscriptions(cfg.CollateralInWei, subs4)

	// Its ignored
	require.Equal(t, Banned, state.Validators[3].ValidatorStatus)
	require.Equal(t, big.NewInt(0), state.Validators[3].PendingRewardsWei)
	// Collateral is returned
	require.Equal(t, big.NewInt(50+2500000), state.Validators[3].AccumulatedRewardsWei)
	// Same as before
	require.Equal(t, big.NewInt(1000070), state.Validators[3].CollateralWei)
}

func Test_Handle_TODO(t *testing.T) {
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
			DepositAddress: "0xac",
		},
		{
			ValidatorIndex: 20,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(1000000), // Enough
			BlockNumber:    0,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
		},
		{
			ValidatorIndex: 30,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(50), // Not enough
			BlockNumber:    0,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
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
		DepositAddress: "0ac",
	}
	state.HandleCorrectBlockProposal(block1)

	// Block from a non-subscribed validator (auto)
	block2 := Block{
		Slot:           0,
		ValidatorIndex: 40,
		ValidatorKey:   "0x",
		Reward:         big.NewInt(3333333),
		RewardType:     VanilaBlock,
		DepositAddress: "0ac",
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
}

// TODO: Add tests for add subscription and remove subscription
// TODO: Add more tests when spec settled
