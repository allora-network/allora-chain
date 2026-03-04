package v0_16_0

import (
	"context"
	"errors"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

type mockTopicQuantileMigrationKeeper struct {
	nextTopicID uint64
	topics      map[uint64]emissionstypes.Topic
	getErr      error
	getErrByID  map[uint64]error
	setErrByID  map[uint64]error
	setCalls    []uint64
}

func (m *mockTopicQuantileMigrationKeeper) GetNextTopicId(_ context.Context) (uint64, error) {
	if m.getErr != nil {
		return 0, m.getErr
	}
	return m.nextTopicID, nil
}

func (m *mockTopicQuantileMigrationKeeper) GetTopic(_ context.Context, topicID uint64) (emissionstypes.Topic, error) {
	if err, ok := m.getErrByID[topicID]; ok {
		return emissionstypes.Topic{}, err
	}
	topic, ok := m.topics[topicID]
	if !ok {
		return emissionstypes.Topic{}, emissionstypes.ErrTopicDoesNotExist
	}
	return topic, nil
}

func (m *mockTopicQuantileMigrationKeeper) SetTopic(_ context.Context, topicID uint64, topic emissionstypes.Topic) error {
	if err, ok := m.setErrByID[topicID]; ok {
		return err
	}
	m.topics[topicID] = topic
	m.setCalls = append(m.setCalls, topicID)
	return nil
}

func TestMigrateTopicActiveQuantilesWithTopicKeeper(t *testing.T) {
	mockKeeper := &mockTopicQuantileMigrationKeeper{
		nextTopicID: 4,
		topics: map[uint64]emissionstypes.Topic{
			1: {
				ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.25"),
				ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.25"),
				ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.25"),
			},
			3: {
				ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.25"),
				ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.25"),
				ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.25"),
			},
		},
		getErrByID: map[uint64]error{},
		setErrByID: map[uint64]error{},
	}

	err := migrateTopicActiveQuantilesWithTopicKeeper(context.Background(), mockKeeper)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint64{1, 3}, mockKeeper.setCalls)

	for _, topicID := range []uint64{1, 3} {
		topic := mockKeeper.topics[topicID]
		require.True(t, topic.ActiveInfererQuantile.Equal(targetActiveTopicQuantile))
		require.True(t, topic.ActiveForecasterQuantile.Equal(targetActiveTopicQuantile))
		require.True(t, topic.ActiveReputerQuantile.Equal(targetActiveTopicQuantile))
	}
}

func TestMigrateTopicActiveQuantilesWithTopicKeeperFailsOnNextTopicID(t *testing.T) {
	mockKeeper := &mockTopicQuantileMigrationKeeper{
		getErr: errors.New("next topic id failed"),
	}

	err := migrateTopicActiveQuantilesWithTopicKeeper(context.Background(), mockKeeper)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get next topic id")
}

func TestMigrateTopicActiveQuantilesWithTopicKeeperFailsOnGetTopic(t *testing.T) {
	mockKeeper := &mockTopicQuantileMigrationKeeper{
		nextTopicID: 2,
		topics:      map[uint64]emissionstypes.Topic{},
		getErrByID: map[uint64]error{
			1: errors.New("get topic failed"),
		},
		setErrByID: map[uint64]error{},
	}

	err := migrateTopicActiveQuantilesWithTopicKeeper(context.Background(), mockKeeper)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch topic 1")
}

func TestMigrateTopicActiveQuantilesWithTopicKeeperFailsOnSetTopic(t *testing.T) {
	mockKeeper := &mockTopicQuantileMigrationKeeper{
		nextTopicID: 2,
		topics: map[uint64]emissionstypes.Topic{
			1: {},
		},
		getErrByID: map[uint64]error{},
		setErrByID: map[uint64]error{
			1: errors.New("set topic failed"),
		},
	}

	err := migrateTopicActiveQuantilesWithTopicKeeper(context.Background(), mockKeeper)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to update topic 1 quantiles")
}
