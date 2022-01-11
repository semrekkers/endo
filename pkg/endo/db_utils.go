package endo

import (
	"context"
	"database/sql"
	"fmt"
)

// DBTX represents a database connection or transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// Scanner scans a sql.Row or sql.Rows.
type Scanner interface {
	Scan(dest ...interface{}) error
}

const (
	// TxMutation denotes a possible mutation, so open a read-write transaction.
	TxMutation = 1 << iota
	// TxMulti denotes a transaction with multiple statements.
	TxMulti

	// TxReadOnly (default) denotes only read(s), so a read-only transaction will be sufficient.
	TxReadOnly = 0
)

type TxFunc func(ctx context.Context, flags uint8, fn func(DBTX) error) error

// UseDB wraps db inside a minimum transaction function handler. The returned TxFunc covers
// only the basic functionality.
func UseDB(db *sql.DB) TxFunc {
	return func(ctx context.Context, flags uint8, fn func(DBTX) error) error {
		var (
			err      error
			dbTx     DBTX = db
			activeTx *sql.Tx
		)
		if flags&TxMulti != 0 {
			activeTx, err = db.BeginTx(ctx, &sql.TxOptions{
				ReadOnly: flags&TxMutation == 0,
			})
			if err != nil {
				return fmt.Errorf("begin transaction: %w", err)
			}
			defer activeTx.Rollback()
			dbTx = activeTx
		}
		if err = fn(dbTx); err != nil {
			return err
		}
		if activeTx != nil {
			if err = activeTx.Commit(); err != nil {
				return fmt.Errorf("commit transaction: %w", err)
			}
		}
		return nil
	}
}
