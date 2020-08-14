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

type WorkersPool interface {
	AddWorker(id string, stream *structs.StreamAccess) error
	SendNext(tr structs.TaskRequest, aw *structs.Await) (failedWorkerID string, err error)
	Ping(ctx context.Context, id string) (time.Duration, error)
	Close(id string) error
	BringOnline(id string) error
	Reconnect(id string) error
	GetWorker(id string) (twi TaskWorkerInfo, ok bool)
	GetWorkers() []TaskWorkerInfo

	SendToWoker(id string, tr structs.TaskRequest, aw *structs.Await) error
}

type State string

const (
	StateInitialized State = "Initialized"
	StateOffline     State = "Offline"
	StateOnline      State = "Online"
)

type WorkerInfo struct {
	NodeSelfID     string                   `json:"node_id"`
	Type           string                   `json:"type"`
	State          State                    `json:"state"`
	ConnectionInfo structs.WorkerConnection `json:"connection"`
	LastCheck      time.Time                `json:"last_check"`
}

type WorkerInfoStatic struct {
	WorkerInfo
	TaskWorkerInfo TaskWorkerInfo `json:"tasks"`
}

type Network struct {
	workers map[string]*WorkerInfo
}

type Manager struct {
	networks    map[string]*Network
	networkLock sync.RWMutex

	transports     map[string]structs.ConnTransport
	transportsLock sync.RWMutex

	nextWorkers     map[structs.WorkerCompositeKey]WorkersPool
	nextWorkersLock sync.RWMutex

	removeWorkerCh chan string
}

func NewManager() *Manager {
	return &Manager{
		networks:       make(map[string]*Network),
		transports:     make(map[string]structs.ConnTransport),
		nextWorkers:    make(map[structs.WorkerCompositeKey]WorkersPool),
		removeWorkerCh: make(chan string, 20),
	}
}

func (m *Manager) Register(id, kind string, connInfo structs.WorkerConnection) error {
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

	m.networkLock.Lock()
	w.State = StateInitialized
	m.networkLock.Unlock()

	if !ok {
		sa := structs.NewStreamAccess(c, connInfo)
		sa.Run()
		return g.AddWorker(id, sa)
	}

	return nil
}

func (m *Manager) GetAllWorkers() map[string]map[string]WorkerInfoStatic {
	m.networkLock.RLock()
	defer m.networkLock.RUnlock()

	// (lukanus): unlink pointers
	winfos := make(map[string]map[string]WorkerInfoStatic)
	for k, netw := range m.networks {
		wif := make(map[string]WorkerInfoStatic)
		for kv, w := range netw.workers {
			wis := WorkerInfoStatic{WorkerInfo: *w}
			m.nextWorkersLock.RLock()
			netWorker, ok := m.nextWorkers[structs.WorkerCompositeKey{Network: k, Version: wis.ConnectionInfo.Version}]
			if ok {
				wis.TaskWorkerInfo, ok = netWorker.GetWorker(kv)
			}
			m.nextWorkersLock.RUnlock()

			wif[kv] = wis
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

func (m *Manager) AddTransport(c structs.ConnTransport) error {
	m.transportsLock.Lock()
	defer m.transportsLock.Unlock()

	_, ok := m.transports[c.Type()]
	if ok {
		return errors.New("Transport already registred ")
	}
	m.transports[c.Type()] = c
	return nil
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
			t.ID = uuids[requestNumber]
			failedID, err = w.SendNext(t, resp)

			log.Println("a ", failedID, err)
			if failedID == "" && err == nil {
				break RETRY_LOOP
			}

			if !errors.Is(err, ErrNoWorkersAvailable) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
				defer cancel()

				log.Printf("Pinging: %s  ", failedID)
				var duration time.Duration
				duration, err = w.Ping(ctx, failedID)
				if err != nil {
					w.Close(failedID)
					log.Printf("Error Pinging: %s %+v ", failedID, err)
					//m.removeWorker(t.Network, failedID)
					log.Printf("Reconnecting: %s  ", failedID)
					err = w.Reconnect(failedID)
				}
				if err == nil {
					log.Printf("PINGED %s SUCCESSFULLY  %s", failedID, duration.String)
					err = w.BringOnline(failedID)
				}

				if err == nil {
					err = w.SendToWoker(failedID, t, resp)

					if err == nil {
						break RETRY_LOOP
					}
					log.Printf("Error Retrying: %s %+v ", failedID, err)
				}
			}

			if err != nil {
				log.Println("Retry failed : ", failedID, err)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("Error sending TaskResponse: %w", err)
		}
	}
	return resp, nil
}

func makeUUIDs(count int) []uuid.UUID {
	uids := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		uids[i], _ = uuid.NewRandom()
	}
	return uids
}
