package queryserver

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// TotalStake defines the handler for the Query/TotalStake RPC method.
func (qs queryServer) GetTotalStake(ctx context.Context, req *types.QueryTotalStakeRequest) (*types.QueryTotalStakeResponse, error) {
	totalStake, err := qs.k.GetTotalStake(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryTotalStakeResponse{Amount: totalStake}, nil
}

// Retrieves all stake in a topic for a given reputer address,
// including reputer's stake in themselves and stake delegated to them.
// Also includes stake that is queued for removal.
func (qs queryServer) GetReputerStakeInTopic(ctx context.Context, req *types.QueryReputerStakeInTopicRequest) (*types.QueryReputerStakeInTopicResponse, error) {
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

	return &types.QueryReputerStakeInTopicResponse{Amount: stake}, nil
}

// Retrieves all stake in a topic for a given set of reputer addresses,
// including their stake in themselves and stake delegated to them.
// Also includes stake that is queued for removal.
func (qs queryServer) GetMultiReputerStakeInTopic(ctx context.Context, req *types.QueryMultiReputerStakeInTopicRequest) (*types.QueryMultiReputerStakeInTopicResponse, error) {
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

	return &types.QueryMultiReputerStakeInTopicResponse{Amounts: stakes}, nil
}

// Retrieves the stake that a reputer has in themselves in a given topic
// this is computed from the differences in the delegated stake data structure
// and the total stake data structure. Which means if invariants are ever violated
// in the data structures for staking, this function will return an incorrect value.
func (qs queryServer) GetStakeFromReputerInTopicInSelf(ctx context.Context, req *types.QueryStakeFromReputerInTopicInSelfRequest) (*types.QueryStakeFromReputerInTopicInSelfResponse, error) {
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
	return &types.QueryStakeFromReputerInTopicInSelfResponse{Amount: stakeFromReputerInTopicInSelf}, nil
}

// Retrieves total delegate stake on a given reputer address in a given topic
func (qs queryServer) GetDelegateStakeInTopicInReputer(ctx context.Context, req *types.QueryDelegateStakeInTopicInReputerRequest) (*types.QueryDelegateStakeInTopicInReputerResponse, error) {
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

	return &types.QueryDelegateStakeInTopicInReputerResponse{Amount: stake}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopicInReputer(ctx context.Context, req *types.QueryStakeFromDelegatorInTopicInReputerRequest) (*types.QueryStakeFromDelegatorInTopicInReputerResponse, error) {
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

	return &types.QueryStakeFromDelegatorInTopicInReputerResponse{Amount: stake.Amount.SdkIntTrim()}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopic(ctx context.Context, req *types.QueryStakeFromDelegatorInTopicRequest) (*types.QueryStakeFromDelegatorInTopicResponse, error) {
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

	return &types.QueryStakeFromDelegatorInTopicResponse{Amount: stake}, nil
}

// Retrieves total stake in a given topic
func (qs queryServer) GetTopicStake(ctx context.Context, req *types.QueryTopicStakeRequest) (*types.QueryTopicStakeResponse, error) {
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

	return &types.QueryTopicStakeResponse{Amount: stake}, nil
}

func (qs queryServer) GetStakeRemovalsForBlock(
	ctx context.Context,
	req *types.QueryStakeRemovalsForBlockRequest,
) (*types.QueryStakeRemovalsForBlockResponse, error) {
	removals, err := qs.k.GetStakeRemovalsForBlock(ctx, req.BlockHeight)
	if err != nil {
		return nil, err
	}
	moduleParams, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxLimit := moduleParams.MaxPageLimit
	removalPointers := make([]*types.StakeRemovalInfo, 0)
	outputLen := uint64(len(removals))
	if uint64(len(removals)) > maxLimit {
		err = status.Error(codes.InvalidArgument, fmt.Sprintf("cannot query more than %d removals at once", maxLimit))
		outputLen = maxLimit
	}
	for i := uint64(0); i < outputLen; i++ {
		removalPointers = append(removalPointers, &removals[i])
	}
	return &types.QueryStakeRemovalsForBlockResponse{Removals: removalPointers}, err
}

func (qs queryServer) GetDelegateStakeRemovalsForBlock(
	ctx context.Context,
	req *types.QueryDelegateStakeRemovalsForBlockRequest,
) (*types.QueryDelegateStakeRemovalsForBlockResponse, error) {
	removals, err := qs.k.GetDelegateStakeRemovalsForBlock(ctx, req.BlockHeight)
	if err != nil {
		return nil, err
	}
	moduleParams, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxLimit := moduleParams.MaxPageLimit
	removalPointers := make([]*types.DelegateStakeRemovalInfo, 0)
	outputLen := uint64(len(removals))
	if uint64(len(removals)) > maxLimit {
		err = status.Error(codes.InvalidArgument, fmt.Sprintf("cannot query more than %d removals at once", maxLimit))
		outputLen = maxLimit
	}
	for i := uint64(0); i < outputLen; i++ {
		removalPointers = append(removalPointers, &removals[i])
	}
	return &types.QueryDelegateStakeRemovalsForBlockResponse{Removals: removalPointers}, err
}

func (qs queryServer) GetStakeRemovalInfo(
	ctx context.Context,
	req *types.QueryStakeRemovalInfoRequest,
) (*types.QueryStakeRemovalInfoResponse, error) {
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
	return &types.QueryStakeRemovalInfoResponse{Removal: &removal}, err
}

func (qs queryServer) GetDelegateStakeRemovalInfo(
	ctx context.Context,
	req *types.QueryDelegateStakeRemovalInfoRequest,
) (*types.QueryDelegateStakeRemovalInfoResponse, error) {
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
	return &types.QueryDelegateStakeRemovalInfoResponse{Removal: &removal}, err
}

func (qs queryServer) GetStakeReputerAuthority(
	ctx context.Context,
	req *types.QueryStakeReputerAuthorityRequest,
) (
	*types.QueryStakeReputerAuthorityResponse,
	error,
) {
	stakeReputerAuthority, err := qs.k.GetStakeReputerAuthority(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryStakeReputerAuthorityResponse{Authority: stakeReputerAuthority}, nil
}

func (qs queryServer) GetDelegateStakePlacement(
	ctx context.Context,
	req *types.QueryDelegateStakePlacementRequest,
) (
	*types.QueryDelegateStakePlacementResponse,
	error,
) {
	delegateStakePlacement, err := qs.k.GetDelegateStakePlacement(ctx, req.TopicId, req.Delegator, req.Target)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegateStakePlacementResponse{DelegatorInfo: &delegateStakePlacement}, nil
}

func (qs queryServer) GetDelegateStakeUponReputer(
	ctx context.Context,
	req *types.QueryDelegateStakeUponReputerRequest,
) (
	*types.QueryDelegateStakeUponReputerResponse,
	error,
) {
	delegateStakeUponReputer, err := qs.k.GetDelegateStakeUponReputer(ctx, req.TopicId, req.Target)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegateStakeUponReputerResponse{Stake: delegateStakeUponReputer}, nil
}

func (qs queryServer) GetDelegateRewardPerShare(
	ctx context.Context,
	req *types.QueryDelegateRewardPerShareRequest,
) (
	*types.QueryDelegateRewardPerShareResponse,
	error,
) {
	delegateRewardPerShare, err := qs.k.GetDelegateRewardPerShare(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegateRewardPerShareResponse{RewardPerShare: delegateRewardPerShare}, nil
}

func (qs queryServer) GetStakeRemovalForReputerAndTopicId(
	ctx context.Context,
	req *types.QueryStakeRemovalForReputerAndTopicIdRequest,
) (
	*types.QueryStakeRemovalForReputerAndTopicIdResponse,
	error,
) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	stakeRemovalInfo, found, err := qs.k.GetStakeRemovalForReputerAndTopicId(sdkCtx, req.Reputer, req.TopicId)
	if err != nil {
		return nil, err
	}
	if !found {
		return &types.QueryStakeRemovalForReputerAndTopicIdResponse{}, nil
	}

	return &types.QueryStakeRemovalForReputerAndTopicIdResponse{StakeRemovalInfo: &stakeRemovalInfo}, nil
}
