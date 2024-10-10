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
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (s *MsgServerTestSuite) TestMsgRegisterReputer() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := s.addrs[0]
	topic1 := s.CreateOneTopic()

	// Topic register
	err := s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	require.NoError(err, "SetTopic should not return an error")
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	require.NoError(err, "ActivateTopic should not return an error")
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	moduleParams, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)

	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topic1.Id, reputerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topic1.Id, reputerAddr.String())
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRemoveRegistration() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := s.addrs[0]
	topic1 := s.CreateOneTopic()

	// Topic register
	err := s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	require.NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	require.NoError(err)
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	moduleParams, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.bankKeeper.MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topic1.Id, reputerAddr.String())
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")

	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: true,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topic1.Id, reputerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRegisterWorker() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := s.addrs[0]
	topic1 := s.CreateOneTopic()

	// Topic register
	err := s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	require.NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	require.NoError(err)
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    workerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: false,
		Owner:     workerAddr.String(),
	}

	moduleParams, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
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

	isWorkerRegistered, err := s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topic1.Id, workerAddr.String())
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should not be registered in topic")

	isReputerRegistered, err := s.emissionsKeeper.IsReputerRegisteredInTopic(ctx, topic1.Id, workerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err = s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topic1.Id, workerAddr.String())
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRemoveRegistrationWorker() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := s.addrs[0]
	topic1 := s.CreateOneTopic()

	// Topic register
	err := s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	require.NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	require.NoError(err)
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    workerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: false,
		Owner:     workerAddr.String(),
	}

	moduleParams, err := s.emissionsKeeper.GetParams(ctx)
	require.NoError(err)
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

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err := s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topic1.Id, workerAddr.String())
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")

	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    workerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: false,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Unregistering worker should not return an error")

	isWorkerRegistered, err = s.emissionsKeeper.IsWorkerRegisteredInTopic(ctx, topic1.Id, workerAddr.String())
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInsufficientBalance() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := s.addrs[0]
	topic1 := s.CreateOneTopic()
	err := s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	require.NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	require.NoError(err)
	// Zero initial stake

	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1))
	// Topic does not exist
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		Owner:     reputerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: true,
	}
	_, err = msgServer.Register(ctx, registerMsg)
	require.Error(err)
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInsufficientDenom() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	topic1 := s.CreateOneTopic()

	// Mock setup for addresses
	reputerAddr := s.addrs[0]
	registrationInitialStake := cosmosMath.NewInt(100)

	// Register Reputer
	reputerRegMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic1.Id,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	err := s.emissionsKeeper.AddReputerStake(ctx, topic1.Id, reputerAddr.String(), registrationInitialStake.QuoRaw(2))
	require.NoError(err)

	// Try to register without any funds to pay fees
	_, err = msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, sdkerrors.ErrInsufficientFunds, "Register should return an error")
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
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  worker.String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              epochLength,
		GroundTruthLag:           epochLength,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	// Get Topic Id
	topicId := res.TopicId

	// Register 1 worker
	workerRegMsg := &types.RegisterRequest{
		Sender:    worker.String(),
		TopicId:   topicId,
		IsReputer: false,
		Owner:     worker.String(),
	}
	_, err = s.msgServer.Register(s.ctx, workerRegMsg)
	s.Require().NoError(err)

	reputerRegMsg := &types.RegisterRequest{
		Sender:    reputer.String(),
		TopicId:   topicId,
		IsReputer: true,
		Owner:     reputer.String(),
	}
	_, err = s.msgServer.Register(s.ctx, reputerRegMsg)
	s.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds, "Register should return an error")
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInvalidTopicNotExist() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := s.addrs[3]

	// Topic does not exist
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		Owner:     reputerAddr.String(),
		TopicId:   topicId,
		IsReputer: true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "Register should return an error")
}
