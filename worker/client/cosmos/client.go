package cosmos

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/figment-networks/indexing-engine/metrics"

	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/google/uuid"
	"go.uber.org/zap"

	api "github.com/figment-networks/cosmos-indexer/worker/api/cosmos"
	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

const page = 100
const maximumHeightsToGet = 100000

var (
	getTransactionDuration *metrics.GroupObserver
	getLatestDuration      *metrics.GroupObserver
	getBlockDuration       *metrics.GroupObserver
)

type IndexerClient struct {
	cosmosEndpoint string
	cosmosKey      string

	logger  *zap.Logger
	streams map[uuid.UUID]*cStructs.StreamAccess
	sLock   sync.Mutex
}

func NewIndexerClient(ctx context.Context, logger *zap.Logger, cosmosEndpoint string, cosmosKey string) *IndexerClient {
	getTransactionDuration = endpointDuration.WithLabels("getTransactions")
	getLatestDuration = endpointDuration.WithLabels("getLatest")
	getBlockDuration = endpointDuration.WithLabels("getBlock")
	api.InitMetrics()

	return &IndexerClient{
		logger:         logger,
		cosmosEndpoint: cosmosEndpoint,
		cosmosKey:      cosmosKey,
		streams:        make(map[uuid.UUID]*cStructs.StreamAccess),
	}
}

func (ic *IndexerClient) CloseStream(ctx context.Context, streamID uuid.UUID) error {
	ic.sLock.Lock()
	defer ic.sLock.Unlock()

	ic.logger.Debug("[COSMOS-CLIENT] Close Stream", zap.Stringer("streamID", streamID))
	delete(ic.streams, streamID)

	return nil
}

func (ic *IndexerClient) RegisterStream(ctx context.Context, stream *cStructs.StreamAccess) error {
	ic.logger.Debug("[COSMOS-CLIENT] Register Stream", zap.Stringer("streamID", stream.StreamID))
	newStreamsMetric.WithLabels().Inc()

	ic.sLock.Lock()
	defer ic.sLock.Unlock()
	ic.streams[stream.StreamID] = stream

	cosmosClient := api.NewClient(ic.cosmosEndpoint, ic.cosmosKey, ic.logger, nil)
	//go sendResp(ctx, cosmosClient, ic.logger, stream)
	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 50; i++ {
		go ic.Run(ctx, cosmosClient, stream)
	}

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
			receivedRequestsMetric.WithLabels(taskRequest.Type).Inc()
			switch taskRequest.Type {
			case "GetTransactions":
				ic.GetTransactions(ctx, taskRequest, stream, client)
			case "GetBlock":
				ic.GetBlock(ctx, taskRequest, stream, client)
			case "GetLatest":
				ic.GetLatest(ctx, taskRequest, stream, client)
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

	go sendResp(ctx, tr.Id, out, ic.logger, stream, fin2)

	count, err := client.SearchTx(sCtx, *hr, out, 1, page, fin)

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
			go client.SearchTx(sCtx, *hr, out, i, page, fin)
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
				close(out)
				//log.Printf("Taken: %s", time.Now().Sub(now))

			}
		case <-fin2:
			return
		}
	}
}

/*
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

	fin := make(chan string, 2)
	defer close(fin)

	uniqueRID, _ := uuid.NewRandom()
	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()
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
			go client.SearchTx(sCtx, tr.Id, uniqueRID, *hr, i, page, fin)
		}
	}

	var received int
	for {
		select {
		case <-fin:
			//	if e != "" {
			//		cancel()
			//	}
			received++
			if received == toBeDone+1 {
				return
			}
		}
	}
}*/

func (ic *IndexerClient) GetBlock(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {
	timer := metrics.NewTimer(getBlockDuration)
	defer timer.ObserveDuration()

	//	log.Printf("Received Block Req: %+v ", tr)
	hr := &structs.HeightHash{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payment"},
			Final: true,
		})
		return
	}

	uniqueRID, _ := uuid.NewRandom()
	sCtx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	block, err := client.GetBlock(sCtx, *hr)
	if err != nil {
		ic.logger.Error("Error getting block", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		RunID:   uniqueRID,
		Type:    "Block",
		Payload: block,
		All:     1,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

/*
func (ic *IndexerClient) GetBlock(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {
	timer := metrics.NewTimer(getBlockDuration)
	defer timer.ObserveDuration()

	//	log.Printf("Received Block Req: %+v ", tr)
	hr := &structs.HeightHash{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payment"},
			Final: true,
		})
		return
	}

	uniqueRID, _ := uuid.NewRandom()
	sCtx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	block, err := client.GetBlock(sCtx, *hr)
	if err != nil {
		ic.logger.Error("Error getting block", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()},
			Final: true,
		})
		return
	}
	client.Out() <- cStructs.OutResp{
		ID:      tr.Id,
		RunID:   uniqueRID,
		Type:    "Block",
		Payload: block,
		All:     1,
	}
}
*/
/*
func (ic *IndexerClient) GetLatest(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {
	timer := metrics.NewTimer(getLatestDuration)
	defer timer.ObserveDuration()

	ldr := &structs.LatestDataRequest{}
	err := json.Unmarshal(tr.Payload, ldr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{Id: tr.Id, Error: cStructs.TaskError{Msg: "Cannot unmarshal payment"}, Final: true})
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	fin := make(chan string, 2)
	defer close(fin)

	// (lukanus): Get latest block (height = 0)
	block, err := client.GetBlock(sCtx, structs.HeightHash{})
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()},
			Final: true,
		})
		return
	}

	client.Out() <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "Block",
		Payload: block,
	}

	startingHeight := ldr.LastHeight
	missing := false

	// (lukanus): When nothing is scraped we want to get only X number of last requests
	if ldr.LastHeight == 0 {
		lastHundred := block.Height - maximumHeightsToGet
		if lastHundred > 0 {
			startingHeight = lastHundred
		}
	} else {
		diff := block.Height - ldr.LastHeight
		if diff > maximumHeightsToGet {
			startingHeight = diff
			missing = true
		}
	}

	hr := structs.HeightRange{
		StartHeight: startingHeight,
		EndHeight:   block.Height,
	}

	uniqueRID, _ := uuid.NewRandom()
	count, err := client.SearchTx(sCtx, tr.Id, uniqueRID, hr, 1, page, fin)

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
			go client.SearchTx(sCtx, tr.Id, uniqueRID, hr, i, page, fin)
		}
	}

	log.Println("print missed", missing)
	var received int
	for {
		select {
		case e := <-fin:
			if e != "" {
				cancel()
			}
			received++
			if received == toBeDone+1 {
				//log.Printf("Taken: %s", time.Now().Sub(now))
				return
			}
		}
	}
}*/

func (ic *IndexerClient) GetLatest(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {
	timer := metrics.NewTimer(getLatestDuration)
	defer timer.ObserveDuration()

	ldr := &structs.LatestDataRequest{}
	err := json.Unmarshal(tr.Payload, ldr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{Id: tr.Id, Error: cStructs.TaskError{Msg: "Cannot unmarshal payment"}, Final: true})
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// (lukanus): Get latest block (height = 0)
	block, err := client.GetBlock(sCtx, structs.HeightHash{})
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()},
			Final: true,
		})
		return
	}

	/*
		out <- cStructs.OutResp{
			ID:      tr.Id,
			Type:    "Block",
			Payload: block,
		}*/

	startingHeight := ldr.LastHeight
	// missing := false

	// (lukanus): When nothing is scraped we want to get only X number of last requests
	if ldr.LastHeight == 0 {
		lastHundred := block.Height - maximumHeightsToGet
		if lastHundred > 0 {
			startingHeight = lastHundred
		}
	} else {
		diff := block.Height - ldr.LastHeight
		if diff > maximumHeightsToGet {
			startingHeight = diff
			//	missing = true
		}
	}

	hr := structs.HeightRange{
		StartHeight: startingHeight,
		EndHeight:   block.Height,
	}

	out := make(chan cStructs.OutResp, page*2+1)

	// TODO(lukanus): use Pools
	fin := make(chan string, 2)
	defer close(fin)
	fin2 := make(chan bool, 2)
	defer close(fin2)

	go sendResp(ctx, tr.Id, out, ic.logger, stream, fin2)

	count, err := client.SearchTx(sCtx, hr, out, 1, page, fin)

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
			go client.SearchTx(sCtx, hr, out, i, page, fin)
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
				close(out)
				//log.Printf("Taken: %s", time.Now().Sub(now))

			}
		case <-fin2:
			return
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

/*
type TData struct {
	Order uint64
	All   uint64
}

func sendResp(ctx context.Context, client *api.Client, logger *zap.Logger, stream *cStructs.StreamAccess) {

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

			//logger.Debug("Task to send", zap.Stringer("taskID", t.ID), zap.Uint64("order", n.Order), zap.Uint64("order", t.All))

			if !t.Additional && n.All == 0 {
				n.All = t.All
			}

			err := enc.Encode(t.Payload)
			if err != nil {
				logger.Error("[COSMOS-CLIENT] Error encoding payload data", zap.Error(err))
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

			err = stream.Send(tr)
			if err != nil {
				logger.Error("[COSMOS-CLIENT] Error sending data", zap.Error(err))
			}

			if !t.Additional {
				if final {
					delete(opened, [2]uuid.UUID{t.ID, t.RunID})
				}
				n.Order++
				sendResponseMetric.WithLabels(t.Type, "yes").Inc()
			} else {
				sendResponseMetric.WithLabels(t.Type, "no").Inc()
			}
		}
	}
}
*/
