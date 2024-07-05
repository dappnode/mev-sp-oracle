package oracle

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	eth2 "github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
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

func Test_GetGetSlotByBlock(t *testing.T) {
	// Uncomment to run
	t.Skip("Skipping test")

	// Folder to store the result

	poolAddress := "0xF21fbbA423f3a893A2402d68240B219308AbCA46" // contract of address to detect events

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:3500",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       poolAddress,
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	genesis, err := onchain.ConsensusClient.Genesis(context.Background(), &eth2.GenesisOpts{})
	if err != nil {
		log.Fatal("Could not get genesis: " + err.Error())
	}

	genesisTime := uint64(genesis.Data.GenesisTime.Unix())

	slot, err := onchain.GetSlotByBlock(big.NewInt(18902677), genesisTime)
	require.NoError(t, err)
	require.Equal(t, uint64(8097330), slot)
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

func Test_BlocksWithInternalTx(t *testing.T) {
	t.Skip("Skipping test")

	// https://etherscan.io/tx/0x6c9adaa16946d1279e0db0fc9348201c48b2f70a62ac5edfe06dc0ba2b4f3e3c
	pool := "0xAdFb8D27671F14f297eE94135e266aAFf8752e35"
	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:3500",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       pool,
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)
	oracle := NewOracle(&Config{})

	fullBlock := onchain.FetchFullBlock(8097330, oracle)
	donations := fullBlock.GetDonations(pool)
	mevReward, isMev, recipient := fullBlock.MevRewardInWei()
	require.Equal(t, big.NewInt(0).SetUint64(31995314350342039), mevReward)
	require.Equal(t, true, isMev)
	require.Equal(t, "0xadfb8d27671f14f297ee94135e266aaff8752e35", recipient)
	require.Equal(t, 0, len(donations))
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
}

func Test_NonExistentValidator(t *testing.T) {
	// This takes long, if timeout hits: go test -v -run Test_EndToEnd -timeout 30m
	t.Skip("Skipping test")

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:3500",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0xAdFb8D27671F14f297eE94135e266aAFf8752e35",
	}

	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	// Does not exist
	val, err := onchain.GetSingleValidator(phase0.ValidatorIndex(999999999999), "finalized")
	require.NoError(t, err)
	require.Nil(t, val)

	// Exists
	val, err = onchain.GetSingleValidator(phase0.ValidatorIndex(0), "finalized")
	require.NoError(t, err)
	require.NotNil(t, val)

	// Works with arbitrary slots
	val, err = onchain.GetSingleValidator(phase0.ValidatorIndex(0), "0")
	require.NoError(t, err)
	require.NotNil(t, val)
	fmt.Println(val)
}

// Run this test to ensure no regressions are introduced
func Test_EndToEnd_Mainnet(t *testing.T) {
	// This takes long, if timeout hits: go test -v -run Test_EndToEnd -timeout 30m
	t.Skip("Skipping test")

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:3500",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0xAdFb8D27671F14f297eE94135e266aAFf8752e35",
	}

	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	cfg := onchain.GetConfigFromContract(cfgOnchain)

	oracleInstance := NewOracle(cfg)

	// Slots where something happened. This saves having to sync everything, which takes too long
	slotsToProcess := []uint64{
		7756830, 7756835, 7757361, 7766538, 7779248, 7781938, 7783745, 7808048, 7822033, 7822251, 7836848, 7865077, 7865648, 7878170, 7878274, 7878509, 7878647, 7878967, 7878971, 7878981, 7878990, 7878996, 7879002, 7879009, 7879419, 7879422, 7879679, 7879761, 7879769, 7879972, 7879978, 7879988, 7880772, 7880784, 7881085, 7881092, 7881854, 7882923, 7882992, 7883031, 7883059, 7885178, 7885600, 7885949, 7888639, 7888782, 7890244, 7890641, 7891577, 7892765, 7892772, 7892777, 7892782, 7892787, 7892797, 7892804, 7892810, 7892814, 7892818, 7892824, 7892829, 7892833, 7892838, 7892840, 7892842, 7892846, 7892851, 7892855, 7892860, 7892865, 7892869, 7892874, 7892878, 7892883, 7892889, 7892893, 7892896, 7892899, 7892902, 7892906, 7892910, 7892914, 7894448, 7898599, 7900936, 7900940, 7900950, 7900953, 7902224, 7902637, 7904360, 7905670, 7907674, 7907745, 7912471, 7912649, 7912796, 7914675, 7915066, 7915071, 7917929, 7917940, 7919272, 7921005, 7922358, 7923248, 7925550, 7932272, 7933255, 7936382, 7941715, 7943854, 7948661, 7951145, 7951150, 7951154, 7952048, 7954878, 7955358, 7956791, 7958749, 7958774, 7958785, 7958806, 7958824, 7958828, 7959118, 7959127, 7959131, 7959139, 7959143, 7959185, 7959191, 7959196, 7959201, 7959206, 7959760, 7961602, 7965768, 7966604, 7966915, 7968732, 7969310, 7969328, 7969438, 7969720, 7970468, 7971205, 7974054, 7974963, 7974968, 7974972, 7974976, 7974981, 7974985, 7974989, 7974994, 7974998, 7975011, 7976709, 7980512, 7980848, 7983787, 7984069, 7988918, 7990055, 7999656, 8002052, 8002528, 8002775, 8006554, 8008026, 8009257, 8009648, 8009923, 8013759, 8017944, 8020418, 8023885, 8024255, 8024356, 8024685, 8024714, 8025916, 8028899, 8030778, 8032965, 8034125, 8034676, 8034793, 8035857, 8036291, 8036870, 8036907, 8038448, 8038850, 8040421, 8041904, 8042367, 8043927, 8047006, 8049725, 8049786, 8049854, 8050190, 8051808, 8052584, 8052619, 8055689, 8060916, 8060938, 8064244, 8065575, 8066225, 8067248, 8067518, 8067523, 8067528, 8067533, 8067540, 8070066, 8073795, 8075703, 8076369, 8076540, 8079845, 8079861, 8081527, 8083977, 8083995, 8084303, 8084527, 8087197, 8087704, 8088182, 8089905, 8092838, 8093532, 8094008, 8095244, 8095482, 8095910, 8096048, 8097330, 8098322, 8099002, 8099815,
	}

	prevSlot := slotsToProcess[0]
	for _, slot := range slotsToProcess {
		if prevSlot > slot {
			t.Fatal("Slots are not in order")
		}

		// we force to process the slots we want
		oracleInstance.State().NextSlotToProcess = slot
		oracleInstance.State().LatestProcessedSlot = slot - 1

		// Fetch block information
		fullBlock := onchain.FetchFullBlock(oracleInstance.State().NextSlotToProcess, oracleInstance)

		// Advance state to next slot based on the information we got from the block
		processedSlot, err := oracleInstance.AdvanceStateToNextSlot(fullBlock)
		require.NoError(t, err)

		log.Info("Processed slot: ", processedSlot)

		if isInSlice(processedSlot, []uint64{7779248, 7808048, 7836848, 7865648, 7894448, 7923248, 7952048, 7980848, 8009648, 8038448, 8067248, 8096048}) {
			oracleInstance.FreezeCheckpoint()
		}
	}
}

func Test_Mainnet_BeaverIssue(t *testing.T) {
	t.Skip("Skipping test")

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:3500",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0xAdFb8D27671F14f297eE94135e266aAFf8752e35",
	}

	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	cfg := onchain.GetConfigFromContract(cfgOnchain)

	oracleInstance := NewOracle(cfg)

	// Subscribe the proposer of slot 9444748
	oracleInstance.addSubscription(12137, "", "")

	// Slots where something happened. This saves having to sync everything, which takes too long
	slotsToProcess := []uint64{
		9444748,
	}

	prevSlot := slotsToProcess[0]
	for _, slot := range slotsToProcess {
		if prevSlot > slot {
			t.Fatal("Slots are not in order")
		}

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
}

func isInSlice(element uint64, slice []uint64) bool {
	for _, value := range slice {
		if element == value {
			return true
		}
	}
	return false
}
