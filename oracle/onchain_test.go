package oracle

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// UnmarshalJSON unmarshals a JSON string into a big.Int.
func (c *Call) UnmarshalJSON(b []byte) error {
	type Alias Call
	var raw struct {
		Value string `json:"value"`
		*Alias
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	c.Calls = raw.Calls
	c.From = raw.From
	c.Gas = raw.Gas
	c.GasUsed = raw.GasUsed
	c.Input = raw.Input
	c.To = raw.To
	c.Type = raw.Type

	c.Value = new(big.Int)
	val, success := c.Value.SetString(strings.Trim(raw.Value, `"`), 0)
	if !success {
		return fmt.Errorf("failed to parse big.Int from string: %s", raw.Value)
	}
	c.Value = val

	return nil
}

// Take from: https://github.com/ethereum/go-ethereum/blob/v1.13.8/eth/tracers/native/call.go#L48-L63
type Call struct {
	Calls   []Call   `json:"calls"`
	From    string   `json:"from"`
	Gas     string   `json:"gas"`
	GasUsed string   `json:"gasUsed"`
	Input   string   `json:"input"`
	To      string   `json:"to"`
	Type    string   `json:"type"`
	Value   *big.Int `json:"value"`
}

type Result struct {
	Calls []Call `json:"calls"`
}

// None of this tests can be executed without a valid consensus and execution client
// so they are disabled by default, only to be run manually.
var skip = true

// Not a test per se, just an util to fetch block and store them for mocking
func Test_GetFullBlockAtSlot(t *testing.T) {
	// Uncomment to run
	//t.Skip("Skipping test")

	// Folder to store the result
	//folder := "../mock"

	// Config params
	slotToFetch := uint64(8097330)                              // slot to fetch
	fetchHeaderAndReceipts := true                              // fetch header and receipts to reconstruct tip
	poolAddress := "0xAdFb8D27671F14f297eE94135e266aAFf8752e35" // contract of address to detect events

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
	//fullBlock := onchain.FetchFullBlock(slotToFetch, oracle, fetchHeaderAndReceipts)

	_ = chaindId

	hash := "0x6c9adaa16946d1279e0db0fc9348201c48b2f70a62ac5edfe06dc0ba2b4f3e3c"
	hashother := common.HexToHash(hash)

	fmt.Println("getting blocks")
	_ = oracle
	_ = slotToFetch
	_ = fetchHeaderAndReceipts

	hash := "0x6c9adaa16946d1279e0db0fc9348201c48b2f70a62ac5edfe06dc0ba2b4f3e3c"
	hashother := common.HexToHash(hash)

	var result Result
	tc := TraceConfig{
		Tracer: "callTracer",
	}

	err = onchain.ExecutionClient.Client().CallContext(context.Background(), &result, "debug_traceTransaction", hashother.String(), tc)
	require.NoError(t, err)

	for _, call := range result.Calls {
		fmt.Println("Call: ", call.To, " ", call.From, " ", call.Value, " ", call.Input, " ", call.Type)
	}

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

func Test_GetValidator(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:3500",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onChain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	vals, err := onChain.GetFinalizedValidators()
	require.NoError(t, err)

	for _, valEl := range vals {
		fmt.Println(valEl.Index)
		fmt.Println("raw: ", hex.EncodeToString(valEl.Validator.WithdrawalCredentials[:]))
		a, b := GetWithdrawalAndType(valEl)
		fmt.Println(valEl.Index, " ", hex.EncodeToString(valEl.Validator.WithdrawalCredentials[:]), "  ", a, "  ", b)
	}
}

func Test_IsAddressWhitelisted(t *testing.T) {
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

	// Hardcoded for this contract: https://goerli.etherscan.io/address/0x8eba4A4A8d4DFa78BCB734efD1eA9f33b61e3243
	address := common.HexToAddress("0x8eba4a4a8d4dfa78bcb734efd1ea9f33b61e3243")
	isWhitelisted, err := onchain.IsAddressWhitelisted(address)
	require.NoError(t, err)
	require.Equal(t, false, isWhitelisted)

	address = common.HexToAddress("0x0017914E98A2f791D59038EC10325152AC1D7438")
	isWhitelisted, err = onchain.IsAddressWhitelisted(address)
	require.NoError(t, err)
	require.Equal(t, true, isWhitelisted)
}

func Test_EndToEnd(t *testing.T) {
	// This takes long, if timeout hits: go test -v -run Test_EndToEnd -timeout 30m
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

	onchain.RefreshBeaconValidators()
	oracleInstance.SetBeaconValidators(onchain.Validators())

	// Uncomment to save validators
	//path := filepath.Join("../mock", "validators.json")
	//jsonData, err := json.MarshalIndent(onchain.Validators(), "", " ")
	//require.NoError(t, err)
	//err = ioutil.WriteFile(path, jsonData, 0644)
	//require.NoError(t, err)
	//log.Fatal("done")

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
		5862054, //donation normal TODO
		5862104, //donation via smart contract TODO:

		5863539,
		5864096,
		5870291,
		5871368,
		5871701,
		5874576,
		5880967,
		5882954,
		5883240,
		5885240,
		5885987,
		5887583,

		// subs
		5888073,
		5888079,
		5888082,
		5888090, // already subs validator
		5888096,
		5888099,
		5888101,
		5888104,
		5888105,
		5888106,
		5888108,
		5888109,
		5888112,
		5888114,
		5888116,
		5888118,
		5888121,
		5888123,
		5888126,
		// freeze state

		// 0xb0f08efb67c59a4b16b143cf3a4850e786c4295909bee85b41cdc7d78db5d329

		5889932, // vanila rewar (auto subs)
		5890341, // vanila rewar (auto subs)
		5892032, // vanila rewar (auto subs)
		5893934, // vanila rewar (auto subs)
		5894030, // vanila rewar (auto subs)
		5895093, // vanila rewar (auto subs)
		5895373, // vanila rewar (already subscribed before with AUTO)

		5895384, // subs non existent validator
		5895415, //BLOCK!	9209397, // subscription of validator with BLS cred. skipped

		5896015, // vanila rewar (auto subs)
		5896730, // vanila rewar (auto subs)
		// 0xd9d4170d0a04dd0406961aaf574c18eda1c4f639226b1f2c85f9e91c5e211def

		5897820, // vanila rewar (auto subs of a validator subscribed before with MANUAL)    -> BUG HERE. subscription TYPE.

		5900857, // vanila rewar (auto subs)
		5901298, // vanila rewar (auto subs)

		5901838, //BLOCK 9214269,// subscription of validator with BLS cred. skipped
		5901840, //BLOCK 9214271// subscription of validator with BLS cred. skipped
		5901841, //BLOCK 9214272// subscription of validator with BLS cred. skipped
		5901843, //BLOCK 9214273// subscription of validator with BLS cred. skipped
		5901845, //BLOCK 9214275// subscription of validator with BLS cred. skipped
		5901846, //BLOCK 9214276// subscription of validator with BLS cred. skipped
		5901847, //BLOCK 9214277// subscription of validator with BLS cred. skipped
		5901849, //BLOCK 9214279// subscription of validator with BLS cred. skipped
		5901850, //BLOCK 9214280// subscription of validator with BLS cred. skipped
		5901852, //BLOCK 9214281// subscription of validator with BLS cred. skipped

		5901856, //block 9214285 // unsubscription
		5901861, //block 9214288 // unsubscription
		5901862, //block 9214289 // unsubscription
		5901865, //block 9214290 // unsubscription
		5901868, // block 9214293 // unsubscription
		5901870, //block 9214295 // unsubscription
		5901872, //block 9214296 // unsubscription
		5901874, //block 9214298 // unsubscription
		5901882, //block 9214306 // unsubscription
		5901885, //block 9214307 // unsubscription
		5901888, //block 9214310 		// unsubscription of an already unsubscribed validator.

		5902555, // vanila rewar (auto subs)
		// 0x5db21e0b873daedd188ff5976f4950d6f03d5db5bc46620036ecce45014792e9

		5904027, // vanila rewar (auto subs)
		5904240, // vanila rewar (auto subs)
		5907004, // vanila rewar (auto subs)
		5907780, // vanila rewar (auto subs)
		5908715, // vanila rewar (auto subs)

		5910468, // vanila rewar (auto subs)

		// 0x3b256a0d99ea9b781fb55349146c21209fd05deb5f33f605aa8868e82fbd3b03

		5911491, // vanila rewar (auto subs)

		5912693, // block number 9222701 // unsubscription of a validator that doesnt exist.

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

		// Store the block for mocking later
		//isPoolRewarded := fullBlock.isAddressRewarded(oracleInstance.cfg.PoolAddress)
		//isFromSubscriber := oracleInstance.isSubscribed(fullBlock.GetProposerIndexUint64())
		//fileName := fmt.Sprintf("fullblock_slot_%d_chainid_%s%s.json", oracleInstance.State().NextSlotToProcess, "5", HasHeader(isFromSubscriber || isPoolRewarded))
		//path := filepath.Join("../mock", fileName)
		//jsonData, err := json.MarshalIndent(fullBlock, "", " ")
		//require.NoError(t, err)
		//err = ioutil.WriteFile(path, jsonData, 0644)
		//require.NoError(t, err)

		// Advance state to next slot based on the information we got from the block
		processedSlot, err := oracleInstance.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		log.Info("Processed slot: ", processedSlot)

		if processedSlot == uint64(5910468) {
			oracleInstance.FreezeCheckpoint()
		}
	}

	// TODO: Run asserts
	//oracleInstance.SaveStateToFile()
	//oracleInstance.SaveToJson()

	require.Equal(t, "0x3b256a0d99ea9b781fb55349146c21209fd05deb5f33f605aa8868e82fbd3b03", oracleInstance.State().CommitedStates[5910468].MerkleRoot)

	// root: 0xf0ecfb7afe96f7f7b570598c71aaac8fae3e6880e078227106e3a29446d5dbf8
	//AccumulatedBalanceWei=131064623899584732 LeafHash=7c049ecd5a07fc5b7d39573db41a1faca70a798112583dce61b0c8761eaa2166 WithdrawalAddress=0xe46f9be81f9a3aca1808bb8c36d353436bb96091
	//AccumulatedBalanceWei=177243009873641532 LeafHash=83303f7cf1e36186b6d97de80db49c77fca6fd2a4fcdac771b4139d46b9abd1c WithdrawalAddress=0xa111b576408b1ccdaca3ef26f22f082c49bcaa55
}
