package oracle

import (
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TODO: Move to utils module

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

func GetUniqueElements(arr []string) []string {
	result := []string{}
	encountered := map[string]bool{}
	for v := range arr {
		encountered[arr[v]] = true
	}
	for key, _ := range encountered {
		result = append(result, key)
	}
	return result
}

func ByteArrayToStringArray(arr [][]byte) string {
	result := []string{}
	for _, v := range arr {
		result = append(result, "0x"+hex.EncodeToString(v))
	}

	return strings.Join(result, ",")
}

func ByteArrayToArray(arr [][]byte) []string {
	result := make([]string, 0)
	for _, v := range arr {
		result = append(result, "0x"+hex.EncodeToString(v))
	}

	return result
}

// Converts from slots to readable time (eg 1 day 9 hours 20 minutes)
func SlotsToTime(slots uint64) string {
	// Hardcoded. Mainnet Ethereum configuration
	SecondsInSlot := uint64(12)

	timeduration := time.Duration(slots*SecondsInSlot) * time.Second
	strDuration := durafmt.Parse(timeduration).String()

	return strDuration
}

func StringToBlsKey(str string) phase0.BLSPubKey {
	validator := phase0.BLSPubKey{}

	if strings.HasPrefix(str, "0x") {
		str = str[2:]
	}
	unboundedBytes := common.Hex2Bytes(str)

	if len(unboundedBytes) != 48 {
		log.Fatal("wrong merkle root length: ", str)
	}
	copy(validator[:], common.Hex2Bytes(str))

	return validator
}

func NumInSlice(a uint64, list []uint64) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// See: https://github.com/ethereum/eth2.0-specs/blob/dev/specs/phase0/validator.md#withdrawal-credentials
// Input example: 00fccee96b30754af30208261e38df169a95aa3c722662a9df8fc057cc7d3a69 (true)
func IsBlsType(withdrawalCred string) bool {
	if len(withdrawalCred) != 64 {
		return false
	}

	/* BLS_WITHDRAWAL_PREFIX */
	if strings.HasPrefix(withdrawalCred, "00") {
		return true
	}
	return false
}

// See: https://github.com/ethereum/eth2.0-specs/blob/dev/specs/phase0/validator.md#withdrawal-credentials
// Input example: 010000000000000000000000dc62f9e8c34be08501cdef4ebde0a280f576d762 (true)
func IsEth1Type(withdrawalCred string) bool {
	if len(withdrawalCred) != 64 {
		return false
	}

	/* ETH1_ADDRESS_WITHDRAWAL_PREFIX*/
	if strings.HasPrefix(withdrawalCred, "010000000000000000000000") {
		return true
	}
	return false
}

// See: https://github.com/ethereum/eth2.0-specs/blob/dev/specs/phase0/validator.md#withdrawal-credentials
// Input example: 01000000000000000000000059b0d71688da01057c08e4c1baa8faa629819c2a
// Output example: 0x59b0d71688da01057c08e4c1baa8faa629819c2a
func GetEth1Address(withdrawalCred string) (string, error) {
	if len(withdrawalCred) != 64 {
		return "", errors.New("Withdrawal credentials are not a valid length")
	}
	/* ETH1_ADDRESS_WITHDRAWAL_PREFIX*/
	if !strings.HasPrefix(withdrawalCred, "010000000000000000000000") {
		return "", errors.New("Withdrawal credentials prefix does not match the spec")
	}
	return "0x" + withdrawalCred[24:], nil
}

func GetEth1AddressByte(withdrawalCredByte []byte) (string, error) {
	withdrawalCred := hex.EncodeToString(withdrawalCredByte)
	if len(withdrawalCred) != 64 {
		return "", errors.New("Withdrawal credentials are not a valid length")
	}
	/* ETH1_ADDRESS_WITHDRAWAL_PREFIX*/
	if !strings.HasPrefix(withdrawalCred, "010000000000000000000000") {
		return "", errors.New("Withdrawal credentials prefix does not match the spec")
	}
	return "0x" + withdrawalCred[24:], nil
}

func AreAddressEqual(address1 string, address2 string) bool {
	if len(address1) != len(address2) {
		log.Fatal("address length mismatch: ",
			"add1: ", address1,
			"add2: ", address2)
	}
	if strings.ToLower(address1) == strings.ToLower(address2) {
		return true
	}
	return false
}

func DecryptKey(cfg *config.Config) (*keystore.Key, error) {
	// Only parse it not in dry run mode
	if !cfg.DryRun {
		jsonBytes, err := ioutil.ReadFile(cfg.UpdaterKeyPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read updater key file")
		}

		account, err := keystore.DecryptKey(jsonBytes, cfg.UpdaterKeyPass)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decrypt updater key")
		}
		return account, nil
	}
	return nil, errors.New("running in dry run mode, key is not needed")
}
