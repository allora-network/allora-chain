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
	s.Require().Equal(originalEpochLength, originalTopic.EpochLength)

	// Create update request with new epoch length
	newEpochLength := int64(150)
	updateMsg := &types.UpdateTopicRequest{
		Sender:                 originalTopic.Creator,
		TopicId:                topicId,
		EpochLength:            []int64{newEpochLength},
		GroundTruthLag:         []int64{newEpochLength}, // Ensure validation passes
		WorkerSubmissionWindow: []int64{20},             // Ensure it's less than epoch length
	}

	// Submit the update request
	_, err = s.msgServer.UpdateTopic(s.ctx, updateMsg)
	s.Require().NoError(err)

	// Verify pending update exists
	hasPending, err := s.emissionsKeeper.HasPendingTopicUpdate(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(hasPending)

	// Original topic should still have old epoch length
	currentTopic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(originalEpochLength, currentTopic.EpochLength)

	// Move to end of current epoch to trigger the update
	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)

	// This should apply the pending update
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(s.ctx, s.emissionsKeeper, block)
	s.Require().NoError(err)

	// Verify the update was applied
	updatedTopicAfter, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(newEpochLength, updatedTopicAfter.EpochLength)

	// Verify new next churning block uses new epoch length
	newNextBlock, isActive, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive)
	s.Require().Equal(block+newEpochLength, newNextBlock)

	// Verify pending update was removed
	hasPending, err = s.emissionsKeeper.HasPendingTopicUpdate(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(hasPending)
}
