package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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

	//Ensure that configured collateral in oracle matches contract collateral
	contractCollateral, err := onchain.GetContractCollateral()
	if err != nil {
		log.Fatal("Could not fetch subscription collateral from smart contract onchain")
	} else if contractCollateral.Cmp(cfg.CollateralInWei) != 0 {
		log.WithFields(log.Fields{
			"Defined Collateral":  cfg.CollateralInWei,
			"Contract Collateral": contractCollateral,
		}).Fatal("Defined collateral does not match contract collateral")
	} else if contractCollateral.Cmp(cfg.CollateralInWei) == 0 {
		log.Info("Defined Collateral matches Contract Collateral onchain")
	}
	// TODO Enabled, but requires further testing
	err = oracleInstance.State.LoadStateFromFile()
	if err == nil {
		log.Info("Found previous state to continue syncing")
	} else {
		log.Info("Previous state not found or could not be loaded, syncing from the begining: ", err)
	}

	api := api.NewApiService(cfg, oracleInstance, onchain)

	go api.StartHTTPServer()
	go mainLoop(oracleInstance, onchain, cfg)

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh

		// Save state in SIGINT or SIGTERM
		if sig == syscall.SIGINT || sig == syscall.SIGTERM {
			oracleInstance.State.SaveStateToFile()
		}

		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}

	log.Info("Oracle gracefully stopped")
}

func mainLoop(oracleInstance *oracle.Oracle, onchain *oracle.Onchain, cfg *config.Config) {

	// Load all the validators from the beacon chain
	onchain.RefreshBeaconValidators()

	log.WithFields(log.Fields{
		"LatestProcessedSlot": oracleInstance.State.LatestProcessedSlot,
		"NextSlotToProcess":   oracleInstance.State.NextSlotToProcess,
	}).Info("Processing, see api for progress")

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

		if finalizedSlot > oracleInstance.State.NextSlotToProcess {

			// Get all the information of the block that was proposed in this slot
			poolBlock, blockSubs, blockUnsubs, blockDonations := onchain.GetAllBlockInfo(oracleInstance.State.NextSlotToProcess)
			processedSlot, err := oracleInstance.AdvanceStateToNextSlot(poolBlock, blockSubs, blockUnsubs, blockDonations)
			if err != nil {
				log.Fatal(err)
			}
			slotToLatestFinalized := finalizedSlot - oracleInstance.State.LatestProcessedSlot

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

		// 1200 slots is 4 hour
		UpdateValidatorsIntervalSlots := uint64(1200)
		if oracleInstance.State.LatestProcessedSlot%UpdateValidatorsIntervalSlots == 0 {
			validators := onchain.Validators()
			lastValidator := validators[phase0.ValidatorIndex(len(validators)-1)]

			// Update only if the oracle advances beyond the last validator we have
			if lastValidator.Validator.ActivationEligibilityEpoch <= phase0.Epoch(oracleInstance.State.LatestProcessedSlot/SlotsInEpoch) {
				onchain.RefreshBeaconValidators()
			}
		}

		// Every CheckPointSizeInSlots we commit the state
		if oracleInstance.State.LatestProcessedSlot%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached, latest processed slot: ", oracleInstance.State.LatestProcessedSlot)

			// mRoot, enoughData := oracle.State.GetMerkleRootIfAny()
			enoughData := oracleInstance.State.StoreLatestOnchainState()

			oracleInstance.State.SaveStateToFile()

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
