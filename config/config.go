package config

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"flag"
	"os"
	"strconv"
	"strings"

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
}

// By default the release is a custom build. CI takes care of upgrading it with
// go build -v -ldflags="-X 'github.com/xxx/yyy/config.ReleaseVersion=x.y.z'"
var ReleaseVersion = "custom-build"

func NewCliConfig() (*Config, error) {
	var version = flag.Bool("version", false, "Prints the release version and exits")
	var consensusEndpoint = flag.String("consensus-endpoint", "", "Ethereum consensus endpoint")
	var executionEndpoint = flag.String("execution-endpoint", "", "Ethereum execution endpoint")
	var network = flag.String("network", "", "Network to run in: mainnet|goerli")
	var poolAddress = flag.String("pool-address", "", "Address of the smoothing pool contract")
	var deployedSlot = flag.Uint64("deployed-slot", 0, "Deployed slot of the smart contract: slot, not block")
	var checkPointSizeInSlots = flag.Uint64("checkpoint-size", 0, "Size in slots for each checkpoint, used to generate dumps and update merkle roots")
	var postgresEndpoint = flag.String("postgres-endpoint", "", "Postgres endpoint")
	var deployerPrivateKey = flag.String("deployer-private-key", "", "Private key of the deployer account")
	var poolFeesPercent = flag.Int("pool-fees-percent", -1, "Percent of fees pool-fees-percent takes [0-100]")
	var poolFeesAddress = flag.String("pool-fees-address", "", "Ethereum account with 0x where pool fees go to")
	var dryRun = flag.Bool("dry-run", false, "If enabled, the pool contract will not be updated")
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
	pKey, err := crypto.HexToECDSA(*deployerPrivateKey)
	if err != nil {
		return nil, errors.New("wrong private key, couldn't parse it: " + err.Error())
	}

	publicKey := pKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA: " + err.Error())
	}

	if !common.IsHexAddress(*poolAddress) {
		return nil, errors.New("pool-address: " + *poolAddress + " is not a valid address")
	}

	if !common.IsHexAddress(*poolFeesAddress) {
		return nil, errors.New("pool-fees-address: " + *poolFeesAddress + " is not a valid address")
	}

	conf := &Config{
		ConsensusEndpoint:     *consensusEndpoint,
		ExecutionEndpoint:     *executionEndpoint,
		Network:               *network,
		PoolAddress:           *poolAddress,
		UpdaterAddress:        crypto.PubkeyToAddress(*publicKeyECDSA).Hex(),
		DeployedSlot:          *deployedSlot,
		CheckPointSizeInSlots: *checkPointSizeInSlots,
		PostgresEndpoint:      *postgresEndpoint,
		DeployerPrivateKey:    *deployerPrivateKey,
		PoolFeesPercent:       *poolFeesPercent,
		PoolFeesAddress:       *poolFeesAddress,
		DryRun:                *dryRun,
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
		"DryRun":                cfg.DryRun,
	}).Info("Cli Config:")

	log.Info("The smoothing pool at ", cfg.PoolAddress, " takes a cut of ", cfg.PoolFeesPercent, "% ensure you control the keys for ", cfg.PoolAddress, " to claim the fees")

	if cfg.DryRun {
		log.Warn("The pool contract will NOT be updated, running in dry-run mode")
	} else {
		log.Warn("The pool contract will be updated. Ensure the account has balance to cover tx fees: ", cfg.UpdaterAddress)
	}
}

// TODO: Unused
func ReadHardcodedSubscriptions(filePath string) ([]uint64, error) {
	preSubscribedIndexes := make([]uint64, 0)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Replace(line, " ", "", -1)
		if len(line) == 0 {
			continue
		}
		valIndexUint64, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			return nil, err
		}
		preSubscribedIndexes = append(preSubscribedIndexes, valIndexUint64)
	}
	return preSubscribedIndexes, nil
}
