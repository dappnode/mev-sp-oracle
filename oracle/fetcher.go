package oracle

import (
	"context"
	"math/big"
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

type Fetcher struct {
	ConsensusClient *http.Service
	ExecutionClient *ethclient.Client
}

// Fetches external data:
// - consensus client
// - execution client
// - pool contract
func NewFetcher(cfg config.Config) *Fetcher {

	executionClient, err := ethclient.Dial(cfg.ExecutionEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	client, err := http.New(context.Background(),
		http.WithTimeout(60*time.Second),
		http.WithAddress(cfg.ConsensusEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		log.Fatal(err)
	}
	consensusClient := client.(*http.Service)

	return &Fetcher{
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
	}
}

// TODO: rename to getConsensusblock?
func (f *Fetcher) GetBlockAtSlot(slot string) (*spec.VersionedSignedBeaconBlock, error) {

	// TODO: set custom timeouts
	signedBeaconBlock, err := f.ConsensusClient.SignedBeaconBlock(context.Background(), slot)

	return signedBeaconBlock, err
}

func (f *Fetcher) GetProposalDuty(slot uint64) (*api.ProposerDuty, error) {
	// Hardcoded
	slotsInEpoch := uint64(32)
	epoch := slot / slotsInEpoch
	// Empty indexes to force fetching all duties
	indexes := make([]phase0.ValidatorIndex, 0)

	duties, err := f.ConsensusClient.ProposerDuties(
		context.Background(),
		phase0.Epoch(epoch),
		indexes)
	if err != nil {
		return &api.ProposerDuty{}, err
	}
	slotWithinEpoch := slot % slotsInEpoch
	return duties[slotWithinEpoch], nil
}

// This function is expensive as gets every tx receipt from the block. Use only if needed
func (f *Fetcher) GetExecHeaderAndReceipts(blockNumber *big.Int, rawTxs []bellatrix.Transaction) (*types.Header, []*types.Receipt, error) {
	header, err := f.ExecutionClient.HeaderByNumber(context.Background(), blockNumber)
	if err != nil {
		log.Fatal(err)
	}

	var receipts []*types.Receipt
	for _, rawTx := range rawTxs {
		tx, _, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal(err)
		}
		receipt, err := f.ExecutionClient.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			log.Fatal(err)
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
			268288: "0x", //TODO: add start/end
			342517: "0x",
			306361: "0x",
			77334:  "0x",
			307966: "0x",

			481020: "0x", // propose mev block at 5323504
			168929: "0x", // proposes vanila block at 5323506
			195242: "0x", // proposes mev block at  5323505 0x4675c7e5baafbffbca748158becba61ef3b0a263

			210588: "0x",
		},
	}
	return &manualSubscriptions

}
