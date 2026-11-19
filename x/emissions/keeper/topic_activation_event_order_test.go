package keeper_test

import (
	"bytes"
	"context"
	"errors"

	"cosmossdk.io/collections"
	corestore "cosmossdk.io/core/store"
	cosmosMath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// selectiveFailStoreService delegates to the emissions store and fails writes
// to one collection only after the test arms it. failOnNth is the 1-based
// ordinal of the matching write to fail; earlier matching writes succeed.
type selectiveFailStoreService struct {
	corestore.KVStoreService
	prefix        []byte
	err           error
	enabled       bool
	failOnNth     int
	matchedWrites int
	failedWrites  int
}

func (s *selectiveFailStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return &selectiveFailStore{
		KVStore: s.KVStoreService.OpenKVStore(ctx),
		service: s,
	}
}

// failMatchedWrite decides one write to the watched collection: it returns the
// injected error for the failOnNth matching write, and nil for every other one.
func (s *selectiveFailStoreService) failMatchedWrite() error {
	nth := s.failOnNth
	if nth <= 0 {
		nth = 1
	}
	s.matchedWrites++
	if s.matchedWrites == nth {
		s.failedWrites++
		return s.err
	}
	return nil
}

type selectiveFailStore struct {
	corestore.KVStore
	service *selectiveFailStoreService
}

func (s *selectiveFailStore) Set(key, value []byte) error {
	if s.service.enabled && bytes.HasPrefix(key, s.service.prefix) {
		if err := s.service.failMatchedWrite(); err != nil {
			return err
		}
	}
	return s.KVStore.Set(key, value)
}

func (s *KeeperTestSuite) topicKeeperWithFailingWrites(prefix collections.Prefix, injectedErr error) (*keeper.TopicKeeper, *selectiveFailStoreService) {
	storeService := &selectiveFailStoreService{
		KVStoreService: s.StoreServiceEmissions(),
		prefix:         prefix.Bytes(),
		err:            injectedErr,
		enabled:        false,
		failOnNth:      0,
		matchedWrites:  0,
		failedWrites:   0,
	}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	schemaBuilder := collections.NewSchemaBuilder(storeService)
	topicKeeper := keeper.NewTopicKeeper(cdc, schemaBuilder, s.ParamsKeeper(), s.NonceKeeper(), s.StakingKeeper())
	return topicKeeper, storeService
}

// failedActivationScenario describes which write of an activation or reactivation attempt
// fails, and which path the attempt takes.
type failedActivationScenario struct {
	// prefix of the collection whose write fails
	prefix collections.Prefix
	// failOnNth is the 1-based ordinal of the matching write to fail; the first
	// matching write fails when zero
	failOnNth int
	// withEviction exercises the reactivation path that first evicts a lower-weight
	// incumbent from a full active set
	withEviction bool
	// expectScheduleAdvanced asserts the topic's schedule moved to its new epoch end;
	// when false, the schedule keeps its previous value or stays unset
	expectScheduleAdvanced bool
}

func (s *KeeperTestSuite) assertFailedActivationEmitsNoStatusEvents(scenario failedActivationScenario) {
	ctx := s.Ctx()
	params := types.DefaultParams()
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	if scenario.withEviction {
		params.MaxActiveTopicsPerBlock = 1
	}
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	epochLength := int64(100)
	challengerId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	weight := alloraMath.MustNewDecFromString("0.4")
	previousSchedule := int64(0)
	if scenario.withEviction {
		incumbentId := s.CreateTopic(testutil.WithEpochLength(2*epochLength), testutil.WithWorkerSubmissionWindow(2*epochLength))
		s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, incumbentId, cosmosMath.NewInt(100)))
		s.Require().NoError(s.TopicKeeper().ActivateTopic(
			ctx,
			incumbentId,
			types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED,
			alloraMath.ZeroDec(),
			alloraMath.ZeroDec(),
		))

		s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, challengerId, cosmosMath.NewInt(200)))
		s.Require().NoError(s.TopicKeeper().ActivateTopic(
			ctx,
			challengerId,
			types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED,
			alloraMath.ZeroDec(),
			alloraMath.ZeroDec(),
		))
		previousSchedule = ctx.BlockHeight() + epochLength
		var err error
		weight, err = s.TopicKeeper().GetTopicWeightFromTopicId(ctx, challengerId)
		s.Require().NoError(err)
		s.WithBlockHeight(epochLength)
		ctx = s.Ctx()
	}

	challenger, err := s.TopicKeeper().GetTopic(ctx, challengerId)
	s.Require().NoError(err)
	epochEndBlock := ctx.BlockHeight() + challenger.EpochLength
	previousLowest, previouslyNoPrior, err := s.TopicKeeper().GetLowestActiveTopicWeightAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	// With an eviction the incumbent is already active at the challenger's new
	// epoch end block, so a lowest weight exists.
	if scenario.withEviction {
		s.Require().False(previouslyNoPrior)
	} else {
		s.Require().True(previouslyNoPrior)
	}

	injectedErr := errors.New("injected store write failure")
	topicKeeper, storeService := s.topicKeeperWithFailingWrites(scenario.prefix, injectedErr)
	storeService.enabled = true
	storeService.failOnNth = scenario.failOnNth
	eventManager := sdk.NewEventManager()
	activationCtx := ctx.WithEventManager(eventManager)
	if scenario.withEviction {
		err = topicKeeper.AttemptTopicReactivation(activationCtx, challengerId, weight, params.MinTopicWeight)
	} else {
		err = topicKeeper.ActivateTopic(
			activationCtx,
			challengerId,
			types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING,
			weight,
			params.MinTopicWeight,
		)
	}
	s.Require().ErrorIs(err, injectedErr)
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(topicStatusChangedEvents(eventManager))

	// Event deferral does not alter the writes performed by the activation attempt.
	activeAtBlock, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{challengerId}, activeAtBlock.TopicIds)
	if scenario.withEviction {
		currentBlockTopics, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, ctx.BlockHeight())
		s.Require().NoError(err)
		s.Require().Equal([]uint64{challengerId}, currentBlockTopics.TopicIds)
	}
	if scenario.expectScheduleAdvanced {
		schedule, err := topicKeeper.GetTopicSchedule(ctx, challengerId)
		s.Require().NoError(err)
		s.Require().Equal(epochEndBlock, schedule)
	} else if scenario.withEviction {
		schedule, err := topicKeeper.GetTopicSchedule(ctx, challengerId)
		s.Require().NoError(err)
		s.Require().Equal(previousSchedule, schedule)
	} else {
		scheduled, err := topicKeeper.IsTopicScheduled(ctx, challengerId)
		s.Require().NoError(err)
		s.Require().False(scheduled)
	}
	lowest, noPrior, err := topicKeeper.GetLowestActiveTopicWeightAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	if previouslyNoPrior {
		s.Require().True(noPrior)
	} else {
		s.Require().False(noPrior)
		s.Require().Equal(previousLowest.TopicId, lowest.TopicId)
		s.Require().True(previousLowest.Weight.Equal(lowest.Weight))
	}
}

func (s *KeeperTestSuite) TestActivationEmitsNoStatusEventWhenScheduleWriteFails() {
	s.assertFailedActivationEmitsNoStatusEvents(failedActivationScenario{
		prefix:                 types.TopicToNextPossibleChurningBlockKey,
		failOnNth:              1,
		withEviction:           false,
		expectScheduleAdvanced: false,
	})
}

func (s *KeeperTestSuite) TestActivationEmitsNoStatusEventWhenLowestWeightWriteFails() {
	s.assertFailedActivationEmitsNoStatusEvents(failedActivationScenario{
		prefix:                 types.BlockToLowestActiveTopicWeightKey,
		failOnNth:              1,
		withEviction:           false,
		expectScheduleAdvanced: true,
	})
}

func (s *KeeperTestSuite) TestReactivationEvictionEmitsNoStatusEventsWhenScheduleWriteFails() {
	s.assertFailedActivationEmitsNoStatusEvents(failedActivationScenario{
		prefix:                 types.TopicToNextPossibleChurningBlockKey,
		failOnNth:              1,
		withEviction:           true,
		expectScheduleAdvanced: false,
	})
}

func (s *KeeperTestSuite) TestReactivationEvictionEmitsNoStatusEventsWhenLowestWeightWriteFails() {
	s.assertFailedActivationEmitsNoStatusEvents(failedActivationScenario{
		prefix:                 types.BlockToLowestActiveTopicWeightKey,
		failOnNth:              1,
		withEviction:           true,
		expectScheduleAdvanced: true,
	})
}

// If the write that marks the topic active fails, the activation event is not
// emitted: the event must never report a topic as active when the activation's
// writes did not all commit.
func (s *KeeperTestSuite) TestActivateTopicEmitsNoStatusEventWhenActiveSetWriteFails() {
	ctx := s.Ctx()
	params := types.DefaultParams()
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	epochLength := int64(100)
	topicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	weight := alloraMath.MustNewDecFromString("0.4")

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	epochEndBlock := ctx.BlockHeight() + topic.EpochLength

	injectedErr := errors.New("injected store write failure")
	topicKeeper, storeService := s.topicKeeperWithFailingWrites(types.ActiveTopicsKey, injectedErr)
	storeService.enabled = true
	eventManager := sdk.NewEventManager()
	err = topicKeeper.ActivateTopic(
		ctx.WithEventManager(eventManager),
		topicId,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING,
		weight,
		params.MinTopicWeight,
	)
	s.Require().ErrorIs(err, injectedErr)
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(topicStatusChangedEvents(eventManager))

	// Event deferral does not alter the writes performed by the activation.
	schedule, err := topicKeeper.GetTopicSchedule(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(epochEndBlock, schedule)
	activeAtBlock, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{topicId}, activeAtBlock.TopicIds)
	// Activity is schedule-derived, so the failed write is visible in the
	// export-compat active topic set only.
	activeTopicIds, err := topicKeeper.GetActiveTopicIds(ctx)
	s.Require().NoError(err)
	s.Require().Empty(activeTopicIds)
}

// If the final previous-weight total update fails, the activation event is not
// emitted even though the topic's scheduling and active-set writes succeeded.
func (s *KeeperTestSuite) TestActivateTopicEmitsNoStatusEventWhenPreviousWeightTotalWriteFails() {
	ctx := s.Ctx()
	params := types.DefaultParams()
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	epochLength := int64(100)
	topicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	weight := alloraMath.MustNewDecFromString("0.4")
	previousWeight := alloraMath.NewDecFromInt64(100)
	s.Require().NoError(s.TopicKeeper().SetPreviousTopicWeight(ctx, topicId, previousWeight))
	// An unscheduled topic's stored weight is added to the total by activation.
	s.Require().NoError(s.TopicKeeper().SetTotalSumPreviousTopicWeights(ctx, alloraMath.ZeroDec()))

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	epochEndBlock := ctx.BlockHeight() + topic.EpochLength

	injectedErr := errors.New("injected store write failure")
	topicKeeper, storeService := s.topicKeeperWithFailingWrites(types.TotalSumPreviousTopicWeightsKey, injectedErr)
	storeService.enabled = true
	eventManager := sdk.NewEventManager()
	err = topicKeeper.ActivateTopic(
		ctx.WithEventManager(eventManager),
		topicId,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING,
		weight,
		params.MinTopicWeight,
	)
	s.Require().ErrorIs(err, injectedErr)
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(topicStatusChangedEvents(eventManager))

	// Event deferral does not alter the writes performed before the failed total update.
	schedule, err := topicKeeper.GetTopicSchedule(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(epochEndBlock, schedule)
	activeAtBlock, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{topicId}, activeAtBlock.TopicIds)
	activeTopicIds, err := topicKeeper.GetActiveTopicIds(ctx)
	s.Require().NoError(err)
	s.Require().Contains(activeTopicIds, topicId)
	storedPreviousWeight, noPrior, err := topicKeeper.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(noPrior)
	s.Require().True(previousWeight.Equal(storedPreviousWeight))
	total, err := topicKeeper.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(total.IsZero(), "the failed total update must leave the previous total untouched")
}

// If dropping a reactivated topic's stale listing during ActivateTopic fails,
// no activation event is emitted.
func (s *KeeperTestSuite) TestActivateTopicEmitsNoStatusEventWhenStaleListingWriteFails() {
	ctx := s.Ctx()
	params := types.DefaultParams()
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	epochLength := int64(20)
	topicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	s.Require().NoError(s.TopicKeeper().ActivateTopic(
		ctx,
		topicId,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED,
		alloraMath.ZeroDec(),
		alloraMath.ZeroDec(),
	))
	previousWeight := alloraMath.NewDecFromInt64(100)
	s.Require().NoError(s.TopicKeeper().SetPreviousTopicWeight(ctx, topicId, previousWeight))
	staleBlock, err := s.TopicKeeper().GetTopicSchedule(ctx, topicId)
	s.Require().NoError(err)

	// A past schedule makes the topic inactive without removing its old listing.
	s.WithBlockHeight(staleBlock + 10)
	ctx = s.Ctx()
	epochEndBlock := ctx.BlockHeight() + epochLength

	injectedErr := errors.New("injected store write failure")
	topicKeeper, storeService := s.topicKeeperWithFailingWrites(types.BlockToActiveTopicsKey, injectedErr)
	// The new epoch-end listing is written first; fail removal of the old listing.
	storeService.enabled = true
	storeService.failOnNth = 2
	eventManager := sdk.NewEventManager()
	err = topicKeeper.ActivateTopic(
		ctx.WithEventManager(eventManager),
		topicId,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING,
		alloraMath.ZeroDec(),
		params.MinTopicWeight,
	)
	s.Require().ErrorIs(err, injectedErr)
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(topicStatusChangedEvents(eventManager))

	// Event deferral does not alter the writes performed before the stale-listing update.
	schedule, err := topicKeeper.GetTopicSchedule(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(epochEndBlock, schedule)
	activeAtBlock, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{topicId}, activeAtBlock.TopicIds)
	staleListing, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, staleBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{topicId}, staleListing.TopicIds)
	activeTopicIds, err := topicKeeper.GetActiveTopicIds(ctx)
	s.Require().NoError(err)
	s.Require().Contains(activeTopicIds, topicId)
	total, err := topicKeeper.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(previousWeight.Equal(total), "an already-scheduled topic's weight must not be added again")
}

// If dropping the reactivated topic's stale listing from its previous churning
// block fails, no activation event is emitted.
func (s *KeeperTestSuite) TestReactivationEmitsNoStatusEventWhenStaleListingWriteFails() {
	ctx := s.Ctx()
	params := types.DefaultParams()
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	epochLength := int64(100)
	topicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, topicId, cosmosMath.NewInt(100)))
	s.Require().NoError(s.TopicKeeper().ActivateTopic(
		ctx,
		topicId,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED,
		alloraMath.ZeroDec(),
		alloraMath.ZeroDec(),
	))
	s.WithBlockHeight(epochLength)
	ctx = s.Ctx()

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	epochEndBlock := ctx.BlockHeight() + topic.EpochLength
	weight, err := s.TopicKeeper().GetTopicWeightFromTopicId(ctx, topicId)
	s.Require().NoError(err)

	injectedErr := errors.New("injected store write failure")
	topicKeeper, storeService := s.topicKeeperWithFailingWrites(types.BlockToActiveTopicsKey, injectedErr)
	// The listing at the new epoch end block is written first; fail the second
	// write to the collection, which is the stale listing's removal at the
	// previous churning block.
	storeService.enabled = true
	storeService.failOnNth = 2
	eventManager := sdk.NewEventManager()
	err = topicKeeper.AttemptTopicReactivation(ctx.WithEventManager(eventManager), topicId, weight, params.MinTopicWeight)
	s.Require().ErrorIs(err, injectedErr)
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(topicStatusChangedEvents(eventManager))

	// Event deferral does not alter the writes performed by the reactivation.
	schedule, err := topicKeeper.GetTopicSchedule(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(epochEndBlock, schedule)
	activeAtBlock, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, epochEndBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{topicId}, activeAtBlock.TopicIds)
	active, err := topicKeeper.IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(active)
	staleListing, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, ctx.BlockHeight())
	s.Require().NoError(err)
	s.Require().Equal([]uint64{topicId}, staleListing.TopicIds)
}

// If the weight reset that follows an inactivation fails, the deactivation
// event is not emitted.
func (s *KeeperTestSuite) TestInactivateTopicEmitsNoStatusEventWhenResetWriteFails() {
	ctx := s.Ctx()
	params := types.DefaultParams()
	// Both topics must be active at the same churning block: the default limit
	// of one would make the second activation evict the first.
	params.MaxActiveTopicsPerBlock = 2
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	// Two topics share the churning block so the post-inactivation weight reset
	// rewrites the block's lowest weight instead of pruning it.
	epochLength := int64(100)
	inactivatedId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	remainingId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	for _, topicId := range []uint64{inactivatedId, remainingId} {
		s.Require().NoError(s.TopicKeeper().ActivateTopic(
			ctx,
			topicId,
			types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED,
			alloraMath.ZeroDec(),
			alloraMath.ZeroDec(),
		))
	}

	churningBlock, wasActive, err := s.TopicKeeper().GetNextPossibleChurningBlockByTopicId(ctx, inactivatedId)
	s.Require().NoError(err)
	s.Require().True(wasActive)

	injectedErr := errors.New("injected store write failure")
	topicKeeper, storeService := s.topicKeeperWithFailingWrites(types.BlockToLowestActiveTopicWeightKey, injectedErr)
	storeService.enabled = true
	eventManager := sdk.NewEventManager()
	err = topicKeeper.InactivateTopic(
		ctx.WithEventManager(eventManager),
		inactivatedId,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_WEIGHT_BELOW_MINIMUM,
		alloraMath.ZeroDec(),
		params.MinTopicWeight,
		params.MinTopicWeight,
	)
	s.Require().ErrorIs(err, injectedErr)
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(topicStatusChangedEvents(eventManager))

	// Event deferral does not alter the writes performed by the inactivation.
	inactive, err := topicKeeper.IsTopicActive(ctx, inactivatedId)
	s.Require().NoError(err)
	s.Require().False(inactive)
	scheduled, err := topicKeeper.IsTopicScheduled(ctx, inactivatedId)
	s.Require().NoError(err)
	s.Require().False(scheduled)
	activeAtBlock, err := topicKeeper.GetActiveTopicIdsAtBlock(ctx, churningBlock)
	s.Require().NoError(err)
	s.Require().Equal([]uint64{remainingId}, activeAtBlock.TopicIds)
}
