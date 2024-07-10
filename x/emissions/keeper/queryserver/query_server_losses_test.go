package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetNetworkLossBundleAtBlock() {
	s.CreateOneTopic()
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)

	// Set up a sample NetworkLossBundle
	expectedBundle := &types.ValueBundle{
		TopicId:   topicId,
		Reputer:   "sample_reputer",
		ExtraData: []byte("sample_extra_data"),
	}

	err := keeper.InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight, *expectedBundle)
	s.Require().NoError(err)

	response, err := queryServer.GetNetworkLossBundleAtBlock(
		ctx,
		&types.QueryNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: int64(blockHeight),
		},
	)

	s.Require().NoError(err)
	s.Require().NotNil(response.LossBundle)
	s.Require().Equal(expectedBundle, response.LossBundle, "Retrieved loss bundle should match the expected bundle")
}

func (s *KeeperTestSuite) TestGetIsReputerNonceUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	req := &types.QueryIsReputerNonceUnfulfilledRequest{
		TopicId:     topicId,
		BlockHeight: newNonce.BlockHeight,
	}
	response, err := s.queryServer.GetIsReputerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().False(response.IsReputerNonceUnfulfilled)

	// Set reputer nonce
	err = keeper.AddReputerNonce(ctx, topicId, newNonce, newNonce)
	s.Require().NoError(err)

	response, err = s.queryServer.GetIsReputerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().True(response.IsReputerNonceUnfulfilled)
}

func (s *KeeperTestSuite) TestGetUnfulfilledReputerNonces() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	req := &types.QueryUnfulfilledReputerNoncesRequest{
		TopicId: topicId,
	}
	response, err := s.queryServer.GetUnfulfilledReputerNonces(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().Len(response.Nonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple reputer nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = keeper.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val}, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Retrieve and verify the nonces
	response, err = s.queryServer.GetUnfulfilledReputerNonces(s.ctx, req)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(response.Nonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range response.Nonces.Nonces {
		s.Require().Equal(nonceValues[len(nonceValues)-i-1], nonce.ReputerNonce.BlockHeight, "Nonce value should match the expected value")
	}
}

func (s *KeeperTestSuite) TestGetReputerLossBundlesAtBlock() {
	ctx := s.ctx
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	reputerLossBundles := types.ReputerValueBundles{}

	req := &types.QueryReputerLossBundlesAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: int64(block),
	}
	response, err := s.queryServer.GetReputerLossBundlesAtBlock(ctx, req)
	require.Error(err)
	require.Nil(response)

	// Test inserting data
	err = s.emissionsKeeper.InsertReputerLossBundlesAtBlock(ctx, topicId, block, reputerLossBundles)
	require.NoError(err, "InsertReputerLossBundlesAtBlock should not return an error")

	response, err = s.queryServer.GetReputerLossBundlesAtBlock(ctx, req)
	require.NotNil(response)
	require.NoError(err)

	result := response.LossBundles
	require.NotNil(result)
	require.Equal(&reputerLossBundles, result, "Retrieved data should match inserted data")
}
