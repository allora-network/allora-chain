package keeper_test

import (
	cosmosMath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// lastTopicStatusChangedEvent returns the most recent EventTopicStatusChanged
// the event manager recorded, or nil.
func lastTopicStatusChangedEvent(em *sdk.EventManager) *types.EventTopicStatusChanged {
	var out *types.EventTopicStatusChanged
	for _, ev := range em.Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if typed, ok := msg.(*types.EventTopicStatusChanged); ok {
			out = typed
		}
	}
	return out
}

// Deactivations carry the reason the caller passed, plus the topic's weight
// and the active-set minimum that applied.
func (s *KeeperTestSuite) TestInactivateTopicEmitsReasonAndWeights() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	params := types.DefaultParams()
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	block := int64(100)
	topicId := uint64(1)
	topic := s.MockTopic()
	topic.Id = topicId
	s.Require().NoError(k.SetTopic(ctx, topicId, topic))
	s.Require().NoError(k.SetTopicToNextPossibleChurningBlock(ctx, topicId, block))
	s.Require().NoError(k.SetActiveTopics(ctx, topicId))
	s.Require().NoError(k.SetBlockToActiveTopics(ctx, block, types.TopicIds{TopicIds: []uint64{topicId}}))
	s.Require().NoError(k.SetPreviousTopicWeight(ctx, topicId, alloraMath.NewDecFromInt64(100)))
	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, topicId, cosmosMath.NewInt(50)))

	weight := alloraMath.MustNewDecFromString("0.4")
	em := sdk.NewEventManager()
	err = k.InactivateTopic(ctx.WithEventManager(em), topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_WEIGHT_BELOW_MINIMUM, weight, params.MinTopicWeight)
	s.Require().NoError(err)

	event := lastTopicStatusChangedEvent(em)
	s.Require().NotNil(event)
	s.Require().Equal(topicId, event.TopicId)
	s.Require().False(event.IsActive)
	s.Require().Equal(types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_WEIGHT_BELOW_MINIMUM, event.Reason)
	s.Require().Equal(params.MinTopicWeight, event.MinWeight)
	s.Require().True(weight.Equal(event.Weight), "expected weight %s, got %s", weight.String(), event.Weight.String())
}

// Activations carry the caller's reason and the weights that applied.
func (s *KeeperTestSuite) TestActivateTopicEmitsReasonAndWeights() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	params := types.DefaultParams()
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	topicId := s.CreateTopic(testutil.WithEpochLength(100), testutil.WithWorkerSubmissionWindow(100))

	weight := alloraMath.MustNewDecFromString("0.4")
	em := sdk.NewEventManager()
	err = k.ActivateTopic(ctx.WithEventManager(em), topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING, weight, params.MinTopicWeight)
	s.Require().NoError(err)

	event := lastTopicStatusChangedEvent(em)
	s.Require().NotNil(event)
	s.Require().Equal(topicId, event.TopicId)
	s.Require().True(event.IsActive)
	s.Require().Equal(types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING, event.Reason)
	s.Require().Equal(params.MinTopicWeight, event.MinWeight)
	s.Require().True(weight.Equal(event.Weight), "expected weight %s, got %s", weight.String(), event.Weight.String())
}

// When a full active set admits a higher-weight topic, the displaced topic's
// deactivation event says it was evicted by the higher weight.
func (s *KeeperTestSuite) TestEvictedTopicEmitsEvictedByHigherWeight() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = 1
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	epochLength := int64(100)
	lowTopicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	highTopicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))

	// Fill the single active slot with the low-weight topic.
	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, lowTopicId, cosmosMath.NewInt(100)))
	s.Require().NoError(k.ActivateTopic(ctx, lowTopicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec()))

	// Give the other topic more stake, then activate it into the full set,
	// evicting the first.
	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, highTopicId, cosmosMath.NewInt(200)))
	highWeight, err := k.GetTopicWeightFromTopicId(ctx, highTopicId)
	s.Require().NoError(err)

	// Snapshot the stored lowest-weight value the eviction decision reads,
	// before the activation replaces it.
	topic, err := k.GetTopic(ctx, lowTopicId)
	s.Require().NoError(err)
	lowEpochEndBlock := ctx.BlockHeight() + topic.EpochLength
	storedLowest, noPrior, err := k.GetLowestActiveTopicWeightAtBlock(ctx, lowEpochEndBlock)
	s.Require().NoError(err)
	s.Require().False(noPrior, "the full active set must have a stored lowest weight")
	s.Require().Equal(lowTopicId, storedLowest.TopicId)

	em := sdk.NewEventManager()
	err = k.ActivateTopic(ctx.WithEventManager(em), highTopicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING, highWeight, params.MinTopicWeight)
	s.Require().NoError(err)

	var evicted, activated *types.EventTopicStatusChanged
	for _, ev := range em.Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		event, ok := msg.(*types.EventTopicStatusChanged)
		if !ok {
			continue
		}
		switch event.TopicId {
		case lowTopicId:
			evicted = event
		case highTopicId:
			activated = event
		}
	}
	s.Require().NotNil(evicted)
	s.Require().False(evicted.IsActive)
	s.Require().Equal(types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_EVICTED_BY_HIGHER_WEIGHT, evicted.Reason)
	s.Require().Equal(params.MinTopicWeight, evicted.MinWeight)
	// The evicted topic's event carries the stored lowest-weight snapshot the
	// eviction read, not a recompute.
	s.Require().True(storedLowest.Weight.Equal(evicted.Weight), "expected weight %s, got %s", storedLowest.Weight.String(), evicted.Weight.String())
	s.Require().NotNil(activated)
	s.Require().True(activated.IsActive)
	s.Require().Equal(types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING, activated.Reason)
	s.Require().Equal(params.MinTopicWeight, activated.MinWeight)
	s.Require().True(highWeight.Equal(activated.Weight), "expected weight %s, got %s", highWeight.String(), activated.Weight.String())
}

// TestInactivateTopicWithoutMinWeightReset_TopicNotFoundInBlockRemovesAllTopics
// Tests that when a topic is not found in the block's active topics list, an error is returned.
func (s *KeeperTestSuite) TestInactivateTopicWithoutMinWeightReset_TopicNotFoundInBlockRemovesAllTopics() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	block := int64(100)
	topicId1 := uint64(1)
	topicId2 := uint64(2)
	topicId3 := uint64(3)
	nonExistentInBlockTopicId := uint64(999)

	// Set up 3 active topics at the block
	activeTopics := types.TopicIds{
		TopicIds: []uint64{topicId1, topicId2, topicId3},
	}
	err := k.SetBlockToActiveTopics(ctx, block, activeTopics)
	s.Require().NoError(err, "Setting active topics should not fail")

	// Verify initial state - should have 3 topics
	retrievedTopics, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 3, "Should have 3 active topics initially")
	s.Require().Contains(retrievedTopics.TopicIds, topicId1)
	s.Require().Contains(retrievedTopics.TopicIds, topicId2)
	s.Require().Contains(retrievedTopics.TopicIds, topicId3)

	// Create a topic that exists and is marked as active in the system but is NOT in the block's active topics list
	topic := s.MockTopic()
	topic.Id = nonExistentInBlockTopicId
	err = k.SetTopic(ctx, nonExistentInBlockTopicId, topic)
	s.Require().NoError(err, "Setting topic should not fail")

	// Mark topic as active in the system so the function proceeds past early return checks
	err = k.SetTopicToNextPossibleChurningBlock(ctx, nonExistentInBlockTopicId, block)
	s.Require().NoError(err, "Setting topic to next possible churning block should not fail")
	err = k.SetActiveTopics(ctx, nonExistentInBlockTopicId)
	s.Require().NoError(err, "Setting active topics should not fail")
	err = k.SetPreviousTopicWeight(ctx, nonExistentInBlockTopicId, alloraMath.NewDecFromInt64(100))
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Verify the topic is marked as active in the system
	_, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, nonExistentInBlockTopicId)
	s.Require().NoError(err)
	s.Require().True(topicIsActive, "Topic should be marked as active in the system")

	// Verify the topic is NOT in the block's active topics list
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().NotContains(retrievedTopics.TopicIds, nonExistentInBlockTopicId,
		"Topic should NOT be in the block's active topics list")

	// Inactivate topic that is not in the block's active topics list
	err = k.InactivateTopic(ctx, nonExistentInBlockTopicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().Error(err, "InactivateTopic should error when topic not found in block")
	s.Require().ErrorIs(err, types.ErrNotFound, "Error should be ErrNotFound")

	// Verify all existing topics remain unchanged (error prevents modification)
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 3, "All topics should remain unchanged when error occurs")
	s.Require().Contains(retrievedTopics.TopicIds, topicId1, "topicId1 should remain")
	s.Require().Contains(retrievedTopics.TopicIds, topicId2, "topicId2 should remain")
	s.Require().Contains(retrievedTopics.TopicIds, topicId3, "topicId3 should remain")
}

// TestInactivateTopicWithoutMinWeightReset_TopicFoundInBlockRemovesOnlyThatTopic
// Tests that when a topic is found in the block's active topics list, only that topic is removed.
func (s *KeeperTestSuite) TestInactivateTopicWithoutMinWeightReset_TopicFoundInBlockRemovesOnlyThatTopic() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	block := int64(200)
	topicId1 := uint64(10)
	topicId2 := uint64(20)
	topicId3 := uint64(30)

	// Set up 3 active topics at the block
	activeTopics := types.TopicIds{
		TopicIds: []uint64{topicId1, topicId2, topicId3},
	}
	err := k.SetBlockToActiveTopics(ctx, block, activeTopics)
	s.Require().NoError(err)

	// Create every topic in the bucket so its lowest-weight metadata can be recomputed.
	for _, topicId := range activeTopics.TopicIds {
		topic := s.MockTopic()
		topic.Id = topicId
		err = k.SetTopic(ctx, topicId, topic)
		s.Require().NoError(err)
	}

	// Make it active in the system
	err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId2, block)
	s.Require().NoError(err)
	err = k.SetActiveTopics(ctx, topicId2)
	s.Require().NoError(err)

	// Set a topic weight
	err = k.SetPreviousTopicWeight(ctx, topicId2, alloraMath.NewDecFromInt64(100))
	s.Require().NoError(err)

	// Verify initial state
	retrievedTopics, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 3)
	s.Require().Contains(retrievedTopics.TopicIds, topicId2)

	// Inactivate topicId2 which is in the block's active topics list
	err = k.InactivateTopic(ctx, topicId2, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	// Verify only topicId2 was removed, others remain
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Len(retrievedTopics.TopicIds, 2, "Should have 2 topics remaining")
	s.Require().NotContains(retrievedTopics.TopicIds, topicId2, "topicId2 should be removed")
	s.Require().Contains(retrievedTopics.TopicIds, topicId1, "topicId1 should remain")
	s.Require().Contains(retrievedTopics.TopicIds, topicId3, "topicId3 should remain")
}

// TestInactivateTopicWithoutMinWeightReset_EmptyBlockListNoOp
// Tests that inactivating a topic when the block has no active topics returns an error.
func (s *KeeperTestSuite) TestInactivateTopicWithoutMinWeightReset_EmptyBlockListNoOp() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	block := int64(300)
	topicId := uint64(100)

	// Create a topic that exists and is marked as active
	topic := s.MockTopic()
	topic.Id = topicId
	err := k.SetTopic(ctx, topicId, topic)
	s.Require().NoError(err)

	// Make it active in the system
	err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId, block)
	s.Require().NoError(err)
	err = k.SetActiveTopics(ctx, topicId)
	s.Require().NoError(err)

	// Set a topic weight
	err = k.SetPreviousTopicWeight(ctx, topicId, alloraMath.NewDecFromInt64(100))
	s.Require().NoError(err)

	// Verify block has no active topics
	retrievedTopics, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Empty(retrievedTopics.TopicIds, "Block should have no active topics")

	// Inactivate topic when block has no active topics
	err = k.InactivateTopic(ctx, topicId, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().Error(err, "InactivateTopic should error when topic not found in block")
	s.Require().ErrorIs(err, types.ErrNotFound, "Error should be ErrNotFound")

	// Verify block still has no active topics (error prevents modification)
	retrievedTopics, err = k.GetActiveTopicIdsAtBlock(ctx, block)
	s.Require().NoError(err)
	s.Require().Empty(retrievedTopics.TopicIds, "Block should still have no active topics")
}

func (s *KeeperTestSuite) TestActivateTopicEmitsOpenReputerWindowEvent() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	topic := s.MockTopic()
	topic.Id = 9_000_004
	topic.EpochLength = 100
	topic.GroundTruthLag = 130
	nonce := types.Nonce{BlockHeight: 1000}

	err := k.SetTopic(ctx, topic.Id, topic)
	s.Require().NoError(err)
	err = s.NonceKeeper().AddReputerNonce(ctx, topic.Id, &nonce)
	s.Require().NoError(err)

	s.WithBlockHeight(1250)
	eventStart := len(s.Ctx().EventManager().Events())
	err = k.ActivateTopic(s.Ctx(), topic.Id, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	openedEvents := 0
	openedEventName := gogoproto.MessageName(&types.EventReputerSubmissionWindowOpened{}) //nolint:exhaustruct // only the registered type name is needed, not an instance
	for _, event := range s.Ctx().EventManager().Events()[eventStart:] {
		if event.Type == openedEventName {
			openedEvents++
		}
	}
	s.Require().Equal(1, openedEvents)

	err = k.InactivateTopic(s.Ctx(), topic.Id, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)
}

func countReputerSubmissionWindowOpenedEvents(s *KeeperTestSuite, eventStart int) int {
	count := 0
	openedEventName := gogoproto.MessageName(&types.EventReputerSubmissionWindowOpened{}) //nolint:exhaustruct // only the registered type name is needed, not an instance
	for _, event := range s.Ctx().EventManager().Events()[eventStart:] {
		if event.Type == openedEventName {
			count++
		}
	}
	return count
}

// A topic activated on the last block of a reputer submission window must
// still announce the window: admission accepts submissions in that block, so
// the open event cannot stop one block earlier than the accepted range.
func (s *KeeperTestSuite) TestActivateTopicEmitsOpenReputerWindowEventAtWindowEnd() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	topic := s.MockTopic()
	topic.Id = 9_000_005
	topic.EpochLength = 100
	topic.GroundTruthLag = 130
	nonce := types.Nonce{BlockHeight: 1000}

	err := k.SetTopic(ctx, topic.Id, topic)
	s.Require().NoError(err)
	err = s.NonceKeeper().AddReputerNonce(ctx, topic.Id, &nonce)
	s.Require().NoError(err)

	// Window bounds: [1200, 1300] inclusive.
	s.WithBlockHeight(1300)
	eventStart := len(s.Ctx().EventManager().Events())
	err = k.ActivateTopic(s.Ctx(), topic.Id, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)
	s.Require().Equal(1, countReputerSubmissionWindowOpenedEvents(s, eventStart))

	err = k.InactivateTopic(s.Ctx(), topic.Id, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	// Past the window end no event is emitted.
	s.WithBlockHeight(1301)
	eventStart = len(s.Ctx().EventManager().Events())
	err = k.ActivateTopic(s.Ctx(), topic.Id, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)
	s.Require().Equal(0, countReputerSubmissionWindowOpenedEvents(s, eventStart))
}
