package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Target struct {
	ChainID string
	Network string
	Version string
	Address string
}

type NVCKey struct {
	Network string
	Version string
	ChainID string
}

func (nv NVCKey) String() string {
	return fmt.Sprintf("%s:%s (%s)", nv.Network, nv.ChainID, nv.Version)
}

type Scheme struct {
	destinations    map[NVCKey][]Target
	destinationLock sync.RWMutex

	logger *zap.Logger

	managers map[string]map[NVCKey]bool
}

type WorkerNetworkStatic struct {
	Workers map[string]WorkerInfoStatic `json:"workers"`
	All     int                         `json:"all"`
	Active  int                         `json:"active"`
}

type WorkerInfoStatic struct {
	NodeSelfID     string             `json:"node_id"`
	Type           string             `json:"type"`
	State          int64              `json:"state"`
	ConnectionInfo []WorkerConnection `json:"connection"`
	LastCheck      time.Time          `json:"last_check"`
}

type WorkerConnection struct {
	Version string `json:"version"`
	ChainID string `json:"chain_id"`
	Type    string `json:"type"`
}

func NewScheme(logger *zap.Logger) *Scheme {
	return &Scheme{
		logger:       logger,
		destinations: make(map[NVCKey][]Target),
		managers:     make(map[string]map[NVCKey]bool),
	}
}

func (s *Scheme) Add(t Target) {
	s.destinationLock.Lock()
	defer s.destinationLock.Unlock()

	i, ok := s.destinations[NVCKey{t.Network, t.Version, t.ChainID}]
	if !ok {
		i = []Target{}
	}
	i = append(i, t)

	s.destinations[NVCKey{t.Network, t.Version, t.ChainID}] = i
}

func (s *Scheme) Get(nv NVCKey) (Target, bool) {
	s.destinationLock.RLock()
	defer s.destinationLock.RUnlock()

	t, ok := s.destinations[nv]
	return t[0], ok
}

func (s *Scheme) AddManager(address string) {
	s.destinationLock.Lock()
	defer s.destinationLock.Unlock()

	if _, ok := s.managers[address]; ok {
		return // (lukanus) already added
	}

	s.logger.Info("[Scheme] Adding Manager", zap.String("address", address))
	s.managers[address] = make(map[NVCKey]bool)
}

func (s *Scheme) Refresh(ctx context.Context) error {
	c := http.Client{}

	s.destinationLock.Lock()
	defer s.destinationLock.Unlock()
	for address := range s.managers {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/get_workers", nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		wns := map[string]WorkerNetworkStatic{}

		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("error making request to  %s : %w", "http://"+address+"/get_workers", err)
		}

		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&wns)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("error making request to  %s : %w", "http://"+address+"/get_workers", err)
		}

		k := make(map[NVCKey]bool)

		for network, sub := range wns {
			for _, w := range sub.Workers {
				for _, ci := range w.ConnectionInfo {
					k[NVCKey{network, ci.Version, ci.ChainID}] = (w.State == 1) // (lukanus): 1 is  online
				}
			}
		}

		s.managers[address] = nil
		s.managers[address] = k
	}

	// (lukanus): link to destination
	for addr := range s.destinations {
		delete(s.destinations, addr)
	}

	for addr, targets := range s.managers {
		for nv, status := range targets {
			if !status {
				continue
			}
			dest, ok := s.destinations[nv]
			if !ok {
				dest = []Target{}
			}
			dest = append(dest, Target{Network: nv.Network, ChainID: nv.ChainID, Version: nv.Version, Address: addr})
			s.destinations[nv] = dest
		}
	}

	return nil
}

type schemeOutp struct {
	Destinations map[string][]Target        `json:"destinations"`
	Managers     map[string]map[string]bool `json:"managers"`
}

func (s *Scheme) handlerListDestination(w http.ResponseWriter, r *http.Request) {
	s.destinationLock.RLock()
	defer s.destinationLock.RUnlock()

	enc := json.NewEncoder(w)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	so := schemeOutp{
		Destinations: make(map[string][]Target),
		Managers:     make(map[string]map[string]bool),
	}

	for k, v := range s.destinations {
		so.Destinations[k.String()] = v
	}
	for k, v := range s.managers {
		m := map[string]bool{}
		for nv, val := range v {
			m[nv.String()] = val
		}
		so.Managers[k] = m
	}
	if err := enc.Encode(so); err != nil {
		s.logger.Error("[Scheme] Error encoding data http ", zap.Error(err))
	}

}

func (s *Scheme) RegisterHandles(smux *http.ServeMux) {
	smux.HandleFunc("/scheduler/destination/list", s.handlerListDestination)
}
