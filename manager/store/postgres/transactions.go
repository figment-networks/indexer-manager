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

	"github.com/figment-networks/indexer-manager/manager/store/params"
	"github.com/figment-networks/indexer-manager/structs"
	"github.com/lib/pq"
)

// StoreTransaction adds transaction to storage buffer
func (d *Driver) StoreTransaction(tx structs.TransactionWithMeta) error {
	select {
	case d.txBuff <- tx:
	default:
		if err := flushTransactions(context.Background(), d); err != nil {
			return err
		}
		d.txBuff <- tx
	}
	return nil
}

// StoreTransactions adds transactions to storage buffer
func (d *Driver) StoreTransactions(txs []structs.TransactionWithMeta) error {
	for _, t := range txs {
		if err := d.StoreTransaction(t); err != nil {
			return err
		}
	}
	return nil
}

const (
	txInsertHead = `INSERT INTO public.transaction_events("network", "chain_id", "version", "epoch", "height", "hash", "block_hash", "time", "type", "parties", "senders", "recipients", "amount", "fee", "gas_wanted", "gas_used", "memo", "data", "raw") VALUES `
	txInsertFoot = ` ON CONFLICT (network, chain_id, hash)
	DO UPDATE SET height = EXCLUDED.height,
	time = EXCLUDED.time,
	type = EXCLUDED.type,
	parties = EXCLUDED.parties,
	senders = EXCLUDED.senders,
	recipients = EXCLUDED.recipients,
	data = EXCLUDED.data,
	raw = EXCLUDED.raw,
	amount = EXCLUDED.amount,
	block_hash = EXCLUDED.block_hash,
	gas_wanted = EXCLUDED.gas_wanted,
	gas_used = EXCLUDED.gas_used,
	memo = EXCLUDED.memo,
	fee = EXCLUDED.fee`
)

// flushTransactions persist buffer into postgres
func flushTransactions(ctx context.Context, d *Driver) error {

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

ReadAll:
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
			senders := []string{}
			recipients := []string{}
			types := []string{}
			amount := .0
			for _, ev := range t.Events {
				for _, sub := range ev.Sub {
					if len(sub.Recipient) > 0 {
						parties = uniqueEntriesEvTransfer(sub.Recipient, parties)
						recipients = uniqueEntriesEvTransfer(sub.Recipient, recipients)
					}
					if len(sub.Sender) > 0 {
						parties = uniqueEntriesEvTransfer(sub.Sender, parties)
						senders = uniqueEntriesEvTransfer(sub.Recipient, senders)
					}

					if len(sub.Node) > 0 {
						for _, accounts := range sub.Node {
							parties = uniqueEntriesAccount(accounts, parties)
						}
					}

					if sub.Error != nil {
						types = uniqueEntry("error", types)
					}

					types = uniqueEntries(sub.Type, types)
				}
			}

			enc.Encode(t.Events)

			if i > 0 {
				qBuilder.WriteString(`,`)
			}
			qBuilder.WriteString(`(`)
			for j := 1; j < 20; j++ {
				qBuilder.WriteString(`$`)
				current := i*19 + j
				qBuilder.WriteString(strconv.Itoa(current))
				if current == 1 || math.Mod(float64(current), 19) != 0 {
					qBuilder.WriteString(`,`)
				}
			}

			qBuilder.WriteString(`)`)

			base := i * 19
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
			va[base+10] = pq.Array(senders)
			va[base+11] = pq.Array(recipients)
			va[base+12] = pq.Array([]float64{amount})
			va[base+13] = 0
			va[base+14] = t.GasWanted
			va[base+15] = t.GasUsed
			va[base+16] = strings.Map(removeCharacters, t.Memo)
			va[base+17] = buff.String()
			va[base+18] = t.Raw
			i++

			last = base + 18

			buff.Reset()

			// (lukanus): do not exceed allocw
			if i == d.txPool.count-1 {
				break ReadAll
			}

		default:
			break ReadAll
		}
	}

	qBuilder.WriteString(txInsertFoot)

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(qBuilder.String(), va[:last+1]...)
	if err != nil {
		log.Println("Rollback flushTx error: ", err)
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Removes ASCII hex 0-7 causing utf-8 error in db
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
	Inner:
		for _, re := range out {
			if r == re {
				exists = true
				break Inner
			}
		}
		if !exists {
			out = append(out, r)
		}
	}
	return out
}

func uniqueEntriesEvTransfer(in []structs.EventTransfer, out []string) []string {
	for _, r := range in { // (lukanus): faster than a map :)
		var exists bool
	Inner:
		for _, re := range out {
			if r.Account.ID == re {
				exists = true
				break Inner
			}
		}
		if !exists {
			out = append(out, r.Account.ID)
		}
	}
	return out
}

func uniqueEntriesAccount(in []structs.Account, out []string) []string {
	for _, r := range in { // (lukanus): faster than a map :)
		var exists bool
	Inner:
		for _, re := range out {
			if r.ID == re {
				exists = true
				break Inner
			}
		}
		if !exists {
			out = append(out, r.ID)
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

// GetTransactions gets transactions based on given criteria the order is forced to be time DESC
func (d *Driver) GetTransactions(ctx context.Context, tsearch params.TransactionSearch) (txs []structs.Transaction, err error) {
	var i = 1

	parts := []string{}
	data := []interface{}{}

	sortType := "time"

	if tsearch.Network != "" {
		parts = append(parts, "network = $"+strconv.Itoa(i))
		data = append(data, tsearch.Network)
		i++
	}

	if len(tsearch.ChainIDs) > 0 {
		chains := "chain_id IN ("
		for j, c := range tsearch.ChainIDs {
			if j > 0 {
				chains += ","
			}
			data = append(data, c)
			chains += "$" + strconv.Itoa(i)
			i++
		}
		parts = append(parts, chains+")")
	}

	if tsearch.Epoch != "" {
		parts = append(parts, "epoch = $"+strconv.Itoa(i))
		data = append(data, tsearch.Epoch)
		i++
	}

	if tsearch.Hash != "" {
		parts = append(parts, "hash = $"+strconv.Itoa(i))
		data = append(data, tsearch.Hash)
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
		sortType = "height"
		i++
	} else {
		if tsearch.AfterHeight > 0 {
			parts = append(parts, "height > $"+strconv.Itoa(i))
			sortType = "height"
			data = append(data, tsearch.AfterHeight)
			i++
		}

		if tsearch.BeforeHeight > 0 {
			parts = append(parts, "height < $"+strconv.Itoa(i))
			sortType = "height"
			data = append(data, tsearch.BeforeHeight)
			i++
		}
	}

	if len(tsearch.Type) > 0 {
		parts = append(parts, "type @> $"+strconv.Itoa(i))
		data = append(data, pq.Array(tsearch.Type))
		i++
	}

	if len(tsearch.Account) > 0 {
		//parts = append(parts, "$"+strconv.Itoa(i)+"=ANY(parties)")
		parts = append(parts, "parties @> $"+strconv.Itoa(i))
		data = append(data, pq.Array(tsearch.Account))
		i++
	}

	if len(tsearch.Sender) > 0 {
		parts = append(parts, "senders @> $"+strconv.Itoa(i))
		data = append(data, pq.Array(tsearch.Sender))
		i++
	}

	if len(tsearch.Receiver) > 0 {
		//parts = append(parts, "$"+strconv.Itoa(i)+"=ANY(recipients)")
		parts = append(parts, "recipients @> $"+strconv.Itoa(i))
		data = append(data, pq.Array(tsearch.Receiver))
		i++
	}

	if len(tsearch.Memo) > 2 {
		parts = append(parts, "memo ILIKE $"+strconv.Itoa(i))
		data = append(data, fmt.Sprintf("%%%s%%", tsearch.Memo))
		i++
	}

	if !tsearch.AfterTime.IsZero() {
		parts = append(parts, "time >= $"+strconv.Itoa(i))
		data = append(data, tsearch.AfterTime)
		sortType = "time"
		i++
	}

	if !tsearch.BeforeTime.IsZero() {
		parts = append(parts, "time <= $"+strconv.Itoa(i))
		data = append(data, tsearch.BeforeTime)
		sortType = "time"
	}

	qBuilder := strings.Builder{}
	qBuilder.WriteString("SELECT id, chain_id, version, epoch, height, hash, block_hash, time, gas_wanted, gas_used, memo, data")
	if tsearch.WithRaw {
		qBuilder.WriteString(", raw ")
	}
	qBuilder.WriteString(" FROM public.transaction_events WHERE ")
	for i, par := range parts {
		if i != 0 {
			qBuilder.WriteString(" AND ")
		}
		qBuilder.WriteString(par)
	}

	if sortType == "time" {
		qBuilder.WriteString(" ORDER BY time DESC")
	} else {
		qBuilder.WriteString(" ORDER BY height DESC")
	}

	if tsearch.Limit > 0 {
		qBuilder.WriteString(" LIMIT " + strconv.FormatUint(uint64(tsearch.Limit), 10))

		if tsearch.Offset > 0 {
			qBuilder.WriteString(" OFFSET " + strconv.FormatUint(uint64(tsearch.Offset), 10))
		}
	}

	a := qBuilder.String()
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
		if tsearch.WithRaw {
			err = rows.Scan(&tx.ID, &tx.ChainID, &tx.Version, &tx.Epoch, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events, &tx.Raw)
		} else {
			err = rows.Scan(&tx.ID, &tx.ChainID, &tx.Version, &tx.Epoch, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events)
		}

		if err != nil {
			return nil, err
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

const getLatestTransactionQuery = `SELECT id, version, epoch, height, hash, block_hash, time
									FROM public.transaction_events
									WHERE
										chain_id = $1 AND
										network = $2 AND
										epoch = $3
									ORDER BY time DESC
									LIMIT 1`

// GetLatestTransaction gets latest transaction from database
func (d *Driver) GetLatestTransaction(ctx context.Context, in structs.TransactionWithMeta) (out structs.Transaction, err error) {
	tx := structs.Transaction{}

	d.Flush()

	row := d.db.QueryRowContext(ctx, getLatestTransactionQuery, in.ChainID, in.Network, in.Transaction.Epoch)
	if row == nil {
		return out, params.ErrNotFound
	}

	err = row.Scan(&tx.ID, &tx.Version, &tx.Epoch, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events)
	if err == sql.ErrNoRows {
		return out, params.ErrNotFound
	}
	return out, err
}

const numTransactionHeightsQuery = "SELECT height, count(height) as count FROM public.transaction_events WHERE network = $1 AND chain_id = $2 AND height >= $3 AND height <= $4 GROUP BY height ORDER BY height ASC"

// GetTransactionsHeightsWithTxCount gets the count of transaction groupped by heights
func (d *Driver) GetTransactionsHeightsWithTxCount(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	rows, err := d.db.QueryContext(ctx, numTransactionHeightsQuery, blx.Network, blx.ChainID, startHeight, endHeight)
	switch {
	case err == sql.ErrNoRows:
		return nil, params.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("query error: %w", err)
	default:
	}

	defer rows.Close()

	hcp := heightCountPair{}
	pairs := [][2]uint64{}
	for rows.Next() {
		if err := rows.Scan(&hcp.Height, &hcp.Count); err != nil {
			return pairs, err
		}
		pairs = append(pairs, [2]uint64{hcp.Height, hcp.Count})
	}

	return pairs, nil
}
