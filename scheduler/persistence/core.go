package persistence

import (
	"context"

	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"github.com/google/uuid"
)

type CDriver interface {
	AddConfig(ctx context.Context, rc structures.RunConfig) (err error)
	GetConfigs(ctx context.Context, runID uuid.UUID) (rcs []structures.RunConfig, err error)
	MarkRunning(ctx context.Context, runID, configID uuid.UUID) (err error)
}

type CoreStorage struct {
	Driver CDriver
}

func (cs *CoreStorage) AddConfig(ctx context.Context, rc structures.RunConfig) (err error) {
	return cs.Driver.AddConfig(ctx, rc)
}

func (cs *CoreStorage) GetConfigs(ctx context.Context, runID uuid.UUID) (rcs []structures.RunConfig, err error) {
	return cs.Driver.GetConfigs(ctx, runID)
}

func (cs *CoreStorage) MarkRunning(ctx context.Context, runID, configID uuid.UUID) (err error) {
	return cs.Driver.MarkRunning(ctx, runID, configID)
}
