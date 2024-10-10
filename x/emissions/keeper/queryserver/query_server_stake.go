package queryserver

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// TotalStake defines the handler for the Get/TotalStake RPC method.
func (qs queryServer) GetTotalStake(ctx context.Context, req *types.GetTotalStakeRequest) (_ *types.GetTotalStakeResponse, err error) {
	defer metrics.RecordMetrics("GetTotalStake", time.Now(), &err == nil)
	totalStake, err := qs.k.GetTotalStake(ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetTotalStakeResponse{Amount: totalStake}, nil
}

// Retrieves all stake in a topic for a given reputer address,
// including reputer's stake in themselves and stake delegated to them.
// Also includes stake that is queued for removal.
func (qs queryServer) GetReputerStakeInTopic(ctx context.Context, req *types.GetReputerStakeInTopicRequest) (_ *types.GetReputerStakeInTopicResponse, err error) {
	defer metrics.RecordMetrics("GetReputerStakeInTopic", time.Now(), &err == nil)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetStakeReputerAuthority(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.GetReputerStakeInTopicResponse{Amount: stake}, nil
}

// Retrieves all stake in a topic for a given set of reputer addresses,
// including their stake in themselves and stake delegated to them.
// Also includes stake that is queued for removal.
func (qs queryServer) GetMultiReputerStakeInTopic(ctx context.Context, req *types.GetMultiReputerStakeInTopicRequest) (_ *types.GetMultiReputerStakeInTopicResponse, err error) {
	defer metrics.RecordMetrics("GetMultiReputerStakeInTopic", time.Now(), &err == nil)
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	maxLimit := types.DefaultParams().MaxPageLimit
	moduleParams, err := qs.k.GetParams(ctx)
	if err == nil {
		maxLimit = moduleParams.MaxPageLimit
	}

	if uint64(len(req.Addresses)) > maxLimit {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("cannot query more than %d addresses at once", maxLimit))
	}

	stakes := make([]*types.StakeInfo, len(req.Addresses))
	for i, address := range req.Addresses {
		stake := cosmosMath.ZeroInt()
		if err := qs.k.ValidateStringIsBech32(address); err == nil {
			stake, err = qs.k.GetStakeReputerAuthority(ctx, req.TopicId, address)
			if err != nil {
				stake = cosmosMath.ZeroInt()
			}
		}
		stakes[i] = &types.StakeInfo{TopicId: req.TopicId, Reputer: address, Amount: stake}
	}
	return &types.GetMultiReputerStakeInTopicResponse{Amounts: stakes}, nil
}

// Retrieves the stake that a reputer has in themselves in a given topic
// this is computed from the differences in the delegated stake data structure
// and the total stake data structure. Which means if invariants are ever violated
// in the data structures for staking, this function will return an incorrect value.
func (qs queryServer) GetStakeFromReputerInTopicInSelf(ctx context.Context, req *types.GetStakeFromReputerInTopicInSelfRequest) (_ *types.GetStakeFromReputerInTopicInSelfResponse, err error) {
	defer metrics.RecordMetrics("GetStakeFromReputerInTopicInSelf", time.Now(), &err == nil)
	if err := qs.k.ValidateStringIsBech32(req.ReputerAddress); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	stakeOnReputerInTopic, err := qs.k.GetStakeReputerAuthority(ctx, req.TopicId, req.ReputerAddress)
	if err != nil {
		return nil, err
	}
	delegateStakeOnReputerInTopic, err := qs.k.GetDelegateStakeUponReputer(ctx, req.TopicId, req.ReputerAddress)
	if err != nil {
		return nil, err
	}
	stakeFromReputerInTopicInSelf := stakeOnReputerInTopic.Sub(delegateStakeOnReputerInTopic)
	if stakeFromReputerInTopicInSelf.IsNegative() {
		return nil, errors.Wrap(types.ErrInvariantFailure, "stake from reputer in topic in self is negative")
	}
	return &types.GetStakeFromReputerInTopicInSelfResponse{Amount: stakeFromReputerInTopicInSelf}, nil
}

// Retrieves total delegate stake on a given reputer address in a given topic
func (qs queryServer) GetDelegateStakeInTopicInReputer(ctx context.Context, req *types.GetDelegateStakeInTopicInReputerRequest) (_ *types.GetDelegateStakeInTopicInReputerResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateStakeInTopicInReputer", time.Now(), &err == nil)
	if err := qs.k.ValidateStringIsBech32(req.ReputerAddress); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetDelegateStakeUponReputer(ctx, req.TopicId, req.ReputerAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetDelegateStakeInTopicInReputerResponse{Amount: stake}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopicInReputer(ctx context.Context, req *types.GetStakeFromDelegatorInTopicInReputerRequest) (_ *types.GetStakeFromDelegatorInTopicInReputerResponse, err error) {
	defer metrics.RecordMetrics("GetStakeFromDelegatorInTopicInReputer", time.Now(), &err == nil)
	if err := qs.k.ValidateStringIsBech32(req.ReputerAddress); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid reputer address: %s", err)
	}
	if err := qs.k.ValidateStringIsBech32(req.DelegatorAddress); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid delegator address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetDelegateStakePlacement(ctx, req.TopicId, req.DelegatorAddress, req.ReputerAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	stakeInt, err := stake.Amount.SdkIntTrim()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetStakeFromDelegatorInTopicInReputerResponse{Amount: stakeInt}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopic(ctx context.Context, req *types.GetStakeFromDelegatorInTopicRequest) (_ *types.GetStakeFromDelegatorInTopicResponse, err error) {
	defer metrics.RecordMetrics("GetStakeFromDelegatorInTopic", time.Now(), &err == nil)
	if err := qs.k.ValidateStringIsBech32(req.DelegatorAddress); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetStakeFromDelegatorInTopic(ctx, req.TopicId, req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetStakeFromDelegatorInTopicResponse{Amount: stake}, nil
}

// Retrieves total stake in a given topic
func (qs queryServer) GetTopicStake(ctx context.Context, req *types.GetTopicStakeRequest) (_ *types.GetTopicStakeResponse, err error) {
	defer metrics.RecordMetrics("GetTopicStake", time.Now(), &err == nil)
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetTopicStake(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicStakeResponse{Amount: stake}, nil
}

func (qs queryServer) GetStakeRemovalsUpUntilBlock(ctx context.Context, req *types.GetStakeRemovalsUpUntilBlockRequest) (_ *types.GetStakeRemovalsUpUntilBlockResponse, err error) {
	defer metrics.RecordMetrics("GetStakeRemovalsUpUntilBlock", time.Now(), &err == nil)
	moduleParams, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxLimit := moduleParams.MaxPageLimit
	removals, left, err := qs.k.GetStakeRemovalsUpUntilBlock(ctx, req.BlockHeight, maxLimit)
	if err != nil {
		return nil, err
	}
	if left {
		err = status.Error(codes.InvalidArgument,
			fmt.Sprintf("more stake removals available, cannot query more than %d removals at once", maxLimit))
	}
	removalPointers := make([]*types.StakeRemovalInfo, 0)
	for i := 0; i < len(removals); i++ {
		removalPointers = append(removalPointers, &removals[i])
	}
	return &types.GetStakeRemovalsUpUntilBlockResponse{Removals: removalPointers}, nil
}

func (qs queryServer) GetDelegateStakeRemovalsUpUntilBlock(ctx context.Context, req *types.GetDelegateStakeRemovalsUpUntilBlockRequest) (_ *types.GetDelegateStakeRemovalsUpUntilBlockResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateStakeRemovalsUpUntilBlock", time.Now(), &err == nil)
	moduleParams, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxLimit := moduleParams.MaxPageLimit
	removals, limitHit, err := qs.k.GetDelegateStakeRemovalsUpUntilBlock(ctx, req.BlockHeight, maxLimit)
	if err != nil {
		return nil, err
	}
	if limitHit {
		err = status.Error(codes.InvalidArgument,
			fmt.Sprintf("more delegate stake removals available, cannot query more than %d removals at once", maxLimit))
	}
	removalPointers := make([]*types.DelegateStakeRemovalInfo, 0)
	for i := 0; i < len(removals); i++ {
		removalPointers = append(removalPointers, &removals[i])
	}
	return &types.GetDelegateStakeRemovalsUpUntilBlockResponse{Removals: removalPointers}, nil
}

func (qs queryServer) GetStakeRemovalInfo(ctx context.Context, req *types.GetStakeRemovalInfoRequest) (_ *types.GetStakeRemovalInfoResponse, err error) {
	defer metrics.RecordMetrics("GetStakeRemovalInfo", time.Now(), &err == nil)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if err := qs.k.ValidateStringIsBech32(req.Reputer); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	removal, found, err := qs.k.GetStakeRemovalForReputerAndTopicId(sdkCtx, req.Reputer, req.TopicId)
	if err != nil && !found {
		return nil, err
	}
	if !found {
		return nil, status.Error(codes.NotFound, "no stake removal found")
	}
	return &types.GetStakeRemovalInfoResponse{Removal: &removal}, nil
}

func (qs queryServer) GetDelegateStakeRemovalInfo(ctx context.Context, req *types.GetDelegateStakeRemovalInfoRequest) (_ *types.GetDelegateStakeRemovalInfoResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateStakeRemovalInfo", time.Now(), &err == nil)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if err := qs.k.ValidateStringIsBech32(req.Reputer); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid reputer address: %s", err)
	}
	if err := qs.k.ValidateStringIsBech32(req.Delegator); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid delegator address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	removal, found, err := qs.k.
		GetDelegateStakeRemovalForDelegatorReputerAndTopicId(sdkCtx, req.Delegator, req.Reputer, req.TopicId)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Error(codes.NotFound, "no delegate stake removal found")
	}
	return &types.GetDelegateStakeRemovalInfoResponse{Removal: &removal}, nil
}

func (qs queryServer) GetStakeReputerAuthority(ctx context.Context, req *types.GetStakeReputerAuthorityRequest) (_ *types.GetStakeReputerAuthorityResponse, err error) {
	defer metrics.RecordMetrics("GetStakeReputerAuthority", time.Now(), &err == nil)
	stakeReputerAuthority, err := qs.k.GetStakeReputerAuthority(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetStakeReputerAuthorityResponse{Authority: stakeReputerAuthority}, nil
}

func (qs queryServer) GetDelegateStakePlacement(ctx context.Context, req *types.GetDelegateStakePlacementRequest) (resp *types.GetDelegateStakePlacementResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateStakePlacement", time.Now(), &err == nil)
	delegateStakePlacement, err := qs.k.GetDelegateStakePlacement(ctx, req.TopicId, req.Delegator, req.Target)
	if err != nil {
		return nil, err
	}

	resp = &types.GetDelegateStakePlacementResponse{DelegatorInfo: &delegateStakePlacement}
	return resp, nil
}

func (qs queryServer) GetDelegateStakeUponReputer(ctx context.Context, req *types.GetDelegateStakeUponReputerRequest) (_ *types.GetDelegateStakeUponReputerResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateStakeUponReputer", time.Now(), &err == nil)
	delegateStakeUponReputer, err := qs.k.GetDelegateStakeUponReputer(ctx, req.TopicId, req.Target)
	if err != nil {
		return nil, err
	}

	return &types.GetDelegateStakeUponReputerResponse{Stake: delegateStakeUponReputer}, nil
}

func (qs queryServer) GetDelegateRewardPerShare(ctx context.Context, req *types.GetDelegateRewardPerShareRequest) (_ *types.GetDelegateRewardPerShareResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateRewardPerShare", time.Now(), &err == nil)
	delegateRewardPerShare, err := qs.k.GetDelegateRewardPerShare(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetDelegateRewardPerShareResponse{RewardPerShare: delegateRewardPerShare}, nil
}

func (qs queryServer) GetStakeRemovalForReputerAndTopicId(ctx context.Context, req *types.GetStakeRemovalForReputerAndTopicIdRequest) (_ *types.GetStakeRemovalForReputerAndTopicIdResponse, err error) {
	defer metrics.RecordMetrics("GetStakeRemovalForReputerAndTopicId", time.Now(), &err == nil)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stakeRemovalInfo, found, err := qs.k.GetStakeRemovalForReputerAndTopicId(sdkCtx, req.Reputer, req.TopicId)
	if err != nil {
		return nil, err
	}
	if !found {
		return &types.GetStakeRemovalForReputerAndTopicIdResponse{StakeRemovalInfo: nil}, nil
	}

	return &types.GetStakeRemovalForReputerAndTopicIdResponse{StakeRemovalInfo: &stakeRemovalInfo}, nil
}

func (qs queryServer) GetDelegateStakeRemoval(ctx context.Context, req *types.GetDelegateStakeRemovalRequest) (_ *types.GetDelegateStakeRemovalResponse, err error) {
	defer metrics.RecordMetrics("GetDelegateStakeRemoval", time.Now(), &err == nil)
	delegateStakeRemoval, err := qs.k.GetDelegateStakeRemoval(ctx, req.BlockHeight, req.TopicId, req.Delegator, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetDelegateStakeRemovalResponse{StakeRemovalInfo: &delegateStakeRemoval}, nil
}
