package cosmos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

var (
	getTransactionDuration *metrics.GroupObserver
	getLatestDuration      *metrics.GroupObserver
	getBlockDuration       *metrics.GroupObserver
)

type IndexerClient struct {
	cosmosEndpoint string
	cosmosKey      string

	httpClient *api.Client

	logger  *zap.Logger
	streams map[uuid.UUID]*cStructs.StreamAccess
	sLock   sync.Mutex

	bigPage             uint64
	maximumHeightsToGet uint64
}

func NewIndexerClient(ctx context.Context, logger *zap.Logger, cosmosEndpoint, cosmosKey string, bigPage, maximumHeightsToGet uint64, reqPerSecLimit int) *IndexerClient {
	getTransactionDuration = endpointDuration.WithLabels("getTransactions")
	getLatestDuration = endpointDuration.WithLabels("getLatest")
	getBlockDuration = endpointDuration.WithLabels("getBlock")
	api.InitMetrics()

	return &IndexerClient{
		logger:              logger,
		cosmosEndpoint:      cosmosEndpoint,
		cosmosKey:           cosmosKey,
		httpClient:          api.NewClient(cosmosEndpoint, cosmosKey, logger, nil, reqPerSecLimit),
		bigPage:             bigPage,
		maximumHeightsToGet: maximumHeightsToGet,
		streams:             make(map[uuid.UUID]*cStructs.StreamAccess),
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

	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 20; i++ {
		go ic.Run(ctx, ic.httpClient, stream)
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
			nCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			switch taskRequest.Type {
			case "GetTransactions":
				ic.GetTransactions(nCtx, taskRequest, stream, client)
			case "GetLatest":
				ic.GetLatest(nCtx, taskRequest, stream, client)
			default:
				stream.Send(cStructs.TaskResponse{
					Id:    taskRequest.Id,
					Error: cStructs.TaskError{"There is no such handler " + taskRequest.Type},
					Final: true,
				})
			}
			cancel()
		}
	}
}

func (ic *IndexerClient) GetTransactions(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {

	timer := metrics.NewTimer(getTransactionDuration)
	defer timer.ObserveDuration()

	hr := &structs.HeightRange{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		ic.logger.Debug("[COSMOS-CLIENT] Cannot umnarshal payload", zap.String("contents", string(tr.Payload)))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "cannot unmarshal payload: " + err.Error()},
			Final: true,
		})
		return
	}

	if hr.EndHeight == 0 {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "end height is zero" + err.Error()},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make(chan cStructs.OutResp, page*2+1)
	fin := make(chan bool, 2)

	go sendResp(sCtx, tr.Id, out, ic.logger, stream, fin)

	diff := hr.EndHeight - hr.StartHeight
	bigPages := uint64(math.Ceil(float64(diff) / float64(ic.bigPage)))

	for i := uint64(0); i < bigPages; i++ {
		hrInner := structs.HeightRange{
			StartHeight: hr.StartHeight + i*ic.bigPage,
			EndHeight:   hr.StartHeight + i*ic.bigPage + ic.bigPage,
		}
		if hrInner.EndHeight > hr.EndHeight {
			hrInner.EndHeight = hr.EndHeight
		}

		if err := getRange(sCtx, ic.logger, client, hrInner, out); err != nil {
			stream.Send(cStructs.TaskResponse{
				Id:    tr.Id,
				Error: cStructs.TaskError{Msg: err.Error()},
				Final: true,
			})
			ic.logger.Error("[COSMOS-CLIENT] Error getting range (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
			return
		}
	}

	ic.logger.Debug("[COSMOS-CLIENT] Received all", zap.Stringer("taskID", tr.Id))
	close(out)

	for {
		select {
		case <-sCtx.Done():
			return
		case <-fin:
			ic.logger.Debug("[COSMOS-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id))
			return
		}
	}
}

func (ic *IndexerClient) GetBlock(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client *api.Client) {
	timer := metrics.NewTimer(getBlockDuration)
	defer timer.ObserveDuration()

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
		Type:    "Block",
		Payload: block,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

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
		stream.Send(cStructs.TaskResponse{Id: tr.Id, Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()}, Final: true})
		return
	}

	ic.logger.Debug("[COSMOS-CLIENT] Get last block ", zap.Any("block", block), zap.Any("in", ldr))
	startingHeight := ldr.LastHeight

	// (lukanus): When nothing is scraped we want to get only X number of last requests
	if ldr.LastHeight == 0 {
		lastX := block.Height - ic.maximumHeightsToGet
		if lastX > 0 {
			startingHeight = lastX
		}
	} else {
		diff := block.Height - ldr.LastHeight
		if diff > ic.maximumHeightsToGet {
			if ic.maximumHeightsToGet > block.Height {
				startingHeight = 0
			} else {
				startingHeight = block.Height - ic.maximumHeightsToGet
			}

		}
	}

	diff := block.Height - startingHeight
	bigPages := uint64(math.Ceil(float64(diff) / float64(ic.bigPage)))

	out := make(chan cStructs.OutResp, page)

	fin := make(chan bool, 2)

	go sendResp(sCtx, tr.Id, out, ic.logger, stream, fin)
	for i := uint64(0); i < bigPages; i++ {
		hr := structs.HeightRange{
			StartHeight: startingHeight + i*ic.bigPage,
			EndHeight:   startingHeight + i*ic.bigPage + (ic.bigPage - 1),
		}
		if hr.EndHeight > block.Height {
			hr.EndHeight = block.Height
		}

		if err := getRange(sCtx, ic.logger, client, hr, out); err != nil {
			stream.Send(cStructs.TaskResponse{
				Id:    tr.Id,
				Error: cStructs.TaskError{Msg: err.Error()},
				Final: true,
			})
			ic.logger.Error("[COSMOS-CLIENT] Error GettingRange from get latest ", zap.Error(err), zap.Stringer("taskID", tr.Id))
			break
		}
	}

	ic.logger.Debug("[COSMOS-CLIENT] Received all", zap.Stringer("taskID", tr.Id))
	close(out)

	for {
		select {
		case <-sCtx.Done():
			return
		case <-fin:
			ic.logger.Debug("[COSMOS-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id))
			return
		}
	}
}

func getRange(ctx context.Context, logger *zap.Logger, client *api.Client, hr structs.HeightRange, out chan cStructs.OutResp) error {
	defer logger.Sync()

	var blocksAllCount = hr.EndHeight - hr.StartHeight
	blocksToBeDone := int(math.Ceil(float64(blocksAllCount) / 20.))

	blocksCtrl := make(chan error, 2)
	defer close(blocksCtrl)
	blocksAll := &api.BlocksMap{Blocks: map[uint64]structs.Block{}}

	for i := 0; i < blocksToBeDone; i++ {
		bhr := structs.HeightRange{
			StartHeight: hr.StartHeight + uint64(i*20),
			EndHeight:   hr.StartHeight + uint64(i*20+19),
		}
		if bhr.EndHeight > hr.EndHeight {
			bhr.EndHeight = hr.EndHeight
		}

		logger.Debug("[COSMOS-CLIENT] Getting blocks for ", zap.Uint64("end", bhr.EndHeight), zap.Uint64("start", bhr.StartHeight))
		go client.GetBlocksMeta(ctx, bhr, blocksAll, blocksCtrl)
	}

	var responses int
	var errors = []error{}
	for err := range blocksCtrl {
		responses++
		if err != nil {
			errors = append(errors, err)
		}
		if responses == blocksToBeDone {
			break
		}
	}
	if len(errors) > 0 {
		errString := ""
		for _, err := range errors {
			errString += err.Error() + " , "
		}
		return fmt.Errorf("Errors Getting Blocks: - %s ", errString)
	}

	// send blocks after all the transaction were sent
	for _, block := range blocksAll.Blocks {
		out <- cStructs.OutResp{
			Type:    "Block",
			Payload: block,
		}
	}

	if blocksAll.NumTxs > 0 {
		fin := make(chan string, 2)
		defer close(fin)
		toBeDone := int(math.Ceil(float64(blocksAll.NumTxs) / float64(page)))

		logger.Debug("[COSMOS-CLIENT] Getting initial data ", zap.Uint64("all", blocksAll.NumTxs), zap.Int64("page", page), zap.Int("toBeDone", toBeDone))
		if toBeDone > 0 {
			for i := 0; i < toBeDone; i++ {
				go client.SearchTx(ctx, hr, blocksAll.Blocks, out, i+1, page, fin)
			}
		}

		var responses int
		for c := range fin {
			responses++
			if c != "" {
				logger.Error("[COSMOS-CLIENT] Getting response from SearchTX", zap.String("error", c))
			}
			if responses == toBeDone {
				break
			}
		}

	}

	return nil
}

func sendResp(ctx context.Context, id uuid.UUID, out chan cStructs.OutResp, logger *zap.Logger, stream *cStructs.StreamAccess, fin chan bool) {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	order := uint64(0)

	var contextDone bool

SEND_LOOP:
	for {
		select {
		case <-ctx.Done():
			contextDone = true
			break SEND_LOOP
		case t, ok := <-out:
			if !ok && t.Type == "" {
				break SEND_LOOP
			}
			b.Reset()

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
		if !contextDone {
			fin <- true
		}
		close(fin)
	}

}
