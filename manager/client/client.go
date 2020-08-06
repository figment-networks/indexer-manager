package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"

	shared "github.com/figment-networks/cosmos-indexer/structs"
)

type NetworkVersion struct {
	Network string
	Version string
}

// HubbleContractor a format agnostic
type HubbleContractor interface {
	GetCurrentHeight(ctx context.Context, nv NetworkVersion)
	GetCurrentBlock(ctx context.Context, nv NetworkVersion)
	GetBlock(ctx context.Context, nv NetworkVersion, id string)
	GetBlocks(ctx context.Context, nv NetworkVersion)
	GetBlockTimes(ctx context.Context, nv NetworkVersion)
	GetBlockTimesInterval(ctx context.Context, nv NetworkVersion)
	GetTransaction(ctx context.Context, nv NetworkVersion) error
	GetTransactions(ctx context.Context, nv NetworkVersion, startHeight int) error
	GetAccounts(ctx context.Context, nv NetworkVersion)
	GetAccount(ctx context.Context, nv NetworkVersion)
}

type IndexerClienter interface {
	Out() <-chan structs.TaskRequest
}

type HubbleClient struct {
	taskOutput chan structs.TaskRequest
}

func NewHubbleClient() *HubbleClient {
	return &HubbleClient{make(chan structs.TaskRequest, 200)}
}

func (hc *HubbleClient) Out() <-chan structs.TaskRequest {
	return hc.taskOutput
}

func (hc *HubbleClient) GetCurrentHeight(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetCurrentBlock(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetBlock(ctx context.Context, nv NetworkVersion, id string) {

}

func (hc *HubbleClient) GetBlocks(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetBlockTimes(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetBlockTimesInterval(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetTransaction(ctx context.Context, nv NetworkVersion, id string) ([]shared.Transaction, error) {
	return hc.GetTransactions(ctx, nv, shared.HeightRange{"", 0, 0})
}

func (hc *HubbleClient) GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange) ([]shared.Transaction, error) {
	respCh := make(chan structs.TaskResponse, 3)
	defer close(respCh)

	b, _ := json.Marshal(heightRange)
	hc.taskOutput <- structs.TaskRequest{
		Network:    nv.Network,
		Version:    nv.Version,
		Type:       "GetTransactions",
		Payload:    b,
		ResponseCh: respCh,
	}

	trs := []shared.Transaction{}
	log.Println("waiting for transactions")

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

WAIT_FOR_ALL_TRANSACTIONS:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Request timed out")
		case response := <-respCh:
			log.Printf("Got Response !!! %s ", string(response.Payload))
			if response.Error.Msg != "" {
				return nil, fmt.Errorf("Error getting response: %s", response.Error.Msg)
			}
			buff.Reset()
			buff.ReadFrom(bytes.NewReader(response.Payload))
			m := &shared.Transaction{}
			dec.Decode(m)
			trs = append(trs, *m)
			if response.Final {
				break WAIT_FOR_ALL_TRANSACTIONS
			}
		}
	}

	log.Println("finished waiting for transactions")
	return trs, nil
}

func (hc *HubbleClient) GetAccounts(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetAccount(ctx context.Context, nv NetworkVersion) {

}
