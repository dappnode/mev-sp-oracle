package oracle

import (
	"context"
	"fmt"
	"strconv"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/avast/retry-go/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Returns the execution payload envelope the builder published for the beacon
// block of the given slot
func (o *Onchain) GetExecutionPayloadEnvelope(slot uint64, opts ...retry.Option) (*gloas.ExecutionPayloadEnvelope, error) {
	slotStr := strconv.FormatUint(slot, 10)
	var envelope *api.Response[*spec.VersionedSignedExecutionPayloadEnvelope]

	err := retry.Do(func() error {
		var err error
		envelope, err = o.ConsensusClient.SignedExecutionPayloadEnvelope(context.Background(), &api.SignedExecutionPayloadEnvelopeOpts{
			Block: slotStr,
		})
		if err != nil {
			log.Warn("Failed attempt to fetch execution payload envelope at slot ", slotStr, ": ", err.Error(), " Retrying...")
			return errors.Wrap(err, "error fetching execution payload envelope at slot "+slotStr)
		}
		return nil
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.Wrap(err, "could not fetch execution payload envelope at slot "+slotStr)
	}
	if envelope.Data == nil || envelope.Data.Gloas == nil || envelope.Data.Gloas.Message == nil {
		return nil, errors.New("empty execution payload envelope at slot " + slotStr)
	}
	return envelope.Data.Gloas.Message, nil
}

// Returns whether the execution payload the bid of the given slot committed to
// became part of the chain, ie whether the slot is full rather than empty
func (o *Onchain) IsExecutionPayloadIncluded(slot uint64, bid *gloas.ExecutionPayloadBid, opts ...retry.Option) (bool, error) {
	finalized, err := o.FinalizedBeaconBlockHeader(opts...)
	if err != nil {
		return false, errors.Wrap(err, "could not get finalized beacon block header")
	}
	blockAtSlot := func(slot uint64) (*spec.VersionedSignedBeaconBlock, error) {
		return o.GetConsensusBlockAtSlot(slot, opts...)
	}
	return isExecutionPayloadIncluded(slot, uint64(finalized.Header.Message.Slot), bid, blockAtSlot)
}

// An execution payload is included when the next beacon block builds on it,
// ie when the parent payload of the next bid is this one. Otherwise the next
// bid builds on the payload before it, and this slot is empty. Only finalized
// beacon blocks are considered, so the answer can't be reorged away.
func isExecutionPayloadIncluded(
	slot uint64,
	finalizedSlot uint64,
	bid *gloas.ExecutionPayloadBid,
	blockAtSlot func(uint64) (*spec.VersionedSignedBeaconBlock, error)) (bool, error) {

	for nextSlot := slot + 1; nextSlot <= finalizedSlot; nextSlot++ {
		nextBlock, err := blockAtSlot(nextSlot)
		if err != nil {
			return false, errors.Wrap(err, fmt.Sprintf("could not get beacon block at slot %d", nextSlot))
		}
		// Missed slot, keep looking
		if nextBlock == nil {
			continue
		}
		if nextBlock.Gloas == nil {
			return false, errors.New(fmt.Sprintf("beacon block at slot %d is not a Gloas one", nextSlot))
		}
		nextBid := nextBlock.Gloas.Message.Body.SignedExecutionPayloadBid.Message
		return nextBid.ParentBlockHash == bid.BlockHash, nil
	}
	return false, errors.New(fmt.Sprintf("no finalized beacon block after slot %d yet, "+
		"cant tell whether its execution payload was included", slot))
}

// Completes a Gloas full block with its execution payload. Returns false when
// the slot is empty, in which case there is no execution block to read events,
// txs or receipts from.
func (o *Onchain) fetchGloasExecutionPayload(fullBlock *FullBlock) bool {
	slot := fullBlock.GetSlotUint64()

	included, err := o.IsExecutionPayloadIncluded(slot, fullBlock.GetExecutionPayloadBid())
	if err != nil {
		log.Fatal("could not tell whether the execution payload of slot ", slot, " was included: ", err)
	}
	if !included {
		log.WithFields(log.Fields{
			"Slot":         slot,
			"BuilderIndex": fullBlock.GetExecutionPayloadBid().BuilderIndex,
		}).Info("Empty slot: the beacon block was proposed but its execution payload was not included")
		return false
	}

	envelope, err := o.GetExecutionPayloadEnvelope(slot)
	if err != nil {
		log.Fatal("could not get the execution payload of full slot ", slot, ": ", err)
	}
	fullBlock.SetExecutionPayload(envelope)
	return true
}
