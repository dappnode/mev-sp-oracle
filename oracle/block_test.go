package oracle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var block1 = &bellatrix.SignedBeaconBlock{
	Message: &bellatrix.BeaconBlock{
		Slot:          5214140,
		ProposerIndex: 0,
		Body: &bellatrix.BeaconBlockBody{
			ExecutionPayload: &bellatrix.ExecutionPayload{
				FeeRecipient: [20]byte{56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151},
				BlockNumber:  0,
				Transactions: []bellatrix.Transaction{
					{1, 2},
					{1, 2}},
			},
		},
	},
}

func Test_FeeRecipientAndSlot(t *testing.T) {
	// Check that existing methods are inherited and new ones are extended
	extendedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: block1}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}
	require.Equal(t, "0x388c818ca8b9251b393131c08a736a67ccb19297", myBlock.GetFeeRecipient())
	require.Equal(t, uint64(5214140), uint64(myBlock.GetSlot()))
}

func Test_Bellatrix_TxType_0_Decode(t *testing.T) {
	// Mainnet tx: 0x8984591d8415482f1638d0893c0febf55fd713ab6fd069ac02f395a623c72a9c
	rawTx := bellatrix.Transaction{248, 110, 129, 174, 133, 2, 150, 3, 101, 156, 130, 109, 96, 148, 56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151, 137, 23, 72, 83, 127, 19, 188, 52, 12, 6, 128, 38, 160, 54, 233, 9, 131, 116, 183, 92, 228, 28, 83, 106, 15, 104, 152, 63, 158, 150, 130, 189, 164, 176, 53, 190, 148, 106, 212, 134, 54, 80, 159, 125, 183, 160, 14, 60, 201, 32, 36, 154, 2, 147, 213, 195, 248, 4, 221, 44, 235, 32, 1, 49, 12, 26, 221, 246, 230, 135, 248, 37, 220, 140, 58, 55, 117, 204}
	tx, msg, err := DecodeTx(rawTx)
	require.NoError(t, err)
	require.Equal(t, tx.Hash().String(), "0x8984591d8415482f1638d0893c0febf55fd713ab6fd069ac02f395a623c72a9c")

	// Type Legacy Tx (0) All values are the same
	require.Equal(t, big.NewInt(11106739612), tx.GasFeeCap())
	require.Equal(t, big.NewInt(11106739612), tx.GasPrice())
	require.Equal(t, big.NewInt(11106739612), tx.GasTipCap())

	require.Equal(t, uint64(28000), tx.Gas())
	require.Equal(t, big.NewInt(1), tx.ChainId())
	require.Equal(t, uint8(0), tx.Type())
	require.Equal(t, "0x388C818CA8B9251b393131C08a736A67ccB19297", tx.To().String())
	require.Equal(t, "0xbd3Afb0bB76683eCb4225F9DBc91f998713C3b01", msg.From().String())
	require.Equal(t, uint64(174), tx.Nonce())

	// This tx contains a value that would overflow an int
	expectedValue, ok := new(big.Int).SetString("429486762611856116742", 10)
	require.True(t, ok)
	require.Equal(t, expectedValue, tx.Value())
}

func Test_Bellatrix_TxType_2_Decode(t *testing.T) {
	// Mainnet tx: 0x04f8069ebbcfe6169e42fb880e084541369a3b9348cde28c0e63d7ef9ea7d7ec
	rawTx := bellatrix.Transaction{2, 248, 113, 1, 131, 1, 235, 156, 128, 133, 3, 138, 43, 116, 33, 130, 82, 8, 148, 203, 250, 136, 64, 68, 84, 109, 85, 105, 226, 171, 255, 63, 180, 41, 48, 27, 97, 86, 42, 135, 209, 4, 207, 48, 167, 232, 100, 128, 192, 1, 160, 231, 96, 155, 44, 168, 65, 53, 57, 47, 197, 200, 232, 81, 67, 183, 6, 244, 187, 193, 52, 34, 8, 209, 217, 37, 226, 87, 27, 223, 205, 7, 199, 160, 113, 195, 124, 35, 35, 216, 255, 145, 88, 118, 134, 134, 42, 193, 6, 95, 25, 176, 124, 172, 249, 43, 250, 196, 217, 37, 35, 53, 151, 103, 232, 120}
	tx, msg, err := DecodeTx(rawTx)
	require.NoError(t, err)
	require.Equal(t, tx.Gas(), uint64(21000))
	require.Equal(t, tx.Hash().String(), "0x04f8069ebbcfe6169e42fb880e084541369a3b9348cde28c0e63d7ef9ea7d7ec")

	require.Equal(t, big.NewInt(15203005473), tx.GasFeeCap())
	require.Equal(t, big.NewInt(15203005473), tx.GasPrice())
	require.Equal(t, big.NewInt(0), tx.GasTipCap())

	require.Equal(t, tx.ChainId(), big.NewInt(1))
	require.Equal(t, tx.Type(), uint8(2))

	// Note addresses are encoded in mixed case as per EIP-55
	require.Equal(t, "0xcBfa884044546d5569E2abFf3fB429301b61562A", tx.To().String())
	require.Equal(t, "0xDAFEA492D9c6733ae3d56b7Ed1ADB60692c98Bc5", msg.From().String())
	require.Equal(t, uint64(125852), tx.Nonce())
	require.Equal(t, big.NewInt(58833558053578852), tx.Value())
}

func Test_Bellatrix_TxType2WithERC20_Decode(t *testing.T) {
	// 0x604c575a6dfce8154411613fdc2a768c906631fb769a61baf098908a140447b5 (mainnettype 2 eip1559, erc20 transfer)
	rawTx := bellatrix.Transaction{2, 248, 179, 1, 130, 11, 30, 132, 7, 34, 115, 24, 133, 46, 144, 237, 208, 0, 131, 3, 13, 64, 148, 218, 193, 127, 149, 141, 46, 229, 35, 162, 32, 98, 6, 153, 69, 151, 193, 61, 131, 30, 199, 128, 184, 68, 169, 5, 156, 187, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 74, 163, 102, 249, 217, 236, 76, 164, 107, 175, 87, 143, 196, 188, 19, 122, 235, 129, 141, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 250, 240, 128, 192, 1, 160, 19, 247, 136, 48, 89, 40, 124, 171, 95, 144, 175, 216, 212, 90, 244, 115, 173, 192, 198, 15, 17, 163, 230, 164, 128, 83, 184, 120, 19, 135, 16, 103, 160, 40, 217, 81, 9, 212, 49, 76, 102, 164, 238, 99, 138, 229, 190, 144, 157, 207, 215, 199, 92, 248, 76, 160, 243, 33, 56, 188, 169, 242, 232, 219, 240}
	tx, msg, err := DecodeTx(rawTx)
	require.NoError(t, err)
	require.Equal(t, tx.Gas(), uint64(200000))
	require.Equal(t, tx.Hash().String(), "0x604c575a6dfce8154411613fdc2a768c906631fb769a61baf098908a140447b5")
	require.Equal(t, big.NewInt(200000000000), tx.GasFeeCap())
	require.Equal(t, big.NewInt(200000000000), tx.GasPrice())
	require.Equal(t, big.NewInt(119698200), tx.GasTipCap())
	require.Equal(t, big.NewInt(1), tx.ChainId())
	require.Equal(t, uint8(2), tx.Type())

	// Note addresses are encoded in mixed case as per EIP-55
	require.Equal(t, "0xdAC17F958D2ee523a2206206994597C13D831ec7", tx.To().String())
	require.Equal(t, "0xaF8162eaE1253ea5Ce016B9DF1EA779993dFb826", msg.From().String())
	require.Equal(t, uint64(2846), tx.Nonce())
	require.Equal(t, big.NewInt(0), tx.Value())
}

func Test_Bellatrix_GoerliTx_Decode(t *testing.T) {
	// Test also that works for goerli testnet
	// TODO
	//require.Equal(t, big.NewInt(5), tx.ChainId())
}

// Proposer tip of vanila block has to be calculated by adding all manually tips
// there is no field available, and it has to be manually recreated using all tx
// receipts present in that block.
func Test_GetProperTip_Mainnet_Slot_5320341(t *testing.T) {
	// Decode a a hardcode block/header/receipts
	fileName := "bellatrix_slot_5320341_mainnet"
	block, header, receipts := LoadBlockHeaderReceiptsBellatrix(fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &block}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}

	// Get proposer tip
	proposerTip, err := myBlock.GetProposerTip(&header, receipts)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1944763730864393), proposerTip)
}

// This block contains a tx with a smart contract deployment
// Tests if the fee is calculated properly as these tx
// are a bit different.
func Test_GetProperTip_Mainnet_Slot_5344344(t *testing.T) {
	fileName := "bellatrix_slot_5344344_mainnet"
	block, header, receipts := LoadBlockHeaderReceiptsBellatrix(fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &block}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}

	mevReward, numTxs, err := myBlock.MevRewardInWei("0x388c818ca8b9251b393131c08a736a67ccb19297")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(99952842017043014), mevReward)
	require.Equal(t, numTxs, 1)

	proposerTip, err := myBlock.GetProposerTip(&header, receipts)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(95434044627649514), proposerTip)
}

func Test_GetProperTip_Goerli_Slot_5214302(t *testing.T) {
	fileName := "capella_slot_5214302_goerli"
	block, header, receipts := LoadBlockHeaderReceiptsCapella(fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Capella: &block}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}

	proposerTip, err := myBlock.GetProposerTip(&header, receipts)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(38657065851824731), proposerTip)
}

func Test_GetMevReward_Goerli_Slot_5214321(t *testing.T) {
	fileName := "capella_slot_5214321_goerli"
	block, header, receipts := LoadBlockHeaderReceiptsCapella(fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Capella: &block}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}

	// Gets the MEV reward that was sent to a specific address
	mevReward, numTxs, err := myBlock.MevRewardInWei("0x4d496ccc28058b1d74b7a19541663e21154f9c84")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(15867629069461526), mevReward)
	require.Equal(t, numTxs, 1)

	// This block was a MEV block, but we can also test the tip
	proposerTip, err := myBlock.GetProposerTip(&header, receipts)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(15992505660349526), proposerTip)
}

func Test_MevReward_Slot_5320342(t *testing.T) {
	fileName := "bellatrix_slot_5320342_mainnet"
	block, _, _ := LoadBlockHeaderReceiptsBellatrix(fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &block}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}

	// Check that mev reward is correct and sent to the address
	mevReward1, numTxs1, err := myBlock.MevRewardInWei("0xf8636377b7a998b51a3cf2bd711b870b3ab0ad56")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(65184406499820485), mevReward1)
	require.Equal(t, numTxs1, 1)

	// Test that it also work for mixed case addresses EIP-55
	mevReward2, numTxs2, err := myBlock.MevRewardInWei("0xf8636377b7a998B51a3Cf2BD711B870B3Ab0Ad56")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(65184406499820485), mevReward2)
	require.Equal(t, numTxs2, 1)

	// Check that no mev was sent to a different address
	mevReward3, numTxs3, err := myBlock.MevRewardInWei("0x4de23f3f0fb3318287378adbde030cf61714b2f3")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(0), mevReward3)
	require.Equal(t, numTxs3, 0)
	require.Equal(t, "0xdafea492d9c6733ae3d56b7ed1adb60692c98bc5", myBlock.GetFeeRecipient())
}

// Test donated amount to a given address in a block
func Test_DonatedAmountInWei_Slot_5320342(t *testing.T) {
	fileName := "bellatrix_slot_5320342_mainnet"
	block, _, _ := LoadBlockHeaderReceiptsBellatrix(fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &block}
	myBlock := VersionedSignedBeaconBlock{&extendedBlock}

	// one donation is sent to this addres: 0x023aa0a3a580e7f3b4bcbb716e0fb6efd86ed25e
	donation1 := myBlock.DonatedAmountInWei("0x023aa0a3a580e7f3b4bcbb716e0fb6efd86ed25e")
	number, ok := new(big.Int).SetString("20000000000000000000", 10)
	require.Equal(t, donation1, number)
	require.Equal(t, ok, true)

	// two tx are done to this adress: 0xef1266370e603ad06cff8304b27f866ca444d434
	donation2 := myBlock.DonatedAmountInWei("0xef1266370e603ad06cff8304b27f866ca444d434")
	require.Equal(t, donation2, big.NewInt(3648455520393139))
}

// Util to load from file
func LoadBlockHeaderReceiptsBellatrix(file string) (bellatrix.SignedBeaconBlock, types.Header, []*types.Receipt) {
	blockJson, err := os.Open("../mock/block_" + file)
	if err != nil {
		log.Fatal(err)
	}
	blockByte, err := ioutil.ReadAll(blockJson)
	if err != nil {
		log.Fatal(err)
	}
	var bellatrixblock bellatrix.SignedBeaconBlock
	err = bellatrixblock.UnmarshalJSON(blockByte)
	if err != nil {
		log.Fatal(err)
	}

	var headerBlock types.Header
	headerJson, err := os.Open("../mock/header_" + file)
	headerByte, err := ioutil.ReadAll(headerJson)
	err = headerBlock.UnmarshalJSON(headerByte)
	if err != nil {
		log.Fatal(err)
	}

	var txReceipts []*types.Receipt
	txReceiptsJson, err := os.Open("../mock/txreceipts_" + file)
	txReceiptsByte, err := ioutil.ReadAll(txReceiptsJson)
	err = json.Unmarshal(txReceiptsByte, &txReceipts)
	if err != nil {
		log.Fatal(err)
	}

	return bellatrixblock, headerBlock, txReceipts
}

func LoadBlockHeaderReceiptsCapella(file string) (capella.SignedBeaconBlock, types.Header, []*types.Receipt) {
	blockJson, err := os.Open("../mock/block_" + file)
	if err != nil {
		log.Fatal(err)
	}
	blockByte, err := ioutil.ReadAll(blockJson)
	if err != nil {
		log.Fatal("could not read json file: ", err)
	}
	var capellaBlock capella.SignedBeaconBlock
	err = capellaBlock.UnmarshalJSON(blockByte)
	if err != nil {
		log.Fatal("could not unmarshal json into capella signed block:", err)
	}

	var headerBlock types.Header
	headerJson, err := os.Open("../mock/header_" + file)
	if err != nil {
		log.Fatal("could not open header file: ", err)
	}
	fmt.Println("jeader", headerJson)
	headerByte, err := ioutil.ReadAll(headerJson)
	if err != nil {
		log.Fatal("could not read header file: ", err)
	}
	err = headerBlock.UnmarshalJSON(headerByte)
	if err != nil {
		log.Fatal("could not unmarshal header block: ", err)
	}

	var txReceipts []*types.Receipt
	txReceiptsJson, err := os.Open("../mock/txreceipts_" + file)
	txReceiptsByte, err := ioutil.ReadAll(txReceiptsJson)
	err = json.Unmarshal(txReceiptsByte, &txReceipts)
	if err != nil {
		log.Fatal("could not unmarshal tx receipt: ", err)
	}

	return capellaBlock, headerBlock, txReceipts
}
