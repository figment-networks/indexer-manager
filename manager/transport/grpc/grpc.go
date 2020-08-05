package grpc

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/figment-networks/cosmos-indexer/manager/client"
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

func NewClient(clienter client.IndexerClienter) *Client {
	return &Client{}
}

func (c *Client) Type() string {
	return "grpc"
}

func (c *Client) Run(ctx context.Context, id string, wc connectivity.WorkerConnection, input <-chan structs.TaskRequest, removeWorkerCh chan<- string) {

	log.Printf("Running connections %+v", wc)
	var conn *grpc.ClientConn
	var err error
	var connectedTo string
	// (lukanus): try every possibility
	for _, address := range wc.Addresses {
		if address.Address != "" {
			connectedTo = address.Address
			conn, err = grpc.Dial(address.Address, grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
				PermitWithoutStream: false,
			}))
		} else {
			connectedTo = address.IP.String()
			conn, err = grpc.Dial(address.IP.String(), grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
				PermitWithoutStream: false}))
		}

		if err == nil && conn != nil {
			break
		}
		log.Printf("Error on %s, %+v", connectedTo, err)
	}

	if conn == nil || err != nil {
		//log.Errorf(ctx, "Cannot connect to any address given by worker : %+v", wc)
		log.Printf("Cannot connect to any address given by worker : %+v", wc)
	}
	defer conn.Close()

	gIndexer := indexer.NewIndexerServiceClient(conn)
	taskStream, err := gIndexer.TaskRPC(ctx)
	if err != nil {
		//log.Errorf(ctx, "Cannot connect to any address given by worker : %+v", wc)
		log.Printf("Cannot connect to any address given by worker : %+v", wc)
	}

	//log.Debugf(ctx, "Sucessfully Dialed %s  ", connectedTo)
	log.Printf("Sucessfully Dialed %s  ", connectedTo)

	receiverClosed := make(chan error)
	defer close(receiverClosed)
	responsesCh := make(chan structs.TaskResponse)
	defer close(responsesCh)

	lookupTable := NewLookupTable()

	go Recv(taskStream, lookupTable, receiverClosed)
CONTROLRPC:
	for {
		select {
		case <-ctx.Done():
			break CONTROLRPC
		case <-receiverClosed:
			break CONTROLRPC
		case req := <-input:
			log.Printf("SENDING MESSAGE %+v", req)
			messageID, _ := uuid.NewRandom() // UUID V4
			id := messageID.String()
			lookupTable.Set(id, &req)

			if err := taskStream.Send(&indexer.TaskRequest{
				//	Version: req.Version,
				Id:      id,
				Type:    req.Type,
				Payload: req.Payload,
			}); err != nil {
				//log.Errorf(ctx, "Error sending TaskRespons: %w", err)
				log.Printf("Error sending TaskRespons: %w", err)
			}
		}
	}

	// (lukanus): has to be closed, otherwise Recv would be running forever
	taskStream.CloseSend()
	log.Printf("Finished")
	removeWorkerCh <- id
}

func Recv(stream indexer.IndexerService_TaskRPCClient, lookupTable *LookupTable, finishCh chan error) {

	log.Println("Started receive")
	for {
		in, err := stream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			close(finishCh)
			log.Println("Finished receive")
			return
		}
		if err != nil {
			log.Println("Finished receive error")
			finishCh <- err
			return
		}

		if in.Final { // (lukanus): On final remove handler
			req, ok := lookupTable.Take(in.Id)
			if !ok {
				//log.Errorf(ctx, "Received message with  unknown id (%s): %+v", string(in.Id), in)
				log.Printf("Received message with  unknown id (%s): %+v", string(in.Id), in)
				continue
			}

			log.Printf("Sending TaskResponse %d %d", len(req.ResponseCh), cap(req.ResponseCh))
			req.ResponseCh <- structs.TaskResponse{
				Order:   in.Order,
				Final:   in.Final,
				Type:    in.Type,
				Payload: in.Payload,
			}
			log.Printf("Sent TaskResponse")

			continue
		}

		req, ok := lookupTable.Get(in.Id)
		if !ok {
			//log.Errorf(ctx, "Received message with  unknown id (%s): %+v", string(in.Id), in)
			log.Printf("Received message with  unknown id (%s): %+v", string(in.Id), in)
			continue
		}
		req.ResponseCh <- structs.TaskResponse{
			Order:   in.Order,
			Type:    in.Type,
			Payload: in.Payload,
		}
	}

}

type LookupTable struct {
	table map[string]*structs.TaskRequest
	lock  sync.RWMutex
}

func NewLookupTable() *LookupTable {
	return &LookupTable{
		table: make(map[string]*structs.TaskRequest),
	}
}

func (lt *LookupTable) Get(id string) (*structs.TaskRequest, bool) {
	lt.lock.RLock()
	defer lt.lock.RUnlock()
	respCh, ok := lt.table[id]
	return respCh, ok
}

func (lt *LookupTable) Set(id string, req *structs.TaskRequest) error {
	lt.lock.Lock()
	defer lt.lock.Unlock()

	_, ok := lt.table[id]
	if ok {
		return errors.New("This subscription already exists")
	}

	lt.table[id] = req

	return nil
}

func (lt *LookupTable) Delete(id string) {
	lt.lock.Lock()
	defer lt.lock.Unlock()

	delete(lt.table, id)
}

func (lt *LookupTable) Take(id string) (*structs.TaskRequest, bool) {
	lt.lock.Lock()
	defer lt.lock.Unlock()
	req, ok := lt.table[id]
	delete(lt.table, id)

	return req, ok
}
