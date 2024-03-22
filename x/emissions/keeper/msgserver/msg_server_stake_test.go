package msgserver_test

import (
	// "time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
)

func (s *KeeperTestSuite) commonStakingSetup(ctx sdk.Context, reputerAddr sdk.AccAddress, workerAddr sdk.AccAddress, registrationInitialStake cosmosMath.Uint) {
	msgServer := s.msgServer
	require := s.Require()
	registrationInitialStakeCoins := sdk.NewCoins(
		sdk.NewCoin(
			params.DefaultBondDenom,
			cosmosMath.NewIntFromBigInt(registrationInitialStake.BigInt())))

	// Create Topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          reputerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		LossLogic:        "logic",
		LossCadence:      10800,
		InferenceLogic:   "Ilogic",
		InferenceMethod:  "Imethod",
		InferenceCadence: 60,
		DefaultArg:       "ETH",
	}
	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")

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
	_, err = msgServer.Register(ctx, reputerRegMsg)
	require.NoError(err, "Registering reputer should not return an error")

	// Register Worker
	workerRegMsg := &types.MsgRegister{
		Creator:      workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicIds:     []uint64{0},
		InitialStake: registrationInitialStake,
		Owner:        workerAddr.String(),
	}
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), workerAddr, types.AlloraStakingAccountName, registrationInitialStakeCoins)
	_, err = msgServer.Register(ctx, workerRegMsg)
	require.NoError(err, "Registering worker should not return an error")
}

// func (s *KeeperTestSuite) TestMsgAddStake() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Add stake from reputer (sender) to worker (target)
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err := msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	// Check updated stake for delegator
// 	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100
// 	// Stake placed upon target: 1000
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch")

// 	// Check updated stake for target
// 	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100
// 	// Stake placed upon target: 1000
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), targetStake, "Target stake amount mismatch")

// 	// Check updated total stake
// 	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)
// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 1000
// 	// Total: 1200
// 	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch")

// 	// Check updated total stake for topic
// 	totalStakeForTopic, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)
// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 1000
// 	// Total: 1200
// 	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic, "Total stake amount for topic mismatch")

// 	// Check bond
// 	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
// 	require.NoError(err)
// 	// Stake placed upon target: 1000
// 	require.Equal(stakeAmount, bond, "Bond amount mismatch")
// }

// func (s *KeeperTestSuite) TestMsgAddAndRemoveStakeWithTargetWorkerRegisteredInMultipleTopics() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Create topic and register target (worker) in additional topic (topic 1)
// 	// Create Topic 1
// 	newTopicMsg := &state.MsgCreateNewTopic{
// 		Creator:          workerAddr.String(),
// 		Metadata:         "Some metadata for the new topic",
// 		WeightLogic:      "logic",
// 		WeightCadence:    10800,
// 		InferenceLogic:   "Ilogic",
// 		InferenceMethod:  "Imethod",
// 		InferenceCadence: 60,
// 	}
// 	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
// 	require.NoError(err, "CreateTopic fails on creation")

// 	// Register Worker in topic 1
// 	workerAddRegMsg := &state.MsgAddNewRegistration{
// 		Creator:      workerAddr.String(),
// 		LibP2PKey:    "test",
// 		MultiAddress: "test",
// 		TopicId:      1,
// 		Owner:        workerAddr.String(),
// 	}
// 	_, err = msgServer.AddNewRegistration(ctx, workerAddRegMsg)
// 	require.NoError(err, "Registering worker should not return an error")

// 	// Add stake from reputer (sender) to worker (target)
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err = msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	// Check updated stake for delegator
// 	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100
// 	// Stake placed upon target: 1000
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch")

// 	// Check updated stake for target
// 	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100 - first registration - topic 0
// 	// Stake placed upon target: 1000
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), targetStake, "Target stake amount mismatch")

// 	// Check updated total stake
// 	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)
// 	// Registration Stake: 200 (100 for reputer, 100 for worker) - topic 0
// 	// Stake placed upon target: 1000
// 	// Total: 1200
// 	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch")

// 	// Check updated total stake for topic 0
// 	totalStakeForTopic0, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)
// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 1000
// 	// Total: 1200
// 	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic0, "Total stake amount for topic mismatch")

// 	// Check updated total stake for topic 1
// 	totalStakeForTopic1, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
// 	require.NoError(err)
// 	// Stake placed upon target: 1100
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), totalStakeForTopic1, "Total stake amount for topic mismatch")

// 	// Check bond
// 	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
// 	require.NoError(err)
// 	// Stake placed upon target: 1000
// 	require.Equal(stakeAmount, bond, "Bond amount mismatch")

// 	// Remove stake from reputer (sender) to worker (target)
// 	removeStakeMsg := &state.MsgStartRemoveStake{
// 		Sender: reputerAddr.String(),
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: workerAddr.String(),
// 				Amount: stakeAmount,
// 			},
// 		},
// 	}
// 	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
// 	require.NoError(err, "StartRemoveStake should not return an error")

// 	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingAccountName, reputerAddr, stakeAmountCoins)
// 	_, err = msgServer.ConfirmRemoveStake(ctx, &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	})
// 	require.NoError(err, "ConfirmRemoveStake should not return an error")

// 	// Check updated stake for delegator
// 	delegatorStake, err = s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)

// 	// Registration Stake: 100 - topic 0
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, delegatorStake, "Delegator stake amount mismatch")

// 	// Check updated stake for target
// 	targetStake, err = s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
// 	require.NoError(err)

// 	// Registration Stake: 100 - topic 0
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, targetStake, "Target stake amount mismatch")

// 	// Check updated total stake
// 	totalStake, err = s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)

// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 0
// 	// Total: 200
// 	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStake, "Total stake amount mismatch")

// 	// Check updated total stake for topic 0
// 	totalStakeForTopic0, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)

// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 0
// 	// Total: 200
// 	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStakeForTopic0, "Total stake amount for topic mismatch")

// 	// Check updated total stake for topic 0
// 	totalStakeForTopic1, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
// 	require.NoError(err)

// 	// Registration Stake: 100 (worker)
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, totalStakeForTopic1, "Total stake amount for topic mismatch")

// 	// Check bond
// 	bond, err = s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
// 	require.NoError(err)
// 	// Stake placed upon target: 0
// 	require.Equal(cosmosMath.ZeroUint(), bond, "Bond amount mismatch")
// }

// func (s *KeeperTestSuite) TestMsgAddAndRemoveStakeWithTargetReputerRegisteredInMultipleTopics() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Create topic and register target (worker) in additional topic (topic 1)
// 	// Create Topic 1
// 	newTopicMsg := &state.MsgCreateNewTopic{
// 		Creator:          workerAddr.String(),
// 		Metadata:         "Some metadata for the new topic",
// 		WeightLogic:      "logic",
// 		WeightCadence:    10800,
// 		InferenceLogic:   "Ilogic",
// 		InferenceMethod:  "Imethod",
// 		InferenceCadence: 60,
// 	}
// 	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
// 	require.NoError(err, "CreateTopic fails on creation")

// 	// Register Reputer in topic 1
// 	reputerAddRegMsg := &state.MsgAddNewRegistration{
// 		Creator:      reputerAddr.String(),
// 		LibP2PKey:    "test",
// 		MultiAddress: "test",
// 		TopicId:      1,
// 		IsReputer:    true,
// 	}
// 	_, err = msgServer.AddNewRegistration(ctx, reputerAddRegMsg)
// 	require.NoError(err, "Registering reputer should not return an error")

// 	// Add stake from reputer (sender) to reputer (target)
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: reputerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err = msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	// Check updated stake for delegator
// 	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100
// 	// Stake placed upon target: 1000
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch")

// 	// Check updated stake for target
// 	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, reputerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100 - first registration - topic 0
// 	// Stake placed upon target: 1000
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), targetStake, "Target stake amount mismatch")

// 	// Check updated total stake
// 	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)
// 	// Registration Stake: 200 (100 for reputer, 100 for worker) - topic 0
// 	// Stake placed upon target: 1000
// 	// Total: 1200
// 	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch")

// 	// Check updated total stake for topic 0
// 	totalStakeForTopic0, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)
// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 1000
// 	// Total: 1200
// 	require.Equal(stakeAmount.Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic0, "Total stake amount for topic mismatch")

// 	// Check updated total stake for topic 1
// 	totalStakeForTopic1, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
// 	require.NoError(err)
// 	// Stake placed upon target: 1100
// 	// Total: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), totalStakeForTopic1, "Total stake amount for topic mismatch")

// 	// Check bond
// 	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, reputerAddr)
// 	require.NoError(err)
// 	// Stake placed upon target: 1100
// 	require.Equal(stakeAmount.Add(registrationInitialStake), bond, "Bond amount mismatch")

// 	// Remove stake from reputer (sender) to reputer (target)
// 	removeStakeMsg := &state.MsgStartRemoveStake{
// 		Sender: reputerAddr.String(),
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputerAddr.String(),
// 				Amount: stakeAmount,
// 			},
// 		},
// 	}
// 	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
// 	require.NoError(err, "StartRemoveStake should not return an error")

// 	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingAccountName, reputerAddr, stakeAmountCoins)
// 	_, err = msgServer.ConfirmRemoveStake(ctx, &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	})
// 	require.NoError(err, "ConfirmRemoveStake should not return an error")

// 	// Check updated stake for delegator
// 	delegatorStake, err = s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)

// 	// Registration Stake: 100 - topic 0
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, delegatorStake, "Delegator stake amount mismatch")

// 	// Check updated stake for target
// 	targetStake, err = s.emissionsKeeper.GetStakePlacedUponTarget(ctx, reputerAddr)
// 	require.NoError(err)

// 	// Registration Stake: 100 - topic 0
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, targetStake, "Target stake amount mismatch")

// 	// Check updated total stake
// 	totalStake, err = s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)

// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 0
// 	// Total: 200
// 	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStake, "Total stake amount mismatch")

// 	// Check updated total stake for topic 0
// 	totalStakeForTopic0, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)

// 	// Registration Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake placed upon target: 0
// 	// Total: 200
// 	require.Equal(registrationInitialStake.Mul(cosmosMath.NewUint(2)), totalStakeForTopic0, "Total stake amount for topic mismatch")

// 	// Check updated total stake for topic 0
// 	totalStakeForTopic1, err = s.emissionsKeeper.GetTopicStake(ctx, uint64(1))
// 	require.NoError(err)

// 	// Registration Stake: 100 (reputer)
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, totalStakeForTopic1, "Total stake amount for topic mismatch")

// 	// Check bond
// 	bond, err = s.emissionsKeeper.GetBond(ctx, reputerAddr, reputerAddr)
// 	require.NoError(err)
// 	// Registration Stake: 100 (reputer)
// 	// Stake placed upon target: 0
// 	// Total: 100
// 	require.Equal(registrationInitialStake, bond, "Bond amount mismatch")
// }

// func (s *KeeperTestSuite) TestAddStakeInvalid() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Common setup for staking
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Scenario 1: Edge Case - Stake Amount Zero
// 	stakeAmountZero := cosmosMath.NewUint(0)
// 	stakeAmountZeroCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmountZero.BigInt())))
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountZeroCoins)
// 	_, err := msgServer.AddStake(ctx, &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmountZero,
// 	})
// 	require.Error(err, "Adding stake of zero should return an error")

// 	// Scenario 3: Incorrect Registrations - Unregistered Reputer
// 	stakeAmount := cosmosMath.NewUint(500)
// 	unregisteredReputerAddr := sdk.AccAddress(PKS[2].Address()) // unregistered delegator
// 	_, err = msgServer.AddStake(ctx, &state.MsgAddStake{
// 		Sender:      unregisteredReputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	})
// 	require.Error(err, "Adding stake from an unregistered reputer should return an error")

// 	// Scenario 4: Incorrect Registrations - Unregistered Reputer
// 	unregisteredWorkerAddr := sdk.AccAddress(PKS[3].Address()) // unregistered worker
// 	_, err = msgServer.AddStake(ctx, &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: unregisteredWorkerAddr.String(),
// 		Amount:      stakeAmount,
// 	})
// 	require.Error(err, "Adding stake from an unregistered reputer should return an error")
// }

// func (s *KeeperTestSuite) TestMsgStartRemoveStake() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
// 	removalAmount := cosmosMath.NewUint(500)

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Add stake first to ensure there is something to remove
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err := msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	timeBefore := uint64(time.Now().UTC().Unix())

// 	// start a stake removal
// 	startRemoveMsg := &state.MsgStartRemoveStake{
// 		Sender: reputerAddr.String(),
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: workerAddr.String(),
// 				Amount: removalAmount,
// 			},
// 		},
// 	}
// 	_, err = msgServer.StartRemoveStake(ctx, startRemoveMsg)

// 	// check the state has changed appropriately after the removal
// 	require.NoError(err, "StartRemoveStake should not return an error")
// 	removalInfo, err := s.emissionsKeeper.GetStakeRemovalQueueByAddress(ctx, reputerAddr)
// 	require.NoError(err, "Stake removal queue should not be empty")
// 	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted, timeBefore, "Time should be valid starting")
// 	timeNow := uint64(time.Now().UTC().Unix())
// 	delayWindow, err := s.emissionsKeeper.GetParamsRemoveStakeDelayWindow(ctx)
// 	s.Require().NoError(err)
// 	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted+delayWindow, timeNow, "Time should be valid ending")
// 	require.Equal(1, len(removalInfo.Placements), "There should be one placement in the removal queue")
// 	require.Equal(removalAmount, removalInfo.Placements[0].Amount, "The amount in the removal queue should be the same as the amount in the message")
// 	require.Equal(workerAddr.String(), removalInfo.Placements[0].Target, "The target in the removal queue should be the same as the target in the message")
// }

// func (s *KeeperTestSuite) TestMsgConfirmRemoveStake() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
// 	removalAmount := cosmosMath.NewUint(500)
// 	removalAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(removalAmount.BigInt())))

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Add stake first to ensure there is something to remove
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err := msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	// if you try to call the geniune msgServer.StartStakeRemoval
// 	// the unix time set there is going to be in the future,
// 	// rather than having to monkey patch the unix time, or some complicated mocking setup,
// 	// lets just directly manipulate the removalInfo in the keeper store
// 	timeNow := uint64(time.Now().UTC().Unix())
// 	err = s.emissionsKeeper.SetStakeRemovalQueueForAddress(ctx, reputerAddr, state.StakeRemoval{
// 		TimestampRemovalStarted: timeNow - 1000,
// 		Placements: []*state.StakeRemovalPlacement{
// 			{
// 				TopicIds: []uint64{0},
// 				Target:   workerAddr.String(),
// 				Amount:   removalAmount,
// 			},
// 		},
// 	})
// 	require.NoError(err, "Set stake removal queue should work")
// 	confirmRemoveStakeMsg := &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingAccountName, reputerAddr, removalAmountCoins)
// 	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveStakeMsg)
// 	require.NoError(err, "ConfirmRemoveStake should not return an error")

// 	// Check updated stake for delegator after removal
// 	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)
// 	// Initial Stake: 100
// 	// Stake added: 1000
// 	// Stake removed: 500
// 	// Total: 600
// 	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake), delegatorStake, "Delegator stake amount mismatch after removal")

// 	// Check updated stake for target after removal
// 	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
// 	require.NoError(err)
// 	// Initial Stake: 100
// 	// Stake added: 1000
// 	// Stake removed: 500
// 	// Total: 600
// 	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake), targetStake, "Target stake amount mismatch after removal")

// 	// Check updated total stake after removal
// 	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)
// 	// Initial Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake added: 1000
// 	// Stake removed: 500
// 	// Total: 700
// 	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStake, "Total stake amount mismatch after removal")

// 	// Check updated total stake for topic after removal
// 	totalStakeForTopic, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)
// 	// Initial Stake: 200 (100 for reputer, 100 for worker)
// 	// Stake added: 1000
// 	// Stake removed: 500
// 	// Total: 700
// 	require.Equal(stakeAmount.Sub(removalAmount).Add(registrationInitialStake.Mul(cosmosMath.NewUint(2))), totalStakeForTopic, "Total stake amount for topic mismatch after removal")

// 	// Check bond after removal
// 	bond, err := s.emissionsKeeper.GetBond(ctx, reputerAddr, workerAddr)
// 	require.NoError(err)
// 	// Stake placed upon target: 1000
// 	// Stake removed: 500
// 	// Total: 500
// 	require.Equal(stakeAmount.Sub(removalAmount), bond, "Bond amount mismatch after removal")
// }

// func (s *KeeperTestSuite) TestMsgStartRemoveAllStake() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
// 	registrationInitialStake := cosmosMath.NewUint(100)

// 	// Common setup for staking
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Add stake first to ensure there is an initial stake
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err := msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	// Remove Registration
// 	removeRegistrationMsg := &state.MsgRemoveRegistration{
// 		Creator:   reputerAddr.String(),
// 		TopicId:   0,
// 		IsReputer: true,
// 	}
// 	_, err = msgServer.RemoveRegistration(ctx, removeRegistrationMsg)
// 	require.NoError(err, "RemoveRegistration should not return an error")

// 	// Remove all stake
// 	removeAllStakeMsg := &state.MsgStartRemoveAllStake{
// 		Sender: reputerAddr.String(),
// 	}

// 	timeBefore := uint64(time.Now().UTC().Unix())
// 	_, err = msgServer.StartRemoveAllStake(ctx, removeAllStakeMsg)

// 	// check the state has changed appropriately after the removal
// 	require.NoError(err, "StartRemoveAllStake should not return an error")
// 	removalInfo, err := s.emissionsKeeper.GetStakeRemovalQueueByAddress(ctx, reputerAddr)
// 	require.NoError(err, "Stake removal queue should not be empty")
// 	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted, timeBefore, "Time should be valid starting")
// 	timeNow := uint64(time.Now().UTC().Unix())
// 	delayWindow, err := s.emissionsKeeper.GetParamsRemoveStakeDelayWindow(ctx)
// 	s.Require().NoError(err)
// 	require.GreaterOrEqual(removalInfo.TimestampRemovalStarted+delayWindow, timeNow, "Time should be valid ending")
// 	require.Equal(2, len(removalInfo.Placements), "There should be two placements in the removal queue")
// 	require.Equal(removalInfo.Placements[0].Target, workerAddr.String(), "The target in the removal queue should be the same as the target in the message")
// 	require.Equal(removalInfo.Placements[0].Amount, stakeAmount, "The amount in the removal queue should be the same as the amount in the message")
// 	require.Equal(removalInfo.Placements[1].Target, reputerAddr.String(), "The target in the removal queue should be the same as the target in the message")
// 	require.Equal(removalInfo.Placements[1].Amount, registrationInitialStake, "The amount in the removal queue should be the same as the amount in the message")
// }

// func (s *KeeperTestSuite) TestMsgConfirmRemoveAllStake() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	registrationInitialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(registrationInitialStake.BigInt())))

// 	// Common setup for staking
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Add stake first to ensure there is an initial stake
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputerAddr.String(),
// 		StakeTarget: workerAddr.String(),
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err := msgServer.AddStake(ctx, addStakeMsg)
// 	require.NoError(err, "AddStake should not return an error")

// 	// Remove all stake
// 	removeAllStakeMsg := &state.MsgStartRemoveAllStake{
// 		Sender: reputerAddr.String(),
// 	}

// 	// Remove registration
// 	removeRegistrationMsg := &state.MsgRemoveRegistration{
// 		Creator:   reputerAddr.String(),
// 		TopicId:   0,
// 		IsReputer: true,
// 	}
// 	_, err = msgServer.RemoveRegistration(ctx, removeRegistrationMsg)
// 	require.NoError(err, "RemoveRegistration should not return an error")

// 	_, err = msgServer.StartRemoveAllStake(ctx, removeAllStakeMsg)

// 	// check the state has changed appropriately after the removal
// 	require.NoError(err, "StartRemoveAllStake should not return an error")

// 	// swap out the timestamp so it's valid for the confirmRemove
// 	stakeRemoveInfo, err := s.emissionsKeeper.GetStakeRemovalQueueByAddress(ctx, reputerAddr)
// 	require.NoError(err, "Stake removal queue should not be empty")
// 	stakeRemoveInfo.TimestampRemovalStarted = uint64(time.Now().UTC().Unix()) - 1000
// 	err = s.emissionsKeeper.SetStakeRemovalQueueForAddress(ctx, reputerAddr, stakeRemoveInfo)
// 	require.NoError(err, "Set stake removal queue should work")

// 	confirmRemoveMsg := &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingAccountName, reputerAddr, registrationInitialStakeCoins)
// 	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingAccountName, reputerAddr, stakeAmountCoins)
// 	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveMsg)
// 	require.NoError(err, "RemoveAllStake should not return an error")

// 	// Check that the sender's total stake is zero after removal
// 	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(ctx, reputerAddr)
// 	require.NoError(err)
// 	require.Equal(cosmosMath.ZeroUint(), delegatorStake, "delegator has zero stake after withdrawal")

// 	// Check that the target's stake is reduced by the stake amount
// 	targetStake, err := s.emissionsKeeper.GetStakePlacedUponTarget(ctx, workerAddr)
// 	require.NoError(err)
// 	require.Equal(registrationInitialStake, targetStake, "Target's stake should be equal to the registration stake after removing all stake")

// 	// Check updated total stake after removal
// 	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err)
// 	require.Equal(registrationInitialStake, totalStake, "Total stake should be equal to the registration stakes after removing all stake")

// 	// Check updated total stake for topic after removal
// 	totalStakeForTopic, err := s.emissionsKeeper.GetTopicStake(ctx, uint64(0))
// 	require.NoError(err)
// 	require.Equal(registrationInitialStake, totalStakeForTopic, "Total stake for the topic should be equal to the registration stakes after removing all stake")
// }

// func (s *KeeperTestSuite) TestStartRemoveStakeInvalidRemoveMoreThanExists() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	totalStake, err := s.emissionsKeeper.GetTotalStake(ctx)
// 	require.NoError(err, "Total stake should not be empty")

// 	// Scenario 1: Attempt to remove more stake than exists in the system
// 	removeStakeMsg := &state.MsgStartRemoveStake{
// 		Sender: reputerAddr.String(),
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: workerAddr.String(),
// 				Amount: stakeAmount.Add(totalStake),
// 			},
// 		},
// 	}
// 	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
// 	require.ErrorIs(err, state.ErrInsufficientStakeToRemove, "RemoveStake should return an error when attempting to remove more stake than exists")
// }

// func (s *KeeperTestSuite) TestStartRemoveStakeInvalidRemoveMoreThanMinimalAmountWhileBeingRegisteredInTopics() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Attempt to remove more stake than exists in the system
// 	removeStakeMsg := &state.MsgStartRemoveStake{
// 		Sender: reputerAddr.String(),
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputerAddr.String(),
// 				Amount: registrationInitialStake,
// 			},
// 		},
// 	}
// 	_, err := msgServer.StartRemoveStake(ctx, removeStakeMsg)
// 	require.ErrorIs(err, state.ErrInsufficientStakeAfterRemoval, "RemoveStake should return an error when attempting to remove more stake than exists")

// 	// Remove Registration
// 	removeRegistrationMsg := &state.MsgRemoveRegistration{
// 		Creator:   reputerAddr.String(),
// 		TopicId:   0,
// 		IsReputer: true,
// 	}
// 	_, err = msgServer.RemoveRegistration(ctx, removeRegistrationMsg)
// 	require.NoError(err, "RemoveRegistration should not return an error")

// 	// Successfully remove stake
// 	_, err = msgServer.StartRemoveStake(ctx, removeStakeMsg)
// 	require.NoError(err, "RemoveStake should not return an error when removing all stake")
// }

// func (s *KeeperTestSuite) TestStartRemoveStakeInvalidNotEnoughStake() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	stakeAmount := cosmosMath.NewUint(1000)

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// Scenario 4: Attempt to remove stake when sender does not have enough stake placed on the target
// 	removeStakeMsg := &state.MsgStartRemoveStake{
// 		Sender: reputerAddr.String(),
// 		PlacementsRemove: []*state.StakePlacement{
// 			{

// 				Target: workerAddr.String(),
// 				Amount: stakeAmount,
// 			},
// 		},
// 	}
// 	_, err := msgServer.StartRemoveStake(ctx, removeStakeMsg)
// 	require.ErrorIs(err, state.ErrInsufficientStakeToRemove, "RemoveStake should return an error when sender does not have enough stake placed on the target")
// }

// func (s *KeeperTestSuite) TestConfirmRemoveStakeInvalidNoRemovalStarted() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	// workerAddr := sdk.AccAddress(PKS[1].Address())  // target

// 	// // Common setup for staking
// 	// registrationInitialStake := cosmosMath.NewUint(100)
// 	// s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	_, err := msgServer.ConfirmRemoveStake(ctx, &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	})
// 	require.ErrorIs(err, state.ErrConfirmRemoveStakeNoRemovalStarted, "ConfirmRemoveStake should return an error when no stake removal has been started")
// }

// func (s *KeeperTestSuite) TestConfirmRemoveStakeInvalidTooEarly() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// if you try to call the geniune msgServer.StartStakeRemoval
// 	// the unix time set there is going to be in the future,
// 	// rather than having to monkey patch the unix time, or some complicated mocking setup,
// 	// lets just directly manipulate the removalInfo in the keeper store
// 	timeNow := uint64(time.Now().UTC().Unix())
// 	err := s.emissionsKeeper.SetStakeRemovalQueueForAddress(ctx, reputerAddr, state.StakeRemoval{
// 		TimestampRemovalStarted: timeNow + 1000000,
// 		Placements: []*state.StakeRemovalPlacement{
// 			{
// 				TopicIds: []uint64{0},
// 				Target:   reputerAddr.String(),
// 				Amount:   registrationInitialStake,
// 			},
// 		},
// 	})
// 	require.NoError(err, "Set stake removal queue should work")
// 	confirmRemoveStakeMsg := &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	}
// 	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveStakeMsg)
// 	require.ErrorIs(err, state.ErrConfirmRemoveStakeTooEarly, "ConfirmRemoveStake should return an error when stake removal is too early")
// }

// func (s *KeeperTestSuite) TestConfirmRemoveStakeInvalidTooLate() {
// 	ctx, msgServer := s.ctx, s.msgServer
// 	require := s.Require()

// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// if you try to call the geniune msgServer.StartStakeRemoval
// 	// the unix time set there is going to be in the future,
// 	// rather than having to monkey patch the unix time, or some complicated mocking setup,
// 	// lets just directly manipulate the removalInfo in the keeper store
// 	err := s.emissionsKeeper.SetStakeRemovalQueueForAddress(ctx, reputerAddr, state.StakeRemoval{
// 		TimestampRemovalStarted: 0,
// 		Placements: []*state.StakeRemovalPlacement{
// 			{
// 				TopicIds: []uint64{0},
// 				Target:   reputerAddr.String(),
// 				Amount:   registrationInitialStake,
// 			},
// 		},
// 	})
// 	require.NoError(err, "Set stake removal queue should work")
// 	confirmRemoveStakeMsg := &state.MsgConfirmRemoveStake{
// 		Sender: reputerAddr.String(),
// 	}
// 	_, err = msgServer.ConfirmRemoveStake(ctx, confirmRemoveStakeMsg)
// 	require.ErrorIs(err, state.ErrConfirmRemoveStakeTooLate, "ConfirmRemoveStake should return an error when stake removal is too early")
// }

// func (s *KeeperTestSuite) TestModifyStakeSimple() {
// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	reputer := reputerAddr.String()
// 	worker := workerAddr.String()

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// modify stake for reputer to put half of their stake in the worker
// 	response, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
// 		Sender: reputer,
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 		PlacementsAdd: []*state.StakePlacement{
// 			{
// 				Target: worker,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 	})
// 	s.Require().NoError(err)
// 	s.Require().Equal(&state.MsgModifyStakeResponse{}, response, "ModifyStake should return an empty response on success")

// 	// Check updated stake for delegator
// 	delegatorStake, err := s.emissionsKeeper.GetDelegatorStake(s.ctx, reputerAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.NewUint(100), delegatorStake, "Delegator stake should not change on modify stake")

// 	bond, err := s.emissionsKeeper.GetBond(s.ctx, reputerAddr, reputerAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.NewUint(50), bond, "Reputer bond amount mismatch")

// 	targetStake1, err := s.emissionsKeeper.GetStakePlacedUponTarget(s.ctx, reputerAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.NewUint(50), targetStake1, "Reputer target stake amount mismatch")

// 	targetStake2, err := s.emissionsKeeper.GetStakePlacedUponTarget(s.ctx, workerAddr)
// 	s.Require().NoError(err)
// 	s.Require().Equal(cosmosMath.NewUint(150), targetStake2, "Worker target stake amount mismatch")

// }

// func (s *KeeperTestSuite) TestModifyStakeInvalidSumChangesNotEqualRemoveMoreThanAdd() {
// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	reputer := reputerAddr.String()
// 	worker := workerAddr.String()

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// modify stake for reputer to put half of their stake in the worker
// 	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
// 		Sender: reputer,
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(60),
// 			},
// 		},
// 		PlacementsAdd: []*state.StakePlacement{
// 			{
// 				Target: worker,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 	})
// 	s.Require().ErrorIs(err, state.ErrModifyStakeSumBeforeNotEqualToSumAfter)
// }

// func (s *KeeperTestSuite) TestModifyStakeInvalidSumChangesNotEqualAddMoreThanRemove() {
// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	reputer := reputerAddr.String()
// 	worker := workerAddr.String()

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// modify stake for reputer to put half of their stake in the worker
// 	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
// 		Sender: reputer,
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 		PlacementsAdd: []*state.StakePlacement{
// 			{
// 				Target: worker,
// 				Amount: cosmosMath.NewUint(60),
// 			},
// 		},
// 	})
// 	s.Require().ErrorIs(err, state.ErrModifyStakeSumBeforeNotEqualToSumAfter)
// }

// func (s *KeeperTestSuite) TestModifyStakeInvalidNotHaveEnoughDelegatorStake() {
// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	reputer := reputerAddr.String()
// 	worker := workerAddr.String()

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// modify stake for reputer to put half of their stake in the worker
// 	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
// 		Sender: reputer,
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 		PlacementsAdd: []*state.StakePlacement{
// 			{
// 				Target: worker,
// 				Amount: cosmosMath.NewUint(200),
// 			},
// 		},
// 	})
// 	s.Require().ErrorIs(err, state.ErrModifyStakeBeforeSumGreaterThanSenderStake)
// }

// func (s *KeeperTestSuite) TestModifyStakeInvalidNotHaveEnoughBond() {
// 	// do the normal setup, add more stake in a third party
// 	// then modify the stake but with more bond than the first party has on them
// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	reputer := reputerAddr.String()
// 	workerAddr1 := sdk.AccAddress(PKS[1].Address()) // target
// 	workerAddr2 := sdk.AccAddress(PKS[2].Address()) // target
// 	worker2 := workerAddr2.String()

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr1, registrationInitialStake)

// 	registrationInitialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.Int(registrationInitialStake)))
// 	s.bankKeeper.EXPECT().MintCoins(gomock.Any(), state.AlloraStakingAccountName, registrationInitialStakeCoins)
// 	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingAccountName, registrationInitialStakeCoins)
// 	s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), state.AlloraStakingAccountName, reputerAddr, registrationInitialStakeCoins)
// 	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingAccountName, reputerAddr, registrationInitialStakeCoins)
// 	// Register Reputer
// 	worker2RegMsg := &state.MsgRegister{
// 		Creator:      worker2,
// 		LibP2PKey:    "test2",
// 		MultiAddress: "test2",
// 		TopicIds:     []uint64{0},
// 		InitialStake: registrationInitialStake,
// 		Owner:        worker2,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), workerAddr2, state.AlloraStakingAccountName, registrationInitialStakeCoins)
// 	_, err := s.msgServer.Register(s.ctx, worker2RegMsg)
// 	s.Require().NoError(err, "Registering worker2 should not return an error")

// 	stakeAmount := cosmosMath.NewUint(1000)
// 	stakeAmountCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewIntFromBigInt(stakeAmount.BigInt())))
// 	// Add stake from reputer to worker2
// 	addStakeMsg := &state.MsgAddStake{
// 		Sender:      reputer,
// 		StakeTarget: worker2,
// 		Amount:      stakeAmount,
// 	}
// 	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), reputerAddr, state.AlloraStakingAccountName, stakeAmountCoins)
// 	_, err = s.msgServer.AddStake(s.ctx, addStakeMsg)
// 	s.Require().NoError(err, "AddStake should not return an error")
// 	bond, err := s.emissionsKeeper.GetBond(s.ctx, reputerAddr, workerAddr2)
// 	s.Require().NoError(err)
// 	s.Require().Equal(stakeAmount.BigInt(), bond.BigInt(), "Bond should have been added")

// 	// modify stake
// 	modifyStakeMsg := &state.MsgModifyStake{
// 		Sender: reputer,
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(300),
// 			},
// 			{
// 				Target: worker2,
// 				Amount: cosmosMath.NewUint(100),
// 			},
// 		},
// 		PlacementsAdd: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(200),
// 			},
// 			{
// 				Target: worker2,
// 				Amount: cosmosMath.NewUint(200),
// 			},
// 		},
// 	}
// 	_, err = s.msgServer.ModifyStake(s.ctx, modifyStakeMsg)
// 	s.Require().ErrorIs(err, state.ErrModifyStakeBeforeBondLessThanAmountModified, "ModifyStake Error not matching expected")
// }

// func (s *KeeperTestSuite) TestModifyStakeInvalidTarget() {
// 	// Mock setup for addresses
// 	reputerAddr := sdk.AccAddress(PKS[0].Address()) // delegator
// 	workerAddr := sdk.AccAddress(PKS[1].Address())  // target
// 	reputer := reputerAddr.String()
// 	randoAddr := sdk.AccAddress(PKS[3].Address()) // delegator
// 	rando := randoAddr.String()

// 	// Common setup for staking
// 	registrationInitialStake := cosmosMath.NewUint(100)
// 	s.commonStakingSetup(s.ctx, reputerAddr, workerAddr, registrationInitialStake)

// 	// modify stake for reputer to put half of their stake in the worker
// 	_, err := s.msgServer.ModifyStake(s.ctx, &state.MsgModifyStake{
// 		Sender: reputer,
// 		PlacementsRemove: []*state.StakePlacement{
// 			{
// 				Target: reputer,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 		PlacementsAdd: []*state.StakePlacement{
// 			{
// 				Target: rando,
// 				Amount: cosmosMath.NewUint(50),
// 			},
// 		},
// 	})
// 	s.Require().ErrorIs(err, state.ErrAddressNotRegistered)
// }
