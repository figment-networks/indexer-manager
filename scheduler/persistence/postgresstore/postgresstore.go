package postgresstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

	rows, err := d.db.QueryContext(ctx, "SELECT runID, network, version, duration, kind FROM schedule WHERE runID != $1", runID)
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
		if err := rows.Scan(&rc.RunID, &rc.Network, &rc.Version, &rc.Duration); err != nil {
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

	row := d.db.QueryRowContext(ctx, "SELECT runID FROM schedule WHERE kind = $1 AND version = $2 AND network = $3 ", rc.Kind, rc.Version, rc.Network)
	if row != nil {
		var rID uuid.UUID
		row.Scan(&rID)
		return errors.New("Already Registred")
	}

	res, err := d.db.ExecContext(ctx, "INSERT INTO schedule SET runID = $1, network = $2, version = $3, duration = $4, kind = $5", rc.RunID, rc.Network, rc.Version, rc.Duration, rc.Kind)
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
