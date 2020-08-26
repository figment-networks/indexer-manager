package client

import "github.com/figment-networks/indexing-engine/metrics"

var (
	callDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "manager_client",
		Name:      "call_duration",
		Desc:      "Duration how long it takes to call respective",
		Tags:      []string{"call"},
	})

	requestsToGet = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "manager_client",
		Name:      "requests_to_get",
		Desc:      "Number of how many heights we're requesting in one call",
		Tags:      []string{""},
	})

	callDurationGetTransaction     *metrics.GroupObserver
	callDurationGetTransactions    *metrics.GroupObserver
	callDurationSearchTransactions *metrics.GroupObserver
	callDurationScrapeLatest       *metrics.GroupObserver
	callDurationInsertTransactions *metrics.GroupObserver
	requestsToGetMetric            *metrics.GroupObserver
)

func InitMetrics() {
	callDurationGetTransactions = callDuration.WithLabels("GetTransactions")
	callDurationSearchTransactions = callDuration.WithLabels("SearchTransactions")
	callDurationScrapeLatest = callDuration.WithLabels("ScrapeLatest")
	callDurationInsertTransactions = callDuration.WithLabels("InsertTransactions")
	requestsToGetMetric = requestsToGet.WithLabels("")
}
