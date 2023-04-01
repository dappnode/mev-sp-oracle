package config

import (
	"crypto/ecdsa"
	"errors"
	"flag"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	ConsensusEndpoint     string
	ExecutionEndpoint     string
	Network               string
	PoolAddress           string
	UpdaterAddress        string
	DeployedSlot          uint64
	CheckPointSizeInSlots uint64
	PostgresEndpoint      string
	DeployerPrivateKey    string
	PoolFeesPercent       int
	PoolFeesAddress       string
	DryRun                bool
	NumRetries            int
	CollateralInWei       *big.Int
}

// By default the release is a custom build. CI takes care of upgrading it with
// go build -v -ldflags="-X 'github.com/xxx/yyy/config.ReleaseVersion=x.y.z'"
var ReleaseVersion = "custom-build"

var MainRelays = []string{
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
var TestRelays = []string{
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
	var deployerPrivateKey = flag.String("deployer-private-key", "", "Private key of the deployer account")
	var numRetries = flag.Int("num-retries", 0, "Number of retries for each interaction (consensus, execution, postgres): 0 infinite")

	// Mandatory flags TODO: Test!
	var consensusEndpoint = flag.String("consensus-endpoint", "", "Ethereum consensus endpoint")
	var executionEndpoint = flag.String("execution-endpoint", "", "Ethereum execution endpoint")
	var network = flag.String("network", "mainnet", "Network to run in: mainnet|goerli")
	var poolAddress = flag.String("pool-address", "", "Address of the smoothing pool contract")
	var deployedSlot = flag.Uint64("deployed-slot", 0, "Deployed slot of the smart contract: slot, not block")
	var checkPointSizeInSlots = flag.Uint64("checkpoint-size", 0, "Size in slots for each checkpoint, used to generate dumps and update merkle roots")
	var postgresEndpoint = flag.String("postgres-endpoint", "", "Postgres endpoint")
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

	if !*dryRun && *deployerPrivateKey == "" {
		return nil, errors.New("you must provide a private key to update the contract root")
	}

	if *dryRun && *deployerPrivateKey != "" {
		return nil, errors.New("dry-run mode specified buy also provided a deployer-private-key")
	}

	// Check deployerPrivateKey is valid
	var pKey *ecdsa.PrivateKey
	var err error
	var publicKeyECDSA *ecdsa.PublicKey
	var updaterAddress string

	// Only parse it not in dry run mode
	if !*dryRun {
		pKey, err = crypto.HexToECDSA(*deployerPrivateKey)
		if err != nil {
			return nil, errors.New("wrong private key, couldn't parse it: " + err.Error())
		}
		publicKey := pKey.Public()
		var ok bool
		publicKeyECDSA, ok = publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("error casting public key to ECDSA: " + err.Error())
		}
		updaterAddress = crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	} else {
		updaterAddress = "NA"
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
		UpdaterAddress:        updaterAddress,
		DeployedSlot:          *deployedSlot,
		CheckPointSizeInSlots: *checkPointSizeInSlots,
		PostgresEndpoint:      *postgresEndpoint,
		DeployerPrivateKey:    *deployerPrivateKey,
		PoolFeesPercent:       *poolFeesPercent,
		PoolFeesAddress:       *poolFeesAddress,
		CollateralInWei:       ethCollateralInWei,
		DryRun:                *dryRun,
		NumRetries:            *numRetries,
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
		"UpdaterAddress":        cfg.UpdaterAddress,
		"DeployedSlot":          cfg.DeployedSlot,
		"CheckPointSizeInSlots": cfg.CheckPointSizeInSlots,
		"PostgresEndpoint":      cfg.PostgresEndpoint,
		"DeployerPrivateKey":    "TODO: use a file with protected password",
		"PoolFeesPercent":       cfg.PoolFeesPercent,
		"PoolFeesAddress":       cfg.PoolFeesAddress,
		"CollateralInWei":       cfg.CollateralInWei,
		"DryRun":                cfg.DryRun,
		"NumRetries":            cfg.NumRetries,
	}).Info("Cli Config:")

	log.Info("The smoothing pool at ", cfg.PoolAddress, " takes a cut of ", cfg.PoolFeesPercent, "% ensure you control the keys for ", cfg.PoolAddress, " to claim the fees")

	if cfg.DryRun {
		log.Warn("The pool contract will NOT be updated, running in dry-run mode")
	} else {
		log.Warn("The pool contract will be updated. Make the account has balance to cover tx fees: ", cfg.UpdaterAddress)
	}
}
