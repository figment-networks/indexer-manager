package postgres

import (
	"context"
	"database/sql"

	"github.com/figment-networks/cosmos-indexer/structs"
)

type Driver struct {
	db *sql.DB

	txBuff chan structs.TransactionWithMeta
	txPool *ValuesPool

	blBuff chan structs.BlockWithMeta
	blPool *ValuesPool
}

func New(ctx context.Context, db *sql.DB) *Driver {
	return &Driver{
		db:     db,
		txBuff: make(chan structs.TransactionWithMeta, 20),
		txPool: NewValuesPool(20, 18, 20),
		blBuff: make(chan structs.BlockWithMeta, 20),
		blPool: NewValuesPool(20, 8, 20),
	}
}

func (d *Driver) Flush() error {
	if len(d.txBuff) > 0 {
		if err := flushTx(context.Background(), d); err != nil {
			return err
		}
	}
	if len(d.blBuff) > 0 {
		if err := flushB(context.Background(), d); err != nil {
			return err
		}
	}

	return nil
}
