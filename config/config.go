package config

import (
	"crypto/ecdsa"
	"errors"
	"flag"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hako/durafmt"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	ConsensusEndpoint     string            `json:"consensus_endpoint"`
	ExecutionEndpoint     string            `json:"execution_endpoint"`
	Network               string            `json:"network"`
	PoolAddress           string            `json:"pool_address"`
	DeployedSlot          uint64            `json:"deployed_slot"`
	CheckPointSizeInSlots uint64            `json:"checkpoint_size"`
	PoolFeesPercent       int               `json:"pool_fees_percent"`
	PoolFeesAddress       string            `json:"pool_fees_address"`
	DryRun                bool              `json:"dry_run"`
	NumRetries            int               `json:"num_retries"`
	CollateralInWei       *big.Int          `json:"collateral_in_wei"`
	UpdaterAddress        string            `json:"updater_address"`
	UpdaterKeyPath        string            `json:"-"`
	UpdaterKey            *ecdsa.PrivateKey `json:"-"`
}

// By default the release is a custom build. CI takes care of upgrading it with
// go build -v -ldflags="-X 'github.com/xxx/yyy/config.ReleaseVersion=x.y.z'"
var ReleaseVersion = "custom-build"

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

func NewCliConfig() (*Config, error) {
	// Optional flags TODO: Test!
	var version = flag.Bool("version", false, "Prints the release version and exits")
	var dryRun = flag.Bool("dry-run", false, "If enabled, the pool contract will not be updated")
	var updaterKeystorePath = flag.String("updater-keystore-path", "", "Path to the password-protected keystore file of the updater")
	var updaterKeystorePass = flag.String("updater-keystore-pass", "", "Password of the updater keystore file")
	var numRetries = flag.Int("num-retries", 0, "Number of retries for each interaction (consensus, execution): 0 infinite")

	// Mandatory flags TODO: Test!
	var consensusEndpoint = flag.String("consensus-endpoint", "", "Ethereum consensus endpoint")
	var executionEndpoint = flag.String("execution-endpoint", "", "Ethereum execution endpoint")
	var network = flag.String("network", "mainnet", "Network to run in: mainnet|goerli")
	var poolAddress = flag.String("pool-address", "", "Address of the smoothing pool contract")
	var deployedSlot = flag.Uint64("deployed-slot", 0, "Deployed slot of the smart contract: slot, not block")
	var checkPointSizeInSlots = flag.Uint64("checkpoint-size", 0, "Size in slots for each checkpoint, used to generate dumps and update merkle roots")
	var poolFeesPercent = flag.Int("pool-fees-percent", -1, "Percent of fees pool-fees-percent takes [0-100]")
	var poolFeesAddress = flag.String("pool-fees-address", "", "Ethereum account with 0x where pool fees go to")
	var ethCollateral = flag.Uint64("collateral-in-wei", MaxUint64, "Amount of collateral in ETH wei")

	flag.Parse()

	if *version {
		log.Info("Version: ", ReleaseVersion)
		os.Exit(0)
	}

	// Some simple cli argument validation

	// Mandatory flag
	if *poolFeesAddress == "" {
		return nil, errors.New("pool-fees-address flag is not present")
	}

	// Mandatory flag
	if *poolFeesPercent == -1 {
		return nil, errors.New("pool-fees-percent flag is not present")
	}

	if *poolFeesPercent < 0 || *poolFeesPercent > 100 {
		return nil, errors.New("pool-fees-percent must be between 0 and 100")
	}

	// Mandatory flag
	if *network != "mainnet" && *network != "goerli" {
		return nil, errors.New("wrong network provided, must be mainnet or goerli")
	}

	if *poolFeesAddress == *poolAddress {
		return nil, errors.New("pool-fees-address and pool-address can't be equal")
	}

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

	// Check deployerPrivateKey is valid
	var privateKey *ecdsa.PrivateKey
	var publicKey string

	// Only parse it not in dry run mode
	if !*dryRun {
		jsonBytes, err := ioutil.ReadFile(*updaterKeystorePath)
		if err != nil {
			log.Fatal(err)
		}

		account, err := keystore.DecryptKey(jsonBytes, *updaterKeystorePass)
		if err != nil {
			log.Fatal(err)
		}

		privateKey = account.PrivateKey
		publicKey = account.Address.String()
	}

	if !common.IsHexAddress(*poolAddress) {
		return nil, errors.New("pool-address: " + *poolAddress + " is not a valid address")
	}

	if !common.IsHexAddress(*poolFeesAddress) {
		return nil, errors.New("pool-fees-address: " + *poolFeesAddress + " is not a valid address")
	}

	if *ethCollateral == MaxUint64 {
		return nil, errors.New("collateral-in-wei flag is not present")
	}
	ethCollateralInWei := big.NewInt(0).SetUint64(*ethCollateral)

	conf := &Config{
		ConsensusEndpoint:     *consensusEndpoint,
		ExecutionEndpoint:     *executionEndpoint,
		Network:               *network,
		PoolAddress:           *poolAddress,
		DeployedSlot:          *deployedSlot,
		CheckPointSizeInSlots: *checkPointSizeInSlots,
		PoolFeesPercent:       *poolFeesPercent,
		PoolFeesAddress:       *poolFeesAddress,
		CollateralInWei:       ethCollateralInWei,
		DryRun:                *dryRun,
		NumRetries:            *numRetries,
		UpdaterAddress:        publicKey,
		UpdaterKey:            privateKey,
		UpdaterKeyPath:        *updaterKeystorePath,
	}
	logConfig(conf)
	return conf, nil
}

func logConfig(cfg *Config) {
	log.WithFields(log.Fields{
		"ConsensusEndpoint":     cfg.ConsensusEndpoint,
		"ExecutionEndpoint":     cfg.ExecutionEndpoint,
		"Network":               cfg.Network,
		"PoolAddress":           cfg.PoolAddress,
		"DeployedSlot":          cfg.DeployedSlot,
		"CheckPointSizeInSlots": cfg.CheckPointSizeInSlots,
		"UpdaterAddress":        cfg.UpdaterAddress,
		"UpdaterKeyPath":        cfg.UpdaterKeyPath,
		"PoolFeesPercent":       cfg.PoolFeesPercent,
		"PoolFeesAddress":       cfg.PoolFeesAddress,
		"CollateralInWei":       cfg.CollateralInWei,
		"DryRun":                cfg.DryRun,
		"NumRetries":            cfg.NumRetries,
	}).Info("Cli Config:")

	log.Info("Configured smoothing pool address: ", cfg.PoolAddress)
	log.Info("Configured fees for smoothing pool: ", cfg.PoolFeesPercent, " %")
	log.Info("Configured address to claim fees (ensure you control its keys): ", cfg.PoolFeesAddress)

	if cfg.DryRun {
		log.Warn("The pool contract will NOT be updated, running in dry-run mode")
	} else {
		log.Warn("Configured address to update the pool merkle root (ensure it has permissions): ", cfg.UpdaterAddress)
		log.Info("The merkle root onchain will be updated every ", cfg.CheckPointSizeInSlots, " slots (", SlotsToTime(cfg.CheckPointSizeInSlots), ")")
	}
}

// Converts from slots to readable time (eg 1 day 9 hours 20 minutes)
func SlotsToTime(slots uint64) string {
	// Hardcoded. Mainnet Ethereum configuration
	SecondsInSlot := uint64(12)

	timeduration := time.Duration(slots*SecondsInSlot) * time.Second
	strDuration := durafmt.Parse(timeduration).String()

	return strDuration
}
