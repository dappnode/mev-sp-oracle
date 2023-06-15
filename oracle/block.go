package oracle

import (
	"fmt"
	"math/big"
	"strings"

	api "github.com/attestantio/go-eth2-client/api/v1"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// All the events that the contract can emit
type Events struct {
	etherReceived                []*contract.ContractEtherReceived
	subscribeValidator           []*contract.ContractSubscribeValidator
	claimRewards                 []*contract.ContractClaimRewards
	setRewardRecipient           []*contract.ContractSetRewardRecipient
	unsubscribeValidator         []*contract.ContractUnsubscribeValidator
	initSmoothingPool            []*contract.ContractInitSmoothingPool
	updatePoolFee                []*contract.ContractUpdatePoolFee
	poolFeeRecipient             []*contract.ContractUpdatePoolFeeRecipient
	checkpointSlotSize           []*contract.ContractUpdateCheckpointSlotSize
	updateSubscriptionCollateral []*contract.ContractUpdateSubscriptionCollateral
	submitReport                 []*contract.ContractSubmitReport
	reportConsolidated           []*contract.ContractReportConsolidated
	updateQuorum                 []*contract.ContractUpdateQuorum
	addOracleMember              []*contract.ContractAddOracleMember
	removeOracleMember           []*contract.ContractRemoveOracleMember
	transferGovernance           []*contract.ContractTransferGovernance
	acceptGovernance             []*contract.ContractAcceptGovernance
}

// Information that we need of each block
type FullBlock struct {

	// consensus data: duty (mandatory, who should propose the block)
	consensusDuty *api.ProposerDuty

	// consensus data: validator (mandatory, who should propose the block)
	validator *v1.Validator

	// consensus data: block (optional, only when not missed)
	consensusBlock *spec.VersionedSignedBeaconBlock

	// execution data: txs (optional, only when interested in vanila reward)
	executionHeader   *types.Header
	executionReceipts []*types.Receipt

	// execution data: events (optional, only when the block was not missed)
	events *Events
}

// Create a new block with the bare minimum information
func NewFullBlock(
	consensusDuty *api.ProposerDuty,
	validator *v1.Validator) *FullBlock {

	if consensusDuty == nil {
		log.Fatal("consensus duty can't be nil")
	}

	// Some sanity checks
	if validator == nil {
		log.Fatal("validator can't be nil, expected index: ", consensusDuty.ValidatorIndex)
	}
	if validator.Index != consensusDuty.ValidatorIndex {
		log.Fatal("Validator index mismatch between consensus duty and validator: ",
			consensusDuty.ValidatorIndex, " vs ", validator.Index)
	}

	fb := &FullBlock{
		consensusDuty: consensusDuty,
		validator:     validator,
		events: &Events{
			etherReceived:                make([]*contract.ContractEtherReceived, 0),
			subscribeValidator:           make([]*contract.ContractSubscribeValidator, 0),
			claimRewards:                 make([]*contract.ContractClaimRewards, 0),
			setRewardRecipient:           make([]*contract.ContractSetRewardRecipient, 0),
			unsubscribeValidator:         make([]*contract.ContractUnsubscribeValidator, 0),
			initSmoothingPool:            make([]*contract.ContractInitSmoothingPool, 0),
			updatePoolFee:                make([]*contract.ContractUpdatePoolFee, 0),
			poolFeeRecipient:             make([]*contract.ContractUpdatePoolFeeRecipient, 0),
			checkpointSlotSize:           make([]*contract.ContractUpdateCheckpointSlotSize, 0),
			updateSubscriptionCollateral: make([]*contract.ContractUpdateSubscriptionCollateral, 0),
			submitReport:                 make([]*contract.ContractSubmitReport, 0),
			reportConsolidated:           make([]*contract.ContractReportConsolidated, 0),
			updateQuorum:                 make([]*contract.ContractUpdateQuorum, 0),
			addOracleMember:              make([]*contract.ContractAddOracleMember, 0),
			removeOracleMember:           make([]*contract.ContractRemoveOracleMember, 0),
			transferGovernance:           make([]*contract.ContractTransferGovernance, 0),
			acceptGovernance:             make([]*contract.ContractAcceptGovernance, 0),
		},
	}

	return fb
}

// Add consensus data the the full block. Done always unless when the block is missed
func (b *FullBlock) SetConsensusBlock(consensusBlock *spec.VersionedSignedBeaconBlock) {
	if consensusBlock == nil {
		log.Fatal("consensus block can't be nil")
	}

	cBlockSlot, err := consensusBlock.Slot()
	if err != nil {
		log.Fatal("failed to get slot from consensus block: ", err)
	}

	if b.consensusDuty.Slot != cBlockSlot {
		log.Fatal("Slot mismatch between consensus duty and consensus block: ",
			b.consensusDuty.Slot, " vs ", cBlockSlot)
	}

	// Expand for upcoming forks
	var proposerIndex uint64
	if consensusBlock.Altair != nil {
		proposerIndex = uint64(consensusBlock.Altair.Message.ProposerIndex)
	} else if consensusBlock.Bellatrix != nil {
		proposerIndex = uint64(consensusBlock.Bellatrix.Message.ProposerIndex)
	} else if consensusBlock.Capella != nil {
		proposerIndex = uint64(consensusBlock.Capella.Message.ProposerIndex)
	} else {
		log.Fatal("Block was empty, cant get proposer index")
	}

	// Sanity check
	if uint64(b.consensusDuty.ValidatorIndex) != proposerIndex {
		log.Fatal("Proposer index mismatch between consensus duty and consensus block: ",
			b.consensusDuty.ValidatorIndex, " vs ", proposerIndex)
	}

	b.consensusBlock = consensusBlock
}

// Add header and receipts. Only needeed when the block i) sends reward to pool (auto/manual sub)
// or ii) the block belongs to a member of the pool. In blocks we are not interested, this can be
// skipped as fecthing this information is too expensive to do it for every single block.
func (b *FullBlock) SetHeaderAndReceipts(header *types.Header, receipts []*types.Receipt) {
	// Some sanity checks
	if header == nil || receipts == nil {
		log.Fatal("header or receipts can't be nil",
			"header: ", header, " receipts: ", receipts)
	}

	if b.consensusBlock == nil {
		log.Fatal("consensus block can't be nil")
	}

	if b.consensusDuty == nil {
		log.Fatal("consensus duty can't be nil")
	}

	if b.GetBlockNumberBigInt().Uint64() != header.Number.Uint64() {
		log.Fatal("Block number mismatch with header: ",
			b.GetBlockNumberBigInt().Uint64(), " vs ", header.Number.Uint64())
	}

	if len(receipts) != 0 {
		if b.GetBlockNumberBigInt().Uint64() != receipts[0].BlockNumber.Uint64() {
			log.Fatal("Block number mismatch with receipts: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", receipts[0].BlockNumber.Uint64())
		}
	}

	b.executionHeader = header
	b.executionReceipts = receipts
}

// Set the events that were triggered in this block. This shall be done always unless the block
// was missed.
func (b *FullBlock) SetEvents(events *Events) {
	// Some sanity checks
	if events == nil {
		log.Fatal("events can't be nil")
	}

	// More sanity checks, boilerplate but safe
	for _, event := range events.etherReceived {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in etherReceived events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.subscribeValidator {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in subscribeValidator events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.claimRewards {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in claimRewards events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.setRewardRecipient {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in setRewardRecipient events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.unsubscribeValidator {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in unsubscribeValidator events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.initSmoothingPool {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in initSmoothingPool events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.updatePoolFee {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in updatePoolFee events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.poolFeeRecipient {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in poolFeeRecipient events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.checkpointSlotSize {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in checkpointSlotSize events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.updateSubscriptionCollateral {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in updateSubscriptionCollateral events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.submitReport {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in submitReport events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.reportConsolidated {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in reportConsolidated events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.updateQuorum {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in updateQuorum events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.addOracleMember {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in addOracleMember events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.removeOracleMember {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in removeOracleMember events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.transferGovernance {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in transferGovernance events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.acceptGovernance {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in acceptGovernance events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	b.events = events
}

// Returns if there was an mev reward and its amount and fee recipient if any
// Example: https://prater.beaconcha.in/slot/5307417 (0.53166 Eth)
func (b *FullBlock) MevRewardInWei() (*big.Int, bool, string) {

	txs := b.GetBlockTransactions()

	// Check if block is empty (no txs)
	if len(txs) == 0 {
		return big.NewInt(0), false, ""
	}

	lastTx := txs[len(txs)-1]

	tx, err := DecodeTx(lastTx)
	if err != nil {
		log.Fatal("could not decode tx: ", err)
	}

	// Its nil when its a smart contract deployment. No mev reward
	if tx.To() == nil {
		return big.NewInt(0), false, ""
	}

	sender, err := GetTxSender(tx)
	if err != nil {
		log.Fatal("could not get tx sender: ", err)
	}

	// Mev rewards are sent in the last tx. This tx sender
	// matches the fee recipient of the protocol
	if Equals(b.GetFeeRecipient(), sender.String()) {
		return tx.Value(), true, strings.ToLower(tx.To().String())
	}

	return big.NewInt(0), false, ""
}

// Returns if the address received any reward, its amount and its type
func (b *FullBlock) GetSentRewardAndType(
	poolAddress string,
	isSubscriber bool) (*big.Int, bool, RewardType) {

	var reward *big.Int = big.NewInt(0)
	var txType RewardType = UnknownRewardType
	var wasRewardSent bool = false

	// TODO: Wondering if I could run the mevReward first as its cheaper

	// We only calculate the tip for automatic subscribers or subscribed validators
	// since its very expensive to calculate the tip for block we are not interested
	if Equals(b.GetFeeRecipient(), poolAddress) || isSubscriber {
		vanilaReward, err := b.GetProposerTip()
		if err != nil {
			log.Fatal("could not get proposer tip: ", err)
		}

		if Equals(b.GetFeeRecipient(), poolAddress) {
			wasRewardSent = true
		}
		txType = VanilaBlock
		reward = vanilaReward
	}

	// possible mev block
	var mevReward *big.Int = big.NewInt(0)
	mevReward, mevPresent, mevRecipient := b.MevRewardInWei()

	if mevPresent {
		txType = MevBlock
		reward = mevReward
	}

	if mevPresent && Equals(mevRecipient, poolAddress) {
		wasRewardSent = true
	}
	return reward, wasRewardSent, txType
}

func (b *FullBlock) IsAddressRewarded(address string) bool {
	if Equals(b.GetFeeRecipient(), address) {
		return true
	}

	_, isMev, mevRec := b.MevRewardInWei()
	if isMev && Equals(mevRec, address) {
		return true
	}
	return false
}

// The reward for vanila block has to be calculated by iterating all
// txs and getting the individual tips as per EIP1559. Note that to
// calculate this we need the execution header and receipts
func (b *FullBlock) GetProposerTip() (*big.Int, error) {

	// Ensure non nil
	if b.executionReceipts == nil {
		return nil, errors.New("receipts of full block are nil, cant calculate tip")
	}

	if b.executionHeader == nil {
		return nil, errors.New("header of full block are nil, cant calculate tip")
	}

	// Ensure tx and their receipts have the same size
	if len(b.GetBlockTransactions()) != len(b.executionReceipts) {
		return nil, errors.New(fmt.Sprintf("txs and receipts not the same length. txs: %d, receipts: %d",
			len(b.GetBlockTransactions()), len(b.executionReceipts)))
	}

	// little-endian to big-endian
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = b.GetBaseFeePerGas()[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)

	for i, rawTx := range b.GetBlockTransactions() {
		tx, err := DecodeTx(rawTx)
		if err != nil {
			return nil, errors.Wrap(err, "could not decode tx")
		}
		if tx.Hash() != b.executionReceipts[i].TxHash {
			return nil, errors.Wrap(err, "tx hash does not match receipt hash")
		}

		tipFee := new(big.Int)
		gasPrice := tx.GasPrice()
		gasUsed := big.NewInt(int64(b.executionReceipts[i].GasUsed))

		switch tx.Type() {
		case 0:
			tipFee.Mul(gasPrice, gasUsed)
		case 1:
			tipFee.Mul(gasPrice, gasUsed)
		case 2:
			// Sum gastipcap and basefee or saturate to gasfeecap
			usedGasPrice := SumAndSaturate(tx.GasTipCap(), b.executionHeader.BaseFee, tx.GasFeeCap())
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

// Returns the donations sent to the pool. There are two types of donations:
// normal tx: https://goerli.etherscan.io/tx/0xfeda23c2e9db46e69615a8bec74c4a9f3f9f7eb650659a13c9ad1f394c13698d
// via sc: https://goerli.etherscan.io/tx/0x277cec5bcb60852b160a29dc9082b7e18a44333194cbe9c7d7b664e4b89b8c46
// This fuction detects both by checking the tx and the EtherReceived event
func (b *FullBlock) GetDonations(poolAddress string) []*contract.ContractEtherReceived {

	// If the block was missed, there cant be any donations
	if b.consensusBlock == nil {
		return []*contract.ContractEtherReceived{}
	}

	// Leaving for reference. Donations via "normal tx" are detected with this
	//for _, rawTx := range b.GetBlockTransactions() {
	//	tx, msg, err := DecodeTx(rawTx)
	//	if err != nil {
	//		log.Fatal("could not decode tx: ", err)
	//	}
	//
	//	// msg.To() is nil for contract creation transactions
	//	if msg.To() == nil {
	//		continue
	//	}
	//
	//	// Detect possible donation. Mev rewards are filtered
	//	if Equals(msg.To().String(), poolAddress) && !Equals(msg.From().String(), b.GetFeeRecipient()) {
	//
	//		// We want pure eth transactions. If its a smart contract interaction (eg subscription)
	//		// we skip it. Otherwise subscriptions would be detected as donations.
	//		if len(msg.Data()) > 0 {
	//			continue
	//		}
	//
	//		donations = append(donations, Donation{
	//			AmountWei: msg.Value(),
	//			Block:     b.GetBlockNumber(),
	//			TxHash:    tx.Hash().String(),
	//		})
	//	}
	//}

	// EtherReceived event mixes: donations + mev rewards
	// We need to filter out mev rewards
	mevReward, isMev, mevRec := b.MevRewardInWei()

	// If no mev reward or mev reward but not to the pool
	if !isMev || !Equals(mevRec, poolAddress) {
		// In this case we dont expect any etherReceived event due to MEV
		// All events are donations
		return b.events.etherReceived
	}

	// If the pool got an mev reward, we must filter the mev reward
	// from the event, as thats not considered a donation
	filteredEvents := make([]*contract.ContractEtherReceived, 0)
	foundMev := false
	for _, etherRxEvent := range b.events.etherReceived {
		if etherRxEvent.DonationAmount.Cmp(mevReward) == 0 {
			foundMev = true
			continue
		}
		filteredEvents = append(filteredEvents, etherRxEvent)
	}

	// Sanity check
	if !foundMev {
		log.Fatal("An mev reward was expected but could not find it. "+
			"Wanted reward: ", mevReward, " Events: ", b.events.etherReceived)
	}

	return filteredEvents
}

// Since storing the full block is expensive, we store a summarized version of it
func (b *FullBlock) SummarizedBlock(oracle *Oracle, poolAddress string) Block { // TODO these inputs are temporal

	// Get the withdrawal credentials and type of the validator that should propose the block
	withdrawalAddress, withdrawalType := GetWithdrawalAndType(b.validator)

	// Init pool block, with relevant information to the pool
	poolBlock := Block{
		Slot:              uint64(b.consensusDuty.Slot),
		ValidatorIndex:    uint64(b.consensusDuty.ValidatorIndex),
		ValidatorKey:      b.consensusDuty.PubKey.String(),
		WithdrawalAddress: withdrawalAddress,
		Reward:            big.NewInt(0),
	}

	if b.consensusBlock == nil {
		// nil means missed proposal
		poolBlock.BlockType = MissedProposal
		return poolBlock

	} else {

		// Check if the proposal is from a subscribed validator
		isFromSubscriber := oracle.isSubscribed(b.GetProposerIndexUint64())
		isPoolRewarded := b.IsAddressRewarded(poolAddress)

		// This calculation is expensive, do it only if the reward went to the pool or
		// if the block is from a subscribed validator (which includes also wrong fee blocks from subscribers)
		if isFromSubscriber || isPoolRewarded {
			/*
				blockNumber := new(big.Int).SetUint64(b.GetBlockNumber())
				header, receipts, err := o.GetExecHeaderAndReceipts(blockNumber, b.GetBlockTransactions())
				if err != nil {
					log.Fatal("failed getting header and receipts: ", err)
				}
				extendedBlock = NewFullBlock(proposedBlock, header, receipts, events)*/
		}

		// TODO:
		//MEVFeeRecipient
		//FeeRecipient

		// Fetch block information
		reward, correctFeeRec, rewardType := b.GetSentRewardAndType(poolAddress, isFromSubscriber)

		// Populate common parameters
		poolBlock.Reward = reward
		poolBlock.RewardType = rewardType
		poolBlock.Block = b.GetBlockNumber()

		if correctFeeRec {
			// If the fee recipient was correct
			poolBlock.BlockType = OkPoolProposal
			if withdrawalType == BlsWithdrawal {
				poolBlock.BlockType = OkPoolProposalBlsKeys
			} else if withdrawalType == Eth1Withdrawal {
				poolBlock.BlockType = OkPoolProposal
			} else {
				log.Fatal("Unknown withdrawal type: ", withdrawalType)
			}
		} else {
			// If the fee recipient was wrong
			poolBlock.BlockType = WrongFeeRecipient
		}
	}

	return poolBlock
}

// Returns the fee recipient of the block, depending on the fork version
func (b *FullBlock) GetFeeRecipient() string {
	var feeRecipient string

	if b.consensusBlock.Altair != nil {
		log.Fatal("Altair block has no fee recipient")
	} else if b.consensusBlock.Bellatrix != nil {
		feeRecipient = b.consensusBlock.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else if b.consensusBlock.Capella != nil {
		feeRecipient = b.consensusBlock.Capella.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else {
		log.Fatal("Block was empty, cant get fee recipient")
	}
	return feeRecipient
}

// Returns the transactions of the block depending on the fork version
func (b *FullBlock) GetBlockTransactions() []bellatrix.Transaction {

	var transactions []bellatrix.Transaction
	if b.consensusBlock.Altair != nil {
		log.Fatal("Altair block has no transactions in the beacon block")
	} else if b.consensusBlock.Bellatrix != nil {
		transactions = b.consensusBlock.Bellatrix.Message.Body.ExecutionPayload.Transactions
	} else if b.consensusBlock.Capella != nil {
		transactions = b.consensusBlock.Capella.Message.Body.ExecutionPayload.Transactions
	} else {
		log.Fatal("Block was empty, cant get transactions")
	}
	return transactions
}

// Returns the block number depending on the fork version (as uint64)
func (b *FullBlock) GetBlockNumber() uint64 {
	var blockNumber uint64

	if b.consensusBlock.Altair != nil {
		log.Fatal("Altair block has no block number")
	} else if b.consensusBlock.Bellatrix != nil {
		blockNumber = b.consensusBlock.Bellatrix.Message.Body.ExecutionPayload.BlockNumber
	} else if b.consensusBlock.Capella != nil {
		blockNumber = b.consensusBlock.Capella.Message.Body.ExecutionPayload.BlockNumber
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

	if b.consensusBlock.Altair != nil {
		slot = b.consensusBlock.Altair.Message.Slot
	} else if b.consensusBlock.Bellatrix != nil {
		slot = b.consensusBlock.Bellatrix.Message.Slot
	} else if b.consensusBlock.Capella != nil {
		slot = b.consensusBlock.Capella.Message.Slot
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

	if b.consensusBlock.Altair != nil {
		proposerIndex = b.consensusBlock.Altair.Message.ProposerIndex
	} else if b.consensusBlock.Bellatrix != nil {
		proposerIndex = b.consensusBlock.Bellatrix.Message.ProposerIndex
	} else if b.consensusBlock.Capella != nil {
		proposerIndex = b.consensusBlock.Capella.Message.ProposerIndex
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

	if b.consensusBlock.Altair != nil {
		log.Fatal("Altair block has no gas used")
	} else if b.consensusBlock.Bellatrix != nil {
		gasUsed = b.consensusBlock.Bellatrix.Message.Body.ExecutionPayload.GasUsed
	} else if b.consensusBlock.Capella != nil {
		gasUsed = b.consensusBlock.Capella.Message.Body.ExecutionPayload.GasUsed
	} else {
		log.Fatal("Block was empty, cant get gas used")
	}
	return gasUsed
}

// Returns the base fee per gas depending on the fork version
func (b *FullBlock) GetBaseFeePerGas() [32]byte {
	var baseFeePerGas [32]byte

	if b.consensusBlock.Altair != nil {
		log.Fatal("Altair block has no base fee per gas")
	} else if b.consensusBlock.Bellatrix != nil {
		baseFeePerGas = b.consensusBlock.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas
	} else if b.consensusBlock.Capella != nil {
		baseFeePerGas = b.consensusBlock.Capella.Message.Body.ExecutionPayload.BaseFeePerGas
	} else {
		log.Fatal("Block was empty, cant get base fee per gas")
	}
	return baseFeePerGas
}
