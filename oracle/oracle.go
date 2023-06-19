package oracle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/sirupsen/logrus"
)

// Default path of persisted state
var StateFolder = "oracle-data"
var StateJsonName = "state.json"

type Oracle struct {
	cfg              *Config
	state            *OracleState
	beaconValidators map[phase0.ValidatorIndex]*v1.Validator
	mutex            sync.RWMutex
}

func NewOracle(cfg *Config) *Oracle {
	state := &OracleState{
		StateHash:            "",
		LatestProcessedBlock: 0,
		LatestProcessedSlot:  cfg.DeployedSlot - 1,
		NextSlotToProcess:    cfg.DeployedSlot,
		PoolAccumulatedFees:  big.NewInt(0),
		Validators:           make(map[uint64]*ValidatorInfo, 0),
		CommitedStates:       make(map[uint64]*OnchainState, 0),
		Subscriptions:        make([]*contract.ContractSubscribeValidator, 0),
		Unsubscriptions:      make([]*contract.ContractUnsubscribeValidator, 0),
		Donations:            make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:       make([]SummarizedBlock, 0),
		MissedBlocks:         make([]SummarizedBlock, 0),
		WrongFeeBlocks:       make([]SummarizedBlock, 0),

		// Config
		PoolFeesPercentOver10000: cfg.PoolFeesPercentOver10000,
		PoolAddress:              cfg.PoolAddress,
		Network:                  cfg.Network,
		PoolFeesAddress:          cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    cfg.CheckPointSizeInSlots,
		DeployedBlock:            cfg.DeployedBlock,
		DeployedSlot:             cfg.DeployedSlot,
		CollateralInWei:          cfg.CollateralInWei,
	}

	oracle := &Oracle{
		cfg:   cfg,
		state: state,
	}

	return oracle
}

// Returns the state of the oracle, containing all the information about the
// validatores, with their state, balances, etc
func (or *Oracle) State() *OracleState {
	or.mutex.RLock()
	defer or.mutex.RUnlock()
	return or.state
}

// Sets the known validators from the beacon chain, must be updated regularly
func (or *Oracle) SetBeaconValidators(
	validators map[phase0.ValidatorIndex]*v1.Validator) {
	or.beaconValidators = validators
}

// Given a previous or.state, this function applies the new block to it, updating the or.state
// with the new subscriptions, unsubscriptions, donations, and rewards to the pool, updating
// the balance of all participating validators. Returns the slot that was processed and if there
// was an error.
func (or *Oracle) AdvanceStateToNextSlot(fullBlock *FullBlock) (uint64, error) {

	or.mutex.Lock()
	defer or.mutex.Unlock()

	// Ensure the slot to process is the last +1
	if or.state.NextSlotToProcess != (or.state.LatestProcessedSlot + 1) {
		return 0, errors.New(fmt.Sprint("Next slot to process is not the last processed slot + 1",
			or.state.NextSlotToProcess, " ", or.state.LatestProcessedSlot))
	}

	// Ensure the block to process matches the expected one
	if or.state.NextSlotToProcess != fullBlock.GetSlotUint64() {
		return 0, errors.New(fmt.Sprint("Next slot to process is not the same as the block slot",
			or.state.NextSlotToProcess, " ", fullBlock.GetSlotUint64()))
	}

	// Some misc validations
	err := or.validateFullBlockConfig(fullBlock, or.cfg)
	if err != nil {
		return 0, errors.Wrap(err, "Error validating full block config")
	}

	// Full block is too heavy to be stored in the state, so we summarize it
	summarizedBlock := fullBlock.SummarizedBlock(or, or.cfg.PoolAddress)

	// Get donations to the pool in this block
	blockDonations := fullBlock.GetDonations(or.cfg.PoolAddress)

	// Handle subscriptions first thing
	or.handleManualSubscriptions(fullBlock.Events.SubscribeValidator)

	// If the validator was subscribed and missed proposed the block in this slot
	if summarizedBlock.BlockType == MissedProposal && or.isSubscribed(summarizedBlock.ValidatorIndex) {
		or.handleMissedBlock(summarizedBlock)
	}

	// If we have a successful block proposal BUT the validator has BLS keys, we cant auto subscribe it
	if summarizedBlock.BlockType == OkPoolProposalBlsKeys {
		or.handleBlsCorrectBlockProposal(summarizedBlock)
	}

	// If fee recipient matches the pool, we distribute the rewards and upate
	// the validator state. Automatic subscriptions are considered here
	if summarizedBlock.BlockType == OkPoolProposal {
		or.handleCorrectBlockProposal(summarizedBlock)
	}

	// If the validator was subscribed but the fee recipient was wrong we ban the validator
	if summarizedBlock.BlockType == WrongFeeRecipient && or.isSubscribed(summarizedBlock.ValidatorIndex) {
		or.handleBanValidator(summarizedBlock)
	}

	// Handle unsubscriptions the last thing after distributing rewards
	or.handleManualUnsubscriptions(fullBlock.Events.UnsubscribeValidator)

	// Handle the donations from this block
	or.handleDonations(blockDonations)

	processedSlot := or.state.NextSlotToProcess
	or.state.LatestProcessedSlot = processedSlot
	or.state.NextSlotToProcess++
	if summarizedBlock.BlockType != MissedProposal {
		or.state.LatestProcessedBlock = summarizedBlock.Block
	}
	return processedSlot, nil
}

// We use the following events to validate that the config has not changed. Dynamic
// parameters are not supported.
// UpdatePoolFee: Indicates the cut in %*100 the pool gets
// PoolFeeRecipient: Indicates the address that receives the pool fees
// CheckpointSlotSize: Indicates the size of the checkpoint in slots
// UpdateSubscriptionCollateral: Indicates the amount of ETH required to subscribe
func (or *Oracle) validateFullBlockConfig(fullBlock *FullBlock, config *Config) error {
	if len(fullBlock.Events.UpdatePoolFee) > 1 ||
		len(fullBlock.Events.PoolFeeRecipient) > 1 ||
		len(fullBlock.Events.CheckpointSlotSize) > 1 ||
		len(fullBlock.Events.UpdateSubscriptionCollateral) > 1 {
		return errors.New("more than one event of the same type in the same block, weird")
	}

	if len(fullBlock.Events.UpdatePoolFee) != 0 && big.NewInt(int64(config.PoolFeesPercentOver10000)).Cmp(fullBlock.Events.UpdatePoolFee[0].NewPoolFee) != 0 {
		return errors.New(fmt.Sprintf("pool fee has changed. config: %d, block: %d",
			config.PoolFeesPercentOver10000, fullBlock.Events.UpdatePoolFee[0].NewPoolFee))
	}

	if len(fullBlock.Events.PoolFeeRecipient) != 0 && !utils.Equals(config.PoolFeesAddress, fullBlock.Events.PoolFeeRecipient[0].NewPoolFeeRecipient.String()) {
		return errors.New(fmt.Sprintf("pool fee recipient has changed. config: %s, block: %s",
			config.PoolFeesAddress, fullBlock.Events.PoolFeeRecipient[0].NewPoolFeeRecipient.String()))
	}

	if len(fullBlock.Events.CheckpointSlotSize) != 0 && config.CheckPointSizeInSlots != fullBlock.Events.CheckpointSlotSize[0].NewCheckpointSlotSize {
		return errors.New(fmt.Sprintf("checkpoint size has changed. config: %d, block: %d",
			config.CheckPointSizeInSlots, fullBlock.Events.CheckpointSlotSize[0].NewCheckpointSlotSize))
	}

	if len(fullBlock.Events.UpdateSubscriptionCollateral) != 0 && config.CollateralInWei.Cmp(fullBlock.Events.UpdateSubscriptionCollateral[0].NewSubscriptionCollateral) != 0 {
		return errors.New(fmt.Sprintf("subscription collateral has changed. config: %d, block: %d",
			config.CollateralInWei, fullBlock.Events.UpdateSubscriptionCollateral[0].NewSubscriptionCollateral))
	}

	return nil
}

// Serialized and saves the oracle state to a human readable json file
func (or *Oracle) SaveToJson() error {
	// Not just read lock since we change the hash, minor thing
	// but it cant be just a read mutex
	or.mutex.Lock()
	defer or.mutex.Unlock()

	log.Info("Saving oracle state to json file")

	err := or.hashStateLockFree()
	if err != nil {
		return errors.Wrap(err, "error hashing the oracle state")
	}

	// Marshal again with the hash
	jsonData, err := json.MarshalIndent(or.state, "", " ")
	if err != nil {
		return errors.Wrap(err, "could not marshal state to json")
	}

	path := filepath.Join(StateFolder, StateJsonName)
	err = os.MkdirAll(StateFolder, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "could not create folder")
	}

	log.Debug("Saving state from file: ", fmt.Sprintf("%s", jsonData))

	err = ioutil.WriteFile(path, jsonData, 0644)
	if err != nil {
		return errors.Wrap(err, "could not write file")
	}

	log.WithFields(log.Fields{
		"LatestProcessedSlot":  or.state.LatestProcessedSlot,
		"LatestProcessedBlock": or.state.LatestProcessedBlock,
		"NextSlotToProcess":    or.state.NextSlotToProcess,
		"TotalValidators":      len(or.state.Validators),
		"Network":              or.state.Network,
		"PoolAddress":          or.state.PoolAddress,
		"Path":                 path,
		"Hash":                 or.state.StateHash,
	}).Info("Saved state to file")

	return nil
}

// Loads the oracle state from a human readable json file. Multiple
// check are performed to ensure the state is valid such as checking
// the hash of the state and ensuring the configuation has not changed
func (or *Oracle) LoadFromJson() (bool, error) {
	or.mutex.Lock()
	defer or.mutex.Unlock()

	path := filepath.Join(StateFolder, StateJsonName)
	log.Info("Loading oracle state from json file: ", path)

	jsonFile, err := os.Open(path)
	defer jsonFile.Close()

	// Dont error if the file wasnt found, just return not found
	if err != nil {
		return false, nil
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return false, errors.Wrap(err, "could not read json file")
	}

	var state OracleState

	err = json.Unmarshal(byteValue, &state)
	if err != nil {
		return false, errors.Wrap(err, "could not unmarshal json file")
	}

	// Store the hash we recovered from the file
	recoveredHash := state.StateHash

	// Reset the hash since we want to hash the content without the hash
	state.StateHash = ""

	// Serialize the state without the hash
	jsonNoHash, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return false, errors.Wrap(err, "could not marshal state without hash")
	}

	log.Debug("Loaded state from file: ", fmt.Sprintf("%s", jsonNoHash))

	// We calculate the hash of the state we read
	calculatedHashByte := sha256.Sum256(jsonNoHash[:])
	calculatedHashStrig := hexutil.Encode(calculatedHashByte[:])

	// Hashes must match
	if !utils.Equals(recoveredHash, calculatedHashStrig) {
		return false, errors.New(fmt.Sprintf("hash mismatch, recovered: %s, calculated: %s",
			recoveredHash, calculatedHashStrig))
	}

	// Sanity check to ensure the oracle config matches the loaded state
	if state.Network != or.cfg.Network {
		return false, errors.New(fmt.Sprintf("network mismatch, recovered: %s, expected: %s",
			state.Network, or.cfg.Network))
	}

	if state.PoolAddress != or.cfg.PoolAddress {
		return false, errors.New(fmt.Sprintf("pool address mismatch, recovered: %s, expected: %s",
			state.PoolAddress, or.cfg.PoolAddress))
	}

	if state.PoolFeesAddress != or.cfg.PoolFeesAddress {
		return false, errors.New(fmt.Sprintf("pool fees address mismatch, recovered: %s, expected: %s",
			state.PoolFeesAddress, or.cfg.PoolFeesAddress))
	}

	if state.PoolFeesPercentOver10000 != or.cfg.PoolFeesPercentOver10000 {
		return false, errors.New(fmt.Sprintf("pool fees percent mismatch, recovered: %d, expected: %d",
			state.PoolFeesPercentOver10000, or.cfg.PoolFeesPercentOver10000))
	}

	if state.CollateralInWei.Cmp(or.cfg.CollateralInWei) != 0 {
		return false, errors.New(fmt.Sprintf("collateral mismatch, recovered: %d, expected: %d",
			state.CollateralInWei, or.cfg.CollateralInWei))
	}

	if state.CheckPointSizeInSlots != or.cfg.CheckPointSizeInSlots {
		return false, errors.New(fmt.Sprintf("check point size mismatch, recovered: %d, expected: %d",
			state.CheckPointSizeInSlots, or.cfg.CheckPointSizeInSlots))
	}

	if state.DeployedBlock != or.cfg.DeployedBlock {
		return false, errors.New(fmt.Sprintf("deployed block mismatch, recovered: %d, expected: %d",
			state.DeployedBlock, or.cfg.DeployedBlock))
	}

	if state.DeployedSlot != or.cfg.DeployedSlot {
		return false, errors.New(fmt.Sprintf("deployed slot mismatch, recovered: %d, expected: %d",
			state.DeployedSlot, or.cfg.DeployedSlot))
	}

	mRoot, enoughData := or.getMerkleRootIfAny()
	log.WithFields(log.Fields{
		"Path":                 path,
		"LatestProcessedSlot":  state.LatestProcessedSlot,
		"LatestProcessedBlock": state.LatestProcessedBlock,
		"NextSlotToProcess":    state.NextSlotToProcess,
		"Network":              state.Network,
		"PoolAddress":          state.PoolAddress,
		"MerkleRoot":           mRoot,
		"EnoughData":           enoughData,
	}).Info("Loaded state from file")

	or.state = &state
	return true, nil
}

// Takes the current state, creates a copy of it and freezes it, storing
// it in a map slot->state. It also creates a set of merkle proof for each
// withdrawal address of each validator. Each of these frozen states maps
// to a commited onchain state, represented by a merkle root.
// Returns false if there wasnt enough data to create a merkle tree
func (or *Oracle) FreezeCheckpoint() bool {
	or.mutex.Lock()
	defer or.mutex.Unlock()

	validatorsCopy := make(map[uint64]*ValidatorInfo)
	utils.DeepCopy(or.state.Validators, &validatorsCopy)

	mk := NewMerklelizer()
	withdrawalToLeaf, withdrawalToRawLeaf, tree, enoughData := mk.GenerateTreeFromState(or.state)
	if !enoughData {
		return false
	}
	merkleRootStr := hexutil.Encode(tree.Root[:])

	log.WithFields(log.Fields{
		"Slot":       or.state.LatestProcessedSlot,
		"MerkleRoot": merkleRootStr,
	}).Info("Freezing state")

	// Merkle proofs for each withdrawal address
	proofs := make(map[string][]string)
	leafs := make(map[string]RawLeaf)
	for WithdrawalAddress, rawLeaf := range withdrawalToRawLeaf {

		// Extra sanity check to make sure the withdrawal address is the same as the key
		if !utils.Equals(WithdrawalAddress, rawLeaf.WithdrawalAddress) {
			log.Fatal("withdrawal address in raw leaf doesnt match the key")
		}

		block := withdrawalToLeaf[WithdrawalAddress]
		proof, err := tree.Proof(block)

		if err != nil {
			log.Fatal("could not generate proof for block: ", err)
		}

		// Store the proofs of the withdrawal address (to be used onchain)
		proofs[WithdrawalAddress] = utils.ByteArrayToArray(proof.Siblings)

		// Store the leafs (to be used onchain)
		leafs[WithdrawalAddress] = rawLeaf
	}

	state := &OnchainState{
		Validators: validatorsCopy,
		MerkleRoot: merkleRootStr,
		Slot:       or.state.LatestProcessedSlot,
		Proofs:     proofs,
		Leafs:      leafs,
	}

	or.state.CommitedStates[state.Slot] = state
	return true
}

// Returns true and the latest commited slot if there is any commited state
// false otherwise. Note that if there are checkpoints but without enough data
// to create a tree, it will still return false
func (or *Oracle) LatestCommitedSlot() (uint64, bool) {
	or.mutex.RLock()
	defer or.mutex.RUnlock()

	if len(or.State().CommitedStates) == 0 {
		return 0, false
	}

	latestCommitedSlot := uint64(0)
	for slot, _ := range or.State().CommitedStates {
		if slot > latestCommitedSlot {
			latestCommitedSlot = slot
		}
	}
	return latestCommitedSlot, true
}

// Returns the last commited state
func (or *Oracle) LatestCommitedState() *OnchainState {
	or.mutex.RLock()
	defer or.mutex.RUnlock()

	latestCommitedSlot, atLeastOne := or.LatestCommitedSlot()

	// If we havent commited any state yet
	if !atLeastOne {
		return nil
	}

	return or.State().CommitedStates[latestCommitedSlot]
}

// Check if the oracle is in sync with a given root and slot. Its considered in sync
// when the latest commited state has the same root and slot as the onchain state
func (or *Oracle) IsOracleInSyncWithChain(onchainRoot string, onchainSlot uint64) (bool, error) {
	latestCommitedSlot, atLeastOne := or.LatestCommitedSlot()

	// If we havent commited any state yet
	if !atLeastOne {
		log.Info("Oracle has no commited states, no checkpoints have passed or there is not enough data to create a merkle tree")
		// If the onchain state is the default, we can consider in sync as the contract has not root also
		if utils.Equals(onchainRoot, DefaultRoot) {
			log.WithFields(log.Fields{
				"OnchainRoot": onchainRoot,
				"OnchainSlot": onchainSlot,
			}).Info("Oracle IS in sync with the latest onchain root, nothing was commited onchain yet")
			return true, nil
		}
		// If the onchain state is not the default, we are not in sync
		log.WithFields(log.Fields{
			"OnchainRoot": onchainRoot,
			"OnchainSlot": onchainSlot,
		}).Info("Oracle IS NOT in sync with the latest onchain root, oracle has no commited states yet")
		return false, nil
	}

	latestOracleRoot := or.State().CommitedStates[latestCommitedSlot].MerkleRoot

	if utils.Equals(onchainRoot, latestOracleRoot) && onchainSlot == latestCommitedSlot {
		log.WithFields(log.Fields{
			"OnchainRoot": onchainRoot,
			"OnchainSlot": onchainSlot,
			"OracleRoot":  latestOracleRoot,
			"OracleSlot":  latestCommitedSlot,
		}).Info("Oracle IS in sync with the latest onchain root")
		return true, nil
	}

	// If roots match but not slots or viceversa. Something is wrong
	if (utils.Equals(onchainRoot, latestOracleRoot) && onchainSlot != latestCommitedSlot) ||
		(!utils.Equals(onchainRoot, latestOracleRoot) && onchainSlot == latestCommitedSlot) {
		return false, errors.New(fmt.Sprintf("Oracle root/slot does not match the onchain root/slot. "+
			"OracleRoot: %s, OracleSlot: %d, OnchainRoot: %s, OnchainSlot: %d",
			latestOracleRoot, latestCommitedSlot, onchainRoot, onchainSlot))
	}

	log.WithFields(log.Fields{
		"OnchainRoot": onchainRoot,
		"OracleRoot":  latestOracleRoot,
		"OracleSlot":  latestCommitedSlot,
	}).Info("Oracle IS NOT in sync with the latest onchain root")
	return false, nil
}

// Returns true if a given validator can subscribe or not to the pool
// Accepted states are:
// -ValidatorStatePendingInitialized
// -ValidatorStatePendingQueued
// -ValidatorStateActiveOngoing
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
	return false
}

func (or *Oracle) hashStateLockFree() error {
	// We remove the hash before hashing, always hashing an empty hash
	or.state.StateHash = ""

	// Serialize the state
	jsonData, err := json.MarshalIndent(or.state, "", " ")
	if err != nil {
		return errors.Wrap(err, "could not marshal state to json")
	}

	// Calculate the hash of the state
	stateHash := sha256.Sum256(jsonData)
	stateHashStr := hexutil.Encode(stateHash[:])

	// Set the hash of the state
	or.state.StateHash = stateHashStr

	return nil
}

// Returns if a validator is subscribed to the pool. A validator is subscribed if
// its state is: active, yellowcard, redcard
func (or *Oracle) isSubscribed(validatorIndex uint64) bool {
	for valIndex, validator := range or.state.Validators {
		if valIndex == validatorIndex &&
			validator.ValidatorStatus != Banned &&
			validator.ValidatorStatus != NotSubscribed &&
			validator.ValidatorStatus != UnknownState {
			return true
		}
	}
	return false
}

// Returns true if a validator is banned
func (or *Oracle) isBanned(validatorIndex uint64) bool {
	validator, found := or.state.Validators[validatorIndex]
	if !found {
		return false
	}
	if validator.ValidatorStatus == Banned {
		return true
	}
	return false
}

// Returns true if a validator is tracked by the oracle. Any state is considered
// as tracked
func (or *Oracle) isTracked(validatorIndex uint64) bool {
	_, found := or.state.Validators[validatorIndex]
	if found {
		return true
	}
	return false
}

// Returns true if the given collateral is greater or equal the inputed one
func (or *Oracle) isCollateralEnough(collateral *big.Int) bool {
	return collateral.Cmp(or.state.CollateralInWei) >= 0
}

// Handles the donations of a given block
func (or *Oracle) handleDonations(donations []*contract.ContractEtherReceived) {
	// Ensure the donations are from the same block
	if len(donations) > 0 {
		blockReference := donations[0].Raw.BlockNumber
		for _, donation := range donations {
			if donation.Raw.BlockNumber != blockReference {
				log.Fatal("Handling donations from different blocks is not possible: ",
					donation.Raw.BlockNumber, " vs ", blockReference)
			}
		}
	}
	for _, donation := range donations {
		or.increaseAllPendingRewards(donation.DonationAmount)
		or.state.Donations = append(or.state.Donations, donation)
		log.WithFields(log.Fields{
			"RewardWei":   donation.DonationAmount,
			"BlockNumber": donation.Raw.BlockNumber,
			"Type":        "Donation",
			"TxHash":      donation.Raw.TxHash.String(),
		}).Info("[Reward]")
	}
}

// Handles a correct block proposal into the pool
func (or *Oracle) handleCorrectBlockProposal(block SummarizedBlock) {
	or.addSubscription(block.ValidatorIndex, block.WithdrawalAddress, block.ValidatorKey)
	or.advanceStateMachine(block.ValidatorIndex, ProposalOk)
	or.increaseAllPendingRewards(block.Reward)
	or.consolidateBalance(block.ValidatorIndex)
	or.state.ProposedBlocks = append(or.state.ProposedBlocks, block)

	log.WithFields(log.Fields{
		"Slot":       block.Slot,
		"Block":      block.Block,
		"ValIndex":   block.ValidatorIndex,
		"RewardWei":  block.Reward,
		"RewardType": block.RewardType.String(),
	}).Info("[Reward]")
}

// Handles the proposal of a block but that has BLS withdrawal keys
func (or *Oracle) handleBlsCorrectBlockProposal(block SummarizedBlock) {
	if block.BlockType != OkPoolProposalBlsKeys {
		log.Fatal("Block type is not OkPoolProposalBlsKeys, BlockType: ", block.BlockType)
	}
	log.WithFields(log.Fields{
		"BlockNumber":    block.Block,
		"Slot":           block.Slot,
		"ValidatorIndex": block.ValidatorIndex,
	}).Warn("Block proposal was ok but bls keys are not supported, sending rewards to pool")
	or.sendRewardToPool(block.Reward)
}

// Handles a manual subscription to the pool, meaning that an event from the smart contract
// was triggered. This function asserts if the subscription was valid and updates the state
// of the validator accordingly
func (or *Oracle) handleManualSubscriptions(
	subsEvents []*contract.ContractSubscribeValidator) {

	// Ensure the subscriptions events are from the same block
	if len(subsEvents) > 0 {
		blockReference := subsEvents[0].Raw.BlockNumber
		for _, donation := range subsEvents {
			if donation.Raw.BlockNumber != blockReference {
				log.Fatal("Handling manual subscriptions from different blocks is not possible: ",
					donation.Raw.BlockNumber, " vs ", blockReference)
			}
		}
	}

	if or.beaconValidators == nil {
		log.Fatal("Beacon validators cant be nil")
	}

	if len(or.beaconValidators) == 0 {
		log.Fatal("Beacon validators cant be empty")
	}

	for _, sub := range subsEvents {

		valIdx := sub.ValidatorID
		collateral := sub.SubscriptionCollateral
		sender := sub.Sender.String()

		validator, found := or.beaconValidators[phase0.ValidatorIndex(valIdx)]

		// Subscription recevied for a validator index that doesnt exist
		if !found {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Raw.BlockNumber,
				"Collateral":     sub.SubscriptionCollateral,
				"TxHash":         sub.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for non existing validator, skipping")
			// Fees go to the pool, since validator does not exist in the pool and it is not tracked
			or.sendRewardToPool(collateral)
			continue
		}

		if valIdx != uint64(validator.Index) {
			log.Fatal("Subscription event validator index doesnt match the validator index. ",
				valIdx, " vs ", validator.Index)
		}

		// Subscription received for a validator that cannot subscribe (see states)
		if !CanValidatorSubscribeToPool(validator) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Raw.BlockNumber,
				"Collateral":     sub.SubscriptionCollateral,
				"TxHash":         sub.Raw.TxHash,
				"ValidatorIndex": valIdx,
				"ValidatorState": validator.Status,
			}).Warn("[Subscription]: for validator that cannot subscribe due to its state, skipping")
			// Fees go to the pool, since validator is not operational (withdrawn, slashed, etc)
			or.sendRewardToPool(collateral)
			continue
		}

		// Subscription received for a validator that dont have eth1 withdrawal address (bls)
		validatorWithdrawal, err := utils.GetEth1AddressByte(validator.Validator.WithdrawalCredentials)
		if err != nil {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Raw.BlockNumber,
				"Collateral":     sub.SubscriptionCollateral,
				"TxHash":         sub.Raw.TxHash,
				"WithdrawalAddr": hexutil.Encode(validator.Validator.WithdrawalCredentials[:]),
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for validator with invalid withdrawal address (bls), skipping")
			// Fees go to the pool. A validator with a bls address can not be tracked since it has not been able to subscribe.
			or.sendRewardToPool(collateral)
			continue
		}

		// Subscription received from an address that is not the validator withdrawal address
		if !utils.Equals(sender, validatorWithdrawal) {
			log.WithFields(log.Fields{
				"BlockNumber":         sub.Raw.BlockNumber,
				"Collateral":          sub.SubscriptionCollateral,
				"TxHash":              sub.Raw.TxHash,
				"ValidatorIndex":      valIdx,
				"Sender":              sender,
				"ValidatorWithdrawal": validatorWithdrawal,
			}).Warn("[Subscription]: but tx sender is not the validator withdrawal address, skipping")
			// Fees go to the pool.
			// TODO: maybe we could check if sender has a validator registered with withdrawal address = sender, and if so, give the collateral back to the sender
			or.sendRewardToPool(collateral)
			continue
		}

		// Subscription received for a banned validator
		if or.isBanned(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Raw.BlockNumber,
				"Collateral":     sub.SubscriptionCollateral,
				"TxHash":         sub.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for banned validator, skipping")
			// Since we track this validator, give the collateral back
			or.increaseValidatorAccumulatedRewards(valIdx, collateral)
			continue
		}

		// Subscription received for an already subscribed validator
		if or.isSubscribed(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Raw.BlockNumber,
				"Collateral":     sub.SubscriptionCollateral,
				"TxHash":         sub.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for an already subscribed validator, skipping")
			// Since we track this validator, return the collateral as accumulated balance
			or.increaseValidatorAccumulatedRewards(valIdx, collateral)
			continue
		}

		// Subscription received for a validator with not enough collateral
		if !or.isCollateralEnough(collateral) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Raw.BlockNumber,
				"Collateral":     sub.SubscriptionCollateral,
				"TxHash":         sub.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for a validator with not enough collateral, skipping")
			// Fees go to the pool, since validator does not exist in the pool
			or.sendRewardToPool(collateral)
			continue
		}

		// Valid subscription
		if !or.isSubscribed(valIdx) &&
			!or.isBanned(valIdx) &&
			or.isCollateralEnough(collateral) {

			// Add valid subscription
			if !or.isTracked(valIdx) {
				// If its not tracked, we create a new subscription
				or.state.Validators[valIdx] = &ValidatorInfo{
					ValidatorStatus:       NotSubscribed,
					AccumulatedRewardsWei: big.NewInt(0),
					PendingRewardsWei:     big.NewInt(0),
					CollateralWei:         collateral,
					WithdrawalAddress:     validatorWithdrawal,
					ValidatorIndex:        valIdx,
					ValidatorKey:          hexutil.Encode(validator.Validator.PublicKey[:]),
				}
			}
			log.WithFields(log.Fields{
				"BlockNumber":      sub.Raw.BlockNumber,
				"Collateral":       sub.SubscriptionCollateral,
				"TxHash":           sub.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": validatorWithdrawal,
			}).Info("[Subscription]: Validator subscribed ok")
			or.increaseValidatorPendingRewards(valIdx, collateral)
			or.advanceStateMachine(valIdx, ManualSubscription)
			or.state.Subscriptions = append(or.state.Subscriptions, sub)
			continue
		}

		// If we reach this point, its a case we havent considered, but its not valid
		log.WithFields(log.Fields{
			"BlockNumber":      sub.Raw.BlockNumber,
			"Collateral":       sub.SubscriptionCollateral,
			"TxHash":           sub.Raw.TxHash,
			"ValidatorIndex":   valIdx,
			"WithdrawaAddress": validatorWithdrawal,
		}).Info("[Subscription]: Not considered case meaning wrong subscription, skipping")
		// Send the collateral to the pool
		or.sendRewardToPool(collateral)
	}
}

// Handle the unsubscriptions detected as events triggered from the contract for a given block
// If the unsubscription matches some criteria, we update the state of the validator. Main criteria
// is that the sender matches the withdrawal address of the validator
func (or *Oracle) handleManualUnsubscriptions(
	unsubEvents []*contract.ContractUnsubscribeValidator) {

	// Ensure the subscriptions events are from the same block
	if len(unsubEvents) > 0 {
		blockReference := unsubEvents[0].Raw.BlockNumber
		for _, donation := range unsubEvents {
			if donation.Raw.BlockNumber != blockReference {
				log.Fatal("Handling manual unsubscriptions from different blocks is not possible: ",
					donation.Raw.BlockNumber, " vs ", blockReference)
			}
		}
	}

	if or.beaconValidators == nil {
		log.Fatal("Beacon validators cant be nil")
	}

	if len(or.beaconValidators) == 0 {
		log.Fatal("Beacon validators cant be empty")
	}

	for _, unsub := range unsubEvents {

		valIdx := unsub.ValidatorID
		sender := unsub.Sender.String()

		validator, found := or.beaconValidators[phase0.ValidatorIndex(valIdx)]

		// Unsubscription but for a validator that doesnt exist
		if !found {
			log.WithFields(log.Fields{
				"BlockNumber":    unsub.Raw.BlockNumber,
				"TxHash":         unsub.Raw.TxHash,
				"Sender":         sender,
				"ValidatorIndex": valIdx,
			}).Warn("[Unsubscription]: for validator index that does not exist, skipping")
			continue
		}

		if validator.Index != phase0.ValidatorIndex(valIdx) {
			log.Fatal("Unsubscription event validator index doesnt match the validator index. ",
				valIdx, " vs ", validator.Index)
		}

		// Unsubscription but for a validator that does not have an eth1 address. Should never happen
		withdrawalAddress, err := utils.GetEth1AddressByte(validator.Validator.WithdrawalCredentials)
		if err != nil {
			log.WithFields(log.Fields{
				"BlockNumber":    unsub.Raw.BlockNumber,
				"TxHash":         unsub.Raw.TxHash,
				"Sender":         sender,
				"ValidatorIndex": valIdx,
			}).Warn("[Unsubscription]: for validator with no eth1 withdrawal addres (bls), skipping")
			continue
		}

		// Its very important to check that the unsubscription was made from the withdrawal address
		// of the validator, otherwise anyone could call the unsubscription function.
		if !utils.Equals(sender, withdrawalAddress) {
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Raw.BlockNumber,
				"TxHash":           unsub.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Warn("[Unsubscription] but sender does not match withdrawal address, skipping")
			continue
		}

		// After all the checks, we can proceed with the unsubscription
		if or.isSubscribed(valIdx) {
			or.advanceStateMachine(valIdx, Unsubscribe)
			or.increaseAllPendingRewards(or.state.Validators[valIdx].PendingRewardsWei)
			or.resetPendingRewards(valIdx)
			or.state.Unsubscriptions = append(or.state.Unsubscriptions, unsub)
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Raw.BlockNumber,
				"TxHash":           unsub.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Info("[Unsubscription] Validator unsubscribed ok")
			continue
		}

		if !or.isSubscribed(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Raw.BlockNumber,
				"TxHash":           unsub.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Warn("[Unsubscription] but the validator is not subscribed, skipping")
			continue
		}

		// If we reach this point, its a case we havent considered, but its not valid
		log.WithFields(log.Fields{
			"BlockNumber":      unsub.Raw.BlockNumber,
			"TxHash":           unsub.Raw.TxHash,
			"ValidatorIndex":   valIdx,
			"WithdrawaAddress": withdrawalAddress,
			"Sender":           sender,
		}).Warn("[Unsubscription] Not considered case meaning wrong unsubscription, skipping")
	}
}

// Banning a validator implies sharing its pending rewards among the rest
// of the validators and setting its pending to zero.
func (or *Oracle) handleBanValidator(block SummarizedBlock) {
	// First of all advance the state machine, so the banned validator is not
	// considered for the pending reward share
	or.advanceStateMachine(block.ValidatorIndex, ProposalWrongFee)
	or.increaseAllPendingRewards(or.state.Validators[block.ValidatorIndex].PendingRewardsWei)
	or.resetPendingRewards(block.ValidatorIndex)

	// Store the proof of the wrong fee block. Reason why it was banned
	or.state.WrongFeeBlocks = append(or.state.WrongFeeBlocks, block)
}

// Handles the case of a validator that has missed a block, only to be used
// with subscribed validators into the pool
func (or *Oracle) handleMissedBlock(block SummarizedBlock) {
	or.advanceStateMachine(block.ValidatorIndex, ProposalMissed)
	or.state.MissedBlocks = append(or.state.MissedBlocks, block)
}

// Subscribes a validator index with a given withdrawal address and validator key
func (or *Oracle) addSubscription(valIndex uint64, withdrawalAddress string, validatorKey string) {
	validator, found := or.state.Validators[valIndex]
	if !found {
		// If not found and not manually subscribed, we trigger the AutoSubscription event
		// Instantiate the validator
		validator = &ValidatorInfo{
			ValidatorStatus:       NotSubscribed,
			AccumulatedRewardsWei: big.NewInt(0),
			PendingRewardsWei:     big.NewInt(0),
			CollateralWei:         big.NewInt(0),
			WithdrawalAddress:     withdrawalAddress,
			ValidatorIndex:        valIndex,
			ValidatorKey:          validatorKey,
		}
		or.state.Validators[valIndex] = validator

		// And update it state according to the event
		or.advanceStateMachine(valIndex, AutoSubscription)
	} else {
		// If we found the validator and is not subscribed, advance the state machine
		// Most likely it was subscribed before, then unsubscribed and now auto subscribes
		if !or.isSubscribed(valIndex) {
			or.advanceStateMachine(valIndex, AutoSubscription)
		}
	}
}

// Consolidate the balance of a given validator index. This means moving the pending to its accumulated
// and setting the pending to zero.
func (or *Oracle) consolidateBalance(valIndex uint64) {

	beforePending := new(big.Int).Set(or.state.Validators[valIndex].PendingRewardsWei)
	beforeAccumulated := new(big.Int).Set(or.state.Validators[valIndex].AccumulatedRewardsWei)

	or.state.Validators[valIndex].AccumulatedRewardsWei.Add(or.state.Validators[valIndex].AccumulatedRewardsWei, or.state.Validators[valIndex].PendingRewardsWei)
	or.state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)

	log.WithFields(log.Fields{
		"AccumulatedAfter":  or.state.Validators[valIndex].AccumulatedRewardsWei,
		"AccumulatedBefore": beforeAccumulated,
		"PendingAfter":      or.state.Validators[valIndex].PendingRewardsWei,
		"PendingBefore":     beforePending,
		"ValIndex":          valIndex,
	}).Debug("Consolidating balance")
}

// Returns a list of all the eligible validators for rewards.
func (or *Oracle) getEligibleValidators() []uint64 {
	eligibleValidators := make([]uint64, 0)

	for valIndex, validator := range or.state.Validators {
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
func (or *Oracle) increaseAllPendingRewards(
	reward *big.Int) {

	eligibleValidators := or.getEligibleValidators()
	numEligibleValidators := big.NewInt(int64(len(eligibleValidators)))

	if len(eligibleValidators) == 0 {
		log.Warn("No validators are eligible to receive rewards, pool fees address will receive all")
		or.state.PoolAccumulatedFees.Add(or.state.PoolAccumulatedFees, reward)
		return
	}

	if or.state.PoolFeesPercentOver10000 > 100*100 {
		log.Fatal("Pool fees percent cannot be greater than 100% (10000) value: ", or.state.PoolFeesPercentOver10000)
	}

	// 100 is the % and the other 100 is because we use two decimals
	// eg 1000 is 10%
	// eg 50 is 0.5%
	over := big.NewInt(100 * 100)

	// The pool takes PoolFeesPercentOver10000 cut of the rewards
	aux := big.NewInt(0).Mul(reward, big.NewInt(int64(or.state.PoolFeesPercentOver10000)))

	// Calculate the pool cut
	poolCut := big.NewInt(0).Div(aux, over)

	// And remainder of above operation
	remainder1 := big.NewInt(0).Mod(aux, over)

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
	or.state.PoolAccumulatedFees.Add(or.state.PoolAccumulatedFees, totalFees)

	log.WithFields(log.Fields{
		"AmountEligibleValidators": numEligibleValidators,
		"RewardPerValidatorWei":    perValidatorReward,
		"PoolFeesWei":              totalFees,
		"TotalRewardWei":           reward,
	}).Info("Increasing pending rewards of eligible validators")

	// Increase eligible validators rewards
	for _, eligibleIndex := range eligibleValidators {
		or.state.Validators[eligibleIndex].PendingRewardsWei.Add(or.state.Validators[eligibleIndex].PendingRewardsWei, perValidatorReward)
	}
}

// Increases the pending rewards of a given validator index.
func (or *Oracle) increaseValidatorPendingRewards(valIndex uint64, reward *big.Int) {
	beforePending := new(big.Int).Set(or.state.Validators[valIndex].PendingRewardsWei)
	or.state.Validators[valIndex].PendingRewardsWei.Add(or.state.Validators[valIndex].PendingRewardsWei, reward)

	log.WithFields(log.Fields{
		"PendingAfter":  or.state.Validators[valIndex].PendingRewardsWei,
		"PendingBefore": beforePending,
		"RewardShare":   reward,
		"ValIndex":      valIndex,
	}).Debug("Increasing validator pending rewards")
}

// Increases the accumulated rewards of a given validator index.
func (or *Oracle) increaseValidatorAccumulatedRewards(valIndex uint64, reward *big.Int) {
	accumulatedBefore := new(big.Int).Set(or.state.Validators[valIndex].AccumulatedRewardsWei)

	or.state.Validators[valIndex].AccumulatedRewardsWei.Add(or.state.Validators[valIndex].AccumulatedRewardsWei, reward)

	log.WithFields(log.Fields{
		"AccumulatedAfter":  or.state.Validators[valIndex].AccumulatedRewardsWei,
		"AccumulatedBefore": accumulatedBefore,
		"RewardShare":       reward,
		"ValIndex":          valIndex,
	}).Debug("Increasing validator accumulated rewards")
}

// Sends the pool cut to the pool reward address.
func (or *Oracle) sendRewardToPool(reward *big.Int) {

	poolAccumulatedBefore := new(big.Int).Set(or.state.PoolAccumulatedFees)
	or.state.PoolAccumulatedFees.Add(or.state.PoolAccumulatedFees, reward)

	log.WithFields(log.Fields{
		"PoolAccumulatedBefore": poolAccumulatedBefore,
		"PoolAccumulatedAfter":  or.state.PoolAccumulatedFees,
		"RewardShare":           reward,
	}).Debug("Sending reward cut to pool reward address")
}

// Resets the pending rewards of a given validator index.
func (or *Oracle) resetPendingRewards(valIndex uint64) {
	log.WithFields(log.Fields{
		"PendingRewardsBefore": or.state.Validators[valIndex].PendingRewardsWei,
		"ValIndex":             valIndex,
	}).Debug("Resetting pending rewards")
	or.state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

// Gets the merkle root of the state and returns if there was enough data
// to generate it or not.
func (or *Oracle) getMerkleRootIfAny() (string, bool) {
	mk := NewMerklelizer()
	_, _, tree, enoughData := mk.GenerateTreeFromState(or.state)
	if !enoughData {
		return "", enoughData
	}
	merkleRootStr := hexutil.Encode(tree.Root[:])

	return merkleRootStr, true
}

// Returns the 0x prefixed withdrawal credentials and its type: BlsWithdrawal or Eth1Withdrawal
func GetWithdrawalAndType(validator *v1.Validator) (string, WithdrawalType) {
	withdrawalCred := hex.EncodeToString(validator.Validator.WithdrawalCredentials)
	if len(withdrawalCred) != 64 {
		log.Fatal("withdrawal credentials are not a valid length: ", len(withdrawalCred))
	}

	if utils.IsBlsType(withdrawalCred) {
		return "0x" + withdrawalCred[2:], BlsWithdrawal
	} else if utils.IsEth1Type(withdrawalCred) {
		return "0x" + withdrawalCred[24:], Eth1Withdrawal
	}
	log.Fatal("withdrawal credentials are not a valid type: ", withdrawalCred)
	return "", 0
}

// See the spec for state diagram with states and transitions. This tracks all the different
// states and state transitions that a given validator can have from the oracle point of view
func (or *Oracle) advanceStateMachine(valIndex uint64, event Event) {
	switch or.state.Validators[valIndex].ValidatorStatus {
	case Active:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":          "ProposalOk",
				"StateChange":    "Active -> Active",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Active
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "Active -> Banned",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "Active -> YellowCard",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = YellowCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "Unsubscribe",
				"StateChange":    "Active -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case YellowCard:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":          "ProposalOk",
				"StateChange":    "YellowCard -> Active",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Active
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "YellowCard -> Banned",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "YellowCard -> RedCard",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = RedCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "Unsubscribe",
				"StateChange":    "YellowCard -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case RedCard:
		switch event {
		case ProposalOk:
			log.WithFields(log.Fields{
				"Event":          "ProposalOk",
				"StateChange":    "RedCard -> YellowCard",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = YellowCard
		case ProposalWrongFee:
			log.WithFields(log.Fields{
				"Event":          "ProposalWrongFee",
				"StateChange":    "RedCard -> Banned",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Banned
		case ProposalMissed:
			log.WithFields(log.Fields{
				"Event":          "ProposalMissed",
				"StateChange":    "RedCard -> RedCard",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = RedCard
		case Unsubscribe:
			log.WithFields(log.Fields{
				"Event":          "Unsubscribe",
				"StateChange":    "RedCard -> NotSubscribed",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = NotSubscribed
		}
	case NotSubscribed:
		switch event {
		case ManualSubscription:
			log.WithFields(log.Fields{
				"Event":          "ManualSubscription",
				"StateChange":    "NotSubscribed -> Active",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Active
		case AutoSubscription:
			log.WithFields(log.Fields{
				"Event":          "AutoSubscription",
				"StateChange":    "NotSubscribed -> Active",
				"ValidatorIndex": valIndex,
				"Slot":           or.state.NextSlotToProcess,
			}).Info("Validator state change")
			or.state.Validators[valIndex].ValidatorStatus = Active
		}
	}
}
