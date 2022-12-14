package oracle

import (
	"math/big"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

const (
	VanilaBlock int = 0
	MevBlock        = 1
)

// TODO: perhaps block_bellatrix
// ant instedo of signedSinedBeaconBlock
// bellatrix.SignedBeaconBlock

// and then one other field for capella.SignedBeaconBlock

// extend with custom methods
type BellatrixBlock struct {
	bellatrix.SignedBeaconBlock
}

func (b *BellatrixBlock) MevRewardInWei(poolAddress string) (*big.Int, int, error) {
	totalMevReward := big.NewInt(0)
	// this should be just 1, but just in case
	numTxs := 0
	for _, rawTx := range b.Message.Body.ExecutionPayload.Transactions {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("todo")
		}
		// This seems to happen in smart contrat deployments
		if msg.To() == nil {
			continue
		}
		// Note that its usually the last tx but we check all just in case
		if strings.ToLower(poolAddress) == strings.ToLower(msg.To().String()) {
			totalMevReward.Add(totalMevReward, msg.Value())
			log.WithFields(log.Fields{
				"Slot":         b.Message.Slot,
				"Block":        b.Message.Body.ExecutionPayload.BlockNumber,
				"ValIndex":     b.Message.ProposerIndex,
				"FeeRecipient": b.FeeRecipient(),
				"PoolAddress":  poolAddress,
				"To":           msg.To().String(),
				"MevReward":    msg.Value(),
				"TxHash":       tx.Hash().String(),
			}).Info("MEV transaction detected to pool")
			numTxs++
		}
	}
	return totalMevReward, numTxs, nil

}

// this call is expensive if its a vanila block. the tip sent to the fee recipient
// has to be calculated by iterating all txs and adding the tips. this requires
// to get every single tx receipt. note that this call is not done in MEV blocks.
func (b *BellatrixBlock) GetSentRewardAndType(
	poolAddress string,
	fetcher Fetcher) (*big.Int, bool, int, error) {

	var reward *big.Int = big.NewInt(0)
	err := *new(error)
	var numTxs int = 0
	var txType int = -1
	var wasRewardSent bool = false

	if b.FeeRecipient() == poolAddress {
		// vanila block, we get the tip from the block
		blockNumber := new(big.Int).SetUint64(b.Message.Body.ExecutionPayload.BlockNumber)
		header, receipts, err := fetcher.GetExecHeaderAndReceipts(blockNumber, b.Message.Body.ExecutionPayload.Transactions)
		if err != nil {
			log.Fatal(err)
		}

		reward, err = b.GetProposerTip(header, receipts)
		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"Slot":         b.Message.Slot,
			"Block":        b.Message.Body.ExecutionPayload.BlockNumber,
			"ValIndex":     b.Message.ProposerIndex,
			"FeeRecipient": b.FeeRecipient(),
			"PoolAddress":  poolAddress,
			"VanilaReward": reward.String(),
		}).Info("Vanila reward found in block")
		txType = VanilaBlock
		wasRewardSent = true
	}

	// possible mev block
	var mevReward *big.Int = big.NewInt(0)
	mevReward, numTxs, err = b.MevRewardInWei(poolAddress)
	if err != nil {
		log.Fatal(err)
	}
	// sanity check. I assume we can't have both mev and vanila rewards. if we do, fail and revisit
	if (mevReward.Cmp(big.NewInt(0)) == 1) && (reward.Cmp(big.NewInt(0)) == 1) {
		log.Fatal("Both VanilaReward and MevReward are !=0. mevReward: ", mevReward, "vanilaReward: ", reward)
	}
	reward = mevReward
	if numTxs == 0 {
		// no mev reward
		log.WithFields(log.Fields{
			"Slot":         b.Message.Slot,
			"Block":        b.Message.Body.ExecutionPayload.BlockNumber,
			"ValIndex":     b.Message.ProposerIndex,
			"FeeRecipient": b.FeeRecipient(),
			"PoolAddress":  poolAddress,
		}).Info("No MEV reward found in block")
		wasRewardSent = false
	} else if numTxs == 1 {
		// mev block
		log.WithFields(log.Fields{
			"Slot":         b.Message.Slot,
			"Block":        b.Message.Body.ExecutionPayload.BlockNumber,
			"ValIndex":     b.Message.ProposerIndex,
			"FeeRecipient": b.FeeRecipient(),
			"PoolAddress":  poolAddress,
			"MevReward":    reward.String(),
		}).Info("MEV reward found in block")
		wasRewardSent = true
		txType = MevBlock
	} else {
		log.Fatal("more than 1 mev tx in a block is not expected. num: ", numTxs)
	}
	return reward, wasRewardSent, txType, nil
}

// get proposer tip went to the fee recepient. not related to MEV
// note that the tip is not included in the block and has to be reconstructed
// iterating all transactions and checking the tip
// returns proposer tip and feeRecipient
func (b *BellatrixBlock) GetProposerTip(blockHeader *types.Header, txReceipts []*types.Receipt) (*big.Int, error) {
	// little-endian to big-endian
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = b.Message.Body.ExecutionPayload.BaseFeePerGas[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)
	if len(b.Message.Body.ExecutionPayload.Transactions) != len(txReceipts) {
		log.Fatal("txs and receipts not the same length")
	}
	for i, rawTx := range b.Message.Body.ExecutionPayload.Transactions {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal()
		}
		if tx.Hash() != txReceipts[i].TxHash {
			log.Fatal("tx and receipt not the same tx")
		}

		tipFee := new(big.Int)
		gasPrice := tx.GasPrice()
		gasUsed := big.NewInt(int64(txReceipts[i].GasUsed))

		switch tx.Type() {
		case 0:
			tipFee.Mul(gasPrice, gasUsed)
		case 1:
			tipFee.Mul(gasPrice, gasUsed)
		case 2:
			// Sum gastipcap and basefee or saturate to gasfeecap
			usedGasPrice := SumAndSaturate(msg.GasTipCap(), blockHeader.BaseFee, msg.GasFeeCap())
			tipFee = new(big.Int).Mul(usedGasPrice, gasUsed)
		default:
			log.Fatal("unknown tx type")
		}
		tips = tips.Add(tips, tipFee)
	}
	burnt := new(big.Int).Mul(big.NewInt(int64(b.Message.Body.ExecutionPayload.GasUsed)), baseFeePerGas)
	proposerReward := new(big.Int).Sub(tips, burnt)

	return proposerReward, nil
}

// Detects "Transfer" transactions to the poolAddress and adds them into a single number
// Note that ERC20 transfers are not supported, not detected by this function.
func (b *BellatrixBlock) DonatedAmountInWei(poolAddress string) (*big.Int, error) {
	donatedAmountInBlock := big.NewInt(0)
	numTxs := 0
	for _, rawTx := range b.Message.Body.ExecutionPayload.Transactions {
		_, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("todo")
		}
		// If a transaction was sent to the pool
		// And the sender is not the fee recipient (exclude MEV transactions)
		// Note that msg.To() is nil for contract creation transactions
		if msg.To() == nil {
			continue
		}
		if strings.ToLower(msg.To().String()) == strings.ToLower(poolAddress) &&
			(strings.ToLower(msg.From().String()) != strings.ToLower(b.FeeRecipient())) {

			donatedAmountInBlock.Add(donatedAmountInBlock, msg.Value())
			log.Info("donated. todo: log blog: ", msg.Value())
			numTxs++
		}
	}
	return donatedAmountInBlock, nil
}

func (b *BellatrixBlock) FeeRecipient() string {
	return b.Message.Body.ExecutionPayload.FeeRecipient.String()
}
