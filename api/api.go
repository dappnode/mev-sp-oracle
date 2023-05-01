package api

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/avast/retry-go/v4"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/oracle"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// TODO: Add getters so that the api cannot screw up the state

// Note that the api has no paging, so it is not suitable for large queries, but
// it should be able to scale to a few thousand subscribed validators without any problem

// Important: These are the retry options when an api call involves external call to
// the beacon node or execution client. The idea is to try once, and fail fast.
// Use this for all onchain calls, otherwise defaultRetryOpts will be aplied
var apiRetryOpts = []retry.Option{
	retry.Attempts(1),
}

const defaultMerkleRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"

const (
	// Available endpoints
	pathStatus            = "/status"
	pathConfig            = "/config"
	pathValidatorRelayers = "/registeredrelays/{valpubkey}"

	// Memory endpoints: what the oracle knows
	pathMemoryValidators             = "/memory/validators"
	pathMemoryValidatorByIndex       = "/memory/validator/{valindex}"
	pathMemoryValidatorsByWithdrawal = "/memory/validators/{withdrawalAddress}"
	pathMemoryFeesInfo               = "/memory/feesinfo"
	pathMemorySubscriptions          = "/memory/subscriptions"   // TODO
	pathMemoryUnsubscriptions        = "/memory/unsubscriptions" // TODO
	pathMemoryAllBlocks              = "/memory/allblocks"
	pathMemoryProposedBlocks         = "/memory/proposedblocks"
	pathMemoryMissedBlocks           = "/memory/missedblocks"
	pathMemoryWrongFeeBlocks         = "/memory/wrongfeeblocks"
	pathMemoryDonations              = "/memory/donations"
	pathMemoryPoolStatistics         = "/memory/statistics"

	// Onchain endpoints: what is submitted to the contract
	pathOnchainValidators             = "/onchain/validators"                     // TODO
	pathOnchainValidatorByIndex       = "/onchain/validator/{valindex}"           // TODO
	pathOnchainValidatorsByWithdrawal = "/onchain/validators/{withdrawalAddress}" // TODO
	pathOnchainFeesInfo               = "/onchain/proof/fees"
	pathOnchainMerkleRoot             = "/onchain/merkleroot" // TODO:
	pathOnchainMerkleProof            = "/onchain/proof/{withdrawalAddress}"
	pathOnchainLatestCheckpoint       = "/onchain/latestcheckpoint" // TODO: needed?
)

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type httpOkStatus struct {
	IsConsensusInSync       bool   `json:"is_consensus_in_sync"`
	IsExecutionInSync       bool   `json:"is_execution_in_sync"`
	IsOracleInSync          bool   `json:"is_oracle_in_sync"`
	LatestProcessedSlot     uint64 `json:"latest_processed_slot"`
	LatestProcessedBlock    uint64 `json:"latest_processed_block"`
	LatestFinalizedEpoch    uint64 `json:"latest_finalized_epoch"`
	LatestFinalizedSlot     uint64 `json:"latest_finalized_slot"`
	OracleHeadDistance      uint64 `json:"oracle_sync_distance_slots"`
	NextCheckpointSlot      uint64 `json:"next_checkpoint_slot"`
	NextCheckpointTime      string `json:"next_checkpoint_time"`
	NextCheckpointRemaining string `json:"next_checkpoint_remaining"`
	PreviousCheckpointSlot  uint64 `json:"previous_checkpoint_slot"`
	PreviousCheckpointTime  string `json:"previous_checkpoint_time"`
	PreviousCheckpointAge   string `json:"previous_checkpoint_age"`
	ConsensusChainId        string `json:"consensus_chainid"`
	ExecutionChainId        string `json:"execution_chainid"`
	DepositContact          string `json:"depositcontract"`
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
	TotalSubscribed    uint64 `json:"total_subscribed_validators"`
	TotalActive        uint64 `json:"total_active_validators"`
	TotalYellowCard    uint64 `json:"total_yellowcard_validators"`
	TotalRedCard       uint64 `json:"total_redcard_validators"`
	TotalBanned        uint64 `json:"total_banned_validators"`
	TotalNotSubscribed uint64 `json:"total_notsubscribed_validators"`

	LatestCheckpointSlot       uint64   `json:"latest_checkpoint_slot"`
	NextCheckpointSlot         uint64   `json:"next_checkpoint_slot"`
	TotalAccumulatedRewardsWei *big.Int `json:"total_accumulated_rewards_wei"`
	TotalPendingRewaradsWei    *big.Int `json:"total_pending_rewards_wei"`

	TotalRewardsSentWei *big.Int `json:"total_rewards_sent_wei"`
	TotalDonationsWei   *big.Int `json:"total_donations_wei"`
	AvgBlockRewardWei   *big.Int `json:"avg_block_reward_wei"`

	// TODO: Split Proposed in Vanila/Mev
	//TotalVanilaBlocks   uint64
	//TotalMevBlocks      uint64
	TotalProposedBlocks uint64 `json:"total_proposed_blocks"`
	TotalMissedBlocks   uint64 `json:"total_missed_blocks"`
	TotalWrongFeeBlocks uint64 `json:"total_wrongfee_blocks"`
}

type httpOkValidatorState struct {
	ValidatorStatus       string   `json:"status"`
	AccumulatedRewardsWei *big.Int `json:"accumulated_rewards_wei"`
	PendingRewardsWei     *big.Int `json:"pending_rewards_wei"`
	CollateralWei         *big.Int `json:"collateral_rewards_wei"`
	WithdrawalAddress     string   `json:"withdrawal_address"`
	ValidatorIndex        uint64   `json:"validator_index"`
	ValidatorKey          string   `json:"validator_key"`
	//ValidatorProposedBlocks   []BlockState
	//ValidatorMissedBlocks     []BlockState
	//ValidatorWrongFeeBlocks   []BlockState

	// TODO: Include ClaimedSoFar from the smart contract for reconciliation
}

type httpOkProofs struct {
	LeafWithdrawalAddress      string   `json:"leaf_withdrawal_address"`
	LeafAccumulatedBalance     *big.Int `json:"leaf_accumulated_balance"`
	MerkleRoot                 string   `json:"merkleroot"`
	CheckpointSlot             uint64   `json:"checkpoint_slot"`
	Proofs                     []string `json:"merkle_proofs"`
	RegisteredValidators       []uint64 `json:"registered_validators"`
	TotalAccumulatedRewardsWei *big.Int `json:"total_accumulated_rewards_wei"`
	AlreadyClaimedRewardsWei   *big.Int `json:"already_claimed_rewards_wei"`
	ClaimableRewardsWei        *big.Int `json:"claimable_rewards_wei"`
	PendingRewardsWei          *big.Int `json:"pending_rewards_wei"`
}

type ApiService struct {
	srv           *http.Server
	config        *config.Config
	Onchain       *oracle.Onchain
	oracle        *oracle.Oracle
	ApiListenAddr string
	Network       string
}

func NewApiService(
	cfg *config.Config,
	oracle *oracle.Oracle,
	onchain *oracle.Onchain) *ApiService {

	return &ApiService{
		// TODO: configure, add cli flag
		ApiListenAddr: "0.0.0.0:7300",
		config:        cfg,
		oracle:        oracle,
		Onchain:       onchain,
		Network:       cfg.Network,
	}
}

func (m *ApiService) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := httpErrorResp{code, message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.WithField("response", resp).WithError(err).Error("Couldn't write error response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *ApiService) respondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithField("response", response).WithError(err).Error("Couldn't write OK response")
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (m *ApiService) getRouter() http.Handler {
	r := mux.NewRouter()

	// Map endpoints and their handlers
	r.HandleFunc("/", m.handleRoot).Methods(http.MethodGet)

	// General endpoints
	r.HandleFunc(pathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(pathConfig, m.handleConfig).Methods(http.MethodGet)
	r.HandleFunc(pathValidatorRelayers, m.handleValidatorRelayers).Methods(http.MethodGet)

	// Memory endpoints
	r.HandleFunc(pathMemoryValidators, m.handleMemoryValidators).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryValidatorByIndex, m.handleMemoryValidatorInfo).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryValidatorsByWithdrawal, m.handleMemoryValidatorsByWithdrawal).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryFeesInfo, m.handleMemoryFeesInfo).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryPoolStatistics, m.handleMemoryStatistics).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryAllBlocks, m.handleMemoryAllBlocks).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryProposedBlocks, m.handleMemoryProposedBlocks).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryMissedBlocks, m.handleMemoryMissedBlocks).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryWrongFeeBlocks, m.handleMemoryWrongFeeBlocks).Methods(http.MethodGet)
	r.HandleFunc(pathMemoryDonations, m.handleMemoryDonations).Methods(http.MethodGet)

	// Onchain endpoints
	r.HandleFunc(pathOnchainFeesInfo, m.handleOnchainFeesInfo).Methods(http.MethodGet)
	r.HandleFunc(pathOnchainMerkleProof, m.handleOnchainMerkleProof).Methods(http.MethodGet)

	//r.HandleFunc(pathLatestCheckpoint, m.handleLatestCheckpoint)

	//r.Use(mux.CORSMethodMiddleware(r))

	return r
}

func (m *ApiService) StartHTTPServer() error {
	log.Info("Starting HTTP server on ", m.ApiListenAddr)
	if m.srv != nil {
		return errors.New("server already running")
	}

	//go m.startBidCacheCleanupTask()

	m.srv = &http.Server{
		Addr:    m.ApiListenAddr,
		Handler: m.getRouter(),

		//ReadTimeout:       time.Duration(config.ServerReadTimeoutMs) * time.Millisecond,
		//ReadHeaderTimeout: time.Duration(config.ServerReadHeaderTimeoutMs) * time.Millisecond,
		//WriteTimeout:      time.Duration(config.ServerWriteTimeoutMs) * time.Millisecond,
		//IdleTimeout:       time.Duration(config.ServerIdleTimeoutMs) * time.Millisecond,

		//MaxHeaderBytes: config.ServerMaxHeaderBytes,
	}

	err := m.srv.ListenAndServe()
	if err != nil {
		log.Fatal("could not start http server: ", err)
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (m *ApiService) handleRoot(w http.ResponseWriter, req *http.Request) {
	m.respondOK(w, "see api doc for available endpoints")
}

func (m *ApiService) handleMemoryStatistics(w http.ResponseWriter, req *http.Request) {
	totalSubscribed := uint64(0)
	totalActive := uint64(0)
	totalYellowCard := uint64(0)
	totalRedCard := uint64(0)
	totalBanned := uint64(0)
	totalNotSubscribed := uint64(0)

	totalAccumulatedRewards := big.NewInt(0)
	totalPendingRewards := big.NewInt(0)

	// TODO: Would be nice to divice en MEV and non-MEV blocks
	//totalVanilaBlocks := 0
	//totalMevBlocks := 0

	for _, validator := range m.oracle.State.Validators {
		if validator.ValidatorStatus == oracle.Active {
			totalActive++
		} else if validator.ValidatorStatus == oracle.YellowCard {
			totalYellowCard++
		} else if validator.ValidatorStatus == oracle.RedCard {
			totalRedCard++
		} else if validator.ValidatorStatus == oracle.Banned {
			totalBanned++
		} else if validator.ValidatorStatus == oracle.NotSubscribed {
			totalNotSubscribed++
		}
		totalAccumulatedRewards.Add(totalAccumulatedRewards, validator.AccumulatedRewardsWei)
		totalPendingRewards.Add(totalPendingRewards, validator.PendingRewardsWei)
	}

	totalSubscribed = totalActive + totalYellowCard + totalRedCard

	totalRewardsSentWei := big.NewInt(0)
	for _, block := range m.oracle.State.ProposedBlocks {
		totalRewardsSentWei.Add(totalRewardsSentWei, block.Reward)
	}
	totalDonationsWei := big.NewInt(0)
	for _, donation := range m.oracle.State.Donations {
		totalDonationsWei.Add(totalDonationsWei, donation.AmountWei)
	}

	totalProposedBlocks := uint64(len(m.oracle.State.ProposedBlocks))
	avgBlockRewardWei := big.NewInt(0)

	// Avoid division by zero
	if totalProposedBlocks != 0 {
		avgBlockRewardWei = big.NewInt(0).Div(totalRewardsSentWei, big.NewInt(0).SetUint64(uint64(len(m.oracle.State.ProposedBlocks))))
	}

	m.respondOK(w, httpOkMemoryStatistics{
		TotalSubscribed:            totalSubscribed,
		TotalActive:                totalActive,
		TotalYellowCard:            totalYellowCard,
		TotalRedCard:               totalRedCard,
		TotalBanned:                totalBanned,
		TotalNotSubscribed:         totalNotSubscribed,
		LatestCheckpointSlot:       m.oracle.State.LatestProcessedSlot,                                       // This is wrong. TODO: convert date
		NextCheckpointSlot:         m.oracle.State.LatestProcessedSlot + m.Onchain.Cfg.CheckPointSizeInSlots, // TODO: Also wrong. convert to date
		TotalAccumulatedRewardsWei: totalAccumulatedRewards,
		TotalPendingRewaradsWei:    totalPendingRewards,
		TotalRewardsSentWei:        totalRewardsSentWei,
		TotalDonationsWei:          totalDonationsWei,
		AvgBlockRewardWei:          avgBlockRewardWei,
		TotalProposedBlocks:        totalProposedBlocks,
		TotalMissedBlocks:          uint64(len(m.oracle.State.MissedBlocks)),
		TotalWrongFeeBlocks:        uint64(len(m.oracle.State.WrongFeeBlocks)),
	})
}

func (m *ApiService) handleStatus(w http.ResponseWriter, req *http.Request) {
	chainId, err := m.Onchain.ExecutionClient.ChainID(context.Background())
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get exex chainid: "+err.Error())
	}

	depositContract, err := m.Onchain.ConsensusClient.DepositContract(context.Background())
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get deposit contract: "+err.Error())
	}

	execSync, err := m.Onchain.ExecutionClient.SyncProgress(context.Background())
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get exec sync progress: "+err.Error())
	}

	// Seems that if nil means its in sync
	execInSync := false
	if execSync == nil {
		execInSync = true
	}

	consSync, err := m.Onchain.ConsensusClient.NodeSyncing(context.Background())
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get consensus sync progress: "+err.Error())
	}

	// Allow some slots to avoid jitter
	consInSync := false
	if uint64(consSync.SyncDistance) < 2 {
		consInSync = true
	}

	finality, err := m.Onchain.ConsensusClient.Finality(context.Background(), "finalized")
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get consensus latest finalized slot: "+err.Error())
	}

	SlotsInEpoch := uint64(32)
	finalizedEpoch := uint64(finality.Finalized.Epoch)
	finalizedSlot := finalizedEpoch * SlotsInEpoch

	oracleSync := false
	if m.oracle.State.LatestProcessedSlot-finalizedSlot == 0 {
		oracleSync = true
	}

	// Slots that passed since last checkpoint
	slotsFromLastCheckpoint := m.oracle.State.LatestProcessedSlot % m.Onchain.Cfg.CheckPointSizeInSlots

	// Remaining slots till next checkpoint
	slotsTillNextCheckpoint := m.Onchain.Cfg.CheckPointSizeInSlots - slotsFromLastCheckpoint

	status := httpOkStatus{
		IsConsensusInSync:       consInSync,
		IsExecutionInSync:       execInSync,
		IsOracleInSync:          oracleSync,
		LatestProcessedSlot:     m.oracle.State.LatestProcessedSlot,
		LatestProcessedBlock:    m.oracle.State.LatestProcessedBlock,
		LatestFinalizedEpoch:    finalizedEpoch,
		LatestFinalizedSlot:     finalizedSlot,
		OracleHeadDistance:      finalizedSlot - m.oracle.State.LatestProcessedSlot,
		NextCheckpointSlot:      m.oracle.State.LatestProcessedSlot + slotsTillNextCheckpoint,
		NextCheckpointTime:      "", // TODO:
		NextCheckpointRemaining: SlotsToTime(slotsTillNextCheckpoint),
		PreviousCheckpointSlot:  0,  // TODO:
		PreviousCheckpointTime:  "", // TODO:
		PreviousCheckpointAge:   SlotsToTime(slotsFromLastCheckpoint),
		ExecutionChainId:        chainId.String(),
		ConsensusChainId:        strconv.FormatUint(depositContract.ChainID, 10),
		DepositContact:          "0x" + hex.EncodeToString(depositContract.Address[:]),
	}

	m.respondOK(w, status)
}

func (m *ApiService) handleConfig(w http.ResponseWriter, req *http.Request) {
	if m.config == nil {
		m.respondError(w, http.StatusInternalServerError, "no config loaded, nil value")
		return
	}
	m.respondOK(w, m.config)
}

func (m *ApiService) handleMemoryValidators(w http.ResponseWriter, req *http.Request) {
	// Perhaps a bit dangerours to access this directly without getters.
	m.respondOK(w, m.oracle.State.Validators)
}

func (m *ApiService) handleMemoryValidatorInfo(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	valIndexStr := vars["valindex"]
	valIndex, ok := IsValidIndex(valIndexStr)

	if !ok {
		m.respondError(w, http.StatusBadRequest, "invalid validator index: "+valIndexStr)
		return
	}

	validator, found := m.oracle.State.Validators[valIndex]
	if !found {
		m.respondError(w, http.StatusBadRequest, fmt.Sprint("could not find validator with index: ", valIndex))
		return
	}

	m.respondOK(w, validator)
}

func (m *ApiService) handleMemoryValidatorsByWithdrawal(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	withdrawalAddress := vars["withdrawalAddress"]

	// Move this to a function and require it for all api calls
	finality, err := m.Onchain.ConsensusClient.Finality(context.Background(), "finalized")
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get consensus latest finalized slot: "+err.Error())
	}

	SlotsInEpoch := uint64(32)
	finalizedSlot := uint64(finality.Finalized.Epoch) * SlotsInEpoch

	oracleInSync := false
	if m.oracle.State.LatestProcessedSlot-finalizedSlot == 0 {
		oracleInSync = true
	}

	if !oracleInSync {
		m.respondError(w, http.StatusInternalServerError, "oracle not in sync yet, try again later")
		return
	}

	// Use always lowercase
	withdrawalAddress = strings.ToLower(withdrawalAddress)

	if !IsValidAddress(withdrawalAddress) {
		m.respondError(w, http.StatusBadRequest, "invalid withdrawalAddress: "+withdrawalAddress)
		return
	}

	if m.Onchain.Validators() == nil {
		m.respondError(w, http.StatusInternalServerError, "finalized validators not loaded yet, try again later")
		return
	}

	// We return
	// 1) validators using this withdrawal address but not tracked by the oracle
	// 2) validators using this withdrawal address and tracked by the oracle (eg already subscribed)
	requestedValidators := make(map[uint64]*oracle.ValidatorInfo, 0)

	// 1) Get all onchain validators for that withdrawal address (untracked)
	for valIndex, validator := range m.Onchain.Validators() {

		// Check if the withdrawal address matches the requested one
		credStr := hex.EncodeToString(validator.Validator.WithdrawalCredentials)
		eth1Add, err := oracle.GetEth1Address(credStr)

		// Skip validators without non eth withdrawal address (bls address)
		if err != nil {
			continue
		}

		// Skip if the address does not match with the requested
		if !AreAddressEqual(eth1Add, withdrawalAddress) {
			continue
		}

		// Skip validators that cannot be subscribed
		if !oracle.CanValidatorSubscribeToPool(validator) {
			continue
		}

		requestedValidators[uint64(valIndex)] = &oracle.ValidatorInfo{
			ValidatorIndex:    uint64(validator.Index),
			WithdrawalAddress: eth1Add,
			ValidatorStatus:   oracle.Untracked,
			ValidatorKey:      "0x" + hex.EncodeToString(validator.Validator.PublicKey[:]),
		}
	}

	// 2) Get all tracked validators for that withdrawal address (tracked)
	for valIndex, validator := range m.oracle.State.Validators {
		// Just overwrite the untracked validators with oracle state
		if AreAddressEqual(validator.WithdrawalAddress, withdrawalAddress) {
			requestedValidators[valIndex] = validator
		}
	}

	// Now we apply the state transition to these validators, based on what we have seen
	// onchain since the latest finalized slot util head. This is neccesary because the
	// oracle runs all calculations on finalized states, but the api must report to the
	// users without this 15 minutes-ish delay.
	// This applies a non-finalized state to the validators, creating a virtual state
	// only used for the api.

	if m.oracle.State.LatestProcessedBlock == 0 {
		m.respondError(w, http.StatusInternalServerError, "latest processed block is 0, try again later")
		return
	}

	firstNotProcessedBlock := m.oracle.State.LatestProcessedBlock + 1

	// TODO: Cache this, very inneficient to get it every time
	allSubsTillHead, err := m.GetSubscriptionsTillHead(firstNotProcessedBlock)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get subscriptions: "+err.Error())
		return
	}
	allUnsubsTillHead, err := m.GetUnsubscriptionsTillHead(firstNotProcessedBlock)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get unsubscriptions: "+err.Error())
		return
	}

	// Apply latest seen events to the existing state. This is a "virtual" state, just for the api
	// so that users are aware of the latest events, without waiting for the next finalized state.
	m.ApplyNonFinalizedState(
		allSubsTillHead,
		allUnsubsTillHead,
		requestedValidators)

	m.respondOK(w, maps.Values(requestedValidators))
}

func (m *ApiService) handleMemoryFeesInfo(w http.ResponseWriter, req *http.Request) {
	type httpOkMemoryFeesInfo struct {
		PoolFeesPercent     int      `json:"pool_fee_percent"`
		PoolFeesAddress     string   `json:"pool_fee_address"`
		PoolAccumulatedFees *big.Int `json:"pool_accumulated_fees"`
	}

	m.respondOK(w, httpOkMemoryFeesInfo{
		PoolFeesPercent:     m.oracle.State.PoolFeesPercent,
		PoolFeesAddress:     m.oracle.State.PoolFeesAddress,
		PoolAccumulatedFees: m.oracle.State.PoolAccumulatedFees,
	})
}

func (m *ApiService) handleMemoryAllBlocks(w http.ResponseWriter, req *http.Request) {
	allBlocks := make([]oracle.Block, 0)

	// Concat all the blocks, order is not guaranteed
	allBlocks = append(allBlocks, m.oracle.State.ProposedBlocks...)
	allBlocks = append(allBlocks, m.oracle.State.MissedBlocks...)
	allBlocks = append(allBlocks, m.oracle.State.WrongFeeBlocks...)

	m.respondOK(w, allBlocks)
}

func (m *ApiService) handleMemoryProposedBlocks(w http.ResponseWriter, req *http.Request) {
	// TODO: Use getter, since its safer and dont make this fields public
	m.respondOK(w, m.oracle.State.ProposedBlocks)
}

func (m *ApiService) handleMemoryMissedBlocks(w http.ResponseWriter, req *http.Request) {
	// TODO: Use getter, since its safer and dont make this fields public
	m.respondOK(w, m.oracle.State.MissedBlocks)
}

func (m *ApiService) handleMemoryWrongFeeBlocks(w http.ResponseWriter, req *http.Request) {
	// TODO: Use getter, since its safer and dont make this fields public
	m.respondOK(w, m.oracle.State.WrongFeeBlocks)
}

func (m *ApiService) handleMemoryDonations(w http.ResponseWriter, req *http.Request) {
	// TODO: Use getter, since its safer and dont make this fields public
	m.respondOK(w, m.oracle.State.Donations)
}

func (m *ApiService) handleOnchainFeesInfo(w http.ResponseWriter, req *http.Request) {

	// Get the merkle root stored onchain
	// TODO: Temporally disabled until we enable submitting state to chain
	/*contractRoot, err := m.Onchain.GetContractMerkleRoot(apiRetryOpts...)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get contract merkle root: "+err.Error())
		return
	}

	if contractRoot == defaultMerkleRoot {
		m.respondError(w, http.StatusInternalServerError, "contract merkle root is default, no state was commited yet")
		return
	}

	if strings.ToLower(contractRoot) != strings.ToLower(m.OracleState.LatestCommitedState.MerkleRoot) {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprint("contract merkle root is not in sync with oracle state: ",
			contractRoot, " vs ", m.OracleState.LatestCommitedState.MerkleRoot))
		return
	}*/

	if len(m.oracle.State.LatestCommitedState.Proofs) == 0 {
		m.respondError(w, http.StatusInternalServerError, "no proofs found: not in sync or nothing commited yet")
		return
	}

	// TODO: Use always lowercase. This is a bit of a workaround
	poolFeesAddress := strings.ToLower(m.oracle.State.PoolFeesAddress)

	proofs, okProof := m.oracle.State.LatestCommitedState.Proofs[poolFeesAddress]
	if !okProof {
		m.respondError(w, http.StatusInternalServerError, "no proof found for pool fees address, perhaps not commited yet")
		return
	}

	leaf, okLeaf := m.oracle.State.LatestCommitedState.Leafs[poolFeesAddress]
	if !okLeaf {
		m.respondError(w, http.StatusInternalServerError, "no leaf found for pool fees address, perhaps not commited yet")
		return
	}

	type httpOkProofsFee struct {
		LeafWithdrawalAddress  string   `json:"leaf_withdrawal_address"`
		LeafAccumulatedBalance *big.Int `json:"leaf_accumulated_balance"`
		MerkleRoot             string   `json:"merkleroot"`
		Proofs                 []string `json:"merkle_proofs"`
	}

	m.respondOK(w, httpOkProofsFee{
		LeafWithdrawalAddress:  leaf.WithdrawalAddress,
		LeafAccumulatedBalance: leaf.AccumulatedBalance,
		MerkleRoot:             m.oracle.State.LatestCommitedState.MerkleRoot,
		Proofs:                 proofs,
	})
}

func (m *ApiService) handleOnchainMerkleProof(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	withdrawalAddress := vars["withdrawalAddress"]

	if !IsValidAddress(withdrawalAddress) {
		m.respondError(w, http.StatusBadRequest, "invalid WithdrawalAddress: "+withdrawalAddress)
		return
	}

	// Use always lowercase
	withdrawalAddress = strings.ToLower(withdrawalAddress)

	// Get the merkle root stored onchain
	// TODO: Temporally disabled until we enable submitting state to chain

	/*contractRoot, err := m.Onchain.GetContractMerkleRoot(apiRetryOpts...)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get contract merkle root: "+err.Error())
		return
	}

	if contractRoot == defaultMerkleRoot {
		m.respondError(w, http.StatusInternalServerError, "contract merkle root is default, no state was commited yet")
		return
	}

	if strings.ToLower(contractRoot) != strings.ToLower(m.OracleState.LatestCommitedState.MerkleRoot) {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprint("contract merkle root is not in sync with oracle state: ",
			contractRoot, " vs ", m.OracleState.LatestCommitedState.MerkleRoot))
		return
	}*/

	// Get the proofs of this withdrawal address (to be used onchain to claim rewards)
	proofs, proofFound := m.oracle.State.LatestCommitedState.Proofs[withdrawalAddress]
	if !proofFound {
		m.respondError(w, http.StatusBadRequest, "could not find proof for WithdrawalAddress: "+withdrawalAddress)
		return
	}

	// Get the leafs of this withdrawal address (to be used onchain to claim rewards)
	leafs, leafsFound := m.oracle.State.LatestCommitedState.Leafs[withdrawalAddress]
	if !leafsFound {
		m.respondError(w, http.StatusBadRequest, "could not find leafs for WithdrawalAddress: "+withdrawalAddress)
		return
	}

	// Get validators that are registered to this withdrawal address in the pool
	registeredValidators := make([]uint64, 0)
	for valIndex, validator := range m.oracle.State.LatestCommitedState.Validators {
		if strings.ToLower(validator.WithdrawalAddress) == strings.ToLower(withdrawalAddress) {
			registeredValidators = append(registeredValidators, valIndex)
		}
	}

	claimed, err := m.Onchain.GetContractClaimedBalance(withdrawalAddress, apiRetryOpts...)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get claimed balance so far from contract: "+err.Error())
		return
	}

	totalPending := big.NewInt(0)

	for _, validator := range m.oracle.State.LatestCommitedState.Validators {
		if strings.ToLower(validator.WithdrawalAddress) == strings.ToLower(withdrawalAddress) {
			totalPending.Add(totalPending, validator.PendingRewardsWei)
		}
	}

	m.respondOK(w, httpOkProofs{
		LeafWithdrawalAddress:      leafs.WithdrawalAddress,
		LeafAccumulatedBalance:     leafs.AccumulatedBalance,
		MerkleRoot:                 m.oracle.State.LatestCommitedState.MerkleRoot,
		CheckpointSlot:             m.oracle.State.LatestCommitedState.Slot,
		Proofs:                     proofs,
		RegisteredValidators:       registeredValidators,
		TotalAccumulatedRewardsWei: leafs.AccumulatedBalance,
		ClaimableRewardsWei:        new(big.Int).Sub(leafs.AccumulatedBalance, claimed),
		AlreadyClaimedRewardsWei:   claimed,
		PendingRewardsWei:          totalPending,
	})
}

func (m *ApiService) handleLatestMerkleRoot(w http.ResponseWriter, req *http.Request) {
	// This is the latest merkle root tracked from the oracle.
	//oracleMerkleRoot := "0x" + m.OracleState.LatestCommitedState.MerkleRoot

	contractMerkleRoot, err := m.Onchain.GetContractMerkleRoot(apiRetryOpts...)
	if err != nil {
		m.respondError(w, http.StatusBadRequest, "could not get latest merkle root from chain")
		return
	}
	m.respondOK(w, httpOkMerkleRoot{
		MerkleRoot: contractMerkleRoot,
	})
}

func (m *ApiService) handleValidatorOnchainStateByIndex(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	valIndex, err := strconv.ParseUint(vars["valindex"], 10, 64)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not parse valIndex: "+err.Error())
		return
	}

	// We look into the LatestCommitedState, since its whats its onchain
	valState, found := m.oracle.State.LatestCommitedState.Validators[uint64(valIndex)]
	if !found {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("validator index not tracked in the oracle: %d", valIndex))
		return
	}
	m.respondOK(w, httpOkValidatorState{
		ValidatorStatus:       oracle.ValidatorStateToString(valState.ValidatorStatus),
		AccumulatedRewardsWei: valState.AccumulatedRewardsWei,
		PendingRewardsWei:     valState.PendingRewardsWei,
		CollateralWei:         valState.CollateralWei,
		WithdrawalAddress:     valState.WithdrawalAddress,
		ValidatorIndex:        valState.ValidatorIndex,
		ValidatorKey:          valState.ValidatorKey,
		// TODO: Missing blocks fields
	})
}

func (m *ApiService) handleValidatorRelayers(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	valPubKey := vars["valpubkey"]
	if !IsValidPubkey(valPubKey) {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("invalid validator pubkey format"))
		return
	}
	var correctFeeRelays []httpRelay
	var wrongFeeRelays []httpRelay
	var unregisteredRelays []httpRelay
	registeredCorrectFee := false
	var relays []string

	if m.Network == "mainnet" {
		relays = config.MainnetRelays
	} else if m.Network == "goerli" {
		relays = config.GoerliRelays
	} else {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("invalid network: %s", m.Network))
		return
	}

	for _, relay := range relays {
		url := fmt.Sprintf("https://%s/relay/v1/data/validator_registration?pubkey=%s", relay, valPubKey)
		resp, err := http.Get(url)
		if err != nil {
			m.respondError(w, http.StatusInternalServerError, "could not call relayer endpoint: "+err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			signedRegistration := &types.SignedValidatorRegistration{}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				m.respondError(w, http.StatusInternalServerError, "could not call relayer endpoint: "+err.Error())
				return
			}

			if err = json.Unmarshal(bodyBytes, signedRegistration); err != nil {
				m.respondError(w, http.StatusInternalServerError, "could not call relayer endpoint: "+err.Error())
				return
			}

			relayRegistration := httpRelay{
				RelayAddress: relay,
				FeeRecipient: signedRegistration.Message.FeeRecipient.String(),
				Timestamp:    fmt.Sprintf("%s", time.Unix(int64(signedRegistration.Message.Timestamp), 0)),
			}

			if strings.ToLower(signedRegistration.Message.FeeRecipient.String()) == strings.ToLower(m.Onchain.Cfg.PoolAddress) {
				correctFeeRelays = append(correctFeeRelays, relayRegistration)
			} else {
				wrongFeeRelays = append(wrongFeeRelays, relayRegistration)
			}
		} else {
			unregisteredRelays = append(unregisteredRelays, httpRelay{
				RelayAddress: relay,
			})
		}
	}

	// Only if there are some correct registrations and no invalid ones, its ok
	if len(wrongFeeRelays) == 0 && len(correctFeeRelays) > 0 {
		registeredCorrectFee = true
	}

	m.respondOK(w, httpOkRelayersState{
		CorrectFeeRecipients: registeredCorrectFee,
		CorrectFeeRelays:     correctFeeRelays,
		WrongFeeRelays:       wrongFeeRelays,
		UnregisteredRelays:   unregisteredRelays,
	})
}

func IsValidIndex(v string) (uint64, bool) {
	//re := regexp.MustCompile("^[0-9]+$")
	val, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func IsValidAddress(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(v)
}

func IsValidPubkey(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-f]{96}$")
	return re.MatchString(v)
}

// Copied from oracle/utils. Cant import due to circular dependency
// TODO: Move to utils package
func AreAddressEqual(address1 string, address2 string) bool {
	if len(address1) != len(address2) {
		log.Fatal("address length mismatch: ",
			"add1: ", address1,
			"add2: ", address2)
	}
	if strings.ToLower(address1) == strings.ToLower(address2) {
		return true
	}
	return false
}

// TODO: unsure if move this somewhere else
func (m *ApiService) GetSubscriptionsTillHead(latestProcessedBlock uint64) ([]oracle.Subscription, error) {
	// TODO: add check here to ensure its a reasonable amount of blocks. should be around 15-20 minutes in blocks
	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: latestProcessedBlock, End: nil}

	// Note that this event can be both donations and mev rewards
	itrSubs, err := m.Onchain.Contract.FilterSubscribeValidator(filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "could not subscribe to validator events")
	}

	// Loop over all found events. Super inneficient. just Proof of concept
	blockSubscriptions := make([]oracle.Subscription, 0)
	for itrSubs.Next() {
		sub := oracle.Subscription{
			Event:     itrSubs.Event,
			Validator: m.Onchain.Validators()[phase0.ValidatorIndex(itrSubs.Event.ValidatorID)],
		}
		blockSubscriptions = append(blockSubscriptions, sub)
	}
	err = itrSubs.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not close subscription iterator")
	}
	return blockSubscriptions, nil
}

func (m *ApiService) GetUnsubscriptionsTillHead(latestProcessedBlock uint64) ([]oracle.Unsubscription, error) {
	// TODO: add check here to ensure its a reasonable amount of blocks. should be around 15-20 minutes in blocks
	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: latestProcessedBlock, End: nil}
	// Note that this event can be both donations and mev rewards
	itrUnsubs, err := m.Onchain.Contract.FilterUnsubscribeValidator(filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "could not subscribe to validator events")
	}

	// Loop over all found events, TODO: inneficient. only finter events of this validator.
	blockUnsubscriptions := make([]oracle.Unsubscription, 0)
	for itrUnsubs.Next() {
		unsub := oracle.Unsubscription{
			Event:     itrUnsubs.Event,
			Validator: m.Onchain.Validators()[phase0.ValidatorIndex(itrUnsubs.Event.ValidatorID)],
		}
		blockUnsubscriptions = append(blockUnsubscriptions, unsub)
	}
	err = itrUnsubs.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not close subscription iterator")
	}
	return blockUnsubscriptions, nil
}

func (m *ApiService) ApplyNonFinalizedState(
	subs []oracle.Subscription,
	unsubs []oracle.Unsubscription,
	validators map[uint64]*oracle.ValidatorInfo) {

	eventsBlocksList := make([]uint64, 0)

	for _, sub := range subs {
		block := sub.Event.Raw.BlockNumber
		found := false
		for _, b := range eventsBlocksList {
			if b == block {
				found = true
			}
		}
		if !found {
			eventsBlocksList = append(eventsBlocksList, block)
		}
	}
	for _, unsub := range unsubs {
		block := unsub.Event.Raw.BlockNumber
		found := false
		for _, b := range eventsBlocksList {
			if b == block {
				found = true
			}
		}
		if !found {
			eventsBlocksList = append(eventsBlocksList, block)
		}
	}

	sort.Slice(eventsBlocksList, func(i, j int) bool { return eventsBlocksList[i] < eventsBlocksList[j] })

	for _, block := range eventsBlocksList {
		blockSub := GetSubInBlock(subs, block)
		blockUnsub := GetUnsubInBlock(unsubs, block)

		for _, subInBlock := range blockSub {
			valIndex := subInBlock.Event.ValidatorID
			val, found := validators[valIndex]
			if found {
				valWithdrawalAddress := val.WithdrawalAddress
				eventAddress := subInBlock.Event.Sender.String()
				if AreAddressEqual(valWithdrawalAddress, eventAddress) {
					if subInBlock.Event.SubscriptionCollateral.Cmp(m.config.CollateralInWei) >= 0 {
						if oracle.CanValidatorSubscribeToPool(subInBlock.Validator) {
							if val.ValidatorStatus == oracle.Untracked || val.ValidatorStatus == oracle.NotSubscribed {
								validators[valIndex].ValidatorStatus = oracle.Active
							}
						}
					}
				}
			}
		}

		for _, unsubInBlock := range blockUnsub {
			valIndex := unsubInBlock.Event.ValidatorID
			val, found := validators[valIndex]
			if found {
				valWithdrawalAddress := val.WithdrawalAddress
				eventAddress := unsubInBlock.Event.Sender.String()
				if AreAddressEqual(valWithdrawalAddress, eventAddress) {
					if val.ValidatorStatus == oracle.Active ||
						val.ValidatorStatus == oracle.YellowCard ||
						val.ValidatorStatus == oracle.RedCard {
						validators[valIndex].ValidatorStatus = oracle.NotSubscribed
					}
				}
			}
		}
	}
}

func GetSubInBlock(subs []oracle.Subscription, block uint64) []oracle.Subscription {
	filteredSubs := make([]oracle.Subscription, 0)
	for _, sub := range subs {
		if sub.Event.Raw.BlockNumber == block {
			filteredSubs = append(filteredSubs, sub)
		}
	}
	return filteredSubs
}

func GetUnsubInBlock(subs []oracle.Unsubscription, block uint64) []oracle.Unsubscription {
	filteredUnsubs := make([]oracle.Unsubscription, 0)
	for _, unsub := range subs {
		if unsub.Event.Raw.BlockNumber == block {
			filteredUnsubs = append(filteredUnsubs, unsub)
		}
	}
	return filteredUnsubs
}

// Do not use this, just as a proof of concept
func DeepCopy(src, dist interface{}) (err error) {
	buf := bytes.Buffer{}
	if err = gob.NewEncoder(&buf).Encode(src); err != nil {
		return
	}
	return gob.NewDecoder(&buf).Decode(dist)
}

// TODO: Duplicated, move to utils and take it from there
// Converts from slots to readable time (eg 1 day 9 hours 20 minutes)
func SlotsToTime(slots uint64) string {
	// Hardcoded. Mainnet Ethereum configuration
	SecondsInSlot := uint64(12)

	timeduration := time.Duration(slots*SecondsInSlot) * time.Second
	strDuration := durafmt.Parse(timeduration).String()

	return strDuration
}
