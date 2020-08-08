package tendermint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type GetBlockParams struct {
	Height uint64
	Hash   string
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock(ctx context.Context, taskID, runUUID uuid.UUID, params structs.HeightHash) (*Block, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/block", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.Height > 0 {
		q.Add("height", strconv.FormatInt(params.Height, 10))
	}

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

	if result.Error.Message != "" {
		return nil, errors.New("error fetching block") //todo make more descriptive
	}

	return &result.Result.Block, nil
}
