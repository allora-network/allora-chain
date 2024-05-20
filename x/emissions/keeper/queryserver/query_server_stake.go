package queryserver

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
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
		return nil, err
	}
	stake, err := qs.k.GetStakeOnReputerInTopic(ctx, req.TopicId, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryReputerStakeInTopicResponse{Amount: stake}, nil
}

// Retrieves total delegate stake on a given reputer address in a given topic
func (qs queryServer) GetDelegateStakeInTopicInReputer(ctx context.Context, req *types.QueryDelegateStakeInTopicInReputerRequest) (*types.QueryDelegateStakeInTopicInReputerResponse, error) {
	if err := qs.k.ValidateStringIsBech32(req.ReputerAddress); err != nil {
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
		return nil, errors.Wrapf(err, "reputer address")
	}
	if err := qs.k.ValidateStringIsBech32(req.DelegatorAddress); err != nil {
		return nil, errors.Wrapf(err, "delegator address")
	}
	stake, err := qs.k.GetDelegateStakePlacement(ctx, req.TopicId, req.DelegatorAddress, req.ReputerAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryStakeFromDelegatorInTopicInReputerResponse{Amount: cosmosMath.NewUintFromString(stake.Amount.String())}, nil
}

func (qs queryServer) GetStakeFromDelegatorInTopic(ctx context.Context, req *types.QueryStakeFromDelegatorInTopicRequest) (*types.QueryStakeFromDelegatorInTopicResponse, error) {
	if err := qs.k.ValidateStringIsBech32(req.DelegatorAddress); err != nil {
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
	stake, err := qs.k.GetTopicStake(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicStakeResponse{Amount: stake}, nil
}
