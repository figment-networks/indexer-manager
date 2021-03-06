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
	"time"

	"github.com/figment-networks/indexer-manager/manager/store/params"
	"github.com/figment-networks/indexer-manager/structs"
)

const (
	insertHead = `INSERT INTO public.blocks("network", "chain_id", "version", "epoch", "height", "hash",  "time", "numtxs" ) VALUES `
	insertFoot = ` ON CONFLICT (network, chain_id, hash)
	DO UPDATE SET
	height = EXCLUDED.height,
	time = EXCLUDED.time,
	numtxs = EXCLUDED.numtxs`
)

// StoreBlock appends data to buffer
func (d *Driver) StoreBlock(bl structs.BlockWithMeta) error {
	select {
	case d.blBuff <- bl:
	default:
		if err := flushBlocks(context.Background(), d); err != nil {
			return err
		}
		d.blBuff <- bl
	}
	return nil
}

// flush buffer of blocks into database
func flushBlocks(ctx context.Context, d *Driver) error {
	qBuilder := strings.Builder{}
	qBuilder.WriteString(insertHead)

	var i = 0
	var last = 0
	deduplicate := map[uint64]bool{}

	var h maphash.Hash

	va := d.blPool.Get()
	defer d.blPool.Put(va)
ReadAll:
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
				break ReadAll
			}
		default:
			break ReadAll
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

const getLatestBlockQuery = `SELECT id, epoch, height, hash, time, numtxs
							FROM public.blocks
							WHERE
								chain_id = $1 AND
								network = $2
							ORDER BY time DESC
							LIMIT 1`

// GetLatestBlock gets latest block
func (d *Driver) GetLatestBlock(ctx context.Context, blx structs.BlockWithMeta) (out structs.Block, err error) {
	returnBlx := structs.Block{}

	row := d.db.QueryRowContext(ctx, getLatestBlockQuery, blx.ChainID, blx.Network)
	if row == nil {
		return out, params.ErrNotFound
	}

	err = row.Scan(&returnBlx.ID, &returnBlx.Epoch, &returnBlx.Height, &returnBlx.Hash, &returnBlx.Time, &returnBlx.NumberOfTransactions)
	if err == sql.ErrNoRows {
		return returnBlx, params.ErrNotFound
	}
	return returnBlx, err
}

const getBlockForMinTimeQuery = `SELECT id, epoch, height, hash, time, numtxs
							FROM public.blocks
							WHERE
								chain_id = $1 AND
								network = $2 AND
								time >= $3
							ORDER BY time ASC
							LIMIT 1`

// GetBlockForMinTime returns first block that comes on or after given time
func (d *Driver) GetBlockForMinTime(ctx context.Context, blx structs.BlockWithMeta, time time.Time) (out structs.Block, err error) {
	returnBlx := structs.Block{}

	row := d.db.QueryRowContext(ctx, getBlockForMinTimeQuery, blx.ChainID, blx.Network, time)
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

const (
	blockContinuityCheckLowerRangeQuery = `
	SELECT height
	FROM blocks
	WHERE
		chain_id = $1 AND
		network = $2 AND
		height >= $3 AND
		height <= $4
	ORDER BY height ASC
	LIMIT 1`

	blockContinuityCheckUpperRangeQuery = `
	SELECT height
	FROM blocks
	WHERE
		chain_id = $1 AND
		network = $2 AND
		height >= $3 AND
		height <= $4
	ORDER BY height DESC
	LIMIT 1`

	blockContinuityCheckPreviousBlockPresent = `
	SELECT height, pre_height
	FROM
		(	SELECT
				height,
				lag(height) over (order by height) as pre_height
			FROM blocks
			WHERE
				chain_id = $1 AND
				network = $2 AND
				height >= $3
			ORDER BY height ASC
		) AS ss
	WHERE
		height != pre_height+1;`

	blockContinuityCheckPreviousBlockPresentRange = `
	SELECT height, pre_height
	FROM
		(	SELECT
				height,
				lag(height) over (order by height) as pre_height
			FROM blocks
			WHERE
				chain_id = $1 AND
				network = $2 AND
				height >= $3 AND
				height <= $4
			ORDER BY height ASC
		) AS ss
	WHERE height != pre_height+1;
	`
)

// BlockContinuityCheck check data consistency between given ranges in network / chain
func (d *Driver) BlockContinuityCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	pairs := [][2]uint64{}

	// get the range before and after (if applies)
	if endHeight > 0 {
		// start - X
		row := d.db.QueryRowContext(ctx, blockContinuityCheckLowerRangeQuery, blx.ChainID, blx.Network, startHeight, endHeight)
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
		row = d.db.QueryRowContext(ctx, blockContinuityCheckUpperRangeQuery, blx.ChainID, blx.Network, startHeight, endHeight)

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
		rows, err = d.db.QueryContext(ctx, blockContinuityCheckPreviousBlockPresent, blx.ChainID, blx.Network, startHeight)
	} else {
		rows, err = d.db.QueryContext(ctx, blockContinuityCheckPreviousBlockPresentRange, blx.ChainID, blx.Network, startHeight, endHeight)
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

const blockTransactionCheckQuery = `SELECT b.height, b.numtxs, count(t.height) AS c
FROM blocks AS b
LEFT JOIN transaction_events AS t ON (t.height = b.height AND t.network = b.network AND b.chain_id = t.chain_id)
WHERE  b.chain_id = $1 AND b.network = $2  AND b.height >= $3 AND b.height <= $4
GROUP BY b.height, b.numtxs
HAVING count(t.hash) != b.numtxs
ORDER BY b.height ASC`

// BlockTransactionCheck check if every block has correct, corresponding number of transactions
func (d *Driver) BlockTransactionCheck(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([]uint64, error) {

	rows, err := d.db.QueryContext(ctx, blockTransactionCheckQuery, blx.ChainID, blx.Network, startHeight, endHeight)

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

type heightCountPair struct {
	Height uint64
	Count  uint64
}

const blockTransactionHeightsQuery = "SELECT height, numtxs FROM blocks WHERE network = $1 AND chain_id = $2 AND height >= $3 AND height <= $4 ORDER BY height ASC"

// GetBlocksHeightsWithNumTx gets heights and number of transactions for each block in range
func (d *Driver) GetBlocksHeightsWithNumTx(ctx context.Context, blx structs.BlockWithMeta, startHeight, endHeight uint64) ([][2]uint64, error) {
	rows, err := d.db.QueryContext(ctx, blockTransactionHeightsQuery, blx.Network, blx.ChainID, startHeight, endHeight)
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
