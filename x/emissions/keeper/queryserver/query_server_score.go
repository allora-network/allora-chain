package queryserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetInfererScoreEma(
	ctx context.Context,
	req *types.QueryGetInfererScoreEmaRequest,
) (
	*types.QueryGetInfererScoreEmaResponse,
	error,
) {
	latestInfererScore, err := qs.k.GetInfererScoreEma(ctx, req.TopicId, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetInfererScoreEmaResponse{Score: &latestInfererScore}, nil
}

func (qs queryServer) GetForecasterScoreEma(
	ctx context.Context,
	req *types.QueryGetForecasterScoreEmaRequest,
) (
	*types.QueryGetForecasterScoreEmaResponse,
	error,
) {
	latestForecasterScore, err := qs.k.GetForecasterScoreEma(ctx, req.TopicId, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetForecasterScoreEmaResponse{Score: &latestForecasterScore}, nil
}

func (qs queryServer) GetReputerScoreEma(
	ctx context.Context,
	req *types.QueryGetReputerScoreEmaRequest,
) (
	*types.QueryGetReputerScoreEmaResponse,
	error,
) {
	latestReputerScore, err := qs.k.GetReputerScoreEma(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetReputerScoreEmaResponse{Score: &latestReputerScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, req *types.QueryGetPreviousTopicQuantileForecasterScoreEmaRequest) (
	*types.QueryGetPreviousTopicQuantileForecasterScoreEmaResponse,
	error,
) {
	previousQuantileForecasterScore, err := qs.k.GetPreviousTopicQuantileForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetPreviousTopicQuantileForecasterScoreEmaResponse{Value: previousQuantileForecasterScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, req *types.QueryGetPreviousTopicQuantileInfererScoreEmaRequest) (
	*types.QueryGetPreviousTopicQuantileInfererScoreEmaResponse,
	error,
) {
	previousQuantileInfererScore, err := qs.k.GetPreviousTopicQuantileInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetPreviousTopicQuantileInfererScoreEmaResponse{Value: previousQuantileInfererScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, req *types.QueryGetPreviousTopicQuantileReputerScoreEmaRequest) (
	*types.QueryGetPreviousTopicQuantileReputerScoreEmaResponse,
	error,
) {
	previousQuantileReputerScore, err := qs.k.GetPreviousTopicQuantileReputerScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetPreviousTopicQuantileReputerScoreEmaResponse{Value: previousQuantileReputerScore}, nil
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

func (qs queryServer) GetCurrentLowestInfererScore(
	ctx context.Context,
	req *types.QueryCurrentLowestInfererScoreRequest,
) (
	*types.QueryCurrentLowestInfererScoreResponse,
	error,
) {
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

	return &types.QueryCurrentLowestInfererScoreResponse{Score: &lowestInfererScore}, nil
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

func (qs queryServer) GetCurrentLowestForecasterScore(
	ctx context.Context,
	req *types.QueryCurrentLowestForecasterScoreRequest,
) (
	*types.QueryCurrentLowestForecasterScoreResponse,
	error,
) {
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

	return &types.QueryCurrentLowestForecasterScoreResponse{Score: &lowestForecasterScore}, nil
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

func (qs queryServer) GetCurrentLowestReputerScore(
	ctx context.Context,
	req *types.QueryCurrentLowestReputerScoreRequest,
) (
	*types.QueryCurrentLowestReputerScoreResponse,
	error,
) {
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

	return &types.QueryCurrentLowestReputerScoreResponse{Score: &lowestReputerScore}, nil
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
