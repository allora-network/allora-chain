// This is copied from the sei-chain
// source: https://github.com/sei-protocol/sei-chain/blob/main/utils/metrics/metrics_util.go
package metrics

import (
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	metrics "github.com/hashicorp/go-metrics"
)

// Measures RPC endpoint request throughput
// Metric Name:
//
//	allora_rpc_request_counter
func IncrementRpcRequestCounter(endpoint string, err *error) {
	success := *err == nil
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "request", "counter"},
		float32(1),
		[]metrics.Label{
			telemetry.NewLabel("endpoint", endpoint),
			telemetry.NewLabel("success", strconv.FormatBool(success)),
		},
	)
}

// Measures the RPC request latency in milliseconds
// Metric Name:
//
//	allora_rpc_request_latency_ms
func MeasureRpcRequestLatency(endpoint string, startTime time.Time) {
	metrics.MeasureSinceWithLabels(
		[]string{"allora", "request", "latency_ms"},
		startTime.UTC(),
		[]metrics.Label{
			telemetry.NewLabel("endpoint", endpoint),
		},
	)
}

// IncrProducerEventCount increments the counter for events produced.
// This metric counts the number of events produced by the system.
// Metric Name:
//
//	allora_loadtest_produce_count
func IncrProducerEventCount(msgType string) {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "loadtest", "produce", "count"},
		1,
		[]metrics.Label{telemetry.NewLabel("msg_type", msgType)},
	)
}

func RecordMetrics(apiMethod string, startTime time.Time, err *error) {
	IncrementRpcRequestCounter(apiMethod, err)
	MeasureRpcRequestLatency(apiMethod, startTime)
}
