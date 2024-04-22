package inference_synthesis_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestRunningWeightedAvgUpdate() {
	tests := []struct {
		name                string
		initialWeightedLoss inference_synthesis.WorkerRunningWeightedLoss
		weight              inference_synthesis.Weight
		nextValue           inference_synthesis.Weight
		epsilon             inference_synthesis.Weight
		expectedLoss        inference_synthesis.WorkerRunningWeightedLoss
		expectedErr         error
	}{
		{
			name:                "normal operation",
			initialWeightedLoss: inference_synthesis.WorkerRunningWeightedLoss{Loss: alloraMath.MustNewDecFromString("0.5"), SumWeight: alloraMath.MustNewDecFromString("1.0")},
			weight:              alloraMath.MustNewDecFromString("1.0"),
			nextValue:           alloraMath.MustNewDecFromString("2.0"),
			epsilon:             alloraMath.MustNewDecFromString("1e-4"),
			expectedLoss:        inference_synthesis.WorkerRunningWeightedLoss{Loss: alloraMath.MustNewDecFromString("0.400514997"), SumWeight: alloraMath.MustNewDecFromString("2.0")},
			expectedErr:         nil,
		},
		{
			name:                "division by zero error",
			initialWeightedLoss: inference_synthesis.WorkerRunningWeightedLoss{Loss: alloraMath.MustNewDecFromString("1.01"), SumWeight: alloraMath.MustNewDecFromString("0")},
			weight:              alloraMath.MustNewDecFromString("2.0"),
			nextValue:           alloraMath.MustNewDecFromString("1.0"),
			epsilon:             alloraMath.MustNewDecFromString("3.0"),
			expectedLoss:        inference_synthesis.WorkerRunningWeightedLoss{},
			expectedErr:         emissions.ErrFractionDivideByZero,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			updatedLoss, err := inference_synthesis.RunningWeightedAvgUpdate(
				&tc.initialWeightedLoss,
				tc.weight,
				tc.nextValue,
				tc.epsilon,
			)
			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr, "Error should match the expected error")
			} else {
				s.Require().NoError(err, "No error expected but got one")
				s.Require().True(alloraMath.InDelta(tc.expectedLoss.Loss, updatedLoss.Loss, alloraMath.MustNewDecFromString("0.00001")), "Loss should match the expected value within epsilon")
				s.Require().Equal(tc.expectedLoss.SumWeight, updatedLoss.SumWeight, "Sum of weights should match the expected value")
			}
		})
	}
}

func TestCalcNetworkLosses(t *testing.T) {
	tests := []struct {
		name            string
		stakesByReputer map[inference_synthesis.Worker]cosmosMath.Uint
		reportedLosses  emissions.ReputerValueBundles
		epsilon         alloraMath.Dec
		expectedOutput  emissions.ValueBundle
		expectedError   error
	}{
		{
			name: "single reputer single inferer",
			stakesByReputer: map[inference_synthesis.Worker]cosmosMath.Uint{
				"worker0": cosmosMath.NewUintFromString("1"),
			},
			reportedLosses: emissions.ReputerValueBundles{
				ReputerValueBundles: []*emissions.ReputerValueBundle{
					{
						ValueBundle: &emissions.ValueBundle{
							Reputer:       "worker0",
							CombinedValue: alloraMath.MustNewDecFromString("0.5"),
							InfererValues: []*emissions.WorkerAttributedValue{
								{Worker: "worker0", Value: alloraMath.MustNewDecFromString("0.5")},
							},
						},
					},
				},
			},
			epsilon: alloraMath.MustNewDecFromString("1e-4"),
			expectedOutput: emissions.ValueBundle{
				CombinedValue: alloraMath.MustNewDecFromString("1.648721"), // e^0.5
				InfererValues: []*emissions.WorkerAttributedValue{
					{Worker: "worker1", Value: alloraMath.MustNewDecFromString("1.648721")},
				},
			},
			expectedError: nil,
		},
		/*
			{
				name: "multiple reputers multiple values",
				stakesByReputer: map[inference_synthesis.Worker]cosmosMath.Uint{
					"worker1": cosmosMath.NewUintFromString("1"),
					"worker2": cosmosMath.NewUintFromString("2"),
				},
				reportedLosses: emissions.ReputerValueBundles{
					ReputerValueBundles: []*emissions.ReputerValueBundle{
						{
							ValueBundle: &emissions.ValueBundle{
								Reputer:       "worker1",
								CombinedValue: alloraMath.MustNewDecFromString("0.5"),
								InfererValues: []*emissions.WorkerAttributedValue{
									{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.5")},
								},
							},
						},
						{
							ValueBundle: &emissions.ValueBundle{
								Reputer:       "worker2",
								CombinedValue: alloraMath.MustNewDecFromString("0.2"),
								InfererValues: []*emissions.WorkerAttributedValue{
									{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.3")},
									{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.2")},
								},
							},
						},
					},
				},
				epsilon: alloraMath.MustNewDecFromString("1e-4"),
				expectedOutput: emissions.ValueBundle{
					CombinedValue: alloraMath.MustNewDecFromString("1.412538"), // Calculated e^(weighted average of 0.5 and 0.2)
					InfererValues: []*emissions.WorkerAttributedValue{
						{Worker: "worker1", Value: alloraMath.MustNewDecFromString("1.481689")},
						{Worker: "worker2", Value: alloraMath.MustNewDecFromString("1.221403")},
					},
				},
				expectedError: nil,
			},
		*/
		/*
			{
				name: "error handling invalid decimal",
				stakesByReputer: map[inference_synthesis.Worker]cosmosMath.Uint{
					"worker1": cosmosMath.NewUintFromString("NaN"),
				},
				reportedLosses: emissions.ReputerValueBundles{},
				epsilon:        alloraMath.MustNewDecFromString("1e-4"),
				expectedOutput: emissions.ValueBundle{},
				expectedError:  fmt.Errorf("invalid decimal value"),
			},
		*/
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := inference_synthesis.CalcNetworkLosses(tc.stakesByReputer, tc.reportedLosses, tc.epsilon)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOutput.CombinedValue, output.CombinedValue)
				require.ElementsMatch(t, tc.expectedOutput.InfererValues, output.InfererValues)
			}
		})
	}
}
