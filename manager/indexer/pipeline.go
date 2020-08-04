package indexer

import (
	"context"

	"github.com/figment-networks/cosmos-indexer/manager/model"
	"github.com/figment-networks/cosmos-indexer/worker/cosmos"
	"github.com/figment-networks/indexing-engine/pipeline"
)

type CosmosClient interface {
	// GetByHeightRange fetcehs raw cosmos transactions for all block heights within range
	GetByHeightRange(heightRange *model.HeightRange) ([]*cosmos.ResultTx, error) //should be range

	// GetBlock fetches latest cosmos block
	GetBlock() (*cosmos.Block, error)
}

type TransactionStore interface {
	// CreateIfNotExists creates new transactions in database
	CreateIfNotExists(t *model.Transaction) error
}

// Config provides the starting config for the pipeline
type Config struct {
	// StartHeight is the starting blockheight for the pipeline
	StartHeight int64

	// BatchSize determines the number of runs
	BatchSize int64

	// HeightRangeInterval determines the size of the HeightRange for each run
	HeightRangeInterval int64
}

type Pipeline struct {
	client CosmosClient
	store  TransactionStore
}

func NewPipeline(c CosmosClient, s TransactionStore) *Pipeline {
	return &Pipeline{
		client: c,
		store:  s,
	}
}

func (txp *Pipeline) Start(config *Config) error {
	//todo validate config
	p := pipeline.NewCustom(NewPayloadFactory(config))

	p.AddStage(
		pipeline.NewStage(pipeline.StageFetcher, NewFetcherTask(txp.client)),
	)

	p.AddStage(
		pipeline.NewStage(pipeline.StageParser, NewParserTask()),
	)

	p.AddStage(
		pipeline.NewStage(pipeline.StagePersistor, NewPersistorTask(txp.store)),
	)

	ctx := context.Background()

	src, err := NewSource(config, txp.client)
	if err != nil {
		return err
	}

	options := &pipeline.Options{}
	if err := p.Start(ctx, src, NewSink(), options); err != nil {
		return err
	}
	return nil
}
