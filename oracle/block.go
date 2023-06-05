package oracle

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/pkg/errors"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

type FullBlock struct {
	*spec.VersionedSignedBeaconBlock
	header   *types.Header
	receipts []*types.Receipt
}

func NewFullBlock(
	versionedBlock *spec.VersionedSignedBeaconBlock,
	header *types.Header,
	receipts []*types.Receipt) *FullBlock {

	// Create the type first to use its methods
	fb := &FullBlock{
		VersionedSignedBeaconBlock: versionedBlock,
		header:                     header,
		receipts:                   receipts,
	}

	// Run some sanity checks to ensure the receipts and header match the block
	if header != nil {
		if fb.GetBlockNumberBigInt().Uint64() != fb.header.Number.Uint64() {
			log.Fatal("Block number mismatch with header: ",
				fb.GetBlockNumberBigInt().Uint64(), " vs ", fb.header.Number.Uint64())
		}
	}
	if receipts != nil && len(receipts) > 0 {
		if fb.GetBlockNumberBigInt().Uint64() != receipts[0].BlockNumber.Uint64() {
			log.Fatal("Block number mismatch with receipts: ",
				fb.GetBlockNumberBigInt().Uint64(), " vs ", receipts[0].BlockNumber.Uint64())
		}
	}

	return fb
}

// Note that the MEV rewards usually goes in the last tx of each block
// from the protocol fee recipient to the validator mev fee recipient
// we check all just in case.

// Example: this block https://prater.beaconcha.in/slot/5307417
// Contains a mev reward of 0.53166 Ether
// With the MEV Reward Recipient (mrr): 0x4D496CcC28058B1D74B7a19541663E21154f9c84
// And a protocol fee recipient of (pfr): 0xb64a30399f7F6b0C154c2E7Af0a3ec7B0A5b131a
// Note how the last tx of the block contains a tx pfr->mrr of 0.53166 Ether
// Returns if a mev reward was present, its amount and the mev reward recipient
func (b *FullBlock) MevRewardInWei() (*big.Int, bool, string) {
	// We detect a MEV reward as the last tx of the block. If this last tx has a transfer
	// from the protocol fee recipient to any address, we consider that a MEV reward.
	txs := b.GetBlockTransactions()
	lastTx := txs[len(txs)-1]

	tx, msg, err := DecodeTx(lastTx)
	if err != nil {
		log.Fatal("could not decode tx: ", err)
	}

	// Its nil when its a smart contract deployment. No mev reward
	if msg.To() == nil {
		return big.NewInt(0), false, ""
	}

	// Last tx is from the protocol fee recipient to an address.
	if Equals(b.GetFeeRecipient(), msg.From().String()) {
		// TODO: Better to not log here
		log.WithFields(log.Fields{
			"Slot":            b.GetSlot(),
			"Block":           b.GetBlockNumber(),
			"ValIndex":        b.GetProposerIndex(),
			"FeeRecipient":    b.GetFeeRecipient(),
			"MEVFeeRecipient": msg.To().String(),
			"Reward":          msg.Value(),
			"TxHash":          tx.Hash().String(),
			"Type":            "MevBlock",
		}).Info("[Reward]")
		return msg.Value(), true, strings.ToLower(tx.To().String())
	}

	return big.NewInt(0), false, ""
}

// This call is expensive if its a vanila block. The tip sent to the fee recipient
// has to be calculated by iterating all txs and adding the tips as per EIP1559.
// This requires to get every single tx receipt from the block, hence needing
// the onchain to get the receipts from the consensus layer.s
// Note that that this call is cheaper when the block is a MEV block, as there is no
// need to reconstruct the tip from the txs.
func (b *FullBlock) GetRewardAndRecipient() (*big.Int, string, RewardType, error) {

	var rewardAmount *big.Int = big.NewInt(0)
	var rewardRecipient string = ""
	var vanilaReward *big.Int = big.NewInt(0)
	var txType RewardType = UnknownRewardType

	// start checking if its a mev block, since its cheaper to check
	mevReward, mevRewardPresent, mevRecipient := b.MevRewardInWei()

	if mevRewardPresent {
		txType = MevBlock
		rewardAmount = mevReward
		rewardRecipient = mevRecipient
	}

	// if its not a mev block, check if its a vanila block. note that this requires
	// header and receipts fields to be populated, which is very expensive in terms
	// of queries to the execution layer.
	if !mevRewardPresent {
		vanilaReward, err := b.GetProposerTip()
		if err != nil {
			log.Fatal("could not get proposer tip: ", err)
		}
		// TODO: Better to not log here
		log.WithFields(log.Fields{
			"Slot":         b.GetSlot(),
			"Block":        b.GetBlockNumber(),
			"ValIndex":     b.GetProposerIndex(),
			"Reward":       vanilaReward.String(),
			"Type":         "VanilaBlock",
			"FeeRecipient": b.GetFeeRecipient(),
		}).Info("[Reward]")

		txType = VanilaBlock
		rewardAmount = vanilaReward
		rewardRecipient = b.GetFeeRecipient()
	}

	// sanity check. I assume we can't have both mev and vanila rewards. if we do, fail and revisit
	// this happened in goerli, but a very weird scenario that should never happen in mainnet.
	// can only occur if the pool address is the same as the fee recipient, aka our pool address
	// is the same as the fee recipient sent by the builders.
	if (mevReward.Cmp(big.NewInt(0)) == 1) && (vanilaReward.Cmp(big.NewInt(0)) == 1) {
		log.Warn("Both VanilaReward and MevReward are !=0. ",
			"mevReward: ", mevReward, "vanilaReward: ", vanilaReward, ". This should never happen in mainnet.")
	}

	return rewardAmount, rewardRecipient, txType, nil
}

// Get proposer the proposer tip that went to the fee recepient.
// Note that calculating the tip requires iterating all txs and getting the
// tip by reconstructing it as specified in EIP1559.
func (b *FullBlock) GetProposerTip() (*big.Int, error) {

	// Ensure non nil
	if b.receipts == nil {
		return nil, errors.New("receipts of full block are nil, cant calculate tip")
	}

	if b.header == nil {
		return nil, errors.New("header of full block are nil, cant calculate tip")
	}

	// Ensure tx and their receipts have the same size
	if len(b.GetBlockTransactions()) != len(b.receipts) {
		return nil, errors.New(fmt.Sprintf("txs and receipts not the same length. txs: %d, receipts: %d",
			len(b.GetBlockTransactions()), len(b.receipts)))
	}

	// little-endian to big-endian
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = b.GetBaseFeePerGas()[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)

	for i, rawTx := range b.GetBlockTransactions() {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			return nil, errors.Wrap(err, "could not decode tx")
		}
		if tx.Hash() != b.receipts[i].TxHash {
			return nil, errors.Wrap(err, "tx hash does not match receipt hash")
		}

		tipFee := new(big.Int)
		gasPrice := tx.GasPrice()
		gasUsed := big.NewInt(int64(b.receipts[i].GasUsed))

		switch tx.Type() {
		case 0:
			tipFee.Mul(gasPrice, gasUsed)
		case 1:
			tipFee.Mul(gasPrice, gasUsed)
		case 2:
			// Sum gastipcap and basefee or saturate to gasfeecap
			usedGasPrice := SumAndSaturate(msg.GasTipCap(), b.header.BaseFee, msg.GasFeeCap())
			tipFee = new(big.Int).Mul(usedGasPrice, gasUsed)
		default:
			return nil, errors.New(fmt.Sprintf("unknown tx type: %d", tx.Type()))
		}
		tips = tips.Add(tips, tipFee)
	}
	burnt := new(big.Int).Mul(big.NewInt(int64(b.GetGasUsed())), baseFeePerGas)
	proposerReward := new(big.Int).Sub(tips, burnt)

	return proposerReward, nil
}

// Returns the fee recipient of the block, depending on the fork version
func (b *FullBlock) GetFeeRecipient() string {
	var feeRecipient string

	if b.Altair != nil {
		log.Fatal("Altair block has no fee recipient")
	} else if b.Bellatrix != nil {
		feeRecipient = b.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else if b.Capella != nil {
		feeRecipient = b.Capella.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else {
		log.Fatal("Block was empty, cant get fee recipient")
	}
	return feeRecipient
}

// Returns the transactions of the block depending on the fork version
func (b *FullBlock) GetBlockTransactions() []bellatrix.Transaction {

	var transactions []bellatrix.Transaction
	if b.Altair != nil {
		log.Fatal("Altair block has no transactions in the beacon block")
	} else if b.Bellatrix != nil {
		transactions = b.Bellatrix.Message.Body.ExecutionPayload.Transactions
	} else if b.Capella != nil {
		transactions = b.Capella.Message.Body.ExecutionPayload.Transactions
	} else {
		log.Fatal("Block was empty, cant get transactions")
	}
	return transactions
}

// Returns the block number depending on the fork version (as uint64)
func (b *FullBlock) GetBlockNumber() uint64 {
	var blockNumber uint64

	if b.Altair != nil {
		log.Fatal("Altair block has no block number")
	} else if b.Bellatrix != nil {
		blockNumber = b.Bellatrix.Message.Body.ExecutionPayload.BlockNumber
	} else if b.Capella != nil {
		blockNumber = b.Capella.Message.Body.ExecutionPayload.BlockNumber
	} else {
		log.Fatal("Block was empty, cant get block number")
	}
	return blockNumber
}

// Returns the block number depending on the fork version (as big.Int)
func (b *FullBlock) GetBlockNumberBigInt() *big.Int {
	return new(big.Int).SetUint64(b.GetBlockNumber())
}

// Returns the slot depending on the fork version
func (b *FullBlock) GetSlot() phase0.Slot {
	var slot phase0.Slot

	if b.Altair != nil {
		slot = b.Altair.Message.Slot
	} else if b.Bellatrix != nil {
		slot = b.Bellatrix.Message.Slot
	} else if b.Capella != nil {
		slot = b.Capella.Message.Slot
	} else {
		log.Fatal("Block was empty, cant get slot")
	}
	return slot
}

func (b *FullBlock) GetSlotUint64() uint64 {
	return uint64(b.GetSlot())
}

// Returns the proposed index depending on the fork version
func (b *FullBlock) GetProposerIndex() phase0.ValidatorIndex {
	var proposerIndex phase0.ValidatorIndex

	if b.Altair != nil {
		proposerIndex = b.Altair.Message.ProposerIndex
	} else if b.Bellatrix != nil {
		proposerIndex = b.Bellatrix.Message.ProposerIndex
	} else if b.Capella != nil {
		proposerIndex = b.Capella.Message.ProposerIndex
	} else {
		log.Fatal("Block was empty, cant get proposer index")
	}
	return proposerIndex
}

func (b *FullBlock) GetProposerIndexUint64() uint64 {
	return uint64(b.GetProposerIndex())
}

// Returns the gas used depending on the fork version
func (b *FullBlock) GetGasUsed() uint64 {
	var gasUsed uint64

	if b.Altair != nil {
		log.Fatal("Altair block has no gas used")
	} else if b.Bellatrix != nil {
		gasUsed = b.Bellatrix.Message.Body.ExecutionPayload.GasUsed
	} else if b.Capella != nil {
		gasUsed = b.Capella.Message.Body.ExecutionPayload.GasUsed
	} else {
		log.Fatal("Block was empty, cant get gas used")
	}
	return gasUsed
}

// Returns the base fee per gas depending on the fork version
func (b *FullBlock) GetBaseFeePerGas() [32]byte {
	var baseFeePerGas [32]byte

	if b.Altair != nil {
		log.Fatal("Altair block has no base fee per gas")
	} else if b.Bellatrix != nil {
		baseFeePerGas = b.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas
	} else if b.Capella != nil {
		baseFeePerGas = b.Capella.Message.Body.ExecutionPayload.BaseFeePerGas
	} else {
		log.Fatal("Block was empty, cant get base fee per gas")
	}
	return baseFeePerGas
}
