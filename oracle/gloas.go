package oracle

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/dappnode/mev-sp-oracle/constants"
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
	epoch := slot / constants.SlotsInEpoch
	return (epoch+2)*constants.SlotsInEpoch - 1
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
