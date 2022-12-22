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

	// TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"
	"mev-sp-oracle/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// None of this tests can be executed without a valid consensus and execution client
// so they are disabled by default, only to be run manually.

// Fetches the balance of a given address
func Test_FetchFromExecution(t *testing.T) {
	t.Skip("Skipping test")
	var cfgFetcher = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	var fetcherTest = NewFetcher(cfgFetcher)
	account := common.HexToAddress("0xf573d99385c05c23b24ed33de616ad16a43a0919")
	balance, err := fetcherTest.ExecutionClient.BalanceAt(context.Background(), account, nil)
	require.NoError(t, err)
	expectedValue, ok := new(big.Int).SetString("25893180161173005034", 10)
	require.True(t, ok)
	require.Equal(t, expectedValue, balance)
}

// Utility that fetches some data and dumps it to a file
func Test_GetBellatrixBlockAtSlot(t *testing.T) {
	t.Skip("Skipping test")

	var cfgFetcher = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	var fetcher = NewFetcher(cfgFetcher)
	folder := "../mock"
	blockType := "bellatrix"
	slotToFetch := uint64(5320330)
	network := "mainnet"

	// Get block
	signedBeaconBlock, err := fetcher.GetBlockAtSlot(slotToFetch)
	require.NoError(t, err)

	// Serialize and dump the block to a file
	mbeel, err := signedBeaconBlock.Bellatrix.MarshalJSON()
	require.NoError(t, err)
	nameBlock := "block_" + blockType + "_slot_" + strconv.FormatInt(int64(slotToFetch), 10) + "_" + network
	fblock, err := os.Create(filepath.Join(folder, nameBlock))
	require.NoError(t, err)
	defer fblock.Close()
	err = binary.Write(fblock, binary.LittleEndian, mbeel)
	defer fblock.Close()

	// Get block header
	header, err := fetcher.ExecutionClient.HeaderByNumber(context.Background(), new(big.Int).SetUint64(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.BlockNumber))
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
	for _, rawTx := range signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.Transactions {
		tx, _, err := DecodeTx(rawTx)
		if err == nil {
			receipt, err := fetcher.ExecutionClient.TransactionReceipt(context.Background(), tx.Hash())
			require.NoError(t, err)
			receiptsBlock = append(receiptsBlock, receipt)
		}
	}
	serializedReceipts, err := json.Marshal(receiptsBlock)
	require.NoError(t, err)
	err = binary.Write(fTxs, binary.LittleEndian, serializedReceipts)
	require.NoError(t, err)
}
