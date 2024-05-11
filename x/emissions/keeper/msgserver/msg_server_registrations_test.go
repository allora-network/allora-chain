package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgRegisterReputer() {
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
	// Reputer register
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputerAddr.String(),
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	err := s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr)
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr)
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *KeeperTestSuite) TestMsgRemoveRegistration() {
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
	// Reputer register
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputerAddr.String(),
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	err := s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr)
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    reputerAddr.String(),
		TopicId:   topicId,
		IsReputer: true,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr)
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *KeeperTestSuite) TestMsgRegisterWorker() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address())
	creatorAddress := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(1)
	topic1 := types.Topic{Id: topicId, Creator: creatorAddress.String()}

	// Topic register
	s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, topicId)
	// Reputer register
	registerMsg := &types.MsgRegister{
		Sender:       workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    false,
		Owner:        workerAddr.String(),
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	err := s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		workerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	isWorkerRegistered, err := s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr)
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should not be registered in topic")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, workerAddr)
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err = s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr)
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *KeeperTestSuite) TestMsgRemoveRegistrationWorker() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[0].Address())
	creatorAddress := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(1)
	topic1 := types.Topic{Id: topicId, Creator: creatorAddress.String()}

	// Topic register
	s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, topicId)
	// Reputer register
	registerMsg := &types.MsgRegister{
		Sender:       workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    false,
		Owner:        workerAddr.String(),
	}

	mintAmount := sdk.NewCoins(sdk.NewInt64Coin(params.DefaultBondDenom, 100))
	s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	err := s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		workerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err := s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr)
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    workerAddr.String(),
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Unregistering worker should not return an error")

	isWorkerRegistered, err = s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr)
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidLibP2PKey() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrLibP2PKeyRequired, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInsufficientBalance() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	topic1 := types.Topic{Id: topicId, Creator: reputerAddr.String()}
	s.emissionsKeeper.SetTopic(ctx, topicId, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, 1)
	// Zero initial stake

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(types.ErrTopicRegistrantNotEnoughDenom, err, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInsufficientDenom() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	topicId := s.CreateOneTopic()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Register Reputer
	reputerRegMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputerAddr.String(),
	}

	s.emissionsKeeper.AddStake(ctx, topicId, reputerAddr, registrationInitialStake.QuoUint64(2))

	// Try to register without any funds to pay fees
	_, err := msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, types.ErrTopicRegistrantNotEnoughDenom, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidTopicNotExist() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Register should return an error")
}
