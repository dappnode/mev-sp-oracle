package oracle

import (
	"math/big"
	"strings"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The withdrawal that pays a builder payment of amountGwei to address
func builderWithdrawal(builderIndex uint64, address string, amountGwei uint64) *capella.Withdrawal {
	return &capella.Withdrawal{
		Index:          capella.WithdrawalIndex(builderIndex),
		ValidatorIndex: builderIndexFlag | phase0.ValidatorIndex(builderIndex),
		Address:        executionAddress(address),
		Amount:         phase0.Gwei(amountGwei),
	}
}

// The sweep of a validator's balance, as before Gloas
func validatorWithdrawal(validatorIndex uint64, address string, amountGwei uint64) *capella.Withdrawal {
	return &capella.Withdrawal{
		ValidatorIndex: phase0.ValidatorIndex(validatorIndex),
		Address:        executionAddress(address),
		Amount:         phase0.Gwei(amountGwei),
	}
}

func inFlightPayment(slot uint64, builderIndex uint64, amountGwei uint64) *BuilderPayment {
	return &BuilderPayment{Slot: slot, ProposerIndex: 1, BuilderIndex: builderIndex, AmountWei: utils.GweiToWei(amountGwei), Paid: true}
}

func Test_IsBuilderWithdrawal(t *testing.T) {
	require.True(t, isBuilderWithdrawal(builderWithdrawal(42, testPoolAddress, 1)))
	require.True(t, isBuilderWithdrawal(builderWithdrawal(0, testPoolAddress, 1)))
	require.False(t, isBuilderWithdrawal(validatorWithdrawal(42, testPoolAddress, 1)))
	// Validator indices are far from reaching the flag
	require.False(t, isBuilderWithdrawal(validatorWithdrawal(uint64(builderIndexFlag)-1, testPoolAddress, 1)))
}

func Test_BuilderWithdrawalsTo(t *testing.T) {
	poolPayment := builderWithdrawal(42, testPoolAddress, 48_200_000)
	otherPoolPayment := builderWithdrawal(7, strings.ToLower(testPoolAddress), 31_000_000)

	fullBlock := gloasScenario{builder: 42, bidFeeRecipient: testPoolAddress, included: true, payloadFeeRecipient: testBuilderAddress,
		withdrawals: []*capella.Withdrawal{
			validatorWithdrawal(5, "0x1111111111111111111111111111111111111111", 18_000_000),
			poolPayment,
			builderWithdrawal(43, testBuilderAddress, 10_000_000),
			// A validator sweeping to the pool is not a builder payment
			validatorWithdrawal(6, testPoolAddress, 1_000_000),
			otherPoolPayment,
		}}.block(t)

	require.Equal(t, []*capella.Withdrawal{poolPayment, otherPoolPayment}, fullBlock.BuilderWithdrawalsTo(testPoolAddress))
	require.Equal(t, utils.GweiToWei(48_200_000+31_000_000), fullBlock.BuilderWithdrawalsWeiTo(testPoolAddress))

	t.Run("empty slot", func(t *testing.T) {
		empty := gloasScenario{builder: 42, bidFeeRecipient: testPoolAddress}.block(t)
		require.Empty(t, empty.BuilderWithdrawalsTo(testPoolAddress))
		require.Zero(t, empty.BuilderWithdrawalsWeiTo(testPoolAddress).Sign())
	})

	t.Run("before Gloas", func(t *testing.T) {
		fulu := buildMevBlock(t, 100, 1000, 7, testPoolAddress, big.NewInt(1))
		require.Empty(t, fulu.BuilderWithdrawalsTo(testPoolAddress))
	})
}

func Test_SettleArrivedBuilderPayments(t *testing.T) {
	blockPaying := func(t *testing.T, withdrawals ...*capella.Withdrawal) *FullBlock {
		return gloasScenario{builder: gloas.BuilderIndexSelfBuild, bidFeeRecipient: testPoolAddress, included: true,
			payloadFeeRecipient: testBuilderAddress, withdrawals: withdrawals}.block(t)
	}

	t.Run("settles the arrived payment only", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		first, second := inFlightPayment(10, 42, 48_200_000), inFlightPayment(11, 7, 31_000_000)
		oracle.state.InFlightBuilderPayments = []*BuilderPayment{first, second}

		require.NoError(t, oracle.settleArrivedBuilderPayments(blockPaying(t, builderWithdrawal(7, testPoolAddress, 31_000_000))))
		require.Equal(t, []*BuilderPayment{first}, oracle.state.InFlightBuilderPayments)
	})

	t.Run("several in the same payload", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.state.InFlightBuilderPayments = []*BuilderPayment{inFlightPayment(10, 42, 48_200_000), inFlightPayment(11, 7, 31_000_000)}

		require.NoError(t, oracle.settleArrivedBuilderPayments(blockPaying(t,
			builderWithdrawal(42, testPoolAddress, 48_200_000), builderWithdrawal(7, testPoolAddress, 31_000_000))))
		require.Empty(t, oracle.state.InFlightBuilderPayments)
		require.Zero(t, oracle.inFlightBuilderPaymentsWei().Sign())
	})

	t.Run("identical payments settle one at a time", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.state.InFlightBuilderPayments = []*BuilderPayment{inFlightPayment(10, 42, 1_000), inFlightPayment(11, 42, 1_000)}

		require.NoError(t, oracle.settleArrivedBuilderPayments(blockPaying(t, builderWithdrawal(42, testPoolAddress, 1_000))))
		require.Len(t, oracle.state.InFlightBuilderPayments, 1)
	})

	t.Run("withdrawals that are not builder payments to the pool are ignored", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		payment := inFlightPayment(10, 42, 48_200_000)
		oracle.state.InFlightBuilderPayments = []*BuilderPayment{payment}

		require.NoError(t, oracle.settleArrivedBuilderPayments(blockPaying(t,
			builderWithdrawal(42, testBuilderAddress, 48_200_000), validatorWithdrawal(42, testPoolAddress, 48_200_000))))
		require.Equal(t, []*BuilderPayment{payment}, oracle.state.InFlightBuilderPayments)
	})

	t.Run("a payment never credited is rejected without touching the state", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		payment := inFlightPayment(10, 42, 48_200_000)
		oracle.state.InFlightBuilderPayments = []*BuilderPayment{payment}

		// The first one matches, but the slot is rejected as a whole
		err := oracle.settleArrivedBuilderPayments(blockPaying(t,
			builderWithdrawal(42, testPoolAddress, 48_200_000), builderWithdrawal(42, testPoolAddress, 1)))
		require.ErrorContains(t, err, "no such builder payment was credited")
		require.Equal(t, []*BuilderPayment{payment}, oracle.state.InFlightBuilderPayments)
	})
}

func Test_RunOnchainReconciliation_CountsInFlightBuilderPayments(t *testing.T) {
	oracle := gloasTestOracle(t)
	oracle.state.PoolAccumulatedFees = utils.GweiToWei(100_000_000)
	oracle.state.InFlightBuilderPayments = []*BuilderPayment{inFlightPayment(10, 42, 30_000_000)}

	// The pool holds what it owes minus what is still on its way
	require.NoError(t, oracle.RunOnchainReconciliation(utils.GweiToWei(70_000_000), nil))
	require.ErrorContains(t, oracle.RunOnchainReconciliation(utils.GweiToWei(100_000_000), nil), "dont match")

	// Once the payment arrives the pool holds everything
	oracle.state.InFlightBuilderPayments = nil
	require.NoError(t, oracle.RunOnchainReconciliation(utils.GweiToWei(100_000_000), nil))
}

// The slots of the worked example: a builder payment credited in one slot
// reaches the pool slots later, and a withheld payload's one even an epoch
// later. Both reconciliations hold at every step, with the pool balance
// following what really reaches it.
func Test_BuilderPayments_FromCreditToArrival(t *testing.T) {
	slot := testProposalSlot
	oracle := gloasTestOracle(t)
	oracle.addSubscription(uint64(testProposer), "0x1111111111111111111111111111111111111111", "0x")
	poolBalance := big.NewInt(0)

	process := func(t *testing.T, fullBlock *FullBlock) {
		t.Helper()
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)
		require.NoError(t, oracle.RunOffchainReconciliation())
		require.NoError(t, oracle.RunOnchainReconciliation(poolBalance, nil))
	}

	// Slot 1: builder 42's payload is included. 0.0482 ETH is owed
	process(t, gloasScenario{slot: slot, builder: 42, bidValueGwei: 48_200_000, bidFeeRecipient: testPoolAddress, included: true,
		payloadFeeRecipient: testBuilderAddress, blockNumber: 5000}.block(t))
	require.Equal(t, utils.GweiToWei(48_200_000), oracle.inFlightBuilderPaymentsWei())

	// Slot 2: builder 7 withholds its payload, but the beacon block got its
	// votes and the protocol pays 0.031 ETH. The slot is empty, so slot 1's
	// payment can't reach the pool yet
	process(t, gloasScenario{slot: slot + 1, builder: 7, bidValueGwei: 31_000_000, bidFeeRecipient: testPoolAddress,
		withheldPaid: true}.block(t))
	require.Equal(t, utils.GweiToWei(48_200_000+31_000_000), oracle.inFlightBuilderPaymentsWei())
	require.Equal(t, uint64(5000), oracle.state.LatestProcessedBlock)

	// Slot 3: missed by another validator
	credentials := make([]byte, 32)
	credentials[0] = 1
	process(t, NewFullBlock(&v1.ProposerDuty{Slot: phase0.Slot(slot + 2), ValidatorIndex: 2000},
		&v1.Validator{Index: 2000, Validator: &phase0.Validator{WithdrawalCredentials: credentials}}, HoodiChainId))

	// Slot 4: a self-built payload with the pool as fee recipient carries slot
	// 1's payment. The pool receives it and the tips in the same block
	selfBuilt := gloasScenario{slot: slot + 3, builder: gloas.BuilderIndexSelfBuild, bidFeeRecipient: testPoolAddress, included: true,
		payloadFeeRecipient: testPoolAddress, tipsToPayloadFeeRec: true, blockNumber: 5001,
		withdrawals: []*capella.Withdrawal{builderWithdrawal(42, testPoolAddress, 48_200_000)}}.block(t)
	poolBalance.Add(poolBalance, utils.GweiToWei(48_200_000))
	poolBalance.Add(poolBalance, tipsOf(t, selfBuilt))
	process(t, selfBuilt)
	require.Equal(t, utils.GweiToWei(31_000_000), oracle.inFlightBuilderPaymentsWei())
	require.Equal(t, uint64(5001), oracle.state.LatestProcessedBlock)

	// An epoch later, slot 2's payment arrives in whatever payload comes
	// next. Here a relay's, whose promised execution payment never came
	poolBalance.Add(poolBalance, utils.GweiToWei(31_000_000))
	process(t, gloasScenario{slot: slot + 4, builder: 43, bidFeeRecipient: testPoolAddress, included: true,
		payloadFeeRecipient: testBuilderAddress, blockNumber: 5002,
		withdrawals: []*capella.Withdrawal{builderWithdrawal(7, testPoolAddress, 31_000_000)}}.block(t))
	require.Empty(t, oracle.state.InFlightBuilderPayments)
	require.Equal(t, uint64(5002), oracle.state.LatestProcessedBlock)

	// Slots 1, 2, 4 and 5 were valid proposals, slot 3 was missed
	require.Len(t, oracle.state.ProposedBlocks, 4)
	require.Equal(t, Active, oracle.state.Validators[uint64(testProposer)].ValidatorStatus)
}

// The forced payment detector reads the pool balance across the block. A
// builder payment arriving in it must count as explained, not as a forced one
func Test_DetectForcedMevPayment_BuilderWithdrawals(t *testing.T) {
	payment := builderWithdrawal(42, testPoolAddress, 48_200_000)
	paymentWei := utils.GweiToWei(48_200_000)
	before := decimal(t, "1000000000000000000")

	t.Run("builder payment alone", func(t *testing.T) {
		fullBlock := gloasScenario{builder: 43, bidFeeRecipient: testBuilderAddress, included: true, payloadFeeRecipient: testBuilderAddress,
			withdrawals: []*capella.Withdrawal{payment}}.block(t)
		o, _ := newForcedPaymentRPC(t, fullBlock, before, new(big.Int).Add(before, paymentWei), nil)

		require.NoError(t, o.DetectForcedMevPayment(fullBlock))
		require.Nil(t, fullBlock.ForcedMevPayment)
	})

	t.Run("a validator withdrawal to the pool is still rejected", func(t *testing.T) {
		fullBlock := gloasScenario{builder: 43, bidFeeRecipient: testBuilderAddress, included: true, payloadFeeRecipient: testBuilderAddress,
			withdrawals: []*capella.Withdrawal{validatorWithdrawal(42, testPoolAddress, 48_200_000)}}.block(t)
		o, _ := newForcedPaymentRPC(t, fullBlock, before, new(big.Int).Add(before, paymentWei), nil)

		require.ErrorContains(t, o.DetectForcedMevPayment(fullBlock), "credits a consensus withdrawal to the pool")
	})

	t.Run("builder payment and a forced payment in the same block", func(t *testing.T) {
		forced := big.NewInt(3_000_000_000_000_000)
		builderKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		forwarder := common.HexToAddress(titanForwarder)
		chainId := new(big.Int).SetUint64(HoodiChainId)
		tx, err := types.SignNewTx(builderKey, types.LatestSignerForChainID(chainId), &types.DynamicFeeTx{
			ChainID: chainId, GasTipCap: big.NewInt(0), GasFeeCap: big.NewInt(0), Gas: 21000, To: &forwarder, Value: forced,
		})
		require.NoError(t, err)
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)

		bid := gloasBid(testProposalSlot, 43, 0, testPoolAddress, 0xaa, 0xbb)
		fullBlock := gloasFullBlock(t, testProposalSlot, testProposer, bid)
		fullBlock.SetExecutionPayload(gloasEnvelope(bid, 5000, crypto.PubkeyToAddress(builderKey.PublicKey).Hex(),
			[]bellatrix.Transaction{raw}, []*capella.Withdrawal{payment}))
		receipt := &types.Receipt{TxHash: tx.Hash(), BlockNumber: big.NewInt(5000), BlockHash: common.Hash(bid.BlockHash),
			TransactionIndex: 0, Status: types.ReceiptStatusSuccessful, Logs: []*types.Log{}}

		after := new(big.Int).Add(before, paymentWei)
		after.Add(after, forced)
		o, _ := newForcedPaymentRPC(t, fullBlock, before, after, receipt)

		require.NoError(t, o.DetectForcedMevPayment(fullBlock))
		require.NotNil(t, fullBlock.ForcedMevPayment)
		require.Equal(t, forced, fullBlock.ForcedMevPayment.AmountWei)
	})
}

// States saved before Gloas have none of the new fields, and must load with
// their hash intact. States with builder payments in flight must round trip
func Test_StateHash_BuilderPaymentFields(t *testing.T) {
	newState := func(oracle *Oracle) *OracleState {
		return &OracleState{
			PoolAccumulatedFees:      big.NewInt(0),
			Validators:               make(map[uint64]*ValidatorInfo),
			CommitedStates:           make(map[uint64]*OnchainState),
			SubscriptionEvents:       make([]*contract.ContractSubscribeValidator, 0),
			UnsubscriptionEvents:     make([]*contract.ContractUnsubscribeValidator, 0),
			EtherReceivedEvents:      make([]*contract.ContractEtherReceived, 0),
			Donations:                make([]*contract.ContractEtherReceived, 0),
			ProposedBlocks:           []SummarizedBlock{{Slot: 10, Block: 9, Reward: big.NewInt(5), RewardType: VanilaBlock, BlockType: OkPoolProposal}},
			MissedBlocks:             make([]SummarizedBlock, 0),
			WrongFeeBlocks:           make([]SummarizedBlock, 0),
			PoolFeesPercentOver10000: oracle.cfg.PoolFeesPercentOver10000,
			PoolAddress:              oracle.cfg.PoolAddress,
			Network:                  oracle.cfg.Network,
			PoolFeesAddress:          oracle.cfg.PoolFeesAddress,
			CheckPointSizeInSlots:    oracle.cfg.CheckPointSizeInSlots,
			DeployedBlock:            oracle.cfg.DeployedBlock,
			DeployedSlot:             oracle.cfg.DeployedSlot,
			CollateralInWei:          oracle.cfg.CollateralInWei,
		}
	}

	t.Run("state from before Gloas", func(t *testing.T) {
		oracle := testOracle(Hoodi, 500)
		rawBytes, err := serializeStateWithHash(newState(oracle))
		require.NoError(t, err)
		require.NotContains(t, string(rawBytes), "builder_payment")

		found, err := oracle.LoadFromBytes(rawBytes)
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("state with builder payments", func(t *testing.T) {
		oracle := testOracle(Hoodi, 500)
		state := newState(oracle)
		state.ProposedBlocks[0].BuilderPaymentWei = big.NewInt(3)
		state.InFlightBuilderPayments = []*BuilderPayment{inFlightPayment(10, 42, 48_200_000)}
		rawBytes, err := serializeStateWithHash(state)
		require.NoError(t, err)

		found, err := oracle.LoadFromBytes(rawBytes)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, big.NewInt(3), oracle.state.ProposedBlocks[0].BuilderPaymentWei)
		require.Equal(t, utils.GweiToWei(48_200_000), oracle.state.InFlightBuilderPayments[0].AmountWei)
	})
}

// A cancelled payment is never tracked, so the onchain reconciliation does not
// count ETH that will never come
func Test_AdvanceStateToNextSlot_CancelledPaymentIsNotTracked(t *testing.T) {
	oracle := gloasTestOracle(t)
	oracle.addSubscription(uint64(testProposer), "0x1111111111111111111111111111111111111111", "0x")

	fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true,
		paymentCancelled: true, payloadFeeRecipient: testBuilderAddress}.block(t)
	_, err := oracle.AdvanceStateToNextSlot(fullBlock)
	require.NoError(t, err)

	require.Empty(t, oracle.state.InFlightBuilderPayments)
	require.NoError(t, oracle.RunOffchainReconciliation())
	require.NoError(t, oracle.RunOnchainReconciliation(big.NewInt(0), nil))
}
