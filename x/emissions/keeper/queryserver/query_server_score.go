package queryserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetInfererScoreEma(ctx context.Context, req *types.GetInfererScoreEmaRequest) (_ *types.GetInfererScoreEmaResponse, err error) {
	defer metrics.RecordMetrics("GetInfererScoreEma", time.Now(), &err)

	latestInfererScore, err := qs.k.GetInfererScoreEma(ctx, req.TopicId, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &types.GetInfererScoreEmaResponse{Score: &latestInfererScore}, nil
}

func (qs queryServer) GetForecasterScoreEma(ctx context.Context, req *types.GetForecasterScoreEmaRequest) (_ *types.GetForecasterScoreEmaResponse, err error) {
	defer metrics.RecordMetrics("GetForecasterScoreEma", time.Now(), &err)

	latestForecasterScore, err := qs.k.GetForecasterScoreEma(ctx, req.TopicId, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &types.GetForecasterScoreEmaResponse{Score: &latestForecasterScore}, nil
}

func (qs queryServer) GetReputerScoreEma(
	ctx context.Context,
	req *types.GetReputerScoreEmaRequest,
) (_ *types.GetReputerScoreEmaResponse, err error) {
	defer metrics.RecordMetrics("GetReputerScoreEma", time.Now(), &err)

	latestReputerScore, err := qs.k.GetReputerScoreEma(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetReputerScoreEmaResponse{Score: &latestReputerScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileForecasterScoreEmaRequest) (_ *types.GetPreviousTopicQuantileForecasterScoreEmaResponse, err error) {
	defer metrics.RecordMetrics("GetPreviousTopicQuantileForecasterScoreEma", time.Now(), &err)
	previousQuantileForecasterScore, err := qs.k.GetPreviousTopicQuantileForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicQuantileForecasterScoreEmaResponse{Value: previousQuantileForecasterScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileInfererScoreEmaRequest) (_ *types.GetPreviousTopicQuantileInfererScoreEmaResponse, err error) {
	defer metrics.RecordMetrics("GetPreviousTopicQuantileInfererScoreEma", time.Now(), &err)
	previousQuantileInfererScore, err := qs.k.GetPreviousTopicQuantileInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicQuantileInfererScoreEmaResponse{Value: previousQuantileInfererScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileReputerScoreEmaRequest) (resp *types.GetPreviousTopicQuantileReputerScoreEmaResponse, err error) {
	defer metrics.RecordMetrics("GetPreviousTopicQuantileReputerScoreEma", time.Now(), &err)
	previousQuantileReputerScore, err := qs.k.GetPreviousTopicQuantileReputerScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	resp = &types.GetPreviousTopicQuantileReputerScoreEmaResponse{Value: previousQuantileReputerScore}
	return resp, nil
}

func (qs queryServer) GetInferenceScoresUntilBlock(ctx context.Context, req *types.GetInferenceScoresUntilBlockRequest) (_ *types.GetInferenceScoresUntilBlockResponse, err error) {
	defer metrics.RecordMetrics("GetInferenceScoresUntilBlock", time.Now(), &err)
	inferenceScores, err := qs.k.GetInferenceScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetInferenceScoresUntilBlockResponse{Scores: inferenceScores}, nil
}

func (qs queryServer) GetWorkerInferenceScoresAtBlock(ctx context.Context, req *types.GetWorkerInferenceScoresAtBlockRequest) (_ *types.GetWorkerInferenceScoresAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetWorkerInferenceScoresAtBlock", time.Now(), &err)
	workerInferenceScores, err := qs.k.GetWorkerInferenceScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetWorkerInferenceScoresAtBlockResponse{Scores: &workerInferenceScores}, nil
}

func (qs queryServer) GetCurrentLowestInfererScore(ctx context.Context, req *types.GetCurrentLowestInfererScoreRequest) (_ *types.GetCurrentLowestInfererScoreResponse, err error) {
	defer metrics.RecordMetrics("GetCurrentLowestInfererScore", time.Now(), &err)
	unfulfilledWorkerNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest inferer score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest inferer score found for this topic")
	}

	return &types.GetCurrentLowestInfererScoreResponse{Score: &lowestInfererScore}, nil
}

func (qs queryServer) GetForecastScoresUntilBlock(ctx context.Context, req *types.GetForecastScoresUntilBlockRequest) (_ *types.GetForecastScoresUntilBlockResponse, err error) {
	defer metrics.RecordMetrics("GetForecastScoresUntilBlock", time.Now(), &err)
	forecastScores, err := qs.k.GetForecastScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetForecastScoresUntilBlockResponse{Scores: forecastScores}, nil
}

func (qs queryServer) GetWorkerForecastScoresAtBlock(ctx context.Context, req *types.GetWorkerForecastScoresAtBlockRequest) (_ *types.GetWorkerForecastScoresAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetWorkerForecastScoresAtBlock", time.Now(), &err)
	workerForecastScores, err := qs.k.GetWorkerForecastScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetWorkerForecastScoresAtBlockResponse{Scores: &workerForecastScores}, nil
}

func (qs queryServer) GetCurrentLowestForecasterScore(ctx context.Context, req *types.GetCurrentLowestForecasterScoreRequest) (_ *types.GetCurrentLowestForecasterScoreResponse, err error) {
	defer metrics.RecordMetrics("GetCurrentLowestForecasterScore", time.Now(), &err)
	unfulfilledWorkerNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest forecaster score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest forecaster score found for this topic")
	}

	return &types.GetCurrentLowestForecasterScoreResponse{Score: &lowestForecasterScore}, nil
}

func (qs queryServer) GetReputersScoresAtBlock(ctx context.Context, req *types.GetReputersScoresAtBlockRequest) (_ *types.GetReputersScoresAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetReputersScoresAtBlock", time.Now(), &err)
	reputersScores, err := qs.k.GetReputersScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetReputersScoresAtBlockResponse{Scores: &reputersScores}, nil
}

func (qs queryServer) GetCurrentLowestReputerScore(ctx context.Context, req *types.GetCurrentLowestReputerScoreRequest) (_ *types.GetCurrentLowestReputerScoreResponse, err error) {
	defer metrics.RecordMetrics("GetCurrentLowestReputerScore", time.Now(), &err)
	unfulfilledReputerNonces, err := qs.k.GetUnfulfilledReputerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest reputer score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest reputer score found for this topic")
	}

	return &types.GetCurrentLowestReputerScoreResponse{Score: &lowestReputerScore}, nil
}

func (qs queryServer) GetListeningCoefficient(ctx context.Context, req *types.GetListeningCoefficientRequest) (_ *types.GetListeningCoefficientResponse, err error) {
	defer metrics.RecordMetrics("GetListeningCoefficient", time.Now(), &err)

	listeningCoefficient, err := qs.k.GetListeningCoefficient(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetListeningCoefficientResponse{ListeningCoefficient: &listeningCoefficient}, nil
}
