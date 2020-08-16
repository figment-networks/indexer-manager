package connectivity

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/google/uuid"
)

var (
	ErrStreamOffline      = errors.New("stream is Offline")
	ErrNoWorkersAvailable = errors.New("no workers available")
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

func (rrw *RoundRobinWorkers) GetWorkers() []TaskWorkerInfo {
	rrw.lock.RLock()
	defer rrw.lock.RUnlock()

	twrs := []TaskWorkerInfo{}
	for _, w := range rrw.trws {
		twrs = append(twrs, getWorker(w))
	}

	return twrs
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
		return 0, errors.New("No Such Worker")
	}

	return t.stream.Ping(ctx)
}

func (rrw *RoundRobinWorkers) BringOnline(id string) error {
	rrw.lock.RLock()
	t, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		return errors.New("No Such Worker")
	}

	rrw.lock.Lock()
	defer rrw.lock.Unlock()

	log.Println("Bringing Online")
	select {
	case rrw.next <- t:
	default:
		return ErrNoWorkersAvailable
	}

	return nil
}

func (rrw *RoundRobinWorkers) SendToWoker(id string, tr structs.TaskRequest, aw *structs.Await) error {
	rrw.lock.RLock()
	twr, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		return errors.New("No Such Worker")
	}

	if err := twr.stream.Req(tr, aw); err != nil {
		return fmt.Errorf("Cannot send to worker channel: %w", err)
	}
	twr.lastSend = time.Now()
	return nil
}

func (rrw *RoundRobinWorkers) Reconnect(id string) error {

	log.Println("RoundRobinWorkers Reconnecting  ")
	rrw.lock.RLock()
	t, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		log.Println("RoundRobinWorkers Reconnecting NO SUCH WORKER  ")
		return errors.New("No Such Worker")
	}
	log.Println("RoundRobinWorkers Reconnecting ")
	return t.stream.Reconnect()
}

func (rrw *RoundRobinWorkers) Close(id string) error {
	rrw.lock.RLock()
	t, ok := rrw.trws[id]
	rrw.lock.RUnlock()
	if !ok {
		return errors.New("No Such Worker")
	}

	log.Println("RoundRobinWorkers Closing ")
	return t.stream.Close()
}

/*
func (rrw *RoundRobinWorkers) DeleteWorker(id string) error {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()


	t, ok := rrw.trws[id]
	if !ok {
		return nil
	}
	t.stream.State = structs.StreamOffline

	return nil
}
*/
