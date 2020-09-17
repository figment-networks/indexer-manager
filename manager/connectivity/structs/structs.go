package structs

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrStreamNotOnline = errors.New("Stream is not Online")

type StreamState int

const (
	StreamUnknown StreamState = iota
	StreamOnline
	StreamError
	StreamReconnecting
	StreamClosing
	StreamOffline
)

type ConnTransport interface {
	Run(ctx context.Context, logger *zap.Logger, stream *StreamAccess)
	Type() string
}

type WorkerCompositeKey struct {
	Network string
	Version string
}

type SimpleWorkerInfo struct {
	Network string
	Version string
	ID      string
}

type WorkerInfo struct {
	NodeSelfID     string           `json:"node_id"`
	Type           string           `json:"type"`
	ChainID        string           `json:"chain_id"`
	State          StreamState      `json:"state"`
	ConnectionInfo WorkerConnection `json:"connection"`
	LastCheck      time.Time        `json:"last_check"`
}

type WorkerConnection struct {
	Version   string          `json:"version"`
	Type      string          `json:"type"`
	ChainID   string          `json:"chain_id"`
	Addresses []WorkerAddress `json:"addresses"`
}

type WorkerAddress struct {
	IP      net.IP `json:"ip"`
	Address string `json:"address"`
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

type ClientControl struct {
	Type string
	Resp chan ClientResponse
}

type ClientResponse struct {
	OK    bool
	Error string
	Time  time.Duration
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

	if len(aw.Uids) == aw.ReceivedFinals {
		log.Printf("Received All %s ", time.Now().Sub(aw.Created).String())
		return true, nil
	}

	return false, nil
}

func (aw *Await) Close() {
	aw.Lock()
	defer aw.Unlock()
	if aw.State != StreamOffline {
		return
	}
	aw.State = StreamOffline

Drain:
	for {
		select {
		case <-aw.Resp:
		default:
			break Drain
		}
	}
	close(aw.Resp)
	aw.Resp = nil
}

type IndexerClienter interface {
	RegisterStream(ctx context.Context, stream *StreamAccess) error
}

type StreamAccess struct {
	State           StreamState
	StreamID        uuid.UUID
	ResponseMap     map[uuid.UUID]*Await
	RequestListener chan TaskRequest

	ManagerID string

	ClientControl chan ClientControl

	CancelConnection context.CancelFunc

	WorkerInfo *WorkerInfo
	Transport  ConnTransport

	respLock sync.RWMutex
	reqLock  sync.RWMutex

	mapLock sync.RWMutex
}

func NewStreamAccess(transport ConnTransport, managerID string, conn *WorkerInfo) *StreamAccess {
	sID, _ := uuid.NewRandom()

	return &StreamAccess{
		StreamID:  sID,
		State:     StreamUnknown,
		ManagerID: managerID,

		Transport:     transport,
		WorkerInfo:    conn,
		ClientControl: make(chan ClientControl, 5),

		ResponseMap:     make(map[uuid.UUID]*Await),
		RequestListener: make(chan TaskRequest, 100),
	}
}

func (sa *StreamAccess) Run(ctx context.Context, logger *zap.Logger) error {
	sa.mapLock.Lock()
	if sa.State == StreamReconnecting {
		return errors.New("Already Reconnecting")
	}

	sa.State = StreamReconnecting
	sa.mapLock.Unlock()

	if sa.CancelConnection != nil {
		sa.CancelConnection()
	}

	var nCtx context.Context
	nCtx, sa.CancelConnection = context.WithCancel(ctx)
	go sa.Transport.Run(nCtx, logger, sa)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := sa.Ping(ctx)

	return err
}

func (sa *StreamAccess) Reconnect(ctx context.Context, logger *zap.Logger) error {
	return sa.Run(ctx, logger)
}

func (sa *StreamAccess) Recv(tr *TaskResponse) error {

	sa.respLock.RLock()
	defer sa.respLock.RUnlock()

	sa.mapLock.Lock()
	resAwait, ok := sa.ResponseMap[tr.ID]
	sa.mapLock.Unlock()

	if !ok {
		return errors.New("No such requests registered")
	}

	all, err := resAwait.Send(tr)
	if err != nil {
		resAwait.State = StreamError
		return err
	}

	resAwait.State = StreamOnline

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
		return ErrStreamNotOnline
	}

	sa.mapLock.Lock()
	sa.ResponseMap[tr.ID] = aw
	sa.mapLock.Unlock()

	sa.RequestListener <- tr

	return nil
}

func (sa *StreamAccess) Ping(ctx context.Context) (time.Duration, error) {
	resp := make(chan ClientResponse, 1) //(lukanus): this can be only closed after write

	select {
	case sa.ClientControl <- ClientControl{
		Type: "PING",
		Resp: resp,
	}:
	default:
		return 0, errors.New("Cannot send PING")
	}

	for {
		select {
		case <-ctx.Done(): // timeout
			sa.reqLock.Lock()
			sa.respLock.Lock()
			defer sa.respLock.Unlock()
			defer sa.reqLock.Unlock()

			sa.WorkerInfo.State = StreamOffline
			sa.State = StreamOffline
			return 0, errors.New("Ping TIMEOUT")
		case a := <-resp:
			sa.reqLock.Lock()
			sa.respLock.Lock()
			defer sa.respLock.Unlock()
			defer sa.reqLock.Unlock()
			sa.WorkerInfo.State = StreamOnline
			sa.State = StreamOnline
			return a.Time, nil
		}
	}
}

func (sa *StreamAccess) Close() error {
	sa.reqLock.Lock()
	sa.respLock.Lock()
	defer sa.respLock.Unlock()
	defer sa.reqLock.Unlock()

	if sa.State == StreamOffline {
		return nil
	}
	sa.State = StreamOffline
	sa.CancelConnection()

	if sa.WorkerInfo != nil {
		sa.WorkerInfo.State = StreamOffline
	}

	for id, aw := range sa.ResponseMap {
		// (lukanus): Close all awaits, they won't get full response anyway.
		aw.Close()
		delete(sa.ResponseMap, id)
	}

	return nil
}
