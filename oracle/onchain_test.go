package oracle

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// None of this tests can be executed without a valid consensus and execution client
// so they are disabled by default, only to be run manually.
var skip = true

// Fetches the balance of a given address
func Test_FetchFromExecution(t *testing.T) {
	t.Skip("Skipping test")
	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onChain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)
	account := common.HexToAddress("0xf573d99385c05c23b24ed33de616ad16a43a0919")
	balance, err := onChain.ExecutionClient.BalanceAt(context.Background(), account, nil)
	require.NoError(t, err)
	expectedValue, ok := new(big.Int).SetString("25893180161173005034", 10)
	require.True(t, ok)
	require.Equal(t, expectedValue, balance)
}

// Utility that fetches some data and dumps it to a file
func Test_GetBellatrixBlockAtSlot(t *testing.T) {
	t.Skip("Skipping test")

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)
	folder := "../mock"
	blockType := "capella"
	network := "goerli"
	slotToFetch := uint64(5307527)

	// Get block
	signedBeaconBlock, err := onchain.GetConsensusBlockAtSlot(slotToFetch)
	require.NoError(t, err)

	// Cast to our custom extended block with extra methods
	extendedSignedBeaconBlock := NewFullBlock(signedBeaconBlock, nil, nil)

	// Serialize and dump the block to a file
	// Change this Bellatrix, Capella or any other block version
	// depending on which field you want to store
	mbeel, err := extendedSignedBeaconBlock.Capella.MarshalJSON()
	require.NoError(t, err)
	nameBlock := "block_" + blockType + "_slot_" + strconv.FormatInt(int64(slotToFetch), 10) + "_" + network
	fblock, err := os.Create(filepath.Join(folder, nameBlock))
	require.NoError(t, err)
	defer fblock.Close()
	err = binary.Write(fblock, binary.LittleEndian, mbeel)
	defer fblock.Close()

	// Get block header
	header, err := onchain.ExecutionClient.HeaderByNumber(context.Background(), new(big.Int).SetUint64(extendedSignedBeaconBlock.GetBlockNumber()))
	require.NoError(t, err)

	// Serialize and dump the block header to a file
	serializedHeader, err := header.MarshalJSON()
	require.NoError(t, err)
	nameHeader := "header_" + blockType + "_slot_" + strconv.FormatInt(int64(slotToFetch), 10) + "_" + network
	fheader, err := os.Create(filepath.Join(folder, nameHeader))
	require.NoError(t, err)
	defer fheader.Close()
	err = binary.Write(fheader, binary.LittleEndian, serializedHeader)
	require.NoError(t, err)

	// Get tx receipts, serialize and dump to file
	nameTxReceipts := "txreceipts_" + blockType + "_slot_" + strconv.FormatInt(int64(slotToFetch), 10) + "_" + network
	fTxs, err := os.Create(filepath.Join(folder, nameTxReceipts))
	require.NoError(t, err)
	defer fTxs.Close()

	var receiptsBlock []*types.Receipt
	for _, rawTx := range extendedSignedBeaconBlock.GetBlockTransactions() {
		tx, _, err := DecodeTx(rawTx)
		if err == nil {
			receipt, err := onchain.ExecutionClient.TransactionReceipt(context.Background(), tx.Hash())
			require.NoError(t, err)
			receiptsBlock = append(receiptsBlock, receipt)
		}
	}
	serializedReceipts, err := json.Marshal(receiptsBlock)
	require.NoError(t, err)
	err = binary.Write(fTxs, binary.LittleEndian, serializedReceipts)
	require.NoError(t, err)
}

func Test_GetBlock(t *testing.T) {
	t.Skip("Skipping test")

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0xf29ff96aaea6c9a1fba851f74737f3c069d4f1a9",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})

	state.AddSubscriptionIfNotAlready(219539, "0x", "0x")

	block, error := onchain.GetBlock(uint64(5739621), state)
	require.NoError(t, error)
	require.Equal(t, uint64(9086629), block.Block)
	require.Equal(t, uint64(5739621), block.Slot)
	require.Equal(t, OkPoolProposal, block.BlockType)
	require.Equal(t, OkPoolProposal, block.RewardType)
	require.Equal(t, big.NewInt(180), block.Reward)
	require.Equal(t, big.NewInt(219539), block.ValidatorIndex)
	require.Equal(t, "0xa701032684e8f1ce499e3e6f6bd2c10fb7c1d418048e230be5ef4c8eef44cf6f0918c800623491396db6464260d2cbd0", block.ValidatorKey)
	require.Equal(t, "", block.WithdrawalAddress)

	log.Info(block)
}

func Test_GetBlock_WrongFee(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x0000000000000000000000000000000000000000",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})

	// Proposer but sent to another address
	state.AddSubscriptionIfNotAlready(234515, "0x", "0x")
	block, error := onchain.GetBlock(uint64(5739624), state)
	require.NoError(t, error)

	expectedBlock := &Block{
		Block:             9086632,
		Slot:              5739624,
		BlockType:         WrongFeeRecipient,
		RewardType:        MevBlock,
		Reward:            big.NewInt(23547931077241917),
		ValidatorIndex:    uint64(234515),
		ValidatorKey:      "0xa2240e4a358a4f87dfece4c85f08b41abda91b558fe2e544885ed21163681576f41af2ec0161955c735803adb5fee910",
		WithdrawalAddress: "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f",
	}

	require.Equal(t, expectedBlock, &block)
	log.Info(block)
}

func Test_GetBlock_OkProposal_Mev(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x94750381bE1AbA0504C666ee1DB118F68f0780D4",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})

	// Proposer but sent to another address
	state.AddSubscriptionIfNotAlready(465713, "0x", "0x")

	block, error := onchain.GetBlock(uint64(5739625), state)
	require.NoError(t, error)

	expectedBlock := &Block{
		Block:             9086633,
		Slot:              5739625,
		BlockType:         OkPoolProposal,
		RewardType:        MevBlock,
		Reward:            big.NewInt(10710177301731360),
		ValidatorIndex:    uint64(465713),
		ValidatorKey:      "0xa6cb2b98d74a2f77b2921ab371e602921dd365bcb5832247c3e2dcd4803cc69be490defd1936af22b3692276a460ce2a",
		WithdrawalAddress: "0xdc62f9e8c34be08501cdef4ebde0a280f576d762",
	}

	require.Equal(t, expectedBlock, &block)
	log.Info(block)
}

func Test_GetBlock_OkProposal_Vanila(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x94750381bE1AbA0504C666ee1DB118F68f0780D4",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})
	state.AddSubscriptionIfNotAlready(426736, "0x", "0x")

	block, error := onchain.GetBlock(uint64(5739629), state)
	require.NoError(t, error)

	expectedBlock := &Block{
		Block:             9086637,
		Slot:              5739629,
		BlockType:         OkPoolProposal,
		RewardType:        VanilaBlock,
		Reward:            big.NewInt(15960095948338108),
		ValidatorIndex:    uint64(426736),
		ValidatorKey:      "0xb6283b7cc2eaedde6f0ced4bffb8bc99c1e9cb3de77d6be8be02bf78fa850b74ee57f6b960fc48ca0ccd4b683521f3f9",
		WithdrawalAddress: "0x59b0d71688da01057c08e4c1baa8faa629819c2a",
	}

	require.Equal(t, expectedBlock, &block)
	log.Info(block)
}

// The fee recipient is wrong but the validator is not subscribed
func Test_GetBlock_WrongFee_NotSubscribed(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x0000000000000000000000000000000000000000",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})
	block, error := onchain.GetBlock(uint64(5739637), state)
	require.NoError(t, error)

	expectedBlock := &Block{
		Block:             9086642,
		Slot:              5739637,
		BlockType:         WrongFeeRecipient,
		RewardType:        UnknownRewardType,
		Reward:            big.NewInt(0),
		ValidatorIndex:    uint64(468452),
		ValidatorKey:      "0x8371d199579f91a966732bf5eaaa940ac037084f95018ddd6530f9003c6b028f0181f52b50bdbe692f49f72c6fc9ad38",
		WithdrawalAddress: "0x0158fea37a1654d872c19f8326df00b7cb07c5cf",
	}

	require.Equal(t, expectedBlock, &block)
	log.Info(block)
}

// The block was missed of a subscribed validator
func Test_GetBlock_Missed_Subscribed(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x0000000000000000000000000000000000000000",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})
	block, error := onchain.GetBlock(uint64(5739640), state)
	require.NoError(t, error)

	expectedBlock := &Block{
		Block:             0,
		Slot:              5739640,
		BlockType:         MissedProposal,
		RewardType:        UnknownRewardType,
		Reward:            big.NewInt(0),
		ValidatorIndex:    uint64(458817),
		ValidatorKey:      "0xb3fda21f2e4d6d93432d0d70c83c81159b2c625576eadbab80a2b55538ebd54a975cdc8a5cbb3909bbbb02bd08a3a009",
		WithdrawalAddress: "0x0997fdeffd9d29710436b2155ed702d845f7061a",
	}

	require.Equal(t, expectedBlock, &block)
	log.Info(block)
}

// Block with vanila reward is fetched but not proposed by a subscribed validator. Assert
// that the rewards are not calculated (which is an expensive calculation)
func Test_GetBlock_Irrelevant_Vanila(t *testing.T) {
	if skip {
		t.Skip("Skipping test")
	}

	var cfgOnchain = &config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
		PoolAddress:       "0x0000000000000000000000000000000000000000",
	}
	onchain, err := NewOnchain(cfgOnchain, nil)
	require.NoError(t, err)

	onchain.RefreshBeaconValidators()

	state := NewOracleState(&config.Config{})
	block, error := onchain.GetBlock(uint64(5739707), state)
	require.NoError(t, error)

	expectedBlock := &Block{
		Block:             9086704,
		Slot:              5739707,
		BlockType:         WrongFeeRecipient,
		RewardType:        UnknownRewardType,
		Reward:            big.NewInt(0),
		ValidatorIndex:    uint64(474819),
		ValidatorKey:      "0xa20fb16d127a22c7502e70db4eef33d1f11070d8bb232c91bf2b8beeadae8836d02774f7b5e96893ed80e9c7020e0d2a",
		WithdrawalAddress: "0x5bdd7b7a48d146b23969218eac5f152760bc072e",
	}

	require.Equal(t, expectedBlock, &block)
	log.Info(block)
}
