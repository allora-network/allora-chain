package inference_synthesis_test

import (
	"log"

	alloraMath "github.com/allora-network/allora-chain/math"

	"github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
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
			expectedLoss:        inference_synthesis.WorkerRunningWeightedLoss{Loss: alloraMath.MustNewDecFromString("0.900514997"), SumWeight: alloraMath.MustNewDecFromString("2.0")},
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
				log.Printf("Expected loss: %v, Actual loss: %v, Epsilon: %v. Loss should match the expected value within epsilon.", tc.expectedLoss.Loss, updatedLoss.Loss, 1e-5)
				s.Require().True(alloraMath.InDelta(tc.expectedLoss.Loss, updatedLoss.Loss, alloraMath.MustNewDecFromString("0.00001")), "Loss should match the expected value within epsilon")
				s.Require().Equal(tc.expectedLoss.SumWeight, updatedLoss.SumWeight, "Sum of weights should match the expected value")
			}
		})
	}
}
