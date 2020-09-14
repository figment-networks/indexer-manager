package postgresstore

import (
	"context"
	"database/sql"

	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
)

func (d *Driver) GetLatest(ctx context.Context, kind, network, chainID, version string) (lRec structures.LatestRecord, err error) {
	row := d.db.QueryRowContext(ctx, "SELECT hash, height, time, nonce FROM schedule_latest WHERE network = $1 AND chain_id = $2 AND version = $3 AND kind = $4 ORDER BY time DESC LIMIT 1", network, chainID, version, kind)
	if row != nil {
		if err := row.Scan(&lRec.Hash, &lRec.Height, &lRec.Time, &lRec.Nonce); err != nil {
			if err == sql.ErrNoRows {
				return lRec, structures.ErrDoesNotExists
			}

			return lRec, err
		}
	}
	return lRec, nil
}

func (d *Driver) SetLatest(ctx context.Context, kind, network, chainID, version string, lRec structures.LatestRecord) (err error) {
	_, err = d.db.ExecContext(ctx, "INSERT INTO schedule_latest (time, network, chain_id, version, kind, hash, height, nonce) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		lRec.Time, network, chainID, version, kind, lRec.Hash, lRec.Height, lRec.Nonce)
	return err
}
