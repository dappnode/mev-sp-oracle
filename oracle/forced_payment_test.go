package oracle

import (
	"math/big"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The contract Titan uses to forward the MEV payment to the pool. It receives
// the ETH in the last tx of the block and forwards it with SELFDESTRUCT, which
// never executes the pool code and therefore emits no EtherReceived event
const titanForwarder = "0xFEEEEEE44046c3f61a8CC081E0918eF0de0a7ffC"

const testPoolAddress = "0xAdFb8D27671F14f297eE94135e266aAFf8752e35"

// Builds a block whose last tx sends `reward` to `to`, sent by the block fee
// recipient, which is what makes the oracle consider it a MEV payment
func buildMevBlock(t *testing.T, slot uint64, blockNumber uint64,
	validatorIndex phase0.ValidatorIndex, to string, reward *big.Int) *FullBlock {

	t.Helper()

	// The builder pays the proposer. Its address is the block fee recipient,
	// which is how MevRewardInWei identifies the payment
	builderKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	builderAddress := crypto.PubkeyToAddress(builderKey.PublicKey)

	toAddress := common.HexToAddress(to)
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(MainnetChainId))
	tx, err := types.SignNewTx(builderKey, signer, &types.DynamicFeeTx{
		ChainID:   new(big.Int).SetUint64(MainnetChainId),
		Nonce:     0,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(0),
		Gas:       21000,
		To:        &toAddress,
		Value:     reward,
	})
	require.NoError(t, err)

	rawTx, err := tx.MarshalBinary()
	require.NoError(t, err)

	var feeRecipient bellatrix.ExecutionAddress
	copy(feeRecipient[:], builderAddress.Bytes())

	withdrawalCredentials := make([]byte, 32)
	withdrawalCredentials[0] = 1
	copy(withdrawalCredentials[12:], common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())

	fullBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           phase0.Slot(slot),
		ValidatorIndex: validatorIndex,
	}, &v1.Validator{
		Index: validatorIndex,
		Validator: &phase0.Validator{
			WithdrawalCredentials: withdrawalCredentials,
		},
	}, MainnetChainId)

	fullBlock.SetConsensusBlock(&spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionBellatrix,
		Bellatrix: &bellatrix.SignedBeaconBlock{
			Message: &bellatrix.BeaconBlock{
				Slot:          phase0.Slot(slot),
				ProposerIndex: validatorIndex,
				Body: &bellatrix.BeaconBlockBody{
					ExecutionPayload: &bellatrix.ExecutionPayload{
						FeeRecipient: feeRecipient,
						BlockNumber:  blockNumber,
						Transactions: []bellatrix.Transaction{rawTx},
					},
				},
			},
		},
	})
	fullBlock.SetEvents(&Events{})

	return fullBlock
}

func etherReceivedEvent(amount *big.Int) *contract.ContractEtherReceived {
	return &contract.ContractEtherReceived{
		Sender:         common.HexToAddress("0x2222222222222222222222222222222222222222"),
		DonationAmount: amount,
	}
}

func subscribeEvent(collateral *big.Int) *contract.ContractSubscribeValidator {
	return &contract.ContractSubscribeValidator{
		Sender:                 common.HexToAddress("0x3333333333333333333333333333333333333333"),
		SubscriptionCollateral: collateral,
		ValidatorID:            123,
	}
}

func claimRewardsEvent(claimed *big.Int) *contract.ContractClaimRewards {
	return &contract.ContractClaimRewards{
		WithdrawalAddress: common.HexToAddress("0x4444444444444444444444444444444444444444"),
		RewardAddress:     common.HexToAddress("0x4444444444444444444444444444444444444444"),
		ClaimableBalance:  claimed,
	}
}

// The balance math is the whole verdict, so pin down every combination of
// flows that can move the pool balance inside a block
func Test_UnexplainedPoolInflow(t *testing.T) {
	reward := big.NewInt(1000)

	tests := []struct {
		name     string
		delta    *big.Int
		events   *Events
		expected *big.Int
	}{
		{
			name:     "nothing happened",
			delta:    big.NewInt(0),
			events:   &Events{},
			expected: big.NewInt(0),
		},
		{
			name:     "forced transfer only, no events explain it",
			delta:    reward,
			events:   &Events{},
			expected: reward,
		},
		{
			name:  "a donation in the same block is explained away",
			delta: big.NewInt(1500),
			events: &Events{
				EtherReceived: []*contract.ContractEtherReceived{etherReceivedEvent(big.NewInt(500))},
			},
			expected: reward,
		},
		{
			name:  "subscription collateral in the same block is explained away",
			delta: big.NewInt(1300),
			events: &Events{
				SubscribeValidator: []*contract.ContractSubscribeValidator{subscribeEvent(big.NewInt(300))},
			},
			expected: reward,
		},
		{
			name:  "a claim in the same block is an outflow, so it is added back",
			delta: big.NewInt(200),
			events: &Events{
				ClaimRewards: []*contract.ContractClaimRewards{claimRewardsEvent(big.NewInt(800))},
			},
			expected: reward,
		},
		{
			name:  "everything at once",
			delta: big.NewInt(1000),
			events: &Events{
				EtherReceived:      []*contract.ContractEtherReceived{etherReceivedEvent(big.NewInt(500))},
				SubscribeValidator: []*contract.ContractSubscribeValidator{subscribeEvent(big.NewInt(300))},
				ClaimRewards:       []*contract.ContractClaimRewards{claimRewardsEvent(big.NewInt(800))},
			},
			expected: reward,
		},
		{
			name:  "a claim larger than the inflows makes the delta negative",
			delta: big.NewInt(-800),
			events: &Events{
				ClaimRewards: []*contract.ContractClaimRewards{claimRewardsEvent(big.NewInt(800))},
			},
			expected: big.NewInt(0),
		},
		{
			name:  "every event accounted for leaves nothing unexplained",
			delta: big.NewInt(500),
			events: &Events{
				EtherReceived: []*contract.ContractEtherReceived{etherReceivedEvent(big.NewInt(500))},
			},
			expected: big.NewInt(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unexplainedPoolInflow(tt.delta, tt.events)
			require.Zero(t, got.Cmp(tt.expected), "got %s, want %s", got, tt.expected)
		})
	}
}

// A payment proven to have reached the pool must be paid to the validator
// exactly like a direct payment would
func Test_ForcedMevPayment_Delivered(t *testing.T) {
	reward := big.NewInt(125947079586393390)
	validatorIndex := phase0.ValidatorIndex(2245785)

	fullBlock := buildMevBlock(t, 15000000, 25568643, validatorIndex, titanForwarder, reward)

	// Before verification the payment looks like it went somewhere else
	mevReward, isMev, recipient := fullBlock.MevRewardInWei()
	require.True(t, isMev)
	require.Equal(t, reward, mevReward)
	require.Equal(t, titanForwarder, common.HexToAddress(recipient).Hex())

	fullBlock.SetForcedMevPayment(&ForcedMevPayment{
		Delivered:   true,
		AmountWei:   reward,
		Payer:       titanForwarder,
		TxHash:      "0x4dfe43fdab0f06b1725123c6ddf72c029409b74980e474ec160539ed98e622db",
		BlockNumber: 25568643,
		BlockHash:   "0x1234567890123456789012345678901234567890123456789012345678901234",
	}, testPoolAddress)

	oracle := NewOracle(&Config{})
	oracle.addSubscription(uint64(validatorIndex), "0x1111111111111111111111111111111111111111", "0x")

	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, OkPoolProposal, summary.BlockType)
	require.Equal(t, MevBlock, summary.RewardType)
	require.Equal(t, reward, summary.Reward)

	// The reward is an asset of the pool, not a donation
	require.Len(t, fullBlock.GetDonations(testPoolAddress), 0)

	// The evidence must point at the real payment, not at zeroed out fields
	// like the hardcoded exceptions do
	require.Len(t, fullBlock.Events.EtherReceived, 1)
	event := fullBlock.Events.EtherReceived[0]
	require.Equal(t, reward, event.DonationAmount)
	require.Equal(t, uint64(25568643), event.Raw.BlockNumber)
	require.Equal(t,
		"0x4dfe43fdab0f06b1725123c6ddf72c029409b74980e474ec160539ed98e622db",
		event.Raw.TxHash.String())
	require.Equal(t, titanForwarder, event.Sender.Hex())
}

// A payment proven NOT to have reached the pool must ban the validator, exactly
// as today. Verification never softens the wrong fee policy
func Test_ForcedMevPayment_NotDelivered(t *testing.T) {
	reward := big.NewInt(125947079586393390)
	validatorIndex := phase0.ValidatorIndex(2245785)

	fullBlock := buildMevBlock(t, 15000000, 25568643, validatorIndex, titanForwarder, reward)

	fullBlock.SetForcedMevPayment(&ForcedMevPayment{
		Delivered:   false,
		AmountWei:   reward,
		Payer:       titanForwarder,
		BlockNumber: 25568643,
	}, testPoolAddress)

	oracle := NewOracle(&Config{})
	oracle.addSubscription(uint64(validatorIndex), "0x1111111111111111111111111111111111111111", "0x")

	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, WrongFeeRecipient, summary.BlockType)

	// No asset was created, since no ETH arrived
	require.Len(t, fullBlock.Events.EtherReceived, 0)
}

// Without verification the block keeps the old behaviour, so a node that never
// reaches the check cannot silently pay out
func Test_ForcedMevPayment_NotVerifiedKeepsWrongFee(t *testing.T) {
	reward := big.NewInt(125947079586393390)
	validatorIndex := phase0.ValidatorIndex(2245785)

	fullBlock := buildMevBlock(t, 15000000, 25568643, validatorIndex, titanForwarder, reward)
	require.Nil(t, fullBlock.ForcedMevPayment)

	oracle := NewOracle(&Config{})
	oracle.addSubscription(uint64(validatorIndex), "0x1111111111111111111111111111111111111111", "0x")

	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, WrongFeeRecipient, summary.BlockType)
}

// A MEV payment sent straight to the pool must not be affected at all
func Test_ForcedMevPayment_DirectPaymentUnaffected(t *testing.T) {
	reward := big.NewInt(125947079586393390)
	validatorIndex := phase0.ValidatorIndex(2245785)

	fullBlock := buildMevBlock(t, 15000000, 25568643, validatorIndex, testPoolAddress, reward)
	require.Nil(t, fullBlock.ForcedMevPayment)

	oracle := NewOracle(&Config{})
	oracle.addSubscription(uint64(validatorIndex), "0x1111111111111111111111111111111111111111", "0x")

	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, OkPoolProposal, summary.BlockType)
	require.Equal(t, reward, summary.Reward)
}

// A donation landing in the same block as a forced payment must stay a
// donation, and the reward must stay a reward
func Test_ForcedMevPayment_DonationInSameBlock(t *testing.T) {
	reward := big.NewInt(125947079586393390)
	donation := big.NewInt(7000000000000000)
	validatorIndex := phase0.ValidatorIndex(2245785)

	fullBlock := buildMevBlock(t, 15000000, 25568643, validatorIndex, titanForwarder, reward)
	fullBlock.Events.EtherReceived = append(fullBlock.Events.EtherReceived, etherReceivedEvent(donation))

	fullBlock.SetForcedMevPayment(&ForcedMevPayment{
		Delivered:   true,
		AmountWei:   reward,
		Payer:       titanForwarder,
		BlockNumber: 25568643,
	}, testPoolAddress)

	donations := fullBlock.GetDonations(testPoolAddress)
	require.Len(t, donations, 1)
	require.Equal(t, donation, donations[0].DonationAmount)

	oracle := NewOracle(&Config{})
	oracle.addSubscription(uint64(validatorIndex), "0x1111111111111111111111111111111111111111", "0x")

	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, OkPoolProposal, summary.BlockType)
	require.Equal(t, reward, summary.Reward)
}

// A forced payment from a validator the pool does not know yet must auto
// subscribe it and pay it, exactly as a direct payment to the pool does.
// Gating detection on an existing subscription left the ETH in the pool with no
// liability against it, which broke reconciliation by exactly that amount.
func Test_ForcedMevPayment_AutoSubscribesUnknownValidator(t *testing.T) {
	reward := big.NewInt(8645848622424368)
	validatorIndex := phase0.ValidatorIndex(999999)

	fullBlock := buildMevBlock(t, 15000000, 25739733, validatorIndex, titanForwarder, reward)

	fullBlock.SetForcedMevPayment(&ForcedMevPayment{
		Delivered:   true,
		AmountWei:   reward,
		Payer:       titanForwarder,
		BlockNumber: 25739733,
	}, testPoolAddress)

	// Deliberately no addSubscription: this validator is unknown to the pool
	oracle := NewOracle(&Config{})
	require.False(t, oracle.isSubscribed(uint64(validatorIndex)))

	// OkPoolProposal is what routes the block to handleCorrectBlockProposal,
	// which calls addSubscription and allocates the reward. Classifying it as
	// WrongFeeRecipient instead is what left the ETH unallocated.
	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, OkPoolProposal, summary.BlockType)
	require.Equal(t, MevBlock, summary.RewardType)
	require.Equal(t, reward, summary.Reward)
}

// A banned validator that pays through a forced transfer must also be seen,
// otherwise the ETH arrives with nothing to allocate it to
func Test_ForcedMevPayment_BannedValidatorStillDetected(t *testing.T) {
	reward := big.NewInt(5127862812305962)
	validatorIndex := phase0.ValidatorIndex(888888)

	fullBlock := buildMevBlock(t, 15000000, 25740093, validatorIndex, titanForwarder, reward)
	fullBlock.SetForcedMevPayment(&ForcedMevPayment{
		Delivered:   true,
		AmountWei:   reward,
		Payer:       titanForwarder,
		BlockNumber: 25740093,
	}, testPoolAddress)

	oracle := NewOracle(&Config{})
	oracle.addSubscription(uint64(validatorIndex), "0x1111111111111111111111111111111111111111", "0x")
	oracle.state.Validators[uint64(validatorIndex)].ValidatorStatus = Banned
	require.False(t, oracle.isSubscribed(uint64(validatorIndex)))

	summary := fullBlock.SummarizedBlock(oracle, testPoolAddress)
	require.Equal(t, OkPoolProposal, summary.BlockType)
	require.Equal(t, reward, summary.Reward)
}

// The detection must not apply before the slot it was deployed at, otherwise a
// resync from the pool deployment would rewrite already published roots
func Test_ForcedMevPayment_ActivationSlot(t *testing.T) {
	activation := ForcedPaymentActivationSlot[MainnetChainId]
	reward := big.NewInt(125947079586393390)

	tests := []struct {
		name   string
		slot   uint64
		active bool
	}{
		{"long before activation", activation - 100000, false},
		{"one slot before activation", activation - 1, false},
		{"exactly at activation", activation, true},
		{"one slot after activation", activation + 1, true},
		{"long after activation", activation + 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullBlock := buildMevBlock(t, tt.slot, 25568643, phase0.ValidatorIndex(2245785),
				titanForwarder, reward)
			require.Equal(t, tt.active, fullBlock.IsForcedPaymentDetectionActive())
		})
	}
}

// Chains with no recorded activation have no published history to preserve
func Test_ForcedMevPayment_ActivationUnknownChain(t *testing.T) {
	_, found := ForcedPaymentActivationSlot[HoleskyChainId]
	require.False(t, found, "test assumes holesky has no activation slot")

	fullBlock := buildMevBlock(t, 1, 100, phase0.ValidatorIndex(1), titanForwarder, big.NewInt(1))
	fullBlock.ChainId = HoleskyChainId
	require.True(t, fullBlock.IsForcedPaymentDetectionActive())
}

// The activation slot must not sit after the range the fix has to repair,
// otherwise the payments that halted the oracle would stay unallocated
func Test_ForcedMevPayment_ActivationCoversKnownExceptions(t *testing.T) {
	activation := ForcedPaymentActivationSlot[MainnetChainId]

	// Exceptions 4 and 5 are inside the window the oracle must replay, so the
	// detection has to be live by then even though they short circuit
	require.Less(t, activation, ExceptionSlotMainnet4)
	require.Less(t, activation, ExceptionSlotMainnet5)

	// Exceptions 1 to 3 predate it and stay handled by the exception table
	require.Greater(t, activation, ExceptionSlotMainnet1)
	require.Greater(t, activation, ExceptionSlotMainnet2)
	require.Greater(t, activation, ExceptionSlotMainnet3)
}

// The five hardcoded exceptions must keep producing the exact same state, since
// changing them would change the historical oracle root
func Test_ForcedMevPayment_ExceptionsStillShortCircuit(t *testing.T) {
	exceptions := map[uint64]string{
		ExceptionSlotMainnet1: "177043568463114308",
		ExceptionSlotMainnet2: "9557629473261564",
		ExceptionSlotMainnet3: "125947079586393390",
		ExceptionSlotMainnet4: "40752704699988562",
		ExceptionSlotMainnet5: "15227018045697363",
	}

	for slot, rewardWei := range exceptions {
		fullBlock := buildMevBlock(t, slot, 25568643, phase0.ValidatorIndex(2245785),
			titanForwarder, big.NewInt(1))

		expected, ok := new(big.Int).SetString(rewardWei, 10)
		require.True(t, ok)

		// The exception table wins before any verification happens, and the
		// recipient it reports is the pool itself
		reward, isMev, recipient := fullBlock.MevRewardInWei()
		require.True(t, isMev)
		require.Equal(t, expected, reward)
		require.Equal(t, testPoolAddress, recipient)
		require.Nil(t, fullBlock.ForcedMevPayment)
	}
}
