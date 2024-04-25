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
	topic1 := types.Topic{Id: 1, Creator: creatorAddress.String()}

	// Topic register
	s.emissionsKeeper.SetTopic(ctx, 1, topic1)
	s.emissionsKeeper.ActivateTopic(ctx, 1)
	// Reputer register
	registerMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      1,
		IsReputer:    true,
		Owner:        "Reputer",
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
	}

	s.emissionsKeeper.AddStake(ctx, topicId, reputerAddr, registrationInitialStake.QuoUint64(2))

	// Try to register without any funds to pay fees
	_, err := msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, types.ErrTopicRegistrantNotEnoughDenom, "Register should return an error")
}

/*
func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidInsufficientStakeToRegisterAfterRemovingRegistration() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	s.CreateOneTopic()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	registrationInitialStake := cosmosMath.NewUint(100)
	registrationInitialStakeCoins := sdk.NewCoins(
		sdk.NewCoin(
			params.DefaultBondDenom,
			cosmosMath.NewIntFromBigInt(registrationInitialStake.BigInt())))

	// Register Reputer
	reputerRegMsg := &types.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicIds:     []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    true,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, types.AlloraStakingAccountName, registrationInitialStakeCoins)
	_, err := msgServer.Register(ctx, reputerRegMsg)
	require.NoError(err, "Registering reputer should not return an error")

	// Deregister Reputer
	_, err = msgServer.RemoveRegistration(ctx, &types.MsgRemoveRegistration{
		Creator: reputerAddr.String(),
		TopicId: 0,
	})
	require.NoError(err, "RemoveRegistration should not return an error")

	// Remove stake half of the initial stake
	removeStakeMsg := &types.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*types.StakePlacement{
			{
				TopicId: 0,
				Amount:  registrationInitialStake.QuoUint64(2),
			},
		},
	}
	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.NoError(err, "StartRemoveStake should not return an error")

	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.AlloraStakingAccountName, reputerAddr, registrationInitialStakeCoins.QuoInt(cosmosMath.NewInt(2)))
	_, err = msgServer.ConfirmRemoveStake(ctx, &types.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	})
	require.NoError(err, "ConfirmRemoveStake should not return an error")

	// Try to register with zero initial stake and having half of the initial stake removed
	reputerRegMsg.InitialStake = cosmosMath.NewUint(0)
	_, err = msgServer.Register(ctx, reputerRegMsg)
	require.ErrorIs(err, types.ErrInsufficientStakeToRegister, "Register should return an error")
}
*/

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

/*
func (s *KeeperTestSuite) TestMsgRegisterReputerAddAndRemoveAdditionalTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	workerAddr := sdk.AccAddress(PKS[1].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Create Topic 1
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          reputerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		LossLogic:        "logic",
		EpochLength:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		DefaultArg:       "ETH",
		AlphaRegret:      alloraMath.NewDecFromInt64(10),
		PrewardReputer:   alloraMath.NewDecFromInt64(11),
		PrewardInference: alloraMath.NewDecFromInt64(12),
		PrewardForecast:  alloraMath.NewDecFromInt64(13),
		FTolerance:       alloraMath.NewDecFromInt64(14),
	}
	s.PrepareForCreateTopic(newTopicMsg.Creator)
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Reputer in additional topic 1
	registerReputerMsg := &types.MsgRegisterWithExistingStake{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "reputerKey",
		MultiAddress: "reputerAddr",
		TopicId:      1,
		IsReputer:    true,
	}
	_, err = msgServer.RegisterWithExistingStake(ctx, registerReputerMsg)
	require.NoError(err, "RegisterReputer should not return an error")

	// Check Topic 1 stake
	// Should have same amount of the account's initial stake
	topicStake, err := s.emissionsKeeper.GetTopicStake(ctx, 1)
	require.NoError(err)
	require.Equal(registrationInitialStake, topicStake, "Topic 1 stake amount mismatch")

	// Check Address Topics
	// Should have two topics
	addressTopics, err := s.emissionsKeeper.GetRegisteredTopicIdByReputerAddress(ctx, reputerAddr)
	require.NoError(err)
	require.Equal(2, len(addressTopics), "Address topics count mismatch")

	// Add Stake to Topic 1
	stakeToAdd := cosmosMath.NewUint(50)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		reputerAddr,
		types.AlloraStakingAccountName,
		sdk.NewCoins(
			sdk.NewCoin(
				params.DefaultBondDenom,
				cosmosMath.NewIntFromBigInt(stakeToAdd.BigInt()))))
	_, err = msgServer.AddStake(ctx, &types.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: reputerAddr.String(),
		Amount:      stakeToAdd,
	})
	require.NoError(err, "AddStake should not return an error")

	// Check Topic 1 stake
	// Should have same amount of the account's initial stake + stakeToAdd
	topicStake, err = s.emissionsKeeper.GetTopicStake(ctx, 1)
	require.NoError(err)
	require.Equal(registrationInitialStake.Add(stakeToAdd), topicStake, "Topic 1 stake amount mismatch")

	// Remove Reputer from Topic 1
	_, err = msgServer.RemoveRegistration(ctx, &types.MsgRemoveRegistration{
		Creator:   reputerAddr.String(),
		TopicId:   1,
		IsReputer: true,
	})
	require.NoError(err, "RemoveRegistration should not return an error")

	// Check Address Topics
	// Should have only one topic
	addressTopics, err = s.emissionsKeeper.GetRegisteredTopicIdByReputerAddress(ctx, reputerAddr)
	require.NoError(err)
	require.Equal(1, len(addressTopics), "Address topics count mismatch")
}
*/

/*
func (s *KeeperTestSuite) TestMsgRegisterWorkerAddAndRemoveAdditionalTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	workerAddr := sdk.AccAddress(PKS[1].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register worker in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Create Topic 1
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          workerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		LossLogic:        "logic",
		EpochLength:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		DefaultArg:       "ETH",
		AlphaRegret:      alloraMath.NewDecFromInt64(10),
		PrewardReputer:   alloraMath.NewDecFromInt64(11),
		PrewardInference: alloraMath.NewDecFromInt64(12),
		PrewardForecast:  alloraMath.NewDecFromInt64(13),
		FTolerance:       alloraMath.NewDecFromInt64(14),
	}
	s.PrepareForCreateTopic(newTopicMsg.Creator)
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Worker in additional topic 1
	registerWorkerMsg := &types.MsgRegisterWithExistingStake{
		Creator:      workerAddr.String(),
		LibP2PKey:    "workerKey",
		MultiAddress: "workerAddr",
		TopicId:      1,
	}
	_, err = msgServer.RegisterWithExistingStake(ctx, registerWorkerMsg)
	require.NoError(err, "RegisterReputer should not return an error")

	// Check Topic 1 stake
	// Should have same amount of the account's initial stake
	topicStake, err := s.emissionsKeeper.GetTopicStake(ctx, 1)
	require.NoError(err)
	require.Equal(registrationInitialStake, topicStake, "Topic 1 stake amount mismatch")

	// Check Address Topics
	// Should have two topics
	addressTopics, err := s.emissionsKeeper.GetRegisteredTopicIdsByWorkerAddress(ctx, workerAddr)
	require.NoError(err)
	require.Equal(2, len(addressTopics), "Address topics count mismatch")

	// Add Stake to Topic 1
	stakeToAdd := cosmosMath.NewUint(50)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		workerAddr,
		types.AlloraStakingAccountName,
		sdk.NewCoins(
			sdk.NewCoin(
				params.DefaultBondDenom,
				cosmosMath.NewIntFromBigInt(stakeToAdd.BigInt()))))
	_, err = msgServer.AddStake(ctx, &types.MsgAddStake{
		Sender:      workerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeToAdd,
	})
	require.NoError(err, "AddStake should not return an error")

	// Check Topic 1 stake
	// Should have same amount of the account's initial stake + stakeToAdd
	topicStake, err = s.emissionsKeeper.GetTopicStake(ctx, 1)
	require.NoError(err)
	require.Equal(registrationInitialStake.Add(stakeToAdd), topicStake, "Topic 1 stake amount mismatch")

	// Remove Reputer from Topic 1
	_, err = msgServer.RemoveRegistration(ctx, &types.MsgRemoveRegistration{
		Creator: workerAddr.String(),
		TopicId: 1,
	})
	require.NoError(err, "RemoveRegistration should not return an error")

	// Check Address Topics
	// Should have only one topic
	addressTopics, err = s.emissionsKeeper.GetRegisteredTopicIdsByWorkerAddress(ctx, workerAddr)
	require.NoError(err)
	require.Equal(1, len(addressTopics), "Address topics count mismatch")
}

func (s *KeeperTestSuite) TestMsgRemoveRegistrationInvalidAddressNotRegistered() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	s.CreateOneTopic()

	// Start Remove Registration
	addr := sdk.AccAddress(PKS[0].Address())
	_, err := msgServer.RemoveRegistration(ctx, &types.MsgRemoveRegistration{
		Creator:   addr.String(),
		TopicId:   0,
		IsReputer: false,
	})
	require.ErrorIs(err, types.ErrAddressIsNotRegisteredInThisTopic, "RemoveRegistration should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterWorkerInvalidTopicNotExist() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[1].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Topic does not exist
	registerMsg := &types.MsgRegister{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicIds:     []uint64{0},
		InitialStake: registrationInitialStake,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrTopicDoesNotExist, "RegisterWorker should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterWorkerInvalidAlreadyRegistered() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	workerAddr := sdk.AccAddress(PKS[1].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register worker in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Try to register again
	registerMsg := &types.MsgRegister{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicIds:     []uint64{0},
		InitialStake: registrationInitialStake,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, types.ErrAddressAlreadyRegisteredInATopic, "RegisterWorker should return an error")
}

*/
