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
	oracleInstance := oracle.NewOracle(cfg, onchain)
	api := api.NewApiService(*cfg, oracleInstance.State, onchain)

	balance, err := onchain.GetEthBalance(cfg.PoolAddress)
	if err != nil {
		log.Fatal("Could not get pool address balance: " + err.Error())
	}
	log.WithFields(log.Fields{
		"address":     cfg.PoolAddress,
		"balance_wei": balance,
	}).Info("Pool Address Balance")

	contractMerkleRoot, err := onchain.GetContractMerkleRoot()
	if err != nil {
		log.Fatal("Could not get contract merkle root: " + err.Error())
	}

	recoveredState, err := oracle.ReadStateFromFile()
	if err != nil {
		log.Info("Could not recover state from file, starting from slot: ", oracleInstance.State.LatestSlot, " err: ", err)
	} else {
		if recoveredState.LatestCommitedState.MerkleRoot != contractMerkleRoot {
			log.Info("Stored onchain state does not match the one in the contract, starting from slot: ", recoveredState.LatestSlot)
		} else {
			// Load the state from the file
			log.Info("Stored onchain state matches the one in the contract, resuming from known state at slot:", recoveredState.LatestSlot)
			oracleInstance.State = recoveredState
		}
	}

	// Check if we are behind the contract
	if oracleInstance.State.LatestCommitedState.MerkleRoot != contractMerkleRoot {
		// Only matters in production, do not care in dry run mode
		if !cfg.DryRun {
			// The oracle is 1 or more checkpoints behind the merkle root in the contract. This is not likely to happen
			// in the oracle in production.
			log.Fatal("Onchain stored state does not match the one in the contract. Oracle is behind "+
				"one ore more checkpoints behind the contract. Review this manually: ",
				oracleInstance.State.LatestCommitedState.MerkleRoot, " vs ", contractMerkleRoot)
		}
	}

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

			oracleInstance.State.LogAccumulatedBalances()
			oracleInstance.State.LogPendingBalances()

			if !enoughData {
				log.Warn("Not enough data to create a merkle tree and hence update the contract. Skipping till next checkpoint")
			} else {
				if !cfg.DryRun {
					txHash, err := onchain.UpdateContractMerkleRoot(oracleInstance.State.LatestCommitedState.MerkleRoot)
					if err != nil {
						log.Fatal("Could not update onchain contract merkle root: ", err)
					}
					// Persist the state only if the tx was validated successfully
					err = oracleInstance.State.SaveStateToFile(txHash)
					if err != nil {
						log.Fatal("Could not save state to file: ", err)
					}
				} else {
					err = oracleInstance.State.SaveStateToFile("0x")
					if err != nil {
						log.Fatal("Could not save state to file: ", err)
					}
				}
			}
		}
	}
}
