package terra

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/figment-networks/indexing-engine/metrics"

	"github.com/figment-networks/cosmos-indexer/structs"
	api "github.com/figment-networks/cosmos-indexer/worker/api/terra"
	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

const page = 100

var (
	getTransactionDuration *metrics.GroupObserver
	getLatestDuration      *metrics.GroupObserver
	getBlockDuration       *metrics.GroupObserver
)

type IndexerClient struct {
	terraAddress string

	logger  *zap.Logger
	streams map[uuid.UUID]*cStructs.StreamAccess
	sLock   sync.Mutex
}

func NewIndexerClient(ctx context.Context, logger *zap.Logger, terraAddress string) *IndexerClient {
	getTransactionDuration = endpointDuration.WithLabels("getTransactions")
	getLatestDuration = endpointDuration.WithLabels("getLatest")
	getBlockDuration = endpointDuration.WithLabels("getBlock")

	return &IndexerClient{
		terraAddress: terraAddress,
		logger:       logger,
		streams:      make(map[uuid.UUID]*cStructs.StreamAccess),
	}
}

func (ic *IndexerClient) CloseStream(ctx context.Context, streamID uuid.UUID) error {
	ic.sLock.Lock()
	delete(ic.streams, streamID)
	ic.sLock.Unlock()
	return nil
}

func (ic *IndexerClient) RegisterStream(ctx context.Context, stream *cStructs.StreamAccess) error {
	log.Println("REGISTER STREAM")
	ic.sLock.Lock()
	ic.streams[stream.StreamID] = stream
	client := api.NewClient(ic.terraAddress, "", ic.logger, nil)

	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 20; i++ {
		go ic.Run(ctx, client, stream)
	}

	ic.sLock.Unlock()
	return nil
}

func (ic *IndexerClient) Run(ctx context.Context, client *api.Client, stream *cStructs.StreamAccess) {

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
				ic.GetTransactions(ctx, taskRequest, stream, client)
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

func (ic *IndexerClient) GetTransactions(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {

	timer := metrics.NewTimer(getTransactionDuration)
	defer timer.ObserveDuration()

	hr := &structs.HeightRange{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "cannot unmarshal payload: " + err.Error()},
			Final: true,
		})

		ic.logger.Debug("[COSMOS-CLIENT] Register Stream", zap.Stringer("streamID", stream.StreamID))
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make(chan cStructs.OutResp, page*2+1)

	// TODO(lukanus): use Pools
	fin := make(chan string, 2)
	defer close(fin)
	fin2 := make(chan bool, 2)
	defer close(fin2)

	wg := &sync.WaitGroup{}
	go sendResp(ctx, tr.Id, out, ic.logger, stream, fin2)

	count, err := client.SearchTx(sCtx, wg, *hr, out, 1, page, fin)
	wg.Add(1)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error Getting Transactions"},
			Final: true,
		})
		return
	}

	toBeDone := int(math.Ceil(float64(count-page) / page))
	if count > page {
		wg.Add(toBeDone)
		for i := 2; i < toBeDone+2; i++ {
			go client.SearchTx(sCtx, wg, *hr, out, i, page, fin)
		}
	}

	wg.Wait()
	ic.logger.Debug("[COSMOS-CLIENT] Received all", zap.Stringer("taskID", tr.Id))
	close(out)
	for {
		select {
		case <-fin2:
			return
		}
	}
}

/*
func (ic *IndexerClient) GetTransactions(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {
	log.Printf("Received: %+v ", tr)
	now := time.Now()
	hr := &structs.HeightRange{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payment"},
			Final: true,
		})
	}

	fin := make(chan string, 2)
	defer close(fin)

	uniqueRID, _ := uuid.NewRandom()
	sCtx, cancel := context.WithCancel(ctx)
	count, err := client.SearchTx(sCtx, tr.Id, uniqueRID, *hr, 1, page, fin)

	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error Getting Transactions"},
			Final: true,
		})
		return
	}

	toBeDone := int(math.Ceil(float64(count-page) / page))
	if count > page {
		for i := 2; i < toBeDone+2; i++ {
			go client.SearchTx(sCtx, tr.Id, uniqueRID, structs.HeightRange{
				StartHeight: hr.StartHeight,
				EndHeight:   hr.EndHeight,
			}, i, page, fin)
		}
	}

	var received int
	for {
		select {
		case e := <-fin:
			if e != "" {
				cancel()
			}
			received++
			if received == toBeDone+1 {
				log.Printf("Taken: %s", time.Now().Sub(now))
				return
			}
		}
	}
}
*/
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

			sendResponseMetric.WithLabels(t.Type, "yes").Inc()
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
