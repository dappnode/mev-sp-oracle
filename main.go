package main

import (
	// TODO: Change when pushed
	//"github.com/dappnode/mev-sp-oracle/config"
	//"github.com/dappnode/mev-sp-oracle/oracle"
	"context"
	"mev-sp-oracle/config"
	"mev-sp-oracle/oracle"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// Hardcoded for Ethereum
var SlotsInEpoch = uint64(32)

// Example: ./mev-sp-oracle --consensus-endpoint="http://127.0.0.1:5051" --execution-endpoint="http://127.0.0.1:8545" --deployed-slot=5365409 --pool-address="0x388C818CA8B9251b393131C08a736A67ccB19297" --checkpoint-size=10
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

	// TODO: resume from file
	log.Info("Starting to process from slot", oracle.State.Slot)

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

		if finalizedSlot > oracle.State.Slot {
			err = oracle.AdvanceStateToNextEpoch()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Info("Waiting for new finalized slot")
			time.Sleep(10 * time.Second)
		}

		// TODO: Rethink this a bit. Do not run in the first block we process, and think about edge cases
		if (oracle.State.Slot-cfg.DeployedSlot)%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached")
			// TODO: Dump to file and generate merkle trees/root/proof
		}
	}

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh
		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}

	log.Info("Stopping mev-sp-oracle")
}
