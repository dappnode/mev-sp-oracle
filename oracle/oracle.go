package oracle

import (
	"github.com/dappnode/mev-sp-oracle/config"
)

type Oracle struct {
	onchain *Onchain
	cfg     *config.Config
	State   *OracleState
}

func NewOracle(
	cfg *config.Config,
	onchain *Onchain) *Oracle {
	state := NewOracleState(cfg)

	oracle := &Oracle{
		cfg:     cfg,
		onchain: onchain,
		State:   state,
	}

	return oracle
}

// Advances the oracle to the next state, processing LatestSlot proposals/donations
// calculating the new state of all validators. It returns the slot that was processed
// and if there was an error.
func (or *Oracle) AdvanceStateToNextSlot() (uint64, error) {

	// TODO: Ensure somehow that we dont process a slot twice. TODO: perhap rename to latestprocessed and change. and to +1 here.
	slotToProcess := or.State.LatestSlot

	// Get all the information of the block that was proposed in this slot
	poolBlock, blockSubs, blockUnsubs, blockDonations := or.onchain.GetAllBlockInfo(slotToProcess)

	// Handle subscriptions first thing
	or.State.HandleManualSubscriptions(or.cfg.CollateralInWei, blockSubs)

	// If the validator was subscribed and missed proposed the block in this slot
	if poolBlock.BlockType != MissedProposal && or.State.IsValidatorSubscribed(poolBlock.ValidatorIndex) {
		or.State.HandleMissedBlock(poolBlock)
	}

	// If a block was proposed in the slot (not missed)
	if poolBlock.BlockType != MissedProposal {

		// Manual subscription. If feeRec is ok, means the reward was sent to the pool
		if poolBlock.BlockType == OkPoolProposal {
			or.State.HandleCorrectBlockProposal(poolBlock)
		}
		// If the validator was subscribed but the fee recipient was wrong
		// we ban the validator as it is not following the protocol rules
		if poolBlock.BlockType == WrongFeeRecipient && or.State.IsValidatorSubscribed(poolBlock.ValidatorIndex) {
			or.State.HandleBanValidator(poolBlock)
		}
	}

	// Handle unsubscriptions the last thing after distributing rewards
	or.State.HandleManualUnsubscriptions(blockUnsubs)

	// Handle the donations from this block
	or.State.HandleDonations(blockDonations)

	or.State.LatestSlot = slotToProcess + 1
	return slotToProcess, nil
}
