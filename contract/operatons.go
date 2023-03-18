package contract

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	log "github.com/sirupsen/logrus"
)

// TODO: move with fetcher and call it chain interactions
type Operations struct {
	cfg             *config.Config
	ExecutionClient *ethclient.Client
}

func NewOperations(cfg *config.Config) *Operations {
	executionClient, err := ethclient.Dial(cfg.ExecutionEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	return &Operations{
		cfg:             cfg,
		ExecutionClient: executionClient,
	}
}

func (o *Operations) UpdateContractMerkleRoot(newMerkleRoot string) {

	log.Info("TODO: sanity check:", newMerkleRoot)

	newMerkleRootBytes := [32]byte{}
	unboundedBytes := common.Hex2Bytes(newMerkleRoot)
	log.Info("unboundedBytes:", unboundedBytes, " ", len(unboundedBytes))

	if len(unboundedBytes) != 32 {
		log.Fatal("wrong merkle root length: ", newMerkleRoot)
	}
	copy(newMerkleRootBytes[:], common.Hex2Bytes(newMerkleRoot))

	log.Info("new merkle root:", hex.EncodeToString(newMerkleRootBytes[:]))

	privateKey, err := crypto.HexToECDSA(o.cfg.DeployerPrivateKey)
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
	nonce, err := o.ExecutionClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal("could not get pending nonce: ", err)
	}

	gasTipCap, err := o.ExecutionClient.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal("could not get gas price suggestion: ", err)
	}

	// Unused, leaving for reference
	_ = gasTipCap

	chaindId, err := o.ExecutionClient.NetworkID(context.Background())
	if err != nil {
		log.Fatal("could not get chaind: ", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chaindId)
	if err != nil {
		log.Fatal("could not create NewKeyedTransactorWithChainID:", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0) // in wei
	//auth.GasLimit = uint64(300000) // in units

	// nil prices automatically estimate prices
	auth.GasPrice = nil
	auth.GasFeeCap = nil
	auth.GasTipCap = nil

	auth.Context = context.Background()
	auth.NoSend = false

	//address := common.HexToAddress(o.cfg.PoolAddress)
	// TODO: hardcoding a different address for testing
	address := common.HexToAddress("0x25eB524fAbe93979D299158a1c7D1FF6628e0356")

	instance, err := NewContract(address, o.ExecutionClient)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := instance.UpdateRewardsRoot(auth, newMerkleRootBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Tx sent updating the merkle root. Tx hash: ", tx.Hash().Hex())

	log.WithFields(log.Fields{
		"TxHash":            tx.Hash().Hex(),
		"NewMerkleRoot":     newMerkleRoot,
		"NewMerleRootBytes": newMerkleRootBytes,
	}).Info("Tx sent to Ethereum updating rewards merkle root, wait for confirmation")

	// TODO: Wait for confirmation of the tx and log if NOK
}
