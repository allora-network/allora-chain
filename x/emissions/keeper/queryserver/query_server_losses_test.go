package queryserver_test

import "github.com/allora-network/allora-chain/x/emissions/types"

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

func (s *KeeperTestSuite) TestNewlyAddedReputerNonceIsUnfulfilled() {
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
