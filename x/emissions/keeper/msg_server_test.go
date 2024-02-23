package keeper_test

import (
	"fmt"
	"math"
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
)

var (
	nonAdminAccounts = simtestutil.CreateRandomAccounts(4)
	// TODO: Change PKS to accounts here and in all the tests (like the above line)
	PKS     = simtestutil.CreateTestPubKeys(4)
	Addr    = sdk.AccAddress(PKS[0].Address())
	ValAddr = sdk.ValAddress(Addr)
)

// ########################################
// #           Topics tests              #
// ########################################

func (s *KeeperTestSuite) TestMsgCreateNewTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a MsgCreateNewTopic message
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          sdk.AccAddress(PKS[0].Address()).String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.emissionsKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(1), "Topic count after first topic is not 1.")
}

func (s *KeeperTestSuite) TestMsgCreateNewTopicInvalidUnauthorized() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a MsgCreateNewTopic message
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          nonAdminAccounts[0].String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.ErrorIs(err, state.ErrNotInTopicCreationWhitelist, "CreateTopic should return an error")
}

func (s *KeeperTestSuite) TestMsgReactivateTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicCreator := sdk.AccAddress(PKS[0].Address()).String()
	s.CreateOneTopic()

	// Deactivate topic
	s.emissionsKeeper.InactivateTopic(ctx, 0)

	// Set unmet demand for topic
	s.emissionsKeeper.SetTopicUnmetDemand(ctx, 0, cosmosMath.NewUint(100))

	// Create a MsgCreateNewTopic message
	reactivateTopicMsg := &state.MsgReactivateTopic{
		Sender:  topicCreator,
		TopicId: 0,
	}

	_, err := msgServer.ReactivateTopic(ctx, reactivateTopicMsg)
	require.NoError(err, "ReactivateTopic should not return an error")

	// Check if topic is active
	topic, err := s.emissionsKeeper.GetTopic(ctx, 0)
	require.NoError(err)
	require.True(topic.Active, "Topic should be active")
}

func (s *KeeperTestSuite) TestMsgReactivateTopicInvalidNotEnoughDemand() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	topicCreator := sdk.AccAddress(PKS[0].Address()).String()
	s.CreateOneTopic()

	// Deactivate topic
	s.emissionsKeeper.InactivateTopic(ctx, 0)

	// Create a MsgCreateNewTopic message
	reactivateTopicMsg := &state.MsgReactivateTopic{
		Sender:  topicCreator,
		TopicId: 0,
	}

	_, err := msgServer.ReactivateTopic(ctx, reactivateTopicMsg)
	require.ErrorIs(err, state.ErrTopicNotEnoughDemand, "ReactivateTopic should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidLibP2PKey() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Topic does not exist
	registerMsg := &state.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, state.ErrLibP2PKeyRequired, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidInsufficientStakeToRegister() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	// Zero initial stake
	registrationInitialStake := cosmosMath.NewUint(0)

	// Topic does not exist
	registerMsg := &state.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, state.ErrInsufficientStakeToRegister, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidTopicNotExist() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Topic does not exist
	registerMsg := &state.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, state.ErrTopicDoesNotExist, "Register should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterReputerInvalidAlreadyRegistered() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address())
	workerAddr := sdk.AccAddress(PKS[1].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Create topic 0 and register reputer in it
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Try to register again
	registerMsg := &state.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    true,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, state.ErrAddressAlreadyRegisteredInATopic, "Register should return an error")
}

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
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          reputerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Reputer in additional topic 1
	registerReputerMsg := &state.MsgAddNewRegistration{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "reputerKey",
		MultiAddress: "reputerAddr",
		TopicId:      1,
		IsReputer:    true,
	}
	_, err = msgServer.AddNewRegistration(ctx, registerReputerMsg)
	require.NoError(err, "RegisterReputer should not return an error")

	// Check Topic 1 stake
	// Should have same amount of the account's initial stake
	topicStake, err := s.emissionsKeeper.GetTopicStake(ctx, 1)
	require.NoError(err)
	require.Equal(registrationInitialStake, topicStake, "Topic 1 stake amount mismatch")

	// Check Address Topics
	// Should have two topics
	addressTopics, err := s.emissionsKeeper.GetRegisteredTopicsIdsByReputerAddress(ctx, reputerAddr)
	require.NoError(err)
	require.Equal(2, len(addressTopics), "Address topics count mismatch")

	// Add Stake to Topic 1
	stakeToAdd := cosmosMath.NewUint(50)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		reputerAddr,
		state.AlloraStakingModuleName,
		sdk.NewCoins(
			sdk.NewCoin(
				params.DefaultBondDenom,
				cosmosMath.NewIntFromBigInt(stakeToAdd.BigInt()))))
	_, err = msgServer.AddStake(ctx, &state.MsgAddStake{
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
	_, err = msgServer.RemoveRegistration(ctx, &state.MsgRemoveRegistration{
		Creator:   reputerAddr.String(),
		TopicId:   1,
		IsReputer: true,
	})
	require.NoError(err, "RemoveRegistration should not return an error")

	// Check Address Topics
	// Should have only one topic
	addressTopics, err = s.emissionsKeeper.GetRegisteredTopicsIdsByReputerAddress(ctx, reputerAddr)
	require.NoError(err)
	require.Equal(1, len(addressTopics), "Address topics count mismatch")
}

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
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          workerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Worker in additional topic 1
	registerWorkerMsg := &state.MsgAddNewRegistration{
		Creator:      workerAddr.String(),
		LibP2PKey:    "workerKey",
		MultiAddress: "workerAddr",
		TopicId:      1,
	}
	_, err = msgServer.AddNewRegistration(ctx, registerWorkerMsg)
	require.NoError(err, "RegisterReputer should not return an error")

	// Check Topic 1 stake
	// Should have same amount of the account's initial stake
	topicStake, err := s.emissionsKeeper.GetTopicStake(ctx, 1)
	require.NoError(err)
	require.Equal(registrationInitialStake, topicStake, "Topic 1 stake amount mismatch")

	// Check Address Topics
	// Should have two topics
	addressTopics, err := s.emissionsKeeper.GetRegisteredTopicsIdsByWorkerAddress(ctx, workerAddr)
	require.NoError(err)
	require.Equal(2, len(addressTopics), "Address topics count mismatch")

	// Add Stake to Topic 1
	stakeToAdd := cosmosMath.NewUint(50)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		workerAddr,
		state.AlloraStakingModuleName,
		sdk.NewCoins(
			sdk.NewCoin(
				params.DefaultBondDenom,
				cosmosMath.NewIntFromBigInt(stakeToAdd.BigInt()))))
	_, err = msgServer.AddStake(ctx, &state.MsgAddStake{
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
	_, err = msgServer.RemoveRegistration(ctx, &state.MsgRemoveRegistration{
		Creator: workerAddr.String(),
		TopicId: 1,
	})
	require.NoError(err, "RemoveRegistration should not return an error")

	// Check Address Topics
	// Should have only one topic
	addressTopics, err = s.emissionsKeeper.GetRegisteredTopicsIdsByWorkerAddress(ctx, workerAddr)
	require.NoError(err)
	require.Equal(1, len(addressTopics), "Address topics count mismatch")
}

func (s *KeeperTestSuite) TestMsgRemoveRegistrationInvalidAddressNotRegistered() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	s.CreateOneTopic()

	// Start Remove Registration
	addr := sdk.AccAddress(PKS[0].Address())
	_, err := msgServer.RemoveRegistration(ctx, &state.MsgRemoveRegistration{
		Creator:   addr.String(),
		TopicId:   0,
		IsReputer: false,
	})
	require.ErrorIs(err, state.ErrAddressIsNotRegisteredInThisTopic, "RemoveRegistration should return an error")
}

func (s *KeeperTestSuite) TestMsgRegisterWorkerInvalidTopicNotExist() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[1].Address())
	registrationInitialStake := cosmosMath.NewUint(100)

	// Topic does not exist
	registerMsg := &state.MsgRegister{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, state.ErrTopicDoesNotExist, "RegisterWorker should return an error")
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
	registerMsg := &state.MsgRegister{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
	}
	_, err := msgServer.Register(ctx, registerMsg)
	require.ErrorIs(err, state.ErrAddressAlreadyRegisteredInATopic, "RegisterWorker should return an error")
}

func (s *KeeperTestSuite) TestMsgSetWeights() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()).String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetWeights message
	weightMsg := &state.MsgSetWeights{
		Sender: reputerAddr,
		Weights: []*state.Weight{
			{
				TopicId: 1,
				Reputer: reputerAddr,
				Worker:  workerAddr,
				Weight:  cosmosMath.NewUint(100),
			},
		},
	}

	_, err := msgServer.SetWeights(ctx, weightMsg)
	require.NoError(err, "SetWeights should not return an error")
}

func (s *KeeperTestSuite) TestMsgSetWeightsInvalidUnauthorized() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := nonAdminAccounts[0].String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetWeights message
	weightMsg := &state.MsgSetWeights{
		Sender: reputerAddr,
		Weights: []*state.Weight{
			{
				TopicId: 1,
				Reputer: reputerAddr,
				Worker:  workerAddr,
				Weight:  cosmosMath.NewUint(100),
			},
		},
	}

	_, err := msgServer.SetWeights(ctx, weightMsg)
	require.ErrorIs(err, state.ErrNotInWeightSettingWhitelist, "SetWeights should return an error")
}

func (s *KeeperTestSuite) TestMsgSetInferences() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetInferences message
	inferencesMsg := &state.MsgSetInferences{
		Inferences: []*state.Inference{
			{
				TopicId:   1,
				Worker:    workerAddr,
				Value:     cosmosMath.NewUint(12),
				ExtraData: []byte("test"),
				Proof:     "test",
			},
		},
	}

	_, err := msgServer.SetInferences(ctx, inferencesMsg)
	require.NoError(err, "SetInferences should not return an error")
}

func (s *KeeperTestSuite) CreateOneTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a topic first
	metadata := "Some metadata for the new topic"
	// Create a MsgCreateNewTopic message
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          sdk.AccAddress(PKS[0].Address()).String(),
		Metadata:         metadata,
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")
}

func (s *KeeperTestSuite) TestUpdateTopicWeightLastRan() {
	ctx := s.ctx
	require := s.Require()
	s.CreateOneTopic()

	// Mock setup for topic
	topicId := uint64(0)
	inferenceTs := uint64(time.Now().UTC().Unix())

	err := s.emissionsKeeper.UpdateTopicWeightLastRan(ctx, topicId, inferenceTs)
	require.NoError(err, "UpdateTopicWeightLastRan should not return an error")

	result, err := s.emissionsKeeper.GetTopicWeightLastRan(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, inferenceTs)
}

func (s *KeeperTestSuite) TestProcessInferencesAndQuery() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	s.CreateOneTopic()

	// Mock setup for inferences
	inferences := []*state.Inference{
		{TopicId: 0, Worker: "worker1", Value: cosmosMath.NewUint(2200)},
		{TopicId: 0, Worker: "worker2", Value: cosmosMath.NewUint(2100)},
		{TopicId: 2, Worker: "worker2", Value: cosmosMath.NewUint(12)},
	}

	// Call the ProcessInferences function to test writes
	processInferencesMsg := &state.MsgProcessInferences{
		Inferences: inferences,
	}
	_, err := msgServer.ProcessInferences(ctx, processInferencesMsg)
	require.NoError(err, "Processing Inferences should not fail")

	/*
	 * Inferences over threshold should be returned
	 */
	// Ensure low ts for topic 1
	var topicId = uint64(0)
	var inferenceTimestamp = uint64(1500000000)

	// _, err = msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)
	err = s.emissionsKeeper.UpdateTopicWeightLastRan(ctx, topicId, inferenceTimestamp)
	require.NoError(err, "Setting latest inference timestamp should not fail")

	allInferences, err := s.emissionsKeeper.GetLatestInferencesFromTopic(ctx, uint64(0))
	require.Equal(len(allInferences), 1)
	for _, inference := range allInferences {
		require.Equal(len(inference.Inferences.Inferences), 2)
	}
	require.NoError(err, "Inferences over ts threshold should be returned")

	/*
	 * Inferences under threshold should not be returned
	 */
	inferenceTimestamp = math.MaxUint64

	err = s.emissionsKeeper.UpdateTopicWeightLastRan(ctx, topicId, inferenceTimestamp)
	require.NoError(err)

	allInferences, err = s.emissionsKeeper.GetLatestInferencesFromTopic(ctx, uint64(1))

	require.Equal(len(allInferences), 0)
	require.NoError(err, "Inferences under ts threshold should not be returned")

}

func (s *KeeperTestSuite) TestCreateSeveralTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	// Mock setup for metadata and validation steps
	metadata := "Some metadata for the new topic"
	// Create a MsgSetInferences message
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          sdk.AccAddress(PKS[0].Address()).String(),
		Metadata:         metadata,
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.emissionsKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(1), "Topic count after first topic is not 1.")

	// Create second topic
	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on second topic")

	result, err = s.emissionsKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(2), "Topic count after second topic insertion is not 2")
}

// ########################################
// #           Staking tests              #
// ########################################

func (s *KeeperTestSuite) commonStakingSetup(ctx sdk.Context, reputerAddr sdk.AccAddress, workerAddr sdk.AccAddress, registrationInitialStake cosmosMath.Uint) {
	msgServer := s.msgServer
	require := s.Require()
	registrationInitialStakeCoins := sdk.NewCoins(
		sdk.NewCoin(
			params.DefaultBondDenom,
			cosmosMath.NewIntFromBigInt(registrationInitialStake.BigInt())))

	// Create Topic
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          reputerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Reputer
	reputerRegMsg := &state.MsgRegister{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		IsReputer:    true,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, registrationInitialStakeCoins)
	_, err = msgServer.Register(ctx, reputerRegMsg)
	require.NoError(err, "Registering reputer should not return an error")

	// Register Worker
	workerRegMsg := &state.MsgRegister{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		Owner:        workerAddr.String(),
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), workerAddr, state.AlloraStakingModuleName, registrationInitialStakeCoins)
	_, err = msgServer.Register(ctx, workerRegMsg)
	require.NoError(err, "Registering worker should not return an error")
}

func (s *KeeperTestSuite) TestMsgAddStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Add stake from reputer (sender) to worker (target)
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Check updated stake for delegator
	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)
	// Registration Stake: 100
	// Stake placed upon target: 1000
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch")

	// Check updated stake for target
	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
	require.NoError(err)
	// Registration Stake: 100
	// Stake placed upon target: 1000
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), targetStake, "Target stake amount mismatch")

	// Check updated total stake
	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)
	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 1000
	// Total: 1200
	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch")

	// Check updated total stake for topic
	totalStakeForTopic, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)
	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 1000
	// Total: 1200
	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic, "Total stake amount for topic mismatch")

	// Check bond
	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
	require.NoError(err)
	// Stake placed upon target: 1000
	require.Equal(stakeAmount, bond, "Bond amount mismatch")
}

func (s *KeeperTestSuite) TestMsgAddAndRemoveStakeWithTargetWorkerRegisteredInMultipleTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Create topic and register target (worker) in additional topic (topic 1)
	// Create Topic 1
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          workerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Worker in topic 1
	workerAddRegMsg := &state.MsgAddNewRegistration{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      1,
		Owner:        workerAddr.String(),
	}
	_, err = msgServer.AddNewRegistration(ctx, workerAddRegMsg)
	require.NoError(err, "Registering worker should not return an error")

	// Add stake from reputer (sender) to worker (target)
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err = msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Check updated stake for delegator
	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)
	// Registration Stake: 100
	// Stake placed upon target: 1000
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch")

	// Check updated stake for target
	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
	require.NoError(err)
	// Registration Stake: 100 - first registration - topic 0
	// Stake placed upon target: 1000
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), targetStake, "Target stake amount mismatch")

	// Check updated total stake
	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)
	// Registration Stake: 200 (100 for reputer, 100 for worker) - topic 0
	// Stake placed upon target: 1000
	// Total: 1200
	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch")

	// Check updated total stake for topic 0
	totalStakeForTopic0, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)
	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 1000
	// Total: 1200
	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic0, "Total stake amount for topic mismatch")

	// Check updated total stake for topic 1
	totalStakeForTopic1, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
	require.NoError(err)
	// Stake placed upon target: 1100
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), totalStakeForTopic1, "Total stake amount for topic mismatch")

	// Check bond
	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
	require.NoError(err)
	// Stake placed upon target: 1000
	require.Equal(stakeAmount, bond, "Bond amount mismatch")

	// Remove stake from reputer (sender) to worker (target)
	removeStakeMsg := &state.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: workerAddr.String(),
				Amount: stakeAmount,
			},
		},
	}
	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.NoError(err, "StartRemoveStake should not return an error")

	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, reputerAddr, stakeAmountCoins)
	_, err = msgServer.ConfirmRemoveStake(ctx, &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	})
	require.NoError(err, "ConfirmRemoveStake should not return an error")

	// Check updated stake for delegator
	delegatorStake, err = s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)

	// Registration Stake: 100 - topic 0
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, delegatorStake, "Delegator stake amount mismatch")

	// Check updated stake for target
	targetStake, err = s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
	require.NoError(err)

	// Registration Stake: 100 - topic 0
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, targetStake, "Target stake amount mismatch")

	// Check updated total stake
	totalStake, err = s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)

	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 0
	// Total: 200
	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStake, "Total stake amount mismatch")

	// Check updated total stake for topic 0
	totalStakeForTopic0, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)

	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 0
	// Total: 200
	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStakeForTopic0, "Total stake amount for topic mismatch")

	// Check updated total stake for topic 0
	totalStakeForTopic1, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
	require.NoError(err)

	// Registration Stake: 100 (worker)
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, totalStakeForTopic1, "Total stake amount for topic mismatch")

	// Check bond
	bond, err = s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
	require.NoError(err)
	// Stake placed upon target: 0
	require.Equal(cosmosMath.ZeroUint(), bond, "Bond amount mismatch")
}

func (s *KeeperTestSuite) TestMsgAddAndRemoveStakeWithTargetReputerRegisteredInMultipleTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Create topic and register target (worker) in additional topic (topic 1)
	// Create Topic 1
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          workerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Reputer in topic 1
	reputerAddRegMsg := &state.MsgAddNewRegistration{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      1,
		IsReputer:    true,
	}
	_, err = msgServer.AddNewRegistration(ctx, reputerAddRegMsg)
	require.NoError(err, "Registering reputer should not return an error")

	// Add stake from reputer (sender) to reputer (target)
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: reputerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err = msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Check updated stake for delegator
	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)
	// Registration Stake: 100
	// Stake placed upon target: 1000
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch")

	// Check updated stake for target
	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, reputerAddr)
	require.NoError(err)
	// Registration Stake: 100 - first registration - topic 0
	// Stake placed upon target: 1000
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), targetStake, "Target stake amount mismatch")

	// Check updated total stake
	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)
	// Registration Stake: 200 (100 for reputer, 100 for worker) - topic 0
	// Stake placed upon target: 1000
	// Total: 1200
	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch")

	// Check updated total stake for topic 0
	totalStakeForTopic0, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)
	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 1000
	// Total: 1200
	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic0, "Total stake amount for topic mismatch")

	// Check updated total stake for topic 1
	totalStakeForTopic1, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
	require.NoError(err)
	// Stake placed upon target: 1100
	// Total: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), totalStakeForTopic1, "Total stake amount for topic mismatch")

	// Check bond
	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, reputerAddr)
	require.NoError(err)
	// Stake placed upon target: 1100
	require.Equal(stakeAmount.Add(registrationInitialStake), bond, "Bond amount mismatch")

	// Remove stake from reputer (sender) to reputer (target)
	removeStakeMsg := &state.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputerAddr.String(),
				Amount: stakeAmount,
			},
		},
	}
	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.NoError(err, "StartRemoveStake should not return an error")

	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, reputerAddr, stakeAmountCoins)
	_, err = msgServer.ConfirmRemoveStake(ctx, &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	})
	require.NoError(err, "ConfirmRemoveStake should not return an error")

	// Check updated stake for delegator
	delegatorStake, err = s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)

	// Registration Stake: 100 - topic 0
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, delegatorStake, "Delegator stake amount mismatch")

	// Check updated stake for target
	targetStake, err = s.emissionsKeeper.GetStakePlacedUponTarget(ctx, reputerAddr)
	require.NoError(err)

	// Registration Stake: 100 - topic 0
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, targetStake, "Target stake amount mismatch")

	// Check updated total stake
	totalStake, err = s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)

	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 0
	// Total: 200
	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStake, "Total stake amount mismatch")

	// Check updated total stake for topic 0
	totalStakeForTopic0, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)

	// Registration Stake: 200 (100 for reputer, 100 for worker)
	// Stake placed upon target: 0
	// Total: 200
	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStakeForTopic0, "Total stake amount for topic mismatch")

	// Check updated total stake for topic 0
	totalStakeForTopic1, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
	require.NoError(err)

	// Registration Stake: 100 (reputer)
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, totalStakeForTopic1, "Total stake amount for topic mismatch")

	// Check bond
	bond, err = s.emissionsKeeper.GetBond(ctx, reputerAddr, reputerAddr)
	require.NoError(err)
	// Registration Stake: 100 (reputer)
	// Stake placed upon target: 0
	// Total: 100
	require.Equal(registrationInitialStake, bond, "Bond amount mismatch")
}

func (s *KeeperTestSuite) TestAddStakeInvalid() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Common setup for staking
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Scenario 1: Edge Case - Stake Amount Zero
	stakeAmountZero := cosmosMath.NewUint(0)
	stakeAmountZeroCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmountZero.BigInt())))
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountZeroCoins)
	_, err := msgServer.AddStake(ctx, &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmountZero,
	})
	require.Error(err, "Adding stake of zero should return an error")

	// Scenario 3: Incorrect Registrations - Unregistered Reputer
	stakeAmount := cosmosMath.NewUint(500)
	unregisteredReputerAddr := sdk.AccAddress(PKS[2].Address()) // unregistered delegator
	_, err = msgServer.AddStake(ctx, &state.MsgAddStake{
		Sender:      unregisteredReputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	})
	require.Error(err, "Adding stake from an unregistered reputer should return an error")

	// Scenario 4: Incorrect Registrations - Unregistered Reputer
	unregisteredWorkerAddr := sdk.AccAddress(PKS[3].Address()) // unregistered worker
	_, err = msgServer.AddStake(ctx, &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: unregisteredWorkerAddr.String(),
		Amount:      stakeAmount,
	})
	require.Error(err, "Adding stake from an unregistered reputer should return an error")
}

func (s *KeeperTestSuite) TestMsgStartRemoveStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	removalAmount := cosmosMath.NewUint(500)

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Add stake first to ensure there is something to remove
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	timeBefore := uint64(time.Now().UTC().Unix())

	// start a stake removal
	startRemoveMsg := &state.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: workerAddr.String(),
				Amount: removalAmount,
			},
		},
	}
	_, err = msgServer.StartRemoveStake(ctx, startRemoveMsg)

	// check the state has changed appropriately after the removal
	require.NoError(err, "StartRemoveStake should not return an error")
	removalInfo, err := s.emissionsKeeper.GetStakeRemovalQueueForDelegator(ctx, reputerAddr)
	require.NoError(err, "Stake removal queue should not be empty")
	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted, timeBefore, "Time should be valid starting")
	timeNow := uint64(time.Now().UTC().Unix())
	delayWindow, err := s.emissionsKeeper.GetRemoveStakeDelayWindow(ctx)
	s.Require().NoError(err)
	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted+delayWindow, timeNow, "Time should be valid ending")
	require.Equal(1, len(removalInfo.Placements), "There should be one placement in the removal queue")
	require.Equal(removalAmount, removalInfo.Placements[0].Amount, "The amount in the removal queue should be the same as the amount in the message")
	require.Equal(workerAddr.String(), removalInfo.Placements[0].Target, "The target in the removal queue should be the same as the target in the message")
}

func (s *KeeperTestSuite) TestMsgConfirmRemoveStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	removalAmount := cosmosMath.NewUint(500)
	removalAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(removalAmount.BigInt())))

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Add stake first to ensure there is something to remove
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// if you try to call the geniune msgServer.StartStakeRemoval
	// the unix time set there is going to be in the future,
	// rather than having to monkey patch the unix time, or some complicated mocking setup,
	// lets just directly manipulate the removalInfo in the keeper store
	timeNow := uint64(time.Now().UTC().Unix())
	err = s.emissionsKeeper.SetStakeRemovalQueueForDelegator(ctx, reputerAddr, state.StakeRemoval{
		TimestampRemovalStarted: timeNow - 1000,
		Placements: []*state.StakeRemovalPlacement{
			{
				TopicsIds: []uint64{0},
				Target:    workerAddr.String(),
				Amount:    removalAmount,
			},
		},
	})
	require.NoError(err, "Set stake removal queue should work")
	confirmRemoveStakeMsg := &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	}
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, reputerAddr, removalAmountCoins)
	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveStakeMsg)
	require.NoError(err, "ConfirmRemoveStake should not return an error")

	// Check updated stake for delegator after removal
	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)
	// Initial Stake: 100
	// Stake added: 1000
	// Stake removed: 500
	// Total: 600
	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch after removal")

	// Check updated stake for target after removal
	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
	require.NoError(err)
	// Initial Stake: 100
	// Stake added: 1000
	// Stake removed: 500
	// Total: 600
	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake), targetStake, "Target stake amount mismatch after removal")

	// Check updated total stake after removal
	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)
	// Initial Stake: 200 (100 for reputer, 100 for worker)
	// Stake added: 1000
	// Stake removed: 500
	// Total: 700
	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch after removal")

	// Check updated total stake for topic after removal
	totalStakeForTopic, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)
	// Initial Stake: 200 (100 for reputer, 100 for worker)
	// Stake added: 1000
	// Stake removed: 500
	// Total: 700
	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic, "Total stake amount for topic mismatch after removal")

	// Check bond after removal
	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
	require.NoError(err)
	// Stake placed upon target: 1000
	// Stake removed: 500
	// Total: 500
	require.Equal(stakeAmount.Sub(removalAmount), bond, "Bond amount mismatch after removal")
}

func (s *KeeperTestSuite) TestMsgStartRemoveAllStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	registrationInitialStake := cosmosMath.NewUint(100)

	// Common setup for staking
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Add stake first to ensure there is an initial stake
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Remove Registration
	removeRegistrationMsg := &state.MsgRemoveRegistration{
		Creator:   reputerAddr.String(),
		TopicId:   0,
		IsReputer: true,
	}
	_, err = msgServer.RemoveRegistration(ctx, removeRegistrationMsg)
	require.NoError(err, "RemoveRegistration should not return an error")

	// Remove all stake
	removeAllStakeMsg := &state.MsgStartRemoveAllStake{
		Sender: reputerAddr.String(),
	}

	timeBefore := uint64(time.Now().UTC().Unix())
	_, err = msgServer.StartRemoveAllStake(ctx, removeAllStakeMsg)

	// check the state has changed appropriately after the removal
	require.NoError(err, "StartRemoveAllStake should not return an error")
	removalInfo, err := s.emissionsKeeper.GetStakeRemovalQueueForDelegator(ctx, reputerAddr)
	require.NoError(err, "Stake removal queue should not be empty")
	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted, timeBefore, "Time should be valid starting")
	timeNow := uint64(time.Now().UTC().Unix())
	delayWindow, err := s.emissionsKeeper.GetRemoveStakeDelayWindow(ctx)
	s.Require().NoError(err)
	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted+delayWindow, timeNow, "Time should be valid ending")
	require.Equal(2, len(removalInfo.Placements), "There should be two placements in the removal queue")
	require.Equal(removalInfo.Placements[0].Target, workerAddr.String(), "The target in the removal queue should be the same as the target in the message")
	require.Equal(removalInfo.Placements[0].Amount, stakeAmount, "The amount in the removal queue should be the same as the amount in the message")
	require.Equal(removalInfo.Placements[1].Target, reputerAddr.String(), "The target in the removal queue should be the same as the target in the message")
	require.Equal(removalInfo.Placements[1].Amount, registrationInitialStake, "The amount in the removal queue should be the same as the amount in the message")
}

func (s *KeeperTestSuite) TestMsgConfirmRemoveAllStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	registrationInitialStake := cosmosMath.NewUint(100)
	registrationInitialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(registrationInitialStake.BigInt())))

	// Common setup for staking
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Add stake first to ensure there is an initial stake
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Remove all stake
	removeAllStakeMsg := &state.MsgStartRemoveAllStake{
		Sender: reputerAddr.String(),
	}

	// Remove registration
	removeRegistrationMsg := &state.MsgRemoveRegistration{
		Creator:   reputerAddr.String(),
		TopicId:   0,
		IsReputer: true,
	}
	_, err = msgServer.RemoveRegistration(ctx, removeRegistrationMsg)
	require.NoError(err, "RemoveRegistration should not return an error")

	_, err = msgServer.StartRemoveAllStake(ctx, removeAllStakeMsg)

	// check the state has changed appropriately after the removal
	require.NoError(err, "StartRemoveAllStake should not return an error")

	// swap out the timestamp so it's valid for the confirmRemove
	stakeRemoveInfo, err := s.emissionsKeeper.GetStakeRemovalQueueForDelegator(ctx, reputerAddr)
	require.NoError(err, "Stake removal queue should not be empty")
	stakeRemoveInfo.TimestampRemovalStarted = uint64(time.Now().UTC().Unix()) - 1000
	err = s.emissionsKeeper.SetStakeRemovalQueueForDelegator(ctx, reputerAddr, stakeRemoveInfo)
	require.NoError(err, "Set stake removal queue should work")

	confirmRemoveMsg := &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	}
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, reputerAddr, registrationInitialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, reputerAddr, stakeAmountCoins)
	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveMsg)
	require.NoError(err, "RemoveAllStake should not return an error")

	// Check that the sender's total stake is zero after removal
	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
	require.NoError(err)
	require.Equal(cosmosMath.ZeroUint(), delegatorStake, "delegator has zero stake after withdrawal")

	// Check that the target's stake is reduced by the stake amount
	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
	require.NoError(err)
	require.Equal(registrationInitialStake, targetStake, "Target's stake should be equal to the registration stake after removing all stake")

	// Check updated total stake after removal
	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err)
	require.Equal(registrationInitialStake, totalStake, "Total stake should be equal to the registration stakes after removing all stake")

	// Check updated total stake for topic after removal
	totalStakeForTopic, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
	require.NoError(err)
	require.Equal(registrationInitialStake, totalStakeForTopic, "Total stake for the topic should be equal to the registration stakes after removing all stake")
}

func (s *KeeperTestSuite) TestStartRemoveStakeInvalidRemoveMoreThanExists() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
	require.NoError(err, "Total stake should not be empty")

	// Scenario 1: Attempt to remove more stake than exists in the system
	removeStakeMsg := &state.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: workerAddr.String(),
				Amount: stakeAmount.Add(totalStake),
			},
		},
	}
	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.ErrorIs(err, state.ErrInsufficientStakeToRemove, "RemoveStake should return an error when attempting to remove more stake than exists")
}

func (s *KeeperTestSuite) TestStartRemoveStakeInvalidRemoveMoreThanMinimalAmountWhileBeingRegisteredInTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Attempt to remove more stake than exists in the system
	removeStakeMsg := &state.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputerAddr.String(),
				Amount: registrationInitialStake,
			},
		},
	}
	_, err := msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.ErrorIs(err, state.ErrInsufficientStakeAfterRemoval, "RemoveStake should return an error when attempting to remove more stake than exists")

	// Remove Registration
	removeRegistrationMsg := &state.MsgRemoveRegistration{
		Creator:   reputerAddr.String(),
		TopicId:   0,
		IsReputer: true,
	}
	_, err = msgServer.RemoveRegistration(ctx, removeRegistrationMsg)
	require.NoError(err, "RemoveRegistration should not return an error")

	// Successfully remove stake
	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.NoError(err, "RemoveStake should not return an error when removing all stake")
}

func (s *KeeperTestSuite) TestStartRemoveStakeInvalidNotEnoughStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Scenario 4: Attempt to remove stake when sender does not have enough stake placed on the target
	removeStakeMsg := &state.MsgStartRemoveStake{
		Sender: reputerAddr.String(),
		PlacementsRemove: []*state.StakePlacement{
			{

				Target: workerAddr.String(),
				Amount: stakeAmount,
			},
		},
	}
	_, err := msgServer.StartRemoveStake(ctx, removeStakeMsg)
	require.ErrorIs(err, state.ErrInsufficientStakeToRemove, "RemoveStake should return an error when sender does not have enough stake placed on the target")
}

func (s *KeeperTestSuite) TestConfirmRemoveStakeInvalidNoRemovalStarted() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	// workerAddr := sdk.AccAddress(PKS[1].Address())  // target

	// // Common setup for staking
	// registrationInitialStake := cosmosMath.NewUint(100)
	// s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	_, err := msgServer.ConfirmRemoveStake(ctx, &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	})
	require.ErrorIs(err, state.ErrConfirmRemoveStakeNoRemovalStarted, "ConfirmRemoveStake should return an error when no stake removal has been started")
}

func (s *KeeperTestSuite) TestConfirmRemoveStakeInvalidTooEarly() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// if you try to call the geniune msgServer.StartStakeRemoval
	// the unix time set there is going to be in the future,
	// rather than having to monkey patch the unix time, or some complicated mocking setup,
	// lets just directly manipulate the removalInfo in the keeper store
	timeNow := uint64(time.Now().UTC().Unix())
	err := s.emissionsKeeper.SetStakeRemovalQueueForDelegator(ctx, reputerAddr, state.StakeRemoval{
		TimestampRemovalStarted: timeNow + 1000000,
		Placements: []*state.StakeRemovalPlacement{
			{
				TopicsIds: []uint64{0},
				Target:    reputerAddr.String(),
				Amount:    registrationInitialStake,
			},
		},
	})
	require.NoError(err, "Set stake removal queue should work")
	confirmRemoveStakeMsg := &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	}
	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveStakeMsg)
	require.ErrorIs(err, state.ErrConfirmRemoveStakeTooEarly, "ConfirmRemoveStake should return an error when stake removal is too early")
}

func (s *KeeperTestSuite) TestConfirmRemoveStakeInvalidTooLate() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// if you try to call the geniune msgServer.StartStakeRemoval
	// the unix time set there is going to be in the future,
	// rather than having to monkey patch the unix time, or some complicated mocking setup,
	// lets just directly manipulate the removalInfo in the keeper store
	err := s.emissionsKeeper.SetStakeRemovalQueueForDelegator(ctx, reputerAddr, state.StakeRemoval{
		TimestampRemovalStarted: 0,
		Placements: []*state.StakeRemovalPlacement{
			{
				TopicsIds: []uint64{0},
				Target:    reputerAddr.String(),
				Amount:    registrationInitialStake,
			},
		},
	})
	require.NoError(err, "Set stake removal queue should work")
	confirmRemoveStakeMsg := &state.MsgConfirmRemoveStake{
		Sender: reputerAddr.String(),
	}
	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveStakeMsg)
	require.ErrorIs(err, state.ErrConfirmRemoveStakeTooLate, "ConfirmRemoveStake should return an error when stake removal is too early")
}

func (s *KeeperTestSuite) TestModifyStakeSimple() {
	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	reputer := reputerAddr.String()
	worker := workerAddr.String()

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

	// modify stake for reputer to put half of their stake in the worker
	response, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
		Sender: reputer,
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(50),
			},
		},
		PlacementsAdd: []*state.StakePlacement{
			{
				Target: worker,
				Amount: cosmosMath.NewUint(50),
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(&state.MsgModifyStakeResponse{}, response, "ModifyStake should return an empty response on success")

	// Check updated stake for delegator
	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(s.ctx, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewUint(100), delegatorStake, "Delegator stake should not change on modify stake")

	bond, err := s.emissionsKeeper.GetBond(s.ctx, reputerAddr, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewUint(50), bond, "Reputer bond amount mismatch")

	targetStake1, err := s.emissionsKeeper.GetStakePlacedUponTarget(s.ctx, reputerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewUint(50), targetStake1, "Reputer target stake amount mismatch")

	targetStake2, err := s.emissionsKeeper.GetStakePlacedUponTarget(s.ctx, workerAddr)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewUint(150), targetStake2, "Worker target stake amount mismatch")

}

func (s *KeeperTestSuite) TestModifyStakeInvalidSumChangesNotEqualRemoveMoreThanAdd() {
	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	reputer := reputerAddr.String()
	worker := workerAddr.String()

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

	// modify stake for reputer to put half of their stake in the worker
	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
		Sender: reputer,
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(60),
			},
		},
		PlacementsAdd: []*state.StakePlacement{
			{
				Target: worker,
				Amount: cosmosMath.NewUint(50),
			},
		},
	})
	s.Require().ErrorIs(err, state.ErrModifyStakeSumBeforeNotEqualToSumAfter)
}

func (s *KeeperTestSuite) TestModifyStakeInvalidSumChangesNotEqualAddMoreThanRemove() {
	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	reputer := reputerAddr.String()
	worker := workerAddr.String()

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

	// modify stake for reputer to put half of their stake in the worker
	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
		Sender: reputer,
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(50),
			},
		},
		PlacementsAdd: []*state.StakePlacement{
			{
				Target: worker,
				Amount: cosmosMath.NewUint(60),
			},
		},
	})
	s.Require().ErrorIs(err, state.ErrModifyStakeSumBeforeNotEqualToSumAfter)
}

func (s *KeeperTestSuite) TestModifyStakeInvalidNotHaveEnoughDelegatorStake() {
	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	reputer := reputerAddr.String()
	worker := workerAddr.String()

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

	// modify stake for reputer to put half of their stake in the worker
	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
		Sender: reputer,
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(50),
			},
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(50),
			},
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(50),
			},
		},
		PlacementsAdd: []*state.StakePlacement{
			{
				Target: worker,
				Amount: cosmosMath.NewUint(200),
			},
		},
	})
	s.Require().ErrorIs(err, state.ErrModifyStakeBeforeSumGreaterThanSenderStake)
}

func (s *KeeperTestSuite) TestModifyStakeInvalidNotHaveEnoughBond() {
	// do the normal setup, add more stake in a third party
	// then modify the stake but with more bond than the first party has on them
	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	reputer := reputerAddr.String()
	workerAddr1 := sdk.AccAddress(PKS[1].Address()) // target
	workerAddr2 := sdk.AccAddress(PKS[2].Address()) // target
	worker2 := workerAddr2.String()

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr1, registrationInitialStake)

	registrationInitialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.Int(registrationInitialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, registrationInitialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, registrationInitialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, reputerAddr, registrationInitialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, reputerAddr, registrationInitialStakeCoins)
	// Register Reputer
	worker2RegMsg := &state.MsgRegister{
		Creator:      worker2,
		LibP2PKey:    "test2",
		MultiAddress: "test2",
		TopicsIds:    []uint64{0},
		InitialStake: registrationInitialStake,
		Owner:        worker2,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), workerAddr2, state.AlloraStakingModuleName, registrationInitialStakeCoins)
	_, err := s.msgServer.Register(s.ctx, worker2RegMsg)
	s.Require().NoError(err, "Registering worker2 should not return an error")

	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	// Add stake from reputer to worker2
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputer,
		StakeTarget: worker2,
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingModuleName, stakeAmountCoins)
	_, err = s.msgServer.AddStake(s.ctx, addStakeMsg)
	s.Require().NoError(err, "AddStake should not return an error")
	bond, err := s.emissionsKeeper.GetBond(s.ctx, reputerAddr, workerAddr2)
	s.Require().NoError(err)
	s.Require().Equal(stakeAmount.BigInt(), bond.BigInt(), "Bond should have been added")

	// modify stake
	modifyStakeMsg := &state.MsgModifyStake{
		Sender: reputer,
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(300),
			},
			{
				Target: worker2,
				Amount: cosmosMath.NewUint(100),
			},
		},
		PlacementsAdd: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(200),
			},
			{
				Target: worker2,
				Amount: cosmosMath.NewUint(200),
			},
		},
	}
	_, err = s.msgServer.ModifyStake(s.ctx, modifyStakeMsg)
	s.Require().ErrorIs(err, state.ErrModifyStakeBeforeBondLessThanAmountModified, "ModifyStake Error not matching expected")
}

func (s *KeeperTestSuite) TestModifyStakeInvalidTarget() {
	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	reputer := reputerAddr.String()
	randoAddr := sdk.AccAddress(PKS[3].Address()) // delegator
	rando := randoAddr.String()

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

	// modify stake for reputer to put half of their stake in the worker
	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
		Sender: reputer,
		PlacementsRemove: []*state.StakePlacement{
			{
				Target: reputer,
				Amount: cosmosMath.NewUint(50),
			},
		},
		PlacementsAdd: []*state.StakePlacement{
			{
				Target: rando,
				Amount: cosmosMath.NewUint(50),
			},
		},
	})
	s.Require().ErrorIs(err, state.ErrAddressNotRegistered)
}

func (s *KeeperTestSuite) TestRequestInferenceSimple() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(s.ctx, senderAddr, state.AlloraRequestsModuleName, initialStakeCoins)
	response, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().Equal(&state.MsgRequestInferenceResponse{}, response, "RequestInference should return an empty response on success")

	// Check updated stake for delegator
	r0 := state.CreateNewInferenceRequestFromListItem(r.Sender, r.Requests[0])
	requestId, err := r0.GetRequestId()
	s.Require().NoError(err)
	storedRequest, err := s.emissionsKeeper.GetMempoolInferenceRequestById(s.ctx, 0, requestId)
	s.Require().NoError(err)
	// the last checked time is not set in the request, so we can't compare it
	// we can compare the rest of the fields
	s.Require().Equal(r0.Sender, storedRequest.Sender, "Stored request sender should match the request")
	s.Require().Equal(r0.Nonce, storedRequest.Nonce, "Stored request nonce should match the request")
	s.Require().Equal(r0.TopicId, storedRequest.TopicId, "Stored request topic id should match the request")
	s.Require().Equal(r0.Cadence, storedRequest.Cadence, "Stored request cadence should match the request")
	s.Require().Equal(r0.MaxPricePerInference, storedRequest.MaxPricePerInference, "Stored request max price per inference should match the request")
	s.Require().Equal(r0.BidAmount, storedRequest.BidAmount, "Stored request bid amount should match the request")
	s.Require().GreaterOrEqual(storedRequest.LastChecked, timeNow, "LastChecked should be greater than timeNow")
	s.Require().Equal(r0.TimestampValidUntil, storedRequest.TimestampValidUntil, "Stored request timestamp valid until should match the request")
	s.Require().Equal(r0.ExtraData, storedRequest.ExtraData, "Stored request extra data should match the request")
}

// test more than one inference in the message
func (s *KeeperTestSuite) TestRequestInferenceBatchSimple() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	var requestStake int64 = 500
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	requestStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(requestStake)))
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
			{
				Nonce:                1,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake)),
				TimestampValidUntil:  timeNow + 400,
				ExtraData:            nil,
			},
		},
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(s.ctx, senderAddr, state.AlloraRequestsModuleName, requestStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(s.ctx, senderAddr, state.AlloraRequestsModuleName, requestStakeCoins)
	response, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().Equal(&state.MsgRequestInferenceResponse{}, response, "RequestInference should return an empty response on success")

	// Check updated stake for delegator
	r0 := state.CreateNewInferenceRequestFromListItem(r.Sender, r.Requests[0])
	requestId, err := r0.GetRequestId()
	s.Require().NoError(err)
	storedRequest, err := s.emissionsKeeper.GetMempoolInferenceRequestById(s.ctx, 0, requestId)
	s.Require().NoError(err)
	s.Require().Equal(r0.Sender, storedRequest.Sender, "Stored request sender should match the request")
	s.Require().Equal(r0.Nonce, storedRequest.Nonce, "Stored request nonce should match the request")
	s.Require().Equal(r0.TopicId, storedRequest.TopicId, "Stored request topic id should match the request")
	s.Require().Equal(r0.Cadence, storedRequest.Cadence, "Stored request cadence should match the request")
	s.Require().Equal(r0.MaxPricePerInference, storedRequest.MaxPricePerInference, "Stored request max price per inference should match the request")
	s.Require().Equal(r0.BidAmount, storedRequest.BidAmount, "Stored request bid amount should match the request")
	s.Require().GreaterOrEqual(storedRequest.LastChecked, timeNow, "LastChecked should be greater than timeNow")
	s.Require().Equal(r0.TimestampValidUntil, storedRequest.TimestampValidUntil, "Stored request timestamp valid until should match the request")
	s.Require().Equal(r0.ExtraData, storedRequest.ExtraData, "Stored request extra data should match the request")
	r1 := state.CreateNewInferenceRequestFromListItem(r.Sender, r.Requests[1])
	requestId, err = r1.GetRequestId()
	s.Require().NoError(err)
	storedRequest, err = s.emissionsKeeper.GetMempoolInferenceRequestById(s.ctx, 0, requestId)
	s.Require().NoError(err)
	s.Require().Equal(r1.Sender, storedRequest.Sender, "Stored request sender should match the request")
	s.Require().Equal(r1.Nonce, storedRequest.Nonce, "Stored request nonce should match the request")
	s.Require().Equal(r1.TopicId, storedRequest.TopicId, "Stored request topic id should match the request")
	s.Require().Equal(r1.Cadence, storedRequest.Cadence, "Stored request cadence should match the request")
	s.Require().Equal(r1.MaxPricePerInference, storedRequest.MaxPricePerInference, "Stored request max price per inference should match the request")
	s.Require().Equal(r1.BidAmount, storedRequest.BidAmount, "Stored request bid amount should match the request")
	s.Require().GreaterOrEqual(storedRequest.LastChecked, timeNow, "LastChecked should be greater than timeNow")
	s.Require().Equal(r1.TimestampValidUntil, storedRequest.TimestampValidUntil, "Stored request timestamp valid until should match the request")
	s.Require().Equal(r1.ExtraData, storedRequest.ExtraData, "Stored request extra data should match the request")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidTopicDoesNotExist() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address()).String()
	r := state.MsgRequestInference{
		Sender: senderAddr,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(100),
				BidAmount:            cosmosMath.NewUint(100),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInvalidTopicId, "RequestInference should return an error when the topic does not exist")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidBidAmountNotEnoughForPriceSet() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake + 20)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestBidAmountLessThanPrice, "RequestInference should return an error when the bid amount is less than the price")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidSendSameRequestTwice() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(s.ctx, senderAddr, state.AlloraRequestsModuleName, initialStakeCoins)
	s.msgServer.RequestInference(s.ctx, &r)

	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestAlreadyInMempool, "RequestInference should return an error when the request already exists")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestInThePast() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow - 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestTimestampValidUntilInPast, "RequestInference should return an error when the request timestamp is in the past")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestTooFarInFuture() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  math.MaxUint64,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestTimestampValidUntilTooFarInFuture, "RequestInference should return an error when the request timestamp is too far in the future")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestCadenceHappensAfterNoLongerValid() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              1000,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestWillNeverBeScheduled, "RequestInference should return an error when the request cadence happens after the request is no longer valid")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestCadenceTooFast() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              1,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestCadenceTooFast, "RequestInference should return an error when the request cadence is too fast")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestCadenceTooSlow() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              math.MaxUint64,
				MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
				BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestCadenceTooSlow, "RequestInference should return an error when the request cadence is too slow")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidBidAmountLessThanGlobalMinimum() {
	timeNow := uint64(time.Now().UTC().Unix())
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, senderAddr, initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: sender,
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              0,
				Cadence:              0,
				MaxPricePerInference: cosmosMath.ZeroUint(),
				BidAmount:            cosmosMath.ZeroUint(),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, state.ErrInferenceRequestBidAmountTooLow, "RequestInference should return an error when the bid amount is below global minimum threshold")
}

// ########################################
// #           Whitelist tests            #
// ########################################

func (s *KeeperTestSuite) TestAddWhitelistAdmin() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := sdk.AccAddress(PKS[0].Address())
	newAdminAddr := nonAdminAccounts[0]

	// Verify that newAdminAddr is not a whitelist admin
	isWhitelistAdmin, err := s.emissionsKeeper.IsWhitelistAdmin(ctx, newAdminAddr)
	require.NoError(err, "IsWhitelistAdmin should not return an error")
	require.False(isWhitelistAdmin, "newAdminAddr should not be a whitelist admin")

	// Attempt to add newAdminAddr to whitelist by adminAddr
	msg := &state.MsgAddToWhitelistAdmin{
		Sender:  adminAddr.String(),
		Address: newAdminAddr.String(),
	}

	_, err = msgServer.AddToWhitelistAdmin(ctx, msg)
	require.NoError(err, "Adding to whitelist admin should succeed")

	// Verify that newAdminAddr is now a whitelist admin
	isWhitelistAdmin, err = s.emissionsKeeper.IsWhitelistAdmin(ctx, newAdminAddr)
	require.NoError(err, "IsWhitelistAdmin should not return an error")
	require.True(isWhitelistAdmin, "newAdminAddr should be a whitelist admin")
}

func (s *KeeperTestSuite) TestAddWhitelistAdminInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	targetAddr := sdk.AccAddress(PKS[1].Address())

	// Attempt to add targetAddr to whitelist by nonAdminAddr
	msg := &state.MsgAddToWhitelistAdmin{
		Sender:  nonAdminAddr.String(),
		Address: targetAddr.String(),
	}

	_, err := s.msgServer.AddToWhitelistAdmin(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Should fail due to unauthorized access")
}

func (s *KeeperTestSuite) TestRemoveWhitelistAdmin() {
	ctx := s.ctx
	require := s.Require()
	msgServer := s.msgServer

	adminAddr := sdk.AccAddress(PKS[0].Address())
	adminToRemove := sdk.AccAddress(PKS[1].Address())

	// Attempt to remove adminToRemove from the whitelist by adminAddr
	removeMsg := &state.MsgRemoveFromWhitelistAdmin{
		Sender:  adminAddr.String(),
		Address: adminToRemove.String(),
	}
	_, err := msgServer.RemoveFromWhitelistAdmin(ctx, removeMsg)
	require.NoError(err, "Removing from whitelist admin should succeed")

	// Verify that adminToRemove is no longer a whitelist admin
	isWhitelistAdmin, err := s.emissionsKeeper.IsWhitelistAdmin(ctx, adminToRemove)
	require.NoError(err, "IsWhitelistAdmin check should not return an error")
	fmt.Println(isWhitelistAdmin)
	require.False(isWhitelistAdmin, "adminToRemove should not be a whitelist admin anymore")
}

func (s *KeeperTestSuite) TestRemoveWhitelistAdminInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]

	// Attempt to remove an admin from whitelist by nonAdminAddr
	msg := &state.MsgRemoveFromWhitelistAdmin{
		Sender:  nonAdminAddr.String(),
		Address: Addr.String(),
	}

	_, err := s.msgServer.RemoveFromWhitelistAdmin(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Should fail due to unauthorized access")
}

func (s *KeeperTestSuite) TestAddToTopicCreationWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	newAddr := nonAdminAccounts[0]

	// Attempt to add newAddr to the topic creation whitelist by adminAddr
	msg := &state.MsgAddToTopicCreationWhitelist{
		Sender:  adminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToTopicCreationWhitelist(ctx, msg)
	require.NoError(err, "Adding to topic creation whitelist should succeed")

	// Verify newAddr is now in the topic creation whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInTopicCreationWhitelist(ctx, newAddr)
	require.NoError(err, "IsInTopicCreationWhitelist should not return an error")
	require.True(isInWhitelist, "newAddr should be in the topic creation whitelist")
}

func (s *KeeperTestSuite) TestAddToTopicCreationWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	newAddr := nonAdminAccounts[1]

	// Attempt to add addressToAdd to the topic creation whitelist by nonAdminAddr
	msg := &state.MsgAddToTopicCreationWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToTopicCreationWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to add to the topic creation whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromTopicCreationWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	addressToRemove := sdk.AccAddress(PKS[1].Address())

	// Attempt to remove addressToRemove from the topic creation whitelist by adminAddr
	removeFromWhitelistMsg := &state.MsgRemoveFromTopicCreationWhitelist{
		Sender:  adminAddr.String(),
		Address: addressToRemove.String(),
	}
	_, err := s.msgServer.RemoveFromTopicCreationWhitelist(ctx, removeFromWhitelistMsg)
	require.NoError(err, "Removing from topic creation whitelist should succeed")

	// Verify if addressToRemove is no longer in the topic creation whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInTopicCreationWhitelist(ctx, addressToRemove)
	require.NoError(err, "IsInTopicCreationWhitelist check should not return an error")
	require.False(isInWhitelist, "addressToRemove should no longer be in the topic creation whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromTopicCreationWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	addressToRemove := nonAdminAccounts[1]

	// Attempt to remove addressToRemove from the topic creation whitelist by nonAdminAddr
	msg := &state.MsgRemoveFromTopicCreationWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: addressToRemove.String(),
	}

	_, err := s.msgServer.RemoveFromTopicCreationWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to remove from the topic creation whitelist")
}

func (s *KeeperTestSuite) TestAddToWeightSettingWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	newAddr := nonAdminAccounts[0]

	// Attempt to add newAddr to the weight setting whitelist by adminAddr
	msg := &state.MsgAddToWeightSettingWhitelist{
		Sender:  adminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToWeightSettingWhitelist(ctx, msg)
	require.NoError(err, "Adding to weight setting whitelist should succeed")

	// Verify if newAddr is now in the weight setting whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInWeightSettingWhitelist(ctx, newAddr)
	require.NoError(err, "IsInWeightSettingWhitelist check should not return an error")
	require.True(isInWhitelist, "newAddr should be in the weight setting whitelist")
}

func (s *KeeperTestSuite) TestAddToWeightSettingWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	newAddr := nonAdminAccounts[1]

	// Attempt to add addressToAdd to the weight setting whitelist by nonAdminAddr
	msg := &state.MsgAddToWeightSettingWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: newAddr.String(),
	}

	_, err := s.msgServer.AddToWeightSettingWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to add to the weight setting whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromWeightSettingWhitelist() {
	ctx := s.ctx
	require := s.Require()

	adminAddr := sdk.AccAddress(PKS[0].Address())
	addressToRemove := sdk.AccAddress(PKS[1].Address())

	// Attempt to remove addressToRemove from the weight setting whitelist by adminAddr
	removeFromWhitelistMsg := &state.MsgRemoveFromWeightSettingWhitelist{
		Sender:  adminAddr.String(),
		Address: addressToRemove.String(),
	}
	_, err := s.msgServer.RemoveFromWeightSettingWhitelist(ctx, removeFromWhitelistMsg)
	require.NoError(err, "Removing from weight setting whitelist should succeed")

	// Verify if addressToRemove is no longer in the weight setting whitelist
	isInWhitelist, err := s.emissionsKeeper.IsInWeightSettingWhitelist(ctx, addressToRemove)
	require.NoError(err, "IsInWeightSettingWhitelist check should not return an error")
	require.False(isInWhitelist, "addressToRemove should no longer be in the weight setting whitelist")
}

func (s *KeeperTestSuite) TestRemoveFromWeightSettingWhitelistInvalidUnauthorized() {
	ctx := s.ctx
	require := s.Require()

	nonAdminAddr := nonAdminAccounts[0]
	addressToRemove := nonAdminAccounts[1]

	// Attempt to remove addressToRemove from the weight setting whitelist by nonAdminAddr
	msg := &state.MsgRemoveFromWeightSettingWhitelist{
		Sender:  nonAdminAddr.String(),
		Address: addressToRemove.String(),
	}

	_, err := s.msgServer.RemoveFromWeightSettingWhitelist(ctx, msg)
	require.ErrorIs(err, state.ErrNotWhitelistAdmin, "Non-admin should not be able to remove from the weight setting whitelist")
}
