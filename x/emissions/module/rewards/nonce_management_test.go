package rewards_test

import (
	"cosmossdk.io/collections"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func countReputerSubmissionWindowOpenedEvents(s *RewardsTestSuite, eventStart int) int {
	count := 0
	for _, event := range s.Ctx().EventManager().Events()[eventStart:] {
		if event.Type == "emissions.v10.EventReputerSubmissionWindowOpened" {
			count++
		}
	}
	return count
}

func (s *RewardsTestSuite) TestUpdateReputerNonceWaitsUntilWindowEnd() {
	topic := s.MockTopic()
	topic.Id = 9_000_001
	topic.EpochLength = 100
	topic.GroundTruthLag = 130
	nonce := types.Nonce{BlockHeight: 1000}

	err := s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &nonce)
	s.Require().NoError(err)

	oldCloseBlock := nonce.BlockHeight + topic.GroundTruthLag + topic.EpochLength
	s.WithBlockHeight(oldCloseBlock)
	err = rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, oldCloseBlock)
	s.Require().NoError(err)

	unfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topic.Id, &nonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)
}

func (s *RewardsTestSuite) TestUpdateReputerNonceReturnsCloseError() {
	topic := s.MockTopic()
	topic.Id = 9_000_002
	topic.EpochLength = 100
	topic.GroundTruthLag = 100
	nonce := types.Nonce{BlockHeight: 1000}

	err := s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &nonce)
	s.Require().NoError(err)

	windowEnd := nonce.BlockHeight + topic.GroundTruthLag + topic.EpochLength
	s.WithBlockHeight(windowEnd)
	err = rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, windowEnd)
	s.Require().ErrorIs(err, sdkerrors.ErrNotFound)
}

func (s *RewardsTestSuite) TestUpdateReputerNonceEmitsMissedOpenEventWithinWindow() {
	topic := s.MockTopic()
	topic.Id = 9_000_003
	topic.EpochLength = 100
	topic.GroundTruthLag = 130
	nonce := types.Nonce{BlockHeight: 1000}

	err := s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &nonce)
	s.Require().NoError(err)

	eventStart := len(s.Ctx().EventManager().Events())
	block := int64(1250)
	s.WithBlockHeight(block)
	err = rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, block)
	s.Require().NoError(err)
	s.Require().Equal(1, countReputerSubmissionWindowOpenedEvents(s, eventStart))
}

// A bundle accepted for a later nonce must not be attributed to, persisted
// under, or reported for the nonce currently being closed.
func (s *RewardsTestSuite) TestCloseReputerNonceSkipsBundlesForOtherNonce() {
	const (
		epochLength            = int64(100)
		groundTruthLag         = int64(100)
		workerSubmissionWindow = int64(10)
		nonce                  = int64(1000)
	)

	workerIndexes := testutil.ReturnIndexes(2, 3)
	reputerIndexes := testutil.ReturnIndexes(0, 2)
	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithWorkerSubmissionWindow(workerSubmissionWindow),
	)

	// First nonce: worker inferences plus a bundle from the first reputer.
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nonce}))
	s.Require().NoError(s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nonce}))
	s.WithBlockHeight(nonce)
	s.SetupInferences(topic.Id, nonce, workerIndexes)
	s.WithBlockHeight(nonce + workerSubmissionWindow)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nonce})

	// Second nonce, one epoch later: worker inferences plus a bundle from the
	// second reputer.
	nextNonce := nonce + epochLength
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nextNonce}))
	s.WithBlockHeight(nextNonce)
	s.SetupInferences(topic.Id, nextNonce, workerIndexes)
	s.WithBlockHeight(nextNonce + workerSubmissionWindow)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nextNonce})

	// The first reputer submits for the first nonce inside its window.
	windowStart, windowEnd, err := keeper.ReputerSubmissionWindowBounds(
		topic,
		types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: nonce}},
	)
	s.Require().NoError(err)
	s.Require().Equal(nonce+groundTruthLag, windowStart)
	s.WithBlockHeight(windowStart)
	s.Require().NoError(s.InsertReputerLossBundle(topic.Id, nonce, []int{reputerIndexes[0]}))

	// The second reputer submits for the next nonce on the shared close/open
	// block, which is accepted for the next nonce before the first nonce closes.
	s.WithBlockHeight(windowEnd)
	s.Require().NoError(s.InsertReputerLossBundle(topic.Id, nextNonce, []int{reputerIndexes[1]}))

	s.WithBlockHeight(windowEnd)
	s.Require().NoError(rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, windowEnd))

	closedBundles, err := s.ReputerLossKeeper().GetReputerLossBundlesAtBlock(s.Ctx(), topic.Id, nonce)
	s.Require().NoError(err)
	s.Require().Len(closedBundles, 1)
	s.Require().Equal(s.AddrsStr(reputerIndexes[0]), closedBundles[0].Reputer)

	networkLoss, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topic.Id, nonce)
	s.Require().NoError(err)
	s.Require().NotNil(networkLoss.ReputerRequestNonce)
	s.Require().NotNil(networkLoss.ReputerRequestNonce.ReputerNonce)
	s.Require().Equal(nonce, networkLoss.ReputerRequestNonce.ReputerNonce.BlockHeight)
}

// Test defer execution of CloseReputerNonce
func (s *RewardsTestSuite) TestCloseReputerNonceTest_DeferExecWhenError() {
	currentBlockHeight := int64(10)
	s.WithBlockHeight(currentBlockHeight)

	reputerIndexes := testutil.ReturnIndexes(0, 5)
	workerIndexes := testutil.ReturnIndexes(5, 5)

	// Create topic
	topic := s.FullTopicSetup(workerIndexes, reputerIndexes)
	// Insert unfullfiled nonces
	err := s.NonceKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)
	err = s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &types.Nonce{
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

	// Insert loss bundle from reputer
	// Use different indexes to enforce different workers are used
	// This will trigger an error and test if the defer execution of CloseReputerNonce works properly
	workerIndexes = testutil.ReturnIndexes(10, 5)
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")
	reputerNonce := types.Nonce{BlockHeight: currentBlockHeight}

	err = s.InsertReputerLossBundle(
		topic.Id,
		currentBlockHeight,
		reputerIndexes,
		testutil.WithReputerValues(reputerValues),
		testutil.WithSkipNetworkInferences(),
	)
	s.Require().NoError(err)

	// before closing the nonce, the nonce should be unfulfilled
	unfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topic.Id, &reputerNonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)

	// before closing the nonce, the active reputers for topic should not be
	activeReputers, err := s.ReputerLossKeeper().GetActiveReputersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(len(reputerIndexes), len(activeReputers))

	// before closing the nonce, the submissions for the topic should not be empty
	for _, idx := range reputerIndexes {
		submissions, err := s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
		s.Require().NoError(err)
		s.Require().NotNil(submissions)
	}

	err = actorutils.CloseReputerNonce(s.EmissionsKeeper(), s.Ctx(), topic, reputerNonce)
	s.Require().Error(err)

	// Check if reputer nonce is fulfilled
	unfulfilled, err = s.NonceKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topic.Id, &reputerNonce)
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	// Check if the active reputers for topic have been reset
	activeReputers, err = s.ReputerLossKeeper().GetActiveReputersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeReputers))

	// Check if the submissions for the topic have been reset
	for _, idx := range reputerIndexes {
		_, err := s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
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
	err := s.NonceKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{
		BlockHeight: currentBlockHeight,
	})
	s.Require().NoError(err)

	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "100")

	// Insert inference from workers
	workerNonce := s.SetupInferences(topic.Id, currentBlockHeight, workerIndexes, workerValues...)

	// Move to end of worker submission window
	s.WithBlockHeight(currentBlockHeight + topic.WorkerSubmissionWindow)

	// Before closing, check nonce is unfulfilled, active workers exist, and submissions exist
	unfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topic.Id, &workerNonce)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)

	activeInferers, err := s.WorkerKeeper().GetActiveInferersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(len(workerIndexes), len(activeInferers))

	for _, idx := range workerIndexes {
		submissions, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
		s.Require().NoError(err)
		s.Require().NotNil(submissions)
	}

	// Enforcing that the active inferer to create an error
	enforcedInferer := s.AddrsStr(10)
	err = s.WorkerKeeper().AddActiveInferer(s.Ctx(), topic.Id, enforcedInferer)
	s.Require().NoError(err)

	// Call CloseWorkerNonce, expecting an error
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, workerNonce)
	s.Require().Error(err)

	// After closing, check nonce is fulfilled, active workers are reset, and submissions are cleared
	unfulfilled, err = s.NonceKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topic.Id, &workerNonce)
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	activeInferers, err = s.WorkerKeeper().GetActiveInferersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal(0, len(activeInferers))

	for _, idx := range workerIndexes {
		_, err = s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topic.Id, s.AddrsStr(idx))
		s.Require().ErrorIs(err, collections.ErrNotFound)
	}
}
