package inference_synthesis_test

import (
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func TestSortByBlockHeight(t *testing.T) {
	// Create some test data
	tests := []struct {
		name   string
		input  *emissionstypes.ReputerRequestNonces
		output *emissionstypes.ReputerRequestNonces
	}{
		{
			name: "Sorted in descending order",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
				},
			},
		},
		{
			name: "Already sorted",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
		},
		{
			name: "Empty input",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{},
			},
		},
		{
			name: "Single element",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Call the sorting function
			inference_synthesis.SortByBlockHeight(test.input.Nonces)

			// Compare the sorted input with the expected output
			require.Equal(t, test.input.Nonces, test.output.Nonces, "Sorting result mismatch.\nExpected: %v\nGot: %v")
		})
	}
}

func TestSelectTopNWorkerNonces(t *testing.T) {
	// Define test cases
	tests := []struct {
		name               string
		workerNonces       emissionstypes.Nonces
		N                  int
		expectedTopNNonces []*emissionstypes.Nonce
	}{
		{
			name: "N greater than length of nonces",
			workerNonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
				},
			},
			N: 5,
			expectedTopNNonces: []*emissionstypes.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
		{
			name: "N less than length of nonces",
			workerNonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
					{BlockHeight: 3},
				},
			},
			N: 2,
			expectedTopNNonces: []*emissionstypes.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := inference_synthesis.SelectTopNWorkerNonces(tc.workerNonces, tc.N)
			require.Equal(t, actual, tc.expectedTopNNonces, "Worker nonces to not match")
		})
	}
}

func TestFilterNoncesWithinEpochLength(t *testing.T) {
	tests := []struct {
		name          string
		nonces        emissionstypes.Nonces
		blockHeight   int64
		epochLength   int64
		expectedNonce emissionstypes.Nonces
	}{
		{
			name: "Nonces within epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
		},
		{
			name: "Nonces outside epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 5},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 15},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := inference_synthesis.FilterNoncesWithinEpochLength(tc.nonces, tc.blockHeight, tc.epochLength)
			require.Equal(t, tc.expectedNonce, actual, "Filter nonces do not match")
		})
	}
}

func TestSelectTopNReputerNonces(t *testing.T) {
	// Define test cases
	tests := []struct {
		name                     string
		reputerRequestNonces     *emissionstypes.ReputerRequestNonces
		N                        int
		expectedTopNReputerNonce []*emissionstypes.ReputerRequestNonce
		currentBlockHeight       int64
		groundTruthLag           int64
		epochLength              int64
	}{
		{
			name: "N greater than length of nonces, zero lag",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				},
			},
			N: 5,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
			epochLength:        1,
		},
		{
			name: "N less than length of nonces, zero lag",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 6}},
				},
			},
			N: 2,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 6}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
			epochLength:        1,
		},
		{
			name: "Ground truth lag cutting selection midway",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     5,
			epochLength:        1,
		},
		{
			name: "Big Ground truth lag, not selecting any nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N:                        3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{},
			currentBlockHeight:       10,
			groundTruthLag:           10,
			epochLength:              1,
		},
		{
			name: "Small ground truth lag, selecting all nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     2,
			epochLength:        1,
		},
		{
			name: "Mid ground truth lag, selecting some nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     3,
			epochLength:        2,
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := inference_synthesis.SelectTopNReputerNonces(tc.reputerRequestNonces, tc.N, tc.currentBlockHeight, tc.groundTruthLag, tc.epochLength)
			require.Equal(t, tc.expectedTopNReputerNonce, actual, "Reputer nonces do not match")
		})
	}
}
