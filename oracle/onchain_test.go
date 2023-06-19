package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// None of this tests can be executed without a valid consensus and execution client
// so they are disabled by default, only to be run manually.
var skip = true

// Not a test per se, just an util to fetch block and store them for mocking
func Test_GetFullBlockAtSlot(t *testing.T) {
	// Uncomment to run
	t.Skip("Skipping test")

	// Folder to store the result
	folder := "../mock"

	// Config params
	slotToFetch := uint64(5864096)                              // slot to fetch
	fetchHeaderAndReceipts := true                              // fetch header and receipts to reconstruct tip
	poolAddress := "0xF21fbbA423f3a893A2402d68240B219308AbCA46" // contract of address to detect events

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       poolAddress,
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)
	oracle := NewOracle(&Config{})
	chaindId, err := onchain.ExecutionClient.ChainID(context.Background())
	require.NoError(t, err)

	// Fetch all information from the blockchain
	fullBlock := onchain.FetchFullBlock(slotToFetch, oracle, fetchHeaderAndReceipts)

	// Serialize to json and dump to file
	jsonData, err := json.MarshalIndent(fullBlock, "", " ")
	require.NoError(t, err)
	fileName := fmt.Sprintf("fullblock_slot_%d_chainid_%s%s.json", slotToFetch, chaindId.String(), HasHeader(fetchHeaderAndReceipts))
	path := filepath.Join(folder, fileName)
	err = ioutil.WriteFile(path, jsonData, 0644)
	require.NoError(t, err)
}

func HasHeader(has bool) string {
	if has {
		return "_withheaders"
	}
	return ""
}

// Fetches the balance of a given address
func Test_FetchFromExecution(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onChain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)
	account := common.HexToAddress("0xf573d99385c05c23b24ed33de616ad16a43a0919")
	balance, err := onChain.ExecutionClient.BalanceAt(context.Background(), account, nil)
	require.NoError(t, err)
	require.NotNil(t, balance)

	//expectedValue, ok := new(big.Int).SetString("25893180161173005034", 10)
	//require.True(t, ok)
	//require.Equal(t, expectedValue, balance)
}

func Test_IsAddressWhitelisted(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x8eba4a4a8d4dfa78bcb734efd1ea9f33b61e3243",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	// Hardcoded for this contract: https://goerli.etherscan.io/address/0x8eba4A4A8d4DFa78BCB734efD1eA9f33b61e3243
	deployedBlock := uint64(9094304)
	isWhitelisted, err := onchain.IsAddressWhitelisted(deployedBlock, "0x14264aD0471ee1f068CFAC40A9FcC352274ced56")
	require.NoError(t, err)
	require.Equal(t, true, isWhitelisted)
}

func Test_EndToEnd(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0xF21fbbA423f3a893A2402d68240B219308AbCA46",
	}

	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	cfg := onchain.GetConfigFromContract(cfgOnchain)

	oracleInstance := NewOracle(cfg)

	// TODO: missing tons of things like subscriptions unsubs, etc.
	onchain.RefreshBeaconValidators()
	oracleInstance.SetBeaconValidators(onchain.Validators())

	slotsToProcess := []uint64{
		5840966, //mev reward
		5843638, //vanila reward (auto subs)
		5844947, //vanila reward (auto subs)
		5846531, //mev reward
		5846747, //vanila reward (auto subs)
		5850959, //vanila reward (auto subs)
		5851651, //vanila reward (auto subs)
		5852212, //vanila reward (auto subs)
		5852262, //vanila reward (auto subs)
		5852659, //vanila reward (auto subs)
		5853824, //vanila reward (auto subs)
		5855268, //vanila reward (auto subs)
		5856619, //vanila reward (auto subs)
		5858585, //vanila reward (auto subs)
		//5862054, //donation normal TODO
		//5862104, //donation via smart contract TODO:
		// TODO: Add more blocks with subs unsubs etc
		// TODO: Randomly add blocks without anything interesting
		// TODO: Randmly add missed blocks
	}

	prevSlot := slotsToProcess[0]
	for _, slot := range slotsToProcess {
		if prevSlot > slot {
			t.Fatal("Slots are not in order")
		}
		// block is not really used
		//oracleInstance.State().LatestProcessedBlock = 5768580

		// we force to process the slots we want
		oracleInstance.State().NextSlotToProcess = slot
		oracleInstance.State().LatestProcessedSlot = slot - 1

		// Fetch block information
		fullBlock := onchain.FetchFullBlock(oracleInstance.State().NextSlotToProcess, oracleInstance)

		// Advance state to next slot based on the information we got from the block
		processedSlot, err := oracleInstance.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		log.Info("Processed slot: ", processedSlot)
	}

	oracleInstance.FreezeCheckpoint()

	// TODO: Run asserts
	//oracleInstance.SaveStateToFile()
	//oracleInstance.SaveToJson()

	//require.Equal(t, "0x000000", oracleInstance.LatestCommitedState().MerkleRoot)

	// root: 0xf0ecfb7afe96f7f7b570598c71aaac8fae3e6880e078227106e3a29446d5dbf8
	//AccumulatedBalanceWei=131064623899584732 LeafHash=7c049ecd5a07fc5b7d39573db41a1faca70a798112583dce61b0c8761eaa2166 WithdrawalAddress=0xe46f9be81f9a3aca1808bb8c36d353436bb96091
	//AccumulatedBalanceWei=177243009873641532 LeafHash=83303f7cf1e36186b6d97de80db49c77fca6fd2a4fcdac771b4139d46b9abd1c WithdrawalAddress=0xa111b576408b1ccdaca3ef26f22f082c49bcaa55
}
