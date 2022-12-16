package oracle

import (
	"math/big"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"

	log "github.com/sirupsen/logrus"
)

// enum type in go?
// example: https://github.com/prysmaticlabs/prysm/blob/6bea17cb546eaad84613ce7e06fd8685eec8a806/beacon-chain/p2p/peers/status.go#L50-L59
const (
	Active       int = 0
	ActiveWarned     = 1
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
// TODO state into is not a good name
type OracleState struct {
	// When the state was updated
	timestampGeneration string // TODO unused

	// Amount of processed blocks
	ProcessedSlots uint64

	// Slot of this state
	Slot        uint64
	Network     string
	PoolAddress string

	// TODO: Rough idea
	//activeSubscriptions []uint64

	PendingRewards   map[uint64]*big.Int // TODO add wei or gwei to all fucking variables.
	ClaimableRewards map[uint64]*big.Int
	UnbanBalance     map[uint64]*big.Int

	// TODO: Rename to states
	validatorState map[uint64]int // TODO not sure if the enum has a type.

	// extra info
	proposedBlocks map[uint64][]uint64
	missedBlocks   map[uint64][]uint64
	wrongFeeBlocks map[uint64][]uint64

	// TODO: Mev contributions to the pool
	// map[uint64][]*big.Int
}

func NewOracleState(cfg *config.Config) *OracleState {
	return &OracleState{
		// Start by default at the first slot when the oracle was deployed
		Slot: cfg.DeployedSlot,

		// Assume no block were processed
		ProcessedSlots: uint64(0),

		// Onchain oracle info
		Network:     cfg.Network,
		PoolAddress: cfg.PoolAddress,

		PendingRewards:   make(map[uint64]*big.Int),
		ClaimableRewards: make(map[uint64]*big.Int),
		UnbanBalance:     make(map[uint64]*big.Int),
		validatorState:   make(map[uint64]int),
		proposedBlocks:   make(map[uint64][]uint64),
		missedBlocks:     make(map[uint64][]uint64),
		wrongFeeBlocks:   make(map[uint64][]uint64),
	}
}

func (state *OracleState) AddSubscriptionIfNotAlready(valIndex uint64) {
	// TODO: refactor this function
	if state.PendingRewards[valIndex] != nil {
		return
	}
	state.PendingRewards[valIndex] = big.NewInt(0)
	state.ClaimableRewards[valIndex] = big.NewInt(0)
	state.UnbanBalance[valIndex] = big.NewInt(0)
	state.validatorState[valIndex] = Active
	state.proposedBlocks[valIndex] = make([]uint64, 0)
	state.missedBlocks[valIndex] = make([]uint64, 0)
	state.wrongFeeBlocks[valIndex] = make([]uint64, 0)
}

func (state *OracleState) ConsolidateBalance(valIndex uint64) {
	state.ClaimableRewards[valIndex].Add(state.ClaimableRewards[valIndex], state.PendingRewards[valIndex])
	state.PendingRewards[valIndex] = big.NewInt(0)
}

func (state *OracleState) GetEligibleValidators() []uint64 {
	eligibleValidators := make([]uint64, 0)
	for valIndex, score := range state.validatorState {
		if score == Active || score == ActiveWarned {
			eligibleValidators = append(eligibleValidators, valIndex)
		}
	}
	return eligibleValidators
}

func (state *OracleState) IncreaseAllPendingRewards(
	totalAmount *big.Int) {

	eligibleValidators := state.GetEligibleValidators()
	numEligibleValidators := big.NewInt(int64(len(eligibleValidators)))

	amountToIncrease := big.NewInt(0).Div(totalAmount, numEligibleValidators)
	// TODO: Rounding problems. Evenly distribute the remainder

	for _, pending := range state.PendingRewards {
		pending.Add(pending, amountToIncrease)
	}
}

func (state *OracleState) ResetPendingRewards(valIndex uint64) {
	state.PendingRewards[valIndex] = big.NewInt(0)
}

func (state *OracleState) SetUnbanBalance(valIndex uint64, amount *big.Int) {
	state.UnbanBalance[valIndex] = amount
}

func (state *OracleState) GetState(valIndex uint64) int {
	return state.validatorState[valIndex]
}

func (state *OracleState) IsActive(valIndex uint64) bool {
	return state.validatorState[valIndex] == Active
}

func (state *OracleState) IsActiveWarned(valIndex uint64) bool {
	return state.validatorState[valIndex] == ActiveWarned
}

func (state *OracleState) IsNotActive(valIndex uint64) bool {
	return state.validatorState[valIndex] == NotActive
}

func (state *OracleState) IsBanned(valIndex uint64) bool {
	return state.validatorState[valIndex] == Banned
}

// See spec for state machine.
func (state *OracleState) AdvanceStateMachine(valIndex uint64, event int) {
	switch state.validatorState[valIndex] {
	case Active:
		switch event {
		case ProposalWithCorrectFee:
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> Active")
		case ProposalWithWrongFee:
			state.validatorState[valIndex] = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> Banned")
		case MissedProposal:
			state.validatorState[valIndex] = ActiveWarned
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> ActiveWarned")
		case UnbanValidator:
			log.Fatal("Can't receive UnbanValidator event in state: Active")
		}
	case ActiveWarned:
		switch event {
		case ProposalWithCorrectFee:
			state.validatorState[valIndex] = Active
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> Active")
		case ProposalWithWrongFee:
			state.validatorState[valIndex] = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> Banned")
		case MissedProposal:
			state.validatorState[valIndex] = NotActive
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> NotActive")
		case UnbanValidator:
			log.Fatal("Can't receive UnbanValidator event in state: ActiveWarned")
		}
	case NotActive:
		switch event {
		case ProposalWithCorrectFee:
			state.validatorState[valIndex] = ActiveWarned
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> ActiveWarned")
		case ProposalWithWrongFee:
			state.validatorState[valIndex] = Banned
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
			state.validatorState[valIndex] = Active
			log.Info("ValIndex: ", valIndex, " state change: ", "Banned -> UnbanValidator")
		}
	}
}
