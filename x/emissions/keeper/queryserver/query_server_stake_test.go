package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetTotalStake() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	expectedTotalStake := cosmosMath.NewInt(1000)
	err := keeper.SetTotalStake(ctx, expectedTotalStake)
	s.Require().NoError(err, "SetTotalStake should not produce an error")

	req := &types.QueryTotalStakeRequest{}
	response, err := queryServer.GetTotalStake(ctx, req)
	s.Require().NoError(err, "GetTotalStake should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(expectedTotalStake, response.Amount, "The retrieved total stake should match the expected value")
}

func (s *KeeperTestSuite) TestGetReputerStakeInTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer, err := sdk.AccAddressFromHexUnsafe(PKS[1].Address().String())
	s.Require().NoError(err)
	reputerAddr := reputer.String()
	initialStake := cosmosMath.NewInt(250)

	err = keeper.AddReputerStake(ctx, topicId, reputerAddr, initialStake)
	s.Require().NoError(err, "AddStake should not produce an error")

	req := &types.QueryReputerStakeInTopicRequest{
		Address: reputerAddr,
		TopicId: topicId,
	}

	response, err := queryServer.GetReputerStakeInTopic(ctx, req)
	s.Require().NoError(err, "GetReputerStakeInTopic should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(initialStake, response.Amount, "The retrieved stake should match the initial stake set for the reputer in the topic")
}

func (s *KeeperTestSuite) TestGetMultiReputerStakeInTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer1, err := sdk.AccAddressFromHexUnsafe(PKS[1].Address().String())
	s.Require().NoError(err)
	reputer2, err := sdk.AccAddressFromHexUnsafe(PKS[2].Address().String())
	s.Require().NoError(err)
	reputer1Addr := reputer1.String()
	reputer2Addr := reputer2.String()
	initialStake1 := cosmosMath.NewInt(250)
	initialStake2 := cosmosMath.NewInt(251)

	err = keeper.AddReputerStake(ctx, topicId, reputer1Addr, initialStake1)
	s.Require().NoError(err, "AddStake should not produce an error")
	err = keeper.AddReputerStake(ctx, topicId, reputer2Addr, initialStake2)
	s.Require().NoError(err, "AddStake should not produce an error")

	req := &types.QueryMultiReputerStakeInTopicRequest{
		Addresses: []string{reputer1Addr, reputer2Addr},
		TopicId:   topicId,
	}

	response, err := queryServer.GetMultiReputerStakeInTopic(ctx, req)
	s.Require().NoError(err, "GetReputerStakeInTopic should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Len(response.Amounts, 2, "The retrieved set of stakes should have length equal to the number of reputers queried")
	s.Require().Equal(initialStake1, response.Amounts[0].Amount, "The retrieved stake should match the initial stake set for the first reputer in the topic")
	s.Require().Equal(initialStake2, response.Amounts[1].Amount, "The retrieved stake should match the initial stake set for the second reputer in the topic")
}

func (s *KeeperTestSuite) TestGetDelegateStakeInTopicInReputer() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegator, err := sdk.AccAddressFromHexUnsafe(PKS[0].Address().String())
	s.Require().NoError(err)
	delegatorAddr := delegator.String()
	reputer, err := sdk.AccAddressFromHexUnsafe(PKS[1].Address().String())
	s.Require().NoError(err)
	reputerAddr := reputer.String()
	initialStakeAmount := cosmosMath.NewInt(1000)

	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err, "AddDelegateStake should not produce an error")

	req := &types.QueryDelegateStakeInTopicInReputerRequest{
		ReputerAddress: reputerAddr,
		TopicId:        topicId,
	}

	response, err := queryServer.GetDelegateStakeInTopicInReputer(ctx, req)
	s.Require().NoError(err, "GetDelegateStakeInTopicInReputer should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(initialStakeAmount, response.Amount, "The retrieved delegate stake should match the initial stake set for the reputer in the topic")
}

func (s *KeeperTestSuite) TestGetStakeFromDelegatorInTopicInReputer() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	delegator, err := sdk.AccAddressFromHexUnsafe(PKS[0].Address().String())
	s.Require().NoError(err)
	delegatorAddr := delegator.String()
	reputer, err := sdk.AccAddressFromHexUnsafe(PKS[1].Address().String())
	s.Require().NoError(err)
	reputerAddr := reputer.String()
	stakeAmount := cosmosMath.NewInt(50)

	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, stakeAmount)
	s.Require().NoError(err, "AddDelegateStake should not produce an error")

	req := &types.QueryStakeFromDelegatorInTopicInReputerRequest{
		DelegatorAddress: delegatorAddr,
		ReputerAddress:   reputerAddr,
		TopicId:          topicId,
	}

	response, err := queryServer.GetStakeFromDelegatorInTopicInReputer(ctx, req)
	s.Require().NoError(err, "GetStakeFromDelegatorInTopicInReputer should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(stakeAmount, response.Amount, "The retrieved stake amount should match the delegated stake")
}

func (s *KeeperTestSuite) TestGetStakeFromDelegatorInTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	delegator, err := sdk.AccAddressFromHexUnsafe(PKS[0].Address().String())
	s.Require().NoError(err)
	delegatorAddr := delegator.String()
	initialStakeAmount := cosmosMath.NewInt(500)
	additionalStakeAmount := cosmosMath.NewInt(300)

	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, PKS[1].Address().String(), initialStakeAmount)
	s.Require().NoError(err)

	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, PKS[1].Address().String(), additionalStakeAmount)
	s.Require().NoError(err)

	req := &types.QueryStakeFromDelegatorInTopicRequest{
		DelegatorAddress: delegatorAddr,
		TopicId:          topicId,
	}

	response, err := queryServer.GetStakeFromDelegatorInTopic(ctx, req)
	s.Require().NoError(err, "GetStakeFromDelegatorInTopic should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	expectedTotalStake := initialStakeAmount.Add(additionalStakeAmount)
	s.Require().Equal(expectedTotalStake, response.Amount, "The retrieved stake amount should match the total delegated stake")
}

func (s *KeeperTestSuite) TestGetTopicStake() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	reputerAddr := PKS[0].Address().String()
	stakeAmount := cosmosMath.NewInt(500)

	err := keeper.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	req := &types.QueryTopicStakeRequest{
		TopicId: topicId,
	}

	response, err := queryServer.GetTopicStake(ctx, req)
	s.Require().NoError(err, "GetTopicStake should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(stakeAmount, response.Amount, "The retrieved topic stake should match the stake amount added")
}

func (s *KeeperTestSuite) TestGetStakeRemovalInfo() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	blockHeight := int64(1234)
	topicId := uint64(1)
	address := sdk.AccAddress(PKS[0].Address()).String()
	removal := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		BlockRemovalCompleted: blockHeight,
		TopicId:               topicId,
		Reputer:               address,
		Amount:                cosmosMath.NewInt(100),
	}
	err := keeper.SetStakeRemoval(ctx, removal)
	s.Require().NoError(err, "SetStakeRemoval should not produce an error")
	req := &types.QueryStakeRemovalInfoRequest{
		TopicId: topicId,
		Reputer: address,
	}
	response, err := queryServer.GetStakeRemovalInfo(ctx, req)
	s.Require().NoError(err, "GetStakeRemovalInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(removal, *response.Removal, "The retrieved stake removal info should match the expected value")
}

func (s *KeeperTestSuite) TestGetDelegateStakeRemovalInfo() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	require := s.Require()
	blockHeight := int64(1234)
	topicId := uint64(1)
	delegatorAddress := sdk.AccAddress(PKS[0].Address()).String()
	reputerAddress := sdk.AccAddress(PKS[1].Address()).String()
	expectedRemoval := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   0,
		BlockRemovalCompleted: blockHeight,
		TopicId:               topicId,
		Delegator:             delegatorAddress,
		Reputer:               reputerAddress,
		Amount:                cosmosMath.NewInt(100),
	}

	err := keeper.SetDelegateStakeRemoval(ctx, expectedRemoval)
	s.Require().NoError(err, "SetStakeRemoval should not produce an error")
	req := &types.QueryDelegateStakeRemovalInfoRequest{
		TopicId:   topicId,
		Delegator: delegatorAddress,
		Reputer:   reputerAddress,
	}
	response, err := queryServer.GetDelegateStakeRemovalInfo(ctx, req)

	require.NoError(err, "GetDelegateStakeRemovalInfo should not produce an error")
	require.NotNil(response, "The response should not be nil")
	require.Equal(&expectedRemoval, response.Removal, "The retrieved stake removal info should match the expected value")
}

func (s *KeeperTestSuite) TestGetStakeRemovalsForBlock() {
	ctx := s.ctx
	qs := s.queryServer
	keeper := s.emissionsKeeper
	require := s.Require()
	blockHeight := int64(1234)
	expecteds := []types.StakeRemovalInfo{
		{
			BlockRemovalStarted:   0,
			BlockRemovalCompleted: blockHeight,
			TopicId:               1,
			Reputer:               sdk.AccAddress(PKS[0].Address()).String(),
			Amount:                cosmosMath.NewInt(100),
		},
		{
			BlockRemovalStarted:   0,
			BlockRemovalCompleted: blockHeight,
			TopicId:               2,
			Reputer:               sdk.AccAddress(PKS[1].Address()).String(),
			Amount:                cosmosMath.NewInt(200),
		},
	}
	for _, e := range expecteds {
		err := keeper.SetStakeRemoval(ctx, e)
		require.NoError(err, "SetStakeRemoval should not produce an error")
	}

	req := &types.QueryStakeRemovalsForBlockRequest{
		BlockHeight: blockHeight,
	}

	response, err := qs.GetStakeRemovalsForBlock(ctx, req)
	require.NoError(err, "GetStakeRemovalsForBlock should not return an error")
	require.NotNil(response, "The response should not be nil")
	require.Len(response.Removals, len(expecteds), "The number of stake removals should match the number of expected stake removals")
	require.Equal(expecteds[0], *response.Removals[0], "The retrieved stake removals should match the expected stake removals")
	require.Equal(expecteds[1], *response.Removals[1], "The retrieved stake removals should match the expected stake removals")
}

func (s *KeeperTestSuite) TestGetDelegateStakeRemovalsForBlock() {
	ctx := s.ctx
	qs := s.queryServer
	keeper := s.emissionsKeeper
	require := s.Require()
	blockHeight := int64(1234)
	expecteds := []types.DelegateStakeRemovalInfo{
		{
			BlockRemovalStarted:   0,
			BlockRemovalCompleted: blockHeight,
			TopicId:               1,
			Reputer:               sdk.AccAddress(PKS[0].Address()).String(),
			Delegator:             sdk.AccAddress(PKS[1].Address()).String(),
			Amount:                cosmosMath.NewInt(100),
		},
		{
			BlockRemovalStarted:   0,
			BlockRemovalCompleted: blockHeight,
			TopicId:               2,
			Reputer:               sdk.AccAddress(PKS[2].Address()).String(),
			Delegator:             sdk.AccAddress(PKS[3].Address()).String(),
			Amount:                cosmosMath.NewInt(200),
		},
	}
	for _, e := range expecteds {
		err := keeper.SetDelegateStakeRemoval(ctx, e)
		require.NoError(err, "SetStakeRemoval should not produce an error")
	}

	req := &types.QueryDelegateStakeRemovalsForBlockRequest{
		BlockHeight: blockHeight,
	}

	response, err := qs.GetDelegateStakeRemovalsForBlock(ctx, req)
	require.NoError(err, "GetStakeRemovalsForBlock should not return an error")
	require.NotNil(response, "The response should not be nil")
	require.Len(response.Removals, len(expecteds), "The number of stake removals should match the number of expected stake removals")
	require.Equal(expecteds[0], *response.Removals[0], "The retrieved stake removals should match the expected stake removals")
	require.Equal(expecteds[1], *response.Removals[1], "The retrieved stake removals should match the expected stake removals")
}
