package destination

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Target struct {
	Network string
	Version string
	Address string
}

type NVKey struct {
	Network string
	Version string
}

type Scheme struct {
	destinations    map[NVKey][]Target
	destinationLock sync.RWMutex

	managers map[string]map[NVKey]bool
}

type WorkerNetworkStatic struct {
	Workers map[string]WorkerInfoStatic `json:"workers"`
	All     int                         `json:"all"`
	Active  int                         `json:"active"`
}

type WorkerInfoStatic struct {
	NodeSelfID string `json:"node_id"`
	Type       string `json:"type"`
	//	State          StreamState      `json:"state"`
	ConnectionInfo WorkerConnection `json:"connection"`
	LastCheck      time.Time        `json:"last_check"`
}

type WorkerConnection struct {
	Version string `json:"version"`
	Type    string `json:"type"`
	//Addresses []WorkerAddress `json:"addresses"`
}

func NewScheme() *Scheme {
	return &Scheme{
		destinations: make(map[NVKey][]Target),

		managers: make(map[string]map[NVKey]bool),
	}
}

func (s *Scheme) Add(t Target) {
	s.destinationLock.Lock()
	defer s.destinationLock.Unlock()

	i, ok := s.destinations[NVKey{t.Network, t.Version}]
	if !ok {
		i = []Target{}
	}
	i = append(i, t)

	s.destinations[NVKey{t.Network, t.Version}] = i
}

func (s *Scheme) Get(nv NVKey) (Target, bool) {
	s.destinationLock.RLock()
	defer s.destinationLock.RUnlock()

	t, ok := s.destinations[nv]
	return t[0], ok
}

func (s *Scheme) AddManager(address string) {
	s.destinationLock.RLock()
	defer s.destinationLock.RUnlock()

	if _, ok := s.managers[address]; ok {
		return // (lukanus) already added
	}
	s.managers[address] = make(map[NVKey]bool)
}

func (s *Scheme) Refresh() error {
	c := http.Client{}

	s.destinationLock.RLock()
	defer s.destinationLock.RUnlock()
	for address := range s.managers {
		req, err := http.NewRequest(http.MethodGet, address+"/get_workers", nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		wns := map[string]WorkerNetworkStatic{}

		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("error making request to  %s : %w", address+"/get_workers", err)
		}

		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&wns)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("error making request to  %s : %w", address+"/get_workers", err)
		}

		k := make(map[NVKey]bool)

		for network, sub := range wns {
			for _, w := range sub.Workers {
				k[NVKey{network, w.ConnectionInfo.Version}] = true
			}
		}

		s.managers[address] = nil
		s.managers[address] = k
	}

	return nil
}
