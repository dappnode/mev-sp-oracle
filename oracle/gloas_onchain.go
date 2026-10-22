package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/avast/retry-go/v4"
	"github.com/dappnode/mev-sp-oracle/constants"
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
// became part of the chain, ie whether the slot is full rather than empty, and
// the slot of the next beacon block, which decided it
func (o *Onchain) IsExecutionPayloadIncluded(slot uint64, bid *gloas.ExecutionPayloadBid, opts ...retry.Option) (bool, uint64, error) {
	finalized, err := o.FinalizedBeaconBlockHeader(opts...)
	if err != nil {
		return false, 0, errors.Wrap(err, "could not get finalized beacon block header")
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
	blockAtSlot func(uint64) (*spec.VersionedSignedBeaconBlock, error)) (bool, uint64, error) {

	for nextSlot := slot + 1; nextSlot <= finalizedSlot; nextSlot++ {
		nextBlock, err := blockAtSlot(nextSlot)
		if err != nil {
			return false, 0, errors.Wrap(err, fmt.Sprintf("could not get beacon block at slot %d", nextSlot))
		}
		// Missed slot, keep looking
		if nextBlock == nil {
			continue
		}
		if nextBlock.Gloas == nil {
			return false, 0, errors.New(fmt.Sprintf("beacon block at slot %d is not a Gloas one", nextSlot))
		}
		nextBid := nextBlock.Gloas.Message.Body.SignedExecutionPayloadBid.Message
		return nextBid.ParentBlockHash == bid.BlockHash, nextSlot, nil
	}
	return false, 0, errors.New(fmt.Sprintf("no finalized beacon block after slot %d yet, "+
		"cant tell whether its execution payload was included", slot))
}

// Completes a Gloas full block with what happened to its bid: whether its
// execution payload was included, and whether the payment it promised to the
// pool is paid. Returns false when the slot is empty, in which case there is
// no execution block to read events, txs or receipts from.
func (o *Onchain) fetchGloasSlot(fullBlock *FullBlock) bool {
	slot := fullBlock.GetSlotUint64()
	bid := fullBlock.GetExecutionPayloadBid()

	included, nextBlockSlot, err := o.IsExecutionPayloadIncluded(slot, bid)
	if err != nil {
		log.Fatal("could not tell whether the execution payload of slot ", slot, " was included: ", err)
	}

	// Whether the payment is made has to be exact: a payment credited but never
	// made would be counted in flight forever, hiding that the pool lacks it
	if fullBlock.BidPromisesPaymentTo(o.PoolAddress) {
		var paid bool
		if included {
			paid, err = o.IsIncludedBidPaid(fullBlock, nextBlockSlot)
		} else {
			paid, err = o.IsWithheldBidPaid(fullBlock)
		}
		if err != nil {
			log.Fatal("could not tell whether the bid of slot ", slot, " is paid: ", err)
		}
		fullBlock.SetBuilderPayment(paid)
	}

	if !included {
		log.WithFields(log.Fields{
			"Slot":         slot,
			"BuilderIndex": bid.BuilderIndex,
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

// Returns whether the protocol pays the bid of a full slot, reading the beacon
// state right after its beacon block. See isIncludedBidPaid
func (o *Onchain) IsIncludedBidPaid(fullBlock *FullBlock, nextBlockSlot uint64, opts ...retry.Option) (bool, error) {
	slot := fullBlock.GetSlotUint64()
	var pendingPayments []*gloas.BuilderPendingPayment

	// When the payment is made regardless of the state, don't read it
	if nextBlockSlot/constants.SlotsInEpoch < slot/constants.SlotsInEpoch+2 {
		var err error
		pendingPayments, err = o.GetBuilderPendingPayments(strconv.FormatUint(slot, 10), opts...)
		if err != nil {
			return false, err
		}
	}
	return isIncludedBidPaid(fullBlock.GetExecutionPayloadBid(), fullBlock.GetProposerIndex(), nextBlockSlot, pendingPayments)
}

// Returns whether the protocol pays the bid of an empty slot, reading the
// beacon state the protocol bases that decision on. See isWithheldBidPaid
func (o *Onchain) IsWithheldBidPaid(fullBlock *FullBlock, opts ...retry.Option) (bool, error) {
	stateID := strconv.FormatUint(builderPaymentDecisionSlot(fullBlock.GetSlotUint64()), 10)

	pendingPayments, err := o.GetBuilderPendingPayments(stateID, opts...)
	if err != nil {
		return false, err
	}
	totalActiveBalance, err := o.GetTotalActiveBalance(stateID, opts...)
	if err != nil {
		return false, err
	}
	return isWithheldBidPaid(fullBlock.GetExecutionPayloadBid(), fullBlock.GetProposerIndex(), pendingPayments, totalActiveBalance)
}

// Returns the builder payments of the last two epochs at the given beacon
// state. go-eth2-client has no call for this endpoint yet
func (o *Onchain) GetBuilderPendingPayments(stateID string, opts ...retry.Option) ([]*gloas.BuilderPendingPayment, error) {
	var payments []*gloas.BuilderPendingPayment
	err := retry.Do(func() error {
		var err error
		payments, err = fetchBuilderPendingPayments(o.ConsensusClient.Address(), stateID)
		if err != nil {
			log.Warn("Failed attempt to fetch builder pending payments at state ", stateID, ": ", err.Error(), " Retrying...")
		}
		return err
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return nil, errors.Wrap(err, "could not fetch builder pending payments at state "+stateID)
	}
	return payments, nil
}

func fetchBuilderPendingPayments(beaconNodeUrl string, stateID string) ([]*gloas.BuilderPendingPayment, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/states/%s/builder_pending_payments", beaconNodeUrl, stateID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create request")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("non-200 response: %d - %s", resp.StatusCode, string(body)))
	}

	var response struct {
		Data []*gloas.BuilderPendingPayment `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "could not decode response")
	}
	return response.Data, nil
}

// Returns the sum of the effective balances of the validators active at the
// given beacon state, ie get_total_active_balance in the spec. It fetches the
// whole active set, which is heavy, so it is only meant for empty slots whose
// bid promised a payment to the pool.
func (o *Onchain) GetTotalActiveBalance(stateID string, opts ...retry.Option) (phase0.Gwei, error) {
	var validators *api.Response[map[phase0.ValidatorIndex]*v1.Validator]
	err := retry.Do(func() error {
		var err error
		validators, err = o.ConsensusClient.Validators(context.Background(), &api.ValidatorsOpts{
			State: stateID,
			ValidatorStates: []v1.ValidatorState{
				v1.ValidatorStateActiveOngoing,
				v1.ValidatorStateActiveExiting,
				v1.ValidatorStateActiveSlashed,
			},
		})
		if err != nil {
			log.Warn("Failed attempt to fetch active validators at state ", stateID, ": ", err.Error(), " Retrying...")
		}
		return err
	}, o.GetRetryOpts(opts)...)

	if err != nil {
		return 0, errors.Wrap(err, "could not fetch active validators at state "+stateID)
	}
	return totalActiveBalance(validators.Data), nil
}

// Sums the effective balances of the active validators. The beacon node was
// asked for active ones only, but filtering again costs nothing
func totalActiveBalance(validators map[phase0.ValidatorIndex]*v1.Validator) phase0.Gwei {
	total := phase0.Gwei(0)
	for _, validator := range validators {
		if validator.Status.IsActive() {
			total += validator.Validator.EffectiveBalance
		}
	}
	return total
}
