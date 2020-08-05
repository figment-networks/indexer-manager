package client

import (
	"context"
	"encoding/json"
	"log"
)

type TaskRequest struct {
	Id      string
	Type    string
	Payload json.RawMessage

	ResponseCh chan TaskResponse
}

type TaskResponse struct {
	Version string
	Id      string
	Type    string
	Order   int64
	Final   bool
	Payload json.RawMessage
}

type IndexerClienter interface {
	In() chan<- TaskRequest
}

type IndexerClient struct {
	taskInput chan TaskRequest
}

func NewIndexerClient(ctx context.Context) *IndexerClient {
	ic := &IndexerClient{make(chan TaskRequest, 100)}
	go ic.Run(ctx) // (lukanus): IndexerClient *MUST* run at least one listener
	return ic
}

func (ic *IndexerClient) Run(ctx context.Context) {
	for {
		select {
		case taskRequest := <-ic.taskInput:
			switch taskRequest.Type {
			case "GetTransactions":
				ic.GetTransactions(ctx, taskRequest)
			}

		}
	}
}

func (ic *IndexerClient) In() chan<- TaskRequest {
	return ic.taskInput
}

type TransactionRequest struct {
	A string `json:"a"`
	B string `json:"b"`
}

func (ic *IndexerClient) GetTransactions(ctx context.Context, tr TaskRequest) {
	log.Println("Received: %+v ", tr)
	trEq := &TransactionRequest{}
	json.Unmarshal(tr.Payload, trEq)

	tr.ResponseCh <- TaskResponse{
		Payload: []byte(`{"ok":"ok"}`),
		Id:      tr.Id,
		Order:   0,
		Final:   false,
	}
	tr.ResponseCh <- TaskResponse{
		Payload: []byte(`{"ok":"ok1"}`),
		Id:      tr.Id,
		Order:   1,
		Final:   false,
	}
	tr.ResponseCh <- TaskResponse{
		Payload: []byte(`{"ok":"ok2"}`),
		Id:      tr.Id,
		Order:   2,
		Final:   true,
	}
}
