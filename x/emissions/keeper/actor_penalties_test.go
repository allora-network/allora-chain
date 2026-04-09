package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

//nolint:exhaustruct
func (s *KeeperTestSuite) TestApplyLivenessPenaltyToInferer() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	givenTopic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		EpochLastEnded:      100,
		EpochLength:         10,
	}
	givenPreviousScore := types.Score{
		TopicId:     givenTopic.Id,
		BlockHeight: int64(55),
		Address:     "allo1l6nc88z4uqs00nnnaqkwjvlk4lxq3k4und7kzy",
		Score:       alloraMath.MustNewDecFromString("300"),
	}
	s.Require().NoError(k.GetScoresKeeper().SetTopicInitialInfererEmaScore(ctx, givenTopic.Id, alloraMath.MustNewDecFromString("200")))

	newScore, err := k.GetActorPenaltiesKeeper().ApplyLivenessPenaltyToInferer(ctx, givenTopic, 105, givenPreviousScore)
	s.Require().NoError(err)
	s.Require().Equal(givenPreviousScore.TopicId, newScore.TopicId)
	s.Require().Equal(givenPreviousScore.Address, newScore.Address)
	s.Require().Equal(int64(55), newScore.BlockHeight)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("265.61"), newScore.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", alloraMath.MustNewDecFromString("265.61"), newScore.Score)
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestApplyLivenessPenaltyToForecaster() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	givenTopic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		EpochLastEnded:      100,
		EpochLength:         10,
	}
	givenPreviousScore := types.Score{
		TopicId:     givenTopic.Id,
		BlockHeight: int64(55),
		Address:     "allo1l6nc88z4uqs00nnnaqkwjvlk4lxq3k4und7kzy",
		Score:       alloraMath.MustNewDecFromString("300"),
	}
	s.Require().NoError(k.GetScoresKeeper().SetTopicInitialForecasterEmaScore(ctx, givenTopic.Id, alloraMath.MustNewDecFromString("200")))

	newScore, err := k.GetActorPenaltiesKeeper().ApplyLivenessPenaltyToForecaster(ctx, givenTopic, 105, givenPreviousScore)
	s.Require().NoError(err)
	s.Require().Equal(givenPreviousScore.TopicId, newScore.TopicId)
	s.Require().Equal(givenPreviousScore.Address, newScore.Address)
	s.Require().Equal(int64(55), newScore.BlockHeight)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("265.61"), newScore.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", alloraMath.MustNewDecFromString("265.61"), newScore.Score)
}

//nolint:exhaustruct
func (s *KeeperTestSuite) TestApplyLivenessPenaltyToReputer() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	givenTopic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
		EpochLastEnded:      110,
		EpochLength:         10,
		GroundTruthLag:      10,
	}
	givenPreviousScore := types.Score{
		TopicId:     givenTopic.Id,
		BlockHeight: int64(55),
		Address:     "allo1l6nc88z4uqs00nnnaqkwjvlk4lxq3k4und7kzy",
		Score:       alloraMath.MustNewDecFromString("300"),
	}
	s.Require().NoError(k.GetScoresKeeper().SetTopicInitialReputerEmaScore(ctx, givenTopic.Id, alloraMath.MustNewDecFromString("200")))

	newScore, err := k.GetActorPenaltiesKeeper().ApplyLivenessPenaltyToReputer(ctx, givenTopic, 105, givenPreviousScore)
	s.Require().NoError(err)
	s.Require().Equal(givenPreviousScore.TopicId, newScore.TopicId)
	s.Require().Equal(givenPreviousScore.Address, newScore.Address)
	s.Require().Equal(int64(55), newScore.BlockHeight)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("265.61"), newScore.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", alloraMath.MustNewDecFromString("265.61"), newScore.Score)
}

//nolint:exhaustruct
func TestApplyLivenessPenaltyToActor(t *testing.T) {
	ctx := testutil.DefaultContextWithDB(t, storetypes.NewKVStoreKey("emissions"), storetypes.NewTransientStoreKey("transient_test")).Ctx
	givenTopic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.1"),
	}
	givenPreviousScore := types.Score{
		TopicId:     givenTopic.Id,
		BlockHeight: int64(100),
		Address:     "address",
		Score:       alloraMath.MustNewDecFromString("300"),
	}

	cases := []struct {
		name              string
		missedEpochs      int64
		withGetPenaltyErr error
		withSetScoreErr   error
		expectedScore     *types.Score
	}{
		{
			name:          "no missed epochs",
			missedEpochs:  0,
			expectedScore: &givenPreviousScore,
		},
		{
			name:         "apply penalty",
			missedEpochs: 4,
			expectedScore: &types.Score{
				TopicId:     givenPreviousScore.TopicId,
				BlockHeight: givenPreviousScore.BlockHeight, // Only applying score penalty, not updating block height
				Address:     givenPreviousScore.Address,
				Score:       alloraMath.MustNewDecFromString("265.61"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newScore, err := keeper.ApplyLivenessPenaltyToActor(
				ctx,
				// Mock missed epochs calculation
				func(topic types.Topic, _ int64) int64 {
					require.Equal(t, givenTopic, topic)
					return tc.missedEpochs
				},
				// Mock penalty retrieval
				func(topicId keeper.TopicId) (alloraMath.Dec, error) {
					require.Equal(t, givenTopic.Id, topicId)
					if tc.withGetPenaltyErr != nil {
						return alloraMath.ZeroDec(), tc.withGetPenaltyErr
					}
					return alloraMath.MustNewDecFromString("200"), nil
				},
				givenTopic,
				200,
				givenPreviousScore,
			)

			if tc.withGetPenaltyErr != nil {
				require.ErrorIs(t, tc.withGetPenaltyErr, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedScore.TopicId, newScore.TopicId, "expected %d, got %d", tc.expectedScore.TopicId, newScore.TopicId)
				require.Equal(t, tc.expectedScore.Address, newScore.Address, "expected %s, got %s", tc.expectedScore.Address, newScore.Address)
				require.Equal(t, tc.expectedScore.BlockHeight, newScore.BlockHeight, "expected %d, got %d", tc.expectedScore.BlockHeight, newScore.BlockHeight)
				inDelta, err := alloraMath.InDelta(tc.expectedScore.Score, newScore.Score, alloraMath.MustNewDecFromString("0.0001"))
				require.NoError(t, err)
				require.True(t, inDelta, "expected %s, got %s", tc.expectedScore.Score.String(), newScore.Score.String())
			}
		})
	}
}

//nolint:exhaustruct
func TestCountWorkerContiguousMissedEpochs(t *testing.T) {
	topic := types.Topic{
		EpochLastEnded: 100,
		EpochLength:    10,
	}

	cases := []struct {
		name                 string
		lastSubmittedNonce   int64
		expectedMissedEpochs int64
	}{
		{
			name:                 "in last epoch",
			lastSubmittedNonce:   95,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "after last epoch",
			lastSubmittedNonce:   105,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "one missed epoch",
			lastSubmittedNonce:   85,
			expectedMissedEpochs: 1,
		},
		{
			name:                 "four missed epoch",
			lastSubmittedNonce:   55,
			expectedMissedEpochs: 4,
		},
		{
			name:                 "on the edge of last epoch",
			lastSubmittedNonce:   90,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "on the edge of an epoch",
			lastSubmittedNonce:   60,
			expectedMissedEpochs: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			missedEpochs := keeper.CountWorkerContiguousMissedEpochs(topic, tc.lastSubmittedNonce)
			if missedEpochs != tc.expectedMissedEpochs {
				require.Equal(t, tc.expectedMissedEpochs, missedEpochs, "expected %d, got %d", tc.expectedMissedEpochs, missedEpochs)
			}
		})
	}
}

//nolint:exhaustruct
func TestCountReputerContiguousMissedEpochs(t *testing.T) {
	topic := types.Topic{
		EpochLastEnded: 110,
		EpochLength:    10,
	}

	cases := []struct {
		name                 string
		groundTruthLag       int64
		lastSubmittedNonce   int64
		expectedMissedEpochs int64
	}{
		{
			name:                 "in last epoch",
			groundTruthLag:       10,
			lastSubmittedNonce:   95,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "in last epoch (big gt-lag)",
			groundTruthLag:       30,
			lastSubmittedNonce:   75,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "after last epoch",
			groundTruthLag:       10,
			lastSubmittedNonce:   95,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "after last epoch (big gt-lag)",
			groundTruthLag:       30,
			lastSubmittedNonce:   75,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "one missed epoch",
			groundTruthLag:       10,
			lastSubmittedNonce:   85,
			expectedMissedEpochs: 1,
		},
		{
			name:                 "one missed epoch (big gt-lag)",
			groundTruthLag:       30,
			lastSubmittedNonce:   65,
			expectedMissedEpochs: 1,
		},
		{
			name:                 "four missed epoch",
			groundTruthLag:       10,
			lastSubmittedNonce:   55,
			expectedMissedEpochs: 4,
		},
		{
			name:                 "four missed epoch (big gt-lag)",
			groundTruthLag:       30,
			lastSubmittedNonce:   35,
			expectedMissedEpochs: 4,
		},
		{
			name:                 "on the edge of last epoch",
			groundTruthLag:       10,
			lastSubmittedNonce:   90,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "on the edge of last epoch (big gt-lag)",
			groundTruthLag:       30,
			lastSubmittedNonce:   70,
			expectedMissedEpochs: 0,
		},
		{
			name:                 "on the edge of an epoch",
			groundTruthLag:       10,
			lastSubmittedNonce:   60,
			expectedMissedEpochs: 3,
		},
		{
			name:                 "on the edge of an epoch (big gt-lag)",
			groundTruthLag:       30,
			lastSubmittedNonce:   40,
			expectedMissedEpochs: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			topic.GroundTruthLag = tc.groundTruthLag
			missedEpochs := keeper.CountReputerContiguousMissedEpochs(topic, tc.lastSubmittedNonce)
			if missedEpochs != tc.expectedMissedEpochs {
				require.Equal(t, tc.expectedMissedEpochs, missedEpochs, "expected %d, got %d", tc.expectedMissedEpochs, missedEpochs)
			}
		})
	}
}
