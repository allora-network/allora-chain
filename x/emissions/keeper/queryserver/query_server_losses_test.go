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
