package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/maphash"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/figment-networks/cosmos-indexer/manager/store/params"
	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/lib/pq"
)

func (d *Driver) StoreTransaction(tx structs.TransactionWithMeta) error {
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

func (d *Driver) StoreTransactions(txs []structs.TransactionWithMeta) error {
	for _, t := range txs {
		if err := d.StoreTransaction(t); err != nil {
			return err
		}
	}
	return nil
}

const (
	txInsertHead = `INSERT INTO public.transaction_events("network", "chain_id", "version", "epoch", "height", "hash", "block_hash", "time", "type", "parties", "amount", "fee", "gas_wanted", "gas_used", "memo", "data") VALUES `
	txInsertFoot = ` ON CONFLICT (network, chain_id, epoch, hash)
	DO UPDATE SET height = EXCLUDED.height,
	time = EXCLUDED.time,
	type = EXCLUDED.type,
	parties = EXCLUDED.parties,
	data = EXCLUDED.data,
	amount = EXCLUDED.amount,
	block_hash = EXCLUDED.block_hash,
	gas_wanted = EXCLUDED.gas_wanted,
	gas_used = EXCLUDED.gas_used,
	memo = EXCLUDED.memo,
	fee = EXCLUDED.fee`
)

func flushTx(ctx context.Context, d *Driver) error {

	buff := &bytes.Buffer{}
	enc := json.NewEncoder(buff)

	qBuilder := strings.Builder{}
	qBuilder.WriteString(txInsertHead)

	var i = 0
	var last = 0

	var h maphash.Hash

	va := d.txPool.Get()
	defer d.txPool.Put(va)
	deduplicate := map[uint64]bool{}

READ_ALL:
	for {
		select {
		case transaction := <-d.txBuff:
			t := &transaction.Transaction

			h.Reset()
			h.WriteString(transaction.Network)
			h.WriteString(t.ChainID)
			h.WriteString(t.Epoch)
			h.WriteString(t.Hash)
			key := h.Sum64()
			if _, ok := deduplicate[key]; ok {
				// already exists
				continue
			}
			deduplicate[key] = true

			parties := []string{}
			types := []string{}
			amount := .0
			for _, ev := range t.Events {
				for _, sub := range ev.Sub {
					if len(sub.Recipient) > 0 {
						parties = uniqueEntries(sub.Recipient, parties)
					}
					if len(sub.Sender) > 0 {
						parties = uniqueEntries(sub.Sender, parties)
					}

					if len(sub.Validator) > 0 {
						for _, v := range sub.Validator {
							parties = uniqueEntries(v, parties)
						}
					}

					if len(sub.Feeder) > 0 {
						parties = uniqueEntries(sub.Feeder, parties)
					}

					if sub.Error != nil {
						types = uniqueEntry("error", types)
					}

					types = uniqueEntry(sub.Type, types)
				}
			}

			err := enc.Encode(t.Events)
			if err != nil {

			}

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
			/*
				log.Printf("Append :  %+v ", t)
				log.Printf("Append_2 :  %#v  ", t)

				log.Println("t.Memo: ", t.Height, " ", t.Memo)
				log.Println("string: ", buff.String())
			*/

			base := i * 16
			va[base] = transaction.Network
			va[base+1] = t.ChainID
			va[base+2] = transaction.Version
			va[base+3] = t.Epoch
			va[base+4] = t.Height
			va[base+5] = t.Hash
			va[base+6] = t.BlockHash
			va[base+7] = t.Time
			va[base+8] = pq.Array(types)
			va[base+9] = pq.Array(parties)
			va[base+10] = pq.Array([]float64{amount})
			va[base+11] = 0
			va[base+12] = t.GasWanted
			va[base+13] = t.GasUsed
			va[base+14] = strings.Map(removeCharacters, t.Memo)
			va[base+15] = buff.String()
			i++

			last = base + 15

			buff.Reset()

			// (lukanus): do not exceed allocw
			if i == d.txPool.count-1 {
				break READ_ALL
			}

		default:
			break READ_ALL
		}
	}

	qBuilder.WriteString(txInsertFoot)

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	//	log.Printf("Query  :  %s ", qBuilder.String())
	//	log.Printf("Buffer :  %+v ", va)
	//	log.Printf("BufferPart :  %+v ", va[:last+1])
	_, err = tx.Exec(qBuilder.String(), va[:last+1]...)
	if err != nil {
		log.Println("Rollback flushTx error: ", err)
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Removes ASCII hex 0-7 causingf utf-8 error in db
func removeCharacters(r rune) rune {
	if r < 7 {
		return -1
	}
	return r
}

func uniqueEntries(in, out []string) []string {
	for _, r := range in { // (lukanus): faster than a map :)
		if r == "" {
			continue
		}
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
	if in == "" {
		return out
	}
	for _, re := range out {
		if in == re {
			return out
		}
	}
	return append(out, in)
}

func (d *Driver) GetTransactions(ctx context.Context, tsearch params.TransactionSearch) (txs []structs.Transaction, err error) {
	var i = 1

	parts := []string{}
	data := []interface{}{}

	if tsearch.Network != "" {
		parts = append(parts, "network = $"+strconv.Itoa(i))
		data = append(data, tsearch.Network)
		i++
	}

	if tsearch.BlockHash != "" {
		parts = append(parts, "block_hash = $"+strconv.Itoa(i))
		data = append(data, tsearch.BlockHash)
		i++
	}

	if tsearch.Height > 0 {
		parts = append(parts, "height = $"+strconv.Itoa(i))
		data = append(data, tsearch.Height)
		i++
	} else {
		if tsearch.AfterHeight > 0 {
			parts = append(parts, "height > $"+strconv.Itoa(i))
			data = append(data, tsearch.AfterHeight)
			i++
		}

		if tsearch.BeforeHeight > 0 {
			parts = append(parts, "height < $"+strconv.Itoa(i))
			data = append(data, tsearch.BeforeHeight)
			i++
		}
	}

	if len(tsearch.Type) > 0 {
		parts = append(parts, "type @> $"+strconv.Itoa(i))
		data = append(data, pq.Array(tsearch.Type))
		i++
	}

	if tsearch.Account != "" {
		parts = append(parts, "$"+strconv.Itoa(i)+"=ANY(parties)")
		data = append(data, tsearch.Account) // (lukanus): one would be filled
		i++
	}

	if tsearch.Sender != "" {
		parts = append(parts, "$"+strconv.Itoa(i)+"=ANY(senders)")
		data = append(data, tsearch.Sender)
		i++
	}

	if tsearch.Receiver != "" {
		parts = append(parts, "$"+strconv.Itoa(i)+"=ANY(recipients)")
		data = append(data, tsearch.Receiver)
		i++
	}

	if len(tsearch.Memo) > 2 {
		parts = append(parts, "memo ILIKE $"+strconv.Itoa(i))
		data = append(data, fmt.Sprintf("%%%s%%", tsearch.Memo))
		i++
	}

	if !tsearch.StartTime.IsZero() {
		parts = append(parts, "time >= $"+strconv.Itoa(i))
		data = append(data, tsearch.StartTime)
		i++
	}

	if !tsearch.EndTime.IsZero() {
		parts = append(parts, "time <= $"+strconv.Itoa(i))
		data = append(data, tsearch.EndTime)
	}

	qBuilder := strings.Builder{}
	qBuilder.WriteString("SELECT id, version, epoch, height, hash, block_hash, time, gas_wanted, gas_used, memo, data FROM public.transaction_events WHERE ")
	for i, par := range parts {
		if i != 0 {
			qBuilder.WriteString(" AND ")
		}
		qBuilder.WriteString(par)
	}

	qBuilder.WriteString(" ORDER BY time DESC")

	if tsearch.Limit > 0 {
		qBuilder.WriteString(" LIMIT " + strconv.FormatUint(uint64(tsearch.Limit), 10))

		if tsearch.Offset > 0 {
			qBuilder.WriteString(" OFFSET  " + strconv.FormatUint(uint64(tsearch.Limit), 10))
		}
	}

	a := qBuilder.String()
	//log.Printf("DEBUG QUERY: %s %+v ", a, data)
	rows, err := d.db.QueryContext(ctx, a, data...)
	switch {
	case err == sql.ErrNoRows:
		return nil, params.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("query error: %w", err)
	default:
	}

	defer rows.Close()
	for rows.Next() {
		tx := structs.Transaction{}
		if err := rows.Scan(&tx.ID, &tx.Version, &tx.Epoch, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

func (d *Driver) GetLatestTransaction(ctx context.Context, in structs.TransactionWithMeta) (out structs.Transaction, err error) {
	tx := structs.Transaction{}

	d.Flush()

	row := d.db.QueryRowContext(ctx, "SELECT id, version, epoch, height, hash, block_hash, time FROM public.transaction_events WHERE version = $1 AND network = $2 ORDER BY time DESC LIMIT 1", in.ChainID, in.Network)
	if row == nil {
		return out, params.ErrNotFound
	}

	err = row.Scan(&tx.ID, &tx.Version, &tx.Epoch, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events)
	if err == sql.ErrNoRows {
		return out, params.ErrNotFound
	}
	return out, err
}
