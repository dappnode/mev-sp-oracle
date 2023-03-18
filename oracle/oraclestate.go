package oracle

import (
	"encoding/gob"
	"encoding/hex"
	"math/big"
	"os"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"

	log "github.com/sirupsen/logrus"
)

var StateFileName = "state.gob"

// States of the state machine
const (
	Eligible      int = 0
	YellowCard        = 1
	RedCard           = 2
	NotSubscribed     = 3
	Banned            = 4
)

// Events in the state machine that trigger transitions
const (
	ProposalWithCorrectFee int = 0 // TODO: rename as in the spec
	ProposalWithWrongFee       = 1
	MissedProposal             = 2
)

type ValidatorInfo struct {
	ValidatorStatus       int
	AccumulatedRewardsWei *big.Int // TODO not sure if this is gwei or wei
	PendingRewardsWei     *big.Int // TODO not sure if this is gwei or wei
	CollateralWei         *big.Int // TODO not sure if this is gwei or wei
	DepositAddress        string
	ValidatorIndex        string
	ValidatorKey          string
	ProposedBlocksSlots   []uint64
	MissedBlocksSlots     []uint64
	WrongFeeBlocksSlots   []uint64

	// TODO: Include ClaimedSoFar from the smart contract for reconciliation
}

type OracleState struct {
	LatestSlot  uint64
	Network     string
	PoolAddress string
	Validators  map[uint64]*ValidatorInfo
}

func (p *OracleState) SaveStateToFile() {
	file, err := os.Create(StateFileName)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	mRoot, enoughData := p.GetMerkleRootIfAny()

	encoder := gob.NewEncoder(file)
	log.WithFields(log.Fields{
		"File":            StateFileName,
		"LatestSlot":      p.LatestSlot,
		"TotalValidators": len(p.Validators),
		"Network":         p.Network,
		"PoolAddress":     p.PoolAddress,
		"MerkleRoot":      mRoot,
		"EnoughData":      enoughData,
	}).Info("Saving state to file")
	encoder.Encode(p)
}

func ReadStateFromFile() (*OracleState, error) {
	state := OracleState{}

	file, err := os.Open(StateFileName)

	if err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&state)
	if err != nil {
		log.Fatal(err)
	}

	mRoot, enoughData := state.GetMerkleRootIfAny()

	log.WithFields(log.Fields{
		"File":            StateFileName,
		"LatestSlot":      state.LatestSlot,
		"TotalValidators": len(state.Validators),
		"Network":         state.Network,
		"PoolAddress":     state.PoolAddress,
		"MerkleRoot":      mRoot,
		"EnoughData":      enoughData,
	}).Info("Loaded state from file")

	return &state, nil
}

func NewOracleState(cfg *config.Config) *OracleState {
	return &OracleState{
		// Start by default at the first slot when the oracle was deployed
		LatestSlot: cfg.DeployedSlot, // TODO: Not sure if -1

		// Onchain oracle info
		Network:     cfg.Network,
		PoolAddress: cfg.PoolAddress,

		Validators: make(map[uint64]*ValidatorInfo, 0),
	}
}

func (state *OracleState) IsValidatorSubscribed(validatorIndex uint64) bool {
	// TODO
	// Detect subscriptions with smart contract event. If subscribed but never unsubscribed, then it is subscribed

	for valIndex, validator := range state.Validators {
		if valIndex == validatorIndex && validator.ValidatorStatus != Banned && validator.ValidatorStatus != NotSubscribed {
			return true
		}
	}
	return false
}

func (state *OracleState) AddSubscriptionIfNotAlready(valIndex uint64, depositAddress string, validatorKey string) {
	validator, found := state.Validators[valIndex]
	if !found {
		validator = &ValidatorInfo{
			ValidatorStatus:       Eligible,
			AccumulatedRewardsWei: big.NewInt(0),
			PendingRewardsWei:     big.NewInt(0),
			DepositAddress:        depositAddress,
			ValidatorKey:          validatorKey,
			ProposedBlocksSlots:   make([]uint64, 0),
			MissedBlocksSlots:     make([]uint64, 0),
			WrongFeeBlocksSlots:   make([]uint64, 0),
		}
		state.Validators[valIndex] = validator
	}
}

func (state *OracleState) ConsolidateBalance(valIndex uint64) {
	state.Validators[valIndex].AccumulatedRewardsWei.Add(state.Validators[valIndex].AccumulatedRewardsWei, state.Validators[valIndex].PendingRewardsWei)
	state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

func (state *OracleState) GetEligibleValidators() []uint64 {
	eligibleValidators := make([]uint64, 0)

	for valIndex, validator := range state.Validators {
		if validator.ValidatorStatus == Eligible || validator.ValidatorStatus == YellowCard {
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

	for _, eligibleIndex := range eligibleValidators {
		state.Validators[eligibleIndex].PendingRewardsWei.Add(state.Validators[eligibleIndex].PendingRewardsWei, amountToIncrease)
	}
}

func (state *OracleState) ResetPendingRewards(valIndex uint64) {
	state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

func (state *OracleState) LogPendingBalances() {
	for valIndex, validator := range state.Validators {
		log.Info("SlotState: ", state.LatestSlot, " Pending: ", valIndex, ": ", validator.PendingRewardsWei)
	}
}

func (state *OracleState) LogClaimableBalances() {
	for valIndex, validator := range state.Validators {
		log.Info("SlotState: ", state.LatestSlot, " Claimable: ", valIndex, ": ", validator.AccumulatedRewardsWei)
	}
}

// See spec for state machine.
// TODO: Review this!!
func (state *OracleState) AdvanceStateMachine(valIndex uint64, event int) {
	switch state.Validators[valIndex].ValidatorStatus {
	// TODO: Print also event + state change
	case Eligible:
		switch event {
		case ProposalWithCorrectFee:
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> Active")
		case ProposalWithWrongFee:
			state.Validators[valIndex].ValidatorStatus = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> Banned")
		case MissedProposal:
			state.Validators[valIndex].ValidatorStatus = YellowCard
			log.Info("ValIndex: ", valIndex, " state change: ", "Active -> ActiveWarned")
		}
	case YellowCard:
		switch event {
		case ProposalWithCorrectFee:
			state.Validators[valIndex].ValidatorStatus = Eligible
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> Active")
		case ProposalWithWrongFee:
			state.Validators[valIndex].ValidatorStatus = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> Banned")
		case MissedProposal:
			state.Validators[valIndex].ValidatorStatus = NotSubscribed // TODO: probably wrong
			log.Info("ValIndex: ", valIndex, " state change: ", "ActiveWarned -> NotActive")
		}
	case NotSubscribed:
		switch event {
		case ProposalWithCorrectFee:
			state.Validators[valIndex].ValidatorStatus = YellowCard
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> ActiveWarned")
		case ProposalWithWrongFee:
			state.Validators[valIndex].ValidatorStatus = Banned
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> Banned")
		case MissedProposal:
			log.Info("ValIndex: ", valIndex, " state change: ", "NotActive -> NotActive")
		}
	}
}

// Dumps all the oracle state to the db
// Note that this is a proof of concept. All data is stored in the memory
// and dumped to the db on each checkpoint, but at some point
// this may become unfeasible.

/* TODO: Move this somewhere else
func (state *OracleState) DumpOracleStateToDatabase() (error, string, bool) { // TOOD: returning here the merkle root doesnt make sense. quick workaround
	log.Info("Dumping all state to database")

	// TODO: Define a type on validator parameters to store and stop
	// using that many maps

	mk := NewMerklelizer()
	// TODO: returning orderedRawLeafs as a quick workaround to get the proofs
	depositToLeaf, depositToRawLeaf, tree, enoughData := mk.GenerateTreeFromState(state)
	if !enoughData {
		return nil, "", enoughData
	}
	merkleRootStr := hex.EncodeToString(tree.Root)
	log.Info("Merkle root: ", merkleRootStr)

	// TODO: Add also validator key on top of the index
	for valIndex, validator := range state.Validators {
		log.Info("Generating root for deposit: ", validator.DepositAddress)
		block := depositToLeaf[validator.DepositAddress]
		serrr, err := block.Serialize()
		if err != nil {
			log.Fatal("Error serializing block", err)
		}
		log.Info("Hash of leaf is: ", hex.EncodeToString(serrr))
		proof, err := tree.GenerateProof(block)
		if err != nil {
			log.Fatal("Error generating proof", err)
		}

		_, err = state.Postgres.Db.Exec(
			context.Background(),
			postgres.InsertRewardsTable,

			validator.DepositAddress, //TODO: This is empty?
			validator.ValidatorKey,
			valIndex,
			validator.PendingRewardsWei.Uint64(), // TODO: can we overflow a uint64?
			validator.AccumulatedRewardsWei.Uint64(),
			uint64(0), // TODO: remove unbann balance
			len(validator.ProposedBlocksSlots),
			len(validator.MissedBlocksSlots),
			len(validator.WrongFeeBlocksSlots),
			state.LatestSlot,
			ByteArrayToStringArray(proof.Siblings),
			"0x"+hex.EncodeToString(tree.Root))
		if err != nil {
			return err, "", false
		}
	}

	_ = depositToRawLeaf

	for depositAddress, rawLeaf := range depositToRawLeaf {
		// Extra check to make sure the deposit address is the same as the key
		if depositAddress != rawLeaf.DepositAddress {
			log.Fatal("Deposit address in raw leaf doesnt match the key")
		}
		log.Info("deposit", depositAddress)
		log.Info("rawLeaf", rawLeaf)

		// TODO some duplicated code here
		block := depositToLeaf[depositAddress]
		proof, err := tree.GenerateProof(block)

		test := ByteArrayToStringArray(proof.Siblings)
		_ = test

		_, err = state.Postgres.Db.Exec(
			context.Background(),
			postgres.InsertDepositAddressRewardsTable,
			depositAddress,
			"TODO: add keys for this address",
			uint64(0), // TODO: pending rewards. is it stored somewhere else?
			rawLeaf.AccumulatedBalance.Uint64(),
			uint64(0), //TODO remove unbann balance,
			state.LatestSlot,
			ByteArrayToStringArray(proof.Siblings),
			"0x"+hex.EncodeToString(tree.Root),
		)
		if err != nil {
			// improve error handling
			log.Fatal(err)
			//return err, ""
		}

	}

	return nil, merkleRootStr, true

}
*/

func (state *OracleState) GetMerkleRootIfAny() (string, bool) {
	mk := NewMerklelizer()
	// TODO: returning orderedRawLeafs as a quick workaround to get the proofs
	_, _, tree, enoughData := mk.GenerateTreeFromState(state)
	if !enoughData {
		return "", enoughData
	}
	merkleRootStr := hex.EncodeToString(tree.Root)
	log.Info("Merkle root: ", merkleRootStr)

	return merkleRootStr, true
}
