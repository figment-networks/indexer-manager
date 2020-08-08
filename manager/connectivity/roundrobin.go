package connectivity

import (
	"errors"
	"fmt"
	"sync"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
)

type TaskWorkerRecord struct {
	workerID string
	//	chTr     chan structs.TaskRequest
	stream *structs.StreamAccess
}

type RoundRobinWorkers struct {
	next chan *TaskWorkerRecord
	lock sync.RWMutex
}

func NewRoundRobinWorkers() *RoundRobinWorkers {
	return &RoundRobinWorkers{
		next: make(chan *TaskWorkerRecord, 100),
	}
}

func (rrw *RoundRobinWorkers) AddWorker(id string, stream *structs.StreamAccess) error {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()

	select {
	case rrw.next <- &TaskWorkerRecord{id, stream}:
	default:
		return errors.New("No workers available")
	}

	return nil
}

func (rrw *RoundRobinWorkers) SendNext(tr structs.TaskRequest, aw *structs.Await) (failedWorkerID string, err error) {
	rrw.lock.RLock()
	defer rrw.lock.RUnlock()

	select {
	case ch := <-rrw.next:
		err := ch.stream.Req(tr, aw)
		if err != nil {
			return ch.workerID, fmt.Errorf("Cannot send to worker channel: %w", err)
		}
		rrw.next <- ch
	default:
		return "", errors.New("No workers available")
	}

	return "", nil
}
