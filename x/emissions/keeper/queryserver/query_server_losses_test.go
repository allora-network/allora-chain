package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetNetworkLossBundleAtBlock() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)

	// Set up a sample NetworkLossBundle
	expectedBundle := &types.ValueBundle{
		TopicId: topicId,
		Reputer: s.AddrsStr(0),
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{
				BlockHeight: blockHeight,
			},
		},
		ExtraData: []byte("sample_extra_data"),
		// Initialize InfererValues with at least one element to meet the new validation requirement
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				Value:  alloraMath.MustNewDecFromString("1"),
			},
		},
		CombinedValue:                 alloraMath.ZeroDec(),
		NaiveValue:                    alloraMath.ZeroDec(),
		ForecasterValues:              nil,
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}

	err := s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(ctx, topicId, blockHeight, *expectedBundle)
	s.Require().NoError(err)

	response, err := queryServer.GetNetworkLossBundleAtBlock(
		ctx,
		&types.GetNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: blockHeight,
		},
	)

	s.Require().NoError(err)
	s.Require().NotNil(response.LossBundle)
	s.Require().Equal(expectedBundle, response.LossBundle, "Retrieved loss bundle should match the expected bundle")
}

func (s *QueryServerTestSuite) TestIsReputerNonceUnfulfilled() {
	ctx := s.Ctx()
	topicId := uint64(1)
	newNonce := &types.Nonce{BlockHeight: 42}

	req := &types.IsReputerNonceUnfulfilledRequest{
		TopicId:     topicId,
		BlockHeight: newNonce.BlockHeight,
	}
	response, err := s.EmissionsQueryServer().IsReputerNonceUnfulfilled(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().False(response.IsReputerNonceUnfulfilled)

	// Set reputer nonce
	err = s.NonceKeeper().AddReputerNonce(ctx, topicId, newNonce)
	s.Require().NoError(err)

	response, err = s.EmissionsQueryServer().IsReputerNonceUnfulfilled(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().True(response.IsReputerNonceUnfulfilled)
}

func (s *QueryServerTestSuite) TestGetUnfulfilledReputerNonces() {
	ctx := s.Ctx()
	topicId := uint64(1)

	// Initially, ensure no unfulfilled nonces exist
	req := &types.GetUnfulfilledReputerNoncesRequest{
		TopicId: topicId,
	}
	response, err := s.EmissionsQueryServer().GetUnfulfilledReputerNonces(s.Ctx(), req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "Response should not be nil")
	s.Require().Len(response.Nonces.Nonces, 0, "Initial unfulfilled nonces should be empty")

	// Set multiple reputer nonces
	nonceValues := []int64{42, 43, 44}
	for _, val := range nonceValues {
		err = s.NonceKeeper().AddReputerNonce(ctx, topicId, &types.Nonce{BlockHeight: val})
		s.Require().NoError(err, "Failed to add reputer nonce")
	}

	// Retrieve and verify the nonces
	response, err = s.EmissionsQueryServer().GetUnfulfilledReputerNonces(s.Ctx(), req)
	s.Require().NoError(err, "Error retrieving nonces after adding")
	s.Require().Len(response.Nonces.Nonces, len(nonceValues), "Should match the number of added nonces")

	// Check that all the expected nonces are present and correct
	for i, nonce := range response.Nonces.Nonces {
		s.Require().Equal(nonceValues[len(nonceValues)-i-1], nonce.ReputerNonce.BlockHeight, "Nonce value should match the expected value")
	}
}

func (s *QueryServerTestSuite) TestGetReputerLossBundlesAtBlock() {
	ctx := s.Ctx()
	require := s.Require()
	topicId := uint64(1)
	block := types.BlockHeight(100)
	//nolint:exhaustruct
	valueBundle := &types.ValueBundle{
		TopicId:             topicId,
		Reputer:             s.AddrsStr(0),
		ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: block}},
		ExtraData:           []byte("sample_extra_data"),
		CombinedValue:       alloraMath.NewDecFromInt64(100),
		NaiveValue:          alloraMath.NewDecFromInt64(100),
		// Initialize InfererValues with at least one element to meet the new validation requirement
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
				Value:  alloraMath.MustNewDecFromString("1"),
			},
		},
	}
	reputerLossBundles := types.LossBundles{valueBundle}
	req := &types.GetReputerLossBundlesAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: block,
	}
	response, err := s.EmissionsQueryServer().GetReputerLossBundlesAtBlock(ctx, req)
	require.NoError(err)
	require.Empty(response.LossBundles)

	// Test inserting data
	err = s.ReputerLossKeeper().InsertActiveReputerLosses(ctx, topicId, block, reputerLossBundles)
	require.NoError(err, "InsertActiveReputerLosses should not return an error")

	response, err = s.EmissionsQueryServer().GetReputerLossBundlesAtBlock(ctx, req)
	require.NotEmpty(response)
	require.NoError(err)

	result := response.GetLossBundles()
	require.NotEmpty(result)
	require.Equal(reputerLossBundles, types.LossBundles(result), "Retrieved data should match inserted data")
}

func (s *QueryServerTestSuite) TestGetDeleteDelegateStake() {
	ctx := s.Ctx()

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
	err := s.StakingKeeper().SetDelegateStakeRemoval(ctx, removalInfo)
	s.Require().NoError(err)

	req := &types.GetDelegateStakeRemovalRequest{
		BlockHeight: removalInfo.BlockRemovalStarted,
		TopicId:     removalInfo.TopicId,
		Reputer:     removalInfo.Reputer,
		Delegator:   removalInfo.Delegator,
	}
	response, err := s.EmissionsQueryServer().GetDelegateStakeRemoval(ctx, req)
	s.Require().Error(err)
	s.Require().Nil(response)

	req.BlockHeight = removalInfo.BlockRemovalCompleted

	response, err = s.EmissionsQueryServer().GetDelegateStakeRemoval(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	retrievedInfo := response.StakeRemovalInfo

	s.Require().Equal(removalInfo.BlockRemovalStarted, retrievedInfo.BlockRemovalStarted)
	s.Require().Equal(removalInfo.TopicId, retrievedInfo.TopicId)
	s.Require().Equal(removalInfo.Reputer, retrievedInfo.Reputer)
	s.Require().Equal(removalInfo.Delegator, retrievedInfo.Delegator)
	s.Require().Equal(removalInfo.Amount, retrievedInfo.Amount)
}
