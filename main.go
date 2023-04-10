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

	onchain, err := oracle.NewOnchain(*cfg)
	if err != nil {
		log.Fatal("Could not create new onchain object: ", err)
	}
	oracleInstance := oracle.NewOracle(cfg)

	balance, err := onchain.GetEthBalance(cfg.PoolAddress)
	if err != nil {
		log.Fatal("Could not get pool address balance: " + err.Error())
	}
	log.WithFields(log.Fields{
		"Address":    cfg.PoolAddress,
		"BalanceWei": balance,
	}).Info("Pool Address Balance")

	// TODO Enabled, but requires further testing
	recoveredState, err := oracle.ReadStateFromFile()
	if err == nil {
		log.Info("Found previous state to continue syncing")
		oracleInstance.State = recoveredState
	} else {
		log.Info("Previous state not found or could not be loaded, syncing from the begining")
	}

	api := api.NewApiService(cfg, oracleInstance.State, onchain)

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

	log.Info("Starting to process from slot (see api for progress): ", oracleInstance.State.LatestSlot)

	for {
		// Ensure that the nodes we are using are in sync with the blockchain (consensus + execution)
		inSync, err := onchain.AreNodesInSync()
		if err != nil {
			log.Fatal("Could not get nodes in sync status:", err)
		}
		if !inSync {
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

			// Get all the information of the block that was proposed in this slot
			poolBlock, blockSubs, blockUnsubs, blockDonations := onchain.GetAllBlockInfo(oracleInstance.State.LatestSlot)
			processedSlot, err := oracleInstance.AdvanceStateToNextSlot(poolBlock, blockSubs, blockUnsubs, blockDonations)
			if err != nil {
				log.Fatal(err)
			}
			slotToLatestFinalized := finalizedSlot - oracleInstance.State.LatestSlot

			_ = processedSlot
			_ = slotToLatestFinalized

			// Do not log progress every slot, it is too much. See api for progress
			// Log progress every x slots when syncing
			/*logEverySlots := uint64(300)
			if finalizedSlot%logEverySlots == 0 {
				log.Info("[", processedSlot, "/", finalizedSlot, "] Processed until slot, remaining: ",
					slotToLatestFinalized, " (", oracle.SlotsToTime(slotToLatestFinalized), " ago)")
			}*/
		} else {
			/*log.WithFields(log.Fields{
				"FinalizedSlot":   finalizedSlot,
				"FinalizedEpoch":  finalizedEpoch,
				"OracleStateSlot": oracleInstance.State.LatestSlot,
			}).Info("Waiting for new finalized slot")*/
			// No new finalized slot, wait a bit
			time.Sleep(3 * time.Minute)
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
			oracleInstance.State.LogBalances()

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
