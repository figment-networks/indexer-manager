package connectivity

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/figment-networks/indexer-manager/manager/connectivity/structs"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrStreamOffline       = errors.New("stream is Offline")
	ErrNoWorkersAvailable  = errors.New("no workers available")
	ErrWorkerDoesNotExists = errors.New("workers does not exists")
)

type TaskWorkerInfo struct {
	WorkerID string    `json:"worker_id"`
	LastSend time.Time `json:"last_send"`

	StreamState structs.StreamState `json:"stream_state"`
	StreamID    uuid.UUID           `json:"stream_id"`
	ResponseMap map[uuid.UUID]TaskWorkerInfoAwait
}

type TaskWorkerInfoAwait struct {
	Created        time.Time           `json:"created"`
	State          structs.StreamState `json:"state"`
	ReceivedFinals int                 `json:"received_finals"`
	Uids           []uuid.UUID         `json:"expected"`
}

type TaskWorkerRecordInfo struct {
	Workers []TaskWorkerInfo `json:"workers"`
	All     int              `json:"all"`
	Active  int              `json:"active"`
}

type TaskWorkerRecord struct {
	workerID string
	stream   *structs.StreamAccess
	lastSend time.Time
}

type RoundRobinWorkers struct {
	trws map[string]*TaskWorkerRecord

	next chan *TaskWorkerRecord
	lock sync.RWMutex
}

func NewRoundRobinWorkers() *RoundRobinWorkers {
	return &RoundRobinWorkers{
		next: make(chan *TaskWorkerRecord, 100),
		trws: make(map[string]*TaskWorkerRecord),
	}
}

func (rrw *RoundRobinWorkers) AddWorker(id string, stream *structs.StreamAccess) error {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()

	trw := &TaskWorkerRecord{workerID: id, stream: stream}
	select {
	case rrw.next <- trw:
		rrw.trws[id] = trw
	default:
		return ErrNoWorkersAvailable
	}

	return nil
}

func (rrw *RoundRobinWorkers) SendNext(tr structs.TaskRequest, aw *structs.Await) (failedWorkerID string, err error) {
	rrw.lock.RLock()
	defer rrw.lock.RUnlock()

	select {
	case twr := <-rrw.next:
		if err := twr.stream.Req(tr, aw); err != nil {
			return twr.workerID, fmt.Errorf("Cannot send to worker channel: %w", err)
		}

		twr.lastSend = time.Now()
		rrw.next <- twr
	default:
		return "", ErrNoWorkersAvailable
	}

	return "", nil
}

// GetWorkers returns current workers information
func (rrw *RoundRobinWorkers) GetWorkers() TaskWorkerRecordInfo {
	rrw.lock.RLock()
	defer rrw.lock.RUnlock()

	twri := TaskWorkerRecordInfo{
		All:    len(rrw.trws),
		Active: len(rrw.next),
	}
	for _, w := range rrw.trws {
		twri.Workers = append(twri.Workers, getWorker(w))
	}

	return twri
}

func (rrw *RoundRobinWorkers) GetWorker(id string) (twi TaskWorkerInfo, ok bool) {
	rrw.lock.RLock()
	defer rrw.lock.RUnlock()

	t, ok := rrw.trws[id]
	if !ok {
		return twi, ok
	}

	return getWorker(t), ok
}

func getWorker(w *TaskWorkerRecord) TaskWorkerInfo {
	twr := TaskWorkerInfo{
		WorkerID: w.workerID,
		LastSend: w.lastSend,
	}

	if w.stream != nil {
		twr.StreamID = w.stream.StreamID
		twr.StreamState = w.stream.State

		twr.ResponseMap = map[uuid.UUID]TaskWorkerInfoAwait{}
		for k, r := range w.stream.ResponseMap {
			if r != nil {
				twia := TaskWorkerInfoAwait{
					Created:        r.Created,
					State:          r.State,
					ReceivedFinals: r.ReceivedFinals,
					Uids:           make([]uuid.UUID, len(r.Uids)),
				}
				copy(twia.Uids, r.Uids)
				twr.ResponseMap[k] = twia
			}
		}
	}
	return twr
}

func (rrw *RoundRobinWorkers) Ping(ctx context.Context, id string) (time.Duration, error) {
	rrw.lock.RLock()
	t, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		return 0, ErrWorkerDoesNotExists
	}

	return t.stream.Ping(ctx)
}

// BringOnline Brings worker Online, removing duplicates
func (rrw *RoundRobinWorkers) BringOnline(id string) error {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()
	t, ok := rrw.trws[id]
	if !ok {
		return ErrWorkerDoesNotExists
	}
	// (lukanus): Remove duplicates
	removeFromChannel(rrw.next, id)

	select {
	case rrw.next <- t:
	default:
		return ErrNoWorkersAvailable
	}

	return nil
}

// SendToWoker sends task to worker
func (rrw *RoundRobinWorkers) SendToWoker(id string, tr structs.TaskRequest, aw *structs.Await) error {
	rrw.lock.RLock()
	twr, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		return ErrWorkerDoesNotExists
	}

	if err := twr.stream.Req(tr, aw); err != nil {
		return fmt.Errorf("Cannot send to worker channel: %w", err)
	}
	twr.lastSend = time.Now()
	return nil
}

// Reconnect reconnects stream if exists
func (rrw *RoundRobinWorkers) Reconnect(ctx context.Context, logger *zap.Logger, id string) error {
	rrw.lock.RLock()
	t, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		return ErrWorkerDoesNotExists
	}
	return t.stream.Reconnect(ctx, logger)
}

// Close closes worker of given id
func (rrw *RoundRobinWorkers) Close(id string) error {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()

	t, ok := rrw.trws[id]
	if !ok {
		return ErrWorkerDoesNotExists
	}

	removeFromChannel(rrw.next, id)
	// t.stream.State = structs.StreamOffline

	return t.stream.Close()
}

func removeFromChannel(next chan *TaskWorkerRecord, id string) {
	inCh := len(next)
	for i := 0; i < inCh; i++ {
		w := <-next
		if w.workerID == id {
			continue
		}
		next <- w
	}
}
