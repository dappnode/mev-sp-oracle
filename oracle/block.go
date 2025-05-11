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
	"github.com/dappnode/mev-sp-oracle/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Whitelisted builders for each chain. A transactions that comes from any
// of those AND if its last tx in the block, its considered MEV reward.
var WhitelistedBuilders = map[uint64][]string{
	MainnetChainId: {
		"0xae0A3D884E746599BD6C893a674E556C36a47f1e",
	},
	HoleskyChainId: {},
	HoodiChainId:   {},
}

// See MevRewardInWei for more info
// https://beaconcha.in/slot/10400574
var ExceptionSlotMainnet1 = uint64(10400574)

// Create a new block with the bare minimum information
func NewFullBlock(
	consensusDuty *api.ProposerDuty,
	validator *v1.Validator,
	chainId uint64) *FullBlock {

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
		ConsensusDuty: consensusDuty,
		Validator:     validator,
		Events: &Events{
			EtherReceived:                make([]*contract.ContractEtherReceived, 0),
			SubscribeValidator:           make([]*contract.ContractSubscribeValidator, 0),
			ClaimRewards:                 make([]*contract.ContractClaimRewards, 0),
			SetRewardRecipient:           make([]*contract.ContractSetRewardRecipient, 0),
			UnsubscribeValidator:         make([]*contract.ContractUnsubscribeValidator, 0),
			InitSmoothingPool:            make([]*contract.ContractInitSmoothingPool, 0),
			UpdatePoolFee:                make([]*contract.ContractUpdatePoolFee, 0),
			PoolFeeRecipient:             make([]*contract.ContractUpdatePoolFeeRecipient, 0),
			CheckpointSlotSize:           make([]*contract.ContractUpdateCheckpointSlotSize, 0),
			UpdateSubscriptionCollateral: make([]*contract.ContractUpdateSubscriptionCollateral, 0),
			SubmitReport:                 make([]*contract.ContractSubmitReport, 0),
			ReportConsolidated:           make([]*contract.ContractReportConsolidated, 0),
			UpdateQuorum:                 make([]*contract.ContractUpdateQuorum, 0),
			AddOracleMember:              make([]*contract.ContractAddOracleMember, 0),
			RemoveOracleMember:           make([]*contract.ContractRemoveOracleMember, 0),
			TransferGovernance:           make([]*contract.ContractTransferGovernance, 0),
			AcceptGovernance:             make([]*contract.ContractAcceptGovernance, 0),
		},
		ChainId: chainId,
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

	if b.ConsensusDuty.Slot != cBlockSlot {
		log.Fatal("Slot mismatch between consensus duty and consensus block: ",
			b.ConsensusDuty.Slot, " vs ", cBlockSlot)
	}

	// Expand for upcoming forks
	var proposerIndex uint64
	if consensusBlock.Altair != nil {
		proposerIndex = uint64(consensusBlock.Altair.Message.ProposerIndex)
	} else if consensusBlock.Bellatrix != nil {
		proposerIndex = uint64(consensusBlock.Bellatrix.Message.ProposerIndex)
	} else if consensusBlock.Capella != nil {
		proposerIndex = uint64(consensusBlock.Capella.Message.ProposerIndex)
	} else if consensusBlock.Deneb != nil {
		proposerIndex = uint64(consensusBlock.Deneb.Message.ProposerIndex)
	} else if consensusBlock.Electra != nil {
		proposerIndex = uint64(consensusBlock.Electra.Message.ProposerIndex)
	} else {
		log.Fatal("Block was empty, cant get proposer index")
	}

	// Sanity check
	if uint64(b.ConsensusDuty.ValidatorIndex) != proposerIndex {
		log.Fatal("Proposer index mismatch between consensus duty and consensus block: ",
			b.ConsensusDuty.ValidatorIndex, " vs ", proposerIndex)
	}

	b.ConsensusBlock = consensusBlock
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

	if b.ConsensusBlock == nil {
		log.Fatal("consensus block can't be nil")
	}

	if b.ConsensusDuty == nil {
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

	b.ExecutionHeader = header
	b.ExecutionReceipts = receipts
}

// Set the events that were triggered in this block. This shall be done always unless the block
// was missed.
func (b *FullBlock) SetEvents(events *Events) {
	// Some sanity checks
	if events == nil {
		log.Fatal("events can't be nil")
	}

	// More sanity checks, boilerplate but safe
	for _, event := range events.EtherReceived {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in etherReceived events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.SubscribeValidator {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in subscribeValidator events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.ClaimRewards {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in claimRewards events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.SetRewardRecipient {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in setRewardRecipient events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.UnsubscribeValidator {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in unsubscribeValidator events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.InitSmoothingPool {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in initSmoothingPool events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.UpdatePoolFee {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in updatePoolFee events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.PoolFeeRecipient {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in poolFeeRecipient events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.CheckpointSlotSize {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in checkpointSlotSize events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.UpdateSubscriptionCollateral {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in updateSubscriptionCollateral events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.SubmitReport {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in submitReport events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.ReportConsolidated {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in reportConsolidated events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.UpdateQuorum {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in updateQuorum events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.AddOracleMember {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in addOracleMember events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.RemoveOracleMember {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in removeOracleMember events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.TransferGovernance {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in transferGovernance events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	for _, event := range events.AcceptGovernance {
		if b.GetBlockNumberBigInt().Uint64() != event.Raw.BlockNumber {
			log.Fatal("Block number mismatch in acceptGovernance events: ",
				b.GetBlockNumberBigInt().Uint64(), " vs ", event.Raw.BlockNumber)
		}
	}

	b.Events = events

	// Special case. Temporal fix. We artificially add the mev reward for this block.
	// See: https://github.com/dappnode/mev-sp-oracle/pull/230
	// See: https://beaconcha.in/slot/10400574
	if b.ChainId == MainnetChainId && b.GetSlotUint64() == ExceptionSlotMainnet1 {
		log.Warn("Special case: MEV reward hardcoded. See MevRewardInWei for more info")
		b.Events.EtherReceived = append(b.Events.EtherReceived, &contract.ContractEtherReceived{
			Sender:         common.Address{},
			DonationAmount: big.NewInt(177043568463114308),
			Raw: types.Log{
				Address:     common.Address{},
				Topics:      []common.Hash{},
				Data:        []byte{},
				BlockNumber: 0,
				TxHash:      common.Hash{},
				TxIndex:     0,
				BlockHash:   common.Hash{},
				Index:       0,
				Removed:     false,
			},
		})
	}
}

// Returns if there was an mev reward and its amount and fee recipient if any
// Example: https://prater.beaconcha.in/slot/5307417 (0.53166 Eth)
func (b *FullBlock) MevRewardInWei() (*big.Int, bool, string) {

	txs := b.GetBlockTransactions()

	// Check if block is empty (no txs)
	if len(txs) == 0 {
		return big.NewInt(0), false, ""
	}

	// Get the last tx which is the one that contains the mev reward
	lastTx := txs[len(txs)-1]

	tx, err := utils.DecodeTx(lastTx)
	if err != nil {
		log.Fatal("could not decode tx: ", err)
	}

	// Its nil when its a smart contract deployment. No mev reward
	if tx.To() == nil {
		return big.NewInt(0), false, ""
	}

	sender, err := utils.GetTxSender(tx, b.ChainId)
	if err != nil {
		log.Fatal("could not get tx sender: ", err)
	}

	whitelistedBuilders, found := WhitelistedBuilders[b.ChainId]
	if !found {
		log.Fatal("Chain not found in whitelisted builders: ", b.ChainId)
	}

	// Special case. To be fixed.
	// This block has a mev but the last contains a self destruct which
	// makes the EtherReceived event to not be triggered.
	// It could be solved with debug_traceTransaction but this call is expensive
	// and requires an archival execution node running with debug.
	// Since this is rare, we just hardcode the mev reward for this block by now.
	// https://beaconcha.in/slot/10400574
	if b.ChainId == MainnetChainId && b.GetSlotUint64() == ExceptionSlotMainnet1 {
		hardcodedMevReward := big.NewInt(177043568463114308)
		hardcodedMevRecipient := "0xAdFb8D27671F14f297eE94135e266aAFf8752e35"
		log.WithFields(log.Fields{
			"Network":            MainnetChainId,
			"Slot":               ExceptionSlotMainnet1,
			"HardcodedMevReward": hardcodedMevReward,
			"MevRecipient":       hardcodedMevRecipient,
		}).Info("Special case: MEV reward hardcoded")

		return hardcodedMevReward,
			true,
			hardcodedMevRecipient
	}

	// Mev rewards are sent in the last tx. This tx sender
	// matches the fee recipient of the protocol.
	// We also consider a MEV reward if the tx comes from a whitelisted builder. This
	// is rare, but has happened: https://beaconcha.in/slot/9444748
	if utils.Equals(b.GetFeeRecipient(), sender.String()) ||
		utils.IsIn(sender.String(), whitelistedBuilders) {

		if utils.IsIn(sender.String(), whitelistedBuilders) {
			log.WithFields(log.Fields{
				"LastTxSender":       sender.String(),
				"WhitelistedAddress": whitelistedBuilders,
			}).Info("Last block tx was sent by whitelisted builder")
		}
		// MEV reward can also be sent via a smart contract, in which case the
		// receiver is the pool address. Example:
		// https://etherscan.io/tx/0x6c9adaa16946d1279e0db0fc9348201c48b2f70a62ac5edfe06dc0ba2b4f3e3c
		// Note that the sender is still the protocol fee recipient, which allows us to distinguish
		// between a mev reward and a donation as the last tx of the block
		if b.Events.EtherReceived != nil {
			for _, event := range b.Events.EtherReceived {
				if event.DonationAmount.Cmp(tx.Value()) == 0 {
					return tx.Value(), true, strings.ToLower(event.Raw.Address.String())
				}
			}
		}
		return tx.Value(), true, strings.ToLower(tx.To().String())
	}

	// Otherwise, there is no MEV reward
	return big.NewInt(0), false, ""
}

// Returns if the address received any reward, its amount and its type. A reward
// can be i) mev (MEV reward) or ii) vanila (just fees as per EIP1559)
// For the oracle, a reward is either one type or the other. It cannot be both
func (b *FullBlock) GetSentRewardAndType(
	poolAddress string,
	isSubscriber bool) (*big.Int, bool, RewardType) {

	var reward *big.Int = big.NewInt(0)
	var txType RewardType = UnknownRewardType
	var wasRewardSent bool = false

	// i) check if mev reward (first as its cheaper to check)
	mevReward, mevPresent, mevRecipient := b.MevRewardInWei()

	if mevPresent {
		// there is mev, store its value and set type
		txType = MevBlock
		reward = mevReward
		wasRewardSent = false

		// if the mev reward was sent to the pool address
		if utils.Equals(mevRecipient, poolAddress) {
			wasRewardSent = true
		}

		return reward, wasRewardSent, txType
	}

	// ii) check if vanila reward (calculating this is expensive as requires headers)
	// so its done only if needed. Note that this reward does not trigger EtherReceived
	// events, as its built in the protocol
	if utils.Equals(b.GetFeeRecipient(), poolAddress) || isSubscriber {
		vanilaReward, err := b.GetProposerTip()
		if err != nil {
			log.Fatal("could not get proposer tip: ", err)
		}

		if utils.Equals(b.GetFeeRecipient(), poolAddress) {
			wasRewardSent = true
		}
		txType = VanilaBlock

		// This reward is only set here, otherwise we dont realy care about it
		// and its expensive to calculate as it requires the headers
		reward = vanilaReward
	}

	return reward, wasRewardSent, txType
}

func (b *FullBlock) isAddressRewarded(address string) bool {
	if utils.Equals(b.GetFeeRecipient(), address) {
		return true
	}

	_, isMev, mevRec := b.MevRewardInWei()
	if isMev && utils.Equals(mevRec, address) {
		return true
	}
	return false
}

// The reward for vanila block has to be calculated by iterating all
// txs and getting the individual tips as per EIP1559. Note that to
// calculate this we need the execution header and receipts
func (b *FullBlock) GetProposerTip() (*big.Int, error) {

	// Ensure non nil
	if b.ExecutionReceipts == nil {
		return nil, errors.New("receipts of full block are nil, cant calculate tip")
	}

	if b.ExecutionHeader == nil {
		return nil, errors.New("header of full block are nil, cant calculate tip")
	}

	// Ensure tx and their receipts have the same size
	if len(b.GetBlockTransactions()) != len(b.ExecutionReceipts) {
		return nil, errors.New(fmt.Sprintf("txs and receipts not the same length. txs: %d, receipts: %d",
			len(b.GetBlockTransactions()), len(b.ExecutionReceipts)))
	}

	// little-endian to big-endian
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = b.GetBaseFeePerGas()[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)

	for i, rawTx := range b.GetBlockTransactions() {
		tx, err := utils.DecodeTx(rawTx)
		if err != nil {
			return nil, errors.Wrap(err, "could not decode tx")
		}
		if tx.Hash() != b.ExecutionReceipts[i].TxHash {
			return nil, errors.Wrap(err, "tx hash does not match receipt hash")
		}

		tipFee := new(big.Int)
		gasPrice := tx.GasPrice()
		gasUsed := big.NewInt(int64(b.ExecutionReceipts[i].GasUsed))

		switch tx.Type() {
		case 0:
			tipFee.Mul(gasPrice, gasUsed)
		case 1:
			tipFee.Mul(gasPrice, gasUsed)
		case 2, 3, 4:
			// Sum gastipcap and basefee or saturate to gasfeecap
			usedGasPrice := utils.SumAndSaturate(tx.GasTipCap(), b.ExecutionHeader.BaseFee, tx.GasFeeCap())
			tipFee = new(big.Int).Mul(usedGasPrice, gasUsed)
		default:
			return nil, errors.New(fmt.Sprintf("unknown tx type: %d, hash: %s", tx.Type(), tx.Hash().String()))
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
	if b.ConsensusBlock == nil {
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
	if !isMev || !utils.Equals(mevRec, poolAddress) {
		// In this case we dont expect any etherReceived event due to MEV
		// All events are donations
		return b.Events.EtherReceived
	}

	// If the pool got an mev reward, we must filter the mev reward
	// from the event, as thats not considered a donation
	filteredEvents := make([]*contract.ContractEtherReceived, 0)
	foundMev := false
	for _, etherRxEvent := range b.Events.EtherReceived {
		if etherRxEvent.DonationAmount.Cmp(mevReward) == 0 {
			foundMev = true
			continue
		}
		filteredEvents = append(filteredEvents, etherRxEvent)
	}

	// Sanity check
	if !foundMev {
		log.Fatal("An mev reward was expected but could not find it. "+
			"Wanted reward: ", mevReward, " Events: ", b.Events.EtherReceived)
	}

	return filteredEvents
}

// Since storing the full block is expensive, we store a summarized version of it
func (b *FullBlock) SummarizedBlock(oracle *Oracle, poolAddress string) SummarizedBlock {

	// Get the withdrawal credentials and type of the validator that should propose the block
	withdrawalAddress, withdrawalType := GetWithdrawalAndType(b.Validator)

	// Init pool block, with relevant information to the pool
	poolBlock := SummarizedBlock{
		Slot:              uint64(b.ConsensusDuty.Slot),
		ValidatorIndex:    uint64(b.ConsensusDuty.ValidatorIndex),
		ValidatorKey:      b.ConsensusDuty.PubKey.String(),
		WithdrawalAddress: withdrawalAddress,
		Reward:            big.NewInt(0),
	}

	if b.ConsensusBlock == nil {
		// nil means missed proposal
		poolBlock.BlockType = MissedProposal
		return poolBlock

	} else {
		// Check if the proposer is subscribed to the pool
		isFromSubscriber := oracle.isSubscribed(b.GetProposerIndexUint64())

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
			} else if withdrawalType == ElectraWithdrawal {
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

	if b.ConsensusBlock.Altair != nil {
		log.Fatal("Altair block has no fee recipient")
	} else if b.ConsensusBlock.Bellatrix != nil {
		feeRecipient = b.ConsensusBlock.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else if b.ConsensusBlock.Capella != nil {
		feeRecipient = b.ConsensusBlock.Capella.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else if b.ConsensusBlock.Deneb != nil {
		feeRecipient = b.ConsensusBlock.Deneb.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else if b.ConsensusBlock.Electra != nil {
		feeRecipient = b.ConsensusBlock.Electra.Message.Body.ExecutionPayload.FeeRecipient.String()
	} else {
		log.Fatal("Block was empty, cant get fee recipient")
	}
	return feeRecipient
}

// Returns the transactions of the block depending on the fork version
func (b *FullBlock) GetBlockTransactions() []bellatrix.Transaction {

	var transactions []bellatrix.Transaction
	if b.ConsensusBlock.Altair != nil {
		log.Fatal("Altair block has no transactions in the beacon block")
	} else if b.ConsensusBlock.Bellatrix != nil {
		transactions = b.ConsensusBlock.Bellatrix.Message.Body.ExecutionPayload.Transactions
	} else if b.ConsensusBlock.Capella != nil {
		transactions = b.ConsensusBlock.Capella.Message.Body.ExecutionPayload.Transactions
	} else if b.ConsensusBlock.Deneb != nil {
		transactions = b.ConsensusBlock.Deneb.Message.Body.ExecutionPayload.Transactions
	} else if b.ConsensusBlock.Electra != nil {
		transactions = b.ConsensusBlock.Electra.Message.Body.ExecutionPayload.Transactions
	} else {
		log.Fatal("Block was empty, cant get transactions")
	}
	return transactions
}

// Returns the block number depending on the fork version (as uint64)
func (b *FullBlock) GetBlockNumber() uint64 {
	var blockNumber uint64

	if b.ConsensusBlock.Altair != nil {
		log.Fatal("Altair block has no block number")
	} else if b.ConsensusBlock.Bellatrix != nil {
		blockNumber = b.ConsensusBlock.Bellatrix.Message.Body.ExecutionPayload.BlockNumber
	} else if b.ConsensusBlock.Capella != nil {
		blockNumber = b.ConsensusBlock.Capella.Message.Body.ExecutionPayload.BlockNumber
	} else if b.ConsensusBlock.Deneb != nil {
		blockNumber = b.ConsensusBlock.Deneb.Message.Body.ExecutionPayload.BlockNumber
	} else if b.ConsensusBlock.Electra != nil {
		blockNumber = b.ConsensusBlock.Electra.Message.Body.ExecutionPayload.BlockNumber
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

	if b.ConsensusBlock.Altair != nil {
		slot = b.ConsensusBlock.Altair.Message.Slot
	} else if b.ConsensusBlock.Bellatrix != nil {
		slot = b.ConsensusBlock.Bellatrix.Message.Slot
	} else if b.ConsensusBlock.Capella != nil {
		slot = b.ConsensusBlock.Capella.Message.Slot
	} else if b.ConsensusBlock.Deneb != nil {
		slot = b.ConsensusBlock.Deneb.Message.Slot
	} else if b.ConsensusBlock.Electra != nil {
		slot = b.ConsensusBlock.Electra.Message.Slot
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

	if b.ConsensusBlock.Altair != nil {
		proposerIndex = b.ConsensusBlock.Altair.Message.ProposerIndex
	} else if b.ConsensusBlock.Bellatrix != nil {
		proposerIndex = b.ConsensusBlock.Bellatrix.Message.ProposerIndex
	} else if b.ConsensusBlock.Capella != nil {
		proposerIndex = b.ConsensusBlock.Capella.Message.ProposerIndex
	} else if b.ConsensusBlock.Deneb != nil {
		proposerIndex = b.ConsensusBlock.Deneb.Message.ProposerIndex
	} else if b.ConsensusBlock.Electra != nil {
		proposerIndex = b.ConsensusBlock.Electra.Message.ProposerIndex
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

	if b.ConsensusBlock.Altair != nil {
		log.Fatal("Altair block has no gas used")
	} else if b.ConsensusBlock.Bellatrix != nil {
		gasUsed = b.ConsensusBlock.Bellatrix.Message.Body.ExecutionPayload.GasUsed
	} else if b.ConsensusBlock.Capella != nil {
		gasUsed = b.ConsensusBlock.Capella.Message.Body.ExecutionPayload.GasUsed
	} else if b.ConsensusBlock.Deneb != nil {
		gasUsed = b.ConsensusBlock.Deneb.Message.Body.ExecutionPayload.GasUsed
	} else if b.ConsensusBlock.Electra != nil {
		gasUsed = b.ConsensusBlock.Electra.Message.Body.ExecutionPayload.GasUsed
	} else {
		log.Fatal("Block was empty, cant get gas used")
	}
	return gasUsed
}

// Returns the base fee per gas depending on the fork version
func (b *FullBlock) GetBaseFeePerGas() [32]byte {
	var baseFeePerGas [32]byte

	if b.ConsensusBlock.Altair != nil {
		log.Fatal("Altair block has no base fee per gas")
	} else if b.ConsensusBlock.Bellatrix != nil {
		baseFeePerGas = b.ConsensusBlock.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas
	} else if b.ConsensusBlock.Capella != nil {
		baseFeePerGas = b.ConsensusBlock.Capella.Message.Body.ExecutionPayload.BaseFeePerGas
	} else if b.ConsensusBlock.Deneb != nil {
		// Due to this change: https://github.com/attestantio/go-eth2-client/commit/acadd726168dac047ab3b13b4aceaf2a6103dab5
		// the base fee is no longer stored as a [32]byte little endian, but as a big endian. To avoid considering is as an special
		// case, we convert it to little endian, so that the interface is respected.
		baseFeePerGasBigEndian := b.ConsensusBlock.Deneb.Message.Body.ExecutionPayload.BaseFeePerGas.Bytes32()

		// big-endian to little-endian
		for i := 0; i < 32; i++ {
			baseFeePerGas[i] = baseFeePerGasBigEndian[32-1-i]
		}

	} else if b.ConsensusBlock.Electra != nil {
		// Due to this change: https://github.com/attestantio/go-eth2-client/commit/acadd726168dac047ab3b13b4aceaf2a6103dab5
		// the base fee is no longer stored as a [32]byte little endian, but as a big endian. To avoid considering is as an special
		// case, we convert it to little endian, so that the interface is respected.
		baseFeePerGasBigEndian := b.ConsensusBlock.Electra.Message.Body.ExecutionPayload.BaseFeePerGas.Bytes32()

		// big-endian to little-endian
		for i := 0; i < 32; i++ {
			baseFeePerGas[i] = baseFeePerGasBigEndian[32-1-i]
		}

	} else {
		log.Fatal("Block was empty, cant get base fee per gas")
	}
	return baseFeePerGas
}
