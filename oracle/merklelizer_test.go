package oracle

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/stretchr/testify/require"
)

func Test_GenerateTreeFromState(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{
		PoolFeesAddress: "0x0000000000000000000000000000000000000000",
	})

	// Note that the leafs contain also PoolAddress at the begining

	state.Validators[0] = &ValidatorInfo{
		WithdrawalAddress:     "0x1000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[1] = &ValidatorInfo{
		WithdrawalAddress:     "0x2000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[2] = &ValidatorInfo{
		WithdrawalAddress:     "0x3000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[3] = &ValidatorInfo{
		WithdrawalAddress:     "0x4000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		WithdrawalAddress:     "0x5000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[5] = &ValidatorInfo{
		WithdrawalAddress:     "0x6000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}

	// TODO: add test to _, _
	_, _, tree, _ := merklelizer.GenerateTreeFromState(state)
	require.Equal(t, "7c58e94268a0d3d89578d2e90e483e3d53a3cb26315852d1544a5a386c83335e", hex.EncodeToString(tree.Root))

}

func Test_AggregateValidatorsIndexes_NoAggregation(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{
		PoolFeesAddress: "0x0000000000000000000000000000000000000000",
	})

	state.PoolAccumulatedFees = big.NewInt(999999999999999)

	state.Validators[0] = &ValidatorInfo{
		WithdrawalAddress:     "0x1000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[1] = &ValidatorInfo{
		WithdrawalAddress:     "0x2000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[2] = &ValidatorInfo{
		WithdrawalAddress:     "0x3000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[3] = &ValidatorInfo{
		WithdrawalAddress:     "0x4000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		WithdrawalAddress:     "0x5000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[5] = &ValidatorInfo{
		WithdrawalAddress:     "0x6000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}

	expected := []RawLeaf{
		{
			WithdrawalAddress:  "0x0000000000000000000000000000000000000000",
			AccumulatedBalance: big.NewInt(999999999999999),
		},
		{
			WithdrawalAddress:  "0x1000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(10000),
		},
		{
			WithdrawalAddress:  "0x2000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(20000),
		},
		{
			WithdrawalAddress:  "0x3000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(30000),
		},
		{
			WithdrawalAddress:  "0x4000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(40000),
		},
		{
			WithdrawalAddress:  "0x5000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(50000),
		},
		{
			WithdrawalAddress:  "0x6000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(60000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	fmt.Println(rawLeafs)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_NoAggregationOrdered(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{
		PoolFeesAddress: "0x0000000000000000000000000000000000000000",
	})

	state.PoolAccumulatedFees = big.NewInt(2345678987654)

	state.Validators[0] = &ValidatorInfo{
		WithdrawalAddress:     "0x3000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		WithdrawalAddress:     "0x6000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		WithdrawalAddress:     "0x1000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		WithdrawalAddress:     "0x2000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[4] = &ValidatorInfo{
		WithdrawalAddress:     "0x4000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[5] = &ValidatorInfo{
		WithdrawalAddress:     "0x5000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	expected := []RawLeaf{
		{
			WithdrawalAddress:  "0x0000000000000000000000000000000000000000",
			AccumulatedBalance: big.NewInt(2345678987654),
		},
		{
			WithdrawalAddress:  "0x1000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(10000),
		},
		{
			WithdrawalAddress:  "0x2000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(20000),
		},
		{
			WithdrawalAddress:  "0x3000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(30000),
		},
		{
			WithdrawalAddress:  "0x4000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(40000),
		},
		{
			WithdrawalAddress:  "0x5000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(50000),
		},
		{
			WithdrawalAddress:  "0x6000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(60000),
		},
	}

	// TODO: add checks on merkle root

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	fmt.Println(rawLeafs)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_AggregationAll(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{
		PoolFeesAddress: "0x0000000000000000000000000000000000000000",
	})

	state.PoolAccumulatedFees = big.NewInt(0)

	state.Validators[0] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(20000),
	}
	state.Validators[4] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[5] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	expected := []RawLeaf{
		{
			WithdrawalAddress:  "0x0000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(0),
		},
		{
			WithdrawalAddress:  "0xaa00000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(210000),
		},
	}

	// TODO: add checks on merkle root

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_Aggregation_And_Leftover(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{
		PoolFeesAddress: "0x0000000000000000000000000000000000000000",
	})

	state.PoolAccumulatedFees = new(big.Int).SetUint64(1)

	state.Validators[0] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		WithdrawalAddress:     "0xaa00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	state.Validators[5] = &ValidatorInfo{
		WithdrawalAddress:     "0xbb00000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(500000),
	}

	expected := []RawLeaf{
		{
			WithdrawalAddress:  "0x0000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(1),
		},
		{
			WithdrawalAddress:  "0xaa00000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(190000),
		},
		{
			WithdrawalAddress:  "0xbb00000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(500000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_AggregateValidatorsIndexes_Aggregation_NoOrder(t *testing.T) {
	merklelizer := NewMerklelizer()
	state := NewOracleState(&config.Config{
		PoolFeesAddress: "0x0000000000000000000000000000000000000000",
	})

	state.PoolAccumulatedFees = big.NewInt(234567)

	state.Validators[0] = &ValidatorInfo{
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(30000),
	}
	state.Validators[1] = &ValidatorInfo{
		WithdrawalAddress:     "0xb000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(60000),
	}
	state.Validators[2] = &ValidatorInfo{
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(10000),
	}
	state.Validators[3] = &ValidatorInfo{
		WithdrawalAddress:     "0xc000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[4] = &ValidatorInfo{
		WithdrawalAddress:     "0xc000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[5] = &ValidatorInfo{
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(40000),
	}
	state.Validators[6] = &ValidatorInfo{
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[7] = &ValidatorInfo{
		WithdrawalAddress:     "0xc000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[8] = &ValidatorInfo{
		WithdrawalAddress:     "0xb000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}
	state.Validators[9] = &ValidatorInfo{
		WithdrawalAddress:     "0xb000000000000000000000000000000000000000",
		AccumulatedRewardsWei: big.NewInt(50000),
	}

	expected := []RawLeaf{
		{
			WithdrawalAddress:  "0x0000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(234567),
		},
		{
			WithdrawalAddress:  "0xa000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(130000),
		},
		{
			WithdrawalAddress:  "0xb000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(160000),
		},
		{
			WithdrawalAddress:  "0xc000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(140000),
		},
	}

	rawLeafs := merklelizer.AggregateValidatorsIndexes(state)
	require.Equal(t, expected, rawLeafs)
}

func Test_OrderByWithdrawalAddress(t *testing.T) {
	merklelizer := NewMerklelizer()

	leafs := []RawLeaf{
		{
			WithdrawalAddress:  "0x3000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(1),
		},
		{
			WithdrawalAddress:  "0x5000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(3),
		},
		{
			WithdrawalAddress:  "0x1000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			WithdrawalAddress:  "0xa000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			WithdrawalAddress:  "0x9900000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
	}

	expected := []RawLeaf{
		{
			WithdrawalAddress:  "0x1000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			WithdrawalAddress:  "0x3000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(1),
		},
		{
			WithdrawalAddress:  "0x5000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(3),
		},
		{
			WithdrawalAddress:  "0x9900000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
		{
			WithdrawalAddress:  "0xa000000000000000000000000000000000000000",
			AccumulatedBalance: new(big.Int).SetUint64(5),
		},
	}

	ordered := merklelizer.OrderByWithdrawalAddress(leafs)
	require.Equal(t, expected, ordered)
}
