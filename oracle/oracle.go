package oracle

import (
	"math/big"
	"strconv"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"

	log "github.com/sirupsen/logrus"
)

type Oracle struct {
	fetcher *Fetcher
	cfg     *config.Config
	// TODO: Find a better name
	checkpointInfo    *CheckpointInfo
	LastProcessedSlot uint64
}

func NewOracle(
	cfg *config.Config,
	fetcher *Fetcher) *Oracle {
	checkpointInfo := NewCheckpointInfo()
	oracle := &Oracle{
		cfg:            cfg,
		fetcher:        fetcher,
		checkpointInfo: checkpointInfo,
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

func (or *Oracle) CalculateCheckpointRewards(slotToProcess uint64) error {
	//lastSlot := (checkpointNumber+1)*or.cfg.CheckPointSizeInSlots + or.cfg.DeployedSlot // TODO: not sure if -1
	/*
		log.WithFields(log.Fields{
			"checkpoint": checkpointNumber,
			"startSlot":  or.cfg.DeployedSlot,
			"lastSlot":   lastSlot,
		}).Info("Calculating checkpoint rewards")
	*/

	// TODO: Automatically update the subscriptions for the smart contract on every block
	// TODO: init is wrong. it has to be "update"
	contractSubscriptions := or.fetcher.GetSubscriptions()
	or.checkpointInfo.InitWithSubscriptions(contractSubscriptions)

	// Checkpoints are zero indexed
	if (slotToProcess - or.LastProcessedSlot) != 1 {
		log.Fatal("slotToProcess:", slotToProcess, "lastProcessedSlot:", or.LastProcessedSlot)
	}

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
	block, err := or.fetcher.GetBlockAtSlot(strconv.FormatUint(slotToProcess, 10))
	if err != nil {
		log.Fatal("err:", err) //TODO: Error
	}

	missedBlock := false
	var myBlock VersionedSignedBeaconBlock
	var reward *big.Int = big.NewInt(0)
	var sentOk bool = false

	if block == nil {
		missedBlock = true
	} else {
		myBlock = VersionedSignedBeaconBlock{*block}
		reward, sentOk, _, err = myBlock.GetSentRewardAndType(or.cfg.PoolAddress, *or.fetcher)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Check if proposal belongs to a subscription from the smart contract
	if or.IsValidatorSubscribed(valIndexDuty, contractSubscriptions) {
		if missedBlock {
			// if block was missed, advance state machine.
			or.checkpointInfo.AdvanceStateMachine(valIndexDuty, MissedProposal)

			// If new state is NotActive, means second lost block. Share its pending rewards and reset
			if or.checkpointInfo.IsNotActive(valIndexDuty) {
				or.checkpointInfo.IncreaseAllPendingRewards(or.checkpointInfo.pendingRewards[valIndexDuty])
				or.checkpointInfo.ResetPendingRewards(valIndexDuty)
			}

		} else {
			if sentOk {
				or.checkpointInfo.AdvanceStateMachine(valIndexDuty, ProposalWithCorrectFee)
				or.checkpointInfo.IncreaseAllPendingRewards(reward)
				or.checkpointInfo.ConsolidateBalance(valIndexDuty)
			} else {
				// reward was not sent to the pool, advance state machine -> ban.
				or.checkpointInfo.AdvanceStateMachine(valIndexDuty, ProposalWithWrongFee)
				or.checkpointInfo.IncreaseAllPendingRewards(or.checkpointInfo.pendingRewards[valIndexDuty])
				or.checkpointInfo.ResetPendingRewards(valIndexDuty)
				or.checkpointInfo.SetUnbanBalance(valIndexDuty, reward)
				// LogUpdateMetrics(valIndex, reward, duty, etc, state? event?)
			}
		}
	} else {
		// If the block was not missed and the validator is not subscribed
		// check if the reward was sent to the pool, and automatically subscribe it.
		if !missedBlock && sentOk {
			// TODO: subscribe
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
		donatedInBlock, err := myBlock.DonatedAmountInWei(or.cfg.PoolAddress)
		if err != nil {
			log.Fatal(err)
		}
		or.checkpointInfo.IncreaseAllPendingRewards(donatedInBlock)
		//TODO: add info on who donated and put to db. this can be useful for social stuff.
	}
	or.LastProcessedSlot = slotToProcess
	return nil // TODO: improve error handling
}
