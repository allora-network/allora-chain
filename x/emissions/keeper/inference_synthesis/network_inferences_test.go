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
					Inferer: "worker1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				{
					TopicId: 102,
					Inferer: "worker2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				{
					TopicId: 103,
					Inferer: "worker3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
			expected: map[string]*emissionstypes.Inference{
				"worker1": {
					TopicId: 101,
					Inferer: "worker1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				"worker2": {
					TopicId: 102,
					Inferer: "worker2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				"worker3": {
					TopicId: 103,
					Inferer: "worker3",
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
	epochGet := GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[2]
	epoch3Get := epochGet[3]

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

	worker0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	worker1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	worker2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	worker3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	worker4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"

	// Set Previous Loss
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emissionstypes.ValueBundle{
		CombinedValue:       epoch2Get("network_loss_reputers"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
	})
	require.NoError(err)

	// Set Inferences
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				Inferer:     worker0,
				Value:       epoch3Get("inference_0"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     worker1,
				Value:       epoch3Get("inference_1"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     worker2,
				Value:       epoch3Get("inference_2"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     worker3,
				Value:       epoch3Get("inference_3"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     worker4,
				Value:       epoch3Get("inference_4"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	// Set Forecasts
	forecasts := emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: worker0,
						Value:   epoch3Get("forecasted_loss_0_for_0"),
					},
					{
						Inferer: worker1,
						Value:   epoch3Get("forecasted_loss_0_for_1"),
					},
					{
						Inferer: worker2,
						Value:   epoch3Get("forecasted_loss_0_for_2"),
					},
					{
						Inferer: worker3,
						Value:   epoch3Get("forecasted_loss_0_for_3"),
					},
					{
						Inferer: worker4,
						Value:   epoch3Get("forecasted_loss_0_for_4"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: worker0,
						Value:   epoch3Get("forecasted_loss_1_for_0"),
					},
					{
						Inferer: worker1,
						Value:   epoch3Get("forecasted_loss_1_for_1"),
					},
					{
						Inferer: worker2,
						Value:   epoch3Get("forecasted_loss_1_for_2"),
					},
					{
						Inferer: worker3,
						Value:   epoch3Get("forecasted_loss_1_for_3"),
					},
					{
						Inferer: worker4,
						Value:   epoch3Get("forecasted_loss_1_for_4"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: worker0,
						Value:   epoch3Get("forecasted_loss_2_for_0"),
					},
					{
						Inferer: worker1,
						Value:   epoch3Get("forecasted_loss_2_for_1"),
					},
					{
						Inferer: worker2,
						Value:   epoch3Get("forecasted_loss_2_for_2"),
					},
					{
						Inferer: worker3,
						Value:   epoch3Get("forecasted_loss_2_for_3"),
					},
					{
						Inferer: worker4,
						Value:   epoch3Get("forecasted_loss_2_for_4"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
	s.Require().NoError(err)

	// Set inferer network regrets
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		worker0: epoch2Get("inference_regret_worker_0"),
		worker1: epoch2Get("inference_regret_worker_1"),
		worker2: epoch2Get("inference_regret_worker_2"),
		worker3: epoch2Get("inference_regret_worker_3"),
		worker4: epoch2Get("inference_regret_worker_4"),
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set forecaster network regrets
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		forecaster0: epoch2Get("inference_regret_worker_5"),
		forecaster1: epoch2Get("inference_regret_worker_6"),
		forecaster2: epoch2Get("inference_regret_worker_7"),
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set one in forecaster network regrets
	setOneInForecasterNetworkRegret := func(forecaster string, inferer string, value string) {
		keeper.SetOneInForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       alloraMath.MustNewDecFromString(value),
			},
		)
	}

	/// Epoch 3 values
	setOneInForecasterNetworkRegret(forecaster0, worker0, epoch2Get("inference_regret_worker_0_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker1, epoch2Get("inference_regret_worker_1_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker2, epoch2Get("inference_regret_worker_2_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker3, epoch2Get("inference_regret_worker_3_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker4, epoch2Get("inference_regret_worker_4_onein_0").String())

	keeper.SetOneInForecasterSelfNetworkRegret(s.ctx, topicId, forecaster0, emissionstypes.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       epoch2Get("inference_regret_worker_5_onein_0"),
	})

	setOneInForecasterNetworkRegret(forecaster1, worker0, epoch2Get("inference_regret_worker_0_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker1, epoch2Get("inference_regret_worker_1_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker2, epoch2Get("inference_regret_worker_2_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker3, epoch2Get("inference_regret_worker_3_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker4, epoch2Get("inference_regret_worker_4_onein_1").String())

	keeper.SetOneInForecasterSelfNetworkRegret(s.ctx, topicId, forecaster1, emissionstypes.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       epoch2Get("inference_regret_worker_5_onein_1"),
	})

	setOneInForecasterNetworkRegret(forecaster2, worker0, epoch2Get("inference_regret_worker_0_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker1, epoch2Get("inference_regret_worker_1_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker2, epoch2Get("inference_regret_worker_2_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker3, epoch2Get("inference_regret_worker_3_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker4, epoch2Get("inference_regret_worker_4_onein_2").String())

	keeper.SetOneInForecasterSelfNetworkRegret(s.ctx, topicId, forecaster2, emissionstypes.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       epoch2Get("inference_regret_worker_5_onein_2"),
	})

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

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case worker0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_0").String())
		case worker1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_1").String())
		case worker2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_2").String())
		case worker3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_3").String())
		case worker4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_4").String())
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

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

func (s *InferenceSynthesisTestSuite) TestGetLatestNetworkInference() {
	s.SetupTest()
	epochGet := GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[2]
	epoch3Get := epochGet[3]

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

	worker0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	worker1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	worker2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	worker3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	worker4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"

	// Set Previous Loss
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emissionstypes.ValueBundle{
		CombinedValue:       epoch2Get("network_loss_reputers"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
	})
	require.NoError(err)

	// Set Inferences
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				Inferer:     worker0,
				Value:       epoch3Get("inference_0"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     worker1,
				Value:       epoch3Get("inference_1"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     worker2,
				Value:       epoch3Get("inference_2"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     worker3,
				Value:       epoch3Get("inference_3"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     worker4,
				Value:       epoch3Get("inference_4"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
		},
	}

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	// Set Forecasts
	forecasts := emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: worker0,
						Value:   epoch3Get("forecasted_loss_0_for_0"),
					},
					{
						Inferer: worker1,
						Value:   epoch3Get("forecasted_loss_0_for_1"),
					},
					{
						Inferer: worker2,
						Value:   epoch3Get("forecasted_loss_0_for_2"),
					},
					{
						Inferer: worker3,
						Value:   epoch3Get("forecasted_loss_0_for_3"),
					},
					{
						Inferer: worker4,
						Value:   epoch3Get("forecasted_loss_0_for_4"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: worker0,
						Value:   epoch3Get("forecasted_loss_1_for_0"),
					},
					{
						Inferer: worker1,
						Value:   epoch3Get("forecasted_loss_1_for_1"),
					},
					{
						Inferer: worker2,
						Value:   epoch3Get("forecasted_loss_1_for_2"),
					},
					{
						Inferer: worker3,
						Value:   epoch3Get("forecasted_loss_1_for_3"),
					},
					{
						Inferer: worker4,
						Value:   epoch3Get("forecasted_loss_1_for_4"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: worker0,
						Value:   epoch3Get("forecasted_loss_2_for_0"),
					},
					{
						Inferer: worker1,
						Value:   epoch3Get("forecasted_loss_2_for_1"),
					},
					{
						Inferer: worker2,
						Value:   epoch3Get("forecasted_loss_2_for_2"),
					},
					{
						Inferer: worker3,
						Value:   epoch3Get("forecasted_loss_2_for_3"),
					},
					{
						Inferer: worker4,
						Value:   epoch3Get("forecasted_loss_2_for_4"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
		},
	}

	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)
	s.Require().NoError(err)

	// Set inferer network regrets
	infererNetworkRegrets := map[string]inferencesynthesis.Regret{
		worker0: epoch2Get("inference_regret_worker_0"),
		worker1: epoch2Get("inference_regret_worker_1"),
		worker2: epoch2Get("inference_regret_worker_2"),
		worker3: epoch2Get("inference_regret_worker_3"),
		worker4: epoch2Get("inference_regret_worker_4"),
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeightInferences, Value: regret},
		)
	}

	// Set forecaster network regrets
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		forecaster0: epoch2Get("inference_regret_worker_5"),
		forecaster1: epoch2Get("inference_regret_worker_6"),
		forecaster2: epoch2Get("inference_regret_worker_7"),
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeightInferences, Value: regret},
		)
	}

	// Set one in forecaster network regrets
	setOneInForecasterNetworkRegret := func(forecaster string, inferer string, value string) {
		keeper.SetOneInForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeightInferences,
				Value:       alloraMath.MustNewDecFromString(value),
			},
		)
	}

	/// Epoch 3 values
	setOneInForecasterNetworkRegret(forecaster0, worker0, epoch2Get("inference_regret_worker_0_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker1, epoch2Get("inference_regret_worker_1_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker2, epoch2Get("inference_regret_worker_2_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker3, epoch2Get("inference_regret_worker_3_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, worker4, epoch2Get("inference_regret_worker_4_onein_0").String())

	keeper.SetOneInForecasterSelfNetworkRegret(s.ctx, topicId, forecaster0, emissionstypes.TimestampedValue{
		BlockHeight: blockHeightInferences,
		Value:       epoch2Get("inference_regret_worker_5_onein_0"),
	})

	setOneInForecasterNetworkRegret(forecaster1, worker0, epoch2Get("inference_regret_worker_0_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker1, epoch2Get("inference_regret_worker_1_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker2, epoch2Get("inference_regret_worker_2_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker3, epoch2Get("inference_regret_worker_3_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, worker4, epoch2Get("inference_regret_worker_4_onein_1").String())

	keeper.SetOneInForecasterSelfNetworkRegret(s.ctx, topicId, forecaster1, emissionstypes.TimestampedValue{
		BlockHeight: blockHeightInferences,
		Value:       epoch2Get("inference_regret_worker_5_onein_1"),
	})

	setOneInForecasterNetworkRegret(forecaster2, worker0, epoch2Get("inference_regret_worker_0_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker1, epoch2Get("inference_regret_worker_1_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker2, epoch2Get("inference_regret_worker_2_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker3, epoch2Get("inference_regret_worker_3_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, worker4, epoch2Get("inference_regret_worker_4_onein_2").String())

	keeper.SetOneInForecasterSelfNetworkRegret(s.ctx, topicId, forecaster2, emissionstypes.TimestampedValue{
		BlockHeight: blockHeightInferences,
		Value:       epoch2Get("inference_regret_worker_5_onein_2"),
	})

	// Calculate
	valueBundle, _, _, _, err :=
		inferencesynthesis.GetLatestNetworkInference(
			s.ctx,
			s.emissionsKeeper,
			topicId,
		)
	require.NoError(err)
	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, epoch3Get("network_inference").String())
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, epoch3Get("network_naive_inference").String())

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

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case worker0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_0").String())
		case worker1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_1").String())
		case worker2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_2").String())
		case worker3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_3").String())
		case worker4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, epoch3Get("network_inference_oneout_4").String())
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

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
