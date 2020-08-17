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

	"github.com/figment-networks/cosmos-indexer/worker/api/coda"
	api "github.com/figment-networks/cosmos-indexer/worker/api/coda"
	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

const page = 100

type IndexerClient struct {
	streams      map[uuid.UUID]*cStructs.StreamAccess
	sLock        sync.Mutex
	codaEndpoint string
}

func NewIndexerClient(ctx context.Context, codaEndpoint string) *IndexerClient {
	return &IndexerClient{streams: make(map[uuid.UUID]*cStructs.StreamAccess), codaEndpoint: codaEndpoint}
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

	go sendResp(ctx, codaClient, stream)
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

				if err := api.MapTransactions(taskRequest.Id, block, client.Out()); err != nil {
					stream.Send(cStructs.TaskResponse{
						Id:    taskRequest.Id,
						Error: cStructs.TaskError{Msg: "Error getting transactions: " + err.Error()},
						Final: true,
					})

				}
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

type TData struct {
	Order uint64
	All   uint64
}

func sendResp(ctx context.Context, client *api.Client, stream *cStructs.StreamAccess) {

	opened := make(map[[2]uuid.UUID]*TData)

	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)

	resp := client.Out()
SEND_LOOP:
	for {
		select {
		case <-ctx.Done():
			break SEND_LOOP
		case t := <-resp:
			n, ok := opened[[2]uuid.UUID{t.ID, t.RunID}]
			if !ok {
				n = &TData{0, t.All}
				if !t.Additional {
					opened[[2]uuid.UUID{t.ID, t.RunID}] = n
				}
			}
			b.Reset()

			log.Printf("Task to send: %s, %d - %d , new(%d)", t.ID.String(), n.Order, t.All, ok)

			if !t.Additional && n.All == 0 {
				n.All = t.All
			}

			err := enc.Encode(t.Payload)
			if err != nil {
				log.Printf("%s", err.Error())
			}

			var final = (n.Order == n.All-1)

			tr := cStructs.TaskResponse{
				Id:      t.ID,
				Type:    t.Type,
				Payload: make([]byte, b.Len()),
			}
			if !t.Additional {
				tr.Order = n.Order
				tr.Final = final
			}
			b.Read(tr.Payload)

			//	log.Printf("Task out: %d,  %+v %s", n.Order, tr, string(tr.Payload))
			stream.Send(tr)
			if err != nil {
				log.Printf("%s", err.Error())
			}

			if !t.Additional {
				if final {
					delete(opened, [2]uuid.UUID{t.ID, t.RunID})
				}
				n.Order++
			}
		}
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
