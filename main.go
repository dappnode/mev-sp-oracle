package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dappnode/mev-sp-oracle/api"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/oracle"

	log "github.com/sirupsen/logrus"
)

// Hardcoded for Ethereum
var SlotsInEpoch = uint64(32)

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	log.Info("Starting smoothing pool oracle")
	cfg, err := config.NewCliConfig()
	if err != nil {
		log.Fatal(err)
	}

	onchain := oracle.NewOnchain(*cfg)
	oracleInstance := oracle.NewOracle(cfg, onchain)
	api := api.NewApiService(*cfg, oracleInstance.State, onchain)

	balnace := onchain.GetEthBalance(cfg.PoolAddress)
	log.WithFields(log.Fields{
		"address":     cfg.PoolAddress,
		"balance_wei": balnace,
	}).Info("Pool Address Balance")

	// TODO: Try to resume syncing from latest known state from file
	// TODO: Temporally disabled until further tested
	//recoveredState, err := or.ReadStateFromFile()
	//if err == nil {
	//	oracle.State = recoveredState
	//} else {
	//	log.Info("Previous state not found or could not be loaded, syncing from the begining")
	//}

	go api.StartHTTPServer()
	go mainLoop(oracleInstance, onchain, cfg)

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh
		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
		// TODO: Save state in SIGINT or SIGTERM
	}

	log.Info("Oracle gracefully stopped")
}

func mainLoop(oracleInstance *oracle.Oracle, onchain *oracle.Onchain, cfg *config.Config) {

	log.Info("Starting to process from slot: ", oracleInstance.State.LatestSlot)

	for {
		// Ensure that the nodes we are using are in sync with the blockchain (consensus + execution)
		if !onchain.AreNodesInSync() {
			log.Error("Nodes are not in sync, skipping until in sync")
			time.Sleep(15 * time.Second)
			continue
		}

		finality, err := onchain.ConsensusClient.Finality(context.Background(), "finalized")
		if err != nil {
			log.Error("Could not get finalized status:", err)
			time.Sleep(15 * time.Second)
			continue
		}

		finalizedEpoch := uint64(finality.Finalized.Epoch)
		finalizedSlot := finalizedEpoch * SlotsInEpoch

		if finalizedSlot > oracleInstance.State.LatestSlot {
			processedSlot, err := oracleInstance.AdvanceStateToNextSlot()
			if err != nil {
				log.Fatal(err)
			}
			slotToLatestFinalized := finalizedSlot - oracleInstance.State.LatestSlot

			// Log progress every x slots
			//if finalizedSlot%300 == 0 {
			log.Info("[", processedSlot, "/", finalizedSlot, "] Processed until slot, remaining: ",
				slotToLatestFinalized, " (", oracle.SlotsToTime(slotToLatestFinalized), " ago)")
			//}
		} else {
			log.WithFields(log.Fields{
				"finalizedSlot":   finalizedSlot,
				"finalizedEpoch":  finalizedEpoch,
				"oracleStateSlot": oracleInstance.State.LatestSlot,
			}).Info("Waiting for new finalized slot")
			time.Sleep(60 * time.Second)
			continue
		}

		// How often we store data in the database in slots
		UpdateDbIntervalSlots := uint64(1)
		if oracleInstance.State.LatestSlot%UpdateDbIntervalSlots == 0 {
			// TODO: Unused. As a nice to have we can store
			// the intermediate validator balances in db
			// So a valaidator can see it balance over time.
			// Not feasible to store this in memory
		}

		// Every CheckPointSizeInSlots we commit the state
		if oracleInstance.State.LatestSlot%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached, slot: ", oracleInstance.State.LatestSlot)

			// mRoot, enoughData := oracle.State.GetMerkleRootIfAny()
			enoughData := oracleInstance.State.StoreLatestOnchainState()

			oracleInstance.State.SaveStateToFile()
			oracleInstance.State.LogAccumulatedBalances()
			oracleInstance.State.LogPendingBalances()

			if !enoughData {
				log.Warn("Not enough data to create a merkle tree and hence update the contract. Skipping till next checkpoint")
			} else {
				//txHash := ""
				if !cfg.DryRun {
					txHash := onchain.UpdateContractMerkleRoot(oracleInstance.State.LatestCommitedState.MerkleRoot)
					_ = txHash
				}
			}
		}
	}
}
