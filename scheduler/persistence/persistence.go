package persistence

import "github.com/figment-networks/cosmos-indexer/scheduler/structures"

type PDriver interface {
	GetLatest(kind string) (structures.LatestRecord, error)
	SetLatest(kind string, latest structures.LatestRecord) error
}

type Storage struct {
	Driver PDriver
}

func (s *Storage) GetLatest(kind string) (structures.LatestRecord, error) {
	return s.Driver.GetLatest(kind)
}

func (s *Storage) SetLatest(kind string, latest structures.LatestRecord) error {
	return s.Driver.SetLatest(kind, latest)
}
