package queryserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetInfererScoreEma(
	ctx context.Context,
	req *types.GetInfererScoreEmaRequest,
) (
	*types.GetInfererScoreEmaResponse,
	error,
) {
	latestInfererScore, err := qs.k.GetInfererScoreEma(ctx, req.TopicId, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &types.GetInfererScoreEmaResponse{Score: &latestInfererScore}, nil
}

func (qs queryServer) GetForecasterScoreEma(
	ctx context.Context,
	req *types.GetForecasterScoreEmaRequest,
) (
	*types.GetForecasterScoreEmaResponse,
	error,
) {
	latestForecasterScore, err := qs.k.GetForecasterScoreEma(ctx, req.TopicId, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &types.GetForecasterScoreEmaResponse{Score: &latestForecasterScore}, nil
}

func (qs queryServer) GetReputerScoreEma(
	ctx context.Context,
	req *types.GetReputerScoreEmaRequest,
) (
	*types.GetReputerScoreEmaResponse,
	error,
) {
	latestReputerScore, err := qs.k.GetReputerScoreEma(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetReputerScoreEmaResponse{Score: &latestReputerScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileForecasterScoreEmaRequest) (
	*types.GetPreviousTopicQuantileForecasterScoreEmaResponse,
	error,
) {
	previousQuantileForecasterScore, err := qs.k.GetPreviousTopicQuantileForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.GetPreviousTopicQuantileForecasterScoreEmaResponse{Value: previousQuantileForecasterScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileInfererScoreEmaRequest) (
	*types.GetPreviousTopicQuantileInfererScoreEmaResponse,
	error,
) {
	previousQuantileInfererScore, err := qs.k.GetPreviousTopicQuantileInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.GetPreviousTopicQuantileInfererScoreEmaResponse{Value: previousQuantileInfererScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileReputerScoreEmaRequest) (
	*types.GetPreviousTopicQuantileReputerScoreEmaResponse,
	error,
) {
	previousQuantileReputerScore, err := qs.k.GetPreviousTopicQuantileReputerScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.GetPreviousTopicQuantileReputerScoreEmaResponse{Value: previousQuantileReputerScore}, nil
}

func (qs queryServer) GetInferenceScoresUntilBlock(
	ctx context.Context,
	req *types.GetInferenceScoresUntilBlockRequest,
) (
	*types.GetInferenceScoresUntilBlockResponse,
	error,
) {
	inferenceScores, err := qs.k.GetInferenceScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetInferenceScoresUntilBlockResponse{Scores: inferenceScores}, nil
}

func (qs queryServer) GetWorkerInferenceScoresAtBlock(
	ctx context.Context,
	req *types.GetWorkerInferenceScoresAtBlockRequest,
) (
	*types.GetWorkerInferenceScoresAtBlockResponse,
	error,
) {
	workerInferenceScores, err := qs.k.GetWorkerInferenceScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetWorkerInferenceScoresAtBlockResponse{Scores: &workerInferenceScores}, nil
}

func (qs queryServer) GetCurrentLowestInfererScore(
	ctx context.Context,
	req *types.GetCurrentLowestInfererScoreRequest,
) (
	*types.GetCurrentLowestInfererScoreResponse,
	error,
) {
	lowestInfererScore, found, err := qs.k.GetLowestInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest inferer score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest inferer score found for this topic")
	}

	return &types.GetCurrentLowestInfererScoreResponse{Score: &lowestInfererScore}, nil
}

func (qs queryServer) GetForecastScoresUntilBlock(
	ctx context.Context,
	req *types.GetForecastScoresUntilBlockRequest,
) (
	*types.GetForecastScoresUntilBlockResponse,
	error,
) {
	forecastScores, err := qs.k.GetForecastScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetForecastScoresUntilBlockResponse{Scores: forecastScores}, nil
}

func (qs queryServer) GetWorkerForecastScoresAtBlock(
	ctx context.Context,
	req *types.GetWorkerForecastScoresAtBlockRequest,
) (
	*types.GetWorkerForecastScoresAtBlockResponse,
	error,
) {
	workerForecastScores, err := qs.k.GetWorkerForecastScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetWorkerForecastScoresAtBlockResponse{Scores: &workerForecastScores}, nil
}

func (qs queryServer) GetCurrentLowestForecasterScore(
	ctx context.Context,
	req *types.GetCurrentLowestForecasterScoreRequest,
) (
	*types.GetCurrentLowestForecasterScoreResponse,
	error,
) {
	lowestForecasterScore, found, err := qs.k.GetLowestForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest forecaster score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest forecaster score found for this topic")
	}

	return &types.GetCurrentLowestForecasterScoreResponse{Score: &lowestForecasterScore}, nil
}

func (qs queryServer) GetReputersScoresAtBlock(
	ctx context.Context,
	req *types.GetReputersScoresAtBlockRequest,
) (
	*types.GetReputersScoresAtBlockResponse,
	error,
) {
	reputersScores, err := qs.k.GetReputersScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetReputersScoresAtBlockResponse{Scores: &reputersScores}, nil
}

func (qs queryServer) GetCurrentLowestReputerScore(
	ctx context.Context,
	req *types.GetCurrentLowestReputerScoreRequest,
) (
	*types.GetCurrentLowestReputerScoreResponse,
	error,
) {
	lowestReputerScore, found, err := qs.k.GetLowestReputerScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest reputer score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest reputer score found for this topic")
	}

	return &types.GetCurrentLowestReputerScoreResponse{Score: &lowestReputerScore}, nil
}

func (qs queryServer) GetListeningCoefficient(
	ctx context.Context,
	req *types.GetListeningCoefficientRequest,
) (
	*types.GetListeningCoefficientResponse,
	error,
) {
	listeningCoefficient, err := qs.k.GetListeningCoefficient(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetListeningCoefficientResponse{ListeningCoefficient: &listeningCoefficient}, nil
}
