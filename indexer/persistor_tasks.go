package indexer

import (
	"context"

	"github.com/figment-networks/indexing-engine/pipeline"
)

const (
	persistorTaskName = "TransactionPersistor"
)

func NewPersistorTask(db TransactionStore) pipeline.Task {
	return &persistorTask{
		db: db,
	}
}

type persistorTask struct {
	db TransactionStore
}

func (t *persistorTask) GetName() string {
	return persistorTaskName
}

func (t *persistorTask) Run(ctx context.Context, p pipeline.Payload) error {
	payload := p.(*payload)

	for _, tx := range payload.Transactions {
		// fmt.Printf("\nsaving tx- height: %d, memo: %s, hash: %v\n", tx.Height, tx.Memo, tx.Hash)
		err := t.db.CreateIfNotExists(tx)
		if err != nil {
			return err
		}
	}
	return nil
}
