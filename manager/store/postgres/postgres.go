package postgres

import (
	"context"
	"database/sql"

	"github.com/figment-networks/indexer-manager/structs"
)

// Driver is postgres database driver implementation
type Driver struct {
	db *sql.DB

	txBuff chan structs.TransactionWithMeta
	txPool *ValuesPool

	blBuff chan structs.BlockWithMeta
	blPool *ValuesPool
}

// NewDriver is Driver constructor
func NewDriver(ctx context.Context, db *sql.DB) *Driver {
	return &Driver{
		db:     db,
		txBuff: make(chan structs.TransactionWithMeta, NumberOfTxFields),
		txPool: NewValuesPool(20, NumberOfTxFields, 20),
		blBuff: make(chan structs.BlockWithMeta, 20),
		blPool: NewValuesPool(20, 8, 20),
	}
}

// Flush contents of buffers to database
func (d *Driver) Flush() error {
	if len(d.txBuff) > 0 {
		if err := flushTransactions(context.Background(), d); err != nil {
			return err
		}
	}
	if len(d.blBuff) > 0 {
		if err := flushBlocks(context.Background(), d); err != nil {
			return err
		}
	}

	return nil
}
