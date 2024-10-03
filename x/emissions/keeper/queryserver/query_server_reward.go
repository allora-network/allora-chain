package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetPreviousReputerRewardFraction(
	ctx context.Context,
	req *types.GetPreviousReputerRewardFractionRequest,
) (
	_ *types.GetPreviousReputerRewardFractionResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousReputerRewardFraction", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetPreviousInferenceRewardFractionResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousInferenceRewardFraction", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetPreviousForecastRewardFractionResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousForecastRewardFraction", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetPreviousPercentageRewardToStakedReputersResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousPercentageRewardToStakedReputers", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetTotalRewardToDistributeResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetTotalRewardToDistribute", "rpc", time.Now(), returnErr == nil)
	totalReward, err := qs.k.GetTotalRewardToDistribute(ctx)
	if err != nil {
		return nil, err
	}

	return &types.GetTotalRewardToDistributeResponse{TotalReward: totalReward}, nil
}
