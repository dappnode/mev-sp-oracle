package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func main() {
	executionClientEndpoint := "http://127.0.0.1:8545"
	privateKeyString := "UseYoursOnlyInTestnet" // Do not commit this, use only for testnet
	poolContractAddress := "0x553BD5a94bcC09FFab6550274d5db140a95AE9bC"
	collateralAmountWei := big.NewInt(0).SetUint64(10000000000000000) // 0.01 Eth

	executionClient, err := ethclient.Dial(executionClientEndpoint)
	if err != nil {
		log.Fatal("could not connect to execution client: ", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Println(fromAddress.Hex())

	// Unused, leaving for reference. We rely on automatic gas estimation, see below (nil values)
	gasTipCap, err := executionClient.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal("could not get gas price suggestion: ", err)
	}
	_ = gasTipCap

	chaindId, err := executionClient.NetworkID(context.Background())
	if err != nil {
		log.Fatal("could not get chaind: ", err)
	}

	address := common.HexToAddress(poolContractAddress)

	instance, err := contract.NewContract(address, executionClient)
	if err != nil {
		log.Fatal(err)
	}

	validatorIndexes := []uint32{
		//done: 408081, 408111, 408132, 408124, 408142, 408103, 408071, 408146, 408089, 408059, 408125, 408076, 408055, 408090, 408063, 408092, 408147, 408113, 408100, 408060, 408054, 408105, 408128, 408096, 408053,
		//done: 408075, 408148, 408135, 408080, 408084, 408082, 408056, 408115, 408123, 408126, 408112, 408119, 408116, 408070, 408099, 408074, 408065, 408078, 408094, 408057, 408086, 408072, 408145, 408095, 408085,
		//done: 408139, 408091, 408108, 408127, 408104,  408083, 408133, 408136, 408143,408118, 408088, 408066, 408068, 408117, 408067, 408130, 408079, 408107, 408101, 408149, 408058, 408102, 408097, 408106, 408134,
		//done : 408073, 408069, 408131, 408129, 408141, 408151, 408144, 408138, 408062, 408114, 408110, 408122, 408064, 408109, 408087, 408152, 408093, 408120, 408098, 408061, 408121, 408150, 408140, 408077, 408137,
	}
	//validatorIndexes := []uint32{ /*408081, 408111, 408132 408124,*/ 408142, 408103}

	for _, validatorIndex := range validatorIndexes {
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chaindId)
		if err != nil {
			log.Fatal("could not create NewKeyedTransactorWithChainID:", err)
		}
		nonce, err := executionClient.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Fatal("could not get pending nonce: ", err)
		}
		auth.Nonce = big.NewInt(int64(nonce))
		auth.Value = collateralAmountWei
		// nil prices automatically estimate prices
		// TODO: Perhaps overpay to make sure the tx is not stuck forever.
		auth.GasPrice = nil
		auth.GasFeeCap = nil
		auth.GasTipCap = nil
		auth.Context = context.Background()
		auth.NoSend = false

		tx, err := instance.SuscribeValidator(auth, validatorIndex)
		if err != nil {
			log.Fatal("could not call SuscribeValidator: ", err)
		}

		log.WithFields(log.Fields{
			"TxHash":         tx.Hash().Hex(),
			"CollateralWei":  collateralAmountWei,
			"ValidatorIndex": validatorIndex,
		}).Info("Tx sent to Ethereum subscribing validator")

		// Leave 5 minutes for the tx to be validated
		deadline := time.Now().Add(5 * time.Minute)
		ctx, cancelCtx := context.WithDeadline(context.Background(), deadline)
		defer cancelCtx()

		// It stops waiting when the context is canceled.
		receipt, err := bind.WaitMined(ctx, executionClient, tx)
		if ctx.Err() != nil {
			log.Fatal("Timeout expired for waiting for tx to be validated, txHash: ", tx.Hash().Hex(), " err:", err)
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatal("Tx failed, err: ", receipt.Status, " hash: ", tx.Hash().Hex())
		}

		// Tx was sent and validated correctly, print receipt info
		log.WithFields(log.Fields{
			"Status":            receipt.Status,
			"CumulativeGasUsed": receipt.CumulativeGasUsed,
			"TxHash":            receipt.TxHash,
			"GasUsed":           receipt.GasUsed,
			"BlockHash":         receipt.BlockHash.Hex(),
			"BlockNumber":       receipt.BlockNumber,
		}).Info("Tx: ", tx.Hash().Hex(), " was validated ok. Receipt info:")

		time.Sleep(30 * time.Second)
	}
}
