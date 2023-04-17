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
	"github.com/dappnode/mev-sp-oracle/postgres"

	api "github.com/attestantio/go-eth2-client/api/v1"
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
	Postgres        *postgres.Postgresql
	NumRetries      int
}

func NewOnchain(cfg config.Config) (*Onchain, error) {

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
		http.WithTimeout(60*time.Second),
		http.WithAddress(cfg.ConsensusEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		return nil, errors.New("Error dialing consensus client: " + err.Error())
	}
	consensusClient := client.(*http.Service)

	// Get deposit contract to ensure the endpoint is working
	depositContract, err := consensusClient.DepositContract(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching deposit contract from consensus client: " + err.Error())
	}
	log.Info("Connected succesfully to consensus client. Deposit contract: ", depositContract)

	if depositContract.ChainID != uint64(chainId.Int64()) {
		return nil, fmt.Errorf("ChainId from consensus and execution client do not match: %d vs %d", depositContract.ChainID, uint64(chainId.Int64()))
	}

	// Print sync status of consensus and execution client
	execSync, err := executionClient.SyncProgress(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching execution client sync progress: " + err.Error())
	}

	log.Info("Execution client sync state (nil is synced): ", execSync)

	consSync, err := consensusClient.NodeSyncing(context.Background())
	if err != nil {
		return nil, errors.New("Error fetching consensus client sync progress: " + err.Error())
	}

	log.Info("Consensus client sync state: ", consSync)

	// TODO: Get this from Config.
	// Instantiate the smoothing pool contract to run get/set operations on it
	address := common.HexToAddress(cfg.PoolAddress)
	contract, err := contract.NewContract(address, executionClient)
	if err != nil {
		return nil, errors.New("Error instantiating contract: " + err.Error())
	}

	postgres, err := postgres.New(cfg.PostgresEndpoint, cfg.NumRetries)
	if err != nil {
		return nil, errors.New("Error instantiating postgres: " + err.Error())
	}

	return &Onchain{
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
		Cfg:             &cfg,
		Contract:        contract,
		Postgres:        postgres,
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

// Given a validator key, returns the validator index
func (o *Onchain) GetValidatorIndexByKey(valKey string, opts ...retry.Option) (uint64, error) {
	var err error
	var validators map[phase0.ValidatorIndex]*api.Validator

	err = retry.Do(func() error {
		validators, err = o.ConsensusClient.ValidatorsByPubKey(context.Background(), "finalized", []phase0.BLSPubKey{StringToBlsKey(valKey)})
		if err != nil {
			log.Warn("Failed attempt to fetch validator index: ", err.Error(), " Retrying...")
			return errors.New("Error fetching validator index: " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return 0, errors.New("Could not fetch validator index: " + err.Error())
	}

	// A bit convoluted, refactor
	for _, v := range validators {
		recValKey := v.Validator.PublicKey.String()
		if recValKey == valKey {
			return uint64(v.Index), nil
		}
	}
	return 0, errors.New("Could not fetch validator index:")
}

// Given a validator index, returns the validator key
func (o *Onchain) GetValidatorKeyByIndex(valIndex uint64, opts ...retry.Option) (string, error) {
	var err error
	var validators map[phase0.ValidatorIndex]*api.Validator

	err = retry.Do(func() error {
		validators, err = o.ConsensusClient.Validators(context.Background(), "finalized", []phase0.ValidatorIndex{phase0.ValidatorIndex(valIndex)})
		if err != nil {
			log.Warn("Failed attempt to fetch validator key: ", err.Error(), " Retrying...")
			return errors.New("Error fetching validator index: " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return "", errors.New("Could not fetch validator index: " + err.Error())
	}

	validator, ok := validators[phase0.ValidatorIndex(valIndex)]
	if !ok {
		return "", errors.New("Could not fetch validator index:")
	}
	return validator.Validator.PublicKey.String(), nil
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
	var itr *contract.ContractSuscribeValidatorIterator

	err = retry.Do(func() error {
		// Note that this event can be both donations and mev rewards
		itr, err = o.Contract.FilterSuscribeValidator(filterOpts)
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
		event := itr.Event

		// And add some extra data to the return subscription struct
		valKey, err := o.GetValidatorKeyByIndex(uint64(event.ValidatorID))
		if err != nil {
			return nil, errors.New("could not get validator key: " + err.Error())
		}
		depositAddress := o.GetDepositAddressOfValidator(valKey, uint64(event.ValidatorID))

		blockSubscriptions = append(blockSubscriptions, Subscription{
			ValidatorIndex: uint64(event.ValidatorID),
			ValidatorKey:   valKey,
			Collateral:     event.SuscriptionCollateral,
			BlockNumber:    blockNumber,
			TxHash:         event.Raw.TxHash.Hex(),
			DepositAddress: depositAddress,
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
	var itr *contract.ContractUnsuscribeValidatorIterator

	err = retry.Do(func() error {
		// Note that this event can be both donations and mev rewards
		itr, err = o.Contract.FilterUnsuscribeValidator(filterOpts)
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
		event := itr.Event

		// Fetch also extra data
		valKey, err := o.GetValidatorKeyByIndex(uint64(event.ValidatorID))
		if err != nil {
			return nil, errors.New("could not get validator key: " + err.Error())
		}
		depositAddress := o.GetDepositAddressOfValidator(valKey, uint64(event.ValidatorID))

		blockUnsubscriptions = append(blockUnsubscriptions, Unsubscription{
			ValidatorIndex: uint64(event.ValidatorID),
			Sender:         event.Sender.String(),
			BlockNumber:    blockNumber,
			TxHash:         event.Raw.TxHash.Hex(),
			ValidatorKey:   valKey,
			DepositAddress: depositAddress,
		})
	}
	err = itr.Close()
	if err != nil {
		log.Fatal("could not close iterator for new donation events", err)
	}
	return blockUnsubscriptions, nil
}

func (o *Onchain) GetContractCollateral(opts ...retry.Option) (*big.Int, error) {
	subscriptionCollateral := new(big.Int)
	err := retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			var err error
			subscriptionCollateral, err = o.Contract.SuscriptionCollateral(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get subscription collateral from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get subscription collateral from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return big.NewInt(0), errors.New("could not get subscription collateral from contract: " + err.Error())
	}
	return subscriptionCollateral, nil
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
func (o *Onchain) GetContractClaimedBalance(depositAddress string, opts ...retry.Option) (*big.Int, error) {
	var claimedBalance *big.Int
	var err error

	if !common.IsHexAddress(depositAddress) {
		log.Fatal("Invalid deposit address: ", depositAddress)
	}

	hexDepAddres := common.HexToAddress(depositAddress)

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
			poolBlock.DepositAddress = o.GetDepositAddressOfValidator(valPublicKey, slot)
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

	// TODO: Extract some of these things out of the function
	// Load private key signing the tx. This address must hold enough Eth
	// to pay for the tx fees, otherwise it will fail
	privateKey, err := crypto.HexToECDSA(o.Cfg.DeployerPrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
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

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chaindId)
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

	auth.Context = context.Background()
	auth.NoSend = false

	//address := common.HexToAddress(o.cfg.PoolAddress)
	// TODO: hardcoding a different address for testing
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

	// Leave 5 minutes for the tx to be validated
	deadline := time.Now().Add(5 * time.Minute)
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

// TODO: Remove the slot from the input, makes no sense
// TODO: Do not go to mainnet with this, only for goerli since some validators
// dont have a deposit address
func (o *Onchain) GetDepositAddressOfValidator(validatorPubKey string, slot uint64) string {
	depositAddress, err := o.Postgres.GetDepositAddressOfValidatorKey(validatorPubKey)
	if err == nil {
		return depositAddress
	}
	log.Warn("Deposit key not found for ", validatorPubKey, ". Expected in goerli. Using a default one. err: ", err)

	// TODO: Remove this in production. Used in goerli for testing with differenet addresses
	// TODO: Dont go to mainnet with this. If there is a bug in the code, we will be using
	// and invalid address. In mainnet, fail if we cant find the deposit address.
	someDepositAddresses := []string{
		"0x001eDa52592fE2f8a28dA25E8033C263744b1b6E",
		"0x0029a125E6A3f058628Bd619C91f481e4470D673",
		"0x003718fb88964A1F167eCf205c7f04B25FF46B8E",
		"0x004b1EaBc3ea60331a01fFfC3D63E5F6B3aB88B3",
		"0x005CD1608e40d1e775a97d12e4f594029567C071",
		"0x0069c9017BDd6753467c138449eF98320be1a4E4",
		"0x007cF0936ACa64Ef22C0019A616801Bec7FCCECF",
	}
	//Just pick a "random" one to not always the same
	return someDepositAddresses[slot%7]
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
