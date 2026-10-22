package oracle

import (
	"fmt"
	"math/big"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/constants"
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// A builder's payment to the pool is credited when its slot is processed, but
// it reaches the pool later, as a withdrawal in another execution payload (see
// gloas.go). Until then it is in flight: the oracle already owes it to the
// validators, but the pool doesn't hold it yet. This file keeps track of them.

// BUILDER_INDEX_FLAG in the Gloas spec. Builder withdrawals reuse the
// validator index field of the withdrawal, with this bit set
const builderIndexFlag = phase0.ValidatorIndex(1 << 40)

// Whether the withdrawal pays a builder payment rather than a validator's
func isBuilderWithdrawal(withdrawal *capella.Withdrawal) bool {
	return withdrawal.ValidatorIndex&builderIndexFlag != 0
}

// Returns the builder payments the execution payload of the slot paid to the
// given address. Empty before Gloas and for slots without a payload
func (b *FullBlock) BuilderWithdrawalsTo(address string) []*capella.Withdrawal {
	if !b.IsGloas() || !b.HasExecutionPayload() {
		return nil
	}
	received := make([]*capella.Withdrawal, 0)
	for _, withdrawal := range b.gloasExecutionPayload().Withdrawals {
		// A zero amount moves no ETH, so there is nothing to match
		if withdrawal.Amount > 0 && isBuilderWithdrawal(withdrawal) && utils.Equals(withdrawal.Address.String(), address) {
			received = append(received, withdrawal)
		}
	}
	return received
}

// Total amount of the builder payments the slot paid to the given address
func (b *FullBlock) BuilderWithdrawalsWeiTo(address string) *big.Int {
	total := big.NewInt(0)
	for _, withdrawal := range b.BuilderWithdrawalsTo(address) {
		total.Add(total, utils.GweiToWei(uint64(withdrawal.Amount)))
	}
	return total
}

// Starts tracking a builder payment just credited to the pool
func (or *Oracle) trackBuilderPayment(payment *BuilderPayment) {
	or.state.InFlightBuilderPayments = append(or.state.InFlightBuilderPayments, payment)
	log.WithFields(log.Fields{
		"Slot":         payment.Slot,
		"BuilderIndex": payment.BuilderIndex,
		"AmountWei":    payment.AmountWei,
	}).Info("Builder payment to the pool credited, waiting for its withdrawal")
}

// Stops tracking the builder payments whose withdrawal reached the pool in the
// execution payload of this slot. Every builder withdrawal to the pool must
// match a credited payment: otherwise that ETH would be left unallocated, so
// the slot is rejected and the state left untouched.
//
// A payment is matched by builder and amount. When two in flight share both,
// which one is matched first makes no difference to the books.
func (or *Oracle) settleArrivedBuilderPayments(fullBlock *FullBlock) error {
	withdrawals := fullBlock.BuilderWithdrawalsTo(or.cfg.PoolAddress)
	if len(withdrawals) == 0 {
		return nil
	}

	remaining := append([]*BuilderPayment{}, or.state.InFlightBuilderPayments...)
	arrived := make([]*BuilderPayment, 0, len(withdrawals))

	for _, withdrawal := range withdrawals {
		builderIndex := uint64(withdrawal.ValidatorIndex &^ builderIndexFlag)
		amountWei := utils.GweiToWei(uint64(withdrawal.Amount))

		matched := -1
		for i, payment := range remaining {
			if payment.BuilderIndex == builderIndex && payment.AmountWei.Cmp(amountWei) == 0 {
				matched = i
				break
			}
		}
		if matched < 0 {
			return errors.New(fmt.Sprintf("withdrawal %d in block %d pays %s wei from builder %d to the pool, "+
				"but no such builder payment was credited", withdrawal.Index, fullBlock.GetBlockNumber(), amountWei, builderIndex))
		}
		arrived = append(arrived, remaining[matched])
		remaining = append(remaining[:matched], remaining[matched+1:]...)
	}

	or.state.InFlightBuilderPayments = remaining
	for _, payment := range arrived {
		log.WithFields(log.Fields{
			"Slot":         payment.Slot,
			"BuilderIndex": payment.BuilderIndex,
			"AmountWei":    payment.AmountWei,
			"ArrivalBlock": fullBlock.GetBlockNumber(),
		}).Info("Builder payment reached the pool")
	}
	return nil
}

// A builder payment reaches the pool within about two epochs of its slot: in
// the next full payload, or after the end of the next epoch if its payload was
// withheld. One still in flight long after may never arrive, and meanwhile it
// counts as an asset in the onchain reconciliation, so it is worth a look.
//
// This only warns. Stopping would be wrong: slots are processed in order, so
// the oracle could never reach the one where a slow payment finally lands.
const lateBuilderPaymentSlots = 4 * 32

// Warns, once per epoch, about the builder payments still in flight long
// after their slot
func (or *Oracle) warnAboutLateBuilderPayments(processedSlot uint64) {
	if processedSlot%constants.SlotsInEpoch != 0 {
		return
	}
	for _, payment := range or.state.InFlightBuilderPayments {
		if processedSlot > payment.Slot+lateBuilderPaymentSlots {
			log.WithFields(log.Fields{
				"Slot":          payment.Slot,
				"BuilderIndex":  payment.BuilderIndex,
				"AmountWei":     payment.AmountWei,
				"SlotsInFlight": processedSlot - payment.Slot,
			}).Warn("Builder payment credited long ago still has not reached the pool, check it was paid")
		}
	}
}

// Total amount of the builder payments credited but not in the pool yet
func (or *Oracle) inFlightBuilderPaymentsWei() *big.Int {
	total := big.NewInt(0)
	for _, payment := range or.state.InFlightBuilderPayments {
		total.Add(total, payment.AmountWei)
	}
	return total
}
