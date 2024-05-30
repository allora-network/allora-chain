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

	result, err := inference_synthesis.ComputeAndBuildEMRegret(
		lossA,
		lossB,
		previous,
		alpha,
		blockHeight,
		false,
	)
	require.NoError(err)

	expected, err := alloraMath.NewDecFromString("210")
	require.NoError(err)

	require.True(
		alloraMath.InDelta(expected, result.Value, alloraMath.MustNewDecFromString("0.0001")),
	)
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

	worker3LastRegret, worker3NoPriorRegret, err := k.GetInfererNetworkRegret(
		s.ctx,
		topicId,
		worker3,
	)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetForecasterNetworkRegret(
		s.ctx,
		topicId,
		worker3,
	)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(
		s.ctx,
		topicId,
		worker3,
		worker1,
	)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(
		s.ctx,
		topicId,
		worker3,
		worker2,
	)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	worker3LastRegret, worker3NoPriorRegret, err = k.GetOneInForecasterNetworkRegret(
		s.ctx,
		topicId,
		worker3,
		worker3,
	)
	require.NoError(err)
	require.Equal(worker3LastRegret.Value, alloraMath.ZeroDec())
	require.True(worker3NoPriorRegret)

	for _, acc := range bothAccs {
		lastRegret, noPriorRegret, err := k.GetInfererNetworkRegret(s.ctx, topicId, acc)
		require.NoError(err)
		require.True(
			alloraMath.InDelta(
				expected,
				lastRegret.Value,
				alloraMath.MustNewDecFromString("0.0001"),
			),
		)
		require.False(noPriorRegret)

		lastRegret, noPriorRegret, err = k.GetForecasterNetworkRegret(s.ctx, topicId, acc)
		require.NoError(err)
		require.True(
			alloraMath.InDelta(
				expected,
				lastRegret.Value,
				alloraMath.MustNewDecFromString("0.0001"),
			),
		)
		require.False(noPriorRegret)

		for _, accInner := range bothAccs {
			lastRegret, noPriorRegret, err = k.GetOneInForecasterNetworkRegret(
				s.ctx,
				topicId,
				acc,
				accInner,
			)
			require.NoError(err)
			require.True(
				alloraMath.InDelta(
					expectedOneIn,
					lastRegret.Value,
					alloraMath.MustNewDecFromString("0.0001"),
				),
			)
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
		require.True(
			alloraMath.InDelta(
				expected,
				lastRegret.Value,
				alloraMath.MustNewDecFromString("0.0001"),
			),
		)
		require.False(noPriorRegret)

		lastRegret, noPriorRegret, err = k.GetForecasterNetworkRegret(s.ctx, topicId, workerAcc)
		require.NoError(err)
		require.False(noPriorRegret)

		for _, innerWorkerAcc := range allWorkerAccs {
			lastRegret, noPriorRegret, err = k.GetOneInForecasterNetworkRegret(
				s.ctx,
				topicId,
				workerAcc,
				innerWorkerAcc,
			)
			require.NoError(err)
			require.False(noPriorRegret)
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
	nonce := types.Nonce{BlockHeight: blockHeight}
	alpha := alloraMath.MustNewDecFromString("0.1")

	worker0 := "worker0"
	worker1 := "worker1"
	worker2 := "worker2"

	networkLossesValueBundle0 := types.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*types.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.2")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
	}

	networkLossesValueBundle1 := types.ValueBundle{
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*types.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{Worker: worker0, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker1, Value: alloraMath.MustNewDecFromString("0.3")},
			{Worker: worker2, Value: alloraMath.MustNewDecFromString("0.4")},
		},
	}

	resetRegrets := func() {
		timestampedValue0_1 := types.TimestampedValue{
			BlockHeight: blockHeight,
			Value:       alloraMath.MustNewDecFromString("0.1"),
		}

		timestampedValue0_2 := types.TimestampedValue{
			BlockHeight: blockHeight,
			Value:       alloraMath.MustNewDecFromString("0.2"),
		}

		timestampedValue0_3 := types.TimestampedValue{
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
	)
	require.NoError(err)

	// Record resulting regrets

	infererRegret0_0, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker0)
	require.False(notFound)
	require.NoError(err)
	infererRegret0_1, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker1)
	require.False(notFound)
	require.NoError(err)
	infererRegret0_2, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker2)
	require.False(notFound)
	require.NoError(err)

	forecasterRegret0_0, notFound, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker0)
	require.False(notFound)
	require.NoError(err)
	forecasterRegret0_1, notFound, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker1)
	require.False(notFound)
	require.NoError(err)
	forecasterRegret0_2, notFound, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker2)
	require.False(notFound)
	require.NoError(err)

	// Test 1

	resetRegrets()

	err = inference_synthesis.GetCalcSetNetworkRegrets(
		s.ctx,
		k,
		topicId,
		networkLossesValueBundle1,
		nonce,
		alpha,
	)
	require.NoError(err)

	// Record resulting regrets

	infererRegret1_0, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker0)
	require.False(notFound)
	require.NoError(err)
	infererRegret1_1, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker1)
	require.False(notFound)
	require.NoError(err)
	infererRegret1_2, notFound, err := k.GetInfererNetworkRegret(s.ctx, topicId, worker2)
	require.False(notFound)
	require.NoError(err)

	forecasterRegret1_0, notFound, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker0)
	require.False(notFound)
	require.NoError(err)
	forecasterRegret1_1, notFound, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker1)
	require.False(notFound)
	require.NoError(err)
	forecasterRegret1_2, notFound, err := k.GetForecasterNetworkRegret(s.ctx, topicId, worker2)
	require.False(notFound)
	require.NoError(err)

	// Test

	require.True(infererRegret0_0.Value.Gt(infererRegret1_0.Value))
	require.Equal(infererRegret0_1.Value, infererRegret1_1.Value)
	require.Equal(infererRegret0_2.Value, infererRegret1_2.Value)

	require.True(forecasterRegret0_0.Value.Gt(forecasterRegret1_0.Value))
	require.Equal(forecasterRegret0_1.Value, forecasterRegret1_1.Value)
	require.Equal(forecasterRegret0_2.Value, forecasterRegret1_2.Value)
}
