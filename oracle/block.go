package oracle

import (
	"errors"
	"math/big"
	"strings"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

// This type extends the existing VersionedSignedBeaconBlock with some useful
// custom methods to get relevant information from the block to be used by
// the oracle.
type VersionedSignedBeaconBlock struct {
	*spec.VersionedSignedBeaconBlock
}

func (b *VersionedSignedBeaconBlock) MevRewardInWei(poolAddress string) (*big.Int, int, error) {
	totalMevReward := big.NewInt(0)
	numTxs := 0

	// Iterate all txs in the block
	for _, rawTx := range b.GetBlockTransactions() {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			return nil, 0, errors.New("could not decode tx: " + err.Error())
		}
		// This seems to happen in smart contrat deployments
		if msg.To() == nil {
			continue
		}
		// Note that the MEV rewards usually goes in the last tx of each block
		// from the protocol fee recipient to the validator mev fee recipient
		// we check all just in case.

		// Example: this block https://prater.beaconcha.in/slot/5307417
		// Contains a mev reward of 0.53166 Ether
		// With the MEV Reward Recipient (mrr): 0x4D496CcC28058B1D74B7a19541663E21154f9c84
		// And a protocol fee recipient of (pfr): 0xb64a30399f7F6b0C154c2E7Af0a3ec7B0A5b131a
		// Note how the last tx of the block contains a tx pfr->mrr of 0.53166 Ether

		// Check sizes are equal (to avoid 0x a non 0x prefix addresses mixups)
		if len(poolAddress) != len(msg.To().String()) {
			log.Fatal("Pool address and msg.to() have different length. poolAddress: ",
				poolAddress, " msg.to(): ", msg.To().String())
		}

		// Check sizes are equal (to avoid 0x a non 0x prefix addresses mixups)
		if len(b.GetFeeRecipient()) != len(msg.From().String()) {
			log.Fatal("Fee recipient and msg.from() have different length. FeeRecipient: ",
				b.GetFeeRecipient(), " msg.from(): ", msg.From().String())
		}

		// First of all the tx must go to the pool address
		if strings.ToLower(poolAddress) == strings.ToLower(msg.To().String()) {
			// And for it to be a MEV reward it must be a transfer from the protocol
			// fee recipient to the validator mev fee recipient
			if strings.ToLower(b.GetFeeRecipient()) == strings.ToLower(msg.From().String()) {
				totalMevReward.Add(totalMevReward, msg.Value())
				log.WithFields(log.Fields{
					"Slot":         b.GetSlot(),
					"Block":        b.GetBlockNumber(),
					"ValIndex":     b.GetProposerIndex(),
					"FeeRecipient": b.GetFeeRecipient()[0:4],
					"To":           msg.To().String(),
					"Reward":       msg.Value(),
					"TxHash":       tx.Hash().String()[0:4],
					"Type":         "MevBlock",
				}).Info("[Reward]")
				numTxs++
			}
		}
	}

	// If there are more than one tx with this characteristic, something is wrong
	// Check manually
	if numTxs > 1 {
		log.Fatal("Multiple MEV rewards to the same address found within a block. This should not happen.")
	}
	return totalMevReward, numTxs, nil
}

// This call is expensive if its a vanila block. The tip sent to the fee recipient
// has to be calculated by iterating all txs and adding the tips as per EIP1559.
// This requires to get every single tx receipt from the block, hence needing
// the onchain to get the receipts from the consensus layer.s
// Note that that this call is cheaper when the block is a MEV block, as there is no
// need to reconstruct the tip from the txs.
func (b *VersionedSignedBeaconBlock) GetSentRewardAndType(
	poolAddress string,
	onchain Onchain) (*big.Int, bool, RewardType, error) {

	var reward *big.Int = big.NewInt(0)
	err := *new(error)
	var numTxs int = 0
	var txType RewardType = UnknownRewardType
	var wasRewardSent bool = false

	if len(b.GetFeeRecipient()) != len(poolAddress) {
		log.Fatal("Fee recipient and pool address have different lengths. FeeRecipient: ",
			b.GetFeeRecipient(), "PoolAddress: ", poolAddress)
	}

	if strings.ToLower(b.GetFeeRecipient()) == strings.ToLower(poolAddress) {
		// vanila block, we get the tip from the block
		blockNumber := new(big.Int).SetUint64(b.GetBlockNumber())
		header, receipts, err := onchain.GetExecHeaderAndReceipts(blockNumber, b.GetBlockTransactions())
		if err != nil {
			log.Fatal(err)
		}

		reward, err = b.GetProposerTip(header, receipts)
		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"Slot":         b.GetSlot(),
			"Block":        b.GetBlockNumber(),
			"ValIndex":     b.GetProposerIndex(),
			"PoolAddress":  poolAddress,
			"Reward":       reward.String(),
			"Type":         "VanilaBlock",
			"FeeRecipient": b.GetFeeRecipient(),
		}).Info("[Reward]")
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
	// this happened in goerli, but a very weird scenario that should never happen in mainnet.
	// can only occur if the pool address is the same as the fee recipient, aka our pool address
	// is the same as the fee recipient sent by the builders.
	if (mevReward.Cmp(big.NewInt(0)) == 1) && (reward.Cmp(big.NewInt(0)) == 1) {
		log.Warn("Both VanilaReward and MevReward are !=0. mevReward: ", mevReward, "vanilaReward: ", reward, ". This should never happen in mainnet.")
	}
	// perhaps refactor this logic
	if mevReward.Cmp(big.NewInt(0)) == 1 {
		reward = mevReward
	}
	if numTxs == 0 {
		// no mev reward, do nothing
	} else if numTxs == 1 {
		// mev block
		wasRewardSent = true
		txType = MevBlock
	} else {
		// TODO: Set to fatal in mainnet
		log.Warn("more than 1 mev tx in a block is not expected. num: ", numTxs)
	}
	return reward, wasRewardSent, txType, nil
}

// Get proposer the proposer tip that went to the fee recepient.
// Note that calculating the tip requires iterating all txs and getting the
// tip by reconstructing it as specified in EIP1559.
func (b *VersionedSignedBeaconBlock) GetProposerTip(blockHeader *types.Header, txReceipts []*types.Receipt) (*big.Int, error) {
	// little-endian to big-endian
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = b.GetBaseFeePerGas()[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)
	if len(b.GetBlockTransactions()) != len(txReceipts) {
		log.Fatal("txs and receipts not the same length")
	}
	for i, rawTx := range b.GetBlockTransactions() {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("could not decode tx")
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
	burnt := new(big.Int).Mul(big.NewInt(int64(b.GetGasUsed())), baseFeePerGas)
	proposerReward := new(big.Int).Sub(tips, burnt)

	return proposerReward, nil
}

// This function detects an Eth transaction sent to an address. Note that if it sent
// by interacting with an smart contract, it will not be detected here.
// This does not takes into account txs made from the fee recipient (MEV txs)
func (b *VersionedSignedBeaconBlock) SentEthToAddress(poolAddress string) *big.Int {
	sentEth := big.NewInt(0)
	numTxs := 0
	for _, rawTx := range b.GetBlockTransactions() {
		_, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("could not decode tx: ", err)
		}
		// If a transaction was sent to the pool
		// And the sender is not the fee recipient (exclude MEV transactions)
		// Note that msg.To() is nil for contract creation transactions
		if msg.To() == nil {
			continue
		}
		// This just detect normal eth transactions sent to the pool address, not via
		// smart conrtacts interactions.
		if strings.ToLower(msg.To().String()) == strings.ToLower(poolAddress) &&
			(strings.ToLower(msg.From().String()) != strings.ToLower(b.GetFeeRecipient())) {

			sentEth.Add(sentEth, msg.Value())
			log.Info("Sent amount: ", msg.Value())
			numTxs++
		}
	}
	return sentEth
}

// Note that this does not detect tx made from smart contract, just plain eth txs
// This function is called on everyblock and MevRewardInWei, which iterate the same
// set of transactions. As a TODO: we can refactor this to only iterate once and get
// both information.
func (b *VersionedSignedBeaconBlock) GetDonations(poolAddress string) []Donation {
	donations := []Donation{}

	for _, rawTx := range b.GetBlockTransactions() {
		tx, msg, err := DecodeTx(rawTx)
		if err != nil {
			log.Fatal("could not decode tx: ", err)
		}
		// If a transaction was sent to the pool
		// And the sender is not the fee recipient (exclude MEV transactions)
		// Note that msg.To() is nil for contract creation transactions
		if msg.To() == nil {
			continue
		}

		// Make sure we are not comparing different length addresses
		if len(msg.To().String()) != len(poolAddress) {
			log.Fatal("pool address is not the same length as msg.To()",
				msg.To().String(), " ", poolAddress)
		}

		// This just detect normal eth transactions sent to the pool address, not via
		// smart conrtacts interactions.
		// It also ignores txs made by the fee recipient (MEV txs)
		if strings.ToLower(msg.To().String()) == strings.ToLower(poolAddress) &&
			(strings.ToLower(msg.From().String()) != strings.ToLower(b.GetFeeRecipient())) {

			// We want pure eth transactions. If its a smart contract interaction (eg subscription)
			// we skip it. Otherwise subscriptions would be detected as donations.
			if len(msg.Data()) > 0 {
				continue
			}

			log.WithFields(log.Fields{
				"RewardWei":   msg.Value(),
				"BlockNumber": b.GetBlockNumber(),
				"Type":        "Donation",
				"TxHash":      tx.Hash().String(),
			}).Info("[Reward]")

			donations = append(donations, Donation{
				AmountWei: msg.Value(),
				Block:     b.GetBlockNumber(),
				TxHash:    tx.Hash().String(),
			},
			)
		}
	}
	return donations
}

// Returns the fee recipient of the block, depending on the fork version
func (b *VersionedSignedBeaconBlock) GetFeeRecipient() string {
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
func (b *VersionedSignedBeaconBlock) GetBlockTransactions() []bellatrix.Transaction {

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

// Returns the block number depending on the fork version
func (b *VersionedSignedBeaconBlock) GetBlockNumber() uint64 {
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

// Returns the slot depending on the fork version
func (b *VersionedSignedBeaconBlock) GetSlot() phase0.Slot {
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

// Returns the proposed index depending on the fork version
func (b *VersionedSignedBeaconBlock) GetProposerIndex() phase0.ValidatorIndex {
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

// Returns the gas used depending on the fork version
func (b *VersionedSignedBeaconBlock) GetGasUsed() uint64 {
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
func (b *VersionedSignedBeaconBlock) GetBaseFeePerGas() [32]byte {
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
