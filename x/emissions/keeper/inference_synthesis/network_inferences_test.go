package inference_synthesis_test

import (
	"reflect"
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/stretchr/testify/assert"

	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/testdata"
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
	epochGet := testdata.GetSimulatedValuesGetterForEpochs()
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

	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

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
				Inferer:     inferer0,
				Value:       epoch3Get("inference_0"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer1,
				Value:       epoch3Get("inference_1"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer2,
				Value:       epoch3Get("inference_2"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer3,
				Value:       epoch3Get("inference_3"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_0_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_0_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_0_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_0_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_1_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_1_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_1_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_1_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_2_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_2_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_2_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_2_for_3"),
					},
					{
						Inferer: inferer4,
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
		inferer0: epoch2Get("inference_regret_worker_0"),
		inferer1: epoch2Get("inference_regret_worker_1"),
		inferer2: epoch2Get("inference_regret_worker_2"),
		inferer3: epoch2Get("inference_regret_worker_3"),
		inferer4: epoch2Get("inference_regret_worker_4"),
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
	setOneInForecasterNetworkRegret(forecaster0, inferer0, epoch2Get("inference_regret_worker_0_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer1, epoch2Get("inference_regret_worker_1_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer2, epoch2Get("inference_regret_worker_2_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer3, epoch2Get("inference_regret_worker_3_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer4, epoch2Get("inference_regret_worker_4_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, forecaster0, epoch2Get("inference_regret_worker_5_onein_0").String())

	setOneInForecasterNetworkRegret(forecaster1, inferer0, epoch2Get("inference_regret_worker_0_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer1, epoch2Get("inference_regret_worker_1_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer2, epoch2Get("inference_regret_worker_2_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer3, epoch2Get("inference_regret_worker_3_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer4, epoch2Get("inference_regret_worker_4_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, forecaster1, epoch2Get("inference_regret_worker_5_onein_1").String())

	setOneInForecasterNetworkRegret(forecaster2, inferer0, epoch2Get("inference_regret_worker_0_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer1, epoch2Get("inference_regret_worker_1_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer2, epoch2Get("inference_regret_worker_2_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer3, epoch2Get("inference_regret_worker_3_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer4, epoch2Get("inference_regret_worker_4_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, forecaster2, epoch2Get("inference_regret_worker_5_onein_2").String())

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

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithJustOneNotNewForecaster() {
	epochGet := testdata.GetSimulatedValuesGetterForEpochs()
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
		Epsilon:         alloraMath.MustNewDecFromString("0.0001"),
	})
	s.Require().NoError(err)

	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

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
				Inferer:     inferer0,
				Value:       epoch3Get("inference_0"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer1,
				Value:       epoch3Get("inference_1"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer2,
				Value:       epoch3Get("inference_2"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer3,
				Value:       epoch3Get("inference_3"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_0_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_0_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_0_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_0_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_1_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_1_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_1_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_1_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_2_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_2_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_2_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_2_for_3"),
					},
					{
						Inferer: inferer4,
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
		inferer0: epoch2Get("inference_regret_worker_0"),
		inferer1: epoch2Get("inference_regret_worker_1"),
		inferer2: epoch2Get("inference_regret_worker_2"),
		inferer3: epoch2Get("inference_regret_worker_3"),
		inferer4: epoch2Get("inference_regret_worker_4"),
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set forecaster network regrets - just one of the forecasters
	forecasterNetworkRegrets := map[string]inferencesynthesis.Regret{
		forecaster0: epoch2Get("inference_regret_worker_5"),
	}
	allRegrets := make([]inferencesynthesis.Regret, 0)
	for _, regret := range infererNetworkRegrets {

		allRegrets = append(allRegrets, regret)
	}
	allRegrets = append(allRegrets, forecasterNetworkRegrets[forecaster0])

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
	setOneInForecasterNetworkRegret(forecaster0, inferer0, epoch2Get("inference_regret_worker_0_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer1, epoch2Get("inference_regret_worker_1_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer2, epoch2Get("inference_regret_worker_2_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer3, epoch2Get("inference_regret_worker_3_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer4, epoch2Get("inference_regret_worker_4_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, forecaster0, epoch2Get("inference_regret_worker_5_onein_0").String())

	// Update topic initial regret for new participants
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	initialRegret, err := inferencesynthesis.CalcTopicInitialRegret(allRegrets, epsilon, pNorm, cNorm)
	require.NoError(err)
	testutil.InEpsilon5(s.T(), initialRegret, "0.3150416097077003")

	err = s.emissionsKeeper.UpdateTopicInitialRegret(s.ctx, topicId, initialRegret)
	require.NoError(err)

	// Calculate network inference
	valueBundle, _, _, _, err :=
		inferencesynthesis.GetNetworkInferencesAtBlock(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			blockHeight,
			blockHeightPreviousLosses,
		)
	require.NoError(err)

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "-0.20730026705848031")
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, "-0.217498746751143482")

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
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.16104230535974168")
		case forecaster1:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.2129694366166174")
		case forecaster2:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.19537706255730902")
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.18159007177591316")
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.1891891776070881")
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.2453618383732347")
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.17135248130644976")
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.2519675192553942")
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.21430649513970956")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.20645616254043958")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.20875138413550787")
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.20808933985257652")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.21674386172872248")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.21381179938550443")
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithAllInferersForecastersNew() {
	s.SetupTest()
	epochGet := testdata.GetSimulatedValuesGetterForEpochs()
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
		Epsilon:         alloraMath.MustNewDecFromString("0.0001"),
	})
	s.Require().NoError(err)

	inferer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	inferer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	inferer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	inferer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	inferer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

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
				Inferer:     inferer0,
				Value:       epoch3Get("inference_0"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer1,
				Value:       epoch3Get("inference_1"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer2,
				Value:       epoch3Get("inference_2"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer3,
				Value:       epoch3Get("inference_3"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_0_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_0_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_0_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_0_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_1_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_1_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_1_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_1_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_2_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_2_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_2_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_2_for_3"),
					},
					{
						Inferer: inferer4,
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

	// Calculate
	valueBundle, _, _, _, err :=
		inferencesynthesis.GetNetworkInferencesAtBlock(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			blockHeightInferences,
			blockHeightPreviousLosses,
		)

	require.NoError(err)
	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "-0.20711031728617318")
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, "-0.217498746751143482")

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
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.16104230535974168")
		case forecaster1:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.2129694366166174")
		case forecaster2:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "-0.19537706255730902")
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.17891655703176346")
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.18952169458753146")
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.24439774598242406")
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.1731395049235943")
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "-0.24978380449844517")
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.21369146184709198")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.20627330023896687")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "-0.2087864965331538")
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.2080893398525765")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.21674386172872245")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "-0.2138117993855044")
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetLatestNetworkInference() {
	s.SetupTest()
	epochGet := testdata.GetSimulatedValuesGetterForEpochs()
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

	// Set Inferences
	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				Inferer:     inferer0,
				Value:       epoch3Get("inference_0"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer1,
				Value:       epoch3Get("inference_1"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer2,
				Value:       epoch3Get("inference_2"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer3,
				Value:       epoch3Get("inference_3"),
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
			},
			{
				Inferer:     inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_0_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_0_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_0_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_0_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_1_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_1_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_1_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_1_for_3"),
					},
					{
						Inferer: inferer4,
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
						Inferer: inferer0,
						Value:   epoch3Get("forecasted_loss_2_for_0"),
					},
					{
						Inferer: inferer1,
						Value:   epoch3Get("forecasted_loss_2_for_1"),
					},
					{
						Inferer: inferer2,
						Value:   epoch3Get("forecasted_loss_2_for_2"),
					},
					{
						Inferer: inferer3,
						Value:   epoch3Get("forecasted_loss_2_for_3"),
					},
					{
						Inferer: inferer4,
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
		inferer0: epoch2Get("inference_regret_worker_0"),
		inferer1: epoch2Get("inference_regret_worker_1"),
		inferer2: epoch2Get("inference_regret_worker_2"),
		inferer3: epoch2Get("inference_regret_worker_3"),
		inferer4: epoch2Get("inference_regret_worker_4"),
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

	// Set naive inferer network regrets
	infererNaiveNetworkRegrets :=
		map[string]inferencesynthesis.Regret{
			inferer0: epoch2Get("naive_inference_regret_worker_0"),
			inferer1: epoch2Get("naive_inference_regret_worker_1"),
			inferer2: epoch2Get("naive_inference_regret_worker_2"),
			inferer3: epoch2Get("naive_inference_regret_worker_3"),
			inferer4: epoch2Get("naive_inference_regret_worker_4"),
		}
	for inferer, regret := range infererNaiveNetworkRegrets {
		s.emissionsKeeper.SetNaiveInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
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

	// Set one in forecaster network regrets
	setOneInForecasterNetworkRegret(forecaster0, inferer0, epoch2Get("inference_regret_worker_0_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer1, epoch2Get("inference_regret_worker_1_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer2, epoch2Get("inference_regret_worker_2_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer3, epoch2Get("inference_regret_worker_3_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, inferer4, epoch2Get("inference_regret_worker_4_onein_0").String())
	setOneInForecasterNetworkRegret(forecaster0, forecaster0, epoch2Get("inference_regret_worker_5_onein_0").String())

	setOneInForecasterNetworkRegret(forecaster1, inferer0, epoch2Get("inference_regret_worker_0_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer1, epoch2Get("inference_regret_worker_1_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer2, epoch2Get("inference_regret_worker_2_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer3, epoch2Get("inference_regret_worker_3_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, inferer4, epoch2Get("inference_regret_worker_4_onein_1").String())
	setOneInForecasterNetworkRegret(forecaster1, forecaster1, epoch2Get("inference_regret_worker_5_onein_1").String())

	setOneInForecasterNetworkRegret(forecaster2, inferer0, epoch2Get("inference_regret_worker_0_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer1, epoch2Get("inference_regret_worker_1_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer2, epoch2Get("inference_regret_worker_2_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer3, epoch2Get("inference_regret_worker_3_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, inferer4, epoch2Get("inference_regret_worker_4_onein_2").String())
	setOneInForecasterNetworkRegret(forecaster2, forecaster2, epoch2Get("inference_regret_worker_5_onein_2").String())

	// Set one-out inferer inferer network regrets
	setOneOutInfererInfererNetworkRegret := func(infererIndex int, infererIndex2 int) {
		infererAddress := infererAddresses[infererIndex]
		infererAddress2 := infererAddresses[infererIndex2]
		headerName := "inference_regret_worker_" + strconv.Itoa(infererIndex) + "_oneout_" + strconv.Itoa(infererIndex2)
		keeper.SetOneOutInfererInfererNetworkRegret(
			s.ctx,
			topicId,
			infererAddress2,
			infererAddress,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeightInferences,
				Value:       epoch2Get(headerName),
			},
		)
	}
	for inferer := 0; inferer < 5; inferer++ {
		for inferer2 := 0; inferer2 < 5; inferer2++ {
			setOneOutInfererInfererNetworkRegret(inferer, inferer2)
		}
	}
	// Set one-out inferer forecaster network regrets
	setOneOutInfererForecasterNetworkRegret := func(infererIndex int, forecasterIndex int) {
		infererName := infererAddresses[infererIndex]
		forecasterName := forecasterAddresses[forecasterIndex-5]
		headerName := "inference_regret_worker_" + strconv.Itoa(forecasterIndex) + "_oneout_" + strconv.Itoa(infererIndex)
		keeper.SetOneOutInfererForecasterNetworkRegret(
			s.ctx,
			topicId,
			infererName,
			forecasterName,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeightInferences,
				Value:       epoch2Get(headerName),
			},
		)
	}
	for forecaster := 5; forecaster < 8; forecaster++ {
		for inferer := 0; inferer < 5; inferer++ {
			setOneOutInfererForecasterNetworkRegret(inferer, forecaster)
		}
	}
	// Set one-out forecaster inferer network regrets
	setOneOutForecasterInfererNetworkRegret := func(infererIndex int, forecasterIndex int) {
		infererName := infererAddresses[infererIndex]
		forecasterName := forecasterAddresses[forecasterIndex-5]
		headerName := "inference_regret_worker_" + strconv.Itoa(infererIndex) + "_oneout_" + strconv.Itoa(forecasterIndex)
		keeper.SetOneOutForecasterInfererNetworkRegret(
			s.ctx,
			topicId,
			forecasterName,
			infererName,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeightInferences,
				Value:       epoch2Get(headerName),
			},
		)
	}
	for inferer := 0; inferer < 5; inferer++ {
		for forecaster := 5; forecaster < 8; forecaster++ {
			setOneOutForecasterInfererNetworkRegret(inferer, forecaster)
		}
	}
	// Set one-out forecaster forecaster network regrets
	setOneOutForecasterForecasterNetworkRegret := func(forecasterIndex int, forecasterIndex2 int) {
		forecasterName := forecasterAddresses[forecasterIndex-5]
		forecasterName2 := forecasterAddresses[forecasterIndex2-5]
		headerName := "inference_regret_worker_" + strconv.Itoa(forecasterIndex) + "_oneout_" + strconv.Itoa(forecasterIndex2)
		keeper.SetOneOutForecasterForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecasterName2,
			forecasterName,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeightInferences,
				Value:       epoch2Get(headerName),
			},
		)
	}
	for forecaster := 5; forecaster < 8; forecaster++ {
		for forecaster2 := 5; forecaster2 < 8; forecaster2++ {
			setOneOutForecasterForecasterNetworkRegret(forecaster, forecaster2)
		}
	}

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
