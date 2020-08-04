package grpc

import (
	"fmt"
	"io"

	"github.com/figment-networks/cosmos-indexer/worker/client"
	"github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"
)

type IndexerServer struct {
	client client.IndexerClienter
}

func NewIndexerServer(client client.IndexerClienter) *IndexerServer {
	return &IndexerServer{client}
}

func (is *IndexerServer) TaskRPC(taskStream indexer.IndexerService_TaskRPCServer) error {

	receiverClosed := make(chan error)
	defer close(receiverClosed)
	responsesCh := make(chan client.TaskResponse)
	defer close(responsesCh)

	go Recv(taskStream, is.client.In(), responsesCh, receiverClosed)
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
			}); err != nil {
				return fmt.Errorf("Error sending TaskRespons: %w", err)
			}
		}
	}

	return nil
}

func Recv(stream indexer.IndexerService_TaskRPCServer, sendCh chan<- client.TaskRequest, respCh chan client.TaskResponse, finishCh chan error) {

	var in *indexer.TaskRequest
	var err error

	for {
		in, err = stream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			close(finishCh)
			return
		}
		if err != nil {
			finishCh <- err
		}
		sendCh <- client.TaskRequest{
			Id:         in.Id,
			Type:       in.Type,
			Payload:    in.Payload,
			ResponseCh: respCh,
		}
	}
}
