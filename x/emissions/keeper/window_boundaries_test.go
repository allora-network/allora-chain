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
			blockHeight:      1250,
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
			blockHeight:      1199,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Block is one before the window opens; the window opens at the end of the reveal epoch, not at the reveal",
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
			blockHeight:      1200,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Block is exactly when the window opens: revealedGroundTruthBlock + extraLag",
		},
		{
			name: "GTLag not divisible by EpochLength - reveal block is not yet the window",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 1000},
			},
			blockHeight:      1130,
			expectedInWindow: false,
			expectedErr:      nil,
			description:      "Ground truth is revealed at 1130 but the window only opens at the epoch boundary 1200",
		},
		{
			name: "GTLag not divisible by EpochLength - window opens exactly when the previous nonce closes",
			topic: types.Topic{ //nolint:exhaustruct
				EpochLength:    100,
				GroundTruthLag: 130,
			},
			nonce: types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{BlockHeight: 900},
			},
			blockHeight:      1199,
			expectedInWindow: true,
			expectedErr:      nil,
			description:      "Previous nonce is still open at 1199 and closes at 1200, where the next nonce opens",
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

// TestReputerSubmissionWindowBounds verifies shared window arithmetic and overflow handling.
func (s *KeeperTestSuite) TestReputerSubmissionWindowBounds() {
	tests := []struct {
		name          string
		topic         types.Topic
		nonceHeight   int64
		expectedStart int64
		expectedEnd   int64
		expectedErr   error
		description   string
	}{
		{
			name:          "aligned lag opens at the reveal",
			topic:         types.Topic{EpochLength: 100, GroundTruthLag: 100}, //nolint:exhaustruct
			nonceHeight:   1000,
			expectedStart: 1100,
			expectedEnd:   1200,
			expectedErr:   nil,
			description:   "extraLag is 0, so the reveal block is already an epoch boundary",
		},
		{
			name:          "fractional lag waits for the end of the reveal epoch",
			topic:         types.Topic{EpochLength: 100, GroundTruthLag: 130}, //nolint:exhaustruct
			nonceHeight:   1000,
			expectedStart: 1200,
			expectedEnd:   1300,
			expectedErr:   nil,
			description:   "reveal is 1130, extraLag 70, so the window opens at 1200",
		},
		{
			name:          "production-shaped fractional lag",
			topic:         types.Topic{EpochLength: 35, GroundTruthLag: 3388}, //nolint:exhaustruct
			nonceHeight:   1_000_000,
			expectedStart: 1_003_395,
			expectedEnd:   1_003_430,
			expectedErr:   nil,
			description:   "window is exactly one epoch long and starts on the grid",
		},
		{
			name:          "overflow is rejected",
			topic:         types.Topic{EpochLength: 100, GroundTruthLag: 100}, //nolint:exhaustruct
			nonceHeight:   math.MaxInt64 - 150,
			expectedStart: 0,
			expectedEnd:   0,
			expectedErr:   types.ErrInvalidValue,
			description:   "adding the window must not wrap",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			start, end, err := keeper.ReputerSubmissionWindowBounds(
				tt.topic,
				types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: tt.nonceHeight}},
			)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tt.expectedErr, tt.description)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(tt.expectedStart, start, tt.description)
			s.Require().Equal(tt.expectedEnd, end, tt.description)
			// The window is always exactly one epoch wide.
			s.Require().Equal(tt.topic.EpochLength, end-start, tt.description)
			// And it must not open before the previous nonce closes: the previous
			// nonce (this one minus an epoch) closes at exactly this start.
			prevStart, prevEnd, err := keeper.ReputerSubmissionWindowBounds(
				tt.topic,
				types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: tt.nonceHeight - tt.topic.EpochLength}},
			)
			s.Require().NoError(err)
			s.Require().Equal(start, prevEnd, "previous window must end exactly where this one starts")
			s.Require().Less(prevStart, start, tt.description)
		})
	}
}
