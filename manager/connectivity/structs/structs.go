package structs

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type WorkerCompositeKey struct {
	Network string
	Version string
}

type WorkerInfo struct {
	Network string
	Version string
	ID      string
}

type TaskRequest struct {
	ID      uuid.UUID
	Network string
	Version string

	Type    string
	Payload json.RawMessage
}

type TaskErrorType string

type TaskError struct {
	Msg  string
	Type TaskErrorType
}

type TaskResponse struct {
	ID      uuid.UUID
	Version string
	Type    string
	Order   int64
	Final   bool
	Error   TaskError
	Payload json.RawMessage
}

type Await struct {
	sync.RWMutex
	Created        time.Time
	State          StreamState
	Resp           chan *TaskResponse
	ReceivedFinals int
	Uids           []uuid.UUID
}

func NewAwait(sendIDs []uuid.UUID) (aw *Await) {
	return &Await{
		Created: time.Now(),
		State:   StreamOnline,
		Uids:    sendIDs,
		Resp:    make(chan *TaskResponse, 400),
	}
}

func (aw *Await) Send(tr *TaskResponse) (bool, error) {
	aw.RLock()
	defer aw.RUnlock()

	if aw.State != StreamOnline {
		return false, errors.New("Cannot send recipient unavailable")
	}

	if tr.Final {
		aw.ReceivedFinals++
	}

	aw.Resp <- tr
	//	select {
	//	case aw.Resp <- tr:
	//log.Printf("Successfully sent")
	//	default:
	//		log.Printf("STREAM ERROR")
	//	}

	if len(aw.Uids) == aw.ReceivedFinals {
		log.Printf("Received All %s ", time.Now().Sub(aw.Created).String())
		return true, nil
	}

	return false, nil
}

func (aw *Await) Close() {

	//	log.Println("CLOSING AWAIT")
	aw.Lock()
	defer aw.Unlock()
	aw.State = StreamOffline

DRAIN:
	for {
		select {
		case <-aw.Resp:
		default:
			break DRAIN
		}
	}
	close(aw.Resp)
	aw.Resp = nil

	//log.Println("CLOSED AWAIT")
}

type IndexerClienter interface {
	RegisterStream(ctx context.Context, stream *StreamAccess) error
}

type StreamState int

const (
	StreamUnknown StreamState = iota
	StreamOnline
	StreamOffline
)

type StreamAccess struct {
	State           StreamState
	StreamID        uuid.UUID
	ResponseMap     map[uuid.UUID]*Await
	RequestListener chan TaskRequest

	respLock sync.RWMutex
	reqLock  sync.RWMutex

	mapLock sync.RWMutex
}

func NewStreamAccess() *StreamAccess {
	sID, _ := uuid.NewRandom()

	return &StreamAccess{
		StreamID: sID,
		State:    StreamOnline,

		ResponseMap:     make(map[uuid.UUID]*Await),
		RequestListener: make(chan TaskRequest, 100),
	}
}

func (sa *StreamAccess) Recv(tr *TaskResponse) error {

	sa.respLock.RLock()
	defer sa.respLock.RUnlock()
	if sa.State != StreamOnline {
		return errors.New("Stream is not Online")
	}

	sa.mapLock.Lock()
	resAwait, ok := sa.ResponseMap[tr.ID]
	sa.mapLock.Unlock()

	if !ok {
		return errors.New("No such requests registred")
	}

	all, err := resAwait.Send(tr)
	if err != nil {
		return err
	}

	if all {
		sa.mapLock.Lock()
		for _, u := range resAwait.Uids {
			delete(sa.ResponseMap, u)
		}
		sa.mapLock.Unlock()
	}
	return nil
}

func (sa *StreamAccess) Req(tr TaskRequest, aw *Await) error {

	sa.reqLock.RLock()
	defer sa.reqLock.RUnlock()
	if sa.State != StreamOnline {
		return errors.New("Stream is not Online")
	}

	sa.mapLock.Lock()
	sa.ResponseMap[tr.ID] = aw
	sa.mapLock.Unlock()

	sa.RequestListener <- tr

	return nil
}

func (sa *StreamAccess) RebufferRequest(tr TaskRequest) {
	sa.reqLock.RLock()
	defer sa.reqLock.RUnlock()

	sa.RequestListener <- tr
}

func (sa *StreamAccess) Close() error {
	sa.reqLock.Lock()
	sa.respLock.Lock()
	defer sa.respLock.Unlock()
	defer sa.reqLock.Unlock()
	if sa.State == StreamOffline {
		return nil
	}

	sa.ResponseMap = nil
	sa.State = StreamOffline
	close(sa.RequestListener)

	return nil
}
