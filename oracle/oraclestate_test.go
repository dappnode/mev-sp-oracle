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
		Slot:        1,
		Network:     "mainnet",
		PoolAddress: "0x1234",
		Validators:  make(map[uint64]*ValidatorInfo),
	}

	// TODO: Add all fields in ValidatorState
	original.Validators[1] = &ValidatorInfo{ValidatorStatus: Eligible}
	original.Validators[90] = &ValidatorInfo{ValidatorStatus: YellowCard}

	err := original.SaveStateToFile()
	defer os.Remove("state.gob")
	require.NoError(t, err)

	recovered, err := ReadStateFromFile()

	log.Info("orgiinal", original)
	log.Info("recovered", recovered)

	require.Equal(t, original, recovered)
}

// TODO: Add more tests when spec settled
