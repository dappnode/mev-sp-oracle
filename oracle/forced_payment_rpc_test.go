package oracle

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/avast/retry-go/v4"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// Exercise the real ethclient JSON-RPC calls, including block parameters and
// receipt decoding. No external services run in CI.
type forcedPaymentRPC struct {
	mu       sync.Mutex
	balances map[string]*big.Int
	receipts map[common.Hash]*types.Receipt
	calls    []string
	fail     string
}

func (s *forcedPaymentRPC) GetBalance(_ context.Context, address common.Address, block string) (*hexutil.Big, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, "balance:"+block)
	if address != common.HexToAddress(testPoolAddress) || s.fail == "balance:"+block {
		return nil, fmt.Errorf("balance unavailable")
	}
	balance, ok := s.balances[block]
	if !ok {
		return nil, fmt.Errorf("unexpected balance block %s", block)
	}
	return (*hexutil.Big)(balance), nil
}

func (s *forcedPaymentRPC) GetTransactionReceipt(_ context.Context, hash common.Hash) (*types.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, "receipt:"+hash.Hex())
	if s.fail == "receipt" {
		return nil, fmt.Errorf("receipt unavailable")
	}
	return s.receipts[hash], nil
}

func (s *forcedPaymentRPC) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func newForcedPaymentRPC(t testing.TB, b *FullBlock, before, after *big.Int, receipt *types.Receipt) (*Onchain, *forcedPaymentRPC) {
	t.Helper()
	backend := &forcedPaymentRPC{
		balances: map[string]*big.Int{
			hexutil.EncodeUint64(b.GetBlockNumber() - 1): before,
			hexutil.EncodeUint64(b.GetBlockNumber()):     after,
		},
		receipts: make(map[common.Hash]*types.Receipt),
	}
	if receipt != nil {
		backend.receipts[receipt.TxHash] = receipt
	}
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", backend))
	client := ethclient.NewClient(rpc.DialInProc(server))
	t.Cleanup(client.Close)
	t.Cleanup(server.Stop)
	return &Onchain{ExecutionClient: client, PoolAddress: testPoolAddress, NumRetries: 1}, backend
}

type forcedPaymentFixture struct {
	Slot           uint64
	ProposerIndex  phase0.ValidatorIndex
	Payload        *deneb.ExecutionPayload
	BalanceBefore  string
	BalanceAfter   string
	EtherReceived  []*contract.ContractEtherReceived
	ExpectedReward string
	Receipt        *types.Receipt
}

func decimal(t testing.TB, value string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(value, 10)
	require.True(t, ok)
	return n
}

func (f forcedPaymentFixture) block(t testing.TB) *FullBlock {
	t.Helper()
	credentials := make([]byte, 32)
	credentials[0] = 1
	copy(credentials[12:], common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
	b := NewFullBlock(&v1.ProposerDuty{Slot: phase0.Slot(f.Slot), ValidatorIndex: f.ProposerIndex},
		&v1.Validator{Index: f.ProposerIndex, Validator: &phase0.Validator{WithdrawalCredentials: credentials}}, MainnetChainId)
	b.SetConsensusBlock(&spec.VersionedSignedBeaconBlock{Version: spec.DataVersionFulu,
		Fulu: &electra.SignedBeaconBlock{Message: &electra.BeaconBlock{
			Slot: phase0.Slot(f.Slot), ProposerIndex: f.ProposerIndex,
			Body: &electra.BeaconBlockBody{ExecutionPayload: f.Payload},
		}},
	})
	for _, event := range f.EtherReceived {
		copied := *event
		copied.DonationAmount = new(big.Int).Set(event.DonationAmount)
		b.Events.EtherReceived = append(b.Events.EtherReceived, &copied)
	}
	return b
}

// Index of the payment to the pool in the incident block
const incidentPaymentIndex = 3

// Fixed keys so every build of the incident block has the same tx hashes, which
// the receipt refers to
var (
	incidentBuilderKey = mustHexKey("4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318")
	incidentOtherKey   = mustHexKey("8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63")
)

func mustHexKey(hex string) *ecdsa.PrivateKey {
	key, err := crypto.HexToECDSA(hex)
	if err != nil {
		panic(err)
	}
	return key
}

// Shaped like mainnet block 25956850 (slot 15194839), with its real amounts: the
// builder, which is the fee recipient, pays four parties through the forwarder,
// the fourth being the pool. Unrelated txs follow, the last of them with zero
// value and emitting a donation to the pool.
func incidentFixture(t testing.TB) forcedPaymentFixture {
	t.Helper()

	const blockNumber = 25956850
	reward := decimal(t, "13131532323905851")
	donation := decimal(t, "103797680070839")
	before := decimal(t, "64806968647549848541")
	after := new(big.Int).Add(before, reward)
	after.Add(after, donation)

	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(MainnetChainId))
	nonces := make(map[*ecdsa.PrivateKey]uint64)
	sign := func(key *ecdsa.PrivateKey, to string, value *big.Int) *types.Transaction {
		toAddress := common.HexToAddress(to)
		tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   new(big.Int).SetUint64(MainnetChainId),
			Nonce:     nonces[key],
			GasTipCap: big.NewInt(0),
			GasFeeCap: big.NewInt(0),
			Gas:       21000,
			To:        &toAddress,
			Value:     value,
		})
		require.NoError(t, err)
		nonces[key]++
		return tx
	}

	txs := []*types.Transaction{
		sign(incidentBuilderKey, titanForwarder, big.NewInt(4622541964425610)),
		sign(incidentBuilderKey, titanForwarder, big.NewInt(527426380532538)),
		sign(incidentBuilderKey, titanForwarder, big.NewInt(6206743076091)),
		sign(incidentBuilderKey, titanForwarder, reward),
		sign(incidentOtherKey, "0xf4e8b391eae59bc848848a73230b7b174a785d5c", big.NewInt(137183620275739076)),
		sign(incidentOtherKey, "0x5f2876944247f302dff857431e26951c2b8dfa33", big.NewInt(0)),
	}
	rawTxs := make([]bellatrix.Transaction, 0, len(txs))
	for _, tx := range txs {
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		rawTxs = append(rawTxs, raw)
	}

	var feeRecipient bellatrix.ExecutionAddress
	copy(feeRecipient[:], crypto.PubkeyToAddress(incidentBuilderKey.PublicKey).Bytes())
	var blockHash phase0.Hash32
	copy(blockHash[:], common.HexToHash("0x25956850").Bytes())

	abi, err := contract.ContractMetaData.GetAbi()
	require.NoError(t, err)
	lastTx := len(txs) - 1
	donationEvent := etherReceivedEvent(donation)
	donationEvent.Raw = types.Log{
		Address:     common.HexToAddress(testPoolAddress),
		Topics:      []common.Hash{abi.Events["EtherReceived"].ID},
		BlockNumber: blockNumber,
		TxHash:      txs[lastTx].Hash(),
		TxIndex:     uint(lastTx),
		BlockHash:   common.Hash(blockHash),
	}

	return forcedPaymentFixture{
		Slot:          15194839,
		ProposerIndex: 2245769,
		Payload: &deneb.ExecutionPayload{
			FeeRecipient: feeRecipient,
			BlockNumber:  blockNumber,
			BlockHash:    blockHash,
			Transactions: rawTxs,
		},
		BalanceBefore:  before.String(),
		BalanceAfter:   after.String(),
		EtherReceived:  []*contract.ContractEtherReceived{donationEvent},
		ExpectedReward: reward.String(),
		Receipt: &types.Receipt{
			Status:           types.ReceiptStatusSuccessful,
			TxHash:           txs[incidentPaymentIndex].Hash(),
			BlockHash:        common.Hash(blockHash),
			BlockNumber:      big.NewInt(blockNumber),
			TransactionIndex: incidentPaymentIndex,
			Logs:             []*types.Log{},
		},
	}
}

func Test_AnyPositionForcedPayment_IncidentStateAndReconciliation(t *testing.T) {
	f := incidentFixture(t)
	b := f.block(t)
	last, _ := b.GetLastTxValueAndRecipient()
	require.Zero(t, last.Sign(), "v1.2.12 skipped this block")
	require.Less(t, incidentPaymentIndex, len(b.GetBlockTransactions())-1, "the payment is not the last tx")
	o, backend := newForcedPaymentRPC(t, b, decimal(t, f.BalanceBefore), decimal(t, f.BalanceAfter), f.Receipt)
	// Production has already fetched receipts for a subscribed validator.
	b.ExecutionReceipts = make([]*types.Receipt, len(b.GetBlockTransactions()))
	b.ExecutionReceipts[incidentPaymentIndex] = f.Receipt
	require.NoError(t, o.DetectForcedMevPayment(b))
	require.Equal(t, 2, backend.callCount(), "reuse the correct receipt, no extra receipt RPC")
	require.Equal(t, f.Receipt.TxHash, b.Events.EtherReceived[1].Raw.TxHash)
	require.Equal(t, uint(incidentPaymentIndex), b.Events.EtherReceived[1].Raw.TxIndex)
	require.Len(t, b.GetDonations(testPoolAddress), 1)
	require.Equal(t, "103797680070839", b.GetDonations(testPoolAddress)[0].DonationAmount.String())

	oracle := NewOracle(&Config{Network: Mainnet, PoolAddress: testPoolAddress, DeployedSlot: f.Slot,
		PoolFeesPercentOver10000: 500, PoolFeesAddress: "0x2222222222222222222222222222222222222222"})
	oracle.addSubscription(uint64(f.ProposerIndex), "0x1111111111111111111111111111111111111111", "0x")
	oracle.addSubscription(42, "0x3333333333333333333333333333333333333333", "0x")
	oracle.SetGetSetOfValidatorsFunc(func(indices []phase0.ValidatorIndex, _ string, _ ...retry.Option) (map[phase0.ValidatorIndex]*v1.Validator, error) {
		validators := make(map[phase0.ValidatorIndex]*v1.Validator)
		for _, index := range indices {
			validators[index] = &v1.Validator{Index: index, Validator: &phase0.Validator{EffectiveBalance: 32000000000}}
		}
		return validators, nil
	})
	// Synthetic starting accounts with the real pre-block total, including
	// the pending balance which the bug wrongly confiscated from this proposer.
	pending := decimal(t, "4096459189661146")
	oracle.state.Validators[uint64(f.ProposerIndex)].PendingRewardsWei.Set(pending)
	oracle.state.PoolAccumulatedFees.Sub(decimal(t, f.BalanceBefore), pending)
	oracle.state.EtherReceivedEvents = append(oracle.state.EtherReceivedEvents, etherReceivedEvent(decimal(t, f.BalanceBefore)))
	require.NoError(t, oracle.RunOnchainReconciliation(decimal(t, f.BalanceBefore), nil))
	_, err := oracle.AdvanceStateToNextSlot(b)
	require.NoError(t, err)
	require.Equal(t, Active, oracle.state.Validators[uint64(f.ProposerIndex)].ValidatorStatus)
	require.Empty(t, oracle.state.WrongFeeBlocks)
	require.Len(t, oracle.state.ProposedBlocks, 1)
	require.Equal(t, f.ExpectedReward, oracle.state.ProposedBlocks[0].Reward.String())
	require.Greater(t, oracle.state.Validators[uint64(f.ProposerIndex)].AccumulatedRewardsWei.Cmp(pending), 0)
	require.NoError(t, oracle.RunOffchainReconciliation())
	require.NoError(t, oracle.RunOnchainReconciliation(decimal(t, f.BalanceAfter), nil))
	// Re-running detection must not duplicate the event or make more RPCs.
	require.NoError(t, o.DetectForcedMevPayment(b))
	require.Len(t, b.Events.EtherReceived, 2)
	require.Equal(t, 2, backend.callCount())
}

func Test_AnyPositionForcedPayment_FailuresDoNotCredit(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*FullBlock, *forcedPaymentRPC, *types.Receipt)
		want   string
	}{
		{"after balance unavailable", func(b *FullBlock, s *forcedPaymentRPC, _ *types.Receipt) {
			s.fail = "balance:" + hexutil.EncodeUint64(b.GetBlockNumber())
		}, "balance"},
		{"before balance unavailable", func(b *FullBlock, s *forcedPaymentRPC, _ *types.Receipt) {
			s.fail = "balance:" + hexutil.EncodeUint64(b.GetBlockNumber()-1)
		}, "balance"},
		{"receipt unavailable", func(_ *FullBlock, s *forcedPaymentRPC, _ *types.Receipt) { s.fail = "receipt" }, "receipt"},
		{"receipt missing", func(_ *FullBlock, s *forcedPaymentRPC, _ *types.Receipt) { s.receipts = nil }, "receipt"},
		{"receipt reverted", func(_ *FullBlock, _ *forcedPaymentRPC, r *types.Receipt) { r.Status = 0 }, "unsuccessful"},
		{"wrong receipt index", func(_ *FullBlock, _ *forcedPaymentRPC, r *types.Receipt) {
			r.TransactionIndex = incidentPaymentIndex + 1
		}, "invalid"},
		{"wrong receipt block hash", func(_ *FullBlock, _ *forcedPaymentRPC, r *types.Receipt) { r.BlockHash = common.Hash{} }, "invalid"},
		{"wrong receipt block number", func(_ *FullBlock, _ *forcedPaymentRPC, r *types.Receipt) { r.BlockNumber = big.NewInt(1) }, "invalid"},
		{"wrong receipt tx hash", func(_ *FullBlock, _ *forcedPaymentRPC, r *types.Receipt) { r.TxHash = common.Hash{} }, "invalid"},
		{"ambiguous equal values", func(b *FullBlock, _ *forcedPaymentRPC, _ *types.Receipt) {
			b.ConsensusBlock.Fulu.Message.Body.ExecutionPayload.Transactions = append(b.GetBlockTransactions(), b.GetBlockTransactions()[incidentPaymentIndex])
		}, "ambiguous"},
		{"partial or split payment", func(b *FullBlock, s *forcedPaymentRPC, _ *types.Receipt) {
			s.balances[hexutil.EncodeUint64(b.GetBlockNumber())].Sub(s.balances[hexutil.EncodeUint64(b.GetBlockNumber())], big.NewInt(1))
		}, "no exact transaction match"},
		{"unexplained outflow", func(b *FullBlock, s *forcedPaymentRPC, _ *types.Receipt) {
			s.balances[hexutil.EncodeUint64(b.GetBlockNumber())] = big.NewInt(0)
		}, "outflow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := incidentFixture(t)
			b := f.block(t)
			o, backend := newForcedPaymentRPC(t, b, decimal(t, f.BalanceBefore), decimal(t, f.BalanceAfter), f.Receipt)
			tt.mutate(b, backend, f.Receipt)
			err := o.DetectForcedMevPayment(b)
			require.ErrorContains(t, err, tt.want)
			require.Nil(t, b.ForcedMevPayment)
			require.Len(t, b.Events.EtherReceived, 1, "no synthetic credit on failure")
		})
	}
}

func Test_AnyPositionForcedPayment_ActivationAndVanilla(t *testing.T) {
	for _, slot := range []uint64{14950447, 14950448, 15194838, 15194839, 15194840} {
		t.Run(fmt.Sprint(slot), func(t *testing.T) {
			f := incidentFixture(t)
			f.Slot = slot
			b := f.block(t)
			o, backend := newForcedPaymentRPC(t, b, decimal(t, f.BalanceBefore), decimal(t, f.BalanceAfter), f.Receipt)
			require.Equal(t, slot >= 15194839, b.IsAnyPositionForcedPaymentDetectionActive())
			require.NoError(t, o.DetectForcedMevPayment(b))
			if slot < 15194839 {
				require.Zero(t, backend.callCount())
				require.Nil(t, b.ForcedMevPayment)
			} else {
				require.Equal(t, 3, backend.callCount())
				require.NotNil(t, b.ForcedMevPayment)
			}
		})
	}
	f := incidentFixture(t)
	b := f.block(t)
	copy(b.ConsensusBlock.Fulu.Message.Body.ExecutionPayload.FeeRecipient[:], common.HexToAddress(testPoolAddress).Bytes())
	o, backend := newForcedPaymentRPC(t, b, decimal(t, f.BalanceBefore), decimal(t, f.BalanceAfter), f.Receipt)
	require.NoError(t, o.DetectForcedMevPayment(b))
	require.Nil(t, b.ForcedMevPayment)
	require.Zero(t, backend.callCount(), "vanilla fee-recipient accounting remains on the existing path")
	b.ConsensusBlock = nil
	require.NoError(t, o.DetectForcedMevPayment(b))
	require.Zero(t, backend.callCount(), "missed slots do not cause balance RPCs")
}

func Test_AnyPositionForcedPayment_EqualDonationIsPreserved(t *testing.T) {
	f := incidentFixture(t)
	b := f.block(t)
	donation := decimal(t, f.ExpectedReward)
	difference := new(big.Int).Sub(donation, b.Events.EtherReceived[0].DonationAmount)
	b.Events.EtherReceived[0].DonationAmount.Set(donation)
	after := new(big.Int).Add(decimal(t, f.BalanceAfter), difference)
	o, _ := newForcedPaymentRPC(t, b, decimal(t, f.BalanceBefore), after, f.Receipt)
	require.NoError(t, o.DetectForcedMevPayment(b))
	require.Len(t, b.GetDonations(testPoolAddress), 1)
	require.Equal(t, donation, b.GetDonations(testPoolAddress)[0].DonationAmount)
}

func Test_AnyPositionForcedPayment_Positions(t *testing.T) {
	for _, index := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			b := buildMevBlock(t, 15195000, 123, 42, titanForwarder, big.NewInt(1000))
			paymentRaw := b.GetBlockTransactions()[0]
			tail := buildMevBlock(t, 15195000, 123, 42, "0x1111111111111111111111111111111111111111", big.NewInt(0)).GetBlockTransactions()[0]
			txs := []bellatrix.Transaction{tail, tail, tail}
			txs[index] = paymentRaw
			b.ConsensusBlock.Bellatrix.Message.Body.ExecutionPayload.Transactions = txs
			tx, err := utils.DecodeTx(paymentRaw)
			require.NoError(t, err)
			receipt := &types.Receipt{TxHash: tx.Hash(), BlockNumber: big.NewInt(123), TransactionIndex: uint(index), Status: 1, Logs: []*types.Log{}}
			o, backend := newForcedPaymentRPC(t, b, big.NewInt(0), big.NewInt(1000), receipt)
			require.NoError(t, o.DetectForcedMevPayment(b))
			require.Equal(t, tx.Hash().Hex(), b.ForcedMevPayment.TxHash)
			require.Equal(t, uint(index), b.ForcedMevPayment.TxIndex)
			require.Equal(t, 3, backend.callCount())
		})
	}
}

func Test_AnyPositionForcedPayment_ExplainedFlows(t *testing.T) {
	tests := []struct {
		name   string
		delta  int64
		events *Events
	}{
		{"unrelated transfers", 0, &Events{}},
		{"donation", 1000, &Events{EtherReceived: []*contract.ContractEtherReceived{etherReceivedEvent(big.NewInt(1000))}}},
		{"subscription", 1000, &Events{SubscribeValidator: []*contract.ContractSubscribeValidator{subscribeEvent(big.NewInt(1000))}}},
		{"claim", -1000, &Events{ClaimRewards: []*contract.ContractClaimRewards{claimRewardsEvent(big.NewInt(1000))}}},
		{"donation and claim", -1000, &Events{EtherReceived: []*contract.ContractEtherReceived{etherReceivedEvent(big.NewInt(1000))}, ClaimRewards: []*contract.ContractClaimRewards{claimRewardsEvent(big.NewInt(2000))}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := buildMevBlock(t, 15195000, 123, 42, titanForwarder, big.NewInt(1000))
			b.Events = tt.events
			o, backend := newForcedPaymentRPC(t, b, big.NewInt(2000), big.NewInt(2000+tt.delta), nil)
			require.NoError(t, o.DetectForcedMevPayment(b))
			require.Nil(t, b.ForcedMevPayment, "a coincidentally equal tx value must not turn explained flows into MEV")
			require.Equal(t, 2, backend.callCount())
		})
	}
}

func Test_AnyPositionForcedPayment_ConsensusWithdrawalIsNotMev(t *testing.T) {
	f := incidentFixture(t)
	b := f.block(t)
	withdrawal := &capella.Withdrawal{Amount: 1}
	copy(withdrawal.Address[:], common.HexToAddress(testPoolAddress).Bytes())
	b.ConsensusBlock.Fulu.Message.Body.ExecutionPayload.Withdrawals = append(b.ConsensusBlock.Fulu.Message.Body.ExecutionPayload.Withdrawals, withdrawal)
	o, _ := newForcedPaymentRPC(t, b, decimal(t, f.BalanceBefore), decimal(t, f.BalanceAfter), f.Receipt)
	require.ErrorContains(t, o.DetectForcedMevPayment(b), "consensus withdrawal")
	require.Nil(t, b.ForcedMevPayment)
}
