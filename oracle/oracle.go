package oracle

import (
	"crypto/sha256"
	"encoding/gob"
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
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/sirupsen/logrus"
)

var StateFileName = "state.gob"
var StateFolder = "oracle-data"

var StateJsonName = "state.json"

type Oracle struct {
	cfg   *config.Config
	state *OracleState
	mutex sync.RWMutex
}

// TODO: Make private the functions that should not be accessed outside of the package

func NewOracle(cfg *config.Config) *Oracle {
	state := &OracleState{
		StateHash:            "",
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
		LatestCommitedState: OnchainState{
			Validators: make(map[uint64]*ValidatorInfo, 0),
			Slot:       0,
			TxHash:     "",
			MerkleRoot: DefaultRoot,
			Proofs:     make(map[string][]string, 0),
			Leafs:      make(map[string]RawLeaf, 0),
		},
		CommitedStates: make(map[string]OnchainState, 0),
	}

	oracle := &Oracle{
		cfg:   cfg,
		state: state,
	}

	return oracle
}

func (or *Oracle) State() *OracleState {
	or.mutex.RLock()
	defer or.mutex.RUnlock()
	return or.state
}

// Advances the oracle to the next state, processing LatestSlot proposals/donations
// calculating the new state of all validators. It returns the slot that was processed
// and if there was an error.

// TODO: Here provide the block class, that will contain all events etc.
func (or *Oracle) AdvanceStateToNextSlot(
	blockPool Block,
	blockSubs []Subscription,
	blockUnsubs []Unsubscription,
	blockDonations []Donation) (uint64, error) {

	or.mutex.Lock()
	defer or.mutex.Unlock()

	// TODO: Ensure blockPool is == nextSlot to process? or smlt

	if or.state.NextSlotToProcess != (or.state.LatestProcessedSlot + 1) {
		log.Fatal("Next slot to process is not the last processed slot + 1",
			or.state.NextSlotToProcess, " ", or.state.LatestProcessedSlot)
	}

	err := or.validateParameters(blockPool, blockSubs, blockUnsubs, blockDonations)
	if err != nil {
		return 0, err
	}

	// Handle subscriptions first thing
	or.handleManualSubscriptions(blockSubs)

	// If the validator was subscribed and missed proposed the block in this slot
	if blockPool.BlockType == MissedProposal && or.isSubscribed(blockPool.ValidatorIndex) {
		or.handleMissedBlock(blockPool)
	}

	// If we have a successful block proposal BUT the validator has BLS keys, we cant auto subscribe it
	if blockPool.BlockType == OkPoolProposalBlsKeys {
		or.handleBlsCorrectBlockProposal(blockPool)
	}

	// Manual subscription. If feeRec is ok, means the reward was sent to the pool
	if blockPool.BlockType == OkPoolProposal {
		or.handleCorrectBlockProposal(blockPool)
	}

	// If the validator was subscribed but the fee recipient was wrong we ban the validator
	if blockPool.BlockType == WrongFeeRecipient && or.isSubscribed(blockPool.ValidatorIndex) {
		or.handleBanValidator(blockPool)
	}

	// Handle unsubscriptions the last thing after distributing rewards
	or.handleManualUnsubscriptions(blockUnsubs)

	// Handle the donations from this block
	or.handleDonations(blockDonations)

	processedSlot := or.state.NextSlotToProcess
	or.state.LatestProcessedSlot = processedSlot
	or.state.NextSlotToProcess++
	if blockPool.BlockType != MissedProposal {
		or.state.LatestProcessedBlock = blockPool.Block
	}
	return processedSlot, nil
}

func (or *Oracle) validateParameters(
	blockPool Block,
	blockSubs []Subscription,
	blockUnsubs []Unsubscription,
	blockDonations []Donation) error {

	if blockPool.Slot != or.state.NextSlotToProcess {
		return errors.New(fmt.Sprint("Slot of blockPool is not the same as the latest slot of the oracle. BlockPool: ",
			blockPool.Slot, " Oracle: ", or.state.NextSlotToProcess))
	}

	// TODO: Add more validators to block subs unsubs, donations, etc
	return nil
}

func (or *Oracle) hashStateLockFree() error {
	// We remove the hash before hashing, always hashing an empty hash
	or.state.StateHash = ""

	// Serialize the state
	jsonData, err := json.Marshal(or.state)
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

func (or *Oracle) SaveToJson() error {
	// Not just read lock since we change the hash
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

	log.Trace(fmt.Sprintf("Saving oracle state: %s", jsonData))

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

func (or *Oracle) LoadFromJson() error {
	or.mutex.Lock()
	defer or.mutex.Unlock()

	path := filepath.Join(StateFolder, StateJsonName)
	log.Info("Loading oracle state from json file: ", path)

	jsonFile, err := os.Open(path)
	defer jsonFile.Close()
	if err != nil {
		return errors.Wrap(err, "could not open json file")
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return errors.Wrap(err, "could not read json file")
	}

	var state OracleState

	err = json.Unmarshal(byteValue, &state)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal json file")
	}

	// Store the hash we recovered from the file
	recoveredHash := state.StateHash

	// Reset the hash since we want to hash the content without the hash
	state.StateHash = ""

	// Serialize the state without the hash
	jsonNoHash, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return errors.Wrap(err, "could not marshal state without hash")
	}

	// We calculate the hash of the state we read
	calculatedHashByte := sha256.Sum256(jsonNoHash[:])
	calculatedHashStrig := hexutil.Encode(calculatedHashByte[:])

	// Hashes must match
	if !Equals(recoveredHash, calculatedHashStrig) {
		return errors.Wrap(err, fmt.Sprintf("hash mismatch, recovered: %s, calculated: %s",
			recoveredHash, calculatedHashStrig))
	}

	if state.Config.Network != or.state.Config.Network {
		return errors.Wrap(err, fmt.Sprintf("network mismatch, recovered: %s, expected: %s",
			state.Config.Network, or.state.Config.Network))
	}

	if state.Config.PoolAddress != or.state.Config.PoolAddress {
		return errors.Wrap(err, fmt.Sprintf("pool address mismatch, recovered: %s, expected: %s",
			state.Config.PoolAddress, or.state.Config.PoolAddress))
	}

	if state.Config.PoolFeesAddress != or.state.Config.PoolFeesAddress {
		return errors.Wrap(err, fmt.Sprintf("pool fees address mismatch, recovered: %s, expected: %s",
			state.Config.PoolFeesAddress, or.state.Config.PoolFeesAddress))
	}

	if state.Config.PoolFeesPercent != or.state.Config.PoolFeesPercent {
		return errors.Wrap(err, fmt.Sprintf("pool fees percent mismatch, recovered: %d, expected: %d",
			state.Config.PoolFeesPercent, or.state.Config.PoolFeesPercent))
	}

	// TODO: Add more checks?
	// TODO: Run reconcilization?

	or.state = &state
	return nil
}

// TODO: Remove when migrated to json
func (or *Oracle) SaveStateToFile() {
	or.mutex.RLock()
	defer or.mutex.RUnlock()

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
		"LatestProcessedSlot":  or.state.LatestProcessedSlot,
		"LatestProcessedBlock": or.state.LatestProcessedBlock,
		"NextSlotToProcess":    or.state.NextSlotToProcess,
		"TotalValidators":      len(or.state.Validators),
		"Network":              or.state.Network,
		"PoolAddress":          or.state.PoolAddress,
		"Path":                 path,
		//"MerkleRoot":      mRoot,
		//"EnoughData":      enoughData,
	}).Info("Saving state to file")

	err = encoder.Encode(or.state)
	if err != nil {
		log.Fatal("could not encode state into file: ", err)
	}
}

// TODO: Remove when migrated to json
func (or *Oracle) LoadStateFromFile() error {
	or.mutex.Lock()
	defer or.mutex.Unlock()
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

	if readState.Config.Network != or.state.Config.Network {
		log.Fatal("Error loading state from file. Network mismatch. Expected: ",
			or.state.Config.Network, " Got: ", readState.Config.Network)
	}

	if readState.Config.PoolAddress != or.state.Config.PoolAddress {
		log.Fatal("Error loading state from file. PoolAddress mismatch. Expected: ",
			or.state.Config.PoolAddress, " Got: ", readState.Config.PoolAddress)
	}

	if readState.Config.PoolFeesAddress != or.state.Config.PoolFeesAddress {
		log.Fatal("Error loading state from file. PoolFeesAddress mismatch. Expected: ",
			or.state.Config.PoolFeesAddress, " Got: ", readState.Config.PoolFeesAddress)
	}

	if readState.Config.PoolFeesPercent != or.state.Config.PoolFeesPercent {
		log.Fatal("Error loading state from file. PoolFeesPercent mismatch. Expected: ",
			or.state.Config.PoolFeesPercent, " Got: ", readState.Config.PoolFeesPercent)
	}

	mRoot, enoughData := or.GetMerkleRootIfAny()

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
	or.state.LatestProcessedSlot = readState.LatestProcessedSlot
	or.state.NextSlotToProcess = readState.NextSlotToProcess
	or.state.LatestProcessedBlock = readState.LatestProcessedBlock
	//state.Network = readState.Network
	//state.PoolAddress = readState.PoolAddress
	or.state.Validators = readState.Validators
	or.state.LatestCommitedState = readState.LatestCommitedState
	or.state.PoolFeesPercent = readState.PoolFeesPercent
	or.state.PoolFeesAddress = readState.PoolFeesAddress
	or.state.PoolAccumulatedFees = readState.PoolAccumulatedFees
	or.state.Subscriptions = readState.Subscriptions
	or.state.Unsubscriptions = readState.Unsubscriptions
	or.state.Donations = readState.Donations
	or.state.ProposedBlocks = readState.ProposedBlocks
	or.state.MissedBlocks = readState.MissedBlocks
	or.state.WrongFeeBlocks = readState.WrongFeeBlocks

	return nil
}

// Returns false if there wasnt enough data to create a merkle tree
func (or *Oracle) StoreLatestOnchainState() bool {
	or.mutex.Lock()
	defer or.mutex.Unlock()

	validatorsCopy := make(map[uint64]*ValidatorInfo)
	DeepCopy(or.state.Validators, &validatorsCopy)

	mk := NewMerklelizer()
	withdrawalToLeaf, withdrawalToRawLeaf, tree, enoughData := mk.GenerateTreeFromState(or.state)
	if !enoughData {
		return false
	}
	merkleRootStr := "0x" + hex.EncodeToString(tree.Root)

	log.WithFields(log.Fields{
		"Slot":       or.state.LatestProcessedSlot,
		"MerkleRoot": merkleRootStr,
	}).Info("Freezing state")

	// Merkle proofs for each withdrawal address
	proofs := make(map[string][]string)
	leafs := make(map[string]RawLeaf)
	for WithdrawalAddress, rawLeaf := range withdrawalToRawLeaf {

		// Extra sanity check to make sure the withdrawal address is the same as the key
		if !Equals(WithdrawalAddress, rawLeaf.WithdrawalAddress) {
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

	or.state.LatestCommitedState = OnchainState{
		Validators: validatorsCopy,
		//TxHash:     txHash, // TODO: Not sure if to store it
		MerkleRoot: merkleRootStr,
		Slot:       or.state.LatestProcessedSlot,
		Proofs:     proofs,
		Leafs:      leafs,
	}

	// besides the latestCommitedState as a "standalone" state,
	// we also store it in the commitedStates map, where we keep all
	// the states that have been commited onchain by hash

	// TODO: This should be the slot not the root I think.
	or.state.CommitedStates[merkleRootStr] = or.state.LatestCommitedState
	return true
}

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

func (or *Oracle) IsBanned(validatorIndex uint64) bool {
	validator, found := or.state.Validators[validatorIndex]
	if !found {
		return false
	}
	if validator.ValidatorStatus == Banned {
		return true
	}
	return false
}

func (or *Oracle) IsTracked(validatorIndex uint64) bool {
	_, found := or.state.Validators[validatorIndex]
	if found {
		return true
	}
	return false
}

func (or *Oracle) IsCollateralEnough(collateral *big.Int) bool {
	return collateral.Cmp(or.state.Config.CollateralInWei) >= 0
}

func (or *Oracle) handleDonations(donations []Donation) {
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
		or.IncreaseAllPendingRewards(donation.AmountWei)
		or.state.Donations = append(or.state.Donations, donation)
	}
}

func (or *Oracle) handleCorrectBlockProposal(block Block) {
	or.AddSubscriptionIfNotAlready(block.ValidatorIndex, block.WithdrawalAddress, block.ValidatorKey)
	or.AdvanceStateMachine(block.ValidatorIndex, ProposalOk)
	or.IncreaseAllPendingRewards(block.Reward)
	or.ConsolidateBalance(block.ValidatorIndex)
	or.state.ProposedBlocks = append(or.state.ProposedBlocks, block)
}

func (or *Oracle) handleBlsCorrectBlockProposal(block Block) {
	if block.BlockType != OkPoolProposalBlsKeys {
		log.Fatal("Block type is not OkPoolProposalBlsKeys, BlockType: ", block.BlockType)
	}
	log.WithFields(log.Fields{
		"BlockNumber":    block.Block,
		"Slot":           block.Slot,
		"ValidatorIndex": block.ValidatorIndex,
	}).Warn("Block proposal was ok but bls keys are not supported, sending rewards to pool")
	or.SendRewardToPool(block.Reward)
}

func (or *Oracle) handleManualSubscriptions(
	subscriptions []Subscription) {

	for _, sub := range subscriptions {

		valIdx := sub.Event.ValidatorID
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
			// Fees go to the pool, since validator does not exist in the pool and it is not tracked
			or.SendRewardToPool(collateral)
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
			// Fees go to the pool, since validator is not operational (withdrawn, slashed, etc)
			or.SendRewardToPool(collateral)
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
			// Fees go to the pool. A validator with a bls address can not be tracked since it has not been able to subscribe.
			or.SendRewardToPool(collateral)
			continue
		}

		// Subscription received from an address that is not the validator withdrawal address
		if !Equals(sender, validatorWithdrawal) {
			log.WithFields(log.Fields{
				"BlockNumber":         sub.Event.Raw.BlockNumber,
				"Collateral":          sub.Event.SubscriptionCollateral,
				"TxHash":              sub.Event.Raw.TxHash,
				"ValidatorIndex":      valIdx,
				"Sender":              sender,
				"ValidatorWithdrawal": validatorWithdrawal,
			}).Warn("[Subscription]: but tx sender is not the validator withdrawal address, skipping")
			// Fees go to the pool.
			// TODO: maybe we could check if sender has a validator registered with withdrawal address = sender, and if so, give the collateral back to the sender
			or.SendRewardToPool(collateral)
			continue
		}

		// Subscription received for a banned validator
		if or.IsBanned(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for banned validator, skipping")
			// Since we track this validator, give the collateral back
			or.IncreaseValidatorAccumulatedRewards(valIdx, collateral)
			continue
		}

		// Subscription received for an already subscribed validator
		if or.isSubscribed(valIdx) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for an already subscribed validator, skipping")
			// Since we track this validator, return the collateral as accumulated balance
			or.IncreaseValidatorAccumulatedRewards(valIdx, collateral)
			continue
		}

		// Subscription received for a validator with not enough collateral
		if !or.IsCollateralEnough(collateral) {
			log.WithFields(log.Fields{
				"BlockNumber":    sub.Event.Raw.BlockNumber,
				"Collateral":     sub.Event.SubscriptionCollateral,
				"TxHash":         sub.Event.Raw.TxHash,
				"ValidatorIndex": valIdx,
			}).Warn("[Subscription]: for a validator with not enough collateral, skipping")
			// Fees go to the pool, since validator does not exist in the pool
			or.SendRewardToPool(collateral)
			continue
		}

		// Valid subscription
		if !or.isSubscribed(valIdx) &&
			!or.IsBanned(valIdx) &&
			or.IsCollateralEnough(collateral) {

			// Add valid subscription
			if !or.IsTracked(valIdx) {
				// If its not tracked, we create a new subscription
				or.state.Validators[valIdx] = &ValidatorInfo{
					ValidatorStatus:       NotSubscribed,
					AccumulatedRewardsWei: big.NewInt(0),
					PendingRewardsWei:     big.NewInt(0),
					CollateralWei:         collateral,
					WithdrawalAddress:     validatorWithdrawal,
					ValidatorIndex:        valIdx,
					ValidatorKey:          "0x" + hex.EncodeToString(sub.Validator.Validator.PublicKey[:]),
				}
			}
			log.WithFields(log.Fields{
				"BlockNumber":      sub.Event.Raw.BlockNumber,
				"Collateral":       sub.Event.SubscriptionCollateral,
				"TxHash":           sub.Event.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": validatorWithdrawal,
			}).Info("[Subscription]: Validator subscribed ok")
			or.IncreaseValidatorPendingRewards(valIdx, collateral)
			or.AdvanceStateMachine(valIdx, ManualSubscription)
			or.state.Subscriptions = append(or.state.Subscriptions, sub)
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
		// Send the collateral to the pool
		or.SendRewardToPool(collateral)
	}
}

// Banning a validator implies sharing its pending rewards among the rest
// of the validators and setting its pending to zero.
func (or *Oracle) handleBanValidator(block Block) {
	// First of all advance the state machine, so the banned validator is not
	// considered for the pending reward share
	or.AdvanceStateMachine(block.ValidatorIndex, ProposalWrongFee)
	or.IncreaseAllPendingRewards(or.state.Validators[block.ValidatorIndex].PendingRewardsWei)
	or.ResetPendingRewards(block.ValidatorIndex)

	// Store the proof of the wrong fee block. Reason why it was banned
	or.state.WrongFeeBlocks = append(or.state.WrongFeeBlocks, block)
}

func (or *Oracle) handleMissedBlock(block Block) {
	or.AdvanceStateMachine(block.ValidatorIndex, ProposalMissed)
	or.state.MissedBlocks = append(or.state.MissedBlocks, block)
}

func (or *Oracle) handleManualUnsubscriptions(
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
		if !Equals(sender, withdrawalAddress) {
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
		if or.isSubscribed(valIdx) {
			or.AdvanceStateMachine(valIdx, Unsubscribe)
			or.IncreaseAllPendingRewards(or.state.Validators[valIdx].PendingRewardsWei)
			or.ResetPendingRewards(valIdx)
			or.state.Unsubscriptions = append(or.state.Unsubscriptions, unsub)
			log.WithFields(log.Fields{
				"BlockNumber":      unsub.Event.Raw.BlockNumber,
				"TxHash":           unsub.Event.Raw.TxHash,
				"ValidatorIndex":   valIdx,
				"WithdrawaAddress": withdrawalAddress,
				"Sender":           sender,
			}).Info("[Unsubscription] Validator unsubscribed ok")
			continue
		}

		if !or.isSubscribed(valIdx) {
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
// TODO: rename to handle autoSubscription. Passs v1.Validator Instead.
func (or *Oracle) AddSubscriptionIfNotAlready(valIndex uint64, WithdrawalAddress string, validatorKey string) {
	validator, found := or.state.Validators[valIndex]
	if !found {
		// If not found and not manually subscribed, we trigger the AutoSubscription event
		// Instantiate the validator
		validator = &ValidatorInfo{
			ValidatorStatus:       NotSubscribed,
			AccumulatedRewardsWei: big.NewInt(0),
			PendingRewardsWei:     big.NewInt(0),
			CollateralWei:         big.NewInt(0),
			WithdrawalAddress:     WithdrawalAddress,
			ValidatorIndex:        valIndex,
			ValidatorKey:          validatorKey,
		}
		or.state.Validators[valIndex] = validator

		// And update it state according to the event
		or.AdvanceStateMachine(valIndex, AutoSubscription)
	} else {
		// If we found the validator and is not subscribed, advance the state machine
		// Most likely it was subscribed before, then unsubscribed and now auto subscribes
		if !or.isSubscribed(valIndex) {
			or.AdvanceStateMachine(valIndex, AutoSubscription)
		}
	}
}

func (or *Oracle) ConsolidateBalance(valIndex uint64) {
	or.state.Validators[valIndex].AccumulatedRewardsWei.Add(or.state.Validators[valIndex].AccumulatedRewardsWei, or.state.Validators[valIndex].PendingRewardsWei)
	or.state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

func (or *Oracle) GetEligibleValidators() []uint64 {
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
func (or *Oracle) IncreaseAllPendingRewards(
	reward *big.Int) {

	eligibleValidators := or.GetEligibleValidators()
	numEligibleValidators := big.NewInt(int64(len(eligibleValidators)))

	if len(eligibleValidators) == 0 {
		log.Warn("No validators are eligible to receive rewards, pool fees address will receive all")
		or.state.PoolAccumulatedFees.Add(or.state.PoolAccumulatedFees, reward)
		return
	}

	if or.state.PoolFeesPercent > 100*100 {
		log.Fatal("Pool fees percent cannot be greater than 100% (10000) value: ", or.state.PoolFeesPercent)
	}

	// 100 is the % and the other 100 is because we use two decimals
	// eg 1000 is 10%
	// eg 50 is 0.5%
	over := big.NewInt(100 * 100)

	// The pool takes PoolFeesPercent cut of the rewards
	aux := big.NewInt(0).Mul(reward, big.NewInt(int64(or.state.PoolFeesPercent)))

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

func (or *Oracle) IncreaseValidatorPendingRewards(valIndex uint64, reward *big.Int) {
	or.state.Validators[valIndex].PendingRewardsWei.Add(or.state.Validators[valIndex].PendingRewardsWei, reward)
}

func (or *Oracle) IncreaseValidatorAccumulatedRewards(valIndex uint64, reward *big.Int) {
	or.state.Validators[valIndex].AccumulatedRewardsWei.Add(or.state.Validators[valIndex].AccumulatedRewardsWei, reward)
}

func (or *Oracle) SendRewardToPool(reward *big.Int) {
	or.state.PoolAccumulatedFees.Add(or.state.PoolAccumulatedFees, reward)
}

func (or *Oracle) ResetPendingRewards(valIndex uint64) {
	or.state.Validators[valIndex].PendingRewardsWei = big.NewInt(0)
}

// See the spec for state diagram with states and transitions. This tracks all the different
// states and state transitions that a given validator can have from the oracle point of view
func (or *Oracle) AdvanceStateMachine(valIndex uint64, event Event) {
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

func (or *Oracle) LogBalances() {
	for valIndex, validator := range or.state.Validators {
		log.WithFields(log.Fields{
			"LatestProcessedSlot": or.state.LatestProcessedSlot,
			"ValIndex":            valIndex,
			"PendingRewards":      validator.PendingRewardsWei,
			"AccumulatedRewards":  validator.AccumulatedRewardsWei,
		}).Info("Validator balances")
	}
}

// TODO: Remove this and get the merkle tree from somewhere else. See stored state
func (or *Oracle) GetMerkleRootIfAny() (string, bool) {
	mk := NewMerklelizer()
	// TODO: returning orderedRawLeafs as a quick workaround to get the proofs
	_, _, tree, enoughData := mk.GenerateTreeFromState(or.state)
	if !enoughData {
		return "", enoughData
	}
	merkleRootStr := "0x" + hex.EncodeToString(tree.Root)

	return merkleRootStr, true
}
