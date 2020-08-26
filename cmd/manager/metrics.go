package main

import "github.com/figment-networks/indexing-engine/metrics"

var (
	receivedSyncMetric = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "manager_main",
		Name:      "received_sync",
		Desc:      "Register attempts received from workers",
		Tags:      []string{"network", "version", "address"},
	})
)
