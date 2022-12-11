package oracle

import (
	"math/big"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Todo: perhaps move. e2e test, requiere a beacon node.
func Test_TODO10(t *testing.T) {

	require.Equal(t, 1, 1)
	log.Info("checkpoint", "hi")
}

func Test_IncreasePendingRewards(t *testing.T) {
	checkpoint := NewCheckpointInfo()
	checkpoint.pendingRewards[12] = big.NewInt(100)
	checkpoint.scores[12] = Active
	totalAmount := big.NewInt(130)
	log.Info("dummy")

	log.Info("pending before", checkpoint.pendingRewards)

	checkpoint.IncreaseAllPendingRewards(totalAmount)

	log.Info("pending after", checkpoint.pendingRewards)
}

func Test_ConsolidateBalance_Eligible(t *testing.T) {
	valIndex := uint64(10)
	checkpoint := NewCheckpointInfo()
	checkpoint.claimableRewards[valIndex] = big.NewInt(77)
	checkpoint.pendingRewards[valIndex] = big.NewInt(23)
	checkpoint.scores[valIndex] = Active

	log.Info("before claimableRewards", checkpoint.claimableRewards)
	log.Info("before pendingRewards", checkpoint.pendingRewards)
	log.Info("before scores", checkpoint.scores)

	checkpoint.ConsolidateBalance(valIndex)

	log.Info("after claimableRewards", checkpoint.claimableRewards)
	log.Info("after pendingRewards", checkpoint.pendingRewards)
	log.Info("after scores", checkpoint.scores)
}

func Test_ConsolidateBalance_Strike1(t *testing.T) {
	valIndex := uint64(10)
	checkpoint := NewCheckpointInfo()
	checkpoint.claimableRewards[valIndex] = big.NewInt(77)
	checkpoint.pendingRewards[valIndex] = big.NewInt(23)
	checkpoint.scores[valIndex] = ActiveWarned

	log.Info("before claimableRewards", checkpoint.claimableRewards)
	log.Info("before pendingRewards", checkpoint.pendingRewards)
	log.Info("before scores", checkpoint.scores)

	checkpoint.ConsolidateBalance(valIndex)

	log.Info("after claimableRewards", checkpoint.claimableRewards)
	log.Info("after pendingRewards", checkpoint.pendingRewards)
	log.Info("after scores", checkpoint.scores)

	//TODO Add asserts
}

// fucking garbage, boilerplate just PoC
func Test_ConsolidateBalance_Strike2(t *testing.T) {
	valIndex := uint64(10)
	checkpoint := NewCheckpointInfo()
	checkpoint.claimableRewards[valIndex] = big.NewInt(77)
	checkpoint.pendingRewards[valIndex] = big.NewInt(23)
	checkpoint.scores[valIndex] = NotActive

	log.Info("before claimableRewards", checkpoint.claimableRewards)
	log.Info("before pendingRewards", checkpoint.pendingRewards)
	log.Info("before scores", checkpoint.scores)

	// when it proposes a block
	checkpoint.ConsolidateBalance(valIndex)

	log.Info("after claimableRewards", checkpoint.claimableRewards)
	log.Info("after pendingRewards", checkpoint.pendingRewards)
	log.Info("after scores", checkpoint.scores)

	// and another block
	checkpoint.ConsolidateBalance(valIndex)

	log.Info("after claimableRewards", checkpoint.claimableRewards)
	log.Info("after pendingRewards", checkpoint.pendingRewards)
	log.Info("after scores", checkpoint.scores)

	// TODO Add asserts
}

func Test_StateMachine(t *testing.T) {
	checkpoint := NewCheckpointInfo()
	valIndex1 := uint64(1000)
	valIndex2 := uint64(2000)

	type stateTest struct {
		From  int
		Event int
		End   int
	}

	stateMachineTestVector := []stateTest{
		{Active, ProposalWithCorrectFee, Active},
		{Active, ProposalWithWrongFee, Banned},
		{Active, MissedProposal, ActiveWarned},
		//{Active, UnbanValidator, Active}, // TODO: Test that fails

		{ActiveWarned, ProposalWithCorrectFee, Active},
		{ActiveWarned, ProposalWithWrongFee, Banned},
		{ActiveWarned, MissedProposal, NotActive},
		//{ActiveWarned, UnbanValidator, ActiveWarned}, // TODO: Test that fails

		{NotActive, ProposalWithCorrectFee, ActiveWarned},
		{NotActive, ProposalWithWrongFee, Banned},
		{NotActive, MissedProposal, MissedProposal},
		// {NotActive, UnbanValidator, NotActive}, // TODO: Test that fails

		{Banned, ProposalWithCorrectFee, Banned},
		{Banned, ProposalWithWrongFee, Banned},
		{Banned, MissedProposal, Banned},
		{Banned, UnbanValidator, Active},
	}

	for _, testState := range stateMachineTestVector {
		checkpoint.scores[valIndex1] = testState.From
		checkpoint.scores[valIndex2] = testState.From

		checkpoint.AdvanceStateMachine(valIndex1, testState.Event)
		checkpoint.AdvanceStateMachine(valIndex2, testState.Event)

		require.Equal(t, testState.End, checkpoint.scores[valIndex1])
		require.Equal(t, testState.End, checkpoint.scores[valIndex2])
	}
}

// TODO: Add tests for capella fork

// TODO: safety chgeck. use Timestamp from prev block and check its increasing.
