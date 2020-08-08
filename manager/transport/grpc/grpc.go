package grpc

import (
	"context"
	"io"
	"log"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity"
	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/figment-networks/cosmos-indexer/manager/transport/grpc/indexer"
	"github.com/google/uuid"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Client struct {
	state connectivity.State
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Type() string {
	return "grpc"
}

func (c *Client) Run(ctx context.Context, id string, wc connectivity.WorkerConnection, stream *structs.StreamAccess, removeWorkerCh chan<- string) {
	defer stream.Close()

	log.Printf("Running connections %+v", wc)
	var conn *grpc.ClientConn
	var errSet []error
	var dialErr error
	var connectedTo string
	// (lukanus): try every possibility
	for _, address := range wc.Addresses {
		if address.Address != "" {
			connectedTo = address.Address
			conn, dialErr = grpc.Dial(address.Address, grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
				PermitWithoutStream: false,
			}))
		} else {
			connectedTo = address.IP.String()
			conn, dialErr = grpc.Dial(address.IP.String(), grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
				PermitWithoutStream: false,
			}))
		}

		if dialErr != nil {
			errSet = append(errSet, dialErr)
		}

		if dialErr == nil && conn != nil {
			break
		}
	}

	if conn == nil || len(errSet) > 0 {
		log.Printf("Cannot connect to any address given by worker:   %+v", errSet)
		return
	}
	defer conn.Close()

	gIndexer := indexer.NewIndexerServiceClient(conn)
	taskStream, err := gIndexer.TaskRPC(ctx, grpc.WaitForReady(true))
	if err != nil {
		//log.Errorf(ctx, "Cannot connect to any address given by worker : %+v", wc)
		log.Printf("Cannot connect to any address given by worker : %+v", wc)
	}

	log.Printf("Sucessfully Dialed %s  ", connectedTo)

	receiverClosed := make(chan error)

	go Recv(taskStream, stream, receiverClosed)

CONTROLRPC:
	for {
		select {
		case <-ctx.Done():
			break CONTROLRPC
		case <-receiverClosed:
			break CONTROLRPC
		case req := <-stream.RequestListener:
			log.Printf("SENDING MESSAGE %+v", req)

			if err := taskStream.Send(&indexer.TaskRequest{
				Id:      req.ID.String(),
				Type:    req.Type,
				Payload: req.Payload,
			}); err != nil {
				log.Printf("Error sending TaskResponse: %w", err)
			}
		}
	}

	if taskStream != nil {
		taskStream.CloseSend()
	}
	log.Printf("Finished")
	removeWorkerCh <- id
}

func Recv(stream indexer.IndexerService_TaskRPCClient, internalStream *structs.StreamAccess, finishCh chan error) {

	log.Println("Started receive")
	defer close(finishCh)
	for {
		in, err := stream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			log.Println("Finished receive")
			return
		}
		if err != nil {
			log.Println("Finished receive error: ", err.Error())
			finishCh <- err
			return
		}

		log.Printf("Receiver TaskResponse %+v", in)

		id := uuid.MustParse(in.Id)
		resp := &structs.TaskResponse{
			ID:      id,
			Order:   in.Order,
			Final:   in.Final,
			Type:    in.Type,
			Payload: in.Payload,
		}

		if in.Error != nil {
			resp.Error = structs.TaskError{
				Msg:  in.Error.Msg,
				Type: structs.TaskErrorType(in.Error.Type),
			}
		}
		err = internalStream.Recv(resp)

		if err != nil {
			log.Printf("Error in Recv ", err.Error())
		}
	}

}
