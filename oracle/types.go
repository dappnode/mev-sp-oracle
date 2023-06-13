package oracle

import (
	"encoding/json"
	"math/big"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/pkg/errors"
)

const DefaultRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"
const DefaultAddress = "0x0000000000000000000000000000000000000000"

// Types of block rewards
type RewardType uint8

const (
	UnknownRewardType RewardType = 0
	VanilaBlock       RewardType = 1
	MevBlock          RewardType = 2
)

// States of the state machine
type ValidatorStatus uint8

const (
	UnknownState  ValidatorStatus = 0
	Active        ValidatorStatus = 1
	YellowCard    ValidatorStatus = 2
	RedCard       ValidatorStatus = 3
	NotSubscribed ValidatorStatus = 4
	Banned        ValidatorStatus = 5
	Untracked     ValidatorStatus = 6
)

// Events in the state machine that trigger transition
type Event uint8

const (
	UnknownEvent       Event = 0
	ProposalOk         Event = 1
	ProposalMissed     Event = 2
	ProposalWrongFee   Event = 3
	ManualSubscription Event = 4
	AutoSubscription   Event = 5
	Unsubscribe        Event = 6
)

// Block type
type BlockType uint8

const (
	UnknownBlockType      BlockType = 0
	MissedProposal        BlockType = 1
	WrongFeeRecipient     BlockType = 2
	OkPoolProposal        BlockType = 3
	OkPoolProposalBlsKeys BlockType = 4
)

// Withdrawal credentials type
type WithdrawalType uint8

const (
	BlsWithdrawal  WithdrawalType = 0
	Eth1Withdrawal WithdrawalType = 1
)

// Represents a block with information relevant for the pool
// TODO: Call SummarizedBlock?
// This is to avoid storing the whole block
type Block struct {
	Slot              uint64     `json:"slot"`
	Block             uint64     `json:"block"`
	ValidatorIndex    uint64     `json:"validator_index"`
	ValidatorKey      string     `json:"validator_key"`
	BlockType         BlockType  `json:"block_type"`
	Reward            *big.Int   `json:"reward_wei"`
	RewardType        RewardType `json:"reward_type"`
	WithdrawalAddress string     `json:"withdrawal_address"`
}

// Represents a donation made to the pool
// TODO: deprecate this? donations are detected from the block content
type Donation struct {
	AmountWei *big.Int
	Block     uint64
	TxHash    string
}

// Subscription event and the associated validator (if any)
// TODO: Store directly the event?¿
type Subscription struct {
	Event     *contract.ContractSubscribeValidator
	Validator *v1.Validator
}

// Unsubscription event and the associated validator (if any)
type Unsubscription struct {
	Event     *contract.ContractUnsubscribeValidator
	Validator *v1.Validator
}

// Represents all the information that is stored of a validator
type ValidatorInfo struct {
	ValidatorStatus       ValidatorStatus `json:"status"`
	AccumulatedRewardsWei *big.Int        `json:"accumulated_rewards_wei"`
	PendingRewardsWei     *big.Int        `json:"pending_rewards_wei"`
	CollateralWei         *big.Int        `json:"collateral_wei"`
	WithdrawalAddress     string          `json:"withdrawal_address"`
	ValidatorIndex        uint64          `json:"validator_index"`
	ValidatorKey          string          `json:"validator_key"`
}

// Represents the latest commited state onchain
type OnchainState struct {
	Slot       uint64                    `json:"slot"`
	TxHash     string                    `json:"tx_hash"`
	MerkleRoot string                    `json:"merkle_root"`
	Validators map[uint64]*ValidatorInfo `json:"validators"`
	Leafs      map[string]RawLeaf        `json:"leafs"`
	Proofs     map[string][]string       `json:"proofs"`
}

type OracleState struct {
	StateHash            string                    `json:"state_hash"`
	LatestProcessedSlot  uint64                    `json:"latest_processed_slot"`
	LatestProcessedBlock uint64                    `json:"latest_processed_block"`
	NextSlotToProcess    uint64                    `json:"next_slot_to_process"`
	Network              string                    `json:"network"`
	PoolAddress          string                    `json:"pool_address"`
	Validators           map[uint64]*ValidatorInfo `json:"validators"`
	CommitedStates       map[string]OnchainState   `json:"commited_states"`
	LatestCommitedState  OnchainState              `json:"latest_commited_state"`

	// TODO: is this redundant? its in the config
	PoolFeesPercent     int      `json:"pool_fees_percent"` // TODO: is this % or scaled by *100
	PoolFeesAddress     string   `json:"pool_fees_address"`
	PoolAccumulatedFees *big.Int `json:"pool_accumulated_fees"`

	Subscriptions   []Subscription   `json:"subscriptions"`
	Unsubscriptions []Unsubscription `json:"unsubscriptions"`
	Donations       []Donation       `json:"donations"`
	ProposedBlocks  []Block          `json:"proposed_blocks"`
	MissedBlocks    []Block          `json:"missed_blocks"`
	WrongFeeBlocks  []Block          `json:"wrong_fee_blocks"`

	// unsure if config should be here. maybe not TODO:
	Config *config.Config `json:"todo_unsure"`
}

type RawLeaf struct {
	WithdrawalAddress     string   `json:"withdrawal_address"`
	AccumulatedBalanceWei *big.Int `json:"accumulated_balance_wei"`
}

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

// TODO: Test all this
func (r *RewardType) String() string {
	if *r == VanilaBlock {
		return "vanila"
	} else if *r == MevBlock {
		return "mev"
	} else if *r == UnknownRewardType {
		return "unknownrewardtype"
	}
	return ""
}

func (s *RewardType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *RewardType) UnmarshalJSON(b []byte) error {
	var rewardtype string
	if err := json.Unmarshal(b, &rewardtype); err != nil {
		return errors.Wrap(err, "unmarshaling reward type")
	}

	if rewardtype == "vanila" {
		*s = VanilaBlock
	} else if rewardtype == "mev" {
		*s = MevBlock
	} else if rewardtype == "unknownrewardtype" {
		*s = UnknownRewardType
	} else {
		return errors.New("unknown reward type")
	}
	return nil
}

func (v ValidatorStatus) String() string {
	if v == Active {
		return "active"
	} else if v == YellowCard {
		return "yellowcard"
	} else if v == RedCard {
		return "redcard"
	} else if v == NotSubscribed {
		return "notsubscribed"
	} else if v == Banned {
		return "banned"
	} else if v == Untracked {
		return "untracked"
	} else if v == UnknownState {
		return "unknownstate"
	}
	return ""
}
func (s *ValidatorStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *ValidatorStatus) UnmarshalJSON(b []byte) error {
	var status string
	if err := json.Unmarshal(b, &status); err != nil {
		return errors.Wrap(err, "unmarshaling validator status")
	}

	if status == "active" {
		*s = Active
	} else if status == "yellowcard" {
		*s = YellowCard
	} else if status == "redcard" {
		*s = RedCard
	} else if status == "notsubscribed" {
		*s = NotSubscribed
	} else if status == "banned" {
		*s = Banned
	} else if status == "untracked" {
		*s = Untracked
	} else if status == "unknownstate" {
		*s = UnknownState
	} else {
		return errors.New("unknown validator status")
	}
	return nil
}

func (b *BlockType) String() string {
	if *b == MissedProposal {
		return "missedproposal"
	} else if *b == WrongFeeRecipient {
		return "wrongfeerecipient"
	} else if *b == OkPoolProposal {
		return "okpoolproposal"
	} else if *b == OkPoolProposalBlsKeys {
		return "okpoolproposalblskeys"
	} else if *b == UnknownBlockType {
		return "unknownblocktype"
	}
	return ""
}

func (s *BlockType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *BlockType) UnmarshalJSON(b []byte) error {
	var blocktype string
	if err := json.Unmarshal(b, &blocktype); err != nil {
		return errors.Wrap(err, "unmarshaling block type")
	}

	if blocktype == "missedproposal" {
		*s = MissedProposal
	} else if blocktype == "wrongfeerecipient" {
		*s = WrongFeeRecipient
	} else if blocktype == "okpoolproposal" {
		*s = OkPoolProposal
	} else if blocktype == "okpoolproposalblskeys" {
		*s = OkPoolProposalBlsKeys
	} else if blocktype == "unknownblocktype" {
		*s = UnknownBlockType
	} else {
		return errors.New("unknown block type")
	}
	return nil
}
