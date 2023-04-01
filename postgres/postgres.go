package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// This file is outdated and some of these functions are not used

type Postgresql struct {
	Db         *pgx.Conn
	NumRetries int
}

// postgres://xxx:yyy@url:5432
func New(postgresEndpoint string, numRetries int) (*Postgresql, error) {
	var conn *pgx.Conn
	var err error
	if postgresEndpoint != "" {
		conn, err = pgx.Connect(context.Background(), postgresEndpoint)
	}

	if err != nil {
		return nil, err
	}

	return &Postgresql{
		Db:         conn,
		NumRetries: numRetries,
	}, nil
}

// Returns the validator keys for the given deposit addresses
func (a *Postgresql) GetValidatorKeysFromDepositAddress(fromAddresses []string, opts ...retry.Option) ([][]byte, error) {
	var err error
	var rows pgx.Rows

	err = retry.Do(func() error {
		rows, err = a.Db.Query(context.Background(),
			`select encode(f_validator_pubkey, 'hex')
		from t_eth1_deposits
		where (`+getDepositsWhereClause(fromAddresses)+")")
		if err != nil {
			log.Warn("Retrying get validator keys from deposit address: ", fromAddresses)
			return errors.Wrap(err, "could not get keys for pool")
		}
		return nil
	}, a.GetRetryOpts(opts)...)

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
func (a *Postgresql) GetDepositAddressOfValidatorKey(validatorKey string, opts ...retry.Option) (string, error) {
	var depositAddress string
	var err error

	err = retry.Do(func() error {
		err = a.Db.QueryRow(context.Background(),
			"select encode(f_eth1_sender, 'hex') from t_eth1_deposits where encode(f_validator_pubkey::bytea, 'hex') = $1",
			strings.Replace(validatorKey, "0x", "", -1)).Scan(&depositAddress)
		if err != nil {
			log.Warn("Retrying get deposit address of validator key: ", validatorKey)
			return errors.Wrap(err, "could not get deposit address of validator key")
		}
		return nil
	}, a.GetRetryOpts(opts)...)

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

func (a *Postgresql) GetRetryOpts(opts []retry.Option) []retry.Option {
	// Default retry options. This specifies what to do when a call to the
	// consensus or execution client fails. Default is to retry 5 times
	// with a 15 seconds delay and the default backoff strategy (see avas/retry-go)
	// Note that in some cases we might want to avoid retrying at all, for example
	// when serving data to an api, we may want to just fail fast and return an error
	if len(opts) == 0 {
		return []retry.Option{
			retry.Attempts(uint(a.NumRetries)),
			retry.Delay(15 * time.Second),
		}
	} else {
		return opts
	}
}
