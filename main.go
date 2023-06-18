package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/rand"
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

	// TODO: Add flag to enable this. Logs the line and file of the log
	//log.SetReportCaller(true)

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

	found, err := oracleInstance.LoadFromJson()
	if err != nil {
		log.Fatal("Critical error loading state from json: ", err)
	}
	if !found {
		log.Warn("Previous state not found or could not be loaded, syncing from the begining slot=", oracleInstance.State().DeployedSlot)
	} else {
		log.Info("Found previous state to continue syncing")
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
			err := oracleInstance.SaveToJson()
			if err != nil {
				log.Error("Could not save state to json: ", err)
			} else {
				log.Info("State saved to json")
			}
		}

		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}

	log.Info("Oracle gracefully stopped")
}

func mainLoop(oracleInstance *oracle.Oracle, onchain *oracle.Onchain, cfg *oracle.Config) {

	// Load all the validators from the beacon chain
	// TODO: This is duplicated, not very nice
	onchain.RefreshBeaconValidators()
	oracleInstance.SetBeaconValidators(onchain.Validators())

	log.WithFields(log.Fields{
		"LatestProcessedSlot": oracleInstance.State().LatestProcessedSlot,
		"NextSlotToProcess":   oracleInstance.State().NextSlotToProcess,
	}).Info("Processing, see api for progress")

	// Check if we are in sync with the latest onchain root
	onchainRoot, onchainSlot, err := onchain.GetOnchainSlotAndRoot()
	if err != nil {
		log.Fatal("Could not get onchain slot and root: ", err)
	}

	inSync, err := oracleInstance.IsOracleInSyncWithChain(onchainRoot, onchainSlot)
	if err != nil {
		log.Fatal("Could not check if oracle is in sync with the chain: ", err)
	}
	if !inSync {
		log.Info("Oracle is not in sync with the chain, syncing in progress to get same root")
	}

	for {
		// Ensure that the nodes we are using are in sync with the blockchain (consensus + execution)
		inSync, err := onchain.AreNodesInSync()
		if err != nil {
			log.Fatal("Could not get nodes in sync status:", err)
		}
		if !inSync {
			log.Warn("Nodes are not in sync, skipping until in sync")
			time.Sleep(15 * time.Second)
			continue
		}

		finality, err := onchain.ConsensusClient.Finality(context.Background(), "finalized")
		if err != nil {
			log.Error("Could not get finalized status, sleeping and retrying:", err)
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

		// 600 slots is 2 hours. TODO: move this somewhere else
		UpdateValidatorsIntervalSlots := uint64(600)
		if oracleInstance.State().LatestProcessedSlot%UpdateValidatorsIntervalSlots == 0 {
			onchain.RefreshBeaconValidators()
			oracleInstance.SetBeaconValidators(onchain.Validators())
		}

		// Every CheckPointSizeInSlots we commit the state given some conditions
		if oracleInstance.State().LatestProcessedSlot%cfg.CheckPointSizeInSlots == 0 { // TODO: extract to oracle method
			log.Info("Checkpoint reached, latest processed slot: ", oracleInstance.State().LatestProcessedSlot)

			// Freeze state
			enoughData := oracleInstance.StoreLatestOnchainState() // TODO: perhaps not the best name
			if !enoughData {
				log.Warn("Not enough data to create a merkle tree and hence update the contract. Skipping till next checkpoint")
				continue
			}

			// Get new state
			newState := oracleInstance.LatestCommitedState()

			// Update metrics
			metrics.KnownRootAndSlot.WithLabelValues(
				fmt.Sprintf("%d", newState.Slot),
				newState.MerkleRoot).Set(1)

			// Get onchain root and slot
			onchainRoot, onchainSlot, err := onchain.GetOnchainSlotAndRoot()
			if err != nil {
				log.Fatal("Could not get onchain slot and root: ", err)
			}

			// For logging, display if we are in sync with the chain
			oracleInSync, err := oracleInstance.IsOracleInSyncWithChain(onchainRoot, onchainSlot)
			if err != nil {
				log.Fatal("Could not check if oracle is in sync with chain: ", err)
			}
			if oracleInSync {
				log.Info("Oracle is now in sync with the chain")
			} else {
				log.Info("Oracle is not yet in sync with the chain")
			}

			// If the new state is not the one onchain + checkpoint size, do nothing
			if newState.Slot != onchainSlot+cfg.CheckPointSizeInSlots {
				continue
			}

			// Otherwise, we are in the next state, so we can update the contract
			if !cfg.DryRun && enoughData {
				// Random sleep between 0 and 10 minutes to avoid all oracles updating at the same time
				r := rand.Intn(11)
				time.Sleep(time.Duration(r) * time.Minute)

				err := onchain.UpdateContractMerkleRoot(newState.Slot, newState.MerkleRoot)
				if err != nil {
					log.Fatal("Could not update contract merkle root: ", err)
				}

				// Wait until the state we submitted is consolidated in the contract
				for {
					onchainRoot, onchainSlot, err = onchain.GetOnchainSlotAndRoot()
					if err != nil {
						log.Fatal("Could not get onchain slot and root: ", err)
					}

					if onchainRoot == newState.MerkleRoot && onchainSlot == newState.Slot {
						log.WithFields(log.Fields{
							"OnchainRoot": onchainRoot,
							"OnchainSlot": onchainSlot,
							"OracleRoot":  newState.MerkleRoot,
							"OracleSlot":  newState.Slot,
						}).Info("The submitted state is now consolidated in the contract")
						break
					} else {
						log.Info("Contract not yet updated, waiting")
						time.Sleep(30 * time.Second)
					}
				}
			}

			// Persist new state in file only if everything went fine
			err = oracleInstance.SaveToJson()
			if err != nil {
				log.Error("Could not save state to json: ", err)
			} else {
				log.Info("State saved to json")
			}
		}
	}
}
