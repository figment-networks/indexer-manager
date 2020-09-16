package lastdata

import (
	"context"
	"fmt"

	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"github.com/figment-networks/cosmos-indexer/structs"
)

const RunnerName = "lastdata"

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
func (c *Client) Name() string {
	return RunnerName
}

func (c *Client) Run(ctx context.Context, network, chainID, version string) error {
	latest, err := c.store.GetLatest(ctx, RunnerName, network, chainID, version)
	if err != nil && err != structures.ErrDoesNotExists {
		return &structures.RunError{Contents: fmt.Errorf("error getting data from store GetLatest [%s]:  %w", RunnerName, err)}
	}

	resp, err := c.transport.GetLastData(ctx, structs.LatestDataRequest{
		Network: network,
		ChainID: chainID,
		Version: version,

		LastHeight: latest.Height,
		LastHash:   latest.Hash,
		LastTime:   latest.Time,
		Nonce:      latest.Nonce,

		SelfCheck: true,
	})
	if err != nil {
		return &structures.RunError{Contents: fmt.Errorf("error getting data from GetLastData [%s]:  %w", RunnerName, err)}
	}

	err = c.store.SetLatest(ctx, RunnerName, network, chainID, version, structures.LatestRecord{
		Hash:   resp.LastHash,
		Height: resp.LastHeight,
		Time:   resp.LastTime,
		Nonce:  resp.Nonce,
	})

	if err != nil {
		return &structures.RunError{Contents: fmt.Errorf("error writing last record SetLatest [%s]:  %w", RunnerName, err)}
	}

	return nil
}
