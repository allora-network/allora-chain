package inferencesynthesis_test

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

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesWhenNoInferences() {
	require := s.Require()
	topicId := uint64(1)
	blockHeight := int64(300)

	_, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			&blockHeight,
		)
	require.Error(err)
	require.Equal("while getting inferences: no inferences found for topic 1 at block 300: invalid request", err.Error())

	_, err =
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			nil,
		)
	require.Error(err)
	require.Equal("while getting inferences: no inferences found for topic 1 at latest block: invalid request", err.Error())
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

	topic := s.mockTopic()
	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	s.Require().NoError(err)

	inferer0 := s.addrsStr[0]
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]
	inferer4 := s.addrsStr[4]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrsStr[5]
	forecaster1 := s.addrsStr[6]
	forecaster2 := s.addrsStr[7]
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Set Previous Loss
	valueBundlePrevious := s.mockEmptyValueBundle(epoch2Get("network_loss"))
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, valueBundlePrevious)
	require.NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch3Get)
	s.Require().NoError(err)

	err = keeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	s.Require().NoError(err)

	forecasts, err := testutil.GetForecastsFromCsv(
		topicId,
		blockHeight,
		infererAddresses,
		forecasterAddresses,
		epoch3Get,
	)
	s.Require().NoError(err)

	err = keeper.InsertActiveForecasts(s.ctx, topicId, simpleNonce.BlockHeight, forecasts)
	s.Require().NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(s.ctx, s.emissionsKeeper, topicId,
		blockHeight,
		infererAddresses,
		forecasterAddresses,
		epoch2Get,
	)
	s.Require().NoError(err)

	// Calculate
	result, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			&blockHeight,
		)
	require.NoError(err)
	valueBundle := result.NetworkInferences

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, epoch3Get("network_inference").String())
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, epoch3Get("network_naive_inference").String())

	s.Require().Len(valueBundle.InfererValues, 5)
	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if inference.Inferer == infererValue.Worker {
				found = true
				require.Equal(inference.Value, infererValue.Value)
			}
		}
		require.True(found, "Inference not found")
	}

	s.Require().Len(valueBundle.ForecasterValues, 3)
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch forecasterValue.Worker {
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
		switch oneOutInfererValue.Worker {
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
		switch oneOutForecasterValue.Worker {
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
		switch oneInForecasterValue.Worker {
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

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithNoPreviousLossesFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[302]
	topicId := uint64(1)
	blockHeight := int64(300)
	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}

	topic := s.mockTopic()
	topic.InitialRegret = alloraMath.ZeroDec()
	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	s.Require().NoError(err)

	inferer0 := s.addrsStr[0]
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]
	inferer4 := s.addrsStr[4]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch2Get)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	s.Require().NoError(err)

	result, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			&blockHeight,
		)
	s.Require().NoError(err)
	testutil.InEpsilon5(s.T(), result.NetworkInferences.CombinedValue, "0.1997509073157136")
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithOneOldInfererNoForecastersFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch1Get := epochGet[301]
	epoch2Get := epochGet[302]
	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)
	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}

	topic := s.mockTopic()
	topic.InitialRegret = alloraMath.MustNewDecFromString("-1.8331309069480215")
	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	s.Require().NoError(err)

	inferer0 := s.addrsStr[0]
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]
	inferer4 := s.addrsStr[4]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	// Set Previous Loss
	valueBundlePrevious := s.mockEmptyValueBundle(epoch1Get("network_loss"))
	err = s.emissionsKeeper.InsertNetworkLossBundleAtBlock(
		s.ctx, topicId, blockHeightPreviousLosses, valueBundlePrevious)
	s.Require().NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch2Get)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	s.Require().NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(
		s.ctx, s.emissionsKeeper, topicId, blockHeight, []string{inferer0}, []string{}, epoch1Get)
	s.Require().NoError(err)

	result, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			&blockHeight,
		)
	s.Require().NoError(err)
	valueBundle := result.NetworkInferences

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "0.20059970801966293")

	s.Require().Len(valueBundle.OneOutInfererValues, 5)
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch oneOutInfererValue.Worker {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.1404419672286048")
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.1288457756437288")
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.1431887680171583")
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.18794792677471128")
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.17208154172014362")
		default:
			s.Require().Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithOldInferersOneOldForecasterFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch1Get := epochGet[301]
	epoch2Get := epochGet[302]

	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)
	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}

	topic := s.mockTopic()
	topic.InitialRegret = alloraMath.MustNewDecFromString("-3.7780955644806307")

	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	s.Require().NoError(err)

	inferer0 := s.addrsStr[0]
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]
	inferer4 := s.addrsStr[4]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrsStr[5]
	forecaster1 := s.addrsStr[6]
	forecaster2 := s.addrsStr[7]
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Set Previous Loss
	emptyValueBundle := s.mockEmptyValueBundle(epoch1Get("network_loss"))
	err = s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emptyValueBundle)
	s.Require().NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch2Get)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	s.Require().NoError(err)

	forecasts, err := testutil.GetForecastsFromCsv(topicId, blockHeight, infererAddresses, forecasterAddresses, epoch2Get)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InsertActiveForecasts(s.ctx, topicId, simpleNonce.BlockHeight, forecasts)
	s.Require().NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		infererAddresses,
		[]string{forecaster0},
		epoch1Get,
	)
	s.Require().NoError(err)

	// Calculate
	result, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			&blockHeight,
		)
	s.Require().NoError(err)
	valueBundle := result.NetworkInferences

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "0.09643700801928372")

	s.Require().Len(valueBundle.InfererValues, 5)
	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if inference.Inferer == infererValue.Worker {
				found = true
				s.Require().Equal(inference.Value, infererValue.Value)
			}
		}
		s.Require().True(found, "Inference not found")
	}

	s.Require().Len(valueBundle.ForecasterValues, 3)
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch forecasterValue.Worker {
		case forecaster0:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "0.08475997300723974")
		case forecaster1:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "0.0807110022144602")
		case forecaster2:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "0.07851400736008668")
		default:
			s.Require().Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutInfererValues, 5)
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch oneOutInfererValue.Worker {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.10278833087892551")
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.10309434130026068")
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.10749347667557652")
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.09957413455954356")
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.23133130005198607")
		default:
			s.Require().Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch oneOutForecasterValue.Worker {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "0.19384484001403682")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "0.13368285139309616")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "0.1339967078008638")
		default:
			s.Require().Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneInForecasterValues, 3)
	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch oneInForecasterValue.Worker {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "0.086385958713291")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "0.14220283026646785")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "0.1418366644574056")
		default:
			s.Require().Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockWithOldInferersAllNewForecastersFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch1Get := epochGet[301]
	epoch2Get := epochGet[302]

	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)

	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}

	topic := s.mockTopic()
	topic.InitialRegret = alloraMath.MustNewDecFromString("-3.0557074373274475")
	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	s.Require().NoError(err)

	inferer0 := s.addrsStr[0]
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]
	inferer4 := s.addrsStr[4]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrsStr[5]
	forecaster1 := s.addrsStr[6]
	forecaster2 := s.addrsStr[7]
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Set Previous Loss
	emptyValueBundle := s.mockEmptyValueBundle(epoch1Get("network_loss"))
	err = s.emissionsKeeper.InsertNetworkLossBundleAtBlock(
		s.ctx, topicId, blockHeightPreviousLosses,
		emptyValueBundle,
	)
	s.Require().NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeight, infererAddresses, epoch2Get)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	s.Require().NoError(err)

	forecasts, err := testutil.GetForecastsFromCsv(topicId, blockHeight, infererAddresses, forecasterAddresses, epoch2Get)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InsertActiveForecasts(s.ctx, topicId, simpleNonce.BlockHeight, forecasts)
	s.Require().NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(s.ctx, s.emissionsKeeper, topicId, blockHeight, infererAddresses, []string{}, epoch1Get)
	s.Require().NoError(err)

	// Calculate
	result, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			&blockHeight,
		)
	s.Require().NoError(err)
	valueBundle := result.NetworkInferences

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, "0.20065795737590336")

	s.Require().Len(valueBundle.InfererValues, 5)
	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if inference.Inferer == infererValue.Worker {
				found = true
				s.Require().Equal(inference.Value, infererValue.Value)
			}
		}
		s.Require().True(found, "Inference not found")
	}

	s.Require().Len(valueBundle.ForecasterValues, 3)
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch forecasterValue.Worker {
		case forecaster0:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "0.08475997300723974")
		case forecaster1:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "0.0807110022144602")
		case forecaster2:
			testutil.InEpsilon5(s.T(), forecasterValue.Value, "0.07851400736008668")
		default:
			s.Require().Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutInfererValues, 5)
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch oneOutInfererValue.Worker {
		case inferer0:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.195653799650153")
		case inferer1:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.1768775314107972")
		case inferer2:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.1931955480873375")
		case inferer3:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.20197760150788802")
		case inferer4:
			testutil.InEpsilon5(s.T(), oneOutInfererValue.Value, "0.21429866100721226")
		default:
			s.Require().Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch oneOutForecasterValue.Worker {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "0.13310442699412764")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "0.13368285139309616")
		case forecaster2:

			testutil.InEpsilon5(s.T(), oneOutForecasterValue.Value, "0.1339967078008638")
		default:
			s.Require().Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}

	s.Require().Len(valueBundle.OneInForecasterValues, 3)
	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch oneInForecasterValue.Worker {
		case forecaster0:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "0.1428776587319311")
		case forecaster1:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "0.14220283026646785")
		case forecaster2:
			testutil.InEpsilon5(s.T(), oneInForecasterValue.Value, "0.1418366644574056")
		default:
			s.Require().Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}
}

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

	topic := s.mockTopic()
	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	require.NoError(err)

	inferer0 := s.addrsStr[0]
	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]
	inferer4 := s.addrsStr[4]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrsStr[5]
	forecaster1 := s.addrsStr[6]
	forecaster2 := s.addrsStr[7]
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Set Previous Loss
	valueBundlePrevious := s.mockEmptyValueBundle(epoch2Get("network_loss"))
	err = keeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, valueBundlePrevious)
	require.NoError(err)

	inferences, err := testutil.GetInferencesFromCsv(topicId, blockHeightInferences, infererAddresses, epoch3Get)
	require.NoError(err)

	err = keeper.InsertActiveInferences(s.ctx, topicId, simpleNonce.BlockHeight, inferences)
	require.NoError(err)

	forecasts, err := testutil.GetForecastsFromCsv(
		topicId, blockHeightInferences, infererAddresses, forecasterAddresses, epoch3Get)
	require.NoError(err)

	err = keeper.InsertActiveForecasts(s.ctx, topicId, simpleNonce.BlockHeight, forecasts)
	require.NoError(err)

	// Set regrets from the previous epoch
	err = testutil.SetRegretsFromPreviousEpoch(
		s.ctx, s.emissionsKeeper, topicId, blockHeightInferences, infererAddresses, forecasterAddresses, epoch2Get)
	require.NoError(err)

	// Calculate
	result, err :=
		inferencesynthesis.GetNetworkInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			nil,
		)
	require.NoError(err)
	valueBundle := result.NetworkInferences

	testutil.InEpsilon5(s.T(), valueBundle.CombinedValue, epoch3Get("network_inference").String())
	testutil.InEpsilon5(s.T(), valueBundle.NaiveValue, epoch3Get("network_naive_inference").String())

	require.Len(valueBundle.InfererValues, 5)
	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if inference.Inferer == infererValue.Worker {
				found = true
				require.Equal(inference.Value, infererValue.Value)
			}
		}
		require.True(found, "Inference not found")
	}

	require.Len(valueBundle.ForecasterValues, 3)
	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch forecasterValue.Worker {
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

	require.Len(valueBundle.OneOutInfererValues, 5)
	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch oneOutInfererValue.Worker {
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

	require.Len(valueBundle.OneOutForecasterValues, 3)
	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch oneOutForecasterValue.Worker {
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

	require.Len(valueBundle.OneInForecasterValues, 3)
	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch oneInForecasterValue.Worker {
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

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesWithMedianCalculation() {
	require := s.Require()
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(300)

	inferer1 := s.addrsStr[1]
	inferer2 := s.addrsStr[2]
	inferer3 := s.addrsStr[3]

	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     inferer1,
				Value:       alloraMath.MustNewDecFromString("10.0"),
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     inferer2,
				Value:       alloraMath.MustNewDecFromString("30.0"),
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     inferer3,
				Value:       alloraMath.MustNewDecFromString("20.0"),
			},
		},
	}

	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	err := keeper.InsertActiveInferences(s.ctx, topicId, nonce.BlockHeight, inferences)
	s.Require().NoError(err)

	result, err := inferencesynthesis.GetNetworkInferences(s.ctx, keeper, topicId, &blockHeight)
	s.Require().NoError(err)
	valueBundle := result.NetworkInferences

	expectedMedian := alloraMath.MustNewDecFromString("20.0")
	s.Require().True(expectedMedian.Equal(valueBundle.CombinedValue), "The combined value should be the median of the inferences")

	require.Len(valueBundle.InfererValues, len(inferences.Inferences))
	for _, infererValue := range valueBundle.InfererValues {
		found := false
		for _, inference := range inferences.Inferences {
			if inference.Inferer == infererValue.Worker {
				found = true
				s.Require().True(inference.Value.Equal(infererValue.Value))
			}
		}
		s.Require().True(found, "Inference not found in the result")
	}
}
