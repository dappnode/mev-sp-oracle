package postgres

import (
	"context"
	"fmt"
	"math/big"

	//"mev-sp-oracle/oracle"
	"strings"

	//"mev-sp-oracle/oracle"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Postgresql struct {
	Db *pgx.Conn
}

// postgres://xxx:yyy@url:5432
func New(postgresEndpoint string) (*Postgresql, error) {
	var conn *pgx.Conn
	var err error
	if postgresEndpoint != "" {
		conn, err = pgx.Connect(context.Background(), postgresEndpoint)
	}

	if err != nil {
		return nil, err
	}

	return &Postgresql{
		Db: conn,
	}, nil
}

// Returns the validator keys for the given deposit addresses
func (a *Postgresql) GetValidatorKeysFromDepositAddress(fromAddresses []string) ([][]byte, error) {
	rows, err := a.Db.Query(context.Background(),
		`select encode(f_validator_pubkey, 'hex')
		from t_eth1_deposits
		where (`+getDepositsWhereClause(fromAddresses)+")")

	if err != nil {
		return nil, errors.Wrap(err,
			fmt.Sprintf("%s: %s", "could not get keys for pool",
				fromAddresses))
	}

	keys := make([][]byte, 0)
	defer rows.Close()
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		for _, keyStr := range values {
			byteKey, err := hexutil.Decode(fmt.Sprintf("0x%s", keyStr.(string)))
			if err != nil {
				return nil, err
			}
			keys = append(keys, byteKey)
		}
	}

	return keys, nil
}

// Given a validator key in hex prefixed with 0x, return its deposit address
// also with 0x prefix
func (a *Postgresql) GetDepositAddressOfValidatorKey(validatorKey string) (string, error) {
	var depositAddress string
	err := a.Db.QueryRow(context.Background(),
		"select encode(f_eth1_sender, 'hex') from t_eth1_deposits where encode(f_validator_pubkey::bytea, 'hex') = $1",
		strings.Replace(validatorKey, "0x", "", -1)).Scan(&depositAddress)

	if err != nil {
		return "", err
	}
	return "0x" + depositAddress, nil
}

// Gets the merkle proofs for a given deposit address for the latest known checkpoint (slot)
// Deposit address should be prefixed with 0x
func (a *Postgresql) GetLatestMerkleProofByDeposit(depositAddress string) ([]string, string, uint64, string, string, error) {
	// TODO: Get also merkle root
	// TODO: check if not starts with 0x, return err
	var merkleRoots string
	var merkleRoot string
	var checkpointSlot uint64
	var availableBalance string // TODO: perhaps bigInt is better workaround
	var unbanBalance string     // TODO: perhaps bigInt is better workaround

	// TODO: CAST(f_claimable_balance as TEXT) dirty workaround. Find a better way
	log.Info("depositAddress:", depositAddress)
	err := a.Db.QueryRow(context.Background(),
		"select f_checkpoint_proofs, f_checkpoint_root, f_checkpoint_slot, CAST(f_claimable_balance as TEXT), CAST(f_unban_balance as TEXT) from t_oracle_depositaddress_rewards where LOWER(f_deposit_address) = $1 and f_checkpoint_slot = (select max(f_checkpoint_slot) from t_oracle_depositaddress_rewards)",
		strings.ToLower(depositAddress)).Scan(&merkleRoots, &merkleRoot, &checkpointSlot, &availableBalance, &unbanBalance)

	if err != nil {
		return []string{}, "", 0, "", "", err
	}

	// TODO: add some validation
	return strings.Split(merkleRoots, ","), merkleRoot, checkpointSlot, availableBalance, unbanBalance, err
}

// TODO: passing everything, dirty
func (a *Postgresql) StoreBlockInDb(
	timestamp string,
	slot uint64,
	validatorKey string,
	validatorIndex uint64,
	vanilaOrMev string,
	rewardWei big.Int,
	okWrongMissed uint64) error {
	_, err := a.Db.Exec(
		context.Background(),
		InsertBlocksTable,
		timestamp,
		slot,
		validatorKey,
		validatorIndex,
		vanilaOrMev,
		rewardWei.Uint64(), // TODO can overflow
		okWrongMissed,
	)
	if err != nil {
		return err
	}
	return nil
}

func getDepositsWhereClause(fromAddresses []string) string {
	whereElements := make([]string, 0)
	for _, address := range fromAddresses {
		whereElements = append(
			whereElements,
			fmt.Sprintf("f_eth1_sender = decode('%s', 'hex')",
				strings.TrimPrefix(address, "0x")))
	}
	return strings.Join(whereElements, " or ")
}

// TODO: rename to validatorRewards vs DepositAddressRewards
// TODO remove the proofs from here.
var CreateRewardsTable = `
CREATE TABLE IF NOT EXISTS t_oracle_validator_balances (
	 f_deposit_address TEXT,
	 f_validator_key TEXT,
	 f_validator_index NUMERIC,
	 f_pending_balance BIGINT,
	 f_claimable_balance BIGINT,
	 f_unban_balance BIGINT,
	 f_num_proposed_blocks BIGINT,
	 f_num_missed_blocks BIGINT,
	 f_num_wrongfee_blocks BIGINT,
	 f_checkpoint_slot BIGINT,
	 f_checkpoint_proofs TEXT,
	 f_checkpoint_root TEXT,

	 PRIMARY KEY (f_validator_key, f_checkpoint_slot)
);
`

// TODO: pool recipient address no longer exists
// TODO: pending is not populated now, but a nice to have.
var CreateDepositAddressRewardsTable = `
CREATE TABLE IF NOT EXISTS t_oracle_depositaddress_rewards (
	 f_deposit_address TEXT,
	 f_validator_keys TEXT,
	 f_pending_balance BIGINT,
	 f_claimable_balance BIGINT,
	 f_unban_balance BIGINT,
	 f_checkpoint_slot BIGINT,
	 f_checkpoint_proofs TEXT,
	 f_checkpoint_root TEXT,

	 PRIMARY KEY (f_deposit_address, f_checkpoint_slot)
);
`

var CreateBlocksTable = `
CREATE TABLE IF NOT EXISTS t_pool_blocks (
	f_timestamp TEXT,
	f_slot NUMERIC,
	f_validator_key TEXT,
	f_validator_index NUMERIC,
	f_vanila_or_mev TEXT,
	f_reward_wei BIGINT,
	f_ok_wrong_missed NUMERIC,

	PRIMARY KEY (f_slot)
);
`

// TODO: add validator state?
var InsertRewardsTable = `
INSERT INTO t_oracle_validator_balances(
	f_deposit_address,
	f_validator_key,
	f_validator_index,
	f_pending_balance,
	f_claimable_balance,
	f_unban_balance,
	f_num_proposed_blocks,
	f_num_missed_blocks,
	f_num_wrongfee_blocks,
	f_checkpoint_slot,
	f_checkpoint_proofs,
	f_checkpoint_root)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

var InsertBlocksTable = `
INSERT INTO t_pool_blocks(
	f_timestamp,
	f_slot,
	f_validator_key,
	f_validator_index,
	f_vanila_or_mev,
	f_reward_wei,
	f_ok_wrong_missed)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`

var InsertDepositAddressRewardsTable = `
INSERT INTO t_oracle_depositaddress_rewards(
	f_deposit_address,
	f_validator_keys,
	f_pending_balance,
	f_claimable_balance,
	f_unban_balance,
	f_checkpoint_slot,
	f_checkpoint_proofs,
	f_checkpoint_root)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
