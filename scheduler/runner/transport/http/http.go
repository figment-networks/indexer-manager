package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/figment-networks/cosmos-indexer/scheduler/destination"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"github.com/figment-networks/cosmos-indexer/structs"
)

type LastDataHTTPTransport struct {
	client *http.Client
	dest   *destination.Scheme
}

func NewLastDataHTTPTransport(dest *destination.Scheme) *LastDataHTTPTransport {
	return &LastDataHTTPTransport{
		dest: dest,
		client: &http.Client{
			Timeout: time.Second * 40,
		},
	}
}

func (ld LastDataHTTPTransport) GetLastData(ctx context.Context, ldReq structs.LatestDataRequest) (ldr structs.LatestDataResponse, err error) {

	t, ok := ld.dest.Get(destination.NVKey{Network: ldReq.Network, Version: ldReq.Version})
	if !ok {
		return ldr, &structures.RunError{Contents: fmt.Errorf("error getting response:  %w", structures.ErrNoWorkersAvailable)}
	}

	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)

	if err := enc.Encode(ldReq); err != nil {
		return ldr, &structures.RunError{Contents: fmt.Errorf("error encoding request: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Address+"/scrape_latest", b)
	if err != nil {
		return ldr, &structures.RunError{Contents: fmt.Errorf("error creating response: %w", err)}
	}

	resp, err := ld.client.Do(req)
	if err != nil {
		return ldr, &structures.RunError{Contents: fmt.Errorf("error getting response:  %w", err)}
	}

	lhr := &structs.LatestDataResponse{}
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	err = dec.Decode(lhr)
	return *lhr, &structures.RunError{Contents: fmt.Errorf("error decoding response:  %w", err)}
}
