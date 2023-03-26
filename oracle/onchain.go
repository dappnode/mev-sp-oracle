package oracle

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/contract"

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

// Default retry options. This specifies what to do when a call to the
// consensus or execution client fails. Default is to retry 5 times
// with a 15 seconds delay and the default backoff strategy (see avas/retry-go)
// Note that in some cases we might want to avoid retrying at all, for example
// when serving data to an api, we may want to just fail fast and return an error
var defaultRetryOpts = []retry.Option{
	retry.Attempts(5),
	retry.Delay(15 * time.Second),
}

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
	address := common.HexToAddress("0x25eb524fabe93979d299158a1c7d1ff6628e0356")
	contract, err := contract.NewContract(address, executionClient)
	if err != nil {
		return nil, errors.New("Error instantiating contract: " + err.Error())
	}

	return &Onchain{
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
		Cfg:             &cfg,
		Contract:        contract,
	}, nil
}

func (f *Onchain) AreNodesInSync() bool {
	var err error
	var execSync *ethereum.SyncProgress
	var consSync *api.SyncState

	// TODO: Perhaps in all interactions allow a max number of failures and then error/panic
	for {
		execSync, err = f.ExecutionClient.SyncProgress(context.Background())
		if err != nil {
			log.Warn("Error fetching execution client sync progress: ", err)
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}

	for {
		consSync, err = f.ConsensusClient.NodeSyncing(context.Background())
		if err != nil {
			log.Warn("Error fetching consensus client sync progress: ", err)
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}

	// Exeuction client returns nil if not syncing (in sync)
	// Give couple of slots to consensus client
	if execSync == nil && (consSync.SyncDistance < 2) {
		return true
	}
	return false
}

// TODO: rename to getConsensusblock?
func (f *Onchain) GetBlockAtSlot(slot uint64) (*spec.VersionedSignedBeaconBlock, error) {

	// TODO: set custom timeouts
	slotStr := strconv.FormatUint(slot, 10)
	var signedBeaconBlock *spec.VersionedSignedBeaconBlock
	var err error

	for {
		signedBeaconBlock, err = f.ConsensusClient.SignedBeaconBlock(context.Background(), slotStr)
		if err != nil {
			log.Warn("Error fetching block at slot ", slot, ": ", err, " Retrying in 15 seconds...")
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}
	return signedBeaconBlock, err
}

func (f *Onchain) GetProposalDuty(slot uint64) (*api.ProposerDuty, error) {
	// Hardcoded
	slotsInEpoch := uint64(32)
	epoch := slot / slotsInEpoch
	slotWithinEpoch := slot % slotsInEpoch

	// If cache hit, return the result
	if ProposalDutyCache.Epoch == epoch {
		// Health check that should never happen
		if ProposalDutyCache.Epoch != uint64(ProposalDutyCache.Duties[slotWithinEpoch].Slot/phase0.Slot(slotsInEpoch)) {
			log.Fatal("Proposal duty epoch does not match when converting slot to epoch")
		}
		return ProposalDutyCache.Duties[slotWithinEpoch], nil
	}

	// Empty indexes to force fetching all duties
	indexes := make([]phase0.ValidatorIndex, 0)
	var duties []*api.ProposerDuty
	var err error

	for {
		duties, err = f.ConsensusClient.ProposerDuties(
			context.Background(),
			phase0.Epoch(epoch),
			indexes)
		if err != nil {
			log.Warn("Error fetching proposer duties for epoch ", epoch, ": ", err, " Retrying in 15 seconds...")
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}

	// Store result in cache
	ProposalDutyCache = EpochDuties{epoch, duties}

	return duties[slotWithinEpoch], nil
}

// This function is expensive as gets every tx receipt from the block. Use only if needed
func (f *Onchain) GetExecHeaderAndReceipts(blockNumber *big.Int, rawTxs []bellatrix.Transaction) (*types.Header, []*types.Receipt, error) {

	var header *types.Header
	var err error

	for {
		header, err = f.ExecutionClient.HeaderByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Warn("Error fetching header at block ", blockNumber, ": ", err, " Retrying in 15 seconds...")
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}

	var receipts []*types.Receipt
	for _, rawTx := range rawTxs {
		// This should never happen
		tx, _, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal(err)
		}
		var receipt *types.Receipt
		for {
			receipt, err = f.ExecutionClient.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				log.Warn("Error fetching receipt for tx ", tx.Hash(), ": ", err, " Retrying in 15 seconds...")
				time.Sleep(15 * time.Second)
				continue
			}
			break
		}
		receipts = append(receipts, receipt)
	}
	return header, receipts, nil
}

// TODO: Wondering if we can be sure that the smart contract can differentiate
// between subscriptions and donations.
func (o *Onchain) GetDonationEvents(blockNumber uint64) []Donation {
	// Not the most effective way, but we just need to advance one by one.
	startBlock := uint64(blockNumber)
	endBlock := uint64(blockNumber)

	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}

	itr, err := o.Contract.FilterDonation(filterOpts)
	if err != nil {
		log.Fatal("coult not filter donations for block: ", blockNumber, " err: ", err)
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
	return donations
}

func (o *Onchain) GetContractMerkleRoot(opts ...retry.Option) (string, error) {
	var rewardsRootStr string

	// Retries multiple times before errorings
	err := retry.Do(
		func() error {
			callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
			rewardsRoot, err := o.Contract.RewardsRoot(callOpts)
			if err != nil {
				return err
			}
			rewardsRootStr = "0x" + hex.EncodeToString(rewardsRoot[:])
			return nil
		}, GetRetryOpts(opts)...)

	if err != nil {
		return "", errors.New("could not get merkle root from contract: " + err.Error())
	}

	return rewardsRootStr, nil
}

func (o *Onchain) GetEthBalance(address string) *big.Int {
	account := common.HexToAddress(address)
	balanceWei, err := o.ExecutionClient.BalanceAt(context.Background(), account, nil)

	// Allow some retries before failing
	if err != nil {
		log.Fatal(err)
	}

	return balanceWei
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
	address := common.HexToAddress("0x25eB524fAbe93979D299158a1c7D1FF6628e0356")

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

func GetRetryOpts(opts []retry.Option) []retry.Option {
	if len(opts) == 0 {
		return defaultRetryOpts
	} else {
		return opts
	}
}
