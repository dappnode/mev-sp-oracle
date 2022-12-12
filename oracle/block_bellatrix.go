package oracle

import (
	"math/big"
	"strings"

	"github.com/attestantio/go-eth2-client/spec"
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
type VersionedSignedBeaconBlock struct {
	spec.VersionedSignedBeaconBlock
}

func (b *VersionedSignedBeaconBlock) MevRewardInWei(poolAddress string) (*big.Int, int, error) {
	totalMevReward := big.NewInt(0)
	// this should be just 1, but just in case
	numTxs := 0
	for _, rawTx := range b.Bellatrix.Message.Body.ExecutionPayload.Transactions {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("todo")
		}
		// This seems to happen in smart contrat deployments
		if msg.To() == nil {
			//continue
			// TODO: check. smart contract deployment have this field to nil
		}
		// Note that its usually the last tx but we check all just in case
		if strings.ToLower(poolAddress) == strings.ToLower(msg.To().String()) {
			totalMevReward.Add(totalMevReward, msg.Value())
			log.WithFields(log.Fields{
				"Slot":         b.Bellatrix.Message.Slot,
				"Block":        b.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
				"ValIndex":     b.Bellatrix.Message.ProposerIndex,
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

// get proposer reward (vanila or mev block) and indicate which type of block it was
// and also feerecepient
// TODO: rename to GetRewardsSentToPool
// this can either be:
// -mev rewards. we check every single tx and check if they sent any amnount to the pool (differenciate from donations)
// -if the "normal" feerecipient was us, clculate fees and use that as a reward.

// this call is expensive if its a vanila block. the tip sent to the fee recipient
// has to be calculated by iterating all txs and adding the tips. this requires
// to get every single tx receipt. note that this call is not done in MEV blocks.
func (b *VersionedSignedBeaconBlock) GetSentRewardAndType(
	poolAddress string,
	fetcher Fetcher) (*big.Int, bool, int, error) {

	var reward *big.Int = big.NewInt(0)
	err := *new(error)
	var numTxs int = 0
	var txType int = -1
	var wasRewardSent bool = false

	if b.FeeRecipient() == poolAddress {
		// vanila block, we get the tip from the block
		blockNumber := new(big.Int).SetUint64(b.Bellatrix.Message.Body.ExecutionPayload.BlockNumber)
		header, receipts, err := fetcher.GetExecHeaderAndReceipts(blockNumber, b.Bellatrix.Message.Body.ExecutionPayload.Transactions)
		if err != nil {
			log.Fatal(err)
		}

		reward, err = b.GetProposerTip(header, receipts)
		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"Slot":         b.Bellatrix.Message.Slot,
			"Block":        b.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
			"ValIndex":     b.Bellatrix.Message.ProposerIndex,
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
			"Slot":         b.Bellatrix.Message.Slot,
			"Block":        b.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
			"ValIndex":     b.Bellatrix.Message.ProposerIndex,
			"FeeRecipient": b.FeeRecipient(),
			"PoolAddress":  poolAddress,
		}).Info("No MEV reward found in block")
		wasRewardSent = false
	} else if numTxs == 1 {
		// mev block
		log.WithFields(log.Fields{
			"Slot":         b.Bellatrix.Message.Slot,
			"Block":        b.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
			"ValIndex":     b.Bellatrix.Message.ProposerIndex,
			"FeeRecipient": b.FeeRecipient(),
			"PoolAddress":  poolAddress,
			"MevReward":    reward.String(),
		}).Info("MEV reward found in block")
		wasRewardSent = true
		txType = MevBlock
	} else {
		log.Fatal("more than 1 mev tx in a block is not expected. num: ", numTxs)
	}
	return reward, wasRewardSent, txType, nil // TODO: log.fatal so no need to return error
}

// get proposer tip went to the fee recepient. not related to MEV
// note that the tip is not included in the block and has to be reconstructed
// iterating all transactions and checking the tip
// returns proposer tip and feeRecipient
func (b *VersionedSignedBeaconBlock) GetProposerTip(blockHeader *types.Header, txReceipts []*types.Receipt) (*big.Int, error) {
	// little-endian to big-endian
	// TODO: to util function. LittleToBigEndianBigInt
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = b.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)
	if len(b.Bellatrix.Message.Body.ExecutionPayload.Transactions) != len(txReceipts) {
		log.Fatal("txs and receipts not the same length")
	}
	for i, rawTx := range b.Bellatrix.Message.Body.ExecutionPayload.Transactions {
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
			// TODO: move to function, saturate
			val1 := new(big.Int).Add(msg.GasTipCap(), blockHeader.BaseFee)
			usedGasPrice := new(big.Int) // TODO: better naming
			if val1.Cmp(msg.GasFeeCap()) >= 0 {
				usedGasPrice = msg.GasFeeCap()
			} else {
				usedGasPrice = val1
			}
			//realPrice := new(big.Int).Min(val1, big.NewInt(int64(receipt.GasUsed)))
			// TODO limit in baseFee?
			tipFee = new(big.Int).Mul(usedGasPrice, gasUsed)
		default:
			log.Fatal("unknown tx type")
		}
		//log.Info(i, " ", "tipFee:", tipFee)
		tips = tips.Add(tips, tipFee)

		// TODO: remove this?
		if strings.ToLower(msg.From().String()) == strings.ToLower(b.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String()) {
		}
	}
	burnt := new(big.Int).Mul(big.NewInt(int64(b.Bellatrix.Message.Body.ExecutionPayload.GasUsed)), baseFeePerGas)
	proposerReward := new(big.Int).Sub(tips, burnt)

	log.Info("txfees:", tips)
	log.Info("burndeotherway:", burnt)
	log.Info("proposer rewards: ", proposerReward)

	return proposerReward, nil

}

// Detects "Transfer" transactions to the poolAddress and adds them into a single number
// Note that ERC20 transfers are not supported, not detected by this function.
func (b *VersionedSignedBeaconBlock) DonatedAmountInWei(poolAddress string) (*big.Int, error) {
	donatedAmountInBlock := big.NewInt(0)
	numTxs := 0
	for _, rawTx := range b.Bellatrix.Message.Body.ExecutionPayload.Transactions {
		_, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("todo")
		}
		// If a transaction was sent to the pool
		// And the sender is not the fee recipient (exclude MEV transactions)
		// msg.To() is nil for contract creation transactions TODO:
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

// TODO: Unit test
func (b *VersionedSignedBeaconBlock) FeeRecipient() string {
	return b.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String()
}

// TODO: As it is it wont work for Capella fork

// Perhaps move this? not block methods per se

func DecodeTx(rawTx []byte) (*types.Transaction, *types.Message, error) {
	var tx types.Transaction
	err := tx.UnmarshalBinary(rawTx)
	if err != nil {
		return nil, nil, err
	}

	// Supports EIP-2930 and EIP-2718 and EIP-1559 and EIP-155 and legacy transactions.
	msg, err := tx.AsMessage(types.LatestSignerForChainID(tx.ChainId()), big.NewInt(0))
	if err != nil {
		return nil, nil, err
	}
	return &tx, &msg, err
}
