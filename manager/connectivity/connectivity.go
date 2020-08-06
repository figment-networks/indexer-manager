package connectivity

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
)

type WorkersPool interface {
	AddWorker(id string, trCh chan structs.TaskRequest) error
	SendNext(tr structs.TaskRequest) (failedWorkerID string, err error)
}

type State string

const (
	StateInitialized State = "Initialized"
	StateOffline     State = "Offline"
	StateOnline      State = "Offline"
)

type ConnTransport interface {
	Run(ctx context.Context, id string, wc WorkerConnection, input <-chan structs.TaskRequest, removeWorkerCh chan<- string)
	Type() string
}

type WorkerAddress struct {
	IP      net.IP `json:"ip"`
	Address string `json:"address"`
}

type WorkerConnection struct {
	Version   string          `json:"version"`
	Type      string          `json:"type"`
	Addresses []WorkerAddress `json:"addresses"`
}

type WorkerInfo struct {
	NodeSelfID     string           `json:"node_id"`
	Type           string           `json:"type"`
	State          State            `json:"state"`
	ConnectionInfo WorkerConnection `json:"connection"`
	LastCheck      time.Time        `json:"last_check"`

	Cancelation context.CancelFunc `json:"-"`
}

type Network struct {
	workers map[string]*WorkerInfo
}

type Manager struct {
	networks    map[string]*Network
	networkLock sync.RWMutex

	transports     map[string]ConnTransport
	transportsLock sync.RWMutex

	nextWorkers     map[structs.WorkerCompositeKey]WorkersPool
	nextWorkersLock sync.RWMutex

	removeWorkerCh chan string
}

func NewManager() *Manager {
	return &Manager{
		networks:       make(map[string]*Network),
		transports:     make(map[string]ConnTransport),
		nextWorkers:    make(map[structs.WorkerCompositeKey]WorkersPool),
		removeWorkerCh: make(chan string, 20),
	}
}

func (m *Manager) Run(ctx context.Context, taskCh <-chan structs.TaskRequest) {
	// fan out client requests into their respective node
	for {
		select {
		case <-ctx.Done():
			return
		case wID := <-m.removeWorkerCh:
			m.removeWorker("", wID)
		case t := <-taskCh:
			m.nextWorkersLock.RLock()
			w, ok := m.nextWorkers[structs.WorkerCompositeKey{t.Network, t.Version}]
			if !ok {
				err := fmt.Errorf("No such worker for %s - %s", t.Network, t.Version)
				t.ResponseCh <- structs.TaskResponse{Error: structs.TaskError{
					Msg: err.Error(),
				}}
				m.nextWorkersLock.RUnlock()
				continue
			}

			failedID, err := w.SendNext(t)
			if failedID != "" {
				log.Println("FAILED: ", failedID, err)
				m.removeWorker(t.Network, failedID)
				// (lukanus): if thats only one worker failure, try few times
				for i := 0; i < 2; i++ {
					failedID, err = w.SendNext(t)
					if failedID != "" {
						m.removeWorker(t.Network, failedID)
					}
				}
			}
			if err != nil {
				log.Printf("Error sending TaskResponse: %w", err)
				t.ResponseCh <- structs.TaskResponse{Error: structs.TaskError{
					Msg: err.Error(),
				}}
			}

			m.nextWorkersLock.RUnlock()

		}
	}
}

func (m *Manager) Register(id, kind string, connInfo WorkerConnection) error {
	m.networkLock.Lock()

	n, ok := m.networks[kind]
	if !ok {
		n = &Network{workers: make(map[string]*WorkerInfo)}
		m.networks[kind] = n
	}

	w, ok := n.workers[id]
	if !ok {
		w = &WorkerInfo{
			NodeSelfID:     id,
			Type:           kind,
			ConnectionInfo: connInfo,
			State:          StateInitialized,
		}
		n.workers[id] = w
	}
	m.networkLock.Unlock()

	w.LastCheck = time.Now()

	if ok && w.State != StateOffline { // (lukanus): node already registered
		return nil
	}

	log.Printf("Registering %s %+v", connInfo.Type, connInfo)

	m.transportsLock.RLock()
	c, ok := m.transports[connInfo.Type]
	m.transportsLock.RUnlock()
	if !ok {
		return fmt.Errorf("Transport %s cannot be found", connInfo.Type)
	}

	m.nextWorkersLock.Lock()
	g, ok := m.nextWorkers[structs.WorkerCompositeKey{kind, connInfo.Version}]
	if !ok {
		g = NewRoundRobinWorkers()
		m.nextWorkers[structs.WorkerCompositeKey{kind, connInfo.Version}] = g
	}
	m.nextWorkersLock.Unlock()

	trCh := make(chan structs.TaskRequest, 20)
	ctx, cancel := context.WithCancel(context.Background())

	m.networkLock.Lock()
	w.State = StateInitialized
	w.Cancelation = cancel
	m.networkLock.Unlock()

	go c.Run(ctx, id, connInfo, trCh, m.removeWorkerCh)
	err := g.AddWorker(id, trCh)

	return err
}

func (m *Manager) GetAllWorkers() map[string]map[string]WorkerInfo {
	m.networkLock.RLock()
	defer m.networkLock.RUnlock()

	// (lukanus): unlink pointers
	winfos := make(map[string]map[string]WorkerInfo)
	for k, netw := range m.networks {
		wif := make(map[string]WorkerInfo)
		for kv, w := range netw.workers {
			wif[kv] = *w
		}
		winfos[k] = wif
	}
	return winfos
}

func (m *Manager) GetWorkers(kind string) []WorkerInfo {
	m.networkLock.RLock()
	defer m.networkLock.RUnlock()
	k, ok := m.networks[kind]
	if !ok {
		return nil
	}

	workers := make([]WorkerInfo, len(k.workers))
	var i int
	for _, worker := range k.workers {
		workers[i] = *worker
		i++
	}

	return workers
}

func (m *Manager) AddTransport(c ConnTransport) error {
	m.transportsLock.Lock()
	defer m.transportsLock.Unlock()

	_, ok := m.transports[c.Type()]
	if ok {
		return errors.New("Transport already registred ")
	}
	m.transports[c.Type()] = c
	return nil
}

func (m *Manager) removeWorker(kind, id string) {
	m.networkLock.RLock()
	defer m.networkLock.RUnlock()

	var w *WorkerInfo
	if kind == "" {
		for _, n := range m.networks {
			if worker, ok := n.workers[id]; ok {
				w = worker
			}
		}
	} else {
		n, ok := m.networks[kind]
		if !ok {
			return
		}

		w, ok = n.workers[id]
	}

	if w == nil {
		return
	}

	if w.Cancelation != nil {
		w.Cancelation()
		w.Cancelation = nil
	}
	w.State = StateOffline
}
