package oracle

import (
	"math/big"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/stretchr/testify/require"
)

func Test_ToBytes20(t *testing.T) {
	test1 := ToBytes20([]byte("this is a test"))
	test2 := ToBytes20([]byte("this is another longer test"))
	require.Equal(t, [20]uint8{0x74, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x74, 0x65, 0x73, 0x74, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, test1)
	require.Equal(t, [20]uint8{0x74, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x6e, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x20, 0x6c, 0x6f, 0x6e, 0x67}, test2)
}

func Test_DecodeTx(t *testing.T) {
	// Mainnet tx: 0x8984591d8415482f1638d0893c0febf55fd713ab6fd069ac02f395a623c72a9c
	rawTx1 := bellatrix.Transaction{248, 110, 129, 174, 133, 2, 150, 3, 101, 156, 130, 109, 96, 148, 56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151, 137, 23, 72, 83, 127, 19, 188, 52, 12, 6, 128, 38, 160, 54, 233, 9, 131, 116, 183, 92, 228, 28, 83, 106, 15, 104, 152, 63, 158, 150, 130, 189, 164, 176, 53, 190, 148, 106, 212, 134, 54, 80, 159, 125, 183, 160, 14, 60, 201, 32, 36, 154, 2, 147, 213, 195, 248, 4, 221, 44, 235, 32, 1, 49, 12, 26, 221, 246, 230, 135, 248, 37, 220, 140, 58, 55, 117, 204}
	tx1, msg1, err := DecodeTx(rawTx1)
	require.NoError(t, err)
	require.Equal(t, tx1.To().String(), "0x388C818CA8B9251b393131C08a736A67ccB19297")
	require.Equal(t, msg1.From().String(), "0xbd3Afb0bB76683eCb4225F9DBc91f998713C3b01")

	// Mainnet tx: 0x04f8069ebbcfe6169e42fb880e084541369a3b9348cde28c0e63d7ef9ea7d7ec
	rawTx2 := bellatrix.Transaction{2, 248, 113, 1, 131, 1, 235, 156, 128, 133, 3, 138, 43, 116, 33, 130, 82, 8, 148, 203, 250, 136, 64, 68, 84, 109, 85, 105, 226, 171, 255, 63, 180, 41, 48, 27, 97, 86, 42, 135, 209, 4, 207, 48, 167, 232, 100, 128, 192, 1, 160, 231, 96, 155, 44, 168, 65, 53, 57, 47, 197, 200, 232, 81, 67, 183, 6, 244, 187, 193, 52, 34, 8, 209, 217, 37, 226, 87, 27, 223, 205, 7, 199, 160, 113, 195, 124, 35, 35, 216, 255, 145, 88, 118, 134, 134, 42, 193, 6, 95, 25, 176, 124, 172, 249, 43, 250, 196, 217, 37, 35, 53, 151, 103, 232, 120}
	tx2, msg2, err := DecodeTx(rawTx2)
	require.Equal(t, tx2.To().String(), "0xcBfa884044546d5569E2abFf3fB429301b61562A")
	require.Equal(t, msg2.From().String(), "0xDAFEA492D9c6733ae3d56b7Ed1ADB60692c98Bc5")

	// 0x604c575a6dfce8154411613fdc2a768c906631fb769a61baf098908a140447b5 (mainnettype 2 eip1559, erc20 transfer)
	rawTx3 := bellatrix.Transaction{2, 248, 179, 1, 130, 11, 30, 132, 7, 34, 115, 24, 133, 46, 144, 237, 208, 0, 131, 3, 13, 64, 148, 218, 193, 127, 149, 141, 46, 229, 35, 162, 32, 98, 6, 153, 69, 151, 193, 61, 131, 30, 199, 128, 184, 68, 169, 5, 156, 187, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 74, 163, 102, 249, 217, 236, 76, 164, 107, 175, 87, 143, 196, 188, 19, 122, 235, 129, 141, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 250, 240, 128, 192, 1, 160, 19, 247, 136, 48, 89, 40, 124, 171, 95, 144, 175, 216, 212, 90, 244, 115, 173, 192, 198, 15, 17, 163, 230, 164, 128, 83, 184, 120, 19, 135, 16, 103, 160, 40, 217, 81, 9, 212, 49, 76, 102, 164, 238, 99, 138, 229, 190, 144, 157, 207, 215, 199, 92, 248, 76, 160, 243, 33, 56, 188, 169, 242, 232, 219, 240}
	tx3, msg3, err := DecodeTx(rawTx3)
	require.Equal(t, tx3.To().String(), "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	require.Equal(t, msg3.From().String(), "0xaF8162eaE1253ea5Ce016B9DF1EA779993dFb826")
}

func Test_SumAndSaturate(t *testing.T) {
	test1 := SumAndSaturate(big.NewInt(5), big.NewInt(5), big.NewInt(1))
	require.Equal(t, big.NewInt(1), test1)

	test2 := SumAndSaturate(big.NewInt(5), big.NewInt(5), big.NewInt(5))
	require.Equal(t, big.NewInt(5), test2)

	test3 := SumAndSaturate(big.NewInt(500), big.NewInt(700), big.NewInt(1000000))
	require.Equal(t, big.NewInt(1200), test3)
}
