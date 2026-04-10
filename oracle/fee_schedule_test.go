package oracle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// Helper to create a minimal Config for testing
func testConfig(network string, fee int) *Config {
	return &Config{
		Network:                  network,
		PoolAddress:              "0x0000000000000000000000000000000000000001",
		PoolFeesAddress:          "0x0000000000000000000000000000000000000002",
		PoolFeesPercentOver10000: fee,
		CheckPointSizeInSlots:    100,
		CollateralInWei:          big.NewInt(1000),
		DeployedSlot:             1000,
		DeployedBlock:            1000,
	}
}

// Helper to create a minimal Oracle for testing
func testOracle(network string, fee int) *Oracle {
	cfg := testConfig(network, fee)
	return NewOracle(cfg)
}

// Helper to create a FullBlock with an UpdatePoolFee event at a given slot
func fullBlockWithFeeEvent(slot uint64, newFee int64) *FullBlock {
	return &FullBlock{
		ConsensusDuty: &v1.ProposerDuty{
			Slot: phase0.Slot(slot),
		},
		Events: &Events{
			UpdatePoolFee: []*contract.ContractUpdatePoolFee{
				{NewPoolFee: big.NewInt(newFee)},
			},
		},
	}
}

// Helper to create a FullBlock with no events at a given slot
func fullBlockNoEvents(slot uint64) *FullBlock {
	return &FullBlock{
		ConsensusDuty: &v1.ProposerDuty{
			Slot: phase0.Slot(slot),
		},
		Events: &Events{},
	}
}

// Helper: serialize a state, compute its hash, set it, and return the bytes.
// Mimics what SaveToJson does.
func serializeStateWithHash(state *OracleState) ([]byte, error) {
	state.StateHash = ""
	jsonNoHash, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return nil, err
	}
	hashByte := sha256.Sum256(jsonNoHash)
	state.StateHash = hexutil.Encode(hashByte[:])
	jsonWithHash, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return nil, err
	}
	return jsonWithHash, nil
}

// =============================================================================
// getFeeSchedule tests
// =============================================================================

func Test_getFeeSchedule_KnownNetwork(t *testing.T) {
	schedule := getFeeSchedule(Mainnet)
	require.NotEmpty(t, schedule)
	fee, ok := schedule[14082460]
	require.True(t, ok)
	require.Equal(t, 500, fee)
}

func Test_getFeeSchedule_KnownNetworkHoodi(t *testing.T) {
	schedule := getFeeSchedule(Hoodi)
	require.NotEmpty(t, schedule)
	fee, ok := schedule[2801050]
	require.True(t, ok)
	require.Equal(t, 500, fee)
}

func Test_getFeeSchedule_UnknownNetwork(t *testing.T) {
	schedule := getFeeSchedule("unknown_network")
	require.Empty(t, schedule)
}

// =============================================================================
// applyFeeScheduleUpTo tests
// =============================================================================

func Test_applyFeeScheduleUpTo_BeforeAnyChange(t *testing.T) {
	cfg := testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 10000000) // before slot 14082460
	require.Equal(t, 700, cfg.PoolFeesPercentOver10000, "fee should remain initial value")
}

func Test_applyFeeScheduleUpTo_ExactlyAtChange(t *testing.T) {
	cfg := testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 14082460)
	require.Equal(t, 500, cfg.PoolFeesPercentOver10000, "fee should be updated to 500")
}

func Test_applyFeeScheduleUpTo_AfterChange(t *testing.T) {
	cfg := testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 20000000)
	require.Equal(t, 500, cfg.PoolFeesPercentOver10000, "fee should be updated to 500")
}

func Test_applyFeeScheduleUpTo_UnknownNetwork(t *testing.T) {
	cfg := testConfig("goerli", 1000)
	applyFeeScheduleUpTo(cfg, 99999999)
	require.Equal(t, 1000, cfg.PoolFeesPercentOver10000, "fee should remain unchanged for unknown network")
}

func Test_applyFeeScheduleUpTo_MultipleEntries_PicksLatest(t *testing.T) {
	// Temporarily add a second entry to test map ordering safety
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		14082460: 500,
		20000000: 300,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	// Slot before all changes
	cfg := testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 10000000)
	require.Equal(t, 700, cfg.PoolFeesPercentOver10000)

	// Slot between changes
	cfg = testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 17000000)
	require.Equal(t, 500, cfg.PoolFeesPercentOver10000)

	// Slot after all changes — must pick highest qualifying (20000000 -> 300)
	cfg = testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 25000000)
	require.Equal(t, 300, cfg.PoolFeesPercentOver10000)

	// Exactly at second change
	cfg = testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 20000000)
	require.Equal(t, 300, cfg.PoolFeesPercentOver10000)

	// Exactly at first change (second doesn't apply yet)
	cfg = testConfig(Mainnet, 700)
	applyFeeScheduleUpTo(cfg, 14082460)
	require.Equal(t, 500, cfg.PoolFeesPercentOver10000)
}

// Run the multi-entry test many times to catch Go map iteration randomness
func Test_applyFeeScheduleUpTo_MultipleEntries_Deterministic(t *testing.T) {
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		14082460: 500,
		20000000: 300,
		25000000: 200,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	for i := 0; i < 100; i++ {
		cfg := testConfig(Mainnet, 700)
		applyFeeScheduleUpTo(cfg, 30000000)
		require.Equal(t, 200, cfg.PoolFeesPercentOver10000,
			fmt.Sprintf("iteration %d: should pick highest slot (25000000 -> 200)", i))
	}
}

// =============================================================================
// validateFullBlockConfig tests — UpdatePoolFee event handling
// =============================================================================

func Test_validateFullBlockConfig_ExpectedFeeChange(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	block := fullBlockWithFeeEvent(14082460, 500)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)

	require.NoError(t, err)
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000, "config should be updated")
	require.Equal(t, 500, oracle.state.PoolFeesPercentOver10000, "state should be updated")
}

func Test_validateFullBlockConfig_UnexpectedSlot(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	// Fee event at slot not in the schedule
	block := fullBlockWithFeeEvent(99999999, 500)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected pool fee change")
	require.Equal(t, 700, oracle.cfg.PoolFeesPercentOver10000, "config should be unchanged")
	require.Equal(t, 700, oracle.state.PoolFeesPercentOver10000, "state should be unchanged")
}

func Test_validateFullBlockConfig_WrongFeeAtExpectedSlot(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	// Right slot but wrong fee value
	block := fullBlockWithFeeEvent(14082460, 300)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected pool fee change")
	require.Equal(t, 700, oracle.cfg.PoolFeesPercentOver10000, "config should be unchanged")
}

func Test_validateFullBlockConfig_NoFeeEvent(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	block := fullBlockNoEvents(14082460)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)

	require.NoError(t, err)
	require.Equal(t, 700, oracle.cfg.PoolFeesPercentOver10000, "config should be unchanged when no event")
}

func Test_validateFullBlockConfig_HoodiExpectedFeeChange(t *testing.T) {
	oracle := testOracle(Hoodi, 1000)

	block := fullBlockWithFeeEvent(2801050, 500)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)

	require.NoError(t, err)
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000)
	require.Equal(t, 500, oracle.state.PoolFeesPercentOver10000)
}

func Test_validateFullBlockConfig_MultipleFeeEventsInSameBlock(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	block := &FullBlock{
		ConsensusDuty: &v1.ProposerDuty{
			Slot: phase0.Slot(14082460),
		},
		Events: &Events{
			UpdatePoolFee: []*contract.ContractUpdatePoolFee{
				{NewPoolFee: big.NewInt(500)},
				{NewPoolFee: big.NewInt(400)},
			},
		},
	}
	err := oracle.validateFullBlockConfig(block, oracle.cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "more than one event of the same type")
}

// =============================================================================
// LoadFromBytes tests — state recovery with fee schedule reconciliation
// =============================================================================

func Test_LoadFromBytes_StateBefore_FeeChange(t *testing.T) {
	// Config starts with initial fee 700
	oracle := testOracle(Mainnet, 700)

	// State saved before the fee change, fee=700
	state := &OracleState{
		LatestProcessedSlot:      10000000,
		LatestProcessedBlock:     9000000,
		NextSlotToProcess:        10000001,
		PoolAccumulatedFees:      big.NewInt(0),
		Validators:               make(map[uint64]*ValidatorInfo),
		CommitedStates:           make(map[uint64]*OnchainState),
		SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
		UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
		EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
		Donations:                make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:           make([]SummarizedBlock, 0),
		MissedBlocks:             make([]SummarizedBlock, 0),
		WrongFeeBlocks:           make([]SummarizedBlock, 0),
		PoolFeesPercentOver10000: 700,
		PoolAddress:              oracle.cfg.PoolAddress,
		Network:                  Mainnet,
		PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
		DeployedBlock:            oracle.cfg.DeployedBlock,
		DeployedSlot:             oracle.cfg.DeployedSlot,
		CollateralInWei:          oracle.cfg.CollateralInWei,
	}

	rawBytes, err := serializeStateWithHash(state)
	require.NoError(t, err)

	found, err := oracle.LoadFromBytes(rawBytes)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 700, oracle.cfg.PoolFeesPercentOver10000)
	require.Equal(t, 700, oracle.state.PoolFeesPercentOver10000)
}

func Test_LoadFromBytes_StateAfter_FeeChange(t *testing.T) {
	// Config starts with initial fee 700
	oracle := testOracle(Mainnet, 700)

	// State saved after the fee change, fee=500
	state := &OracleState{
		LatestProcessedSlot:      20000000,
		LatestProcessedBlock:     18000000,
		NextSlotToProcess:        20000001,
		PoolAccumulatedFees:      big.NewInt(0),
		Validators:               make(map[uint64]*ValidatorInfo),
		CommitedStates:           make(map[uint64]*OnchainState),
		SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
		UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
		EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
		Donations:                make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:           make([]SummarizedBlock, 0),
		MissedBlocks:             make([]SummarizedBlock, 0),
		WrongFeeBlocks:           make([]SummarizedBlock, 0),
		PoolFeesPercentOver10000: 500,
		PoolAddress:              oracle.cfg.PoolAddress,
		Network:                  Mainnet,
		PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
		DeployedBlock:            oracle.cfg.DeployedBlock,
		DeployedSlot:             oracle.cfg.DeployedSlot,
		CollateralInWei:          oracle.cfg.CollateralInWei,
	}

	rawBytes, err := serializeStateWithHash(state)
	require.NoError(t, err)

	found, err := oracle.LoadFromBytes(rawBytes)
	require.NoError(t, err)
	require.True(t, found)
	// Config should have been reconciled to 500 via applyFeeScheduleUpTo
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000)
	require.Equal(t, 500, oracle.state.PoolFeesPercentOver10000)
}

func Test_LoadFromBytes_StateExactlyAt_FeeChange(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	state := &OracleState{
		LatestProcessedSlot:      14082460,
		LatestProcessedBlock:     13000000,
		NextSlotToProcess:        14082461,
		PoolAccumulatedFees:      big.NewInt(0),
		Validators:               make(map[uint64]*ValidatorInfo),
		CommitedStates:           make(map[uint64]*OnchainState),
		SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
		UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
		EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
		Donations:                make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:           make([]SummarizedBlock, 0),
		MissedBlocks:             make([]SummarizedBlock, 0),
		WrongFeeBlocks:           make([]SummarizedBlock, 0),
		PoolFeesPercentOver10000: 500,
		PoolAddress:              oracle.cfg.PoolAddress,
		Network:                  Mainnet,
		PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
		DeployedBlock:            oracle.cfg.DeployedBlock,
		DeployedSlot:             oracle.cfg.DeployedSlot,
		CollateralInWei:          oracle.cfg.CollateralInWei,
	}

	rawBytes, err := serializeStateWithHash(state)
	require.NoError(t, err)

	found, err := oracle.LoadFromBytes(rawBytes)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000)
}

func Test_LoadFromBytes_StateWithWrongFee_Rejected(t *testing.T) {
	oracle := testOracle(Mainnet, 700)

	// State claims fee=300 at slot 20000000, but schedule says 500 at that point
	state := &OracleState{
		LatestProcessedSlot:      20000000,
		LatestProcessedBlock:     18000000,
		NextSlotToProcess:        20000001,
		PoolAccumulatedFees:      big.NewInt(0),
		Validators:               make(map[uint64]*ValidatorInfo),
		CommitedStates:           make(map[uint64]*OnchainState),
		SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
		UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
		EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
		Donations:                make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:           make([]SummarizedBlock, 0),
		MissedBlocks:             make([]SummarizedBlock, 0),
		WrongFeeBlocks:           make([]SummarizedBlock, 0),
		PoolFeesPercentOver10000: 300, // wrong — schedule says 500 at this slot
		PoolAddress:              oracle.cfg.PoolAddress,
		Network:                  Mainnet,
		PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
		DeployedBlock:            oracle.cfg.DeployedBlock,
		DeployedSlot:             oracle.cfg.DeployedSlot,
		CollateralInWei:          oracle.cfg.CollateralInWei,
	}

	rawBytes, err := serializeStateWithHash(state)
	require.NoError(t, err)

	_, err = oracle.LoadFromBytes(rawBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pool fees percent mismatch")
}

func Test_LoadFromBytes_MultipleScheduleEntries(t *testing.T) {
	// Add a second entry temporarily
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		14082460: 500,
		20000000: 300,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	// State saved between changes: fee=500
	oracle := testOracle(Mainnet, 700)
	state := &OracleState{
		LatestProcessedSlot:      17000000,
		LatestProcessedBlock:     16000000,
		NextSlotToProcess:        17000001,
		PoolAccumulatedFees:      big.NewInt(0),
		Validators:               make(map[uint64]*ValidatorInfo),
		CommitedStates:           make(map[uint64]*OnchainState),
		SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
		UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
		EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
		Donations:                make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:           make([]SummarizedBlock, 0),
		MissedBlocks:             make([]SummarizedBlock, 0),
		WrongFeeBlocks:           make([]SummarizedBlock, 0),
		PoolFeesPercentOver10000: 500,
		PoolAddress:              oracle.cfg.PoolAddress,
		Network:                  Mainnet,
		PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
		DeployedBlock:            oracle.cfg.DeployedBlock,
		DeployedSlot:             oracle.cfg.DeployedSlot,
		CollateralInWei:          oracle.cfg.CollateralInWei,
	}
	rawBytes, err := serializeStateWithHash(state)
	require.NoError(t, err)

	found, err := oracle.LoadFromBytes(rawBytes)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000)

	// State saved after both changes: fee=300
	oracle2 := testOracle(Mainnet, 700)
	state2 := &OracleState{
		LatestProcessedSlot:      25000000,
		LatestProcessedBlock:     23000000,
		NextSlotToProcess:        25000001,
		PoolAccumulatedFees:      big.NewInt(0),
		Validators:               make(map[uint64]*ValidatorInfo),
		CommitedStates:           make(map[uint64]*OnchainState),
		SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
		UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
		EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
		Donations:                make([]*contract.ContractEtherReceived, 0),
		ProposedBlocks:           make([]SummarizedBlock, 0),
		MissedBlocks:             make([]SummarizedBlock, 0),
		WrongFeeBlocks:           make([]SummarizedBlock, 0),
		PoolFeesPercentOver10000: 300,
		PoolAddress:              oracle2.cfg.PoolAddress,
		Network:                  Mainnet,
		PoolFeesAddress:          oracle2.cfg.PoolFeesAddress,
		CheckPointSizeInSlots:    oracle2.cfg.CheckPointSizeInSlots,
		DeployedBlock:            oracle2.cfg.DeployedBlock,
		DeployedSlot:             oracle2.cfg.DeployedSlot,
		CollateralInWei:          oracle2.cfg.CollateralInWei,
	}
	rawBytes2, err := serializeStateWithHash(state2)
	require.NoError(t, err)

	found2, err := oracle2.LoadFromBytes(rawBytes2)
	require.NoError(t, err)
	require.True(t, found2)
	require.Equal(t, 300, oracle2.cfg.PoolFeesPercentOver10000)
}

// Run the multi-entry LoadFromBytes test many times to catch map ordering issues
func Test_LoadFromBytes_MultipleEntries_Deterministic(t *testing.T) {
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		14082460: 500,
		20000000: 300,
		25000000: 200,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	for i := 0; i < 100; i++ {
		oracle := testOracle(Mainnet, 700)
		state := &OracleState{
			LatestProcessedSlot:      30000000,
			LatestProcessedBlock:     28000000,
			NextSlotToProcess:        30000001,
			PoolAccumulatedFees:      big.NewInt(0),
			Validators:               make(map[uint64]*ValidatorInfo),
			CommitedStates:           make(map[uint64]*OnchainState),
			SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
			UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
			EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
			Donations:                make([]*contract.ContractEtherReceived, 0),
			ProposedBlocks:           make([]SummarizedBlock, 0),
			MissedBlocks:             make([]SummarizedBlock, 0),
			WrongFeeBlocks:           make([]SummarizedBlock, 0),
			PoolFeesPercentOver10000: 200,
			PoolAddress:              oracle.cfg.PoolAddress,
			Network:                  Mainnet,
			PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
			CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
			DeployedBlock:            oracle.cfg.DeployedBlock,
			DeployedSlot:             oracle.cfg.DeployedSlot,
			CollateralInWei:          oracle.cfg.CollateralInWei,
		}
		rawBytes, err := serializeStateWithHash(state)
		require.NoError(t, err)

		found, err := oracle.LoadFromBytes(rawBytes)
		require.NoError(t, err, "iteration %d", i)
		require.True(t, found, "iteration %d", i)
		require.Equal(t, 200, oracle.cfg.PoolFeesPercentOver10000,
			"iteration %d: config fee should be 200 (latest schedule entry)", i)
	}
}

// =============================================================================
// End-to-end: simulate sequential slot processing across fee changes
// =============================================================================

func Test_SequentialFeeChanges(t *testing.T) {
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		100: 500,
		200: 300,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	oracle := testOracle(Mainnet, 700)

	// Slot before first change — no event
	block := fullBlockNoEvents(50)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)
	require.NoError(t, err)
	require.Equal(t, 700, oracle.cfg.PoolFeesPercentOver10000)

	// Slot at first change — event 700->500
	block = fullBlockWithFeeEvent(100, 500)
	err = oracle.validateFullBlockConfig(block, oracle.cfg)
	require.NoError(t, err)
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000)
	require.Equal(t, 500, oracle.state.PoolFeesPercentOver10000)

	// Slot between changes — no event
	block = fullBlockNoEvents(150)
	err = oracle.validateFullBlockConfig(block, oracle.cfg)
	require.NoError(t, err)
	require.Equal(t, 500, oracle.cfg.PoolFeesPercentOver10000)

	// Slot at second change — event 500->300
	block = fullBlockWithFeeEvent(200, 300)
	err = oracle.validateFullBlockConfig(block, oracle.cfg)
	require.NoError(t, err)
	require.Equal(t, 300, oracle.cfg.PoolFeesPercentOver10000)
	require.Equal(t, 300, oracle.state.PoolFeesPercentOver10000)

	// Slot after all changes — no event
	block = fullBlockNoEvents(300)
	err = oracle.validateFullBlockConfig(block, oracle.cfg)
	require.NoError(t, err)
	require.Equal(t, 300, oracle.cfg.PoolFeesPercentOver10000)
}

func Test_UnexpectedFeeChange_AtScheduledSlot_WrongFee(t *testing.T) {
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		100: 500,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	oracle := testOracle(Mainnet, 700)

	// Event at the right slot but with wrong fee
	block := fullBlockWithFeeEvent(100, 999)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected pool fee change")
	require.Equal(t, 700, oracle.cfg.PoolFeesPercentOver10000, "config unchanged on error")
}

func Test_UnexpectedFeeChange_AtUnscheduledSlot(t *testing.T) {
	origSchedule := feeSchedule[Mainnet]
	feeSchedule[Mainnet] = map[uint64]int{
		100: 500,
	}
	defer func() { feeSchedule[Mainnet] = origSchedule }()

	oracle := testOracle(Mainnet, 700)

	// Event at an unscheduled slot
	block := fullBlockWithFeeEvent(999, 500)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected pool fee change")
}

// =============================================================================
// Edge case: network with no schedule at all
// =============================================================================

func Test_NoSchedule_NoEvent_OK(t *testing.T) {
	oracle := testOracle(Goerli, 1000)

	block := fullBlockNoEvents(5000)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)
	require.NoError(t, err)
	require.Equal(t, 1000, oracle.cfg.PoolFeesPercentOver10000)
}

func Test_NoSchedule_WithEvent_Rejected(t *testing.T) {
	oracle := testOracle(Goerli, 1000)

	block := fullBlockWithFeeEvent(5000, 500)
	err := oracle.validateFullBlockConfig(block, oracle.cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected pool fee change")
}
