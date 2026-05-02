package actorutils_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type WorkerTestSuite struct {
	testutil.TestSuite
}

func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, &WorkerTestSuite{
		testutil.NewTestSuite("actor_utils_worker"),
	})
}

func (s *WorkerTestSuite) TestCloseWorkerNonce_Multi() {
	// Create a MULTI topic
	blockHeight := int64(101)
	s.WithBlockHeight(blockHeight)

	// Create topic using MsgServer like in rewards_test.go
	topicId := s.CreateTopic(testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI))

	// Get the topic
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Register workers using MsgServer
	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)

	workerRegMsg0 := &types.RegisterRequest{
		Sender:    worker0,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     s.AddrsStr(4),
	}
	_, err = s.EmissionsMsgServer().Register(s.Ctx(), workerRegMsg0)
	s.Require().NoError(err)

	workerRegMsg1 := &types.RegisterRequest{
		Sender:    worker1,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     s.AddrsStr(4),
	}
	_, err = s.EmissionsMsgServer().Register(s.Ctx(), workerRegMsg1)
	s.Require().NoError(err)

	// Add worker nonce
	nonce := types.Nonce{BlockHeight: blockHeight}
	err = s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce)
	s.Require().NoError(err)

	// v2: stage raw InputInference per worker. The registry is materialized
	// at CloseWorkerNonce time from the final active workers' staged inputs.
	mustBounded := func(x string) alloraMath.BoundedExp40Dec {
		return alloraMath.MustNewBoundedExp40DecFromString(x)
	}
	input0 := types.InputInference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Inferer:     worker0,
		ExtraData:   nil,
		Proof:       "",
		Value:       mustBounded("0"),
		Values: []*types.InputLabeledValue{
			{Label: "a", Value: mustBounded("-0.035995138925040600")},
			{Label: "b", Value: mustBounded("0.100000000000000000")},
			{Label: "c", Value: mustBounded("0.200000000000000000")},
		},
	}
	input1 := types.InputInference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Inferer:     worker1,
		ExtraData:   nil,
		Proof:       "",
		Value:       mustBounded("0"),
		Values: []*types.InputLabeledValue{
			{Label: "a", Value: mustBounded("-0.07333303938740420")},
			{Label: "b", Value: mustBounded("0.110000000000000000")},
			{Label: "c", Value: mustBounded("0.210000000000000000")},
		},
	}

	err = s.WorkerKeeper().SetWorkerLatestInputInference(s.Ctx(), topicId, worker0, input0)
	s.Require().NoError(err)
	err = s.WorkerKeeper().SetWorkerLatestInputInference(s.Ctx(), topicId, worker1, input1)
	s.Require().NoError(err)

	err = s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, worker0)
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, worker1)
	s.Require().NoError(err)

	// ------------------------------------------------------------------------------------------------
	// Move the blockheight until end of wsw
	// ------------------------------------------------------------------------------------------------
	s.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)

	// Test closing the worker nonce
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce)
	s.Require().NoError(err)

	// Verify nonce is no longer unfulfilled
	isUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topicId, &nonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "Nonce should no longer be unfulfilled")

	// Verify network inferences were created
	networkInferences, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().NotNil(networkInferences, "Network inferences should exist")

	// Verify outlier resistant network inferences were created
	outlierResistantInferences, err := s.EmissionsKeeper().GetOutlierResistantNetworkInferences(s.Ctx(), topic, blockHeight)
	s.Require().ErrorContains(err, "outlier resistant network inferences are not supported for multi-label topics")
	s.Require().Nil(outlierResistantInferences, "Outlier resistant network inferences should not exist")
}

func (s *WorkerTestSuite) TestCloseWorkerNonceFailures() {
	blockHeight := int64(101)
	s.WithBlockHeight(blockHeight)

	// Create topic using MsgServer
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.AddrsStr(0),
		Metadata:                 "test",
		LossMethod:               "mse",
		AllowNegative:            false,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
	}
	res, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId

	// Get the topic
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Test 1: Closing without valid nonce
	nonce := types.Nonce{BlockHeight: blockHeight}
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrUnfulfilledNonceNotFound)

	// Test 2: Closing without active inferers
	topic.EpochLastEnded = blockHeight - 100 // Fix the window
	err = s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce)
	s.Require().NoError(err)

	// Move to end of worker submission window
	s.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrNoQualifiedInferers)
}

func (s *WorkerTestSuite) TestProcessAndStoreNetworkInferencesCatchesOutliers() {
	ctx := s.Ctx()
	keeper := s.EmissionsKeeper()
	require := s.Require()

	// Create topic using MsgServer
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.AddrsStr(0),
		Metadata:                 "test",
		LossMethod:               "mse",
		AllowNegative:            false,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
	}
	res, err := s.EmissionsMsgServer().CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err)
	topicId := res.TopicId
	blockHeight := int64(100)

	// Set up workers/forecasters using existing suite addresses
	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	forecaster0 := s.AddrsStr(4)
	forecaster1 := s.AddrsStr(5)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	_, err = s.TopicKeeper().BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, blockHeight, nil)
	require.NoError(err)

	// Create inferences where worker3 is an obvious outlier
	inferences := &types.Inferences{
		Inferences: []*types.Inference{
			{Inferer: worker0, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.0")}, TopicId: topicId, BlockHeight: blockHeight},
			{Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.1")}, TopicId: topicId, BlockHeight: blockHeight},
			{Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.9")}, TopicId: topicId, BlockHeight: blockHeight},
			{Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("100.0")}, TopicId: topicId, BlockHeight: blockHeight}, // outlier
		},
	}

	// Set up forecasts
	forecasts := &types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("1.05")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("1.15")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.95")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("9.5")},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("0.95")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("1.05")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.85")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("10.5")},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	// Set up the last median and MAD for outlier detection
	lastMedian := alloraMath.MustNewDecFromString("1.0")
	mad := alloraMath.MustNewDecFromString("0.2")
	err = s.TopicKeeper().SetLastMedianInferences(ctx, topicId, lastMedian)
	require.NoError(err)
	err = s.TopicKeeper().SetMadInferences(ctx, topicId, mad)
	require.NoError(err)

	// Set the outlier threshold in params
	params, err := s.ParamsKeeper().GetParams(ctx)
	require.NoError(err)
	params.InferenceOutlierDetectionThreshold = alloraMath.MustNewDecFromString("3.0") // 3 * MAD threshold
	err = s.ParamsKeeper().SetParams(ctx, params)
	require.NoError(err)

	topic, err := keeper.GetTopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)

	// Call the function we're testing
	err = actorutils.ProcessAndStoreNetworkInferences(keeper, ctx, topic, blockHeight, inferences, forecasts)
	require.NoError(err)

	// Retrieve both regular and outlier-resistant network inferences
	regularInferences, err := keeper.GetNetworkInferences(ctx, topicId, blockHeight)
	require.NoError(err)

	outlierResistantInferences, err := keeper.GetOutlierResistantNetworkInferences(ctx, topic, blockHeight)
	require.NoError(err)

	// Regular network inferences should include all values
	require.Len(regularInferences.InfererValues, 4)

	// Outlier resistant network inferences should exclude worker3
	require.Len(outlierResistantInferences.InfererValues, 3)

	// Verify worker3 is not in outlier resistant results
	for _, infValue := range outlierResistantInferences.InfererValues {
		require.NotEqual(worker3, infValue.Worker, "Outlier should not be present in outlier-resistant results")
	}

	regularVal := regularInferences.CombinedValue[0].Value
	outlierVal := outlierResistantInferences.CombinedValue[0].Value

	// Verify the values are different between regular and outlier-resistant
	require.False(regularVal.Equal(outlierVal), "Regular and outlier-resistant combined values should differ")

	// Verify the outlier-resistant combined value is closer to the median
	regularDiff, err := regularVal.Sub(lastMedian)
	require.NoError(err)
	regularDiffAbs, err := regularDiff.Abs()
	require.NoError(err)

	outlierDiff, err := outlierVal.Sub(lastMedian)
	require.NoError(err)
	outlierDiffAbs, err := outlierDiff.Abs()
	require.NoError(err)

	require.True(
		outlierDiffAbs.Lt(regularDiffAbs),
		"Outlier-resistant value should be closer to the median",
	)
}

func (s *WorkerTestSuite) TestProcessAndStoreNetworkInferencesNoOutliers() {
	ctx := s.Ctx()
	keeper := s.EmissionsKeeper()
	require := s.Require()

	// Create topic using MsgServer
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.AddrsStr(0),
		Metadata:                 "test",
		LossMethod:               "mse",
		AllowNegative:            false,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   10,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                types.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   types.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
	}
	res, err := s.EmissionsMsgServer().CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err)
	topicId := res.TopicId
	blockHeight := int64(100)

	// Set up workers/forecasters using existing suite addresses
	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	forecaster0 := s.AddrsStr(4)
	forecaster1 := s.AddrsStr(5)

	// Create inferences where all values are within normal bounds
	inferences := &types.Inferences{
		Inferences: []*types.Inference{
			{Inferer: worker0, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.0")}, TopicId: topicId, BlockHeight: blockHeight},
			{Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.1")}, TopicId: topicId, BlockHeight: blockHeight},
			{Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.9")}, TopicId: topicId, BlockHeight: blockHeight},
			{Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.2")}, TopicId: topicId, BlockHeight: blockHeight},
		},
	}

	// Set up forecasts with similarly normal values
	forecasts := &types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("1.05")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("1.15")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.95")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("1.25")},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*types.ForecastElement{
					{Inferer: worker0, Value: alloraMath.MustNewDecFromString("0.95")},
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("1.05")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.85")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("1.15")},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	// Set up the last median and MAD for outlier detection
	lastMedian := alloraMath.MustNewDecFromString("1.0")
	mad := alloraMath.MustNewDecFromString("0.2")
	err = s.TopicKeeper().SetLastMedianInferences(ctx, topicId, lastMedian)
	require.NoError(err)
	err = s.TopicKeeper().SetMadInferences(ctx, topicId, mad)
	require.NoError(err)

	// Set the outlier threshold in params (same as other test)
	params, err := s.ParamsKeeper().GetParams(ctx)
	require.NoError(err)
	params.InferenceOutlierDetectionThreshold = alloraMath.MustNewDecFromString("3.0") // 3 * MAD threshold
	err = s.ParamsKeeper().SetParams(ctx, params)
	require.NoError(err)
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)
	_, err = s.TopicKeeper().BuildFinalEpochLabelRegistryFromActiveSet(ctx, topic, blockHeight, nil)
	require.NoError(err)

	topic, err := keeper.GetTopicKeeper().GetTopic(ctx, topicId)
	require.NoError(err)

	// Call the function we're testing
	err = actorutils.ProcessAndStoreNetworkInferences(keeper, ctx, topic, blockHeight, inferences, forecasts)
	require.NoError(err)
	// Retrieve both regular and outlier-resistant network inferences
	regularInferences, err := keeper.GetNetworkInferences(ctx, topicId, blockHeight)
	require.NoError(err)
	outlierResistantInferences, err := keeper.GetOutlierResistantNetworkInferences(ctx, topic, blockHeight)
	require.NoError(err)

	// Both should include all values
	require.Len(regularInferences.InfererValues, 4)
	require.Len(outlierResistantInferences.InfererValues, 4)

	// Verify all workers are present in both results
	regularWorkers := make(map[string]bool)
	outlierWorkers := make(map[string]bool)

	for _, infValue := range regularInferences.InfererValues {
		regularWorkers[infValue.Worker] = true
	}
	for _, infValue := range outlierResistantInferences.InfererValues {
		outlierWorkers[infValue.Worker] = true
	}

	require.Equal(regularWorkers, outlierWorkers, "Both results should contain the same workers")

	regularVal := regularInferences.CombinedValue[0].Value
	outlierVal := outlierResistantInferences.CombinedValue[0].Value

	// Verify the combined values are equal (within decimal precision)
	require.True(
		regularVal.Equal(outlierVal),
		"Combined values should match when no outliers exist",
	)
}

// countFrozenEvents returns the number of EventEpochLabelRegistryFrozen
// events currently on the given SDK context's event manager, along with the
// RegistrySize field from the last such event (so callers can assert on it).
// It uses sdk.ParseTypedEvent to reconstruct the typed event from its ABCI
// representation, which keeps the helper agnostic to the proto package
// version (e.g. v10 -> v11 bumps require no test changes) and reads
// registry_size as a typed uint64 rather than a stringified attribute.
func countFrozenEvents(ctx sdk.Context) (int, uint64) {
	count := 0
	var lastSize uint64
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		frozen, ok := msg.(*types.EventEpochLabelRegistryFrozen)
		if !ok {
			continue
		}
		count++
		lastSize = frozen.RegistrySize
	}
	return count, lastSize
}

// TestCloseActiveInferencesSet_EmitsEpochLabelRegistryFrozenEventOnce pins
// the close-time contract: exactly one EventEpochLabelRegistryFrozen is
// emitted per successful CloseWorkerNonce, carrying the registry's cardinality
// as the registry_size attribute. A regression that double-emitted (e.g. by
// emitting both pre- and post-write) or skipped emission would be caught here.
func (s *WorkerTestSuite) TestCloseActiveInferencesSet_EmitsEpochLabelRegistryFrozenEventOnce() {
	blockHeight := int64(101)
	s.WithBlockHeight(blockHeight)

	topicId := s.CreateTopic(testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI))
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	for _, w := range []string{worker0, worker1} {
		_, err = s.EmissionsMsgServer().Register(s.Ctx(), &types.RegisterRequest{
			Sender:    w,
			TopicId:   topicId,
			IsReputer: false,
			Owner:     s.AddrsStr(4),
		})
		s.Require().NoError(err)
	}

	nonce := types.Nonce{BlockHeight: blockHeight}
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce))

	mustBounded := func(x string) alloraMath.BoundedExp40Dec {
		return alloraMath.MustNewBoundedExp40DecFromString(x)
	}

	input0 := types.InputInference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Inferer:     worker0,
		ExtraData:   nil,
		Proof:       "",
		Value:       mustBounded("0"),
		Values: []*types.InputLabeledValue{
			{Label: "a", Value: mustBounded("0.1")},
			{Label: "b", Value: mustBounded("0.2")},
			{Label: "c", Value: mustBounded("0.3")},
		},
	}
	input1 := types.InputInference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Inferer:     worker1,
		ExtraData:   nil,
		Proof:       "",
		Value:       mustBounded("0"),
		Values: []*types.InputLabeledValue{
			{Label: "a", Value: mustBounded("0.15")},
			{Label: "b", Value: mustBounded("0.25")},
			{Label: "d", Value: mustBounded("0.35")},
		},
	}
	s.Require().NoError(s.WorkerKeeper().SetWorkerLatestInputInference(s.Ctx(), topicId, worker0, input0))
	s.Require().NoError(s.WorkerKeeper().SetWorkerLatestInputInference(s.Ctx(), topicId, worker1, input1))

	s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, worker0))
	s.Require().NoError(s.WorkerKeeper().AddActiveInferer(s.Ctx(), topicId, worker1))

	s.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)

	before, _ := countFrozenEvents(s.Ctx())

	s.Require().NoError(actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce))

	after, size := countFrozenEvents(s.Ctx())
	s.Require().Equal(before+1, after,
		"CloseWorkerNonce must emit exactly one EventEpochLabelRegistryFrozen")
	s.Require().Equal(uint64(4), size,
		"registry_size must match the union of canonical labels {a,b,c,d}")
}

// TestCloseActiveInferencesSet_NoEventWhenActiveSetEmpty pins the negative
// branch: when the active inferer set is empty at close time, CloseWorkerNonce
// short-circuits with ErrNoQualifiedInferers before reaching the registry
// materializer, and therefore EventEpochLabelRegistryFrozen must NOT be
// emitted. A regression that moved the emission in front of the active-set
// gate would be caught here.
func (s *WorkerTestSuite) TestCloseActiveInferencesSet_NoEventWhenActiveSetEmpty() {
	blockHeight := int64(101)
	s.WithBlockHeight(blockHeight)

	topicId := s.CreateTopic(testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI))
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	nonce := types.Nonce{BlockHeight: blockHeight}
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce))

	s.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)

	before, _ := countFrozenEvents(s.Ctx())

	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce)
	s.Require().ErrorIs(err, types.ErrNoQualifiedInferers)

	after, _ := countFrozenEvents(s.Ctx())
	s.Require().Equal(before, after,
		"EventEpochLabelRegistryFrozen must not be emitted when the active inferer set is empty")
}
