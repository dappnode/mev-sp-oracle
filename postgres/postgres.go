package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

type Postgresql struct {
	db       *pgx.Conn
	PoolName string
}

// postgres://xxx:yyy@url:5432
func New(postgresEndpoint string) (*Postgresql, error) {
	conn, err := pgx.Connect(context.Background(), postgresEndpoint)

	if err != nil {
		return nil, err
	}

	return &Postgresql{
		db: conn,
	}, nil
}

// Returns the validator keys for the given deposit addresses
func (a *Postgresql) GetValidatorKeysFromDepositAddress(fromAddresses []string) ([][]byte, error) {
	rows, err := a.db.Query(context.Background(),
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
	err := a.db.QueryRow(context.Background(),
		"select encode(f_eth1_sender, 'hex') from t_eth1_deposits where encode(f_validator_pubkey::bytea, 'hex') = $1",
		strings.Replace(validatorKey, "0x", "", -1)).Scan(&depositAddress)

	if err != nil {
		return "", err
	}
	return "0x" + depositAddress, nil
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

var createTable = `
CREATE TABLE IF NOT EXISTS t_oracle_validator_balances (
	 f_deposit_address TEXT,
	 f_validator_key TEXT,
	 f_slot_checkpoint BIGINT,
	 f_pending_balance BIGINT,
	 f_claimed_balance BIGINT,
	 f_num_proposed_blocks BIGINT,
	 f_num_missed_blocks BIGINT,
	 f_checkpoint_slot BIGINT,
	 f_checkpoint_proofs TEXT,
	 f_checkpoint_root TEXT,
);
`

func (a *Postgresql) DumpOracleStateToDatabase() error {
	if _, err := a.db.Exec(
		context.Background(),
		createTable); err != nil {
		return err
	}
	return nil
}
