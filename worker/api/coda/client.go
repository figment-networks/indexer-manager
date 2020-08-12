package coda

import (
	"context"
	"net/http"
	"time"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"

	log "github.com/sirupsen/logrus"

	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

var (
	ErrBlockNotFound = errors.New("block not found")
	ErrBlockInvalid  = errors.New("block is invalid")
)

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	endpoint string
	client   *http.Client
	debug    bool

	out chan cStruct.OutResp
}

// NewClient returns a new client for a given endpoint
func NewClient(url string, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}

	cli := &Client{
		endpoint: url, //tendermint rpc url
		client:   c,
		//	inTx:     make(chan TxResponse, 20),
		out: make(chan cStruct.OutResp, 20),
	}
	//	ctx := context.Background()
	/*
		for i := 0; i < 5; i++ {
			go rawToTransaction(ctx, cli, cli.inTx, cli.out)
		}
	*/
	return cli
}

func (c *Client) Out() chan cStruct.OutResp {
	return c.out
}

func (c *Client) SetDebug(enabled bool) {
	c.debug = enabled
}

// Execute make a GraphQL query and returns the response
func (c Client) Execute(ctx context.Context, q string) (*GraphResponse, error) {
	r := map[string]string{"query": q}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	reqBody := bytes.NewReader(data)

	if c.debug {
		fmt.Printf("%s\n", q)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if c.debug {
		log.Debugf("client response: %s\n", respBody)
	}

	graphResp := GraphResponse{}
	if err := json.Unmarshal(respBody, &graphResp); err != nil {
		switch err.(type) {
		case *json.UnmarshalTypeError, *json.UnmarshalFieldError:
			return nil, errors.New(string(respBody))
		default:
			return nil, err
		}
	}

	if len(graphResp.Errors) > 0 {
		return nil, errors.New(graphResp.Errors[0].Message)
	}

	return &graphResp, nil
}

// Query executes the query and parses the result
func (c Client) Query(ctx context.Context, input string, out interface{}) error {
	resp, err := c.Execute(ctx, input)
	if err != nil {
		return err
	}
	return resp.Decode(out)
}

// GetDaemonStatus returns current node daemon status
func (c Client) GetDaemonStatus(ctx context.Context) (*DaemonStatus, error) {
	var result struct {
		DaemonStatus `json:"daemonStatus"`
	}
	if err := c.Query(ctx, queryDaemonStatus, &result); err != nil {
		return nil, err
	}
	return &result.DaemonStatus, nil
}

// GetCurrentHeight returns the current blockchain height
func (c Client) GetCurrentHeight(ctx context.Context) (int64, error) {
	block, err := c.GetLastBlock(ctx)
	if err != nil {
		return 0, err
	}
	if block == nil {
		return 0, ErrBlockNotFound
	}
	if block.ProtocolState == nil {
		return 0, ErrBlockInvalid
	}
	if block.ProtocolState.ConsensusState == nil {
		return 0, ErrBlockInvalid
	}

	height := block.ProtocolState.ConsensusState.BlockHeight
	return strconv.ParseInt(height, 10, 64)
}

// GetBestChain returns the blocks from the canonical chain
func (c Client) GetBestChain(ctx context.Context) ([]Block, error) {
	var result struct {
		Blocks []Block `json:"bestChain"`
	}
	q := buildBestChainQuery()
	if err := c.Query(ctx, q, &result); err != nil {
		return nil, err
	}
	return result.Blocks, nil
}

// GetBlock returns a single block for the given state hash
func (c Client) GetBlock(ctx context.Context, hash string) (*Block, error) {
	q := fmt.Sprintf(queryBlock, hash, queryBlockFields)
	result := &Block{}

	if err := c.Query(ctx, q, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetBlocks returns blocks for a filter
func (c Client) GetBlocks(ctx context.Context, filter string) ([]Block, error) {
	var result struct {
		Blocks struct {
			Nodes []Block `json:"nodes"`
		} `json:"blocks"`
	}

	q := buildBlocksQuery(filter)
	if err := c.Query(ctx, q, &result); err != nil {
		return nil, err
	}

	return result.Blocks.Nodes, nil
}

// GetSingleBlock returns a single block record from the result
func (c Client) GetSingleBlock(ctx context.Context, filter string) (*Block, error) {
	blocks, err := c.GetBlocks(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, nil
	}
	return &blocks[0], nil
}

// GetFirstBlock returns the first block available in the chain node
func (c Client) GetFirstBlock(ctx context.Context) (*Block, error) {
	return c.GetSingleBlock(ctx, "first:1")
}

// GetFirstBlocks returns the first n blocks
func (c Client) GetFirstBlocks(ctx context.Context, n int) ([]Block, error) {
	filter := fmt.Sprintf("first:%v", n)
	return c.GetBlocks(ctx, filter)
}

// GetLastBlock returns the last block available in the chain node
func (c Client) GetLastBlock(ctx context.Context) (*Block, error) {
	return c.GetSingleBlock(ctx, "last:1")
}

// GetNextBlock returns the next block after the given block's hash
func (c Client) GetNextBlock(ctx context.Context, after string) (*Block, error) {
	if after == "" {
		return c.GetFirstBlock(ctx)
	}

	filter := fmt.Sprintf("after:%q,first:1", after)
	return c.GetSingleBlock(ctx, filter)
}

// GetNextBlocks returns a next N blocks after a given block hash
func (c Client) GetNextBlocks(ctx context.Context, after string, n int) ([]Block, error) {
	return c.GetBlocks(ctx, fmt.Sprintf("after:%q,first:%v", after, n))
}

// GetAccount returns account for a given public key
func (c Client) GetAccount(ctx context.Context, publicKey string) (*Account, error) {
	var result struct {
		Account Account `json:"account"`
	}
	if err := c.Query(ctx, buildAccountQuery(publicKey), &result); err != nil {
		return nil, err
	}
	return &result.Account, nil
}
