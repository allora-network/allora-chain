package inference_synthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestConvertValueBundleToNetworkLossesByWorker() {
	require := s.Require()
	valueBundle := types.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*types.WorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.1")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
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

	result, err := inference_synthesis.ComputeAndBuildEMRegret(lossA, lossB, previous, alpha, blockHeight, false)
	require.NoError(err)

	expected, err := alloraMath.NewDecFromString("210")
	require.NoError(err)

	require.True(alloraMath.InDelta(expected, result.Value, alloraMath.MustNewDecFromString("0.0001")))
	require.Equal(blockHeight, result.BlockHeight)
}

func (s *InferenceSynthesisTestSuite) TestGetCalcSetNetworkRegretsTwoWorkers() {
	require := s.Require()
	k := s.emissionsKeeper

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	valueBundle := types.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("500"),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
		InfererValues: []*types.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
		},
	}
	blockHeight := int64(42)
	nonce := types.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")
	topicId := uint64(1)

	timestampedValue := types.TimestampedValue{
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

	err := inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		valueBundle,
		nonce,
		alpha,
	)
	require.NoError(err)

	bothAccs := []string{worker1, worker2}
	expected := alloraMath.MustNewDecFromString("210")
	expectedOneIn := alloraMath.MustNewDecFromString("180")

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
			lastRegret, noPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, acc, accInner)
			require.NoError(err)
			require.True(alloraMath.InDelta(expectedOneIn, lastRegret.Value, alloraMath.MustNewDecFromString("0.0001")))
			require.False(noPriorRegret)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestGetCalcSetNetworkRegretsThreeWorkers() {
	require := s.Require()
	k := s.emissionsKeeper

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	valueBundle := types.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("500"),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
		InfererValues: []*types.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker3, Value: alloraMath.MustNewDecFromString("200")},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker3, Value: alloraMath.MustNewDecFromString("200")},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("200")},
			{Worker: worker3, Value: alloraMath.MustNewDecFromString("200")},
		},
	}
	blockHeight := int64(42)
	nonce := types.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")
	topicId := uint64(1)

	timestampedValue := types.TimestampedValue{
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
	)
	require.NoError(err)

	allWorkerAccs := []string{worker1, worker2, worker3}
	expected := alloraMath.MustNewDecFromString("210")
	// expectedOneIn := alloraMath.MustNewDecFromString("180")

	for _, workerAcc := range allWorkerAccs {
		lastRegret, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, workerAcc)
		require.NoError(err)
		require.True(alloraMath.InDelta(expected, lastRegret.Value, alloraMath.MustNewDecFromString("0.0001")))
		require.False(noPriorRegret)

		lastRegret, noPriorRegret, err = k.GetForecasterNetworkRegret(s.ctx, topicId, workerAcc)
		require.NoError(err)
		require.False(noPriorRegret)

		for _, innerWorkerAcc := range allWorkerAccs {
			lastRegret, noPriorRegret, err = k.GetOneInForecasterNetworkRegret(s.ctx, topicId, workerAcc, innerWorkerAcc)
			require.NoError(err)
			require.False(noPriorRegret)
		}
	}
}
