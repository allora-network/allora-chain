package msgserver_test

import (
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (s *MsgServerTestSuite) TestMsgRegisterReputer() {
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

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRemoveRegistration() {
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

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    reputerAddr.String(),
		TopicId:   topicId,
		IsReputer: true,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRegisterWorker() {
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

	isWorkerRegistered, err := s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr.String())
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should not be registered in topic")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topicId, workerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err = s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr.String())
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRemoveRegistrationWorker() {
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

	isWorkerRegistered, err := s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr.String())
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")

	unregisterMsg := &types.MsgRemoveRegistration{
		Sender:    workerAddr.String(),
		TopicId:   topicId,
		IsReputer: false,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Unregistering worker should not return an error")

	isWorkerRegistered, err = s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topicId, workerAddr.String())
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInvalidLibP2PKey() {
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

func (s *MsgServerTestSuite) TestMsgRegisterReputerInsufficientBalance() {
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

func (s *MsgServerTestSuite) TestMsgRegisterReputerInsufficientDenom() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	topicId := s.CreateOneTopic()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	registrationInitialStake := cosmosMath.NewInt(100)

	// Register Reputer
	reputerRegMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputerAddr.String(),
	}

	s.emissionsKeeper.AddReputerStake(ctx, topicId, reputerAddr.String(), registrationInitialStake.QuoRaw(2))

	// Try to register without any funds to pay fees
	_, err := msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, types.ErrTopicRegistrantNotEnoughDenom, "Register should return an error")
}

func (s *MsgServerTestSuite) TestBlocklistedAddressUnableToRegister() {
	// Reputer Addresses
	reputer := s.addrs[2]
	// Worker Addresses
	worker := s.addrs[3]
	cosmosOneE18, ok := cosmosMath.NewIntFromString("1000000000000000000")
	s.Require().True(ok)

	s.bankKeeper = bankkeeper.NewBaseKeeper(
		s.codec,
		s.storeService,
		s.accountKeeper,
		map[string]bool{
			s.addrsStr[0]: true,
		},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	s.emissionsKeeper = keeper.NewKeeper(
		s.codec,
		s.addressCodec,
		s.storeService,
		s.accountKeeper,
		s.bankKeeper,
		authtypes.FeeCollectorName,
	)

	blockHeight := int64(600)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)
	epochLength := int64(10800)

	s.MintTokensToAddress(worker, cosmosMath.NewInt(10).Mul(cosmosOneE18))
	// Create topic
	newTopicMsg := &types.MsgCreateNewTopic{
<<<<<<< HEAD
		Creator:                  worker.String(),
		Metadata:                 "test",
		LossLogic:                "logic",
		LossMethod:               "method",
		EpochLength:              epochLength,
		InferenceLogic:           "Ilogic",
		InferenceMethod:          "Imethod",
		DefaultArg:               "ETH",
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.25"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.25"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.25"),
=======
		Creator:         worker.String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     epochLength,
		GroundTruthLag:  epochLength,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
>>>>>>> dev
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	// Get Topic Id
	topicId := res.TopicId

	// Register 1 worker
	workerRegMsg := &types.MsgRegister{
		Sender:       worker.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    false,
		Owner:        worker.String(),
	}
	_, err = s.msgServer.Register(s.ctx, workerRegMsg)
	s.Require().NoError(err)

	reputerRegMsg := &types.MsgRegister{
		Sender:       reputer.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
		Owner:        reputer.String(),
	}
	_, err = s.msgServer.Register(s.ctx, reputerRegMsg)
	s.Require().ErrorIs(err, types.ErrTopicRegistrantNotEnoughDenom, "Register should return an error")
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInvalidTopicNotExist() {
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
