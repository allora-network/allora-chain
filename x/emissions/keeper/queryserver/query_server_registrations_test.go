package queryserver_test

import (
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetWorkerNodeInfo() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer

	libP2PKey := "worker-libp2p-key-sample"

	expectedNode := types.OffchainNode{
		LibP2PKey:    libP2PKey,
		MultiAddress: "worker-multi-address-sample",
		Owner:        "worker-owner-sample",
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
	}

	worker := "sampleWorkerAddress"
	topicId := uint64(401)
	err := keeper.InsertWorker(ctx, topicId, worker, expectedNode)
	s.Require().NoError(err, "InsertWorker should not produce an error")

	req := &types.QueryWorkerNodeInfoRequest{
		Libp2PKey: libP2PKey,
	}

	response, err := queryServer.GetWorkerNodeInfo(ctx, req)

	s.Require().NoError(err, "GetWorkerNodeInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().NotNil(response.NodeInfo, "The response NodeInfo should not be nil")
	s.Require().Equal(&expectedNode, response.NodeInfo, "The retrieved node information should match the expected node information")

	invalidReq := &types.QueryWorkerNodeInfoRequest{
		Libp2PKey: "nonexistent-libp2p-key",
	}
	_, err = queryServer.GetWorkerNodeInfo(ctx, invalidReq)
	s.Require().Error(err, "Expected an error for nonexistent LibP2PKey")
}

func (s *KeeperTestSuite) TestGetReputerNodeInfo() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	reputerKey := "someLibP2PKey123"

	expectedReputer := types.OffchainNode{
		NodeAddress: "cosmosNodeAddress",
		Owner:       "cosmos1...",
	}

	reputer := "sampleReputerAddress"
	topicId := uint64(501)
	err := keeper.InsertReputer(ctx, topicId, reputer, expectedReputer)
	s.Require().NoError(err, "InsertReputer should not produce an error")

	req := &types.QueryReputerNodeInfoRequest{
		Libp2PKey: reputerKey,
	}

	response, err := queryServer.GetReputerNodeInfo(ctx, req)

	s.Require().NoError(err, "GetReputerNodeInfo should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().NotNil(response.NodeInfo, "The response NodeInfo should not be nil")
	s.Require().Equal(&expectedReputer, response.NodeInfo, "The retrieved node information should match the expected node information")

	invalidReq := &types.QueryReputerNodeInfoRequest{
		Libp2PKey: "nonExistentKey123",
	}
	_, err = queryServer.GetReputerNodeInfo(ctx, invalidReq)
	s.Require().Error(err, "Expected an error for nonexistent LibP2PKey")
}

func (s *KeeperTestSuite) TestUnregisteredWorkerIsUnregisteredInTopicId() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryServer := s.queryServer

	notRegisteredWorkerAddr := "allo12gjf2mrtva0p33gqtvsxp37zgglmdgpwaq22m2"

	// Test: Worker is not registered under the topic
	notRegisteredRequest := &types.QueryIsWorkerRegisteredInTopicIdRequest{
		TopicId: uint64(1),
		Address: notRegisteredWorkerAddr,
	}
	invalidResponse, err := queryServer.IsWorkerRegisteredInTopicId(ctx, notRegisteredRequest)
	s.Require().NoError(err, "IsWorkerRegisteredInTopicId should handle non-registered addresses without error")
	s.Require().NotNil(invalidResponse, "The response for non-registered worker should not be nil")
	s.Require().False(invalidResponse.IsRegistered, "The worker should not be registered for the topic")
}

func (s *KeeperTestSuite) TestRegisteredWorkerIsRegisteredInTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address())
	workerAddrString := workerAddr.String()
	creatorAddress := sdk.AccAddress(PKS[1].Address()).String()
	topicId := uint64(1)
	topic1 := types.Topic{Id: topicId, Creator: creatorAddress}

	// Topic register
	s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, topicId)
	// Worker register
	registerMsg := &types.MsgRegister{
		Sender:       workerAddrString,
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    false,
		Owner:        workerAddrString,
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	err := s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		workerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	queryReq := &types.QueryIsWorkerRegisteredInTopicIdRequest{
		Address: workerAddrString,
		TopicId: topicId,
	}
	queryResp, err := s.queryServer.IsWorkerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "QueryIsWorkerRegisteredInTopicId should not return an error")
	require.False(queryResp.IsRegistered, "Query response should confirm worker is registered")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	queryResp, err = s.queryServer.IsWorkerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "QueryIsWorkerRegisteredInTopicId should not return an error")
	require.True(queryResp.IsRegistered, "Query response should confirm worker is registered")
}

func (s *KeeperTestSuite) TestRegisteredReputerIsRegisteredInTopicId() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	creatorAddress := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(1)
	topic1 := types.Topic{Id: topicId, Creator: creatorAddress.String()}

	// Topic register
	s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, topicId)
	// Register reputer
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputerAddr.String(),
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	err := s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, reputerAddr, mintAmount)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	queryReq := &types.QueryIsReputerRegisteredInTopicIdRequest{
		Address: reputerAddr.String(),
		TopicId: topicId,
	}
	queryResp, err := s.queryServer.IsReputerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "QueryIsReputerRegisteredInTopicId should not return an error")
	require.False(queryResp.IsRegistered, "Query response should confirm reputer is registered")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	queryResp, err = s.queryServer.IsReputerRegisteredInTopicId(ctx, queryReq)
	require.NoError(err, "QueryIsReputerRegisteredInTopicId should not return an error")
	require.True(queryResp.IsRegistered, "Query response should confirm reputer is registered")
}
