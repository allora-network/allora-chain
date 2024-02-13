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
	PKS     = simtestutil.CreateTestPubKeys(4)
	Addr    = sdk.AccAddress(PKS[0].Address())
	ValAddr = sdk.ValAddress(Addr)
)

// ########################################
// #           Topics tests              #
// ########################################

func (s *KeeperTestSuite) TestMsgSetWeights() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()).String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetWeights message
	weightMsg := &state.MsgSetWeights{
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

func (s *KeeperTestSuite) TestMsgSetLatestTimestampsInference() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for topic
	topicId := uint64(1)
	inferenceTs := uint64(time.Now().UTC().Unix())

	// Create a MsgSetInferences message
	inferencesMsg := &state.MsgSetLatestInferencesTimestamp{
		TopicId:            topicId,
		InferenceTimestamp: inferenceTs,
	}

	_, err := msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)
	require.NoError(err, "SetLatestTimestampInferences should not return an error")

	result, err := s.emissionsKeeper.GetLatestInferenceTimestamp(s.ctx, topicId)
	fmt.Printf("The timestamp value is %d.\n", result)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, inferenceTs)
}

func (s *KeeperTestSuite) TestProcessInferencesAndQuery() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for inferences
	inferences := []*state.Inference{
		{TopicId: 1, Worker: "worker1", Value: cosmosMath.NewUint(2200)},
		{TopicId: 1, Worker: "worker2", Value: cosmosMath.NewUint(2100)},
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
	inferencesMsg := &state.MsgSetLatestInferencesTimestamp{
		TopicId:            uint64(1),
		InferenceTimestamp: 1500000000,
	}
	_, err = msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)
	require.NoError(err, "Setting latest inference timestamp should not fail")

	allInferences, err := s.emissionsKeeper.GetLatestInferencesFromTopic(ctx, uint64(1))
	require.Equal(len(allInferences), 1)
	for _, inference := range allInferences {
		require.Equal(len(inference.Inferences.Inferences), 2)
	}
	require.NoError(err, "Inferences over ts threshold should be returned")

	/*
	 * Inferences under threshold should not be returned
	 */
	// Ensure highest ts for topic 1
	inferencesMsg = &state.MsgSetLatestInferencesTimestamp{
		TopicId:            uint64(1),
		InferenceTimestamp: math.MaxUint64,
	}
	_, err = msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)
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
		Active:           true,
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
	registrationInitialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(registrationInitialStake.BigInt())))

	// Create Topic
	newTopicMsg := &state.MsgCreateNewTopic{
		Creator:          reputerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		Active:           true,
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

	// Register Reputer
	reputerRegMsg := &state.MsgRegisterReputer{
		Creator:      reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      0,
		InitialStake: registrationInitialStake,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.ModuleName, registrationInitialStakeCoins)
	_, err = msgServer.RegisterReputer(ctx, reputerRegMsg)
	require.NoError(err, "Registering reputer should not return an error")

	// Register Worker
	workerRegMsg := &state.MsgRegisterWorker{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      0,
		InitialStake: registrationInitialStake,
		Owner:        workerAddr.String(),
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), workerAddr, state.ModuleName, registrationInitialStakeCoins)
	_, err = msgServer.RegisterWorker(ctx, workerRegMsg)
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
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.ModuleName, stakeAmountCoins)
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
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.ModuleName, stakeAmountZeroCoins)
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

/*

func (s *KeeperTestSuite) TestMsgRemoveStake() {
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
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.ModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Remove stake from reputer (sender) to worker (target)
	removeStakeMsg := &state.MsgRemoveStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      removalAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.ModuleName, reputerAddr, removalAmountCoins)
	_, err = msgServer.RemoveStake(ctx, removeStakeMsg)
	require.NoError(err, "RemoveStake should not return an error")

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

func (s *KeeperTestSuite) TestRemoveStakeInvalid() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Scenario 1: Attempt to remove more stake than exists
	removeStakeMsg := &state.MsgRemoveStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	_, err := msgServer.RemoveStake(ctx, removeStakeMsg)
	require.Error(err, "RemoveStake should return an error when attempting to remove more stake than exists")

	// Scenario 2: Attempt to remove stake from unregistered target
	unregisteredWorkerAddr := sdk.AccAddress(PKS[2].Address()) // unregistered target
	removeStakeMsg = &state.MsgRemoveStake{
		Sender:      reputerAddr.String(),
		StakeTarget: unregisteredWorkerAddr.String(),
		Amount:      registrationInitialStake,
	}
	_, err = msgServer.RemoveStake(ctx, removeStakeMsg)
	require.Error(err, "RemoveStake should return an error when attempting to remove stake from a unregistered target")

	// Scenario 3: Attempt to remove stake as a unregistered sender
	unregisteredReputerAddr := sdk.AccAddress(PKS[3].Address()) // unregistered sender
	removeStakeMsg = &state.MsgRemoveStake{
		Sender:      unregisteredReputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      registrationInitialStake,
	}
	_, err = msgServer.RemoveStake(ctx, removeStakeMsg)
	require.Error(err, "RemoveStake should return an error when attempting to remove stake as a unregistered sender")

	// Scenario 4: Attempt to remove stake when sender does not have enough stake placed on the target
	removeStakeMsg = &state.MsgRemoveStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	_, err = msgServer.RemoveStake(ctx, removeStakeMsg)
	require.Error(err, "RemoveStake should return an error when sender does not have enough stake placed on the target")
}

func (s *KeeperTestSuite) TestMsgRemoveAllStake() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	registrationInitialStake := cosmosMath.NewUint(100)
	senderTotalStake := stakeAmount.Add(registrationInitialStake) // Total stake including registration stake
	senderTotalStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(senderTotalStake.BigInt())))

	// Common setup for staking
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Add stake first to ensure there is an initial stake
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputerAddr.String(),
		StakeTarget: workerAddr.String(),
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.ModuleName, stakeAmountCoins)
	_, err := msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")

	// Remove all stake
	removeAllStakeMsg := &state.MsgRemoveAllStake{
		Sender: reputerAddr.String(),
	}
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.ModuleName, reputerAddr, senderTotalStakeCoins)
	_, err = msgServer.RemoveAllStake(ctx, removeAllStakeMsg)
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

func (s *KeeperTestSuite) TestRemoveAllStakeInvalid() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

	// Common setup for staking
	registrationInitialStake := cosmosMath.NewUint(100)
	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

	// Scenario 1: Attempt to remove all stake as an unregistered sender
	unregisteredReputerAddr := sdk.AccAddress(PKS[2].Address()) // unregistered sender
	removeAllStakeMsg := &state.MsgRemoveAllStake{
		Sender: unregisteredReputerAddr.String(),
	}
	_, err := msgServer.RemoveAllStake(ctx, removeAllStakeMsg)
	require.Error(err, "Scenario 1: RemoveAllStake should return an error when attempting to remove all stake as an unregistered sender")

	// Scenario 2: Attempt to remove all stake when sender has no stake
	noStakeSenderAddr := sdk.AccAddress(PKS[3].Address()) // target
	removeAllStakeMsg = &state.MsgRemoveAllStake{
		Sender: noStakeSenderAddr.String(),
	}
	_, err = msgServer.RemoveAllStake(ctx, removeAllStakeMsg)
	require.Error(err, "Scenario 2: RemoveAllStake should return an error when sender has no stake")
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
	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.ModuleName, registrationInitialStakeCoins)
	s.bankKeeper.MintCoins(s.ctx, state.ModuleName, registrationInitialStakeCoins)
	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.ModuleName, reputerAddr, registrationInitialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.ModuleName, reputerAddr, registrationInitialStakeCoins)
	// Register Reputer
	worker2RegMsg := &state.MsgRegisterWorker{
		Creator:      worker2,
		LibP2PKey:    "test2",
		MultiAddress: "test2",
		TopicId:      0,
		InitialStake: registrationInitialStake,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), workerAddr2, state.ModuleName, registrationInitialStakeCoins)
	_, err := s.msgServer.RegisterWorker(s.ctx, worker2RegMsg)
	s.Require().NoError(err, "Registering worker2 should not return an error")

	stakeAmount := cosmosMath.NewUint(1000)
	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
	// Add stake from reputer to worker2
	addStakeMsg := &state.MsgAddStake{
		Sender:      reputer,
		StakeTarget: worker2,
		Amount:      stakeAmount,
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.ModuleName, stakeAmountCoins)
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
	fmt.Println("Reputer is ", reputer)
	randoAddr := sdk.AccAddress(PKS[3].Address()) // delegator
	rando := randoAddr.String()
	fmt.Println("Rando is ", rando)
	fmt.Println("Worker is ", workerAddr.String())

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

*/
