package oracle

import (
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
		From  int
		Event int
		End   int
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
		PoolAddress: "0x0000000000000000000000000000000000000000",
		Network:     "mainnet",
	})

	state.Validators[10] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(1000),
		PendingRewardsWei:     big.NewInt(1000),
		CollateralWei:         big.NewInt(1000),
		DepositAddress:        "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        "0xb000000000000000000000000000000000000000",
		ValidatorKey:          "0xc", // TODO: Fix this, should be uint64
		ProposedBlocksSlots: []BlockState{
			BlockState{
				Reward:    big.NewInt(1000),
				BlockType: VanilaBlock,
				Slot:      1000,
			}, BlockState{
				Reward:    big.NewInt(12000),
				BlockType: VanilaBlock,
				Slot:      3000,
			}, BlockState{
				Reward:    big.NewInt(7000),
				BlockType: MevBlock,
				Slot:      6000,
			}},
		MissedBlocksSlots: []BlockState{BlockState{
			Reward:    big.NewInt(1000),
			BlockType: VanilaBlock,
			Slot:      500,
		}, BlockState{
			Reward:    big.NewInt(1000),
			BlockType: VanilaBlock,
			Slot:      12000,
		}},
		WrongFeeBlocksSlots: []BlockState{BlockState{
			Reward:    big.NewInt(1000),
			BlockType: VanilaBlock,
			Slot:      500,
		}, BlockState{
			Reward:    big.NewInt(1000),
			BlockType: VanilaBlock,
			Slot:      12000,
		}},
	}

	state.Validators[20] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(13000),
		PendingRewardsWei:     big.NewInt(100),
		CollateralWei:         big.NewInt(1000000),
		DepositAddress:        "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        "0xb000000000000000000000000000000000000000",
		ValidatorKey:          "0xc",
		ProposedBlocksSlots: []BlockState{
			BlockState{
				Reward:    big.NewInt(1000),
				BlockType: VanilaBlock,
				Slot:      1000,
			}, BlockState{
				Reward:    big.NewInt(12000),
				BlockType: VanilaBlock,
				Slot:      3000,
			}, BlockState{
				Reward:    big.NewInt(7000),
				BlockType: MevBlock,
				Slot:      6000,
			}},
		MissedBlocksSlots: []BlockState{BlockState{
			Reward:    big.NewInt(33000),
			BlockType: VanilaBlock,
			Slot:      800,
		}, BlockState{
			Reward:    big.NewInt(11000),
			BlockType: VanilaBlock,
			Slot:      15000,
		}},
		WrongFeeBlocksSlots: []BlockState{BlockState{
			Reward:    big.NewInt(14000),
			BlockType: VanilaBlock,
			Slot:      700,
		}, BlockState{
			Reward:    big.NewInt(18000),
			BlockType: VanilaBlock,
			Slot:      19000,
		}},
	}

	state.Validators[30] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(53000),
		PendingRewardsWei:     big.NewInt(000),
		CollateralWei:         big.NewInt(4000000),
		DepositAddress:        "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        "0xb000000000000000000000000000000000000000",
		ValidatorKey:          "0xc",
		// Empty Proposed blocks
		MissedBlocksSlots: []BlockState{BlockState{
			Reward:    big.NewInt(303000),
			BlockType: VanilaBlock,
			Slot:      12200,
		}},
		WrongFeeBlocksSlots: []BlockState{BlockState{
			Reward:    big.NewInt(15000),
			BlockType: VanilaBlock,
			Slot:      800,
		}, BlockState{
			Reward:    big.NewInt(189000),
			BlockType: VanilaBlock,
			Slot:      232000,
		}},
	}

	StateFileName = "test_state.gob"
	defer os.Remove(StateFileName)
	state.SaveStateToFile()

	recovered, err := ReadStateFromFile()
	require.NoError(t, err)
	require.Equal(t, state, recovered)
}

// TODO: Add more tests when spec settled
