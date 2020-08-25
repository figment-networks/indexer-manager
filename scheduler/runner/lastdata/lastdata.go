package lastdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/figment-networks/cosmos-indexer/scheduler/destination"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"github.com/figment-networks/cosmos-indexer/structs"
)

const RunnerName = "lastData"

type Client struct {
	store  persistence.Storage
	client *http.Client

	dest *destination.Scheme
}

func NewClient(store persistence.Storage, dest *destination.Scheme) *Client {

	return &Client{
		store: store,
		dest:  dest,
		client: &http.Client{
			Timeout: time.Second * 40,
		},
	}
}

func (c *Client) Run(ctx context.Context, network, version string) error {

	latest, err := c.store.GetLatest(ctx, RunnerName, network, version)

	if err != nil && err != structures.ErrDoesNotExists {
		log.Println("ERROR getting latest: ")
	}

	t, ok := c.dest.Get(destination.NVKey{network, version})
	if !ok {
		return structures.ErrNoWorkersAvailable
	}

	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)

	if err := enc.Encode(structs.LatestDataRequest{
		LastHeight: latest.Height,
		LastHash:   latest.Hash,
		LastTime:   latest.Time,
		Nonce:      latest.Nonce,
	}); err != nil {
		return fmt.Errorf("error encoding ")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Address+"/scrape_latest", b)
	if err != nil {
		return fmt.Errorf("error creating response: (%s) %w", RunnerName, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting response: (%s) %w", RunnerName, err)
	}

	lhr := &structs.LatestDataResponse{}
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	return dec.Decode(lhr)
}
