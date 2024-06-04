package inference_synthesis_test

import (
	"reflect"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/assert"

	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// instantiate a AllWorkersAreNew struct
// func NewWorkersAreNew(v bool) inference_synthesis.AllWorkersAreNew {
// 	return inference_synthesis.AllWorkersAreNew{
// 		AllInferersAreNew:    v,
// 		AllForecastersAreNew: v,
// 	}
// }

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
			result := inference_synthesis.MakeMapFromInfererToTheirInference(tc.inferences)
			assert.True(t, reflect.DeepEqual(result, tc.expected), "Expected and actual maps should be equal")
		})
	}
}

/*
func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlock() {
	require := s.Require()
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	blockHeight := int64(3)
	require.True(blockHeight >= s.ctx.BlockHeight())
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeight},
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
	})
	s.Require().NoError(err)

	reputer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	reputer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	reputer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	reputer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	reputer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"

	// Set Loss bundles

	reputerLossBundles := emissionstypes.ReputerValueBundles{
		ReputerValueBundles: []*emissionstypes.ReputerValueBundle{
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer0,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer1,
					CombinedValue:       alloraMath.MustNewDecFromString(".00000962701954026944"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer2,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer3,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000123986052417188"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer4,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000115363240547692"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
		},
	}

	err = keeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, blockHeight, reputerLossBundles)
	require.NoError(err)

	// Set Stake

	stake1, ok := cosmosMath.NewIntFromString("210535101370326000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer0, stake1)
	require.NoError(err)
	stake2, ok := cosmosMath.NewIntFromString("216697093951021000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer1, stake2)
	require.NoError(err)
	stake3, ok := cosmosMath.NewIntFromString("161740241803855000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer2, stake3)
	require.NoError(err)
	stake4, ok := cosmosMath.NewIntFromString("394848305052250000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer3, stake4)
	require.NoError(err)
	stake5, ok := cosmosMath.NewIntFromString("206169717590569000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer4, stake5)
	require.NoError(err)

	// Set Inferences

	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				Inferer:     reputer0,
				Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer1,
				Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer2,
				Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer3,
				Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer4,
				Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
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
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("0.003305466418410120"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("0.0002788248228566030"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString(".0000240536828602367"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("0.0008240378476798250"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString(".0000186192181193532"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("0.002308441286328890"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString(".0000214380788596749"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("0.012560171044167200"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("0.017998563880697900"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("0.00020024906252089700"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("0.005369218152594270"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("0.0002578158768320300"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("0.0076008583603885900"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("0.0076269073955871000"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("0.00035670236460009500"),
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

	infererNetworkRegrets := map[string]inference_synthesis.Regret{
		reputer0: alloraMath.MustNewDecFromString("0.29240710390153500"),
		reputer1: alloraMath.MustNewDecFromString("0.4182220944854450"),
		reputer2: alloraMath.MustNewDecFromString("0.17663501719135000"),
		reputer3: alloraMath.MustNewDecFromString("0.49617463489106400"),
		reputer4: alloraMath.MustNewDecFromString("0.27996060999688600"),
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// set forecaster network regrets

	forecasterNetworkRegrets := map[string]inference_synthesis.Regret{
		forecaster0: alloraMath.MustNewDecFromString("0.816066375505268"),
		forecaster1: alloraMath.MustNewDecFromString("0.8234558901838660"),
		forecaster2: alloraMath.MustNewDecFromString("0.8196673550408280"),
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

	setOneInForecasterNetworkRegret(forecaster0, reputer0, "-0.005488956369080480")
	setOneInForecasterNetworkRegret(forecaster0, reputer1, "0.17091263821766800")
	setOneInForecasterNetworkRegret(forecaster0, reputer2, "-0.15988639638192800")
	setOneInForecasterNetworkRegret(forecaster0, reputer3, "0.28690775330189800")
	setOneInForecasterNetworkRegret(forecaster0, reputer4, "-0.019476319822263300")

	setOneInForecasterNetworkRegret(forecaster0, forecaster0, "0.7370268872154170")

	setOneInForecasterNetworkRegret(forecaster1, reputer0, "-0.023601485104528100")
	setOneInForecasterNetworkRegret(forecaster1, reputer1, "0.1528001094822210")
	setOneInForecasterNetworkRegret(forecaster1, reputer2, "-0.1779989251173760")
	setOneInForecasterNetworkRegret(forecaster1, reputer3, "0.2687952245664510")
	setOneInForecasterNetworkRegret(forecaster1, reputer4, "-0.03758884855771100")

	setOneInForecasterNetworkRegret(forecaster1, forecaster1, "0.7307121775422120")

	setOneInForecasterNetworkRegret(forecaster2, reputer0, "-0.025084585281804600")
	setOneInForecasterNetworkRegret(forecaster2, reputer1, "0.15131700930494400")
	setOneInForecasterNetworkRegret(forecaster2, reputer2, "-0.17948202529465200")
	setOneInForecasterNetworkRegret(forecaster2, reputer3, "0.26731212438917400")
	setOneInForecasterNetworkRegret(forecaster2, reputer4, "-0.03907194873498750")

	setOneInForecasterNetworkRegret(forecaster2, forecaster2, "0.722844746771044")

	// Calculate

	valueBundle, err :=
		inference_synthesis.GetNetworkInferencesAtBlock(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			blockHeight,
			blockHeight,
		)
	require.NoError(err)

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "-0.08418238013037833391277949761424359")
	testutil.InEpsilon3(s.T(), valueBundle.NaiveValue, "-0.09089296942031617121265381217201911")

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
			testutil.InEpsilon2(s.T(), forecasterValue.Value, "-0.07360672083447152549990990449686835")
		case forecaster1:
			testutil.InEpsilon2(s.T(), forecasterValue.Value, "-0.07263773178885971876894786458169429")
		case forecaster2:
			testutil.InEpsilon2(s.T(), forecasterValue.Value, "-0.07333303938740419999999999999997501")
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case reputer0:
			testutil.InEpsilon2(s.T(), oneOutInfererValue.Value, "-0.09112256843970400263910853205529701")
		case reputer1:
			testutil.InEpsilon2(s.T(), oneOutInfererValue.Value, "-0.0849762526781571419651680849185323")
		case reputer2:
			testutil.InEpsilon2(s.T(), oneOutInfererValue.Value, "-0.07508631218815497306553765277306250")
		case reputer3:
			testutil.InEpsilon2(s.T(), oneOutInfererValue.Value, "-0.07762408532626815421861778958359602")
		case reputer4:
			testutil.InEpsilon2(s.T(), oneOutInfererValue.Value, "-0.097732445271841")
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon2(s.T(), oneInForecasterValue.Value, "-0.08562185282145071310963674889631515")
		case forecaster1:
			testutil.InEpsilon2(s.T(), oneInForecasterValue.Value, "-0.0857186447720307")
		case forecaster2:
			testutil.InEpsilon2(s.T(), oneInForecasterValue.Value, "-0.0853937827718047")
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}

	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			testutil.InEpsilon2(s.T(), oneOutForecasterValue.Value, "-0.08571218484894173358533220915281566")
		case forecaster1:
			testutil.InEpsilon2(s.T(), oneOutForecasterValue.Value, "-0.08575177379258356927513673523336433")
		case forecaster2:
			testutil.InEpsilon2(s.T(), oneOutForecasterValue.Value, "-0.08585235237690237323017634246068422")
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}
}
*/
