package inferencesynthesis_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestRunningWeightedAvgUpdate() {
	tests := []struct {
		name                string
		initialWeightedLoss inferencesynthesis.RunningWeightedLoss
		nextWeight          inferencesynthesis.Weight
		nextValue           inferencesynthesis.Weight
		expectedLoss        inferencesynthesis.RunningWeightedLoss
		expectedErr         error
	}{
		{
			name:                "normal operation",
			initialWeightedLoss: inferencesynthesis.RunningWeightedLoss{UnnormalizedWeightedLoss: alloraMath.MustNewDecFromString("0.5"), SumWeight: alloraMath.MustNewDecFromString("1.0")},
			nextWeight:          alloraMath.MustNewDecFromString("1.0"),
			nextValue:           alloraMath.MustNewDecFromString("2.0"),
			expectedLoss:        inferencesynthesis.RunningWeightedLoss{UnnormalizedWeightedLoss: alloraMath.MustNewDecFromString("2.5"), SumWeight: alloraMath.MustNewDecFromString("2.0")},
			expectedErr:         nil,
		},
		{
			name:                "simple example",
			initialWeightedLoss: inferencesynthesis.RunningWeightedLoss{UnnormalizedWeightedLoss: alloraMath.ZeroDec(), SumWeight: alloraMath.ZeroDec()},
			nextWeight:          alloraMath.MustNewDecFromString("1.0"),
			nextValue:           alloraMath.MustNewDecFromString("0.1"),
			expectedLoss:        inferencesynthesis.RunningWeightedLoss{UnnormalizedWeightedLoss: alloraMath.MustNewDecFromString("0.1"), SumWeight: alloraMath.MustNewDecFromString("1.0")},
			expectedErr:         nil,
		},
		{
			name:                "simple example2",
			initialWeightedLoss: inferencesynthesis.RunningWeightedLoss{UnnormalizedWeightedLoss: alloraMath.ZeroDec(), SumWeight: alloraMath.ZeroDec()},
			nextWeight:          alloraMath.MustNewDecFromString("1.0"),
			nextValue:           alloraMath.MustNewDecFromString("0.2"),
			expectedLoss:        inferencesynthesis.RunningWeightedLoss{UnnormalizedWeightedLoss: alloraMath.MustNewDecFromString("0.2"), SumWeight: alloraMath.MustNewDecFromString("1.0")},
			expectedErr:         nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			updatedLoss, err := inferencesynthesis.RunningWeightedAvgUpdate(
				&tc.initialWeightedLoss,
				tc.nextWeight,
				tc.nextValue,
			)
			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr, "Error should match the expected error")
			} else {
				s.Require().NoError(err, "No error expected but got one")
				inDelta, err := alloraMath.InDelta(tc.expectedLoss.UnnormalizedWeightedLoss, updatedLoss.UnnormalizedWeightedLoss, alloraMath.MustNewDecFromString("0.00001"))
				s.Require().NoError(err)
				s.Require().True(inDelta, "UnnormalizedWeightedLoss should match the expected value within epsilon")
				s.Require().Equal(tc.expectedLoss.SumWeight, updatedLoss.SumWeight, "Sum of weights should match the expected value")
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) getTestCasesOneWorker() []struct {
	name            string
	stakesByReputer map[inferencesynthesis.Worker]cosmosMath.Int
	reportedLosses  emissions.ReputerValueBundles
	epsilon         alloraMath.Dec
	expectedOutput  emissions.ValueBundle
	expectedError   error
} {
	valueBundle := &emissions.ValueBundle{
		TopicId: uint64(1),
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.addrsStr[1],
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature := s.signValueBundle(valueBundle, s.privKeys[1])
	return []struct {
		name            string
		stakesByReputer map[inferencesynthesis.Worker]cosmosMath.Int
		reportedLosses  emissions.ReputerValueBundles
		epsilon         alloraMath.Dec
		expectedOutput  emissions.ValueBundle
		expectedError   error
	}{
		{
			name: "simple one reputer combined loss",
			stakesByReputer: map[inferencesynthesis.Worker]cosmosMath.Int{
				s.addrsStr[1]: inferencesynthesis.CosmosIntOneE18(), // 1 token
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: valueBundle,
						Signature:   signature,
						Pubkey:      s.pubKeyHexStr[1],
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				TopicId: uint64(1),
				Reputer: s.addrsStr[1],
				ReputerRequestNonce: &emissions.ReputerRequestNonce{
					ReputerNonce: &emissions.Nonce{BlockHeight: 100},
				},
				ExtraData:     nil,
				CombinedValue: alloraMath.MustNewDecFromString("0.1587401051968199"),
				NaiveValue:    alloraMath.MustNewDecFromString("0.1587401051968199"),
				InfererValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				ForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneInForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneOutInfererForecasterValues: nil,
			},
			expectedError: nil,
		},
	}
}

func (s *InferenceSynthesisTestSuite) getTestCasesTwoWorkers() []struct {
	name            string
	stakesByReputer map[inferencesynthesis.Worker]cosmosMath.Int
	reportedLosses  emissions.ReputerValueBundles
	epsilon         alloraMath.Dec
	expectedOutput  emissions.ValueBundle
	expectedError   error
} {
	valueBundle1 := emissions.ValueBundle{
		TopicId: uint64(1),
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.addrsStr[1],
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature1 := s.signValueBundle(&valueBundle1, s.privKeys[1])

	valueBundle2 := emissions.ValueBundle{
		ExtraData: nil,
		TopicId:   uint64(1),
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.addrsStr[2],
		CombinedValue: alloraMath.MustNewDecFromString("0.2"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.2"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.addrsStr[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.addrsStr[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature2 := s.signValueBundle(&valueBundle2, s.privKeys[2])
	return []struct {
		name            string
		stakesByReputer map[inferencesynthesis.Worker]cosmosMath.Int
		reportedLosses  emissions.ReputerValueBundles
		epsilon         alloraMath.Dec
		expectedOutput  emissions.ValueBundle
		expectedError   error
	}{
		{
			name: "simple two reputer combined loss",
			stakesByReputer: map[inferencesynthesis.Worker]cosmosMath.Int{
				s.addrsStr[1]: inferencesynthesis.CosmosIntOneE18(),           // 1 token
				s.addrsStr[2]: inferencesynthesis.CosmosIntOneE18().MulRaw(2), // 2 tokens
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: &valueBundle1,
						Signature:   signature1,
						Pubkey:      s.pubKeyHexStr[1],
					},
					{
						ValueBundle: &valueBundle2,
						Signature:   signature2,
						Pubkey:      s.pubKeyHexStr[2],
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				TopicId: uint64(1),
				Reputer: s.addrsStr[1],
				ReputerRequestNonce: &emissions.ReputerRequestNonce{
					ReputerNonce: &emissions.Nonce{BlockHeight: 100},
				},
				ExtraData:     nil,
				CombinedValue: alloraMath.MustNewDecFromString("0.166666666"),
				NaiveValue:    alloraMath.MustNewDecFromString("0.166666666"),
				InfererValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.addrsStr[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				ForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.addrsStr[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.addrsStr[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.addrsStr[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneInForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.addrsStr[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.addrsStr[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneOutInfererForecasterValues: nil,
			},
			expectedError: nil,
		},
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLosses() {
	tests := s.getTestCasesTwoWorkers()

	topicId := uint64(1)
	block := int64(100)
	require := s.Require()

	for _, tc := range tests {
		s.Run(tc.name, func() {
			output, err := inferencesynthesis.CalcNetworkLosses(topicId, block, tc.stakesByReputer, tc.reportedLosses)
			if tc.expectedError != nil {
				require.Error(err)
				require.EqualError(err, tc.expectedError.Error())
			} else {
				require.NoError(err)
				require.True(alloraMath.InDelta(tc.expectedOutput.CombinedValue, output.CombinedValue, alloraMath.MustNewDecFromString("0.00001")))
				require.True(alloraMath.InDelta(tc.expectedOutput.NaiveValue, output.NaiveValue, alloraMath.MustNewDecFromString("0.00001")))

				if tc.expectedOutput.InfererValues != nil {
					require.Len(output.InfererValues, len(tc.expectedOutput.InfererValues))
					for i, expectedValue := range tc.expectedOutput.InfererValues {
						require.True(alloraMath.InDelta(expectedValue.Value, output.InfererValues[i].Value, alloraMath.MustNewDecFromString("0.00001")))
					}
				}
				if tc.expectedOutput.ForecasterValues != nil {
					require.Len(output.ForecasterValues, len(tc.expectedOutput.ForecasterValues))
					for i, expectedValue := range tc.expectedOutput.ForecasterValues {
						require.True(alloraMath.InDelta(expectedValue.Value, output.ForecasterValues[i].Value, alloraMath.MustNewDecFromString("0.00001")))
					}
				}
				if tc.expectedOutput.OneOutInfererValues != nil {
					require.Len(output.OneOutInfererValues, len(tc.expectedOutput.OneOutInfererValues))
					for i, expectedValue := range tc.expectedOutput.OneOutInfererValues {
						require.True(alloraMath.InDelta(expectedValue.Value, output.OneOutInfererValues[i].Value, alloraMath.MustNewDecFromString("0.00001")))
					}
				}
				if tc.expectedOutput.OneOutForecasterValues != nil {
					require.Len(output.OneOutForecasterValues, len(tc.expectedOutput.OneOutForecasterValues))
					for i, expectedValue := range tc.expectedOutput.OneOutForecasterValues {
						require.True(alloraMath.InDelta(expectedValue.Value, output.OneOutForecasterValues[i].Value, alloraMath.MustNewDecFromString("0.00001")))
					}
				}
				if tc.expectedOutput.OneInForecasterValues != nil {
					require.Len(output.OneInForecasterValues, len(tc.expectedOutput.OneInForecasterValues))
					for i, expectedValue := range tc.expectedOutput.OneInForecasterValues {
						require.True(alloraMath.InDelta(expectedValue.Value, output.OneInForecasterValues[i].Value, alloraMath.MustNewDecFromString("0.00001")))
					}
				}
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLossesFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch301Get := epochGet[301]
	topicId := uint64(1)
	blockHeight := int64(301)

	reputer0 := s.addrsStr[0]
	reputer1 := s.addrsStr[1]
	reputer2 := s.addrsStr[2]
	reputer3 := s.addrsStr[3]
	reputer4 := s.addrsStr[4]
	reputers := []testutil.ReputerKey{
		{
			Address:    reputer0,
			PrivateKey: s.privKeys[0],
			PubKeyHex:  s.pubKeyHexStr[0],
		},
		{
			Address:    reputer1,
			PrivateKey: s.privKeys[1],
			PubKeyHex:  s.pubKeyHexStr[1],
		},
		{
			Address:    reputer2,
			PrivateKey: s.privKeys[2],
			PubKeyHex:  s.pubKeyHexStr[2],
		},
		{
			Address:    reputer3,
			PrivateKey: s.privKeys[3],
			PubKeyHex:  s.pubKeyHexStr[3],
		},
		{
			Address:    reputer4,
			PrivateKey: s.privKeys[4],
			PubKeyHex:  s.pubKeyHexStr[4],
		},
	}

	inferer0 := s.addrsStr[5]
	inferer1 := s.addrsStr[6]
	inferer2 := s.addrsStr[7]
	inferer3 := s.addrsStr[8]
	inferer4 := s.addrsStr[9]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.addrsStr[10]
	forecaster1 := s.addrsStr[11]
	forecaster2 := s.addrsStr[12]
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	cosmosOneE18Dec, err := alloraMath.NewDecFromSdkInt(cosmosOneE18)
	s.Require().NoError(err)

	reputer0Stake, err := epoch301Get("reputer_stake_0").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer0StakeInt, err := reputer0Stake.BigInt()
	s.Require().NoError(err)
	reputer1Stake, err := epoch301Get("reputer_stake_1").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer1StakeInt, err := reputer1Stake.BigInt()
	s.Require().NoError(err)
	reputer2Stake, err := epoch301Get("reputer_stake_2").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer2StakeInt, err := reputer2Stake.BigInt()
	s.Require().NoError(err)
	reputer3Stake, err := epoch301Get("reputer_stake_3").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer3StakeInt, err := reputer3Stake.BigInt()
	s.Require().NoError(err)
	reputer4Stake, err := epoch301Get("reputer_stake_4").Mul(cosmosOneE18Dec)
	s.Require().NoError(err)
	reputer4StakeInt, err := reputer4Stake.BigInt()
	s.Require().NoError(err)

	var stakesByReputer = map[string]cosmosMath.Int{
		reputer0: cosmosMath.NewIntFromBigInt(reputer0StakeInt),
		reputer1: cosmosMath.NewIntFromBigInt(reputer1StakeInt),
		reputer2: cosmosMath.NewIntFromBigInt(reputer2StakeInt),
		reputer3: cosmosMath.NewIntFromBigInt(reputer3StakeInt),
		reputer4: cosmosMath.NewIntFromBigInt(reputer4StakeInt),
	}

	reportedLosses, err := testutil.GetReputersDataFromCsv(
		topicId,
		blockHeight,
		infererAddresses,
		forecasterAddresses,
		reputers,
		epoch301Get,
	)
	s.Require().NoError(err)

	networkLosses, err := inferencesynthesis.CalcNetworkLosses(topicId, blockHeight, stakesByReputer, reportedLosses)
	s.Require().NoError(err)

	expectedNetworkLosses, err := testutil.GetNetworkLossFromCsv(
		topicId,
		blockHeight,
		infererAddresses,
		forecasterAddresses,
		reputer0,
		epoch301Get,
	)
	s.Require().NoError(err)

	testutil.InEpsilon5(s.T(), expectedNetworkLosses.CombinedValue, networkLosses.CombinedValue.String())
	testutil.InEpsilon5(s.T(), expectedNetworkLosses.NaiveValue, networkLosses.NaiveValue.String())
	s.Require().Len(networkLosses.InfererValues, len(expectedNetworkLosses.InfererValues))
	for _, expectedValue := range expectedNetworkLosses.InfererValues {
		found := false
		for _, workerAttributedValue := range networkLosses.InfererValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
	s.Require().Len(networkLosses.ForecasterValues, len(expectedNetworkLosses.ForecasterValues))
	for _, expectedValue := range expectedNetworkLosses.ForecasterValues {
		found := false
		for _, workerAttributedValue := range networkLosses.ForecasterValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
	s.Require().Len(networkLosses.OneOutInfererForecasterValues, len(expectedNetworkLosses.OneOutInfererForecasterValues))
	for _, expectedValue := range expectedNetworkLosses.OneOutInfererForecasterValues {
		found := false
		for _, workerAttributedValue := range networkLosses.OneOutInfererForecasterValues {
			if workerAttributedValue.Forecaster == expectedValue.Forecaster {
				found = true
				s.Require().Len(workerAttributedValue.OneOutInfererValues, len(expectedValue.OneOutInfererValues))
				for _, expectedOneOutInfererValue := range expectedValue.OneOutInfererValues {
					foundOneOutInferer := false
					for _, oneOutInfererValue := range workerAttributedValue.OneOutInfererValues {
						if oneOutInfererValue.Worker == expectedOneOutInfererValue.Worker {
							foundOneOutInferer = true
							testutil.InEpsilon5(s.T(), expectedOneOutInfererValue.Value, oneOutInfererValue.Value.String())
						}
					}
					s.Require().True(foundOneOutInferer)
				}
			}
		}
		s.Require().True(found)
	}
	s.Require().Len(networkLosses.OneOutInfererValues, len(expectedNetworkLosses.OneOutInfererValues))
	for _, expectedValue := range expectedNetworkLosses.OneOutInfererValues {
		found := false
		for _, workerAttributedValue := range networkLosses.OneOutInfererValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
	s.Require().Len(networkLosses.OneOutForecasterValues, len(expectedNetworkLosses.OneOutForecasterValues))
	for _, expectedValue := range expectedNetworkLosses.OneOutForecasterValues {
		found := false
		for _, workerAttributedValue := range networkLosses.OneOutForecasterValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
	s.Require().Len(networkLosses.OneInForecasterValues, len(expectedNetworkLosses.OneInForecasterValues))
	for _, expectedValue := range expectedNetworkLosses.OneInForecasterValues {
		found := false
		for _, workerAttributedValue := range networkLosses.OneInForecasterValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLossesCombined() {
	tests := append(s.getTestCasesOneWorker(), s.getTestCasesTwoWorkers()...)

	topicId := uint64(1)
	block := int64(100)
	require := s.Require()

	for _, tc := range tests {
		s.Run(tc.name, func() {
			output, err := inferencesynthesis.CalcNetworkLosses(topicId, block, tc.stakesByReputer, tc.reportedLosses)
			if tc.expectedError != nil {
				require.Error(err)
				require.EqualError(err, tc.expectedError.Error())
			} else {
				require.NoError(err)

				// Verify the length of each attribute in the ValueBundle
				require.Len(output.InfererValues, len(tc.expectedOutput.InfererValues), "Mismatch in number of InfererValues")
				require.Len(output.ForecasterValues, len(tc.expectedOutput.ForecasterValues), "Mismatch in number of ForecasterValues")
				require.Len(output.OneInForecasterValues, len(tc.expectedOutput.OneInForecasterValues), "Mismatch in number of OneInForecasterValues")
				require.Len(output.OneOutInfererValues, len(tc.expectedOutput.OneOutInfererValues), "Mismatch in number of OneOutInfererValues")
				require.Len(output.OneOutForecasterValues, len(tc.expectedOutput.OneOutForecasterValues), "Mismatch in number of OneOutForecasterValues")
			}
		})
	}
}
