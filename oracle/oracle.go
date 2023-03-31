package oracle

import (
	"github.com/dappnode/mev-sp-oracle/config"
)

type Oracle struct {
	cfg   *config.Config
	State *OracleState
}

func NewOracle(cfg *config.Config) *Oracle {
	state := NewOracleState(cfg)

	oracle := &Oracle{
		cfg:   cfg,
		State: state,
	}

	return oracle
}

// Advances the oracle to the next state, processing LatestSlot proposals/donations
// calculating the new state of all validators. It returns the slot that was processed
// and if there was an error.
func (or *Oracle) AdvanceStateToNextSlot(
	blockPool Block,
	blockSubs []Subscription,
	blockUnsubs []Unsubscription,
	blockDonations []Donation) (uint64, error) {

	// TODO: Ensure block are from the same slot and pool

	// Handle subscriptions first thing
	or.State.HandleManualSubscriptions(or.cfg.CollateralInWei, blockSubs)

	// If the validator was subscribed and missed proposed the block in this slot
	if blockPool.BlockType != MissedProposal && or.State.IsValidatorSubscribed(blockPool.ValidatorIndex) {
		or.State.HandleMissedBlock(blockPool)
	}

	// If a block was proposed in the slot (not missed)
	if blockPool.BlockType != MissedProposal {

		// Manual subscription. If feeRec is ok, means the reward was sent to the pool
		if blockPool.BlockType == OkPoolProposal {
			or.State.HandleCorrectBlockProposal(blockPool)
		}
		// If the validator was subscribed but the fee recipient was wrong
		// we ban the validator as it is not following the protocol rules
		if blockPool.BlockType == WrongFeeRecipient && or.State.IsValidatorSubscribed(blockPool.ValidatorIndex) {
			or.State.HandleBanValidator(blockPool)
		}
	}

	// Handle unsubscriptions the last thing after distributing rewards
	or.State.HandleManualUnsubscriptions(blockUnsubs)

	// Handle the donations from this block
	or.State.HandleDonations(blockDonations)

	processedSlot := or.State.LatestSlot
	or.State.LatestSlot = or.State.LatestSlot + 1
	return processedSlot, nil
}
