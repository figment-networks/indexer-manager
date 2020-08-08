package tendermint

import (
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"

	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	cdc        *codec.Codec

	inTx chan TxResponse
	out  chan cStruct.OutResp
	//conn *client.WSClient
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}

	/* (lukanus): to use  ws in future "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	conn, err := client.NewWS(addr, "/websocket")
	err = conn.Start()
	*/

	cli := &Client{
		baseURL:    url, //tendermint rpc url
		key:        key,
		httpClient: c,
		cdc:        makeCodec(),
		inTx:       make(chan TxResponse, 20),
		out:        make(chan cStruct.OutResp, 20),
	}

	go rawToTransaction(cli.inTx, cli.out, cli.cdc)
	go rawToTransaction(cli.inTx, cli.out, cli.cdc)
	go rawToTransaction(cli.inTx, cli.out, cli.cdc)
	go rawToTransaction(cli.inTx, cli.out, cli.cdc)

	return cli
}

func (c *Client) Out() chan cStruct.OutResp {
	return c.out
}

func makeCodec() *codec.Codec {
	var cdc = codec.New()
	bank.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)
	distr.RegisterCodec(cdc)
	slashing.RegisterCodec(cdc)
	gov.RegisterCodec(cdc)
	crisis.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	codec.RegisterEvidences(cdc)
	return cdc
}
