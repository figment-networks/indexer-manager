package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/lib/pq"
)

func (d *Driver) StoreTransaction(tx structs.TransactionExtra) error {
	select {
	case d.txBuff <- tx:
	default:
		if err := flushTx(context.Background(), d); err != nil {
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

func flushTx(ctx context.Context, d *Driver) error {

	buff := &bytes.Buffer{}
	enc := json.NewEncoder(buff)

	qBuilder := strings.Builder{}
	qBuilder.WriteString(`INSERT INTO public.transaction_events("network", "chain_id", "event_id", "height", "hash", "block_hash", "time", "type", "senders", "recipients", "amount", "fee", "gas_wanted", "gas_used", "memo", "data") VALUES `)

	var i = 0
	valueArgs := []interface{}{}
READ_ALL:
	for {
		select {
		case transaction := <-d.txBuff:
			t := transaction.Transaction
			for _, ev := range t.Events {
				recipients := []string{}
				senders := []string{}
				types := []string{}
				amount := .0

				for _, sub := range ev.Sub {
					if len(sub.Recipient) > 0 {
						recipients = uniqueEntries(sub.Recipient, recipients)
					}
					if len(sub.Sender) > 0 {
						senders = uniqueEntries(sub.Sender, senders)
					}
					types = uniqueEntry(sub.Type, types)
				}

				enc.Encode(ev.Sub)
				if i > 0 {
					qBuilder.WriteString(`,`)
				}

				qBuilder.WriteString(`(`)
				for j := 1; j < 17; j++ {
					qBuilder.WriteString(`$`)
					current := i*16 + j
					qBuilder.WriteString(strconv.Itoa(current))
					if current == 1 || math.Mod(float64(current), 16) != 0 {
						qBuilder.WriteString(`,`)
					}
				}

				qBuilder.WriteString(`)`)
				valueArgs = append(valueArgs, transaction.Network)
				valueArgs = append(valueArgs, transaction.ChainID)
				valueArgs = append(valueArgs, ev.ID)
				valueArgs = append(valueArgs, t.Height)
				valueArgs = append(valueArgs, t.Hash)
				valueArgs = append(valueArgs, t.BlockHash)
				valueArgs = append(valueArgs, t.Time)
				valueArgs = append(valueArgs, pq.Array(types))
				valueArgs = append(valueArgs, pq.Array(senders))
				valueArgs = append(valueArgs, pq.Array(recipients))
				valueArgs = append(valueArgs, amount)
				valueArgs = append(valueArgs, 0)
				valueArgs = append(valueArgs, t.GasWanted)
				valueArgs = append(valueArgs, t.GasUsed)
				valueArgs = append(valueArgs, t.Memo)
				valueArgs = append(valueArgs, buff.String())
				buff.Reset()
				i++
			}
		default:
			break READ_ALL
		}
	}

	qBuilder.WriteString(` ON CONFLICT (network, chain_id, hash, event_id) DO NOTHING`)

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	a := qBuilder.String()
	_, err = tx.Exec(a, valueArgs...)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func uniqueEntries(in, out []string) []string {

	for _, r := range in { // (lukanus): faster than a map :)
		var exists bool
	INNER:
		for _, re := range out {
			if r == re {
				exists = true
				break INNER
			}
		}
		if !exists {
			out = append(out, r)
		}
	}
	return out
}

func uniqueEntry(in string, out []string) []string {
	for _, re := range out {
		if in == re {
			return out
		}
	}
	return append(out, in)
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
