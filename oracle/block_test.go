package oracle

import (
	"math/big"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var block1 = &spec.VersionedSignedBeaconBlock{
	Version: spec.DataVersionBellatrix,
	Bellatrix: &bellatrix.SignedBeaconBlock{
		Message: &bellatrix.BeaconBlock{
			Slot:          5214140,
			ProposerIndex: 0,
			Body: &bellatrix.BeaconBlockBody{
				ExecutionPayload: &bellatrix.ExecutionPayload{
					FeeRecipient: [20]byte{},
					BlockNumber:  0,
					Transactions: []bellatrix.Transaction{
						{1, 2},
						{1, 2}},
				},
			},
		},
	},
}

// TODO: move to utils
func ToBytes20(x []byte) [20]byte {
	var y [20]byte
	copy(y[:], x)
	return y
}

// TODO: Tests are not complete

func Test_Legacy_0_Tx_Decode(t *testing.T) {
	// 0x8984591d8415482f1638d0893c0febf55fd713ab6fd069ac02f395a623c72a9c (mainnet legacy type 0)
	rawTx := bellatrix.Transaction{248, 110, 129, 174, 133, 2, 150, 3, 101, 156, 130, 109, 96, 148, 56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151, 137, 23, 72, 83, 127, 19, 188, 52, 12, 6, 128, 38, 160, 54, 233, 9, 131, 116, 183, 92, 228, 28, 83, 106, 15, 104, 152, 63, 158, 150, 130, 189, 164, 176, 53, 190, 148, 106, 212, 134, 54, 80, 159, 125, 183, 160, 14, 60, 201, 32, 36, 154, 2, 147, 213, 195, 248, 4, 221, 44, 235, 32, 1, 49, 12, 26, 221, 246, 230, 135, 248, 37, 220, 140, 58, 55, 117, 204}
	// TODO: test that this also works: 0xbd3Afb0bB76683eCb4225F9DBc91f998713C3b01
	tx, msg, err := DecodeTx(rawTx)
	require.NoError(t, err)
	require.Equal(t, tx.Gas(), uint64(28000))
	require.Equal(t, tx.ChainId(), big.NewInt(1))
	require.Equal(t, tx.Type(), uint8(0))
	require.Equal(t, tx.To().String(), "0x388C818CA8B9251b393131C08a736A67ccB19297")
	require.Equal(t, msg.From().String(), "0xbd3Afb0bB76683eCb4225F9DBc91f998713C3b01")

	// number, ok := new(big.Int).SetString("20000000000000000000", 10)
}

func Test_eip1559_2_Tx_Decode(t *testing.T) {
	// 0x04f8069ebbcfe6169e42fb880e084541369a3b9348cde28c0e63d7ef9ea7d7ec (mainnet type 2 eip1559)
	rawTx := bellatrix.Transaction{2, 248, 113, 1, 131, 1, 235, 156, 128, 133, 3, 138, 43, 116, 33, 130, 82, 8, 148, 203, 250, 136, 64, 68, 84, 109, 85, 105, 226, 171, 255, 63, 180, 41, 48, 27, 97, 86, 42, 135, 209, 4, 207, 48, 167, 232, 100, 128, 192, 1, 160, 231, 96, 155, 44, 168, 65, 53, 57, 47, 197, 200, 232, 81, 67, 183, 6, 244, 187, 193, 52, 34, 8, 209, 217, 37, 226, 87, 27, 223, 205, 7, 199, 160, 113, 195, 124, 35, 35, 216, 255, 145, 88, 118, 134, 134, 42, 193, 6, 95, 25, 176, 124, 172, 249, 43, 250, 196, 217, 37, 35, 53, 151, 103, 232, 120}
	tx, msg, err := DecodeTx(rawTx)
	require.NoError(t, err)
	log.Info("tx", tx.Gas())
	log.Info("tx", tx.ChainId()) // TODO: test in goerli
	log.Info("tx", tx.To())
	log.Info("tx", tx.Hash())
	log.Info("tx", tx.Value())
	log.Info("tx", tx.Type())
	log.Info("tx", msg.From())

	//TODO: Asserts
}

func Test_eip1559_2_ERC20_Decode(t *testing.T) {
	// 0x604c575a6dfce8154411613fdc2a768c906631fb769a61baf098908a140447b5 (mainnettype 2 eip1559, erc20 transfer)
	rawTx := bellatrix.Transaction{2, 248, 179, 1, 130, 11, 30, 132, 7, 34, 115, 24, 133, 46, 144, 237, 208, 0, 131, 3, 13, 64, 148, 218, 193, 127, 149, 141, 46, 229, 35, 162, 32, 98, 6, 153, 69, 151, 193, 61, 131, 30, 199, 128, 184, 68, 169, 5, 156, 187, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 74, 163, 102, 249, 217, 236, 76, 164, 107, 175, 87, 143, 196, 188, 19, 122, 235, 129, 141, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 250, 240, 128, 192, 1, 160, 19, 247, 136, 48, 89, 40, 124, 171, 95, 144, 175, 216, 212, 90, 244, 115, 173, 192, 198, 15, 17, 163, 230, 164, 128, 83, 184, 120, 19, 135, 16, 103, 160, 40, 217, 81, 9, 212, 49, 76, 102, 164, 238, 99, 138, 229, 190, 144, 157, 207, 215, 199, 92, 248, 76, 160, 243, 33, 56, 188, 169, 242, 232, 219, 240}
	tx, msg, err := DecodeTx(rawTx)
	require.NoError(t, err)
	log.Info("tx", tx.Gas())
	log.Info("tx", tx.ChainId()) // TODO: test in goerli
	log.Info("tx", tx.To())
	log.Info("tx", tx.Hash())
	log.Info("tx", tx.Value())
	log.Info("tx", tx.Type())
	log.Info("tx", msg.From())

	// TODO: add asserts
}

/*
func Test_keepincaseineedit(t *testing.T) {
	// 0x8984591d8415482f1638d0893c0febf55fd713ab6fd069ac02f395a623c72a9c transfer to smart contract Txn Type: 0 (Legacy)
	var tx1 = bellatrix.Transaction{248, 110, 129, 174, 133, 2, 150, 3, 101, 156, 130, 109, 96, 148, 56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151, 137, 23, 72, 83, 127, 19, 188, 52, 12, 6, 128, 38, 160, 54, 233, 9, 131, 116, 183, 92, 228, 28, 83, 106, 15, 104, 152, 63, 158, 150, 130, 189, 164, 176, 53, 190, 148, 106, 212, 134, 54, 80, 159, 125, 183, 160, 14, 60, 201, 32, 36, 154, 2, 147, 213, 195, 248, 4, 221, 44, 235, 32, 1, 49, 12, 26, 221, 246, 230, 135, 248, 37, 220, 140, 58, 55, 117, 204}

	feeRec, err := hexutil.Decode("0xbd3afb0bb76683ecb4225f9dbc91f998713c3b01")
	// TODO: test that this also works: 0xbd3Afb0bB76683eCb4225F9DBc91f998713C3b01
	//var tst [20]byte{}
	block1.Bellatrix.Message.Body.ExecutionPayload.Transactions = []bellatrix.Transaction{tx1}
	block1.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient = bellatrix.ExecutionAddress(ToBytes20(feeRec))

	myBlock := VersionedSignedBeaconBlock{*block1}
	require.NoError(t, err)
	_ = myBlock

	// TODO: test that this works: 0xbd3Afb0bB76683eCb4225F9DBc91f998713C3b01 mixed case
}*/

func Test_GetProperTip_Mainnet_16153706(t *testing.T) {
	var bellatrixBlock_16153706 bellatrix.SignedBeaconBlock
	err := bellatrixBlock_16153706.UnmarshalJSON(BellatrixBlock_16153706)
	require.NoError(t, err)
	log.Info(bellatrixBlock_16153706.Message.Body.ExecutionPayload.FeeRecipient.String())

	versionedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &bellatrixBlock_16153706}
	extendedBlock := VersionedSignedBeaconBlock{versionedBlock}

	var headerBlock_16153706 types.Header
	err = headerBlock_16153706.UnmarshalJSON(HeaderBlock_16153706)
	require.NoError(t, err)

	var receiptsBlock_16153706 []*types.Receipt
	for _, receipt := range ReceiptsBlock_16153706 {
		var decodedReceipt types.Receipt
		err = decodedReceipt.UnmarshalJSON(receipt)
		require.NoError(t, err)
		receiptsBlock_16153706 = append(receiptsBlock_16153706, &decodedReceipt)
	}

	proposerTip, err := extendedBlock.GetProposerTip(&headerBlock_16153706, receiptsBlock_16153706)
	require.NoError(t, err)
	log.Info(proposerTip.String())
	// 1944763730864393

}

// Addd more test for other blocks
// this block contains mev reward. to 0xf8636377b7a998b51a3cf2bd711b870b3ab0ad56
// check that it doesnt break for non mev blocks.
func Test_MevReward(t *testing.T) {
	var bellatrixBlock_16153707 bellatrix.SignedBeaconBlock
	err := bellatrixBlock_16153707.UnmarshalJSON(BellatrixBlock_16153707)
	require.NoError(t, err)
	log.Info(bellatrixBlock_16153707.Message.Body.ExecutionPayload.FeeRecipient.String())

	versionedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &bellatrixBlock_16153707}
	extendedBlock := VersionedSignedBeaconBlock{versionedBlock}

	amount, numTxs, err := extendedBlock.MevReward("0xf8636377b7a998b51a3cf2bd711b870b3ab0ad56")
	require.NoError(t, err)
	log.Info("nbew mev rewars, ", amount)
	require.Equal(t, amount, big.NewInt(65184406499820485))
	require.Equal(t, numTxs, 1)

}

// TODO: test with a block that contains mev reward AND a donation.
func Test_DonatedAmountInWei(t *testing.T) {
	var bellatrixBlock_16153707 bellatrix.SignedBeaconBlock
	err := bellatrixBlock_16153707.UnmarshalJSON(BellatrixBlock_16153707)
	require.NoError(t, err)

	versionedBlock := spec.VersionedSignedBeaconBlock{Bellatrix: &bellatrixBlock_16153707}
	extendedBlock := VersionedSignedBeaconBlock{versionedBlock}

	// one donation is sent to this addres: 0x023aa0a3a580e7f3b4bcbb716e0fb6efd86ed25e
	donation1, err := extendedBlock.DonatedAmountInWei("0x023aa0a3a580e7f3b4bcbb716e0fb6efd86ed25e")
	require.NoError(t, err)
	log.Info("donation1", donation1)
	number, ok := new(big.Int).SetString("20000000000000000000", 10)
	require.Equal(t, donation1, number)
	require.Equal(t, ok, true)
	log.Info("yolo:  ", number)

	// two tx are done to this adress: 0xef1266370e603ad06cff8304b27f866ca444d434
	donation2, err := extendedBlock.DonatedAmountInWei("0xef1266370e603ad06cff8304b27f866ca444d434")
	require.NoError(t, err)
	log.Info("donation2", donation2)
	require.Equal(t, donation2, big.NewInt(3648455520393139))

}
