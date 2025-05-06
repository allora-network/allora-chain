package queryserver

import (
	"context"
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (qs queryServer) GetInfererScoreEma(ctx context.Context, req *types.GetInfererScoreEmaRequest) (_ *types.GetInfererScoreEmaResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"address":  req.Inferer,
	}
	defer metrics.RecordMetrics("GetInfererScoreEma", time.Now(), &err, labels)

	latestInfererScore, err := qs.k.GetInfererScoreEma(ctx, req.TopicId, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &types.GetInfererScoreEmaResponse{Score: &latestInfererScore}, nil
}

func (qs queryServer) GetForecasterScoreEma(ctx context.Context, req *types.GetForecasterScoreEmaRequest) (_ *types.GetForecasterScoreEmaResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"address":  req.Forecaster,
	}
	defer metrics.RecordMetrics("GetForecasterScoreEma", time.Now(), &err, labels)

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
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"address":  req.Reputer,
	}
	defer metrics.RecordMetrics("GetReputerScoreEma", time.Now(), &err, labels)

	latestReputerScore, err := qs.k.GetReputerScoreEma(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetReputerScoreEmaResponse{Score: &latestReputerScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileForecasterScoreEmaRequest) (_ *types.GetPreviousTopicQuantileForecasterScoreEmaResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetPreviousTopicQuantileForecasterScoreEma", time.Now(), &err, labels)
	previousQuantileForecasterScore, err := qs.k.GetPreviousTopicQuantileForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicQuantileForecasterScoreEmaResponse{Value: previousQuantileForecasterScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileInfererScoreEmaRequest) (_ *types.GetPreviousTopicQuantileInfererScoreEmaResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetPreviousTopicQuantileInfererScoreEma", time.Now(), &err, labels)
	previousQuantileInfererScore, err := qs.k.GetPreviousTopicQuantileInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicQuantileInfererScoreEmaResponse{Value: previousQuantileInfererScore}, nil
}

func (qs queryServer) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, req *types.GetPreviousTopicQuantileReputerScoreEmaRequest) (resp *types.GetPreviousTopicQuantileReputerScoreEmaResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetPreviousTopicQuantileReputerScoreEma", time.Now(), &err, labels)
	previousQuantileReputerScore, err := qs.k.GetPreviousTopicQuantileReputerScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	resp = &types.GetPreviousTopicQuantileReputerScoreEmaResponse{Value: previousQuantileReputerScore}
	return resp, nil
}

func (qs queryServer) GetInferenceScoresUntilBlock(ctx context.Context, req *types.GetInferenceScoresUntilBlockRequest) (_ *types.GetInferenceScoresUntilBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":     strconv.FormatUint(req.TopicId, 10),
		"block_height": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetInferenceScoresUntilBlock", time.Now(), &err, labels)
	inferenceScores, err := qs.k.GetInferenceScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetInferenceScoresUntilBlockResponse{Scores: inferenceScores}, nil
}

func (qs queryServer) GetWorkerInferenceScoresAtBlock(ctx context.Context, req *types.GetWorkerInferenceScoresAtBlockRequest) (_ *types.GetWorkerInferenceScoresAtBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":     strconv.FormatUint(req.TopicId, 10),
		"block_height": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetWorkerInferenceScoresAtBlock", time.Now(), &err, labels)
	workerInferenceScores, err := qs.k.GetWorkerInferenceScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetWorkerInferenceScoresAtBlockResponse{Scores: &workerInferenceScores}, nil
}

func (qs queryServer) GetCurrentLowestInfererScore(ctx context.Context, req *types.GetCurrentLowestInfererScoreRequest) (_ *types.GetCurrentLowestInfererScoreResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetCurrentLowestInfererScore", time.Now(), &err, labels)
	lowestInfererScore, found, err := qs.k.GetLowestInfererScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest inferer score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest inferer score found for this topic")
	}

	return &types.GetCurrentLowestInfererScoreResponse{Score: &lowestInfererScore}, nil
}

func (qs queryServer) GetForecastScoresUntilBlock(ctx context.Context, req *types.GetForecastScoresUntilBlockRequest) (_ *types.GetForecastScoresUntilBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":     strconv.FormatUint(req.TopicId, 10),
		"block_height": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetForecastScoresUntilBlock", time.Now(), &err, labels)
	forecastScores, err := qs.k.GetForecastScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetForecastScoresUntilBlockResponse{Scores: forecastScores}, nil
}

func (qs queryServer) GetWorkerForecastScoresAtBlock(ctx context.Context, req *types.GetWorkerForecastScoresAtBlockRequest) (_ *types.GetWorkerForecastScoresAtBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":     strconv.FormatUint(req.TopicId, 10),
		"block_height": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetWorkerForecastScoresAtBlock", time.Now(), &err, labels)
	workerForecastScores, err := qs.k.GetWorkerForecastScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetWorkerForecastScoresAtBlockResponse{Scores: &workerForecastScores}, nil
}

func (qs queryServer) GetCurrentLowestForecasterScore(ctx context.Context, req *types.GetCurrentLowestForecasterScoreRequest) (_ *types.GetCurrentLowestForecasterScoreResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetCurrentLowestForecasterScore", time.Now(), &err, labels)
	lowestForecasterScore, found, err := qs.k.GetLowestForecasterScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest forecaster score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest forecaster score found for this topic")
	}

	return &types.GetCurrentLowestForecasterScoreResponse{Score: &lowestForecasterScore}, nil
}

func (qs queryServer) GetReputersScoresAtBlock(ctx context.Context, req *types.GetReputersScoresAtBlockRequest) (_ *types.GetReputersScoresAtBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":     strconv.FormatUint(req.TopicId, 10),
		"block_height": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetReputersScoresAtBlock", time.Now(), &err, labels)
	reputersScores, err := qs.k.GetReputersScoresAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetReputersScoresAtBlockResponse{Scores: &reputersScores}, nil
}

func (qs queryServer) GetCurrentLowestReputerScore(ctx context.Context, req *types.GetCurrentLowestReputerScoreRequest) (_ *types.GetCurrentLowestReputerScoreResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetCurrentLowestReputerScore", time.Now(), &err, labels)
	lowestReputerScore, found, err := qs.k.GetLowestReputerScoreEma(ctx, req.TopicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting lowest reputer score EMA")
	} else if !found {
		return nil, errorsmod.Wrap(err, "no lowest reputer score found for this topic")
	}

	return &types.GetCurrentLowestReputerScoreResponse{Score: &lowestReputerScore}, nil
}

func (qs queryServer) GetListeningCoefficient(ctx context.Context, req *types.GetListeningCoefficientRequest) (_ *types.GetListeningCoefficientResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"address":  req.Reputer,
	}
	defer metrics.RecordMetrics("GetListeningCoefficient", time.Now(), &err, labels)

	listeningCoefficient, err := qs.k.GetListeningCoefficient(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.GetListeningCoefficientResponse{ListeningCoefficient: &listeningCoefficient}, nil
}

func (qs queryServer) GetTopicInitialInfererEmaScore(ctx context.Context, req *types.GetTopicInitialInfererEmaScoreRequest) (_ *types.GetTopicInitialInfererEmaScoreResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicInitialInfererEmaScore", time.Now(), &err, labels)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	score, err := qs.k.GetTopicInitialInfererEmaScore(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicInitialInfererEmaScoreResponse{Score: score}, nil
}

func (qs queryServer) GetTopicInitialForecasterEmaScore(ctx context.Context, req *types.GetTopicInitialForecasterEmaScoreRequest) (_ *types.GetTopicInitialForecasterEmaScoreResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicInitialForecasterEmaScore", time.Now(), &err, labels)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	score, err := qs.k.GetTopicInitialForecasterEmaScore(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicInitialForecasterEmaScoreResponse{Score: score}, nil
}

func (qs queryServer) GetTopicInitialReputerEmaScore(ctx context.Context, req *types.GetTopicInitialReputerEmaScoreRequest) (_ *types.GetTopicInitialReputerEmaScoreResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicInitialReputerEmaScore", time.Now(), &err, labels)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	score, err := qs.k.GetTopicInitialReputerEmaScore(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicInitialReputerEmaScoreResponse{Score: score}, nil
}
