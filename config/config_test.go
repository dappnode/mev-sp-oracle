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

func Test_ReadHardcodedSubscriptions(t *testing.T) {
	// Test file containing 4 validator indexes, one per line
	fileName := "hardcoded_subscriptions.txt"
	someValidatorIndexes := "1234\n2132\n890\n2343"
	CreateMockKeysFile(t, fileName, someValidatorIndexes)
	defer os.Remove(fileName)

	indexes, err := ReadHardcodedSubscriptions(fileName)
	require.NoError(t, err)
	require.Equal(t, 4, len(indexes))
	require.Equal(t, uint64(1234), indexes[0])
	require.Equal(t, uint64(2132), indexes[1])
	require.Equal(t, uint64(890), indexes[2])
	require.Equal(t, uint64(2343), indexes[3])

	// Test file containing 4 validator indexes, one per line, with extra line/space
	fileName2 := "hardcoded_subscriptions.txt"
	someValidatorIndexes2 := "1\n22\n8\n2\n " // <- extra line/space
	CreateMockKeysFile(t, fileName2, someValidatorIndexes2)
	defer os.Remove(fileName2)

	indexes2, err := ReadHardcodedSubscriptions(fileName2)
	require.NoError(t, err)
	require.Equal(t, 4, len(indexes2))
	require.Equal(t, uint64(1), indexes2[0])
	require.Equal(t, uint64(22), indexes2[1])
	require.Equal(t, uint64(8), indexes2[2])
	require.Equal(t, uint64(2), indexes2[3])
}
