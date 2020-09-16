package connectivity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WorkersPool interface {
	AddWorker(id string, stream *structs.StreamAccess) error
	SendNext(tr structs.TaskRequest, aw *structs.Await) (failedWorkerID string, err error)
	Ping(ctx context.Context, id string) (time.Duration, error)
	Close(id string) error
	BringOnline(id string) error
	Reconnect(ctx context.Context, logger *zap.Logger, id string) error
	GetWorker(id string) (twi TaskWorkerInfo, ok bool)
	GetWorkers() TaskWorkerRecordInfo
	SendToWoker(id string, tr structs.TaskRequest, aw *structs.Await) error
}

type WorkerNetworkStatic struct {
	Workers map[string]WorkerInfoStatic `json:"workers"`
	All     int                         `json:"all"`
	Active  int                         `json:"active"`
}

type WorkerInfoStatic struct {
	structs.WorkerInfo
	TaskWorkerInfo TaskWorkerInfo `json:"tasks"`
}

type Network struct {
	workers map[string]*structs.WorkerInfo
}

type Manager struct {
	ID          string
	networks    map[string]*Network
	networkLock sync.RWMutex

	transports     map[string]structs.ConnTransport
	transportsLock sync.RWMutex

	nextWorkers     map[structs.WorkerCompositeKey]WorkersPool
	nextWorkersLock sync.RWMutex

	removeWorkerCh chan string

	logger *zap.Logger
}

// NewManager is Manager constructor
func NewManager(id string, logger *zap.Logger) *Manager {
	return &Manager{
		ID:             id,
		logger:         logger,
		networks:       make(map[string]*Network),
		transports:     make(map[string]structs.ConnTransport),
		nextWorkers:    make(map[structs.WorkerCompositeKey]WorkersPool),
		removeWorkerCh: make(chan string, 20),
	}
}

// Register new worker in Manager
func (m *Manager) Register(id, kind string, connInfo structs.WorkerConnection) error {
	m.networkLock.Lock()
	n, ok := m.networks[kind]
	if !ok {
		n = &Network{workers: make(map[string]*structs.WorkerInfo)}
		m.networks[kind] = n
	}
	m.networkLock.Unlock()

	w, ok := n.workers[id]
	if !ok {
		// (lukanus): check if the node is not previously registred under old selfID
		for _, work := range n.workers {
			if work.Type == kind {
				for _, oldAddr := range work.ConnectionInfo.Addresses {
					for _, newAddr := range connInfo.Addresses {
						if (newAddr.Address != "" && newAddr.Address == oldAddr.Address) ||
							(!newAddr.IP.IsUnspecified() && newAddr.IP.Equal(oldAddr.IP)) {
							// Same Address!
							m.logger.Info("[Manager] Node under the same address previously registered. Removing.", zap.Any("connection_info", oldAddr))
							if err := m.Unregister(work.NodeSelfID, work.Type, work.ConnectionInfo.Version); err != nil {
								m.logger.Error("[Manager] Error unregistring node.", zap.Error(err), zap.Any("connection_info", oldAddr))
							}
						}
					}
				}
			}
		}

		w = &structs.WorkerInfo{
			NodeSelfID:     id,
			Type:           kind,
			ConnectionInfo: connInfo,
			State:          structs.StreamUnknown,
		}
		n.workers[id] = w
	}

	w.LastCheck = time.Now()

	// log.Printf("STATE -  %+v, %+v  %+v ", w.State, w.LastCheck, w)

	if ok && w.State != structs.StreamOffline { // (lukanus): node already registered
		return nil
	}

	m.logger.Info("[Manager] Registering ", zap.String("type", connInfo.Type), zap.Any("connection_info", connInfo))
	m.transportsLock.RLock()
	c, ok := m.transports[connInfo.Type]
	m.transportsLock.RUnlock()
	if !ok {
		return fmt.Errorf("Transport %s cannot be found", connInfo.Type)
	}
	m.nextWorkersLock.RLock()
	g, ok := m.nextWorkers[structs.WorkerCompositeKey{Network: kind, Version: connInfo.Version}]
	m.nextWorkersLock.RUnlock()

	if !ok {
		g = NewRoundRobinWorkers()
		m.nextWorkersLock.Lock()
		m.nextWorkers[structs.WorkerCompositeKey{Network: kind, Version: connInfo.Version}] = g
		m.nextWorkersLock.Unlock()
	}

	if !ok { // Ading new worker
		m.networkLock.Lock()
		w.State = structs.StreamReconnecting
		m.networkLock.Unlock()

		sa := structs.NewStreamAccess(c, m.ID, w)
		err := sa.Run(context.Background(), m.logger)
		m.networkLock.Lock()
		if err != nil {
			w.State = structs.StreamOffline
		} else {
			w.State = structs.StreamOnline
		}
		m.networkLock.Unlock()

		return g.AddWorker(id, sa)
	}

	m.logger.Info("Reconnecting ", zap.String("type", connInfo.Type), zap.Any("connection_info", connInfo))
	if err := g.Reconnect(context.Background(), m.logger, id); err != nil {
		m.logger.Error("Reconnecting Error ", zap.Error(err), zap.Any("connection_info", connInfo))

		m.networkLock.Lock()
		w.State = structs.StreamReconnecting
		m.networkLock.Unlock()

		sa := structs.NewStreamAccess(c, m.ID, w)
		err := sa.Run(context.Background(), m.logger)
		m.networkLock.Lock()
		if err != nil {
			w.State = structs.StreamOffline
		} else {
			w.State = structs.StreamOnline
		}
		m.networkLock.Unlock()
		return g.AddWorker(id, sa)
	}

	return g.BringOnline(id)
}

// GetAllWorkers returns static list of workers
func (m *Manager) GetAllWorkers() map[string]WorkerNetworkStatic {
	m.networkLock.RLock()
	defer m.networkLock.RUnlock()

	// (lukanus): unlink pointers
	winfos := make(map[string]WorkerNetworkStatic)
	for k, netw := range m.networks {
		wif := WorkerNetworkStatic{
			Workers: make(map[string]WorkerInfoStatic),
		}
		for kv, w := range netw.workers {
			wis := WorkerInfoStatic{WorkerInfo: *w}
			m.nextWorkersLock.RLock()
			netWorker, ok := m.nextWorkers[structs.WorkerCompositeKey{Network: k, Version: wis.ConnectionInfo.Version}]
			if ok {
				wis.TaskWorkerInfo, ok = netWorker.GetWorker(kv)
				allWorkers := netWorker.GetWorkers()
				wif.Active = allWorkers.Active
				wif.All = allWorkers.All
			}
			m.nextWorkersLock.RUnlock()
			wif.Workers[kv] = wis
		}

		winfos[k] = wif
	}

	return winfos
}

// Unregister unregistring worker
func (m *Manager) Unregister(id, kind, version string) error {
	m.nextWorkersLock.Lock()
	defer m.nextWorkersLock.Unlock()
	nw := m.nextWorkers[structs.WorkerCompositeKey{Network: kind, Version: version}]
	return nw.Close(id)
}

// GetWorkers gets workers of kind
func (m *Manager) GetWorkers(kind string) []structs.WorkerInfo {
	m.networkLock.RLock()
	defer m.networkLock.RUnlock()
	k, ok := m.networks[kind]
	if !ok {
		return nil
	}

	workers := make([]structs.WorkerInfo, len(k.workers))
	var i int
	for _, worker := range k.workers {
		workers[i] = *worker
		i++
	}

	return workers
}

// AddTransport for connectivity
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

// Send sends a set of requests
func (m *Manager) Send(trs []structs.TaskRequest) (*structs.Await, error) {
	if len(trs) == 0 {
		return nil, errors.New("there is no transaction to be send")
	}

	first := trs[0]
	m.nextWorkersLock.RLock()
	w, ok := m.nextWorkers[structs.WorkerCompositeKey{Network: first.Network, Version: first.Version}]
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
	RetryLoop:
		for i := 0; i < 3; i++ {
			t.ID = uuids[requestNumber]
			failedID, err = w.SendNext(t, resp)

			if failedID == "" && err == nil {
				break RetryLoop
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
					err = w.Reconnect(context.Background(), m.logger, failedID)
				}
				if err == nil {
					log.Printf("PINGED %s SUCCESSFULLY  %s", failedID, duration.String())
					err = w.BringOnline(failedID)
				}

				if err == nil {
					err = w.SendToWoker(failedID, t, resp)

					if err == nil {
						break RetryLoop
					}
					log.Printf("Error Retrying: %s %+v ", failedID, err)
				}
			}

			if err != nil {
				log.Println("Retry failed: ", failedID, err)
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

// PingInfo contract is defined here
type PingInfo struct {
	ID           string           `json:"id"`
	Kind         string           `json:"kind"`
	Connectivity ConnectivityInfo `json:"connectivity"`
}
type ConnectivityInfo struct {
	Address string `json:"address"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

func (m *Manager) AttachToMux(mux *http.ServeMux) {
	b := &bytes.Buffer{}
	block := &sync.Mutex{}
	dec := json.NewDecoder(b)

	mux.HandleFunc("/client_ping", func(w http.ResponseWriter, r *http.Request) {
		pi := &PingInfo{}
		block.Lock()
		b.Reset()
		_, err := b.ReadFrom(r.Body)
		defer r.Body.Close()
		if err != nil {
			block.Unlock()
			m.logger.Error("Error getting request body in /client_ping", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = dec.Decode(pi)
		if err != nil {
			block.Unlock()
			m.logger.Error("Error decoding request body in /client_ping", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		block.Unlock()
		/*
			receivedSyncMetric = metrics.MustNewCounterWithTags(metrics.Options{
				Namespace: "indexers",
				Subsystem: "manager_main",
				Name:      "received_sync",
				Desc:      "Register attempts received from workers",
				Tags:      []string{"network", "version", "address"},
			})
		*/
		//	receivedSyncMetric.WithLabels(pi.Kind, pi.Connectivity.Version, pi.Connectivity.Address)
		ipTo := net.ParseIP(r.RemoteAddr)
		fwd := r.Header.Get("X-FORWARDED-FOR")
		if fwd != "" {
			ipTo = net.ParseIP(fwd)
		}
		m.Register(pi.ID, pi.Kind, structs.WorkerConnection{
			Version: pi.Connectivity.Version,
			Type:    pi.Connectivity.Type,
			Addresses: []structs.WorkerAddress{{
				IP:      ipTo,
				Address: pi.Connectivity.Address,
			}},
		})
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/get_workers", func(w http.ResponseWriter, r *http.Request) {
		m, err := json.Marshal(m.GetAllWorkers())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error marshaling data"}`))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(m)
		return
	})
}
