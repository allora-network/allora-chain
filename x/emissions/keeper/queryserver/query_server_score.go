package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetLatestInfererScore(
	ctx context.Context,
	req *types.QueryLatestInfererScoreRequest,
) (
	*types.QueryLatestInfererScoreResponse,
	error,
) {
	latestInfererScore, err := qs.k.GetLatestInfererScore(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestInfererScoreResponse{Score: &latestInfererScore}, nil
}

func (qs queryServer) GetLatestForecasterScore(
	ctx context.Context,
	req *types.QueryLatestForecasterScoreRequest,
) (
	*types.QueryLatestForecasterScoreResponse,
	error,
) {
	latestForecasterScore, err := qs.k.GetLatestForecasterScore(ctx, req.TopicId, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestForecasterScoreResponse{Score: &latestForecasterScore}, nil
}

func (qs queryServer) GetLatestReputerScore(
	ctx context.Context,
	req *types.QueryLatestReputerScoreRequest,
) (
	*types.QueryLatestReputerScoreResponse,
	error,
) {
	latestReputerScore, err := qs.k.GetLatestReputerScore(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestReputerScoreResponse{Score: &latestReputerScore}, nil
}

func (qs queryServer) GetInferenceScoresUntilBlock(
	ctx context.Context,
	req *types.QueryInferenceScoresUntilBlockRequest,
) (
	*types.QueryInferenceScoresUntilBlockResponse,
	error,
) {
	inferenceScores, err := qs.k.GetInferenceScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryInferenceScoresUntilBlockResponse{Scores: inferenceScores}, nil
}

func (qs queryServer) GetWorkerInferenceScoresAtBlock(
	ctx context.Context,
	req *types.QueryWorkerInferenceScoresAtBlockRequest,
) (
	*types.QueryWorkerInferenceScoresAtBlockResponse,
	error,
) {
	workerInferenceScores, err := qs.k.GetWorkerInferenceScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerInferenceScoresAtBlockResponse{Scores: &workerInferenceScores}, nil
}

func (qs queryServer) GetForecastScoresUntilBlock(
	ctx context.Context,
	req *types.QueryForecastScoresUntilBlockRequest,
) (
	*types.QueryForecastScoresUntilBlockResponse,
	error,
) {
	forecastScores, err := qs.k.GetForecastScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryForecastScoresUntilBlockResponse{Scores: forecastScores}, nil
}
