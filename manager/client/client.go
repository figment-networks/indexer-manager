package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"

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

type TaskSender interface {
	Send([]structs.TaskRequest) (*structs.Await, error)
}

type HubbleClient struct {
	sender TaskSender
}

func NewHubbleClient() *HubbleClient {
	return &HubbleClient{}
}

func (hc *HubbleClient) LinkSender(sender TaskSender) {
	hc.sender = sender
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

	log.Println("waiting for transactions")

	req := []structs.TaskRequest{}

	times := 1
	diff := float64(heightRange.EndHeight - heightRange.StartHeight)
	if diff > 0 {
		times = int(math.Ceil(diff / 1000))
	}

	for i := 0; i < times; i++ {
		b, _ := json.Marshal(shared.HeightRange{
			StartHeight: heightRange.StartHeight + int64(i*100),
			EndHeight:   heightRange.StartHeight + int64((i+1)*100),
		})

		req = append(req, structs.TaskRequest{
			Network: nv.Network,
			Version: nv.Version,
			Type:    "GetTransactions",
			Payload: b,
		})
	}

	respAwait, err := hc.sender.Send(req)
	if err != nil {
		log.Println("Error Sending data")
		return nil, err
	}

	defer respAwait.Close()

	trs := []shared.Transaction{}

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	var receivedTransactions int
WAIT_FOR_ALL_TRANSACTIONS:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Request timed out")
		case response := <-respAwait.Resp:
			//	if !ok {
			//		return nil, errors.New("Response closed.")
			//	}
			log.Printf("Got Response !!! %s ", string(response.Payload))
			if response.Error.Msg != "" {
				return nil, fmt.Errorf("Error getting response: %s", response.Error.Msg)
			}
			buff.Reset()
			buff.ReadFrom(bytes.NewReader(response.Payload))
			m := &shared.Transaction{}
			err := dec.Decode(m)
			if err != nil {
				return nil, fmt.Errorf("Error getting response: %w", err)
			}
			trs = append(trs, *m)

			if response.Final {
				receivedTransactions++
			}

			if receivedTransactions == times {
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
