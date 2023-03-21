package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/avast/retry-go/v4"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/dappnode/mev-sp-oracle/oracle"
	"github.com/dappnode/mev-sp-oracle/postgres"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// These are the retry options when an api call involves external call to the beacon node
// or execution client. The idea is to try once, and fail fast.
// Use this for all onchain calls, otherwise defaultRetryOpts will be aplied
var apiRetryOpts = []retry.Option{
	retry.Attempts(1),
}

const (
	// Available endpoints

	pathStatus                = "/status"
	pathLatestCheckpoint      = "/latestcheckpoint"
	pathLatestMerkleProof     = "/proof/{depositaddress}"
	pathLatestMerkleRoot      = "/merkleroot"
	pathDepositAddressByIndex = "/depositadddress/{valindex}"
	pathPoolStatistics        = "/poolstatistics"
	pathDonations             = "/donations"
	pathValidatorRelayers     = "/registeredrelays/{valpubkey}"

	// TODO: better valindex=xxx

	// TODO: Perhaps rethink this a bit. There are two types of state:
	// - The state that the oracle knows of
	// - The state that is already submitted onchain
	pathValidatorOnchainStateByIndex  = "/validatoronchainstate/{valindex}"
	pathValidatorOffchainStateByIndex = "/validatoroffchainstate/{valindex}"

	// TODO: Get all validators for a deposit address

	pathValidatorStateByDeposit = ""

	// TODO: Fees generated (list claimable of fee account)

	// TODO:
	// proof
	// missed block, proposed blocks ok, etc.
)

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type httpOkStatus struct {
	IsConsensusInSync         bool   `json:"is_consensus_in_sync"`
	IsExecutionInSync         bool   `json:"is_execution_in_sync"`
	OracleLatestProcessedSlot uint64 `json:"oracle_latest_processed_slot"`
	ChainFinalizedSlot        uint64 `json:"chain_head_slot"`
	OracleHeadDistance        uint64 `json:"oracle_head_distance"`
	ChainId                   string `json:"chainid"`
	DepositContact            string `json:"depositcontract"`
}

type httpOkLatestCheckpoint struct {
	MerkleRoot     string `json:"merkleroot"`
	CheckpointSlot uint64 `json:"checkpointslot"`
}

type httpOkDepositAddress struct {
	DepositAddress   string `json:"deposit_address"`
	ValidatorIndex   uint64 `json:"validator_index"`
	ValidatorAddress string `json:"validator_address"`
}

type httpOkMerkleRoot struct {
	MerkleRoot string `json:"merkle_root"`
}

type httpOkDonations struct {
	DonationAmountWei []*big.Int `json:"donation_amount_wei"`
	DonationBlock     []uint64   `json:"donation_block"`
	DonationTxHash    []string   `json:"donation_tx_hash"`
}

type httpOkPoolStatatistics struct {
	OnchainMerkleRoot  string `json:"onchain_merkle_root"`
	TotalSubscribed    uint64 `json:"total_subscribed_validators"`
	TotalActive        uint64 `json:"total_active_validators"`
	TotalYellowCard    uint64 `json:"total_yellowcard_validators"`
	TotalRedCard       uint64 `json:"total_redcard_validators"`
	TotalBanned        uint64 `json:"total_banned_validators"`
	TotalNotSubscribed uint64 `json:"total_notsubscribed_validators"`

	LatestCheckpointSlot uint64 `json:"latest_checkpoint_slot"`
	NextCheckpointSlot   uint64 `json:"next_checkpoint_slot"`

	// TODO: Clarify if wei or gwei and add to name
	TotalAccumulatedRewards *big.Int `json:"total_accumulated_rewards"`
	// TODO: Clarify if wei or gwei and add to name
	TotalPendingRewarads *big.Int `json:"total_pending_rewards"`

	// TODO: Split Proposed in Vanila/Mev
	//TotalVanilaBlocks   uint64
	//TotalMevBlocks      uint64
	TotalProposedBlocks uint64 `json:"total_proposed_blocks"`
	TotalMissedBlocks   uint64 `json:"total_missed_blocks"`
	TotalWrongFeeBlocks uint64 `json:"total_wrongfee_blocks"`
}

type httpOkValidatorState struct {
	StatusType            string   `json:"statustype"` // TODO: populate
	ValidatorStatus       string   `json:"validatorstatus"`
	AccumulatedRewardsWei *big.Int `json:"accumulated_rewards_wei"`
	PendingRewardsWei     *big.Int `json:"pending_rewards_wei"`
	CollateralWei         *big.Int `json:"collateral_rewards_wei"` // TODO: unsure if its we or gwei
	DepositAddress        string   `json:"deposit_address"`
	ValidatorIndex        string   `json:"validator_index"`
	ValidatorKey          string   `json:"validator_key"`
	//ProposedBlocksSlots   []BlockState
	//MissedBlocksSlots     []BlockState
	//WrongFeeBlocksSlots   []BlockState

	// TODO: Include ClaimedSoFar from the smart contract for reconciliation
}

type httpOkProofs struct {
	LeafDepositAddress     string   `json:"leaf_deposit_address"`
	LeafAccumulatedBalance *big.Int `json:"leaf_accumulated_balance"`
	MerkleRoot             string   `json:"merkleroot"`
	CheckpointSlot         uint64   `json:"checkpoint_slot"`
	Proofs                 []string `json:"merkle_proofs"`
	RegisteredValidators   []uint64 `json:"registered_validators"`
}

type httpOkRelayersState struct {
	Registered           bool     `json:"registered"`
	RegisteredRelayers   []string `json:"registered_relayers"`
	UnRegisteredRelayers []string `json:"unregistered_relayers"`
}

type ApiService struct {
	srv           *http.Server
	Postgres      *postgres.Postgresql
	OracleState   *oracle.OracleState
	Onchain       *oracle.Onchain
	ApiListenAddr string
	Network       string
}

func NewApiService(cfg config.Config, state *oracle.OracleState, onchain *oracle.Onchain) *ApiService {
	postgres, err := postgres.New(cfg.PostgresEndpoint)
	if err != nil {
		// TODO: Return error instead of fatal
		log.Fatal(err)
	}

	return &ApiService{
		// TODO: configure, add cli flag
		ApiListenAddr: "0.0.0.0:7300",
		Postgres:      postgres,
		OracleState:   state,
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
	r.HandleFunc(pathStatus, m.handleStatus).Methods(http.MethodGet)
	r.HandleFunc(pathLatestMerkleProof, m.handleLatestMerkleProof)
	r.HandleFunc(pathLatestMerkleRoot, m.handleLatestMerkleRoot)
	r.HandleFunc(pathLatestCheckpoint, m.handleLatestCheckpoint)
	r.HandleFunc(pathDepositAddressByIndex, m.handleDepositAddressByIndex)
	r.HandleFunc(pathValidatorRelayers, m.handleValidatorRelayers).Methods(http.MethodGet)

	r.HandleFunc(pathValidatorOnchainStateByIndex, m.handleValidatorOnchainStateByIndex)
	r.HandleFunc(pathValidatorOffchainStateByIndex, m.handleValidatorOffchainStateByIndex)
	r.HandleFunc(pathPoolStatistics, m.handlePoolStatistics)
	r.HandleFunc(pathDonations, m.handleDonations)

	//r.Use(mux.CORSMethodMiddleware(r))

	// TODO: Add logging
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
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (m *ApiService) handleRoot(w http.ResponseWriter, req *http.Request) {
	m.respondOK(w, "see api doc for available endpoints")
}

func (m *ApiService) handlePoolStatistics(w http.ResponseWriter, req *http.Request) {
	// TODO: Compare they are equal. They will be unless the oracle is restarted and
	// has to catch up
	//oracleMerkleRoot := "0x" + m.OracleState.LatestCommitedState.MerkleRoot

	contractMerkleRoot, err := m.Onchain.GetContractMerkleRoot(apiRetryOpts...)
	if err != nil {
		m.respondError(w, http.StatusBadRequest, "could not get latest merkle root from chain")
		return
	}

	totalSubscribed := uint64(0)
	totalActive := uint64(0)
	totalYellowCard := uint64(0)
	totalRedCard := uint64(0)
	totalBanned := uint64(0)
	totalNotSubscribed := uint64(0)

	totalAccumulatedRewards := big.NewInt(0)
	totalPendingRewards := big.NewInt(0)

	//totalVanilaBlocks := 0
	//totalMevBlocks := 0
	totalProposedBlocks := uint64(0)
	totalMissedBlocks := uint64(0)
	totalWrongFeeBlocks := uint64(0)

	// Unsure if its better to return the latest commited or the oracle state
	for _, validator := range m.OracleState.LatestCommitedState.Validators {
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

		totalProposedBlocks += uint64(len(validator.ProposedBlocksSlots))
		totalMissedBlocks += uint64(len(validator.MissedBlocksSlots))
		totalWrongFeeBlocks += uint64(len(validator.WrongFeeBlocksSlots))
	}

	totalSubscribed = totalActive + totalYellowCard + totalRedCard

	m.respondOK(w, httpOkPoolStatatistics{
		OnchainMerkleRoot:  contractMerkleRoot,
		TotalSubscribed:    totalSubscribed,
		TotalActive:        totalActive,
		TotalYellowCard:    totalYellowCard,
		TotalRedCard:       totalRedCard,
		TotalBanned:        totalBanned,
		TotalNotSubscribed: totalNotSubscribed,

		LatestCheckpointSlot: m.OracleState.LatestSlot,
		NextCheckpointSlot:   m.OracleState.LatestSlot + m.Onchain.Cfg.CheckPointSizeInSlots,

		TotalAccumulatedRewards: totalAccumulatedRewards,
		TotalPendingRewarads:    totalPendingRewards,

		TotalProposedBlocks: totalProposedBlocks,
		TotalMissedBlocks:   totalMissedBlocks,
		TotalWrongFeeBlocks: totalWrongFeeBlocks,
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

	status := httpOkStatus{
		IsConsensusInSync:         consInSync,
		IsExecutionInSync:         execInSync,
		OracleLatestProcessedSlot: m.OracleState.LatestSlot,
		ChainFinalizedSlot:        finalizedSlot,
		OracleHeadDistance:        finalizedSlot - m.OracleState.LatestSlot,
		ChainId:                   chainId.String(),
		DepositContact:            depositContract.String(),
	}

	m.respondOK(w, status)
}

func (m *ApiService) handleLatestMerkleProof(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	depositAddress := vars["depositaddress"]

	if !IsValidAddress(depositAddress) {
		m.respondError(w, http.StatusBadRequest, "invalid depositAddress: "+depositAddress)
		return
	}

	// Use always lowercase
	depositAddress = strings.ToLower(depositAddress)

	// Get the proofs of this deposit address (to be used onchain to claim rewards)
	proofs, proofFound := m.OracleState.LatestCommitedState.Proofs[depositAddress]
	if !proofFound {
		m.respondError(w, http.StatusBadRequest, "could not find proof for depositAddress: "+depositAddress)
		return
	}

	// Get the leafs of this deposit address (to be used onchain to claim rewards)
	leafs, leafsFound := m.OracleState.LatestCommitedState.Leafs[depositAddress]
	if !leafsFound {
		m.respondError(w, http.StatusBadRequest, "could not find leafs for depositAddress: "+depositAddress)
		return
	}

	// Get validators that are registered to this deposit address in the pool
	registeredValidators := make([]uint64, 0)
	for valIndex, validator := range m.OracleState.LatestCommitedState.Validators {
		if validator.DepositAddress == depositAddress {
			registeredValidators = append(registeredValidators, valIndex)
		}
	}

	m.respondOK(w, httpOkProofs{
		LeafDepositAddress:     leafs.DepositAddress,
		LeafAccumulatedBalance: leafs.AccumulatedBalance,
		MerkleRoot:             m.OracleState.LatestCommitedState.MerkleRoot,
		CheckpointSlot:         m.OracleState.LatestCommitedState.Slot,
		Proofs:                 proofs,
		RegisteredValidators:   registeredValidators,
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

func (m *ApiService) handleDonations(w http.ResponseWriter, req *http.Request) {
	httpOkDonations := httpOkDonations{}
	for _, donation := range m.OracleState.Donations {
		httpOkDonations.DonationAmountWei = append(httpOkDonations.DonationAmountWei, donation.AmountWei)
		httpOkDonations.DonationBlock = append(httpOkDonations.DonationBlock, donation.Block)
		httpOkDonations.DonationTxHash = append(httpOkDonations.DonationTxHash, donation.TxHash)
	}
	m.respondOK(w, httpOkDonations)
}

func (m *ApiService) handleDepositAddressByIndex(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	valIndex, err := strconv.ParseUint(vars["valindex"], 10, 64)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not parse valIndex: "+err.Error())
		return
	}

	valInfo, err := m.Onchain.ConsensusClient.Validators(context.Background(), "finalized", []phase0.ValidatorIndex{phase0.ValidatorIndex(valIndex)})
	valPubKeyByte := valInfo[phase0.ValidatorIndex(valIndex)].Validator.PublicKey
	valPubKeyStr := "0x" + hex.EncodeToString(valPubKeyByte[:])

	depositAddress, err := m.Postgres.GetDepositAddressOfValidatorKey(valPubKeyStr)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not get deposit address for valindex: "+err.Error())
		return
	}

	m.respondOK(w, httpOkDepositAddress{
		DepositAddress:   depositAddress,
		ValidatorIndex:   valIndex,
		ValidatorAddress: valPubKeyStr,
	})
}

func (m *ApiService) handleLatestCheckpoint(w http.ResponseWriter, req *http.Request) {
	log.Info("/latestCheckpoint")

	mRoot, slot, err := m.Postgres.GetLatestCheckpoint()
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	m.respondOK(w, httpOkLatestCheckpoint{mRoot, slot})
}

func (m *ApiService) handleValidatorOnchainStateByIndex(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	valIndex, err := strconv.ParseUint(vars["valindex"], 10, 64)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not parse valIndex: "+err.Error())
		return
	}

	// We look into the LatestCommitedState, since its whats its onchain
	valState, found := m.OracleState.LatestCommitedState.Validators[uint64(valIndex)]
	if !found {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("validator index not tracked in the oracle: %d", valIndex))
		return
	}
	m.respondOK(w, httpOkValidatorState{
		ValidatorStatus:       oracle.ValidatorStateToString(valState.ValidatorStatus),
		AccumulatedRewardsWei: valState.AccumulatedRewardsWei,
		PendingRewardsWei:     valState.PendingRewardsWei,
		CollateralWei:         valState.CollateralWei,
		DepositAddress:        valState.DepositAddress,
		ValidatorIndex:        valState.ValidatorIndex,
		ValidatorKey:          valState.ValidatorKey,
		// TODO: Missing blocks fields
	})
}

func (m *ApiService) handleValidatorOffchainStateByIndex(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	valIndex, err := strconv.ParseUint(vars["valindex"], 10, 64)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, "could not parse valIndex: "+err.Error())
		return
	}

	// We look into the local state. This can contain data that the oracle tracks but that its not
	// yet published onchain
	valState, found := m.OracleState.Validators[uint64(valIndex)]
	if !found {
		m.respondError(w, http.StatusInternalServerError, fmt.Sprintf("validator index not tracked in the oracle: %d", valIndex))
		return
	}
	m.respondOK(w, httpOkValidatorState{
		ValidatorStatus:       oracle.ValidatorStateToString(valState.ValidatorStatus),
		AccumulatedRewardsWei: valState.AccumulatedRewardsWei,
		PendingRewardsWei:     valState.PendingRewardsWei,
		CollateralWei:         valState.CollateralWei,
		DepositAddress:        valState.DepositAddress,
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
	var registeredRelays []string
	var unregisteredRelays []string
	registered := false
	var relays []string

	if m.Network == "mainnet" {
		relays = config.MainRelays

	} else {
		relays = config.TestRelays
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
			registeredRelays = append(registeredRelays, relay)
		} else {
			unregisteredRelays = append(unregisteredRelays, relay)
		}
	}

	if len(registeredRelays) != 0 {
		registered = true
	}

	m.respondOK(w, httpOkRelayersState{
		Registered:           registered,
		RegisteredRelayers:   registeredRelays,
		UnRegisteredRelayers: unregisteredRelays,
	})
}

func IsValidAddress(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(v)
}

func IsValidPubkey(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-f]{96}$")
	return re.MatchString(v)
}
