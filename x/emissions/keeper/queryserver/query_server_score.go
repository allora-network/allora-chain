package queryserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetInfererScoreEma(
	ctx context.Context,
	req *types.GetInfererScoreEmaRequest,
) (
	_ *types.GetInfererScoreEmaResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetInfererScoreEma", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetForecasterScoreEmaResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetForecasterScoreEma", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetReputerScoreEmaResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetReputerScoreEma", "rpc", time.Now(), returnErr == nil)
	latestReputerScore, err := qs.k.GetReputerScoreEma(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetReputerScoreEmaResponse{Score: &latestReputerScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileForecasterScoreEmaRequest) (
	_ *types.GetPreviousTopicQuantileForecasterScoreEmaResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousTopicQuantileForecasterScoreEma", "rpc", time.Now(), returnErr == nil)
	previousQuantileForecasterScore, err := qs.k.GetPreviousTopicQuantileForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.GetPreviousTopicQuantileForecasterScoreEmaResponse{Value: previousQuantileForecasterScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileInfererScoreEmaRequest) (
	_ *types.GetPreviousTopicQuantileInfererScoreEmaResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousTopicQuantileInfererScoreEma", "rpc", time.Now(), returnErr == nil)
	previousQuantileInfererScore, err := qs.k.GetPreviousTopicQuantileInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.GetPreviousTopicQuantileInfererScoreEmaResponse{Value: previousQuantileInfererScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileReputerScoreEmaRequest) (
	_ *types.GetPreviousTopicQuantileReputerScoreEmaResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousTopicQuantileReputerScoreEma", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetInferenceScoresUntilBlockResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetInferenceScoresUntilBlock", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetWorkerInferenceScoresAtBlockResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetWorkerInferenceScoresAtBlock", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetCurrentLowestInfererScoreResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetCurrentLowestInfererScore", "rpc", time.Now(), returnErr == nil)
	unfulfilledWorkerNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	if len(unfulfilledWorkerNonces.Nonces) == 0 {
		return nil,
			errorsmod.Wrap(types.ErrWorkerNonceWindowNotAvailable,
				"no unfulfilled nonces right now, is the topic active?")
	}
	highestNonce := unfulfilledWorkerNonces.Nonces[0]
	for _, nonce := range unfulfilledWorkerNonces.Nonces {
		if nonce.BlockHeight > highestNonce.BlockHeight {
			highestNonce = nonce
		}
	}

	inferences, err := qs.k.GetInferencesAtBlock(ctx, req.TopicId, highestNonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	lowestInfererScore, _, err := emissionskeeper.GetLowScoreFromAllInferences(
		ctx,
		&qs.k,
		req.TopicId,
		*inferences,
	)
	if err != nil {
		return nil, err
	}

	return &types.GetCurrentLowestInfererScoreResponse{Score: &lowestInfererScore}, nil
}

func (qs queryServer) GetForecastScoresUntilBlock(
	ctx context.Context,
	req *types.GetForecastScoresUntilBlockRequest,
) (
	_ *types.GetForecastScoresUntilBlockResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetForecastScoresUntilBlock", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetWorkerForecastScoresAtBlockResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetWorkerForecastScoresAtBlock", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetCurrentLowestForecasterScoreResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetCurrentLowestForecasterScore", "rpc", time.Now(), returnErr == nil)
	unfulfilledWorkerNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	if len(unfulfilledWorkerNonces.Nonces) == 0 {
		return nil,
			errorsmod.Wrap(types.ErrWorkerNonceWindowNotAvailable,
				"no unfulfilled nonces right now, is the topic active?")
	}
	highestNonce := unfulfilledWorkerNonces.Nonces[0]
	for _, nonce := range unfulfilledWorkerNonces.Nonces {
		if nonce.BlockHeight > highestNonce.BlockHeight {
			highestNonce = nonce
		}
	}
	forecasts, err := qs.k.GetForecastsAtBlock(ctx, req.TopicId, highestNonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	if len(forecasts.Forecasts) == 0 {
		return nil, errorsmod.Wrap(types.ErrInvalidLengthScore, "no scores found for this epoch")
	}

	lowestForecasterScore, _, err := emissionskeeper.GetLowScoreFromAllForecasts(
		ctx,
		&qs.k,
		req.TopicId,
		*forecasts,
	)
	if err != nil {
		return nil, err
	}

	return &types.GetCurrentLowestForecasterScoreResponse{Score: &lowestForecasterScore}, nil
}

func (qs queryServer) GetReputersScoresAtBlock(
	ctx context.Context,
	req *types.GetReputersScoresAtBlockRequest,
) (
	_ *types.GetReputersScoresAtBlockResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetReputersScoresAtBlock", "rpc", time.Now(), returnErr == nil)
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
	_ *types.GetCurrentLowestReputerScoreResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetCurrentLowestReputerScore", "rpc", time.Now(), returnErr == nil)
	unfulfilledReputerNonces, err := qs.k.GetUnfulfilledReputerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	if len(unfulfilledReputerNonces.Nonces) == 0 {
		return nil,
			errorsmod.Wrap(types.ErrWorkerNonceWindowNotAvailable,
				"no unfulfilled nonces right now, is the topic active?")
	}
	highestNonce := unfulfilledReputerNonces.Nonces[0]
	for _, nonce := range unfulfilledReputerNonces.Nonces {
		if nonce.ReputerNonce.BlockHeight > highestNonce.ReputerNonce.BlockHeight {
			highestNonce = nonce
		}
	}
	lossBundles, err := qs.k.GetReputerLossBundlesAtBlock(ctx, req.TopicId, highestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	lowestReputerScore, _, err := emissionskeeper.GetLowScoreFromAllLossBundles(
		ctx,
		&qs.k,
		req.TopicId,
		*lossBundles,
	)
	if err != nil {
		return nil, err
	}

	return &types.GetCurrentLowestReputerScoreResponse{Score: &lowestReputerScore}, nil
}

func (qs queryServer) GetListeningCoefficient(
	ctx context.Context,
	req *types.GetListeningCoefficientRequest,
) (
	_ *types.GetListeningCoefficientResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetListeningCoefficient", "rpc", time.Now(), returnErr == nil)
	listeningCoefficient, err := qs.k.GetListeningCoefficient(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetListeningCoefficientResponse{ListeningCoefficient: &listeningCoefficient}, nil
}
