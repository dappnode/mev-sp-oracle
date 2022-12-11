package oracle

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	// TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"
	"mev-sp-oracle/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var cfgFetcher = config.Config{
	ConsensusEndpoint: "localhost:5051",
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

func Test_GetBlockAtSlot(t *testing.T) {

	// 5107234 my block. vanila.
	// slot: 5320341 block: 16153706 few txs only 13.

	signedBeaconBlock, err := fetcherTest.GetBlockAtSlot("5320342")
	require.NoError(t, err)

	log.Info(signedBeaconBlock.Version)
	log.Info(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String())

	mbeel, err := signedBeaconBlock.Bellatrix.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(mbeel)
	//log.Info(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.Timestamp)

	client, err := ethclient.Dial("http://127.0.0.1:8545")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("number!!:", signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.BlockNumber)

	header, err := client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.BlockNumber))
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(header.MarshalJSON())

	// little-endian to big-endian
	var baseFeePerGasBEBytes [32]byte
	for i := 0; i < 32; i++ {
		baseFeePerGasBEBytes[i] = signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas[32-1-i]
	}
	baseFeePerGas := new(big.Int).SetBytes(baseFeePerGasBEBytes[:])

	tips := big.NewInt(0)
	for i, rawTx := range signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.Transactions {
		tx, msg, err := DecodeTx(rawTx)
		if err == nil {
			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			//lol, err := receipt.MarshalJSON()
			if err != nil {
				log.Fatal(err)
			}
			//fmt.Println(i, " ", lol)
			_ = i
			fmt.Println(" ")
			if err != nil {
				log.Fatal(err)
			}
			tipFee := new(big.Int)

			switch tx.Type() {
			case 0:
				tipFee.Mul(tx.GasPrice(), big.NewInt(int64(receipt.GasUsed)))
			case 1: //same as 0
				tipFee.Mul(tx.GasPrice(), big.NewInt(int64(receipt.GasUsed)))
			case 2:
				val1 := new(big.Int).Add(msg.GasTipCap(), header.BaseFee)
				usedGasPrice := new(big.Int)
				if val1.Cmp(msg.GasFeeCap()) >= 0 {
					usedGasPrice = msg.GasFeeCap() // saturate
					//log.Info("saturate:", usedGasPrice)
					//log.Info("GasFeeCap: ", msg.GasFeeCap())
					//log.Info("GasTipCap: ", msg.GasTipCap())
				} else {
					usedGasPrice = val1
				}
				//realPrice := new(big.Int).Min(val1, big.NewInt(int64(receipt.GasUsed)))
				// TODO limit in baseFee?
				tipFee = new(big.Int).Mul(usedGasPrice, big.NewInt(int64(receipt.GasUsed)))
			default:
				log.Fatal("unknown tx type")
			}
			//log.Info(i, " ", "tipFee:", tipFee)
			tips = tips.Add(tips, tipFee)

			if strings.ToLower(msg.From().String()) == strings.ToLower(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String()) {
				log.Info("-----found")
			}
		} else {
			log.Fatal(err)
		}
	}
	burnt := new(big.Int).Mul(big.NewInt(int64(signedBeaconBlock.Bellatrix.Message.Body.ExecutionPayload.GasUsed)), baseFeePerGas)
	proposerReward := new(big.Int).Sub(tips, burnt)

	log.Info("txfees:", tips)
	log.Info("burndeotherway:", burnt)
	log.Info("proposer rewards: ", proposerReward)
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
