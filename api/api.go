package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"
	"mev-sp-oracle/postgres"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	// Available endpoints

	pathStatus            = "/status"
	pathLatestMerkleProof = "/oracle/merkleproof/depositaddress/{depositaddress}" // TODO: validate with some regex
)

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type httpOkProofs struct {
	DepositAddress   string   `json:"depositaddress"`
	MerkleRoot       string   `json:"merkleroot"`
	CheckpointSlot   uint64   `json:"checkpointslot"`
	Proofs           []string `json:"proofs"`
	AvailableBalance string   `json:"availablebalance"`
	UnbanBalance     string   `json:"unbanbalance"`
}

type ApiService struct {
	srv           *http.Server
	Postgres      *postgres.Postgresql
	ApiListenAddr string
}

func NewApiService(cfg config.Config) *ApiService {
	postgres, err := postgres.New(cfg.PostgresEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	return &ApiService{
		// TODO: configure, add cli flag
		ApiListenAddr: "0.0.0.0:7300",
		Postgres:      postgres,
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
	m.respondOK(w, "")
}

func (m *ApiService) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("some-header", "some info")
	errorTODO := false
	if errorTODO {
		m.respondError(w, http.StatusServiceUnavailable, "todo")
	} else {
		m.respondOK(w, "TODO: ok")
	}
}

func (m *ApiService) handleLatestMerkleProof(w http.ResponseWriter, req *http.Request) {
	// TODO: some validation is not found
	vars := mux.Vars(req)
	depositAddress := vars["depositaddress"]

	// TODO: move to debug
	log.WithFields(logrus.Fields{
		// TODO: more fields
		"depositaddress": depositAddress,
	}).Info("handleLatestMerkleProof")

	// TODO get also the root for trazability
	mPoof, mRoot, slot, avBalance, unbanBalance, err := m.Postgres.GetLatestMerkleProofByDeposit(depositAddress)
	if err != nil {
		m.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	m.respondOK(w, httpOkProofs{depositAddress, mRoot, slot, mPoof, avBalance, unbanBalance})
}
