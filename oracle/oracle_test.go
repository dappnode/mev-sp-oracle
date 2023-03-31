package oracle

import (
	"math/big"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/stretchr/testify/require"
)

// TODO:
func Test_Oracle(t *testing.T) {
	oracle := NewOracle(&config.Config{
		Network:               "",
		PoolAddress:           "",
		UpdaterAddress:        "",
		DeployedSlot:          uint64(0),
		CheckPointSizeInSlots: uint64(0),
		PoolFeesPercent:       10,
		PoolFeesAddress:       "",
		CollateralInWei:       big.NewInt(1000000),
	})

	subs := []Subscription{
		{
			1,
			"0xaa",
			big.NewInt(1000000),
			0,
			"0xab",
			"0xac",
		},
		{
			ValidatorIndex: 2,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(1000000),
			BlockNumber:    0,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
		},
		{
			ValidatorIndex: 3,
			ValidatorKey:   "0xba",
			Collateral:     big.NewInt(50),
			BlockNumber:    0,
			TxHash:         "0xbb",
			DepositAddress: "0xbc",
		},
	}

	blockOk := Block{
		Slot:           uint64(0),
		ValidatorIndex: uint64(1),
		ValidatorKey:   "0xxx",
		BlockType:      OkPoolProposal,
		Reward:         big.NewInt(0),
		RewardType:     MevBlock,
		DepositAddress: "0xaaa",
	}

	blockMissed := Block{
		Slot:           uint64(0),
		ValidatorIndex: uint64(1),
		ValidatorKey:   "0xxx",
		BlockType:      MissedProposal,
	}

	blockWrongFee := Block{
		Slot:           uint64(0),
		ValidatorIndex: uint64(1),
		ValidatorKey:   "0xxx",
		BlockType:      WrongFeeRecipient,
		Reward:         big.NewInt(0),
		RewardType:     MevBlock,
	}

	_ = blockMissed
	_ = blockWrongFee

	processedSlot, err := oracle.AdvanceStateToNextSlot(blockOk, subs, []Unsubscription{}, []Donation{})
	require.NoError(t, err)
	require.Equal(t, uint64(0), processedSlot)
}
