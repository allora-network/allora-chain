package keeper_test

import (
	"math"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestBlockWithinReputerSubmissionWindowOfNonce() {
	tests := []struct {
		name             string
		topic            types.Topic
		nonce            types.ReputerRequestNonce
		blockHeight      int64
		expectedInWindow bool
		expectedErr      error
		description      string
	}{
		{
			name: "Simple case - Block within window",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 100,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1150,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within the submission window when GTLag equals EpochLength",
		},
		{
			name: "Simple case - Block outside window (too early)",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 100,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1099,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is before ground truth is revealed",
		},
		{
			name: "Simple case - Block outside window (too late)",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 100,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1201,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is after submission window ends",
		},
		{
			name: "Simple case - Block exactly at window start",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 100,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1100, // Exactly when ground truth is revealed
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is exactly at the start of submission window",
		},
		{
			name: "Simple case - Block exactly at window end",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 100,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1200, // Last valid block
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is exactly at the end of submission window",
		},
		{
			name: "GTLag not divisible by EpochLength - Block within window",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1150,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag > EpochLength and not divisible, Block within window",
		},
		{
			name: "GTLag not divisible by EpochLength - Lower boundary out",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1129,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is within window when GTLag > EpochLength and not divisible, Lower boundary out",
		},
		{
			name: "GTLag not divisible by EpochLength - Lower boundary in",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1130,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag > EpochLength and not divisible, Lower boundary in",
		},
		{
			name: "GTLag not divisible by EpochLength - Upper boundary in",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1300,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag > EpochLength and not divisible, Upper boundary in",
		},
		{
			name: "GTLag not divisible by EpochLength - Upper boundary out",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1301,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is outside window when GTLag > EpochLength and not divisible, Upper boundary out",
		},
		{
			name: "GTLag less than EpochLength - Block within window",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 70,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1200,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag < EpochLength",
		},
		{
			name: "GTLag multiple of EpochLength",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 200, // Exactly 2 epochs
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1250,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag is multiple of EpochLength",
		},
		{
			name: "GTLag multiple of EpochLength - Lower boundary out",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 200, // Exactly 2 epochs
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1199,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is outside window when GTLag is multiple of EpochLength, Lower boundary out",
		},
		{
			name: "GTLag multiple of EpochLength - Lower boundary in",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 200, // Exactly 2 epochs
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1200,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag is multiple of EpochLength, Lower boundary in",
		},
		{
			name: "GTLag multiple of EpochLength - Upper boundary in",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 200, // Exactly 2 epochs
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1300,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is within window when GTLag is multiple of EpochLength, Upper boundary in",
		},
		{
			name: "GTLag multiple of EpochLength - Upper boundary out",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 200, // Exactly 2 epochs
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1301,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is outside window when GTLag is multiple of EpochLength, Upper boundary out",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := keeper.BlockWithinReputerSubmissionWindowOfNonce(
				tt.topic,
				tt.nonce,
				tt.blockHeight,
			)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tt.expectedErr, tt.description)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tt.expectedInWindow, result, tt.description)
			}
		})
	}
}

func (s *KeeperTestSuite) TestBlockWithinWorkerSubmissionWindowOfNonce() {
	tests := []struct {
		name             string
		topic            types.Topic
		nonce            types.Nonce
		blockHeight      int64
		expectedInWindow bool
		expectedErr      error
		description      string
	}{
		{
			name: "Simple case - Block in the middle of the window",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      1050,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is in the middle of submission window",
		},
		{
			name: "Simple case - Block outside window (too early)",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      999,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is before nonce block height",
		},
		{
			name: "Simple case - Block outside window (too late)",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      1101,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is after submission window ends",
		},
		{
			name: "Edge case - Block exactly at window start",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      1000,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is exactly at nonce block height (inclusive)",
		},
		{
			name: "Edge case - Block exactly at window end",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      1100,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is at last valid block (exclusive of end)",
		},
		{
			name: "Edge case - Zero submission window",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 0,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      1000,
			expectedInWindow: true,
			expectedErr:      types.ErrInvalidValue,
			description:      "Only the nonce block itself should be valid with zero window",
		},
		{
			name: "Edge case - Large submission window",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 10000,
			},
			nonce: types.Nonce{
				BlockHeight: 1000,
			},
			blockHeight:      5000,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block well within a large submission window",
		},
		{
			name: "Edge case - Nonce at block zero",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: 0,
			},
			blockHeight:      50,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Nonce starting at genesis block",
		},
		{
			name: "Edge case - Block at max int64",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 100,
			},
			nonce: types.Nonce{
				BlockHeight: math.MaxInt64 - 50,
			},
			blockHeight:      math.MaxInt64 - 25,
			expectedInWindow: true,
			expectedErr:      types.ErrInvalidValue,
			description:      "Testing near max int64 boundary",
		},
		{
			name: "Edge case - Overflow",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 10,
			},
			nonce: types.Nonce{
				BlockHeight: math.MaxInt64 - 5,
			},
			blockHeight:      math.MaxInt64 - 2,
			expectedInWindow: false,
			expectedErr:      types.ErrInvalidValue,
			description:      "Testing overflow condition",
		},
		{
			name: "Edge case - Near overflow but valid",
			topic: types.Topic{ //nolint:exhaustruct
				WorkerSubmissionWindow: 10,
			},
			nonce: types.Nonce{
				BlockHeight: math.MaxInt64 - 15,
			},
			blockHeight:      math.MaxInt64 - 5,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Testing near overflow but valid case",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := keeper.BlockWithinWorkerSubmissionWindowOfNonce(
				tt.topic,
				tt.nonce,
				tt.blockHeight,
			)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tt.expectedErr, tt.description)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tt.expectedInWindow, result, tt.description)
			}
		})
	}
}
