package store

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type Store struct {
	db *gorm.DB

	// Heights      HeightsStore
	// Blocks       BlocksStore
	// Accounts     AccountsStore
	// Validators   ValidatorsStore
	// Transactions TransactionsStore
	// Jobs         JobsStore
	// Snarkers     SnarkersStore
	// FeeTransfers FeeTransfersStore
	// Stats        StatsStore
}

// New creates new store
func New(connStr string) (*Store, error) {

	conn, err := gorm.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: conn,
	}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}
