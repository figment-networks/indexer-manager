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

/*
func NoOp(ctx context.Context, trs chan cStructs.TaskResponse) {
NOOP_LOOP:
	for {
		select {
		case <-ctx.Done():
		case t, ok := <-trs:
			if !ok {
				break NOOP_LOOP
			}
			log.Println("NOOP: SKIPPED SENDING ", t)
			if t.Final {
				break NOOP_LOOP
			}

		}
	}
	cStructs.TaskResponseChanPool.Put(trs)

}

func (is *IndexerServer) Run(ctx context.Context) {

	a := map[uuid.UUID]*cStructs.StreamAccess{}

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-is.streamDelegation:
			switch t.Type {
			case cStructs.StreamDelegationResponse:
				s, ok := a[t.StreamID]
				if !ok || s.State != cStructs.StreamOnline {
					// If stream is not available, relieve worker
					tCtx, _ := context.WithTimeout(ctx, time.Minute*10)
					go NoOp(tCtx, t.ResponseListener)
					continue
				}

			READ_AS_MANY:
				for {
					select {
					case task, ok := <-t.ResponseListener:
						if !ok {
							break READ_AS_MANY
						}
						s.ResponseListener <- task

					default:
						break READ_AS_MANY
					}
				}

			case cStructs.StreamDelegationAdd:
				a[t.StreamID] = &cStructs.StreamAccess{
					State:            cStructs.StreamOnline,
					StreamID:         t.StreamID,
					ResponseListener: t.ResponseListener,
				}
			case cStructs.StreamDelegationDelete:
				s, ok := a[t.StreamID]
				if !ok {
					continue
				}
				s.State = cStructs.StreamOffline
			}
		}
	}
}
*/
func (is *IndexerServer) TaskRPC(taskStream indexer.IndexerService_TaskRPCServer) error {

	ctx := taskStream.Context()
	receiverClosed := make(chan error)
	defer close(receiverClosed)

	var after bool
	var stream *cStructs.StreamAccess
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
			return nil
		}

		if err != nil {
			receiverClosed <- err
			return err
		}

		if !after {
			stream = cStructs.NewStreamAccess()
			is.client.RegisterStream(ctx, stream)
			go Send(taskStream, stream, receiverClosed)
			defer stream.Close()
			after = true
		}

		uid, err := uuid.Parse(in.Id)
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
				log.Printf("Error sending TaskResponse: %s", err.Error())
			}
			//accessCh.RUnlock()
		}
	}
}
