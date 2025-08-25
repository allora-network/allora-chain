package rewards_test

import (
	"cosmossdk.io/collections"

	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Test defer execution of CloseReputerNonce
func (s *RewardsTestSuite) TestCloseReputerNonceTest_DeferExecWhenError() {
	currentBlockHeight := int64(10)
	s.WithBlockHeight(currentBlockHeight)

	reputerIndexes := testutil.ReturnIndexes(0, 5)
	workerIndexes := testutil.ReturnIndexes(5, 5)

	// Create topic
	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)
	// Insert unfullfiled nonces
	err := s.EmissionsKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)
	err = s.EmissionsKeeper().AddReputerNonce(s.Ctx(), topic.Id, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)

	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "100")

	// Insert inference from workers
	workerNonce := s.SetupInferences(topic.Id, currentBlockHeight, workerIndexes, workerValues...)

	// Move to end of worker submission window
	s.WithBlockHeight(currentBlockHeight + topic.WorkerSubmissionWindow)
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, workerNonce)
	s.Require().NoError(err)

	newBlockheight := currentBlockHeight + topic.GroundTruthLag
	s.WithBlockHeight(newBlockheight)
	// Trigger end block - rewards distribution
	s.EndBlock()

	reputerNonce := types.Nonce{BlockHeight: currentBlockHeight}

	// Skip inserting reputer values, as we want to trigger an error and test the defer block execution

	// before closing the nonce, the nonce should be unfulfilled
	unfulfilled, err := s.EmissionsKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topic.Id, &reputerNonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)

	// before closing the nonce, the active reputers for topic should be empty, since no reputer values were inserted
	activeReputers, err := s.EmissionsKeeper().GetActiveReputersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeReputers))

	err = actorutils.CloseReputerNonce(s.EmissionsKeeper(), s.Ctx(), topic, reputerNonce)
	s.Require().Error(err)

	// Check if reputer nonce is fulfilled
	unfulfilled, err = s.EmissionsKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topic.Id, &reputerNonce)
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	// Check if the active reputers for topic have been reset
	activeReputers, err = s.EmissionsKeeper().GetActiveReputersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeReputers))

	// Check if the submissions for the topic have been reset
	for _, idx := range reputerIndexes {
		_, err := s.EmissionsKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
		s.Require().ErrorIs(err, collections.ErrNotFound)
	}
}

// Test defer execution of CloseWorkerNonce
func (s *RewardsTestSuite) TestCloseWorkerNonce_DeferExecWhenError() {
	currentBlockHeight := int64(20)
	s.WithBlockHeight(currentBlockHeight)

	reputerIndexes := testutil.ReturnIndexes(0, 5)
	workerIndexes := testutil.ReturnIndexes(5, 5)

	// Create topic
	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)

	// Insert unfulfilled worker nonce
	err := s.EmissionsKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)

	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "100")

	// Insert inference from workers
	workerNonce := s.SetupInferences(topic.Id, currentBlockHeight, workerIndexes, workerValues...)

	// Move to end of worker submission window
	s.WithBlockHeight(currentBlockHeight + topic.WorkerSubmissionWindow)

	// Before closing, check nonce is unfulfilled, active workers exist, and submissions exist
	unfulfilled, err := s.EmissionsKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topic.Id, &workerNonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)

	activeInferers, err := s.EmissionsKeeper().GetActiveInferersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(len(workerIndexes), len(activeInferers))

	for _, idx := range workerIndexes {
		submissions, err := s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
		s.Require().NoError(err)
		s.Require().NotNil(submissions)
	}

	// Enforcing that the active inferer to create an error
	enforcedInferer := s.AddrsStr(10)
	err = s.EmissionsKeeper().AddActiveInferer(s.Ctx(), topic.Id, enforcedInferer)
	s.Require().NoError(err)

	// Call CloseWorkerNonce, expecting an error
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, workerNonce)
	s.Require().Error(err)

	// After closing, check nonce is fulfilled, active workers are reset, and submissions are cleared
	unfulfilled, err = s.EmissionsKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topic.Id, &workerNonce)
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	activeInferers, err = s.EmissionsKeeper().GetActiveInferersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeInferers))

	for _, idx := range workerIndexes {
		_, err = s.EmissionsKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
		s.Require().ErrorIs(err, collections.ErrNotFound)
	}
}
