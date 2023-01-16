package postgres

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Running this tests require a synced chaind instance

func Test_GetValidatorKeysFromDepositAddress(t *testing.T) {
	t.Skip("Skipping e2e test")
	db, err := New("postgres://xxx:yyy@localhost:5432")
	require.NoError(t, err)
	fromAdd, err := db.GetValidatorKeysFromDepositAddress([]string{"0x4554f8ca6104a361f20e75213d980f7ee56b24c3"})
	require.NoError(t, err)
	fmt.Println(fromAdd)
	require.Equal(t, 1, 1)
}

func Test_GetDepositAddressOfValidator(t *testing.T) {
	t.Skip("Skipping e2e test")
	db, err := New("postgres://xxx:yyy@localhost:5432")
	require.NoError(t, err)
	depositAddress, err := db.GetDepositAddressOfValidatorKey("0x858dff42301025ce01ac3ac6389b9a1a9020c210ef2ca2a21d17337729cf17e4bceca64aa6413c01d82d145bf1505116")
	require.NoError(t, err)
	fmt.Println(depositAddress)
	require.Equal(t, "0xd2322a361cbd77dd8bd45386cf3e70a1ac525f3f", depositAddress)
}
