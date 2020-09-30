package connectivity

import "github.com/figment-networks/indexing-engine/metrics"

var (
	workerChecksTaskDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_connectivity",
		Name:      "checks_timing",
		Desc:      "Time reaching manager",
		Tags:      []string{"address"},
	})

	workerStatus = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "worker_connectivity",
		Name:      "checks_status",
		Desc:      "Statuses reaching manager",
		Tags:      []string{"address", "code", "error"},
	})
)
