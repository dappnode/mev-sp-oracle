package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// None of this tests can be executed without a valid consensus and execution client
// so they are disabled by default, only to be run manually.
var skip = false

// Not a test per se, just an util to fetch block and store them for mocking
func Test_GetFullBlockAtSlot(t *testing.T) {
	// Uncomment to run
	t.Skip("Skipping test")

	// Folder to store the result
	folder := "../mock"

	// Slot to fetch
	slotToFetch := uint64(5307527)
	fetchHeaderAndReceipts := true
	// fetchReceipts := true // TODO:

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f",
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

// TODO: This test is testing both FetchFullBlock and SummarizeBlock
func Test_FetchFullBlock(t *testing.T) {

	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	type test struct {
		// Input
		Name               string
		PoolAddress        string
		ProposerSubscribed bool
		Slot               uint64

		// Output
		ExpectedBlock          uint64
		ExpectedBlockType      BlockType
		ExpectedRewardType     RewardType
		ExpectedReward         *big.Int
		ExpectedValidatorIndex uint64
		ExpectedValKey         string
		ExpeectedWithCred      string
	}

	tests := []test{
		// subscribed validator proposes mev block with correct fee https://prater.beaconcha.in/slot/5739624
		{"1", "0xf4e8263979a89dc357d7f9f79533febc7f3e287b", true, uint64(5739624), uint64(9086632), OkPoolProposal, MevBlock, big.NewInt(23547931077241917), uint64(234515), "0xa2240e4a358a4f87dfece4c85f08b41abda91b558fe2e544885ed21163681576f41af2ec0161955c735803adb5fee910", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// subscribed validator proposes vanila block with correct fee https://prater.beaconcha.in/slot/5739629
		{"2", "0x94750381be1aba0504c666ee1db118f68f0780d4", true, uint64(5739629), uint64(9086637), OkPoolProposal, VanilaBlock, big.NewInt(15960095948338108), uint64(426736), "0xb6283b7cc2eaedde6f0ced4bffb8bc99c1e9cb3de77d6be8be02bf78fa850b74ee57f6b960fc48ca0ccd4b683521f3f9", "0x59b0d71688da01057c08e4c1baa8faa629819c2a"},

		// non subscribed validator proposes vanila block with correct fee https://prater.beaconcha.in/slot/5739634
		{"3", "0xa111B576408B1CcDacA3eF26f22f082C49bcaa55", false, uint64(5739634), uint64(9086639), OkPoolProposal, VanilaBlock, big.NewInt(41035389197072885), uint64(408206), "0xa57f9cbd211d3219ac54c8f329d1e2a4c65c54978444d7e5ff71d6129dd33ebc2e26bdfd611fc391a7a84b4d43418ac0", "0xa111b576408b1ccdaca3ef26f22f082c49bcaa55"},

		// non subscribed validator proposes mev block with correct fee https://prater.beaconcha.in/slot/5739644
		{"4", "0xF4e8263979A89Dc357d7f9F79533Febc7f3e287B", false, uint64(5739644), uint64(9086648), OkPoolProposal, MevBlock, big.NewInt(37799556930427516), uint64(234604), "0xb67e026940ccc26a478dcb020767d1391ccd6dc1f66f5bee328750cbbc4eb909665f7340c58411b6c29c01bdca3951c4", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// subscribed validator proposes a mev block with wrong fee recipient https://prater.beaconcha.in/slot/5739624
		{"5", "0x0000000000000000000000000000000000000000", true, uint64(5739624), uint64(9086632), WrongFeeRecipient, MevBlock, big.NewInt(23547931077241917), uint64(234515), "0xa2240e4a358a4f87dfece4c85f08b41abda91b558fe2e544885ed21163681576f41af2ec0161955c735803adb5fee910", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// subscribed validator proposes a vanila block with wrong fee recipient https://prater.beaconcha.in/slot/5739637
		{"6", "0x0000000000000000000000000000000000000000", true, uint64(5739637), uint64(9086642), WrongFeeRecipient, VanilaBlock, big.NewInt(11591726353544658), uint64(468452), "0x8371d199579f91a966732bf5eaaa940ac037084f95018ddd6530f9003c6b028f0181f52b50bdbe692f49f72c6fc9ad38", "0x0158fea37a1654d872c19f8326df00b7cb07c5cf"},

		// non subscribed validator proposes a block with wrong fee recipient (kind of ignored) https://prater.beaconcha.in/slot/5739637
		{"7", "0x0000000000000000000000000000000000000000", false, uint64(5739637), uint64(9086642), WrongFeeRecipient, UnknownRewardType, big.NewInt(0), uint64(468452), "0x8371d199579f91a966732bf5eaaa940ac037084f95018ddd6530f9003c6b028f0181f52b50bdbe692f49f72c6fc9ad38", "0x0158fea37a1654d872c19f8326df00b7cb07c5cf"},

		// subscribed validator misses a block https://prater.beaconcha.in/slot/5739640
		{"8", "0x0000000000000000000000000000000000000000", true, uint64(5739640), uint64(0), MissedProposal, UnknownRewardType, big.NewInt(0), uint64(458817), "0xb3fda21f2e4d6d93432d0d70c83c81159b2c625576eadbab80a2b55538ebd54a975cdc8a5cbb3909bbbb02bd08a3a009", "0x0997fdeffd9d29710436b2155ed702d845f7061a"},

		// unsubscribed validator misses a block (kind of ignored) https://prater.beaconcha.in/slot/5739640
		{"9", "0x0000000000000000000000000000000000000000", false, uint64(5739640), uint64(0), MissedProposal, UnknownRewardType, big.NewInt(0), uint64(458817), "0xb3fda21f2e4d6d93432d0d70c83c81159b2c625576eadbab80a2b55538ebd54a975cdc8a5cbb3909bbbb02bd08a3a009", "0x0997fdeffd9d29710436b2155ed702d845f7061a"},

		// subscribed validator proposes a block with correct fee recipient but BLS credentials (note: this test can fail if withdrawal is updated) https://prater.beaconcha.in/slot/5739736
		{"10", "0xe0a2Bd4258D2768837BAa26A28fE71Dc079f84c7", true, uint64(5739736), uint64(9086730), OkPoolProposalBlsKeys, VanilaBlock, big.NewInt(12805869897561244), uint64(319479), "0xb3e1c989c0d27824da29480a4bc09f4c561c2ce75d0a2ba7b3a57480d93d5ddb627d5fa0923402fd33145ded5eaa9d98", "0x95068c3ce9e71d7d4ca51df4230045e150d28d6c49727cb0d994d50b1cdeff"},

		// non subscribed validator proposes a vanila block with a wrong fee recipient (kind of ignored) most blocks are this https://prater.beaconcha.in/slot/5739707
		// reward is not calculated as its very expensive
		{"11", "0x0000000000000000000000000000000000000000", false, uint64(5739707), uint64(9086704), WrongFeeRecipient, UnknownRewardType, big.NewInt(0), uint64(474819), "0xa20fb16d127a22c7502e70db4eef33d1f11070d8bb232c91bf2b8beeadae8836d02774f7b5e96893ed80e9c7020e0d2a", "0x5bdd7b7a48d146b23969218eac5f152760bc072e"},

		// non subscribed validator proposes a mev block with a wrong fee recipient (kind of ignored) most blocks are this https://prater.beaconcha.in/slot/5739722
		// reward is calculated. not used but cheap to calculate it
		{"12", "0x0000000000000000000000000000000000000000", false, uint64(5739722), uint64(9086717), WrongFeeRecipient, MevBlock, big.NewInt(28327464143130026), uint64(232204), "0xb1294f2c149ee1cd0b2d9dd8bd8781cb4920353623426e64eb4a915b553c4dbefea53bc8c83f6b3dcee44223bdcd3c6c", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			oracle := NewOracle(&Config{})
			oracle.state.Config.PoolAddress = tt.PoolAddress
			onchain.CliCfg.PoolAddress = tt.PoolAddress

			if tt.ProposerSubscribed {
				oracle.addSubscriptionIfNotAlready(tt.ExpectedValidatorIndex, "0x", "0x")
			}

			fullBlock := onchain.FetchFullBlock(tt.Slot, oracle)
			block := fullBlock.SummarizedBlock(oracle, tt.PoolAddress)

			require.Equal(t, tt.Slot, block.Slot)
			require.Equal(t, tt.ExpectedBlock, block.Block)
			require.Equal(t, tt.ExpectedBlockType, block.BlockType)
			require.Equal(t, tt.ExpectedRewardType, block.RewardType)
			require.Equal(t, tt.ExpectedReward, block.Reward)
			require.Equal(t, tt.ExpectedValidatorIndex, block.ValidatorIndex)
			require.Equal(t, tt.ExpectedValKey, block.ValidatorKey)
			require.Equal(t, tt.ExpeectedWithCred, block.WithdrawalAddress)
		})
	}
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

func Test_GetDonationEvents(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	/* TODO:
	var cfgOnchain = &config.CliConfig{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x8eba4a4a8d4dfa78bcb734efd1ea9f33b61e3243",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	oracle := NewOracle(&Config{})

	// 1) contains a donation
	// https://goerli.etherscan.io/tx/0x789a23de09eab6b5b252cadefe9df35a7a2cd85a6ae4dbccea4f0a346977ca5f
	slotNum1 := uint64(5803850)
	blockNum1 := uint64(9139046)

	// 2) doesnt contain anything
	// https://goerli.etherscan.io/block/9139051
	slotNum2 := uint64(5803862)
	blockNum2 := uint64(9139055)

	// 3) contains only a mev reward
	slotNum3 := uint64(5798095)
	blockNum3 := uint64(9134612)

	block1 := onchain.GetBlockFromSlot(slotNum1, oracle)
	require.NoError(t, err)
	require.Equal(t, blockNum1, block1.Block)
	require.Equal(t, slotNum1, block1.Slot)
	require.Equal(t, uint64(466564), block1.ValidatorIndex)
	require.Equal(t, "0xdc62f9e8c34be08501cdef4ebde0a280f576d762", block1.WithdrawalAddress)

	block2 := onchain.GetBlockFromSlot(slotNum2, oracle)
	require.NoError(t, err)

	block3 := onchain.GetBlockFromSlot(slotNum3, oracle)
	require.NoError(t, err)

	donatons1, err := onchain.GetDonationEvents(blockNum1, block1)
	require.NoError(t, err)
	require.Equal(t, 1, len(donatons1))
	require.Equal(t, big.NewInt(3000000000000000000), donatons1[0].AmountWei)
	require.Equal(t, uint64(9139046), donatons1[0].Block)
	require.Equal(t, "0x789a23de09eab6b5b252cadefe9df35a7a2cd85a6ae4dbccea4f0a346977ca5f", donatons1[0].TxHash)

	donatons2, err := onchain.GetDonationEvents(blockNum2, block2)
	require.NoError(t, err)
	require.Equal(t, 0, len(donatons2))

	donatons3, err := onchain.GetDonationEvents(blockNum3, block3)
	require.NoError(t, err)
	require.Equal(t, 0, len(donatons3))
	*/

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
	oracleInstance.SetBeaconValidators(onchain.Validators()) // TODO: dirty having both

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

	oracleInstance.StoreLatestOnchainState()
	//oracleInstance.SaveStateToFile()
	//oracleInstance.SaveToJson()

	//require.Equal(t, "0x000000", oracleInstance.LatestCommitedState().MerkleRoot)

	// root: 0xf0ecfb7afe96f7f7b570598c71aaac8fae3e6880e078227106e3a29446d5dbf8
	//AccumulatedBalanceWei=131064623899584732 LeafHash=7c049ecd5a07fc5b7d39573db41a1faca70a798112583dce61b0c8761eaa2166 WithdrawalAddress=0xe46f9be81f9a3aca1808bb8c36d353436bb96091
	//AccumulatedBalanceWei=177243009873641532 LeafHash=83303f7cf1e36186b6d97de80db49c77fca6fd2a4fcdac771b4139d46b9abd1c WithdrawalAddress=0xa111b576408b1ccdaca3ef26f22f082c49bcaa55
}
