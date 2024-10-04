package msgserver

import (
	"context"
	"time"

	"errors"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Function for reputers to call to add stake to an existing stake position.
func (ms msgServer) AddStake(ctx context.Context, msg *types.AddStakeRequest,
) (
	_ *types.AddStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("AddStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	// Check the topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check sender is registered in topic
	isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !isReputerRegistered {
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
	err = ms.k.AddReputerStake(ctx, msg.TopicId, msg.Sender, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = activateTopicIfWeightAtLeastGlobalMin(ctx, ms, msg.TopicId)
	return &types.AddStakeResponse{}, err
}

// RemoveStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then the ABCI endBlocker will automatically pay out the stake removal
// if this function is called twice, it will overwrite the previous stake removal and the delay will reset.
func (ms msgServer) RemoveStake(ctx context.Context, msg *types.RemoveStakeRequest,
) (
	_ *types.RemoveStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("RemoveStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check the sender has enough stake already placed on the topic to remove the stake
	stakePlaced, err := ms.k.GetStakeReputerAuthority(ctx, msg.TopicId, msg.Sender)
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

	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// find out if we have a stake removal in progress, if so overwrite it
	removal, found, err := ms.k.GetStakeRemovalForReputerAndTopicId(sdkCtx, msg.Sender, msg.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error while searching previous stake removal")
	}
	if found {
		err = ms.k.DeleteStakeRemoval(ctx, removal.BlockRemovalCompleted, removal.TopicId, removal.Reputer)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to delete previous stake removal")
		}
	}
	stakeToRemove := types.StakeRemovalInfo{
		BlockRemovalStarted:   sdkCtx.BlockHeight(),
		BlockRemovalCompleted: sdkCtx.BlockHeight() + moduleParams.RemoveStakeDelayWindow,
		TopicId:               msg.TopicId,
		Reputer:               msg.Sender,
		Amount:                msg.Amount,
	}

	// If no errors have occurred and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetStakeRemoval(ctx, stakeToRemove)
	if err != nil {
		return nil, err
	}
	return &types.RemoveStakeResponse{}, nil
}

// cancel a request to remove your stake, during the delay window
func (ms msgServer) CancelRemoveStake(ctx context.Context, msg *types.CancelRemoveStakeRequest,
) (
	_ *types.CancelRemoveStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("CancelRemoveStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	removal, found, err := ms.k.GetStakeRemovalForReputerAndTopicId(sdkCtx, msg.Sender, msg.TopicId)
	// if the specific error is that we somehow got into a buggy invariant state
	// where more than one removal request exists in the queue
	// still allow people to cancel withdrawing their stake (fail open rather than closed)
	if err != nil && !errors.Is(err, types.ErrInvariantFailure) {
		return nil, errorsmod.Wrap(err, "error while searching previous stake removal")
	}
	if !found {
		return nil, types.ErrStakeRemovalNotFound
	}
	err = ms.k.DeleteStakeRemoval(ctx, removal.BlockRemovalCompleted, removal.TopicId, removal.Reputer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to delete previous stake removal")
	}
	return &types.CancelRemoveStakeResponse{}, nil
}

// Delegates a stake to a reputer. Sender does not have to be registered to delegate stake.
func (ms msgServer) DelegateStake(ctx context.Context, msg *types.DelegateStakeRequest,
) (
	_ *types.DelegateStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("DelegateStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrap(types.ErrAddressIsNotRegisteredInThisTopic, "reputer address")
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
	err = ms.k.AddDelegateStake(ctx, msg.TopicId, msg.Sender, msg.Reputer, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = activateTopicIfWeightAtLeastGlobalMin(ctx, ms, msg.TopicId)
	return &types.DelegateStakeResponse{}, err
}

// RemoveDelegateStake kicks off a stake removal process. Stake Removals are placed into a delayed queue.
// once the withdrawal delay has passed then the ABCI endBlocker will automatically pay out the stake removal
// if this function is called twice, it will overwrite the previous stake removal and the delay will reset.
func (ms msgServer) RemoveDelegateStake(ctx context.Context, msg *types.RemoveDelegateStakeRequest,
) (
	_ *types.RemoveDelegateStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("RemoveDelegateStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrap(types.ErrAddressIsNotRegisteredInThisTopic, "reputer address")
	}

	// Check the delegator has enough stake already placed on the topic to remove the stake
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

	// Check the reputer has enough stake already placed on the topic to remove the stake
	totalStakeOnReputer, err := ms.k.GetStakeReputerAuthority(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	}
	if totalStakeOnReputer.LT(msg.Amount) {
		return nil, types.ErrInsufficientStakeToRemove
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	// find out if we have a stake removal in progress, if so overwrite it
	removal, found, err := ms.k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(
		sdkCtx, msg.Sender, msg.Reputer, msg.TopicId,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error during finding delegate stake removal")
	}
	if found {
		err = ms.k.DeleteDelegateStakeRemoval(
			ctx,
			removal.BlockRemovalCompleted,
			removal.TopicId,
			removal.Reputer,
			removal.Delegator,
		)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to delete previous delegate stake removal")
		}
	}
	stakeToRemove := types.DelegateStakeRemovalInfo{
		BlockRemovalStarted:   sdkCtx.BlockHeight(),
		BlockRemovalCompleted: sdkCtx.BlockHeight() + moduleParams.RemoveStakeDelayWindow,
		TopicId:               msg.TopicId,
		Reputer:               msg.Reputer,
		Delegator:             msg.Sender,
		Amount:                msg.Amount,
	}

	// If no errors have occurred and the removal is valid, add the stake removal to the delayed queue
	err = ms.k.SetDelegateStakeRemoval(ctx, stakeToRemove)
	if err != nil {
		return nil, err
	}
	return &types.RemoveDelegateStakeResponse{}, nil
}

// cancel an ongoing stake removal request during the delay period
func (ms msgServer) CancelRemoveDelegateStake(ctx context.Context, msg *types.CancelRemoveDelegateStakeRequest,
) (
	_ *types.CancelRemoveDelegateStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("CancelRemoveDelegateStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrap(types.ErrAddressIsNotRegisteredInThisTopic, "reputer address")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	removal, found, err := ms.k.GetDelegateStakeRemovalForDelegatorReputerAndTopicId(
		sdkCtx, msg.Sender, msg.Reputer, msg.TopicId,
	)
	// if the specific error is that we somehow got into a buggy invariant state
	// where more than one removal request exists in the queue
	// still allow people to cancel withdrawing their stake (fail open rather than closed)
	if err != nil && !errors.Is(err, types.ErrInvariantFailure) {
		return nil, errorsmod.Wrap(err, "error while searching previous delegate stake removal")
	}
	if !found {
		return nil, types.ErrStakeRemovalNotFound
	}
	err = ms.k.DeleteDelegateStakeRemoval(
		ctx,
		removal.BlockRemovalCompleted,
		removal.TopicId,
		removal.Reputer,
		removal.Delegator,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to delete previous delegate stake removal")
	}
	return &types.CancelRemoveDelegateStakeResponse{}, nil
}

func (ms msgServer) RewardDelegateStake(ctx context.Context, msg *types.RewardDelegateStakeRequest,
) (
	_ *types.RewardDelegateStakeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("RewardDelegateStake", time.Now(), returnErr == nil)
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !topicExists {
		return nil, types.ErrTopicDoesNotExist
	}

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, msg.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrap(types.ErrAddressIsNotRegisteredInThisTopic, "reputer address")
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
		pendingRewardInt, err := pendingReward.SdkIntTrim()
		if err != nil {
			return nil, err
		}
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, pendingRewardInt))
		err = ms.k.SendCoinsFromModuleToAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName, msg.Sender, coins)
		if err != nil {
			return nil, err
		}
		delegateInfo.RewardDebt, err = delegateInfo.Amount.Mul(share)
		if err != nil {
			return nil, err
		}
		err = ms.k.SetDelegateStakePlacement(ctx, msg.TopicId, msg.Sender, msg.Reputer, delegateInfo)
		if err != nil {
			return nil, err
		}
	}
	return &types.RewardDelegateStakeResponse{}, nil
}
