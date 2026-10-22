package oracle

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/constants"
	"github.com/stretchr/testify/require"
)

const ethInGwei = phase0.Gwei(1_000_000_000)

func Test_BuilderPaymentQuorum(t *testing.T) {
	// 38.4M ETH active is 1.2M ETH per slot, of which 60% is 720k ETH
	require.Equal(t, 720_000*ethInGwei, builderPaymentQuorum(38_400_000*ethInGwei))

	// Integer division as in the spec: per slot first, then the ratio.
	// 1000000000031 / 32 = 31250000000 (rounded down), * 6 / 10 = 18750000000
	require.Equal(t, phase0.Gwei(18_750_000_000), builderPaymentQuorum(1_000_000_000_031))

	// The total active balance has a floor of one effective balance increment
	require.Equal(t, phase0.Gwei(18_750_000), builderPaymentQuorum(0))
	require.Equal(t, builderPaymentQuorum(0), builderPaymentQuorum(effectiveBalanceIncrement))
}

func Test_BuilderPaymentDecisionSlot(t *testing.T) {
	// Last slot of the epoch after the slot's one, for any slot of the epoch
	require.Equal(t, uint64(95), builderPaymentDecisionSlot(32))
	require.Equal(t, uint64(95), builderPaymentDecisionSlot(63))
	require.Equal(t, uint64(127), builderPaymentDecisionSlot(64))
}

// The builder pending payments vector at the decision state: the first half is
// the previous epoch, where the withheld bid sits, the second the current one
func pendingPaymentsWith(slot uint64, payment *gloas.BuilderPendingPayment) []*gloas.BuilderPendingPayment {
	payments := make([]*gloas.BuilderPendingPayment, 2*constants.SlotsInEpoch)
	for i := range payments {
		payments[i] = &gloas.BuilderPendingPayment{Withdrawal: &gloas.BuilderPendingWithdrawal{}}
	}
	payments[slot%constants.SlotsInEpoch] = payment
	return payments
}

func Test_IsWithheldBidPaid(t *testing.T) {
	slot := testGloasSlot + 5
	proposer := phase0.ValidatorIndex(1000)
	bid := gloasBid(slot, 42, 48_200_000, testPoolAddress, 0xaa, 0xbb)
	totalActive := 38_400_000 * ethInGwei
	quorum := builderPaymentQuorum(totalActive)

	paymentWithWeight := func(weight phase0.Gwei) *gloas.BuilderPendingPayment {
		return &gloas.BuilderPendingPayment{
			Weight:        weight,
			ProposerIndex: proposer,
			Withdrawal: &gloas.BuilderPendingWithdrawal{
				FeeRecipient: bid.FeeRecipient,
				Amount:       bid.Value,
				BuilderIndex: bid.BuilderIndex,
			},
		}
	}

	tests := []struct {
		name     string
		payments []*gloas.BuilderPendingPayment
		paid     bool
		err      string
	}{
		{"votes below the quorum", pendingPaymentsWith(slot, paymentWithWeight(quorum-1)), false, ""},
		{"votes at the quorum", pendingPaymentsWith(slot, paymentWithWeight(quorum)), true, ""},
		{"votes above the quorum", pendingPaymentsWith(slot, paymentWithWeight(quorum*2)), true, ""},
		// Slashing the proposer empties its pending payments
		{"cancelled payment", pendingPaymentsWith(slot, &gloas.BuilderPendingPayment{
			Weight: quorum * 2, Withdrawal: &gloas.BuilderPendingWithdrawal{}}), false, ""},
		{"no withdrawal", pendingPaymentsWith(slot, &gloas.BuilderPendingPayment{Weight: quorum * 2}), false, ""},
		{"other builder", pendingPaymentsWith(slot, func() *gloas.BuilderPendingPayment {
			p := paymentWithWeight(quorum)
			p.Withdrawal.BuilderIndex = 43
			return p
		}()), false, "does not match its bid"},
		{"other amount", pendingPaymentsWith(slot, func() *gloas.BuilderPendingPayment {
			p := paymentWithWeight(quorum)
			p.Withdrawal.Amount++
			return p
		}()), false, "does not match its bid"},
		{"other fee recipient", pendingPaymentsWith(slot, func() *gloas.BuilderPendingPayment {
			p := paymentWithWeight(quorum)
			p.Withdrawal.FeeRecipient = executionAddress(testBuilderAddress)
			return p
		}()), false, "does not match its bid"},
		{"other proposer", pendingPaymentsWith(slot, func() *gloas.BuilderPendingPayment {
			p := paymentWithWeight(quorum)
			p.ProposerIndex = 1001
			return p
		}()), false, "does not match its bid"},
		{"not a full vector", pendingPaymentsWith(slot, paymentWithWeight(quorum))[:32], false, "expected 64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paid, err := isWithheldBidPaid(bid, proposer, tt.payments, totalActive)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.paid, paid)
		})
	}

	// The current epoch half belongs to other slots, even if it looks alike
	t.Run("reads the previous epoch half", func(t *testing.T) {
		payments := pendingPaymentsWith(slot, paymentWithWeight(0))
		payments[constants.SlotsInEpoch+slot%constants.SlotsInEpoch] = paymentWithWeight(quorum * 2)
		paid, err := isWithheldBidPaid(bid, proposer, payments, totalActive)
		require.NoError(t, err)
		require.False(t, paid)
	})
}

func Test_BidPromisesPaymentTo(t *testing.T) {
	tests := []struct {
		name     string
		bid      *gloas.ExecutionPayloadBid
		promises bool
	}{
		{"builder pays the pool", gloasBid(testGloasSlot, 42, 1, testPoolAddress, 0xaa, 0xbb), true},
		{"builder pays nothing", gloasBid(testGloasSlot, 42, 0, testPoolAddress, 0xaa, 0xbb), false},
		{"builder pays someone else", gloasBid(testGloasSlot, 42, 1, testBuilderAddress, 0xaa, 0xbb), false},
		{"self-built", gloasBid(testGloasSlot, gloas.BuilderIndexSelfBuild, 0, testPoolAddress, 0xaa, 0xbb), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullBlock := gloasFullBlock(t, testGloasSlot, 1000, tt.bid)
			require.Equal(t, tt.promises, fullBlock.BidPromisesPaymentTo(testPoolAddress))
			require.Equal(t, tt.promises, fullBlock.BidPromisesPaymentTo(strings.ToLower(testPoolAddress)))
		})
	}
}

func Test_SetBuilderPayment(t *testing.T) {
	bid := gloasBid(testGloasSlot+3, 42, 48_200_000, testPoolAddress, 0xaa, 0xbb)
	fullBlock := gloasFullBlock(t, testGloasSlot+3, 1000, bid)

	fullBlock.SetBuilderPayment(true)
	require.Equal(t, &BuilderPayment{
		Slot:          testGloasSlot + 3,
		ProposerIndex: 1000,
		BuilderIndex:  42,
		AmountWei:     decimal(t, "48200000000000000"),
		Paid:          true,
	}, fullBlock.BuilderPayment)

	fullBlock.SetBuilderPayment(false)
	require.False(t, fullBlock.BuilderPayment.Paid)
}

func Test_FetchBuilderPendingPayments(t *testing.T) {
	// Shaped as the beacon API serves it: numbers as strings
	body := `{"version":"gloas","execution_optimistic":false,"finalized":true,"data":[` +
		`{"weight":"720000000000000","withdrawal":{"fee_recipient":"` + strings.ToLower(testPoolAddress) +
		`","amount":"48200000","builder_index":"42"},"proposer_index":"1000"},` +
		`{"weight":"0","withdrawal":{"fee_recipient":"0x0000000000000000000000000000000000000000","amount":"0","builder_index":"0"},"proposer_index":"0"}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eth/v1/beacon/states/4235327/builder_pending_payments" {
			http.Error(w, `{"code":404,"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	payments, err := fetchBuilderPendingPayments(server.URL, "4235327")
	require.NoError(t, err)
	require.Len(t, payments, 2)
	require.Equal(t, 720_000*ethInGwei, payments[0].Weight)
	require.Equal(t, phase0.ValidatorIndex(1000), payments[0].ProposerIndex)
	require.Equal(t, gloas.BuilderIndex(42), payments[0].Withdrawal.BuilderIndex)
	require.Equal(t, phase0.Gwei(48_200_000), payments[0].Withdrawal.Amount)
	require.Equal(t, executionAddress(testPoolAddress), payments[0].Withdrawal.FeeRecipient)
	require.Zero(t, payments[1].Withdrawal.Amount)

	_, err = fetchBuilderPendingPayments(server.URL, "1")
	require.ErrorContains(t, err, "non-200 response: 404")
}

func Test_TotalActiveBalance(t *testing.T) {
	validator := func(status v1.ValidatorState, effectiveBalanceEth phase0.Gwei) *v1.Validator {
		return &v1.Validator{Status: status, Validator: &phase0.Validator{EffectiveBalance: effectiveBalanceEth * ethInGwei}}
	}
	validators := map[phase0.ValidatorIndex]*v1.Validator{
		1: validator(v1.ValidatorStateActiveOngoing, 32),
		2: validator(v1.ValidatorStateActiveExiting, 2048),
		3: validator(v1.ValidatorStateActiveSlashed, 31),
		// Not active, so not counted even if the beacon node returned them
		4: validator(v1.ValidatorStatePendingQueued, 32),
		5: validator(v1.ValidatorStateExitedUnslashed, 32),
	}
	require.Equal(t, (32+2048+31)*ethInGwei, totalActiveBalance(validators))
	require.Zero(t, totalActiveBalance(nil))
}
