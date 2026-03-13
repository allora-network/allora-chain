package actorutils_test

import (
	"testing"

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
	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
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
	err = s.EmissionsKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce)
	s.Require().NoError(err)

	// MULTI: ensure epoch label registry exists for this nonce (and has labels)
	// (Pick whatever labels your tests commonly use)
	_, err = s.EmissionsKeeper().RegisterEpochLabel(s.Ctx(), topicId, nonce.BlockHeight, "a")
	s.Require().NoError(err)
	_, err = s.EmissionsKeeper().RegisterEpochLabel(s.Ctx(), topicId, nonce.BlockHeight, "b")
	s.Require().NoError(err)
	_, err = s.EmissionsKeeper().RegisterEpochLabel(s.Ctx(), topicId, nonce.BlockHeight, "c")
	s.Require().NoError(err)

	reg, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, nonce.BlockHeight)
	s.Require().NoError(err)
	s.Require().NotEmpty(reg.GetLabels())

	// Create and insert inferences (Values length must match registry length)
	inf0 := types.Inference{
		Inferer:     worker0,
		TopicId:     topicId,
		BlockHeight: blockHeight,
		ExtraData:   nil,
		Proof:       "",
		Values: []alloraMath.Dec{
			alloraMath.MustNewDecFromString("-0.035995138925040600"),
			alloraMath.MustNewDecFromString("0.100000000000000000"),
			alloraMath.MustNewDecFromString("0.200000000000000000"),
		},
	}
	inf1 := types.Inference{
		Inferer:     worker1,
		TopicId:     topicId,
		BlockHeight: blockHeight,
		ExtraData:   nil,
		Proof:       "",
		Values: []alloraMath.Dec{
			alloraMath.MustNewDecFromString("-0.07333303938740420"),
			alloraMath.MustNewDecFromString("0.110000000000000000"),
			alloraMath.MustNewDecFromString("0.210000000000000000"),
		},
	}

	s.Require().Equal(len(reg.GetLabels()), len(inf0.Values))
	s.Require().Equal(len(reg.GetLabels()), len(inf1.Values))

	err = s.EmissionsKeeper().InsertInference(s.Ctx(), topicId, inf0)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().InsertInference(s.Ctx(), topicId, inf1)
	s.Require().NoError(err)

	// Artificially add the workers as active inferers
	err = s.EmissionsKeeper().AddActiveInferer(s.Ctx(), topicId, worker0)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().AddActiveInferer(s.Ctx(), topicId, worker1)
	s.Require().NoError(err)

	// ------------------------------------------------------------------------------------------------
	// Move the blockheight until end of wsw
	// ------------------------------------------------------------------------------------------------
	s.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)

	// Test closing the worker nonce
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce)
	s.Require().NoError(err)

	// Verify nonce is no longer unfulfilled
	isUnfulfilled, err := s.EmissionsKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topicId, &nonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "Nonce should no longer be unfulfilled")

	// Verify network inferences were created
	networkInferences, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().NotNil(networkInferences, "Network inferences should exist")

	// Verify outlier resistant network inferences were created
	outlierResistantInferences, err := s.EmissionsKeeper().GetOutlierResistantNetworkInferences(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().NotNil(outlierResistantInferences, "Outlier resistant network inferences should exist")
}

func (s *WorkerTestSuite) TestCloseWorkerNonce_Multi_DerivesRegistryFromFinalActiveSet() {
	blockHeight := int64(200)
	s.WithBlockHeight(blockHeight)

	topicId := s.CreateTopic(testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI))
	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	nonce := types.Nonce{BlockHeight: blockHeight}
	s.Require().NoError(s.EmissionsKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce))

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	params.MaxTopInferersToReward = 2
	s.Require().NoError(s.EmissionsKeeper().SetParams(s.Ctx(), params))

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)

	s.Require().NoError(s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, worker0, types.Score{
		TopicId:     topicId,
		Address:     worker0,
		BlockHeight: blockHeight - 1,
		Score:       alloraMath.MustNewDecFromString("90"),
	}))
	s.Require().NoError(s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, worker1, types.Score{
		TopicId:     topicId,
		Address:     worker1,
		BlockHeight: blockHeight - 1,
		Score:       alloraMath.MustNewDecFromString("95"),
	}))
	s.Require().NoError(s.EmissionsKeeper().SetInfererScoreEma(s.Ctx(), topicId, worker2, types.Score{
		TopicId:     topicId,
		Address:     worker2,
		BlockHeight: blockHeight - 1,
		Score:       alloraMath.MustNewDecFromString("99"),
	}))

	submit := func(worker string, labels []string, values []string) {
		s.Require().Equal(len(labels), len(values))
		labeled := make([]*types.InputLabeledValue, 0, len(labels))
		for i := range labels {
			labeled = append(labeled, &types.InputLabeledValue{
				Label: labels[i],
				Value: alloraMath.MustNewBoundedExp40DecFromString(values[i]),
			})
		}
		raw := types.InputInference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     worker,
			Value:       alloraMath.MustNewBoundedExp40DecFromString("0"),
			ExtraData:   nil,
			Proof:       "",
			Values:      labeled,
		}
		normalized, err := s.EmissionsKeeper().NormalizeInputInference(s.Ctx(), topic, blockHeight, &raw)
		s.Require().NoError(err)
		s.Require().NoError(s.EmissionsKeeper().SetWorkerLatestInputInference(s.Ctx(), topicId, blockHeight, raw))
		s.Require().NoError(s.EmissionsKeeper().AppendInference(s.Ctx(), topic, blockHeight, normalized, params.MaxTopInferersToReward))
	}

	submit(worker0, []string{"a", "b"}, []string{"1", "2"})
	submit(worker1, []string{"b", "c"}, []string{"10", "20"})
	submit(worker2, []string{"c", "d"}, []string{"30", "40"})

	s.WithBlockHeight(blockHeight + topic.WorkerSubmissionWindow)
	s.Require().NoError(actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce))

	reg, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().Len(reg.Labels, 3)
	s.Require().Equal("b", reg.Labels[0].Name)
	s.Require().Equal("c", reg.Labels[1].Name)
	s.Require().Equal("d", reg.Labels[2].Name)

	inferences, err := s.EmissionsKeeper().GetInferencesAtBlock(s.Ctx(), topicId, blockHeight, false)
	s.Require().NoError(err)
	s.Require().Len(inferences.Inferences, 2)

	byInferer := make(map[string][]string, 2)
	for _, inf := range inferences.Inferences {
		gotVals := make([]string, 0, len(inf.Values))
		for _, v := range inf.Values {
			gotVals = append(gotVals, v.String())
		}
		byInferer[inf.Inferer] = gotVals
	}
	s.Require().Equal([]string{"10", "20", "0"}, byInferer[worker1])
	s.Require().Equal([]string{"0", "30", "40"}, byInferer[worker2])
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
	}
	res, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), newTopicMsg)
	s.Require().NoError(err)
	topicId := res.TopicId

	// Get the topic
	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Test 1: Closing without valid nonce
	nonce := types.Nonce{BlockHeight: blockHeight}
	err = actorutils.CloseWorkerNonce(s.EmissionsKeeper(), s.Ctx(), topic, nonce)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrUnfulfilledNonceNotFound)

	// Test 2: Closing without active inferers
	topic.EpochLastEnded = blockHeight - 100 // Fix the window
	err = s.EmissionsKeeper().AddWorkerNonce(s.Ctx(), topicId, &nonce)
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

	_, err = s.EmissionsKeeper().RegisterEpochLabel(ctx, topicId, blockHeight, "y")
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
	err = keeper.SetLastMedianInferences(ctx, topicId, lastMedian)
	require.NoError(err)
	err = keeper.SetMadInferences(ctx, topicId, mad)
	require.NoError(err)

	// Set the outlier threshold in params
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	params.InferenceOutlierDetectionThreshold = alloraMath.MustNewDecFromString("3.0") // 3 * MAD threshold
	err = keeper.SetParams(ctx, params)
	require.NoError(err)

	// Call the function we're testing
	err = actorutils.ProcessAndStoreNetworkInferences(keeper, ctx, topicId, blockHeight, inferences, forecasts)
	require.NoError(err)

	// Retrieve both regular and outlier-resistant network inferences
	regularInferences, err := keeper.GetNetworkInferences(ctx, topicId, blockHeight)
	require.NoError(err)
	outlierResistantInferences, err := keeper.GetOutlierResistantNetworkInferences(ctx, topicId, blockHeight)
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
	err = keeper.SetLastMedianInferences(ctx, topicId, lastMedian)
	require.NoError(err)
	err = keeper.SetMadInferences(ctx, topicId, mad)
	require.NoError(err)

	// Set the outlier threshold in params (same as other test)
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	params.InferenceOutlierDetectionThreshold = alloraMath.MustNewDecFromString("3.0") // 3 * MAD threshold
	err = keeper.SetParams(ctx, params)
	require.NoError(err)
	_, err = s.EmissionsKeeper().RegisterEpochLabel(ctx, topicId, blockHeight, "y")
	require.NoError(err)

	// Call the function we're testing
	err = actorutils.ProcessAndStoreNetworkInferences(keeper, ctx, topicId, blockHeight, inferences, forecasts)
	require.NoError(err)
	// Retrieve both regular and outlier-resistant network inferences
	regularInferences, err := keeper.GetNetworkInferences(ctx, topicId, blockHeight)
	require.NoError(err)
	outlierResistantInferences, err := keeper.GetOutlierResistantNetworkInferences(ctx, topicId, blockHeight)
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
