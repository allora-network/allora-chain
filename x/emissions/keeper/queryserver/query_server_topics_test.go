package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetNextTopicId() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	// Get the initial next topic ID
	initialNextTopicId, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err, "Fetching the initial next topic ID should not fail")

	topicsToCreate := 5
	for i := 1; i <= topicsToCreate; i++ {
		topicId, err := keeper.IncrementTopicId(ctx)
		s.Require().NoError(err, "Incrementing topic ID should not fail")

		newTopic := types.Topic{Id: topicId}
		err = keeper.SetTopic(ctx, topicId, newTopic)
		s.Require().NoError(err, "Setting a new topic should not fail")
	}

	req := &types.GetNextTopicIdRequest{}

	response, err := queryServer.GetNextTopicId(ctx, req)
	s.Require().NoError(err, "GetNextTopicId should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	expectedNextTopicId := initialNextTopicId + uint64(topicsToCreate)
	s.Require().Equal(expectedNextTopicId, response.NextTopicId, "The next topic ID should match the expected value after topic creation")
}

func (s *QueryServerTestSuite) TestGetTopic() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err)
	metadata := "metadata"
	req := &types.GetTopicRequest{TopicId: topicId}

	// Setting up a new topic
	newTopic := types.Topic{Id: topicId, Metadata: metadata}
	err = keeper.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test retrieving an existing topic
	response, err := queryServer.GetTopic(ctx, req)
	s.Require().NoError(err, "Retrieving an existing topic should not fail")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().NotNil(response.Topic, "The response's Topic should not be nil")
	s.Require().Equal(newTopic, *response.Topic, "Retrieved topic should match the set topic")
	s.Require().Equal(metadata, response.Topic.Metadata, "The metadata of the retrieved topic should match")
}

func (s *QueryServerTestSuite) TestGetLatestCommit() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	blockHeight := 100
	nonce := types.Nonce{
		BlockHeight: 95,
	}

	topic := types.Topic{Id: 1}
	_ = keeper.SetReputerTopicLastCommit(
		ctx,
		topic.Id,
		int64(blockHeight),
		&nonce,
	)

	req := &types.GetTopicLastReputerCommitInfoRequest{
		TopicId: topic.Id,
	}

	response, err := queryServer.GetTopicLastReputerCommitInfo(ctx, req)
	s.Require().NoError(err, "GetTopicLastReputerCommitInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(int64(blockHeight), response.LastCommit.BlockHeight, "Retrieved blockheight should match")
	s.Require().Equal(&nonce, response.LastCommit.Nonce, "The metadata of the retrieved nonce should match")

	topic2 := types.Topic{Id: 2}
	blockHeight = 101
	nonce = types.Nonce{
		BlockHeight: 98,
	}

	_ = keeper.SetWorkerTopicLastCommit(
		ctx,
		topic2.Id,
		int64(blockHeight),
		&nonce,
	)

	req2 := &types.GetTopicLastWorkerCommitInfoRequest{
		TopicId: topic2.Id,
	}

	response2, err := queryServer.GetTopicLastWorkerCommitInfo(ctx, req2)
	s.Require().NoError(err, "GetTopicLastWorkerCommitInfo should not produce an error")
	s.Require().NotNil(response2, "The response should not be nil")
	s.Require().Equal(int64(blockHeight), response2.LastCommit.BlockHeight, "Retrieved blockheight should match")
	s.Require().Equal(&nonce, response2.LastCommit.Nonce, "The metadata of the retrieved nonce should match")
}

func (s *QueryServerTestSuite) TestGetSetDeleteTopicRewardNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test Get on an unset topicId, should return 0
	req := &types.GetTopicRewardNonceRequest{
		TopicId: topicId,
	}
	response, err := s.queryServer.GetTopicRewardNonce(ctx, req)
	nonce := response.Nonce
	s.Require().NoError(err, "Getting an unset topic reward nonce should not fail")
	s.Require().Equal(int64(0), nonce, "Nonce for an unset topicId should be 0")

	// Test Set
	expectedNonce := int64(12345)
	err = keeper.SetTopicRewardNonce(ctx, topicId, expectedNonce)
	s.Require().NoError(err, "Setting topic reward nonce should not fail")

	// Test Get after Set, should return the set value
	response, err = s.queryServer.GetTopicRewardNonce(ctx, req)
	nonce = response.Nonce
	s.Require().NoError(err, "Getting set topic reward nonce should not fail")
	s.Require().Equal(expectedNonce, nonce, "Nonce should match the value set earlier")

	// Test Delete
	err = keeper.DeleteTopicRewardNonce(ctx, topicId)
	s.Require().NoError(err, "Deleting topic reward nonce should not fail")

	// Test Get after Delete, should return 0
	response, err = s.queryServer.GetTopicRewardNonce(ctx, req)
	nonce = response.Nonce
	s.Require().NoError(err, "Getting deleted topic reward nonce should not fail")
	s.Require().Equal(int64(0), nonce, "Nonce should be 0 after deletion")
}

func (s *QueryServerTestSuite) TestGetPreviousTopicWeight() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Set previous topic weight
	weightToSet := alloraMath.NewDecFromInt64(10)
	err := keeper.SetPreviousTopicWeight(ctx, topicId, weightToSet)
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Get the previously set topic weight
	req := &types.GetPreviousTopicWeightRequest{TopicId: topicId}
	response, err := s.queryServer.GetPreviousTopicWeight(ctx, req)
	retrievedWeight := response.Weight

	s.Require().NoError(err, "Getting previous topic weight should not fail")
	s.Require().Equal(weightToSet, retrievedWeight, "Retrieved weight should match the set weight")
}

func (s *QueryServerTestSuite) TestTopicExists() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Test a topic ID that does not exist
	nonExistentTopicId := uint64(999) // Assuming this ID has not been used
	req := &types.TopicExistsRequest{TopicId: nonExistentTopicId}
	response, err := s.queryServer.TopicExists(ctx, req)
	exists := response.Exists
	s.Require().NoError(err, "Checking existence for a non-existent topic should not fail")
	s.Require().False(exists, "No topic should exist for an unused topic ID")

	// Create a topic to test existence
	existentTopicId, err := keeper.IncrementTopicId(ctx)
	s.Require().NoError(err, "Incrementing topic ID should not fail")

	newTopic := types.Topic{Id: existentTopicId}

	err = keeper.SetTopic(ctx, existentTopicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test the newly created topic ID
	req = &types.TopicExistsRequest{TopicId: existentTopicId}
	response, err = s.queryServer.TopicExists(ctx, req)
	exists = response.Exists
	s.Require().NoError(err, "Checking existence for an existent topic should not fail")
	s.Require().True(exists, "Topic should exist for a newly created topic ID")
}

func (s *QueryServerTestSuite) TestIsTopicActive() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(3)

	// Assume topic initially active
	initialTopic := types.Topic{Id: topicId}
	_ = keeper.SetTopic(ctx, topicId, initialTopic)

	// Activate the topic
	err := keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active
	req := &types.IsTopicActiveRequest{TopicId: topicId}
	response, err := s.queryServer.IsTopicActive(ctx, req)
	topicActive := response.IsActive

	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")

	// Inactivate the topic
	err = keeper.InactivateTopic(ctx, topicId)
	s.Require().NoError(err, "Inactivating topic should not fail")

	// Check if topic is inactive
	req = &types.IsTopicActiveRequest{TopicId: topicId}
	response, err = s.queryServer.IsTopicActive(ctx, req)
	topicActive = response.IsActive
	s.Require().NoError(err, "Getting topic should not fail after inactivation")
	s.Require().False(topicActive, "Topic should be inactive")

	// Activate the topic
	err = keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active again
	req = &types.IsTopicActiveRequest{TopicId: topicId}
	response, err = s.queryServer.IsTopicActive(ctx, req)
	topicActive = response.IsActive
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")
}

func (s *QueryServerTestSuite) TestGetTopicFeeRevenue() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	newTopic := types.Topic{Id: topicId}
	err := keeper.SetTopic(ctx, topicId, newTopic)
	s.Require().NoError(err, "Setting a new topic should not fail")

	// Test getting revenue for a topic with no existing revenue
	req := &types.GetTopicFeeRevenueRequest{TopicId: topicId}
	response, err := s.queryServer.GetTopicFeeRevenue(ctx, req)
	feeRev := response.FeeRevenue
	s.Require().NoError(err, "Should not error when revenue does not exist")
	s.Require().Equal(cosmosMath.ZeroInt(), feeRev, "Revenue should be zero for non-existing entries")

	// Setup a topic with some revenue
	initialRevenue := cosmosMath.NewInt(100)
	initialRevenueInt := cosmosMath.NewInt(100)
	err = keeper.AddTopicFeeRevenue(ctx, topicId, initialRevenue)
	s.Require().NoError(err, "Adding revenue should not fail")

	// Test getting revenue for a topic with existing revenue
	req = &types.GetTopicFeeRevenueRequest{TopicId: topicId}
	response, err = s.queryServer.GetTopicFeeRevenue(ctx, req)
	feeRev = response.FeeRevenue
	s.Require().NoError(err, "Should not error when retrieving existing revenue")
	s.Require().Equal(feeRev.String(), initialRevenueInt.String(), "Revenue should match the initial setup")
}
