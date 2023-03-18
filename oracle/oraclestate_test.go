package oracle

import (
	"math/big"
	"mev-sp-oracle/config"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
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

func Test_IncreasePendingRewards(t *testing.T) {
	state := NewOracleState(&config.Config{})
	state.Validators[12] = &ValidatorInfo{
		DepositAddress:    "0xaa",
		ValidatorStatus:   Eligible,
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

	_ = state
	_ = valIndex1
	_ = valIndex2

	type stateTest struct {
		From  int
		Event int
		End   int
	}

	stateMachineTestVector := []stateTest{
		/*
			{Active, ProposalWithCorrectFee, Active},
			{Active, ProposalWithWrongFee, Banned},
			{Active, MissedProposal, ActiveWarned},
		*/
		//{Active, UnbanValidator, Active}, // TODO: Test that fails

		/*
			{ActiveWarned, ProposalWithCorrectFee, Active},
			{ActiveWarned, ProposalWithWrongFee, Banned},
			{ActiveWarned, MissedProposal, NotActive},
		*/
		//{ActiveWarned, UnbanValidator, ActiveWarned}, // TODO: Test that fails

		/*
			{NotActive, ProposalWithCorrectFee, ActiveWarned},
			{NotActive, ProposalWithWrongFee, Banned},
			{NotActive, MissedProposal, MissedProposal},
		*/
		// {NotActive, UnbanValidator, NotActive}, // TODO: Test that fails

		{Banned, ProposalWithCorrectFee, Banned},
		{Banned, ProposalWithWrongFee, Banned},
		//{Banned, MissedProposal, Banned},
	}
	_ = stateMachineTestVector

	/*
		for _, testState := range stateMachineTestVector {
			state.ValidatorState[valIndex1] = testState.From
			state.ValidatorState[valIndex2] = testState.From

			state.AdvanceStateMachine(valIndex1, testState.Event)
			state.AdvanceStateMachine(valIndex2, testState.Event)

			require.Equal(t, testState.End, state.ValidatorState[valIndex1])
			require.Equal(t, testState.End, state.ValidatorState[valIndex2])
		}
	*/
}
func Test_SaveLoadFromToFile(t *testing.T) {

	original := &OracleState{
		LatestSlot:  1,
		Network:     "mainnet",
		PoolAddress: "0x1234",
		Validators:  make(map[uint64]*ValidatorInfo),
	}

	original.Validators[1] = &ValidatorInfo{
		ValidatorStatus:       Eligible,
		AccumulatedRewardsWei: big.NewInt(1000),
		PendingRewardsWei:     big.NewInt(1000),
		CollateralWei:         big.NewInt(1000),
		DepositAddress:        "0xa",
		ValidatorIndex:        "0xb",
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

	original.Validators[13] = &ValidatorInfo{
		ValidatorStatus:       Eligible,
		AccumulatedRewardsWei: big.NewInt(13000),
		PendingRewardsWei:     big.NewInt(100),
		CollateralWei:         big.NewInt(1000000),
		DepositAddress:        "0xa",
		ValidatorIndex:        "0xb",
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

	original.Validators[200] = &ValidatorInfo{
		ValidatorStatus:       Eligible,
		AccumulatedRewardsWei: big.NewInt(53000),
		PendingRewardsWei:     big.NewInt(000),
		CollateralWei:         big.NewInt(4000000),
		DepositAddress:        "0xa",
		ValidatorIndex:        "0xb",
		ValidatorKey:          "0xc",
		ProposedBlocksSlots:   []BlockState{},
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
	original.SaveStateToFile()
	defer os.Remove(StateFileName)

	recovered, err := ReadStateFromFile()
	require.NoError(t, err)

	log.Info("orgiinal", original)
	log.Info("recovered", recovered)

	require.Equal(t, original, recovered)
}

// TODO: Add more tests when spec settled
