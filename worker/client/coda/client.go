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

	api "github.com/figment-networks/cosmos-indexer/worker/api/coda"
	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

const page = 100

type IndexerClient struct {
	client *api.Client

	streams map[uuid.UUID]*cStructs.StreamAccess
	sLock   sync.Mutex
}

func NewIndexerClient(ctx context.Context, client *api.Client) *IndexerClient {
	return &IndexerClient{client: client, streams: make(map[uuid.UUID]*cStructs.StreamAccess)}
}

func (ic *IndexerClient) RegisterStream(ctx context.Context, stream *cStructs.StreamAccess) error {
	log.Println("REGISTER STREAM")
	ic.sLock.Lock()
	ic.streams[stream.StreamID] = stream

	go sendResp(ctx, ic.client.Out(), stream)
	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 20; i++ {
		go ic.Run(ctx, stream)
	}

	ic.sLock.Unlock()
	return nil
}

func (ic *IndexerClient) Run(ctx context.Context, stream *cStructs.StreamAccess) {

	for {
		select {
		case <-ctx.Done():
			ic.sLock.Lock()
			delete(ic.streams, stream.StreamID)
			ic.sLock.Unlock()
			return
		case <-stream.Finish:
			return
		case taskRequest := <-stream.RequestListener:
			switch taskRequest.Type {
			case "GetTransactions":
				block, err := ic.GetBlock(ctx, taskRequest)
				if err != nil {
					stream.Send(cStructs.TaskResponse{
						Id:    taskRequest.Id,
						Error: cStructs.TaskError{Msg: "Error getting block: " + err.Error()},
						Final: true,
					})
					continue
				}

				if err := api.MapTransactions(taskRequest.Id, block, ic.client.Out()); err != nil {
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

func sendResp(ctx context.Context, resp chan cStructs.OutResp, stream *cStructs.StreamAccess) {

	opened := make(map[[2]uuid.UUID]*TData)

	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)

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

func (ic *IndexerClient) GetBlock(ctx context.Context, tr cStructs.TaskRequest) (*api.Block, error) {

	hr := &structs.HeightHash{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		return nil, err
	}

	log.Printf("Received Block Req: %+v  %+v  ", tr, hr)
	sCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	return ic.client.GetBlock(sCtx, hr.Hash)
}
