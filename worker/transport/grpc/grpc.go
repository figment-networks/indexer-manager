package grpc

import (
	"io"
	"log"

	"github.com/figment-networks/cosmos-indexer/worker/client"
	"github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"
)

type IndexerServer struct {
	indexer.UnimplementedIndexerServiceServer
	client client.IndexerClienter
}

func NewIndexerServer(client client.IndexerClienter) *IndexerServer {
	return &IndexerServer{client: client}
}

func (is *IndexerServer) TaskRPC(taskStream indexer.IndexerService_TaskRPCServer) error {

	ctx := taskStream.Context()

	receiverClosed := make(chan error)
	defer close(receiverClosed)
	responsesCh := make(chan client.TaskResponse, 5)
	defer close(responsesCh)

	log.Printf("TaskRPC run")
	defer log.Printf("TaskRPC finish ")

	var after bool
	for {
		select {
		case <-ctx.Done():
			log.Println("CONTEXT DONE")
			return ctx.Err()
		default:
		}
		in, err := taskStream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			log.Printf("Stream io.EOF")
			close(receiverClosed)
			return nil
		}
		if err != nil {
			receiverClosed <- err
			return err
		}
		if !after {
			go Send(taskStream, responsesCh, receiverClosed)
			after = true
		}
		is.client.In() <- client.TaskRequest{
			Id:         in.Id,
			Type:       in.Type,
			Payload:    in.Payload,
			ResponseCh: responsesCh,
		}
	}

}

func Send(taskStream indexer.IndexerService_TaskRPCServer, responsesCh chan client.TaskResponse, receiverClosed chan error) {
	defer log.Printf("Send finished ")
	defer log.Printf("Send started ")
CONTROLRPC:
	for {
		select {
		case <-receiverClosed:
			break CONTROLRPC
		case resp := <-responsesCh:
			if err := taskStream.Send(&indexer.TaskResponse{
				Version: resp.Version,
				Id:      resp.Id,
				Type:    resp.Type,
				Payload: resp.Payload,
				Order:   resp.Order,
				Final:   resp.Final,
			}); err != nil {
				log.Printf("Error sending TaskRespons: %s", err.Error())
			}
		}
	}
}
