package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetPreviousReputerRewardFraction(
	ctx context.Context,
	req *types.GetPreviousReputerRewardFractionRequest,
) (
	*types.GetPreviousReputerRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousReputerRewardFraction(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousReputerRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}

func (qs queryServer) GetPreviousInferenceRewardFraction(
	ctx context.Context,
	req *types.GetPreviousInferenceRewardFractionRequest,
) (
	*types.GetPreviousInferenceRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousInferenceRewardFraction(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousInferenceRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}

func (qs queryServer) GetPreviousForecastRewardFraction(
	ctx context.Context,
	req *types.GetPreviousForecastRewardFractionRequest,
) (
	*types.GetPreviousForecastRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousForecastRewardFraction(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousForecastRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}

func (qs queryServer) GetPreviousPercentageRewardToStakedReputers(
	ctx context.Context,
	req *types.GetPreviousPercentageRewardToStakedReputersRequest,
) (
	*types.GetPreviousPercentageRewardToStakedReputersResponse,
	error,
) {
	percentageReward, err := qs.k.GetPreviousPercentageRewardToStakedReputers(ctx)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousPercentageRewardToStakedReputersResponse{PercentageReward: percentageReward}, nil
}

func (qs queryServer) GetTotalRewardToDistribute(
	ctx context.Context,
	req *types.GetTotalRewardToDistributeRequest,
) (
	*types.GetTotalRewardToDistributeResponse,
	error,
) {
	totalReward, err := qs.k.GetTotalRewardToDistribute(ctx)
	if err != nil {
		return nil, err
	}

	return &types.GetTotalRewardToDistributeResponse{TotalReward: totalReward}, nil
}
