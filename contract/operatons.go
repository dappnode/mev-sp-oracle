package contract

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"mev-sp-oracle/config" // TODO: Change when pushed "github.com/dappnode/mev-sp-oracle/config"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	log "github.com/sirupsen/logrus"
)

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

	log.Info("new merkle root:", newMerkleRootBytes)

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
		log.Fatal(err)
	}

	gasPrice, err := o.ExecutionClient.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	chaindId, err := o.ExecutionClient.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chaindId)
	if err != nil {
		log.Fatal(err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0) // in wei
	//auth.GasLimit = uint64(300000) // in units
	// gasPrice works well in testnets?
	auth.GasPrice = gasPrice
	//auth.GasPrice = new(big.Int).SetUint64(10)
	auth.Context = context.Background() // TODO:
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

	log.Info("Tx sent: ", tx.Hash().Hex())
}