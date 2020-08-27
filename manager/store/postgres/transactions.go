package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/figment-networks/cosmos-indexer/manager/store/params"
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
	//	qBuilder.WriteString(`INSERT INTO public.transaction_events("network", "chain_id",  "height", "hash", "block_hash", "time", "type", "senders", "recipients", "amount", "fee", "gas_wanted", "gas_used", "memo", "data") VALUES `)
	qBuilder.WriteString(`INSERT INTO public.transaction_events("network", "chain_id", "version", "height", "hash", "block_hash", "time", "type", "parties", "amount", "fee", "gas_wanted", "gas_used", "memo", "data") VALUES `)

	var i = 0
	valueArgs := []interface{}{}
READ_ALL:
	for {
		select {
		case transaction := <-d.txBuff:
			t := transaction.Transaction
			//recipients := []string{}
			//senders := []string{}
			parties := []string{}
			types := []string{}
			amount := .0
			for _, ev := range t.Events {
				for _, sub := range ev.Sub {
					if len(sub.Recipient) > 0 {
						parties = uniqueEntries(sub.Recipient, parties)
						//recipients = uniqueEntries(sub.Recipient, recipients)
					}
					if len(sub.Sender) > 0 {
						parties = uniqueEntries(sub.Sender, parties)
						//senders = uniqueEntries(sub.Sender, senders)
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

			enc.Encode(t.Events)

			if i > 0 {
				qBuilder.WriteString(`,`)
			}
			qBuilder.WriteString(`(`)
			for j := 1; j < 16; j++ {
				qBuilder.WriteString(`$`)
				current := i*15 + j
				qBuilder.WriteString(strconv.Itoa(current))
				if current == 1 || math.Mod(float64(current), 15) != 0 {
					qBuilder.WriteString(`,`)
				}
			}

			qBuilder.WriteString(`)`)
			valueArgs = append(valueArgs, transaction.Network)
			valueArgs = append(valueArgs, transaction.ChainID)
			valueArgs = append(valueArgs, "0.0.1")
			valueArgs = append(valueArgs, t.Height)
			valueArgs = append(valueArgs, t.Hash)
			valueArgs = append(valueArgs, t.BlockHash)
			valueArgs = append(valueArgs, t.Time)
			valueArgs = append(valueArgs, pq.Array(types))
			//		valueArgs = append(valueArgs, pq.Array(senders))
			//		valueArgs = append(valueArgs, pq.Array(recipients))
			valueArgs = append(valueArgs, pq.Array(parties))
			valueArgs = append(valueArgs, pq.Array([]float64{amount}))
			valueArgs = append(valueArgs, 0)
			valueArgs = append(valueArgs, t.GasWanted)
			valueArgs = append(valueArgs, t.GasUsed)
			valueArgs = append(valueArgs, t.Memo)
			valueArgs = append(valueArgs, buff.String())
			buff.Reset()
			i++
		default:
			break READ_ALL
		}
	}

	qBuilder.WriteString(` ON CONFLICT (network, chain_id, hash ) DO NOTHING`)

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
	}

	if len(tsearch.Type) > 0 {
		parts = append(parts, "type @> $"+strconv.Itoa(i))
		data = append(data, pq.Array(tsearch.Type))
		i++
	}

	if tsearch.Account != "" || tsearch.Sender != "" || tsearch.Receiver != "" {
		parts = append(parts, "$"+strconv.Itoa(i)+"=ANY(parties)")
		data = append(data, tsearch.Account+tsearch.Sender+tsearch.Receiver) // (lukanus): one would be filled
		i++
	}

	/*
		if tsearch.Account != "" {
			parts = append(parts, "($"+strconv.Itoa(i)+"=ANY(senders) OR $"+strconv.Itoa(i+1)+")=ANY(recipients)")
			data = append(data, tsearch.Account)
			data = append(data, tsearch.Account)
			i += 2
		} else {
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
		}
	*/

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
	qBuilder.WriteString("SELECT id, version, height, hash, block_hash, time, gas_wanted, gas_used, memo, data FROM public.transaction_events WHERE ")
	for i, par := range parts {
		if i != 0 {
			qBuilder.WriteString(" AND ")
		}
		qBuilder.WriteString(par)
	}

	qBuilder.WriteString(" ORDER BY time DESC")

	if tsearch.Limit > 0 {
		qBuilder.WriteString(" LIMIT " + strconv.FormatUint(uint64(tsearch.Limit), 64))
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
		if err := rows.Scan(&tx.ID, &tx.Version, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

func (d *Driver) GetLatestTransaction(ctx context.Context, in structs.TransactionExtra) (out structs.Transaction, err error) {
	tx := structs.Transaction{}

	row := d.db.QueryRowContext(ctx, "SELECT id, version, height, hash, block_hash, time FROM public.transaction_events WHERE version = $1 AND network = $2 ORDER BY time DESC LIMIT 1", in.ChainID, in.Network)
	if row == nil {
		return out, params.ErrNotFound
	}

	err = row.Scan(&tx.ID, &tx.Version, &tx.Height, &tx.Hash, &tx.BlockHash, &tx.Time, &tx.GasWanted, &tx.GasUsed, &tx.Memo, &tx.Events)

	return out, err
}
