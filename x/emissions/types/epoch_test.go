package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTopicExtraLag(t *testing.T) {
	require.Equal(t, int64(0), TopicExtraLag(Topic{EpochLength: 100, GroundTruthLag: 100}))
	require.Equal(t, int64(0), TopicExtraLag(Topic{EpochLength: 100, GroundTruthLag: 200}))
	require.Equal(t, int64(50), TopicExtraLag(Topic{EpochLength: 100, GroundTruthLag: 150}))
	require.Equal(t, int64(0), TopicExtraLag(Topic{EpochLength: 0, GroundTruthLag: 150}))
}

func TestEpochLegacyNonce(t *testing.T) {
	epoch := Epoch{StartBlockHeight: 42}
	require.Equal(t, Nonce{BlockHeight: 42}, epoch.LegacyNonce())
}

func TestNewEpochAppliesExtraLagToReputerWindow(t *testing.T) {
	start := time.Unix(1_700_000_000, 0).UTC()
	topic := Topic{
		Id:                     7,
		EpochLength:            100,
		GroundTruthLag:         150,
		WorkerSubmissionWindow: 10,
	}
	epoch := NewEpoch(ZeroNonce().NextNonce(), topic, start)
	require.Equal(t, start, epoch.WorkerSubmissionWindow.OpenAt)
	require.Equal(t, start.Add(10*time.Second), epoch.WorkerSubmissionWindow.CloseAt)
	require.Equal(t, start.Add(150*time.Second), epoch.ReputerSubmissionWindow.OpenAt)
	require.Equal(t, start.Add(300*time.Second), epoch.ReputerSubmissionWindow.CloseAt)
}
