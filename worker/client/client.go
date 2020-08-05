package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/figment-networks/cosmos-indexer/worker/api/tendermint"
	"github.com/figment-networks/cosmos-indexer/worker/model"
)

const page = 30

type TaskRequest struct {
	Id      string
	Type    string
	Payload json.RawMessage

	ResponseCh chan TaskResponse
}

type TaskError struct {
	Msg string
}

type TaskResponse struct {
	Version string
	Id      string
	Type    string
	Order   int64
	Final   bool
	Error   TaskError
	Payload json.RawMessage
}

type IndexerClienter interface {
	In() chan<- TaskRequest
}

type IndexerClient struct {
	taskInput chan TaskRequest
	client    *tendermint.Client
}

func NewIndexerClient(ctx context.Context, client *tendermint.Client) *IndexerClient {
	ic := &IndexerClient{taskInput: make(chan TaskRequest, 100), client: client}

	// (lukanus): IndexerClient *MUST* run at least one listener
	for i := 0; i < 10; i++ {
		go ic.Run(ctx)
	}
	return ic
}

func (ic *IndexerClient) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case taskRequest := <-ic.taskInput:
			switch taskRequest.Type {
			case "GetTransactions":
				ic.GetTransactions(ctx, taskRequest)
			default:
				taskRequest.ResponseCh <- TaskResponse{
					Id:    taskRequest.Id,
					Error: TaskError{"There is no such handler " + taskRequest.Type},
				}
			}
		}
	}
}

func (ic *IndexerClient) In() chan<- TaskRequest {
	return ic.taskInput
}

func (ic *IndexerClient) GetTransactions(ctx context.Context, tr TaskRequest) {
	log.Printf("Received: %+v ", tr)

	hr := &structs.HeightRange{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		tr.ResponseCh <- TaskResponse{
			Error: TaskError{"Error unmarshaling message"},
		}
	}

	txs := make(chan tendermint.OutTx, 10)

	go sendTransactions(ctx, tr.Id, txs, tr.ResponseCh)

	count, err := ic.client.SearchTx(model.HeightRange{
		StartHeight: hr.StartHeight,
		EndHeight:   hr.EndHeight,
	}, 1, page, txs)

	if err == nil {
		if count > page {
			toBeDone := int(math.Ceil(float64(count-page) / page))
			for i := 2; i < toBeDone+2; i++ {
				go ic.client.SearchTx(model.HeightRange{
					StartHeight: hr.StartHeight,
					EndHeight:   hr.EndHeight,
				}, i, page, txs)
			}
		}
	}
}

func sendTransactions(ctx context.Context, id string, txs chan tendermint.OutTx, resp chan<- TaskResponse) {
	order := int64(0)
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)

	all := int64(0)
SEND_LOOP:
	for {
		select {
		case <-ctx.Done():
		case t, ok := <-txs:
			if !ok {
				resp <- TaskResponse{
					Id:      id,
					Order:   order,
					Payload: nil,
					Final:   true,
				}
				break SEND_LOOP
			}
			all = t.All

			/*if !ok {
				resp <- TaskResponse{
					Id:      id,
					Order:   order,
					Payload: nil,
					Final:   true,
				}
				break SEND_LOOP
			}*/
			b.Reset()

			enc.Encode(t.Tx)

			resp <- TaskResponse{
				Id:      id,
				Order:   order,
				Payload: b.Bytes(),
				Final:   (order == all-1),
			}

			if order == all {
				defer close(txs)
			}
			order++
		}
	}
}
