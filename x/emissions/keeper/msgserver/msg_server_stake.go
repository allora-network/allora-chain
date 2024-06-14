package msgserver

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Function for reputers to call to add stake to an existing stake position.
func (ms msgServer) AddStake(ctx context.Context, msg *types.MsgAddStake) (*types.MsgAddStakeResponse, error) {
	if msg.Amount.IsZero() {
		return nil, types.ErrReceivedZeroAmount
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
	isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isReputerRegistered {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	// Check the sender has enough funds to add the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send the funds
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, msg.Amount))
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, types.AlloraStakingAccountName, coins)
	if err != nil {
		return nil, err
	}

	// Update the stake data structures, spread the stake across all topics evenly
	err = ms.k.AddStake(ctx, msg.TopicId, msg.Sender, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = ms.ActivateTopicIfWeightAtLeastGlobalMin(ctx, msg.TopicId, msg.Amount)
	return &types.MsgAddStakeResponse{}, err
}

// StartRemoveStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then ConfirmRemoveStake can be called to remove the stake.
// if a stake removal is not confirmed within a certain time period, the stake removal becomes invalid
// and one must start the stake removal process again and wait the delay again.
func (ms msgServer) StartRemoveStake(ctx context.Context, msg *types.MsgStartRemoveStake) (*types.MsgStartRemoveStakeResponse, error) {
	if msg.Amount.IsZero() {
		return nil, types.ErrReceivedZeroAmount
	}

	// Check the sender has enough stake already placed on the topic to remove the stake
	stakePlaced, err := ms.k.GetStakeOnReputerInTopic(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	}

	delegateStakeUponReputerInTopic, err := ms.k.GetDelegateStakeUponReputer(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	}
	reputerStakeInTopicWithoutDelegateStake := stakePlaced.Sub(delegateStakeUponReputerInTopic)
	if msg.Amount.GT(reputerStakeInTopicWithoutDelegateStake) {
		return nil, types.ErrInsufficientStakeToRemove
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stakeToRemove := types.StakePlacement{
		BlockRemovalStarted: sdkCtx.BlockHeight(),
		TopicId:             msg.TopicId,
		Reputer:             msg.Sender,
		Amount:              msg.Amount,
	}

	// If no errors have occurred and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetStakeRemoval(ctx, msg.Sender, stakeToRemove)
	if err != nil {
		return nil, err
	}
	return &types.MsgStartRemoveStakeResponse{}, nil
}

// Function for reputers or workers to call to remove stake from an existing stake position.
func (ms msgServer) ConfirmRemoveStake(ctx context.Context, msg *types.MsgConfirmRemoveStake) (*types.MsgConfirmRemoveStakeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Pull the stake removal from the delayed queue
	stakeRemoval, err := ms.k.GetStakeRemovalByTopicAndAddress(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the timestamp is valid
	currentBlock := sdkCtx.BlockHeight()
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	removalWindowOpens := stakeRemoval.BlockRemovalStarted + moduleParams.RemoveStakeDelayWindow
	// if the delay window has not yet passed, its too early
	if currentBlock < removalWindowOpens {
		return nil, types.ErrConfirmRemoveStakeTooEarly
	}
	// if the delay window has passed and then so has the active window, it's too late
	if currentBlock > removalWindowOpens+moduleParams.RemoveStakeActiveWindow {
		return nil, types.ErrConfirmRemoveStakeTooLate
	}
	// Check the module has enough funds to send back to the sender
	// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send the funds
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, stakeRemoval.Amount))
	err = ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, msg.Sender, coins)
	if err != nil {
		return nil, err
	}

	// Update the stake data structures
	err = ms.k.RemoveStake(
		ctx,
		stakeRemoval.TopicId,
		msg.Sender,
		stakeRemoval.Amount,
	)
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

	if msg.Reputer == msg.Sender {
		return nil, types.ErrCantSelfDelegate
	}

	// Check the target reputer exists and is registered
	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	}
	if !isRegistered {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	// Check the sender has enough funds to delegate the stake
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check here
	// Send the funds
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, msg.Amount))
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, types.AlloraStakingAccountName, coins)
	if err != nil {
		return nil, err
	}

	// Update the stake data structures
	err = ms.k.AddStake(ctx, msg.TopicId, msg.Reputer, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = ms.k.AddDelegateStake(ctx, msg.TopicId, msg.Sender, msg.Reputer, msg.Amount)
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

	// Check the reputer has enough stake already placed on the topic to remove the stake
	stakePlaced, err := ms.k.GetStakeOnReputerInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	}
	if stakePlaced.LT(msg.Amount) {
		return nil, types.ErrInsufficientStakeToRemove
	}

	// Check the reputer has enough stake already placed on the topic to remove the stake
	delegateStakePlaced, err := ms.k.GetDelegateStakePlacement(ctx, msg.TopicId, msg.Sender, msg.Reputer)
	if err != nil {
		return nil, err
	}
	amountDec, err := alloraMath.NewDecFromSdkInt(msg.Amount)
	if err != nil {
		return nil, err
	}
	if delegateStakePlaced.Amount.Lt(amountDec) {
		return nil, types.ErrInsufficientDelegateStakeToRemove
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stakeToRemove := types.DelegateStakePlacement{
		BlockRemovalStarted: sdkCtx.BlockHeight(),
		TopicId:             msg.TopicId,
		Reputer:             msg.Reputer,
		Delegator:           msg.Sender,
		Amount:              msg.Amount,
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
	stakeRemoval, err := ms.k.GetDelegateStakeRemovalByTopicAndAddress(ctx, msg.TopicId, msg.Reputer, msg.Sender)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrConfirmRemoveStakeNoRemovalStarted
		}
		return nil, err
	}
	// check the block it should start is valid
	currentBlock := sdkCtx.BlockHeight()
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	removalWindowOpens := stakeRemoval.BlockRemovalStarted + moduleParams.RemoveStakeDelayWindow
	// if the delay window has not yet passed, its too early
	if currentBlock < removalWindowOpens {
		return nil, types.ErrConfirmRemoveStakeTooEarly
	}
	// if the delay window has passed and then so has the active window, it's too late
	if currentBlock > removalWindowOpens+moduleParams.RemoveStakeActiveWindow {
		return nil, types.ErrConfirmRemoveStakeTooLate
	}

	// Check the module has enough funds to send back to the sender
	// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send the funds
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, stakeRemoval.Amount))
	err = ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, msg.Sender, coins)
	if err != nil {
		return nil, err
	}

	// Update the stake data structures
	err = ms.k.RemoveDelegateStake(ctx, stakeRemoval.TopicId, msg.Sender, msg.Reputer, stakeRemoval.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgConfirmRemoveDelegateStakeResponse{}, nil
}

func (ms msgServer) RewardDelegateStake(ctx context.Context, msg *types.MsgRewardDelegateStake) (*types.MsgRewardDelegateStakeResponse, error) {
	// Check the target reputer exists and is registered
	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	}
	if !isRegistered {
		return nil, types.ErrAddressIsNotRegisteredInThisTopic
	}

	delegateInfo, err := ms.k.GetDelegateStakePlacement(ctx, msg.TopicId, msg.Sender, msg.Reputer)
	if err != nil {
		return nil, err
	}
	share, err := ms.k.GetDelegateRewardPerShare(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	}
	pendingReward, err := delegateInfo.Amount.Mul(share)
	if err != nil {
		return nil, err
	}
	pendingReward, err = pendingReward.Sub(delegateInfo.RewardDebt)
	if err != nil {
		return nil, err
	}
	if pendingReward.Gt(alloraMath.NewDecFromInt64(0)) {
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, pendingReward.SdkIntTrim()))
		err = ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName, msg.Sender, coins)
		if err != nil {
			return nil, err
		}
		delegateInfo.RewardDebt, err = delegateInfo.Amount.Mul(share)
		if err != nil {
			return nil, err
		}
		ms.k.SetDelegateStakePlacement(ctx, msg.TopicId, msg.Sender, msg.Reputer, delegateInfo)
	}
	return &types.MsgRewardDelegateStakeResponse{}, nil
}
