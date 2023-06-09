package api

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type httpOkStatus struct {
	IsConsensusInSync           bool   `json:"is_consensus_in_sync"`
	IsExecutionInSync           bool   `json:"is_execution_in_sync"`
	IsOracleInSync              bool   `json:"is_oracle_in_sync"`
	LatestProcessedSlot         uint64 `json:"latest_processed_slot"`
	LatestProcessedBlock        uint64 `json:"latest_processed_block"`
	LatestFinalizedEpoch        uint64 `json:"latest_finalized_epoch"`
	LatestFinalizedSlot         uint64 `json:"latest_finalized_slot"`
	OracleHeadDistance          uint64 `json:"oracle_sync_distance_slots"`
	NextCheckpointSlot          uint64 `json:"next_checkpoint_slot"`
	NextCheckpointTime          string `json:"next_checkpoint_time"`
	NextCheckpointRemaining     string `json:"next_checkpoint_remaining"`
	NextCheckpointRemainingUnix uint64 `json:"next_checkpoint_remaining_unix"`
	PreviousCheckpointSlot      uint64 `json:"previous_checkpoint_slot"`
	PreviousCheckpointTime      string `json:"previous_checkpoint_time"`
	PreviousCheckpointAge       string `json:"previous_checkpoint_age"`
	PreviousCheckpointAgeUnix   uint64 `json:"previous_checkpoint_age_unix"`
	ConsensusChainId            string `json:"consensus_chainid"`
	ExecutionChainId            string `json:"execution_chainid"`
	DepositContact              string `json:"depositcontract"`
}

type httpOkRelayersState struct {
	CorrectFeeRecipients bool        `json:"correct_fee_recipients"`
	CorrectFeeRelays     []httpRelay `json:"correct_fee_relayers"`
	WrongFeeRelays       []httpRelay `json:"wrong_fee_relayers"`
	UnregisteredRelays   []httpRelay `json:"unregistered_relayers"`
}

type httpRelay struct {
	RelayAddress string `json:"relay_address"`
	FeeRecipient string `json:"fee_recipient"`
	Timestamp    string `json:"timestamp"`
}

type httpOkWithdrawalAddress struct {
	WithdrawalAddress string `json:"withdrawal_address"`
	ValidatorIndex    uint64 `json:"validator_index"`
	ValidatorAddress  string `json:"validator_address"`
}

type httpOkLatestCheckpoint struct {
	MerkleRoot     string `json:"merkleroot"`
	CheckpointSlot uint64 `json:"checkpointslot"`
}

type httpOkMerkleRoot struct {
	MerkleRoot string `json:"merkle_root"`
}

type httpOkMemoryStatistics struct {
	TotalSubscribed            uint64 `json:"total_subscribed_validators"`
	TotalActive                uint64 `json:"total_active_validators"`
	TotalYellowCard            uint64 `json:"total_yellowcard_validators"`
	TotalRedCard               uint64 `json:"total_redcard_validators"`
	TotalBanned                uint64 `json:"total_banned_validators"`
	TotalNotSubscribed         uint64 `json:"total_notsubscribed_validators"`
	LatestCheckpointSlot       uint64 `json:"latest_checkpoint_slot"`
	NextCheckpointSlot         uint64 `json:"next_checkpoint_slot"`
	TotalAccumulatedRewardsWei string `json:"total_accumulated_rewards_wei"`
	TotalPendingRewaradsWei    string `json:"total_pending_rewards_wei"`
	TotalRewardsSentWei        string `json:"total_rewards_sent_wei"`
	TotalDonationsWei          string `json:"total_donations_wei"`
	AvgBlockRewardWei          string `json:"avg_block_reward_wei"`
	TotalProposedBlocks        uint64 `json:"total_proposed_blocks"`
	TotalMissedBlocks          uint64 `json:"total_missed_blocks"`
	TotalWrongFeeBlocks        uint64 `json:"total_wrongfee_blocks"`
}

type httpOkValidatorState struct {
	ValidatorStatus       string `json:"status"`
	AccumulatedRewardsWei string `json:"accumulated_rewards_wei"`
	PendingRewardsWei     string `json:"pending_rewards_wei"`
	CollateralWei         string `json:"collateral_rewards_wei"`
	WithdrawalAddress     string `json:"withdrawal_address"`
	ValidatorIndex        uint64 `json:"validator_index"`
	ValidatorKey          string `json:"validator_key"`
}

type httpOkProofs struct {
	LeafWithdrawalAddress      string   `json:"leaf_withdrawal_address"`
	LeafAccumulatedBalance     string   `json:"leaf_accumulated_balance"`
	MerkleRoot                 string   `json:"merkleroot"`
	CheckpointSlot             uint64   `json:"checkpoint_slot"`
	Proofs                     []string `json:"merkle_proofs"`
	RegisteredValidators       []uint64 `json:"registered_validators"`
	TotalAccumulatedRewardsWei string   `json:"total_accumulated_rewards_wei"`
	AlreadyClaimedRewardsWei   string   `json:"already_claimed_rewards_wei"`
	ClaimableRewardsWei        string   `json:"claimable_rewards_wei"`
	PendingRewardsWei          string   `json:"pending_rewards_wei"`
}

type httpOkConfig struct {
	Network               string `json:"network"`
	PoolAddress           string `json:"pool_address"`
	DeployedSlot          uint64 `json:"deployed_slot"`
	CheckPointSizeInSlots uint64 `json:"checkpoint_size"`
	PoolFeesPercent       int    `json:"pool_fees_percent"`
	PoolFeesAddress       string `json:"pool_fees_address"`
	DryRun                bool   `json:"dry_run"`
	CollateralInWei       string `json:"collateral_in_wei"`
}

type httpOkMemoryFeesInfo struct {
	PoolFeesPercent     int    `json:"pool_fee_percent"`
	PoolFeesAddress     string `json:"pool_fee_address"`
	PoolAccumulatedFees string `json:"pool_accumulated_fees"`
}

type httpOkDonation struct {
	AmountWei string `json:"amount_wei"`
	Block     uint64 `json:"block_number"`
	TxHash    string `json:"tx_hash"`
}

type httpOkBlock struct {
	Slot              uint64 `json:"slot"`
	Block             uint64 `json:"block"`
	ValidatorIndex    uint64 `json:"validator_index"`
	ValidatorKey      string `json:"validator_key"`
	BlockType         string `json:"block_type"`
	Reward            string `json:"reward_wei"`
	RewardType        string `json:"reward_type"`
	WithdrawalAddress string `json:"withdrawal_address"`
}

type httpOkValidatorInfo struct {
	ValidatorStatus       string `json:"status"`
	AccumulatedRewardsWei string `json:"accumulated_rewards_wei"`
	PendingRewardsWei     string `json:"pending_rewards_wei"`
	CollateralWei         string `json:"collateral_wei"`
	WithdrawalAddress     string `json:"withdrawal_address"`
	ValidatorIndex        uint64 `json:"validator_index"`
	ValidatorKey          string `json:"validator_key"`
}
