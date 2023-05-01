package oracle

import (
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/contract"

	log "github.com/sirupsen/logrus"
	mt "github.com/txaty/go-merkletree"
)

// Description of the state machine:
// -State: States of the validators, related to wether they earn rewards or not.
// -Events: Actions that can trigger and state transition from state a to state b.
// -Handlers: Action that is performed after an event is triggered when landing a new state.

// Default filename to persist the state of the oracle
var StateFileName = "state.gob"
var StateFolder = "oracle-data"

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
	UnknownBlockType      BlockType = 0
	MissedProposal        BlockType = 1
	WrongFeeRecipient     BlockType = 2
	OkPoolProposal        BlockType = 3
	OkPoolProposalBlsKeys BlockType = 4 // TODO: this state is a bit hackish
)

// Represents a block with information relevant for the pool
type Block struct {
	Slot              uint64     `json:"slot"`
	Block             uint64     `json:"block"`
	ValidatorIndex    uint64     `json:"validator_index"`
	ValidatorKey      string     `json:"validator_key"`
	BlockType         BlockType  `json:"block_type"`
	Reward            *big.Int   `json:"reward_wei"`
	RewardType        RewardType `json:"reward_type"`
	WithdrawalAddress string     `json:"withdrawal_address"`

	/* As a nice to have would be nice to refactor to this:
	Duty     *api.ProposerDuty
	Block    *spec.VersionedSignedBeaconBlock
	Header   *types.Header
	Receipts []*types.Receipt*/
}

// Represents a donation made to the pool
// TODO: deprecate this? donations are detected from the block content
type Donation struct {
	AmountWei *big.Int `json:"amount_wei"`
	Block     uint64   `json:"block_number"`
	TxHash    string   `json:"tx_hash"`
}

// Subscription event and the associated validator (if any)
type Subscription struct {
	Event     *contract.ContractSubscribeValidator
	Validator *v1.Validator
}

// Unsubscription event and the associated validator (if any)
type Unsubscription struct {
	Event     *contract.ContractUnsubscribeValidator
	Validator *v1.Validator
}

// Represents all the information that is stored of a validator
type ValidatorInfo struct {
	ValidatorStatus         ValidatorStatus `json:"status"`
	AccumulatedRewardsWei   *big.Int        `json:"accumulated_rewards_wei"`
	PendingRewardsWei       *big.Int        `json:"pending_rewards_wei"`
	CollateralWei           *big.Int        `json:"collateral_wei"`
	WithdrawalAddress       string          `json:"withdrawal_address"` // TODO: Rename to: withdrawal_address (keeping it for backwards compatibility by now)
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
	LatestProcessedSlot  uint64
	LatestProcessedBlock uint64
	NextSlotToProcess    uint64
	Network              string
	PoolAddress          string
	Validators           map[uint64]*ValidatorInfo
	LatestCommitedState  OnchainState

	PoolFeesPercent     int
	PoolFeesAddress     string
	PoolAccumulatedFees *big.Int

	Subscriptions   []Subscription
	Unsubscriptions []Unsubscription
	Donations       []Donation
	ProposedBlocks  []Block
	MissedBlocks    []Block
	WrongFeeBlocks  []Block

	Config *config.Config
}

func NewOracleState(cfg *config.Config) *OracleState {
	return &OracleState{
		LatestProcessedSlot:  cfg.DeployedSlot - 1,
		LatestProcessedBlock: 0,
		NextSlotToProcess:    cfg.DeployedSlot,
		Network:              cfg.Network,
		PoolAddress:          cfg.PoolAddress,

		Validators: make(map[uint64]*ValidatorInfo, 0),

		PoolFeesPercent:     cfg.PoolFeesPercent,
		PoolFeesAddress:     cfg.PoolFeesAddress,
		PoolAccumulatedFees: big.NewInt(0),

		Subscriptions:   make([]Subscription, 0),
		Unsubscriptions: make([]Unsubscription, 0),
		Donations:       make([]Donation, 0),
		ProposedBlocks:  make([]Block, 0),
		MissedBlocks:    make([]Block, 0),
		WrongFeeBlocks:  make([]Block, 0),
		Config:          cfg,
	}
}

func (state *OracleState) SaveStateToFile() {
	path := filepath.Join(StateFolder, StateFileName)
	err := os.MkdirAll(StateFolder, os.ModePerm)
	if err != nil {
		log.Fatal("could not create folder: ", err)
	}
	file, err := os.Create(path)
	if err != nil {
		log.Fatal("could not create file at path: ", path, ":", err)
	}

	defer file.Close()

	// Dont run this again, take the existing data
	//mRoot, enoughData := p.GetMerkleRootIfAny()

	encoder := gob.NewEncoder(file)
	log.WithFields(log.Fields{
		"LatestProcessedSlot":  state.LatestProcessedSlot,
		"LatestProcessedBlock": state.LatestProcessedBlock,
		"NextSlotToProcess":    state.NextSlotToProcess,
		"TotalValidators":      len(state.Validators),
		"Network":              state.Network,
		"PoolAddress":          state.PoolAddress,
		"Path":                 path,
		//"MerkleRoot":      mRoot,
		//"EnoughData":      enoughData,
	}).Info("Saving state to file")

	err = encoder.Encode(state)
	if err != nil {
		log.Fatal("could not encode state into file: ", err)
	}
}

func (state *OracleState) LoadStateFromFile() error {
	// Init all fields in case any was stored empty in the file
	readState := OracleState{
		Validators:          make(map[uint64]*ValidatorInfo, 0),
		PoolAccumulatedFees: big.NewInt(0),
		Subscriptions:       make([]Subscription, 0),
		Unsubscriptions:     make([]Unsubscription, 0),
		Donations:           make([]Donation, 0),
		ProposedBlocks:      make([]Block, 0),
		MissedBlocks:        make([]Block, 0),
		WrongFeeBlocks:      make([]Block, 0),
		Config:              &config.Config{},
	}

	// TODO: Run reconciliation here to ensure the state is correct
	// TODO: Run checks here on config. Same testnet, same fees, same addresses
	path := filepath.Join(StateFolder, StateFileName)
	file, err := os.Open(path)

	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&readState)
	if err != nil {
		return err
	}

	if readState.Config.Network != state.Config.Network {
		log.Fatal("Network mismatch. Expected: ", state.Config.Network, " Got: ", readState.Config.Network)
	}

	if readState.Config.PoolAddress != state.Config.PoolAddress {
		log.Fatal("PoolAddress mismatch. Expected: ", state.Config.PoolAddress, " Got: ", readState.Config.PoolAddress)
	}

	if readState.Config.PoolFeesAddress != state.Config.PoolFeesAddress {
		log.Fatal("PoolFeesAddress mismatch. Expected: ", state.Config.PoolFeesAddress, " Got: ", readState.Config.PoolFeesAddress)
	}

	if readState.Config.PoolFeesPercent != state.Config.PoolFeesPercent {
		log.Fatal("PoolFeesPercent mismatch. Expected: ", state.Config.PoolFeesPercent, " Got: ", readState.Config.PoolFeesPercent)
	}

	mRoot, enoughData := readState.GetMerkleRootIfAny()

	log.WithFields(log.Fields{
		"Path":                 path,
		"LatestProcessedSlot":  readState.LatestProcessedSlot,
		"LatestProcessedBlock": readState.LatestProcessedBlock,
		"NextSlotToProcess":    readState.NextSlotToProcess,
		"Network":              readState.Network,
		"PoolAddress":          readState.PoolAddress,
		"MerkleRoot":           mRoot,
		"EnoughData":           enoughData,
	}).Info("Loaded state from file")

	// This could be nicer. Note that adding a new field to the state
	// requires adding it here as well
	state.LatestProcessedSlot = readState.LatestProcessedSlot
	state.NextSlotToProcess = readState.NextSlotToProcess
	state.LatestProcessedBlock = readState.LatestProcessedBlock
	//state.Network = readState.Network
	//state.PoolAddress = readState.PoolAddress
	state.Validators = readState.Validators
	state.LatestCommitedState = readState.LatestCommitedState
	state.PoolFeesPercent = readState.PoolFeesPercent
	state.PoolFeesAddress = readState.PoolFeesAddress
	state.PoolAccumulatedFees = readState.PoolAccumulatedFees
	state.Subscriptions = readState.Subscriptions
	state.Unsubscriptions = readState.Unsubscriptions
	state.Donations = readState.Donations
	state.ProposedBlocks = readState.ProposedBlocks
	state.MissedBlocks = readState.MissedBlocks
	state.WrongFeeBlocks = readState.WrongFeeBlocks

	return nil
}

// Returns false if there wasnt enough data to create a merkle tree
func (state *OracleState) StoreLatestOnchainState() bool {

	// Quick way of coping the whole state
	validatorsCopy := make(map[uint64]*ValidatorInfo)
	for k2, v2 := range state.Validators {
		validatorsCopy[k2] = v2
	}

	mk := NewMerklelizer()
	// TODO: returning orderedRawLeafs as a quick workaround to get the proofs
	withdrawalToLeaf, withdrawalToRawLeaf, tree, enoughData := mk.GenerateTreeFromState(state)
	if !enoughData {
		return false
	}
	merkleRootStr := hex.EncodeToString(tree.Root)

	log.WithFields(log.Fields{
		"LatestProcessedSlot": state.LatestProcessedSlot,
		"MerkleRoot":          merkleRootStr,
	}).Info("Freezing state")

	// Merkle proofs for each withdrawal address
	proofs := make(map[string][]string)
	leafs := make(map[string]RawLeaf)
	for WithdrawalAddress, rawLeaf := range withdrawalToRawLeaf {

		// Extra sanity check to make sure the withdrawal address is the same as the key
		if WithdrawalAddress != rawLeaf.WithdrawalAddress {
			log.Fatal("withdrawal address in raw leaf doesnt match the key")
		}

		block := withdrawalToLeaf[WithdrawalAddress]
		proof, err := tree.GenerateProof(block)

		if err != nil {
			log.Fatal("could not generate proof for block: ", err)
		}

		// Store the proofs of the withdrawal address (to be used onchain)
		proofs[WithdrawalAddress] = ByteArrayToArray(proof.Siblings)

		// Store the leafs (to be used onchain)
		leafs[WithdrawalAddress] = rawLeaf
	}

	state.LatestCommitedState = OnchainState{
		Validators: validatorsCopy,
		//TxHash:     txHash, // TODO: Not sure if to store it
		MerkleRoot: merkleRootStr,
		Slot:       state.LatestProcessedSlot,
		Proofs:     proofs,
		Leafs:      leafs,
	}

	return true
}

func (state *OracleState) IsSubscribed(validatorIndex uint64) bool {
	for valIndex, validator := range state.Validators {
		if valIndex == validatorIndex &&
			validator.ValidatorStatus != Banned &&
			validator.ValidatorStatus != NotSubscribed &&
			validator.ValidatorStatus != UnknownState {
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

func (state *OracleState) IsTracked(validatorIndex uint64) bool {
	_, found := state.Validators[validatorIndex]
	if found {
		return true
	}
	return false
}

func (state *OracleState) IsCollateralEnough(collateral *big.Int) bool {
	return collateral.Cmp(state.Config.CollateralInWei) >= 0
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
	state.AddSubscriptionIfNotAlready(block.ValidatorIndex, block.WithdrawalAddress, block.ValidatorKey)
	state.AdvanceStateMachine(block.ValidatorIndex, ProposalOk)
	state.IncreaseAllPendingRewards(block.Reward)
	state.ConsolidateBalance(block.ValidatorIndex)
	state.Validators[block.ValidatorIndex].ValidatorProposedBlocks = append(state.Validators[block.ValidatorIndex].ValidatorProposedBlocks, block)
	state.ProposedBlocks = append(state.ProposedBlocks, block)
}

func (state *OracleState) HandleManualSubscriptions(
	subscriptions []Subscription) {

	for _, sub := range subscriptions {

		valIdx := uint64(sub.Event.ValidatorID) // TODO: Contract should be uint64
		collateral := sub.Event.SubscriptionCollateral
		sender := sub.Event.Sender.String()

		// Subscription recevied for a validator index that doesnt exist
		if sub.Validator == nil {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for non existing validator, skipping")
			// Fees go to the pool, as we dont know who is the sender
			state.SendRewardToPool(collateral)
			continue
		}

		if valIdx != uint64(sub.Validator.Index) {
			log.Fatal("Subscription event validator index doesnt match the validator index. ",
				valIdx, " vs ", sub.Validator.Index)
		}

		// Subscription received for a validator that cannot subscribe (see states)
		if !CanValidatorSubscribeToPool(sub.Validator) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
				"ValidatorState": sub.Validator.Status,
			}).Warn("[Subscription]: for validator that cannot subscribe due to its state, skipping")
			// Fees go to the pool, as we dont know who is the sender
			state.SendRewardToPool(collateral)
			continue
		}

		// Subscription received for a validator that dont have eth1 withdrawal address (bls)
		validatorWithdrawal, err := GetEth1AddressByte(sub.Validator.Validator.WithdrawalCredentials)
		if err != nil {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"WithdrawalAddr": "0x" + hex.EncodeToString(sub.Validator.Validator.WithdrawalCredentials[:]),
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for validator with invalid withdrawal address (bls), skipping")
			// Fees go to the pool, as we dont know who is the sender
			state.SendRewardToPool(collateral)
			continue
		}

		// Subscription received from an address that is not the validator withdrawal address
		if !AreAddressEqual(sender, validatorWithdrawal) {
			log.WithFields(log.Fields{
				"BlockNumber":         sub.Event.Raw.BlockNumber,
				"Collateral":          sub.Event.SubscriptionCollateral,
				"TxHash":              sub.Event.Raw.TxHash,
				"ValidatorIndex":      valIdx,
				"Sender":              sender,
				"ValidatorWithdrawal": validatorWithdrawal,
			}).Warn("[Subscription]: but tx sender is not the validator withdrawal address, skipping")
			// Fees go to the pool, as we dont know who is the sender
			state.SendRewardToPool(collateral)
			continue
		}

		// Subscription received for a banned validator
		if state.IsBanned(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for banned validator, skipping")
			// Since we track this validator, give the collateral back
			state.IncreaseValidatorAccumulatedRewards(valIdx, collateral)
			continue
		}

		// Subscription received for an already subscribed validator
		if state.IsSubscribed(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for an already subscribed validator, skipping")
			// Return the collateral as accumulated balance
			state.IncreaseValidatorAccumulatedRewards(valIdx, collateral)
			continue
		}

		// Subscription received for a validator with not enough collateral
		if !state.IsCollateralEnough(collateral) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for a validator with not enough collateral, skipping")
			// Wrong, collateral goes to pool
			state.SendRewardToPool(collateral)
			continue
		}

		// Valid subscription
		if !state.IsSubscribed(valIdx) &&
			!state.IsBanned(valIdx) &&
			state.IsCollateralEnough(collateral) {

			// Add valid subscription
			if !state.IsTracked(valIdx) {
				// If its not tracked, we create a new subscription
				state.Validators[valIdx] = &ValidatorInfo{
					ValidatorStatus:         NotSubscribed,
					AccumulatedRewardsWei:   big.NewInt(0),
					PendingRewardsWei:       big.NewInt(0),
					CollateralWei:           collateral,
					WithdrawalAddress:       validatorWithdrawal, // TODO: Rename withdrawal Address
					ValidatorIndex:          valIdx,
					ValidatorKey:            "0x" + hex.EncodeToString(sub.Validator.Validator.PublicKey[:]),
					ValidatorProposedBlocks: make([]Block, 0),
					ValidatorMissedBlocks:   make([]Block, 0),
					ValidatorWrongFeeBlocks: make([]Block, 0),
				}
			}
			log.WithFields(log.Fields{
				"BlockNumber":      sub.Event.Raw.BlockNumber,
				"Collateral":       sub.Event.SubscriptionCollateral,
				"TxHash":           sub.Event.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": validatorWithdrawal,
			}).Info("[Subscription]: Validator subscribed ok")
			state.IncreaseValidatorPendingRewards(valIdx, collateral)
			state.AdvanceStateMachine(valIdx, ManualSubscription)
			state.Subscriptions = append(state.Subscriptions, sub)
			continue
		}

		// If we reach this point, its a case we havent considered, but its not valid
		log.WithFields(log.Fields{
			"BlockNumber":      sub.Event.Raw.BlockNumber,
			"Collateral":       sub.Event.SubscriptionCollateral,
			"TxHash":           sub.Event.Raw.TxHash,
			"ValidatorIndex":   valIdx,
			"WithdrawaAddress": validatorWithdrawal,
		}).Info("[Subscription]: Not considered case meaning wrong subscription, skipping")
		state.SendRewardToPool(collateral)
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

	for _, unsub := range newUnsubscriptions {

		valIdx := uint64(unsub.Event.ValidatorID) // TODO: should be uint64 in the contract
		sender := unsub.Event.Sender.String()

		// Unsubscription but for a validator that doesnt exist
		if unsub.Validator == nil {
			log.WithFields(log.Fields{
				"BlockNumber":    unsub.Event.Raw.BlockNumber,
				"TxHash":         unsub.Event.Raw.TxHash,
				"Sender":         sender,
				"ValidatorIndex": valIdx,
			}).Warn("[Unsubscription]: for validator index that does not exist, skipping")
			continue
		}

		if valIdx != uint64(unsub.Validator.Index) {
			log.Fatal("Unsubscription event validator index doesnt match the validator index. ",
				valIdx, " vs ", unsub.Validator.Index)
		}

		// Unsubscription but for a validator that does not have an eth1 address. Should never happen
		withdrawalAddress, err := GetEth1AddressByte(unsub.Validator.Validator.WithdrawalCredentials)
		if err != nil {
			log.WithFields(log.Fields{
				"BlockNumber":    unsub.Event.Raw.BlockNumber,
				"TxHash":         unsub.Event.Raw.TxHash,
				"Sender":         sender,
				"ValidatorIndex": valIdx,
			}).Warn("[Unsubscription]: for validator with no eth1 withdrawal addres (bls), skipping")
			continue
		}

		// Its very important to check that the unsubscription was made from the withdrawal address
		// of the validator, otherwise anyone could call the unsubscription function.
		if !AreAddressEqual(sender, withdrawalAddress) {
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Event.Raw.BlockNumber,
				"TxHash":           unsub.Event.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Warn("[Unsubscription] but sender does not match withdrawal address, skipping")
			continue
		}

		// After all the checks, we can proceed with the unsubscription
		if state.IsSubscribed(valIdx) {
			state.AdvanceStateMachine(valIdx, Unsubscribe)
			state.IncreaseAllPendingRewards(state.Validators[valIdx].PendingRewardsWei)
			state.ResetPendingRewards(valIdx)
			state.Unsubscriptions = append(state.Unsubscriptions, unsub)
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Event.Raw.BlockNumber,
				"TxHash":           unsub.Event.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Info("[Unsubscription] Validator unsubscribed ok")
			continue
		}

		if !state.IsSubscribed(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Event.Raw.BlockNumber,
				"TxHash":           unsub.Event.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Warn("[Unsubscription] but the validator is not subscribed, skipping")
			continue
		}

		// If we reach this point, its a case we havent considered, but its not valid
		log.WithFields(log.Fields{
			"BlockNumber":      unsub.Event.Raw.BlockNumber,
			"TxHash":           unsub.Event.Raw.TxHash,
			"ValidatorIndex":   valIdx,
			"WithdrawaAddress": withdrawalAddress,
			"Sender":           sender,
		}).Warn("[Unsubscription] Not considered case meaning wrong unsubscription, skipping")
	}
}

// TODO: This is more related to automatic subscriptions. Rename and refactor accordingly
func (state *OracleState) AddSubscriptionIfNotAlready(valIndex uint64, WithdrawalAddress string, validatorKey string) {
	validator, found := state.Validators[valIndex]
	if !found {
		// If not found and not manually subscribed, we trigger the AutoSubscription event
		// Instantiate the validator
		validator = &ValidatorInfo{
			ValidatorStatus:         NotSubscribed,
			AccumulatedRewardsWei:   big.NewInt(0),
			PendingRewardsWei:       big.NewInt(0),
			WithdrawalAddress:       WithdrawalAddress,
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

func (state *OracleState) IncreaseValidatorPendingRewards(valIndex uint64, reward *big.Int) {
	state.Validators[valIndex].PendingRewardsWei.Add(state.Validators[valIndex].PendingRewardsWei, reward)
}

func (state *OracleState) IncreaseValidatorAccumulatedRewards(valIndex uint64, reward *big.Int) {
	state.Validators[valIndex].AccumulatedRewardsWei.Add(state.Validators[valIndex].AccumulatedRewardsWei, reward)
}

func (state *OracleState) SendRewardToPool(reward *big.Int) {
	state.PoolAccumulatedFees.Add(state.PoolAccumulatedFees, reward)
}

func (state *OracleState) ResetPendingRewards(valIndex uint64) {
	state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
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
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "Active -> Banned",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "Active -> YellowCard",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = YellowCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "Unsubscribe",
				"StateChange":    "Active -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case YellowCard:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":          "ProposalOk",
				"StateChange":    "YellowCard -> Active",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "YellowCard -> Banned",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "YellowCard -> RedCard",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = RedCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "Unsubscribe",
				"StateChange":    "YellowCard -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case RedCard:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":          "ProposalOk",
				"StateChange":    "RedCard -> YellowCard",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = YellowCard
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "RedCard -> Banned",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "RedCard -> RedCard",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = RedCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "Unsubscribe",
				"StateChange":    "RedCard -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case NotSubscribed:
		switch event {
		case ManualSubscription:
			log.WithFields(log.Fields{
				"Event":          "ManualSubscription",
				"StateChange":    "NotSubscribed -> Active",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		case AutoSubscription:
			log.WithFields(log.Fields{
				"Event":          "AutoSubscription",
				"StateChange":    "NotSubscribed -> Active",
				"ValidatorIndex": valIndex,
				"ProcessedSlot":  state.LatestProcessedSlot,
			}).Info("Validator state change")
			state.Validators[valIndex].ValidatorStatus = Active
		}
	}
}

func CanValidatorSubscribeToPool(validator *v1.Validator) bool {
	if validator.Status != v1.ValidatorStateActiveExiting &&
		validator.Status != v1.ValidatorStateActiveSlashed &&
		validator.Status != v1.ValidatorStateExitedUnslashed &&
		validator.Status != v1.ValidatorStateExitedSlashed &&
		validator.Status != v1.ValidatorStateWithdrawalPossible &&
		validator.Status != v1.ValidatorStateWithdrawalDone &&
		validator.Status != v1.ValidatorStateUnknown {
		return true
	}
	// Accepted states are:
	// -ValidatorStatePendingInitialized
	// -ValidatorStatePendingQueued
	// -ValidatorStateActiveOngoing
	return false
}

func (state *OracleState) LogBalances() {
	for valIndex, validator := range state.Validators {
		log.WithFields(log.Fields{
			"LatestProcessedSlot": state.LatestProcessedSlot,
			"ValIndex":            valIndex,
			"PendingRewards":      validator.PendingRewardsWei,
			"AccumulatedRewards":  validator.AccumulatedRewardsWei,
		}).Info("Validator balances")
	}
}

// TODO: Remove this and get the merkle tree from somewhere else. See stored state
func (state *OracleState) GetMerkleRootIfAny() (string, bool) {
	mk := NewMerklelizer()
	// TODO: returning orderedRawLeafs as a quick workaround to get the proofs
	_, _, tree, enoughData := mk.GenerateTreeFromState(state)
	if !enoughData {
		return "", enoughData
	}
	merkleRootStr := hex.EncodeToString(tree.Root)

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
