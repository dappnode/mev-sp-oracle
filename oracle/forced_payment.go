package oracle

import (
	"context"
	"fmt"
	"math/big"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/avast/retry-go/v4"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// DetectForcedMevPayment is called before advancing the oracle state. Balance
// reads happen once per block, not once per candidate. Ordinary blocks return
// before decoding transactions; only an unexplained inflow requires a scan.
// Like the original detector, this proves the amount at block granularity, not
// a call trace. A unique transaction with exactly that value identifies the
// candidate. Ambiguous or unsupported inflows stop processing instead of
// silently banning a validator or allocating an arbitrary amount.
func (o *Onchain) DetectForcedMevPayment(b *FullBlock, opts ...retry.Option) error {
	if b.ConsensusBlock == nil || !b.IsForcedPaymentDetectionActive() ||
		!b.IsAnyPositionForcedPaymentDetectionActive() {
		return nil
	}
	return o.detectAnyPositionForcedMevPayment(b, opts...)
}

func (o *Onchain) detectAnyPositionForcedMevPayment(b *FullBlock, opts ...retry.Option) error {
	if _, _, exception := b.mevRewardException(); exception {
		return nil
	}
	// Protocol tips also increase the balance without an event. Keep the
	// existing vanilla-reward path for blocks whose fee recipient is the pool.
	if utils.Equals(b.GetFeeRecipient(), o.PoolAddress) {
		return nil
	}
	blockNumber := b.GetBlockNumber()
	if blockNumber == 0 {
		return fmt.Errorf("cannot verify forced payment for block 0")
	}
	if b.ForcedMevPayment != nil && b.ForcedMevPayment.Delivered {
		return nil // Do not insert a second synthetic event on repeated calls.
	}
	after, err := o.GetPoolEthBalance(new(big.Int).SetUint64(blockNumber), opts...)
	if err != nil {
		return err
	}
	before, err := o.GetPoolEthBalance(new(big.Int).SetUint64(blockNumber-1), opts...)
	if err != nil {
		return err
	}
	inflow := unexplainedPoolInflow(new(big.Int).Sub(after, before), b.Events)
	if inflow.Sign() == 0 {
		return nil
	}
	if inflow.Sign() < 0 {
		return fmt.Errorf("block %d has unexplained pool outflow %s wei", blockNumber, inflow)
	}
	if _, isMev, recipient := b.MevRewardInWei(); isMev && utils.Equals(recipient, o.PoolAddress) {
		return fmt.Errorf("block %d has both a direct MEV payment and unexplained inflow %s wei; cannot select a single proposer reward", blockNumber, inflow)
	}
	// A consensus withdrawal can credit any address, including the pool. It
	// must not be mistaken for a proposer payment of the same value.
	if b.ConsensusBlock.Version >= spec.DataVersionCapella {
		withdrawals, err := b.ConsensusBlock.Withdrawals()
		if err != nil {
			return fmt.Errorf("read withdrawals at block %d: %w", blockNumber, err)
		}
		for _, withdrawal := range withdrawals {
			if withdrawal.Amount > 0 && utils.Equals(withdrawal.Address.String(), o.PoolAddress) {
				return fmt.Errorf("block %d credits a consensus withdrawal to the pool; cannot attribute unexplained inflow %s wei to MEV", blockNumber, inflow)
			}
		}
	}
	tx, index, err := b.findForcedPaymentCandidate(o.PoolAddress, inflow)
	if err != nil {
		return err
	}
	receipt, err := o.forcedPaymentReceipt(b, tx, index, opts...)
	if err != nil {
		return err
	}
	payer := receipt.ContractAddress
	if tx.To() != nil {
		payer = *tx.To()
	}
	if payer == (common.Address{}) {
		return fmt.Errorf("forced payment transaction %s has no recipient or created contract", tx.Hash())
	}
	b.SetForcedMevPayment(&ForcedMevPayment{
		Delivered: true, AmountWei: new(big.Int).Set(inflow),
		Payer: payer.Hex(), TxHash: tx.Hash().Hex(), TxIndex: uint(index),
		BlockNumber: blockNumber, BlockHash: receipt.BlockHash.Hex(),
	}, o.PoolAddress)
	return nil
}

func (b *FullBlock) findForcedPaymentCandidate(poolAddress string, inflow *big.Int) (*types.Transaction, int, error) {
	var candidate *types.Transaction
	candidateIndex := 0
	for index, rawTx := range b.GetBlockTransactions() {
		tx, err := utils.DecodeTx(rawTx)
		if err != nil {
			return nil, 0, fmt.Errorf("decode transaction %d in block %d: %w", index, b.GetBlockNumber(), err)
		}
		if tx.Value().Sign() <= 0 || tx.Value().Cmp(inflow) != 0 ||
			(tx.To() != nil && utils.Equals(tx.To().Hex(), poolAddress)) {
			continue
		}
		if candidate != nil {
			return nil, 0, fmt.Errorf("block %d has ambiguous forced payment: transactions %s and %s both match unexplained inflow %s wei", b.GetBlockNumber(), candidate.Hash(), tx.Hash(), inflow)
		}
		candidate, candidateIndex = tx, index
	}
	if candidate == nil {
		return nil, 0, fmt.Errorf("block %d has unexplained pool inflow %s wei with no exact transaction match", b.GetBlockNumber(), inflow)
	}
	return candidate, candidateIndex, nil
}

// Reuse already fetched receipts. An unknown proposer only needs this one
// receipt, rather than the old detector's header plus every block receipt.
func (o *Onchain) forcedPaymentReceipt(b *FullBlock, tx *types.Transaction, index int, opts ...retry.Option) (*types.Receipt, error) {
	var receipt *types.Receipt
	if index < len(b.ExecutionReceipts) {
		receipt = b.ExecutionReceipts[index]
	} else {
		err := retry.Do(func() error {
			var err error
			receipt, err = o.ExecutionClient.TransactionReceipt(context.Background(), tx.Hash())
			return err
		}, o.GetRetryOpts(opts)...)
		if err != nil {
			return nil, fmt.Errorf("read forced payment receipt %s: %w", tx.Hash(), err)
		}
	}
	blockHash, err := b.ConsensusBlock.ExecutionBlockHash()
	if err != nil {
		return nil, err
	}
	if receipt == nil || receipt.TxHash != tx.Hash() || receipt.BlockNumber == nil ||
		receipt.BlockNumber.Uint64() != b.GetBlockNumber() || receipt.BlockHash != common.Hash(blockHash) ||
		receipt.TransactionIndex != uint(index) || receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("invalid or unsuccessful forced payment receipt for transaction %s in block %d at index %d", tx.Hash(), b.GetBlockNumber(), index)
	}
	return receipt, nil
}
