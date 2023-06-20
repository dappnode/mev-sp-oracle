package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewCliConfig(t *testing.T) {

	cliConf, err := NewCliConfig()
	_ = cliConf
	require.Error(t, err)
}
