package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func CreateMockKeysFile(t *testing.T, customKeysFile string, content string) {
	f, err := os.Create(customKeysFile)
	require.NoError(t, err)

	_, err = f.WriteString(content)
	require.NoError(t, err)
	f.Close()
}

func Test_TODO(t *testing.T) {

	require.Equal(t, 1, 1)
	// TODO:
}
