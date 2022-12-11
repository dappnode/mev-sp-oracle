package config

import (
	"bufio"
	"flag"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	ConsensusEndpoint     string
	ExecutionEndpoint     string
	Network               string
	PoolAddress           string
	DeployedSlot          uint64
	CheckPointSizeInSlots uint64

	// Debug flags, never use in production
	DebugHardcodedSubscriptions []uint64
}

// By default the release is a custom build. CI takes care of upgrading it with
// go build -v -ldflags="-X 'github.com/xxx/yyy/config.ReleaseVersion=x.y.z'"
var ReleaseVersion = "custom-build"

func NewCliConfig() (*Config, error) {
	var version = flag.Bool("version", false, "Prints the release version and exits")
	var consensusEndpoint = flag.String("consensus-endpoint", "", "")
	var executionEndpoint = flag.String("execution-endpoint", "", "xxx")
	var network = flag.String("network", "mainnet", "Network to run in: mainnet|goerli")
	var poolAddress = flag.String("pool-address", "", "Address of the smoothing pool contract")
	var deployedSlot = flag.Uint64("deployed-slot", 0, "Deployed slot of the smart contract: slot, not block")
	var checkPointSizeInSlots = flag.Uint64("checkpoint-size", 0, "Size in slots for each checkpoint, used to generate dumps and update merkle roots")

	// Debug flags, never use in production
	var debugHardcodedSubscriptionsFile = flag.String("debug-hardcoded-subscriptions-file", "", "Path to file containing a list of hardcoded validator indexes, one per line")
	flag.Parse()

	if *version {
		log.Info("Version: ", ReleaseVersion)
		os.Exit(0)
	}

	// Only to debug: Read hardcoded subscriptions from a file
	debugHardcodedSubscriptions, err := ReadHardcodedSubscriptions(*debugHardcodedSubscriptionsFile)
	if err != nil {
		log.Fatal(err)
	}

	conf := &Config{
		ConsensusEndpoint:           *consensusEndpoint,
		ExecutionEndpoint:           *executionEndpoint,
		Network:                     *network,
		PoolAddress:                 *poolAddress,
		DeployedSlot:                *deployedSlot,
		CheckPointSizeInSlots:       *checkPointSizeInSlots,
		DebugHardcodedSubscriptions: debugHardcodedSubscriptions,
	}
	logConfig(conf)
	return conf, nil
}

func logConfig(cfg *Config) {
	log.WithFields(log.Fields{
		"ConsensusEndpoint":           cfg.ConsensusEndpoint,
		"ExecutionEndpoint":           cfg.ExecutionEndpoint,
		"Network":                     cfg.Network,
		"PoolAddress":                 cfg.PoolAddress,
		"DeployedSlot":                cfg.DeployedSlot,
		"CheckPointSizeInSlots":       cfg.CheckPointSizeInSlots,
		"DebugHardcodedSubscriptions": cfg.DebugHardcodedSubscriptions,
	}).Info("Cli Config:")
}

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
