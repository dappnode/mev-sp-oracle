package oracle

import (
	"fmt"
	"math/big"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/constants"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Glamsterdam's consensus fork, Gloas, takes the execution payload out of the
// beacon block (EIP-7732). The beacon block carries a builder's bid instead,
// which commits to the payload by its hash, and the builder publishes the
// payload separately, in an envelope, a few seconds later. So a slot can end:
//   - full: its beacon block and its execution payload are part of the chain
//   - empty: only its beacon block is. There is no execution block, no txs
//   - missed: nothing was proposed, as before Gloas

// First slot of the Gloas fork on each chain. A chain must be listed before it
// forks: the oracle stops at a Gloas beacon block it can't place rather than
// processing it, so history is never computed with the wrong rules.
var GloasForkSlot = map[uint64]uint64{
	// Epoch 132352, https://blog.ethereum.org/2026/10/16/glamsterdam-hoodi-announcement
	HoodiChainId: 4235264,
}

// Whether the slot belongs to the Gloas fork on the given chain
func IsGloasSlot(chainId uint64, slot uint64) bool {
	forkSlot, found := GloasForkSlot[chainId]
	return found && slot >= forkSlot
}

// Returns the slot that must be finalized before the given slot can be
// processed. Before Gloas that is the slot itself. From Gloas on, the outcome of
// a slot is decided later: whether its execution payload was included is decided
// by the next beacon block, and whether a bid whose payload was withheld is still
// paid is decided at the end of the following epoch. Waiting until that epoch is
// finalized covers both, at the cost of trailing finality by about one epoch.
func SettlementSlot(chainId uint64, slot uint64) uint64 {
	if !IsGloasSlot(chainId, slot) {
		return slot
	}
	return builderPaymentDecisionSlot(slot)
}

// The last slot of the epoch after the given slot's one. The protocol decides
// whether a withheld payload's bid is paid in the epoch transition right after
// it, so its beacon state holds what that decision is based on
func builderPaymentDecisionSlot(slot uint64) uint64 {
	epoch := slot / constants.SlotsInEpoch
	return (epoch+2)*constants.SlotsInEpoch - 1
}

// BUILDER_PAYMENT_THRESHOLD_NUMERATOR and DENOMINATOR in the Gloas spec: the
// share of a slot's stake that must vote for a beacon block in its own slot for
// its bid to be paid when the payload is withheld
const (
	builderPaymentThresholdNumerator   = 6
	builderPaymentThresholdDenominator = 10
)

// EFFECTIVE_BALANCE_INCREMENT, the floor of the total active balance
const effectiveBalanceIncrement = phase0.Gwei(1_000_000_000)

// get_builder_payment_quorum_threshold in the Gloas spec
func builderPaymentQuorum(totalActiveBalance phase0.Gwei) phase0.Gwei {
	perSlotBalance := max(totalActiveBalance, effectiveBalanceIncrement) / phase0.Gwei(constants.SlotsInEpoch)
	return perSlotBalance * builderPaymentThresholdNumerator / builderPaymentThresholdDenominator
}

// Whether the protocol pays a bid whose execution payload was withheld. It
// does if the votes its beacon block got in its own slot reach the quorum,
// unless the payment was cancelled before, as slashing the proposer does.
//
// pendingPayments and totalActiveBalance must come from the beacon state at
// builderPaymentDecisionSlot, which is just before the protocol decides it in
// process_builder_pending_payments. Effective balances only change after that
// step, so the quorum calculated here is the one the protocol uses.
func isWithheldBidPaid(bid *gloas.ExecutionPayloadBid, proposerIndex phase0.ValidatorIndex,
	pendingPayments []*gloas.BuilderPendingPayment, totalActiveBalance phase0.Gwei) (bool, error) {

	if uint64(len(pendingPayments)) != 2*constants.SlotsInEpoch {
		return false, errors.New(fmt.Sprintf("expected %d builder pending payments, got %d",
			2*constants.SlotsInEpoch, len(pendingPayments)))
	}

	// The first half holds the payments of the previous epoch, the bid's one
	payment := pendingPayments[uint64(bid.Slot)%constants.SlotsInEpoch]

	// An empty entry means the payment was cancelled
	if payment == nil || payment.Withdrawal == nil || payment.Withdrawal.Amount == 0 {
		return false, nil
	}

	if payment.Withdrawal.BuilderIndex != bid.BuilderIndex ||
		payment.Withdrawal.Amount != bid.Value ||
		payment.Withdrawal.FeeRecipient != bid.FeeRecipient ||
		payment.ProposerIndex != proposerIndex {
		return false, errors.New(fmt.Sprintf("builder pending payment of slot %d does not match its bid. "+
			"payment: %s, bid: %s", bid.Slot, payment.String(), bid.String()))
	}

	return payment.Weight >= builderPaymentQuorum(totalActiveBalance), nil
}

// Whether the beacon block of this slot is a Gloas one, ie carries a bid
// instead of the execution payload
func (b *FullBlock) IsGloas() bool {
	return b.ConsensusBlock != nil && b.ConsensusBlock.Version >= spec.DataVersionGloas
}

// Whether the slot has an execution payload in the chain, and with it txs,
// events and an execution block number. False for missed and empty slots
func (b *FullBlock) HasExecutionPayload() bool {
	if b.ConsensusBlock == nil {
		return false
	}
	return !b.IsGloas() || b.ExecutionPayload != nil
}

// Returns the bid the proposer included in its beacon block. Only for Gloas
func (b *FullBlock) GetExecutionPayloadBid() *gloas.ExecutionPayloadBid {
	if !b.IsGloas() {
		log.Fatal("only Gloas beacon blocks carry a bid, slot: ", b.ConsensusDuty.Slot)
	}
	return b.ConsensusBlock.Gloas.Message.Body.SignedExecutionPayloadBid.Message
}

// Whether the proposer built its own execution payload instead of picking a
// builder's bid
func (b *FullBlock) IsSelfBuilt() bool {
	return b.GetExecutionPayloadBid().BuilderIndex == gloas.BuilderIndexSelfBuild
}

// Whether the bid promised a payment to the given address. A self-built
// payload promises none, its proposer collects the tips instead
func (b *FullBlock) BidPromisesPaymentTo(address string) bool {
	bid := b.GetExecutionPayloadBid()
	return !b.IsSelfBuilt() && bid.Value > 0 && utils.Equals(bid.FeeRecipient.String(), address)
}

// Records the payment the bid promised to the pool, and whether the protocol
// pays it
func (b *FullBlock) SetBuilderPayment(paid bool) {
	bid := b.GetExecutionPayloadBid()
	b.BuilderPayment = &BuilderPayment{
		Slot:          uint64(bid.Slot),
		ProposerIndex: b.GetProposerIndexUint64(),
		BuilderIndex:  uint64(bid.BuilderIndex),
		AmountWei:     utils.GweiToWei(uint64(bid.Value)),
		Paid:          paid,
	}
}

// Adds the execution payload of a full slot. It must be the payload the bid in
// the beacon block committed to
func (b *FullBlock) SetExecutionPayload(envelope *gloas.ExecutionPayloadEnvelope) {
	if envelope == nil || envelope.Payload == nil {
		log.Fatal("execution payload envelope can't be nil")
	}

	bid := b.GetExecutionPayloadBid()
	payload := envelope.Payload

	if payload.BlockHash != bid.BlockHash {
		log.Fatal("Execution payload does not match the bid. Payload: ", payload.BlockHash.String(),
			" bid: ", bid.BlockHash.String())
	}
	if envelope.BuilderIndex != bid.BuilderIndex {
		log.Fatal("Execution payload was not published by the builder of the bid. Envelope: ",
			envelope.BuilderIndex, " bid: ", bid.BuilderIndex)
	}
	if payload.SlotNumber != uint64(bid.Slot) {
		log.Fatal("Execution payload slot does not match the bid slot: ", payload.SlotNumber, " vs ", bid.Slot)
	}

	b.ExecutionPayload = payload
}

// Returns the execution payload of a Gloas slot. Stops when the slot is empty:
// any caller reaching here should have checked HasExecutionPayload first
func (b *FullBlock) gloasExecutionPayload() *gloas.ExecutionPayload {
	if b.ExecutionPayload == nil {
		log.Fatal("Slot ", b.ConsensusDuty.Slot, " is empty, it has no execution payload")
	}
	return b.ExecutionPayload
}

// Checks that the beacon block version matches the fork the slot belongs to.
// A Gloas beacon block on a chain with no fork slot configured, or with the
// wrong one, means GloasForkSlot has to be updated before processing on
func (b *FullBlock) checkGloasForkSlot(blockVersion spec.DataVersion, slot uint64) {
	isGloasBlock := blockVersion >= spec.DataVersionGloas
	if isGloasBlock != IsGloasSlot(b.ChainId, slot) {
		forkSlot, configured := GloasForkSlot[b.ChainId]
		log.WithFields(log.Fields{
			"Slot":                slot,
			"ChainId":             b.ChainId,
			"BlockVersion":        blockVersion.String(),
			"GloasForkSlot":       forkSlot,
			"GloasForkConfigured": configured,
		}).Fatal("Beacon block version does not match the configured Gloas fork slot, update GloasForkSlot")
	}
}

// How a Gloas proposal went for the pool
type gloasProposal struct {
	// Counts as missed: nothing reached the pool and no execution payload was
	// included, so the proposer's beacon block was most likely late
	missed bool

	// The proposer pointed its proposal at the pool
	correctFeeRecipient bool

	// Everything the proposal sent to the pool, and its type
	reward     *big.Int
	rewardType RewardType

	// The part of reward paid from the builder's deposit, nil if none. It
	// reaches the pool as a withdrawal instead of through the payload
	builderPaymentWei *big.Int
}

// Classifies a Gloas proposal and adds up what it sent to the pool. The
// proposer is accountable for what it chose: the bid and where it pays, or the
// fee recipient when it built its own payload. It is not for what the builder
// failed to do, like withholding the payload or skipping a promised payment.
func (b *FullBlock) gloasProposal(poolAddress string, isSubscriber bool) gloasProposal {
	if b.IsSelfBuilt() {
		// The proposer's own payload, judged as before Gloas
		if !b.HasExecutionPayload() {
			return gloasProposal{missed: true, reward: big.NewInt(0)}
		}
		reward, sent, rewardType := b.GetSentRewardAndType(poolAddress, isSubscriber)
		return gloasProposal{correctFeeRecipient: sent, reward: reward, rewardType: rewardType}
	}

	proposal := gloasProposal{reward: big.NewInt(0), rewardType: MevBlock}

	if b.BuilderPayment != nil && b.BuilderPayment.Paid {
		proposal.builderPaymentWei = new(big.Int).Set(b.BuilderPayment.AmountWei)
		proposal.reward.Add(proposal.reward, proposal.builderPaymentWei)
	}

	// Whatever the payload itself sent to the pool: a tx paying the bid's
	// execution_payment, or the tips if the builder made the pool its fee
	// recipient. Builders keep the tips otherwise, so they don't count
	payloadPaidPool := false
	if b.HasExecutionPayload() {
		payloadReward, sent, payloadRewardType := b.GetSentRewardAndType(poolAddress, false)
		if sent {
			payloadPaidPool = true
			proposal.reward.Add(proposal.reward, payloadReward)
			proposal.rewardType = payloadRewardType
		}
	}

	bid := b.GetExecutionPayloadBid()
	if utils.Equals(bid.FeeRecipient.String(), poolAddress) {
		// A withheld payload whose payment was dropped sent nothing
		if !b.HasExecutionPayload() && proposal.builderPaymentWei == nil {
			return gloasProposal{missed: true, reward: big.NewInt(0)}
		}
		proposal.correctFeeRecipient = true
		return proposal
	}

	// The bid paid someone else. Should the payload still have paid the pool,
	// that ETH is credited as before Gloas, otherwise it would be left
	// unallocated
	if payloadPaidPool {
		proposal.correctFeeRecipient = true
		return proposal
	}
	return gloasProposal{correctFeeRecipient: false, reward: utils.GweiToWei(uint64(bid.Value)), rewardType: MevBlock}
}
