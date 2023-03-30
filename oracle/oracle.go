package oracle

import (
	"encoding/hex"
	"errors"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/postgres"

	log "github.com/sirupsen/logrus"
)

type Oracle struct {
	onchain  *Onchain
	cfg      *config.Config
	State    *OracleState
	Postgres *postgres.Postgresql
}

func NewOracle(
	cfg *config.Config,
	onchain *Onchain) *Oracle {
	state := NewOracleState(cfg)

	postgres, err := postgres.New(cfg.PostgresEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	oracle := &Oracle{
		cfg:      cfg,
		onchain:  onchain,
		State:    state,
		Postgres: postgres,
	}

	return oracle
}

// Returns the validator index that should propose the block at a given slot, followed
// by wether the block was missed or not (true = ok proposal) and the block if it was not missed
func (or *Oracle) GetBlockIfAny(slot uint64) (uint64, string, bool, *VersionedSignedBeaconBlock) {
	// Gets the duty, indicating which validator should propose the block at this slot
	slotDuty, err := or.onchain.GetProposalDuty(slot)
	if err != nil {
		log.Fatal("could not get proposal duty: ", err)
	}

	// The validator that should propose the block
	valIndexDuty := uint64(slotDuty.ValidatorIndex)
	valPublicKey := "0x" + hex.EncodeToString(slotDuty.PubKey[:])

	proposedBlock, err := or.onchain.GetConsensusBlockAtSlot(slot)
	if err != nil {
		log.Fatal("could not get block at slot:", err)
	}

	// A nil block means that the validator did not propose a block (missed proposal)
	if proposedBlock == nil {
		return valIndexDuty, valPublicKey, false, nil
	}
	return valIndexDuty, valPublicKey, true, &VersionedSignedBeaconBlock{proposedBlock}
}

// Advances the oracle to the next state, processing LatestSlot proposals/donations
// calculating the new state of all validators. It returns the slot that was processed
// and if there was an error.
func (or *Oracle) AdvanceStateToNextSlot() (uint64, error) {

	// TODO: Ensure somehow that we dont process a slot twice.
	slotToProcess := or.State.LatestSlot

	// Get the block if any and who proposed it (or should have proposed it)
	proposerIndex, proposerKey, proposedOk, block := or.GetBlockIfAny(slotToProcess)

	customBlock := Block{
		Slot:           slotToProcess,
		ValidatorIndex: proposerIndex,
		ValidatorKey:   proposerKey,
	}

	// If the block was proposed (not missed)
	if proposedOk {
		blockNumber := block.GetBlockNumber()

		// or.onchain.GetRewardsRoot()

		// Fetch block proposal parameters such as rewards
		reward, correctFeeRec, rewardType, err := block.GetSentRewardAndType(or.cfg.PoolAddress, *or.onchain)
		if err != nil {
			return uint64(0), errors.New("could not get reward from block: " + err.Error())
		}

		// Fetch subscription data
		newBlockSubs, err := or.onchain.GetBlockSubscriptions(blockNumber)
		if err != nil {
			return uint64(0), errors.New("could not get block subscriptions: " + err.Error())
		}

		// Fetch unsubscription data
		newBlockUnsub, err := or.onchain.GetBlockUnsubscriptions(blockNumber)
		if err != nil {
			return uint64(0), errors.New("could not get block unsubscriptions: " + err.Error())
		}

		// TODO: This is wrong, as this event will also be triggered when a validator proposes a MEV block
		// Fetch donations in this block
		blockDonations, err := or.onchain.GetDonationEvents(blockNumber)
		if err != nil {
			return uint64(0), errors.New("could not get block donations: " + err.Error())
		}

		// Handle subscriptions first thing before distributing rewards
		or.State.HandleManualSubscriptions(or.cfg.CollateralInWei, newBlockSubs)

		// Manual subscription. If feeRec is ok, means the reward was sent to the pool
		if correctFeeRec {
			proposerDepositAddress := or.onchain.GetDepositAddressOfValidator(proposerKey, slotToProcess)

			customBlock.Reward = reward
			customBlock.RewardType = rewardType
			customBlock.DepositAddress = proposerDepositAddress

			or.State.HandleCorrectBlockProposal(customBlock)
		}
		// If the validator was subscribed but the fee recipient was wrong
		// we ban the validator as it is not following the protocol rules
		if !correctFeeRec && or.State.IsValidatorSubscribed(proposerIndex) {
			or.State.HandleBanValidator(customBlock)
		}

		// Handle unsubscriptions the last thing after distributing rewards
		or.State.HandleManualUnsubscriptions(newBlockUnsub)

		// Handle the donations from this block
		or.State.HandleDonations(blockDonations)
	}

	// If the validator was subscribed and missed proposed the block in this slot
	if !proposedOk && or.State.IsValidatorSubscribed(proposerIndex) {
		or.State.HandleMissedBlock(customBlock)
	}

	or.State.LatestSlot = slotToProcess + 1
	return slotToProcess, nil
}
