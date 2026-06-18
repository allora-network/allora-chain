package keeper_test

import (
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestSetAndGetPreviousTopicWeight() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
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
	k := s.TopicKeeper()
	topicId := uint64(2)

	// Attempt to get a weight for a topic that has no set weight
	retrievedWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	s.Require().NoError(err, "Getting weight for an unset topic should not error but return zero value")
	s.Require().True(alloraMath.ZeroDec().Equal(retrievedWeight), "Weight for an unset topic should be zero")
	s.Require().True(noPrior, "Should indicate no prior weight for an unset topic")
}

func (s *KeeperTestSuite) TestInactivateAndActivateTopic() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	topicId := uint64(3)

	maxActiveTopicsNum := uint64(5)
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	err := s.ParamsKeeper().SetParams(ctx, params)
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
	k := s.TopicKeeper()

	maxActiveTopicsNum := uint64(2)
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	params.MaxPageLimit = 100
	params.MinEpochLength = 1
	err := s.ParamsKeeper().SetParams(ctx, params)
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
	k := s.TopicKeeper()

	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = uint64(3)
	params.MaxPageLimit = uint64(100)
	params.MinEpochLength = 1
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.MustNewDecFromString("1")
	err := s.ParamsKeeper().SetParams(s.Ctx(), params)
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
		err = s.StakingKeeper().SetTopicStake(s.Ctx(), topicId, cosmosMath.NewInt(stake))
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
	k := s.TopicKeeper()

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
	k := s.TopicKeeper()

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
	k := s.TopicKeeper()
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
	k := s.TopicKeeper()

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
	k := s.TopicKeeper()

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
	k := s.TopicKeeper()
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
	k := s.TopicKeeper()

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

// TOPIC REWARD NONCE
func (s *KeeperTestSuite) TestGetSetDeleteTopicRewardNonce() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
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
			got, err := s.TopicKeeper().GetTargetWeight(tc.topicStake, tc.topicFeeRevenue, tc.stakeImportance, tc.feeImportance)
			if tc.expectError {
				s.Require().Error(err, "Expected an error for case: %s", tc.name)
			} else {
				s.Require().NoError(err, "Did not expect an error for case: %s", tc.name)
				s.Require().True(tc.want.Equal(got), "Expected %s, got %s for case %s", tc.want.String(), got.String(), tc.name)
			}
		})
	}
}

func (s *KeeperTestSuite) TestDripTopicFeeRevenue() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.TopicKeeper()
	require := s.Require()

	// Define test data
	block := int64(100)
	// Calculated expected drip with these values: 26
	expectedDrip := cosmosMath.NewInt(26)
	initialRevenue := cosmosMath.NewInt(1000000) // 0.001 in Int representation (assuming 6 decimal places)

	params := types.DefaultParams()
	params.MinEpochLength = 1
	err := s.ParamsKeeper().SetParams(ctx, params)
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
	k := s.TopicKeeper()
	require := s.Require()

	// Define test data
	topicId1 := uint64(2)
	topicId2 := uint64(3)
	block := int64(100)
	initialRevenue := cosmosMath.NewInt(1000000) // 0.001 in Int representation (assuming 6 decimal places)

	params := types.DefaultParams()
	params.MinEpochLength = 1
	err := s.ParamsKeeper().SetParams(ctx, params)
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

// TestRemoveTopicFromPreviousTopicWeights tests that when a topic is removed from previous topic weights,
// its weight is correctly subtracted from the total sum while preserving the weight itself
func (s *KeeperTestSuite) TestRemoveTopicFromPreviousTopicWeights() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(1000)
	feeRevenue := cosmosMath.NewInt(100)

	// Set up params
	params := types.DefaultParams()
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	err := s.ParamsKeeper().SetParams(ctx, params)
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
	err = s.StakingKeeper().AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
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

// TestUpdateTopic_RejectsLabelCaseSensitiveChange pins the keeper-level guard
// that LabelCaseSensitive is immutable after topic creation. The msgserver
// rebuilds updatedTopic from the stored topic, so this branch is only reachable
// via direct keeper calls.
func (s *KeeperTestSuite) TestUpdateTopic_RejectsLabelCaseSensitiveChange() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	testCases := []struct {
		name             string
		initialSensitive bool
		updatedSensitive bool
	}{
		{
			name:             "false to true",
			initialSensitive: false,
			updatedSensitive: true,
		},
		{
			name:             "true to false",
			initialSensitive: true,
			updatedSensitive: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			topicId := s.CreateTopic(testutil.WithLabelCaseSensitive(tc.initialSensitive))

			storedTopic, err := k.GetTopic(ctx, topicId)
			s.Require().NoError(err)
			s.Require().Equal(tc.initialSensitive, storedTopic.LabelCaseSensitive)

			updatedTopic := storedTopic
			updatedTopic.LabelCaseSensitive = tc.updatedSensitive

			_, err = k.UpdateTopic(ctx, storedTopic, updatedTopic)
			s.Require().Error(err)
			s.Require().True(errorsmod.IsOf(err, types.ErrInvalidTopicUpdate))
			s.Require().ErrorContains(err, "label_case_sensitive is immutable after topic creation")

			unchangedTopic, err := k.GetTopic(ctx, topicId)
			s.Require().NoError(err)
			s.Require().Equal(storedTopic, unchangedTopic)
		})
	}
}

// TestRegisterEpochLabels_OverCapAllowsIdempotentRejectsNew pins the contract
// that once a stored registry already exceeds maxRegistrySize — only reachable
// if MaxEpochLabelRegistrySize is lowered by governance after labels were
// registered — resubmitting an existing label stays idempotent (no growth, no
// error) while a genuinely new label is rejected with
// ErrEpochLabelRegistrySaturated.
func (s *KeeperTestSuite) TestRegisterEpochLabels_OverCapAllowsIdempotentRejectsNew() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	topicId := uint64(1)
	nonce := int64(10)
	maxBytes := uint64(64)

	// Fill the registry up to its cap of 3 labels.
	ids, reg, err := k.RegisterEpochLabels(ctx, topicId, false, nonce, []string{"a", "b", "c"}, maxBytes, uint64(3))
	s.Require().NoError(err)
	s.Require().Equal([]keeper.LabelId{1, 2, 3}, ids)
	s.Require().Len(reg.Labels, 3)

	// Simulate governance lowering MaxEpochLabelRegistrySize below the current
	// registry size: the stored registry (3) now exceeds the cap (1).
	loweredCap := uint64(1)

	// Existing labels remain idempotent even over the cap: original ids, no growth.
	existingIds, existingReg, err := k.RegisterEpochLabels(ctx, topicId, false, nonce, []string{"b", "a"}, maxBytes, loweredCap)
	s.Require().NoError(err)
	s.Require().Equal([]keeper.LabelId{2, 1}, existingIds)
	s.Require().Len(existingReg.Labels, 3)

	// A genuinely new label over the cap is rejected.
	_, _, err = k.RegisterEpochLabels(ctx, topicId, false, nonce, []string{"d"}, maxBytes, loweredCap)
	s.Require().Error(err)
	s.Require().True(errorsmod.IsOf(err, types.ErrEpochLabelRegistrySaturated))

	// The rejected call must not have grown the registry.
	finalReg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
	s.Require().NoError(err)
	s.Require().Len(finalReg.Labels, 3)
}

// TestGetEpochLabelRegistryEmpty pins the invariant that GetEpochLabelRegistry
// returns an empty-but-well-formed registry (no error) when nothing has been
// written for (topicId, nonce).
func (s *KeeperTestSuite) TestGetEpochLabelRegistryEmpty() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	topicId := s.CreateTopic()
	nonce := types.BlockHeight(7)

	reg, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
	s.Require().NoError(err)
	s.Require().Equal(topicId, reg.TopicId)
	s.Require().Equal(uint64(nonce), reg.EpochId) //nolint:gosec // nonce is a non-negative block height; cast is safe
	s.Require().Empty(reg.Labels)
}

// TestSetTopic_CanonicalizesLabelWhitelist pins that SetTopic
// canonicalizes label_whitelist using the topic's LabelCaseSensitive flag, so
// that create/update-time canonicalization matches submission-time
// canonicalization byte-for-byte. A mismatch here would silently reject or
// accept the wrong labels at submission time.
//
// Case-insensitive mode lowercases ASCII letters (so "Foo" and "foo" collide),
// while case-sensitive mode preserves case (so they remain distinct). Both
// modes trim surrounding whitespace.
func (s *KeeperTestSuite) TestSetTopic_CanonicalizesLabelWhitelist() {
	ctx := s.Ctx()
	k := s.TopicKeeper()

	testCases := []struct {
		name          string
		caseSensitive bool
		input         []string
		expected      []string
		expectErr     bool
	}{
		{
			name:          "case-insensitive lowercases and trims mixed-case labels",
			caseSensitive: false,
			input:         []string{"Foo", "BAR", "  baz  "},
			expected:      []string{"foo", "bar", "baz"},
			expectErr:     false,
		},
		{
			name:          "case-insensitive rejects entries that collide after lowercasing",
			caseSensitive: false,
			input:         []string{"Foo", "foo"},
			expected:      nil,
			expectErr:     true,
		},
		{
			name:          "case-sensitive preserves case and trims mixed-case labels",
			caseSensitive: true,
			input:         []string{"Foo", "BAR", "  baz  "},
			expected:      []string{"Foo", "BAR", "baz"},
			expectErr:     false,
		},
		{
			name:          "case-sensitive keeps case-differing labels distinct",
			caseSensitive: true,
			input:         []string{"Foo", "foo"},
			expected:      []string{"Foo", "foo"},
			expectErr:     false,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			topicId := uint64(i + 1)
			topic := s.MockTopic()
			topic.Id = topicId
			topic.LabelCaseSensitive = tc.caseSensitive
			topic.LabelWhitelist = append([]string(nil), tc.input...)

			err := k.SetTopic(ctx, topicId, topic)
			if tc.expectErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			stored, err := k.GetTopic(ctx, topicId)
			s.Require().NoError(err)
			s.Require().Equal(tc.expected, stored.LabelWhitelist)
			s.Require().Equal(tc.caseSensitive, stored.LabelCaseSensitive)
		})
	}
}
