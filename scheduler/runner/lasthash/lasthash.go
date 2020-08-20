package lasthash

import (
	"context"
	"net/http"
	"time"

	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
)

type Client struct {
	store  persistence.Storage
	client *http.Client
}

func NewClient(store persistence.Storage) *Client {

	return &Client{
		store: store,
		client: &http.Client{
			Timeout: time.Second * 40,
		},
	}
}

func (c *Client) Run(ctx context.Context) error {
	/*
		latest, err := c.store.GetLatest()
		if err != nil && err != structures.ErrDoesNotExists {

			log.Println("ERROR getting latest: ")
		}

		http.NewRequestWithContext(ctx, http.MethodPost, "", nil)
	*/
	return nil
}
