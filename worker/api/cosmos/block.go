package cosmos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
)

type GetBlockParams struct {
	Height uint64
	Hash   string
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock(ctx context.Context, params structs.HeightHash) (block structs.Block, er error) {

	var ok bool
	if params.Height != 0 {
		block, ok = c.Sbc.Get(params.Height)
		if ok {
			blockCacheEfficiencyHit.Inc()
			return block, nil
		}
		blockCacheEfficiencyMissed.Inc()
	}

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/block", nil)
	if err != nil {
		return block, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.Height > 0 {
		q.Add("height", strconv.FormatUint(params.Height, 10))
	}
	req.URL.RawQuery = q.Encode()

	n := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return block, err
	}
	rawRequestDuration.WithLabels("/block", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockResponse
	if err = decoder.Decode(&result); err != nil {
		return block, err
	}

	if result.Error.Message != "" {
		return block, fmt.Errorf("error fetching block: %s ", result.Error.Message)
	}
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)

	block = structs.Block{
		Hash:   result.Result.BlockMeta.BlockID.Hash,
		Height: uHeight,
		Time:   bTime,
	}

	c.Sbc.Add(block)
	return block, nil
}

type BlockErrorPair struct {
	Height uint64
	Block  structs.Block
	Err    error
}

func (c Client) GetBlockAsync(ctx context.Context, in chan uint64, out chan<- BlockErrorPair) {

	for height := range in {
		req, err := http.NewRequest(http.MethodGet, c.baseURL+"/block", nil)
		if err != nil {
			out <- BlockErrorPair{
				Height: height,
				Err:    err,
			}
			continue
		}

		req.Header.Add("Content-Type", "application/json")
		if c.key != "" {
			req.Header.Add("Authorization", c.key)
		}

		q := req.URL.Query()
		q.Add("height", strconv.FormatUint(height, 10))
		req.URL.RawQuery = q.Encode()

		n := time.Now()
		resp, err := c.httpClient.Do(req)
		if err != nil {
			out <- BlockErrorPair{
				Height: height,
				Err:    err,
			}
			continue
		}
		rawRequestDuration.WithLabels("/block", resp.Status).Observe(time.Since(n).Seconds())

		decoder := json.NewDecoder(resp.Body)

		var result *GetBlockResponse
		err = decoder.Decode(&result)

		resp.Body.Close()
		if err != nil {
			out <- BlockErrorPair{
				Height: height,
				Err:    err,
			}
			continue
		}

		if result.Error.Message != "" {
			log.Printf("err %+v", result)
			out <- BlockErrorPair{
				Height: height,
				Err:    fmt.Errorf("error fetching block: %s ", result.Error.Message),
			}
			continue
		}

		bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
		uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)

		out <- BlockErrorPair{
			Height: uHeight,
			Block: structs.Block{
				Hash:   result.Result.BlockMeta.BlockID.Hash,
				Height: uHeight,
				Time:   bTime,
			},
		}
	}
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
