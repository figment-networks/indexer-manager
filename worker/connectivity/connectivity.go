package connectivity

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type WorkerConnections struct {
	workerid                string
	workerAccessibleAddress string
	managerAddresses        map[string]bool
	managerAddressesLock    sync.RWMutex
}

func NewWorkerConnections(id, address string) *WorkerConnections {
	return &WorkerConnections{workerid: id, workerAccessibleAddress: address, managerAddresses: make(map[string]bool)}
}

func (wc *WorkerConnections) AddManager(managerAddress string) {
	wc.managerAddressesLock.Lock()
	defer wc.managerAddressesLock.Unlock()
	wc.managerAddresses[managerAddress] = true
}

func (wc *WorkerConnections) RemoveManager(managerAddress string) {
	wc.managerAddressesLock.Lock()
	defer wc.managerAddressesLock.Unlock()
	delete(wc.managerAddresses, managerAddress)
}

func (wc *WorkerConnections) Run(ctx context.Context, dur time.Duration) {
	tckr := time.NewTicker(dur)

	client := &http.Client{}

	readr := strings.NewReader(fmt.Sprintf(`{"id":"%s","kind":"cosmos", "connectivity": {"version": "0.0.1", "type":"grpc", "address": "%s" }}`, wc.workerid, wc.workerAccessibleAddress))

	for {
		select {
		case <-ctx.Done():
			tckr.Stop()
			return
		case <-tckr.C:
			wc.managerAddressesLock.RLock()
			for address := range wc.managerAddresses {
				readr.Seek(0, 0)
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address, readr)
				if err != nil {
					log.Fatal("Error creating request")
				}
				resp, err := client.Do(req)
				if err != nil || resp.StatusCode > 399 {
					log.Println(fmt.Errorf("Error connecting to manager on %s, %w", address, err))
					continue
				}

				resp.Body.Close()
			}
			wc.managerAddressesLock.RUnlock()
		}
	}
}
