package oracle

import (
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/stretchr/testify/require"
)

// TODO:
func Test_Oracle_1(t *testing.T) {
	var cfg = config.Config{}
	onchain, err := NewOnchain(cfg)
	require.NoError(t, err)
	oracle := NewOracle(&cfg, onchain)
	_ = oracle
}
