package client

import (
	"context"
	"encoding/json"
	"log"
)

type TaskRequest struct {
	Type    string
	Payload json.RawMessage

	ResponseCh chan TaskResponse
}

type TaskResponse struct {
	Version string
	Type    string
	Order   int64
	Final   bool
	Payload json.RawMessage
}

// HubbleContractor a format agnostic
type HubbleContractor interface {
	GetCurrentHeight(ctx context.Context)
	GetCurrentBlock(ctx context.Context)
	GetBlock(ctx context.Context, id string)
	GetBlocks(ctx context.Context)
	GetBlockTimes(ctx context.Context)
	GetBlockTimesInterval(ctx context.Context)
	GetTransaction(ctx context.Context)
	GetTransactions(ctx context.Context)
	GetAccounts(ctx context.Context)
	GetAccount(ctx context.Context)
}

type IndexerClienter interface {
	Out() <-chan TaskRequest
}

type HubbleClient struct {
	taskOutput chan TaskRequest
}

func NewHubbleClient() *HubbleClient {
	return &HubbleClient{make(chan TaskRequest, 20)}
}

func (hc *HubbleClient) Out() <-chan TaskRequest {
	return hc.taskOutput
}

func (hc *HubbleClient) GetCurrentHeight(ctx context.Context) {

}

func (hc *HubbleClient) GetCurrentBlock(ctx context.Context) {

}

func (hc *HubbleClient) GetBlock(ctx context.Context, id string) {

}

func (hc *HubbleClient) GetBlocks(ctx context.Context) {

}

func (hc *HubbleClient) GetBlockTimes(ctx context.Context) {

}

func (hc *HubbleClient) GetBlockTimesInterval(ctx context.Context) {

}

func (hc *HubbleClient) GetTransaction(ctx context.Context) {

}

type TransactionRequest struct {
	A string `json:"a"`
	B string `json:"b"`
}

func (hc *HubbleClient) GetTransactions(ctx context.Context) {
	respCh := make(chan TaskResponse, 3)

	b, _ := json.Marshal(TransactionRequest{"asdf", "xcvb"})
	hc.taskOutput <- TaskRequest{
		Type:       "GetTransactions",
		Payload:    b,
		ResponseCh: respCh,
	}
	log.Println("waiting for transactions")
WAIT_FOR_ALL_TRANSACTIONS:
	for {
		select {
		case response := <-respCh:
			log.Println("Got Response !!!")
			if response.Final {
				break WAIT_FOR_ALL_TRANSACTIONS
			}
		}
	}
}

func (hc *HubbleClient) GetAccounts(ctx context.Context) {

}

func (hc *HubbleClient) GetAccount(ctx context.Context) {

}
