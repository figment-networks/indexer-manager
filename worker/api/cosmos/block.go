package cosmos

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
	log "github.com/sirupsen/logrus"
)

type GetBlockParams struct {
	Height uint64
	Hash   string
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock(ctx context.Context, params structs.HeightHash) (b structs.Block, er error) {
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
		q.Add("height", strconv.FormatInt(params.Height, 10))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return b, err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockResponse
	if err = decoder.Decode(&result); err != nil {
		log.WithError(err).Error("unable to decode result body")
		return b, err
	}

	if result.Error.Message != "" {
		return b, errors.New("error fetching block") //todo make more descriptive
	}
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)

	return structs.Block{
		Hash:   result.Result.BlockMeta.BlockID.Hash,
		Height: uHeight,
		Time:   bTime,
	}, nil
}
