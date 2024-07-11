package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetNextTopicId() {
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

	req := &types.QueryNextTopicIdRequest{}

	response, err := queryServer.GetNextTopicId(ctx, req)
	s.Require().NoError(err, "GetNextTopicId should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	expectedNextTopicId := initialNextTopicId + uint64(topicsToCreate)
	s.Require().Equal(expectedNextTopicId, response.NextTopicId, "The next topic ID should match the expected value after topic creation")
}

func (s *KeeperTestSuite) TestGetTopic() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId, err := keeper.GetNextTopicId(ctx)
	s.Require().NoError(err)
	metadata := "metadata"
	req := &types.QueryTopicRequest{TopicId: topicId}

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

func (s *KeeperTestSuite) TestGetActiveTopics() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topic1 := types.Topic{Id: 1}
	topic2 := types.Topic{Id: 2}
	topic3 := types.Topic{Id: 3}

	_ = keeper.SetTopic(ctx, topic1.Id, topic1)
	_ = keeper.ActivateTopic(ctx, topic1.Id)
	_ = keeper.SetTopic(ctx, topic2.Id, topic2) // Inactive topic
	_ = keeper.SetTopic(ctx, topic3.Id, topic3)
	_ = keeper.ActivateTopic(ctx, topic3.Id)

	req := &types.QueryActiveTopicsRequest{
		Pagination: &types.SimpleCursorPaginationRequest{
			Key:   nil,
			Limit: 10,
		},
	}

	response, err := queryServer.GetActiveTopics(ctx, req)
	s.Require().NoError(err, "GetActiveTopics should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(len(response.Topics), 2, "Should retrieve exactly two active topics")

	for _, topic := range response.Topics {
		s.Require().True(topic.Id == 1 || topic.Id == 3, "Only active topic IDs (1 or 3) should be returned")
		isActive, err := keeper.IsTopicActive(ctx, topic.Id)
		s.Require().NoError(err, "Checking topic activity should not fail")
		s.Require().True(isActive, "Only active topics should be returned")
	}
}

func (s *KeeperTestSuite) TestGetLatestCommit() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	blockHeight := 100
	nonce := types.Nonce{
		BlockHeight: 95,
	}
	actor := "TestReputer"

	topic := types.Topic{Id: 1}
	_ = keeper.SetTopicLastCommit(
		ctx,
		topic.Id,
		int64(blockHeight),
		&nonce,
		actor,
		types.ActorType_REPUTER,
	)

	req := &types.QueryTopicLastCommitRequest{
		TopicId: topic.Id,
	}

	response, err := queryServer.GetTopicLastReputerCommitInfo(ctx, req)
	s.Require().NoError(err, "GetActiveTopics should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(int64(blockHeight), response.LastCommit.BlockHeight, "Retrieved blockheight should match")
	s.Require().Equal(&nonce, response.LastCommit.Nonce, "The metadata of the retrieved nonce should match")
	s.Require().Equal(actor, response.LastCommit.Actor, "The metadata of the retrieved nonce should match")

	topic2 := types.Topic{Id: 2}
	blockHeight = 101
	nonce = types.Nonce{
		BlockHeight: 98,
	}
	actor = "TestWorker"

	_ = keeper.SetTopicLastCommit(
		ctx,
		topic2.Id,
		int64(blockHeight),
		&nonce,
		actor,
		types.ActorType_INFERER,
	)

	req2 := &types.QueryTopicLastCommitRequest{
		TopicId: topic2.Id,
	}

	response2, err := queryServer.GetTopicLastWorkerCommitInfo(ctx, req2)
	s.Require().NoError(err, "GetActiveTopics should not produce an error")
	s.Require().NotNil(response2, "The response should not be nil")
	s.Require().Equal(int64(blockHeight), response2.LastCommit.BlockHeight, "Retrieved blockheight should match")
	s.Require().Equal(&nonce, response2.LastCommit.Nonce, "The metadata of the retrieved nonce should match")
	s.Require().Equal(actor, response2.LastCommit.Actor, "The metadata of the retrieved nonce should match")
}

func (s *KeeperTestSuite) TestGetSetDeleteTopicRewardNonce() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Test Get on an unset topicId, should return 0
	req := &types.QueryTopicRewardNonceRequest{
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

func (s *KeeperTestSuite) TestGetPreviousTopicWeight() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Set previous topic weight
	weightToSet := alloraMath.NewDecFromInt64(10)
	err := keeper.SetPreviousTopicWeight(ctx, topicId, weightToSet)
	s.Require().NoError(err, "Setting previous topic weight should not fail")

	// Get the previously set topic weight
	req := &types.QueryPreviousTopicWeightRequest{TopicId: topicId}
	response, err := s.queryServer.GetPreviousTopicWeight(ctx, req)
	retrievedWeight := response.Weight

	s.Require().NoError(err, "Getting previous topic weight should not fail")
	s.Require().Equal(weightToSet, retrievedWeight, "Retrieved weight should match the set weight")
}

func (s *KeeperTestSuite) TestTopicExists() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	// Test a topic ID that does not exist
	nonExistentTopicId := uint64(999) // Assuming this ID has not been used
	req := &types.QueryTopicExistsRequest{TopicId: nonExistentTopicId}
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
	req = &types.QueryTopicExistsRequest{TopicId: existentTopicId}
	response, err = s.queryServer.TopicExists(ctx, req)
	exists = response.Exists
	s.Require().NoError(err, "Checking existence for an existent topic should not fail")
	s.Require().True(exists, "Topic should exist for a newly created topic ID")
}

func (s *KeeperTestSuite) TestIsTopicActive() {
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
	req := &types.QueryIsTopicActiveRequest{TopicId: topicId}
	response, err := s.queryServer.IsTopicActive(ctx, req)
	topicActive := response.IsActive

	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")

	// Inactivate the topic
	err = keeper.InactivateTopic(ctx, topicId)
	s.Require().NoError(err, "Inactivating topic should not fail")

	// Check if topic is inactive
	req = &types.QueryIsTopicActiveRequest{TopicId: topicId}
	response, err = s.queryServer.IsTopicActive(ctx, req)
	topicActive = response.IsActive
	s.Require().NoError(err, "Getting topic should not fail after inactivation")
	s.Require().False(topicActive, "Topic should be inactive")

	// Activate the topic
	err = keeper.ActivateTopic(ctx, topicId)
	s.Require().NoError(err, "Reactivating topic should not fail")

	// Check if topic is active again
	req = &types.QueryIsTopicActiveRequest{TopicId: topicId}
	response, err = s.queryServer.IsTopicActive(ctx, req)
	topicActive = response.IsActive
	s.Require().NoError(err, "Getting topic should not fail after reactivation")
	s.Require().True(topicActive, "Topic should be active again")
}

func (s *KeeperTestSuite) TestGetIdsOfActiveTopics() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic1 := types.Topic{Id: 1}
	topic2 := types.Topic{Id: 2}
	topic3 := types.Topic{Id: 3}

	_ = keeper.SetTopic(ctx, topic1.Id, topic1)
	_ = keeper.ActivateTopic(ctx, topic1.Id)
	_ = keeper.SetTopic(ctx, topic2.Id, topic2) // Inactive topic
	_ = keeper.SetTopic(ctx, topic3.Id, topic3)
	_ = keeper.ActivateTopic(ctx, topic3.Id)

	// Fetch only active topics
	pagination := &types.SimpleCursorPaginationRequest{
		Key:   nil,
		Limit: 10,
	}
	req := &types.QueryIdsOfActiveTopicsRequest{Pagination: pagination}
	response, err := s.queryServer.GetIdsOfActiveTopics(ctx, req)
	activeTopics := response.ActiveTopicIds
	s.Require().NoError(err, "Fetching active topics should not produce an error")

	s.Require().Equal(2, len(activeTopics), "Should retrieve exactly two active topics")

	for _, topicId := range activeTopics {
		isActive, err := keeper.IsTopicActive(ctx, topicId)
		s.Require().NoError(err, "Checking topic activity should not fail")
		s.Require().True(isActive, "Only active topics should be returned")
		switch topicId {
		case 1:
			s.Require().Equal(topic1.Id, topicId, "The details of topic 1 should match")
		case 3:
			s.Require().Equal(topic3.Id, topicId, "The details of topic 3 should match")
		default:
			s.Fail("Unexpected topic ID retrieved")
		}
	}
}

func (s *KeeperTestSuite) TestGetTopicEpochLastEnded() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	epochLastEnded := types.BlockHeight(100)

	// Setup a topic initially
	initialTopic := types.Topic{Id: topicId}
	_ = keeper.SetTopic(ctx, topicId, initialTopic)

	// Update the epoch last ended
	err := keeper.UpdateTopicEpochLastEnded(ctx, topicId, epochLastEnded)
	s.Require().NoError(err, "Updating topic epoch last ended should not fail")

	// Retrieve the last ended epoch for the topic
	req := &types.QueryTopicEpochLastEndedRequest{TopicId: topicId}
	response, err := s.queryServer.GetTopicEpochLastEnded(ctx, req)
	retrievedEpoch := response.EpochLastEnded
	s.Require().NoError(err, "Retrieving topic epoch last ended should not fail")
	s.Require().Equal(epochLastEnded, retrievedEpoch, "The retrieved epoch last ended should match the updated value")
}
