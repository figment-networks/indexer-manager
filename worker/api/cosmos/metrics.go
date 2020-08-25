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

	transactionConversionDuration = conversionDuration.WithLabels("transaction")

	rawRequestDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "request_duration",
		Desc:      "Duration how long it takes to take data from cosmos",
		Tags:      []string{"endpoint", "status"},
	})

	numberOfPages = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "number_of_pages",
		Desc:      "Number of pages returned from api",
		Tags:      []string{"type"},
	})
	numberOfPagesTransactions = numberOfPages.WithLabels("transactions")

	numberOfItems = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "number_of_items",
		Desc:      "Number of all transactions returned from one request",
		Tags:      []string{"type"},
	})
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")

	blockCacheEfficiency = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "worker_api_cosmos",
		Name:      "block_cache_efficiency",
		Desc:      "How Efficient the shared block cache is",
		Tags:      []string{"cache"},
	})

	blockCacheEfficiencyHit    = blockCacheEfficiency.WithLabels("hit")
	blockCacheEfficiencyMissed = blockCacheEfficiency.WithLabels("missed")
)
