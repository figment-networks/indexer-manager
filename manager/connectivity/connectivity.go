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
	"github.com/google/uuid"
)

type WorkersPool interface {
	//AddWorker(id string, trCh chan structs.TaskRequest) error
	AddWorker(id string, stream *structs.StreamAccess) error
	SendNext(tr structs.TaskRequest, aw *structs.Await) (failedWorkerID string, err error)
}

type State string

const (
	StateInitialized State = "Initialized"
	StateOffline     State = "Offline"
	StateOnline      State = "Offline"
)

type ConnTransport interface {
	Run(ctx context.Context, id string, wc WorkerConnection, stream *structs.StreamAccess, removeWorkerCh chan<- string)
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

	//	awaits     map[uuid.UUID]*structs.Await
	//	awaitsLock sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		networks:    make(map[string]*Network),
		transports:  make(map[string]ConnTransport),
		nextWorkers: make(map[structs.WorkerCompositeKey]WorkersPool),

		//	awaits:         make(map[uuid.UUID]*structs.Await),
		removeWorkerCh: make(chan string, 20),
	}
}

func (m *Manager) Run(ctx context.Context) {
	// fan out client requests into their respective node
	for {
		select {
		case <-ctx.Done():
			return
		case wID := <-m.removeWorkerCh:
			m.removeWorker("", wID)
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
	m.nextWorkersLock.RLock()
	g, ok := m.nextWorkers[structs.WorkerCompositeKey{kind, connInfo.Version}]
	m.nextWorkersLock.RUnlock()

	if !ok {
		g = NewRoundRobinWorkers()
		m.nextWorkersLock.Lock()
		m.nextWorkers[structs.WorkerCompositeKey{kind, connInfo.Version}] = g
		m.nextWorkersLock.Unlock()
	}

	ctx, cancel := context.WithCancel(context.Background())

	m.networkLock.Lock()
	w.State = StateInitialized
	w.Cancelation = cancel
	m.networkLock.Unlock()

	sa := structs.NewStreamAccess()
	go c.Run(ctx, id, connInfo, sa, m.removeWorkerCh)
	err := g.AddWorker(id, sa)
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

func (m *Manager) retryWorker(kind, id string) (bool, error) {
	log.Println("Retrying connection to worker")
	m.networkLock.RLock()

	var w *WorkerInfo
	n, ok := m.networks[kind]
	if !ok {
		m.networkLock.RUnlock()
		return false, nil
	}

	w, ok = n.workers[id]
	if !ok {
		m.networkLock.RUnlock()
		return false, nil
	}
	m.networkLock.RUnlock()

	if w != nil {
		if w.State == StateOffline {
			err := m.Register(id, kind, w.ConnectionInfo)
			return (err == nil), err
		} else if w.State == StateInitialized { // (lukanus) something else brought it up in the meanwhile - let'r retry
			return true, nil
		}
	}
	return false, nil
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

func makeUUIDs(count int) []uuid.UUID {
	uids := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		uids[i], _ = uuid.NewRandom()
	}
	return uids
}

func (m *Manager) Send(trs []structs.TaskRequest) (*structs.Await, error) {
	if len(trs) == 0 {
		return nil, errors.New("there is no transaction to be send")
	}

	first := trs[0]
	m.nextWorkersLock.RLock()
	w, ok := m.nextWorkers[structs.WorkerCompositeKey{first.Network, first.Version}]
	m.nextWorkersLock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("No such worker for %s - %s", first.Network, first.Version)
	}

	uuids := makeUUIDs(len(trs))
	//resp := m.registerRequest(uuids)
	resp := structs.NewAwait(uuids)
	for requestNumber, t := range trs {
		var err error
		var failedID string
		// (lukanus): if thats only one worker failure, try few times
	RETRY_LOOP:
		for i := 0; i < 3; i++ {
			var ok bool
			t.ID = uuids[requestNumber]
			failedID, err = w.SendNext(t, resp)
			if failedID == "" && err == nil {
				break RETRY_LOOP
			}

			ok, err = m.retryWorker(t.Network, failedID)
			if !ok {
				m.removeWorker(t.Network, failedID)
			}

			if err != nil {
				log.Println("Retrying failed : ", failedID, err)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("Error sending TaskResponse: %w", err)
		}
	}
	return resp, nil
}

/*
func (m *Manager) registerRequest(sendIDs []uuid.UUID) *structs.Await {
	//	m.awaitsLock.Lock()
	//	defer m.awaitsLock.Unlock()
	aw :=
	//	for _, id := range sendIDs {
	//		m.awaits[id] = aw
	//	}
	return aw
}*/
