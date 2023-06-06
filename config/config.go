package config

import (
	"errors"
	"flag"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hako/durafmt"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	ConsensusEndpoint     string   `json:"consensus_endpoint"`
	ExecutionEndpoint     string   `json:"execution_endpoint"`
	Network               string   `json:"network"`
	PoolAddress           string   `json:"pool_address"`
	DeployedSlot          uint64   `json:"deployed_slot"`
	DeployedBlock         uint64   `json:"deployed_block"`
	CheckPointSizeInSlots uint64   `json:"checkpoint_size"`
	PoolFeesPercent       int      `json:"pool_fees_percent"` // With 2 decimals (eg 1.5% = 150)
	PoolFeesAddress       string   `json:"pool_fees_address"`
	DryRun                bool     `json:"dry_run"`
	NumRetries            int      `json:"num_retries"`
	CollateralInWei       *big.Int `json:"collateral_in_wei"`
	UpdaterKeyPass        string   `json:"-"`
	UpdaterKeyPath        string   `json:"-"`
}

type CliConfig struct {
	DryRun            bool
	UpdaterKeyPath    string
	UpdaterKeyPass    string
	NumRetries        int
	ConsensusEndpoint string
	ExecutionEndpoint string
	PoolAddress       string
}

// By default the release is a custom build. CI takes care of upgrading it with
// go build -v -ldflags="-X 'github.com/dappnode/mev-sp-oracle/config.ReleaseVersion=x.y.z'"
var ReleaseVersion = "custom-build-your-own-risk"

var MainnetRelays = []string{
	"boost-relay.flashbots.net",
	"bloxroute.max-profit.blxrbdn.com",
	"bloxroute.ethical.blxrbdn.com",
	"bloxroute.regulated.blxrbdn.com",
	"builder-relay-mainnet.blocknative.com",
	"relay.edennetwork.io",
	"mainnet-relay.securerpc.com",
	"relayooor.wtf",
	"relay.ultrasound.money",
	"agnostic-relay.net",
	"aestus.live",
}
var GoerliRelays = []string{
	"builder-relay-goerli.flashbots.net",
	"bloxroute.max-profit.builder.goerli.blxrbdn.com",
	"builder-relay-goerli.blocknative.com/",
	"relay-goerli.edennetwork.io",
	"goerli-relay.securerpc.com",
}

const MaxUint64 = ^uint64(0)

func NewCliConfig() (*CliConfig, error) {
	// Optional flags TODO: Test!
	var version = flag.Bool("version", false, "Prints the release version and exits")
	var dryRun = flag.Bool("dry-run", false, "If enabled, the pool contract will not be updated")
	var updaterKeystorePath = flag.String("updater-keystore-path", "", "Path to the password-protected keystore file of the updater")
	var updaterKeystorePass = flag.String("updater-keystore-pass", "", "Password of the updater keystore file")
	var numRetries = flag.Int("num-retries", 0, "Number of retries for each interaction (consensus, execution): 0 infinite")

	// Mandatory flags TODO: Test!
	var consensusEndpoint = flag.String("consensus-endpoint", "", "Ethereum consensus endpoint")
	var executionEndpoint = flag.String("execution-endpoint", "", "Ethereum execution endpoint")
	var poolAddress = flag.String("pool-address", "", "Address of the smoothing pool contract")

	flag.Parse()

	if *version {
		log.Info("Version: ", ReleaseVersion)
		os.Exit(0)
	}

	// Some simple cli argument validation

	if !*dryRun && *updaterKeystorePath == "" {
		return nil, errors.New("you must provide a keystore file to update the contract root")
	}

	if !*dryRun && *updaterKeystorePass == "" {
		return nil, errors.New("you must provide a password for the keystore file")
	}

	if *dryRun && *updaterKeystorePath != "" {
		return nil, errors.New("you can't provide a keystore file in dry run mode")
	}

	if *dryRun && *updaterKeystorePass != "" {
		return nil, errors.New("you can't provide a password for the keystore file in dry run mode")
	}

	if !common.IsHexAddress(*poolAddress) {
		return nil, errors.New("pool-address: " + *poolAddress + " is not a valid address")
	}

	cliConf := &CliConfig{
		DryRun:            *dryRun,
		UpdaterKeyPath:    *updaterKeystorePath,
		UpdaterKeyPass:    *updaterKeystorePass,
		NumRetries:        *numRetries,
		ConsensusEndpoint: *consensusEndpoint,
		ExecutionEndpoint: *executionEndpoint,
		PoolAddress:       *poolAddress,
	}
	logConfig(cliConf)
	return cliConf, nil
}

func logConfig(cfg *CliConfig) {
	log.WithFields(log.Fields{
		"DryRun":            cfg.DryRun,
		"UpdaterKeyPath":    cfg.UpdaterKeyPath,
		"UpdaterKeyPass":    cfg.UpdaterKeyPass,
		"NumRetries":        cfg.NumRetries,
		"ConsensusEndpoint": cfg.ConsensusEndpoint,
		"ExecutionEndpoint": cfg.ExecutionEndpoint,
		"PoolAddress":       cfg.PoolAddress,
	}).Info("Cli Config:")
}

// Converts from slots to readable time (eg 1 day 9 hours 20 minutes)
func SlotsToTime(slots uint64) string {
	// Hardcoded. Mainnet Ethereum configuration
	SecondsInSlot := uint64(12)

	timeduration := time.Duration(slots*SecondsInSlot) * time.Second
	strDuration := durafmt.Parse(timeduration).String()

	return strDuration
}
