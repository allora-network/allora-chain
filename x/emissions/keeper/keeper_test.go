//nolint:exhaustruct,
package keeper_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type KeeperTestSuite struct {
	testutil.TestSuite
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, &KeeperTestSuite{
		testutil.NewTestSuite("emissions_keeper"),
	})
}

// WORKER NONCE TESTS

func (s *KeeperTestSuite) TestAddWorkerNonce() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Empty(unfulfilledNonces.Nonces, "Unfulfilled nonces should be empty")

	// Set worker nonce
	newNonce := &types.Nonce{BlockHeight: 42}
	err = k.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err)

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(newNonce.BlockHeight, unfulfilledNonces.Nonces[0].BlockHeight, "Unfulfilled nonces should contain the new nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedWorkerNonceIsUnfulfilled() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	isUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "non existent nonce should not be listed as unfulfilled")

	// Set worker nonce
	err = k.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	isUnfulfilled, err = k.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "new nonce should be unfulfilled")
}

func (s *KeeperTestSuite) TestCanFulfillNewWorkerNonce() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	// Set worker nonce
	err := k.AddWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	isUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "new nonce should not be unfulfilled")

	// Fulfill the nonce
	success, err := k.FulfillWorkerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().True(success, "nonce should be able to be fulfilled")

	// Check that the nonce is no longer unfulfilled
	isUnfulfilled, err = k.IsWorkerNonceUnfulfilled(ctx, topicId, newNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "new nonce should be fulfilled")
}

func (s *KeeperTestSuite) TestGetMultipleUnfulfilledWorkerNonces() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Empty(initialNonces.Nonces, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = k.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(nonceValues[len(nonceValues)-i-1], nonce.BlockHeight, "Nonce value should match the expected value")
	}
}

func (s *KeeperTestSuite) TestGetAndFulfillMultipleUnfulfilledWorkerNonces() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Empty(initialNonces.Nonces, "Initial unfulfilled nonces should be empty")

	// Set multiple worker nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = k.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}
	// Retrieve and verify the nonces
	retrievedNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues), "Should match the number of unfulfilled nonces")

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		success, err := k.FulfillWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().True(success, "Nonce should be successfully fulfilled")
		s.Require().NoError(err, "Error fulfilling nonce")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err = k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues)-len(fulfillNonces), "Should match the number of unfulfilled nonces")

	// Check that all the expected unfulfilled nonces are present and correct
	expectedUnfulfilled := []int64{46, 44, 42} // Expected remaining unfulfilled nonces
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(expectedUnfulfilled[i], nonce.BlockHeight, "Remaining nonce value should match the expected unfulfilled value")
	}
}

func (s *KeeperTestSuite) TestWorkerNonceLimitEnforcement() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	maxUnfulfilledRequests := 3
	// Set the maximum number of unfulfilled worker nonces
	params := types.DefaultParams()
	params.MaxUnfulfilledWorkerRequests = uint64(maxUnfulfilledRequests)

	// Set the maximum number of unfulfilled worker nonces via the SetParams method
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	// Initially add nonces to exceed the maxUnfulfilledRequests
	nonceValues := []int64{10, 20, 30, 40, 50}
	for _, val := range nonceValues {
		err = k.AddWorkerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add worker nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, maxUnfulfilledRequests, "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{50, 40, 30} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.BlockHeight, "Nonce should match the expected recent nonce")
	}
}

// REPUTER NONCE TESTS

func (s *KeeperTestSuite) TestAddReputerNonce() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")

	s.Require().Empty(unfulfilledNonces.Nonces, "Unfulfilled nonces should be empty")

	// Set reputer nonce
	newReputerNonce := &types.Nonce{BlockHeight: 42}
	err = k.AddReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)

	unfulfilledNonces, err = k.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	s.Require().Len(unfulfilledNonces.Nonces, 1, "Unfulfilled nonces should not be empty")

	// Check that the nonce is the correct nonce
	s.Require().Equal(
		newReputerNonce.BlockHeight,
		unfulfilledNonces.Nonces[0].ReputerNonce.BlockHeight,
		"Unfulfilled nonces should contain the new reputer nonce")
}

func (s *KeeperTestSuite) TestNewlyAddedReputerNonceIsUnfulfilled() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{BlockHeight: 42}

	isUnfulfilled, err := k.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "Non-existent nonce should not be listed as unfulfilled")

	// Set reputer nonce
	err = k.AddReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)

	isUnfulfilled, err = k.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "New nonce should be unfulfilled")
}

func (s *KeeperTestSuite) TestCanFulfillNewReputerNonce() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	newReputerNonce := &types.Nonce{BlockHeight: 42}

	// Set reputer nonce
	err := k.AddReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)

	// Check that the nonce is the correct nonce
	isUnfulfilled, err := k.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(isUnfulfilled, "New nonce should be unfulfilled")

	// Fulfill the nonce
	nonceIsUnfulfilled, err := k.FulfillReputerNonce(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().True(nonceIsUnfulfilled, "Nonce should be able to be fulfilled")

	// Check that the nonce is no longer unfulfilled
	isUnfulfilled, err = k.IsReputerNonceUnfulfilled(ctx, topicId, newReputerNonce)
	s.Require().NoError(err)
	s.Require().False(isUnfulfilled, "New nonce should be fulfilled")
}

func (s *KeeperTestSuite) TestGetAndFulfillMultipleUnfulfilledReputerNonces() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	initialNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces")
	s.Require().Empty(initialNonces.Nonces, "Initial unfulfilled nonces should be empty")

	// Set multiple reputer nonces
	nonceValues := []int64{42, 43, 44, 45, 46}
	for _, val := range nonceValues {
		err = k.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Fulfill some nonces: 43 and 45
	fulfillNonces := []int64{43, 45}
	for _, val := range fulfillNonces {
		nonceIsUnfulfilled, err := k.FulfillReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Error fulfilling nonce")
		s.Require().True(nonceIsUnfulfilled, "Nonce should be able to be fulfilled")
	}

	// Retrieve and verify the nonces
	retrievedNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after fulfilling some")
	s.Require().Len(retrievedNonces.Nonces, len(nonceValues)-len(fulfillNonces), "Should match the number of unfulfilled nonces")

	// Check that all the expected unfulfilled nonces are present and correct
	expectedUnfulfilled := []int64{46, 44, 42} // Expected remaining unfulfilled nonces
	for i, nonce := range retrievedNonces.Nonces {
		s.Require().Equal(expectedUnfulfilled[i], nonce.ReputerNonce.BlockHeight, "Remaining nonce value should match the expected unfulfilled value")
	}
}

func (s *KeeperTestSuite) TestGetOpenReputerSubmissionWindows() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(50),
	)

	// Initially, no open nonces
	openNonces, err := k.GetOpenReputerSubmissionWindows(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(openNonces.Nonces, "Initially should have no open nonces")

	// Add nonces at different block heights
	// Nonce 1: block 1000, will be open at block 1100-1200 (1000 + 100 GTLag to 1000 + 100 + 0 + 100)
	nonce1 := &types.Nonce{BlockHeight: 1000}
	err = k.AddReputerNonce(ctx, topicId, nonce1)
	s.Require().NoError(err)

	// Nonce 2: block 2000, will be open at block 2100-2200
	nonce2 := &types.Nonce{BlockHeight: 2000}
	err = k.AddReputerNonce(ctx, topicId, nonce2)
	s.Require().NoError(err)

	tests := []struct {
		name           string
		blockHeight    int64
		expectedNonces []int64
		description    string
	}{
		{
			name:           "before_window_opens_nonce1",
			blockHeight:    1050,
			expectedNonces: []int64{},
			description:    "Should have no open nonces before window opens for nonce1",
		},
		{
			name:           "one_block_before_window_opens_nonce1",
			blockHeight:    1099,
			expectedNonces: []int64{},
			description:    "Should have no open nonces one block before window opens (tests exclusive lower bound)",
		},
		{
			name:           "exactly_at_window_start_nonce1",
			blockHeight:    1100,
			expectedNonces: []int64{1000},
			description:    "Should have nonce1 open exactly at window start (tests inclusive lower bound)",
		},
		{
			name:           "within_window_nonce1",
			blockHeight:    1150,
			expectedNonces: []int64{1000},
			description:    "Should have nonce1 open within window, nonce2 closed",
		},
		{
			name:           "exactly_at_window_end_nonce1",
			blockHeight:    1200,
			expectedNonces: []int64{1000},
			description:    "Should have nonce1 open exactly at window end (tests inclusive upper bound)",
		},
		{
			name:           "after_window_closes_nonce1",
			blockHeight:    1201,
			expectedNonces: []int64{},
			description:    "Should have no open nonces after window closes for nonce1",
		},
		{
			name:           "one_block_before_window_opens_nonce2",
			blockHeight:    2099,
			expectedNonces: []int64{},
			description:    "Should have no open nonces one block before window opens for nonce2 (tests exclusive lower bound)",
		},
		{
			name:           "exactly_at_window_start_nonce2",
			blockHeight:    2100,
			expectedNonces: []int64{2000},
			description:    "Should have nonce2 open exactly at window start (tests inclusive lower bound)",
		},
		{
			name:           "within_window_nonce2",
			blockHeight:    2150,
			expectedNonces: []int64{2000},
			description:    "Should have nonce2 open within window, nonce1 closed",
		},
		{
			name:           "exactly_at_window_end_nonce2",
			blockHeight:    2200,
			expectedNonces: []int64{2000},
			description:    "Should have nonce2 open exactly at window end (tests inclusive upper bound)",
		},
		{
			name:           "after_window_closes_nonce2",
			blockHeight:    2201,
			expectedNonces: []int64{},
			description:    "Should have no open nonces after window closes for nonce2",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.WithBlockHeight(tt.blockHeight)
			openNonces, err := k.GetOpenReputerSubmissionWindows(s.Ctx(), topicId)
			s.Require().NoError(err, tt.description)
			s.Require().Len(openNonces.Nonces, len(tt.expectedNonces), tt.description)

			// Verify the expected nonces are present
			actualNonceHeights := make([]int64, len(openNonces.Nonces))
			for i, nonce := range openNonces.Nonces {
				actualNonceHeights[i] = nonce.ReputerNonce.BlockHeight
			}

			s.Require().ElementsMatch(tt.expectedNonces, actualNonceHeights, tt.description)
		})
	}
}

func (s *KeeperTestSuite) TestGetOpenWorkerSubmissionWindows() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(50),
	)

	// Initially, no open nonces
	openNonces, err := k.GetOpenWorkerSubmissionWindows(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(openNonces.Nonces, "Initially should have no open nonces")

	// Add nonces at different block heights
	// Nonce 1: block 1000, will be open at block 1000-1050
	nonce1 := &types.Nonce{BlockHeight: 1000}
	err = k.AddWorkerNonce(ctx, topicId, nonce1)
	s.Require().NoError(err)

	// Nonce 2: block 2000, will be open at block 2000-2050
	nonce2 := &types.Nonce{BlockHeight: 2000}
	err = k.AddWorkerNonce(ctx, topicId, nonce2)
	s.Require().NoError(err)

	tests := []struct {
		name           string
		blockHeight    int64
		expectedNonces []int64
		description    string
	}{
		{
			name:           "one_block_before_window_opens_nonce1",
			blockHeight:    999,
			expectedNonces: []int64{},
			description:    "Should have no open nonces one block before window opens (tests exclusive lower bound)",
		},
		{
			name:           "exactly_at_window_start_nonce1",
			blockHeight:    1000,
			expectedNonces: []int64{1000},
			description:    "Should have nonce1 open exactly at window start (tests inclusive lower bound)",
		},
		{
			name:           "within_window_nonce1",
			blockHeight:    1025,
			expectedNonces: []int64{1000},
			description:    "Should have nonce1 open within window, nonce2 closed",
		},
		{
			name:           "exactly_at_window_end_nonce1",
			blockHeight:    1050,
			expectedNonces: []int64{1000},
			description:    "Should have nonce1 open exactly at window end (tests inclusive upper bound)",
		},
		{
			name:           "after_window_closes_nonce1",
			blockHeight:    1051,
			expectedNonces: []int64{},
			description:    "Should have no open nonces after window closes for nonce1",
		},
		{
			name:           "one_block_before_window_opens_nonce2",
			blockHeight:    1999,
			expectedNonces: []int64{},
			description:    "Should have no open nonces one block before window opens for nonce2 (tests exclusive lower bound)",
		},
		{
			name:           "exactly_at_window_start_nonce2",
			blockHeight:    2000,
			expectedNonces: []int64{2000},
			description:    "Should have nonce2 open exactly at window start (tests inclusive lower bound)",
		},
		{
			name:           "within_window_nonce2",
			blockHeight:    2025,
			expectedNonces: []int64{2000},
			description:    "Should have nonce2 open within window, nonce1 closed",
		},
		{
			name:           "exactly_at_window_end_nonce2",
			blockHeight:    2050,
			expectedNonces: []int64{2000},
			description:    "Should have nonce2 open exactly at window end (tests inclusive upper bound)",
		},
		{
			name:           "after_window_closes_nonce2",
			blockHeight:    2051,
			expectedNonces: []int64{},
			description:    "Should have no open nonces after window closes for nonce2",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.WithBlockHeight(tt.blockHeight)
			openNonces, err := k.GetOpenWorkerSubmissionWindows(s.Ctx(), topicId)
			s.Require().NoError(err, tt.description)
			s.Require().Len(openNonces.Nonces, len(tt.expectedNonces), tt.description)

			// Verify the expected nonces are present
			actualNonceHeights := make([]int64, len(openNonces.Nonces))
			for i, nonce := range openNonces.Nonces {
				actualNonceHeights[i] = nonce.BlockHeight
			}

			s.Require().ElementsMatch(tt.expectedNonces, actualNonceHeights, tt.description)
		})
	}
}

func (s *KeeperTestSuite) TestGetOpenReputerSubmissionWindowsWithMultipleNoncesInWindow() {
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(50),
	)

	// Add multiple nonces that will be open at the same time
	// nonce1: BlockHeight 1000, opens at 1100 (1000 + 100 GTLag), closes at 1200 (1100 + 100 epoch)
	// nonce2: BlockHeight 1100, opens at 1200 (1100 + 100 GTLag), closes at 1300 (1200 + 100 epoch)
	nonce1 := &types.Nonce{BlockHeight: 1000} // Open at 1100-1200
	nonce2 := &types.Nonce{BlockHeight: 1100} // Open at 1200-1300
	err := k.AddReputerNonce(s.Ctx(), topicId, nonce1)
	s.Require().NoError(err)
	err = k.AddReputerNonce(s.Ctx(), topicId, nonce2)
	s.Require().NoError(err)

	tests := []struct {
		name           string
		blockHeight    int64
		expectedNonces []int64
		description    string
	}{
		{
			name:           "before_nonce1_opens",
			blockHeight:    1099,
			expectedNonces: []int64{},
			description:    "Should have no open nonces before nonce1 window opens",
		},
		{
			name:           "exactly_when_nonce1_opens",
			blockHeight:    1100,
			expectedNonces: []int64{1000},
			description:    "Should have only nonce1 open exactly when its window opens",
		},
		{
			name:           "middle_of_nonce1_only",
			blockHeight:    1150,
			expectedNonces: []int64{1000},
			description:    "Should have only nonce1 open in the middle of its window",
		},
		{
			name:           "exactly_when_both_overlap",
			blockHeight:    1200,
			expectedNonces: []int64{1000, 1100},
			description:    "Should have both nonces open exactly when windows overlap (nonce1 at end, nonce2 at start)",
		},
		{
			name:           "middle_of_both_windows",
			blockHeight:    1250,
			expectedNonces: []int64{1100},
			description:    "Should have only nonce2 open in the middle of its window (nonce1 closed)",
		},
		{
			name:           "exactly_when_nonce2_closes",
			blockHeight:    1300,
			expectedNonces: []int64{1100},
			description:    "Should have only nonce2 open exactly when its window closes (inclusive upper bound)",
		},
		{
			name:           "after_both_close",
			blockHeight:    1301,
			expectedNonces: []int64{},
			description:    "Should have no open nonces after both windows close",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.WithBlockHeight(tt.blockHeight)
			openNonces, err := k.GetOpenReputerSubmissionWindows(s.Ctx(), topicId)
			s.Require().NoError(err, tt.description)
			s.Require().Len(openNonces.Nonces, len(tt.expectedNonces), tt.description)

			// Verify the expected nonces are present
			actualNonceHeights := make([]int64, len(openNonces.Nonces))
			for i, nonce := range openNonces.Nonces {
				actualNonceHeights[i] = nonce.ReputerNonce.BlockHeight
			}

			s.Require().ElementsMatch(tt.expectedNonces, actualNonceHeights, tt.description)
		})
	}
}

func (s *KeeperTestSuite) TestGetOpenReputerSubmissionWindowsWithNonExistentTopic() {
	k := s.EmissionsKeeper()
	nonExistentTopicId := uint64(99999)

	_, err := k.GetOpenReputerSubmissionWindows(s.Ctx(), nonExistentTopicId)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrTopicDoesNotExist, "Retrieving a non-existent topic should result in an error")
}

func (s *KeeperTestSuite) TestGetOpenWorkerSubmissionWindowsWithNonExistentTopic() {
	k := s.EmissionsKeeper()
	nonExistentTopicId := uint64(99999)

	_, err := k.GetOpenWorkerSubmissionWindows(s.Ctx(), nonExistentTopicId)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrTopicDoesNotExist, "Retrieving a non-existent topic should result in an error")
}

func (s *KeeperTestSuite) TestReputerNonceLimitEnforcement() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	maxUnfulfilledRequests := 3

	// Set the maximum number of unfulfilled reputer nonces
	params := types.DefaultParams()
	params.MaxUnfulfilledReputerRequests = uint64(maxUnfulfilledRequests)

	// Set the maximum number of unfulfilled reputer nonces via the SetParams method
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Failed to set parameters")

	// Initially add nonces to exceed the maxUnfulfilledRequests
	nonceValues := []int64{10, 20, 30, 40, 50}
	for _, val := range nonceValues {
		err := k.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Retrieve and verify the nonces to check if only the last 'maxUnfulfilledRequests' are retained
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	s.Require().NoError(err, "Error retrieving nonces after addition")
	s.Require().Len(unfulfilledNonces.Nonces, maxUnfulfilledRequests, "Should only contain max unfulfilled nonces")

	// Check that the nonces are the most recent ones
	expectedNonces := []int64{50, 40, 30} // These should be the last three nonces added
	for i, nonce := range unfulfilledNonces.Nonces {
		s.Require().Equal(expectedNonces[i], nonce.ReputerNonce.BlockHeight, "Nonce should match the expected recent nonce")
	}
}

// REGRET TESTS

func (s *KeeperTestSuite) TestSetAndGetInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(1)
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	// Set Inferer Network Regret
	err := k.SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	gotRegret, _, err := k.GetInfererNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(3) // Assuming sdk.AccAddress is initialized with a string representing the address

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	// Set Forecaster Network Regret
	err := k.SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	gotRegret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
}

func (s *KeeperTestSuite) TestSetAndGetOneInForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)

	regret := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(30)}

	// Set One-In Forecaster Network Regret
	err := k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One-In Forecaster Network Regret
	gotRegret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentInfererRegrets() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	worker := s.AddrsStr(1)

	// Topic IDs
	topicId1 := s.CreateTopic()
	topicId2 := s.CreateTopic()

	// Zero regret for initial check
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, _, err := k.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")

	gotRegret2, _, err := k.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same worker under different topic IDs
	err = k.SetInfererNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, _, err = k.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err = k.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentForecasterRegrets() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	worker := s.AddrsStr(1)

	// Topic IDs
	topicId1 := s.CreateTopic()
	topicId2 := s.CreateTopic()

	// Regrets
	noRagret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	gotRegret1, _, err := k.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRagret, gotRegret1)

	// Set regrets for the same worker under different topic IDs
	err = k.SetForecasterNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, _, err = k.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err := k.GetForecasterNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentOneInForecasterNetworkRegrets() {
	ctx := s.Ctx()
	topicId1 := s.CreateTopic() // Topic 1
	topicId2 := s.CreateTopic() // Topic 2
	k := s.EmissionsKeeper()
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)

	// Zero regret for initial checks
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")

	gotRegret2, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same forecaster-inferer pair under different topic IDs
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer, regret1)
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, _, err = k.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err = k.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestSetAndGetNaiveInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(1)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	err := k.SetNaiveInfererNetworkRegret(ctx, topicId, inferer, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetNaiveInfererNetworkRegret(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutInfererInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	inferer1 := s.AddrsStr(1)
	inferer2 := s.AddrsStr(2)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(15)}

	err := k.SetOneOutInfererInfererNetworkRegret(ctx, topicId, inferer1, inferer2, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutInfererInfererNetworkRegret(ctx, topicId, inferer1, inferer2)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutInfererForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(1)
	forecaster := s.AddrsStr(3)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	err := k.SetOneOutInfererForecasterNetworkRegret(ctx, topicId, inferer, forecaster, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutInfererForecasterNetworkRegret(ctx, topicId, inferer, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutForecasterInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(25)}

	err := k.SetOneOutForecasterInfererNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutForecasterInfererNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutForecasterForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	forecaster1 := s.AddrsStr(3)
	forecaster2 := s.AddrsStr(4)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(30)}

	err := k.SetOneOutForecasterForecasterNetworkRegret(ctx, topicId, forecaster1, forecaster2, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutForecasterForecasterNetworkRegret(ctx, topicId, forecaster1, forecaster2)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

// PARAMS TESTS
func (s *KeeperTestSuite) TestSetGetMaxTopicsPerBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := uint64(100)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxActiveTopicsPerBlock
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetRemoveStakeDelayWindow() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := types.BlockHeight(50)

	// Set the parameter
	params := types.DefaultParams()
	params.RemoveStakeDelayWindow = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RemoveStakeDelayWindow
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetValidatorsVsAlloraPercentReward() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := alloraMath.MustNewDecFromString("0.25") // Assume a function to create LegacyDec

	// Set the parameter
	params := types.DefaultParams()
	params.ValidatorsVsAlloraPercentReward = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.ValidatorsVsAlloraPercentReward
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinTopicUnmetDemand() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := alloraMath.NewDecFromInt64(300)

	// Set the parameter
	params := types.DefaultParams()
	params.MinTopicWeight = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinTopicWeight
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsRequiredMinimumStake() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue, ok := cosmosMath.NewIntFromString("500")
	s.Require().True(ok)

	// Set the parameter
	params := types.DefaultParams()
	params.RequiredMinimumStake = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RequiredMinimumStake
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinEpochLength() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := types.BlockHeight(720)

	// Set the parameter
	params := types.DefaultParams()
	params.MinEpochLength = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinEpochLength
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsEpsilon() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := alloraMath.MustNewDecFromString("0.1234")

	// Set the parameter
	params := types.DefaultParams()
	params.EpsilonReputer = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.EpsilonReputer
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsTopicCreationFee() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := cosmosMath.NewInt(1000)

	// Set the parameter
	params := types.DefaultParams()
	params.CreateTopicFee = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.CreateTopicFee
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsRegistrationFee() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := cosmosMath.NewInt(500)

	// Set the parameter
	params := types.DefaultParams()
	params.RegistrationFee = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RegistrationFee
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsMaxSamplesToScaleScores() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := uint64(1500)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxSamplesToScaleScores
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMaxTopInferersToReward() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxTopInferersToReward = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxTopInferersToReward
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopInferersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxTopForecastersToReward() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxTopForecastersToReward = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter

	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxTopForecastersToReward
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopForecastersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxTopForecasterElementToSubmit() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxElementsPerForecast = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter

	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxElementsPerForecast
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxElementsPerForecast should match the expected value")
}

func (s *KeeperTestSuite) TestGetMinEpochLengthRecordLimit() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := int64(10)

	// Set the parameter
	params := types.DefaultParams()
	params.MinEpochLengthRecordLimit = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinEpochLengthRecordLimit
	s.Require().Equal(expectedValue, actualValue, "The retrieved MinEpochLengthRecordLimit should be equal to the expected value")
}

func (s *KeeperTestSuite) TestGetMaxSerializedMsgLength() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	expectedValue := int64(2048)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxSerializedMsgLength = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxSerializedMsgLength
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxSerializedMsgLength should be equal to the expected value")
}

// INFERENCES, FORECASTS

func (s *KeeperTestSuite) TestGetInferencesAtBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)}, // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertActiveInferences correctly sets up inferences
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	// Retrieve inferences
	actualInferences, err := k.GetInferencesAtBlock(ctx, topicId, block, false)
	s.Require().NoError(err)
	s.Require().Equal(&expectedInferences, actualInferences)
}

func (s *KeeperTestSuite) TestGetInferencesAtBlockOutlierResistant() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	// Force setting values of MAD and last_median to 10
	err := k.SetLastMedianInferences(ctx, topicId, alloraMath.NewDecFromInt64(150))
	s.Require().NoError(err)
	err = k.SetMadInferences(ctx, topicId, alloraMath.NewDecFromInt64(10))
	s.Require().NoError(err)

	// Create a set of inferences with a high value to test outlier resistant filtering
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(100)}, // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(200)},
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(10000)},
				Inferer:     "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Insert directly as active inferences
	nonce := types.Nonce{BlockHeight: block}
	err = k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err)

	// Confirm the non-or keeps all inferences
	actualInferences, err := k.GetInferencesAtBlock(ctx, topicId, block, false)
	s.Require().NoError(err)
	s.Require().Len(actualInferences.Inferences, 3)

	actualInferences, err = k.GetInferencesAtBlock(ctx, topicId, block, true)
	s.Require().NoError(err)
	s.Require().Len(actualInferences.Inferences, 2)
	s.Require().Equal(alloraMath.NewDecFromInt64(100), actualInferences.Inferences[0].Values[0])
	s.Require().Equal(alloraMath.NewDecFromInt64(200), actualInferences.Inferences[1].Values[0])
}

func (s *KeeperTestSuite) TestGetLatestTopicInferences() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topicId := uint64(1)

	// Initially, there should be no inferences, so we expect an empty result
	emptyInferences, emptyBlockHeight, err := k.GetLatestTopicInferences(ctx, topicId, false)
	s.Require().NoError(err, "Retrieving latest inferences when none exist should not result in an error")
	s.Require().Equal(&types.Inferences{Inferences: []*types.Inference{}}, emptyInferences, "Expected no inferences initially")
	s.Require().Equal(types.BlockHeight(0), emptyBlockHeight, "Expected block height to be zero initially")

	// Insert first set of inferences
	blockHeight1 := types.BlockHeight(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("10")},
		ExtraData:   []byte("data1"),
		Proof:       "proof1",
	}
	inferences1 := types.Inferences{
		Inferences: []*types.Inference{&newInference1},
	}
	nonce1 := types.Nonce{BlockHeight: blockHeight1}
	err = k.InsertActiveInferences(ctx, topicId, nonce1.BlockHeight, inferences1)
	s.Require().NoError(err, "Inserting first set of inferences should not fail")

	// Insert second set of inferences
	blockHeight2 := types.BlockHeight(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     "allo1dwxj49n0t5969uj4zfuemxg8a2ty85njn9xy9t",
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("20")},
		ExtraData:   []byte("data2"),
		Proof:       "proof2",
	}
	inferences2 := types.Inferences{
		Inferences: []*types.Inference{&newInference2},
	}
	nonce2 := types.Nonce{BlockHeight: blockHeight2}
	err = k.InsertActiveInferences(ctx, topicId, nonce2.BlockHeight, inferences2)
	s.Require().NoError(err, "Inserting second set of inferences should not fail")

	// Retrieve the latest inferences
	latestInferences, latestBlockHeight, err := k.GetLatestTopicInferences(ctx, topicId, false)
	s.Require().NoError(err, "Retrieving latest inferences should not fail")
	s.Require().Equal(&inferences2, latestInferences, "Latest inferences should match the second inserted set")
	s.Require().Equal(blockHeight2, latestBlockHeight, "Latest block height should match the second inserted set")
}

func (s *KeeperTestSuite) TestGetWorkerLatestInferenceByTopicId() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topicId := uint64(1)
	workerAccStr := "allo1xy0pf5hq85j873glav6aajkvtennmg3fpu3cec"

	_, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().Error(err, "Retrieving an inference that does not exist should result in an error")

	blockHeight1 := int64(12345)
	newInference1 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight1,
		Inferer:     workerAccStr,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("10")},
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.InsertInference(ctx, topicId, newInference1)
	s.Require().NoError(err, "Inserting inferences should not fail")

	blockHeight2 := int64(12346)
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight2,
		Inferer:     workerAccStr,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("10")},
		ExtraData:   []byte("data"),
		Proof:       "proof123",
	}
	err = k.InsertInference(ctx, topicId, newInference2)
	s.Require().NoError(err, "Inserting inferences should not fail")

	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, workerAccStr)
	s.Require().NoError(err, "Retrieving an existing inference should not fail")
	s.Require().Equal(newInference2, retrievedInference, "Retrieved inference should match the inserted one")
}

func (s *KeeperTestSuite) TestGetForecastsAtBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   alloraMath.MustNewDecFromString("1"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   alloraMath.MustNewDecFromString("2"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
						Value:   alloraMath.MustNewDecFromString("3"),
					},
					{
						Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
						Value:   alloraMath.MustNewDecFromString("4"),
					},
				},
			},
		},
	}

	// Assume InsertActiveForecasts correctly sets up forecasts
	nonce := types.Nonce{BlockHeight: block}
	err := k.InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	// Retrieve forecasts
	actualForecasts, err := k.GetForecastsAtBlock(ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().Equal(&expectedForecasts, actualForecasts)
}

func (s *KeeperTestSuite) TestInsertActiveReputerLosses() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	//nolint:exhaustruct
	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: block},
		},
		Reputer:       s.AddrsStr(0),
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}

	// Test inserting data
	err := s.EmissionsKeeper().InsertActiveReputerLosses(ctx, topicId, block, types.LossBundles{valueBundle})
	require.NoError(err, "InsertActiveReputerLosses should not return an error")

	// Retrieve data to verify insertion
	result, err := s.EmissionsKeeper().GetReputerLossBundlesAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(types.LossBundles{valueBundle}, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetReputerLossBundlesAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Test getting data before any insert, should return error or nil
	result, err := s.EmissionsKeeper().GetReputerLossBundlesAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.Empty(result, "Result should be empty for non-existent data")
}

func (s *KeeperTestSuite) TestInsertNetworkLossBundleAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	//nolint:exhaustruct
	lossBundle := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: block},
		},
		Reputer:       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}

	err := s.EmissionsKeeper().InsertNetworkLossBundleAtBlock(ctx, topicId, block, lossBundle)
	require.NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Verify the insertion
	result, err := s.EmissionsKeeper().GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err)
	require.NotNil(result)
	require.Equal(&lossBundle, result, "Retrieved data should match inserted data")
}

// this unit test needs to be completely rewritten, PROTO-2575
func (s *KeeperTestSuite) TestGetNetworkLossBundleAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)

	// Attempt to retrieve before insertion
	result, err := s.EmissionsKeeper().GetNetworkLossBundleAtBlock(ctx, topicId, block)
	require.NoError(err, "Should not return error for empty loss bundle")
	require.NotNil(result)
	require.Equal(topicId, result.TopicId, "Result should be not be nil for non-existent data")
}

func (s *KeeperTestSuite) TestGetLatestNetworkLossBundle() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()

	// Initially, there should be no loss bundle, so we expect a zero result
	emptyLossBundle, err := k.GetLatestNetworkLossBundle(ctx, topicId)
	s.Require().ErrorIs(err, types.ErrNotFound)
	s.Require().Nil(emptyLossBundle, "Expected no network loss bundle initially")

	// Insert first network loss bundle
	blockHeight1 := types.BlockHeight(100)
	//nolint:exhaustruct
	lossBundle1 := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight1},
		},
		Reputer:       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}
	err = k.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight1, lossBundle1)
	s.Require().NoError(err, "Inserting first network loss bundle should not fail")

	// Insert second network loss bundle
	blockHeight2 := types.BlockHeight(200)
	//nolint:exhaustruct
	lossBundle2 := types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight2},
		},
		Reputer:       "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("456"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}
	err = k.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight2, lossBundle2)
	s.Require().NoError(err, "Inserting second network loss bundle should not fail")

	// Retrieve the latest network loss bundle
	latestLossBundle, err := k.GetLatestNetworkLossBundle(ctx, topicId)
	s.Require().NoError(err, "Retrieving latest network loss bundle should not fail")
	s.Require().Equal(&lossBundle2, latestLossBundle, "Latest network loss bundle should match the second inserted set")
}

// ########################################
// #           Staking tests              #
// ########################################

func (s *KeeperTestSuite) TestGetSetTotalStake() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Set total stake
	newTotalStake := cosmosMath.NewInt(1000)
	err := k.SetTotalStake(ctx, newTotalStake)
	s.Require().NoError(err)

	// Check total stake
	totalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(newTotalStake, totalStake)
}

func (s *KeeperTestSuite) TestAddReputerStake() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)

	// Initial Values
	initialTotalStake := cosmosMath.NewInt(0)
	initialTopicStake := cosmosMath.NewInt(0)

	// Add stake
	err := k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := k.GetStakeReputerAuthority(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, delegatorStake, "Delegator stake should be equal to stake amount after addition")

	// Check updated topic stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(initialTopicStake.Add(stakeAmount), topicStake, "Topic stake should be incremented by stake amount after addition")

	// Check updated total stake
	totalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().Equal(initialTotalStake.Add(stakeAmount), totalStake, "Total stake should be incremented by stake amount after addition")
}

func (s *KeeperTestSuite) TestAddDelegateStake() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	reputerAddr := s.AddrsStr(1)
	initialStakeAmount := cosmosMath.NewInt(500)
	additionalStakeAmount := cosmosMath.NewInt(300)

	// Setup initial stake
	err := k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount, delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")

	// Add additional stake
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, additionalStakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator
	delegatorStake, err = k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Add(additionalStakeAmount), delegatorStake, "Total delegator stake should be the sum of initial and additional stake amounts")
}

func (s *KeeperTestSuite) TestAddReputerStakeZeroAmount() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	zeroStakeAmount := cosmosMath.NewInt(0)

	// Try to add zero stake
	err := k.AddReputerStake(ctx, topicId, delegatorAddr, zeroStakeAmount)
	s.Require().ErrorIs(err, types.ErrInvalidValue)
}

func (s *KeeperTestSuite) TestRemoveStake() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Capture the initial total and topic stakes after adding stake
	initialTotalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)

	// make a request to remove stake
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check updated stake for delegator after removal
	delegatorStake, err := k.GetStakeReputerAuthority(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), delegatorStake, "Delegator stake should be zero after removal")

	// Check updated topic stake after removal
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")

	// Check updated total stake after removal
	finalTotalStake, err := k.GetTotalStake(ctx)
	s.Require().NoError(err)
	s.Require().True(initialTotalStake.Sub(stakeAmount).Equal(finalTotalStake), "Total stake should be decremented by stake amount after removal")
}

func (s *KeeperTestSuite) TestRemovePartialStakeFromDelegator() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	reputerAddr := s.AddrsStr(1)
	initialStakeAmount := cosmosMath.NewInt(1000)
	removeStakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = k.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                removeStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = k.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, removeStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for delegator
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveEntireStakeFromDelegator() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	delegatorAddr := s.AddrsStr(0)
	reputerAddr := s.AddrsStr(1)
	initialStakeAmount := cosmosMath.NewInt(1000)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = k.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                initialStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = k.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for Reputer
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestRemoveStakeZeroAmount() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	initialStakeAmount := cosmosMath.NewInt(500)
	zeroStakeAmount := cosmosMath.NewInt(0)

	// Setup initial stake
	err := k.AddReputerStake(ctx, topicId, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// Try to remove zero stake
	err = k.RemoveReputerStake(ctx, ctx.BlockHeight(), topicId, reputerAddr, zeroStakeAmount)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestRemoveStakeNonExistingDelegatorOrTarget() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	nonExistingDelegatorAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)

	// Try to remove stake with non-existing delegator or target
	err := k.RemoveReputerStake(ctx, ctx.BlockHeight(), topicId, nonExistingDelegatorAddr, stakeAmount)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestGetAllStakeForDelegator() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	delegatorAddr := s.AddrsStr(0)

	// Mock setup
	topicId := uint64(1)
	targetAddr := s.AddrsStr(1)
	stakeAmount := cosmosMath.NewInt(500)

	// Add stake to create bonds
	err := k.AddDelegateStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount)
	s.Require().NoError(err)

	// Add stake to create bonds
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, targetAddr, stakeAmount.Mul(cosmosMath.NewInt(2)))
	s.Require().NoError(err)

	// Get all bonds for delegator
	amount, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)

	s.Require().NoError(err, "Getting all bonds for delegator should not return an error")
	s.Require().Equal(stakeAmount.Mul(cosmosMath.NewInt(3)), amount, "The total amount is incorrect")
}

func (s *KeeperTestSuite) TestSetGetDeleteStakeRemovalByAddressWithDetailedPlacement() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic0 := uint64(101)
	reputer0 := "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n"

	topic1 := uint64(102)
	reputer1 := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	// Create a sample stake removal information
	removalInfo0 := types.StakeRemovalInfo{
		BlockRemovalStarted:   12,
		BlockRemovalCompleted: 13,
		TopicId:               topic0,
		Reputer:               reputer0,
		Amount:                cosmosMath.NewInt(100),
	}
	removalInfo1 := types.StakeRemovalInfo{
		BlockRemovalStarted:   13,
		BlockRemovalCompleted: 14,
		TopicId:               topic1,
		Reputer:               reputer1,
		Amount:                cosmosMath.NewInt(200),
	}

	// Set stake removal information
	err := k.SetStakeRemoval(ctx, removalInfo0)
	s.Require().NoError(err)
	err = k.SetStakeRemoval(ctx, removalInfo1)
	s.Require().NoError(err)

	// Topic 101

	// Retrieve the stake removal information
	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 1)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 1, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit, "The limit should not be hit")
	s.Require().Equal(removalInfo0.BlockRemovalStarted, retrievedInfo[0].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo0.BlockRemovalCompleted, retrievedInfo[0].BlockRemovalCompleted, "Block removal completed should match")
	s.Require().Equal(removalInfo0.TopicId, retrievedInfo[0].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo0.Reputer, retrievedInfo[0].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo0.Amount, retrievedInfo[0].Amount, "Amounts should match for all placements")

	// Topic 102

	// Retrieve the stake removal information
	retrievedInfo, limitHit, err = k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 2)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 2, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit, "The limit should not be hit")
	s.Require().Equal(removalInfo1.BlockRemovalStarted, retrievedInfo[1].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo1.BlockRemovalCompleted, retrievedInfo[1].BlockRemovalCompleted, "Block removal started should match")
	s.Require().Equal(removalInfo1.TopicId, retrievedInfo[1].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo1.Reputer, retrievedInfo[1].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo1.Amount, retrievedInfo[1].Amount, "Amounts should match for all placements")

	// delete 101
	err = k.DeleteStakeRemoval(ctx, removalInfo0.BlockRemovalCompleted, removalInfo0.TopicId, removalInfo0.Reputer)
	s.Require().NoError(err)
	removals, limitHit, err := k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")

	// delete 102
	err = k.DeleteStakeRemoval(ctx, removalInfo1.BlockRemovalCompleted, removalInfo1.TopicId, removalInfo1.Reputer)
	s.Require().NoError(err)
	removals, limitHit, err = k.GetStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockNotFound() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Attempt to retrieve stake removal info for an address with no set info
	removals, limitHit, err := k.GetStakeRemovalsUpUntilBlock(ctx, 202, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitPreviousBlocks() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := int64(13)

	topicId := topicIdStart
	reputer := s.AddrsStr(2)
	removalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   blockRemovalsStart,
		BlockRemovalCompleted: blockRemovalsEnd,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                cosmosMath.NewInt(100),
	}
	err := k.SetStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+1, // note how we are getting a block AFTER blockRemovalsEnd
		1000,
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, 1)
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitExactBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := int64(13)

	topicId := topicIdStart
	reputer := s.AddrsStr(2)
	removalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   blockRemovalsStart,
		BlockRemovalCompleted: blockRemovalsEnd,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                cosmosMath.NewInt(100),
	}
	err := k.SetStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd,
		1000,
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, 1)
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitGreaterThanNumRemovals() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	numRemovals := int64(5)
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := types.DefaultParams().RemoveStakeDelayWindow + blockRemovalsStart

	for i := int64(0); i < numRemovals; i++ {
		topicId := topicIdStart + uint64(i)
		reputer := s.AddrsStr(2)
		// Create a sample stake removal information
		removalInfo := types.StakeRemovalInfo{
			BlockRemovalStarted:   blockRemovalsStart + i,
			BlockRemovalCompleted: blockRemovalsEnd + i,
			TopicId:               topicId,
			Reputer:               reputer,
			Amount:                cosmosMath.NewInt(100),
		}
		err := k.SetStakeRemoval(ctx, removalInfo)
		s.Require().NoError(err)
	}
	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+numRemovals,
		uint64(numRemovals),
	)
	s.Require().NoError(err)
	s.Require().False(limitHit)
	s.Require().Len(retrievedInfo, int(numRemovals))
}

func (s *KeeperTestSuite) TestGetStakeRemovalsUpUntilBlockLimitLessThanNumRemovals() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	numRemovals := int64(5)
	limitRemovals := numRemovals - 2
	topicIdStart := uint64(100)
	blockRemovalsStart := int64(12)
	blockRemovalsEnd := types.DefaultParams().RemoveStakeDelayWindow + blockRemovalsStart

	for i := int64(0); i < numRemovals; i++ {
		topicId := topicIdStart + uint64(i)
		reputer := s.AddrsStr(2)
		// Create a sample stake removal information
		removalInfo := types.StakeRemovalInfo{
			BlockRemovalStarted:   blockRemovalsStart + i,
			BlockRemovalCompleted: blockRemovalsEnd + i,
			TopicId:               topicId,
			Reputer:               reputer,
			Amount:                cosmosMath.NewInt(100),
		}
		err := k.SetStakeRemoval(ctx, removalInfo)
		s.Require().NoError(err)
	}
	retrievedInfo, limitHit, err := k.GetStakeRemovalsUpUntilBlock(
		ctx,
		blockRemovalsEnd+numRemovals,
		uint64(limitRemovals),
	)
	s.Require().NoError(err)
	s.Require().True(limitHit)
	s.Require().Len(retrievedInfo, int(limitRemovals))
}

func (s *KeeperTestSuite) TestSetGetDeleteDelegateStakeRemovalByAddress() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic0 := uint64(201)
	reputer0 := "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n"
	delegator0 := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	topic1 := uint64(202)
	reputer1 := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"
	delegator1 := "allo16skpmhw8etsu70kknkmxquk5ut7lsewgtqqtlu"

	// Create sample delegate stake removal information
	removalInfo0 := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   12,
		BlockRemovalCompleted: 13,
		TopicId:               topic0,
		Reputer:               reputer0,
		Delegator:             delegator0,
		Amount:                cosmosMath.NewInt(300),
	}
	removalInfo1 := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   13,
		BlockRemovalCompleted: 14,
		TopicId:               topic1,
		Reputer:               reputer1,
		Delegator:             delegator1,
		Amount:                cosmosMath.NewInt(400),
	}

	// Set delegate stake removal information
	err := k.SetDelegateStakeRemoval(ctx, removalInfo0)
	s.Require().NoError(err)
	err = k.SetDelegateStakeRemoval(ctx, removalInfo1)
	s.Require().NoError(err)

	// Topic 201

	// Retrieve the delegate stake removal information
	retrievedInfo, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 1, "There should be only one delegate stake removal information for the block")
	s.Require().False(limitHit)
	s.Require().Equal(removalInfo0.BlockRemovalStarted, retrievedInfo[0].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo0.TopicId, retrievedInfo[0].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo0.Reputer, retrievedInfo[0].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo0.Delegator, retrievedInfo[0].Delegator, "Delegator addresses should match for all placements")
	s.Require().Equal(removalInfo0.Amount, retrievedInfo[0].Amount, "Amounts should match for all placements")

	// Topic 202

	// Retrieve the delegate stake removal information
	retrievedInfo, limitHit, err = k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Len(retrievedInfo, 2)
	s.Require().False(limitHit)
	s.Require().Equal(removalInfo1.BlockRemovalStarted, retrievedInfo[1].BlockRemovalStarted, "Block removal started should match")
	s.Require().Equal(removalInfo1.TopicId, retrievedInfo[1].TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(removalInfo1.Reputer, retrievedInfo[1].Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(removalInfo1.Delegator, retrievedInfo[1].Delegator, "Delegator addresses should match for all placements")
	s.Require().Equal(removalInfo1.Amount, retrievedInfo[1].Amount, "Amounts should match for all placements")

	// delete 101
	err = k.DeleteDelegateStakeRemoval(ctx, removalInfo0.BlockRemovalCompleted, removalInfo0.TopicId, removalInfo0.Reputer, removalInfo0.Delegator)
	s.Require().NoError(err)
	removals, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo0.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit)

	// delete 102
	err = k.DeleteDelegateStakeRemoval(ctx, removalInfo1.BlockRemovalCompleted, removalInfo1.TopicId, removalInfo1.Reputer, removalInfo1.Delegator)
	s.Require().NoError(err)
	removals, limitHit, err = k.GetDelegateStakeRemovalsUpUntilBlock(ctx, removalInfo1.BlockRemovalCompleted, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit)
}

func (s *KeeperTestSuite) TestGetDeleteDelegateStake() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Create sample delegate stake removal information
	removalInfo := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   int64(12),
		BlockRemovalCompleted: int64(13),
		TopicId:               uint64(201),
		Reputer:               "allo146fyx5akdrcpn2ypjpg4tra2l7q2wevs05pz2n",
		Delegator:             "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		Amount:                cosmosMath.NewInt(300),
	}

	// Set delegate stake removal information
	err := k.SetDelegateStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	_, err = k.GetDelegateStakeRemoval(ctx,
		removalInfo.BlockRemovalStarted,
		removalInfo.TopicId,
		removalInfo.Delegator,
		removalInfo.Reputer,
	)
	// index is on BlockRemovalCompleted not BlockRemovalStarted
	s.Require().Error(err)

	retrievedInfo, err := k.GetDelegateStakeRemoval(ctx,
		removalInfo.BlockRemovalCompleted,
		removalInfo.TopicId,
		removalInfo.Delegator,
		removalInfo.Reputer,
	)
	s.Require().NoError(err)

	s.Require().Equal(removalInfo.BlockRemovalStarted, retrievedInfo.BlockRemovalStarted)
	s.Require().Equal(removalInfo.TopicId, retrievedInfo.TopicId)
	s.Require().Equal(removalInfo.Reputer, retrievedInfo.Reputer)
	s.Require().Equal(removalInfo.Delegator, retrievedInfo.Delegator)
	s.Require().Equal(removalInfo.Amount, retrievedInfo.Amount)
}

func (s *KeeperTestSuite) TestGetDelegateStakeRemovalByAddressNotFound() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Attempt to retrieve delegate stake removal info for an address with no set info
	removals, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(ctx, 201, 100)
	s.Require().NoError(err)
	s.Require().Empty(removals)
	s.Require().False(limitHit, "The limit should not be hit")
}

func (s *KeeperTestSuite) TestSetParams() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	params := types.DefaultParams()
	// Set params
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Check params
	paramsFromKeeper, err := k.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, paramsFromKeeper, "Params should be equal to the set params")
}

// REPUTERS AND WORKER
func (s *KeeperTestSuite) TestInsertWorker() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	topicId := uint64(401)

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h",
		NodeAddress: worker,
	}

	// Attempt to insert the worker for multiple topics
	err := k.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(err)

	node, err := k.GetWorkerInfo(ctx, worker)

	s.Require().NoError(err)
	s.Require().Equal(workerInfo.Owner, node.Owner)
	s.Require().Equal(workerInfo.NodeAddress, node.NodeAddress)
}

func (s *KeeperTestSuite) TestUpdateWorkerOwner() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topicId := uint64(402)
	worker := s.AddrsStr(0)
	initialOwner := s.AddrsStr(1)
	newOwner := s.AddrsStr(2)
	nonRegisteredWorker := s.AddrsStr(3)

	err := k.InsertWorker(ctx, topicId, worker, types.OffchainNode{
		NodeAddress: worker,
		Owner:       initialOwner,
	})
	s.Require().NoError(err)

	oldOwner, err := k.UpdateWorkerOwner(ctx, worker, newOwner)
	s.Require().NoError(err)
	s.Require().Equal(initialOwner, oldOwner)

	node, err := k.GetWorkerInfo(ctx, worker)
	s.Require().NoError(err)
	s.Require().Equal(newOwner, node.Owner)

	_, err = k.UpdateWorkerOwner(ctx, nonRegisteredWorker, newOwner)
	s.Require().ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *KeeperTestSuite) TestRemoveWorker() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	topicId := uint64(401) // Assume the worker is associated with this topicId initially

	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       "allo1wmvlvr82nlnu2y6hewgjwex30spyqgzvjhc80h",
		NodeAddress: "allo195jgulwj7vd292m0fth5gwqu4r2447dnarunmx",
	}

	// Insert the worker
	insertErr := k.InsertWorker(ctx, topicId, worker, workerInfo)
	s.Require().NoError(insertErr, "Failed to insert worker initially")

	// Verify the worker is registered in the topic
	isRegisteredPre, preErr := k.IsWorkerRegisteredInTopic(ctx, topicId, worker)
	s.Require().NoError(preErr, "Failed to check worker registration before removal")
	s.Require().True(isRegisteredPre, "Worker should be registered in the topic before removal")

	// Perform the removal
	removeErr := k.RemoveWorker(ctx, topicId, worker)
	s.Require().NoError(removeErr, "Failed to remove worker")

	// Verify the worker is no longer registered in the topic
	isRegisteredPost, postErr := k.IsWorkerRegisteredInTopic(ctx, topicId, worker)
	s.Require().NoError(postErr, "Failed to check worker registration after removal")
	s.Require().False(isRegisteredPost, "Worker should not be registered in the topic after removal")
}

func (s *KeeperTestSuite) TestInsertReputer() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	reputer := s.AddrsStr(0)
	topicId := uint64(501)

	// Define sample OffchainNode information for a reputer
	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(1),
		NodeAddress: s.AddrsStr(2),
	}

	// Attempt to insert the reputer for multiple topics
	err := k.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	// Optionally check if reputer is registered in each topic using an assumed IsReputerRegisteredInTopic method
	isRegistered, regErr := k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(regErr, "Checking reputer registration should not fail")
	s.Require().True(isRegistered, "Reputer should be registered in each topic")
}

func (s *KeeperTestSuite) TestUpdateReputerOwner() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(502)
	reputer := s.AddrsStr(4)
	initialOwner := s.AddrsStr(5)
	newOwner := s.AddrsStr(6)
	nonRegisteredReputer := s.AddrsStr(7)

	err := k.InsertReputer(ctx, topicId, reputer, types.OffchainNode{
		NodeAddress: reputer,
		Owner:       initialOwner,
	})
	s.Require().NoError(err)

	oldOwner, err := k.UpdateReputerOwner(ctx, reputer, newOwner)
	s.Require().NoError(err)
	s.Require().Equal(initialOwner, oldOwner)

	stored, err := k.GetReputerInfo(ctx, reputer)
	s.Require().NoError(err)
	s.Require().Equal(newOwner, stored.Owner)

	_, err = k.UpdateReputerOwner(ctx, nonRegisteredReputer, newOwner)
	s.Require().ErrorIs(err, types.ErrAddressNotRegistered)
}

func (s *KeeperTestSuite) TestGetReputerInfo() {
	ctx := s.Ctx()
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"
	topicId := uint64(501)
	k := s.EmissionsKeeper()
	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(2),
		NodeAddress: reputer,
	}

	err := k.InsertReputer(ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)

	actualReputer, err := k.GetReputerInfo(ctx, reputer)
	s.Require().NoError(err)
	s.Require().Equal(reputerInfo, actualReputer)

	nonExistentKey := "nonExistentKey123"
	_, err = k.GetReputerInfo(ctx, nonExistentKey)
	s.Require().Error(err)
}

func (s *KeeperTestSuite) TestRemoveReputer() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"
	topicId := uint64(501)

	// Pre-setup: Insert the reputer for initial setup
	err := k.InsertReputer(ctx, topicId, reputer, types.OffchainNode{
		Owner:       s.AddrsStr(1),
		NodeAddress: reputer,
	})
	s.Require().NoError(err, "InsertReputer failed during setup")

	// Verify the reputer is registered in the topic
	isRegisteredPre, preErr := k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(preErr, "Failed to check reputer registration before removal")
	s.Require().True(isRegisteredPre, "Reputer should be registered in the topic before removal")

	// Perform the removal
	removeErr := k.RemoveReputer(ctx, topicId, reputer)
	s.Require().NoError(removeErr, "Failed to remove reputer")

	// Verify the reputer is no longer registered in the topic
	isRegisteredPost, postErr := k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	s.Require().NoError(postErr, "Failed to check reputer registration after removal")
	s.Require().False(isRegisteredPost, "Reputer should not be registered in the topic after removal")
}

// TOPICS

func (s *KeeperTestSuite) TestSetAndGetPreviousTopicWeight() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	// Set previous topic weight
	weightToSet := alloraMath.NewDecFromInt64(10)
	err := k.SetPreviousTopicWeight(ctx, topicId, weightToSet)
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Get the previously set topic weight
	retrievedWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err, "Getting previous topic weight should not fail")
	s.Require().Equal(weightToSet, retrievedWeight, "Retrieved weight should match the set weight")
	s.Require().False(noPrior, "Should indicate prior weight for a set topic")
}

func (s *KeeperTestSuite) TestGetPreviousTopicWeightNotFound() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(2)

	// Attempt to get a weight for a topic that has no set weight
	retrievedWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err, "Getting weight for an unset topic should not error but return zero value")
	s.Require().True(alloraMath.ZeroDec().Equal(retrievedWeight), "Weight for an unset topic should be zero")
	s.Require().True(noPrior, "Should indicate no prior weight for an unset topic")
}

func (s *KeeperTestSuite) TestInactivateAndActivateTopic() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(3)

	maxActiveTopicsNum := uint64(5)
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Assume topic initially active
	initialTopic := s.MockTopic()
	err = k.SetTopic(ctx, topicId, initialTopic)
	s.Require().NoError(err, "Setting topic should not fail")

	// Activate the topic
	err = k.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active
	topicActive, err := k.IsTopicActive(ctx, topicId)
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")

	// Inactivate the topic
	err = k.InactivateTopic(ctx, topicId)
	s.Require().NoError(err, "Inactivating topic should not fail")

	// Check if topic is inactive
	topicActive, err = k.IsTopicActive(ctx, topicId)
	s.Require().NoError(err, "Getting topic should not fail after inactivation")
	s.Require().False(topicActive, "Topic should be inactive")

	// Activate the topic
	err = k.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active again
	topicActive, err = k.IsTopicActive(ctx, topicId)
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")
}

func (s *KeeperTestSuite) TestGetActiveTopicIdsAtBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	maxActiveTopicsNum := uint64(2)
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	params.MaxPageLimit = 100
	params.MinEpochLength = 1
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Create topics using CreateTopic with functional options
	topic1Id := s.CreateTopic(
		testutil.WithEpochLength(5),
		testutil.WithWorkerSubmissionWindow(5),
	)
	s.CreateTopic(
		testutil.WithEpochLength(5),
		testutil.WithWorkerSubmissionWindow(5),
	) // Inactive topic
	topic3Id := s.CreateTopic(
		testutil.WithEpochLength(15),
		testutil.WithWorkerSubmissionWindow(15),
	)

	err = k.ActivateTopic(ctx, topic1Id)
	s.Require().NoError(err, "Activating topic should not fail")
	err = k.ActivateTopic(ctx, topic3Id)
	s.Require().NoError(err, "Activating topic should not fail")

	// Fetch only active topics
	activeTopics, err := k.GetActiveTopicIdsAtBlock(ctx, 5)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 1, "Should retrieve exactly one active topic")

	activeTopics, err = k.GetActiveTopicIdsAtBlock(ctx, 15)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 1, "Should retrieve exactly one active topic")
	s.Require().Equal(activeTopics.TopicIds[0], topic3Id, "The details of topic 3 should match")
}

func (s *KeeperTestSuite) TestTopicGoesInactivateOnEpochEndBlockIfLowWeight() {
	k := s.EmissionsKeeper()

	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = uint64(3)
	params.MaxPageLimit = uint64(100)
	params.MinEpochLength = 1
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.MustNewDecFromString("1")
	err := k.SetParams(s.Ctx(), params)
	s.Require().NoError(err, "Setting parameters should not fail")

	epochLength1 := int64(15)
	epochLength2 := int64(15)
	epochLength3 := int64(5)
	epochLength4 := int64(5)

	topic1Id := s.CreateTopic(
		testutil.WithEpochLength(epochLength1),
		testutil.WithWorkerSubmissionWindow(epochLength1),
	)
	topic2Id := s.CreateTopic(
		testutil.WithEpochLength(epochLength2),
		testutil.WithWorkerSubmissionWindow(epochLength2),
	)
	topic3Id := s.CreateTopic(
		testutil.WithEpochLength(epochLength3),
		testutil.WithWorkerSubmissionWindow(epochLength3),
	)
	topic4Id := s.CreateTopic(
		testutil.WithEpochLength(epochLength4),
		testutil.WithWorkerSubmissionWindow(epochLength4),
	)

	setTopicWeight := func(topicId uint64, revenue, stake int64) {
		err := k.AddTopicFeeRevenue(s.Ctx(), topicId, cosmosMath.NewInt(revenue))
		s.Require().NoError(err, "Adding topic fee revenue should not fail")
		err = k.SetTopicStake(s.Ctx(), topicId, cosmosMath.NewInt(stake))
		s.Require().NoError(err, "Setting topic stake should not fail")
	}

	setTopicWeight(topic1Id, 10, 10)
	err = k.ActivateTopic(s.Ctx(), topic1Id)
	s.Require().NoError(err, "Activating topic should not fail")

	setTopicWeight(topic2Id, 20, 10)
	err = k.ActivateTopic(s.Ctx(), topic2Id)
	s.Require().NoError(err, "Activating topic should not fail")

	// Fetch next page -- should only return topic 5
	activeTopics, err := k.GetActiveTopicIdsAtBlock(s.Ctx(), 15)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 2, "Should retrieve exactly two active topics")

	s.WithBlockHeight(15)
	err = k.AttemptTopicReactivation(s.Ctx(), topic1Id)
	s.Require().NoError(err, "Attempting to reactivate topic should not fail")
	err = k.AttemptTopicReactivation(s.Ctx(), topic2Id)
	s.Require().NoError(err, "Attempting to reactivate topic should not fail")

	s.WithBlockHeight(25)
	setTopicWeight(topic3Id, 50, 10)
	err = k.ActivateTopic(s.Ctx(), topic3Id)
	s.Require().NoError(err, "Activating topic should not fail")

	activeTopics, err = k.GetActiveTopicIdsAtBlock(s.Ctx(), 30)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Len(activeTopics.TopicIds, 3, "Should retrieve exactly two active topics")
	s.Require().Equal(topic1Id, activeTopics.TopicIds[0])
	s.Require().Equal(topic2Id, activeTopics.TopicIds[1])
	s.Require().Equal(topic3Id, activeTopics.TopicIds[2])

	s.WithBlockHeight(30)
	setTopicWeight(topic4Id, 1, 1)
	isActive, err := k.IsTopicActive(s.Ctx(), topic4Id)
	s.Require().NoError(err, "Is topic active should not produce an error")
	s.Require().False(isActive, "Topic4 should not be activated")
}

func (s *KeeperTestSuite) TestIncrementTopicId() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Initial check for the current topic ID
	initialTopicId, err := k.IncrementTopicId(ctx)
	s.Require().NoError(err, "Getting initial topic ID should not fail")

	// Increment the topic ID
	newTopicId, err := k.IncrementTopicId(ctx)
	s.Require().NoError(err, "Incrementing topic ID should not fail")
	s.Require().Equal(initialTopicId+1, newTopicId, "New topic ID should be one more than the initial topic ID")
}

func (s *KeeperTestSuite) TestGetNumTopicsWithActualTopicCreation() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	nextTopicIdStart, err := k.GetNextTopicId(ctx)
	s.Require().NoError(err, "Fetching the number of topics should not fail")

	// Create multiple topics to simulate actual usage
	topicsToCreate := 5
	for i := 1; i <= topicsToCreate; i++ {
		topicId, err := k.IncrementTopicId(ctx)
		s.Require().NoError(err, "Incrementing topic ID should not fail")

		newTopic := s.MockTopic()
		newTopic.Id = topicId

		err = k.SetTopic(ctx, topicId, newTopic)
		s.Require().NoError(err, "Setting a new topic should not fail")
	}

	// Now retrieve the total number of topics
	nextTopicIdEnd, err := k.GetNextTopicId(ctx)
	s.Require().NoError(err, "Fetching the number of topics should not fail")
	s.Require().Equal(uint64(topicsToCreate), nextTopicIdEnd-nextTopicIdStart)
}

func (s *KeeperTestSuite) TestUpdateAndGetTopicEpochLastEnded() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(2)
	epochLastEnded := types.BlockHeight(100)

	// Setup a topic initially
	initialTopic := s.MockTopic()
	err := k.SetTopic(ctx, topicId, initialTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Update the epoch last ended
	err = k.UpdateTopicEpochLastEnded(ctx, topicId, epochLastEnded)
	s.Require().NoError(err, "Updating topic epoch last ended should not fail")

	// Retrieve the last ended epoch for the topic
	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err, "Retrieving topic epoch last ended should not fail")
	s.Require().Equal(epochLastEnded, topic.EpochLastEnded, "The retrieved epoch last ended should match the updated value")
}

func (s *KeeperTestSuite) TestTopicExists() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Test a topic ID that does not exist
	nonExistentTopicId := uint64(999) // Assuming this ID has not been used
	exists, err := k.TopicExists(ctx, nonExistentTopicId)
	s.Require().NoError(err, "Checking existence for a non-existent topic should not fail")
	s.Require().False(exists, "No topic should exist for an unused topic ID")

	// Create a topic to test existence
	existentTopicId, err := k.IncrementTopicId(ctx)
	s.Require().NoError(err, "Incrementing topic ID should not fail")

	newTopic := s.MockTopic()
	err = k.SetTopic(ctx, existentTopicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test the newly created topic ID
	exists, err = k.TopicExists(ctx, existentTopicId)
	s.Require().NoError(err, "Checking existence for an existent topic should not fail")
	s.Require().True(exists, "Topic should exist for a newly created topic ID")
}

func (s *KeeperTestSuite) TestGetTopic() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topicId := uint64(2)
	_, err := k.GetTopic(ctx, topicId)
	s.Require().ErrorIs(err, types.ErrTopicDoesNotExist, "Retrieving a non-existent topic should result in an error")

	newTopic := s.MockTopic()

	err = k.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	retrievedTopic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err, "Retrieving an existent topic should not fail")
	s.Require().Equal(newTopic, retrievedTopic, "Retrieved topic should match the set topic")
	s.Require().Equal(newTopic.Metadata, retrievedTopic.Metadata, "Retrieved topic should match the set topic")
}

// FEE REVENUE

func (s *KeeperTestSuite) TestGetTopicFeeRevenue() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(2)

	newTopic := s.MockTopic()
	err := k.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test getting revenue for a topic with no existing revenue
	feeRev, err := k.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Should not error when revenue does not exist")
	s.Require().Equal(cosmosMath.ZeroInt(), feeRev, "Revenue should be zero for non-existing entries")

	// Setup a topic with some revenue
	initialRevenue := cosmosMath.NewInt(100)
	initialRevenueInt := cosmosMath.NewInt(100)
	err = k.AddTopicFeeRevenue(ctx, topicId, initialRevenue)
	s.Require().NoError(err, "Adding initial revenue should not fail")

	// Test getting revenue for a topic with existing revenue
	feeRev, err = k.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Should not error when retrieving existing revenue")
	s.Require().Equal(feeRev.String(), initialRevenueInt.String(), "Revenue should match the initial setup")
}

func (s *KeeperTestSuite) TestAddTopicFeeRevenue() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	block := int64(100)
	topicId := uint64(2)
	newTopic := s.MockTopic()
	newTopic.Id = topicId

	err := k.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	params := types.DefaultParams()

	blocksPerWeek, err := alloraMath.CalculateBlocksPerWeek(params.BlocksPerMonth)
	s.Require().NoError(err, "error calculating blocks per week")

	err = k.DripTopicFeeRevenue(ctx, newTopic, blocksPerWeek, block)
	s.Require().NoError(err, "Resetting topic fee revenue should not fail")

	// Add initial revenue
	initialAmount := cosmosMath.NewInt(100)
	err = k.AddTopicFeeRevenue(ctx, topicId, initialAmount)
	s.Require().NoError(err, "Adding initial revenue should not fail")

	// Verify initial revenue
	feeRev, err := k.GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err, "Getting topic fee revenue should not fail")
	s.Require().Equal(initialAmount, feeRev, "Initial revenue should be correctly recorded")
}

// SCORES

func (s *KeeperTestSuite) TestGetScoreEmas() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	forecaster := "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
	reputer := "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu"

	// Test getting latest scores when none are set
	infererScore, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching latest inferer score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     worker,
		Score:       alloraMath.ZeroDec(),
	}, infererScore, "Inferer score should be zero if not set")

	forecasterScore, err := k.GetForecasterScoreEma(ctx, topicId, forecaster)
	s.Require().NoError(err, "Fetching latest forecaster score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     forecaster,
		Score:       alloraMath.ZeroDec(),
	}, forecasterScore, "Forecaster score should be empty if not set")

	reputerScore, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching latest reputer score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     reputer,
		Score:       alloraMath.ZeroDec(),
	}, reputerScore, "Reputer score should be empty if not set")
}

func (s *KeeperTestSuite) TestSetScoreEmas() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	forecaster := "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
	reputer := "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu"
	score := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set an initial score for inferer and attempt to update with an older score
	err := k.SetInfererScoreEma(ctx, topicId, worker, score)
	s.Require().NoError(err)
	infererScore, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, infererScore.Score, "Newer inferer score should be set")

	// Set a new score for forecaster
	err = k.SetForecasterScoreEma(ctx, topicId, forecaster, score)
	s.Require().NoError(err)
	forecasterScore, err := k.GetForecasterScoreEma(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, forecasterScore.Score, "Newer forecaster score should be set")

	// Set a new score for reputer
	err = k.SetReputerScoreEma(ctx, topicId, reputer, score)
	s.Require().NoError(err)
	reputerScore, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, reputerScore.Score, "Newer reputer score should be set")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Score:       alloraMath.NewDecFromInt64(95),
	}

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	params.MaxTopInferersToReward = 1
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ {
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore2() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	params.MaxTopInferersToReward = 1
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		scoreValue := alloraMath.NewDecFromInt64(int64(90 + i)) // Increment score value to simulate variation
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
			Score:       scoreValue,
		}
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")

	// Check that the retained scores are the last five inserted
	for idx, score := range scores.Scores {
		expectedScoreValue := alloraMath.NewDecFromInt64(int64(92 + idx)) // Expecting the last 5 scores: 94, 95, 96, 97
		s.Require().Equal(expectedScoreValue, score.Score, "Score should match the expected last scores")
	}
}

func (s *KeeperTestSuite) TestGetInferenceScoresUntilBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	workerAddress := s.Addrs(0)
	blockHeight := int64(105)

	// Insert scores for different workers and blocks
	for blockHeight := int64(100); blockHeight <= 110; blockHeight++ {
		// Scores for the targeted worker
		scoreForWorker := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     workerAddress.String(),
			Score:       alloraMath.NewDecFromInt64(blockHeight),
		}
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, scoreForWorker)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Get scores for the worker up to block 105
	scores, err := k.GetInferenceScoresUntilBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching worker inference scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")

	// Verify that the scores are correct and ordered as expected (descending block number)
	expectedBlock := blockHeight
	for _, score := range scores {
		s.Require().Equal(workerAddress.String(), score.Address, "Only scores for the specified worker should be returned")
		s.Require().Equal(expectedBlock, score.BlockHeight, "Scores should be returned in descending order by block")
		s.Require().Equal(alloraMath.NewDecFromInt64(expectedBlock), score.Score, "Score value should match expected")
		expectedBlock--
	}
}

func (s *KeeperTestSuite) TestInsertWorkerForecastScore() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	params.MaxTopForecastersToReward = 1
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := k.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetForecastScoresUntilBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(105)

	// Insert scores for the worker at various blocks
	for i := int64(100); i <= 110; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: i,
			Score:       alloraMath.NewDecFromInt64(i),
			Address:     s.AddrsStr(0),
		}
		err := k.InsertWorkerForecastScore(ctx, topicId, i, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Get forecast scores for the worker up to block 105
	scores, err := k.GetForecastScoresUntilBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching worker forecast scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")
}

func (s *KeeperTestSuite) TestGetWorkerForecastScoresAtBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.AddrsStr(i),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		err := k.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores at the specific block
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestInsertReputerScore() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := k.InsertReputerScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetReputersScoresAtBlock() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert multiple scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.AddrsStr(i),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		err := k.InsertReputerScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores at the specific block
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestSetListeningCoefficient() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"

	// Define a listening coefficient
	coefficient := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(10),
	}

	// Set the listening coefficient
	err := k.SetListeningCoefficient(ctx, topicId, reputer, coefficient)
	s.Require().NoError(err, "Setting listening coefficient should not fail")

	// Retrieve the set coefficient to verify it was set correctly
	retrievedCoef, err := k.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching listening coefficient should not fail")
	s.Require().Equal(coefficient.Coefficient, retrievedCoef.Coefficient, "The retrieved coefficient should match the set value")
}

func (s *KeeperTestSuite) TestGetListeningCoefficient() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"

	// Attempt to fetch a coefficient before setting it
	defaultCoef, err := k.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail when not set")
	s.Require().Equal(alloraMath.NewDecFromInt64(1), defaultCoef.Coefficient, "Should return the default coefficient when not set")

	// Now set a specific coefficient
	setCoef := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(5),
	}
	err = k.SetListeningCoefficient(ctx, topicId, reputer, setCoef)
	s.Require().NoError(err, "Setting listening coefficient should not fail")
	// Fetch and verify the coefficient after setting
	fetchedCoef, err := k.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail after setting")
	s.Require().Equal(setCoef.Coefficient, fetchedCoef.Coefficient, "The fetched coefficient should match the set value")
}

// REWARD FRACTION

func (s *KeeperTestSuite) TestSetPreviousReputerRewardFraction() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputer := s.AddrsStr(2)

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(75) // Assuming 0.75 as a fraction example

	// Set the reward fraction
	err := k.SetPreviousReputerRewardFraction(ctx, topicId, reputer, rewardFraction)
	s.Require().NoError(err, "Setting previous reputer reward fraction should not fail")

	// Verify by fetching the same
	fetchedReward, noPrior, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching the set reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousReputerRewardFraction() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputer := s.AddrsStr(2)

	// Attempt to fetch a reward fraction before setting it
	defaultReward, _, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(50) // Assuming 0.50 as a fraction example
	err = k.SetPreviousReputerRewardFraction(ctx, topicId, reputer, setReward)
	s.Require().NoError(err, "Setting previous reputer reward fraction should not fail")

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetPreviousInferenceRewardFraction() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(1)

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(25)

	// Set the reward fraction
	err := k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, rewardFraction)
	s.Require().NoError(err, "Setting previous inference reward fraction should not fail")

	// Verify by fetching the same
	fetchedReward, noPrior, err := k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousInferenceRewardFraction() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(1)

	// Attempt to fetch a reward fraction before setting it
	defaultReward, noPrior, err := k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75)
	err = k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, setReward)
	s.Require().NoError(err, "Setting previous inference reward fraction should not fail")
	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetPreviousForecastRewardFraction() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(3)

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(50) // Assume setting the fraction to 0.50

	// Set the forecast reward fraction
	err := k.SetPreviousForecastRewardFraction(ctx, topicId, worker, rewardFraction)
	s.Require().NoError(err, "Setting previous forecast reward fraction should not fail")

	// Verify by fetching the set value
	fetchedReward, noPrior, err := k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set forecast reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousForecastRewardFraction() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(3)

	// Attempt to fetch the reward fraction before setting it, expecting default value
	defaultReward, noPrior, err := k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero forecast reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75) // Assume setting it to 0.75
	err = k.SetPreviousForecastRewardFraction(ctx, topicId, worker, setReward)
	s.Require().NoError(err, "Setting previous forecast reward fraction should not fail")

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetGetPreviousPercentageRewardToStakedReputers() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	previousPercentageReward := alloraMath.NewDecFromInt64(50)

	// Set the previous percentage reward to staked reputers
	err := k.SetPreviousPercentageRewardToStakedReputers(ctx, previousPercentageReward)
	s.Require().NoError(err, "Setting previous percentage reward to staked reputers should not fail")

	// Get the previous percentage reward to staked reputers
	fetchedPercentageReward, err := k.GetPreviousPercentageRewardToStakedReputers(ctx)
	s.Require().NoError(err, "Fetching previous percentage reward to staked reputers should not fail")
	s.Require().Equal(previousPercentageReward, fetchedPercentageReward, "The fetched percentage reward should match the set value")
}

// TOPIC REWARD NONCE

func (s *KeeperTestSuite) TestGetSetDeleteTopicRewardNonce() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	// Test Get on an unset topicId, should return 0
	nonce, err := k.GetTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Getting an unset topic reward nonce should not fail")
	s.Require().Equal(int64(0), nonce, "Nonce for an unset topicId should be 0")

	// Test Set
	expectedNonce := int64(12345)
	err = k.SetTopicRewardNonce(ctx, topicId, expectedNonce)
	s.Require().NoError(err, "Setting topic reward nonce should not fail")

	// Test Get after Set, should return the set value
	nonce, err = k.GetTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Getting set topic reward nonce should not fail")
	s.Require().Equal(expectedNonce, nonce, "Nonce should match the value set earlier")

	// Test Delete
	err = k.DeleteTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Deleting topic reward nonce should not fail")

	// Test Get after Delete, should return 0
	nonce, err = k.GetTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Getting deleted topic reward nonce should not fail")
	s.Require().Equal(int64(0), nonce, "Nonce should be 0 after deletion")
}

// UTILS

func (s *KeeperTestSuite) TestCalcAppropriatePaginationForUint64Cursor() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	defaultLimit := uint64(20)
	maxLimit := uint64(50)

	params := types.DefaultParams()
	params.DefaultPageLimit = defaultLimit
	params.MaxPageLimit = maxLimit
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting default and max limit parameters should not fail")

	paramsActual, err := k.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(maxLimit, paramsActual.MaxPageLimit, "Max limit should be set correctly")
	s.Require().Equal(defaultLimit, paramsActual.DefaultPageLimit, "Default limit should be set correctly")

	// Test 1: Pagination request is nil
	limit, cursor, err := k.CalcAppropriatePaginationForUint64Cursor(ctx, nil)
	s.Require().NoError(err, "Should handle nil pagination request without error")
	s.Require().Equal(defaultLimit, limit, "Limit should default to the default limit")
	s.Require().Equal(uint64(0), cursor, "Cursor should be 0 when key nil")

	// Test 2: Pagination Key is empty and Limit is zero
	pagination := &types.SimpleCursorPaginationRequest{Key: []byte{}, Limit: 0}
	limit, cursor, err = k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Should handle empty key and zero limit without error")
	s.Require().Equal(defaultLimit, limit, "Limit should default to the default limit")
	s.Require().Equal(uint64(0), cursor, "Cursor should be 0 when key is empty")

	// Test 3: Valid key and non-zero limit within bounds
	validKey := binary.BigEndian.AppendUint64(nil, uint64(12345)) // Convert 12345 to big-endian byte slice
	pagination = &types.SimpleCursorPaginationRequest{Key: validKey, Limit: 30}
	limit, cursor, err = k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Handling valid key and valid limit should not fail")
	s.Require().Equal(uint64(30), limit, "Limit should be as specified")
	s.Require().Equal(uint64(12345), cursor, "Cursor should decode correctly from key")

	// Test 4: Limit exceeds maximum limit
	pagination = &types.SimpleCursorPaginationRequest{Key: validKey, Limit: 60}
	limit, _, err = k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Handling limit exceeding maximum should not fail")
	s.Require().Equal(maxLimit, limit, "Limit should be capped at the maximum limit")
}

// STATE MANAGEMENT

func (s *KeeperTestSuite) TestPruneRecordsAfterRewards() {
	// Set infereces, forecasts, and reputations for a topic
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)}, // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     s.AddrsStr(0),
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
				Inferer:     s.AddrsStr(1),
			},
		},
	}
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := s.EmissionsKeeper().InsertActiveInferences(s.Ctx(), topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err, "Inserting inferences should not fail")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  s.AddrsStr(6),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.AddrsStr(0),
						Value:   alloraMath.MustNewDecFromString("1"),
					},
					{
						Inferer: s.AddrsStr(1),
						Value:   alloraMath.MustNewDecFromString("2"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  s.AddrsStr(7),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.AddrsStr(0),
						Value:   alloraMath.MustNewDecFromString("3"),
					},
					{
						Inferer: s.AddrsStr(1),
						Value:   alloraMath.MustNewDecFromString("4"),
					},
				},
			},
		},
	}
	err = s.EmissionsKeeper().InsertActiveForecasts(s.Ctx(), topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: block},
	}
	//nolint:exhaustruct
	networkLosses := types.ValueBundle{
		Reputer:             s.AddrsStr(4),
		CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
		InfererValues:       s.createDefaultInfererValues(),
		NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
	}

	reputerLossBundles := types.LossBundles{&networkLosses}
	err = s.EmissionsKeeper().InsertActiveReputerLosses(s.Ctx(), topicId, block, reputerLossBundles)
	s.Require().NoError(err, "InsertActiveReputerLosses should not return an error")

	err = s.EmissionsKeeper().InsertNetworkLossBundleAtBlock(s.Ctx(), topicId, block, networkLosses)
	s.Require().NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Check if the records are set
	_, err = s.EmissionsKeeper().GetInferencesAtBlock(s.Ctx(), topicId, block, false)
	s.Require().NoError(err, "Getting inferences should not fail")
	_, err = s.EmissionsKeeper().GetForecastsAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting forecasts should not fail")
	lossBundles, err := s.EmissionsKeeper().GetReputerLossBundlesAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting reputer loss bundles should not fail")
	s.Require().NotNil(lossBundles)
	_, err = s.EmissionsKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting network loss bundle should not fail")

	// Prune records in the subsequent block
	err = s.EmissionsKeeper().PruneRecordsAfterRewards(s.Ctx(), topicId, block+1)
	s.Require().NoError(err, "Pruning records after rewards should not fail")

	// Check if the records are pruned
	inferences, err := s.EmissionsKeeper().GetInferencesAtBlock(s.Ctx(), topicId, block, false)
	s.Require().NoError(err, "Getting inferences should not fail")
	s.Require().Empty(inferences.Inferences, "Must be pruned")
	forecasts, err := s.EmissionsKeeper().GetForecastsAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting forecasts should not fail")
	s.Require().Empty(forecasts.Forecasts, "Must be pruned")
	lossbundles, err := s.EmissionsKeeper().GetReputerLossBundlesAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting reputer loss bundles should not fail")
	s.Require().Empty(lossbundles, "Must be pruned")
	networkBundles, err := s.EmissionsKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting network loss bundle should not fail but be empty")
	s.Require().Equal(topicId, networkBundles.TopicId, "topic id returned")
	s.Require().Empty(networkBundles.InfererValues, "inferer values is empty")
	s.Require().Empty(networkBundles.ForecasterValues, "forecaster values is empty")
	s.Require().Empty(networkBundles.OneOutInfererValues, "one out inferer values is empty")
	s.Require().Empty(networkBundles.OneOutForecasterValues, "one out forecaster values is empty")
	s.Require().Empty(networkBundles.OneInForecasterValues, "one in forecaster values is empty")
	s.Require().Empty(networkBundles.OneOutInfererForecasterValues, "one out inferer forecaster values is empty")
	s.Require().Equal("0", networkBundles.CombinedValue.String(), "Must be pruned as evidenced by empty combined value")
	s.Require().Equal("0", networkBundles.NaiveValue.String(), "Must be pruned as evidenced by empty naive value")
}

func (s *KeeperTestSuite) TestPruneWorkerNoncesLogicNoNonces() {
	k := s.EmissionsKeeper()
	topicId1 := uint64(1)
	blockHeightThreshold := int64(10)
	err := k.DeleteUnfulfilledWorkerNonces(s.Ctx(), topicId1)
	s.Require().NoError(err, "Failed to delete unfulfilled worker nonces, topicId1")

	// Call pruneWorkerNonces
	err = s.EmissionsKeeper().PruneWorkerNonces(s.Ctx(), topicId1, blockHeightThreshold)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	// Check remaining nonces
	nonces, err := s.EmissionsKeeper().GetUnfulfilledWorkerNonces(s.Ctx(), topicId1)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestPruneWorkerNoncesLogicCorrectness() {
	tests := []struct {
		name                 string
		blockHeightThreshold int64
		nonces               []*types.Nonce
		expectedNonces       []*types.Nonce
	}{

		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: All Nonces Pruned",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 5}, {BlockHeight: 7}},
			expectedNonces:       []*types.Nonce{},
		},
		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: Some Nonces Pruned",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 5}, {BlockHeight: 15}},
			expectedNonces:       []*types.Nonce{{BlockHeight: 15}},
		},
		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: Nonces Pruned on the Edge",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 5}, {BlockHeight: 10}, {BlockHeight: 15}},
			expectedNonces:       []*types.Nonce{{BlockHeight: 10}, {BlockHeight: 15}},
		},
		{
			name:                 "TestPruneWorkerNoncesLogicCorrectness: No Nonces Pruned",
			blockHeightThreshold: 10,
			nonces:               []*types.Nonce{{BlockHeight: 15}, {BlockHeight: 20}},
			expectedNonces:       []*types.Nonce{{BlockHeight: 15}, {BlockHeight: 20}},
		},
	}
	k := s.EmissionsKeeper()
	topicId1 := uint64(1)
	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := k.DeleteUnfulfilledWorkerNonces(s.Ctx(), topicId1)
			s.Require().NoError(err, "Failed to delete unfulfilled worker nonces, topicId1")
			// Set multiple worker nonces
			for _, val := range tt.nonces {
				err := k.AddWorkerNonce(s.Ctx(), topicId1, val)
				s.Require().NoError(err, "Failed to add worker nonce, topicId1")
			}

			// Call pruneWorkerNonces
			err = s.EmissionsKeeper().PruneWorkerNonces(s.Ctx(), topicId1, tt.blockHeightThreshold)
			s.Require().NoError(err)

			// Check remaining nonces
			nonces, err := s.EmissionsKeeper().GetUnfulfilledWorkerNonces(s.Ctx(), topicId1)
			s.Require().NoError(err)
			// for loop nonces
			for _, nonce := range nonces.Nonces {
				s.Require().Contains(tt.expectedNonces, nonce)
			}
			for _, nonce := range tt.expectedNonces {
				s.Require().Contains(nonces.Nonces, nonce)
			}
		})
	}
}

func (s *KeeperTestSuite) TestPruneReputerNoncesLogicCorrectness() {
	tests := []struct {
		name                 string
		blockHeightThreshold int64
		nonces               []*types.ReputerRequestNonce
		expectedNonces       []*types.ReputerRequestNonce
	}{
		{
			name:                 "No nonces",
			blockHeightThreshold: 10,
			nonces:               []*types.ReputerRequestNonce{},
			expectedNonces:       []*types.ReputerRequestNonce{},
		},
		{
			name:                 "All nonces pruned",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 5}},
				{ReputerNonce: &types.Nonce{BlockHeight: 7}}},
			expectedNonces: []*types.ReputerRequestNonce{},
		},
		{
			name:                 "Some nonces pruned",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 5}},
				{ReputerNonce: &types.Nonce{BlockHeight: 15}},
			},
			expectedNonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 15}}},
		},
		{
			name:                 "Nonces pruned on the edge",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 5}},
				{ReputerNonce: &types.Nonce{BlockHeight: 10}},
				{ReputerNonce: &types.Nonce{BlockHeight: 15}}},
			expectedNonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 10}},
				{ReputerNonce: &types.Nonce{BlockHeight: 15}}},
		},
		{
			name:                 "No nonces pruned",
			blockHeightThreshold: 10,
			nonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 15}},
				{ReputerNonce: &types.Nonce{BlockHeight: 20}}},
			expectedNonces: []*types.ReputerRequestNonce{
				{ReputerNonce: &types.Nonce{BlockHeight: 15}},
				{ReputerNonce: &types.Nonce{BlockHeight: 20}}},
		},
	}
	k := s.EmissionsKeeper()
	topicId1 := uint64(1)
	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := k.DeleteUnfulfilledReputerNonces(s.Ctx(), topicId1)
			s.Require().NoError(err, "Failed to delete unfulfilled reputer nonces, topicId1")
			// Set multiple reputer nonces
			for _, val := range tt.nonces {
				err := k.AddReputerNonce(s.Ctx(), topicId1, val.ReputerNonce)
				s.Require().NoError(err, "Failed to add reputer nonce, topicId1")
			}

			// Call PruneReputerNonces
			err = s.EmissionsKeeper().PruneReputerNonces(s.Ctx(), topicId1, tt.blockHeightThreshold)
			s.Require().NoError(err)

			// Check remaining nonces
			nonces, err := s.EmissionsKeeper().GetUnfulfilledReputerNonces(s.Ctx(), topicId1)
			s.Require().NoError(err)
			// for loop nonces
			for _, nonce := range nonces.Nonces {
				s.Require().Contains(tt.expectedNonces, nonce)
			}
			for _, nonce := range tt.expectedNonces {
				s.Require().Contains(nonces.Nonces, nonce)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetTargetWeight() {
	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	if err != nil {
		s.T().Fatalf("Failed to get parameters: %v", err)
	}

	// Value for full calculation
	dec, err := alloraMath.NewDecFromString("70.71067811865475244008443621048489")
	s.Require().NoError(err)

	testCases := []struct {
		name            string
		topicStake      alloraMath.Dec
		topicFeeRevenue alloraMath.Dec
		stakeImportance alloraMath.Dec
		feeImportance   alloraMath.Dec
		want            alloraMath.Dec
		expectError     bool
	}{
		{
			name:            "Basic valid inputs",
			topicStake:      alloraMath.NewDecFromInt64(100),
			topicFeeRevenue: alloraMath.NewDecFromInt64(50),
			stakeImportance: params.TopicRewardStakeImportance,
			feeImportance:   params.TopicRewardFeeRevenueImportance,
			want:            dec,
			expectError:     false,
		},
		{
			name:            "Zero topic Fee revenue should return stake only",
			topicStake:      alloraMath.NewDecFromInt64(100),
			topicFeeRevenue: alloraMath.ZeroDec(),
			stakeImportance: params.TopicRewardStakeImportance,
			feeImportance:   params.TopicRewardFeeRevenueImportance,
			want:            alloraMath.ZeroDec(),
			expectError:     false,
		},
		{
			name:            "Zero topic stake should return fee only",
			topicStake:      alloraMath.ZeroDec(),
			topicFeeRevenue: alloraMath.NewDecFromInt64(50),
			stakeImportance: params.TopicRewardStakeImportance,
			feeImportance:   params.TopicRewardFeeRevenueImportance,
			want:            alloraMath.ZeroDec(),
			expectError:     false,
		},
		{
			name:            "Negative stake",
			topicStake:      alloraMath.NewDecFromInt64(-100),
			topicFeeRevenue: alloraMath.NewDecFromInt64(50),
			stakeImportance: params.TopicRewardStakeImportance,
			feeImportance:   params.TopicRewardFeeRevenueImportance,
			want:            alloraMath.Dec{},
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			got, err := s.EmissionsKeeper().GetTargetWeight(tc.topicStake, tc.topicFeeRevenue, tc.stakeImportance, tc.feeImportance)
			if tc.expectError {
				s.Require().Error(err, "Expected an error for case: %s", tc.name)
			} else {
				s.Require().NoError(err, "Did not expect an error for case: %s", tc.name)
				s.Require().True(tc.want.Equal(got), "Expected %s, got %s for case %s", tc.want.String(), got.String(), tc.name)
			}
		})
	}
}

func (s *KeeperTestSuite) TestDeleteUnfulfilledWorkerNonces() {
	topicId := uint64(1)
	k := s.EmissionsKeeper()
	// Setup initial nonces
	err := k.AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 10})
	s.Require().NoError(err)
	err = k.AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 20})
	s.Require().NoError(err)

	// Call DeleteUnfulfilledWorkerNonces
	err = s.EmissionsKeeper().DeleteUnfulfilledWorkerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Check that the nonces were removed
	nonces, err := s.EmissionsKeeper().GetUnfulfilledWorkerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestDeleteUnfulfilledreputerNonces() {
	topicId := uint64(1)
	k := s.EmissionsKeeper()
	// Setup initial nonces
	err := k.AddReputerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 50})
	s.Require().NoError(err)
	err = k.AddReputerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 60})
	s.Require().NoError(err)

	// Call DeleteUnfulfilledWorkerNonces
	err = s.EmissionsKeeper().DeleteUnfulfilledReputerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Check that the nonces were removed
	nonces, err := s.EmissionsKeeper().GetUnfulfilledReputerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

// TestActiveTopicStakeRemoval tests that when a stake is removed from an active topic,
// it correctly updates the stake
func (s *KeeperTestSuite) TestActiveTopicStakeRemoval() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create a topic and activate it
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)
	err = k.ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Verify the topic is active
	isActive, err := k.IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive, "Topic should be active")

	// Add some stake to the topic
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Verify the stake was added correctly
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, topicStake, "Topic stake should match the added amount")

	// Setup for stake removal
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check that the stake was removed correctly
	topicStake, err = k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")
}

// TestDelegateStakeRemoval tests that when a delegate stake is removed,
// it correctly updates the stake
func (s *KeeperTestSuite) TestDelegateStakeRemoval() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(1)
	delegatorAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create a topic and activate it
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)
	err = k.ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Verify the topic is active
	isActive, err := k.IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(isActive, "Topic should be active")

	// Add some delegate stake to the topic
	err = k.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Verify the stake was added correctly
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, topicStake, "Topic stake should match the added amount")

	// Setup for delegate stake removal
	err = k.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove delegate stake
	err = k.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check that the stake was removed correctly
	topicStake, err = k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")
}

// TestInactiveTopicStakeRemoval tests that when a stake is removed from an inactive topic,
// it still correctly updates the stake but does not affect the total weight calculations
func (s *KeeperTestSuite) TestInactiveTopicStakeRemoval() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create a topic but do NOT activate it
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)

	// Verify the topic is not active
	isActive, err := k.IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(isActive, "Topic should not be active")

	// Add some stake to the topic
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Verify the stake was added correctly
	topicStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, topicStake, "Topic stake should match the added amount")

	// Setup for stake removal
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)

	// Remove stake first (this should work even for inactive topics)
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	// Check that the stake was removed correctly
	topicStake, err = k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.ZeroInt(), topicStake, "Topic stake should be zero after removal")
}

// TestTopicWeightRecalculationAfterStakeRemoval tests that when a stake is removed,
// the topic weight is recalculated correctly based on the remaining stake
func (s *KeeperTestSuite) TestTopicWeightRecalculationAfterStakeRemoval() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(2)
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(1000)
	feeRevenue := cosmosMath.NewInt(100)
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow
	epochLength := int64(100)

	// Create and activate topic
	s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(epochLength),
	)
	err = k.ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Add stake and fee revenue
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)
	err = k.AddTopicFeeRevenue(ctx, topicId, feeRevenue)
	s.Require().NoError(err)

	// Get initial weight and set it
	initialWeight, _, _, err := k.GetCurrentTopicWeight(
		ctx,
		topicId,
		epochLength,
		moduleParams.TopicRewardAlpha,
		moduleParams.TopicRewardStakeImportance,
		moduleParams.TopicRewardFeeRevenueImportance,
		moduleParams.BlocksPerMonth,
	)
	s.Require().NoError(err)
	err = k.SetPreviousTopicWeight(ctx, topicId, initialWeight)
	s.Require().NoError(err)

	// Remove half the stake
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount.QuoRaw(2),
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)
	err = k.RemoveReputerStake(ctx, endBlock, topicId, reputerAddr, stakeAmount.QuoRaw(2))
	s.Require().NoError(err)

	// Verify weight and stake changes
	newWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(noPrior, "Should still have a prior weight")
	s.Require().True(newWeight.Lt(initialWeight), "New weight should be less than initial weight")

	remainingStake, err := k.GetTopicStake(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount.QuoRaw(2), remainingStake, "Remaining stake should be half of initial stake")
}

// TestTopicWeightRecalculationWithMultipleTopics tests that when stakes are removed,
// the weights of multiple topics are recalculated correctly and the total sum is updated
func (s *KeeperTestSuite) TestTopicWeightRecalculationWithMultipleTopics() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	reputerAddr := s.AddrsStr(0)

	// Set up test parameters
	stakeAmount1 := cosmosMath.NewInt(1000)
	stakeAmount2 := cosmosMath.NewInt(2000)
	epochLength := int64(100)
	feeRevenue1 := cosmosMath.NewInt(100)
	feeRevenue2 := cosmosMath.NewInt(200)

	// Set up params
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = uint64(5)
	params.MinTopicWeight = alloraMath.MustNewDecFromString("1")
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Create and activate both topics
	topicId1, topicId2 :=
		s.CreateTopic(testutil.WithEpochLength(epochLength)),
		s.CreateTopic(testutil.WithEpochLength(epochLength))
	err = k.ActivateTopic(ctx, topicId1)
	s.Require().NoError(err)
	err = k.ActivateTopic(ctx, topicId2)
	s.Require().NoError(err)

	// Add stake and fee revenue to both topics
	err = k.AddReputerStake(ctx, topicId1, reputerAddr, stakeAmount1)
	s.Require().NoError(err)
	err = k.AddReputerStake(ctx, topicId2, reputerAddr, stakeAmount2)
	s.Require().NoError(err)
	err = k.AddTopicFeeRevenue(ctx, topicId1, feeRevenue1)
	s.Require().NoError(err)
	err = k.AddTopicFeeRevenue(ctx, topicId2, feeRevenue2)
	s.Require().NoError(err)

	// Calculate and set initial weights for both topics
	for id := range map[uint64]struct{}{topicId1: {}, topicId2: {}} {
		weight, _, _, err := k.GetCurrentTopicWeight(
			ctx,
			id,
			epochLength, // epochLength
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			params.BlocksPerMonth,
		)
		s.Require().NoError(err)
		s.Require().True(weight.Gt(params.MinTopicWeight), "Initial weight should be greater than minimum weight")
		err = k.SetPreviousTopicWeight(ctx, id, weight)
		s.Require().NoError(err)
	}

	// Get total sum before stake removal
	totalSumBefore, err := k.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)

	// Remove stake from first topic
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + params.RemoveStakeDelayWindow
	err = k.SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               topicId1,
		Reputer:               reputerAddr,
		Amount:                stakeAmount1,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)
	err = k.RemoveReputerStake(ctx, endBlock, topicId1, reputerAddr, stakeAmount1)
	s.Require().NoError(err)

	// Verify total sum was updated correctly
	totalSumAfter, err := k.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(totalSumAfter.Lt(totalSumBefore), "Total sum should decrease after stake removal")
}

func (s *KeeperTestSuite) TestGetFirstStakeRemovalForReputerAndTopicId() {
	k := s.EmissionsKeeper()
	ctx := s.Ctx()
	reputer := s.AddrsStr(2)
	topicId := uint64(1)

	// Create a stake removal info
	stakeRemovalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(100),
		BlockRemovalCompleted: 30,
	}
	anotherStakeRemoval := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               "allo13c8tjxmlv32s6d76f9anh6cu6c767v0ddnn0uh",
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(200),
		BlockRemovalCompleted: 30,
	}

	// Set the stake removal info in the keeper
	err := k.SetStakeRemoval(ctx, stakeRemovalInfo)
	s.Require().NoError(err)
	err = k.SetStakeRemoval(ctx, anotherStakeRemoval)
	s.Require().NoError(err)

	// Get the first stake removal for the reputer and topic ID
	result, found, err := k.GetStakeRemovalForReputerAndTopicId(ctx, reputer, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(stakeRemovalInfo, result)
}

func (s *KeeperTestSuite) TestGetFirstStakeRemovalForReputerAndTopicIdNotFound() {
	k := s.EmissionsKeeper()
	ctx := s.Ctx()
	reputer := s.AddrsStr(2)
	topicId := uint64(1)

	_, found, err := k.GetStakeRemovalForReputerAndTopicId(ctx, reputer, topicId)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestGetFirstDelegateStakeRemovalForDelegatorReputerAndTopicId() {
	k := s.EmissionsKeeper()
	ctx := s.Ctx()
	delegator := s.AddrsStr(5)
	reputer := s.AddrsStr(2)
	reputer2 := s.AddrsStr(3)
	topicId := uint64(1)

	// Create a stake removal info
	stakeRemovalInfo := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer,
		Delegator:             delegator,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(100),
		BlockRemovalCompleted: 30,
	}
	anotherStakeRemoval := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer2,
		Delegator:             delegator,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(200),
		BlockRemovalCompleted: 30,
	}

	// Set the stake removal info in the keeper
	err := k.SetDelegateStakeRemoval(ctx, stakeRemovalInfo)
	s.Require().NoError(err)
	err = k.SetDelegateStakeRemoval(ctx, anotherStakeRemoval)
	s.Require().NoError(err)

	// Get the first stake removal for the reputer and topic ID
	result, found, err := k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(stakeRemovalInfo, result)
}

func (s *KeeperTestSuite) TestGetFirstDelegateStakeRemovalForDelegatorReputerAndTopicIdNotFound() {
	k := s.EmissionsKeeper()
	ctx := s.Ctx()
	delegator := "delegator"
	reputer := s.AddrsStr(2)
	topicId := uint64(1)

	_, found, err := k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestAppendInference() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	// Topic IDs
	topicId := s.CreateTopic()
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Set previous topic quantile inferer score ema
	err := k.SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("1000"))
	s.Require().NoError(err)

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	worker1 := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	worker2 := "allo1dwxj49n0t5969uj4zfuemxg8a2ty85njn9xy9t"
	worker3 := "allo1wha0sj6pldjwjasc0pm28htgpqa5uf69kafe5y"
	worker4 := "allo19n8h9zwsqawpfn9qk9v773zj6rcjmqt28a2gyd"
	worker5 := "allo18d5wvwlhc08kfu27l25q9zr2shhanlh9fvt225"
	ogWorker2Score := alloraMath.MustNewDecFromString("90")

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: ogWorker2Score}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	score4 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker4, Score: alloraMath.NewDecFromInt64(91)}
	score5 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker5, Score: alloraMath.NewDecFromInt64(96)}
	err = k.SetInfererScoreEma(ctx, topicId, worker1, score1)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker2, score2)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker3, score3)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker4, score4)
	s.Require().NoError(err)
	err = k.SetInfererScoreEma(ctx, topicId, worker5, score5)
	s.Require().NoError(err)

	// Ensure that the number of top inferers is capped at the max top inferers to reward
	// New high-score entrant should replace earlier low-score entrant
	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err = k.SetParams(ctx, params)
	s.Require().NoError(err)

	allInferences := types.Inferences{
		Inferences: []*types.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")}},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.71")}},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker3, Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.71")}},
		},
	}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker4,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newInference2 := types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker5,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		ExtraData:   nil,
		Proof:       "",
	}

	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().NoError(err)
	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	// New high-score entrant should replace earlier low-score entrant
	worker5OgScore, err := k.GetInfererScoreEma(ctx, topicId, worker5)
	s.Require().NoError(err)
	worker5Found := false
	for _, address := range activeInferers {
		if address == worker5 {
			worker5Found = true
		}
	}
	s.Require().True(worker5Found)

	// Ensure EMA score of active set is not yet updated
	// This will happen later during epoch reward calculation, not here
	worker5NewScore, err := k.GetInfererScoreEma(ctx, topicId, worker5)
	s.Require().NoError(err)
	// EMA score should be updated higher because saved topic quantile ema is higher
	s.Require().True(worker5OgScore.Score.Equal(worker5NewScore.Score))
	// EMA score should be updated with the new time of update given that it was updated then
	s.Require().Equal(worker5OgScore.BlockHeight, worker5NewScore.BlockHeight)

	// Ensure EMA score of actor moved to passive set is updated
	updatedWorker2Score, err := k.GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	// EMA score should be updated higher because saved topic quantile ema is higher
	updatedWorker2ScoreVal, err := updatedWorker2Score.Score.Int64()
	s.Require().NoError(err)
	ogWorker2ScoreVal, err := ogWorker2Score.Int64()
	s.Require().NoError(err)
	worker5OgScoreVal, err := worker5OgScore.Score.Int64()
	s.Require().NoError(err)
	s.Require().Greater(updatedWorker2ScoreVal, ogWorker2ScoreVal, "worker2 score should go up given large ema value")
	s.Require().Greater(updatedWorker2ScoreVal, worker5OgScoreVal, "worker2 could not overtake worker5, but not in this epoch")
	// EMA score should be updated with the new time of update given that it was updated then
	s.Require().Equal(nonce.BlockHeight, updatedWorker2Score.BlockHeight)

	// Ensure passive set participant can't update their score within the same epoch
	blockHeightInferences = blockHeightInferences + 1 // within the same epoch => no update
	newInference2 = types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		Inferer:     worker2,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		ExtraData:   nil,
		Proof:       "",
	}
	err = k.AppendInference(ctx, topic, nonce.BlockHeight, &newInference2, params.MaxTopInferersToReward)
	s.Require().Error(err, types.ErrCantUpdateEmaMoreThanOncePerWindow.Error())
	// Confirm no change in EMA score
	updateAttemptForWorker2, err := k.GetInfererScoreEma(ctx, topicId, worker2)
	s.Require().NoError(err)
	updateAttemptForWorker2Val, err := updateAttemptForWorker2.Score.Int64()
	s.Require().NoError(err)
	s.Require().Equal(updateAttemptForWorker2Val, updatedWorker2ScoreVal, "unchanged score")
	s.Require().Equal(updateAttemptForWorker2.BlockHeight, updatedWorker2Score.BlockHeight, "unchanged height")
}

func getNewAddress() string {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	return addr.String()
}

func (s *KeeperTestSuite) TestAppendInferenceWithResetActiveWorkers() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic(testutil.WithEpochLength(10801), testutil.WithGroundTruthLag(10801))
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Set previous topic quantile inferer score ema
	err := k.SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, alloraMath.MustNewDecFromString("1000"))
	s.Require().NoError(err)

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Create workers
	workers := make([]string, 6)
	for i := range workers {
		workers[i] = getNewAddress()
	}

	// Set scores for workers 2-6
	for i := 1; i < len(workers); i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     workers[i],
			Score:       alloraMath.NewDecFromInt64(int64(91 + i)),
		}
		err = k.SetInfererScoreEma(ctx, topicId, workers[i], score)
		s.Require().NoError(err)
	}

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err = k.SetParams(ctx, params)
	s.Require().NoError(err)

	formatValue := func(val float64) string {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", val), "0"), ".")
	}

	// Create first set of inferences
	inferences := make([]*types.Inference, 5)
	for i := range inferences {
		//nolint:exhaustruct
		inferences[i] = &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeightInferences,
			Inferer:     workers[i],
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(formatValue(0.11 + float64(i)*0.01))},
		}
	}

	allInferences := types.Inferences{Inferences: inferences}
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	lowestEmaScore, found, err := k.GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestEmaScore.Address, workers[1])

	err = k.ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)

	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeInferers)

	lowestEmaScore, found, err = k.GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestEmaScore.Address, workers[1])

	// Create second set of inferences
	blockHeightInferences = blockHeightInferences + topic.EpochLength
	inferences = make([]*types.Inference, 5)
	for i := 1; i <= len(inferences); i++ {
		//nolint:exhaustruct
		inferences[i-1] = &types.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeightInferences,
			Inferer:     workers[i],
			Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString(formatValue(0.22 + float64(i-1)*0.01))},
		}
	}

	allInferences = types.Inferences{Inferences: inferences}
	nonce.BlockHeight++
	for _, inference := range allInferences.Inferences {
		err = k.AppendInference(ctx, topic, nonce.BlockHeight, inference, params.MaxTopInferersToReward)
		s.Require().NoError(err)
	}

	activeInferers, err = k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, uint64(len(activeInferers)))

	lowestEmaScore, found, err = k.GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestEmaScore.Address, workers[2])
}

func mockUninitializedParams() types.Params {
	return types.Params{
		Version:                             "v2",
		MinTopicWeight:                      alloraMath.MustNewDecFromString("0"),
		RequiredMinimumStake:                cosmosMath.NewInt(0),
		RemoveStakeDelayWindow:              0,
		MinEpochLength:                      1,
		BetaEntropy:                         alloraMath.MustNewDecFromString("0"),
		LearningRate:                        alloraMath.MustNewDecFromString("0.0001"),
		GradientDescentMaxIters:             uint64(100),
		MaxGradientThreshold:                alloraMath.MustNewDecFromString("0.0001"),
		MinStakeFraction:                    alloraMath.MustNewDecFromString("0"),
		EpsilonReputer:                      alloraMath.MustNewDecFromString("0.0001"),
		EpsilonSafeDiv:                      alloraMath.MustNewDecFromString("0.0001"),
		MaxUnfulfilledWorkerRequests:        uint64(0),
		MaxUnfulfilledReputerRequests:       uint64(0),
		TopicRewardStakeImportance:          alloraMath.MustNewDecFromString("0"),
		TopicRewardFeeRevenueImportance:     alloraMath.MustNewDecFromString("0"),
		TopicRewardAlpha:                    alloraMath.MustNewDecFromString("0.1"),
		TaskRewardAlpha:                     alloraMath.MustNewDecFromString("0.1"),
		ValidatorsVsAlloraPercentReward:     alloraMath.MustNewDecFromString("0"),
		MaxSamplesToScaleScores:             uint64(10),
		MaxTopInferersToReward:              uint64(0),
		MaxTopForecastersToReward:           uint64(0),
		MaxTopReputersToReward:              uint64(0),
		CreateTopicFee:                      cosmosMath.NewInt(0),
		RegistrationFee:                     cosmosMath.NewInt(0),
		DefaultPageLimit:                    uint64(0),
		MaxPageLimit:                        uint64(0),
		MinEpochLengthRecordLimit:           int64(0),
		MaxSerializedMsgLength:              int64(0),
		BlocksPerMonth:                      uint64(864000),
		PRewardInference:                    alloraMath.NewDecFromInt64(1),
		PRewardForecast:                     alloraMath.NewDecFromInt64(1),
		PRewardReputer:                      alloraMath.NewDecFromInt64(1),
		CRewardInference:                    alloraMath.MustNewDecFromString("0.1"),
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.1"),
		CNorm:                               alloraMath.ZeroDec(), // deprecated global c_norm kept for compat
		HalfMaxProcessStakeRemovalsEndBlock: uint64(1),
		DataSendingFee:                      cosmosMath.NewInt(0),
		MaxElementsPerForecast:              uint64(0),
		MaxActiveTopicsPerBlock:             uint64(0),
		MaxStringLength:                     uint64(0),
		InitialRegretQuantile:               alloraMath.ZeroDec(),
		PNormSafeDiv:                        alloraMath.ZeroDec(),
		GlobalWhitelistEnabled:              true,
		TopicCreatorWhitelistEnabled:        true,
		MinExperiencedWorkerRegrets:         uint64(10),
		InferenceOutlierDetectionThreshold:  alloraMath.MustNewDecFromString("11"),
		InferenceOutlierDetectionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		LambdaInitialScore:                  alloraMath.MustNewDecFromString("2"),
		GlobalWorkerWhitelistEnabled:        true,
		GlobalReputerWhitelistEnabled:       true,
		GlobalAdminWhitelistAppended:        true,
		MaxWhitelistInputArrayLength:        uint64(10),
		MinWeightThresholdForStdnorm:        alloraMath.MustNewDecFromString("0.000001"),
	}
}

func (s *KeeperTestSuite) TestAppendForecast() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	workers := make([]string, 5)
	for i := range workers {
		workers[i] = s.AddrsStr(i)
	}

	// Set initial scores
	scores := []int64{95, 90, 99, 91, 96}
	for i, worker := range workers {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     worker,
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := k.SetForecasterScoreEma(ctx, topicId, worker, score)
		s.Require().NoError(err)
	}

	params := mockUninitializedParams()
	params.MaxTopForecastersToReward = 4
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	//nolint:exhaustruct
	newForecast := types.Forecast{
		TopicId:     topicId,
		BlockHeight: blockHeightInferences,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: workers[0],
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
			{
				Inferer: workers[1],
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
	}

	// Create forecasts for first 4 workers
	forecasts := make([]*types.Forecast, 4)
	for i := range forecasts {
		forecast := newForecast
		forecast.Forecaster = workers[i]
		forecasts[i] = &forecast
	}

	allForecasts := types.Forecasts{Forecasts: forecasts}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append initial forecasts
	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))

	// Try to add forecast for the fifth worker
	blockHeightInferences = blockHeightInferences + topic.EpochLength
	newForecast.Forecaster = workers[4]
	newForecast.BlockHeight = blockHeightInferences

	// MaxTopForecastersToReward is 4, so this should not increase active forecasters
	err = k.AppendForecast(ctx, topic, nonce.BlockHeight, &newForecast, params.MaxTopForecastersToReward)
	s.Require().NoError(err)

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))
}

func (s *KeeperTestSuite) TestAppendForecastWithResetActiveForecasters() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	nonce := types.Nonce{BlockHeight: 10}
	blockHeightInferences := int64(10)

	// Create workers and set their scores
	workers := make([]string, 5)
	scores := []int64{95, 90, 99, 91, 96}
	for i := range workers {
		workers[i] = s.AddrsStr(i)
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     workers[i],
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := k.SetForecasterScoreEma(ctx, topicId, workers[i], score)
		s.Require().NoError(err)
	}

	// Set params
	params := mockUninitializedParams()
	params.MaxTopForecastersToReward = 4
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Create forecast elements template
	forecastElements := []*types.ForecastElement{
		{
			Inferer: workers[0],
			Value:   alloraMath.MustNewDecFromString("0.52"),
		},
		{
			Inferer: workers[1],
			Value:   alloraMath.MustNewDecFromString("0.52"),
		},
	}

	// Create forecasts for all workers
	allForecasts := types.Forecasts{Forecasts: make([]*types.Forecast, len(workers))}
	for i, worker := range workers {
		//nolint:exhaustruct
		allForecasts.Forecasts[i] = &types.Forecast{
			TopicId:          topicId,
			BlockHeight:      blockHeightInferences,
			Forecaster:       worker,
			ForecastElements: forecastElements,
		}
	}

	// Append initial forecasts and verify
	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))

	// Reset active forecasters and verify
	err = k.ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeForecasters)

	// Re-append forecasts and verify again
	topic, err = k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	nonce.BlockHeight++
	for _, forecast := range allForecasts.Forecasts {
		err = k.AppendForecast(ctx, topic, nonce.BlockHeight, forecast, params.MaxTopForecastersToReward)
		s.Require().NoError(err)
	}

	activeForecasters, err = k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopForecastersToReward, uint64(len(activeForecasters)))
}

func (s *KeeperTestSuite) TestAppendReputerLoss() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	blockHeight := int64(10)
	nonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Create reputers and set their scores
	reputers := make([]string, 5)
	scores := []int64{95, 90, 99, 91, 96}
	for i := range reputers {
		reputers[i] = s.AddrsStr(i)
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     reputers[i],
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := k.SetReputerScoreEma(ctx, topicId, reputers[i], score)
		s.Require().NoError(err)
	}

	// Set params
	params := types.DefaultParams()
	params.MaxTopReputersToReward = 4
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Create value bundles for all reputers
	reputerValueBundles := make(types.LossBundles, len(reputers))
	for i, reputer := range reputers {
		//nolint:exhaustruct
		valueBundle := &types.ValueBundle{
			Reputer:             reputer,
			CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
			ReputerRequestNonce: reputerRequestNonce,
			TopicId:             topicId,
			InfererValues:       s.createDefaultInfererValues(),
			NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
		}
		reputerValueBundles[i] = valueBundle
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append first three reputer losses
	for i := 0; i < 3; i++ {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundles[i])
		s.Require().NoError(err)
	}

	// Add fourth reputer and verify active reputers count
	err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundles[3])
	s.Require().NoError(err)
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))

	// Add fifth reputer and verify active reputers count remains at max
	err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundles[4])
	s.Require().NoError(err)
	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))
}

func (s *KeeperTestSuite) TestAppendReputerLossWithResetActiveReputers() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	blockHeight := int64(10)
	nonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Setup reputers and their scores
	reputers := make([]string, 5)
	scores := []int64{95, 90, 99, 91, 96}
	for i := range reputers {
		reputers[i] = s.AddrsStr(i)
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: 2,
			Address:     reputers[i],
			Score:       alloraMath.NewDecFromInt64(scores[i]),
		}
		err := k.SetReputerScoreEma(ctx, topicId, reputers[i], score)
		s.Require().NoError(err)
	}

	// Set params
	params := types.DefaultParams()
	params.MaxTopReputersToReward = 4
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Create reputer value bundles
	allReputerLosses := make(types.LossBundles, len(reputers))
	for i, reputer := range reputers {
		//nolint:exhaustruct
		valueBundle := types.ValueBundle{
			Reputer:             reputer,
			CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
			ReputerRequestNonce: reputerRequestNonce,
			TopicId:             topicId,
			InfererValues:       s.createDefaultInfererValues(),
			NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
		}
		allReputerLosses[i] = &valueBundle
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// First round: append all reputer losses
	for _, reputerValueBundle := range allReputerLosses {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundle)
		s.Require().NoError(err)
	}

	// Verify active reputers count
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))

	// Reset active reputers
	err = k.ResetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)

	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeReputers)

	// Second round: append all reputer losses again
	nonce.BlockHeight++
	for _, reputerValueBundle := range allReputerLosses {
		err = k.AppendReputerLoss(ctx, topic, params, nonce.BlockHeight, reputerValueBundle)
		s.Require().NoError(err)
	}

	// Verify active reputers count again
	activeReputers, err = k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopReputersToReward, uint64(len(activeReputers)))
}

func (s *KeeperTestSuite) TestDripTopicFeeRevenue() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	require := s.Require()

	// Define test data
	block := int64(100)
	// Calculated expected drip with these values: 26
	expectedDrip := cosmosMath.NewInt(26)
	initialRevenue := cosmosMath.NewInt(1000000) // 0.001 in Int representation (assuming 6 decimal places)

	params := types.DefaultParams()
	params.MinEpochLength = 1
	err := k.SetParams(ctx, params)
	require.NoError(err, "Setting a new topic should not fail")

	// Create and activate a topic
	topic := s.MockTopic()
	topic.Id = 2
	topic.EpochLength = 5
	topic.WorkerSubmissionWindow = 5
	topic.GroundTruthLag = 5
	err = k.SetTopic(ctx, topic.Id, topic)
	require.NoError(err, "Setting a new topic should not fail")

	err = k.ActivateTopic(ctx, topic.Id)
	require.NoError(err, "Activating the topic should not fail")

	// Set up initial topic fee revenue
	err = k.AddTopicFeeRevenue(ctx, topic.Id, initialRevenue)
	require.NoError(err, "Setting initial topic fee revenue should not fail")

	// Calculate the blocks per week
	blocksPerWeek, err := alloraMath.CalculateBlocksPerWeek(params.BlocksPerMonth)
	require.NoError(err, "error calculating blocks per week")

	// Call the function under test
	err = k.DripTopicFeeRevenue(ctx, topic, blocksPerWeek, block)
	require.NoError(err, "DripTopicFeeRevenue should not return an error")

	// Retrieve the updated topic fee revenue
	updatedTopicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topic.Id)
	require.NoError(err, "Getting topic fee revenue should not fail")

	// Assert the expected results
	require.True(updatedTopicFeeRevenue.LT(initialRevenue),
		"The topic fee revenue should have decreased after dripping")

	// Calculate expected revenue (this may need adjustment based on your actual implementation)
	expectedRevenue := initialRevenue.Sub(expectedDrip)
	require.Equal(expectedRevenue.String(), updatedTopicFeeRevenue.String(),
		"The topic fee revenue should match the expected value after dripping")
}

func (s *KeeperTestSuite) TestDripTopicFeeRevenueWithTwoTopicsDifferentEpochLengths() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	require := s.Require()

	// Define test data
	topicId1 := uint64(2)
	topicId2 := uint64(3)
	block := int64(100)
	initialRevenue := cosmosMath.NewInt(1000000) // 0.001 in Int representation (assuming 6 decimal places)

	params := types.DefaultParams()
	params.MinEpochLength = 1
	err := k.SetParams(ctx, params)
	require.NoError(err, "Setting a new topic should not fail")

	// Create topics with different epoch lengths
	topics := []struct {
		id          uint64
		epochLength int64
		expected    cosmosMath.Int
	}{
		{topicId1, 5, cosmosMath.NewInt(26)},  // shorter epoch length
		{topicId2, 10, cosmosMath.NewInt(51)}, // longer epoch length
	}

	for _, t := range topics {
		topic := s.MockTopic()
		topic.Id = t.id
		topic.EpochLength = t.epochLength
		topic.WorkerSubmissionWindow = t.epochLength
		topic.GroundTruthLag = t.epochLength

		err = k.SetTopic(ctx, t.id, topic)
		require.NoError(err, "Setting a new topic should not fail")

		err = k.ActivateTopic(ctx, topic.Id)
		require.NoError(err, "Activating the topic should not fail")

		err = k.AddTopicFeeRevenue(ctx, topic.Id, initialRevenue)
		require.NoError(err, "Setting initial topic fee revenue should not fail")
	}

	// Calculate the blocks per week
	blocksPerWeek, err := alloraMath.CalculateBlocksPerWeek(params.BlocksPerMonth)
	require.NoError(err, "error calculating blocks per week")

	// Process drips and verify results for each topic
	for _, t := range topics {
		topic, err := k.GetTopic(ctx, t.id)
		require.NoError(err)

		err = k.DripTopicFeeRevenue(ctx, topic, blocksPerWeek, block)
		require.NoError(err, "DripTopicFeeRevenue should not return an error")

		updatedRevenue, err := k.GetTopicFeeRevenue(ctx, t.id)
		require.NoError(err, "Getting topic fee revenue should not fail")

		require.True(updatedRevenue.LT(initialRevenue),
			"The topic fee revenue should have decreased after dripping")

		expectedRevenue := initialRevenue.Sub(t.expected)
		require.Equal(expectedRevenue.String(), updatedRevenue.String(),
			"The topic fee revenue should match the expected value after dripping")
	}
}

func (s *KeeperTestSuite) TestActiveInfererFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(0)

	err := k.AddActiveInferer(ctx, topicId, inferer)
	s.Require().NoError(err)

	isActive, err := k.IsActiveInferer(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeInferers, 1)
	s.Require().Equal(inferer, activeInferers[0])
}

func (s *KeeperTestSuite) TestActiveForecasterFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(1)

	err := k.AddActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)

	isActive, err := k.IsActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeForecasters, 1)
	s.Require().Equal(forecaster, activeForecasters[0])

	err = k.RemoveActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)

	isActive, err = k.IsActiveForecaster(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

func (s *KeeperTestSuite) TestLowestScoreEmaFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	address := s.AddrsStr(2)

	lowestInfererScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 100,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(50),
	}
	err := k.SetLowestInfererScoreEma(ctx, topicId, lowestInfererScore)
	s.Require().NoError(err)

	retrievedScore, found, err := k.GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestInfererScore, retrievedScore)

	lowestForecasterScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 200,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(75),
	}
	err = k.SetLowestForecasterScoreEma(ctx, topicId, lowestForecasterScore)
	s.Require().NoError(err)

	retrievedScore, found, err = k.GetLowestForecasterScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestForecasterScore, retrievedScore)
}

func (s *KeeperTestSuite) TestActiveReputerFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	reputer := s.AddrsStr(3)

	err := k.AddActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)

	isActive, err := k.IsActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().True(isActive)

	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(activeReputers, 1)
	s.Require().Equal(reputer, activeReputers[0])

	err = k.RemoveActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)

	isActive, err = k.IsActiveReputer(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().False(isActive)
}

func (s *KeeperTestSuite) TestResetFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)

	err := k.AddActiveReputer(ctx, topicId, s.AddrsStr(0))
	s.Require().NoError(err)
	err = k.AddActiveInferer(ctx, topicId, s.AddrsStr(1))
	s.Require().NoError(err)
	err = k.AddActiveForecaster(ctx, topicId, s.AddrsStr(2))
	s.Require().NoError(err)

	err = k.ResetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	activeReputers, err := k.GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeReputers)

	err = k.ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)
	activeInferers, err := k.GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeInferers)
	activeForecasters, err := k.GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeForecasters)

	err = k.ResetReputersIndividualSubmissionsForTopic(ctx, topicId)
	s.Require().NoError(err)

	err = k.ResetWorkersIndividualSubmissionsForTopic(ctx, topicId)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestLowestReputerScoreEmaFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	address := s.AddrsStr(4)

	lowestReputerScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 300,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(60),
	}
	err := k.SetLowestReputerScoreEma(ctx, topicId, lowestReputerScore)
	s.Require().NoError(err)

	retrievedScore, found, err := k.GetLowestReputerScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestReputerScore, retrievedScore)
}

func (s *KeeperTestSuite) TestRemoveReputerLoss() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	reputer := s.AddrsStr(0)

	// Create a reputer loss bundle
	//nolint:exhaustruct
	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: 100},
		},
		Reputer:       reputer,
		ExtraData:     []byte("data"),
		CombinedValue: alloraMath.MustNewDecFromString("123"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("123"),
	}

	// Insert the reputer loss bundle
	err := k.InsertReputerLoss(ctx, topicId, *valueBundle)
	s.Require().NoError(err)

	// Verify the reputer loss was added
	retrievedLoss, err := k.GetReputerLatestLossByTopicId(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(*valueBundle, retrievedLoss)

	// Remove the reputer loss
	err = k.RemoveReputerLoss(ctx, topicId, reputer)
	s.Require().NoError(err)

	// Verify the reputer loss was removed
	_, err = k.GetReputerLatestLossByTopicId(ctx, topicId, reputer)
	s.Require().Error(err) // Expect an error since the loss should be removed
}

func (s *KeeperTestSuite) TestRemoveForecast() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	forecaster := s.AddrsStr(0)

	// Create a forecast
	forecast := types.Forecast{
		TopicId:     topicId,
		BlockHeight: 100,
		Forecaster:  forecaster,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   alloraMath.MustNewDecFromString("1"),
			},
			{
				Inferer: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
				Value:   alloraMath.MustNewDecFromString("2"),
			},
		},
		ExtraData: []byte("data"),
	}

	// Insert the forecast
	err := k.InsertForecast(ctx, topicId, forecast)
	s.Require().NoError(err)

	// Verify the forecast was added
	retrievedForecast, err := k.GetWorkerLatestForecastByTopicId(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(forecast, retrievedForecast)

	// Remove the forecast
	err = k.RemoveForecast(ctx, topicId, forecaster)
	s.Require().NoError(err)

	// Verify the forecast was removed
	_, err = k.GetWorkerLatestForecastByTopicId(ctx, topicId, forecaster)
	s.Require().Error(err) // Expect an error since the forecast should be removed
}

func (s *KeeperTestSuite) TestRemoveInference() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	inferer := s.AddrsStr(0)

	// Create an inference
	inference := types.Inference{
		TopicId:     topicId,
		BlockHeight: 100,
		Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
		Inferer:     inferer,
		ExtraData:   []byte("data"),
		Proof:       "",
	}

	// Insert the inference
	err := k.InsertInference(ctx, topicId, inference)
	s.Require().NoError(err)

	// Verify the inference was added
	retrievedInference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(inference, retrievedInference)

	// Remove the inference
	err = k.RemoveInference(ctx, topicId, inferer)
	s.Require().NoError(err)

	// Verify the inference was removed
	_, err = k.GetWorkerLatestInferenceByTopicId(ctx, topicId, inferer)
	s.Require().Error(err) // Expect an error since the inference should be removed
}

func (s *KeeperTestSuite) TestGetCountInfererInclusionsInTopic() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	inferer1 := s.AddrsStr(0)
	inferer2 := s.AddrsStr(1)
	err := k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	err = k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	err = k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer2)
	require.NoError(err)

	// Assert the expected results
	count, err := k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	require.Equal(uint64(2), count)
	count, err = k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer2)
	require.NoError(err)
	require.Equal(uint64(1), count)
}

func (s *KeeperTestSuite) TestGetCountForecasterInclusionsInTopic() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	forecaster1 := s.AddrsStr(0)
	forecaster2 := s.AddrsStr(1)
	err := k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	err = k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	err = k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster2)
	require.NoError(err)

	// Assert the expected results
	count, err := k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	require.Equal(uint64(2), count)
	count, err = k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster2)
	require.NoError(err)
	require.Equal(uint64(1), count)
}

func (s *KeeperTestSuite) TestScoreLimiting() {
	k := s.EmissionsKeeper()
	ctx := s.Ctx()
	topicId := s.CreateTopic()
	blockHeight := int64(10)

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 2
	params.MaxSamplesToScaleScores = 3
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	for i := 0; i < 8; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.AddrsStr(i),
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)),
		}
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err)
	}

	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().Len(scores.Scores, 6, "Should keep MaxSamplesToScaleScores * MaxTopInferersToReward scores")

	for i, score := range scores.Scores {
		expectedWorker := s.AddrsStr(i + 2)
		s.Require().Equal(expectedWorker, score.Address)
	}
}

func (s *KeeperTestSuite) TestUpdateNetworkInferencesOutlierMetrics() {
	// Create one topic
	topicId := s.CreateTopic()
	blockHeight := int64(1)

	// Create specific inferences for testing
	inferences := []*types.Inference{
		{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(10)},
			Inferer:     s.AddrsStr(0),
		},
		{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(11)},
			Inferer:     s.AddrsStr(1),
		},
		{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(12)},
			Inferer:     s.AddrsStr(2),
		},
	}

	inferencesWrapper := types.Inferences{Inferences: inferences}
	err := s.EmissionsKeeper().InsertActiveInferences(s.Ctx(), topicId, blockHeight, inferencesWrapper)
	s.Require().NoError(err)
	// Test the update function
	err = s.EmissionsKeeper().UpdateNetworkInferencesOutlierMetrics(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)

	// Verify results
	mad, err := s.EmissionsKeeper().GetMadInferences(s.Ctx(), topicId)
	s.Require().NoError(err)
	median, err := s.EmissionsKeeper().GetLastMedianInferences(s.Ctx(), topicId)
	s.Require().NoError(err)
	// Verify expected values
	s.Require().Equal(alloraMath.NewDecFromInt64(1), mad)
	s.Require().Equal(alloraMath.NewDecFromInt64(11), median)

	// Modify a copy of the previous inferences and run it again
	inferencesWrapper.Inferences[0].Values = []alloraMath.Dec{alloraMath.NewDecFromInt64(100)}
	inferencesWrapper.Inferences[1].Values = []alloraMath.Dec{alloraMath.NewDecFromInt64(50)}
	err = s.EmissionsKeeper().InsertActiveInferences(s.Ctx(), topicId, blockHeight, inferencesWrapper)
	s.Require().NoError(err)

	err = s.EmissionsKeeper().UpdateNetworkInferencesOutlierMetrics(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)

	mad, err = s.EmissionsKeeper().GetMadInferences(s.Ctx(), topicId)
	s.Require().NoError(err)
	median, err = s.EmissionsKeeper().GetLastMedianInferences(s.Ctx(), topicId)
	s.Require().NoError(err)

	s.Require().Equal(alloraMath.MustNewDecFromString("8.4"), mad)
	s.Require().Equal(alloraMath.NewDecFromInt64(50), median)
}

func (s *KeeperTestSuite) TestFilterOutlierResistantInferences() {
	topicId := s.CreateTopic()

	// Ensure param is set to 11
	params := types.DefaultParams()
	params.InferenceOutlierDetectionThreshold = alloraMath.MustNewDecFromString("11")

	// Set the maximum number of unfulfilled worker nonces via the SetParams method
	err := s.EmissionsKeeper().SetParams(s.Ctx(), params)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	testCases := []struct {
		name          string
		setupMetrics  func()
		inferences    types.Inferences
		expectedCount int
		expectedVals  []alloraMath.Dec
	}{
		{
			name: "filter with median=10, mad=1",
			setupMetrics: func() {
				err := s.EmissionsKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(10))
				s.Require().NoError(err)
				err = s.EmissionsKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(1))
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(9)}},  // within bounds (|-1| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(11)}}, // within bounds (|1| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(25)}}, // outlier (|15| > 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(-5)}}, // outlier (|-15| > 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(10)}}, // within bounds (|0| < 11*1)
				},
			},
			expectedCount: 3,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(9),
				alloraMath.NewDecFromInt64(11),
				alloraMath.NewDecFromInt64(10),
			},
		},
		{
			name: "filter with median=100, mad=10",
			setupMetrics: func() {
				err := s.EmissionsKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(100))
				s.Require().NoError(err)
				err = s.EmissionsKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(10))
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(80)}},  // within bounds (|-20| < 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(120)}}, // within bounds (|20| < 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(250)}}, // outlier (|150| > 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(-50)}}, // outlier (|-150| > 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(100)}}, // within bounds (|0| < 11*10)
				},
			},
			expectedCount: 3,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(80),
				alloraMath.NewDecFromInt64(120),
				alloraMath.NewDecFromInt64(100),
			},
		},
		{
			name: "zero mad - should return all inferences",
			setupMetrics: func() {
				err := s.EmissionsKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(10))
				s.Require().NoError(err)
				err = s.EmissionsKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.ZeroDec())
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(10)}},
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(11)}},
				},
			},
			expectedCount: 2,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(10),
				alloraMath.NewDecFromInt64(11),
			},
		},
		{
			name: "zero last_median - should return all inferences",
			setupMetrics: func() {
				err := s.EmissionsKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.ZeroDec())
				s.Require().NoError(err)
				err = s.EmissionsKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(1))
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(5)}},  // within bounds (|5| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(-5)}}, // within bounds (|-5| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(15)}}, // outlier (|15| > 11*1)
				},
			},
			expectedCount: 3,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(5),
				alloraMath.NewDecFromInt64(-5),
				alloraMath.NewDecFromInt64(15),
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setupMetrics()

			filtered, err := s.EmissionsKeeper().FilterOutlierResistantInferences(s.Ctx(), topicId, tc.inferences)
			s.Require().NoError(err)

			s.Require().Len(filtered.Inferences, tc.expectedCount)

			for i, inf := range filtered.Inferences {
				s.Require().Equal(tc.expectedVals[i], inf.Values[0])
			}
		})
	}
}

func (s *KeeperTestSuite) TestInitialEmaScoreSettingInAppendInference() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	worker := s.AddrsStr(0)
	blockHeight := int64(10)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("95.5")
	err := k.SetTopicInitialInfererEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create and append a new inference
	//nolint:exhaustruct
	inference := &types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		Inferer:     worker,
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the inference
	err = k.AppendInference(ctx, topic, blockHeight, inference, 4)
	s.Require().NoError(err)

	// Verify the worker received the initial EMA score
	score, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestInitialEmaScoreSettingInAppendForecast() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	worker := s.AddrsStr(0)
	blockHeight := int64(10)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("92.5")
	err := k.SetTopicInitialForecasterEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create and append a new forecast
	forecast := &types.Forecast{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Forecaster:  worker,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: s.AddrsStr(1),
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
		ExtraData: nil,
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the forecast
	err = k.AppendForecast(ctx, topic, blockHeight, forecast, 4)
	s.Require().NoError(err)

	// Verify the worker received the initial EMA score
	score, err := k.GetForecasterScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestInitialEmaScoreSettingInAppendReputer() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()
	reputer := s.AddrsStr(0)
	blockHeight := int64(10)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("97.5")
	err := k.SetTopicInitialReputerEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create and append a new reputer value bundle
	//nolint:exhaustruct
	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
		},
		Reputer:       reputer,
		CombinedValue: alloraMath.MustNewDecFromString("0.52"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("0.52"),
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	params := types.DefaultParams()
	// Append the reputer value bundle
	err = k.AppendReputerLoss(ctx, topic, params, blockHeight, valueBundle)
	s.Require().NoError(err)

	// Verify the reputer received the initial EMA score
	score, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(reputer, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestFirstSubmissionDoesNotUpdateEMAUsingQuantile() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := s.CreateTopic()

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	//nolint:gosec
	for i := int64(0); i < int64(params.MaxTopInferersToReward); i++ {
		addr := s.AddrsStr(int(i))
		score := types.Score{
			TopicId:     topicId,
			Address:     addr,
			BlockHeight: 1,
			Score:       alloraMath.NewDecFromInt64(90 + i),
		}
		err := k.SetInfererScoreEma(ctx, topicId, addr, score)
		s.Require().NoError(err)

		err = k.AddActiveInferer(ctx, topicId, addr)
		s.Require().NoError(err)

		if i == 0 {
			err = k.SetLowestInfererScoreEma(ctx, topicId, score)
			s.Require().NoError(err)
		}
	}

	// Set a low initial score for new actors
	initialScore := alloraMath.NewDecFromInt64(50)
	err = k.SetTopicInitialInfererEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create a new inference from a new actor
	inference := &types.Inference{
		TopicId:     topicId,
		BlockHeight: 2,
		Inferer:     s.AddrsStr(9), // Using a different address
		Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(100)},
		ExtraData:   nil,
		Proof:       "",
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Submit inference - should not trigger EMA update using quantile since it's first submission
	// and score is lower than active set
	err = k.AppendInference(ctx, topic, 2, inference, params.MaxTopInferersToReward)
	s.Require().NoError(err)

	// Verify score remains at initial value
	score, err := k.GetInfererScoreEma(ctx, topicId, s.AddrsStr(9))
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
}

func (s *KeeperTestSuite) TestLivenessPenaltyAppliedInAppendInference() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	epochLength := int64(1000)
	groundTruthLag := int64(1000)

	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithInitialRegret("0.5"),
		testutil.WithEpochLastEnded(10000),
	)
	worker := s.AddrsStr(0)
	blockHeight := int64(10000)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("50")
	s.Require().NoError(k.SetTopicInitialInfererEmaScore(ctx, topicId, initialScore))

	s.Require().NoError(k.SetInfererScoreEma(ctx, topicId, worker, types.Score{
		TopicId:     topicId,
		BlockHeight: 5000,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("100"),
	}))

	// Create and append a new inference
	//nolint:exhaustruct
	inference := &types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		Inferer:     worker,
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the inference
	err = k.AppendInference(ctx, topic, blockHeight, inference, 4)
	s.Require().NoError(err)

	// Verify the worker's EMA score trended toward the topic initial score especially when there is a lapse in their
	// liveness
	score, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("82.805"), score.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", "82.805", score.Score.String())
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestLivenessPenaltyAppliedInAppendForecast() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	epochLength := int64(1000)
	groundTruthLag := int64(1000)

	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithEpochLastEnded(10000),
	)
	worker := s.AddrsStr(0)
	blockHeight := int64(10000)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("50")
	s.Require().NoError(k.SetTopicInitialForecasterEmaScore(ctx, topicId, initialScore))

	s.Require().NoError(k.SetForecasterScoreEma(ctx, topicId, worker, types.Score{
		TopicId:     topicId,
		BlockHeight: 5000,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("100"),
	}))

	// Create and append a new forecast
	forecast := &types.Forecast{
		TopicId:     topicId,
		BlockHeight: 10000,
		Forecaster:  worker,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: worker,
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
		ExtraData: nil,
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the forecast
	err = k.AppendForecast(ctx, topic, blockHeight, forecast, 4)
	s.Require().NoError(err)
	s.Require().NoError(err)

	// Verify the worker's EMA score trended toward the topic initial score especially when there is a lapse in their
	// liveness
	score, err := k.GetForecasterScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("82.805"), score.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", "82.805", score.Score.String())
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestLivenessPenaltyAppliedInAppendReputerLoss() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	epochLength := int64(1000)
	groundTruthLag := int64(1000)

	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithEpochLastEnded(10000),
	)
	reputer := s.AddrsStr(0)
	blockHeight := int64(10000)
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("50")
	s.Require().NoError(k.SetTopicInitialReputerEmaScore(ctx, topicId, initialScore))

	s.Require().NoError(k.SetReputerScoreEma(ctx, topicId, reputer, types.Score{
		TopicId:     topicId,
		BlockHeight: 5000,
		Address:     reputer,
		Score:       alloraMath.MustNewDecFromString("100"),
	}))

	// Create and append a new reputer loss
	//nolint:exhaustruct
	valueBundleReputer := types.ValueBundle{
		Reputer:             reputer,
		CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
		InfererValues:       s.createDefaultInfererValues(),
		NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
	}

	topic, err := k.GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the reputer loss
	err = k.AppendReputerLoss(ctx, topic, types.DefaultParams(), blockHeight, &valueBundleReputer)
	s.Require().NoError(err)

	// Verify the reputer's EMA score trended toward the topic initial score especially when there is a lapse in their
	// liveness
	score, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("86.450"), score.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", "86.450", score.Score.String())
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(reputer, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestLatestForecasterWeightFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(0)
	weight := alloraMath.NewDecFromInt64(100)

	// Test initial state (should be zero)
	initialWeight, err := k.GetLatestForecasterWeight(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(initialWeight.IsZero(), "Initial weight should be zero")

	// Set weight
	err = k.SetLatestForecasterWeight(ctx, topicId, forecaster, weight)
	s.Require().NoError(err, "Setting latest forecaster weight should not fail")

	// Get and verify weight
	retrievedWeight, err := k.GetLatestForecasterWeight(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(weight, retrievedWeight, "Retrieved weight should match set weight")

	// Test with different topic ID
	differentTopicId := uint64(2)
	differentWeight, err := k.GetLatestForecasterWeight(ctx, differentTopicId, forecaster)
	s.Require().NoError(err)
	s.Require().True(differentWeight.IsZero(), "Weight for different topic should be zero")
}

func (s *KeeperTestSuite) TestLatestInfererWeightFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(1)
	weight := alloraMath.NewDecFromInt64(75)

	// Test initial state (should be zero)
	initialWeight, err := k.GetLatestInfererWeight(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().True(initialWeight.IsZero(), "Initial weight should be zero")

	// Set weight
	err = k.SetLatestInfererWeight(ctx, topicId, inferer, weight)
	s.Require().NoError(err, "Setting latest inferer weight should not fail")

	// Get and verify weight
	retrievedWeight, err := k.GetLatestInfererWeight(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(weight, retrievedWeight, "Retrieved weight should match set weight")

	// Test with different topic ID
	differentTopicId := uint64(2)
	differentWeight, err := k.GetLatestInfererWeight(ctx, differentTopicId, inferer)
	s.Require().NoError(err)
	s.Require().True(differentWeight.IsZero(), "Weight for different topic should be zero")
}

func (s *KeeperTestSuite) TestLatestRegretScaleFunctions() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicId := uint64(1)
	regretScale := alloraMath.NewDecFromInt64(50)

	// Test initial state (should be zero)
	initialRegretScale, err := k.GetLatestRegretScale(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(initialRegretScale.IsZero(), "Initial regret scale should be zero")

	// Set regret scale
	err = k.SetLatestRegretScale(ctx, topicId, regretScale)
	s.Require().NoError(err, "Setting latest regret scale should not fail")

	// Get and verify regret scale
	retrievedRegretScale, err := k.GetLatestRegretScale(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(regretScale, retrievedRegretScale, "Retrieved regret scale should match set regret scale")

	// Test with different topic ID
	differentTopicId := uint64(2)
	differentRegretScale, err := k.GetLatestRegretScale(ctx, differentTopicId)
	s.Require().NoError(err)
	s.Require().True(differentRegretScale.IsZero(), "Regret scale for different topic should be zero")

	// Test setting zero value (should fail)
	err = k.SetLatestRegretScale(ctx, topicId, alloraMath.ZeroDec())
	s.Require().Error(err, "Setting zero regret scale should fail")
}

// createDefaultInfererValues generates a set of inferer values including all test addresses
// This can be used to ensure ValueBundle validation passes when InfererValues cannot be nil
func (s *KeeperTestSuite) createDefaultInfererValues() []*types.WorkerAttributedValue {
	// Create an array to hold all worker attributed values
	infererValues := make([]*types.WorkerAttributedValue, s.LenAccounts())

	// Add a WorkerAttributedValue for each address in the test suite
	for i := range s.LenAccounts() {
		infererValues[i] = &types.WorkerAttributedValue{
			Worker: s.AddrsStr(i),
			Value:  alloraMath.NewDecFromInt64(int64(i + 1)), // Using different values based on index
		}
	}

	return infererValues
}

func (s *KeeperTestSuite) TestMonthlyRewards() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	// Initial state should be zero
	reputerRewards, err := k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.IsZero(), "Initial monthly reputer rewards should be zero")

	topicRewards, err := k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.IsZero(), "Initial monthly topic rewards should be zero")

	// Add some rewards
	addReputerAmount := cosmosMath.NewInt(1000)
	addTopicAmount := cosmosMath.NewInt(5000)
	err = k.AddMonthlyRewards(ctx, addReputerAmount, addTopicAmount)
	s.Require().NoError(err)

	// Check if rewards were added
	reputerRewards, err = k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.Equal(addReputerAmount), "Monthly reputer rewards should be updated after adding")

	topicRewards, err = k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.Equal(addTopicAmount), "Monthly topic rewards should be updated after adding")

	// Add more rewards
	addMoreReputerAmount := cosmosMath.NewInt(500)
	addMoreTopicAmount := cosmosMath.NewInt(2500)
	err = k.AddMonthlyRewards(ctx, addMoreReputerAmount, addMoreTopicAmount)
	s.Require().NoError(err)

	// Check total rewards
	totalExpectedReputer := addReputerAmount.Add(addMoreReputerAmount)
	reputerRewards, err = k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.Equal(totalExpectedReputer), "Monthly reputer rewards should accumulate")

	totalExpectedTopic := addTopicAmount.Add(addMoreTopicAmount)
	topicRewards, err = k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.Equal(totalExpectedTopic), "Monthly topic rewards should accumulate")

	// Reset rewards
	err = k.ResetMonthlyRewards(ctx)
	s.Require().NoError(err)

	// Check if rewards are reset to zero
	reputerRewards, err = k.GetMonthlyReputerRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.IsZero(), "Monthly reputer rewards should be zero after reset")

	topicRewards, err = k.GetMonthlyTopicRewards(ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.IsZero(), "Monthly topic rewards should be zero after reset")
}

// TestRemoveTopicFromPreviousTopicWeights tests that when a topic is removed from previous topic weights,
// its weight is correctly subtracted from the total sum while preserving the weight itself
func (s *KeeperTestSuite) TestRemoveTopicFromPreviousTopicWeights() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(1000)
	feeRevenue := cosmosMath.NewInt(100)

	// Set up params
	params := types.DefaultParams()
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	epochLength := int64(100)
	workerSubmissionWindow := int64(100)

	// Create and activate topic
	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithWorkerSubmissionWindow(workerSubmissionWindow),
	)
	err = k.ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	// Add stake and fee revenue
	err = k.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)
	err = k.AddTopicFeeRevenue(ctx, topicId, feeRevenue)
	s.Require().NoError(err)

	// Calculate and set initial weight
	initialWeight, _, _, err := k.GetCurrentTopicWeight(
		ctx,
		topicId,
		epochLength,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)
	err = k.SetPreviousTopicWeight(ctx, topicId, initialWeight)
	s.Require().NoError(err)

	// Get initial total sum
	initialTotalSum, err := k.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(initialTotalSum.Equal(initialWeight), "Initial total sum should equal initial weight")

	// Remove topic from previous weights
	err = k.RemoveTopicFromPreviousTopicWeights(ctx, topicId)
	s.Require().NoError(err)

	// Verify total sum is updated
	newTotalSum, err := k.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(newTotalSum.IsZero(), "Total sum should be zero after removal")

	// Verify the topic's weight is still preserved
	topicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(noPrior, "Topic weight should still exist")
	s.Require().True(topicWeight.Equal(initialWeight), "Topic weight should remain unchanged")

	// Test removing a topic that has no prior weight
	nonExistentTopicId := uint64(999)
	err = k.RemoveTopicFromPreviousTopicWeights(ctx, nonExistentTopicId)
	s.Require().NoError(err, "Removing non-existent topic should not error")

	// Verify total sum remains unchanged after removing non-existent topic
	finalTotalSum, err := k.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().True(finalTotalSum.Equal(newTotalSum), "Total sum should remain unchanged after removing non-existent topic")
}

//nolint:staticcheck
func (s *KeeperTestSuite) TestEpochLabelRegistry() {
	type testCase struct {
		name string
		run  func(ctx sdk.Context, k *keeper.Keeper)
	}

	newFixture := func() (sdk.Context, *keeper.Keeper, types.TopicId, types.BlockHeight) {
		ctx := s.Ctx()
		k := s.EmissionsKeeper()
		topicId := s.CreateTopic()
		nonce := types.BlockHeight(7)
		return ctx, k, topicId, nonce
	}

	tests := []testCase{
		{
			name: "Get empty registry returns empty (no error)",
			run: func(ctx sdk.Context, k *keeper.Keeper) {
				_, _, topicId, nonce := newFixture()
				reg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
				s.Require().NoError(err)
				s.Require().Equal(topicId, reg.TopicId)
				s.Require().Equal(uint64(nonce), reg.EpochId)
				s.Require().Empty(reg.Labels)
			},
		},
		{
			name: "Register one label assigns ID=1 and persists",
			run: func(ctx sdk.Context, k *keeper.Keeper) {
				ctx, k, topicId, nonce := newFixture()

				id, err := k.RegisterEpochLabel(ctx, topicId, nonce, "UP")
				s.Require().NoError(err)
				s.Require().Equal(keeper.LabelId(1), id)

				reg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
				s.Require().NoError(err)
				s.Require().Len(reg.Labels, 1)
				s.Require().Equal(uint32(1), reg.Labels[0].Id)
				s.Require().Equal("UP", reg.Labels[0].Name)
			},
		},
		{
			name: "Register two labels assigns ID=2 on second label",
			run: func(ctx sdk.Context, k *keeper.Keeper) {
				ctx, k, topicId, nonce := newFixture()

				_, err := k.RegisterEpochLabel(ctx, topicId, nonce, "UP")
				s.Require().NoError(err)

				id, err := k.RegisterEpochLabel(ctx, topicId, nonce, "DOWN")
				s.Require().NoError(err)
				s.Require().Equal(keeper.LabelId(2), id)

				gotID, ok, err := k.GetEpochLabelId(ctx, topicId, nonce, "UP")
				s.Require().NoError(err)
				s.Require().True(ok)
				s.Require().Equal(keeper.LabelId(1), gotID)

				gotName, ok, err := k.GetEpochLabelName(ctx, topicId, nonce, keeper.LabelId(2))
				s.Require().NoError(err)
				s.Require().True(ok)
				s.Require().Equal("DOWN", gotName)
			},
		},
		{
			name: "Duplicate register does not create new ID or new label",
			run: func(ctx sdk.Context, k *keeper.Keeper) {
				ctx, k, topicId, nonce := newFixture()

				id1, err := k.RegisterEpochLabel(ctx, topicId, nonce, "UP")
				s.Require().NoError(err)
				s.Require().Equal(keeper.LabelId(1), id1)

				id2, err := k.RegisterEpochLabel(ctx, topicId, nonce, "UP")
				s.Require().NoError(err)
				s.Require().Equal(keeper.LabelId(1), id2)

				reg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
				s.Require().NoError(err)
				s.Require().Len(reg.Labels, 1)
				s.Require().Equal(uint32(1), reg.Labels[0].Id)
				s.Require().Equal("UP", reg.Labels[0].Name)
			},
		},
		{
			name: "Missing lookups return ok=false (no error)",
			run: func(ctx sdk.Context, k *keeper.Keeper) {
				ctx, k, topicId, nonce := newFixture()

				_, err := k.RegisterEpochLabel(ctx, topicId, nonce, "UP")
				s.Require().NoError(err)

				_, ok, err := k.GetEpochLabelId(ctx, topicId, nonce, "MISSING")
				s.Require().NoError(err)
				s.Require().False(ok)

				_, ok, err = k.GetEpochLabelName(ctx, topicId, nonce, keeper.LabelId(999))
				s.Require().NoError(err)
				s.Require().False(ok)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctx, k, _, _ := newFixture()
			tc.run(ctx, k)
		})
	}
}

func (s *KeeperTestSuite) TestInputInferenceForecastBundleConvert() {
	validInference := &types.InputInference{
		TopicId:     1,
		BlockHeight: 100,
		Inferer:     "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
		Value:       alloraMath.MustNewBoundedExp40DecFromString("1.23"),
		Values:      []*types.InputLabeledValue{{Label: "", Value: alloraMath.MustNewBoundedExp40DecFromString("1.23")}},
		ExtraData:   []byte("extra"),
		Proof:       "proof",
	}

	validForecast := &types.InputForecast{
		TopicId:     1,
		BlockHeight: 100,
		Forecaster:  "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		ForecastElements: []*types.InputForecastElement{
			{
				Inferer: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
				Value:   alloraMath.MustNewBoundedExp40DecFromString("1.23"),
			},
		},
		ExtraData: []byte("extra"),
	}

	tests := []struct {
		name    string
		input   *types.InputInferenceForecastBundle
		wantErr bool
	}{
		{
			name: "valid input",
			input: &types.InputInferenceForecastBundle{
				Inference: validInference,
				Forecast:  validForecast,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), validInference.TopicId)
			s.Require().NoError(err)

			got, err := s.EmissionsKeeper().NewInferenceForecastBundleFromInput(s.Ctx(), topic, validInference.BlockHeight, tt.input)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			if tt.input == nil {
				s.Require().Nil(got)
				return
			}
			s.Require().NotNil(got.Inference)
			s.Require().NotNil(got.Forecast)
		})
	}
}

func (s *KeeperTestSuite) TestNormalizeInputInference() {
	type tc struct {
		name         string
		arity        types.TopicOutputArity
		requireUnity bool
		unityTol     string

		nonce int64

		preRegisterLabels []string

		scalarValue string
		labeled     []struct {
			label string
			value string
		}

		wantErr   bool
		wantErrIs error

		wantValuesStr []string
		wantRegLabels []string
	}

	cases := []tc{
		{
			name:         "SINGLE_uses_labeled_when_len1_over_scalar",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			scalarValue:  "999",
			labeled: []struct {
				label string
				value string
			}{
				{label: "x", value: "7"},
			},
			wantValuesStr: []string{"7"},
			wantRegLabels: nil,
		},
		{
			name:          "SINGLE_uses_scalar_when_no_labeled",
			arity:         types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity:  false,
			unityTol:      "0",
			nonce:         1,
			scalarValue:   "42",
			labeled:       nil,
			wantValuesStr: []string{"42"},
			wantRegLabels: nil,
		},
		{
			name:         "SINGLE_rejects_when_labeled_len_gt_1",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			scalarValue:  "1",
			labeled: []struct {
				label string
				value string
			}{
				{label: "x", value: "1"},
				{label: "y", value: "2"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_requires_labeled_values",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			scalarValue:  "123",
			labeled:      nil,
			wantErr:      true,
			wantErrIs:    sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_registers_labels_and_aligns_dense",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			labeled: []struct {
				label string
				value string
			}{
				{label: "A", value: "0.2"},
				{label: "B", value: "0.8"},
			},
			wantValuesStr: []string{"0.2", "0.8"},
			wantRegLabels: []string{"A", "B"},
		},
		{
			name:         "MULTI_duplicate_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			labeled: []struct {
				label string
				value string
			}{
				{label: "A", value: "0.1"},
				{label: "A", value: "0.2"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_empty_label_rejected",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			labeled: []struct {
				label string
				value string
			}{
				{label: "   ", value: "0.1"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_missing_labels_are_zero_against_existing_registry",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			preRegisterLabels: []string{
				"a", "b", "c",
			},
			labeled: []struct {
				label string
				value string
			}{
				{label: "a", value: "1"},
				{label: "b", value: "2"},
			},
			wantValuesStr: []string{"1", "2", "0"},
			wantRegLabels: []string{"a", "b", "c"},
		},
		{
			name:         "MULTI_require_unity_ok",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: true,
			unityTol:     "0.000001",
			nonce:        1,
			labeled: []struct {
				label string
				value string
			}{
				{label: "A", value: "0.2"},
				{label: "B", value: "0.8"},
			},
			wantValuesStr: []string{"0.2", "0.8"},
			wantRegLabels: []string{"A", "B"},
		},
		{
			name:         "MULTI_require_unity_rejected_outside_tol",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: true,
			unityTol:     "0.01",
			nonce:        1,
			labeled: []struct {
				label string
				value string
			}{
				{label: "A", value: "0.2"},
				{label: "B", value: "0.7"},
			},
			wantErr:   true,
			wantErrIs: sdkerrors.ErrInvalidRequest,
		},
		{
			name:         "MULTI_trims_labels_and_is_idempotent_on_registry",
			arity:        types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			requireUnity: false,
			unityTol:     "0",
			nonce:        1,
			labeled: []struct {
				label string
				value string
			}{
				{label: "  Z  ", value: "5"},
			},
			wantValuesStr: []string{"5"},
			wantRegLabels: []string{"Z"},
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			ctx := s.Ctx()
			k := s.EmissionsKeeper()

			topicId := s.CreateTopic()
			topic, err := k.GetTopic(ctx, topicId)
			s.Require().NoError(err)

			topic.OutputArity = c.arity
			topic.RequireUnity = c.requireUnity
			topic.UnityTolerance = alloraMath.MustNewDecFromString(c.unityTol)
			s.Require().NoError(k.SetTopic(ctx, topicId, topic))

			for _, l := range c.preRegisterLabels {
				_, err := k.RegisterEpochLabel(ctx, topicId, c.nonce, l)
				s.Require().NoError(err)
			}

			in := &types.InputInference{
				TopicId:     topicId,
				BlockHeight: c.nonce,
				Inferer:     "inferer",
				Value:       alloraMath.MustNewBoundedExp40DecFromString(c.scalarValue),
			}
			if c.labeled != nil {
				in.Values = make([]*types.InputLabeledValue, 0, len(c.labeled))
				for _, lv := range c.labeled {
					in.Values = append(in.Values, &types.InputLabeledValue{
						Label: lv.label,
						Value: alloraMath.MustNewBoundedExp40DecFromString(lv.value),
					})
				}
			}

			got, err := k.NormalizeInputInference(ctx, topic, c.nonce, in)
			if c.wantErr {
				s.Require().Error(err)
				if c.wantErrIs != nil {
					s.Require().True(errorsmod.IsOf(err, c.wantErrIs), "expected error to be %v, got %v", c.wantErrIs, err)
				}
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(got)

			s.Require().Equal(len(c.wantValuesStr), len(got.Values))
			for i := range c.wantValuesStr {
				s.Require().Equal(c.wantValuesStr[i], got.Values[i].String())
			}

			reg, err := k.GetEpochLabelRegistry(ctx, topicId, c.nonce)
			s.Require().NoError(err)

			if c.arity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
				s.Require().Len(reg.Labels, 1)
				return
			}

			s.Require().Equal(len(c.wantRegLabels), len(reg.Labels))
			for i := range c.wantRegLabels {
				s.Require().Equal(c.wantRegLabels[i], reg.Labels[i].Name)
			}
		})
	}

	s.Run("MULTI_preserves_label_ids_across_calls_even_if_submission_order_changes", func() {
		s.SetupTest()

		ctx := s.Ctx()
		k := s.EmissionsKeeper()

		topicId := s.CreateTopic()
		topic, err := k.GetTopic(ctx, topicId)
		s.Require().NoError(err)
		topic.OutputArity = types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
		topic.RequireUnity = false
		topic.UnityTolerance = alloraMath.ZeroDec()
		s.Require().NoError(k.SetTopic(ctx, topicId, topic))

		nonce := types.BlockHeight(1)

		in1 := &types.InputInference{
			TopicId:     topicId,
			BlockHeight: 1,
			Inferer:     "inferer",
			Value:       alloraMath.MustNewBoundedExp40DecFromString("0"),
			Values: []*types.InputLabeledValue{
				{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("1")},
				{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("2")},
			},
		}
		got1, err := k.NormalizeInputInference(ctx, topic, nonce, in1)
		s.Require().NoError(err)
		s.Require().Equal([]string{"1", "2"}, []string{got1.Values[0].String(), got1.Values[1].String()})

		in2 := &types.InputInference{
			TopicId:     topicId,
			BlockHeight: 1,
			Inferer:     "inferer",
			Value:       alloraMath.MustNewBoundedExp40DecFromString("0"),
			Values: []*types.InputLabeledValue{
				{Label: "b", Value: alloraMath.MustNewBoundedExp40DecFromString("20")},
				{Label: "a", Value: alloraMath.MustNewBoundedExp40DecFromString("10")},
			},
		}
		got2, err := k.NormalizeInputInference(ctx, topic, nonce, in2)
		s.Require().NoError(err)
		s.Require().Equal([]string{"10", "20"}, []string{got2.Values[0].String(), got2.Values[1].String()})

		reg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
		s.Require().NoError(err)
		s.Require().Len(reg.Labels, 2)
		s.Require().Equal("a", reg.Labels[0].Name)
		s.Require().Equal("b", reg.Labels[1].Name)
	})
}

func (s *KeeperTestSuite) TestGetWorkersLatestInferencesByTopicIdValuesPadded() {
	topicId := keeper.TopicId(1)
	nonce := int64(1)

	type tc struct {
		name         string
		outputArity  types.TopicOutputArity
		setup        func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string)
		workersOrder []int
		wantErrIs    error
		wantValues   map[int][]string
	}

	mustDec := func(x string) alloraMath.Dec { return alloraMath.MustNewDecFromString(x) }

	setTopic := func(ctx context.Context, k *keeper.Keeper, topicId uint64, arity types.TopicOutputArity) {
		topic, err := k.GetTopic(ctx, topicId)
		s.Require().NoError(err)
		topic.OutputArity = arity
		topic.RequireUnity = false
		topic.UnityTolerance = alloraMath.ZeroDec()
		err = k.SetTopic(ctx, topicId, topic)
		s.Require().NoError(err)
	}

	setLatestInf := func(ctx context.Context, k *keeper.Keeper, topicId uint64, inf types.Inference) {
		err := k.InsertInference(ctx, topicId, inf)
		s.Require().NoError(err)
	}

	cases := []tc{
		{
			name:        "SINGLE_scalar_only_populates_values_len1",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("42")},
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"42"},
			},
		},
		{
			name:        "SINGLE_values_len1_used_and_consistent_with_scalar",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("7")},
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"7"},
			},
		},
		{
			name:        "SINGLE_scalar_and_values_mismatch_canonicalizes_to_values0",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("2")},
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"2"},
			},
		}, {
			name:        "SINGLE_rejects_values_len_gt_1",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("1"), mustDec("2")},
				})
			},
			workersOrder: []int{0},
			wantErrIs:    sdkerrors.ErrLogic,
		},
		{
			name:        "SINGLE_two_workers_scalar_only_both_values_len1_and_sorted_by_inferer",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("42")},
				})
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w2,
					Values:      []alloraMath.Dec{mustDec("7")},
				})
			},
			workersOrder: []int{1, 0},
			wantValues: map[int][]string{
				0: {"42"},
				1: {"7"},
			},
		},
		{
			name:        "MULTI_pads_shorter_values_to_registry_len_and_sorts_inferers",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				_, err := k.RegisterEpochLabel(ctx, topicId, nonce, "a")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "b")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "c")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "d")
				s.Require().NoError(err)

				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("1"), mustDec("2"), mustDec("3")},
				})
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w2,
					Values:      []alloraMath.Dec{mustDec("10"), mustDec("20"), mustDec("0"), mustDec("40")},
				})
			},
			workersOrder: []int{1, 0},
			wantValues: map[int][]string{
				0: {"1", "2", "3", "0"},
				1: {"10", "20", "0", "40"},
			},
		},
		{
			name:        "MULTI_values_longer_than_registry_rejected",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				_, err := k.RegisterEpochLabel(ctx, topicId, nonce, "a")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "b")
				s.Require().NoError(err)

				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("1"), mustDec("2"), mustDec("3")},
				})
			},
			workersOrder: []int{0},
			wantErrIs:    sdkerrors.ErrLogic,
		}, {
			name:        "MULTI_registry_len_zero_rejects_any_nonempty_inference_values",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				// do NOT register labels => registry len = 0
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("1")},
				})
			},
			workersOrder: []int{0},
			wantErrIs:    sdkerrors.ErrLogic,
		},
		{
			name:        "MULTI_registry_len_zero_allows_empty_values_no_padding_needed",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				// registry len = 0
				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{},
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {},
			},
		},
		{
			name:        "MULTI_pads_multiple_missing_entries_not_just_one",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				_, err := k.RegisterEpochLabel(ctx, topicId, nonce, "a")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "b")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "c")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "d")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "e")
				s.Require().NoError(err)

				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("1")}, // should become [1,0,0,0,0]
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"1", "0", "0", "0", "0"},
			},
		},
		{
			name:        "MULTI_does_not_mutate_stored_inference_when_padding",
			outputArity: types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
			setup: func(ctx context.Context, k *keeper.Keeper, topicId uint64, nonce int64, w1, w2 string) {
				_, err := k.RegisterEpochLabel(ctx, topicId, nonce, "a")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "b")
				s.Require().NoError(err)
				_, err = k.RegisterEpochLabel(ctx, topicId, nonce, "c")
				s.Require().NoError(err)

				setLatestInf(ctx, k, topicId, types.Inference{
					TopicId:     topicId,
					BlockHeight: nonce,
					Inferer:     w1,
					Values:      []alloraMath.Dec{mustDec("9")}, // stored len=1
				})
			},
			workersOrder: []int{0},
			wantValues: map[int][]string{
				0: {"9", "0", "0"},
			},
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.SetupTest()

			ctx := s.Ctx()
			k := s.EmissionsKeeper()

			w1 := s.AddrsStr(0)
			w2 := s.AddrsStr(1)
			workers := []string{w1, w2}

			setTopic(ctx, k, topicId, c.outputArity)

			if c.setup != nil {
				c.setup(ctx, k, topicId, nonce, w1, w2)
			}

			reqWorkers := make([]string, 0, len(c.workersOrder))
			for _, idx := range c.workersOrder {
				reqWorkers = append(reqWorkers, workers[idx])
			}

			topic, err := k.GetTopic(ctx, topicId)
			s.Require().NoError(err)

			got, err := k.GetWorkersLatestInferencesByTopicIdValuesPadded(ctx, topic, nonce, reqWorkers)
			if c.wantErrIs != nil {
				s.Require().ErrorIs(err, c.wantErrIs)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(got)

			for workerIdx, want := range c.wantValues {
				addr := workers[workerIdx]
				var found *types.Inference
				for _, inf := range got.Inferences {
					if inf.Inferer == addr {
						found = inf
						break
					}
				}
				s.Require().NotNil(found)
				s.Require().Equal(len(want), len(found.Values))
				for i := range want {
					s.Require().Equal(want[i], found.Values[i].String())
				}
			}
		})
	}
}
