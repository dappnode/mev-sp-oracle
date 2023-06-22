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
	"github.com/dappnode/mev-sp-oracle/utils"

	log "github.com/sirupsen/logrus"
)

// Hardcoded for Ethereum
var SlotsInEpoch = uint64(32)

// How often onchain validators are reloaded: 600 slots is 2 hours
var UpdateValidatorsIntervalSlots = uint64(600)

func main() {
	// Load config from cli
	cliCfg, err := config.NewCliConfig()
	if err != nil {
		log.Fatal("error parsing the cli config: ", err)
	}

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	//log.SetReportCaller(true)

	log.Info("Starting smoothing pool oracle")
	log.Info("Version: ", config.ReleaseVersion)
	metrics.Version.WithLabelValues(config.ReleaseVersion).Set(1)

	// Set log-level
	logLevel, err := log.ParseLevel(cliCfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logLevel)

	// Load key with rights to update the oracle (if not dry run)
	var updaterKey *ecdsa.PrivateKey
	var updaterAddress string
	if !cliCfg.DryRun {
		keystore, err := utils.DecryptKey(cliCfg)
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
			err := oracleInstance.SaveToJson(false)
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
				slotToLatestFinalized, " (", utils.SlotsToTime(slotToLatestFinalized), " ago)")

		} else {
			log.WithFields(log.Fields{
				"ChainFinalizedSlot": finalizedSlot,
				"OracleStateSlot":    oracleInstance.State().LatestProcessedSlot,
			}).Debug("Waiting for new finalized slot")
			// No new finalized slot, wait a bit
			time.Sleep(1 * time.Minute)
			continue
		}

		if oracleInstance.State().LatestProcessedSlot%UpdateValidatorsIntervalSlots == 0 {
			onchain.RefreshBeaconValidators()
			oracleInstance.SetBeaconValidators(onchain.Validators())
		}

		// Every CheckPointSizeInSlots we commit the state given some conditions
		if oracleInstance.State().LatestProcessedSlot%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached, latest processed slot: ", oracleInstance.State().LatestProcessedSlot)

			// This wont work since we need an archival geth node to fetch balances at specific blocks that are not the last
			// as it is it errors "missing trie node". Leaving here for reference
			//uniqueAddresses := oracleInstance.GetUniqueWithdrawalAddresses()
			//poolEthBalanceWei, err := onchain.GetPoolEthBalance(big.NewInt(0).SetUint64(oracleInstance.State().LatestProcessedBlock))
			//if err != nil {
			//	log.Error("Could not get pool eth balance: ", err)
			//}
			//claimedPerAccount := onchain.GetClaimedPerWithdrawalAddress(uniqueAddresses, oracleInstance.State().LatestProcessedBlock)
			//err = oracleInstance.RunReconciliation(poolEthBalanceWei, claimedPerAccount)
			//if err != nil {
			//	log.Fatal("Reconciliation failed, state was not commited: ", err)
			//}

			// Freeze state
			enoughData := oracleInstance.FreezeCheckpoint()
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

			// If so we are ready to update the contract, but multiple oracles will be racing here.
			// Lets say we have m oracles with a quorum on n (n/m). The oracles will be racing to update the root
			// and only n txs will go through and (m-n) will be reverted, as the new state will be consolidated.
			// In order to avoid txs being reverted (which costs gas), we add a random sleep between 0 and 15 minutes
			// to avoid a collision. This is not perfect, but it should be good enough. Statistically, it would be
			// very improbable that n+1 oracles will wait the same amount of time producing a collision.
			if !cfg.DryRun && enoughData {
				// This also blocks sync in some cases, can be optimized
				r := rand.Intn(16)
				time.Sleep(time.Duration(r) * time.Minute)
			}

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

			// If the oracle has permission to update the contract root (!dryRun), we have enough data
			// to construct a merkle tree and also the new state slot is the one onchain + checkpoint size
			// Then we can update the new merkle root
			if !cfg.DryRun && enoughData && newState.Slot == onchainSlot+cfg.CheckPointSizeInSlots {
				err := onchain.UpdateContractMerkleRoot(newState.Slot, newState.MerkleRoot)
				if err != nil {
					// There is a very improbable case that this tx is expected to fail. If quorum is n for
					// m oracles, if n+1 oracles submit the tx at the same time, the last tx will revert.
					// In this case it would be expected to fail, but note that the above delay should
					// prevent this from happening.
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
						log.Info("Submitted merkle root is not consolidated, waiting for other oracles to update it")
						time.Sleep(1 * time.Minute)
					}
				}
			}

			// Persist new state in file only if everything went fine
			err = oracleInstance.SaveToJson(true)
			if err != nil {
				log.Error("Could not save state to json: ", err)
			} else {
				log.Info("State saved to json")
			}
		}
	}
}
