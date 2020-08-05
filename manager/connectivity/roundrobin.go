package connectivity

import (
	"errors"
	"sync"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
)

type TaskWorkerRecord struct {
	workerID string
	chTr     chan structs.TaskRequest
}

type RoundRobinWorkers struct {
	next chan *TaskWorkerRecord
	lock sync.Mutex
}

func NewRoundRobinWorkers() *RoundRobinWorkers {
	return &RoundRobinWorkers{
		next: make(chan *TaskWorkerRecord, 100),
	}
}

func (rrw *RoundRobinWorkers) AddWorker(id string, trCh chan structs.TaskRequest) error {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()

	select {
	case rrw.next <- &TaskWorkerRecord{id, trCh}:
	default:
		return errors.New("No workers available")
	}

	return nil
}

func (rrw *RoundRobinWorkers) SendNext(tr structs.TaskRequest) (failedWorkerID string, err error) {
	rrw.lock.Lock()
	defer rrw.lock.Unlock()

	select {
	case ch := <-rrw.next:
		select {
		case ch.chTr <- tr:
			rrw.next <- ch
		default:
			// remove broken from the pool
			return ch.workerID, errors.New("Cannot send to worker channel")
		}
	default:
		return "", errors.New("No workers available")
	}

	return "", nil
}
