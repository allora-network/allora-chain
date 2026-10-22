package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func TestInferMigratedEpochState(t *testing.T) {
	topic := types.Topic{
		EpochLength:            100,
		GroundTruthLag:         150,
		WorkerSubmissionWindow: 10,
	}
	legacy := int64(1000)

	cases := []struct {
		name                    string
		current                 int64
		workerOpen, reputerOpen bool
		want                    types.EpochState
	}{
		{"worker open in window", 1005, true, false, types.EpochState_WORKER_SUBMISSION},
		{"worker open past close waiting gtl", 1020, true, false, types.EpochState_WAITING_GROUND_TRUTH},
		{"reputer open mid window", 1160, false, true, types.EpochState_REPUTER_SUBMISSION},
		{"reputer past close", 1400, false, true, types.EpochState_PENDING_COMPLETION},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferMigratedEpochState(topic, legacy, tc.current, tc.workerOpen, tc.reputerOpen)
			require.Equal(t, tc.want, got)
		})
	}
}
