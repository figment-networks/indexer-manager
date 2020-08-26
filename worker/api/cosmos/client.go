package cosmos

import (
	"context"
	"net/http"
	"sync"
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
	"go.uber.org/zap"

	"github.com/figment-networks/cosmos-indexer/structs"
	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	cdc        *codec.Codec
	logger     *zap.Logger

	inTx chan TxResponse
	out  chan cStruct.OutResp

	Sbc *SimpleBlockCache
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, logger *zap.Logger, c *http.Client) *Client {
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
		Sbc:        NewSimpleBlockCache(400),
	}
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		go rawToTransaction(ctx, cli, cli.inTx, cli.out, logger, cli.cdc)
	}

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

type SimpleBlockCache struct {
	space  map[uint64]structs.Block
	blocks chan *structs.Block
	l      sync.RWMutex
}

func NewSimpleBlockCache(cap int) *SimpleBlockCache {
	return &SimpleBlockCache{
		space:  make(map[uint64]structs.Block),
		blocks: make(chan *structs.Block, cap),
	}
}

func (sbc *SimpleBlockCache) Add(bl structs.Block) {
	sbc.l.Lock()
	defer sbc.l.Unlock()

	_, ok := sbc.space[bl.Height]
	if ok {
		return
	}

	sbc.space[bl.Height] = bl
	select {
	case sbc.blocks <- &bl:
	default:
		oldBlock := <-sbc.blocks
		if oldBlock != nil {
			delete(sbc.space, oldBlock.Height)
		}
		sbc.blocks <- &bl
	}

}

func (sbc *SimpleBlockCache) Get(height uint64) (bl structs.Block, ok bool) {
	sbc.l.RLock()
	defer sbc.l.RUnlock()

	bl, ok = sbc.space[bl.Height]
	return bl, ok
}

func InitMetrics() {
	convertionDurationObserver = conversionDuration.WithLabels("conversion")
	transactionConversionDuration = conversionDuration.WithLabels("transaction")
	blockCacheEfficiencyHit = blockCacheEfficiency.WithLabels("hit")
	blockCacheEfficiencyMissed = blockCacheEfficiency.WithLabels("missed")

	numberOfPagesTransactions = numberOfPages.WithLabels("transactions")
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
}
