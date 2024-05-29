package msgserver_test

import (
	"errors"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) commonStakingSetup(
	ctx sdk.Context,
	reputer string,
	worker string,
	reputerInitialBalanceUint cosmosMath.Int,
) uint64 {
	workerAddr := sdk.MustAccAddressFromBech32(worker)
	reputerAddr := sdk.MustAccAddressFromBech32(reputer)
	msgServer := s.msgServer
	require := s.Require()

	// Create Topic
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:          reputerAddr.String(),
		Metadata:         "Some metadata for the new topic",
		LossLogic:        "logic",
		LossMethod:       "method",
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

	reputerInitialBalance := types.DefaultParams().CreateTopicFee.Add(cosmosMath.Int(reputerInitialBalanceUint))

	reputerInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, reputerInitialBalance))

	s.bankKeeper.MintCoins(ctx, types.AlloraStakingAccountName, reputerInitialBalanceCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, reputerAddr, reputerInitialBalanceCoins)

	response, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on creation")
	topicId := response.TopicId

	// Register Reputer
	reputerRegMsg := &types.MsgRegister{
		Sender:       reputerAddr.String(),
		Owner:        reputerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
		IsReputer:    true,
	}
	_, err = msgServer.Register(ctx, reputerRegMsg)
	require.NoError(err, "Registering reputer should not return an error")

	workerInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(1000)))

	s.bankKeeper.MintCoins(ctx, types.AlloraStakingAccountName, workerInitialBalanceCoins)
	s.bankKeeper.MintCoins(ctx, types.AlloraRewardsAccountName, workerInitialBalanceCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, workerAddr, workerInitialBalanceCoins)

	// Register Worker
	workerRegMsg := &types.MsgRegister{
		Sender:       workerAddr.String(),
		Owner:        workerAddr.String(),
		LibP2PKey:    "test",
		MultiAddress: "test",
		TopicId:      topicId,
	}
	_, err = msgServer.Register(ctx, workerRegMsg)
	require.NoError(err, "Registering worker should not return an error")

	return topicId
}

func (s *KeeperTestSuite) TestMsgAddStake() {
	ctx := s.ctx
	require := s.Require()

	reputerAddr := sdk.AccAddress(PKS[0].Address()).String() // delegator
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()  // target
	stakeAmount := cosmosMath.NewInt(10)
	registrationInitialBalance := cosmosMath.NewInt(100)

	topicId := s.commonStakingSetup(ctx, reputerAddr, workerAddr, registrationInitialBalance)

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	reputerStake, err := s.emissionsKeeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr)
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), reputerStake, "Stake amount mismatch")

	topicStake, err := s.emissionsKeeper.GetTopicStake(ctx, topicId)
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), topicStake, "Stake amount mismatch")

	response, err := s.msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	reputerStake, err = s.emissionsKeeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr)
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	topicStake, err = s.emissionsKeeper.GetTopicStake(ctx, topicId)
	require.NoError(err)
	require.Equal(stakeAmount, topicStake, "Stake amount mismatch")
}

func (s *KeeperTestSuite) TestStartRemoveStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	senderAddr := sdk.AccAddress(PKS[0].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)

	s.MintTokensToAddress(senderAddr, cosmosMath.NewInt(1000))

	// Assuming you have methods to directly manipulate the state
	// Simulate that sender has already staked the required amount
	s.emissionsKeeper.AddStake(ctx, topicId, senderAddr.String(), stakeAmount)

	msg := &types.MsgStartRemoveStake{
		Sender:  senderAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	response, err := s.msgServer.StartRemoveStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response)

	retrievedInfo, err := keeper.GetStakeRemovalByTopicAndAddress(ctx, topicId, senderAddr.String())
	require.NoError(err)
	require.NotNil(retrievedInfo)
	s.Require().Equal(msg.TopicId, retrievedInfo.Placement.TopicId, "Topic IDs should match for all placements")
	s.Require().Equal(msg.Sender, retrievedInfo.Placement.Reputer, "Reputer addresses should match for all placements")
	s.Require().Equal(msg.Amount, retrievedInfo.Placement.Amount, "Amounts should match for all placements")
}

func (s *KeeperTestSuite) TestStartRemoveStakeInsufficientStake() {
	ctx := s.ctx
	require := s.Require()

	senderAddr := sdk.AccAddress(PKS[0].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)

	msg := &types.MsgStartRemoveStake{
		Sender:  senderAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	_, err := s.msgServer.StartRemoveStake(ctx, msg)
	require.ErrorIs(err, types.ErrInsufficientStakeToRemove)
}

func (s *KeeperTestSuite) TestConfirmRemoveStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	senderAddr := sdk.AccAddress(PKS[0].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := sdkCtx.BlockHeight()

	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow

	s.MintTokensToAddress(senderAddr, cosmosMath.NewInt(1000))
	s.MintTokensToModule(types.AlloraStakingAccountName, cosmosMath.NewInt(1000))
	s.emissionsKeeper.AddStake(ctx, topicId, senderAddr.String(), stakeAmount)

	// Simulate the stake removal request.
	placement := &types.StakePlacement{
		TopicId: topicId,
		Reputer: senderAddr.String(),
		Amount:  stakeAmount,
	}
	stakeRemoval := types.StakeRemoval{
		BlockRemovalStarted: startBlock,
		Placement:           placement,
	}

	// Manually setting the removal in state (this part would normally involve interacting with the keeper to set up state).
	keeper.SetStakeRemoval(ctx, senderAddr.String(), stakeRemoval) // This assumes such a method exists.

	msg := &types.MsgConfirmRemoveStake{
		Sender:  senderAddr.String(),
		TopicId: topicId,
	}

	ctx = ctx.WithBlockHeight(startBlock + removalDelay + 1)

	// Perform the stake confirmation
	response, err := s.msgServer.ConfirmRemoveStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after confirming stake removal")

	// Verifications to ensure the stake was properly removed could be included here if there are methods to query the state
	// Example: check that the stake amount at the topic is zero or reduced appropriately
	finalStake, err := keeper.GetStakeOnReputerInTopic(ctx, topicId, senderAddr.String())
	require.NoError(err)
	require.True(finalStake.IsZero(), "Stake amount should be zero after removal is confirmed")
}

func (s *KeeperTestSuite) TestCantConfirmRemoveStakeWithoutStartingRemoval() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	senderAddr := sdk.AccAddress(PKS[0].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := sdkCtx.BlockHeight()

	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow

	s.emissionsKeeper.AddStake(ctx, topicId, senderAddr.String(), stakeAmount)

	msg := &types.MsgConfirmRemoveStake{
		Sender:  senderAddr.String(),
		TopicId: topicId,
	}

	ctx = ctx.WithBlockHeight(startBlock + removalDelay + 1)

	// Perform the stake confirmation
	_, err = s.msgServer.ConfirmRemoveStake(ctx, msg)
	require.ErrorIs(err, types.ErrConfirmRemoveStakeNoRemovalStarted)

	// Verifications to ensure the stake was properly removed could be included here if there are methods to query the state
	// Example: check that the stake amount at the topic is zero or reduced appropriately
	finalStake, err := keeper.GetStakeOnReputerInTopic(ctx, topicId, senderAddr.String())
	require.NoError(err)
	require.False(finalStake.IsZero())
}

func (s *KeeperTestSuite) TestConfirmRemoveStakeTooEarly() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	senderAddr := sdk.AccAddress(PKS[0].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := sdkCtx.BlockHeight()

	// Fetch the delay window for removing stake
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow

	// Simulate that sender has already staked the required amount
	s.emissionsKeeper.AddStake(ctx, topicId, senderAddr.String(), stakeAmount)

	// Simulate the stake removal request
	placement := &types.StakePlacement{
		TopicId: topicId,
		Reputer: senderAddr.String(),
		Amount:  stakeAmount,
	}
	stakeRemoval := types.StakeRemoval{
		BlockRemovalStarted: startBlock,
		Placement:           placement,
	}

	// Manually setting the removal in state (this part would normally involve interacting with the keeper to set up state).
	keeper.SetStakeRemoval(ctx, senderAddr.String(), stakeRemoval) // This assumes such a method exists.

	msg := &types.MsgConfirmRemoveStake{
		Sender:  senderAddr.String(),
		TopicId: topicId,
	}

	// Set the current block height to simulate an attempt to confirm removal too early, before the delay has fully passed.
	ctx = ctx.WithBlockHeight(startBlock + removalDelay - 1) // Attempting to confirm just before the delay period ends

	// Perform the stake confirmation
	response, err := s.msgServer.ConfirmRemoveStake(ctx, msg)
	require.Error(err, "Should error because the stake removal is being confirmed too early")
	require.Nil(response, "Response should be nil when confirming too early")
	require.ErrorIs(types.ErrConfirmRemoveStakeTooEarly, err, "Error should be ErrConfirmRemoveStakeTooEarly")

	// Verify the stake has not been removed
	finalStake, err := keeper.GetStakeOnReputerInTopic(ctx, topicId, senderAddr.String())
	require.NoError(err)
	require.False(finalStake.IsZero(), "Stake amount should not be zero since removal is not confirmed")
}

func (s *KeeperTestSuite) TestDelegateStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)

	msg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	reputerStake, err := s.emissionsKeeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), reputerStake, "Stake amount mismatch")

	amount0, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegatorAddr.String(), reputerAddr.String())
	require.NoError(err)
	require.Equal(alloraMath.NewDecFromInt64(0), amount0.Amount)

	// Perform the stake delegation
	response, err := s.msgServer.DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	reputerStake, err = s.emissionsKeeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	amount1, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegatorAddr.String(), reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, amount1.Amount.SdkIntTrim())
}

func (s *KeeperTestSuite) TestReputerCantSelfDelegateStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[1].Address())
	reputerAddr := delegatorAddr
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)

	msg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	_, err := s.msgServer.DelegateStake(ctx, msg)
	require.Error(err, types.ErrCantSelfDelegate)
}

func (s *KeeperTestSuite) TestDelegateeCantWithdrawDelegatedStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)

	delegateStakeMsg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	response, err := s.msgServer.DelegateStake(ctx, delegateStakeMsg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	reputerStake, err := s.emissionsKeeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	amount1, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegatorAddr.String(), reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, amount1.Amount.SdkIntTrim())

	// Attempt to withdraw the delegated stake
	removeMsg := &types.MsgStartRemoveStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	_, err = s.msgServer.StartRemoveStake(ctx, removeMsg)
	require.Error(err)
}

func (s *KeeperTestSuite) TestDelegateStakeUnregisteredReputer() {
	ctx := s.ctx
	require := s.Require()

	senderAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)

	// Do not register the reputer to simulate failure case

	msg := &types.MsgDelegateStake{
		Sender:  senderAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	// Attempt to perform the stake delegation
	response, err := s.msgServer.DelegateStake(ctx, msg)
	require.Error(err)
	require.Nil(response, "Response should be nil when delegation fails due to unregistered reputer")
	require.True(errors.Is(err, types.ErrAddressIsNotRegisteredInThisTopic), "Error should indicate that the reputer is not registered in the topic")
}

func (s *KeeperTestSuite) TestStartRemoveDelegateStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)

	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	msg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	response, err := s.msgServer.DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	msg2 := &types.MsgStartRemoveDelegateStake{
		Sender:  delegatorAddr.String(),
		Reputer: reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	_, err = keeper.GetDelegateStakeRemovalByTopicAndAddress(ctx, topicId, reputerAddr.String(), delegatorAddr.String())
	require.Error(err)

	// Perform the stake removal initiation
	response2, err := s.msgServer.StartRemoveDelegateStake(ctx, msg2)
	require.NoError(err)
	require.NotNil(response2, "Response should not be nil after successful stake removal initiation")

	// Verification: Check if the removal has been queued
	removalInfo, err := keeper.GetDelegateStakeRemovalByTopicAndAddress(ctx, topicId, reputerAddr.String(), delegatorAddr.String())
	require.NoError(err)
	require.NotNil(removalInfo, "Stake removal should be recorded in the state")
}

func (s *KeeperTestSuite) TestStartRemoveDelegateStakeError() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)

	reputerInfo := types.OffchainNode{
		LibP2PKey:    "reputer-libp2p-key-sample",
		MultiAddress: "reputer-multi-address-sample",
		Owner:        "reputer-owner-sample",
		NodeAddress:  "reputer-node-address-sample",
		NodeId:       "reputer-node-id-sample",
	}

	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	msg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		Reputer: reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	response, err := s.msgServer.DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	msg2 := &types.MsgStartRemoveDelegateStake{
		Sender:  delegatorAddr.String(),
		Reputer: reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount.Mul(cosmosMath.NewInt(2)),
	}

	// Perform the stake removal initiation
	_, err = s.msgServer.StartRemoveDelegateStake(ctx, msg2)
	require.Error(err, types.ErrInsufficientStakeToRemove)
}

func (s *KeeperTestSuite) TestConfirmRemoveDelegateStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	topicId := uint64(123)
	stakeAmount := cosmosMath.NewInt(50)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := sdkCtx.BlockHeight()

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	// Simulate adding a reputer and delegating stake to them
	keeper.InsertReputer(ctx, topicId, reputerAddr.String(), types.OffchainNode{})
	_, err := s.msgServer.DelegateStake(ctx, &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	})
	require.NoError(err)

	// Start removing the delegated stake
	_, err = s.msgServer.StartRemoveDelegateStake(ctx, &types.MsgStartRemoveDelegateStake{
		Sender:  delegatorAddr.String(),
		Reputer: reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	// Try to confirm removal too early
	_, err = s.msgServer.ConfirmRemoveDelegateStake(ctx, &types.MsgConfirmDelegateRemoveStake{
		Sender:  delegatorAddr.String(),
		Reputer: reputerAddr.String(),
		TopicId: topicId,
	})
	require.Error(err, types.ErrConfirmRemoveStakeTooEarly)

	// Simulate passing of time to surpass the withdrawal delay
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow

	ctx = ctx.WithBlockHeight(startBlock + removalDelay + 1)

	// Try to confirm removal after delay window
	response, err := s.msgServer.ConfirmRemoveDelegateStake(ctx, &types.MsgConfirmDelegateRemoveStake{
		Sender:  delegatorAddr.String(),
		Reputer: reputerAddr.String(),
		TopicId: topicId,
	})
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful confirmation of stake removal")

	// Check that the stake was actually removed
	delegateStakePlaced, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegatorAddr.String(), reputerAddr.String())
	require.NoError(err)
	require.True(delegateStakePlaced.Amount.IsZero(), "Delegate stake should be zero after successful removal")
}

func (s *KeeperTestSuite) TestRewardDelegateStake() {
	ctx := s.ctx
	require := s.Require()
	keeper := s.emissionsKeeper
	block := int64(1003)
	newBlock := int64(1004)
	score := alloraMath.MustNewDecFromString("17.53436")

	delegatorAddr := sdk.AccAddress(PKS[0].Address())
	reputerAddr := sdk.AccAddress(PKS[1].Address())
	workerAddr := sdk.AccAddress(PKS[2].Address()) // target
	delegator2Addr := sdk.AccAddress(PKS[3].Address())
	stakeAmount := cosmosMath.NewInt(500000)
	registrationInitialBalance := cosmosMath.NewInt(1000)
	delegatorStakeAmount := cosmosMath.NewInt(500)
	delegatorStakeAmount2 := cosmosMath.NewInt(500)
	delegator2StakeAmount := cosmosMath.NewInt(5000)

	topicId := s.commonStakingSetup(ctx, reputerAddr.String(), workerAddr.String(), registrationInitialBalance)
	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegator2Addr, cosmosMath.NewInt(1000000))

	addStakeMsg := &types.MsgAddStake{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	response, err := s.msgServer.AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	msg := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegatorStakeAmount,
	}

	msg2 := &types.MsgDelegateStake{
		Sender:  delegator2Addr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegator2StakeAmount,
	}

	reputerStake, err := s.emissionsKeeper.GetStakeOnReputerInTopic(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	amount0, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegatorAddr.String(), reputerAddr.String())
	require.NoError(err)
	require.Equal(alloraMath.NewDecFromInt64(0), amount0.Amount)

	// Perform the stake delegation
	responseDelegator, err := s.msgServer.DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(responseDelegator, "Response should not be nil after successful delegation")

	responseDelegator2, err := s.msgServer.DelegateStake(ctx, msg2)
	require.NoError(err)
	require.NotNil(responseDelegator2, "Response should not be nil after successful delegation")

	var reputerValueBundles types.ReputerValueBundles
	scoreToAdd := types.Score{
		TopicId:     topicId,
		BlockNumber: block,
		Address:     reputerAddr.String(),
		Score:       score,
	}
	err = s.emissionsKeeper.InsertReputerScore(s.ctx, topicId, block, scoreToAdd)
	s.Require().NoError(err)

	reputerValueBundle := &types.ReputerValueBundle{
		ValueBundle: &types.ValueBundle{
			TopicId:       topicId,
			Reputer:       reputerAddr.String(),
			CombinedValue: alloraMath.MustNewDecFromString("1500.0"),
			NaiveValue:    alloraMath.MustNewDecFromString("1500.0"),
		},
	}
	reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, reputerValueBundle)
	_ = s.emissionsKeeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, block, reputerValueBundles)

	// Calculate and Set the reputer scores
	scores, err := rewards.GenerateReputerScores(s.ctx, s.emissionsKeeper, topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Generate rewards
	reputers, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(1, len(reputerRewards))

	msg3 := &types.MsgDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegatorStakeAmount2,
	}

	responseDelegator3, err := s.msgServer.DelegateStake(ctx, msg3)
	require.NoError(err)
	require.NotNil(responseDelegator3, "Response should not be nil after successful delegation")

	var newReputerValueBundles types.ReputerValueBundles
	newScoreToAdd := types.Score{
		TopicId:     topicId,
		BlockNumber: newBlock,
		Address:     reputerAddr.String(),
		Score:       score,
	}
	err = s.emissionsKeeper.InsertReputerScore(s.ctx, topicId, newBlock, newScoreToAdd)
	s.Require().NoError(err)

	newReputerValueBundle := &types.ReputerValueBundle{
		ValueBundle: &types.ValueBundle{
			TopicId:       topicId,
			Reputer:       reputerAddr.String(),
			CombinedValue: alloraMath.MustNewDecFromString("1500.0"),
			NaiveValue:    alloraMath.MustNewDecFromString("1500.0"),
		},
	}
	newReputerValueBundles.ReputerValueBundles = append(newReputerValueBundles.ReputerValueBundles, newReputerValueBundle)
	_ = s.emissionsKeeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, newBlock, newReputerValueBundles)

	// Calculate and Set the reputer scores
	scores, err = rewards.GenerateReputerScores(s.ctx, s.emissionsKeeper, topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Generate new rewards
	reputers, reputersRewardFractions, err = rewards.GetReputersRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)
	newReputerRewards, err := rewards.GetRewardPerReputer(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		alloraMath.MustNewDecFromString("1020.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(1, len(newReputerRewards))

	beforeBalance := s.bankKeeper.GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	rewardMsg := &types.MsgRewardDelegateStake{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.msgServer.RewardDelegateStake(ctx, rewardMsg)
	afterBalance := s.bankKeeper.GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	s.Require().NoError(err)
	s.Require().Greater(afterBalance.Amount.Uint64(), beforeBalance.Amount.Uint64(), "Balance must be increased")

	beforeBalance2 := s.bankKeeper.GetBalance(ctx, delegator2Addr, params.DefaultBondDenom)
	rewardMsg2 := &types.MsgRewardDelegateStake{
		Sender:  delegator2Addr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.msgServer.RewardDelegateStake(ctx, rewardMsg2)
	afterBalance2 := s.bankKeeper.GetBalance(ctx, delegator2Addr, params.DefaultBondDenom)
	s.Require().NoError(err)
	s.Require().Greater(afterBalance2.Amount.Uint64(), beforeBalance2.Amount.Uint64(), "Balance must be increased")
}
