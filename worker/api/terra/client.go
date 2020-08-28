package terra

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, logger *zap.Logger, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}

	cli := &Client{
		baseURL:    url, //tendermint rpc url
		key:        key,
		logger:     logger,
		httpClient: c,
	}

	return cli
}
