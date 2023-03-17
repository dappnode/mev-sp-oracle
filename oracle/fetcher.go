package oracle

import (
	"context"
	"math/big"
	"strconv"
	"time"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
)

type EpochDuties struct {
	Epoch  uint64
	Duties []*api.ProposerDuty
}

// Simple cache storing epoch -> proposer duties
// This is useful to not query the beacon node for each slot
// since ProposerDuties returns the duties for the whole epoch
// Note that the cache is meant to store only one epoch's duties
var ProposalDutyCache EpochDuties

type Fetcher struct {
	ConsensusClient *http.Service
	ExecutionClient *ethclient.Client
}

// Fetches external data:
// - consensus client
// - execution client
// - pool contract
func NewFetcher(cfg config.Config) *Fetcher {

	// Dial the execution client
	executionClient, err := ethclient.Dial(cfg.ExecutionEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	// Get chainid to ensure the endpoint is working
	chainId, err := executionClient.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Connected succesfully to execution client. ChainId: ", chainId)

	// Dial the consensus client
	client, err := http.New(context.Background(),
		http.WithTimeout(60*time.Second),
		http.WithAddress(cfg.ConsensusEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		log.Fatal(err)
	}
	consensusClient := client.(*http.Service)

	// Get deposit contract to ensure the endpoint is working
	depositContract, err := consensusClient.DepositContract(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Connected succesfully to consensus client. Deposit contract: ", depositContract)

	// Print sync status of consensus and execution client
	execSync, err := executionClient.SyncProgress(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Execution client sync state (nil is synced): ", execSync)

	consSync, err := consensusClient.NodeSyncing(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Consensus client sync state: ", consSync)

	return &Fetcher{
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
	}
}

// TODO: rename to getConsensusblock?
func (f *Fetcher) GetBlockAtSlot(slot uint64) (*spec.VersionedSignedBeaconBlock, error) {

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

func (f *Fetcher) GetProposalDuty(slot uint64) (*api.ProposerDuty, error) {
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
func (f *Fetcher) GetExecHeaderAndReceipts(blockNumber *big.Int, rawTxs []bellatrix.Transaction) (*types.Header, []*types.Receipt, error) {

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

// TODO:
func (f *Fetcher) GetSubscriptions() *Subscriptions {
	// manual subscriptions from the smart contract
	var manualSubscriptions = Subscriptions{
		blockHeigh: "0", // todo whatever.
		slotHeigh:  "",  //todo not sure

		// TODO: perhaps define subscription type
		subscriptions: map[uint64]string{
			/*
				268288: "0x", //TODO: add start/end
				342517: "0x",
				306361: "0x",
				77334:  "0x",
				307966: "0x",

				481020: "0x", // propose mev block at 5323504
				168929: "0x", // proposes vanila block at 5323506
				195242: "0x", // proposes mev block at  5323505 0x4675c7e5baafbffbca748158becba61ef3b0a263

				210588: "0x",*/
		},
	}
	return &manualSubscriptions

}
