package cosmos

import "github.com/figment-networks/indexing-engine/metrics"

var (
	conversionDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "conversion_duration",
		Desc:      "Duration how long it takes to convert from raw to format",
		Tags:      []string{"type"},
	})

	rawRequestDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "request_duration",
		Desc:      "Duration how long it takes to take data from cosmos",
		Tags:      []string{"endpoint", "status"},
	})

	numberOfItems = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "number_of_items",
		Desc:      "Number of all transactions returned from one request",
		Tags:      []string{"type"},
	})

	blockCacheEfficiency = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "block_cache_efficiency",
		Desc:      "How Efficient the shared block cache is",
		Tags:      []string{"cache"},
	})

	blockCacheEfficiencyHit       *metrics.GroupCounter
	blockCacheEfficiencyMissed    *metrics.GroupCounter
	numberOfItemsTransactions     *metrics.GroupObserver
	transactionConversionDuration *metrics.GroupObserver
	convertionDurationObserver    *metrics.GroupObserver
)
