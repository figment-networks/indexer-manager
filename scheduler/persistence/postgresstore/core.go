package postgresstore

import (
	"context"

	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
)

func (d *Driver) GetLatest(ctx context.Context, kind, network, version string) (lRec structures.LatestRecord, err error) {
	row := d.db.QueryRowContext(ctx, "SELECT hash, height, time, nonce FROM schedule_latest WHERE network = $1 AND version = $2 AND kind = $3 ORDER BY time DESC LIMIT 1", network, version, kind)
	if row == nil {
		return lRec, structures.ErrDoesNotExists
	}

	if err := row.Scan(&lRec.Hash, &lRec.Height, &lRec.Time, &lRec.Nonce); err != nil {
		return lRec, err
	}

	return lRec, nil
}

func (d *Driver) SetLatest(ctx context.Context, kind, network, version string, lRec structures.LatestRecord) (err error) {
	_, err = d.db.ExecContext(ctx, "INSERT INTO schedule_latest time, network, version, kind, hash, height, nonce VALUES ($1,$2,$3,$4,$5,$6,$7)",
		lRec.Time, network, version, kind, lRec.Hash, lRec.Height, lRec.Nonce)
	return err
}
