package queryserver_test

import (
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
