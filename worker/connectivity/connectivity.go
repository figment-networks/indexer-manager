package connectivity

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

type WorkerConnections struct {
	network                 string
	version                 string
	workerid                string
	workerAccessibleAddress string
	managerAddresses        map[string]Address
	managerAddressesLock    sync.RWMutex
}

func NewWorkerConnections(id, address, network, version string) *WorkerConnections {
	return &WorkerConnections{
		network:                 network,
		version:                 version,
		workerid:                id,
		workerAccessibleAddress: address,
		managerAddresses:        make(map[string]Address),
	}
}

func (wc *WorkerConnections) AddManager(managerAddress string) {
	wc.managerAddressesLock.Lock()
	defer wc.managerAddressesLock.Unlock()

	wc.managerAddresses[managerAddress] = Address{
		Observer: workerChecksTaskDuration.WithLabels(managerAddress),
		StatusOK: workerStatus.WithLabels(managerAddress, "200", ""),
		Address:  managerAddress,
	}
}

func (wc *WorkerConnections) RemoveManager(managerAddress string) {
	wc.managerAddressesLock.Lock()
	defer wc.managerAddressesLock.Unlock()
	delete(wc.managerAddresses, managerAddress)
}

func (wc *WorkerConnections) Run(ctx context.Context, logger *zap.Logger, dur time.Duration) {
	defer logger.Sync()

	tckr := time.NewTicker(dur)

	client := &http.Client{}

	readr := strings.NewReader(fmt.Sprintf(`{"id":"%s","kind":"%s", "connectivity": {"version": "%s", "type":"grpc", "address": "%s" }}`, wc.workerid, wc.network, wc.version, wc.workerAccessibleAddress))

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
					logger.Error(fmt.Sprintf("Error returned from manager on %s, %s", ad.Address, err.Error()), zap.String("address", ad.Address))
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
