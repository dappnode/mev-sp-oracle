package oracle

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	// TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"
	"mev-sp-oracle/config"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var cfgFetcher = config.Config{
	ConsensusEndpoint: "http://127.0.0.1:5051",
	ExecutionEndpoint: "http://127.0.0.1:8545",
}
var fetcherTest = NewFetcher(cfgFetcher)

// remove. just debuging some stuff
func Test_FetchFromExecution(t *testing.T) {
	// TODO: Move this to fetcher in case we need to access some parameters.

	account := common.HexToAddress("0xf573d99385c05c23b24ed33de616ad16a43a0919")
	balance, err := fetcherTest.ExecutionClient.BalanceAt(context.Background(), account, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(balance) // 25893180161173005034

}

// TODO convert this test func into a utility. Fetches blocks from beaconchain
// and dumps them into a serialized file.
func Test_GetBellatrixBlockAtSlot(t *testing.T) {

	var cfgFetcher = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	var fetcher = NewFetcher(cfgFetcher)
	folder := "../mock"
	blockType := "bellatrix"
	slotToFetch := "5344344"
	network := "mainnet"

	// Get block
	signedBeaconBlock, err := fetcher.GetBlockAtSlot(slotToFetch)
	require.NoError(t, err)

	// Serialize and dump the block to a file
	mbeel, err := signedBeaconBlock.Bellatrix.MarshalJSON()
	require.NoError(t, err)
	nameBlock := "block_" + blockType + "_slot_" + slotToFetch + "_" + network
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
	nameHeader := "header_" + blockType + "_slot_" + slotToFetch + "_" + network
	fheader, err := os.Create(filepath.Join(folder, nameHeader))
	require.NoError(t, err)
	defer fheader.Close()
	err = binary.Write(fheader, binary.LittleEndian, serializedHeader)
	require.NoError(t, err)

	// Get tx receipts, serialize and dump to file
	nameTxReceipts := "txreceipts_" + blockType + "_slot_" + slotToFetch + "_" + network
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

func Test_Retrieve(t *testing.T) {
	// TODO check errors
	blockJson, err := os.Open("../mock/block_bellatrix_slot_5344344_mainnet")
	require.NoError(t, err)
	blockByte, err := ioutil.ReadAll(blockJson)
	log.Info("dasda", blockByte[0])
	var bellatrixblock bellatrix.SignedBeaconBlock
	err = bellatrixblock.UnmarshalJSON(blockByte)

	var headerBlock types.Header
	headerJson, err := os.Open("../mock/header_bellatrix_slot_5344344_mainnet")
	headerByte, err := ioutil.ReadAll(headerJson)
	err = headerBlock.UnmarshalJSON(headerByte)
	require.NoError(t, err)
	log.Info(headerBlock.Number.String())

	var txReceipts []types.Receipt
	txReceiptsJson, err := os.Open("../mock/txreceipts_bellatrix_slot_5344344_mainnet")
	//log.Info("--", txReceiptsJson.Read())
	require.NoError(t, err)
	txReceiptsByte, err := ioutil.ReadAll(txReceiptsJson)
	//log.Info(txReceiptsByte)
	err = json.Unmarshal(txReceiptsByte, &txReceipts)
	require.NoError(t, err)
	log.Info(txReceipts[0].Status)

}

func Test_Example_Data(t *testing.T) {

	signedBeaconBlock, err := fetcherTest.GetBlockAtSlot("5214140")
	require.NoError(t, err)

	log.Info(signedBeaconBlock.Version)
	log.Info(signedBeaconBlock.Bellatrix.Message.Slot)
	// we have feerec and mev-fee rec.
	log.Info("FeeRecipient", signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String())
	log.Info(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.Timestamp)
	for i, rawTx := range signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.Transactions {
		tx := new(types.Transaction)
		err := tx.UnmarshalBinary(rawTx)
		if err == nil {
			//log.Info(i, " ", tx.ChainId().String())
			//log.Info(i, " ", tx.Data())
			//log.Info(i, " ", tx.Cost())
			//log.Info(i, " ", tx.Type())
			log.Info(i, " ", tx.To().String())
			log.Info(i, " ", tx.Value())
			log.Info(i, " ", tx.Hash())
			signer := types.NewEIP155Signer(tx.ChainId())
			msg, err := tx.AsMessage(signer, nil)
			if err == nil {
				log.Info("from ", i, " ", msg.From().String())
			} else {
				log.Error(err)
			}
		} else {
			log.Error(err)
		}
	}

	require.Equal(t, 1, 1)
}
