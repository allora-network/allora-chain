package inference_synthesis_test

import (
	"log"

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
			initialWeightedLoss: inference_synthesis.WorkerRunningWeightedLoss{Loss: 0.5, SumWeight: 1.0},
			weight:              1.0,
			nextValue:           2.0,
			epsilon:             1e-4,
			expectedLoss:        inference_synthesis.WorkerRunningWeightedLoss{Loss: 0.900514997, SumWeight: 2.0},
			expectedErr:         nil,
		},
		{
			name:                "division by zero error",
			initialWeightedLoss: inference_synthesis.WorkerRunningWeightedLoss{Loss: 1.01, SumWeight: 0},
			weight:              2.0,
			nextValue:           1.0,
			epsilon:             3.0,
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
				s.Require().InEpsilon(tc.expectedLoss.Loss, updatedLoss.Loss, 1e-5, "Loss should match the expected value within epsilon")
				s.Require().Equal(tc.expectedLoss.SumWeight, updatedLoss.SumWeight, "Sum of weights should match the expected value")
			}
		})
	}
}
