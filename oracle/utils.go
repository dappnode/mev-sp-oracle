package oracle

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

func ToBytes20(x []byte) [20]byte {
	var y [20]byte
	copy(y[:], x)
	return y
}

func DecodeTx(rawTx []byte) (*types.Transaction, *types.Message, error) {
	var tx types.Transaction
	err := tx.UnmarshalBinary(rawTx)
	if err != nil {
		return nil, nil, err
	}

	// Supports EIP-2930 and EIP-2718 and EIP-1559 and EIP-155 and legacy transactions.
	msg, err := tx.AsMessage(types.LatestSignerForChainID(tx.ChainId()), big.NewInt(0))
	if err != nil {
		return nil, nil, err
	}
	return &tx, &msg, err
}

// Sum two numbers, and if the sum is bigger than saturate, return saturate
func SumAndSaturate(a *big.Int, b *big.Int, saturate *big.Int) *big.Int {
	aPlusB := new(big.Int).Add(a, b)
	if aPlusB.Cmp(saturate) >= 0 {
		return saturate
	}
	return aPlusB
}
