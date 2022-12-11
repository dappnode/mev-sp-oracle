package main

import (
	// TODO: Change when pushed
	//"github.com/dappnode/mev-sp-oracle/config"
	//"github.com/dappnode/mev-sp-oracle/oracle"
	"context"
	"mev-sp-oracle/config"
	"mev-sp-oracle/oracle"
	"time"

	log "github.com/sirupsen/logrus"
)

// Hardcoded for Ethereum
var SlotsInEpoch = uint64(32)

// Example: ./mev-sp-oracle --consensus-endpoint="http://127.0.0.1:5051" --execution-endpoint="http://127.0.0.1:8545" --deployed-slot=5324453 --pool-address="0x" --checkpoint-size=10 --debug-hardcoded-subscriptions-file=file.txt
func main() {
	log.Info("mev-sp-oracle")
	cfg, err := config.NewCliConfig()
	if err != nil {
		log.Fatal(err)
	}

	fetcher := oracle.NewFetcher(*cfg)
	oracle := oracle.NewOracle(cfg, fetcher)
	/*
		syncProgress, err := fetcher.ExecutionClient.SyncProgress(context.Background())
		if err != nil {
			log.Error(err)
		}
	*/

	// TODO: Quick and dirty
	oracle.LastProcessedSlot = cfg.DeployedSlot - 1

	for {

		headSlot, err := fetcher.ConsensusClient.NodeSyncing(context.Background())
		if err != nil {
			log.Error("Could not get node sync status:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if headSlot.IsSyncing {
			log.Error("Node is not in sync")
			time.Sleep(5 * time.Second)
			continue
		}

		finality, err := fetcher.ConsensusClient.Finality(context.Background(), "finalized")
		if err != nil {
			log.Error("Could not get finalized status:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		finalizedEpoch := uint64(finality.Finalized.Epoch)
		finalizedSlot := finalizedEpoch * SlotsInEpoch

		if finalizedSlot > oracle.LastProcessedSlot {
			err = oracle.CalculateCheckpointRewards(oracle.LastProcessedSlot + 1)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Info("Waiting for new finalized slot")
			time.Sleep(10 * time.Second)
		}

		// TODO: Rethink this a bit. Do not run in the first block we process, and think about edge cases
		if (oracle.LastProcessedSlot-cfg.DeployedSlot)%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached")
			// TODO: Dump to file and generate merkle trees/root/proof
		}
	}
}

// TODO: handle sigint and sigterm signals
