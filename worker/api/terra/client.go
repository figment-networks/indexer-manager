package terra

import (
	"context"
	"net/http"
	"time"

	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	//	cdc        *codec.Codec

	inTx chan TxResponse
	out  chan cStruct.OutResp
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}

	cli := &Client{
		baseURL:    url, //tendermint rpc url
		key:        key,
		httpClient: c,
		//cdc:        makeCodec(),
		inTx: make(chan TxResponse, 20),
		out:  make(chan cStruct.OutResp, 20),
	}
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		go rawToTransaction(ctx, cli, cli.inTx, cli.out)
	}

	return cli
}

func (c *Client) Out() chan cStruct.OutResp {
	return c.out
}

/*
func makeCodec() *codec.Codec {
	var cdc = codec.New()

	auth.RegisterCodec(cdc)
	//oracle.RegisterCodec(cdc)
	//market.RegisterCodec(cdc)
	/*	bank.RegisterCodec(cdc)
		staking.RegisterCodec(cdc)
		distr.RegisterCodec(cdc)
		slashing.RegisterCodec(cdc)
		gov.RegisterCodec(cdc)
		crisis.RegisterCodec(cdc)
		auth.RegisterCodec(cdc)
		sdk.RegisterCodec(cdc)
		codec.RegisterCrypto(cdc)
		codec.RegisterEvidences(cdc)8
	return cdc
}
*/
