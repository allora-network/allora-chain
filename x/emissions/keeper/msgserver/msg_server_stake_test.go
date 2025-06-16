package msgserver_test

import (
	"errors"
	"fmt"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/utils/ptr"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestMsgAddStakeWithWhitelistCheck() {
	ctx := s.Ctx()
	require := s.Require()

	reputerIndexes := testutil.ReturnIndexes(0, 1)
	workerIndexes := testutil.ReturnIndexes(1, 1)
	reputer := s.AddrsStr(reputerIndexes[0])
	stakeAmount := cosmosMath.NewInt(10)

	topicId := s.FullTopicSetup(workerIndexes, reputerIndexes, testutil.WithReputerStake(ptr.To(cosmosMath.ZeroInt()))).Id

	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	reputerStake, err := s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputer)
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), reputerStake, "Stake amount mismatch")

	topicStake, err := s.EmissionsKeeper().GetTopicStake(ctx, topicId)
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), topicStake, "Stake amount mismatch")

	// Remove sender from whitelist
	err = s.EmissionsKeeper().RemoveFromTopicReputerWhitelist(ctx, topicId, reputer)
	require.NoError(err)
	err = s.EmissionsKeeper().RemoveFromGlobalWhitelist(ctx, reputer)
	require.NoError(err)

	// Should now fail
	response, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.ErrorIs(err, types.ErrNotPermittedToAddStake)
	require.Nil(response)

	// Add sender back to whitelist
	err = s.EmissionsKeeper().AddToTopicReputerWhitelist(ctx, topicId, reputer)
	require.NoError(err)

	// Should now succeed
	response, err = s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	reputerStake, err = s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputer)
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	topicStake, err = s.EmissionsKeeper().GetTopicStake(ctx, topicId)
	require.NoError(err)
	require.Equal(stakeAmount, topicStake, "Stake amount mismatch")
}

func (s *MsgServerTestSuite) TestMsgAddStakeNil() {
	ctx := s.Ctx()
	require := s.Require()

	addStakeMsg := &types.AddStakeRequest{
		Sender:  s.AddrsStr(0),
		TopicId: uint64(1),
		Amount:  cosmosMath.Int{},
	}

	_, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.ErrorIs(err, sdkerrors.ErrInvalidCoins)
}

func (s *MsgServerTestSuite) TestStartRemoveStake() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)

	s.MintTokensToAddress(senderAddr, cosmosMath.NewInt(1000))

	// Assuming you have methods to directly manipulate the state
	// Simulate that sender has already staked the required amount
	err := s.EmissionsKeeper().AddReputerStake(ctx, topicId, sender, stakeAmount)
	require.NoError(err)

	msg := &types.RemoveStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	response, err := s.EmissionsMsgServer().RemoveStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response)

	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)
	expectedUnstake := ctx.BlockHeight() + moduleParams.RemoveStakeDelayWindow

	retrievedInfo, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, expectedUnstake, 100)
	require.NoError(err)
	require.NotNil(retrievedInfo)
	require.Len(retrievedInfo, 1)
	require.False(limitHit)

	expected := types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               sender,
		Amount:                stakeAmount,
		BlockRemovalStarted:   ctx.BlockHeight(),
		BlockRemovalCompleted: expectedUnstake,
	}
	s.Require().Equal(expected, retrievedInfo[0], "Stake removal info should match")
}

func (s *MsgServerTestSuite) TestStartRemoveStakeInsufficientStake() {
	ctx := s.Ctx()
	require := s.Require()

	sender := s.AddrsStr(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)

	msg := &types.RemoveStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	_, err := s.EmissionsMsgServer().RemoveStake(ctx, msg)
	require.ErrorIs(err, types.ErrInsufficientStakeToRemove)

	moduleParams, err := s.EmissionsKeeper().GetParams(ctx)
	require.NoError(err)
	expectedUnstake := ctx.BlockHeight() + moduleParams.RemoveStakeDelayWindow
	retrievedInfo, limitHit, err := s.EmissionsKeeper().GetStakeRemovalsUpUntilBlock(ctx, expectedUnstake, 100)
	require.NoError(err)
	require.Len(retrievedInfo, 0)
	require.False(limitHit)
}

func (s *MsgServerTestSuite) TestConfirmRemoveStake() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	senderAddr := s.Addrs(0)
	sender := s.AddrsStr(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()

	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow

	s.MintTokensToAddress(senderAddr, cosmosMath.NewInt(1000))
	s.MintTokensToModule(types.AlloraStakingAccountName, cosmosMath.NewInt(1000))
	err = s.EmissionsKeeper().AddReputerStake(s.Ctx(), topicId, sender, stakeAmount)
	require.NoError(err)
	blockEnd := startBlock + removalDelay

	// Simulate the stake removal request.
	placement := types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               senderAddr.String(),
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: blockEnd,
	}

	// Manually setting the removal in state (this part would normally involve interacting with the keeper to set up state).
	err = keeper.SetStakeRemoval(s.Ctx(), placement) // This assumes such a method exists.
	require.NoError(err)

	s.WithBlockHeight(blockEnd)

	// Perform the stake confirmation
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	require.NoError(err)

	finalStake, err := keeper.GetStakeReputerAuthority(s.Ctx(), topicId, senderAddr.String())
	require.NoError(err)
	require.True(finalStake.IsZero(), "Stake amount should be zero after removal is confirmed")

	// Check that the stake removal has been removed from the state
	removals, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(s.Ctx(), blockEnd, 100)
	require.NoError(err)
	require.Len(removals, 0)
	require.False(limitHit)
}

func (s *MsgServerTestSuite) TestStartRemoveStakeTwiceInSameBlock() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	sender := s.AddrsStr(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()

	// Fetch the delay window for removing stake
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	removeBlock := startBlock + removalDelay

	// Simulate that sender has already staked the required amount
	err = s.EmissionsKeeper().AddReputerStake(ctx, topicId, sender, stakeAmount)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().RemoveStake(ctx, &types.RemoveStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Amount:  stakeAmount,
	})
	s.Require().NoError(err)

	stakePlacements, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, removeBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements, 1)
	require.False(limitHit)

	expected := types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               sender,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: removeBlock,
	}
	require.Equal(expected, stakePlacements[0])

	newStake := stakeAmount.SubRaw(10)
	_, err = s.EmissionsMsgServer().RemoveStake(ctx, &types.RemoveStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Amount:  newStake,
	})
	s.Require().NoError(err)

	stakePlacements2, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, removeBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements2, 1)
	require.False(limitHit)
	expected2 := types.StakeRemovalInfo{
		TopicId:               expected.TopicId,
		Reputer:               expected.Reputer,
		Amount:                newStake,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: removeBlock,
	}
	require.Equal(expected2, stakePlacements2[0])
}

func (s *MsgServerTestSuite) TestRemoveStakeTwiceInDifferentBlocks() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	sender := s.AddrsStr(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()

	// Fetch the delay window for removing stake
	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	removeBlock := startBlock + removalDelay

	// Simulate that sender has already staked the required amount
	err = s.EmissionsKeeper().AddReputerStake(s.Ctx(), topicId, sender, stakeAmount)
	require.NoError(err)

	_, err = s.EmissionsMsgServer().RemoveStake(s.Ctx(), &types.RemoveStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Amount:  stakeAmount,
	})
	s.Require().NoError(err)

	stakePlacements, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(s.Ctx(), removeBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements, 1)
	require.False(limitHit)

	expected := types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               sender,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: removeBlock,
	}
	require.Equal(expected, stakePlacements[0])

	newStartBlock := startBlock + 5
	newRemoveBlock := newStartBlock + removalDelay
	newStake := stakeAmount.SubRaw(10)
	s.WithBlockHeight(newStartBlock)
	_, err = s.EmissionsMsgServer().RemoveStake(s.Ctx(), &types.RemoveStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Amount:  newStake,
	})
	s.Require().NoError(err)

	stakePlacements, limitHit, err = keeper.GetStakeRemovalsUpUntilBlock(s.Ctx(), removeBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements, 0)
	require.False(limitHit)
	stakePlacements, limitHit, err = keeper.GetStakeRemovalsUpUntilBlock(s.Ctx(), newRemoveBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements, 1)
	require.False(limitHit)
	expected.BlockRemovalStarted = newStartBlock
	expected.BlockRemovalCompleted = newRemoveBlock
	expected.Amount = newStake
	require.Equal(expected, stakePlacements[0])
}

func (s *MsgServerTestSuite) TestRemoveMultipleReputersSameBlock() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()
	senderAddr1 := s.Addrs(0)
	senderAddr2 := s.Addrs(1)
	topicId := uint64(1)
	stakeAmount1 := cosmosMath.NewInt(50)
	stakeAmount2 := cosmosMath.NewInt(30)
	startBlock := s.Ctx().BlockHeight()
	// Fetch the delay window for removing stake
	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	removeBlock := startBlock + removalDelay
	// Simulate that sender1 has already staked the required amount
	err = s.EmissionsKeeper().AddReputerStake(ctx, topicId, senderAddr1.String(), stakeAmount1)
	require.NoError(err)
	_, err = s.EmissionsMsgServer().RemoveStake(ctx, &types.RemoveStakeRequest{
		Sender:  senderAddr1.String(),
		TopicId: topicId,
		Amount:  stakeAmount1,
	})
	s.Require().NoError(err)
	stakePlacements1, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, removeBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements1, 1)
	require.False(limitHit)
	expected1 := types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               senderAddr1.String(),
		Amount:                stakeAmount1,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: removeBlock,
	}
	require.Equal(expected1, stakePlacements1[0])
	// Simulate that sender2 has already staked the required amount
	err = s.EmissionsKeeper().AddReputerStake(ctx, topicId, senderAddr2.String(), stakeAmount2)
	require.NoError(err)
	_, err = s.EmissionsMsgServer().RemoveStake(ctx, &types.RemoveStakeRequest{
		Sender:  senderAddr2.String(),
		TopicId: topicId,
		Amount:  stakeAmount2,
	})
	s.Require().NoError(err)
	stakePlacements2, limitHit, err := keeper.GetStakeRemovalsUpUntilBlock(ctx, removeBlock, 100)
	require.NoError(err)
	require.Len(stakePlacements2, 2)
	require.False(limitHit)
	expected2 := types.StakeRemovalInfo{
		TopicId:               topicId,
		Reputer:               senderAddr2.String(),
		Amount:                stakeAmount2,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: removeBlock,
	}
	require.Contains(stakePlacements2, expected1)
	require.Contains(stakePlacements2, expected2)
}

func (s *MsgServerTestSuite) TestStartRemoveStakeNegative() {
	ctx := s.Ctx()
	require := s.Require()
	senderAddr := s.Addrs(0)

	msg := &types.RemoveStakeRequest{
		Sender:  senderAddr.String(),
		TopicId: uint64(123),
		Amount:  cosmosMath.NewInt(-1),
	}

	_, err := s.EmissionsMsgServer().RemoveStake(ctx, msg)
	require.ErrorIs(err, sdkerrors.ErrInvalidCoins)
}

func (s *MsgServerTestSuite) TestStartRemoveStakeNil() {
	ctx := s.Ctx()
	require := s.Require()
	senderAddr := s.Addrs(0)

	msg := &types.RemoveStakeRequest{
		Sender:  senderAddr.String(),
		TopicId: uint64(123),
		Amount:  cosmosMath.Int{},
	}

	_, err := s.EmissionsMsgServer().RemoveStake(ctx, msg)
	require.ErrorIs(err, sdkerrors.ErrInvalidCoins)
}

func (s *MsgServerTestSuite) TestDelegateStake() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(7),
		NodeAddress: reputer,
	}

	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	require.NoError(err)

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	}

	reputerStake, err := s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputer)
	require.NoError(err)
	require.Equal(cosmosMath.ZeroInt(), reputerStake, "Stake amount mismatch")

	amount0, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	require.NoError(err)
	require.Equal(alloraMath.NewDecFromInt64(0), amount0.Amount)

	// Perform the stake delegation
	response, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	reputerStake, err = s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputer)
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	amount1, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	require.NoError(err)
	amountInt, err := amount1.Amount.SdkIntTrim()
	require.NoError(err)
	require.Equal(stakeAmount, amountInt)
}

func (s *MsgServerTestSuite) TestDelegateStakeNil() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(7),
		NodeAddress: reputer,
	}

	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	require.NoError(err)

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  cosmosMath.Int{},
	}
	_, err = s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.ErrorIs(err, sdkerrors.ErrInvalidCoins)
}

func (s *MsgServerTestSuite) TestReputerCantSelfDelegateStake() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	reputerAddr := delegatorAddr
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: s.AddrsStr(0),
	}

	err := keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)
	require.NoError(err)

	msg := &types.DelegateStakeRequest{
		Sender:  delegatorAddr.String(),
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	_, err = s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (s *MsgServerTestSuite) TestDelegateeCantWithdrawDelegatedStake() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputerAddr := s.Addrs(1)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: s.AddrsStr(0),
	}

	err := keeper.InsertReputer(ctx, topicId, reputerAddr.String(), reputerInfo)
	require.NoError(err)

	delegateStakeMsg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	}

	response, err := s.EmissionsMsgServer().DelegateStake(ctx, delegateStakeMsg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	reputerStake, err := s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputer)
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	amount1, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	require.NoError(err)
	amountInt, err := amount1.Amount.SdkIntTrim()
	require.NoError(err)
	require.Equal(stakeAmount, amountInt)

	// Attempt to withdraw the delegated stake
	removeMsg := &types.RemoveStakeRequest{
		Sender:  reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	_, err = s.EmissionsMsgServer().RemoveStake(ctx, removeMsg)
	require.Error(err)
}

func (s *MsgServerTestSuite) TestDelegateStakeUnregisteredReputer() {
	ctx := s.Ctx()
	require := s.Require()

	sender := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)

	// Do not register the reputer to simulate failure case

	msg := &types.DelegateStakeRequest{
		Sender:  sender,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	}

	// Attempt to perform the stake delegation
	response, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.Error(err)
	require.Nil(response, "Response should be nil when delegation fails due to unregistered reputer")
	require.True(errors.Is(err, types.ErrAddressIsNotRegisteredInThisTopic), "Error should indicate that the reputer is not registered in the topic")
}

func (s *MsgServerTestSuite) TestStartRemoveDelegateStake() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := moduleParams.RemoveStakeDelayWindow

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: reputer,
	}

	err = keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	require.NoError(err)

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	response, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	// Perform the stake removal initiation
	msg2 := &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	}
	response2, err := s.EmissionsMsgServer().RemoveDelegateStake(ctx, msg2)
	require.NoError(err)
	require.NotNil(response2, "Response should not be nil after successful stake removal initiation")

	// Verification: Check if the removal has been queued
	removeBlock := ctx.BlockHeight() + removalDelay
	removalInfo, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, removeBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removalInfo, 1)
	require.NotNil(removalInfo[0])
}

func (s *MsgServerTestSuite) TestStartRemoveDelegateStakeOnUnregisteredReputer() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	moduleParams, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := moduleParams.RemoveStakeDelayWindow

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: reputer,
	}

	err = keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	require.NoError(err)

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	response, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	err = keeper.RemoveReputer(ctx, topicId, reputer)
	require.NoError(err)

	// Perform the stake removal initiation
	msg2 := &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	}
	response2, err := s.EmissionsMsgServer().RemoveDelegateStake(ctx, msg2)
	require.NoError(err)
	require.NotNil(response2, "Response should not be nil after successful stake removal initiation")

	// Verification: Check if the removal has been queued
	removeBlock := ctx.BlockHeight() + removalDelay
	removalInfo, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, removeBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removalInfo, 1)
	require.NotNil(removalInfo[0])
}

func (s *MsgServerTestSuite) TestStartRemoveDelegateStakeError() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: s.AddrsStr(0),
	}

	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	require.NoError(err)

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	response, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	msg2 := &types.RemoveDelegateStakeRequest{
		Sender:  delegatorAddr.String(),
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount.Mul(cosmosMath.NewInt(2)),
	}

	// Perform the stake removal initiation
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(ctx, msg2)
	require.ErrorIs(err, types.ErrInsufficientDelegateStakeToRemove)
}

func (s *MsgServerTestSuite) TestConfirmRemoveDelegateStake() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()

	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	endBlock := startBlock + removalDelay

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	// Simulate adding a reputer and delegating stake to them
	err = keeper.InsertReputer(s.Ctx(), topicId, reputer, types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: reputer,
	})
	require.NoError(err)

	_, err = s.EmissionsMsgServer().DelegateStake(s.Ctx(), &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	// Start removing the delegated stake
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	// Simulate passing of time to surpass the withdrawal delay
	s.WithBlockHeight(endBlock)

	// Try to confirm removal after delay window
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	require.NoError(err)

	// Check that the stake was actually removed
	delegateStakePlaced, err := keeper.GetDelegateStakePlacement(s.Ctx(), topicId, delegator, reputer)
	require.NoError(err)
	require.True(delegateStakePlaced.Amount.IsZero(), "Delegate stake should be zero after successful removal")

	// Check that the stake removal has been removed from the state
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removals, 0)
}

// test you are able to restart the stake withdrawal
// if your stake expires or whatever other reason you may want
func (s *MsgServerTestSuite) TestStartRemoveDelegateStakeTwiceSameBlock() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()

	params, err := keeper.GetParams(ctx)
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	endBlock := startBlock + removalDelay

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	// Simulate adding a reputer and delegating stake to them
	err = keeper.InsertReputer(ctx, topicId, reputer, types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: reputer,
	})
	require.NoError(err)
	_, err = s.EmissionsMsgServer().DelegateStake(ctx, &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	// Start removing the delegated stake
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(ctx, &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	expected := types.DelegateStakeRemovalInfo{
		TopicId:               topicId,
		Delegator:             delegator,
		Reputer:               reputer,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	}

	stakePlacements, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(stakePlacements, 1)
	require.Equal(expected, stakePlacements[0])

	// Start removing the delegated stake again
	newStakeAmount := stakeAmount.SubRaw(10)
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(ctx, &types.RemoveDelegateStakeRequest{
		Sender:  delegatorAddr.String(),
		Reputer: reputer,
		TopicId: topicId,
		Amount:  newStakeAmount,
	})
	require.NoError(err)

	stakePlacements, limitHit, err = keeper.GetDelegateStakeRemovalsUpUntilBlock(ctx, endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(stakePlacements, 1)
	expected.Amount = newStakeAmount
	require.Equal(expected, stakePlacements[0])
}

func (s *MsgServerTestSuite) TestStartRemoveDelegateStakeTwice() {
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()

	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	endBlock := startBlock + removalDelay

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	// Simulate adding a reputer and delegating stake to them
	err = keeper.InsertReputer(s.Ctx(), topicId, reputer, types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: reputer,
	})
	require.NoError(err)
	_, err = s.EmissionsMsgServer().DelegateStake(s.Ctx(), &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputer,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	// Start removing the delegated stake
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	})
	require.NoError(err)

	expected := types.DelegateStakeRemovalInfo{
		TopicId:               topicId,
		Delegator:             delegator,
		Reputer:               reputer,
		Amount:                stakeAmount,
		BlockRemovalStarted:   startBlock,
		BlockRemovalCompleted: endBlock,
	}

	stakePlacements, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(stakePlacements, 1)
	require.Equal(expected, stakePlacements[0])

	// Now wait 5 blocks
	newStartBlock := s.Ctx().BlockHeight() + 5
	newEndBlock := endBlock + 5
	s.WithBlockHeight(newStartBlock)
	// Start removing the delegated stake again
	newStakeAmount := stakeAmount.SubRaw(10)
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  newStakeAmount,
	})
	require.NoError(err)

	stakePlacements, limitHit, err = keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(stakePlacements, 0)

	stakePlacements, limitHit, err = keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), newEndBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(stakePlacements, 1)

	expected.Amount = newStakeAmount
	expected.BlockRemovalStarted = newStartBlock
	expected.BlockRemovalCompleted = newEndBlock
	require.Equal(expected, stakePlacements[0])
}

func (s *MsgServerTestSuite) TestStartRemoveDelegateStakeNegative() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)

	reputerInfo := types.OffchainNode{
		Owner:       s.AddrsStr(7),
		NodeAddress: reputer,
	}

	err := keeper.InsertReputer(ctx, topicId, reputer, reputerInfo)
	require.NoError(err)

	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	// Perform the stake delegation
	response, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(response, "Response should not be nil after successful delegation")

	msg2 := &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(-1),
	}

	// Perform the stake removal initiation
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(ctx, msg2)
	require.ErrorIs(err, sdkerrors.ErrInvalidCoins)

	msg2 = &types.RemoveDelegateStakeRequest{
		Sender:  delegator,
		Reputer: reputer,
		TopicId: topicId,
		Amount:  cosmosMath.Int{},
	}

	// Perform the stake removal initiation
	_, err = s.EmissionsMsgServer().RemoveDelegateStake(ctx, msg2)
	require.ErrorIs(err, sdkerrors.ErrInvalidCoins)
}

func (s *MsgServerTestSuite) TestRemoveDelegateStakeMultipleReputersSameDelegator() {
	require := s.Require()
	keeper := s.EmissionsKeeper()
	delegatorAddr := s.Addrs(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	startBlock := s.Ctx().BlockHeight()
	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	endBlock := startBlock + removalDelay
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))
	// Simulate adding multiple reputers and delegating stake to them
	reputers := []sdk.AccAddress{
		s.Addrs(1),
		s.Addrs(2),
		s.Addrs(3),
	}
	for _, reputer := range reputers {
		reputerAddr := reputer.String()
		err := keeper.InsertReputer(s.Ctx(), topicId, reputerAddr, types.OffchainNode{
			Owner:       s.AddrsStr(0),
			NodeAddress: reputerAddr,
		})
		require.NoError(err)
		_, err = s.EmissionsMsgServer().DelegateStake(s.Ctx(), &types.DelegateStakeRequest{
			Sender:  delegatorAddr.String(),
			TopicId: topicId,
			Reputer: reputerAddr,
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}
	// Start removing the delegated stake for each reputer
	for _, reputer := range reputers {
		_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
			Sender:  delegatorAddr.String(),
			Reputer: reputer.String(),
			TopicId: topicId,
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}
	// Call ctx.WithBlockHeight to simulate passing time
	s.WithBlockHeight(endBlock)
	// Call EndBlock to trigger stake removal
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	require.NoError(err)
	// Check that the stake was actually removed for each reputer
	for _, reputer := range reputers {
		delegateStakePlaced, err := keeper.GetDelegateStakePlacement(s.Ctx(), topicId, delegatorAddr.String(), reputer.String())
		require.NoError(err)
		require.True(delegateStakePlaced.Amount.IsZero(), "Delegate stake should be zero after successful removal")
	}
	// Check that the stake removals have been removed from the state
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removals, 0)
}

func (s *MsgServerTestSuite) TestRemoveOneDelegateMultipleTargetsDifferentBlocks() {
	require := s.Require()
	keeper := s.EmissionsKeeper()
	delegator := s.AddrsStr(0)
	delegatorAddr := s.Addrs(0)
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	startBlock := s.Ctx().BlockHeight()
	endBlock := startBlock + removalDelay
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000))
	// Simulate adding multiple reputers and delegating stake to them
	reputers := []sdk.AccAddress{
		s.Addrs(1),
		s.Addrs(2),
		s.Addrs(3),
	}
	for _, reputer := range reputers {
		reputerStr := reputer.String()
		err := keeper.InsertReputer(s.Ctx(), topicId, reputerStr, types.OffchainNode{
			Owner:       s.AddrsStr(0),
			NodeAddress: reputerStr,
		})
		require.NoError(err)
		_, err = s.EmissionsMsgServer().DelegateStake(s.Ctx(), &types.DelegateStakeRequest{
			Sender:  delegator,
			TopicId: topicId,
			Reputer: reputerStr,
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}
	// Start removing the delegated stake for each reputer
	for _, reputer := range reputers {
		_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
			Sender:  delegator,
			Reputer: reputer.String(),
			TopicId: topicId,
			Amount:  stakeAmount,
		})
		require.NoError(err)
		s.WithBlockHeight(s.Ctx().BlockHeight() + 1)
	}

	// verify the removals are put in correctly
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock+int64(len(reputers)), 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removals, len(reputers))

	// Call ctx.WithBlockHeight to simulate passing time
	s.WithBlockHeight(endBlock)
	for i := 0; i < len(reputers); i++ {
		// Call EndBlock to trigger stake removal
		err = s.EmissionsAppModule().EndBlock(s.Ctx())
		require.NoError(err)

		// Check that the stake removals have been removed from the state
		removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
		require.NoError(err)
		require.False(limitHit)
		require.Len(removals, 0)
		s.WithBlockHeight(s.Ctx().BlockHeight() + 1)
	}
	// Check that the stake was actually removed for each reputer
	for _, reputer := range reputers {
		delegateStakePlaced, err := keeper.GetDelegateStakePlacement(
			s.Ctx(),
			topicId,
			delegator,
			reputer.String(),
		)
		require.NoError(err)
		require.True(
			delegateStakePlaced.Amount.IsZero(),
			"Delegate stake should be zero after successful removal",
			delegateStakePlaced.Amount,
			reputer.String(),
		)
	}
}

func (s *MsgServerTestSuite) TestRemoveMultipleDelegatesSameTargetSameBlock() {
	require := s.Require()
	keeper := s.EmissionsKeeper()
	delegators := []sdk.AccAddress{
		s.Addrs(0),
		s.Addrs(1),
		s.Addrs(2),
	}
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	startBlock := s.Ctx().BlockHeight()
	endBlock := startBlock + removalDelay
	for _, delegator := range delegators {
		s.MintTokensToAddress(delegator, cosmosMath.NewInt(1000))
	}
	// Simulate adding multiple reputers and delegating stake to them
	reputer := s.Addrs(3)
	err = keeper.InsertReputer(s.Ctx(), topicId, reputer.String(), types.OffchainNode{
		Owner:       s.AddrsStr(0),
		NodeAddress: reputer.String(),
	})
	require.NoError(err)
	for _, delegator := range delegators {
		_, err = s.EmissionsMsgServer().DelegateStake(s.Ctx(), &types.DelegateStakeRequest{
			Sender:  delegator.String(),
			TopicId: topicId,
			Reputer: reputer.String(),
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}
	// Start removing the delegated stake for each reputer
	for _, delegator := range delegators {
		_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
			Sender:  delegator.String(),
			Reputer: reputer.String(),
			TopicId: topicId,
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}

	// verify the removals are put in correctly
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removals, len(delegators))

	// Call ctx.WithBlockHeight to simulate passing time
	s.WithBlockHeight(endBlock)
	for i := 0; i < len(delegators); i++ {
		// Call EndBlock to trigger stake removal
		err = s.EmissionsAppModule().EndBlock(s.Ctx())
		require.NoError(err)

		// Check that the stake removals have been removed from the state
		removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
		require.NoError(err)
		require.False(limitHit)
		require.Len(removals, 0)
	}
	// Check that the stake was actually removed for each reputer
	for _, delegator := range delegators {
		delegateStakePlaced, err := keeper.GetDelegateStakePlacement(
			s.Ctx(),
			topicId,
			delegator.String(),
			reputer.String(),
		)
		require.NoError(err)
		require.True(
			delegateStakePlaced.Amount.IsZero(),
			"Delegate stake should be zero after successful removal",
			delegateStakePlaced.Amount,
			reputer.String(),
		)
	}
}

func (s *MsgServerTestSuite) TestRemoveMultipleDelegatesDifferentTargetsSameBlock() {
	require := s.Require()
	keeper := s.EmissionsKeeper()
	delegators := []sdk.AccAddress{
		s.Addrs(0),
		s.Addrs(1),
	}
	topicId := uint64(1)
	stakeAmount := cosmosMath.NewInt(50)
	params, err := keeper.GetParams(s.Ctx())
	require.NoError(err)
	removalDelay := params.RemoveStakeDelayWindow
	startBlock := s.Ctx().BlockHeight()
	endBlock := startBlock + removalDelay
	s.MintTokensToAddress(delegators[0], cosmosMath.NewInt(1000))
	s.MintTokensToAddress(delegators[1], cosmosMath.NewInt(1000))
	// Simulate adding multiple reputers and delegating stake to them
	reputers := []sdk.AccAddress{
		s.Addrs(2),
		s.Addrs(3),
	}
	for i := 0; i < len(delegators); i++ {
		reputerAddr := reputers[i].String()
		delegatorAddr := delegators[i].String()
		err := keeper.InsertReputer(s.Ctx(), topicId, reputerAddr, types.OffchainNode{
			Owner:       s.AddrsStr(0),
			NodeAddress: reputerAddr,
		})
		require.NoError(err)
		_, err = s.EmissionsMsgServer().DelegateStake(s.Ctx(), &types.DelegateStakeRequest{
			Sender:  delegatorAddr,
			TopicId: topicId,
			Reputer: reputerAddr,
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}
	// Start removing the delegated stake for each reputer
	for i := 0; i < len(delegators); i++ {
		_, err = s.EmissionsMsgServer().RemoveDelegateStake(s.Ctx(), &types.RemoveDelegateStakeRequest{
			Sender:  delegators[i].String(),
			Reputer: reputers[i].String(),
			TopicId: topicId,
			Amount:  stakeAmount,
		})
		require.NoError(err)
	}

	// verify the removals are put in correctly
	removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
	require.NoError(err)
	require.False(limitHit)
	require.Len(removals, len(delegators))

	// Call ctx.WithBlockHeight to simulate passing time
	s.WithBlockHeight(endBlock)
	for i := 0; i < len(reputers); i++ {
		// Call EndBlock to trigger stake removal
		err = s.EmissionsAppModule().EndBlock(s.Ctx())
		require.NoError(err)

		// Check that the stake removals have been removed from the state
		removals, limitHit, err := keeper.GetDelegateStakeRemovalsUpUntilBlock(s.Ctx(), endBlock, 100)
		require.NoError(err)
		require.False(limitHit)
		require.Len(removals, 0)
	}
	// Check that the stake was actually removed for each reputer
	for i := 0; i < len(reputers); i++ {
		delegateStakePlaced, err := keeper.GetDelegateStakePlacement(
			s.Ctx(),
			topicId,
			delegators[i].String(),
			reputers[i].String(),
		)
		require.NoError(err)
		require.True(
			delegateStakePlaced.Amount.IsZero(),
			"Delegate stake should be zero after successful removal",
			delegateStakePlaced.Amount,
			reputers[i].String(),
		)
	}
}

func (s *MsgServerTestSuite) TestRewardDelegateStake() {
	ctx := s.Ctx()
	require := s.Require()
	keeper := s.EmissionsKeeper()

	block := int64(1003)
	newBlock := int64(1004)
	score := alloraMath.MustNewDecFromString("17.53436")

	delegator := s.AddrsStr(0)
	delegatorAddr := s.Addrs(0)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	workerIndexes := testutil.ReturnIndexes(2, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	delegator2 := s.AddrsStr(3)
	delegator2Addr := s.Addrs(3)
	stakeAmount := cosmosMath.NewInt(500000)
	delegatorStakeAmount := cosmosMath.NewInt(500)
	delegatorStakeAmount2 := cosmosMath.NewInt(500)
	delegator2StakeAmount := cosmosMath.NewInt(5000)

	topicId := s.FullTopicSetup(workerIndexes, reputerIndexes, testutil.WithReputerStake(ptr.To(cosmosMath.ZeroInt()))).Id
	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegator2Addr, cosmosMath.NewInt(1000000))

	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	response, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegatorStakeAmount,
	}

	msg2 := &types.DelegateStakeRequest{
		Sender:  delegator2,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegator2StakeAmount,
	}

	reputerStake, err := s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	amount0, err := keeper.GetDelegateStakePlacement(ctx, topicId, delegator, reputerAddr.String())
	require.NoError(err)
	require.Equal(alloraMath.NewDecFromInt64(0), amount0.Amount)

	// Perform the stake delegation
	responseDelegator, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(responseDelegator, "Response should not be nil after successful delegation")

	responseDelegator2, err := s.EmissionsMsgServer().DelegateStake(ctx, msg2)
	require.NoError(err)
	require.NotNil(responseDelegator2, "Response should not be nil after successful delegation")

	var reputerValueBundles types.ReputerValueBundles
	scoreToAdd := types.Score{
		TopicId:     topicId,
		BlockHeight: block,
		Address:     reputerAddr.String(),
		Score:       score,
	}
	err = s.EmissionsKeeper().InsertReputerScore(s.Ctx(), topicId, block, scoreToAdd)
	s.Require().NoError(err)

	//nolint:exhaustruct
	reputerValueBundle := &types.ReputerValueBundle{
		Pubkey: s.PubKeyHexStr(1),
		ValueBundle: &types.ValueBundle{
			TopicId:             topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: block}},
			Reputer:             reputerAddr.String(),
			InfererValues:       []*types.WorkerAttributedValue{{Worker: s.AddrsStr(workerIndexes[0])}},
		},
		Signature: []byte{},
	}
	reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, reputerValueBundle)
	err = s.EmissionsKeeper().InsertActiveReputerLosses(s.Ctx(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err := rewards.GenerateReputerScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Generate rewards
	reputers, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(1, len(reputerRewards))

	msg3 := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegatorStakeAmount2,
	}

	responseDelegator3, err := s.EmissionsMsgServer().DelegateStake(ctx, msg3)
	require.NoError(err)
	require.NotNil(responseDelegator3, "Response should not be nil after successful delegation")

	var newReputerValueBundles types.ReputerValueBundles
	newScoreToAdd := scoreToAdd
	newScoreToAdd.BlockHeight = newBlock

	err = s.EmissionsKeeper().InsertReputerScore(s.Ctx(), topicId, newBlock, newScoreToAdd)
	s.Require().NoError(err)

	newReputerValueBundle := reputerValueBundle
	newReputerValueBundle.ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight = newBlock
	newReputerValueBundle.Pubkey = s.PubKeyHexStr(1)

	newReputerValueBundles.ReputerValueBundles = append(newReputerValueBundles.ReputerValueBundles, newReputerValueBundle)
	err = s.EmissionsKeeper().InsertActiveReputerLosses(s.Ctx(), topicId, newBlock, newReputerValueBundles)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err = rewards.GenerateReputerScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Generate new rewards
	reputers, reputersRewardFractions, err = rewards.GetReputersRewardFractions(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)
	newReputerRewards, err := rewards.GetRewardPerReputer(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		alloraMath.MustNewDecFromString("1020.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(1, len(newReputerRewards))

	beforeBalance := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	rewardMsg := &types.RewardDelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)
	afterBalance := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	s.Require().NoError(err)
	s.Require().Greater(afterBalance.Amount.Uint64(), beforeBalance.Amount.Uint64(), "Balance must be increased")

	beforeBalance2 := s.BankKeeper().GetBalance(ctx, delegator2Addr, params.DefaultBondDenom)
	rewardMsg2 := &types.RewardDelegateStakeRequest{
		Sender:  delegator2,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg2)
	afterBalance2 := s.BankKeeper().GetBalance(ctx, delegator2Addr, params.DefaultBondDenom)
	s.Require().NoError(err)
	s.Require().Greater(afterBalance2.Amount.Uint64(), beforeBalance2.Amount.Uint64(), "Balance must be increased")
}

func (s *MsgServerTestSuite) TestRewardDelegateStakeOnUnregisteredReputer() {
	ctx := s.Ctx()
	require := s.Require()

	block := int64(1003)
	score := alloraMath.MustNewDecFromString("17.53436")

	delegator := s.AddrsStr(0)
	delegatorAddr := s.Addrs(0)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	workerIndexes := testutil.ReturnIndexes(2, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])

	stakeAmount := cosmosMath.NewInt(500000)
	delegatorStakeAmount := cosmosMath.NewInt(500)

	topicId := s.FullTopicSetup(workerIndexes, reputerIndexes, testutil.WithReputerStake(ptr.To(cosmosMath.ZeroInt()))).Id
	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000000))

	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	response, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegatorStakeAmount,
	}

	// Perform the stake delegation
	responseDelegator, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(responseDelegator, "Response should not be nil after successful delegation")

	var reputerValueBundles types.ReputerValueBundles
	scoreToAdd := types.Score{
		TopicId:     topicId,
		BlockHeight: block,
		Address:     reputerAddr.String(),
		Score:       score,
	}
	err = s.EmissionsKeeper().InsertReputerScore(s.Ctx(), topicId, block, scoreToAdd)
	s.Require().NoError(err)

	//nolint:exhaustruct
	reputerValueBundle := &types.ReputerValueBundle{
		Pubkey: s.PubKeyHexStr(1),
		ValueBundle: &types.ValueBundle{
			TopicId:             topicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: block}},
			Reputer:             reputerAddr.String(),
			InfererValues:       []*types.WorkerAttributedValue{{Worker: s.AddrsStr(workerIndexes[0])}},
		},
		Signature: []byte{},
	}
	reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, reputerValueBundle)
	err = s.EmissionsKeeper().InsertActiveReputerLosses(s.Ctx(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err := rewards.GenerateReputerScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Generate rewards
	reputers, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(1, len(reputerRewards))

	beforeBalance := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	s.Require().NoError(s.EmissionsKeeper().RemoveReputer(ctx, topicId, reputerAddr.String()))

	rewardMsg := &types.RewardDelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)
	afterBalance := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	s.Require().NoError(err)
	s.Require().Greater(afterBalance.Amount.Uint64(), beforeBalance.Amount.Uint64(), "Balance must be increased")
}

func (s *MsgServerTestSuite) insertValueBundlesAndGetRewards(
	reputerPrivKey secp256k1.PrivKey,
	reputer string,
	reputerPubKeyHex string,
	topicId uint64,
	block int64,
	score alloraMath.Dec,
) []types.TaskReward {
	keeper := *s.EmissionsKeeper()
	var reputerValueBundles types.ReputerValueBundles
	scoreToAdd := types.Score{
		TopicId:     topicId,
		BlockHeight: block,
		Address:     reputer,
		Score:       score,
	}
	err := keeper.InsertReputerScore(s.Ctx(), topicId, block, scoreToAdd)
	s.Require().NoError(err)
	//nolint:exhaustruct
	valueBundle := &types.ValueBundle{
		TopicId:             topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: block}},
		Reputer:             reputer,
		InfererValues:       []*types.WorkerAttributedValue{{Worker: s.AddrsStr(0)}},
	}
	signature := s.SignValueBundle(valueBundle, reputerPrivKey)
	reputerValueBundle := &types.ReputerValueBundle{
		ValueBundle: valueBundle,
		Signature:   signature,
		Pubkey:      reputerPubKeyHex,
	}
	reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, reputerValueBundle)
	err = keeper.InsertActiveReputerLosses(s.Ctx(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Calculate and Set the reputer scores
	scores, err := rewards.GenerateReputerScores(s.Ctx(), *s.EmissionsKeeper(), topicId, block, reputerValueBundles)
	s.Require().NoError(err)

	// Generate rewards
	reputers, reputersRewardFractions, err := rewards.GetReputersRewardFractions(
		s.Ctx(),
		keeper,
		topicId,
		alloraMath.OneDec(),
		scores,
	)
	s.Require().NoError(err)
	reputerRewards, err := rewards.GetRewardPerReputer(
		s.Ctx(),
		keeper,
		topicId,
		alloraMath.MustNewDecFromString("1017.5559072418691"),
		reputers,
		reputersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(1, len(reputerRewards))

	return reputerRewards
}

func (s *MsgServerTestSuite) TestRewardConversionsOverInt64Limit() {
	// Initialize with a value > int64 max (9223372036854775807)
	intValueAsString := "18395576023021260086"
	newDec := alloraMath.MustNewDecFromString(intValueAsString + ".00000000000000")
	rewardInt, err := newDec.SdkIntTrim()
	s.Require().NoError(err)
	s.Require().Equal(intValueAsString, rewardInt.String(), "The SdkIntTrim method should return int part")

	// Create cosmos int from string
	cosmosIntFromString, ok := cosmosMath.NewIntFromString(intValueAsString)
	s.Require().Equal(true, ok)
	// Assert the expected result
	s.Require().True(rewardInt.Equal(cosmosIntFromString), "The cosmos ints created from string or Dec should match: %s = %s", rewardInt.String(), cosmosIntFromString.String())

	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosIntFromString))
	s.Require().Equal(intValueAsString+"uallo", coins.String(), "The sdk.Coins object should be created with the correct amount")

	// Create cosmos int from cosmos int
	cosmosInt := cosmosIntFromString
	s.Require().True(rewardInt.Equal(cosmosInt), "The cosmos ints created from ints should match: %s = %s", rewardInt.String(), cosmosInt.String())
}

func (s *MsgServerTestSuite) TestRewardConversionsZeroIntWithDecimals() {
	// Initialize with a value > int64 max (9223372036854775807)
	zeroStr := "0"
	newDec := alloraMath.MustNewDecFromString(zeroStr + ".000000000001")
	s.Require().Equal(newDec.IsZero(), false)
	decTrimmedToInt, err := newDec.SdkIntTrim()
	s.Require().NoError(err)
	s.Require().Equal(zeroStr, decTrimmedToInt.String(), "SdkIntTrim with zero-int should return int part")

	// Create cosmos int from string
	intFromStr, ok := cosmosMath.NewIntFromString(zeroStr)
	s.Require().Equal(true, ok)
	// Assert the expected result
	s.Require().True(intFromStr.Equal(decTrimmedToInt), "Trimming a decimal 1 > dec > 0 to int should return 0")

	// Create cosmos int from cosmos int
	intCreatedFromInt := intFromStr
	s.Require().True(
		intCreatedFromInt.Equal(decTrimmedToInt),
		"A cosmos zero-ints created from another int should still equal a dec trimmed to zero")
}

func (s *MsgServerTestSuite) TestEqualStakeRewardsToDelegatorAndReputer() {
	ctx := s.Ctx()
	require := s.Require()

	block := int64(1003)
	score := alloraMath.MustNewDecFromString("17.53436")
	stakeAmount := cosmosMath.NewInt(500000)

	delegator := s.AddrsStr(0)
	delegatorAddr := s.Addrs(0)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	workerIndexes := testutil.ReturnIndexes(2, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerPubKeyHex := s.PubKeyHexStr(reputerIndexes[0])
	reputerPrivKey := s.PrivKeys(reputerIndexes[0])

	topicId := s.FullTopicSetup(workerIndexes, reputerIndexes, testutil.WithReputerStake(ptr.To(cosmosMath.ZeroInt()))).Id
	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000000))

	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  stakeAmount,
	}

	response, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  stakeAmount,
	}

	reputerStake, err := s.EmissionsKeeper().GetStakeReputerAuthority(ctx, topicId, reputerAddr.String())
	require.NoError(err)
	require.Equal(stakeAmount, reputerStake, "Stake amount mismatch")

	// Perform the stake delegation
	responseDelegator, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(responseDelegator, "Response should not be nil after successful delegation")

	reputerRewards := s.insertValueBundlesAndGetRewards(reputerPrivKey, reputerAddr.String(), reputerPubKeyHex, topicId, block, score)

	delegatorBal0 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	rewardMsg := &types.RewardDelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)
	s.Require().NoError(err)

	delegatorBal1 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	s.Require().NoError(err)

	s.Require().Greater(delegatorBal1.Amount.Uint64(), delegatorBal0.Amount.Uint64(), "Balance must be increased")

	delegatorReward0 := delegatorBal1.Amount.Sub(delegatorBal0.Amount)
	reputerReward, err := reputerRewards[0].Reward.SdkIntTrim()
	s.Require().NoError(err)

	// in the case where the rewards is an odd number e.g.
	// 9 / 2 = 4.5
	// the delegator gets the number rounded down, e.g. 4
	// and the reputer gets the number rounded up, e.g. 5
	condition := delegatorReward0.Equal(reputerReward) || delegatorReward0.AddRaw(1).Equal(reputerReward)
	s.Require().True(condition,
		fmt.Sprintf("Delegator and reputer rewards must be equal: %s | %s",
			delegatorReward0.String(), reputerReward.String()),
	)

	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)
	s.Require().NoError(err)
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)
	s.Require().NoError(err)
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)
	s.Require().NoError(err)

	delegatorBal2 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	delegatorReward1 := delegatorBal2.Amount.Sub(delegatorBal1.Amount)

	s.Require().True(delegatorReward1.Equal(cosmosMath.NewInt(0)), "Delegator cant double claim rewards")
}

func (s *MsgServerTestSuite) Test1000xDelegatorStakeVsReputerStake() {
	ctx := s.Ctx()
	require := s.Require()
	block := int64(1003)
	score := alloraMath.MustNewDecFromString("17.53436")

	delegatorAddr := s.Addrs(0)
	delegator := s.AddrsStr(0)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	workerIndexes := testutil.ReturnIndexes(2, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerPubKeyHex := s.PubKeyHexStr(reputerIndexes[0])
	reputerPrivKey := s.PrivKeys(reputerIndexes[0])

	reputerStakeAmount := cosmosMath.NewInt(1e2)
	delegatorRatio := cosmosMath.NewInt(1e3)
	delegatorStakeAmount := reputerStakeAmount.Mul(delegatorRatio)

	topicId := s.FullTopicSetup(workerIndexes, reputerIndexes, testutil.WithReputerStake(ptr.To(cosmosMath.ZeroInt()))).Id
	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000000))
	err := s.BankKeeper().MintCoins(ctx, types.AlloraRewardsAccountName,
		sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(1000000))))
	require.NoError(err)

	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  reputerStakeAmount,
	}

	response, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	msg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  delegatorStakeAmount,
	}

	// Perform the stake delegation
	delegateResponse, err := s.EmissionsMsgServer().DelegateStake(ctx, msg)
	require.NoError(err)
	require.NotNil(delegateResponse, "Response should not be nil after successful delegation")

	reputerRewards := s.insertValueBundlesAndGetRewards(reputerPrivKey, reputerAddr.String(), reputerPubKeyHex, topicId, block, score)

	delegatorBal0 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	rewardMsg := &types.RewardDelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, rewardMsg)

	delegatorBal1 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	s.Require().NoError(err)

	delegatorRewardRaw := delegatorBal1.Amount.Sub(delegatorBal0.Amount)
	reputerReward, err := reputerRewards[0].Reward.SdkIntTrim()
	s.Require().NoError(err)
	normalizedDelegatorReward, err := alloraMath.NewDecFromInt64(delegatorRewardRaw.Int64()).Quo(alloraMath.NewDecFromInt64(delegatorRatio.Int64()))
	s.Require().NoError(err)

	normalizedDelegatorRewardInt, err := normalizedDelegatorReward.SdkIntTrim()
	s.Require().NoError(err)
	s.Require().Equal(normalizedDelegatorRewardInt, reputerReward, "Delegator and reputer rewards must be equal")
}

func (s *MsgServerTestSuite) TestMultiRoundReputerStakeVs1000xDelegatorStake() {
	ctx := s.Ctx()
	require := s.Require()
	block := int64(1000)
	score := alloraMath.MustNewDecFromString("17.53436")

	reputerIndexes := testutil.ReturnIndexes(0, 1)
	reputerAddr := s.Addrs(reputerIndexes[0])
	reputerPubKeyHex := s.PubKeyHexStr(reputerIndexes[0])
	reputerPrivKey := s.PrivKeys(reputerIndexes[0])
	delegatorAddr := s.Addrs(1)
	delegator := s.AddrsStr(1)
	largeDelegatorAddr := s.Addrs(2)
	largeDelegator := s.AddrsStr(2)
	workerIndexes := testutil.ReturnIndexes(3, 1)

	reputerStakeAmount := cosmosMath.NewInt(1e2)
	largeDelegatorRatio := cosmosMath.NewInt(1e3)
	largeDelegatorStakeAmount := reputerStakeAmount.Mul(largeDelegatorRatio)

	zero := cosmosMath.ZeroInt()
	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithInitialTopicStake(zero),
		testutil.WithReputerStake(&zero),
	)
	topicId := topic.GetId()

	s.MintTokensToAddress(reputerAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(delegatorAddr, cosmosMath.NewInt(1000000))
	s.MintTokensToAddress(largeDelegatorAddr, cosmosMath.NewInt(1000000))
	err := s.BankKeeper().MintCoins(ctx, types.AlloraRewardsAccountName, sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(1000000))))
	require.NoError(err)

	// STEP 1 stake equal amount for reputer and delegator
	addStakeMsg := &types.AddStakeRequest{
		Sender:  reputerAddr.String(),
		TopicId: topicId,
		Amount:  reputerStakeAmount,
	}

	response, err := s.EmissionsMsgServer().AddStake(ctx, addStakeMsg)
	require.NoError(err, "AddStake should not return an error")
	require.NotNil(response)

	addDelegateStakeMsg := &types.DelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  reputerStakeAmount,
	}

	delegateStakeResponse, err := s.EmissionsMsgServer().DelegateStake(ctx, addDelegateStakeMsg)
	require.NoError(err)
	require.NotNil(delegateStakeResponse, "Response should not be nil after successful delegation")

	// STEP 2 Calculate rewards for the first round
	reputerReward0, err := s.insertValueBundlesAndGetRewards(
		reputerPrivKey, reputerAddr.String(), reputerPubKeyHex, topicId, block, score)[0].Reward.SdkIntTrim()
	require.NoError(err)

	delegatorBal0 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)

	delegateRewardsMsg := &types.RewardDelegateStakeRequest{
		Sender:  delegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, delegateRewardsMsg)
	require.NoError(err)

	delegatorBal1 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)

	delegatorReward0 := delegatorBal1.Amount.Sub(delegatorBal0.Amount)

	// STEP 2 Calculate rewards for the second round
	block++

	reputerReward1, err := s.insertValueBundlesAndGetRewards(
		reputerPrivKey, reputerAddr.String(), reputerPubKeyHex, topicId, block, score)[0].Reward.SdkIntTrim()
	require.NoError(err)

	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, delegateRewardsMsg)
	require.NoError(err)

	delegatorBal2 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)

	delegatorReward1 := delegatorBal2.Amount.Sub(delegatorBal1.Amount)

	// STEP 3 stake 1000x more for large delegator
	addLargeDelegateStakeMsg := &types.DelegateStakeRequest{
		Sender:  largeDelegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
		Amount:  largeDelegatorStakeAmount,
	}

	largeDelegateStakeResponse, err := s.EmissionsMsgServer().DelegateStake(ctx, addLargeDelegateStakeMsg)
	require.NoError(err)
	require.NotNil(largeDelegateStakeResponse, "Response should not be nil after successful delegation")

	largeDelegatorBal2 := s.BankKeeper().GetBalance(ctx, largeDelegatorAddr, params.DefaultBondDenom)

	// STEP 4 Calculate rewards for the third round
	block++
	reputerReward2, err := s.insertValueBundlesAndGetRewards(
		reputerPrivKey, reputerAddr.String(), reputerPubKeyHex, topicId, block, score)[0].Reward.SdkIntTrim()
	require.NoError(err)

	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, delegateRewardsMsg)
	require.NoError(err)

	largeDelegateRewardsMsg := &types.RewardDelegateStakeRequest{
		Sender:  largeDelegator,
		TopicId: topicId,
		Reputer: reputerAddr.String(),
	}
	_, err = s.EmissionsMsgServer().RewardDelegateStake(ctx, largeDelegateRewardsMsg)
	require.NoError(err)

	delegatorBal3 := s.BankKeeper().GetBalance(ctx, delegatorAddr, params.DefaultBondDenom)
	largeDelegatorBal3 := s.BankKeeper().GetBalance(ctx, largeDelegatorAddr, params.DefaultBondDenom)

	delegatorReward2 := delegatorBal3.Amount.Sub(delegatorBal2.Amount)
	largeDelegatorReward2 := largeDelegatorBal3.Amount.Sub(largeDelegatorBal2.Amount)

	condition := delegatorReward0.Equal(reputerReward0) || delegatorReward0.AddRaw(1).Equal(reputerReward0)
	require.True(condition,
		fmt.Sprintf("Delegator and reputer rewards must be equal (or reputer = delegator + 1) in all rounds: %s | %s",
			delegatorReward0.String(), reputerReward0.String()),
	)
	condition = delegatorReward1.Equal(reputerReward1) || delegatorReward1.AddRaw(1).Equal(reputerReward1)
	require.True(condition, fmt.Sprintf("Delegator and reputer rewards must be equal in all rounds %s | %s",
		delegatorReward1.String(), reputerReward1.String()),
	)
	require.Equal(reputerReward0, reputerReward1, "Delegator and reputer rewards must be equal from the first to the second round")
	condition = delegatorReward2.Equal(reputerReward2) || delegatorReward2.AddRaw(1).Equal(reputerReward2)
	require.True(condition, fmt.Sprintf("Delegator and reputer rewards must be equal in all rounds %s | %s",
		delegatorReward2.String(), reputerReward2.String()),
	)

	normalizedLargeDelegatorReward, err := alloraMath.NewDecFromInt64(largeDelegatorReward2.Int64()).Quo(alloraMath.NewDecFromInt64(largeDelegatorRatio.Int64()))
	require.NoError(err)

	normalizedLargeDelegatorRewardInt, err := normalizedLargeDelegatorReward.SdkIntTrim()
	require.NoError(err)
	require.Equal(normalizedLargeDelegatorRewardInt, reputerReward2, "Normalized large delegator rewards must be equal to reputer rewards")
	require.Equal(normalizedLargeDelegatorRewardInt, delegatorReward2, "Normalized large delegator rewards must be equal to delegator rewards")

	totalRewardsSecondRound := reputerReward1.Add(delegatorReward1)
	totalRewardsThirdRound := reputerReward2.Add(delegatorReward2).Add(largeDelegatorReward2)

	alloratestutil.InEpsilon3(s.T(), alloraMath.MustNewDecFromString(totalRewardsSecondRound.String()), totalRewardsThirdRound.String())
}

func (s *MsgServerTestSuite) TestCancelRemoveStake() {
	ctx := s.Ctx()
	require := s.Require()

	// Set up test data
	reputer := s.AddrsStr(0)
	topicId := uint64(1)
	amount := cosmosMath.NewInt(50)

	// Add a delegate stake removal
	stakeToRemove := types.StakeRemovalInfo{
		BlockRemovalStarted:   10,
		TopicId:               topicId,
		Reputer:               reputer,
		Amount:                amount,
		BlockRemovalCompleted: 20,
	}
	err := s.EmissionsKeeper().SetStakeRemoval(ctx, stakeToRemove)
	require.NoError(err)

	// Call CancelRemoveDelegateStake
	msg := &types.CancelRemoveStakeRequest{
		Sender:  reputer,
		TopicId: topicId,
	}
	_, err = s.EmissionsMsgServer().CancelRemoveStake(ctx, msg)
	require.NoError(err)

	// Verify that the stake removal is deleted
	_, found, err := s.EmissionsKeeper().
		GetStakeRemovalForReputerAndTopicId(ctx, reputer, topicId)
	require.NoError(err)
	require.False(found, "Stake removal should be deleted")
}

func (s *MsgServerTestSuite) TestCancelRemoveStakeNotExist() {
	ctx := s.Ctx()
	require := s.Require()
	// Set up test data
	reputer := s.AddrsStr(0)
	topicId := uint64(1)

	// Call CancelRemoveDelegateStake
	msg := &types.CancelRemoveStakeRequest{
		Sender:  reputer,
		TopicId: topicId,
	}
	_, err := s.EmissionsMsgServer().CancelRemoveStake(ctx, msg)
	require.Error(err)
	require.True(errors.Is(err, types.ErrStakeRemovalNotFound), "Expected stake removal not found error")
}

func (s *MsgServerTestSuite) TestCancelRemoveDelegateStake() {
	ctx := s.Ctx()
	require := s.Require()
	// Set up test data
	delegator := s.AddrsStr(0)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	workerIndexes := testutil.ReturnIndexes(2, 1)
	reputer := s.AddrsStr(reputerIndexes[0])
	amount := cosmosMath.NewInt(50)

	zero := cosmosMath.ZeroInt()
	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithInitialTopicStake(zero),
		testutil.WithReputerStake(&zero),
	)
	topicId := topic.GetId()

	// Add a delegate stake removal
	stakeToRemove := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   10,
		TopicId:               topicId,
		Reputer:               reputer,
		Delegator:             delegator,
		Amount:                amount,
		BlockRemovalCompleted: 20,
	}
	err := s.EmissionsKeeper().SetDelegateStakeRemoval(ctx, stakeToRemove)
	require.NoError(err)

	// Call CancelRemoveDelegateStake
	msg := &types.CancelRemoveDelegateStakeRequest{
		Sender:    delegator,
		Reputer:   reputer,
		TopicId:   topicId,
		Delegator: delegator,
	}
	_, err = s.EmissionsMsgServer().CancelRemoveDelegateStake(ctx, msg)
	require.NoError(err)

	// Verify that the stake removal is deleted
	_, found, err := s.EmissionsKeeper().
		GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	require.NoError(err)
	require.False(found, "Stake removal should be deleted")
}

func (s *MsgServerTestSuite) TestCancelRemoveDelegateStakeOnUnregisteredReputer() {
	ctx := s.Ctx()
	require := s.Require()
	// Set up test data
	delegator := s.AddrsStr(0)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	reputer := s.AddrsStr(reputerIndexes[0])
	workerIndexes := testutil.ReturnIndexes(2, 1)
	amount := cosmosMath.NewInt(50)

	topicId := s.FullTopicSetup(workerIndexes, reputerIndexes).Id

	// Add a delegate stake removal
	stakeToRemove := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   10,
		TopicId:               topicId,
		Reputer:               reputer,
		Delegator:             delegator,
		Amount:                amount,
		BlockRemovalCompleted: 20,
	}
	err := s.EmissionsKeeper().SetDelegateStakeRemoval(ctx, stakeToRemove)
	require.NoError(err)

	s.Require().NoError(s.EmissionsKeeper().RemoveReputer(ctx, topicId, reputer))

	// Call CancelRemoveDelegateStake
	msg := &types.CancelRemoveDelegateStakeRequest{
		Sender:    delegator,
		Reputer:   reputer,
		TopicId:   topicId,
		Delegator: delegator,
	}
	_, err = s.EmissionsMsgServer().CancelRemoveDelegateStake(ctx, msg)
	require.NoError(err)

	// Verify that the stake removal is deleted
	_, found, err := s.EmissionsKeeper().
		GetDelegateStakeRemovalForDelegatorReputerAndTopicId(ctx, delegator, reputer, topicId)
	require.NoError(err)
	require.False(found, "Stake removal should be deleted")
}

func (s *MsgServerTestSuite) TestCancelRemoveDelegateStakeNotExist() {
	ctx := s.Ctx()
	require := s.Require()
	// Set up test data
	delegator := s.AddrsStr(0)
	reputer := s.AddrsStr(1)
	topicId := uint64(1)

	// Call CancelRemoveDelegateStake
	msg := &types.CancelRemoveDelegateStakeRequest{
		Sender:    delegator,
		Reputer:   reputer,
		TopicId:   topicId,
		Delegator: delegator,
	}
	_, err := s.EmissionsMsgServer().CancelRemoveDelegateStake(ctx, msg)
	require.Error(err)
	require.True(errors.Is(err, types.ErrStakeRemovalNotFound), "Expected delegate stake removal not found error")
}
