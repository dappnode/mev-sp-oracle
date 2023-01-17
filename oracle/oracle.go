package oracle

import (
	"encoding/hex"
	"math/big"
	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"
	"mev-sp-oracle/postgres"

	log "github.com/sirupsen/logrus"
)

type Oracle struct {
	fetcher  *Fetcher
	cfg      *config.Config
	State    *OracleState
	postgres *postgres.Postgresql
}

func NewOracle(
	cfg *config.Config,
	fetcher *Fetcher) *Oracle {
	state := NewOracleState(cfg)

	postgres, err := postgres.New(cfg.PostgresEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	oracle := &Oracle{
		cfg:      cfg,
		fetcher:  fetcher,
		State:    state,
		postgres: postgres,
	}

	return oracle
}

// TODO: prove that a given adress is the owner of a given validator
// TODO: check if we need to reconstruct

// TODO: this can't just be the latest subscriptions from the smart contract.
// otherwise the following won't work.
// lets say validators 1, 2, 3 subscribe at t=0
// they prose some blocks and at t=100 all of them unsubscribe
// if we generate a checkpoint at t=200, we will be missing the subscriptions
// so we have to store, slot_start, and slot_end or something similar.

func (or *Oracle) IsValidatorSubscribed(validatorIndex uint64, subscriptions *Subscriptions) bool {
	for valIndex, _ := range subscriptions.subscriptions {
		if valIndex == validatorIndex {
			log.Info("Proposal duty from a subscribed validator:. TODO add block", validatorIndex)
			return true
		}
	}
	return false
}

func (or *Oracle) AdvanceStateToNextEpoch() error {
	//log.Info("Processing slot: ", or.State.Slot)
	//lastSlot := (checkpointNumber+1)*or.cfg.CheckPointSizeInSlots + or.cfg.DeployedSlot // TODO: not sure if -1
	/*
		log.WithFields(log.Fields{
			"checkpoint": checkpointNumber,
			"startSlot":  or.cfg.DeployedSlot,
			"lastSlot":   lastSlot,
		}).Info("Calculating checkpoint rewards")
	*/

	// TODO: Not really considering manual subscriptions now
	// Some dead logic here.

	// TODO: Automatically update the subscriptions for the smart contract on every block
	// TODO: init is wrong. it has to be "update"
	contractSubscriptions := or.fetcher.GetSubscriptions()
	_ = contractSubscriptions
	//or.state.InitWithSubscriptions(contractSubscriptions)

	// TODO: Ensure somehow that we dont process a slot twice.
	slotToProcess := or.State.Slot

	// Checkpoints are zero indexed
	/*
		if (slotToProcess - or.State.Slot) != 1 {
			log.Fatal("slotToProcess:", slotToProcess, "lastProcessedSlot:", or.State.Slot)
		}*/

	// TODO create function. GetBlockAndProposerAtSlot(): both block y duties.
	// get who should have proposed the block
	slotDuty, err := or.fetcher.GetProposalDuty(slotToProcess)
	if err != nil {
		// TODO: Return err
		// loop here until we get it? uf not sure. keep trying until no error.
		log.Fatal(err)
	}
	valIndexDuty := uint64(slotDuty.ValidatorIndex)
	// get the block
	// compare slot against block.Capella.Message.Slot slotDuty.Slot. do it down not here
	block, err := or.fetcher.GetBlockAtSlot(slotToProcess)
	if err != nil {
		log.Fatal("err:", err) //TODO: Error
	}

	missedBlock := false

	// TODO: Check type and use Capella/Bellatrix
	var myBlock BellatrixBlock
	var reward *big.Int = big.NewInt(0)
	var sentOk bool = false
	var rewardType int = -1

	if block == nil {
		missedBlock = true
	} else {
		myBlock = BellatrixBlock{*block.Bellatrix}
		reward, sentOk, rewardType, err = myBlock.GetSentRewardAndType(or.cfg.PoolAddress, *or.fetcher)
		if err != nil {
			log.Fatal(err)
		}
	}
	_ = rewardType

	// Check if proposal belongs to a subscription from the smart contract
	//if or.IsValidatorSubscribed(valIndexDuty, contractSubscriptions) {
	// Temporally disable auto subscriptions
	if false {
		if missedBlock {
			// if block was missed, advance state machine.
			or.State.AdvanceStateMachine(valIndexDuty, MissedProposal)

			// If new state is NotActive, means second lost block. Share its pending rewards and reset
			if or.State.IsNotActive(valIndexDuty) {
				or.State.IncreaseAllPendingRewards(or.State.PendingRewards[valIndexDuty])
				or.State.ResetPendingRewards(valIndexDuty)
			}

		} else {
			if sentOk {
				or.State.AdvanceStateMachine(valIndexDuty, ProposalWithCorrectFee)
				or.State.IncreaseAllPendingRewards(reward)
				or.State.ConsolidateBalance(valIndexDuty)
			} else {
				// reward was not sent to the pool, advance state machine -> ban.
				or.State.AdvanceStateMachine(valIndexDuty, ProposalWithWrongFee)
				or.State.IncreaseAllPendingRewards(or.State.PendingRewards[valIndexDuty])
				or.State.ResetPendingRewards(valIndexDuty)
				or.State.SetUnbanBalance(valIndexDuty, reward)
				// LogUpdateMetrics(valIndex, reward, duty, etc, state? event?)
			}
		}
	} else {
		// If the block was not missed and the validator is not subscribed
		// check if the reward was sent to the pool, and automatically subscribe it.
		if !missedBlock && sentOk {
			// If not already subscribed
			pubKey := "0x" + hex.EncodeToString(slotDuty.PubKey[:])
			// Move this somewhere else
			log.Info(pubKey)
			depositAddress, err := or.postgres.GetDepositAddressOfValidatorKey(pubKey)
			// TODO: Remove this in production
			if err != nil {
				log.Warn("Deposit key not found for ", pubKey, ". Expected in goerli. Using a default one. err: ", err)
				// If it errors, use a goerli address we control, only for debuging
				depositAddress = "0xc1B3c3F3Ff91ABd602BF3CAc449FFe9B852934f0"
			}
			log.Info("Auto subscribing validator: ", valIndexDuty, " with deposit address: ", depositAddress)
			or.State.AddSubscriptionIfNotAlready(valIndexDuty)
			or.State.IncreaseAllPendingRewards(reward)
			or.State.ConsolidateBalance(valIndexDuty)
		}

	}

	// TODO: detect unban transactions
	//unbannTxs := GetUnBanTx(block)

	//for _, tx := range unbannTxs {
	//	if tx.ValIndex == anyOfBannedValidators { // val index is not in the tree.
	//		if tx.Value() == checkpointInfo.unbanBalance[uint64(slotDuty.ValidatorIndex)] {
	//			checkpointInfo.scores[uint64(slotDuty.ValidatorIndex)] = Active
	//
	//		}
	//}
	//}

	if !missedBlock {
		// TODO: What if the first time we enter here there are no registered validators?
		// TODO: Not sure if donations are confused here with mev rewards
		//donatedInBlock, err := myBlock.DonatedAmountInWei(or.cfg.PoolAddress)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//or.State.IncreaseAllPendingRewards(donatedInBlock)
		//TODO: add info on who donated and put to db. this can be useful for social stuff.
	}
	or.State.Slot = slotToProcess + 1
	or.State.ProcessedSlots = or.State.ProcessedSlots + 1
	return nil // TODO: improve error handling
}
