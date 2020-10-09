package persistence

import (
	"context"

	"github.com/figment-networks/indexer-manager/scheduler/structures"
)

type PDriver interface {
	GetLatest(ctx context.Context, kind, network, chainID, version string) (structures.LatestRecord, error)
	SetLatest(ctx context.Context, kind, network, chainID, version string, latest structures.LatestRecord) error
}

type Storage struct {
	Driver PDriver
}

func (s *Storage) GetLatest(ctx context.Context, kind, network, chainID, version string) (structures.LatestRecord, error) {
	return s.Driver.GetLatest(ctx, kind, network, chainID, version)
}

func (s *Storage) SetLatest(ctx context.Context, kind, network, chainID, version string, latest structures.LatestRecord) error {
	return s.Driver.SetLatest(ctx, kind, network, version, chainID, latest)
}
