package cosmos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/figment-networks/cosmos-indexer/worker/model"
	log "github.com/sirupsen/logrus"
)

const (
	txSearchEndpoint = "/tx_search"
	blockEndpoint    = "/block"
)

type GetTxResponse struct {
	ID     string          `json:"id"`
	RPC    string          `json:"jsonrpc"`
	Result *ResultTxSearch `json:"result"`
	Error  *Error          `json:"error"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// // Result of searching for txs
type ResultTxSearch struct {
	Txs        []*ResultTx `json:"txs"`
	TotalCount string      `json:"total_count"`
}

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	httpClient *http.Client
	key        string
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 10,
		}
	}
	return &Client{
		//tendermint rpc url
		baseURL:    url, //todo strip trailing '/'
		key:        key,
		httpClient: c,
	}
}

type GetBlockResponse struct {
	ID     string       `json:"id"`
	RPC    string       `json:"jsonrpc"`
	Result *ResultBlock `json:"result"`
	Error  *Error       `json:"error"`
}

// GetByHeightRange fetches transactions for all block heights within given range
func (c Client) GetByHeightRange(r *model.HeightRange) ([]*ResultTx, error) {
	fmt.Println("[GetByHeightRange] StartHeight ", r.StartHeight)

	req, err := http.NewRequest(http.MethodGet, c.baseURL+txSearchEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.key)

	q := req.URL.Query()
	q.Add("query", fmt.Sprintf("\"tx.height>=%d AND tx.height<=%d\"", r.StartHeight, r.EndHeight))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var result *GetTxResponse

	if err = decoder.Decode(&result); err != nil {
		log.WithError(err).Error("unable to decode result body")
		return nil, err
	}

	if result.Error != nil {
		return nil, errors.New("error fetching transactions") //todo make more descriptive
	}

	return result.Result.Txs, nil
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock() (*Block, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+blockEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.key)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockResponse

	if err = decoder.Decode(&result); err != nil {
		log.WithError(err).Error("unable to decode result body")
		return nil, err
	}

	if result.Error != nil {
		return nil, errors.New("error fetching block") //todo make more descriptive
	}

	return &result.Result.Block, nil
}
