package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dappnode/mev-sp-oracle/api"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/metrics"
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
	log.Info("Version: ", config.ReleaseVersion)

	// Load config from cli
	cliCfg, err := config.NewCliConfig()
	if err != nil {
		log.Fatal("error parsing the cli config: ", err)
	}

	// Load key with rights to update the oracle (if not dry run)
	var updaterKey *ecdsa.PrivateKey
	var updaterAddress string
	if !cliCfg.DryRun {
		keystore, err := oracle.DecryptKey(cliCfg)
		if err != nil {
			log.Fatal("Could not decrypt updater key: ", err)
		}
		updaterKey = keystore.PrivateKey
		updaterAddress = keystore.Address.String()
		log.Info("Oracle contract will be update with address: ", updaterAddress)
	}

	// Instance of the onchain object to handle onchain interactions
	onchain, err := oracle.NewOnchain(cliCfg, updaterKey)
	if err != nil {
		log.Fatal("Could not create new onchain object: ", err)
	}

	// Populate config, most of the parameters are loaded from the smart contract
	cfg := onchain.GetConfigFromContract(cliCfg)

	// If we are not in dry run mode, means this instance will update the contract
	if !cfg.DryRun {
		log.Info("Checking if configured address ", updaterAddress, " is whitelisted to update the contract")
		isWhitelisted, err := onchain.IsAddressWhitelisted(cfg.DeployedBlock, updaterAddress)
		if err != nil {
			log.Fatal("Could not get whitelist status: " + err.Error())
		}
		if !isWhitelisted {
			log.Fatal("Pool address is not whitelisted, please run the 'whitelist' command first")
		}
		log.Info("Ok ", updaterAddress, " is whitelisted")
	}

	// Create the oracle instance
	oracleInstance := oracle.NewOracle(cfg)

	err = oracleInstance.LoadStateFromFile()
	if err == nil {
		log.Info("Found previous state to continue syncing")
	} else {
		log.Info("Previous state not found or could not be loaded, syncing from the begining: ", err)
		log.Info("Starting to process from slot: ", cfg.DeployedSlot)
	}

	api := api.NewApiService(cfg, oracleInstance, onchain)

	metrics.RunMetrics(8008)
	go api.StartHTTPServer()
	go mainLoop(oracleInstance, onchain, cfg)

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh

		// Save state in SIGINT or SIGTERM
		if sig == syscall.SIGINT || sig == syscall.SIGTERM {
			oracleInstance.SaveStateToFile()
		}

		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}

	log.Info("Oracle gracefully stopped")
}

func mainLoop(oracleInstance *oracle.Oracle, onchain *oracle.Onchain, cfg *oracle.Config) {

	// Check if we are in sync with the latest onchain root. If not we wont be updating
	// the state until we are in sync with the latest. This prevents from the oracle
	// losing sync, restarting and updating the roots again with old ones.
	syncedWithOnchainRoot := false

	// Load all the validators from the beacon chain
	// TODO: This is duplicated, not very nice
	onchain.RefreshBeaconValidators()
	oracleInstance.SetBeaconValidators(onchain.Validators())

	log.WithFields(log.Fields{
		"LatestProcessedSlot": oracleInstance.State().LatestProcessedSlot,
		"NextSlotToProcess":   oracleInstance.State().NextSlotToProcess,
	}).Info("Processing, see api for progress")

	// Check if we are in sync with the latest onchain root
	latestOnchainRoot, err := onchain.GetContractMerkleRoot()
	prevOracleRoot := oracleInstance.State().LatestCommitedState.MerkleRoot
	if err != nil {
		log.Fatal("Could not get latest onchain root: ", err)
	}

	if oracle.Equals(latestOnchainRoot, prevOracleRoot) {
		log.WithFields(log.Fields{
			"LatestOnChainRoot": latestOnchainRoot,
			"NewCalculateRoot":  prevOracleRoot,
			"RootSlot":          oracleInstance.State().LatestCommitedState.Slot,
		}).Info("Oracle IS in sync with the latest onchain root")
	} else {
		log.WithFields(log.Fields{
			"LatestOnChainRoot": latestOnchainRoot,
			"NewCalculateRoot":  prevOracleRoot,
			"RootSlot":          oracleInstance.State().LatestCommitedState.Slot,
		}).Info("Oracle IS NOT in sync with the latest onchain root")
	}

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

		if finalizedSlot >= oracleInstance.State().NextSlotToProcess {

			// Fetch block information
			fullBlock := onchain.FetchFullBlock(oracleInstance.State().NextSlotToProcess, oracleInstance)

			// Process the block
			processedSlot, err := oracleInstance.AdvanceStateToNextSlot(fullBlock)
			if err != nil {
				log.Fatal(err)
			}
			slotToLatestFinalized := finalizedSlot - oracleInstance.State().LatestProcessedSlot

			// Update metrics
			metrics.DistanceFromFinalizedSlot.Set(float64(slotToLatestFinalized))
			metrics.LatestProcessedSlot.Set(float64(oracleInstance.State().LatestProcessedSlot))
			metrics.LatestProcessedBlock.Set(float64(oracleInstance.State().LatestProcessedBlock))

			log.Debug("[", processedSlot, "/", finalizedSlot, "] Processed until slot, remaining: ",
				slotToLatestFinalized, " (", oracle.SlotsToTime(slotToLatestFinalized), " ago)")

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
			time.Sleep(1 * time.Minute)
			continue
		}

		// 600 slots is 2 hours
		UpdateValidatorsIntervalSlots := uint64(600)
		if oracleInstance.State().LatestProcessedSlot%UpdateValidatorsIntervalSlots == 0 {
			onchain.RefreshBeaconValidators()
			oracleInstance.SetBeaconValidators(onchain.Validators())
		}

		// Every CheckPointSizeInSlots we commit the state
		if oracleInstance.State().LatestProcessedSlot%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached, latest processed slot: ", oracleInstance.State().LatestProcessedSlot)

			// Get the latest onchain root (from the contract)
			latestOnchainRoot, err := onchain.GetContractMerkleRoot()
			if err != nil {
				log.Fatal("Could not get latest onchain root: ", err)
			}

			// Get the latest calculated root (from the oracle)
			prevOracleRoot := oracleInstance.State().LatestCommitedState.MerkleRoot

			// Ensure we didnt fell behind sync. If we did, we wont update the contract
			if !oracle.Equals(latestOnchainRoot, prevOracleRoot) {
				syncedWithOnchainRoot = false
				log.WithFields(log.Fields{
					"LatestOnChainRoot": latestOnchainRoot,
					"NewCalculateRoot":  prevOracleRoot,
					"RootSlot":          oracleInstance.State().LatestCommitedState.Slot,
				}).Info("Oracle IS NOT in sync with the latest onchain root")
			} else {
				syncedWithOnchainRoot = true
				log.WithFields(log.Fields{
					"LatestOnChainRoot": latestOnchainRoot,
					"NewCalculateRoot":  prevOracleRoot,
					"RootSlot":          oracleInstance.State().LatestCommitedState.Slot,
				}).Info("Oracle IS in sync with the latest onchain root")
			}

			// Calculate new state with new root
			enoughData := oracleInstance.StoreLatestOnchainState()
			newOracleRoot := oracleInstance.State().LatestCommitedState.MerkleRoot

			// Update metrics
			metrics.KnownRootAndSlot.WithLabelValues(
				fmt.Sprintf("%d", oracleInstance.State().LatestCommitedState.Slot),
				newOracleRoot).Set(1)

			// If we were not in sync and the new root matches the latest onchain root, we are now in sync
			// meaning that in the next checkpoint we will update the contract
			if !syncedWithOnchainRoot && oracle.Equals(latestOnchainRoot, newOracleRoot) {
				syncedWithOnchainRoot = true
				log.WithFields(log.Fields{
					"LatestOnChainRoot": latestOnchainRoot,
					"NewCalculateRoot":  newOracleRoot,
					"RootSlot":          oracleInstance.State().LatestCommitedState.Slot,
				}).Info("New oracle root IS in sync with the latest onchain root")
			}

			// If we were not in sync and the new roots doesnt match, just log the progress
			if !syncedWithOnchainRoot && !oracle.Equals(latestOnchainRoot, newOracleRoot) {
				log.WithFields(log.Fields{
					"LatestOnChainRoot": latestOnchainRoot,
					"NewCalculateRoot":  newOracleRoot,
					"RootSlot":          oracleInstance.State().LatestCommitedState.Slot,
				}).Info("New oracle root IS NOT in sync with the latest onchain root")
			}

			if !enoughData {
				log.Warn("Not enough data to create a merkle tree and hence update the contract. Skipping till next checkpoint")
			} else {
				if !cfg.DryRun && syncedWithOnchainRoot && !oracle.Equals(latestOnchainRoot, newOracleRoot) {
					txHash := onchain.UpdateContractMerkleRoot(oracleInstance.State().LatestCommitedState.Slot, newOracleRoot)
					_ = txHash
				}
			}

			// Persist new state in file only if everything went fine
			oracleInstance.SaveStateToFile()
		}
	}
}
