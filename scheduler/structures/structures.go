package structures

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDoesNotExists      = errors.New("does not exists")
	ErrNoWorkersAvailable = errors.New("no workers available")
)

type RunConfig struct {
	ID uuid.UUID `json:"id"`

	RunID uuid.UUID `json:"runID"`

	Network string `json:"network"`
	Version string `json:"version"`

	Duration time.Duration `json:"duration"`
	Kind     string        `json:"kind"`
}

type LatestRecord struct {
	Hash   string    `json:"hash"`
	Height uint64    `json:"height"`
	Time   time.Time `json:"time"`
	From   string    `json:"from"`
	Nonce  []byte    `json:"nonce"`
}
