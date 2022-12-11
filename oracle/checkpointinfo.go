package oracle

import (
	"math/big"

	log "github.com/sirupsen/logrus"
)

// enum type in go?
// example: https://github.com/prysmaticlabs/prysm/blob/6bea17cb546eaad84613ce7e06fd8685eec8a806/beacon-chain/p2p/peers/status.go#L50-L59
const (
	Active       int = 0
	ActiveWarned     = 1 // pol: yellow card!
	NotActive        = 2
	Banned           = 3
)

// events from the state machine
const (
	ProposalWithCorrectFee int = 0
	ProposalWithWrongFee       = 1
	MissedProposal             = 2
	UnbanValidator             = 3
)

// TODO: add stuff to serialize to json
// TODO checkpoint into is not a good name
type CheckpointInfo struct {
	timestampGeneration string
	blockStart          string
	blockEnd            string
	amountOfBlocks      string
	blockHeigh          string
	slotHeigh           string
	network             string
	poolContract        string

	pendingRewards   map[uint64]*big.Int // TODO add wei or gwei to all fucking variables.
	claimableRewards map[uint64]*big.Int
	unbanBalance     map[uint64]*big.Int

	// TODO: Rename to states
	scores map[uint64]int // TODO not sure if the enum has a type.

	// extra info
	proposedBlocks map[uint64][]uint64
	missedBlocks   map[uint64][]uint64
	wrongFeeBlocks map[uint64][]uint64

	// TODO: Mev contributions to the pool
	// map[uint64][]*big.Int
}

func NewCheckpointInfo() *CheckpointInfo {
	return &CheckpointInfo{
		// TODO: set other info.

		pendingRewards:   make(map[uint64]*big.Int),
		claimableRewards: make(map[uint64]*big.Int),
		unbanBalance:     make(map[uint64]*big.Int),
		scores:           make(map[uint64]int), // or rename to state?
		proposedBlocks:   make(map[uint64][]uint64),
		missedBlocks:     make(map[uint64][]uint64),
		wrongFeeBlocks:   make(map[uint64][]uint64),
	}
}

func (checkpoint *CheckpointInfo) InitValIndexData(valIndex uint64) {
	checkpoint.pendingRewards[valIndex] = big.NewInt(0)
	checkpoint.claimableRewards[valIndex] = big.NewInt(0)
	checkpoint.unbanBalance[valIndex] = big.NewInt(0)
	checkpoint.scores[valIndex] = Active
	checkpoint.proposedBlocks[valIndex] = make([]uint64, 0)
	checkpoint.missedBlocks[valIndex] = make([]uint64, 0)
	checkpoint.wrongFeeBlocks[valIndex] = make([]uint64, 0)
}

// bad naming
func (checkpoint *CheckpointInfo) InitWithSubscriptions(subs *Subscriptions) {
	for valIndex, _ := range subs.subscriptions {
		checkpoint.InitValIndexData(valIndex)
	}
}

func (checkpoint *CheckpointInfo) ConsolidateBalance(valIndex uint64) {
	checkpoint.claimableRewards[valIndex].Add(checkpoint.claimableRewards[valIndex], checkpoint.pendingRewards[valIndex])
	checkpoint.pendingRewards[valIndex] = big.NewInt(0)
}

func (checkpoint *CheckpointInfo) MissedBlock(valIndex uint64) {
	// TODO:
}

func (checkpoint *CheckpointInfo) GetEligibleValidators() []uint64 {
	eligibleValidators := make([]uint64, 0)
	for valIndex, score := range checkpoint.scores {
		if score == Active || score == ActiveWarned {
			eligibleValidators = append(eligibleValidators, valIndex)
		}
	}
	return eligibleValidators
}

func (checkpoint *CheckpointInfo) IncreaseAllPendingRewards(
	totalAmount *big.Int) {

	eligibleValidators := checkpoint.GetEligibleValidators()
	numEligibleValidators := big.NewInt(int64(len(eligibleValidators)))

	amountToIncrease := big.NewInt(0).Div(totalAmount, numEligibleValidators)
	// TODO: Rounding problems.
	// randomly distribute the remainder to the eligible validators

	// TODO: missing the subscriptions??

	for _, pending := range checkpoint.pendingRewards {
		pending.Add(pending, amountToIncrease)
	}
}

// todo perhaps add this and next function.
func (checkpoint *CheckpointInfo) ResetPendingRewards(valIndex uint64) {
	checkpoint.pendingRewards[valIndex] = big.NewInt(0)
}

func (checkpoint *CheckpointInfo) SetUnbanBalance(valIndex uint64, amount *big.Int) {
	checkpoint.unbanBalance[valIndex] = amount
}

func (checkpoint *CheckpointInfo) GetState(valIndex uint64) int {
	return checkpoint.scores[valIndex] // todo rename to state
}

func (checkpoint *CheckpointInfo) IsActive(valIndex uint64) bool {
	return checkpoint.scores[valIndex] == Active
}

func (checkpoint *CheckpointInfo) IsActiveWarned(valIndex uint64) bool {
	return checkpoint.scores[valIndex] == ActiveWarned
}

func (checkpoint *CheckpointInfo) IsNotActive(valIndex uint64) bool {
	return checkpoint.scores[valIndex] == NotActive
}

func (checkpoint *CheckpointInfo) IsBanned(valIndex uint64) bool {
	return checkpoint.scores[valIndex] == Banned
}

// TODO: Link to state machine diagram.
func (checkpoint *CheckpointInfo) AdvanceStateMachine(valIndex uint64, event int) {
	// TODO add [slot] to all logs
	switch checkpoint.scores[valIndex] {
	case Active:
		switch event {
		case ProposalWithCorrectFee:
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> Active")
		case ProposalWithWrongFee:
			checkpoint.scores[valIndex] = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> Banned")
		case MissedProposal:
			checkpoint.scores[valIndex] = ActiveWarned
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> ActiveWarned")
		case UnbanValidator:
			log.Fatal("Can't receive UnbanValidator event in state: Active")
		}
	case ActiveWarned:
		switch event {
		case ProposalWithCorrectFee:
			checkpoint.scores[valIndex] = Active
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> Active")
		case ProposalWithWrongFee:
			checkpoint.scores[valIndex] = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> Banned")
		case MissedProposal:
			checkpoint.scores[valIndex] = NotActive
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> NotActive")
		case UnbanValidator:
			log.Fatal("Can't receive UnbanValidator event in state: ActiveWarned")
		}
	case NotActive:
		switch event {
		case ProposalWithCorrectFee:
			checkpoint.scores[valIndex] = ActiveWarned
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> ActiveWarned")
		case ProposalWithWrongFee:
			checkpoint.scores[valIndex] = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> Banned")
		case MissedProposal:
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> NotActive")
		case UnbanValidator:
			log.Fatal("Can't receive UnbanValidator event in state: NotActive")
		}
	case Banned:
		switch event {
		case ProposalWithCorrectFee:
			log.Info("ValIndex: ", valIndex, " event: ProposalWithCorrectFee but Banned. Do nothing")
		case ProposalWithWrongFee:
			log.Info("ValIndex: ", valIndex, " event: ProposalWithWrongFee but Banned. Do nothing")
		case MissedProposal:
			log.Info("ValIndex: ", valIndex, " event: MissedProposal but Banned. Do nothing")
		case UnbanValidator:
			checkpoint.scores[valIndex] = Active
			log.Info("ValIndex: ", valIndex, " state change: ", "Banned -> UnbanValidator")
		}
	}
}
