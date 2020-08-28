package structs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var TaskResponseChanPool = NewtaskResponsePool(40)

type taskResponsePool struct {
	pool chan chan TaskResponse
}

func NewtaskResponsePool(size int) *taskResponsePool {
	return &taskResponsePool{make(chan chan TaskResponse, size)}
}

func (tr *taskResponsePool) Get() chan TaskResponse {
	select {
	case t := <-tr.pool:
		return t
	default:
	}

	return make(chan TaskResponse, 40)
}

func (tr *taskResponsePool) Put(t chan TaskResponse) {
	select {
	case tr.pool <- t:
	default:
		close(t)
	}
}

type StreamState int

const (
	StreamUnknown StreamState = iota
	StreamOnline
	StreamOffline
)

type StreamAccess struct {
	Finish           chan bool
	State            StreamState
	StreamID         uuid.UUID
	ResponseListener chan TaskResponse
	respLock         sync.RWMutex

	RequestListener chan TaskRequest
	reqLock         sync.RWMutex
}

func NewStreamAccess() *StreamAccess {

	responsesCh := TaskResponseChanPool.Get()
	sID, _ := uuid.NewRandom()

	return &StreamAccess{
		State:            StreamOnline,
		StreamID:         sID,
		ResponseListener: responsesCh,
		RequestListener:  make(chan TaskRequest, 30),
	}
}

func (sa *StreamAccess) Send(tr TaskResponse) error {
	sa.respLock.RLock()
	defer sa.respLock.RUnlock()

	if sa.State != StreamOnline {
		return errors.New("Stream is not Online")
	}

	sa.ResponseListener <- tr
	return nil

}

func (sa *StreamAccess) Req(tr TaskRequest) error {
	sa.reqLock.RLock()
	defer sa.reqLock.RUnlock()
	if sa.State != StreamOnline {
		return errors.New("Stream is not Online")
	}

	sa.RequestListener <- tr
	return nil
}

func (sa *StreamAccess) Close() error {
	sa.reqLock.Lock()
	sa.respLock.RLock()
	defer sa.respLock.RUnlock()
	defer sa.reqLock.Unlock()

	if sa.State == StreamOffline {
		return nil
	}

	TaskResponseChanPool.Put(sa.ResponseListener)
	close(sa.RequestListener)
	return nil
}

type OutResp struct {
	ID      uuid.UUID
	Type    string
	Payload interface{} // to be encoded
	Error   error
}

type TaskRequest struct {
	Id      uuid.UUID
	Type    string
	Payload json.RawMessage
}

type TaskError struct {
	Msg string
}

type TaskResponse struct {
	Version string
	Id      uuid.UUID
	Type    string
	Order   uint64
	Final   bool
	Error   TaskError
	Payload json.RawMessage
}

type IndexerClienter interface {
	RegisterStream(ctx context.Context, stream *StreamAccess) error
	CloseStream(ctx context.Context, streamID uuid.UUID) error
}
