package grpc

import (
	"io"

	"github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"
	"github.com/google/uuid"
	"go.uber.org/zap"

	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

type IndexerServer struct {
	indexer.UnimplementedIndexerServiceServer
	client cStructs.IndexerClienter
	logger *zap.Logger
}

func NewIndexerServer(client cStructs.IndexerClienter, logger *zap.Logger) *IndexerServer {
	return &IndexerServer{
		client: client,
		logger: logger,
	}
}

func (is *IndexerServer) TaskRPC(taskStream indexer.IndexerService_TaskRPCServer) error {

	ctx := taskStream.Context()
	receiverClosed := make(chan error)
	defer close(receiverClosed)
	defer is.logger.Sync()

	var after bool
	var stream *cStructs.StreamAccess
	var streamID uuid.UUID

	for {
		select {
		case <-ctx.Done():
			is.logger.Debug("[GRPC] TaskRPC CONTEXT DONE")
			is.client.CloseStream(ctx, streamID)
			return ctx.Err()
		default:
		}

		in, err := taskStream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			is.logger.Debug("[GRPC] Indexer stream io.EOF")
			is.client.CloseStream(ctx, streamID)
			return nil
		}

		if err != nil {
			receiverClosed <- err
			is.client.CloseStream(ctx, streamID)
			return err
		}

		if !after {
			stream = cStructs.NewStreamAccess()
			streamID = stream.StreamID
			defer stream.Close()

			errorStreamReg := is.client.RegisterStream(ctx, stream)
			if errorStreamReg != nil {
				is.logger.Error("[GRPC] Error sending Ping:", zap.Error(err))
				continue
			}

			go Send(taskStream, stream, is.logger, receiverClosed)
			after = true
		}

		requestMetric.WithLabels(in.From, in.Type, "").Inc()

		uid, err := uuid.Parse(in.Id)

		if in.Type == "PING" {
			err := stream.Send(cStructs.TaskResponse{
				Id:   uid,
				Type: "PONG",
			})
			if err != nil {
				is.logger.Error("[GRPC] Error sending Ping:", zap.Error(err))
			}
			continue
		}

		stream.Req(cStructs.TaskRequest{
			Id:      uid,
			Type:    in.Type,
			Payload: in.Payload,
		})
	}

}

func Send(taskStream indexer.IndexerService_TaskRPCServer, accessCh *cStructs.StreamAccess, logger *zap.Logger, receiverClosed chan error) {
	logger.Debug("[GRPC] Send started ")
	defer logger.Debug("[GRPC] Send finished ")
	defer logger.Sync()

	ctx := taskStream.Context()
CONTROLRPC:
	for {
		select {
		case <-ctx.Done():
			break CONTROLRPC
		case <-receiverClosed:
			break CONTROLRPC
		case resp := <-accessCh.ResponseListener:
			tr := &indexer.TaskResponse{
				Version: resp.Version,
				Id:      resp.Id.String(),
				Type:    resp.Type,
				Payload: resp.Payload,
				Order:   int64(resp.Order),
				Final:   resp.Final,
			}

			if resp.Error.Msg != "" {
				tr.Error = &indexer.TaskError{
					Msg: resp.Error.Msg,
				}
			}

			err := taskStream.Send(tr)
			if err == nil {
				if resp.Error.Msg != "" {
					responseMetric.WithLabels(resp.Type, "response_error").Inc()
				} else {
					responseMetric.WithLabels(resp.Type, "").Inc()
				}

				continue CONTROLRPC
			}
			if err == io.EOF {

				logger.Debug("[GRPC] io.EOF")
				break CONTROLRPC
			}
			responseMetric.WithLabels(resp.Type, "send_error").Inc()
			logger.Error("[GRPC] Error sending TaskResponse", zap.Error(err))
		}
	}
}
