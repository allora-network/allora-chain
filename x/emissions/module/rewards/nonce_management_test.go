package rewards_test

import (
	"cosmossdk.io/collections"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gogoproto "github.com/cosmos/gogoproto/proto"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func countReputerSubmissionWindowOpenedEvents(s *RewardsTestSuite, eventStart int) int {
	count := 0
	openedEventName := gogoproto.MessageName(&types.EventReputerSubmissionWindowOpened{}) //nolint:exhaustruct // only the registered type name is needed, not an instance
	for _, event := range s.Ctx().EventManager().Events()[eventStart:] {
		if event.Type == openedEventName {
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

	// The bundle submitted for the next nonce on the shared block survives
	// the close of the previous nonce, and its submitter stays active.
	nextBundle, err := s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(reputerIndexes[1]))
	s.Require().NoError(err)
	s.Require().NotNil(nextBundle.ReputerRequestNonce)
	s.Require().NotNil(nextBundle.ReputerRequestNonce.ReputerNonce)
	s.Require().Equal(nextNonce, nextBundle.ReputerRequestNonce.ReputerNonce.BlockHeight)

	activeReputers, err := s.ReputerLossKeeper().GetActiveReputersForTopic(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Equal([]string{s.AddrsStr(reputerIndexes[1])}, activeReputers)

	// The previous nonce's submitter is retired by its own close.
	_, err = s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(reputerIndexes[0]))
	s.Require().ErrorIs(err, collections.ErrNotFound)

	// The next nonce's close consumes the surviving bundle.
	s.WithBlockHeight(windowEnd + epochLength)
	s.Require().NoError(rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, windowEnd+epochLength))

	nextNetworkLoss, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topic.Id, nextNonce)
	s.Require().NoError(err)
	s.Require().NotNil(nextNetworkLoss.ReputerRequestNonce)
	s.Require().NotNil(nextNetworkLoss.ReputerRequestNonce.ReputerNonce)
	s.Require().Equal(nextNonce, nextNetworkLoss.ReputerRequestNonce.ReputerNonce.BlockHeight)

	// The next nonce's submitter is retired after its own close.
	_, err = s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(reputerIndexes[1]))
	s.Require().ErrorIs(err, collections.ErrNotFound)
}

// A close that finds no bundle of its own nonce errors, but its deferred
// cleanup must not destroy a bundle stored for a later nonce: the next
// nonce's close consumes it and the topic settles.
func (s *RewardsTestSuite) TestCloseReputerNonceWithoutOwnBundleKeepsNextNonceBundle() {
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

	// First nonce: worker inferences only, no reputer submits for it.
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nonce}))
	s.Require().NoError(s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nonce}))
	s.WithBlockHeight(nonce)
	s.SetupInferences(topic.Id, nonce, workerIndexes)
	s.WithBlockHeight(nonce + workerSubmissionWindow)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nonce})

	// Second nonce, one epoch later: worker inferences.
	nextNonce := nonce + epochLength
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nextNonce}))
	s.WithBlockHeight(nextNonce)
	s.SetupInferences(topic.Id, nextNonce, workerIndexes)
	s.WithBlockHeight(nextNonce + workerSubmissionWindow)
	s.CloseWorkerNonce(topic, types.Nonce{BlockHeight: nextNonce})

	windowStart, windowEnd, err := keeper.ReputerSubmissionWindowBounds(
		topic,
		types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: nonce}},
	)
	s.Require().NoError(err)
	s.Require().Equal(nonce+groundTruthLag, windowStart)

	// The only submission, on the shared close/open block, is for the next nonce.
	s.WithBlockHeight(windowEnd)
	s.Require().NoError(s.InsertReputerLossBundle(topic.Id, nextNonce, []int{reputerIndexes[1]}))

	// The close of the first nonce errors with no bundle of its own, but its
	// deferred cleanup fulfills the nonce and leaves the next nonce's bundle.
	err = rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, windowEnd)
	s.Require().ErrorIs(err, sdkerrors.ErrNotFound)

	unfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: nonce})
	s.Require().NoError(err)
	s.Require().False(unfulfilled)

	nextBundle, err := s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(reputerIndexes[1]))
	s.Require().NoError(err)
	s.Require().Equal(nextNonce, nextBundle.ReputerRequestNonce.ReputerNonce.BlockHeight)

	// The next nonce's close consumes the surviving bundle and the topic settles.
	s.WithBlockHeight(windowEnd + epochLength)
	s.Require().NoError(rewards.UpdateReputerNonce(s.Ctx(), *s.EmissionsKeeper(), topic, windowEnd+epochLength))

	nextNetworkLoss, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topic.Id, nextNonce)
	s.Require().NoError(err)
	s.Require().NotNil(nextNetworkLoss.ReputerRequestNonce)
	s.Require().NotNil(nextNetworkLoss.ReputerRequestNonce.ReputerNonce)
	s.Require().Equal(nextNonce, nextNetworkLoss.ReputerRequestNonce.ReputerNonce.BlockHeight)
}

// Pruning reputer nonces retires submissions left over from nonces that can no
// longer be closed, so their reputers are not blocked from submitting again,
// while submissions of nonces still awaiting their close are kept.
func (s *RewardsTestSuite) TestPruneRetiresSubmissionsOfPrunedNonces() {
	topic := s.MockTopic()
	topic.Id = 9_000_004
	topic.EpochLength = 100
	topic.GroundTruthLag = 130

	prunedNonce := int64(100)
	keptNonce := int64(10_000)

	s.Require().NoError(s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: prunedNonce}))
	s.Require().NoError(s.NonceKeeper().AddReputerNonce(s.Ctx(), topic.Id, &types.Nonce{BlockHeight: keptNonce}))

	insertBundle := func(accountIndex int, nonceHeight int64) {
		//nolint:exhaustruct
		valueBundle := types.ValueBundle{
			TopicId: topic.Id,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: nonceHeight},
			},
			Reputer: s.AddrsStr(accountIndex),
			InfererValues: []*types.WorkerAttributedValue{
				{Worker: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("123")}, //nolint:exhaustruct // value bundle fields optional for storage
			},
			CombinedValue: alloraMath.MustNewDecFromString("123"),
			NaiveValue:    alloraMath.MustNewDecFromString("123"),
		}
		signature := s.SignValueBundle(&valueBundle, s.PrivKeys(accountIndex))
		err := s.ReputerLossKeeper().InsertReputerLoss(s.Ctx(), topic.Id, types.ReputerValueBundle{
			ValueBundle: &valueBundle,
			Signature:   signature,
			Pubkey:      s.PubKeyHexStr(accountIndex),
		})
		s.Require().NoError(err)
	}

	insertBundle(0, prunedNonce)
	insertBundle(1, keptNonce)
	for _, accountIndex := range []int{0, 1} {
		err := s.ReputerLossKeeper().AddActiveReputer(s.Ctx(), topic.Id, s.AddrsStr(accountIndex))
		s.Require().NoError(err)
	}

	// With the default MaxUnfulfilledReputerRequests of 100 the pruning
	// threshold at block 20000 is 9770: the pruned nonce falls below it, the
	// kept nonce does not.
	block := int64(20_000)
	s.WithBlockHeight(block)
	s.Require().NoError(rewards.PruneReputerAndWorkerNonces(s.Ctx(), *s.EmissionsKeeper(), topic, block))

	// The pruned nonce's submission and marker are gone.
	_, err := s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(0))
	s.Require().ErrorIs(err, collections.ErrNotFound)
	isActive, err := s.ReputerLossKeeper().IsActiveReputer(s.Ctx(), topic.Id, s.AddrsStr(0))
	s.Require().NoError(err)
	s.Require().False(isActive)

	// The kept nonce's submission and marker survive.
	keptBundle, err := s.ReputerLossKeeper().GetReputerLatestLossByTopicId(s.Ctx(), topic.Id, s.AddrsStr(1))
	s.Require().NoError(err)
	s.Require().Equal(keptNonce, keptBundle.ReputerRequestNonce.ReputerNonce.BlockHeight)
	isActive, err = s.ReputerLossKeeper().IsActiveReputer(s.Ctx(), topic.Id, s.AddrsStr(1))
	s.Require().NoError(err)
	s.Require().True(isActive)

	// The pruned nonce is gone from the unfulfilled set, the kept one remains.
	nonces, err := s.NonceKeeper().GetUnfulfilledReputerNonces(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().Len(nonces.Nonces, 1)
	s.Require().Equal(keptNonce, nonces.Nonces[0].ReputerNonce.BlockHeight)
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
