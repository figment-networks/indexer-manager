package cosmos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
)

type GetBlockParams struct {
	Height uint64
	Hash   string
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock(ctx context.Context, params structs.HeightHash) (b structs.Block, er error) {

	block, ok := c.Sbc.Get(params.Height)
	if ok {
		blockCacheEfficiencyHit.Inc()
		return block, nil
	}
	blockCacheEfficiencyMissed.Inc()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/block", nil)
	if err != nil {
		return b, err
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
		return b, err
	}
	rawRequestDuration.WithLabels("/block", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockResponse
	if err = decoder.Decode(&result); err != nil {
		return b, err
	}

	if result.Error.Message != "" {
		return b, fmt.Errorf("error fetching block: %s ", result.Error.Message)
	}
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)

	return structs.Block{
		Hash:   result.Result.BlockMeta.BlockID.Hash,
		Height: uHeight,
		Time:   bTime,
	}, nil
}
