package grpc

import (
	"io"
	"log"

	"github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"
	"github.com/google/uuid"

	cStructs "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
)

type IndexerServer struct {
	indexer.UnimplementedIndexerServiceServer
	client cStructs.IndexerClienter
	//streamDelegation chan cStructs.StreamDelegation
}

func NewIndexerServer(client cStructs.IndexerClienter) *IndexerServer {
	return &IndexerServer{
		client: client,
	}
}

func (is *IndexerServer) TaskRPC(taskStream indexer.IndexerService_TaskRPCServer) error {

	ctx := taskStream.Context()
	receiverClosed := make(chan error)
	defer close(receiverClosed)

	log.Println("NEW TASKRPC")
	var after bool
	var stream *cStructs.StreamAccess
	var streamID uuid.UUID

	log.Println("FINISHED TASKRPC")
	for {
		select {
		case <-ctx.Done():
			log.Println("TaskRPC CONTEXT DONE")
			is.client.CloseStream(ctx, streamID)
			return ctx.Err()
		default:
		}

		in, err := taskStream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			log.Printf("Stream io.EOF")
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
				log.Printf("Error sending Ping: %s", err.Error())
				continue
			}

			go Send(taskStream, stream, receiverClosed)
			after = true
		}

		uid, err := uuid.Parse(in.Id)

		if in.Type == "PING" {
			err := stream.Send(cStructs.TaskResponse{
				Id:   uid,
				Type: "PONG",
			})
			if err != nil {
				log.Printf("Error sending Ping: %s", err.Error())
			}
			continue
		}
		log.Println("in", in.Type)

		stream.Req(cStructs.TaskRequest{
			Id:      uid,
			Type:    in.Type,
			Payload: in.Payload,
		})
	}

}

func Send(taskStream indexer.IndexerService_TaskRPCServer, accessCh *cStructs.StreamAccess, receiverClosed chan error) {
	log.Printf("Send started ")
	defer log.Printf("Send finished ")

	ctx := taskStream.Context()

CONTROLRPC:
	for {
		select {
		case <-ctx.Done():
			break CONTROLRPC
		case <-receiverClosed:
			break CONTROLRPC
		case resp := <-accessCh.ResponseListener:
			//accessCh.RLock()
			if err := taskStream.Send(&indexer.TaskResponse{
				Version: resp.Version,
				Id:      resp.Id.String(),
				Type:    resp.Type,
				Payload: resp.Payload,
				Order:   int64(resp.Order),
				Final:   resp.Final,
			}); err != nil {
				if err == io.EOF {
					log.Printf("Stream io.EOF")
					break CONTROLRPC
				}
				log.Printf("Error sending TaskResponse: %s", err.Error())
			}
			//accessCh.RUnlock()
		}
	}
}
