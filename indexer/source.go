package indexer

import (
	"context"
	"errors"

	"github.com/figment-networks/indexing-engine/pipeline"
)

var (
	_ pipeline.Source = (*source)(nil)

	ErrNothingToProcess = errors.New("nothing to process")
)

type source struct {
	startHeight   int64
	batchIndex    int64
	batchSize     int64
	rangeInterval int64
	err           error
}

func NewSource(config *Config) (*source, error) {
	src := &source{
		rangeInterval: config.HeightRangeInterval,
	}

	if err := src.init(config); err != nil {
		return nil, err
	}

	return src, nil
}

func (s *source) Next(ctx context.Context, p pipeline.Payload) bool {
	if s.err == nil && s.batchIndex < (s.batchSize-1) {
		s.batchIndex++
		return true
	}
	return false
}

// Current returns StartHeight of Range for current batch
func (s *source) Current() int64 {
	return s.startHeight + (s.batchIndex * s.rangeInterval)
}

func (s *source) Err() error {
	return s.err
}

func (s *source) init(config *Config) error {
	if err := s.setStart(config); err != nil {
		return err
	}
	if err := s.setEnd(config); err != nil {
		return err
	}
	if err := s.validate(); err != nil {
		return err
	}
	return nil
}

func (s *source) setStart(config *Config) error {
	var startH int64

	if config.StartHeight > 0 {
		startH = config.StartHeight
	} else {
		// todo fetch last tx from db?
	}

	s.startHeight = startH

	return nil
}

func (s *source) setEnd(config *Config) error {
	// todo fetch latest block (`/block` endpoint returns latest block)
	// then calculate batchsize. doesn't make sense to index blocks that don't exist yet

	s.batchSize = config.BatchSize
	return nil
}

func (s *source) validate() error {
	if s.batchSize == 0 {
		return ErrNothingToProcess
	}
	return nil
}
