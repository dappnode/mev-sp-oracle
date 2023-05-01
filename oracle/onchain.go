package oracle

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/contract"

	api "github.com/attestantio/go-eth2-client/api/v1"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
)

// This file provides different functions to access the blockchain state from both consensus and
// execution layer and modifying the its state via smart contract calls.
type EpochDuties struct {
	Epoch  uint64
	Duties []*api.ProposerDuty
}

// Simple cache storing epoch -> proposer duties
// This is useful to not query the beacon node for each slot
// since ProposerDuties returns the duties for the whole epoch
// Note that the cache is meant to store only one epoch's duties
var ProposalDutyCache EpochDuties

type Onchain struct {
	ConsensusClient *http.Service
	ExecutionClient *ethclient.Client
	Cfg             *config.Config
	Contract        *contract.Contract
	NumRetries      int
	updaterKey      *ecdsa.PrivateKey
	validators      map[phase0.ValidatorIndex]*v1.Validator
}

func NewOnchain(cfg *config.Config, updaterKey *ecdsa.PrivateKey) (*Onchain, error) {

	// Dial the execution client
	executionClient, err := ethclient.Dial(cfg.ExecutionEndpoint)
	if err != nil {
		return nil, errors.New("Error dialing execution client: " + err.Error())
	}

	// Get chainid to ensure the endpoint is working
	chainId, err := executionClient.ChainID(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching chainId from execution client: " + err.Error())
	}
	log.Info("Connected succesfully to execution client. ChainId: ", chainId)

	// Dial the consensus client
	client, err := http.New(context.Background(),
		http.WithTimeout(120*time.Second),
		http.WithAddress(cfg.ConsensusEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		return nil, errors.New("Error dialing consensus client. " + err.Error())
	}
	consensusClient := client.(*http.Service)

	// Get deposit contract to ensure the endpoint is working
	depositContract, err := consensusClient.DepositContract(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching deposit contract from consensus client: " + err.Error())
	}
	log.Info("Connected succesfully to consensus client. ChainId: ", depositContract.ChainID,
		" DepositContract: ", hex.EncodeToString(depositContract.Address[:]))

	if depositContract.ChainID != uint64(chainId.Int64()) {
		return nil, fmt.Errorf("ChainId from consensus and execution client do not match: %d vs %d", depositContract.ChainID, uint64(chainId.Int64()))
	}

	// Print sync status of consensus and execution client
	execSync, err := executionClient.SyncProgress(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching execution client sync progress: " + err.Error())
	}

	// nil means synced
	if execSync == nil {
		header, err := executionClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return nil, errors.New("Error fetching execution client sync progress: " + err.Error())
		}
		log.Info("Execution client is in sync, block number: ", header.Number)
	} else {
		log.Info("Execution client is NOT in sync, current block: ", execSync.CurrentBlock)
	}

	consSync, err := consensusClient.NodeSyncing(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching consensus client sync progress: " + err.Error())
	}

	if consSync.SyncDistance == 0 {
		log.Info("Consensus client is in sync, head slot: ", consSync.HeadSlot)
	} else {
		log.Info("Consensus client is NOT in sync, slots behind: ", consSync.SyncDistance)
	}

	// TODO: Get this from Config.
	// Instantiate the smoothing pool contract to run get/set operations on it
	address := common.HexToAddress(cfg.PoolAddress)
	contract, err := contract.NewContract(address, executionClient)
	if err != nil {
		return nil, errors.New("Error instantiating contract: " + err.Error())
	}

	return &Onchain{
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
		Cfg:             cfg,
		Contract:        contract,
		updaterKey:      updaterKey,
	}, nil
}

func (o *Onchain) AreNodesInSync(opts ...retry.Option) (bool, error) {
	var err error
	var execSync *ethereum.SyncProgress
	var consSync *api.SyncState

	err = retry.Do(func() error {
		execSync, err = o.ExecutionClient.SyncProgress(context.Background())
		if err != nil {
			log.Warn("Failed attempt to fetch execution client sync progress: ", err.Error(), " Retrying...")
			return errors.New("Error fetching execution client sync progress: " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return false, errors.New("Could not fetch execution client sync progress: " + err.Error())
	}

	err = retry.Do(func() error {
		consSync, err = o.ConsensusClient.NodeSyncing(context.Background())
		if err != nil {
			log.Warn("Failed attempt to fetch consensus client sync progress: ", err.Error(), " Retrying...")
			return errors.New("Error fetching execution client sync progress: " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return false, errors.New("Could not fetch consensus client sync progress: " + err.Error())
	}

	// Exeuction client returns nil if not syncing (in sync)
	// Give couple of slots to consensus client
	if execSync == nil && (consSync.SyncDistance < 2) {
		return true, nil
	}
	return false, nil
}

func (o *Onchain) GetConsensusBlockAtSlot(slot uint64, opts ...retry.Option) (*spec.VersionedSignedBeaconBlock, error) {
	slotStr := strconv.FormatUint(slot, 10)
	var signedBeaconBlock *spec.VersionedSignedBeaconBlock
	var err error

	err = retry.Do(func() error {
		signedBeaconBlock, err = o.ConsensusClient.SignedBeaconBlock(context.Background(), slotStr)
		if err != nil {
			log.Warn("Failed attempt to fetch block at slot ", slotStr, ": ", err.Error(), " Retrying...")
			return errors.New("Error fetching block at slot " + slotStr + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Could not fetch block at slot " + slotStr + ": " + err.Error())
	}
	return signedBeaconBlock, err
}

func (o *Onchain) GetFinalizedValidators(opts ...retry.Option) (map[phase0.ValidatorIndex]*api.Validator, error) {
	var validators map[phase0.ValidatorIndex]*api.Validator
	var err error

	err = retry.Do(func() error {
		validators, err = o.ConsensusClient.Validators(context.Background(), "finalized", nil)
		if err != nil {
			log.Warn("Failed attempt to fetch finalized validators: ", err.Error(), " Retrying...")
			return errors.New("Error fetching finalized validators: " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Could not fetch finalized validators: " + err.Error())
	}
	return validators, err
}

func (o *Onchain) GetProposalDuty(slot uint64, opts ...retry.Option) (*api.ProposerDuty, error) {
	// Hardcoded value, slots in an epoch
	slotsInEpoch := uint64(32)
	epoch := slot / slotsInEpoch
	slotWithinEpoch := slot % slotsInEpoch
	slotStr := strconv.FormatUint(slot, 10)

	// If cache hit, return the result
	if ProposalDutyCache.Epoch == epoch {
		// Sanity check that should never happen
		if ProposalDutyCache.Epoch != uint64(ProposalDutyCache.Duties[slotWithinEpoch].Slot/phase0.Slot(slotsInEpoch)) {
			return nil, errors.New("Proposal duty epoch does not match when converting slot to epoch")
		}
		return ProposalDutyCache.Duties[slotWithinEpoch], nil
	}

	// Empty indexes to force fetching all duties
	indexes := make([]phase0.ValidatorIndex, 0)
	var duties []*api.ProposerDuty
	var err error

	err = retry.Do(func() error {
		duties, err = o.ConsensusClient.ProposerDuties(
			context.Background(), phase0.Epoch(epoch), indexes)
		if err != nil {
			log.Warn("Failed attempt to fetch proposal duties at slot ", slotStr, ": ", err.Error(), " Retrying...")
			return errors.New("Error fetching proposal duties at slot " + slotStr + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Error fetching proposal duties at slot " + slotStr + ": " + err.Error())
	}

	// If success, store result in cache
	ProposalDutyCache = EpochDuties{epoch, duties}

	return duties[slotWithinEpoch], nil
}

// This function is expensive as gets every tx receipt from the block. Use only if needed
func (o *Onchain) GetExecHeaderAndReceipts(
	blockNumber *big.Int,
	rawTxs []bellatrix.Transaction,
	opts ...retry.Option) (*types.Header, []*types.Receipt, error) {

	var header *types.Header
	var err error

	err = retry.Do(func() error {
		header, err = o.ExecutionClient.HeaderByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Warn("Failed attempt to fetch header for block ", blockNumber.String(), ": ", err.Error(), " Retrying...")
			return errors.New("Error fetching header for block " + blockNumber.String() + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, nil, errors.New("Could not fetch header for block " + blockNumber.String() + ": " + err.Error())
	}

	var receipts []*types.Receipt
	for _, rawTx := range rawTxs {
		// This should never happen
		tx, _, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal(err)
		}
		var receipt *types.Receipt

		err = retry.Do(func() error {
			receipt, err = o.ExecutionClient.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				log.Warn("Failed attempt to fetch receipt for tx ", tx.Hash().String(), ": ", err.Error(), " Retrying...")
				return errors.New("Error fetching receipt for tx " + tx.Hash().String() + ": " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

		if err != nil {
			return nil, nil, errors.New("Could not fetch receipt for tx " + tx.Hash().String() + ": " + err.Error())
		}
		receipts = append(receipts, receipt)
	}
	return header, receipts, nil
}

// TODO: Rethink this function. Its not just donations but eth rx to the contract
// in general
// TODO:? Unused?
func (o *Onchain) GetDonationEvents(blockNumber uint64, opts ...retry.Option) ([]Donation, error) {
	log.Fatal("This function is deprecated. Use GetDonations instead")
	startBlock := uint64(blockNumber)
	endBlock := uint64(blockNumber)

	// Not the most effective way, but we just need to advance one by one.
	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}

	var err error
	var itr *contract.ContractEtherReceivedIterator

	err = retry.Do(func() error {
		// Note that this event can be both donations and mev rewards
		itr, err = o.Contract.FilterEtherReceived(filterOpts)
		if err != nil {
			log.Warn("Failed attempt to filter donations for block ", strconv.FormatUint(blockNumber, 10), ": ", err.Error(), " Retrying...")
			return errors.New("Error filtering donations for block " + strconv.FormatUint(blockNumber, 10) + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Could not filter donations for block " + strconv.FormatUint(blockNumber, 10) + ": " + err.Error())
	}

	// Loop over all found events
	donations := make([]Donation, 0)
	for itr.Next() {
		event := itr.Event

		log.WithFields(log.Fields{
			"RewardWei":   event.DonationAmount,
			"BlockNumber": event.Raw.BlockNumber,
			"Type":        "Donation",
			"TxHash":      event.Raw.TxHash.Hex()[0:8],
		}).Info("New Reward")

		donations = append(donations, Donation{
			AmountWei: event.DonationAmount,
			Block:     blockNumber,
			TxHash:    event.Raw.TxHash.Hex(),
		})
	}
	err = itr.Close()
	if err != nil {
		log.Fatal("could not close iterator for new donation events", err)
	}
	return donations, nil
}

func (o *Onchain) GetBlockSubscriptions(blockNumber uint64, opts ...retry.Option) ([]Subscription, error) {
	startBlock := uint64(blockNumber)
	endBlock := uint64(blockNumber)

	// TODO: Consider
	// Not the most effective way, but we just need to advance one by one.
	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}

	var err error
	var itr *contract.ContractSubscribeValidatorIterator

	err = retry.Do(func() error {
		// Note that this event can be both donations and mev rewards
		itr, err = o.Contract.FilterSubscribeValidator(filterOpts)
		if err != nil {
			log.Warn("Failed attempt to filter subscriptions for block ", strconv.FormatUint(blockNumber, 10), ": ", err.Error(), " Retrying...")
			return errors.New("Error getting validator subscriptions for block " + strconv.FormatUint(blockNumber, 10) + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Error getting validator subscriptions for block " + strconv.FormatUint(blockNumber, 10) + ": " + err.Error())
	}

	// Loop over all found events
	blockSubscriptions := make([]Subscription, 0)
	for itr.Next() {
		blockSubscriptions = append(blockSubscriptions, Subscription{
			Event:     itr.Event,
			Validator: o.validators[phase0.ValidatorIndex(itr.Event.ValidatorID)],
		})
	}
	err = itr.Close()
	if err != nil {
		log.Fatal("could not close iterator for new donation events", err)
	}
	return blockSubscriptions, nil
}

func (o *Onchain) GetBlockUnsubscriptions(blockNumber uint64, opts ...retry.Option) ([]Unsubscription, error) {
	startBlock := uint64(blockNumber)
	endBlock := uint64(blockNumber)

	// Not the most effective way, but we just need to advance one by one.
	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}

	var err error
	var itr *contract.ContractUnsubscribeValidatorIterator

	err = retry.Do(func() error {
		// Note that this event can be both donations and mev rewards
		itr, err = o.Contract.FilterUnsubscribeValidator(filterOpts)
		if err != nil {
			log.Warn("Failed attempt to filter unsubscriptions for block ", strconv.FormatUint(blockNumber, 10), ": ", err.Error(), " Retrying...")
			return errors.New("Error getting validator unsubscriptions for block " + strconv.FormatUint(blockNumber, 10) + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Error getting validator unsubscriptions for block " + strconv.FormatUint(blockNumber, 10) + ": " + err.Error())
	}

	// Loop over all found events
	blockUnsubscriptions := make([]Unsubscription, 0)
	for itr.Next() {
		blockUnsubscriptions = append(blockUnsubscriptions, Unsubscription{
			Event:     itr.Event,
			Validator: o.validators[phase0.ValidatorIndex(itr.Event.ValidatorID)],
		})
	}
	err = itr.Close()
	if err != nil {
		log.Fatal("could not close iterator for new donation events", err)
	}
	return blockUnsubscriptions, nil
}

func (o *Onchain) GetContractMerkleRoot(opts ...retry.Option) (string, error) {
	var rewardsRootStr string

	// Retries multiple times before errorings
	err := retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			rewardsRoot, err := o.Contract.RewardsRoot(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get merkle root from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get rewards root from contract: " + err.Error())
			}
			rewardsRootStr = "0x" + hex.EncodeToString(rewardsRoot[:])
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return "", errors.New("could not get merkle root from contract: " + err.Error())
	}

	return rewardsRootStr, nil
}

// TODO: Only in finalized slots!
func (o *Onchain) GetContractClaimedBalance(WithdrawalAddress string, opts ...retry.Option) (*big.Int, error) {
	var claimedBalance *big.Int
	var err error

	if !common.IsHexAddress(WithdrawalAddress) {
		log.Fatal("Invalid withdrawal address: ", WithdrawalAddress)
	}

	hexDepAddres := common.HexToAddress(WithdrawalAddress)

	// Retries multiple times before errorings
	err = retry.Do(
		func() error {
			// TODO: This should be performed in the last finalized slot for consistency
			// Otherwise our local view and remote view can be different. See if it applies to other functions, like merkle tree.
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			claimedBalance, err = o.Contract.ClaimedBalance(callOpts, hexDepAddres)
			if err != nil {
				log.Warn("Failed attempt to get claimed balance from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get claimed balance from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return big.NewInt(0), errors.New("could not get claimed balance from contract: " + err.Error())
	}

	return claimedBalance, nil
}

func (o *Onchain) GetEthBalance(address string, opts ...retry.Option) (*big.Int, error) {
	account := common.HexToAddress(address)
	var err error
	var balanceWei *big.Int

	err = retry.Do(func() error {
		balanceWei, err = o.ExecutionClient.BalanceAt(context.Background(), account, nil)
		if err != nil {
			log.Warn("Failed attempt to get balance for address ", address, ": ", err.Error(), " Retrying...")
			return errors.New("could not get balance for address " + address + ": " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("could not get balance for address " + address + ": " + err.Error())
	}

	return balanceWei, nil
}

// Given a slot number, this function fetches all the information that the oracle need to process
// the block (if not missed) that was proposed in this slot.
func (o *Onchain) GetAllBlockInfo(slot uint64) (Block, []Subscription, []Unsubscription, []Donation) {

	// Get who should propose the block
	slotDuty, err := o.GetProposalDuty(slot)
	if err != nil {
		log.Fatal("could not get proposal duty: ", err)
	}

	if uint64(slotDuty.Slot) != slot {
		log.Fatal("slot duty slot does not match requested slot: ", slotDuty.Slot, " vs ", slot)
	}

	// The validator that should propose the block
	valPublicKey := strings.ToLower("0x" + hex.EncodeToString(slotDuty.PubKey[:]))

	proposedBlock, err := o.GetConsensusBlockAtSlot(slot)
	if err != nil {
		log.Fatal("could not get block at slot:", err)
	}

	// Only populated if a valid block was proposed
	var extendedBlock *VersionedSignedBeaconBlock

	// Init pool block, with relevant information to the pool
	poolBlock := Block{
		Slot:           uint64(slotDuty.Slot),
		ValidatorIndex: uint64(slotDuty.ValidatorIndex),
		ValidatorKey:   valPublicKey,
	}

	// Fetch block info
	if proposedBlock == nil {
		// A nil block means that the validator did not propose a block (missed proposal)
		poolBlock.BlockType = MissedProposal

		// Return early, a missed block wont contain any information
		return poolBlock, []Subscription{}, []Unsubscription{}, []Donation{}

	} else {
		// Cast the block to our extended version with utils functions
		extendedBlock = &VersionedSignedBeaconBlock{proposedBlock}
		// If the proposal was succesfull, we check if this block contained a reward for the pool
		reward, correctFeeRec, rewardType, err := extendedBlock.GetSentRewardAndType(o.Cfg.PoolAddress, *o)
		if err != nil {
			log.Fatal("could not get reward and type: ", err)
		}

		// We populate the parameters of the pool block
		poolBlock.Reward = reward
		poolBlock.RewardType = rewardType

		// And check if it contained a reward for the pool or not
		if correctFeeRec {
			poolBlock.BlockType = OkPoolProposal
			withdrawalAddress, err := GetEth1AddressByte(o.validators[slotDuty.ValidatorIndex].Validator.WithdrawalCredentials)
			if err != nil {
				poolBlock.BlockType = OkPoolProposalBlsKeys
			} else {
				poolBlock.BlockType = OkPoolProposal
				poolBlock.WithdrawalAddress = withdrawalAddress
			}
		} else {
			poolBlock.BlockType = WrongFeeRecipient
		}
	}

	// Fetch subscription data
	blockSubs, err := o.GetBlockSubscriptions(extendedBlock.GetBlockNumber())
	if err != nil {
		log.Fatal("could not get block subscriptions: ", err)
	}

	// Fetch unsubscription data
	blockUnsubs, err := o.GetBlockUnsubscriptions(extendedBlock.GetBlockNumber())
	if err != nil {
		log.Fatal("could not get block unsubscriptions: ", err)
	}

	// TODO: This is wrong, as this event will also be triggered when a validator proposes a MEV block
	// Fetch donations in this block
	//blockDonations, err := o.GetDonationEvents(extendedBlock.GetBlockNumber())
	//if err != nil {
	//	log.Fatal("could not get block donations: ", err)
	//}

	blockDonations := extendedBlock.GetDonations(o.Cfg.PoolAddress)

	return poolBlock, blockSubs, blockUnsubs, blockDonations
}

func (o *Onchain) UpdateContractMerkleRoot(newMerkleRoot string) string {

	// Parse merkle root to byte array
	newMerkleRootBytes := [32]byte{}
	unboundedBytes := common.Hex2Bytes(newMerkleRoot)

	if len(unboundedBytes) != 32 {
		log.Fatal("wrong merkle root length: ", newMerkleRoot)
	}
	copy(newMerkleRootBytes[:], common.Hex2Bytes(newMerkleRoot))

	// Sanity check to ensure the converted tree matches the original
	if hex.EncodeToString(newMerkleRootBytes[:]) != newMerkleRoot {
		log.Fatal("merkle trees dont match, expected: ", newMerkleRoot)
	}

	publicKey := o.updaterKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Println(fromAddress.Hex())
	nonce, err := o.ExecutionClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("could not get pending nonce: ", err)
	}

	// Unused, leaving for reference. We rely on automatic gas estimation, see below (nil values)
	gasTipCap, err := o.ExecutionClient.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal("could not get gas price suggestion: ", err)
	}
	_ = gasTipCap

	chaindId, err := o.ExecutionClient.NetworkID(context.Background())
	if err != nil {
		log.Fatal("could not get chaind: ", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(o.updaterKey, chaindId)
	if err != nil {
		log.Fatal("could not create NewKeyedTransactorWithChainID:", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// Important that the value is 0. Otherwise we would be sending Eth
	// and thats not neccessary.
	auth.Value = big.NewInt(0)

	// nil prices automatically estimate prices
	// TODO: Perhaps overpay to make sure the tx is not stuck forever.
	auth.GasPrice = nil
	auth.GasFeeCap = nil
	auth.GasTipCap = nil
	auth.NoSend = false
	auth.Context = context.Background()

	address := common.HexToAddress(o.Cfg.PoolAddress)

	instance, err := contract.NewContract(address, o.ExecutionClient)
	if err != nil {
		log.Fatal(err)
	}

	// Create a tx calling the update rewards root function with the new merkle root
	tx, err := instance.UpdateRewardsRoot(auth, newMerkleRootBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.WithFields(log.Fields{
		"TxHash":        tx.Hash().Hex(),
		"NewMerkleRoot": newMerkleRoot,
	}).Info("Tx sent to Ethereum updating rewards merkle root, wait to be validated")

	// Leave 15 minutes for the tx to be validated
	deadline := time.Now().Add(15 * time.Minute)
	ctx, cancelCtx := context.WithDeadline(context.Background(), deadline)
	defer cancelCtx()

	// It stops waiting when the context is canceled.
	receipt, err := bind.WaitMined(ctx, o.ExecutionClient, tx)
	if ctx.Err() != nil {
		log.Fatal("Timeout expired for waiting for tx to be validated, txHash: ", tx.Hash().Hex(), " err:", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatal("Tx failed, err: ", receipt.Status, " hash: ", tx.Hash().Hex())
	}

	// Tx was sent and validated correctly, print receipt info
	log.WithFields(log.Fields{
		"Status":            receipt.Status,
		"CumulativeGasUsed": receipt.CumulativeGasUsed,
		"TxHash":            receipt.TxHash,
		"GasUsed":           receipt.GasUsed,
		"BlockHash":         receipt.BlockHash.Hex(),
		"BlockNumber":       receipt.BlockNumber,
	}).Info("Tx: ", tx.Hash().Hex(), " was validated ok. Receipt info:")

	return tx.Hash().Hex()
}

// Loads all validator from the beacon chain into the oracle, must be called periodically
func (o *Onchain) RefreshBeaconValidators() {
	log.Info("Loading existing validators in the beacon chain")
	vals, err := o.GetFinalizedValidators()
	if err != nil {
		log.Fatal("Could not get validators: ", err)
	}
	o.validators = vals
	log.Info("Done loading existing validators in the beacon chain total: ", len(vals))
}

func (o *Onchain) Validators() map[phase0.ValidatorIndex]*v1.Validator {
	return o.validators
}

func (o *Onchain) GetRetryOpts(opts []retry.Option) []retry.Option {
	// Default retry options. This specifies what to do when a call to the
	// consensus or execution client fails. Default is to retry x times (see config)
	// with a 15 seconds delay and the default backoff strategy (see avas/retry-go)
	// Note that in some cases we might want to avoid retrying at all, for example
	// when serving data to an api, we may want to just fail fast and return an error
	// If this function is called with retry options, we use those instead as a way
	// to override the default retry options
	if len(opts) == 0 {
		return []retry.Option{
			retry.Attempts(uint(o.Cfg.NumRetries)),
			retry.Delay(15 * time.Second),
		}
	} else {
		return opts
	}
}
