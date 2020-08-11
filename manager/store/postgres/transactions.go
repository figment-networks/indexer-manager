package postgres

import (
	"bytes"
	"encoding/json"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/lib/pq"
)

func (d *Driver) StoreTransaction(tx structs.TransactionExtra) error {
	select {
	case d.txBuff <- tx:
	default:
		if err := flushTx(d); err != nil {
			return err
		}
		d.txBuff <- tx
	}
	return nil
}

func (d *Driver) StoreTransactions(txs []structs.TransactionExtra) error {
	for _, t := range txs {
		if err := d.StoreTransaction(t); err != nil {
			return err
		}
	}
	return nil
}

func flushTx(d *Driver) error {
	txn, err := d.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("transaction_events",
		"network", "chain_id", "height", "hash", "block_hash", "time", "type", "senders", "recipients", "amount", "fee", "gas_wanted", "gas_used", "memo", "data"))
	if err != nil {
		return err
	}

	buff := &bytes.Buffer{}
	enc := json.NewEncoder(buff)

READ_ALL:
	for {
		select {
		case transaction := <-d.txBuff:
			t := transaction.Transaction
			for _, ev := range t.Events {
				recipients := []string{}
				senders := []string{}
				amount := .0
				for _, sub := range ev.Sub {
					if len(sub.Recipient) > 0 {
						uniqueEntries(sub.Recipient, recipients)
					}
					if len(sub.Sender) > 0 {
						uniqueEntries(sub.Sender, senders)
					}
					//	amount += sub.Amount.Numeric
				}
				enc.Encode(ev)
				_, err = stmt.Exec(transaction.Network, transaction.ChainID, t.Height, t.Hash, t.BlockHash, t.Time, ev.Type, pq.Array(senders), pq.Array(recipients), amount, 0, t.GasWanted, t.GasUsed, t.Memo, buff.String())
				buff.Reset()
				if err != nil {
					return err
				}
			}
		default:
			break READ_ALL
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return txn.Commit()
}

func uniqueEntries(in, out []string) {
	for _, r := range in { // (lukanus): faster than a map :)
		var exists bool
		for _, re := range out {
			if r == re {
				exists = true
				break
			}
		}
		if !exists {
			out = append(out, r)
		}
	}
}

/*
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
*/
