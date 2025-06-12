package rewards_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type TopicUpdateTestSuite struct {
	testutil.TestSuite
}

func TestTopicUpdateTestSuite(t *testing.T) {
	suite.Run(t, &TopicUpdateTestSuite{
		testutil.NewTestSuite("topic_update"),
	})
}

// Test that pending topic updates are applied at epoch end
func (s *TopicUpdateTestSuite) TestPendingTopicUpdateAppliedAtEpochEnd() {
	block := int64(1)
	s.WithBlockHeight(block)

	s.SetParamsForTest()

	reputerIndexes := testutil.ReturnIndexes(0, 3)
	workerIndexes := testutil.ReturnIndexes(3, 3)
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	originalEpochLength := int64(100)

	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithAlphaRegret(alphaRegret),
		testutil.WithEpochLength(originalEpochLength),
		testutil.WithReputerStake(&stake),
	)
	topicId := topic.Id

	// Get original topic
	originalTopic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal("metadata", originalTopic.Metadata) // Check initial metadata

	// Create update request with new metadata
	newMetadata := "updated metadata"
	updateMsg := &types.UpdateTopicRequest{
		Sender:     originalTopic.Creator,
		TopicId:    topicId,
		Metadata:   []string{newMetadata},
		LossMethod: nil,
	}

	// Submit the update request
	_, err = s.EmissionsMsgServer().UpdateTopic(s.Ctx(), updateMsg)
	s.Require().NoError(err)

	// Verify pending update exists
	hasPending, err := s.EmissionsKeeper().HasPendingTopicUpdate(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().True(hasPending)

	// Original topic should still have old metadata
	currentTopic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal("metadata", currentTopic.Metadata)

	// Move to end of current epoch to trigger the update
	nextBlock, _, err := s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.WithBlockHeight(nextBlock)

	// Set reward emission for this block to ensure the topic is processed
	err = s.EmissionsKeeper().SetRewardCurrentBlockEmission(s.Ctx(), cosmosMath.NewInt(100))
	s.Require().NoError(err)

	s.EndBlock()

	// Verify the update was applied
	updatedTopicAfter, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal(newMetadata, updatedTopicAfter.Metadata)

	// Verify pending update was removed
	hasPending, err = s.EmissionsKeeper().HasPendingTopicUpdate(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().False(hasPending)
}
