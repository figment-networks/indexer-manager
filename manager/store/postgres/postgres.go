package postgres

import (
	"context"
	"database/sql"

	"github.com/figment-networks/cosmos-indexer/structs"
)

type Driver struct {
	db *sql.DB

	txBuff chan structs.TransactionExtra
	blBuff chan structs.BlockExtra
}

func New(ctx context.Context, db *sql.DB) *Driver {

	return &Driver{
		db:     db,
		txBuff: make(chan structs.TransactionExtra, 10),
		blBuff: make(chan structs.BlockExtra, 10),
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
