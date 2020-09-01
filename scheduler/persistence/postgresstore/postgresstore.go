package postgresstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/figment-networks/cosmos-indexer/scheduler/persistence/params"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"

	"github.com/google/uuid"
)

type Driver struct {
	db *sql.DB
}

func NewDriver(db *sql.DB) *Driver {
	return &Driver{
		db: db,
	}
}

func (d *Driver) GetConfigs(ctx context.Context, runID uuid.UUID) (rcs []structures.RunConfig, err error) {

	rows, err := d.db.QueryContext(ctx, "SELECT id, runID, network, version, duration, kind FROM schedule WHERE runID = $1", runID)
	switch {
	case err == sql.ErrNoRows:
		return nil, params.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("query error: %w", err)
	default:
	}

	defer rows.Close()
	for rows.Next() {
		rc := structures.RunConfig{}
		if err := rows.Scan(&rc.ID, &rc.RunID, &rc.Network, &rc.Version, &rc.Duration, &rc.Kind); err != nil {
			return nil, err
		}
		rcs = append(rcs, rc)
	}

	return rcs, nil
}

func (d *Driver) MarkRunning(ctx context.Context, runID, configID uuid.UUID) error {

	res, err := d.db.ExecContext(ctx, "UPDATE schedule SET runID = $1 WHERE id = $2 ", runID, configID)
	if err != nil {
		return err
	}

	i, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if i == 0 {
		return errors.New("No rows updated")
	}

	return nil
}

func (d *Driver) AddConfig(ctx context.Context, rc structures.RunConfig) (err error) {

	row := d.db.QueryRowContext(ctx, "SELECT runID, duration FROM schedule WHERE kind = $1 AND version = $2 AND network = $3 ", rc.Kind, rc.Version, rc.Network)

	var rID uuid.UUID
	var duration time.Duration

	if row != nil {
		if err := row.Scan(&rID, &duration); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			res, err := d.db.ExecContext(ctx, "INSERT INTO schedule (runID, network, version, duration, kind) VALUES ($1, $2, $3, $4, $5)", rc.RunID, rc.Network, rc.Version, rc.Duration, rc.Kind)
			if err != nil {
				return err
			}

			i, err := res.RowsAffected()
			if err != nil {
				return err
			}

			if i == 0 {
				return errors.New("No rows updated")
			}

			return nil
		}

		if (duration > 0 && duration != rc.Duration) || rc.RunID != rID {
			res, err := d.db.ExecContext(ctx, "UPDATE schedule SET duration = $1, runID = $2  WHERE network = $3 AND version = $4 AND kind = $5", rc.Duration, rc.RunID, rc.Network, rc.Version, rc.Kind)
			if err != nil {
				return err
			}
			i, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if i == 0 {
				return errors.New("No rows updated")
			}
			return nil
		}

		return params.ErrAlreadyRegistred
	}
	return nil
}
