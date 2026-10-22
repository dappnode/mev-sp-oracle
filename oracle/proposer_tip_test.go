package oracle

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const gwei = int64(1_000_000_000)

// A tx of the block together with the gas its receipt says it was charged for
type chargedTx struct {
	tx      *types.Transaction
	gasUsed uint64
}

// Signs one tx of every type the oracle supports. Fee caps are chosen so that
// some txs pay their full tip and others are capped by the fee cap
func txsOfEveryType(t *testing.T, key *ecdsa.PrivateKey) []chargedTx {
	t.Helper()
	chainId := new(big.Int).SetUint64(MainnetChainId)
	signer := types.LatestSignerForChainID(chainId)
	to := common.HexToAddress("0x5555555555555555555555555555555555555555")

	sign := func(data types.TxData) *types.Transaction {
		tx, err := types.SignNewTx(key, signer, data)
		require.NoError(t, err)
		return tx
	}
	txs := []*types.Transaction{
		sign(&types.LegacyTx{Nonce: 0, GasPrice: big.NewInt(13 * gwei), Gas: 21000, To: &to}),
		sign(&types.AccessListTx{ChainID: chainId, Nonce: 1, GasPrice: big.NewInt(11 * gwei), Gas: 30000, To: &to}),
		// Full tip paid: base fee + tip is below the fee cap
		sign(&types.DynamicFeeTx{ChainID: chainId, Nonce: 2, GasTipCap: big.NewInt(2 * gwei), GasFeeCap: big.NewInt(50 * gwei), Gas: 90000, To: &to}),
		// Tip capped by the fee cap
		sign(&types.DynamicFeeTx{ChainID: chainId, Nonce: 3, GasTipCap: big.NewInt(9 * gwei), GasFeeCap: big.NewInt(12 * gwei), Gas: 50000, To: &to}),
		sign(&types.BlobTx{ChainID: uint256.NewInt(MainnetChainId), Nonce: 4, GasTipCap: uint256.NewInt(uint64(3 * gwei)),
			GasFeeCap: uint256.NewInt(uint64(40 * gwei)), Gas: 21000, To: to, BlobFeeCap: uint256.NewInt(1),
			BlobHashes: []common.Hash{{0x01}}}),
		sign(&types.SetCodeTx{ChainID: uint256.NewInt(MainnetChainId), Nonce: 5, GasTipCap: uint256.NewInt(uint64(1 * gwei)),
			GasFeeCap: uint256.NewInt(uint64(30 * gwei)), Gas: 60000, To: to}),
	}
	gasUsed := []uint64{21000, 25400, 61234, 50000, 21000, 47311}

	charged := make([]chargedTx, len(txs))
	for i := range txs {
		charged[i] = chargedTx{tx: txs[i], gasUsed: gasUsed[i]}
	}
	return charged
}

// Builds a Fulu block containing the given txs and their receipts. blockGasUsed
// is what the block header says, which since Glamsterdam can differ from the
// sum of the receipts
func blockWithReceipts(t *testing.T, txs []chargedTx, baseFeeWei int64, blockGasUsed uint64) *FullBlock {
	t.Helper()
	rawTxs := make([]bellatrix.Transaction, len(txs))
	receipts := make([]*types.Receipt, len(txs))
	for i, charged := range txs {
		raw, err := charged.tx.MarshalBinary()
		require.NoError(t, err)
		rawTxs[i] = raw
		receipts[i] = &types.Receipt{TxHash: charged.tx.Hash(), GasUsed: charged.gasUsed}
	}

	fullBlock := NewFullBlock(&v1.ProposerDuty{Slot: 100, ValidatorIndex: 7},
		&v1.Validator{Index: 7, Validator: &phase0.Validator{}}, MainnetChainId)
	fullBlock.SetConsensusBlock(&spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionFulu,
		Fulu: &electra.SignedBeaconBlock{Message: &electra.BeaconBlock{
			Slot: 100, ProposerIndex: 7,
			Body: &electra.BeaconBlockBody{ExecutionPayload: &deneb.ExecutionPayload{
				BlockNumber:   1000,
				GasUsed:       blockGasUsed,
				BaseFeePerGas: uint256.NewInt(uint64(baseFeeWei)),
				Transactions:  rawTxs,
			}},
		}},
	})
	fullBlock.ExecutionHeader = &types.Header{Number: big.NewInt(1000), BaseFee: big.NewInt(baseFeeWei), GasUsed: blockGasUsed}
	fullBlock.ExecutionReceipts = receipts
	return fullBlock
}

func sumGasUsed(txs []chargedTx) uint64 {
	total := uint64(0)
	for _, charged := range txs {
		total += charged.gasUsed
	}
	return total
}

// The calculation used until Glamsterdam: every fee paid minus what the block
// burnt, using the block gas_used. Kept here to prove the new one matches it
func formerProposerTip(txs []chargedTx, baseFee *big.Int, blockGasUsed uint64) *big.Int {
	fees := big.NewInt(0)
	for _, charged := range txs {
		pricePerGas := new(big.Int).Add(charged.tx.GasTipCap(), baseFee)
		if pricePerGas.Cmp(charged.tx.GasFeeCap()) > 0 {
			pricePerGas = charged.tx.GasFeeCap()
		}
		fees.Add(fees, new(big.Int).Mul(pricePerGas, new(big.Int).SetUint64(charged.gasUsed)))
	}
	burnt := new(big.Int).Mul(new(big.Int).SetUint64(blockGasUsed), baseFee)
	return fees.Sub(fees, burnt)
}

// Before Glamsterdam the block gas_used is the sum of the receipts, and the new
// calculation must give exactly the old result so past rewards don't change
func Test_GetProposerTip_MatchesFormerCalculationBeforeGlamsterdam(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	txs := txsOfEveryType(t, key)

	for _, baseFeeGwei := range []int64{1, 7, 10} {
		baseFee := big.NewInt(baseFeeGwei * gwei)
		fullBlock := blockWithReceipts(t, txs, baseFee.Int64(), sumGasUsed(txs))

		tip, err := fullBlock.GetProposerTip()
		require.NoError(t, err)
		require.Equal(t, formerProposerTip(txs, baseFee, sumGasUsed(txs)), tip, "base fee %d gwei", baseFeeGwei)
	}
}

// EIP-7778 stops subtracting refunds from the block gas_used, so it can exceed
// what the txs were charged. The tip must only count what was actually charged
func Test_GetProposerTip_BlockGasUsedIncludesRefunds(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	txs := txsOfEveryType(t, key)
	baseFee := big.NewInt(7 * gwei)
	refunds := uint64(19_800)
	fullBlock := blockWithReceipts(t, txs, baseFee.Int64(), sumGasUsed(txs)+refunds)

	tip, err := fullBlock.GetProposerTip()
	require.NoError(t, err)

	// Worked out by hand at a 7 gwei base fee: legacy 6 gwei, access list
	// 4 gwei, dynamic fee 2 gwei (full tip) and 5 gwei (capped by its 12 gwei
	// fee cap), blob 3 gwei and set code 1 gwei per unit of gas
	expected := big.NewInt(0)
	for i, tipGwei := range []int64{6, 4, 2, 5, 3, 1} {
		perTx := new(big.Int).Mul(big.NewInt(tipGwei*gwei), new(big.Int).SetUint64(txs[i].gasUsed))
		expected.Add(expected, perTx)
	}
	require.Equal(t, expected, tip)

	// The former calculation would have burnt the refunds out of the proposer's
	// reward, leaving the pool with more ETH than the oracle accounts for
	under := new(big.Int).Sub(expected, formerProposerTip(txs, baseFee, sumGasUsed(txs)+refunds))
	require.Equal(t, new(big.Int).Mul(new(big.Int).SetUint64(refunds), baseFee), under)
}

func Test_GetProposerTip_Errors(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	txs := txsOfEveryType(t, key)

	t.Run("no receipts", func(t *testing.T) {
		fullBlock := blockWithReceipts(t, txs, 7*gwei, sumGasUsed(txs))
		fullBlock.ExecutionReceipts = nil
		_, err := fullBlock.GetProposerTip()
		require.ErrorContains(t, err, "receipts of full block are nil")
	})

	t.Run("missing receipt", func(t *testing.T) {
		fullBlock := blockWithReceipts(t, txs, 7*gwei, sumGasUsed(txs))
		fullBlock.ExecutionReceipts = fullBlock.ExecutionReceipts[1:]
		_, err := fullBlock.GetProposerTip()
		require.ErrorContains(t, err, "not the same length")
	})

	// Used to return a nil error and a nil tip
	t.Run("receipt of another tx", func(t *testing.T) {
		fullBlock := blockWithReceipts(t, txs, 7*gwei, sumGasUsed(txs))
		fullBlock.ExecutionReceipts[0].TxHash = common.Hash{0x42}
		tip, err := fullBlock.GetProposerTip()
		require.ErrorContains(t, err, "tx hash does not match receipt hash")
		require.Nil(t, tip)
	})

	t.Run("fee cap below base fee", func(t *testing.T) {
		// The capped dynamic fee tx pays at most 12 gwei
		fullBlock := blockWithReceipts(t, txs, 20*gwei, sumGasUsed(txs))
		_, err := fullBlock.GetProposerTip()
		require.ErrorContains(t, err, "could not get effective tip")
	})
}

func Test_GetBaseFeePerGasBigInt(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	fullBlock := blockWithReceipts(t, txsOfEveryType(t, key), 1_234_567_891, 0)
	require.Equal(t, big.NewInt(1_234_567_891), fullBlock.GetBaseFeePerGasBigInt())
}
