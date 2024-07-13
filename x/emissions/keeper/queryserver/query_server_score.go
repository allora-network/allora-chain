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

func (qs queryServer) GetWorkerForecastScoresAtBlock(
	ctx context.Context,
	req *types.QueryWorkerForecastScoresAtBlockRequest,
) (
	*types.QueryWorkerForecastScoresAtBlockResponse,
	error,
) {
	workerForecastScores, err := qs.k.GetWorkerForecastScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerForecastScoresAtBlockResponse{Scores: &workerForecastScores}, nil
}

func (qs queryServer) GetReputersScoresAtBlock(
	ctx context.Context,
	req *types.QueryReputersScoresAtBlockRequest,
) (
	*types.QueryReputersScoresAtBlockResponse,
	error,
) {
	reputersScores, err := qs.k.GetReputersScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputersScoresAtBlockResponse{Scores: &reputersScores}, nil
}

func (qs queryServer) GetListeningCoefficient(
	ctx context.Context,
	req *types.QueryListeningCoefficientRequest,
) (
	*types.QueryListeningCoefficientResponse,
	error,
) {
	listeningCoefficient, err := qs.k.GetListeningCoefficient(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryListeningCoefficientResponse{ListeningCoefficient: &listeningCoefficient}, nil
}
