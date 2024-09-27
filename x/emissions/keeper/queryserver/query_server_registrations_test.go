package queryserver_test

import (
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *QueryServerTestSuite) TestGetWorkerNodeInfo() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	worker := s.addrsStr[0]

	expectedNode := types.OffchainNode{
		Owner:       s.addrsStr[1],
		NodeAddress: worker,
	}

	topicId := uint64(401)
	err := keeper.InsertWorker(ctx, topicId, worker, expectedNode)
	s.Require().NoError(err, "InsertWorker should not produce an error")

	req := &types.GetWorkerNodeInfoRequest{
		Address: worker,
	}

	response, err := queryServer.GetWorkerNodeInfo(ctx, req)

	s.Require().NoError(err, "GetWorkerNodeInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().NotNil(response.NodeInfo, "The response NodeInfo should not be nil")
	s.Require().Equal(&expectedNode, response.NodeInfo, "The retrieved node information should match the expected node information")

	invalidReq := &types.GetWorkerNodeInfoRequest{
		Address: "nonexistent-key",
	}
	_, err = queryServer.GetWorkerNodeInfo(ctx, invalidReq)
	s.Require().Error(err, "Expected an error for nonexistent key")
}

func (s *QueryServerTestSuite) TestGetReputerNodeInfo() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer

	reputer := s.addrsStr[1]
	expectedReputer := types.OffchainNode{
		NodeAddress: s.addrsStr[0],
		Owner:       reputer,
	}

	topicId := uint64(501)
	err := keeper.InsertReputer(ctx, topicId, reputer, expectedReputer)
	s.Require().NoError(err, "InsertReputer should not produce an error")

	req := &types.GetReputerNodeInfoRequest{
		Address: reputer,
	}

	response, err := queryServer.GetReputerNodeInfo(ctx, req)

	s.Require().NoError(err, "GetReputerNodeInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().NotNil(response.NodeInfo, "The response NodeInfo should not be nil")
	s.Require().Equal(&expectedReputer, response.NodeInfo, "The retrieved node information should match the expected node information")

	invalidReq := &types.GetReputerNodeInfoRequest{
		Address: "nonExistentKey123",
	}
	_, err = queryServer.GetReputerNodeInfo(ctx, invalidReq)
	s.Require().Error(err, "Expected an error for nonexistent key")
}

func (s *QueryServerTestSuite) TestUnregisteredWorkerIsUnregisteredInTopicId() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer

	notRegisteredWorkerAddr := "allo12gjf2mrtva0p33gqtvsxp37zgglmdgpwaq22m2"

	// Test: Worker is not registered under the topic
	notRegisteredRequest := &types.IsWorkerRegisteredInTopicIdRequest{
		TopicId: uint64(1),
		Address: notRegisteredWorkerAddr,
	}
	invalidResponse, err := queryServer.IsWorkerRegisteredInTopicId(ctx, notRegisteredRequest)
	s.Require().NoError(err, "IsWorkerRegisteredInTopicId should handle non-registered addresses without error")
	s.Require().NotNil(invalidResponse, "The response for non-registered worker should not be nil")
	s.Require().False(invalidResponse.IsRegistered, "The worker should not be registered for the topic")
}

func (s *QueryServerTestSuite) TestRegisteredWorkerIsRegisteredInTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := s.addrs[1]
	workerAddrString := workerAddr.String()
	topicId := uint64(1)
	topic1 := s.mockTopic()

	// Topic register
	err := s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	require.NoError(err, "SetTopic should not return an error")
	err = s.emissionsKeeper.ActivateTopic(ctx, topicId)
	require.NoError(err, "ActivateTopic should not return an error")
	// Worker register
	registerMsg := &types.RegisterRequest{
		Sender:    workerAddrString,
		TopicId:   topicId,
		IsReputer: false,
		Owner:     workerAddrString,
	}

	moduleParams, err := s.emissionsKeeper.GetParams(ctx)
	s.Require().NoError(err)
	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		workerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	queryReq := &types.IsWorkerRegisteredInTopicIdRequest{
		Address: workerAddrString,
		TopicId: topicId,
	}
	queryResp, err := s.queryServer.IsWorkerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "IsWorkerRegisteredInTopicId should not return an error")
	require.False(queryResp.IsRegistered, "Query response should confirm worker is registered")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	queryResp, err = s.queryServer.IsWorkerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "IsWorkerRegisteredInTopicId should not return an error")
	require.True(queryResp.IsRegistered, "Query response should confirm worker is registered")
}

func (s *QueryServerTestSuite) TestRegisteredReputerIsRegisteredInTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := s.addrs[2]
	topicId := uint64(1)
	topic1 := s.mockTopic()

	// Topic register
	err := s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	require.NoError(err, "SetTopic should not return an error")
	err = s.emissionsKeeper.ActivateTopic(ctx, topicId)
	require.NoError(err, "ActivateTopic should not return an error")
	// Register reputer
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topicId,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	moduleParams, err := s.emissionsKeeper.GetParams(ctx)
	s.Require().NoError(err)
	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, reputerAddr, mintAmount)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	queryReq := &types.IsReputerRegisteredInTopicIdRequest{
		Address: reputerAddr.String(),
		TopicId: topicId,
	}
	queryResp, err := s.queryServer.IsReputerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "IsReputerRegisteredInTopicId should not return an error")
	require.False(queryResp.IsRegistered, "Query response should confirm reputer is registered")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	queryResp, err = s.queryServer.IsReputerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "IsReputerRegisteredInTopicId should not return an error")
	require.True(queryResp.IsRegistered, "Query response should confirm reputer is registered")
}
