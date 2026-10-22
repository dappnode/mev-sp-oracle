package oracle

import (
	"errors"
	"math/big"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// First slot of the Gloas fork on Hoodi, the only chain with a fork slot yet
var testGloasSlot = GloasForkSlot[HoodiChainId]

// The address of a builder, which builders set as fee recipient of their
// payloads to collect the tips
const testBuilderAddress = "0x4838B106FCe9647Bdf1E7877BF73cE8B0BAD5f97"

// Runs f expecting it to stop the oracle with log.Fatal. The exit is turned
// into a panic so the test survives it
func requireFatal(t *testing.T, f func()) {
	t.Helper()
	logger := log.StandardLogger()
	previousExit := logger.ExitFunc
	logger.ExitFunc = func(int) { panic("log.Fatal") }
	defer func() { logger.ExitFunc = previousExit }()
	require.PanicsWithValue(t, "log.Fatal", f)
}

func executionAddress(address string) bellatrix.ExecutionAddress {
	var executionAddress bellatrix.ExecutionAddress
	copy(executionAddress[:], common.HexToAddress(address).Bytes())
	return executionAddress
}

// A bid of the given builder paying valueGwei to feeRecipient. Its payload
// builds on the payload with hash parentHash, and has hash blockHash
func gloasBid(slot uint64, builderIndex gloas.BuilderIndex, valueGwei uint64, feeRecipient string,
	parentHash byte, blockHash byte) *gloas.ExecutionPayloadBid {
	return &gloas.ExecutionPayloadBid{
		Slot:            phase0.Slot(slot),
		BuilderIndex:    builderIndex,
		Value:           phase0.Gwei(valueGwei),
		FeeRecipient:    executionAddress(feeRecipient),
		ParentBlockHash: phase0.Hash32{parentHash},
		BlockHash:       phase0.Hash32{blockHash},
	}
}

// A Gloas beacon block on Hoodi, proposed by validatorIndex and carrying bid
func gloasBeaconBlock(slot uint64, validatorIndex phase0.ValidatorIndex, bid *gloas.ExecutionPayloadBid) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionGloas,
		Gloas: &gloas.SignedBeaconBlock{Message: &gloas.BeaconBlock{
			Slot:          phase0.Slot(slot),
			ProposerIndex: validatorIndex,
			Body: &gloas.BeaconBlockBody{
				SignedExecutionPayloadBid: &gloas.SignedExecutionPayloadBid{Message: bid},
			},
		}},
	}
}

// A FullBlock for a Gloas slot whose payload has not been set yet, ie that is
// empty until SetExecutionPayload is called. The proposer has 0x01 credentials
func gloasFullBlock(t *testing.T, slot uint64, validatorIndex phase0.ValidatorIndex, bid *gloas.ExecutionPayloadBid) *FullBlock {
	t.Helper()
	credentials := make([]byte, 32)
	credentials[0] = 1
	copy(credentials[12:], common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())

	fullBlock := NewFullBlock(&v1.ProposerDuty{Slot: phase0.Slot(slot), ValidatorIndex: validatorIndex},
		&v1.Validator{Index: validatorIndex, Validator: &phase0.Validator{WithdrawalCredentials: credentials}},
		HoodiChainId)
	fullBlock.SetConsensusBlock(gloasBeaconBlock(slot, validatorIndex, bid))
	return fullBlock
}

// The envelope the builder of bid publishes, with the given execution payload
// contents
func gloasEnvelope(bid *gloas.ExecutionPayloadBid, blockNumber uint64, feeRecipient string,
	txs []bellatrix.Transaction, withdrawals []*capella.Withdrawal) *gloas.ExecutionPayloadEnvelope {
	return &gloas.ExecutionPayloadEnvelope{
		BuilderIndex: bid.BuilderIndex,
		Payload: &gloas.ExecutionPayload{
			BlockHash:     bid.BlockHash,
			SlotNumber:    uint64(bid.Slot),
			BlockNumber:   blockNumber,
			FeeRecipient:  executionAddress(feeRecipient),
			GasUsed:       1_500_000,
			BaseFeePerGas: uint256.NewInt(7_000_000_000),
			Transactions:  txs,
			Withdrawals:   withdrawals,
		},
	}
}

func Test_IsGloasSlot(t *testing.T) {
	require.False(t, IsGloasSlot(HoodiChainId, testGloasSlot-1))
	require.True(t, IsGloasSlot(HoodiChainId, testGloasSlot))
	require.True(t, IsGloasSlot(HoodiChainId, testGloasSlot+100_000))

	// Not scheduled on mainnet yet
	require.False(t, IsGloasSlot(MainnetChainId, 20_000_000))
}

func Test_SettlementSlot(t *testing.T) {
	// Before Gloas a finalized slot is settled
	require.Equal(t, testGloasSlot-1, settlementSlot(HoodiChainId, testGloasSlot-1))
	require.Equal(t, uint64(20_000_000), settlementSlot(MainnetChainId, 20_000_000))

	// From Gloas on, wait until the epoch after the slot's one is finalized.
	// The fork slot is the first of epoch 132352
	require.Equal(t, uint64(132352*32), testGloasSlot)
	lastSlotOfNextEpoch := uint64(132354*32 - 1)
	require.Equal(t, lastSlotOfNextEpoch, settlementSlot(HoodiChainId, testGloasSlot))
	require.Equal(t, lastSlotOfNextEpoch, settlementSlot(HoodiChainId, testGloasSlot+17))
	require.Equal(t, lastSlotOfNextEpoch, settlementSlot(HoodiChainId, testGloasSlot+31))
	require.Equal(t, lastSlotOfNextEpoch+32, settlementSlot(HoodiChainId, testGloasSlot+32))
}

func Test_LatestSettledSlot(t *testing.T) {
	// Before Gloas, and on chains without it, every finalized slot is settled
	require.Equal(t, uint64(20_000_000), LatestSettledSlot(MainnetChainId, 20_000_000))
	require.Equal(t, testGloasSlot-1, LatestSettledSlot(HoodiChainId, testGloasSlot-1))

	// Right after the fork only the slots before it are settled
	require.Equal(t, testGloasSlot-1, LatestSettledSlot(HoodiChainId, testGloasSlot))
	require.Equal(t, testGloasSlot-1, LatestSettledSlot(HoodiChainId, testGloasSlot+62))

	// The first Gloas epoch settles with the last slot of the next one
	require.Equal(t, testGloasSlot+31, LatestSettledSlot(HoodiChainId, testGloasSlot+63))
	// An oracle in sync trails a finalized epoch boundary by 33 slots
	require.Equal(t, testGloasSlot+64-33, LatestSettledSlot(HoodiChainId, testGloasSlot+64))
}

// LatestSettledSlot is the inverse of settlementSlot: the latest slot that
// settlementSlot allows processing. Check it against a brute force search
func Test_LatestSettledSlot_MatchesSettlementSlot(t *testing.T) {
	for finalized := testGloasSlot - 70; finalized < testGloasSlot+300; finalized++ {
		expected := finalized
		for settlementSlot(HoodiChainId, expected) > finalized {
			expected--
		}
		require.Equal(t, expected, LatestSettledSlot(HoodiChainId, finalized), "finalized slot %d", finalized)
	}
}

func Test_GloasBlock_FullSlot(t *testing.T) {
	bid := gloasBid(testGloasSlot, 42, 48_200_000, testPoolAddress, 0xaa, 0xbb)
	fullBlock := gloasFullBlock(t, testGloasSlot, 1000, bid)

	require.True(t, fullBlock.IsGloas())
	require.False(t, fullBlock.IsSelfBuilt())
	require.Equal(t, bid, fullBlock.GetExecutionPayloadBid())
	require.Equal(t, phase0.ValidatorIndex(1000), fullBlock.GetProposerIndex())
	require.Equal(t, testGloasSlot, fullBlock.GetSlotUint64())

	withdrawals := []*capella.Withdrawal{{Index: 9, ValidatorIndex: 3, Address: executionAddress(testPoolAddress), Amount: 5}}
	txs := []bellatrix.Transaction{{0x01}, {0x02}}
	fullBlock.SetExecutionPayload(gloasEnvelope(bid, 5000, testBuilderAddress, txs, withdrawals))

	require.True(t, fullBlock.HasExecutionPayload())
	require.Equal(t, testBuilderAddress, fullBlock.GetFeeRecipient())
	require.Equal(t, uint64(5000), fullBlock.GetBlockNumber())
	require.Equal(t, uint64(1_500_000), fullBlock.GetGasUsed())
	require.Equal(t, big.NewInt(7_000_000_000), fullBlock.GetBaseFeePerGasBigInt())
	require.Equal(t, txs, fullBlock.GetBlockTransactions())

	gotWithdrawals, err := fullBlock.GetWithdrawals()
	require.NoError(t, err)
	require.Equal(t, withdrawals, gotWithdrawals)

	blockHash, err := fullBlock.GetExecutionBlockHash()
	require.NoError(t, err)
	require.Equal(t, bid.BlockHash, blockHash)
}

func Test_GloasBlock_EmptySlot(t *testing.T) {
	bid := gloasBid(testGloasSlot, 42, 48_200_000, testPoolAddress, 0xaa, 0xbb)
	fullBlock := gloasFullBlock(t, testGloasSlot, 1000, bid)

	require.True(t, fullBlock.IsGloas())
	require.False(t, fullBlock.HasExecutionPayload())
	require.Empty(t, fullBlock.GetDonations(testPoolAddress))

	// Reading the execution block of an empty slot is a bug in the caller
	requireFatal(t, func() { fullBlock.GetBlockNumber() })
	requireFatal(t, func() { fullBlock.GetFeeRecipient() })
}

func Test_GloasBlock_SelfBuilt(t *testing.T) {
	bid := gloasBid(testGloasSlot, gloas.BuilderIndexSelfBuild, 0, testPoolAddress, 0xaa, 0xbb)
	fullBlock := gloasFullBlock(t, testGloasSlot, 1000, bid)
	require.True(t, fullBlock.IsSelfBuilt())
}

func Test_HasExecutionPayload_BeforeGloas(t *testing.T) {
	missed := NewFullBlock(&v1.ProposerDuty{Slot: 100, ValidatorIndex: 7},
		&v1.Validator{Index: 7, Validator: &phase0.Validator{}}, MainnetChainId)
	require.False(t, missed.HasExecutionPayload())
	require.False(t, missed.IsGloas())

	proposed := buildMevBlock(t, 100, 1000, 7, testPoolAddress, big.NewInt(1))
	require.True(t, proposed.HasExecutionPayload())
	require.False(t, proposed.IsGloas())
}

func Test_SetExecutionPayload_MustMatchTheBid(t *testing.T) {
	bid := gloasBid(testGloasSlot, 42, 48_200_000, testPoolAddress, 0xaa, 0xbb)

	tests := []struct {
		name   string
		mutate func(*gloas.ExecutionPayloadEnvelope)
	}{
		{"other payload", func(e *gloas.ExecutionPayloadEnvelope) { e.Payload.BlockHash = phase0.Hash32{0xcc} }},
		{"other builder", func(e *gloas.ExecutionPayloadEnvelope) { e.BuilderIndex = 43 }},
		{"other slot", func(e *gloas.ExecutionPayloadEnvelope) { e.Payload.SlotNumber++ }},
		{"no payload", func(e *gloas.ExecutionPayloadEnvelope) { e.Payload = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullBlock := gloasFullBlock(t, testGloasSlot, 1000, bid)
			envelope := gloasEnvelope(bid, 5000, testBuilderAddress, nil, nil)
			tt.mutate(envelope)
			requireFatal(t, func() { fullBlock.SetExecutionPayload(envelope) })
			require.False(t, fullBlock.HasExecutionPayload())
		})
	}
}

// The oracle must stop on a beacon block that doesn't match the configured
// fork, instead of applying the wrong rules to it
func Test_SetConsensusBlock_ChecksGloasForkSlot(t *testing.T) {
	duty := func(slot uint64) *v1.ProposerDuty {
		return &v1.ProposerDuty{Slot: phase0.Slot(slot), ValidatorIndex: 7}
	}
	validator := &v1.Validator{Index: 7, Validator: &phase0.Validator{}}
	bid := gloasBid(testGloasSlot, 42, 1, testPoolAddress, 0xaa, 0xbb)

	t.Run("Gloas block on a chain without a fork slot", func(t *testing.T) {
		fullBlock := NewFullBlock(duty(testGloasSlot), validator, MainnetChainId)
		requireFatal(t, func() { fullBlock.SetConsensusBlock(gloasBeaconBlock(testGloasSlot, 7, bid)) })
	})

	t.Run("Gloas block before the fork slot", func(t *testing.T) {
		fullBlock := NewFullBlock(duty(testGloasSlot-1), validator, HoodiChainId)
		requireFatal(t, func() { fullBlock.SetConsensusBlock(gloasBeaconBlock(testGloasSlot-1, 7, bid)) })
	})

	t.Run("Fulu block after the fork slot", func(t *testing.T) {
		fullBlock := NewFullBlock(duty(testGloasSlot), validator, HoodiChainId)
		fuluBlock := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionFulu,
			Fulu: &electra.SignedBeaconBlock{Message: &electra.BeaconBlock{Slot: phase0.Slot(testGloasSlot), ProposerIndex: 7,
				Body: &electra.BeaconBlockBody{ExecutionPayload: &deneb.ExecutionPayload{}}}}}
		requireFatal(t, func() { fullBlock.SetConsensusBlock(fuluBlock) })
	})

	t.Run("Gloas block from the fork slot", func(t *testing.T) {
		fullBlock := NewFullBlock(duty(testGloasSlot), validator, HoodiChainId)
		fullBlock.SetConsensusBlock(gloasBeaconBlock(testGloasSlot, 7, bid))
		require.True(t, fullBlock.IsGloas())
	})
}

func Test_IsExecutionPayloadIncluded(t *testing.T) {
	slot := testGloasSlot + 10
	bid := gloasBid(slot, 42, 1, testPoolAddress, 0xaa, 0xbb)

	// Next bids either build on this slot's payload (0xbb) or skip it, building
	// on the payload this slot built on (0xaa)
	buildsOnThisPayload := gloasBeaconBlock(slot+1, 8, gloasBid(slot+1, 43, 1, testPoolAddress, 0xbb, 0xcc))
	skipsThisPayload := gloasBeaconBlock(slot+1, 8, gloasBid(slot+1, 43, 1, testPoolAddress, 0xaa, 0xcc))

	chain := func(blocks map[uint64]*spec.VersionedSignedBeaconBlock) func(uint64) (*spec.VersionedSignedBeaconBlock, error) {
		return func(slot uint64) (*spec.VersionedSignedBeaconBlock, error) {
			return blocks[slot], nil
		}
	}

	t.Run("next beacon block builds on the payload", func(t *testing.T) {
		included, nextSlot, err := isExecutionPayloadIncluded(slot, slot+64, bid, chain(map[uint64]*spec.VersionedSignedBeaconBlock{
			slot + 1: buildsOnThisPayload,
		}))
		require.NoError(t, err)
		require.True(t, included)
		require.Equal(t, slot+1, nextSlot)
	})

	t.Run("next beacon block skips the payload", func(t *testing.T) {
		included, nextSlot, err := isExecutionPayloadIncluded(slot, slot+64, bid, chain(map[uint64]*spec.VersionedSignedBeaconBlock{
			slot + 1: skipsThisPayload,
		}))
		require.NoError(t, err)
		require.False(t, included)
		require.Equal(t, slot+1, nextSlot)
	})

	t.Run("missed slots are skipped until the next beacon block", func(t *testing.T) {
		buildsAfterMissedSlots := gloasBeaconBlock(slot+4, 8, gloasBid(slot+4, 43, 1, testPoolAddress, 0xbb, 0xcc))
		included, nextSlot, err := isExecutionPayloadIncluded(slot, slot+64, bid, chain(map[uint64]*spec.VersionedSignedBeaconBlock{
			slot + 4: buildsAfterMissedSlots,
		}))
		require.NoError(t, err)
		require.True(t, included)
		require.Equal(t, slot+4, nextSlot)
	})

	t.Run("only the first next beacon block decides", func(t *testing.T) {
		included, _, err := isExecutionPayloadIncluded(slot, slot+64, bid, chain(map[uint64]*spec.VersionedSignedBeaconBlock{
			slot + 1: skipsThisPayload,
			slot + 2: gloasBeaconBlock(slot+2, 9, gloasBid(slot+2, 44, 1, testPoolAddress, 0xbb, 0xdd)),
		}))
		require.NoError(t, err)
		require.False(t, included)
	})

	t.Run("no finalized beacon block after the slot yet", func(t *testing.T) {
		_, _, err := isExecutionPayloadIncluded(slot, slot+3, bid, chain(map[uint64]*spec.VersionedSignedBeaconBlock{
			slot + 4: buildsOnThisPayload,
		}))
		require.ErrorContains(t, err, "no finalized beacon block after slot")
	})

	t.Run("beacon node error", func(t *testing.T) {
		_, _, err := isExecutionPayloadIncluded(slot, slot+64, bid, func(uint64) (*spec.VersionedSignedBeaconBlock, error) {
			return nil, errors.New("connection refused")
		})
		require.ErrorContains(t, err, "connection refused")
	})

	t.Run("next beacon block is not a Gloas one", func(t *testing.T) {
		_, _, err := isExecutionPayloadIncluded(slot, slot+64, bid, chain(map[uint64]*spec.VersionedSignedBeaconBlock{
			slot + 1: {Version: spec.DataVersionFulu, Fulu: &electra.SignedBeaconBlock{}},
		}))
		require.ErrorContains(t, err, "not a Gloas one")
	})
}
