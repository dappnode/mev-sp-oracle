package utils

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ToBytes20(x []byte) [20]byte {
	var y [20]byte
	copy(y[:], x)
	return y
}

func DecodeTx(rawTx []byte) (*types.Transaction, error) {
	var tx types.Transaction
	err := tx.UnmarshalBinary(rawTx)
	if err != nil {
		return nil, err
	}
	return &tx, err
}

func GetTxSender(tx *types.Transaction) (common.Address, error) {
	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return common.Address{}, err
	}
	return from, nil
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

	// BLS_WITHDRAWAL_PREFIX
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

	// ETH1_ADDRESS_WITHDRAWAL_PREFIX
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
	// ETH1_ADDRESS_WITHDRAWAL_PREFIX
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
	// ETH1_ADDRESS_WITHDRAWAL_PREFIX
	if !strings.HasPrefix(withdrawalCred, "010000000000000000000000") {
		return "", errors.New("Withdrawal credentials prefix does not match the spec")
	}
	return "0x" + withdrawalCred[24:], nil
}

func Equals(a string, b string) bool {
	if len(a) != len(b) {
		log.Fatal("values length mismatch: ",
			"len(a): ", len(a), " len(b): ", len(b), " a: ", a, " b: ", b)
	}
	if strings.ToLower(a) == strings.ToLower(b) {
		return true
	}
	return false
}

func DecryptKey(cfg *config.CliConfig) (*keystore.Key, error) {
	// Only parse it not in dry run mode
	if !cfg.DryRun {
		jsonBytes, err := ioutil.ReadFile(cfg.UpdaterKeyFile)
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

// Not the most efficient way of deep coping, if performance
// matters, dont use this.
func DeepCopy(a, b interface{}) {

	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	enc.Encode(a)
	dec.Decode(b)
}

func GetActivationSlotOfLatestProcessedValidator(
	validators map[phase0.ValidatorIndex]*v1.Validator) uint64 {
	MaxUint := ^uint64(0)
	if len(validators) == 0 {
		log.Fatal("validators map is empty")
	}

	latestEpoch := uint64(0)

	// Could be faster if iterated backwards
	for _, val := range validators {
		// When validators are not processed yet, max uint64 is stored
		activationEpoch := uint64(val.Validator.ActivationEpoch)
		if activationEpoch != MaxUint &&
			activationEpoch > latestEpoch {
			latestEpoch = activationEpoch
		}
	}

	// Could technically happen if oracle were deployed in genesis
	// But useful as a sanity check
	if latestEpoch == 0 {
		log.Fatal("latestEpoch is 0")
	}

	SlotsInEpoch := uint64(32)
	return latestEpoch * SlotsInEpoch
}

func WeiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}
