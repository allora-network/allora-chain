package inference_synthesis_test

import (
	"reflect"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/stretchr/testify/assert"

	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// TestMakeMapFromWorkerToTheirWork tests the makeMapFromWorkerToTheirWork function for correctly mapping workers to their inferences.
func TestMakeMapFromWorkerToTheirWork(t *testing.T) {
	tests := []struct {
		name       string
		inferences []*emissionstypes.Inference
		expected   map[string]*emissionstypes.Inference
	}{
		{
			name: "multiple workers",
			inferences: []*emissionstypes.Inference{
				{
					TopicId: 101,
					Inferer: "inferer1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				{
					TopicId: 102,
					Inferer: "inferer2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				{
					TopicId: 103,
					Inferer: "inferer3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
			expected: map[string]*emissionstypes.Inference{
				"inferer1": {
					TopicId: 101,
					Inferer: "inferer1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				"inferer2": {
					TopicId: 102,
					Inferer: "inferer2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				"inferer3": {
					TopicId: 103,
					Inferer: "inferer3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
		},
		{
			name:       "empty list",
			inferences: []*emissionstypes.Inference{},
			expected:   map[string]*emissionstypes.Inference{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := inferencesynthesis.MakeMapFromInfererToTheirInference(tc.inferences)
			assert.True(t, reflect.DeepEqual(result, tc.expected), "Expected and actual maps should be equal")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlock() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[302]
	epoch3Get := epochGet[303]

	require := s.Require()
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)

	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeightPreviousLosses},
	}

	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, emissionstypes.Topic{
		Id:              topicId,
		Creator:         "creator",
		Metadata:        "metadata",
		LossLogic:       "losslogic",
		LossMethod:      "lossmethod",
		InferenceLogic:  "inferencelogic",
		InferenceMethod: "inferencemethod",
		EpochLastEnded:  0,
		EpochLength:     100,
		GroundTruthLag:  10,
		DefaultArg:      "defaultarg",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   false,
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	})
	s.Require().NoError(err)

	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Set Previous Loss
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emissionstypes.ValueBundle{
		CombinedValue:       epoch2Get("network_loss"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
	})
	require.NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch3Get)
	s.Require().NoError(err)

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	forecasts, err := testutil.GetForecastsFromCsv(topicId, blockHeight, infererAddresses, forecasterAddresses, epoch3Get)
	s.Require().NoError(err)

	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
	s.Require().NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(s.ctx, s.emissionsKeeper, topicId, blockHeight, infererAddresses, forecasterAddresses, epoch2Get)
	s.Require().NoError(err)

	// Calculate
	valueBundle, _, _, _, err :=
		inferencesynthesis.GetNetworkInferencesAtBlock(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			blockHeight,
			blockHeightPreviousLosses,
		)
	require.NoError(err)
	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, epoch3Get("network_inference").String())
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, epoch3Get("network_naive_inference").String())

	s.Require().Len(valueBundle.InfererValues, 5)
	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if string(inference.Inferer) == infererValue.Worker {
				found = true
				require.Equal(inference.Value, infererValue.Value)
			}
		}
		require.True(found, "Inference not found")
	}

	s.Require().Len(valueBundle.ForecasterValues, 3)
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch string(forecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, epoch3Get("forecast_implied_inference_0").String())
		case forecaster1:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, epoch3Get("forecast_implied_inference_1").String())
		case forecaster2:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, epoch3Get("forecast_implied_inference_2").String())
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutInfererValues, 5)
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_0").String())
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_1").String())
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_2").String())
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_3").String())
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_4").String())
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, epoch3Get("network_inference_oneout_5").String())
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, epoch3Get("network_inference_oneout_6").String())
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, epoch3Get("network_inference_oneout_7").String())
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneInForecasterValues, 3)
	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, epoch3Get("network_naive_inference_onein_0").String())
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, epoch3Get("network_naive_inference_onein_1").String())
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, epoch3Get("network_naive_inference_onein_2").String())
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}
}

// TODO: Need to revisit these tests after finshing non-edge case tests
// func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithJustOneNotNewForecaster() {
// 	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
// 	epoch2Get := epochGet[302]
// 	epoch3Get := epochGet[303]

// 	require := s.Require()
// 	keeper := s.emissionsKeeper

// 	topicId := uint64(1)
// 	blockHeight := int64(300)
// 	blockHeightPreviousLosses := int64(200)

// 	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}
// 	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
// 		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeightPreviousLosses},
// 	}

// 	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, emissionstypes.Topic{
// 		Id:              topicId,
// 		Creator:         "creator",
// 		Metadata:        "metadata",
// 		LossLogic:       "losslogic",
// 		LossMethod:      "lossmethod",
// 		InferenceLogic:  "inferencelogic",
// 		InferenceMethod: "inferencemethod",
// 		EpochLastEnded:  0,
// 		EpochLength:     100,
// 		GroundTruthLag:  10,
// 		DefaultArg:      "defaultarg",
// 		PNorm:           alloraMath.NewDecFromInt64(3),
// 		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
// 		AllowNegative:   false,
// 		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
// 		InitialRegret:  alloraMath.MustNewDecFromString("0"),
// 	})
// 	s.Require().NoError(err)

// 	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
// 	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
// 	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
// 	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
// 	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"
// 	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

// 	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
// 	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
// 	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"
// 	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

// 	// Set Previous Loss
// 	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emissionstypes.ValueBundle{
// 		CombinedValue:       epoch2Get("network_loss"),
// 		ReputerRequestNonce: reputerRequestNonce,
// 		TopicId:             topicId,
// 	})
// 	require.NoError(err)

// 	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch3Get)
// 	s.Require().NoError(err)

// 	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
// 	s.Require().NoError(err)

// 	forecasts, err := testutil.GetForecastsFromCsv(topicId, blockHeight, infererAddresses, forecasterAddresses, epoch3Get)
// 	s.Require().NoError(err)

// 	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
// 	s.Require().NoError(err)

// 	// Set inferer network regrets
// 	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
// 		inferer0: epoch2Get("inference_regret_worker_0"),
// 		inferer1: epoch2Get("inference_regret_worker_1"),
// 		inferer2: epoch2Get("inference_regret_worker_2"),
// 		inferer3: epoch2Get("inference_regret_worker_3"),
// 		inferer4: epoch2Get("inference_regret_worker_4"),
// 	}

// 	for inferer, regret := range infererNetworkRegrets {
// 		s.emissionsKeeper.SetInfererNetworkRegret(
// 			s.ctx,
// 			topicId,
// 			inferer,
// 			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
// 		)
// 	}

// 	// Set forecaster network regrets - just one of the forecasters
// 	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
// 		forecaster0: epoch2Get("inference_regret_worker_7"),
// 	}
// 	allRegrets := make([]inferencesynthesis.Regret, 0)
// 	for _, regret := range infererNetworkRegrets {

// 		allRegrets = append(allRegrets, regret)
// 	}
// 	allRegrets = append(allRegrets, forecasterNetworkRegrets[forecaster0])

// 	for forecaster, regret := range forecasterNetworkRegrets {
// 		s.emissionsKeeper.SetForecasterNetworkRegret(
// 			s.ctx,
// 			topicId,
// 			forecaster,
// 			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
// 		)
// 	}

// 	// Set one in forecaster network regrets
// 	setOneInForecasterNetworkRegret := func(forecaster string, inferer string, value string) {
// 		keeper.SetOneInForecasterNetworkRegret(
// 			s.ctx,
// 			topicId,
// 			forecaster,
// 			inferer,
// 			emissionstypes.TimestampedValue{
// 				BlockHeight: blockHeight,
// 				Value:       alloraMath.MustNewDecFromString(value),
// 			},
// 		)
// 	}

// 	/// Epoch 3 values
// 	setOneInForecasterNetworkRegret(forecaster0, inferer0, epoch2Get("inference_regret_worker_2_onein_2").String())
// 	setOneInForecasterNetworkRegret(forecaster0, inferer1, epoch2Get("inference_regret_worker_1_onein_2").String())
// 	setOneInForecasterNetworkRegret(forecaster0, inferer2, epoch2Get("inference_regret_worker_2_onein_2").String())
// 	setOneInForecasterNetworkRegret(forecaster0, inferer3, epoch2Get("inference_regret_worker_3_onein_2").String())
// 	setOneInForecasterNetworkRegret(forecaster0, inferer4, epoch2Get("inference_regret_worker_4_onein_2").String())
// 	setOneInForecasterNetworkRegret(forecaster0, forecaster0, epoch2Get("inference_regret_worker_5_onein_2").String())

// 	// Update topic initial regret for new participants
// 	epsilon := alloraMath.MustNewDecFromString("0.01")
// 	pNorm := alloraMath.MustNewDecFromString("3.0")
// 	cNorm := alloraMath.MustNewDecFromString("0.75")
// 	initialRegret, err := inferencesynthesis.CalcTopicInitialRegret(allRegrets, epsilon, pNorm, cNorm)
// 	require.NoError(err)
// 	testutil.InEpsilon5(s.T(), initialRegret, "-3.112687101514772")

// 	err = s.emissionsKeeper.UpdateTopicInitialRegret(s.ctx, topicId, initialRegret)
// 	require.NoError(err)

// 	// Calculate network inference
// 	valueBundle, _, _, _, err :=
// 		inferencesynthesis.GetNetworkInferencesAtBlock(
// 			s.ctx,
// 			s.emissionsKeeper,
// 			topicId,
// 			blockHeight,
// 			blockHeightPreviousLosses,
// 		)
// 	require.NoError(err)

// 	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "0.13832892076283418")
// 	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, "-0.217498746751143482")

// 	for _, inference := range inferences.Inferences {
// 		found := false
// 		for _, infererValue := range valueBundle.InfererValues {
// 			if string(inference.Inferer) == infererValue.Worker {
// 				found = true
// 				require.Equal(inference.Value, infererValue.Value)
// 			}
// 		}
// 		require.True(found, "Inference not found")
// 	}

// 	for _, forecasterValue := range valueBundle.ForecasterValues {
// 		switch string(forecasterValue.Worker) {
// 		case forecaster0:
// 			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.16104230535974168")
// 		case forecaster1:
// 			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.2129694366166174")
// 		case forecaster2:
// 			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.19537706255730902")
// 		default:
// 			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
// 		}
// 	}

// 	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
// 		switch string(oneOutInfererValue.Worker) {
// 		case inferer0:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.18159007177591316")
// 		case inferer1:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.1891891776070881")
// 		case inferer2:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.2453618383732347")
// 		case inferer3:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.17135248130644976")
// 		case inferer4:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.2519675192553942")
// 		default:
// 			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
// 		}
// 	}

// 	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
// 		switch string(oneOutForecasterValue.Worker) {
// 		case forecaster0:
// 			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.21430649513970956")
// 		case forecaster1:
// 			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.20645616254043958")
// 		case forecaster2:
// 			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.20875138413550787")
// 		default:
// 			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
// 		}
// 	}

// 	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
// 		switch string(oneInForecasterValue.Worker) {
// 		case forecaster0:
// 			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.20808933985257652")
// 		case forecaster1:
// 			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.21674386172872248")
// 		case forecaster2:
// 			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.21381179938550443")
// 		default:
// 			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
// 		}
// 	}
// }

// func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithAllInferersForecastersNew() {
// 	s.SetupTest()
// 	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
// 	epoch2Get := epochGet[302]
// 	epoch3Get := epochGet[303]

// 	require := s.Require()
// 	keeper := s.emissionsKeeper

// 	topicId := uint64(1)
// 	blockHeightInferences := int64(300)
// 	blockHeightPreviousLosses := int64(200)

// 	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeightInferences}
// 	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
// 		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeightPreviousLosses},
// 	}

// 	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, emissionstypes.Topic{
// 		Id:              topicId,
// 		Creator:         "creator",
// 		Metadata:        "metadata",
// 		LossLogic:       "losslogic",
// 		LossMethod:      "lossmethod",
// 		InferenceLogic:  "inferencelogic",
// 		InferenceMethod: "inferencemethod",
// 		EpochLastEnded:  0,
// 		EpochLength:     100,
// 		GroundTruthLag:  10,
// 		DefaultArg:      "defaultarg",
// 		PNorm:           alloraMath.NewDecFromInt64(3),
// 		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
// 		AllowNegative:   false,
// 		Epsilon:         alloraMath.MustNewDecFromString("0.0001"),
// 	})
// 	s.Require().NoError(err)

// 	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
// 	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
// 	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
// 	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
// 	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"
// 	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

// 	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
// 	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
// 	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"
// 	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

// 	// Set Previous Loss
// 	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emissionstypes.ValueBundle{
// 		CombinedValue:       epoch2Get("network_loss"),
// 		ReputerRequestNonce: reputerRequestNonce,
// 		TopicId:             topicId,
// 	})
// 	require.NoError(err)

// 	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeightInferences, infererAddresses, epoch3Get)
// 	s.Require().NoError(err)

// 	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
// 	s.Require().NoError(err)

// 	forecasts, err := testutil.GetForecastsFromCsv(topicId, blockHeightInferences, infererAddresses, forecasterAddresses, epoch3Get)
// 	s.Require().NoError(err)

// 	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
// 	s.Require().NoError(err)

// 	// Calculate
// 	valueBundle, _, _, _, err :=
// 		inferencesynthesis.GetNetworkInferencesAtBlock(
// 			s.ctx,
// 			s.emissionsKeeper,
// 			topicId,
// 			blockHeightInferences,
// 			blockHeightPreviousLosses,
// 		)

// 	require.NoError(err)
// 	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "-0.20711031728617318")
// 	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, "-0.217498746751143482")

// 	for _, inference := range inferences.Inferences {
// 		found := false
// 		for _, infererValue := range valueBundle.InfererValues {
// 			if string(inference.Inferer) == infererValue.Worker {
// 				found = true
// 				require.Equal(inference.Value, infererValue.Value)
// 			}
// 		}
// 		require.True(found, "Inference not found")
// 	}

// 	for _, forecasterValue := range valueBundle.ForecasterValues {
// 		switch string(forecasterValue.Worker) {
// 		case forecaster0:
// 			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.16104230535974168")
// 		case forecaster1:
// 			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.2129694366166174")
// 		case forecaster2:
// 			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.19537706255730902")
// 		default:
// 			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
// 		}
// 	}

// 	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
// 		switch string(oneOutInfererValue.Worker) {
// 		case inferer0:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.17891655703176346")
// 		case inferer1:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.18952169458753146")
// 		case inferer2:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.24439774598242406")
// 		case inferer3:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.1731395049235943")
// 		case inferer4:
// 			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.24978380449844517")
// 		default:
// 			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
// 		}
// 	}

// 	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
// 		switch string(oneOutForecasterValue.Worker) {
// 		case forecaster0:
// 			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.21369146184709198")
// 		case forecaster1:
// 			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.20627330023896687")
// 		case forecaster2:
// 			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.2087864965331538")
// 		default:
// 			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
// 		}
// 	}

// 	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
// 		switch string(oneInForecasterValue.Worker) {
// 		case forecaster0:
// 			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.2080893398525765")
// 		case forecaster1:
// 			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.21674386172872245")
// 		case forecaster2:
// 			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.2138117993855044")
// 		default:
// 			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
// 		}
// 	}
// }

func (s *InferenceSynthesisTestSuite) TestGetLatestNetworkInferenceFromCsv() {
	s.SetupTest()
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[302]
	epoch3Get := epochGet[303]

	require := s.Require()
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	blockHeightInferences := int64(300)
	blockHeightPreviousLosses := int64(200)

	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeightInferences}
	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeightPreviousLosses},
	}

	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, emissionstypes.Topic{
		Id:              topicId,
		Creator:         "creator",
		Metadata:        "metadata",
		LossLogic:       "losslogic",
		LossMethod:      "lossmethod",
		InferenceLogic:  "inferencelogic",
		InferenceMethod: "inferencemethod",
		EpochLastEnded:  0,
		EpochLength:     100,
		GroundTruthLag:  10,
		DefaultArg:      "defaultarg",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   false,
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	})
	s.Require().NoError(err)

	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Set Previous Loss
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emissionstypes.ValueBundle{
		CombinedValue:       epoch2Get("network_loss"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
	})
	require.NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeightInferences, infererAddresses, epoch3Get)
	s.Require().NoError(err)

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	forecasts, err := testutil.GetForecastsFromCsv(topicId, blockHeightInferences, infererAddresses, forecasterAddresses, epoch3Get)
	s.Require().NoError(err)

	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
	s.Require().NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(s.ctx, s.emissionsKeeper, topicId, blockHeightInferences, infererAddresses, forecasterAddresses, epoch2Get)
	s.Require().NoError(err)

	// Calculate
	valueBundle, _, _, _, _, _, err :=
		inferencesynthesis.GetLatestNetworkInference(
			s.ctx,
			s.emissionsKeeper,
			topicId,
		)
	require.NoError(err)
	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, epoch3Get("network_inference").String())
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, epoch3Get("network_naive_inference").String())

	s.Require().Len(valueBundle.InfererValues, 5)
	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if string(inference.Inferer) == infererValue.Worker {
				found = true
				require.Equal(inference.Value, infererValue.Value)
			}
		}
		require.True(found, "Inference not found")
	}

	s.Require().Len(valueBundle.ForecasterValues, 3)
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch string(forecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, epoch3Get("forecast_implied_inference_0").String())
		case forecaster1:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, epoch3Get("forecast_implied_inference_1").String())
		case forecaster2:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, epoch3Get("forecast_implied_inference_2").String())
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutInfererValues, 5)
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_0").String())
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_1").String())
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_2").String())
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_3").String())
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_4").String())
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, epoch3Get("network_inference_oneout_5").String())
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, epoch3Get("network_inference_oneout_6").String())
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, epoch3Get("network_inference_oneout_7").String())
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneInForecasterValues, 3)
	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, epoch3Get("network_naive_inference_onein_0").String())
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, epoch3Get("network_naive_inference_onein_1").String())
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, epoch3Get("network_naive_inference_onein_2").String())
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}
}
