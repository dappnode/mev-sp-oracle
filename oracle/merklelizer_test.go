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

	state.Validators[0] = &ValidatorInfo{
		DepositAddress:        "0x1000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[1] = &ValidatorInfo{
		DepositAddress:        "0x2000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[2] = &ValidatorInfo{
		DepositAddress:        "0x3000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[3] = &ValidatorInfo{
		DepositAddress:        "0x4000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		DepositAddress:        "0x5000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[5] = &ValidatorInfo{
		DepositAddress:        "0x6000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}

	// TODO: add test to _, _
	_, _, tree, _ := merklelizer.GenerateTreeFromState(state)
	require.Equal(t, "9ff9fc5b0bb88bd93251c1d46e3977801ab3275b83bdf7088380cc824fb57932", hex.EncodeToString(tree.Root))

}

func Test_AggregateValidatorsIndexes_NoAggregation(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.Validators[0] = &ValidatorInfo{
		DepositAddress:        "0x1000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[1] = &ValidatorInfo{
		DepositAddress:        "0x2000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[2] = &ValidatorInfo{
		DepositAddress:        "0x3000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[3] = &ValidatorInfo{
		DepositAddress:        "0x4000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		DepositAddress:        "0x5000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[5] = &ValidatorInfo{
		DepositAddress:        "0x6000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}

	expected := []RawLeaf{
		{
			DepositAddress:     "0x1000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(10000),
		},
		{
			DepositAddress:     "0x2000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(20000),
		},
		{
			DepositAddress:     "0x3000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(30000),
		},
		{
			DepositAddress:     "0x4000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(40000),
		},
		{
			DepositAddress:     "0x5000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(50000),
		},
		{
			DepositAddress:     "0x6000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(60000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	fmt.Println(rawLeafs)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_NoAggregationOrdered(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.Validators[0] = &ValidatorInfo{
		DepositAddress:        "0x3000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		DepositAddress:        "0x6000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		DepositAddress:        "0x1000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		DepositAddress:        "0x2000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[4] = &ValidatorInfo{
		DepositAddress:        "0x4000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[5] = &ValidatorInfo{
		DepositAddress:        "0x5000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	expected := []RawLeaf{
		{
			DepositAddress:     "0x1000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(10000),
		},
		{
			DepositAddress:     "0x2000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(20000),
		},
		{
			DepositAddress:     "0x3000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(30000),
		},
		{
			DepositAddress:     "0x4000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(40000),
		},
		{
			DepositAddress:     "0x5000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(50000),
		},
		{
			DepositAddress:     "0x6000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(60000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	fmt.Println(rawLeafs)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_AggregationAll(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.Validators[0] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[4] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[5] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	expected := []RawLeaf{
		{
			DepositAddress:     "0xaa",
			AccumulatedBalance: new(big.Int).SetUint64(210000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_Aggregation_And_Leftover(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.Validators[0] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		DepositAddress:        "0xaa",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	state.Validators[5] = &ValidatorInfo{
		DepositAddress:        "0xbb",
		AccumulatedRewardsWei: big.NewInt(500000),
	}

	expected := []RawLeaf{
		{
			DepositAddress:     "0xaa",
			AccumulatedBalance: new(big.Int).SetUint64(190000),
		},
		{
			DepositAddress:     "0xbb",
			AccumulatedBalance: new(big.Int).SetUint64(500000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_Aggregation_NoOrder(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{})

	state.Validators[0] = &ValidatorInfo{
		DepositAddress:        "0xa",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		DepositAddress:        "0xb",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		DepositAddress:        "0xa",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		DepositAddress:        "0xc",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		DepositAddress:        "0xc",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[5] = &ValidatorInfo{
		DepositAddress:        "0xa",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[6] = &ValidatorInfo{
		DepositAddress:        "0xa",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[7] = &ValidatorInfo{
		DepositAddress:        "0xc",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[8] = &ValidatorInfo{
		DepositAddress:        "0xb",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[9] = &ValidatorInfo{
		DepositAddress:        "0xb",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	expected := []RawLeaf{
		{
			DepositAddress:     "0xa",
			AccumulatedBalance: new(big.Int).SetUint64(130000),
		},
		{
			DepositAddress:     "0xb",
			AccumulatedBalance: new(big.Int).SetUint64(160000),
		},
		{
			DepositAddress:     "0xc",
			AccumulatedBalance: new(big.Int).SetUint64(140000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_OrderByDepositAddress(t *testing.T) {
	merklelizer := NewMerklelizer()

	leafs := []RawLeaf{
		{
			DepositAddress:     "0x30",
			AccumulatedBalance: new(big.Int).SetUint64(1),
		},
		{
			DepositAddress:     "0x50",
			AccumulatedBalance: new(big.Int).SetUint64(3),
		},
		{
			DepositAddress:     "0x10",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			DepositAddress:     "0xa0",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			DepositAddress:     "0x99",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
	}

	expected := []RawLeaf{
		{
			DepositAddress:     "0x10",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			DepositAddress:     "0x30",
			AccumulatedBalance: new(big.Int).SetUint64(1),
		},
		{
			DepositAddress:     "0x50",
			AccumulatedBalance: new(big.Int).SetUint64(3),
		},
		{
			DepositAddress:     "0x99",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			DepositAddress:     "0xa0",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
	}

	ordered := merklelizer.OrderByDepositAddress(leafs)
	require.Equal(t, expected, ordered)
}
