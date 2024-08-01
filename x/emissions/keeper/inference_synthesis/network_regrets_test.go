package inference_synthesis_test

import (
	"context"
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestConvertValueBundleToNetworkLossesByWorker() {
	require := s.Require()
	valueBundle := emissionstypes.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		OneOutForecasterValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
	}

	result := inference_synthesis.ConvertValueBundleToNetworkLossesByWorker(valueBundle)

	// Check if CombinedLoss and NaiveLoss are correctly set
	require.Equal(alloraMath.MustNewDecFromString("0.1"), result.CombinedLoss)
	require.Equal(alloraMath.MustNewDecFromString("0.1"), result.NaiveLoss)

	// Check if each worker's losses are set correctly
	expectedLoss := alloraMath.MustNewDecFromString("0.1")
	expectedLoss2 := alloraMath.MustNewDecFromString("0.2")
	require.Equal(expectedLoss, result.InfererLosses["worker1"])
	require.Equal(expectedLoss2, result.InfererLosses["worker2"])
	require.Equal(expectedLoss, result.ForecasterLosses["worker1"])
	require.Equal(expectedLoss2, result.ForecasterLosses["worker2"])
	require.Equal(expectedLoss, result.OneOutInfererLosses["worker1"])
	require.Equal(expectedLoss2, result.OneOutInfererLosses["worker2"])
	require.Equal(expectedLoss, result.OneOutForecasterLosses["worker1"])
	require.Equal(expectedLoss2, result.OneOutForecasterLosses["worker2"])
	require.Equal(expectedLoss, result.OneInForecasterLosses["worker1"])
	require.Equal(expectedLoss2, result.OneInForecasterLosses["worker2"])
}

func (s *InferenceSynthesisTestSuite) TestComputeAndBuildEMRegret() {
	require := s.Require()

	alpha := alloraMath.MustNewDecFromString("0.1")
	lossA := alloraMath.MustNewDecFromString("500")
	lossB := alloraMath.MustNewDecFromString("200")
	previous := alloraMath.MustNewDecFromString("200")

	blockHeight := int64(123)

	result, err := inference_synthesis.ComputeAndBuildEMRegret(lossA, lossB, previous, alpha, blockHeight)
	require.NoError(err)

	expected, err := alloraMath.NewDecFromString("210")
	require.NoError(err)

	require.True(alloraMath.InDelta(expected, result.Value, alloraMath.MustNewDecFromString("0.0001")))
	require.Equal(blockHeight, result.BlockHeight)
}

func (s *InferenceSynthesisTestSuite) TestGetCalcSetNetworkRegretsTwoWorkers() {
	require := s.Require()
	k := s.emissionsKeeper

	topicId := uint64(2)
	// Create new topic
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
		InitialRegret:   alloraMath.MustNewDecFromString("0"),
	})
	s.Require().NoError(err)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	pNorm := alloraMath.MustNewDecFromString("0.1")
	cNorm := alloraMath.MustNewDecFromString("0.1")
	epsilon := alloraMath.MustNewDecFromString("0.0001")

	valueBundle := emissionstypes.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("500"),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
		},
	}
	blockHeight := int64(42)
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")

	timestampedValue := emissionstypes.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       alloraMath.MustNewDecFromString("200"),
	}

	k.SetInfererNetworkRegret(s.ctx, topicId, worker1, timestampedValue)
	k.SetInfererNetworkRegret(s.ctx, topicId, worker2, timestampedValue)
	k.SetForecasterNetworkRegret(s.ctx, topicId, worker1, timestampedValue)
	k.SetForecasterNetworkRegret(s.ctx, topicId, worker2, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker1, worker1, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker1, worker2, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker2, worker1, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker2, worker2, timestampedValue)

	// New potential participant should start with zero regret at this point since the initial regret in the topic is zero
	// It will be updated after the first regret calculation
	worker3LastRegret, worker3NoPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker3)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetForecasterNetworkRegret(s.ctx, topicId, worker3)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker1)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker2)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker3)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	err = inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		valueBundle,
		nonce,
		alpha,
		cNorm,
		pNorm,
		epsilon,
	)
	require.NoError(err)

	bothAccs := []string{worker1, worker2}
	expected := alloraMath.MustNewDecFromString("210")
	// expectedOneIn := alloraMath.MustNewDecFromString("180")

	// New potential participant should not start with zero regret since we already have participants with prior regrets which will
	// be used to calculate the initial regret in the topic
	worker3LastRegret, worker3NoPriorRegret, err = k.GetInfererNetworkRegret(s.ctx, topicId, worker3)
	require.NoError(err)
	require.NotEqual(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetForecasterNetworkRegret(s.ctx, topicId, worker3)
	require.NoError(err)
	require.NotEqual(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker1)
	require.NoError(err)
	require.NotEqual(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker2)
	require.NoError(err)
	require.NotEqual(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker3)
	require.NoError(err)
	require.NotEqual(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	for _, acc := range bothAccs {
		lastRegret, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, acc)
		require.NoError(err)
		require.True(alloraMath.InDelta(expected, lastRegret.Value, alloraMath.MustNewDecFromString("0.0001")))
		require.False(noPriorRegret)

		lastRegret, noPriorRegret, err = k.GetForecasterNetworkRegret(s.ctx, topicId, acc)
		require.NoError(err)
		require.True(alloraMath.InDelta(expected, lastRegret.Value, alloraMath.MustNewDecFromString("0.0001")))
		require.False(noPriorRegret)

		for _, accInner := range bothAccs {
			lastRegret, _, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, acc, accInner)
			require.NoError(err)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetCalcSetNetworkRegretsThreeWorkers() {
	require := s.Require()
	k := s.emissionsKeeper

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	pNorm := alloraMath.MustNewDecFromString("0.1")
	cNorm := alloraMath.MustNewDecFromString("0.1")
	epsilon := alloraMath.MustNewDecFromString("0.0001")

	valueBundle := emissionstypes.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("500"),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker3, Value: alloraMath.MustNewDecFromString("200")},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker3, Value: alloraMath.MustNewDecFromString("200")},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker3, Value: alloraMath.MustNewDecFromString("200")},
		},
	}
	blockHeight := int64(42)
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")
	topicId := uint64(1)

	timestampedValue := emissionstypes.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       alloraMath.MustNewDecFromString("200"),
	}

	k.SetInfererNetworkRegret(s.ctx, topicId, worker1, timestampedValue)
	k.SetInfererNetworkRegret(s.ctx, topicId, worker2, timestampedValue)
	k.SetInfererNetworkRegret(s.ctx, topicId, worker3, timestampedValue)

	k.SetForecasterNetworkRegret(s.ctx, topicId, worker1, timestampedValue)
	k.SetForecasterNetworkRegret(s.ctx, topicId, worker2, timestampedValue)
	k.SetForecasterNetworkRegret(s.ctx, topicId, worker2, timestampedValue)

	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker1, worker1, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker1, worker2, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker1, worker3, timestampedValue)

	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker2, worker1, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker2, worker2, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker2, worker3, timestampedValue)

	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker1, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker2, timestampedValue)
	k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3, worker3, timestampedValue)

	err := inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		valueBundle,
		nonce,
		alpha,
		cNorm,
		pNorm,
		epsilon,
	)
	require.NoError(err)

	allWorkerAccs := []string{worker1, worker2, worker3}
	expected := alloraMath.MustNewDecFromString("210")
	// expectedOneIn := alloraMath.MustNewDecFromString("180")

	for _, workerAcc := range allWorkerAccs {
		lastRegret, _, err := k.GetInfererNetworkRegret(s.ctx, topicId, workerAcc)
		require.NoError(err)
		require.True(alloraMath.InDelta(expected, lastRegret.Value, alloraMath.MustNewDecFromString("0.0001")))

		lastRegret, _, err = k.GetForecasterNetworkRegret(s.ctx, topicId, workerAcc)
		require.NoError(err)

		for _, innerWorkerAcc := range allWorkerAccs {
			lastRegret, _, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, workerAcc, innerWorkerAcc)
			require.NoError(err)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetCalcSetNetworkRegretsFromCsv() {
	require := s.Require()
	k := s.emissionsKeeper
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epochPrevGet := epochGet[300]
	epoch301Get := epochGet[301]
	topicId := uint64(1)
	blockHeight := int64(1003)
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	epsilon := alloraMath.MustNewDecFromString("1e-4")

	inferer0 := s.addrs[0].String()
	inferer1 := s.addrs[1].String()
	inferer2 := s.addrs[2].String()
	inferer3 := s.addrs[3].String()
	inferer4 := s.addrs[4].String()
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrs[5].String()
	forecaster1 := s.addrs[6].String()
	forecaster2 := s.addrs[7].String()
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	err := testutil.SetRegretsFromPreviousEpoch(s.ctx, s.emissionsKeeper, topicId, blockHeight, infererAddresses, forecasterAddresses, epochPrevGet)
	require.NoError(err)

	networkLosses, err := testutil.GetNetworkLossFromCsv(
		topicId,
		infererAddresses,
		forecasterAddresses,
		epoch301Get,
	)
	s.Require().NoError(err)

	err = inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		k,
		topicId,
		networkLosses,
		nonce,
		alpha,
		cNorm,
		pNorm,
		epsilon,
	)
	require.NoError(err)

	checkRegret := func(worker string, expected alloraMath.Dec, getter func(context.Context, uint64, string) (emissionstypes.TimestampedValue, bool, error)) {
		regret, _, err := getter(s.ctx, topicId, worker)
		require.NoError(err)
		testutil.InEpsilon5(s.T(), expected, regret.Value.String())
	}

	checkOneOutRegret := func(worker string, innerWorker string, expected alloraMath.Dec, getter func(context.Context, uint64, string, string) (emissionstypes.TimestampedValue, bool, error)) {
		regret, _, err := getter(s.ctx, topicId, worker, innerWorker)
		require.NoError(err)
		testutil.InEpsilon5(s.T(), expected, regret.Value.String())
	}

	for i := 0; i < len(infererAddresses); i++ {
		expectedRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v", i))
		checkRegret(infererAddresses[i], expectedRegret, k.GetInfererNetworkRegret)

		expectedRegret = epoch301Get(fmt.Sprintf("naive_inference_regret_worker_%v", i))
		checkRegret(infererAddresses[i], expectedRegret, k.GetNaiveInfererNetworkRegret)
	}

	for i := 0; i < len(forecasterAddresses); i++ {
		forecasterCsvIndex := i + 5
		expectedRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v", forecasterCsvIndex))
		checkRegret(forecasterAddresses[i], expectedRegret, k.GetForecasterNetworkRegret)
	}

	for i, inferer := range infererAddresses {
		for j, infererInner := range infererAddresses {
			expectedRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v_oneout_%v", j, i))
			checkOneOutRegret(inferer, infererInner, expectedRegret, k.GetOneOutInfererInfererNetworkRegret)
		}

		for l, forecaster := range forecasterAddresses {
			forecasterCsvIndex := l + 5
			expectedRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v_oneout_%v", forecasterCsvIndex, i))
			checkOneOutRegret(inferer, forecaster, expectedRegret, k.GetOneOutInfererForecasterNetworkRegret)
		}
	}

	for i, forecaster := range forecasterAddresses {
		forecasterCsvIndex := i + 5
		for j, inferer := range infererAddresses {
			expectedRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v_oneout_%v", j, forecasterCsvIndex))
			checkOneOutRegret(forecaster, inferer, expectedRegret, k.GetOneOutForecasterInfererNetworkRegret)
		}

		for z, forecasterInner := range forecasterAddresses {
			forecasterCsvIndex2 := z + 5
			expectedRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v_oneout_%v", forecasterCsvIndex2, forecasterCsvIndex))
			checkOneOutRegret(forecaster, forecasterInner, expectedRegret, k.GetOneOutForecasterForecasterNetworkRegret)
		}

		expectedOneInRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v_onein_%v", 5, i))
		checkOneOutRegret(forecaster, forecaster, expectedOneInRegret, k.GetOneInForecasterNetworkRegret)

		for l, inferer := range infererAddresses {
			expectedOneInRegret := epoch301Get(fmt.Sprintf("inference_regret_worker_%v_onein_%v", l, i))
			checkOneOutRegret(forecaster, inferer, expectedOneInRegret, k.GetOneInForecasterNetworkRegret)
		}
	}
}

// In this test we run two trials of calculating setting network regrets with different losses.
// We then compare the resulting regrets to see if the higher losses result in lower regrets.
func (s *InferenceSynthesisTestSuite) TestHigherLossesLowerRegret() {
	require := s.Require()
	k := s.emissionsKeeper

	topicId := uint64(1)
	blockHeight := int64(1003)
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")
	pNorm := alloraMath.MustNewDecFromString("0.1")
	cNorm := alloraMath.MustNewDecFromString("0.1")
	epsilon := alloraMath.MustNewDecFromString("0.0001")

	worker0 := "worker0"
	worker1 := "worker1"
	worker2 := "worker2"

	networkLossesValueBundle0 := emissionstypes.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
	}

	networkLossesValueBundle1 := emissionstypes.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
	}

	resetRegrets := func() {
		timestampedValue0_1 := emissionstypes.TimestampedValue{
			BlockHeight: blockHeight,
			Value:       alloraMath.MustNewDecFromString("0.1"),
		}

		timestampedValue0_2 := emissionstypes.TimestampedValue{
			BlockHeight: blockHeight,
			Value:       alloraMath.MustNewDecFromString("0.2"),
		}

		timestampedValue0_3 := emissionstypes.TimestampedValue{
			BlockHeight: blockHeight,
			Value:       alloraMath.MustNewDecFromString("0.3"),
		}

		k.SetInfererNetworkRegret(s.ctx, topicId, worker0, timestampedValue0_1)
		k.SetInfererNetworkRegret(s.ctx, topicId, worker1, timestampedValue0_2)
		k.SetInfererNetworkRegret(s.ctx, topicId, worker2, timestampedValue0_3)

		k.SetForecasterNetworkRegret(s.ctx, topicId, worker0, timestampedValue0_1)
		k.SetForecasterNetworkRegret(s.ctx, topicId, worker1, timestampedValue0_2)
		k.SetForecasterNetworkRegret(s.ctx, topicId, worker2, timestampedValue0_3)
	}

	// Test 0

	resetRegrets()

	err := inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		k,
		topicId,
		networkLossesValueBundle0,
		nonce,
		alpha,
		cNorm,
		pNorm,
		epsilon,
	)
	require.NoError(err)

	// Record resulting regrets

	infererRegret0_0, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker0)
	require.NoError(err)
	require.False(noPriorRegret)
	infererRegret0_1, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker1)
	require.NoError(err)
	require.False(noPriorRegret)
	infererRegret0_2, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker2)
	require.NoError(err)
	require.False(noPriorRegret)

	forecasterRegret0_0, noPriorRegret, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker0)
	require.NoError(err)
	require.False(noPriorRegret)
	forecasterRegret0_1, noPriorRegret, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker1)
	require.NoError(err)
	require.False(noPriorRegret)
	forecasterRegret0_2, noPriorRegret, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker2)
	require.NoError(err)
	require.False(noPriorRegret)

	// Test 1

	resetRegrets()

	err = inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		k,
		topicId,
		networkLossesValueBundle1,
		nonce,
		alpha,
		cNorm,
		pNorm,
		epsilon,
	)
	require.NoError(err)

	// Record resulting regrets

	infererRegret1_0, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker0)
	require.NoError(err)
	require.False(noPriorRegret)
	infererRegret1_1, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker1)
	require.NoError(err)
	require.False(noPriorRegret)
	infererRegret1_2, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker2)
	require.NoError(err)
	require.False(noPriorRegret)

	forecasterRegret1_0, noPriorRegret, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker0)
	require.NoError(err)
	require.False(noPriorRegret)
	forecasterRegret1_1, noPriorRegret, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker1)
	require.NoError(err)
	require.False(noPriorRegret)
	forecasterRegret1_2, noPriorRegret, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker2)
	require.NoError(err)
	require.False(noPriorRegret)

	// Test

	require.True(infererRegret0_0.Value.Gt(infererRegret1_0.Value))
	require.Equal(infererRegret0_1.Value, infererRegret1_1.Value)
	require.Equal(infererRegret0_2.Value, infererRegret1_2.Value)

	require.True(forecasterRegret0_0.Value.Gt(forecasterRegret1_0.Value))
	require.Equal(forecasterRegret0_1.Value, forecasterRegret1_1.Value)
	require.Equal(forecasterRegret0_2.Value, forecasterRegret1_2.Value)
}

func (s *InferenceSynthesisTestSuite) TestCalcTopicInitialRegret() {
	require := s.Require()

	regrets := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.6445506208021189"),
		alloraMath.MustNewDecFromString("1.0216386898413485"),
		alloraMath.MustNewDecFromString("0.6092049398135028"),
		alloraMath.MustNewDecFromString("0.6971588004566455"),
		alloraMath.MustNewDecFromString("0.9030751421888253"),
		alloraMath.MustNewDecFromString("0.8219035038858344"),
	}
	cNorm := alloraMath.MustNewDecFromString("0.75")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	epsilon := alloraMath.MustNewDecFromString("0.0001")

	calculatedInitialRegret, err := inference_synthesis.CalcTopicInitialRegret(regrets, epsilon, pNorm, cNorm)
	require.NoError(err)
	testutil.InEpsilon5(s.T(), calculatedInitialRegret, "0.2869843501503934597325669544281814")
}
