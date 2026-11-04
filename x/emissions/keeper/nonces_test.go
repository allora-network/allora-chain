package keeper_test

import (
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// WORKER NONCE TESTS

func (s *KeeperTestSuite) TestAddWorkerNonce() {
	ctx := s.Ctx()
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
	topicId := uint64(1)
	maxUnfulfilledRequests := 3
	// Set the maximum number of unfulfilled worker nonces
	params := types.DefaultParams()
	params.MaxUnfulfilledWorkerRequests = uint64(maxUnfulfilledRequests)

	// Set the maximum number of unfulfilled worker nonces via the SetParams method
	err := s.ParamsKeeper().SetParams(ctx, params)
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
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
	k := s.NonceKeeper()
	nonExistentTopicId := uint64(99999)

	_, err := k.GetOpenReputerSubmissionWindows(s.Ctx(), nonExistentTopicId)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrTopicDoesNotExist, "Retrieving a non-existent topic should result in an error")
}

func (s *KeeperTestSuite) TestGetOpenWorkerSubmissionWindowsWithNonExistentTopic() {
	k := s.NonceKeeper()
	nonExistentTopicId := uint64(99999)

	_, err := k.GetOpenWorkerSubmissionWindows(s.Ctx(), nonExistentTopicId)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrTopicDoesNotExist, "Retrieving a non-existent topic should result in an error")
}

func (s *KeeperTestSuite) TestReputerNonceLimitEnforcement() {
	ctx := s.Ctx()
	k := s.NonceKeeper()
	topicId := uint64(1)
	maxUnfulfilledRequests := 3

	// Set the maximum number of unfulfilled reputer nonces
	params := types.DefaultParams()
	params.MaxUnfulfilledReputerRequests = uint64(maxUnfulfilledRequests)

	// Set the maximum number of unfulfilled reputer nonces via the SetParams method
	err := s.ParamsKeeper().SetParams(ctx, params)
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

func (s *KeeperTestSuite) TestPruneWorkerNoncesLogicNoNonces() {
	k := s.NonceKeeper()
	topicId1 := uint64(1)
	blockHeightThreshold := int64(10)
	err := k.DeleteUnfulfilledWorkerNonces(s.Ctx(), topicId1)
	s.Require().NoError(err, "Failed to delete unfulfilled worker nonces, topicId1")

	// Call pruneWorkerNonces
	err = k.PruneWorkerNonces(s.Ctx(), topicId1, blockHeightThreshold)
	s.Require().NoError(err)

	// Check remaining nonces
	nonces, err := k.GetUnfulfilledWorkerNonces(s.Ctx(), topicId1)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestAddReputerNonceNilInput() {
	err := s.NonceKeeper().AddReputerNonce(s.Ctx(), uint64(1), nil)
	s.Require().Error(err)
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
	k := s.NonceKeeper()
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
			err = k.PruneWorkerNonces(s.Ctx(), topicId1, tt.blockHeightThreshold)
			s.Require().NoError(err)

			// Check remaining nonces
			nonces, err := k.GetUnfulfilledWorkerNonces(s.Ctx(), topicId1)
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
	k := s.NonceKeeper()
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
			err = k.PruneReputerNonces(s.Ctx(), topicId1, tt.blockHeightThreshold)
			s.Require().NoError(err)

			// Check remaining nonces
			nonces, err := k.GetUnfulfilledReputerNonces(s.Ctx(), topicId1)
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

func (s *KeeperTestSuite) TestDeleteUnfulfilledWorkerNonces() {
	topicId := uint64(1)
	k := s.NonceKeeper()
	// Setup initial nonces
	err := k.AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 10})
	s.Require().NoError(err)
	err = k.AddWorkerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 20})
	s.Require().NoError(err)

	// Call DeleteUnfulfilledWorkerNonces
	err = k.DeleteUnfulfilledWorkerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Check that the nonces were removed
	nonces, err := k.GetUnfulfilledWorkerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

func (s *KeeperTestSuite) TestDeleteUnfulfilledreputerNonces() {
	topicId := uint64(1)
	k := s.NonceKeeper()
	// Setup initial nonces
	err := k.AddReputerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 50})
	s.Require().NoError(err)
	err = k.AddReputerNonce(s.Ctx(), topicId, &types.Nonce{BlockHeight: 60})
	s.Require().NoError(err)

	// Call DeleteUnfulfilledWorkerNonces
	err = k.DeleteUnfulfilledReputerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Check that the nonces were removed
	nonces, err := k.GetUnfulfilledReputerNonces(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Empty(nonces.Nonces)
}

// NIL GUARD TESTS

func (s *KeeperTestSuite) TestIsWorkerNonceUnfulfilledNilNonce() {
	k := s.NonceKeeper()
	_, err := k.IsWorkerNonceUnfulfilled(s.Ctx(), uint64(1), nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nil worker nonce provided")
}

func (s *KeeperTestSuite) TestFulfillWorkerNonceNilNonce() {
	k := s.NonceKeeper()
	_, err := k.FulfillWorkerNonce(s.Ctx(), uint64(1), nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nil worker nonce provided")
}

func (s *KeeperTestSuite) TestAddWorkerNonceNilNonce() {
	k := s.NonceKeeper()
	err := k.AddWorkerNonce(s.Ctx(), uint64(1), nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nil worker nonce provided")
}

func (s *KeeperTestSuite) TestFulfillReputerNonceNilNonce() {
	k := s.NonceKeeper()
	_, err := k.FulfillReputerNonce(s.Ctx(), uint64(1), nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nil reputer nonce provided")
}

func (s *KeeperTestSuite) TestIsReputerNonceUnfulfilledNilNonce() {
	k := s.NonceKeeper()
	_, err := k.IsReputerNonceUnfulfilled(s.Ctx(), uint64(1), nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nil reputer nonce provided")
}
