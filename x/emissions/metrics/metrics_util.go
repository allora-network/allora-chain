package metrics

import (
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	metrics "github.com/hashicorp/go-metrics"
)

// Gauge metric with allorad version and git commit as labels
// Metric Name:
//
//	allorad_version_and_commit
func GaugeAlloradVersionAndCommit(version string, commit string) {
	telemetry.SetGaugeWithLabels(
		[]string{"allorad_version_and_commit"},
		1,
		[]metrics.Label{telemetry.NewLabel("allorad_version", version), telemetry.NewLabel("commit", commit)},
	)
}

// allora_tx_process_type_count
func IncrTxProcessTypeCounter(processType string) {
	metrics.IncrCounterWithLabels(
		[]string{"allora", "tx", "process", "type"},
		1,
		[]metrics.Label{telemetry.NewLabel("type", processType)},
	)
}

// Measures the time taken to process a block by the process type
// Metric Names:
//
//	allora_process_block_miliseconds
//	allora_process_block_miliseconds_count
//	allora_process_block_miliseconds_sum
func BlockProcessLatency(start time.Time, processType string) {
	metrics.MeasureSinceWithLabels(
		[]string{"allora", "process", "block", "milliseconds"},
		start.UTC(),
		[]metrics.Label{telemetry.NewLabel("type", processType)},
	)
}

// Measures the time taken to execute a sudo msg
// Metric Names:
//
//	allora_tx_process_type_count
func IncrDagBuildErrorCounter(reason string) {
	metrics.IncrCounterWithLabels(
		[]string{"allora", "dag", "build", "error"},
		1,
		[]metrics.Label{telemetry.NewLabel("reason", reason)},
	)
}

// Measures the time taken to execute a sudo msg
// Metric Names:
//
//	allora_deliver_tx_duration_miliseconds
//	allora_deliver_tx_duration_miliseconds_count
//	allora_deliver_tx_duration_miliseconds_sum
func MeasureDeliverTxDuration(start time.Time) {
	metrics.MeasureSince(
		[]string{"allora", "deliver", "tx", "milliseconds"},
		start.UTC(),
	)
}

// Measures the time taken to execute a batch tx
// Metric Names:
//
//	allora_deliver_batch_tx_duration_miliseconds
//	allora_deliver_batch_tx_duration_miliseconds_count
//	allora_deliver_batch_tx_duration_miliseconds_sum
func MeasureDeliverBatchTxDuration(start time.Time) {
	metrics.MeasureSince(
		[]string{"allora", "deliver", "batch", "tx", "milliseconds"},
		start.UTC(),
	)
}

// allora_epoch_new
func SetEpochNew(epochNum uint64) {
	metrics.SetGauge(
		[]string{"allora", "epoch", "new"},
		float32(epochNum),
	)
}

// Measures number of new websocket connects
// Metric Name:
//
//	allora_websocket_connect
func IncWebsocketConnects() {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "websocket", "connect"},
		1,
		nil,
	)
}

// Measures throughput per message type
// Metric Name:
//
//	allora_throughput_<metric_name>
func SetThroughputMetricByType(metricName string, value float32, msgType string) {
	telemetry.SetGaugeWithLabels(
		[]string{"allora", "loadtest", "tps", metricName},
		value,
		[]metrics.Label{telemetry.NewLabel("msg_type", msgType)},
	)
}

// Measures the number of times the total block gas wanted in the proposal exceeds the max
// Metric Name:
//
//	allora_failed_total_gas_wanted_check
func IncrFailedTotalGasWantedCheck(proposer string) {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "failed", "total", "gas", "wanted", "check"},
		1,
		[]metrics.Label{telemetry.NewLabel("proposer", proposer)},
	)
}

// Measures the number of times the total block gas wanted in the proposal exceeds the max
// Metric Name:
//
//	allora_failed_total_gas_wanted_check
func IncrValidatorSlashed(proposer string) {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "failed", "total", "gas", "wanted", "check"},
		1,
		[]metrics.Label{telemetry.NewLabel("proposer", proposer)},
	)
}

// Measures the number of times the total block gas wanted in the proposal exceeds the max
// Metric Name:
//
//	allora_tx_gas_counter
func IncrGasCounter(gasType string, value int64) {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "tx", "gas", "counter"},
		float32(value),
		[]metrics.Label{telemetry.NewLabel("type", gasType)},
	)
}

// Measures RPC endpoint request throughput
// Metric Name:
//
//	allora_rpc_request_counter
func IncrementRpcRequestCounter(endpoint string, connectionType string, success bool) {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "rpc", "request", "counter"},
		float32(1),
		[]metrics.Label{
			telemetry.NewLabel("endpoint", endpoint),
			telemetry.NewLabel("connection", connectionType),
			telemetry.NewLabel("success", strconv.FormatBool(success)),
		},
	)
}

// Measures the RPC request latency in milliseconds
// Metric Name:
//
//	allora_rpc_request_latency_ms
func MeasureRpcRequestLatency(endpoint string, connectionType string, startTime time.Time) {
	metrics.MeasureSinceWithLabels(
		[]string{"allora", "rpc", "request", "latency_ms"},
		startTime.UTC(),
		[]metrics.Label{
			telemetry.NewLabel("endpoint", endpoint),
			telemetry.NewLabel("connection", connectionType),
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

func RecordMetrics(apiMethod string, connectionType string, startTime time.Time, success bool) {
	IncrementRpcRequestCounter(apiMethod, connectionType, success)
	MeasureRpcRequestLatency(apiMethod, connectionType, startTime)
}
