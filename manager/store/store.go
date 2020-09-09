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

	GetLatestTransaction(ctx context.Context, tx structs.TransactionExtra) (structs.Transaction, error)
}

type BlockStore interface {
	StoreBlock(structs.BlockExtra) error

	GetLatestBlock(ctx context.Context, blx structs.BlockExtra) (structs.Block, error)

	BlockContinuityCheck(ctx context.Context, blx structs.BlockExtra, startHeight, endHeight uint64) ([][2]uint64, error)
	BlockTransactionCheck(ctx context.Context, blx structs.BlockExtra, startHeight, endHeight uint64) ([]uint64, error)
}

type Store struct {
	driver DBDriver
}

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

func (s *Store) GetTransactions(ctx context.Context, tsearch params.TransactionSearch) ([]structs.Transaction, error) {
	return s.driver.GetTransactions(ctx, tsearch)
}

func (s *Store) GetLatestTransaction(ctx context.Context, in structs.TransactionExtra) (out structs.Transaction, err error) {
	return s.driver.GetLatestTransaction(ctx, in)
}

func (s *Store) StoreBlock(bl structs.BlockExtra) error {
	return s.driver.StoreBlock(bl)
}

func (s *Store) GetLatestBlock(ctx context.Context, blx structs.BlockExtra) (structs.Block, error) {
	return s.driver.GetLatestBlock(ctx, blx)
}

func (s *Store) BlockContinuityCheck(ctx context.Context, blx structs.BlockExtra, startHeight, endHeight uint64) ([][2]uint64, error) {
	return s.driver.BlockContinuityCheck(ctx, blx, startHeight, endHeight)
}

func (s *Store) BlockTransactionCheck(ctx context.Context, blx structs.BlockExtra, startHeight, endHeight uint64) ([]uint64, error) {
	return s.driver.BlockTransactionCheck(ctx, blx, startHeight, endHeight)
}
