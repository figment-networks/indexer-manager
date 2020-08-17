package grpc

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/figment-networks/cosmos-indexer/manager/transport/grpc/indexer"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type ConnectGRPCRunner interface {
	Connect(ctx context.Context, stream *structs.StreamAccess) error
	Run(ctx context.Context, stream *structs.StreamAccess)
}

type AwaitingPing struct {
	t    time.Time
	resp chan structs.ClientResponse
	live bool
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Type() string {
	return "grpc"
}

func (c *Client) Run(ctx context.Context, stream *structs.StreamAccess) {
	defer stream.Close()
	id, _ := uuid.NewRandom()
	log.Printf("Running connections %s %+v", id.String(), stream.WorkerInfo.ConnectionInfo)
	var conn *grpc.ClientConn
	var errSet []error
	var dialErr error
	var connectedTo string
	// (lukanus): try every possibility
	for _, address := range stream.WorkerInfo.ConnectionInfo.Addresses {
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
		log.Printf("Cannot connect to any address given by worker: %s  %+v", id.String(), errSet)
		return
	}
	defer conn.Close()

	gIndexer := indexer.NewIndexerServiceClient(conn)
	taskStream, err := gIndexer.TaskRPC(ctx, grpc.WaitForReady(true))
	if err != nil {
		log.Printf("Cannot connect to any address given by worker: %s  %+v", id.String(), stream.WorkerInfo)
		return
	}

	log.Printf("Sucessfully Dialed %s %s  ", id.String(), connectedTo)

	receiverClosed := make(chan error)

	pingCh := make(chan structs.TaskResponse, 20)
	defer close(pingCh)

	pings := make(map[uuid.UUID]AwaitingPing)

	go Recv(id, taskStream, stream, pingCh, receiverClosed)

CONTROLRPC:
	for {
		select {
		case cT := <-stream.ClientControl:
			if cT.Type == "PING" {
				u, _ := uuid.NewRandom()
				pings[u] = AwaitingPing{time.Now(), cT.Resp, true}
				taskStream.Send(&indexer.TaskRequest{
					Id:   u.String(),
					Type: "PING",
				})
			} else if cT.Type == "CLOSE" {
				taskStream.CloseSend()
				cT.Resp <- structs.ClientResponse{OK: true}
				close(cT.Resp)
				return
			} else {
				log.Printf("Unknown ControlType %s %+v", id.String(), cT)
			}
		case resp := <-pingCh:
			p, ok := pings[resp.ID]
			if !ok {
				log.Printf("Unknown PONG message %s %+v", id.String(), p)
				continue
			}
			if p.resp != nil {
				log.Printf("Sending ClientResponse PONG %s", time.Since(p.t).String())
				p.resp <- structs.ClientResponse{OK: true, Time: time.Since(p.t)}
			}
			delete(pings, resp.ID)
		case <-ctx.Done():
			log.Printf("Context Done: %s", id.String())
			break CONTROLRPC
		case <-receiverClosed:
			log.Printf("ReceiverClosed: %s", id.String())
			break CONTROLRPC
		case req := <-stream.RequestListener:
			log.Printf("SENDING REQUEST %s  - %+v", id, req)
			if err := taskStream.Send(&indexer.TaskRequest{
				Id:      req.ID.String(),
				Type:    req.Type,
				Payload: req.Payload,
			}); err != nil {
				if err == io.EOF {
					log.Printf("Stream io.EOF")
					break CONTROLRPC
				}
				log.Printf("Error sending TaskResponse: %s %w", id.String(), err)
			}
		}

	}

	if taskStream != nil {
		taskStream.CloseSend()
	}

	for _, p := range pings {
		close(p.resp)
	}
	log.Printf("Finished: %s.String()", id)

}

func Recv(id uuid.UUID, stream indexer.IndexerService_TaskRPCClient, internalStream *structs.StreamAccess, pingCh chan<- structs.TaskResponse, finishCh chan error) {

	log.Printf("Started receive %s", id.String())
	defer close(finishCh)
	for {
		in, err := stream.Recv()
		if err == io.EOF { // (lukanus): Done reading/end of stream
			log.Printf("Finished receive %s", id.String())
			return
		}
		if err != nil {
			log.Printf("Finished receive error: %s %+v ", id.String(), err.Error())
			finishCh <- err
			return
		}

		id, err := uuid.Parse(in.Id)
		if err != nil {
			log.Printf("Cannot parseid %+v %s", in, err.Error())
			continue
		}

		if in.Type == "PONG" {
			//	log.Printf("Received PONG %+v ", in)
			select { // (lukanus): Never stuck
			case pingCh <- structs.TaskResponse{ID: id, Type: in.Type}:
			default:
				log.Printf("Cannot send response %+v ", in)
			}
			continue
		}

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

		if err = internalStream.Recv(resp); err != nil { // (lukanus): This error means it Cannot pass data to output
			log.Printf("Error in Recv %s %+v", id.String(), err.Error())
		}
	}

}
