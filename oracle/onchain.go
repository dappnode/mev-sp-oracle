package oracle

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

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
// TODO: Move cache to onchain struct + test it
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
	CliCfg          *config.CliConfig // TODO:  remove?
	Contract        *contract.Contract
	NumRetries      int
	updaterKey      *ecdsa.PrivateKey
	validators      map[phase0.ValidatorIndex]*v1.Validator

	etherReceivedCache []*contract.ContractEtherReceived // TODO: unused
}

func NewOnchain(cliCfg *config.CliConfig, updaterKey *ecdsa.PrivateKey) (*Onchain, error) {

	// Dial the execution client
	executionClient, err := ethclient.Dial(cliCfg.ExecutionEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "Error dialing execution client")
	}

	// Get chainid to ensure the endpoint is working
	chainId, err := executionClient.ChainID(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching chainid from execution client")
	}
	log.Info("Connected succesfully to execution client. ChainId: ", chainId)

	// Dial the consensus client
	client, err := http.New(context.Background(),
		http.WithTimeout(120*time.Second),
		http.WithAddress(cliCfg.ConsensusEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error dialing consensus client")
	}
	consensusClient := client.(*http.Service)

	// Get deposit contract to ensure the endpoint is working
	depositContract, err := consensusClient.DepositContract(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching deposit contract from consensus client")
	}
	log.Info("Connected succesfully to consensus client. ChainId: ", depositContract.ChainID,
		" DepositContract: ", "0x"+hex.EncodeToString(depositContract.Address[:]))

	if depositContract.ChainID != uint64(chainId.Int64()) {
		return nil, errors.Wrap(err, fmt.Sprintf("Consensus and execution clients are not connected to the same chain %d vs %d",
			depositContract.ChainID, chainId))
	}

	// Print sync status of consensus and execution client
	execSync, err := executionClient.SyncProgress(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching execution client sync progress")
	}

	// nil means synced
	if execSync == nil {
		header, err := executionClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return nil, errors.Wrap(err, "Error fetching execution client header")
		}
		log.Info("Execution client is in sync, block number: ", header.Number)
	} else {
		log.Info("Execution client is NOT in sync, current block: ", execSync.CurrentBlock)
	}

	consSync, err := consensusClient.NodeSyncing(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching consensus client sync progress")
	}

	if consSync.SyncDistance == 0 {
		log.Info("Consensus client is in sync, head slot: ", consSync.HeadSlot)
	} else {
		log.Info("Consensus client is NOT in sync, slots behind: ", consSync.SyncDistance)
	}

	// Instantiate the smoothing pool contract to run get/set operations on it
	address := common.HexToAddress(cliCfg.PoolAddress)
	contract, err := contract.NewContract(address, executionClient)
	if err != nil {
		return nil, errors.Wrap(err, "Error instantiating contract")
	}

	return &Onchain{
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
		CliCfg:          cliCfg,
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

func (o *Onchain) BlockByNumber(blockNumber *big.Int, opts ...retry.Option) (*types.Block, error) {
	var err error
	var block *types.Block

	err = retry.Do(func() error {
		block, err = o.ExecutionClient.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Warn("Failed attempt to fetch block by number: ", err.Error(), " Retrying...")
			return errors.New("Error fetching block by number: " + err.Error())
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("Could not fetch block by number: " + err.Error())
	}
	return block, err
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

func (o *Onchain) IsAddressWhitelisted(
	deployedBlock uint64,
	address string,
	opts ...retry.Option) (bool, error) {

	var err error

	latestBlock, err := o.ExecutionClient.BlockNumber(context.Background())
	if err != nil {
		return false, errors.New("Error getting latest block number: " + err.Error())
	}

	if deployedBlock > latestBlock {
		return false, errors.New(fmt.Sprintf("Deployed block is higher than latest block: %d > %d",
			deployedBlock, latestBlock))
	}

	// How many blocks to check at once. A very high value can choke the node
	// Around 10k to 30k should be a reasonable value. 30k is around 4 days of
	// events in one call.
	chunkSize := uint64(30000)

	// Listen for even since the deployed block till the latest block in
	// increments of chunkSize
	for start := deployedBlock; start < latestBlock; start += chunkSize {
		end := start + chunkSize - 1

		if end > latestBlock {
			end = latestBlock
		}

		log.Info("Checking whitelist events from block ", start, " to ", end)

		filterOpts := &bind.FilterOpts{Context: context.Background(), Start: start, End: &end}

		var itr *contract.ContractAddOracleMemberIterator

		err = retry.Do(func() error {
			itr, err = o.Contract.FilterAddOracleMember(filterOpts)
			if err != nil {
				log.Warn("Failed attempt to filter AddOracleMember event. Retrying...")
				return errors.New("Failed attempt to filter AddOracleMember event. Retrying...")
			}
			return nil
		}, o.GetRetryOpts(opts)...)

		if err != nil {
			return false, errors.New("Error getting AddOracleMember events")
		}

		// Loop over all found events
		for itr.Next() {
			newOracleMember := itr.Event.NewOracleMember.String()

			// If we found an event with the address we are looking for, return true
			// as it means the address is whitelisted
			if Equals(address, newOracleMember) {
				log.WithFields(log.Fields{
					"TxHash":          itr.Event.Raw.TxHash.String(),
					"NewOracleMember": itr.Event.NewOracleMember.String(),
				}).Info("Detected AddOracleMember with selected account")
				return true, nil
			}
		}
		err = itr.Close()
		if err != nil {
			log.Fatal("could not close iterator for new donation events", err)
		}
	}
	return false, nil
}

func (o *Onchain) GetContractCollateral(opts ...retry.Option) (*big.Int, error) {
	subscriptionCollateral := new(big.Int)
	err := retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			var err error
			subscriptionCollateral, err = o.Contract.SubscriptionCollateral(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get subscription collateral from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get subscription collateral from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("could not get subscription collateral from contract: " + err.Error())
	}
	return subscriptionCollateral, nil
}

func (o *Onchain) GetSlotCheckpointSize(opts ...retry.Option) (uint64, error) {
	var slotCheckpointSize uint64
	var err error

	err = retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			slotCheckpointSize, err = o.Contract.CheckpointSlotSize(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get slot checkpoint size from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get slot checkpoint size from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return 0, errors.New("could not get claimed balance from contract: " + err.Error())
	}

	return slotCheckpointSize, nil
}

func (o *Onchain) GetContractDeploymentBlock(opts ...retry.Option) (*big.Int, error) {
	var deploymentBlock *big.Int
	var err error

	err = retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			deploymentBlock, err = o.Contract.DeploymentBlockNumber(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get deployment block from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get deployment block from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("could not get deployment block from contract: " + err.Error())
	}

	return deploymentBlock, nil
}

func (o *Onchain) GetPoolFee(opts ...retry.Option) (*big.Int, error) {
	var poolFee *big.Int
	var err error

	err = retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			poolFee, err = o.Contract.PoolFee(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get pool fee from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get pool fee from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.New("could not get pool fee from contract: " + err.Error())
	}

	return poolFee, nil
}

func (o *Onchain) GetPoolFeeAddress(opts ...retry.Option) (string, error) {
	var poolFeeAddress common.Address
	var err error

	err = retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			poolFeeAddress, err = o.Contract.PoolFeeRecipient(callOpts)
			if err != nil {
				log.Warn("Failed attempt to get pool fee address from contract: ", err.Error(), " Retrying...")
				return errors.New("could not get pool fee address from contract: " + err.Error())
			}
			return nil
		}, o.GetRetryOpts(opts)...)

	if err != nil {
		return "", errors.New("could not get pool fee address from contract: " + err.Error())
	}

	return poolFeeAddress.Hex(), nil
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

// TODO: Document logic. events are fetched for every single block.
// some rewards only if subscriber of pool or reward went to pool.
// .. etc
func (o *Onchain) FetchFullBlock(slot uint64, oracle *Oracle) *FullBlock {

	// Get who should propose the block
	slotDuty, err := o.GetProposalDuty(slot)
	if err != nil {
		log.Fatal("could not get proposal duty: ", err)
	}

	// Sanity check to ensure the slot duty is the one we requested
	if uint64(slotDuty.Slot) != slot {
		log.Fatal("slot duty slot does not match requested slot: ", slotDuty.Slot, " vs ", slot)
	}

	// Create the full block with the duty, which is the minimum info it can have
	fullBlock := NewFullBlock(slotDuty, o.Validators()[slotDuty.ValidatorIndex])

	// Fetch the whole consensus block
	proposedBlock, err := o.GetConsensusBlockAtSlot(slot)
	if err != nil {
		log.Fatal("could not get block at slot:", err)
	}

	if proposedBlock == nil {
		// Mised block, nothing to do
	} else {
		// Succesfull proposal, fetch the info we need

		fullBlock.SetConsensusBlock(proposedBlock)

		// Sanity check to ensure the block is the one we requested
		if fullBlock.GetSlotUint64() != slot {
			log.Fatal("slot does not match requested slot: ", fullBlock.GetSlotUint64(), " vs ", slot)
		}

		// TODO: Proof of concept. Fetch all events that we are interested in
		etherReceived, err := o.GetEtherReceivedEvents(fullBlock.GetBlockNumber())
		if err != nil {
			log.Fatal("failed getting ether received events: ", err)
		}

		events := &Events{
			etherReceived: etherReceived,
		}

		// Add the events to the block
		fullBlock.SetEvents(events)

		// Check if the proposal is from a subscribed validator
		isFromSubscriber := oracle.isSubscribed(fullBlock.GetProposerIndexUint64())

		// Check if the reward was sent to the pool
		isPoolRewarded := fullBlock.IsAddressRewarded(o.CliCfg.PoolAddress)

		// This calculation is expensive, do it only if the reward went to the pool or
		// if the block is from a subscribed validator.
		if isFromSubscriber || isPoolRewarded {
			blockNumber := new(big.Int).SetUint64(fullBlock.GetBlockNumber())
			header, receipts, err := o.GetExecHeaderAndReceipts(blockNumber, fullBlock.GetBlockTransactions())
			if err != nil {
				log.Fatal("failed getting header and receipts: ", err)
			}
			fullBlock.SetHeaderAndReceipts(header, receipts)
		}
	}

	return fullBlock
}

func (onchain *Onchain) GetConfigFromContract(
	cliCfg *config.CliConfig) *config.Config {

	MainnetChainId := uint64(1)
	GoerliChainId := uint64(5)

	chainId, err := onchain.ExecutionClient.ChainID(context.Background())
	if err != nil {
		log.Fatal("Could not get chainid: " + err.Error())
	}

	depositContract, err := onchain.ConsensusClient.DepositContract(context.Background())
	if err != nil {
		log.Fatal("Could not get deposit contract: " + err.Error())
	}

	if depositContract.ChainID != uint64(chainId.Int64()) {
		log.Fatal("ChainID from consensus and execution client dont match: ",
			depositContract.ChainID, " != ", chainId.Int64())
	}

	network := ""
	if depositContract.ChainID == MainnetChainId {
		network = "mainnet"
	} else if depositContract.ChainID == GoerliChainId {
		network = "goerli"
	} else {
		log.Fatal("ChainID not supported: ", depositContract.ChainID)
	}

	genesis, err := onchain.ConsensusClient.Genesis(context.Background())
	if err != nil {
		log.Fatal("Could not get genesis: " + err.Error())
	}

	genesisTime := uint64(genesis.GenesisTime.Unix())

	log.Info("Configured smoothing pool address: ", cliCfg.PoolAddress, " in network: ", network)

	balance, err := onchain.GetEthBalance(cliCfg.PoolAddress)
	if err != nil {
		log.Fatal("Could not get pool address balance: " + err.Error())
	}
	log.Info("Pool address balance: ", WeiToEther(balance), " Eth")

	deployedBlock, err := onchain.GetContractDeploymentBlock()
	if err != nil {
		log.Fatal("Could not get contract deployment block: " + err.Error())
	}
	log.Info("[Loaded from contract] Contract deployed at block: ", deployedBlock)

	block, err := onchain.BlockByNumber(deployedBlock)
	if err != nil {
		log.Fatal("Could not get block by number: " + err.Error())
	}

	blockTime := block.Time()
	SecondsInSlot := uint64(12)
	deployedSlot := (blockTime - genesisTime) / SecondsInSlot

	/*
		blockAtSlot, err := onchain.GetConsensusBlockAtSlot(deployedSlot)
		if err != nil {
			log.Fatal("Could not get block at slot: " + err.Error())
		}

		customBlockAtSlot := oracle.VersionedSignedBeaconBlock{blockAtSlot}
		if customBlockAtSlot.GetBlockNumber() != deployedBlock.Uint64() {
			log.Fatal("Could not map the deployed block with a slot, missmatch: ",
				customBlockAtSlot.GetBlockNumber(), " != ", deployedBlock)
		}*/

	log.Info("[Loaded from contract] Contract deployed in slot: ", deployedSlot)

	checkPointSizeInSlots, err := onchain.GetSlotCheckpointSize()
	if err != nil {
		log.Fatal("Could not get slot checkpoint size: " + err.Error())
	}
	log.Info("[Loaded from contract] Checkpoints will be created every ", checkPointSizeInSlots, " slots (", SlotsToTime(checkPointSizeInSlots), ")")

	poolFeesPercentTwoDecimals, err := onchain.GetPoolFee()
	if err != nil {
		log.Fatal("Could not get pool fee: " + err.Error())
	}
	log.Info("[Loaded from contract] Pool fees percent: ", float64(poolFeesPercentTwoDecimals.Uint64())/100, "% (raw value: ", poolFeesPercentTwoDecimals, ")")

	poolFeesAddress, err := onchain.GetPoolFeeAddress()
	if err != nil {
		log.Fatal("Could not get pool fee address: " + err.Error())
	}
	log.Info("[Loaded from contract] Pool fees address: ", poolFeesAddress, " (ensure you control its private key)")

	ethCollateralInWei, err := onchain.GetContractCollateral()
	if err != nil {
		log.Fatal("Could not get contract collateral: " + err.Error())
	}
	log.Info("[Loaded from contract] Required collateral to join the pool: ",
		ethCollateralInWei, " wei (", WeiToEther(ethCollateralInWei), " Eth)")

	if cliCfg.DryRun {
		log.Warn("The pool contract WILL NOT be updated, running in dry-run mode")
	} else {
		log.Warn("The pool contract WILL BE updated, running in normal mode")
	}

	conf := &config.Config{
		ConsensusEndpoint:     cliCfg.ConsensusEndpoint,
		ExecutionEndpoint:     cliCfg.ExecutionEndpoint,
		Network:               network,
		PoolAddress:           cliCfg.PoolAddress,
		DeployedSlot:          deployedSlot,
		DeployedBlock:         deployedBlock.Uint64(),
		CheckPointSizeInSlots: checkPointSizeInSlots,
		PoolFeesPercent:       int(poolFeesPercentTwoDecimals.Uint64()),
		PoolFeesAddress:       poolFeesAddress,
		CollateralInWei:       ethCollateralInWei,
		DryRun:                cliCfg.DryRun,
		NumRetries:            cliCfg.NumRetries,
		UpdaterKeyPass:        cliCfg.UpdaterKeyPass,
		UpdaterKeyPath:        cliCfg.UpdaterKeyPath,
	}

	return conf
}

// Fetch events
func (o *Onchain) GetEtherReceivedEvents(
	blockNumber uint64,
	opts ...retry.Option) ([]*contract.ContractEtherReceived, error) {

	startBlock := uint64(blockNumber)
	endBlock := uint64(blockNumber)

	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}

	var err error
	var itr *contract.ContractEtherReceivedIterator

	err = retry.Do(func() error {
		itr, err = o.Contract.FilterEtherReceived(filterOpts)
		if err != nil {
			log.Warn("Failed attempt GetEtherReceivedEvents for block ", strconv.FormatUint(blockNumber, 10), ": ", err.Error(), " Retrying...")
			return err
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.Wrap(err, "could not get EtherReceived events")
	}

	var events []*contract.ContractEtherReceived
	for itr.Next() {
		events = append(events, itr.Event)
	}
	err = itr.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not close EtherReceived iterator")
	}
	return events, nil
}

func (o *Onchain) GetSubscribeValidatorEvents(
	blockNumber uint64,
	opts ...retry.Option) ([]*contract.ContractSubscribeValidator, error) {

	startBlock := uint64(blockNumber)
	endBlock := uint64(blockNumber)

	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}

	var err error
	var itr *contract.ContractSubscribeValidatorIterator

	err = retry.Do(func() error {
		itr, err = o.Contract.FilterSubscribeValidator(filterOpts)
		if err != nil {
			log.Warn("Failed attempt GetSubscribeValidatorEvents for block ", strconv.FormatUint(blockNumber, 10), ": ", err.Error(), " Retrying...")
			return err
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.Wrap(err, "could not get SubscribeValidator events")
	}

	var events []*contract.ContractSubscribeValidator
	for itr.Next() {
		events = append(events, itr.Event)
	}
	err = itr.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not close SubscribeValidator iterator")
	}
	return events, nil
}

func (o *Onchain) UpdateContractMerkleRoot(slot uint64, newMerkleRoot string) string {

	// Support both 0x prefixed and non prefixed merkle roots
	if strings.HasPrefix(newMerkleRoot, "0x") {
		newMerkleRoot = newMerkleRoot[2:]
	}

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
	log.Info("Preparing tx from address: ", fromAddress.Hex())
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
	auth.GasPrice = nil
	auth.GasFeeCap = nil
	auth.GasTipCap = nil
	auth.NoSend = false
	auth.Context = context.Background()

	address := common.HexToAddress(o.CliCfg.PoolAddress)

	instance, err := contract.NewContract(address, o.ExecutionClient)
	if err != nil {
		log.Fatal(err)
	}

	// Create a tx calling the update rewards root function with the new merkle root
	tx, err := instance.SubmitReport(auth, slot, newMerkleRootBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.WithFields(log.Fields{
		"TxHash":        tx.Hash().Hex(),
		"NewMerkleRoot": newMerkleRoot,
	}).Info("Tx sent to Ethereum updating rewards merkle root, wait to be validated")

	// Leave 60 minutes for the tx to be validated
	deadline := time.Now().Add(60 * time.Minute)
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
	// TODO: protect with mutex?
	log.Info("Loading existing validators from the beacon chain")
	vals, err := o.GetFinalizedValidators()
	if err != nil {
		log.Fatal("Could not get validators: ", err)
	}
	o.validators = vals
	if len(vals) != 0 {
		log.WithFields(log.Fields{
			"TotalValidators":       len(vals),
			"LastIndex":             vals[phase0.ValidatorIndex(len(vals)-1)].Index,
			"ActivationSlotLastVal": GetActivationSlotOfLatestProcessedValidator(vals),
		}).Info("Done loading beacon chain validators")
	} else {
		log.Fatal("No validators were loaded from the beacon chain")
	}
}

func (o *Onchain) Validators() map[phase0.ValidatorIndex]*v1.Validator {
	// TODO: protect with mutex?
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
			retry.Attempts(uint(o.CliCfg.NumRetries)),
			retry.Delay(15 * time.Second),
		}
	} else {
		return opts
	}
}
