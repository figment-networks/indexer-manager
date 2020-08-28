package coda

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/figment-networks/cosmos-indexer/worker/api/coda"
	api "github.com/figment-networks/cosmos-indexer/worker/api/coda"
	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

const page = 100

type IndexerClient struct {
	streams      map[uuid.UUID]*cStructs.StreamAccess
	sLock        sync.Mutex
	logger       *zap.Logger
	codaEndpoint string
}

func NewIndexerClient(ctx context.Context, logger *zap.Logger, codaEndpoint string) *IndexerClient {
	return &IndexerClient{
		streams:      make(map[uuid.UUID]*cStructs.StreamAccess),
		logger:       logger,
		codaEndpoint: codaEndpoint,
	}
}

func (ic *IndexerClient) CloseStream(ctx context.Context, streamID uuid.UUID) error {
	ic.sLock.Lock()
	delete(ic.streams, streamID)
	ic.sLock.Unlock()
	return nil
}

func (ic *IndexerClient) RegisterStream(ctx context.Context, stream *cStructs.StreamAccess) error {
	log.Println("REGISTER STREAM ", stream.StreamID.String())
	ic.sLock.Lock()
	defer ic.sLock.Unlock()
	ic.streams[stream.StreamID] = stream

	codaClient := coda.NewClient(ic.codaEndpoint, nil)

	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 20; i++ {
		go ic.Run(ctx, codaClient, stream)
	}

	return nil
}

func (ic *IndexerClient) Run(ctx context.Context, client *api.Client, stream *cStructs.StreamAccess) {

	for {
		select {
		case <-ctx.Done():
			return
		case <-stream.Finish:
			return
		case taskRequest := <-stream.RequestListener:
			switch taskRequest.Type {
			case "GetTransactions":

				block, err := ic.GetBlock(ctx, client, taskRequest)
				if err != nil {
					stream.Send(cStructs.TaskResponse{
						Id:    taskRequest.Id,
						Error: cStructs.TaskError{Msg: "Error getting block: " + err.Error()},
						Final: true,
					})
					continue
				}

				out := make(chan cStructs.OutResp, 20)
				fin := make(chan bool, 1)

				go sendResp(ctx, taskRequest.Id, out, ic.logger, stream, fin)
				if err := api.MapTransactions(taskRequest.Id, block, out); err != nil {
					stream.Send(cStructs.TaskResponse{
						Id:    taskRequest.Id,
						Error: cStructs.TaskError{Msg: "Error getting transactions: " + err.Error()},
						Final: true,
					})

				}
				<-fin
				close(out)
			default:
				stream.Send(cStructs.TaskResponse{
					Id:    taskRequest.Id,
					Error: cStructs.TaskError{"There is no such handler " + taskRequest.Type},
					Final: true,
				})
			}
		}
	}
}

func sendResp(ctx context.Context, id uuid.UUID, out chan cStructs.OutResp, logger *zap.Logger, stream *cStructs.StreamAccess, fin chan bool) {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	order := uint64(0)

SEND_LOOP:
	for {
		select {
		case <-ctx.Done():
			break SEND_LOOP
		case t, ok := <-out:
			if !ok && t.Type == "" {
				break SEND_LOOP
			}
			b.Reset()
			//logger.Debug("Task to send", zap.Stringer("taskID", t.ID), zap.Uint64("order", n.Order), zap.Uint64("order", t.All))

			err := enc.Encode(t.Payload)
			if err != nil {
				logger.Error("[COSMOS-CLIENT] Error encoding payload data", zap.Error(err))
			}

			tr := cStructs.TaskResponse{
				Id:      id,
				Type:    t.Type,
				Order:   order,
				Payload: make([]byte, b.Len()),
			}

			b.Read(tr.Payload)
			order++
			err = stream.Send(tr)
			if err != nil {
				logger.Error("[COSMOS-CLIENT] Error sending data", zap.Error(err))
			}

		}
	}

	err := stream.Send(cStructs.TaskResponse{
		Id:    id,
		Type:  "END",
		Order: order,
		Final: true,
	})

	if err != nil {
		logger.Error("[COSMOS-CLIENT] Error sending end", zap.Error(err))
	}

	if fin != nil {
		fin <- true
	}
}

func (ic *IndexerClient) GetBlock(ctx context.Context, client *api.Client, tr cStructs.TaskRequest) (*api.Block, error) {
	hr := &structs.HeightHash{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		return nil, err
	}

	log.Printf("Received Block Req: %+v  %+v  ", tr, hr)
	sCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	return client.GetBlock(sCtx, hr.Hash)
}
