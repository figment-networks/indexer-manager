package grpc

import "github.com/figment-networks/indexing-engine/metrics"

var (
	requestMetric = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "transport_grpc",
		Name:      "requests",
		Desc:      "Requests reaching worker",
		Tags:      []string{"from", "type", "error"},
	})

	responseMetric = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "transport_grpc",
		Name:      "responses",
		Desc:      "Responses from worker",
		Tags:      []string{"type", "error"},
	})
)
