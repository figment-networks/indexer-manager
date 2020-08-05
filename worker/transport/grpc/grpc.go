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
	// exit if context is done
	// or continue
	select {
	case <-ctx.Done():
		log.Println("CONTEXT DONE")
		return ctx.Err()
	default:
	}

	receiverClosed := make(chan error)
	defer close(receiverClosed)
	responsesCh := make(chan client.TaskResponse, 5)
	defer close(responsesCh)

	log.Printf("Listening to new ev")

	var after bool
	for {
		log.Printf("Recev")
		in, err := taskStream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			log.Printf("Stream io.EOF")
			close(receiverClosed)
			return nil
		}
		if err != nil {
			log.Printf("Stream receive error")
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

	/*
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
						Order:   resp.Order,
						Final:   resp.Final,
					}); err != nil {
						log.Printf("Error sending TaskRespons: %s", err.Error())
						return fmt.Errorf("Error sending TaskRespons: %w", err)
					}
				}
			}
			log.Print("Finished TaskRpc")
			return nil
	*/
}

func Send(taskStream indexer.IndexerService_TaskRPCServer, responsesCh chan client.TaskResponse, receiverClosed chan error) {
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

func Recv(stream indexer.IndexerService_TaskRPCServer, sendCh chan<- client.TaskRequest, respCh chan client.TaskResponse, finishCh chan error) {

	for {
		log.Printf("Listening to new ev")
		in, err := stream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			log.Printf("Stream io.EOF")
			close(finishCh)
			return
		}
		if err != nil {
			log.Printf("Stream reveive error")
			finishCh <- err
			return
		}
		sendCh <- client.TaskRequest{
			Id:         in.Id,
			Type:       in.Type,
			Payload:    in.Payload,
			ResponseCh: respCh,
		}
	}

	log.Print("Finished Rcv")
}
