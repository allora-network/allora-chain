package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Test that pending topic updates are applied at epoch end
func (s *RewardsTestSuite) TestPendingTopicUpdateAppliedAtEpochEnd() {
	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest()

	reputerIndexes := s.returnIndexes(0, 3)
	workerIndexes := s.returnIndexes(3, 3)
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	originalEpochLength := int64(100)
	topicId := s.setUpTopicWithEpochLength(block, workerIndexes, reputerIndexes, stake, alphaRegret, originalEpochLength)

	// Get original topic
	originalTopic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal("test", originalTopic.Metadata) // Check initial metadata

	// Create update request with new metadata
	newMetadata := "updated metadata"
	updateMsg := &types.UpdateTopicRequest{
		Sender:                 originalTopic.Creator,
		TopicId:                topicId,
		Metadata:               []string{newMetadata},
		LossMethod:             nil,
		GroundTruthLag:         nil,
		WorkerSubmissionWindow: nil,
	}

	// Submit the update request
	_, err = s.msgServer.UpdateTopic(s.ctx, updateMsg)
	s.Require().NoError(err)

	// Verify pending update exists
	hasPending, err := s.emissionsKeeper.HasPendingTopicUpdate(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(hasPending)

	// Original topic should still have old metadata
	currentTopic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal("test", currentTopic.Metadata)

	// Move to end of current epoch to trigger the update
	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId)
	s.Require().NoError(err)
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(nextBlock)

	// This should apply the pending update
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(s.ctx, s.emissionsKeeper, nextBlock)
	s.Require().NoError(err)

	// Verify the update was applied
	updatedTopicAfter, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(newMetadata, updatedTopicAfter.Metadata)

	// Verify pending update was removed
	hasPending, err = s.emissionsKeeper.HasPendingTopicUpdate(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(hasPending)
}
