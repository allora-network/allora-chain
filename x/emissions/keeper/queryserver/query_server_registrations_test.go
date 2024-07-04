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
		LibP2PKey:    reputerKey,
		MultiAddress: "/ip4/127.0.0.1/tcp/4001",
		Owner:        "cosmos1...",
		NodeAddress:  "cosmosNodeAddress",
		NodeId:       "nodeId123",
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

func (s *KeeperTestSuite) TestGetWorkerAddressByP2PKey() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	workerP2pKey := "allo1de602uj38fg4s2rwq7xhh4cmkuvuxfdhkc6jjt"
	nonexistentWorkerP2pKey := "allo19xyq5emxtnt9095t9cr6s556yax8ml6kf5fcdt"

	worker := "allo1chtzkeje04c6n82mgm59vvlgc23fv3knwewsvf"
	workerAcc := worker

	topicId := uint64(401)
	workerInfo := types.OffchainNode{
		LibP2PKey:    workerP2pKey,
		MultiAddress: "worker-multi-address-sample",
		Owner:        worker,
		NodeAddress:  "worker-node-address-sample",
		NodeId:       "worker-node-id-sample",
	}

	err := keeper.InsertWorker(ctx, topicId, workerAcc, workerInfo)
	s.Require().NoError(err, "InsertWorker should not produce an error")

	req := &types.QueryWorkerAddressByP2PKeyRequest{
		Libp2PKey: workerP2pKey,
	}

	response, err := queryServer.GetWorkerAddressByP2PKey(ctx, req)

	s.Require().NoError(err, "GetWorkerAddressByP2PKey should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(worker, response.Address, "The retrieved worker address should match the expected worker address")

	invalidReq := &types.QueryWorkerAddressByP2PKeyRequest{
		Libp2PKey: nonexistentWorkerP2pKey,
	}
	_, err = queryServer.GetWorkerAddressByP2PKey(ctx, invalidReq)
	s.Require().Error(err, "Expected an error for a nonexistent LibP2PKey")
}

func (s *KeeperTestSuite) TestGetReputerAddressByP2PKey() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	reputerP2pKey := "allo1h5tvp00fquwymued2hu7ugrvp2lfscm0f4atp3"
	nonexistentReputerP2pKey := "allo1th6xy9d8gghcps8kyknrk0ry22av0ntjfj3kcc"

	reputer := "allo1cfaw83zzx4rmcq6es0qx9av240xv7hh6jczp4w"
	reputerAcc := reputer

	topicId := uint64(501) // Assuming a different topic ID for reputers
	reputerInfo := types.OffchainNode{
		LibP2PKey:    reputerP2pKey,
		MultiAddress: "reputer-multi-address-sample",
		Owner:        reputer,
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	// Insert the reputer into the keeper
	err := keeper.InsertReputer(ctx, topicId, reputerAcc, reputerInfo)
	s.Require().NoError(err, "InsertReputer should not produce an error")

	// Test valid request
	req := &types.QueryReputerAddressByP2PKeyRequest{
		Libp2PKey: reputerP2pKey,
	}
	response, err := queryServer.GetReputerAddressByP2PKey(ctx, req)

	s.Require().NoError(err, "GetReputerAddressByP2PKey should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(reputer, response.Address, "The retrieved reputer address should match the expected reputer address")

	// Test invalid request
	invalidReq := &types.QueryReputerAddressByP2PKeyRequest{
		Libp2PKey: nonexistentReputerP2pKey,
	}
	_, err = queryServer.GetReputerAddressByP2PKey(ctx, invalidReq)
	s.Require().Error(err, "Expected an error for a nonexistent LibP2PKey")
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
