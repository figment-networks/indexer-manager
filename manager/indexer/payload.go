package indexer

import (
	"sync"

	"github.com/figment-networks/cosmos-indexer/cosmos"
	"github.com/figment-networks/cosmos-indexer/model"
	"github.com/figment-networks/indexing-engine/pipeline"
)

var (
	payloadPool = sync.Pool{
		New: func() interface{} {
			return new(payload)
		},
	}

	_ pipeline.PayloadFactory = (*payloadFactory)(nil)
	_ pipeline.Payload        = (*payload)(nil)
)

func NewPayloadFactory(config *Config) *payloadFactory {
	return &payloadFactory{
		rangeInterval: config.HeightRangeInterval,
	}
}

type payloadFactory struct {
	rangeInterval int64
}

func (pf *payloadFactory) GetPayload(currentRangeStartHeight int64) pipeline.Payload {
	payload := payloadPool.Get().(*payload)
	payload.CurrentRange = &model.HeightRange{
		StartHeight: currentRangeStartHeight,
		EndHeight:   currentRangeStartHeight + pf.rangeInterval,
	}
	return payload
}

type payload struct {
	CurrentRange *model.HeightRange

	// Fetcher Stage
	RawTransactions []*cosmos.ResultTx

	// Parser Stage
	Transactions []*model.Transaction
}

func (p *payload) MarkAsProcessed() {
	payloadPool.Put(p)
}
