package postgres

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/figment-networks/cosmos-indexer/manager/store/params"
	"github.com/figment-networks/cosmos-indexer/structs"
)

func (d *Driver) StoreBlock(bl structs.BlockExtra) error {
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
	qBuilder.WriteString(`INSERT INTO public.blocks("network", "chain_id", "version", "epoch", "height", "hash",  "time" ) VALUES `)

	var i = 0
	valueArgs := []interface{}{}
READ_ALL:
	for {
		select {
		case block := <-d.blBuff:
			b := block.Block
			if i > 0 {
				qBuilder.WriteString(`,`)
			}
			qBuilder.WriteString(`(`)
			for j := 1; j < 8; j++ {
				qBuilder.WriteString(`$`)
				current := i*7 + j
				qBuilder.WriteString(strconv.Itoa(current))
				if current == 1 || math.Mod(float64(current), 7) != 0 {
					qBuilder.WriteString(`,`)
				}
			}

			qBuilder.WriteString(`)`)

			valueArgs = append(valueArgs, block.Network)
			valueArgs = append(valueArgs, block.ChainID)
			valueArgs = append(valueArgs, "0.0.1")
			valueArgs = append(valueArgs, b.Epoch)
			valueArgs = append(valueArgs, b.Height)
			valueArgs = append(valueArgs, b.Hash)
			valueArgs = append(valueArgs, b.Time)
			i++
		default:
			break READ_ALL
		}
	}

	qBuilder.WriteString(` ON CONFLICT (network, chain_id, epoch, hash) DO NOTHING`)

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

func (d *Driver) GetLatestBlock(ctx context.Context, blx structs.BlockExtra) (out structs.Block, err error) {
	returnBlx := structs.Block{}

	row := d.db.QueryRowContext(ctx, "SELECT id, epoch, height, hash, time FROM public.blocks WHERE version = $1 AND network = $2 ORDER BY time DESC LIMIT 1", blx.ChainID, blx.Network)
	if row == nil {
		return out, params.ErrNotFound
	}

	err = row.Scan(&returnBlx.ID, &returnBlx.Epoch, &returnBlx.Height, &returnBlx.Hash, &returnBlx.Time)

	return returnBlx, err
}
