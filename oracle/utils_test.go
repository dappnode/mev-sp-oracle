package oracle

import (
	"math/big"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/config"
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

func Test_GetUniqueElements(t *testing.T) {
	test1 := GetUniqueElements(
		[]string{
			"0xaaa",
			"0xaaa",
			"0xaaa",
			"0xbbb",
		})
	require.ElementsMatch(t, []string{"0xaaa", "0xbbb"}, test1)

	test2 := GetUniqueElements(
		[]string{
			"0xaaa",
			"0xaaa",
			"0xaaa",
			"0xaaa",
		})
	require.ElementsMatch(t, []string{"0xaaa"}, test2)

	test3 := GetUniqueElements(
		[]string{
			"0xaaa",
		})
	require.ElementsMatch(t, []string{"0xaaa"}, test3)

	test4 := GetUniqueElements(
		[]string{
			"0xaaa",
			"0xbbb",
			"0xccc",
		})
	require.ElementsMatch(t, []string{"0xaaa", "0xbbb", "0xccc"}, test4)
}

func Test_ByteArrayToStringArray(t *testing.T) {
	test1 := ByteArrayToStringArray([][]byte{
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	})
	require.Equal(t, "0x31,0x32,0x33", test1)
}

func Test_ByteArrayToArray(t *testing.T) {
	test1 := ByteArrayToArray([][]byte{
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	})
	require.Equal(t, []string{"0x31", "0x32", "0x33"}, test1)
}

func Test_SlotsToTime(t *testing.T) {
	require.Equal(t, "12 seconds", SlotsToTime(1))
	require.Equal(t, "2 minutes", SlotsToTime(10))
	require.Equal(t, "1 day 9 hours 20 minutes", SlotsToTime(10000))
}

func Test_StringToBlsKey(t *testing.T) {
	rec1 := StringToBlsKey("0x800010c20716ef4264a6d93b3873a008ece58fb9312ac2cc3b0ccc40aedb050f2038281e6a92242a35476af9903c7919")
	require.Equal(t, rec1, phase0.BLSPubKey{128, 0, 16, 194, 7, 22, 239, 66, 100, 166, 217, 59, 56, 115, 160, 8, 236, 229, 143, 185, 49, 42, 194, 204, 59, 12, 204, 64, 174, 219, 5, 15, 32, 56, 40, 30, 106, 146, 36, 42, 53, 71, 106, 249, 144, 60, 121, 25})

	rec2 := StringToBlsKey("800010c20716ef4264a6d93b3873a008ece58fb9312ac2cc3b0ccc40aedb050f2038281e6a92242a35476af9903c7919")
	require.Equal(t, rec2, phase0.BLSPubKey{128, 0, 16, 194, 7, 22, 239, 66, 100, 166, 217, 59, 56, 115, 160, 8, 236, 229, 143, 185, 49, 42, 194, 204, 59, 12, 204, 64, 174, 219, 5, 15, 32, 56, 40, 30, 106, 146, 36, 42, 53, 71, 106, 249, 144, 60, 121, 25})
}

func Test_NumInSlice(t *testing.T) {
	require.Equal(t, true, NumInSlice(1, []uint64{1, 2, 3}))
	require.Equal(t, false, NumInSlice(4, []uint64{1, 2, 3}))
	require.Equal(t, true, NumInSlice(2, []uint64{2, 2, 2}))
	require.Equal(t, false, NumInSlice(1000, []uint64{2, 2, 2}))
}

func Test_WithdrawalCredentials(t *testing.T) {
	blsKey1 := "00ed750cbdedaa39da69532eee649a5d3a202b310de2a6645af1dd7daca0fd22"
	blsKey2 := "00b9f30bfce35138f7638d68c1473d1d45693dae775166022a493f38d942deb5"
	eth1Key1 := "010000000000000000000000dc62f9e8c34be08501cdef4ebde0a280f576d762"
	eth1Key2 := "01000000000000000000000059b0d71688da01057c08e4c1baa8faa629819c2a"

	wrongKey1 := "098765"

	require.Equal(t, true, IsBlsType(blsKey1))
	require.Equal(t, true, IsBlsType(blsKey2))
	require.Equal(t, false, IsBlsType(eth1Key1))
	require.Equal(t, false, IsBlsType(eth1Key2))

	require.Equal(t, true, IsEth1Type(eth1Key1))
	require.Equal(t, true, IsEth1Type(eth1Key2))
	require.Equal(t, false, IsEth1Type(blsKey1))
	require.Equal(t, false, IsEth1Type(blsKey2))

	require.Equal(t, false, IsEth1Type(wrongKey1))
	require.Equal(t, false, IsBlsType(wrongKey1))

	_, err := GetEth1Address(blsKey1)
	require.Error(t, err)

	rec1, err := GetEth1Address(eth1Key1)
	require.NoError(t, err)
	require.Equal(t, rec1, "0xdc62f9e8c34be08501cdef4ebde0a280f576d762")

	rec2, err := GetEth1Address(eth1Key2)
	require.NoError(t, err)
	require.Equal(t, rec2, "0x59b0d71688da01057c08e4c1baa8faa629819c2a")

	b1, err := GetEth1AddressByte([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237})
	require.NoError(t, err)
	require.Equal(t, b1, "0x9427a30991170f917d7b83def6e44d26577871ed")
}

func Test_AreAddressEqual(t *testing.T) {
	require.Equal(t, true, AreAddressEqual("0x0000", "0x0000"))
}

func Test_GetUniqueDepositFromState(t *testing.T) {
	state1 := NewOracleState(&config.Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	state1.AddSubscriptionIfNotAlready(1, "0xa", "0x")
	state1.AddSubscriptionIfNotAlready(2, "0xa", "0x")
	state1.AddSubscriptionIfNotAlready(3, "0xa", "0x")
	state1.AddSubscriptionIfNotAlready(4, "0xb", "0x")
	state1.AddSubscriptionIfNotAlready(5, "0xc", "0x")

	unique1 := GetUniqueDepositFromState(state1)
	require.Equal(t, 3, len(unique1))
	require.ElementsMatch(t, []string{"0xa", "0xb", "0xc"}, unique1)

	state2 := NewOracleState(&config.Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	state2.AddSubscriptionIfNotAlready(1, "0xa", "0x")
	state2.AddSubscriptionIfNotAlready(2, "0xa", "0x")
	state2.AddSubscriptionIfNotAlready(3, "0xa", "0x")
	state2.AddSubscriptionIfNotAlready(4, "0xb", "0x")
	state2.AddSubscriptionIfNotAlready(5, "0xb", "0x")
	state2.AddSubscriptionIfNotAlready(6, "0x", "0x")
	state2.AddSubscriptionIfNotAlready(7, "0x", "0x")
	state2.AddSubscriptionIfNotAlready(8, "0xabcde", "0x")
	state2.AddSubscriptionIfNotAlready(9, "0xabcde", "0x")
	state2.AddSubscriptionIfNotAlready(9, "0x", "0x")

	unique2 := GetUniqueDepositFromState(state2)
	require.Equal(t, 4, len(unique2))
	require.ElementsMatch(t, []string{"0xa", "0xb", "0x", "0xabcde"}, unique2)

	state3 := NewOracleState(&config.Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	state3.AddSubscriptionIfNotAlready(1, "0x1", "0x")
	state3.AddSubscriptionIfNotAlready(2, "0x1", "0x")
	state3.AddSubscriptionIfNotAlready(3, "0x1", "0x")
	state3.AddSubscriptionIfNotAlready(4, "0x1", "0x")
	state3.AddSubscriptionIfNotAlready(5, "0x1", "0x")

	unique3 := GetUniqueDepositFromState(state3)
	require.Equal(t, 1, len(unique3))
	require.ElementsMatch(t, []string{"0x1"}, unique3)
}
