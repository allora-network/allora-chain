package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Get timestamp of the last rewards update
func (qs queryServer) GetLastRewardsUpdate(ctx context.Context, req *types.QueryLastRewardsUpdateRequest) (*types.QueryLastRewardsUpdateResponse, error) {
	lastRewardsUpdate, err := qs.k.GetLastRewardsUpdate(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryLastRewardsUpdateResponse{LastRewardsUpdate: lastRewardsUpdate}, nil
}

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
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetStakeOnTopicFromReputer(ctx, req.TopicId, address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryReputerStakeInTopicResponse{Amount: stake}, nil
}

// Retrieves total delegate stake on a given reputer address in a given topic
func (qs queryServer) GetDelegateStakeInTopicInReputer(ctx context.Context, req *types.QueryDelegateStakeInTopicInReputerRequest) (*types.QueryDelegateStakeInTopicInReputerResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	reputerAddress, err := sdk.AccAddressFromBech32(req.ReputerAddress)
	if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetDelegateStakeUponReputer(ctx, req.TopicId, reputerAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDelegateStakeInTopicInReputerResponse{Amount: stake}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopicInReputer(ctx context.Context, req *types.QueryStakeFromDelegatorInTopicInReputerRequest) (*types.QueryStakeFromDelegatorInTopicInReputerResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	reputerAddress, err := sdk.AccAddressFromBech32(req.ReputerAddress)
	if err != nil {
		return nil, err
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetDelegateStakePlacement(ctx, req.TopicId, delegatorAddress, reputerAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryStakeFromDelegatorInTopicInReputerResponse{Amount: stake}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopic(ctx context.Context, req *types.QueryStakeFromDelegatorInTopicRequest) (*types.QueryStakeFromDelegatorInTopicResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	stake, err := qs.k.GetStakeFromDelegatorInTopic(ctx, req.TopicId, delegatorAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryStakeFromDelegatorInTopicResponse{Amount: stake}, nil
}

// Retrieves total stake in a given topic
func (qs queryServer) GetTopicStake(ctx context.Context, req *types.QueryTopicStakeRequest) (*types.QueryTopicStakeResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	stake, err := qs.k.GetTopicStake(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicStakeResponse{Amount: stake}, nil
}
