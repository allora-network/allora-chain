package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetPreviousReputerRewardFraction(
	ctx context.Context,
	req *types.QueryPreviousReputerRewardFractionRequest,
) (
	*types.QueryPreviousReputerRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousReputerRewardFraction(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryPreviousReputerRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}

func (qs queryServer) GetPreviousInferenceRewardFraction(
	ctx context.Context,
	req *types.QueryPreviousInferenceRewardFractionRequest,
) (
	*types.QueryPreviousInferenceRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousInferenceRewardFraction(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &types.QueryPreviousInferenceRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}

func (qs queryServer) GetPreviousForecastRewardFraction(
	ctx context.Context,
	req *types.QueryPreviousForecastRewardFractionRequest,
) (
	*types.QueryPreviousForecastRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousForecastRewardFraction(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &types.QueryPreviousForecastRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}

func (qs queryServer) GetPreviousPercentageRewardToStakedReputers(
	ctx context.Context,
	req *types.QueryPreviousPercentageRewardToStakedReputersRequest,
) (
	*types.QueryPreviousPercentageRewardToStakedReputersResponse,
	error,
) {
	percentageReward, err := qs.k.GetPreviousPercentageRewardToStakedReputers(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryPreviousPercentageRewardToStakedReputersResponse{PercentageReward: percentageReward}, nil
}

func (qs queryServer) GetTotalRewardToDistribute(
	ctx context.Context,
	req *types.QueryTotalRewardToDistributeRequest,
) (
	*types.QueryTotalRewardToDistributeResponse,
	error,
) {
	totalReward, err := qs.k.GetTotalRewardToDistribute(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryTotalRewardToDistributeResponse{TotalReward: totalReward}, nil
}
