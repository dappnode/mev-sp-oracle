package oracle

import (
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"

	"github.com/dappnode/mev-sp-oracle/config"

	log "github.com/sirupsen/logrus"
	mt "github.com/txaty/go-merkletree"
)

// Description of the state machine:
// -State: States of the validators, related to wether they earn rewards or not.
// -Events: Actions that can trigger and state transition from state a to state b.
// -Handlers: Action that is performed after an event is triggered when landing a new state.

// Default filename to persist the state of the oracle
var StateFileName = "state.gob"

type RewardType uint8
type ValidatorStatus uint8
type Event uint8
type BlockType uint8

// TODO: Dont export functions and variables that are not used outside the package

// Types of block rewards
const (
	UnknownRewardType RewardType = 0
	VanilaBlock       RewardType = 1
	MevBlock          RewardType = 2
)

// States of the state machine
const (
	UnknownState  ValidatorStatus = 0
	Active        ValidatorStatus = 1
	YellowCard    ValidatorStatus = 2
	RedCard       ValidatorStatus = 3
	NotSubscribed ValidatorStatus = 4
	Banned        ValidatorStatus = 5
	Untracked     ValidatorStatus = 6
)

// Events in the state machine that trigger transitions
const (
	UnknownEvent       Event = 0
	ProposalOk         Event = 1
	ProposalMissed     Event = 2
	ProposalWrongFee   Event = 3
	ManualSubscription Event = 4
	AutoSubscription   Event = 5
	Unsubscribe        Event = 6
)

// Block type
const (
	UnknownBlockType  BlockType = 0
	MissedProposal    BlockType = 1
	WrongFeeRecipient BlockType = 2
	OkPoolProposal    BlockType = 3
)

// Represents a block with information relevant for the pool
type Block struct {
	Slot           uint64     `json:"slot"`
	ValidatorIndex uint64     `json:"validator_index"`
	ValidatorKey   string     `json:"validator_key"`
	BlockType      BlockType  `json:"block_type"`
	Reward         *big.Int   `json:"reward_wei"`
	RewardType     RewardType `json:"reward_type"`
	DepositAddress string     `json:"deposit_address"`
}

// Represents a donation made to the pool
type Donation struct {
	AmountWei *big.Int `json:"amount_wei"`
	Block     uint64   `json:"block_number"`
	TxHash    string   `json:"tx_hash"`
}

// Subscription of a validator to the pool
type Subscription struct {
	ValidatorIndex uint64   `json:"validator_index"`
	ValidatorKey   string   `json:"validator_key"`
	Collateral     *big.Int `json:"collateral_wei"`
	BlockNumber    uint64   `json:"block_number"`
	TxHash         string   `json:"tx_hash"`
	DepositAddress string   `json:"deposit_address"`
}

// Unsubscription of a validator from the pool
type Unsubscription struct {
	ValidatorIndex uint64 `json:"validator_index"`
	ValidatorKey   string `json:"validator_key"`
	Sender         string `json:"sender"`
	BlockNumber    uint64 `json:"block_number"`
	TxHash         string `json:"tx_hash"`
	DepositAddress string `json:"deposit_address"`
}

// Represents all the information that is stored of a validator
type ValidatorInfo struct {
	ValidatorStatus         ValidatorStatus `json:"status"`
	AccumulatedRewardsWei   *big.Int        `json:"accumulated_rewards_wei"`
	PendingRewardsWei       *big.Int        `json:"pending_rewards_wei"`
	CollateralWei           *big.Int        `json:"collateral_wei"`
	DepositAddress          string          `json:"deposit_address"`
	ValidatorIndex          uint64          `json:"validator_index"`
	ValidatorKey            string          `json:"validator_key"`
	ValidatorProposedBlocks []Block         `json:"proposed_block"`
	ValidatorMissedBlocks   []Block         `json:"missed_blocks"`
	ValidatorWrongFeeBlocks []Block         `json:"wrong_fee_blocks"`

	// TODO: Include ClaimedSoFar from the smart contract for reconciliation
}

// Represents the latest commited state onchain
type OnchainState struct {
	Validators map[uint64]*ValidatorInfo
	Slot       uint64
	TxHash     string
	MerkleRoot string

	Tree   *mt.MerkleTree
	Proofs map[string][]string
	Leafs  map[string]RawLeaf
}

type OracleState struct {
	LatestSlot          uint64
	Network             string
	PoolAddress         string
	Validators          map[uint64]*ValidatorInfo
	LatestCommitedState OnchainState

	PoolFeesPercent     int
	PoolFeesAddress     string
	PoolAccumulatedFees *big.Int

	Subscriptions   []Subscription   // TODO: Populate (unsure if needed)
	Unsubscriptions []Unsubscription // TODO: Populate (unsure if needed)
	Donations       []Donation
	ProposedBlocks  []Block
	MissedBlocks    []Block
	WrongFeeBlocks  []Block
}

func (p *OracleState) SaveStateToFile() {
	file, err := os.Create(StateFileName)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// Dont run this again, take the existing data
	//mRoot, enoughData := p.GetMerkleRootIfAny()

	encoder := gob.NewEncoder(file)
	log.WithFields(log.Fields{
		"File":            StateFileName,
		"LatestSlot":      p.LatestSlot,
		"TotalValidators": len(p.Validators),
		"Network":         p.Network,
		"PoolAddress":     p.PoolAddress,
		//"MerkleRoot":      mRoot,
		//"EnoughData":      enoughData,
	}).Info("Saving state to file")
	encoder.Encode(p)
}

func ReadStateFromFile() (*OracleState, error) {
	// Init all fields in case any was stored empty in the file
	state := OracleState{
		Validators:          make(map[uint64]*ValidatorInfo, 0),
		PoolAccumulatedFees: big.NewInt(0),
		Subscriptions:       make([]Subscription, 0),
		Unsubscriptions:     make([]Unsubscription, 0),
		Donations:           make([]Donation, 0),
		ProposedBlocks:      make([]Block, 0),
		MissedBlocks:        make([]Block, 0),
		WrongFeeBlocks:      make([]Block, 0),
	}

	// TODO: Run reconciliation here to ensure the state is correct
	// TODO: Run checks here on config. Same testnet, same fees, same addresses

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

		PoolFeesPercent:     cfg.PoolFeesPercent,
		PoolFeesAddress:     cfg.PoolFeesAddress,
		PoolAccumulatedFees: big.NewInt(0),

		Subscriptions:   make([]Subscription, 0),   // TODO: Populate
		Unsubscriptions: make([]Unsubscription, 0), // TODO: Populate
		Donations:       make([]Donation, 0),
		ProposedBlocks:  make([]Block, 0),
		MissedBlocks:    make([]Block, 0),
		WrongFeeBlocks:  make([]Block, 0),
	}
}

// Returns false if there wasnt enough data to create a merkle tree
func (state *OracleState) StoreLatestOnchainState() bool {

	log.Info("Freezing Validators state")

	// Quick way of coping the whole state
	validatorsCopy := make(map[uint64]*ValidatorInfo)
	for k2, v2 := range state.Validators {
		validatorsCopy[k2] = v2
	}

	mk := NewMerklelizer()
	// TODO: returning orderedRawLeafs as a quick workaround to get the proofs
	depositToLeaf, depositToRawLeaf, tree, enoughData := mk.GenerateTreeFromState(state)
	if !enoughData {
		return false
	}
	merkleRootStr := hex.EncodeToString(tree.Root)
	log.Info("Merkle root: ", merkleRootStr)

	// Merkle proofs for each deposit address
	proofs := make(map[string][]string)
	leafs := make(map[string]RawLeaf)
	for depositAddress, rawLeaf := range depositToRawLeaf {

		// Extra sanity check to make sure the deposit address is the same as the key
		if depositAddress != rawLeaf.DepositAddress {
			log.Fatal("Deposit address in raw leaf doesnt match the key")
		}

		block := depositToLeaf[depositAddress]
		proof, err := tree.GenerateProof(block)

		if err != nil {
			log.Fatal("could not generate proof for block: ", err)
		}

		// Store the proofs of the deposit address (to be used onchain)
		proofs[depositAddress] = ByteArrayToArray(proof.Siblings)

		// Store the leafs (to be used onchain)
		leafs[depositAddress] = rawLeaf
	}

	state.LatestCommitedState = OnchainState{
		Validators: validatorsCopy,
		//TxHash:     txHash, // TODO: Not sure if to store it
		MerkleRoot: merkleRootStr,
		Slot:       state.LatestSlot,
		Proofs:     proofs,
		Leafs:      leafs,
	}

	return true
}

func (state *OracleState) IsValidatorSubscribed(validatorIndex uint64) bool {
	for valIndex, validator := range state.Validators {
		if valIndex == validatorIndex && validator.ValidatorStatus != Banned && validator.ValidatorStatus != NotSubscribed {
			return true
		}
	}
	return false
}

func (state *OracleState) IsBanned(validatorIndex uint64) bool {
	validator, found := state.Validators[validatorIndex]
	if !found {
		return false
	}
	if validator.ValidatorStatus == Banned {
		return true
	}
	return false
}

func (state *OracleState) HandleDonations(donations []Donation) {
	// Ensure the donations are from the same block
	if len(donations) > 0 {
		blockReference := donations[0].Block
		for _, donation := range donations {
			if donation.Block != blockReference {
				log.Fatal("Handling donations from different blocks is not possible: ",
					donation.Block, " vs ", blockReference)
			}
		}
	}
	for _, donation := range donations {
		state.IncreaseAllPendingRewards(donation.AmountWei)
		state.Donations = append(state.Donations, donation)
	}
}

func (state *OracleState) HandleCorrectBlockProposal(block Block) {
	state.AddSubscriptionIfNotAlready(block.ValidatorIndex, block.DepositAddress, block.ValidatorKey)
	state.AdvanceStateMachine(block.ValidatorIndex, ProposalOk)
	state.IncreaseAllPendingRewards(block.Reward)
	state.ConsolidateBalance(block.ValidatorIndex)
	state.Validators[block.ValidatorIndex].ValidatorProposedBlocks = append(state.Validators[block.ValidatorIndex].ValidatorProposedBlocks, block)
	state.ProposedBlocks = append(state.ProposedBlocks, block)
}

func (state *OracleState) HandleManualSubscriptions(
	minCollateralWei *big.Int,
	subscriptions []Subscription) {

	for _, subscription := range subscriptions {
		valIdx := subscription.ValidatorIndex
		validator, found := state.Validators[valIdx]

		// If validator was banned, return collateral in form of accumulated and ignore
		if found && state.IsBanned(validator.ValidatorIndex) {
			log.WithFields(log.Fields{
				"BlockNumber":    subscription.BlockNumber,
				"Collateral":     subscription.Collateral,
				"TxHash":         subscription.TxHash,
				"ValidatorIndex": subscription.ValidatorIndex,
			}).Warn("Banned validator added more collateral, ignoring + returning it")

			state.Validators[subscription.ValidatorIndex].AccumulatedRewardsWei.Add(
				state.Validators[subscription.ValidatorIndex].AccumulatedRewardsWei,
				subscription.Collateral)
			return
		}

		// If we found it and its already subscribed, weird. Return the collateral
		if found && state.IsValidatorSubscribed(validator.ValidatorIndex) {
			// Okay, but weird that an already subscribed validator deposited collateral
			log.WithFields(log.Fields{
				"BlockNumber":    subscription.BlockNumber,
				"Collateral":     subscription.Collateral,
				"TxHash":         subscription.TxHash,
				"ValidatorIndex": subscription.ValidatorIndex,
			}).Warn("Validator already subscribed sent colateral again (could be targeted donation)")

			// Increase pending, adding the collateral to accumulated, in an attempt
			// to return this extra collateral to the user. It can be seen as a
			// way of donating to a given validator.
			state.Validators[subscription.ValidatorIndex].AccumulatedRewardsWei.Add(
				state.Validators[subscription.ValidatorIndex].AccumulatedRewardsWei,
				subscription.Collateral)

			// Otherwise if we havent found it or is not subscribed
		} else {
			var prevAccumulatedRewardsWei *big.Int = big.NewInt(0)
			var prevPendingRewardsWei *big.Int = big.NewInt(0)

			// Keep previous rewards in case the validator was already present but not subscribed
			// This case happens when a validator subscribes with a lower amount of collateral than
			// needed, and then subscribes again with enough collateral.
			if found {
				prevAccumulatedRewardsWei = validator.AccumulatedRewardsWei
				prevPendingRewardsWei = validator.PendingRewardsWei
			}
			// If the validator is not subscribed, we add it to the state
			// only if the collateral is enough >= minCollateralWei
			// Note that the validator starts in NotSubscribed, and its state
			// its advanced below in AdvanceStateMachine
			if subscription.Collateral.Cmp(minCollateralWei) >= 0 {
				state.Validators[subscription.ValidatorIndex] = &ValidatorInfo{
					ValidatorStatus:         NotSubscribed,
					AccumulatedRewardsWei:   prevAccumulatedRewardsWei,
					PendingRewardsWei:       prevPendingRewardsWei,
					CollateralWei:           subscription.Collateral,
					DepositAddress:          subscription.DepositAddress,
					ValidatorIndex:          subscription.ValidatorIndex,
					ValidatorKey:            subscription.ValidatorKey,
					ValidatorProposedBlocks: make([]Block, 0),
					ValidatorMissedBlocks:   make([]Block, 0),
					ValidatorWrongFeeBlocks: make([]Block, 0),
				}

				// Increase pending, adding the collateral to pending so that whenever
				// the validator proposes a block, the pending is converted into accumulated
				// and it gets back its collateral.
				// The exact collateral that was added is used, just in case by mistake someone
				// adds more than the minimum.
				state.Validators[subscription.ValidatorIndex].PendingRewardsWei.Add(
					state.Validators[subscription.ValidatorIndex].PendingRewardsWei,
					subscription.Collateral)

				log.WithFields(log.Fields{
					"BlockNumber":    subscription.BlockNumber,
					"Collateral":     subscription.Collateral,
					"TxHash":         subscription.TxHash,
					"ValidatorIndex": subscription.ValidatorIndex,
				}).Info("Validator subscribed with ok collateral")

				// And update it state according to the event
				state.AdvanceStateMachine(valIdx, ManualSubscription)
			} else {
				// If the collateral is not enough, we just track it but as unsubscribed
				// and return the collateral to the user
				log.WithFields(log.Fields{
					"BlockNumber":    subscription.BlockNumber,
					"Collateral":     subscription.Collateral,
					"TxHash":         subscription.TxHash,
					"ValidatorIndex": subscription.ValidatorIndex,
				}).Warn("Validator subscribed but collateral is not enough")

				state.Validators[subscription.ValidatorIndex] = &ValidatorInfo{
					ValidatorStatus:         NotSubscribed, // We track it but as unsuscribed
					AccumulatedRewardsWei:   big.NewInt(0),
					PendingRewardsWei:       big.NewInt(0),
					CollateralWei:           big.NewInt(0), // Set to zero since its returned
					DepositAddress:          subscription.DepositAddress,
					ValidatorIndex:          subscription.ValidatorIndex,
					ValidatorKey:            subscription.ValidatorKey,
					ValidatorProposedBlocks: make([]Block, 0),
					ValidatorMissedBlocks:   make([]Block, 0),
					ValidatorWrongFeeBlocks: make([]Block, 0),
				}

				// Return collateral adding it to accumulated rewards. Can be claimed at any time
				state.Validators[subscription.ValidatorIndex].AccumulatedRewardsWei.Add(
					state.Validators[subscription.ValidatorIndex].AccumulatedRewardsWei,
					subscription.Collateral)
			}
		}
	}
}

// Banning a validator implies sharing its pending rewards among the rest
// of the validators and setting its pending to zero.
func (state *OracleState) HandleBanValidator(block Block) {
	// First of all advance the state machine, so the banned validator is not
	// considered for the pending reward share
	state.AdvanceStateMachine(block.ValidatorIndex, ProposalWrongFee)
	state.IncreaseAllPendingRewards(state.Validators[block.ValidatorIndex].PendingRewardsWei)
	state.ResetPendingRewards(block.ValidatorIndex)

	// Store the proof of the wrong fee block. Reason why it was banned
	state.Validators[block.ValidatorIndex].ValidatorWrongFeeBlocks = append(state.Validators[block.ValidatorIndex].ValidatorWrongFeeBlocks, block)
	state.WrongFeeBlocks = append(state.WrongFeeBlocks, block)
}

func (state *OracleState) HandleMissedBlock(block Block) {
	state.AdvanceStateMachine(block.ValidatorIndex, ProposalMissed)
	state.Validators[block.ValidatorIndex].ValidatorMissedBlocks = append(state.Validators[block.ValidatorIndex].ValidatorMissedBlocks, block)
	state.MissedBlocks = append(state.MissedBlocks, block)
}

func (state *OracleState) HandleManualUnsubscriptions(
	newUnsubscriptions []Unsubscription) {

	for _, newUnsubscription := range newUnsubscriptions {
		valIdx := newUnsubscription.ValidatorIndex
		_, found := state.Validators[valIdx]

		// Check the size is the same. To avoid 0x prefixed being mixed with non 0x prefixed
		if len(newUnsubscription.DepositAddress) != len(newUnsubscription.Sender) {
			log.Fatal("Deposit address and sender are not the same length: ", newUnsubscription.DepositAddress, " ", newUnsubscription.Sender)
		}

		// Its very important to check that the unsubscription was made from the deposit address
		// of the validator, otherwise anyone could call the unsubscription function.
		if strings.ToLower(newUnsubscription.DepositAddress) != strings.ToLower(newUnsubscription.Sender) {
			log.WithFields(log.Fields{
				"BlockNumber":    newUnsubscription.BlockNumber,
				"Sender":         newUnsubscription.Sender,
				"TxHash":         newUnsubscription.TxHash,
				"ValidatorIndex": newUnsubscription.ValidatorIndex,
				"ValidatorKey":   newUnsubscription.ValidatorKey,
				"DepositAddress": newUnsubscription.DepositAddress,
			}).Warn("Unsubscription made from a different address than the deposit address")
			continue
		}

		if found {
			// If the validator is subscribed, we update it state according to the event
			state.AdvanceStateMachine(valIdx, Unsubscribe)
			state.IncreaseAllPendingRewards(state.Validators[valIdx].PendingRewardsWei)
			state.ResetPendingRewards(valIdx)
		} else {
			log.WithFields(log.Fields{
				"BlockNumber":    newUnsubscription.BlockNumber,
				"Sender":         newUnsubscription.Sender,
				"TxHash":         newUnsubscription.TxHash,
				"ValidatorIndex": newUnsubscription.ValidatorIndex,
				"ValidatorKey":   newUnsubscription.ValidatorKey,
				"DepositAddress": newUnsubscription.DepositAddress,
			}).Warn("Found and unsubscription event for a validator that is not subscribed")
		}
	}
}

// TODO: This is more related to automatic subscriptions. Rename and refactor accordingly
func (state *OracleState) AddSubscriptionIfNotAlready(valIndex uint64, depositAddress string, validatorKey string) {
	validator, found := state.Validators[valIndex]
	if !found {
		// If not found and not manually subscribed, we trigger the AutoSubscription event
		// Instantiate the validator
		validator = &ValidatorInfo{
			ValidatorStatus:         NotSubscribed,
			AccumulatedRewardsWei:   big.NewInt(0),
			PendingRewardsWei:       big.NewInt(0),
			DepositAddress:          depositAddress,
			ValidatorKey:            validatorKey,
			ValidatorProposedBlocks: make([]Block, 0),
			ValidatorMissedBlocks:   make([]Block, 0),
			ValidatorWrongFeeBlocks: make([]Block, 0),
		}
		state.Validators[valIndex] = validator

		// And update it state according to the event
		// TODO: Perhaps remove this and just use ValidatorStatus: Active
		state.AdvanceStateMachine(valIndex, AutoSubscription)
	}
}

func (state *OracleState) ConsolidateBalance(valIndex uint64) {
	state.Validators[valIndex].AccumulatedRewardsWei.Add(state.Validators[valIndex].AccumulatedRewardsWei, state.Validators[valIndex].PendingRewardsWei)
	state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

func (state *OracleState) GetEligibleValidators() []uint64 {
	eligibleValidators := make([]uint64, 0)

	for valIndex, validator := range state.Validators {
		if validator.ValidatorStatus == Active || validator.ValidatorStatus == YellowCard {
			eligibleValidators = append(eligibleValidators, valIndex)
		}
	}
	return eligibleValidators
}

// Increases the pending rewards of all validators, and gives the pool owner a cut
// of said rewards. Note that pending rewards cant be claimed until a block is proposed
// by the validator. But the pool owner can claim the pool cut at any time, so they are
// added as accumulated rewards.
func (state *OracleState) IncreaseAllPendingRewards(
	reward *big.Int) {

	eligibleValidators := state.GetEligibleValidators()
	numEligibleValidators := big.NewInt(int64(len(eligibleValidators)))

	if len(eligibleValidators) == 0 {
		log.Warn("No validators are eligible to receive rewards, pool fees address will receive all")
		state.PoolAccumulatedFees.Add(state.PoolAccumulatedFees, reward)
		return
	}

	// The pool takes PoolFeesPercent cut of the rewards
	aux := big.NewInt(0).Mul(reward, big.NewInt(int64(state.PoolFeesPercent)))

	// Calculate the pool cut
	poolCut := big.NewInt(0).Div(aux, big.NewInt(100))

	// And remainder of above operation
	remainder1 := big.NewInt(0).Mod(aux, big.NewInt(100))

	// The amount to share is the reward minus the pool cut + remainder
	toShareAllValidators := big.NewInt(0).Sub(reward, poolCut)
	toShareAllValidators.Sub(toShareAllValidators, remainder1)

	// Each validator gets that divided by numEligibleValidators
	perValidatorReward := big.NewInt(0).Div(toShareAllValidators, numEligibleValidators)
	// And remainder of above operation
	remainder2 := big.NewInt(0).Mod(toShareAllValidators, numEligibleValidators)

	// Total fees for the pool are: the cut (%) + the remainders
	totalFees := big.NewInt(0).Add(poolCut, remainder1)
	totalFees.Add(totalFees, remainder2)

	// Increase pool rewards (fees)
	state.PoolAccumulatedFees.Add(state.PoolAccumulatedFees, totalFees)

	log.WithFields(log.Fields{
		"AmountEligibleValidators": numEligibleValidators,
		"RewardPerValidatorWei":    perValidatorReward,
		"PoolFeesWei":              totalFees,
		"TotalRewardWei":           reward,
	}).Info("Increasing pending rewards of eligible validators")

	// Increase eligible validators rewards
	for _, eligibleIndex := range eligibleValidators {
		state.Validators[eligibleIndex].PendingRewardsWei.Add(state.Validators[eligibleIndex].PendingRewardsWei, perValidatorReward)
	}
}

func (state *OracleState) ResetPendingRewards(valIndex uint64) {
	state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

func (state *OracleState) LogBalances() {
	for valIndex, validator := range state.Validators {
		log.Info(
			"SlotState: ", state.LatestSlot,
			" ValIndex: ", valIndex,
			" Pending: ", validator.PendingRewardsWei,
			" Accumulated: ", validator.AccumulatedRewardsWei)
	}
}

// See the spec for state diagram with states and transitions. This tracks all the different
// states and state transitions that a given validator can have from the oracle point of view
func (state *OracleState) AdvanceStateMachine(valIndex uint64, event Event) {
	switch state.Validators[valIndex].ValidatorStatus {
	case Active:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":          "ProposalOk",
				"StateChange":    "Active -> Active",
				"ValidatorIndex": valIndex,
				"Slot/Block":     state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "Active -> Banned",
				"ValidatorIndex": valIndex,
				"Slot/Block":     state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "Active -> YellowCard",
				"ValidatorIndex": valIndex,
				"Slot/Block":     state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = YellowCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "Active -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"Slot/Block":     state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case YellowCard:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":           "ProposalOk",
				"StateChange":     "YellowCard -> Active",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":           "ProposalWrongFee",
				"StateChange":     "YellowCard -> Banned",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":           "ProposalMissed",
				"StateChange":     "YellowCard -> RedCard",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = RedCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":           "ProposalMissed",
				"StateChange":     "YellowCard -> NotSubscribed",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case RedCard:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":           "ProposalOk",
				"StateChange":     "RedCard -> YellowCard",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = YellowCard
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":           "ProposalWrongFee",
				"StateChange":     "RedCard -> Banned",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":           "ProposalMissed",
				"StateChange":     "RedCard -> RedCard",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = RedCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":           "ProposalMissed",
				"StateChange":     "RedCard -> NotSubscribed",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case NotSubscribed:
		switch event {
		case ManualSubscription:
			log.WithFields(log.Fields{
				"Event":           "ManualSubscription",
				"StateChange":     "NotSubscribed -> Active",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		case AutoSubscription:
			log.WithFields(log.Fields{
				"Event":           "AutoSubscription",
				"StateChange":     "NotSubscribed -> Active",
				"ValidatorIndex:": valIndex,
				"Slot/Block":      state.LatestSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		}
	}
}

// TODO: Add function that dumps the current state to the database
// Its a nice to have to track the validator balance evolution over the
// time

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
			len(validator.ValidatorProposedBlocks),
			len(validator.ValidatorMissedBlocks),
			len(validator.ValidatorWrongFeeBlocks),
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

// TODO: Remove this and get the merkle tree from somewhere else. See stored state
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

func RewardTypeToString(rewardType RewardType) string {
	if rewardType == VanilaBlock {
		return "vanila"
	} else if rewardType == MevBlock {
		return "mev"
	}
	log.Fatal("unknown reward type")
	return ""
}

func ValidatorStateToString(valState ValidatorStatus) string {
	if valState == Active {
		return "active"
	} else if valState == YellowCard {
		return "yellowcard"
	} else if valState == RedCard {
		return "redcard"
	} else if valState == NotSubscribed {
		return "notsubscribed"
	} else if valState == Banned {
		return "banned"
	} else if valState == Untracked {
		return "untracked"
	}
	log.Fatal("unknown validator state")
	return ""
}

func EventToString(event Event) string {
	if event == ProposalOk {
		return "proposalok"
	} else if event == ProposalMissed {
		return "proposalmissed"
	} else if event == ProposalWrongFee {
		return "proposalwrongfee"
	} else if event == ManualSubscription {
		return "manualsubscription"
	} else if event == AutoSubscription {
		return "autosubscription"
	} else if event == Unsubscribe {
		return "unsubscribe"
	}
	log.Fatal("unknown event")
	return ""
}

func BlockTypeToString(blockType BlockType) string {
	if blockType == MissedProposal {
		return "missedproposal"
	} else if blockType == WrongFeeRecipient {
		return "wrongfeerecipient"
	} else if blockType == OkPoolProposal {
		return "okpoolproposal"
	}
	log.Fatal("unknown block type")
	return ""
}

func (s RewardType) MarshalJSON() ([]byte, error) {
	return json.Marshal(RewardTypeToString(s))
}
func (s ValidatorStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(ValidatorStateToString(s))
}
func (s Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(EventToString(s))
}
func (s BlockType) MarshalJSON() ([]byte, error) {
	return json.Marshal(BlockTypeToString(s))
}
