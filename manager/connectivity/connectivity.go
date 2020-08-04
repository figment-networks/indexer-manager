package connectivity

import (
	"net"
	"sync"
	"time"
)

type ConnectionManager interface {
}

type State string

const (
	StateInitialized State = "Initialized"
	StateOffline     State = "Offline"
	StateOnline      State = "Offline"
)

type WorkerAddress struct {
	IP     net.IP
	Domain string
}

type WorkerInfo struct {
	NodeSelfID string
	Type       string
	State      State
	Address    WorkerAddress
	LastCheck  time.Time
}

type Network struct {
	workers map[string]*WorkerInfo
}

type Manager struct {
	networks    map[string]*Network
	networkLock sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{networks: make(map[string]*Network)}
}

func (m *Manager) Register(id, kind string, address WorkerAddress) {
	m.networkLock.Lock()
	defer m.networkLock.Unlock()

	n, ok := m.networks[kind]
	if !ok {
		n = &Network{workers: make(map[string]*WorkerInfo)}
		m.networks[kind] = n
	}

	w, ok := n.workers[id]
	if ok { // (lukanus): node already registered
		w.LastCheck = time.Now()
		return
	}

	w = &WorkerInfo{
		NodeSelfID: id,
		Type:       kind,
		Address:    address,
		State:      StateInitialized,
		LastCheck:  time.Now(),
	}

	n.workers[id] = w
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
