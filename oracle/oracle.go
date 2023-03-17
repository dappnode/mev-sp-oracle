package oracle

import (
	"encoding/hex"
	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"
	"mev-sp-oracle/contract"
	"mev-sp-oracle/postgres"

	log "github.com/sirupsen/logrus"
)

type Oracle struct {
	fetcher    *Fetcher
	cfg        *config.Config
	State      *OracleState
	Operations *contract.Operations
	Postgres   *postgres.Postgresql
}

func NewOracle(
	cfg *config.Config,
	fetcher *Fetcher) *Oracle {
	state := NewOracleState(cfg)

	postgres, err := postgres.New(cfg.PostgresEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	operations := contract.NewOperations(cfg)
	oracle := &Oracle{
		cfg:        cfg,
		fetcher:    fetcher,
		State:      state,
		Postgres:   postgres,
		Operations: operations,
	}

	return oracle
}

// Returns the validator index that should propose the block at a given slot, followed
// by wether the block was missed or not (true = ok proposal) and the block if it was not missed
func (or *Oracle) GetBlockIfAny(slot uint64) (uint64, string, bool, *VersionedSignedBeaconBlock) {
	// Gets the duty, indicating which validator should propose the block at this slot
	slotDuty, err := or.fetcher.GetProposalDuty(slot)
	if err != nil {
		log.Fatal("could not get proposal duty: ", err)
	}

	// The validator that should propose the block
	valIndexDuty := uint64(slotDuty.ValidatorIndex)
	valPublicKey := "0x" + hex.EncodeToString(slotDuty.PubKey[:])

	proposedBlock, err := or.fetcher.GetBlockAtSlot(slot)
	if err != nil {
		log.Fatal("could not get block at slot:", err)
	}

	// A nil block means that the validator did not propose a block (missed proposal)
	if proposedBlock == nil {
		return valIndexDuty, valPublicKey, false, nil
	}
	return valIndexDuty, valPublicKey, true, &VersionedSignedBeaconBlock{proposedBlock}
}

func (or *Oracle) UpdateSubscriptions(block VersionedSignedBeaconBlock) {
	// TODO: Listen events from the smart contract
	// Detect manual subscriptions
	/*
		proposerIndex := uint64(block.GetProposerIndex())

		reward, correctFeeRec, rewardType, err := block.GetSentRewardAndType(or.cfg.PoolAddress, *or.fetcher)
		if err != nil {
			log.Fatal(err)
		}*/

}

func (or *Oracle) AdvanceStateToNextSlot() error {
	// TODO: Get block and listen for new subscriptions

	// TODO: Ensure somehow that we dont process a slot twice.
	slotToProcess := or.State.Slot

	// Get the block if any and who proposed it (or should have proposed it)
	proposerIndex, proposerKey, proposedOk, block := or.GetBlockIfAny(slotToProcess)
	_ = proposerKey

	// TODO: Update subscriptions with the info from this block (fee rec) + listening to the smart contract
	// this also updates the deposit address and all the information of the validator.

	if proposedOk {
		// If the block was proposed ok
		reward, correctFeeRec, rewardType, err := block.GetSentRewardAndType(or.cfg.PoolAddress, *or.fetcher)
		_ = rewardType
		if err != nil {
			log.Fatal(err)
		}

		// Manual subscription. If feeRec is ok, means the reward was sent to the pool
		if correctFeeRec {
			or.State.AddSubscriptionIfNotAlready(proposerIndex)
			or.State.AdvanceStateMachine(proposerIndex, ProposalWithCorrectFee)
			or.State.IncreaseAllPendingRewards(reward)
			or.State.ConsolidateBalance(proposerIndex)
		} else {
			// The validator set a wrong fee recipient, ban it forever
			// and give its pending rewards to the rest
			or.State.AdvanceStateMachine(proposerIndex, ProposalWithWrongFee)
			or.State.IncreaseAllPendingRewards(or.State.Validators[proposerIndex].PendingRewardsWei)
			or.State.ResetPendingRewards(proposerIndex)
			// TODO: What about its claimable rewards? ban also?
		}
	} else {
		// If the validator missed a block, just advance the state machine
		// there are no rewards to share, but validator state slighly changes
		//or.State.AdvanceStateMachine(proposerIndex, MissedBlock)
	}

	// Check if proposal belongs to a subscription from the smart contract
	//if or.IsValidatorSubscribed(valIndexDuty, contractSubscriptions) {
	// Temporally disable auto subscriptions
	if false {

	} else {
		// If the block was not missed and the validator is not subscribed
		// check if the reward was sent to the pool, and automatically subscribe it.
		//if !missedBlock && sentOk {
		// If not already subscribed
		//pubKey := "0x" + hex.EncodeToString(slotDuty.PubKey[:])
		// Move this somewhere else
		//log.Info(pubKey)
		//depositAddress, err := or.Postgres.GetDepositAddressOfValidatorKey(pubKey)
		// TODO: Remove this in production
		//if err != nil {
		//	log.Warn("Deposit key not found for ", pubKey, ". Expected in goerli. Using a default one. err: ", err)
		// If it errors, use a goerli address we control, only for debuging. Remove for production
		//	someDepositAddresses := []string{
		//		"0x001eDa52592fE2f8a28dA25E8033C263744b1b6E",
		//		"0x0029a125E6A3f058628Bd619C91f481e4470D673",
		//		"0x003718fb88964A1F167eCf205c7f04B25FF46B8E",
		//		"0x004b1EaBc3ea60331a01fFfC3D63E5F6B3aB88B3",
		//		"0x005CD1608e40d1e775a97d12e4f594029567C071",
		//		"0x0069c9017BDd6753467c138449eF98320be1a4E4",
		//		"0x007cF0936ACa64Ef22C0019A616801Bec7FCCECF",
		//	}
		// Just pick a "random" one to not always the same
		//	depositAddress = someDepositAddresses[slotDuty.Slot%7]
		//}
		/*
			log.Info("Auto subscribing validator: ", valIndexDuty, " with deposit address: ", depositAddress)
			or.State.AddSubscriptionIfNotAlready(valIndexDuty)
			or.State.IncreaseAllPendingRewards(reward)
			or.State.ConsolidateBalance(valIndexDuty)
			or.State.ProposedBlocks[valIndexDuty] = append(or.State.ProposedBlocks[valIndexDuty], uint64(slotDuty.Slot))
		*/

		// if the validator already proposed a block this is already set
		//or.State.DepositAddresses[valIndexDuty] = depositAddress
		//or.State.ValidatorKey[valIndexDuty] = pubKey
		// TODO: perhaps not needed anymore. just same value as deposit address
		//or.State.PoolRecipientAddresses[valIndexDuty] = depositAddress

		// TODO: quick PoC. todo store blocks that were missed.

		/*
			rewardTypeString := "unset"
			if rewardType == VanilaBlock {
				rewardTypeString = "vanila"
			} else if rewardType == MevBlock {
				rewardTypeString = "mev"
			}
			err = or.Postgres.StoreBlockInDb(
				"TODO",
				uint64(slotDuty.Slot),
				pubKey,
				valIndexDuty,
				rewardTypeString,
				*reward,
				1, // TODO
			)
			if err != nil {
				log.Fatal(err)
			}*/
	}

	or.State.Slot = slotToProcess + 1
	or.State.ProcessedSlots = or.State.ProcessedSlots + 1
	return nil
}
