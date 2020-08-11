package tendermint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return b, err
	}

	defer resp.Body.Close()

	by, _ := ioutil.ReadAll(resp.Body)
	decoder := json.NewDecoder(bytes.NewReader(by))

	log.Printf("Payload %+v", string(by))
	var result *GetBlockResponse

	if err = decoder.Decode(&result); err != nil {
		log.WithError(err).Error("unable to decode result body")
		return b, err
	}

	if result.Error.Message != "" {
		return b, errors.New("error fetching block") //todo make more descriptive
	}
	//2020-08-10T11:21:40.090546295Z
	//2017-12-30T05:53:09.287+01:00
	//	bTime, err := time.Parse("2006-01-02T15:04:05.999+07:00", result.Result.Block.Header.Time)
	log.Printf("result.Result.Block %+v %+v", params, result.Result.Block)
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)

	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)

	return structs.Block{
		Hash:   result.Result.BlockMeta.BlockID.Hash,
		Height: uHeight,
		Time:   bTime,
	}, nil
}
