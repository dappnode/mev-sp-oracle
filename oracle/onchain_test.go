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
	"github.com/stretchr/testify/require"
)

// None of this tests can be executed without a valid consensus and execution client
// so they are disabled by default, only to be run manually.

// Fetches the balance of a given address
func Test_FetchFromExecution(t *testing.T) {
	t.Skip("Skipping test")
	var cfgOnchain = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onChain, err := NewOnchain(cfgOnchain)
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

	var cfgOnchain = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfgOnchain)
	require.NoError(t, err)
	folder := "../mock"
	blockType := "capella"
	network := "goerli"
	slotToFetch := uint64(5307527)

	// Get block
	signedBeaconBlock, err := onchain.GetConsensusBlockAtSlot(slotToFetch)
	require.NoError(t, err)

	// Cast to our custom extended block with extra methods
	extendedSignedBeaconBlock := VersionedSignedBeaconBlock{signedBeaconBlock}

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

func Test_GetValidatorIndexByKey(t *testing.T) {
	t.Skip("Skipping test")
	var cfg = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfg)
	require.NoError(t, err)
	vals, err := onchain.GetValidatorIndexByKey("0x800010c20716ef4264a6d93b3873a008ece58fb9312ac2cc3b0ccc40aedb050f2038281e6a92242a35476af9903c7919")
	require.NoError(t, err)
	_ = vals // TODO
}

func Test_GetValidatorKeyByIndex(t *testing.T) {
	t.Skip("Skipping test")
	var cfg = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfg)
	require.NoError(t, err)
	vals, err := onchain.GetValidatorKeyByIndex(123)
	require.NoError(t, err)
	_ = vals // TODO
}
