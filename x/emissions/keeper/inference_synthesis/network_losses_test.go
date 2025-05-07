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
		Reputer:       s.AddrsStr()[1],
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature := s.signValueBundle(valueBundle, s.PrivKeys()[1])
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
				s.AddrsStr()[1]: inferencesynthesis.CosmosIntOneE18(), // 1 token
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: valueBundle,
						Signature:   signature,
						Pubkey:      s.PubKeyHexStr()[1],
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				TopicId: uint64(1),
				Reputer: s.AddrsStr()[1],
				ReputerRequestNonce: &emissions.ReputerRequestNonce{
					ReputerNonce: &emissions.Nonce{BlockHeight: 100},
				},
				ExtraData:     nil,
				CombinedValue: alloraMath.MustNewDecFromString("0.1587401051968199"),
				NaiveValue:    alloraMath.MustNewDecFromString("0.1587401051968199"),
				InfererValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				ForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.1587401051968199"),
					},
				},
				OneInForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
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
		Reputer:       s.AddrsStr()[1],
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature1 := s.signValueBundle(&valueBundle1, s.PrivKeys()[1])

	valueBundle2 := emissions.ValueBundle{
		ExtraData: nil,
		TopicId:   uint64(1),
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.AddrsStr()[2],
		CombinedValue: alloraMath.MustNewDecFromString("0.2"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.2"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature2 := s.signValueBundle(&valueBundle2, s.PrivKeys()[2])
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
				s.AddrsStr()[1]: inferencesynthesis.CosmosIntOneE18(),           // 1 token
				s.AddrsStr()[2]: inferencesynthesis.CosmosIntOneE18().MulRaw(2), // 2 tokens
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: &valueBundle1,
						Signature:   signature1,
						Pubkey:      s.PubKeyHexStr()[1],
					},
					{
						ValueBundle: &valueBundle2,
						Signature:   signature2,
						Pubkey:      s.PubKeyHexStr()[2],
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				TopicId: uint64(1),
				Reputer: s.AddrsStr()[1],
				ReputerRequestNonce: &emissions.ReputerRequestNonce{
					ReputerNonce: &emissions.Nonce{BlockHeight: 100},
				},
				ExtraData:     nil,
				CombinedValue: alloraMath.MustNewDecFromString("0.166666666"),
				NaiveValue:    alloraMath.MustNewDecFromString("0.166666666"),
				InfererValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				ForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneInForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.166666666"),
					},
				},
				OneOutInfererForecasterValues: nil,
			},
			expectedError: nil,
		},
	}
}

func (s *InferenceSynthesisTestSuite) getTestCasesTwoWorkersBadTopicId() []struct {
	name            string
	stakesByReputer map[inferencesynthesis.Worker]cosmosMath.Int
	reportedLosses  emissions.ReputerValueBundles
	epsilon         alloraMath.Dec
	expectedOutput  emissions.ValueBundle
	expectedError   error
} {
	valueBundle1 := emissions.ValueBundle{
		TopicId: uint64(1234), // wrong topic id - this bundle should be ignored
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.AddrsStr()[1],
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature1 := s.signValueBundle(&valueBundle1, s.PrivKeys()[1])

	valueBundle2 := emissions.ValueBundle{
		ExtraData: nil,
		TopicId:   uint64(1),
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.AddrsStr()[2],
		CombinedValue: alloraMath.MustNewDecFromString("0.2"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.2"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature2 := s.signValueBundle(&valueBundle2, s.PrivKeys()[2])
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
				s.AddrsStr()[1]: inferencesynthesis.CosmosIntOneE18(),           // 1 token
				s.AddrsStr()[2]: inferencesynthesis.CosmosIntOneE18().MulRaw(2), // 2 tokens
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: &valueBundle1,
						Signature:   signature1,
						Pubkey:      s.PubKeyHexStr()[1],
					},
					{
						ValueBundle: &valueBundle2,
						Signature:   signature2,
						Pubkey:      s.PubKeyHexStr()[2],
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				TopicId: uint64(1),
				Reputer: s.AddrsStr()[1],
				ReputerRequestNonce: &emissions.ReputerRequestNonce{
					ReputerNonce: &emissions.Nonce{BlockHeight: 100},
				},
				ExtraData:     nil,
				CombinedValue: alloraMath.MustNewDecFromString("0.2"),
				NaiveValue:    alloraMath.MustNewDecFromString("0.2"),
				InfererValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				ForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneInForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneOutInfererForecasterValues: nil,
			},
			expectedError: nil,
		},
	}
}

func (s *InferenceSynthesisTestSuite) getTestCasesTwoWorkersBadBlockHeight() []struct {
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
			ReputerNonce: &emissions.Nonce{BlockHeight: 1234}, // bad block height - this bundle should be ignored
		},
		Reputer:       s.AddrsStr()[1],
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewDecFromString("0.1"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.1"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.1"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature1 := s.signValueBundle(&valueBundle1, s.PrivKeys()[1])

	valueBundle2 := emissions.ValueBundle{
		ExtraData: nil,
		TopicId:   uint64(1),
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: 100},
		},
		Reputer:       s.AddrsStr()[2],
		CombinedValue: alloraMath.MustNewDecFromString("0.2"),
		NaiveValue:    alloraMath.MustNewDecFromString("0.2"),
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		ForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneInForecasterValues: []*emissions.WorkerAttributedValue{
			{
				Worker: s.AddrsStr()[1],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
			{
				Worker: s.AddrsStr()[2],
				Value:  alloraMath.MustNewDecFromString("0.2"),
			},
		},
		OneOutInfererForecasterValues: nil,
	}
	signature2 := s.signValueBundle(&valueBundle2, s.PrivKeys()[2])
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
				s.AddrsStr()[1]: inferencesynthesis.CosmosIntOneE18(),           // 1 token
				s.AddrsStr()[2]: inferencesynthesis.CosmosIntOneE18().MulRaw(2), // 2 tokens
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: &valueBundle1,
						Signature:   signature1,
						Pubkey:      s.PubKeyHexStr()[1],
					},
					{
						ValueBundle: &valueBundle2,
						Signature:   signature2,
						Pubkey:      s.PubKeyHexStr()[2],
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				TopicId: uint64(1),
				Reputer: s.AddrsStr()[1],
				ReputerRequestNonce: &emissions.ReputerRequestNonce{
					ReputerNonce: &emissions.Nonce{BlockHeight: 100},
				},
				ExtraData:     nil,
				CombinedValue: alloraMath.MustNewDecFromString("0.2"),
				NaiveValue:    alloraMath.MustNewDecFromString("0.2"),
				InfererValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				ForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneOutInfererValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
				},
				OneInForecasterValues: []*emissions.WorkerAttributedValue{
					{
						Worker: s.AddrsStr()[1],
						Value:  alloraMath.MustNewDecFromString("0.2"),
					},
					{
						Worker: s.AddrsStr()[2],
						Value:  alloraMath.MustNewDecFromString("0.2"),
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
			output, err := inferencesynthesis.CalcNetworkLosses(s.Ctx(), topicId, block, tc.stakesByReputer, tc.reportedLosses)
			if tc.expectedError != nil {
				require.Error(err)
				require.EqualError(err, tc.expectedError.Error())
			} else {
				require.NoError(err)
				s.expectedEqualNetworkLosses(&tc.expectedOutput, &output)
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLossesFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch301Get := epochGet[301]
	topicId := uint64(1)
	blockHeight := int64(301)

	reputer0 := s.AddrsStr()[0]
	reputer1 := s.AddrsStr()[1]
	reputer2 := s.AddrsStr()[2]
	reputer3 := s.AddrsStr()[3]
	reputer4 := s.AddrsStr()[4]
	reputers := []testutil.ReputerKey{
		{
			Address:    reputer0,
			PrivateKey: s.PrivKeys()[0],
			PubKeyHex:  s.PubKeyHexStr()[0],
		},
		{
			Address:    reputer1,
			PrivateKey: s.PrivKeys()[1],
			PubKeyHex:  s.PubKeyHexStr()[1],
		},
		{
			Address:    reputer2,
			PrivateKey: s.PrivKeys()[2],
			PubKeyHex:  s.PubKeyHexStr()[2],
		},
		{
			Address:    reputer3,
			PrivateKey: s.PrivKeys()[3],
			PubKeyHex:  s.PubKeyHexStr()[3],
		},
		{
			Address:    reputer4,
			PrivateKey: s.PrivKeys()[4],
			PubKeyHex:  s.PubKeyHexStr()[4],
		},
	}

	inferer0 := s.AddrsStr()[5]
	inferer1 := s.AddrsStr()[6]
	inferer2 := s.AddrsStr()[7]
	inferer3 := s.AddrsStr()[8]
	inferer4 := s.AddrsStr()[9]
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := s.AddrsStr()[10]
	forecaster1 := s.AddrsStr()[11]
	forecaster2 := s.AddrsStr()[12]
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

	networkLosses, err := inferencesynthesis.CalcNetworkLosses(s.Ctx(), topicId, blockHeight, stakesByReputer, reportedLosses)
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

	s.expectedEqualNetworkLosses(&expectedNetworkLosses, &networkLosses)

}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLossesCombined() {
	tests := append(s.getTestCasesOneWorker(), s.getTestCasesTwoWorkers()...)

	topicId := uint64(1)
	block := int64(100)
	require := s.Require()

	for _, tc := range tests {
		s.Run(tc.name, func() {
			output, err := inferencesynthesis.CalcNetworkLosses(s.Ctx(), topicId, block, tc.stakesByReputer, tc.reportedLosses)
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

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLossesCombinedBadCasesWrongTopicId() {
	// Setting wrong topicId to the first loss bundle
	badCase := s.getTestCasesTwoWorkersBadTopicId()
	tests := badCase
	topicId := uint64(1)

	block := int64(100)
	require := s.Require()

	for _, tc := range tests {
		s.Run(tc.name, func() {
			output, err := inferencesynthesis.CalcNetworkLosses(s.Ctx(), topicId, block, tc.stakesByReputer, tc.reportedLosses)
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
				require.Len(output.OneOutInfererForecasterValues, len(tc.expectedOutput.OneOutInfererForecasterValues), "Mismatch in number of OneOutInfererForecasterValues")
				s.expectedEqualNetworkLosses(&tc.expectedOutput, &output)
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkLossesCombinedBadCasesWrongBlockHeight() {
	// Setting wrong block height to the first loss bundle
	badCase := s.getTestCasesTwoWorkersBadBlockHeight()
	tests := badCase
	topicId := uint64(1)

	block := int64(100)
	require := s.Require()

	for _, tc := range tests {
		s.Run(tc.name, func() {
			output, err := inferencesynthesis.CalcNetworkLosses(s.Ctx(), topicId, block, tc.stakesByReputer, tc.reportedLosses)
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
				require.Len(output.OneOutInfererForecasterValues, len(tc.expectedOutput.OneOutInfererForecasterValues), "Mismatch in number of OneOutInfererForecasterValues")
				s.expectedEqualNetworkLosses(&tc.expectedOutput, &output)
			}
		})
	}
}

// expectedEqualNetworkLosses compares two network ValueBundle objects and verifies that all values match within epsilon
func (s *InferenceSynthesisTestSuite) expectedEqualNetworkLosses(expected, actual *emissions.ValueBundle) {
	// Compare combined and naive values
	testutil.InEpsilon5(s.T(), expected.CombinedValue, actual.CombinedValue.String())
	testutil.InEpsilon5(s.T(), expected.NaiveValue, actual.NaiveValue.String())

	// Compare InfererValues
	s.Require().Len(actual.InfererValues, len(expected.InfererValues))
	for _, expectedValue := range expected.InfererValues {
		found := false
		for _, workerAttributedValue := range actual.InfererValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}

	// Compare ForecasterValues
	s.Require().Len(actual.ForecasterValues, len(expected.ForecasterValues))
	for _, expectedValue := range expected.ForecasterValues {
		found := false
		for _, workerAttributedValue := range actual.ForecasterValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}

	// Compare OneOutInfererForecasterValues
	s.Require().Len(actual.OneOutInfererForecasterValues, len(expected.OneOutInfererForecasterValues))
	for _, expectedValue := range expected.OneOutInfererForecasterValues {
		found := false
		for _, workerAttributedValue := range actual.OneOutInfererForecasterValues {
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

	// Compare OneOutInfererValues
	s.Require().Len(actual.OneOutInfererValues, len(expected.OneOutInfererValues))
	for _, expectedValue := range expected.OneOutInfererValues {
		found := false
		for _, workerAttributedValue := range actual.OneOutInfererValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}

	// Compare OneOutForecasterValues
	s.Require().Len(actual.OneOutForecasterValues, len(expected.OneOutForecasterValues))
	for _, expectedValue := range expected.OneOutForecasterValues {
		found := false
		for _, workerAttributedValue := range actual.OneOutForecasterValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}

	// Compare OneInForecasterValues
	s.Require().Len(actual.OneInForecasterValues, len(expected.OneInForecasterValues))
	for _, expectedValue := range expected.OneInForecasterValues {
		found := false
		for _, workerAttributedValue := range actual.OneInForecasterValues {
			if workerAttributedValue.Worker == expectedValue.Worker {
				found = true
				testutil.InEpsilon5(s.T(), expectedValue.Value, workerAttributedValue.Value.String())
			}
		}
		s.Require().True(found)
	}
}
