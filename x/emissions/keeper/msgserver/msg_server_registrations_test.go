package msgserver_test

import (
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func (s *MsgServerTestSuite) TestMsgRegisterReputer() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := s.Addrs()[0]
	// Get topic
	topic := uint64(1)
	err := s.EmissionsKeeper().ActivateTopic(ctx, topic)
	require.NoError(err, "ActivateTopic should not return an error")
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	moduleParams, err := s.EmissionsKeeper().GetParams(ctx)
	require.NoError(err)

	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.BankKeeper().MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.BankKeeper().SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	isReputerRegistered, err := s.EmissionsKeeper().IsReputerRegisteredInTopic(ctx, topic, reputerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.EmissionsKeeper().IsReputerRegisteredInTopic(ctx, topic, reputerAddr.String())
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRemoveRegistration() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := s.Addrs()[0]
	// Get topic
	topic := uint64(1)
	err := s.EmissionsKeeper().ActivateTopic(ctx, topic)
	require.NoError(err)
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	moduleParams, err := s.EmissionsKeeper().GetParams(ctx)
	require.NoError(err)
	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.BankKeeper().MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.BankKeeper().SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		reputerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err := s.EmissionsKeeper().IsReputerRegisteredInTopic(ctx, topic, reputerAddr.String())
	require.NoError(err)
	require.True(isReputerRegistered, "Reputer should be registered in topic")

	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic,
		IsReputer: true,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Registering reputer should not return an error")

	isReputerRegistered, err = s.EmissionsKeeper().IsReputerRegisteredInTopic(ctx, topic, reputerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRegisterWorker() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	// Mock setup for addresses
	workerAddr := s.Addrs()[0]
	// Get topic
	topic := uint64(1)
	err := s.EmissionsKeeper().ActivateTopic(ctx, topic)
	require.NoError(err)
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    workerAddr.String(),
		TopicId:   topic,
		IsReputer: false,
		Owner:     workerAddr.String(),
	}

	moduleParams, err := s.EmissionsKeeper().GetParams(ctx)
	require.NoError(err)
	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.BankKeeper().MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.BankKeeper().SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		workerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	isWorkerRegistered, err := s.EmissionsKeeper().IsWorkerRegisteredInTopic(ctx, topic, workerAddr.String())
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should not be registered in topic")

	isReputerRegistered, err := s.EmissionsKeeper().IsReputerRegisteredInTopic(ctx, topic, workerAddr.String())
	require.NoError(err)
	require.False(isReputerRegistered, "Reputer should not be registered in topic")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err = s.EmissionsKeeper().IsWorkerRegisteredInTopic(ctx, topic, workerAddr.String())
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRemoveRegistrationWorker() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	// Mock setup for addresses
	workerAddr := s.Addrs()[0]
	// Get topic
	topic := uint64(1)
	err := s.EmissionsKeeper().ActivateTopic(ctx, topic)
	require.NoError(err)
	// Reputer register
	registerMsg := &types.RegisterRequest{
		Sender:    workerAddr.String(),
		TopicId:   topic,
		IsReputer: false,
		Owner:     workerAddr.String(),
	}

	moduleParams, err := s.EmissionsKeeper().GetParams(ctx)
	require.NoError(err)
	mintAmount := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, moduleParams.RegistrationFee))
	err = s.BankKeeper().MintCoins(ctx, minttypes.ModuleName, mintAmount)
	require.NoError(err, "MintCoins should not return an error")
	err = s.BankKeeper().SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		workerAddr,
		mintAmount,
	)
	require.NoError(err, "SendCoinsFromModuleToAccount should not return an error")

	_, err = msgServer.Register(ctx, registerMsg)
	require.NoError(err, "Registering worker should not return an error")

	isWorkerRegistered, err := s.EmissionsKeeper().IsWorkerRegisteredInTopic(ctx, topic, workerAddr.String())
	require.NoError(err)
	require.True(isWorkerRegistered, "Worker should be registered in topic")

	unregisterMsg := &types.RemoveRegistrationRequest{
		Sender:    workerAddr.String(),
		TopicId:   topic,
		IsReputer: false,
	}

	_, err = msgServer.RemoveRegistration(ctx, unregisterMsg)
	require.NoError(err, "Unregistering worker should not return an error")

	isWorkerRegistered, err = s.EmissionsKeeper().IsWorkerRegisteredInTopic(ctx, topic, workerAddr.String())
	require.NoError(err)
	require.False(isWorkerRegistered, "Worker should be registered in topic")
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInsufficientBalance() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	topic := uint64(1)
	err := s.EmissionsKeeper().ActivateTopic(ctx, topic)
	require.NoError(err)
	// Zero initial stake

	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1))
	// Topic does not exist
	registerMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		Owner:     reputerAddr.String(),
		TopicId:   topic,
		IsReputer: true,
	}
	_, err = msgServer.Register(ctx, registerMsg)
	require.Error(err)
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInsufficientDenom() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()
	topic := uint64(1)

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	registrationInitialStake := cosmosMath.NewInt(100)

	// Register Reputer
	reputerRegMsg := &types.RegisterRequest{
		Sender:    reputerAddr.String(),
		TopicId:   topic,
		IsReputer: true,
		Owner:     reputerAddr.String(),
	}

	err := s.EmissionsKeeper().AddReputerStake(ctx, topic, reputerAddr.String(), registrationInitialStake.QuoRaw(2))
	require.NoError(err)

	// Try to register without any funds to pay fees
	_, err = msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, sdkerrors.ErrInsufficientFunds, "Register should return an error")
}

func (s *MsgServerTestSuite) TestBlocklistedAddressUnableToRegister() {
	// Reputer Addresses
	reputer := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	// Worker Addresses
	worker := s.Addrs()[3]
	cosmosOneE18, ok := cosmosMath.NewIntFromString("1000000000000000000")
	s.Require().True(ok)

	bankKeeper := bankkeeper.NewBaseKeeper(
		s.Codec(),
		s.StoreServiceBank(),
		s.AccountKeeper(),
		map[string]bool{
			s.AddrsStr()[0]: true,
		},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	emissionsKeeper := keeper.NewKeeper(
		s.Codec(),
		codecAddress.NewBech32Codec(params.Bech32PrefixAccAddr),
		s.StoreServiceEmissions(),
		s.AccountKeeper(),
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	msgServer := msgserver.NewMsgServerImpl(emissionsKeeper)

	blockHeight := int64(600)
	s.WithBlockHeight(blockHeight)

	s.MintTokensToAddress(worker, cosmosMath.NewInt(10).Mul(cosmosOneE18))
	topicId := uint64(1) // already included

	// Register 1 worker
	workerRegMsg := &types.RegisterRequest{
		Sender:    worker.String(),
		TopicId:   topicId,
		IsReputer: false,
		Owner:     worker.String(),
	}
	_, err := msgServer.Register(s.Ctx(), workerRegMsg)
	s.Require().NoError(err)

	reputerRegMsg := &types.RegisterRequest{
		Sender:    reputer.String(),
		TopicId:   topicId,
		IsReputer: true,
		Owner:     reputer.String(),
	}
	_, err = msgServer.Register(s.Ctx(), reputerRegMsg)
	s.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds, "Register should return an error")
}

func (s *MsgServerTestSuite) TestMsgRegisterReputerInvalidTopicNotExist() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()

	topicId := uint64(0)

	// Mock setup for addresses
	reputerAddr := s.Addrs()[3]

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
