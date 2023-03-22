package main

import (
	// TODO: Change when pushed
	//"github.com/dappnode/mev-sp-oracle/config"
	//"github.com/dappnode/mev-sp-oracle/oracle"
	"context"
	"mev-sp-oracle/api"
	"mev-sp-oracle/config"
	or "mev-sp-oracle/oracle"
	"mev-sp-oracle/postgres"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// Hardcoded for Ethereum
var SlotsInEpoch = uint64(32)

// Examples:
// Goerli/Prater
// ./mev-sp-oracle --consensus-endpoint="http://127.0.0.1:5051" --execution-endpoint="http://127.0.0.1:8545" --deployed-slot=4500000 --pool-address="0x455e5aa18469bc6ccef49594645666c587a3a71b" --checkpoint-size=10
func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	log.Info("mev-sp-oracle")
	cfg, err := config.NewCliConfig()
	if err != nil {
		log.Fatal(err)
	}

	onchain := or.NewOnchain(*cfg)
	oracle := or.NewOracle(cfg, onchain)
	api := api.NewApiService(*cfg, oracle.State, onchain)
	go api.StartHTTPServer()

	// Preparae the database
	// TODO: Dirty, to be safe. Clean db at startup until we can safely resume. The idea is
	// to resume from the last checkpoint.
	// TODO: Remove all this, just for tests.
	_, err = oracle.Postgres.Db.Exec(context.Background(), "drop table if exists t_oracle_validator_balances")
	if err != nil {
		log.Fatal("error cleaning table t_oracle_validator_balances at startup: ", err)
	}

	_, err = oracle.Postgres.Db.Exec(context.Background(), "drop table if exists t_pool_blocks")
	if err != nil {
		log.Fatal("error cleaning table t_pool_blocks at startup: ", err)
	}

	_, err = oracle.Postgres.Db.Exec(context.Background(), "drop table if exists t_oracle_depositaddress_rewards")
	if err != nil {
		log.Fatal("error cleaning table t_pool_blocks at startup: ", err)
	}

	if _, err := oracle.Postgres.Db.Exec(
		context.Background(),
		postgres.CreateRewardsTable); err != nil {
		log.Fatal("error creating table t_oracle_validator_balances: ", err)
	}

	if _, err := oracle.Postgres.Db.Exec(
		context.Background(),
		postgres.CreateDepositAddressRewardsTable); err != nil {
		log.Fatal("error creating table t_oracle_depositaddress_rewards: ", err)
	}

	if _, err := oracle.Postgres.Db.Exec(
		context.Background(),
		postgres.CreateBlocksTable); err != nil {
		log.Fatal("error creating table t_pool_blocks ", err)
	}

	go mainLoop(oracle, onchain, cfg)

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh
		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}

	// TODO: Dump to file before stopping
	log.Info("Stopping mev-sp-oracle")
}

func mainLoop(oracle *or.Oracle, onchain *or.Onchain, cfg *config.Config) {
	/*
		syncProgress, err := onchain.ExecutionClient.SyncProgress(context.Background())
		if err != nil {
			log.Error(err)
		}
	*/

	// Try to resume syncing from latest known state from file
	// TODO: Temporally disabled until further tested
	//recoveredState, err := or.ReadStateFromFile()
	//if err == nil {
	//	oracle.State = recoveredState
	//} else {
	//	log.Info("Previous state not found or could not be loaded, syncing from the begining")
	//}
	log.Info("Starting to process from slot: ", oracle.State.LatestSlot)

	// TODO: Before doing anything, get merkle root from chain to avoid
	// overriding the existing one. Also load from file

	for {

		headSlot, err := onchain.ConsensusClient.NodeSyncing(context.Background())
		if err != nil {
			log.Error("Could not get node sync status:", err)
			time.Sleep(15 * time.Second)
			continue
		}

		if headSlot.IsSyncing {
			log.Error("Node is not in sync")
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

		if finalizedSlot > oracle.State.LatestSlot {
			err = oracle.AdvanceStateToNextSlot()
			if err != nil {
				log.Fatal(err)
			}
			slotToLatestFinalized := finalizedSlot - oracle.State.LatestSlot
			log.Info("[", oracle.State.LatestSlot, "/", finalizedSlot, "] Done processing slot. Remaining: ",
				slotToLatestFinalized, " (", or.SlotsToTime(slotToLatestFinalized), " ago)")
		} else {
			log.WithFields(log.Fields{
				"finalizedSlot":    finalizedSlot,
				"finalizedEpoch":   finalizedEpoch,
				"oracleStateSlot":  oracle.State.LatestSlot,
				"oracleStateEpoch": oracle.State.LatestSlot / SlotsInEpoch,
			}).Info("Waiting for new finalized slot")

			time.Sleep(30 * time.Second)
		}

		// How often we store data in the database in slots
		UpdateDbIntervalSlots := uint64(1)
		if oracle.State.LatestSlot%UpdateDbIntervalSlots == 0 {
			// TODO: Unused. As a nice to have we can store
			// the intermediate validator balances in db
			// So a valaidator can see it balance over time.
			// Not feasible to store this in memory

		}

		// Every CheckPointSizeInSlots we commit the state
		if oracle.State.LatestSlot%cfg.CheckPointSizeInSlots == 0 {
			log.Info("Checkpoint reached, slot: ", oracle.State.LatestSlot)

			// mRoot, enoughData := oracle.State.GetMerkleRootIfAny()
			enoughData := oracle.State.StoreLatestOnchainState()

			oracle.State.SaveStateToFile()
			oracle.State.LogAccumulatedBalances()
			oracle.State.LogPendingBalances()

			if !enoughData {
				log.Warn("Not enough data to create a merkle tree and hence update the contract. Skipping till next checkpoint")
			} else {
				//txHash := ""
				if !cfg.DryRun {
					txHash := onchain.UpdateContractMerkleRoot(oracle.State.LatestCommitedState.MerkleRoot)
					_ = txHash
				}
			}
		}
	}
}
