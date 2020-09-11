package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"hash/maphash"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/figment-networks/cosmos-indexer/manager/store/params"
	"github.com/figment-networks/cosmos-indexer/structs"
)

const (
	insertHead = `INSERT INTO public.blocks("network", "chain_id", "version", "epoch", "height", "hash",  "time", "numtxs" ) VALUES `
	insertFoot = ` ON CONFLICT (network, chain_id, epoch, hash)
	DO UPDATE SET
	height = EXCLUDED.height,
	time = EXCLUDED.time,
	numtxs = EXCLUDED.numtxs`
)

func (d *Driver) StoreBlock(bl structs.BlockWithMeta) error {
	select {
	case d.blBuff <- bl:
	default:
		if err := flushB(context.Background(), d); err != nil {
			return err
		}
		d.blBuff <- bl
	}
	return nil
}

func flushB(ctx context.Context, d *Driver) error {

	qBuilder := strings.Builder{}
	qBuilder.WriteString(insertHead)

	var i = 0
	var last = 0
	deduplicate := map[uint64]bool{}

	var h maphash.Hash

	va := d.blPool.Get()
	defer d.blPool.Put(va)
READ_ALL:
	for {
		select {
		case block := <-d.blBuff:
			b := &block.Block

			h.Reset()
			h.WriteString(block.Network)
			h.WriteString(block.ChainID)
			h.WriteString(b.Epoch)
			h.WriteString(b.Hash)
			key := h.Sum64()
			if _, ok := deduplicate[key]; ok {
				// already exists
				continue
			}
			deduplicate[key] = true

			if i > 0 {
				qBuilder.WriteString(`,`)
			}
			qBuilder.WriteString(`(`)
			for j := 1; j < 9; j++ {
				qBuilder.WriteString(`$`)
				current := i*8 + j
				qBuilder.WriteString(strconv.Itoa(current))
				if current == 1 || math.Mod(float64(current), 8) != 0 {
					qBuilder.WriteString(`,`)
				}
			}

			qBuilder.WriteString(`)`)

			base := i * 8
			va[base] = block.Network
			va[base+1] = block.ChainID
			va[base+2] = block.Version
			va[base+3] = b.Epoch
			va[base+4] = b.Height
			va[base+5] = b.Hash
			va[base+6] = b.Time
			va[base+7] = b.NumberOfTransactions
			last = base + 7
			i++

			// (lukanus): do not exceed alloc
			if i == d.blPool.count-1 {
				break READ_ALL
			}
		default:
			break READ_ALL
		}
	}

	h.Reset()
	deduplicate = nil

	qBuilder.WriteString(insertFoot)

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(qBuilder.String(), va[:last+1]...)
	if err != nil {
		log.Println("[DB] Rollback flushB error: ", err)
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *Driver) GetLatestBlock(ctx context.Context, blx structs.BlockWithMeta) (out structs.Block, err error) {
	returnBlx := structs.Block{}

	row := d.db.QueryRowContext(ctx, "SELECT id, epoch, height, hash, time, numtxs FROM public.blocks WHERE version = $1 AND network = $2 ORDER BY time DESC LIMIT 1", blx.ChainID, blx.Network)
	if row == nil {
		return out, params.ErrNotFound
	}

	err = row.Scan(&returnBlx.ID, &returnBlx.Epoch, &returnBlx.Height, &returnBlx.Hash, &returnBlx.Time, &returnBlx.NumberOfTransactions)
	if err == sql.ErrNoRows {
		return returnBlx, params.ErrNotFound
	}
	return returnBlx, err
}

type orderPair struct {
	Height    uint64
	PreHeight uint64
}

func (d *Driver) BlockContinuityCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	pairs := [][2]uint64{}

	if endHeight > 0 {
		// start - X
		row := d.db.QueryRowContext(ctx, "SELECT height FROM blocks WHERE chain_id = $1 AND network = $2 AND version = $3 AND height >= $4 AND height <= $5 ORDER BY height ASC LIMIT 1", blx.ChainID, blx.Network, blx.Version, startHeight, endHeight)
		var height uint64

		if err := row.Scan(&height); err != nil {
			if err == sql.ErrNoRows { // no record exists
				return [][2]uint64{{startHeight, endHeight}}, nil
			}
			return nil, err
		}
		if height != startHeight {
			pairs = append(pairs, [2]uint64{startHeight, height - 1})
		}

		//   X - end
		row = d.db.QueryRowContext(ctx, "SELECT height FROM blocks WHERE chain_id = $1 AND network = $2 AND version = $3 AND height >= $4 AND height <= $5 ORDER BY height DESC LIMIT 1", blx.ChainID, blx.Network, blx.Version, startHeight, endHeight)

		if err := row.Scan(&height); err != nil {
			if err == sql.ErrNoRows { // no record exists
				return [][2]uint64{{startHeight, endHeight}}, nil
			}
			return nil, err
		}
		if height != endHeight {
			pairs = append(pairs, [2]uint64{height + 1, endHeight})
		}
	}

	var (
		rows *sql.Rows
		err  error
	)

	if endHeight == 0 {
		rows, err = d.db.QueryContext(ctx, "SELECT height, pre_height FROM (SELECT height, lag(height) over (order by height) as pre_height FROM blocks WHERE chain_id = $1 AND network = $2 AND version = $3 AND height >= $4 ORDER BY height ASC) as ss WHERE height != pre_height+1;", blx.ChainID, blx.Network, blx.Version, startHeight)
	} else {
		rows, err = d.db.QueryContext(ctx, "SELECT height, pre_height FROM (SELECT height, lag(height) over (order by height) as pre_height FROM blocks WHERE chain_id = $1 AND network = $2 AND version = $3 AND height >= $4 AND height <= $5 ORDER BY height ASC) as ss WHERE height != pre_height+1;", blx.ChainID, blx.Network, blx.Version, startHeight, endHeight)
	}

	switch {
	case err == sql.ErrNoRows:
		return pairs, params.ErrNotFound
	case err != nil:
		return pairs, fmt.Errorf("query error: %w", err)
	default:
	}

	op := orderPair{}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&op.Height, &op.PreHeight); err != nil {
			return pairs, err
		}
		pairs = append(pairs, [2]uint64{op.PreHeight, op.Height})
	}

	return pairs, nil
}

func (d *Driver) BlockTransactionCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([]uint64, error) {
	q := `SELECT t.height, count(t.hash) AS c, b.numtxs
	FROM transaction_events AS t
	LEFT JOIN blocks AS b ON (t.height = b.height)
	WHERE t.version = $1 AND t.network = $2 AND t.height >= $3 AND t.height <= $4
	GROUP BY t.height,b.numtxs
	HAVING count(t.hash) != b.numtxs`

	rows, err := d.db.QueryContext(ctx, q, blx.Version, blx.Network, startHeight, endHeight)

	switch {
	case err == sql.ErrNoRows:
		return nil, params.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("query error: %w", err)
	default:
	}

	problems := []uint64{}

	var height, countA, countB uint64
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&height, &countA, &countB); err != nil {
			return nil, err
		}
		problems = append(problems, height)

	}
	return problems, nil
}
