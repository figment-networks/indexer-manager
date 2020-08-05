package connectivity

import (
	"context"
	"errors"
	"log"
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

type ConnTransport interface {
	Run(ctx context.Context, wc WorkerConnection)
	Type() string
}

type WorkerAddress struct {
	IP      net.IP
	Address string
}

type WorkerConnection struct {
	Version   string
	Type      string
	Addresses []WorkerAddress
}

type WorkerInfo struct {
	NodeSelfID     string
	Type           string
	State          State
	ConnectionInfo WorkerConnection
	LastCheck      time.Time
}

type Network struct {
	workers map[string]*WorkerInfo
}

type Manager struct {
	networks    map[string]*Network
	networkLock sync.RWMutex

	transports     map[string]ConnTransport
	transportsLock sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		networks:   make(map[string]*Network),
		transports: make(map[string]ConnTransport),
	}
}

func (m *Manager) Register(id, kind string, connInfo WorkerConnection) {
	m.networkLock.Lock()

	n, ok := m.networks[kind]
	if !ok {
		n = &Network{workers: make(map[string]*WorkerInfo)}
		m.networks[kind] = n
	}

	w, ok := n.workers[id]
	if ok { // (lukanus): node already registered
		w.LastCheck = time.Now()

		m.networkLock.Unlock()
		return
	}

	w = &WorkerInfo{
		NodeSelfID:     id,
		Type:           kind,
		ConnectionInfo: connInfo,
		State:          StateInitialized,
		LastCheck:      time.Now(),
	}

	n.workers[id] = w
	m.networkLock.Unlock()

	log.Printf("Registering %s %+v", connInfo.Type, connInfo)

	m.transportsLock.RLock()
	c := m.transports[connInfo.Type]
	go c.Run(context.Background(), connInfo)
	m.transportsLock.RUnlock()
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
