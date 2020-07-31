package indexer

import (
	"context"

	"github.com/figment-networks/indexing-engine/pipeline"
)

const (
	FetcherTaskName = "TransactionFetcher"
)

func NewFetcherTask(c TransactionClient) pipeline.Task {
	return &FetcherTask{
		client: c,
	}
}

type FetcherTask struct {
	client TransactionClient
}

func (t *FetcherTask) GetName() string {
	return FetcherTaskName
}

func (t *FetcherTask) Run(ctx context.Context, p pipeline.Payload) error {
	payload := (p).(*payload)
	txs, err := t.client.GetByHeightRange(payload.CurrentRange)
	if err != nil {
		return err
	}

	payload.RawTransactions = txs
	return nil
}
