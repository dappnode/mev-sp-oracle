package oracle

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"mev-sp-oracle/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GenerateTreeFromState(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.DepositAddresses[0] = "0x1000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[0] = "0x0100000000000000000000000000000000000000"
	state.ClaimableRewards[0] = big.NewInt(10000)
	state.UnbanBalances[0] = big.NewInt(0)

	state.DepositAddresses[1] = "0x2000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[1] = "0x0200000000000000000000000000000000000000"
	state.ClaimableRewards[1] = big.NewInt(20000)
	state.UnbanBalances[1] = big.NewInt(0)

	state.DepositAddresses[2] = "0x3000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[2] = "0x0300000000000000000000000000000000000000"
	state.ClaimableRewards[2] = big.NewInt(30000)
	state.UnbanBalances[2] = big.NewInt(0)

	state.DepositAddresses[3] = "0x4000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[3] = "0x0400000000000000000000000000000000000000"
	state.ClaimableRewards[3] = big.NewInt(40000)
	state.UnbanBalances[3] = big.NewInt(10)

	state.DepositAddresses[4] = "0x5000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[4] = "0x0500000000000000000000000000000000000000"
	state.ClaimableRewards[4] = big.NewInt(50000)
	state.UnbanBalances[4] = big.NewInt(0)

	state.DepositAddresses[5] = "0x6000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[5] = "0x0600000000000000000000000000000000000000"
	state.ClaimableRewards[5] = big.NewInt(60000)
	state.UnbanBalances[5] = big.NewInt(0)

	// TODO: add test to _, _
	_, _, tree := merklelizer.GenerateTreeFromState(state)
	require.Equal(t, "bcdf38cc9218047f9e86184199c880bfe80fc9eec2ecff0da3484217e6ccc898", hex.EncodeToString(tree.Root))

}

func Test_AggregateValidatorsIndexes_NoAggregation(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.DepositAddresses[0] = "0x1000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[0] = "0x0100000000000000000000000000000000000000"
	state.ClaimableRewards[0] = big.NewInt(10000)
	state.UnbanBalances[0] = big.NewInt(0)

	state.DepositAddresses[1] = "0x2000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[1] = "0x0200000000000000000000000000000000000000"
	state.ClaimableRewards[1] = big.NewInt(20000)
	state.UnbanBalances[1] = big.NewInt(0)

	state.DepositAddresses[2] = "0x3000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[2] = "0x0300000000000000000000000000000000000000"
	state.ClaimableRewards[2] = big.NewInt(30000)
	state.UnbanBalances[2] = big.NewInt(0)

	state.DepositAddresses[3] = "0x4000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[3] = "0x0400000000000000000000000000000000000000"
	state.ClaimableRewards[3] = big.NewInt(40000)
	state.UnbanBalances[3] = big.NewInt(10)

	state.DepositAddresses[4] = "0x5000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[4] = "0x0500000000000000000000000000000000000000"
	state.ClaimableRewards[4] = big.NewInt(50000)
	state.UnbanBalances[4] = big.NewInt(0)

	state.DepositAddresses[5] = "0x6000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[5] = "0x0600000000000000000000000000000000000000"
	state.ClaimableRewards[5] = big.NewInt(60000)
	state.UnbanBalances[5] = big.NewInt(0)

	expected := []RawLeaf{
		{
			DepositAddress:   "0x1000000000000000000000000000000000000000",
			PoolRecipient:    "0x0100000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(10000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x2000000000000000000000000000000000000000",
			PoolRecipient:    "0x0200000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(20000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x3000000000000000000000000000000000000000",
			PoolRecipient:    "0x0300000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(30000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x4000000000000000000000000000000000000000",
			PoolRecipient:    "0x0400000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(40000),
			UnbanBalance:     new(big.Int).SetUint64(10),
		},
		{
			DepositAddress:   "0x5000000000000000000000000000000000000000",
			PoolRecipient:    "0x0500000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(50000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x6000000000000000000000000000000000000000",
			PoolRecipient:    "0x0600000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(60000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	fmt.Println(rawLeafs)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_NoAggregationOrdered(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.DepositAddresses[0] = "0x3000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[0] = "0x0300000000000000000000000000000000000000"
	state.ClaimableRewards[0] = big.NewInt(30000)
	state.UnbanBalances[0] = big.NewInt(0)

	state.DepositAddresses[1] = "0x6000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[1] = "0x0600000000000000000000000000000000000000"
	state.ClaimableRewards[1] = big.NewInt(60000)
	state.UnbanBalances[1] = big.NewInt(0)

	state.DepositAddresses[2] = "0x1000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[2] = "0x0100000000000000000000000000000000000000"
	state.ClaimableRewards[2] = big.NewInt(10000)
	state.UnbanBalances[2] = big.NewInt(0)

	state.DepositAddresses[3] = "0x4000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[3] = "0x0400000000000000000000000000000000000000"
	state.ClaimableRewards[3] = big.NewInt(40000)
	state.UnbanBalances[3] = big.NewInt(10)

	state.DepositAddresses[4] = "0x5000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[4] = "0x0500000000000000000000000000000000000000"
	state.ClaimableRewards[4] = big.NewInt(50000)
	state.UnbanBalances[4] = big.NewInt(0)

	state.DepositAddresses[5] = "0x2000000000000000000000000000000000000000"
	state.PoolRecipientAddresses[5] = "0x0200000000000000000000000000000000000000"
	state.ClaimableRewards[5] = big.NewInt(20000)
	state.UnbanBalances[5] = big.NewInt(0)

	expected := []RawLeaf{
		{
			DepositAddress:   "0x1000000000000000000000000000000000000000",
			PoolRecipient:    "0x0100000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(10000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x2000000000000000000000000000000000000000",
			PoolRecipient:    "0x0200000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(20000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x3000000000000000000000000000000000000000",
			PoolRecipient:    "0x0300000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(30000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x4000000000000000000000000000000000000000",
			PoolRecipient:    "0x0400000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(40000),
			UnbanBalance:     new(big.Int).SetUint64(10),
		},
		{
			DepositAddress:   "0x5000000000000000000000000000000000000000",
			PoolRecipient:    "0x0500000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(50000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
		{
			DepositAddress:   "0x6000000000000000000000000000000000000000",
			PoolRecipient:    "0x0600000000000000000000000000000000000000",
			ClaimableBalance: new(big.Int).SetUint64(60000),
			UnbanBalance:     new(big.Int).SetUint64(0),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	fmt.Println(rawLeafs)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_AggregationAll(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.DepositAddresses[0] = "0xaa"
	state.PoolRecipientAddresses[0] = "0xaa"
	state.ClaimableRewards[0] = big.NewInt(30000)
	state.UnbanBalances[0] = big.NewInt(0)

	state.DepositAddresses[1] = "0xaa"
	state.PoolRecipientAddresses[1] = "0xaa"
	state.ClaimableRewards[1] = big.NewInt(60000)
	state.UnbanBalances[1] = big.NewInt(0)

	state.DepositAddresses[2] = "0xaa"
	state.PoolRecipientAddresses[2] = "0xaa"
	state.ClaimableRewards[2] = big.NewInt(10000)
	state.UnbanBalances[2] = big.NewInt(0)

	state.DepositAddresses[3] = "0xaa"
	state.PoolRecipientAddresses[3] = "0xaa"
	state.ClaimableRewards[3] = big.NewInt(40000)
	state.UnbanBalances[3] = big.NewInt(10)

	state.DepositAddresses[4] = "0xaa"
	state.PoolRecipientAddresses[4] = "0xaa"
	state.ClaimableRewards[4] = big.NewInt(50000)
	state.UnbanBalances[4] = big.NewInt(0)

	state.DepositAddresses[5] = "0xaa"
	state.PoolRecipientAddresses[5] = "0xaa"
	state.ClaimableRewards[5] = big.NewInt(20000)
	state.UnbanBalances[5] = big.NewInt(0)

	expected := []RawLeaf{
		{
			DepositAddress:   "0xaa",
			PoolRecipient:    "0xaa",
			ClaimableBalance: new(big.Int).SetUint64(210000),
			UnbanBalance:     new(big.Int).SetUint64(10),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_Aggregation_And_Leftover(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.DepositAddresses[0] = "0xaa"
	state.PoolRecipientAddresses[0] = "0xaa"
	state.ClaimableRewards[0] = big.NewInt(30000)
	state.UnbanBalances[0] = big.NewInt(0)

	state.DepositAddresses[1] = "0xaa"
	state.PoolRecipientAddresses[1] = "0xaa"
	state.ClaimableRewards[1] = big.NewInt(60000)
	state.UnbanBalances[1] = big.NewInt(0)

	state.DepositAddresses[2] = "0xaa"
	state.PoolRecipientAddresses[2] = "0xaa"
	state.ClaimableRewards[2] = big.NewInt(10000)
	state.UnbanBalances[2] = big.NewInt(0)

	state.DepositAddresses[3] = "0xaa"
	state.PoolRecipientAddresses[3] = "0xaa"
	state.ClaimableRewards[3] = big.NewInt(40000)
	state.UnbanBalances[3] = big.NewInt(10)

	state.DepositAddresses[4] = "0xaa"
	state.PoolRecipientAddresses[4] = "0xaa"
	state.ClaimableRewards[4] = big.NewInt(50000)
	state.UnbanBalances[4] = big.NewInt(0)

	state.DepositAddresses[5] = "0xbb"
	state.PoolRecipientAddresses[5] = "0xbb"
	state.ClaimableRewards[5] = big.NewInt(500000)
	state.UnbanBalances[5] = big.NewInt(500)

	expected := []RawLeaf{
		{
			DepositAddress:   "0xaa",
			PoolRecipient:    "0xaa",
			ClaimableBalance: new(big.Int).SetUint64(190000),
			UnbanBalance:     new(big.Int).SetUint64(10),
		},
		{
			DepositAddress:   "0xbb",
			PoolRecipient:    "0xbb",
			ClaimableBalance: new(big.Int).SetUint64(500000),
			UnbanBalance:     new(big.Int).SetUint64(500),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_Aggregation_NoOrder(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.DepositAddresses[0] = "0xaa"
	state.PoolRecipientAddresses[0] = "0xbb"
	state.ClaimableRewards[0] = big.NewInt(30000)
	state.UnbanBalances[0] = big.NewInt(10)

	state.DepositAddresses[1] = "0xaa"
	state.PoolRecipientAddresses[1] = "0xcc"
	state.ClaimableRewards[1] = big.NewInt(60000)
	state.UnbanBalances[1] = big.NewInt(15)

	state.DepositAddresses[2] = "0xaa"
	state.PoolRecipientAddresses[2] = "0xbb"
	state.ClaimableRewards[2] = big.NewInt(10000)
	state.UnbanBalances[2] = big.NewInt(0)

	expected := []RawLeaf{
		{
			DepositAddress:   "0xaa",
			PoolRecipient:    "0xbb",
			ClaimableBalance: new(big.Int).SetUint64(40000),
			UnbanBalance:     new(big.Int).SetUint64(10),
		},
		{
			DepositAddress:   "0xaa",
			PoolRecipient:    "0xcc",
			ClaimableBalance: new(big.Int).SetUint64(60000),
			UnbanBalance:     new(big.Int).SetUint64(15),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_OrderByDepositAddress(t *testing.T) {
	merklelizer := NewMerklelizer()

	leafs := []RawLeaf{
		{
			DepositAddress:   "0x30",
			PoolRecipient:    "0xaaa",
			ClaimableBalance: new(big.Int).SetUint64(1),
			UnbanBalance:     new(big.Int).SetUint64(2),
		},
		{
			DepositAddress:   "0x50",
			PoolRecipient:    "0xbbb",
			ClaimableBalance: new(big.Int).SetUint64(3),
			UnbanBalance:     new(big.Int).SetUint64(4),
		},
		{
			DepositAddress:   "0x10",
			PoolRecipient:    "0xccc",
			ClaimableBalance: new(big.Int).SetUint64(5),
			UnbanBalance:     new(big.Int).SetUint64(6),
		},
		{
			DepositAddress:   "0xa0",
			PoolRecipient:    "0xccc",
			ClaimableBalance: new(big.Int).SetUint64(5),
			UnbanBalance:     new(big.Int).SetUint64(6),
		},
		{
			DepositAddress:   "0x99",
			PoolRecipient:    "0xccc",
			ClaimableBalance: new(big.Int).SetUint64(5),
			UnbanBalance:     new(big.Int).SetUint64(6),
		},
	}

	expected := []RawLeaf{
		{
			DepositAddress:   "0x10",
			PoolRecipient:    "0xccc",
			ClaimableBalance: new(big.Int).SetUint64(5),
			UnbanBalance:     new(big.Int).SetUint64(6),
		},
		{
			DepositAddress:   "0x30",
			PoolRecipient:    "0xaaa",
			ClaimableBalance: new(big.Int).SetUint64(1),
			UnbanBalance:     new(big.Int).SetUint64(2),
		},
		{
			DepositAddress:   "0x50",
			PoolRecipient:    "0xbbb",
			ClaimableBalance: new(big.Int).SetUint64(3),
			UnbanBalance:     new(big.Int).SetUint64(4),
		},
		{
			DepositAddress:   "0x99",
			PoolRecipient:    "0xccc",
			ClaimableBalance: new(big.Int).SetUint64(5),
			UnbanBalance:     new(big.Int).SetUint64(6),
		},
		{
			DepositAddress:   "0xa0",
			PoolRecipient:    "0xccc",
			ClaimableBalance: new(big.Int).SetUint64(5),
			UnbanBalance:     new(big.Int).SetUint64(6),
		},
	}

	ordered := merklelizer.OrderByDepositAddress(leafs)
	require.Equal(t, expected, ordered)
}
