// This is copied from the sei-chain
// source: https://github.com/sei-protocol/sei-chain/blob/main/utils/metrics/metrics_util.go
package metrics

import (
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	metrics "github.com/hashicorp/go-metrics"
)

// Measures RPC endpoint request throughput
// Metric Name:
//
//	allora_rpc_request_counter
func IncrementRpcRequestCounter(endpoint string, err *error, labels map[string]string) {
	success := *err == nil
	metricLabels := []metrics.Label{
		telemetry.NewLabel("endpoint", endpoint),
		telemetry.NewLabel("success", strconv.FormatBool(success)),
	}

	// Add any additional labels
	for k, v := range labels {
		metricLabels = append(metricLabels, telemetry.NewLabel(k, v))
	}

	telemetry.IncrCounterWithLabels(
		[]string{"allora", "request", "counter"},
		float32(1),
		metricLabels,
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

// IncrProducerEventCountWithLabels increments the counter for events produced, with custom labels.
// This metric counts the number of events produced by the system, with additional labels.
// Metric Name:
//
//	allora_loadtest_produce_count
func IncrProducerEventCountWithLabels(msgType string, labels map[string]string) {
	// Always include the msg_type label
	allLabels := []metrics.Label{telemetry.NewLabel("msg_type", msgType)}
	for k, v := range labels {
		allLabels = append(allLabels, telemetry.NewLabel(k, v))
	}
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "loadtest", "produce", "count"},
		1,
		allLabels,
	)
}

func RecordMetrics(apiMethod string, startTime time.Time, err *error, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	IncrementRpcRequestCounter(apiMethod, err, labels)
	MeasureRpcRequestLatency(apiMethod, startTime)
}

// Helper function to create labels map with topicId and blockHeight if available
func CreateLabelsFromContext(ctx sdk.Context, topicId uint64) map[string]string {
	labels := make(map[string]string)
	if topicId > 0 {
		labels["topic_id"] = strconv.FormatUint(topicId, 10)
	}
	labels["blockHeight"] = strconv.FormatInt(ctx.BlockHeight(), 10)
	return labels
}

// Helper function to create labels for transactions
func CreateTxLabels(ctx sdk.Context, topicId uint64, address string, actorType string) map[string]string {
	labels := CreateLabelsFromContext(ctx, topicId)
	labels["blockHeightTx"] = strconv.FormatInt(ctx.BlockHeight(), 10)
	if address != "" {
		labels["address"] = address
	}
	if actorType != "" {
		labels["actorType"] = actorType
	}
	return labels
}

// Helper function to create labels for payload insertions
func CreatePayloadLabels(ctx sdk.Context, topicId uint64, address string, nonce int64) map[string]string {
	labels := CreateLabelsFromContext(ctx, topicId)
	labels["blockHeightTx"] = strconv.FormatInt(ctx.BlockHeight(), 10)
	labels["address"] = address
	labels["nonce"] = strconv.FormatInt(nonce, 10)
	return labels
}
