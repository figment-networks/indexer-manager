package postgres

import (
	"context"
	"database/sql"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Driver struct {
	db *sql.DB

	txBuff chan structs.TransactionExtra
	blBuff chan structs.Block
}

func New(ctx context.Context, db *sql.DB) *Driver {

	return &Driver{
		db:     db,
		txBuff: make(chan structs.TransactionExtra, 10),
		blBuff: make(chan structs.Block, 10),
	}
}
func RunMigrations(srcPath string, dbURL string) error {
	m, err := migrate.New(srcPath, dbURL)
	defer m.Close()

	if err != nil {
		return err
	}

	return m.Up()
}

func (d *Driver) Flush() error {

	if len(d.txBuff) > 0 {
		if err := flushTx(context.Background(), d); err != nil {
			return err
		}
	}
	if len(d.blBuff) > 0 {
		if err := flushB(d); err != nil {
			return err
		}
	}

	return nil
}

func (d *Driver) StoreBlock(bl structs.Block) error {
	select {
	case d.blBuff <- bl:
	default:
		if err := flushB(d); err != nil {
			return err
		}
		d.blBuff <- bl
	}
	return nil
}

func flushB(d *Driver) error {
	return nil
}
