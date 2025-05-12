// This is copied from the sei-chain
// source: https://github.com/sei-protocol/sei-chain/blob/main/utils/metrics/metrics_util.go
package metrics

import (
	"github.com/cosmos/cosmos-sdk/telemetry"
	metrics "github.com/hashicorp/go-metrics"
)

// IncrProducerEventCount increments the counter for events produced.
// This metric counts the number of events produced by the system.
// Metric Name:
//
//	allora_mint_produce_count
func IncrProducerEventCount(msgType string) {
	telemetry.IncrCounterWithLabels(
		[]string{"allora", "mint", "event", "total"},
		1,
		[]metrics.Label{telemetry.NewLabel("msg_type", msgType)},
	)
}
