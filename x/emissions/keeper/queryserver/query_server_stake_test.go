package queryserver_test

import (
	"strconv"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
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

func (s *KeeperTestSuite) TestGetStakeReputerAuthority() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := PKS[0].Address().String()
	stakeAmount := cosmosMath.NewInt(500)

	// Add stake
	err := keeper.AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	req := &types.QueryStakeReputerAuthorityRequest{
		TopicId: topicId,
		Reputer: reputerAddr,
	}
	response, err := s.queryServer.GetStakeReputerAuthority(ctx, req)
	stakeAuthority := response.Authority

	s.Require().NoError(err)
	s.Require().Equal(stakeAmount, stakeAuthority, "Delegator stake should be equal to stake amount after addition")
}

func (s *KeeperTestSuite) TestGetDelegateStakePlacement() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)

	msg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	reputerStake, err := s.emissionsKeeper.GetStakeReputerAuthority(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), reputerStake, "Stake amount mismatch")

	req := &types.QueryDelegateStakePlacementRequest{
		TopicId:   topicId,
		Delegator: delegatorAddr.String(),
		Target:    reputerAddr.String(),
	}
	queryResponse, err := s.queryServer.GetDelegateStakePlacement(ctx, req)
	require.NoError(err)
	require.Equal(alloraMath.NewDecFromInt64(0), queryResponse.DelegatorInfo.Amount)

	// Perform the stake delegation
	response, err := s.msgServer.DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	reputerStake, err = s.emissionsKeeper.GetStakeReputerAuthority(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	queryResponse, err = s.queryServer.GetDelegateStakePlacement(ctx, req)
	require.NoError(err)

	value0, err := strconv.ParseFloat(stakeAmount.ToLegacyDec().String(), 64)
	require.NoError(err)

	value1, err := strconv.ParseFloat(queryResponse.DelegatorInfo.Amount.String(), 64)
	require.NoError(err)

	require.Equal(value0, value1)
}

func (s *KeeperTestSuite) TestGetDelegateStakeUponReputer() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := PKS[0].Address().String()
	reputerAddr := PKS[1].Address().String()
	initialStakeAmount := cosmosMath.NewInt(1000)
	removeStakeAmount := cosmosMath.NewInt(500)
	moduleParams, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + moduleParams.RemoveStakeDelayWindow

	// Setup initial stake
	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err)

	// make a request to remove stake
	err = keeper.SetDelegateStakeRemoval(ctx, types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
		TopicId:               topicId,
		Delegator:             delegatorAddr,
		Reputer:               reputerAddr,
		Amount:                removeStakeAmount,
	})
	s.Require().NoError(err)

	// Remove a portion of stake
	err = keeper.RemoveDelegateStake(ctx, endBlock, topicId, delegatorAddr, reputerAddr, removeStakeAmount)
	s.Require().NoError(err)

	// Check remaining stake for delegator
	remainingStake, err := keeper.GetStakeFromDelegatorInTopic(ctx, topicId, delegatorAddr)
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), remainingStake, "Remaining delegator stake should be initial minus removed amount")

	// Check remaining stake for delegator
	stakeUponReputer, err := keeper.GetDelegateStakeUponReputer(ctx, topicId, reputerAddr)
	req := &types.QueryDelegateStakeUponReputerRequest{
		TopicId: topicId,
		Target:  reputerAddr,
	}
	queryResponse, err := s.queryServer.GetDelegateStakeUponReputer(ctx, req)
	stakeUponReputer = queryResponse.Stake
	s.Require().NoError(err)
	s.Require().Equal(initialStakeAmount.Sub(removeStakeAmount), stakeUponReputer, "Remaining reputer stake should be initial minus removed amount")
}

func (s *KeeperTestSuite) TestGetStakeRemovalForReputerAndTopicId() {
	k := s.emissionsKeeper
	ctx := s.ctx
	reputer := "reputer"
	topicId := uint64(1)

	// Create a stake removal info
	stakeRemovalInfo := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               reputer,
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(100),
		BlockRemovalCompleted: 30,
	}
	anotherStakeRemoval := types.StakeRemovalInfo{
		BlockRemovalStarted:   0,
		Reputer:               "reputer2",
		TopicId:               topicId,
		Amount:                cosmosMath.NewInt(200),
		BlockRemovalCompleted: 30,
	}

	// Set the stake removal info in the keeper
	err := k.SetStakeRemoval(ctx, stakeRemovalInfo)
	s.Require().NoError(err)
	err = k.SetStakeRemoval(ctx, anotherStakeRemoval)
	s.Require().NoError(err)

	req := &types.QueryStakeRemovalForReputerAndTopicIdRequest{
		Reputer: reputer,
		TopicId: topicId,
	}
	response, err := s.queryServer.GetStakeRemovalForReputerAndTopicId(ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(&stakeRemovalInfo, response.StakeRemovalInfo)
}
