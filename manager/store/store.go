package store

import (
	"context"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/store/params"
	"github.com/figment-networks/cosmos-indexer/structs"
)

type DBDriver interface {
	TransactionStore
	BlockStore
	FlushBuffered
}

type DataStore interface {
	TransactionStore
	BlockStore
}

type FlushBuffered interface {
	Flush() error
}

type TransactionStore interface {
	StoreTransaction(structs.TransactionExtra) error
	StoreTransactions([]structs.TransactionExtra) error

	GetTransactions(ctx context.Context, tsearch params.TransactionSearch) ([]structs.Transaction, error)
}

type BlockStore interface {
	StoreBlock(structs.Block) error
}

type Store struct {
	driver DBDriver
}

//func New(conn *sql.Conn, driver DBDriver) *Store {
func New(driver DBDriver) *Store {
	return &Store{driver: driver}
}

func (s *Store) Run(ctx context.Context, dur time.Duration) {
	tckr := time.NewTicker(dur)
	defer tckr.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tckr.C:
			s.driver.Flush()
		}
	}
}

func (s *Store) StoreTransaction(tx structs.TransactionExtra) error {
	return s.driver.StoreTransaction(tx)
}

func (s *Store) StoreTransactions(txs []structs.TransactionExtra) error {
	return s.driver.StoreTransactions(txs)
}

func (s *Store) StoreBlock(bl structs.Block) error {
	return s.driver.StoreBlock(bl)
}

func (s *Store) GetTransactions(ctx context.Context, tsearch params.TransactionSearch) ([]structs.Transaction, error) {
	return s.driver.GetTransactions(ctx, tsearch)
}
