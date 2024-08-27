package rewards_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) TestGetAllActiveEpochEndingTopics() {
	// Test case 1: No active topics
	block := int64(100)
	topicPageLimit := uint64(10)
	maxTopicPages := uint64(5)

	createNewTopic(s)
	createNewTopic(s)
	result := rewards.GetAllActiveEpochEndingTopics(s.ctx, s.emissionsKeeper, block, topicPageLimit, maxTopicPages)
	s.Require().Len(result, 0)
}

func (s *RewardsTestSuite) TestGetAllActiveEpochEndingTopicsActiveTopicsExistButNotEpochEnding() {
	// Test case 2: Active topics exist but no epoch ending topics
	block := int64(100)
	topicPageLimit := uint64(10)
	maxTopicPages := uint64(5)
	id1 := uint64(1)
	id3 := uint64(3)

	err := s.emissionsKeeper.SetTopic(s.ctx, id1, emissionstypes.Topic{
		Id:                     id1,
		Creator:                "creator",
		Metadata:               "metadata",
		LossMethod:             "mse",
		EpochLastEnded:         10,
		EpochLength:            100,
		GroundTruthLag:         10,
		WorkerSubmissionWindow: 10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		InitialRegret:          alloraMath.MustNewDecFromString("0.0001"),
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetTopic(s.ctx, id3, emissionstypes.Topic{
		Id:                     id3,
		Creator:                "creator",
		Metadata:               "metadata",
		LossMethod:             "mse",
		EpochLastEnded:         10,
		EpochLength:            100,
		GroundTruthLag:         10,
		WorkerSubmissionWindow: 10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		InitialRegret:          alloraMath.MustNewDecFromString("0.0001"),
	})
	s.Require().NoError(err)

	err = s.emissionsKeeper.ActivateTopic(s.ctx, id1)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(s.ctx, id3)
	s.Require().NoError(err)

	result := rewards.GetAllActiveEpochEndingTopics(s.ctx, s.emissionsKeeper, block, topicPageLimit, maxTopicPages)
	s.Require().Len(result, 0)
}

func (s *RewardsTestSuite) TestGetAllActiveEpochEndingTopicsActiveTopicsExistAndSomeEpochEnding() {
	// Test case 3: Active topics exist and some epoch ending topics
	block := int64(300)
	topicPageLimit := uint64(10)
	maxTopicPages := uint64(5)
	id1 := uint64(1)
	id3 := uint64(3)

	err := s.emissionsKeeper.SetTopic(s.ctx, id1, emissionstypes.Topic{
		Id:                     id1,
		Creator:                "creator",
		Metadata:               "metadata",
		LossMethod:             "mse",
		EpochLastEnded:         200,
		EpochLength:            100,
		GroundTruthLag:         10,
		WorkerSubmissionWindow: 10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		InitialRegret:          alloraMath.MustNewDecFromString("0.0001"),
	})
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetTopic(s.ctx, id3, emissionstypes.Topic{
		Id:                     id3,
		Creator:                "creator",
		Metadata:               "metadata",
		LossMethod:             "mse",
		EpochLastEnded:         250,
		EpochLength:            50,
		GroundTruthLag:         10,
		WorkerSubmissionWindow: 10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		InitialRegret:          alloraMath.MustNewDecFromString("0.0001"),
	})
	s.Require().NoError(err)

	err = s.emissionsKeeper.ActivateTopic(s.ctx, id1)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(s.ctx, id3)
	s.Require().NoError(err)

	result := rewards.GetAllActiveEpochEndingTopics(s.ctx, s.emissionsKeeper, block, topicPageLimit, maxTopicPages)
	s.Require().Len(result, 2)
	// Topics in result should be in ascending order of id
	s.Require().Equal(result[0].Id, id1, "topics in result should be in ascending order of id, and id1 should be of active topic")
	s.Require().Equal(result[1].Id, id3, "topics in result should be in ascending order of id, and id3 should be of active topic")
}

func (s *RewardsTestSuite) TestGetAllActiveEpochEndingTopicsActiveTopicsExistAndAbideByPageLimits() {
	// Test case 4: Respecting page limits
	block := int64(100)
	topicPageLimit := uint64(1)
	maxTopicPages := uint64(1)

	id1 := createNewTopic(s)
	id3 := createNewTopic(s)

	err := s.emissionsKeeper.ActivateTopic(s.ctx, id1)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(s.ctx, id3)
	s.Require().NoError(err)

	result := rewards.GetAllActiveEpochEndingTopics(s.ctx, s.emissionsKeeper, block, topicPageLimit, maxTopicPages)
	s.Require().Len(result, 1)
	s.Require().Equal(result[0].Id, id1, "topics in result should be in ascending order of id")
}
