package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
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
			BlockHeight: blockHeight,
		},
	)

	s.Require().NoError(err)
	s.Require().NotNil(response.LossBundle)
	s.Require().Equal(expectedBundle, response.LossBundle, "Retrieved loss bundle should match the expected bundle")
}

func (s *KeeperTestSuite) TestIsReputerNonceUnfulfilled() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	req := &types.QueryIsReputerNonceUnfulfilledRequest{
		TopicId:     topicId,
		BlockHeight: newNonce.BlockHeight,
	}
	response, err := s.queryServer.IsReputerNonceUnfulfilled(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().False(response.IsReputerNonceUnfulfilled)

	// Set reputer nonce
	err = keeper.AddReputerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	response, err = s.queryServer.IsReputerNonceUnfulfilled(s.ctx, req)
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
		err = keeper.AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
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
	reputerLossBundle := types.ReputerValueBundle{
		ValueBundle: &types.ValueBundle{
			TopicId:                topicId,
			ReputerRequestNonce:    nil,
			Reputer:                "reputer1",
			ExtraData:              []byte{},
			CombinedValue:          alloraMath.ZeroDec(),
			InfererValues:          nil,
			ForecasterValues:       nil,
			NaiveValue:             alloraMath.ZeroDec(),
			OneOutInfererValues:    nil,
			OneInForecasterValues:  nil,
			OneOutForecasterValues: nil,
		},
		Signature: []byte{},
		Pubkey:    "",
	}
	reputerLossBundles := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			&reputerLossBundle,
		},
	}

	req := &types.QueryReputerLossBundlesAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: block,
	}
	response, err := s.queryServer.GetReputerLossBundlesAtBlock(ctx, req)
	require.NoError(err)
	require.Nil(response.LossBundles.ReputerValueBundles)

	// Test inserting data
	err = s.emissionsKeeper.ReplaceReputerValueBundles(
		ctx, topicId, types.Nonce{BlockHeight: block}, reputerLossBundles, 0, reputerLossBundle)
	require.NoError(err, "ReplaceReputerValueBundles should not return an error")

	response, err = s.queryServer.GetReputerLossBundlesAtBlock(ctx, req)
	require.NotNil(response)
	require.NoError(err)

	result := response.LossBundles
	require.NotNil(result)
	require.Equal(&reputerLossBundles, result, "Retrieved data should match inserted data")
}

func (s *KeeperTestSuite) TestGetDeleteDelegateStake() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

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
	err := keeper.SetDelegateStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	req := &types.QueryDelegateStakeRemovalRequest{
		BlockHeight: removalInfo.BlockRemovalStarted,
		TopicId:     removalInfo.TopicId,
		Reputer:     removalInfo.Reputer,
		Delegator:   removalInfo.Delegator,
	}
	response, err := s.queryServer.GetDelegateStakeRemoval(ctx, req)
	s.Require().Error(err)
	s.Require().Nil(response)

	req.BlockHeight = removalInfo.BlockRemovalCompleted

	response, err = s.queryServer.GetDelegateStakeRemoval(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	retrievedInfo := response.StakeRemovalInfo

	s.Require().Equal(removalInfo.BlockRemovalStarted, retrievedInfo.BlockRemovalStarted)
	s.Require().Equal(removalInfo.TopicId, retrievedInfo.TopicId)
	s.Require().Equal(removalInfo.Reputer, retrievedInfo.Reputer)
	s.Require().Equal(removalInfo.Delegator, retrievedInfo.Delegator)
	s.Require().Equal(removalInfo.Amount, retrievedInfo.Amount)
}
