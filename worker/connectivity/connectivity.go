package connectivity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"
)

type Address struct {
	Observer metrics.Observer
	StatusOK metrics.Counter
	Address  string
}

// WorkerConnections is connection controller for worker
type WorkerConnections struct {
	network                 string
	chainID                 string
	version                 string
	workerID                string
	workerAccessibleAddress string
	managerAddresses        map[string]Address
	managerAddressesLock    sync.RWMutex
}

// NewWorkerConnections is WorkerConnections constructor
func NewWorkerConnections(id, address, network, chainID, version string) *WorkerConnections {
	return &WorkerConnections{
		network:                 network,
		version:                 version,
		workerID:                id,
		chainID:                 chainID,
		workerAccessibleAddress: address,
		managerAddresses:        make(map[string]Address),
	}
}

// AddManager dynamically adds manager to the list
func (wc *WorkerConnections) AddManager(managerAddress string) {
	wc.managerAddressesLock.Lock()
	defer wc.managerAddressesLock.Unlock()

	wc.managerAddresses[managerAddress] = Address{
		Observer: workerChecksTaskDuration.WithLabels(managerAddress),
		StatusOK: workerStatus.WithLabels(managerAddress, "200", ""),
		Address:  managerAddress,
	}
}

// RemoveManager dynamically removes manager to the list
func (wc *WorkerConnections) RemoveManager(managerAddress string) {
	wc.managerAddressesLock.Lock()
	defer wc.managerAddressesLock.Unlock()
	delete(wc.managerAddresses, managerAddress)
}

type WorkerResponse struct {
	ID           string                 `json:"id"`
	Network      string                 `json:"network"`
	ChainID      string                 `json:"chain_id"`
	Connectivity WorkerInfoConnectivity `json:"connectivity"`
}

type WorkerInfoConnectivity struct {
	Version string `json:"version"`
	Type    string `json:"type"`
	Address string `json:"address"`
}

func (wc *WorkerConnections) gerWorkerInfo() WorkerResponse {
	return WorkerResponse{
		ID:      wc.workerID,
		Network: wc.network,
		ChainID: wc.chainID,
		Connectivity: WorkerInfoConnectivity{
			Version: wc.version,
			Type:    "grpc",
			Address: wc.workerAccessibleAddress,
		},
	}
}

// Run controls the registration of worker in manager. Every tick it sends it's identity (with address and network type) to every configured address.
func (wc *WorkerConnections) Run(ctx context.Context, logger *zap.Logger, dur time.Duration) {
	defer logger.Sync()

	tckr := time.NewTicker(dur)
	client := &http.Client{}

	wInfo, _ := json.Marshal(wc.gerWorkerInfo())
	readr := bytes.NewReader(wInfo)

	for {
		select {
		case <-ctx.Done():
			tckr.Stop()
			return
		case <-tckr.C:
			wc.managerAddressesLock.RLock()
			for _, ad := range wc.managerAddresses {
				readr.Seek(0, 0)
				timer := metrics.NewTimer(ad.Observer)
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+ad.Address, readr)
				if err != nil {
					workerStatus.WithLabels(ad.Address, "600", "err_creating_request").Inc()
					logger.Error(fmt.Sprintf("Error creating request %s", err.Error()), zap.String("address", ad.Address))
					continue
				}
				resp, err := client.Do(req)
				if err != nil {
					workerStatus.WithLabels(ad.Address, "600", "err_getting_response").Inc()
					logger.Error(fmt.Sprintf("Error connecting to manager on %s, %s", ad.Address, err.Error()), zap.String("address", ad.Address))
					continue
				}
				if resp.StatusCode > 399 {
					workerStatus.WithLabels(ad.Address, strconv.Itoa(resp.StatusCode), "err_response").Inc()
					logger.Error(fmt.Sprintf("Error returned from manager", ad.Address), zap.String("address", ad.Address), zap.String("status", resp.Status))
					timer.ObserveDuration()
					continue
				}
				ad.StatusOK.Inc()
				timer.ObserveDuration()
				resp.Body.Close()
			}
			wc.managerAddressesLock.RUnlock()
		}
	}
}
