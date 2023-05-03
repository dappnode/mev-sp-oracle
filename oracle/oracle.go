package oracle

import (
	"errors"
	"fmt"
	"sync"

	"github.com/dappnode/mev-sp-oracle/config"
	log "github.com/sirupsen/logrus"
)

type Oracle struct {
	cfg   *config.Config
	State *OracleState
	mu sync.RWMutex
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

	//declare mutex and release it when function returns
	or.mu.Lock()
	defer or.mu.Unlock()
	
	if or.State.NextSlotToProcess != (or.State.LatestProcessedSlot + 1) {
		log.Fatal("Next slot to process is not the last processed slot + 1",
			or.State.NextSlotToProcess, " ", or.State.LatestProcessedSlot)
	}

	err := or.validateParameters(blockPool, blockSubs, blockUnsubs, blockDonations)
	if err != nil {
		return 0, err
	}

	// Handle subscriptions first thing
	or.State.HandleManualSubscriptions(blockSubs)

	// If the validator was subscribed and missed proposed the block in this slot
	if blockPool.BlockType == MissedProposal && or.State.IsSubscribed(blockPool.ValidatorIndex) {
		or.State.HandleMissedBlock(blockPool)
	}

	// If a block was proposed in the slot (not missed)
	if blockPool.BlockType != MissedProposal {

		if blockPool.BlockType == OkPoolProposalBlsKeys {
			// TODO: This is a bit hackish
			log.Warn("Block proposal was ok but bls keys are not supported, sending rewards to pool")
			or.State.SendRewardToPool(blockPool.Reward)
			// TODO: Send rewards to pool as we dont know any validator address to give it
		}

		// Manual subscription. If feeRec is ok, means the reward was sent to the pool
		if blockPool.BlockType == OkPoolProposal {
			or.State.HandleCorrectBlockProposal(blockPool)
		}
		// If the validator was subscribed but the fee recipient was wrong
		// we ban the validator as it is not following the protocol rules
		if blockPool.BlockType == WrongFeeRecipient && or.State.IsSubscribed(blockPool.ValidatorIndex) {
			or.State.HandleBanValidator(blockPool)
		}
	}

	// Handle unsubscriptions the last thing after distributing rewards
	or.State.HandleManualUnsubscriptions(blockUnsubs)

	// Handle the donations from this block
	or.State.HandleDonations(blockDonations)

	processedSlot := or.State.NextSlotToProcess
	or.State.LatestProcessedSlot = processedSlot
	or.State.NextSlotToProcess++
	if blockPool.BlockType != MissedProposal {
		or.State.LatestProcessedBlock = blockPool.Block
	}
	return processedSlot, nil
}

func (or *Oracle) validateParameters(
	blockPool Block,
	blockSubs []Subscription,
	blockUnsubs []Unsubscription,
	blockDonations []Donation) error {

	if blockPool.Slot != or.State.NextSlotToProcess {
		return errors.New(fmt.Sprint("Slot of blockPool is not the same as the latest slot of the oracle. BlockPool: ",
			blockPool.Slot, " Oracle: ", or.State.NextSlotToProcess))
	}

	// TODO: Add more validators to block subs unsubs, donations, etc
	return nil
}
