package lastdata

import (
	"context"
	"log"

	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"github.com/figment-networks/cosmos-indexer/structs"
)

const RunnerName = "lastData"

type LastDataTransporter interface {
	GetLastData(context.Context, structs.LatestDataRequest) (structs.LatestDataResponse, error)
}

type Client struct {
	store     persistence.Storage
	transport LastDataTransporter
}

func NewClient(store persistence.Storage, transport LastDataTransporter) *Client {
	return &Client{
		store:     store,
		transport: transport,
	}
}

func (c *Client) Run(ctx context.Context, network, version string) error {

	latest, err := c.store.GetLatest(ctx, RunnerName, network, version)
	if err != nil && err != structures.ErrDoesNotExists {
		log.Println("ERROR getting latest: ")
	}

	resp, err := c.transport.GetLastData(ctx, structs.LatestDataRequest{
		LastHeight: latest.Height,
		LastHash:   latest.Hash,
		LastTime:   latest.Time,
		Nonce:      latest.Nonce,
	})
	if err != nil {
		return err
	}

	return c.store.SetLatest(ctx, RunnerName, network, version, structures.LatestRecord{
		Hash:   resp.LastHash,
		Height: resp.LastHeight,
		Time:   resp.LastTime,
		Nonce:  resp.Nonce,
	})
}
