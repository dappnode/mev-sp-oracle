package oracle

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/avast/retry-go/v4"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// A slot after the fork that is not a validator cleanup slot
var testProposalSlot = testGloasSlot + 7

const testProposer = phase0.ValidatorIndex(1000)

// What a bid in the tests promises the pool: 0.0482 ETH
const testBidValueGwei = uint64(48_200_000)

// A payment the builder sends with a tx in its own payload, ie an
// execution_payment, and the EtherReceived event the pool emits for it
func executionPaymentTx(t *testing.T, builderKey *ecdsa.PrivateKey, amount *big.Int) (bellatrix.Transaction, *contract.ContractEtherReceived) {
	t.Helper()
	pool := common.HexToAddress(testPoolAddress)
	chainId := new(big.Int).SetUint64(HoodiChainId)
	tx, err := types.SignNewTx(builderKey, types.LatestSignerForChainID(chainId), &types.DynamicFeeTx{
		ChainID: chainId, GasTipCap: big.NewInt(0), GasFeeCap: big.NewInt(0), Gas: 21000, To: &pool, Value: amount,
	})
	require.NoError(t, err)
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	event := &contract.ContractEtherReceived{
		Sender:         crypto.PubkeyToAddress(builderKey.PublicKey),
		DonationAmount: new(big.Int).Set(amount),
		Raw:            types.Log{Address: pool, TxHash: tx.Hash(), BlockNumber: 5000},
	}
	return raw, event
}

// Each scenario describes a Gloas slot: the bid, what happened to it, and what
// its payload contained
type gloasScenario struct {
	builder         gloas.BuilderIndex
	bidValueGwei    uint64
	bidFeeRecipient string

	// Whether the next beacon block built on the payload, and when it did
	// not, whether the protocol still paid the bid
	included      bool
	withheldPaid  bool
	bidPaidToPool bool

	// Payload contents, for full slots
	payloadFeeRecipient string
	executionPaymentWei *big.Int
	tipsToPayloadFeeRec bool
	withdrawals         []*capella.Withdrawal

	// Default to testProposalSlot and execution block 5000
	slot        uint64
	blockNumber uint64
}

// Builds the FullBlock FetchFullBlock would have built for the scenario
func (s gloasScenario) block(t *testing.T) *FullBlock {
	t.Helper()
	slot, blockNumber := s.slot, s.blockNumber
	if slot == 0 {
		slot = testProposalSlot
	}
	if blockNumber == 0 {
		blockNumber = 5000
	}
	bid := gloasBid(slot, s.builder, s.bidValueGwei, s.bidFeeRecipient, 0xaa, byte(slot))
	fullBlock := gloasFullBlock(t, slot, testProposer, bid)

	if fullBlock.BidPromisesPaymentTo(testPoolAddress) {
		fullBlock.SetBuilderPayment(s.included || s.withheldPaid)
	}
	if !s.included {
		return fullBlock
	}

	var txs []bellatrix.Transaction
	var receipts []*types.Receipt
	payloadFeeRecipient := s.payloadFeeRecipient

	if s.tipsToPayloadFeeRec {
		userKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		for _, charged := range txsOfEveryType(t, userKey, HoodiChainId) {
			raw, err := charged.tx.MarshalBinary()
			require.NoError(t, err)
			txs = append(txs, raw)
			receipts = append(receipts, &types.Receipt{TxHash: charged.tx.Hash(), GasUsed: charged.gasUsed})
		}
	}

	if s.executionPaymentWei != nil {
		builderKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		// The builder pays from the payload's fee recipient, as relays check
		payloadFeeRecipient = crypto.PubkeyToAddress(builderKey.PublicKey).Hex()
		raw, event := executionPaymentTx(t, builderKey, s.executionPaymentWei)
		txs = append(txs, raw)
		fullBlock.Events.EtherReceived = append(fullBlock.Events.EtherReceived, event)
	}

	fullBlock.SetExecutionPayload(gloasEnvelope(bid, blockNumber, payloadFeeRecipient, txs, s.withdrawals))
	if receipts != nil {
		fullBlock.ExecutionReceipts = receipts
	}
	return fullBlock
}

func tipsOf(t *testing.T, fullBlock *FullBlock) *big.Int {
	t.Helper()
	tips, err := fullBlock.GetProposerTip()
	require.NoError(t, err)
	return tips
}

func Test_SummarizedBlock_GloasProposals(t *testing.T) {
	bidValueWei := utils.GweiToWei(testBidValueGwei)
	executionPayment := big.NewInt(3_000_000_000_000_000)

	tests := []struct {
		name              string
		scenario          gloasScenario
		blockType         BlockType
		rewardType        RewardType
		reward            func(*FullBlock) *big.Int
		builderPaymentWei *big.Int
		hasExecutionBlock bool
		expectedDonations int
	}{
		// The proposer built its own payload, judged as before Gloas
		{
			name:              "self-built, pool as fee recipient",
			scenario:          gloasScenario{builder: gloas.BuilderIndexSelfBuild, bidFeeRecipient: testPoolAddress, included: true, payloadFeeRecipient: testPoolAddress, tipsToPayloadFeeRec: true},
			blockType:         OkPoolProposal,
			rewardType:        VanilaBlock,
			reward:            func(b *FullBlock) *big.Int { return tipsOf(t, b) },
			hasExecutionBlock: true,
		},
		{
			name:      "self-built, another fee recipient",
			scenario:  gloasScenario{builder: gloas.BuilderIndexSelfBuild, bidFeeRecipient: testPoolAddress, included: true, payloadFeeRecipient: testBuilderAddress, tipsToPayloadFeeRec: true},
			blockType: WrongFeeRecipient,
			// As before Gloas, the tips are only worked out for subscribers
			rewardType:        UnknownRewardType,
			reward:            func(b *FullBlock) *big.Int { return big.NewInt(0) },
			hasExecutionBlock: true,
		},
		{
			name:      "self-built, payload not included",
			scenario:  gloasScenario{builder: gloas.BuilderIndexSelfBuild, bidFeeRecipient: testPoolAddress},
			blockType: MissedProposal,
			reward:    func(*FullBlock) *big.Int { return big.NewInt(0) },
		},

		// A builder's bid paying the pool
		{
			name:              "bid paid to the pool",
			scenario:          gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true, payloadFeeRecipient: testBuilderAddress},
			blockType:         OkPoolProposal,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return bidValueWei },
			builderPaymentWei: bidValueWei,
			hasExecutionBlock: true,
		},
		{
			name:              "payload withheld, beacon block got its votes",
			scenario:          gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, withheldPaid: true},
			blockType:         OkPoolProposal,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return bidValueWei },
			builderPaymentWei: bidValueWei,
		},
		{
			name:      "payload withheld, beacon block lacked votes",
			scenario:  gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress},
			blockType: MissedProposal,
			reward:    func(*FullBlock) *big.Int { return big.NewInt(0) },
		},
		{
			name: "bid and execution payment, both to the pool",
			scenario: gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true,
				executionPaymentWei: executionPayment},
			blockType:         OkPoolProposal,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return new(big.Int).Add(bidValueWei, executionPayment) },
			builderPaymentWei: bidValueWei,
			hasExecutionBlock: true,
		},
		{
			name: "bid paid to the pool and the pool as payload fee recipient",
			scenario: gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true,
				payloadFeeRecipient: testPoolAddress, tipsToPayloadFeeRec: true},
			blockType:         OkPoolProposal,
			rewardType:        VanilaBlock,
			reward:            func(b *FullBlock) *big.Int { return new(big.Int).Add(bidValueWei, tipsOf(t, b)) },
			builderPaymentWei: bidValueWei,
			hasExecutionBlock: true,
		},

		// A relay's bid, paying by tx only
		{
			name: "execution payment to the pool",
			scenario: gloasScenario{builder: 42, bidFeeRecipient: testPoolAddress, included: true,
				executionPaymentWei: executionPayment},
			blockType:         OkPoolProposal,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return executionPayment },
			hasExecutionBlock: true,
		},
		{
			// The builder broke its promise, the proposer did nothing wrong
			name:              "execution payment promised but never sent",
			scenario:          gloasScenario{builder: 42, bidFeeRecipient: testPoolAddress, included: true, payloadFeeRecipient: testBuilderAddress},
			blockType:         OkPoolProposal,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return big.NewInt(0) },
			hasExecutionBlock: true,
		},

		// The proposer picked a bid paying someone else
		{
			name:              "bid paid to another address",
			scenario:          gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testBuilderAddress, included: true, payloadFeeRecipient: testBuilderAddress},
			blockType:         WrongFeeRecipient,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return bidValueWei },
			hasExecutionBlock: true,
		},
		{
			name:       "bid paid to another address, payload withheld",
			scenario:   gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testBuilderAddress},
			blockType:  WrongFeeRecipient,
			rewardType: MevBlock,
			reward:     func(*FullBlock) *big.Int { return bidValueWei },
		},
		{
			// Credited anyway, or it would be left unallocated in the pool
			name: "bid paid to another address, but the payload paid the pool",
			scenario: gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testBuilderAddress, included: true,
				executionPaymentWei: executionPayment},
			blockType:         OkPoolProposal,
			rewardType:        MevBlock,
			reward:            func(*FullBlock) *big.Int { return executionPayment },
			hasExecutionBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullBlock := tt.scenario.block(t)
			oracle := NewOracle(&Config{Network: Hoodi, PoolAddress: testPoolAddress})

			summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)

			require.Equal(t, tt.blockType, summary.BlockType)
			require.Equal(t, tt.reward(fullBlock), summary.Reward)
			require.Equal(t, tt.builderPaymentWei, summary.BuilderPaymentWei)
			if tt.blockType != MissedProposal {
				require.Equal(t, tt.rewardType, summary.RewardType)
			}
			if tt.hasExecutionBlock {
				require.Equal(t, uint64(5000), summary.Block)
			} else {
				require.Zero(t, summary.Block)
			}

			// An execution payment is a MEV reward, not a donation
			require.Empty(t, fullBlock.GetDonations(testPoolAddress))
		})
	}
}

// A validator with BLS credentials can't be subscribed, its reward goes to the
// pool fees as before Gloas
func Test_SummarizedBlock_GloasBlsCredentials(t *testing.T) {
	fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true,
		payloadFeeRecipient: testBuilderAddress}.block(t)
	fullBlock.Validator.Validator.WithdrawalCredentials = make([]byte, 32)

	summary := fullBlock.SummarizedBlock(NewOracle(&Config{Network: Hoodi, PoolAddress: testPoolAddress}), testPoolAddress)
	require.Equal(t, OkPoolProposalBlsKeys, summary.BlockType)
	require.Equal(t, utils.GweiToWei(testBidValueGwei), summary.Reward)
}

// An oracle on Hoodi starting at testProposalSlot, whose validators all have 32
// ETH of effective balance
func gloasTestOracle(t *testing.T) *Oracle {
	t.Helper()
	oracle := NewOracle(&Config{
		Network:                  Hoodi,
		PoolAddress:              testPoolAddress,
		PoolFeesAddress:          "0x2222222222222222222222222222222222222222",
		PoolFeesPercentOver10000: 500,
		CollateralInWei:          big.NewInt(1),
		CheckPointSizeInSlots:    7200,
		DeployedSlot:             testProposalSlot,
	})
	oracle.SetGetSetOfValidatorsFunc(func(indices []phase0.ValidatorIndex, _ string, _ ...retry.Option) (map[phase0.ValidatorIndex]*v1.Validator, error) {
		validators := make(map[phase0.ValidatorIndex]*v1.Validator)
		for _, index := range indices {
			validators[index] = &v1.Validator{Index: index, Validator: &phase0.Validator{EffectiveBalance: 32_000_000_000}}
		}
		return validators, nil
	})
	return oracle
}

func Test_AdvanceStateToNextSlot_GloasProposals(t *testing.T) {
	withdrawalAddress := "0x1111111111111111111111111111111111111111"

	t.Run("withheld payload paid credits the proposer", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.addSubscription(uint64(testProposer), withdrawalAddress, "0x")
		oracle.state.LatestProcessedBlock = 4999

		fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, withheldPaid: true}.block(t)
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		validator := oracle.state.Validators[uint64(testProposer)]
		require.Equal(t, Active, validator.ValidatorStatus)
		require.Len(t, oracle.state.ProposedBlocks, 1)
		require.Positive(t, validator.AccumulatedRewardsWei.Sign())
		// An empty slot has no execution block
		require.Equal(t, uint64(4999), oracle.state.LatestProcessedBlock)
		require.NoError(t, oracle.RunOffchainReconciliation())
	})

	t.Run("withheld payload not paid counts as missed", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.addSubscription(uint64(testProposer), withdrawalAddress, "0x")

		fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress}.block(t)
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		require.Equal(t, YellowCard, oracle.state.Validators[uint64(testProposer)].ValidatorStatus)
		require.Len(t, oracle.state.MissedBlocks, 1)
		require.Empty(t, oracle.state.ProposedBlocks)
		require.NoError(t, oracle.RunOffchainReconciliation())
	})

	t.Run("bid paying another address bans the proposer", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.addSubscription(uint64(testProposer), withdrawalAddress, "0x")

		fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testBuilderAddress, included: true,
			payloadFeeRecipient: testBuilderAddress}.block(t)
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		require.Equal(t, Banned, oracle.state.Validators[uint64(testProposer)].ValidatorStatus)
		require.NoError(t, oracle.RunOffchainReconciliation())
	})

	t.Run("unpaid execution payment does not ban the proposer", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.addSubscription(uint64(testProposer), withdrawalAddress, "0x")

		fullBlock := gloasScenario{builder: 42, bidFeeRecipient: testPoolAddress, included: true, payloadFeeRecipient: testBuilderAddress}.block(t)
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		require.Equal(t, Active, oracle.state.Validators[uint64(testProposer)].ValidatorStatus)
		require.Equal(t, uint64(5000), oracle.state.LatestProcessedBlock)
		require.NoError(t, oracle.RunOffchainReconciliation())
	})

	t.Run("bid paid to the pool auto subscribes an unknown proposer", func(t *testing.T) {
		oracle := gloasTestOracle(t)

		fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true,
			payloadFeeRecipient: testBuilderAddress}.block(t)
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		validator := oracle.state.Validators[uint64(testProposer)]
		require.Equal(t, Active, validator.ValidatorStatus)
		require.Equal(t, Auto, validator.SubscriptionType)
		require.NoError(t, oracle.RunOffchainReconciliation())
	})

	// Both the builder payment and the tips are credited and accounted for
	t.Run("bid and tips to the pool reconcile", func(t *testing.T) {
		oracle := gloasTestOracle(t)
		oracle.addSubscription(uint64(testProposer), withdrawalAddress, "0x")

		fullBlock := gloasScenario{builder: 42, bidValueGwei: testBidValueGwei, bidFeeRecipient: testPoolAddress, included: true,
			payloadFeeRecipient: testPoolAddress, tipsToPayloadFeeRec: true}.block(t)
		_, err := oracle.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		total := new(big.Int).Add(utils.GweiToWei(testBidValueGwei), tipsOf(t, fullBlock))
		require.Equal(t, total, oracle.state.ProposedBlocks[0].Reward)
		require.NoError(t, oracle.RunOffchainReconciliation())
	})
}
