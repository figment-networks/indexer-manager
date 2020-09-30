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

// BlocksMap map of blocks to control block map
// with extra summary of number of transactions
type BlocksMap struct {
	sync.Mutex
	Blocks map[uint64]structs.Block
	NumTxs uint64
}

// BlockErrorPair to wrap error response
type BlockErrorPair struct {
	Height uint64
	Block  structs.Block
	Err    error
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

	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		return block, err
	}

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
		return block, fmt.Errorf("[COSMOS-API] Error fetching block: %s ", result.Error.Message)
	}
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)
	numTxs, _ := strconv.ParseUint(result.Result.Block.Header.NumTxs, 10, 64)

	block = structs.Block{
		Hash:                 result.Result.BlockMeta.BlockID.Hash,
		Height:               uHeight,
		Time:                 bTime,
		ChainID:              result.Result.Block.Header.ChainID,
		NumberOfTransactions: numTxs,
	}

	c.Sbc.Add(block)
	return block, nil
}

// GetBlockAsync the async version of get block
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

		err = c.rateLimiter.Wait(ctx)
		if err != nil {
			out <- BlockErrorPair{
				Height: height,
				Err:    err,
			}
			continue
		}

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
				Err:    fmt.Errorf("Error fetching block: %s ", result.Error.Message),
			}
			continue
		}

		bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
		uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)
		numTxs, _ := strconv.ParseUint(result.Result.Block.Header.NumTxs, 10, 64)

		out <- BlockErrorPair{
			Height: uHeight,
			Block: structs.Block{
				Hash:                 result.Result.BlockMeta.BlockID.Hash,
				Height:               uHeight,
				Time:                 bTime,
				ChainID:              result.Result.Block.Header.ChainID,
				NumberOfTransactions: numTxs,
			},
		}
	}
}

// GetBlocksMeta fetches block metadata from given range of blocks
func (c Client) GetBlocksMeta(ctx context.Context, params structs.HeightRange, blocks *BlocksMap, end chan<- error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/blockchain", nil)
	if err != nil {
		end <- err
		return
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.StartHeight > 0 {
		q.Add("minHeight", strconv.FormatUint(params.StartHeight, 10))
	}

	if params.EndHeight > 0 {
		q.Add("maxHeight", strconv.FormatUint(params.EndHeight, 10))
	}
	req.URL.RawQuery = q.Encode()

	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		end <- err
		return
	}

	n := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		end <- err
		return
	}
	rawRequestDuration.WithLabels("/blockchain", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockchainResponse
	if err = decoder.Decode(&result); err != nil {
		end <- err
		return
	}

	if result.Error.Message != "" {
		end <- fmt.Errorf("error fetching block: %s ", result.Error.Message)
		return
	}

	blocks.Lock()
	for _, meta := range result.Result.BlockMetas {

		bTime, _ := time.Parse(time.RFC3339Nano, meta.Header.Time)
		uHeight, _ := strconv.ParseUint(meta.Header.Height, 10, 64)
		numTxs, _ := strconv.ParseUint(meta.Header.NumTxs, 10, 64)

		block := structs.Block{
			Hash:                 meta.BlockID.Hash,
			Height:               uHeight,
			ChainID:              meta.Header.ChainID,
			Time:                 bTime,
			NumberOfTransactions: numTxs,
		}
		blocks.NumTxs += numTxs
		blocks.Blocks[block.Height] = block
	}
	blocks.Unlock()

	end <- nil
	return
}
