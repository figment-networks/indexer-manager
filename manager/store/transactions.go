package store

import (
	"errors"
	"fmt"

	shared "github.com/figment-networks/cosmos-indexer/structs"
	"github.com/jinzhu/gorm"
)

var (
	ErrNotFound = errors.New("record not found")
)

// CreateIfNotExists creates the transaction if it does not exist
func (s *Store) CreateIfNotExists(t *shared.Transaction) error {
	_, err := s.findByHash(t.Hash)
	if isNotFound(err) {
		err := s.db.Create(t).Error
		return checkErr(err)
	}
	return err
}

// findByHash returns a transaction for a given hash
func (s *Store) findByHash(hash string) (*shared.Transaction, error) {
	return s.findBy("hash", hash)
}

func (s *Store) findBy(key string, value interface{}) (*shared.Transaction, error) {
	result := &shared.Transaction{}
	err := s.db.
		Model(result).
		Where(fmt.Sprintf("%s = ?", key), value).
		Take(result).
		Error

	return result, checkErr(err)
}

func checkErr(err error) error {
	if gorm.IsRecordNotFoundError(err) {
		return ErrNotFound
	}
	return err
}

func isNotFound(err error) bool {
	return gorm.IsRecordNotFoundError(err) || err == ErrNotFound
}
