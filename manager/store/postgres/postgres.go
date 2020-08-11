package postgres

import (
	"database/sql"
	"log"

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

func New(db *sql.DB) *Driver {
	return &Driver{
		db:     db,
		txBuff: make(chan structs.TransactionExtra, 10),
		blBuff: make(chan structs.Block, 10),
	}
}

func RunMigrations(srcPath string, dbURL string) error {
	log.Println("using migrations from", srcPath)
	m, err := migrate.New(srcPath, dbURL)
	if err != nil {
		return err
	}

	log.Println("running migrations")
	return m.Up()
}

func (d *Driver) Flush() error {

	if len(d.txBuff) > 0 {
		if err := flushTx(d); err != nil {
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
