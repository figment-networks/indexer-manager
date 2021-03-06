package store

import (
	"context"
	"time"

	"github.com/figment-networks/indexer-manager/manager/store/params"
	"github.com/figment-networks/indexer-manager/structs"
)

//go:generate mockgen -destination=./mocks/mock_store.go -package=mocks github.com/figment-networks/indexer-manager/manager/store DataStore

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
	StoreTransaction(structs.TransactionWithMeta) error
	StoreTransactions([]structs.TransactionWithMeta) error

	GetTransactions(ctx context.Context, tsearch params.TransactionSearch) ([]structs.Transaction, error)

	GetLatestTransaction(ctx context.Context, tx structs.TransactionWithMeta) (structs.Transaction, error)

	GetTransactionsHeightsWithTxCount(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error)
}

type BlockStore interface {
	StoreBlock(structs.BlockWithMeta) error

	GetLatestBlock(ctx context.Context, blx structs.BlockWithMeta) (structs.Block, error)
	GetBlockForMinTime(ctx context.Context, blx structs.BlockWithMeta, time time.Time) (structs.Block, error)

	BlockContinuityCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error)
	BlockTransactionCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([]uint64, error)

	GetBlocksHeightsWithNumTx(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error)
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

func (s *Store) StoreTransaction(tx structs.TransactionWithMeta) error {
	return s.driver.StoreTransaction(tx)
}

func (s *Store) StoreTransactions(txs []structs.TransactionWithMeta) error {
	return s.driver.StoreTransactions(txs)
}

func (s *Store) GetTransactions(ctx context.Context, tsearch params.TransactionSearch) ([]structs.Transaction, error) {
	return s.driver.GetTransactions(ctx, tsearch)
}

func (s *Store) GetLatestTransaction(ctx context.Context, in structs.TransactionWithMeta) (out structs.Transaction, err error) {
	return s.driver.GetLatestTransaction(ctx, in)
}

func (s *Store) StoreBlock(bl structs.BlockWithMeta) error {
	return s.driver.StoreBlock(bl)
}

func (s *Store) GetLatestBlock(ctx context.Context, blx structs.BlockWithMeta) (structs.Block, error) {
	return s.driver.GetLatestBlock(ctx, blx)
}

func (s *Store) GetBlockForMinTime(ctx context.Context, blx structs.BlockWithMeta, time time.Time) (structs.Block, error) {
	return s.driver.GetBlockForMinTime(ctx, blx, time)
}

func (s *Store) BlockContinuityCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	return s.driver.BlockContinuityCheck(ctx, blx, startHeight, endHeight)
}

func (s *Store) BlockTransactionCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([]uint64, error) {
	return s.driver.BlockTransactionCheck(ctx, blx, startHeight, endHeight)
}

func (s *Store) GetTransactionsHeightsWithTxCount(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	return s.driver.GetTransactionsHeightsWithTxCount(ctx, blx, startHeight, endHeight)
}

func (s *Store) GetBlocksHeightsWithNumTx(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	return s.driver.GetBlocksHeightsWithNumTx(ctx, blx, startHeight, endHeight)
}
