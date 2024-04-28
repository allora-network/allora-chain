package msgserver

import (
	"context"
	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	"errors"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Function for reputers to call to add stake to an existing stake position.
func (ms msgServer) AddStake(ctx context.Context, msg *types.MsgAddStake) (*types.MsgAddStakeResponse, error) {
	if msg.Amount.IsZero() {
		return nil, types.ErrReceivedZeroAmount
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check the topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check sender is registered in topic
	isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isReputerRegistered {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	// Check the sender has enough funds to add the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraStakingAccountName, coins)

	// Update the stake data structures, spread the stake across all topics evenly
	err = ms.k.AddStake(ctx, msg.TopicId, senderAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddStakeResponse{}, nil
}

// StartRemoveStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveStake(ctx context.Context, msg *types.MsgStartRemoveStake) (*types.MsgStartRemoveStakeResponse, error) {
	if msg.Amount.IsZero() {
		return nil, types.ErrReceivedZeroAmount
	}

	// Check the sender is registered
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check the sender has enough stake already placed on the topic to remove the stake
	stakePlaced, err := ms.k.GetStakeOnTopicFromReputer(ctx, msg.TopicId, sender)
	if err != nil {
		return nil, err
	}
	if stakePlaced.LT(msg.Amount) {
		return nil, types.ErrInsufficientStakeToRemove
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stakeToRemove := types.StakeRemoval{
		BlockRemovalStarted: sdkCtx.BlockHeight(),
		Placement: &types.StakePlacement{
			TopicId: msg.TopicId,
			Reputer: msg.Sender,
			Amount:  msg.Amount,
		},
	}

	// If no errors have occurred and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetStakeRemoval(ctx, sender, stakeToRemove)
	if err != nil {
		return nil, err
	}
	return &types.MsgStartRemoveStakeResponse{}, nil
}

// Function for reputers or workers to call to remove stake from an existing stake position.
func (ms msgServer) ConfirmRemoveStake(ctx context.Context, msg *types.MsgConfirmRemoveStake) (*types.MsgConfirmRemoveStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// Pull the stake removal from the delayed queue
	stakeRemoval, err := ms.k.GetStakeRemovalByTopicAndAddress(ctx, msg.TopicId, sender)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the timestamp is valid
	currentBlock := sdkCtx.BlockHeight()
	delayWindow, err := ms.k.GetParamsRemoveStakeDelayWindow(ctx)
	if err != nil {
		return nil, err
	}
	if stakeRemoval.BlockRemovalStarted+delayWindow >= currentBlock {
		return nil, types.ErrConfirmRemoveStakeTooEarly
	}
	// Check the module has enough funds to send back to the sender
	// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send the funds
	amountInt := cosmosMath.NewIntFromBigInt(stakeRemoval.Placement.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, sender, coins)

	// Update the stake data structures
	err = ms.k.RemoveStake(ctx, stakeRemoval.Placement.TopicId, sender, stakeRemoval.Placement.Amount)
	if err != nil {
		return nil, err
	}
	return &types.MsgConfirmRemoveStakeResponse{}, nil
}

// Delegates a stake to a reputer. Sender need not be registered to delegate stake.
func (ms msgServer) DelegateStake(ctx context.Context, msg *types.MsgDelegateStake) (*types.MsgDelegateStakeResponse, error) {
	if msg.Amount.IsZero() {
		return nil, types.ErrReceivedZeroAmount
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// Check the target reputer exists and is registered
	targetAddr, err := sdk.AccAddressFromBech32(msg.Reputer)
	if err != nil {
		return nil, err
	}
	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, targetAddr)
	if err != nil {
		return nil, err
	}
	if !isRegistered {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	// Check the sender has enough funds to delegate the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check here
	// Send the funds
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraStakingAccountName, coins)

	// Update the stake data structures
	err = ms.k.AddStake(ctx, msg.TopicId, targetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = ms.k.AddDelegateStake(ctx, msg.TopicId, senderAddr, targetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgDelegateStakeResponse{}, nil
}

// StartRemoveDelegateStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveDelegateStake(ctx context.Context, msg *types.MsgStartRemoveDelegateStake) (*types.MsgStartRemoveDelegateStakeResponse, error) {
	if msg.Amount.IsZero() {
		return nil, types.ErrReceivedZeroAmount
	}

	// Check the sender is registered
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	reputerAddr, err := sdk.AccAddressFromBech32(msg.Reputer)
	if err != nil {
		return nil, err
	}

	// Check the reputer has enough stake already placed on the topic to remove the stake
	stakePlaced, err := ms.k.GetStakeOnTopicFromReputer(ctx, msg.TopicId, reputerAddr)
	if err != nil {
		return nil, err
	}
	if stakePlaced.LT(msg.Amount) {
		return nil, types.ErrInsufficientStakeToRemove
	}

	// Check the reputer has enough stake already placed on the topic to remove the stake
	delegateStakePlaced, err := ms.k.GetDelegateStakePlacement(ctx, msg.TopicId, senderAddr, reputerAddr)
	if err != nil {
		return nil, err
	}
	if delegateStakePlaced.Amount.LT(msg.Amount) {
		return nil, types.ErrInsufficientDelegateStakeToRemove
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stakeToRemove := types.DelegateStakeRemoval{
		BlockRemovalStarted: sdkCtx.BlockHeight(),
		Placement: &types.DelegateStakePlacement{
			TopicId:   msg.TopicId,
			Reputer:   msg.Reputer,
			Delegator: msg.Sender,
			Amount:    msg.Amount,
		},
	}

	// If no errors have occurred and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetDelegateStakeRemoval(ctx, stakeToRemove)
	if err != nil {
		return nil, err
	}
	return &types.MsgStartRemoveDelegateStakeResponse{}, nil
}

// Function for delegators to call to remove stake from an existing delegate stake position.
func (ms msgServer) ConfirmRemoveDelegateStake(ctx context.Context, msg *types.MsgConfirmDelegateRemoveStake) (*types.MsgConfirmRemoveDelegateStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Pull the stake removal from the delayed queue
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	reputerAddr, err := sdk.AccAddressFromBech32(msg.Reputer)
	if err != nil {
		return nil, err
	}
	stakeRemoval, err := ms.k.GetDelegateStakeRemovalByTopicAndAddress(ctx, msg.TopicId, reputerAddr, senderAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the block it should start is valid
	currentBlock := sdkCtx.BlockHeight()
	delayWindow, err := ms.k.GetParamsRemoveStakeDelayWindow(ctx)
	if err != nil {
		return nil, err
	}
	if stakeRemoval.BlockRemovalStarted+delayWindow >= currentBlock {
		return nil, types.ErrConfirmRemoveStakeTooEarly
	}

	// Check the module has enough funds to send back to the sender
	// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send the funds
	amountInt := cosmosMath.NewIntFromBigInt(stakeRemoval.Placement.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, senderAddr, coins)

	// Update the stake data structures
	err = ms.k.RemoveDelegateStake(ctx, stakeRemoval.Placement.TopicId, senderAddr, reputerAddr, stakeRemoval.Placement.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgConfirmRemoveDelegateStakeResponse{}, nil
}

func (ms msgServer) RewardDelegateStake(ctx context.Context, msg *types.MsgRewardDelegateStake) (*types.MsgRewardDelegateStakeResponse, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// Check the target reputer exists and is registered
	reputer, err := sdk.AccAddressFromBech32(msg.Reputer)
	if err != nil {
		return nil, err
	}
	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, reputer)
	if err != nil {
		return nil, err
	}
	if !isRegistered {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	delegateInfo, err := ms.k.GetDelegateStakePlacement(ctx, msg.TopicId, senderAddr, reputer)
	if err != nil {
		return nil, err
	}
	share, err := ms.k.GetDelegateRewardPerShare(ctx, msg.TopicId, reputer)
	if err != nil {
		return nil, err
	}
	pendingReward := delegateInfo.Amount.Mul(share).Quo(ms.k.GetAlloraExponent()).Sub(delegateInfo.RewardDebt)
	if pendingReward.GT(cosmosMath.NewUint(0)) {
		amountInt := cosmosMath.NewIntFromBigInt(pendingReward.BigInt())
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
		ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName, senderAddr, coins)
		delegateInfo.RewardDebt = delegateInfo.Amount.Mul(share).Quo(ms.k.GetAlloraExponent())
		ms.k.SetDelegateStakePlacement(ctx, msg.TopicId, senderAddr, reputer, delegateInfo)
	}
	return &types.MsgRewardDelegateStakeResponse{}, nil
}
