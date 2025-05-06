package queryserver

import (
	"context"
	"strconv"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetWorkerLatestInferenceByTopicId handles the query for the latest inference by a specific worker for a given topic.
func (qs queryServer) GetWorkerLatestInferenceByTopicId(ctx context.Context, req *emissionstypes.GetWorkerLatestInferenceByTopicIdRequest) (_ *emissionstypes.GetWorkerLatestInferenceByTopicIdResponse, err error) {
	labels := map[string]string{
		"worker_address": req.WorkerAddress,
		"topic_id":       strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetWorkerLatestInferenceByTopicId", time.Now(), &err, labels)

	if err = qs.k.ValidateStringIsBech32(req.WorkerAddress); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inference, err := qs.k.GetWorkerLatestInferenceByTopicId(ctx, req.TopicId, req.WorkerAddress)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetWorkerLatestInferenceByTopicIdResponse{LatestInference: &inference}, nil
}

func (qs queryServer) GetInferencesAtBlock(ctx context.Context, req *emissionstypes.GetInferencesAtBlockRequest) (_ *emissionstypes.GetInferencesAtBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":    strconv.FormatUint(req.TopicId, 10),
		"blockHeight": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetInferencesAtBlock", time.Now(), &err, labels)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inferences, err := qs.k.GetInferencesAtBlock(ctx, req.TopicId, req.BlockHeight, false)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetInferencesAtBlockResponse{Inferences: inferences}, nil
}

func (qs queryServer) GetActiveInferersForTopic(ctx context.Context, req *emissionstypes.GetActiveInferersForTopicRequest) (_ *emissionstypes.GetActiveInferersForTopicResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetActiveInferersForTopic", time.Now(), &err, labels)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inferers, err := qs.k.GetActiveInferersForTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emissionstypes.GetActiveInferersForTopicResponse{Inferers: inferers}, nil
}

// Return full set of inferences in I_i from the chain
func (qs queryServer) GetNetworkInferencesAtBlock(ctx context.Context, req *emissionstypes.GetNetworkInferencesAtBlockRequest) (_ *emissionstypes.GetNetworkInferencesAtBlockResponse, err error) {
	labels := map[string]string{
		"topic_id":    strconv.FormatUint(req.TopicId, 10),
		"blockHeight": strconv.FormatInt(req.BlockHeightLastInference, 10),
	}
	defer metrics.RecordMetrics("GetNetworkInferencesAtBlock", time.Now(), &err, labels)

	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	}
	if topic.EpochLastEnded == 0 {
		return nil, status.Errorf(codes.NotFound, "network inference not available for topic %v", req.TopicId)
	}

	networkInferences, err := qs.k.GetNetworkInferences(ctx, req.TopicId, req.BlockHeightLastInference)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetNetworkInferencesAtBlockResponse{NetworkInferences: networkInferences}, nil
}

// An outlier resistant version of GetNetworkInferencesAtBlock
func (qs queryServer) GetNetworkInferencesAtBlockOutlierResistant(
	ctx context.Context,
	req *emissionstypes.GetNetworkInferencesAtBlockOutlierResistantRequest) (_ *emissionstypes.GetNetworkInferencesAtBlockOutlierResistantResponse, err error) {
	labels := map[string]string{
		"topic_id":    strconv.FormatUint(req.TopicId, 10),
		"blockHeight": strconv.FormatInt(req.BlockHeightLastInference, 10),
	}
	defer metrics.RecordMetrics("GetNetworkInferencesAtBlockOutlierResistant", time.Now(), &err, labels)

	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	}
	if topic.EpochLastEnded == 0 {
		return nil, status.Errorf(codes.NotFound, "network inference not available for topic %v", req.TopicId)
	}

	outlierResistantNetworkInferences, err := qs.k.GetOutlierResistantNetworkInferences(ctx, req.TopicId, req.BlockHeightLastInference)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetNetworkInferencesAtBlockOutlierResistantResponse{NetworkInferences: outlierResistantNetworkInferences}, nil
}

// Return full set of inferences in I_i from the chain
func (qs queryServer) GetLatestNetworkInferences(ctx context.Context, req *emissionstypes.GetLatestNetworkInferencesRequest) (_ *emissionstypes.GetLatestNetworkInferencesResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetLatestNetworkInferences", time.Now(), &err, labels)

	result, err := qs.k.GetLatestNetworkInferences(ctx, req.TopicId, false)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	return &emissionstypes.GetLatestNetworkInferencesResponse{
		NetworkInferences:    result,
		InferenceBlockHeight: result.ReputerRequestNonce.ReputerNonce.BlockHeight,
	}, nil
}

// Gets latest network inference with outlier resistance
func (qs queryServer) GetLatestNetworkInferencesOutlierResistant(ctx context.Context,
	req *emissionstypes.GetLatestNetworkInferencesOutlierResistantRequest) (
	_ *emissionstypes.GetLatestNetworkInferencesOutlierResistantResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetLatestNetworkInferencesOutlierResistant", time.Now(), &err, labels)

	result, err := qs.k.GetLatestNetworkInferences(ctx, req.TopicId, true)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	return &emissionstypes.GetLatestNetworkInferencesOutlierResistantResponse{
		NetworkInferences:    result,
		InferenceBlockHeight: result.ReputerRequestNonce.ReputerNonce.BlockHeight,
	}, nil
}

func (qs queryServer) GetLatestTopicInferences(ctx context.Context, req *emissionstypes.GetLatestTopicInferencesRequest) (_ *emissionstypes.GetLatestTopicInferencesResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetLatestTopicInferences", time.Now(), &err, labels)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inferences, blockHeight, err := qs.k.GetLatestTopicInferences(ctx, req.TopicId, false)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetLatestTopicInferencesResponse{Inferences: inferences, BlockHeight: blockHeight}, nil
}

func (qs queryServer) IsWorkerNonceUnfulfilled(ctx context.Context, req *emissionstypes.IsWorkerNonceUnfulfilledRequest) (_ *emissionstypes.IsWorkerNonceUnfulfilledResponse, err error) {
	labels := map[string]string{
		"topic_id":    strconv.FormatUint(req.TopicId, 10),
		"blockHeight": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("IsWorkerNonceUnfulfilled", time.Now(), &err, labels)
	isWorkerNonceUnfulfilled, err :=
		qs.k.IsWorkerNonceUnfulfilled(ctx, req.TopicId, &emissionstypes.Nonce{BlockHeight: req.BlockHeight})

	return &emissionstypes.IsWorkerNonceUnfulfilledResponse{IsWorkerNonceUnfulfilled: isWorkerNonceUnfulfilled}, err
}

func (qs queryServer) GetUnfulfilledWorkerNonces(ctx context.Context, req *emissionstypes.GetUnfulfilledWorkerNoncesRequest) (_ *emissionstypes.GetUnfulfilledWorkerNoncesResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetUnfulfilledWorkerNonces", time.Now(), &err, labels)
	unfulfilledNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetUnfulfilledWorkerNoncesResponse{Nonces: &unfulfilledNonces}, nil
}

func (qs queryServer) GetInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetInfererNetworkRegretRequest) (_ *emissionstypes.GetInfererNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"actor_id": req.ActorId,
	}
	defer metrics.RecordMetrics("GetInfererNetworkRegret", time.Now(), &err, labels)
	infererNetworkRegret, _, err := qs.k.GetInfererNetworkRegret(ctx, req.TopicId, req.ActorId)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetInfererNetworkRegretResponse{Regret: &infererNetworkRegret}, nil
}

func (qs queryServer) GetForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetForecasterNetworkRegretRequest) (_ *emissionstypes.GetForecasterNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"worker":   req.Worker,
	}
	defer metrics.RecordMetrics("GetForecasterNetworkRegret", time.Now(), &err, labels)
	forecasterNetworkRegret, _, err := qs.k.GetForecasterNetworkRegret(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetForecasterNetworkRegretResponse{Regret: &forecasterNetworkRegret}, nil
}

func (qs queryServer) GetOneInForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetOneInForecasterNetworkRegretRequest) (_ *emissionstypes.GetOneInForecasterNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id":   strconv.FormatUint(req.TopicId, 10),
		"forecaster": req.Forecaster,
		"inferer":    req.Inferer,
	}
	defer metrics.RecordMetrics("GetOneInForecasterNetworkRegret", time.Now(), &err, labels)
	oneInForecasterNetworkRegret, _, err := qs.k.GetOneInForecasterNetworkRegret(ctx, req.TopicId, req.Forecaster, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneInForecasterNetworkRegretResponse{Regret: &oneInForecasterNetworkRegret}, nil
}

func (qs queryServer) GetNaiveInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetNaiveInfererNetworkRegretRequest) (_ *emissionstypes.GetNaiveInfererNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"inferer":  req.Inferer,
	}
	defer metrics.RecordMetrics("GetNaiveInfererNetworkRegret", time.Now(), &err, labels)
	regret, _, err := qs.k.GetNaiveInfererNetworkRegret(ctx, req.TopicId, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetNaiveInfererNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutInfererInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutInfererInfererNetworkRegretRequest) (_ *emissionstypes.GetOneOutInfererInfererNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"inferer":  req.Inferer,
	}
	defer metrics.RecordMetrics("GetOneOutInfererInfererNetworkRegret", time.Now(), &err, labels)
	regret, _, err := qs.k.GetOneOutInfererInfererNetworkRegret(ctx, req.TopicId, req.OneOutInferer, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutInfererInfererNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutInfererForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutInfererForecasterNetworkRegretRequest) (_ *emissionstypes.GetOneOutInfererForecasterNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id":        strconv.FormatUint(req.TopicId, 10),
		"one_out_inferer": req.OneOutInferer,
		"forecaster":      req.Forecaster,
	}
	defer metrics.RecordMetrics("GetOneOutInfererForecasterNetworkRegret", time.Now(), &err, labels)
	regret, _, err := qs.k.GetOneOutInfererForecasterNetworkRegret(ctx, req.TopicId, req.OneOutInferer, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutInfererForecasterNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutForecasterInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutForecasterInfererNetworkRegretRequest) (_ *emissionstypes.GetOneOutForecasterInfererNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id":           strconv.FormatUint(req.TopicId, 10),
		"one_out_forecaster": req.OneOutForecaster,
		"inferer":            req.Inferer,
	}
	defer metrics.RecordMetrics("GetOneOutForecasterInfererNetworkRegret", time.Now(), &err, labels)
	regret, _, err := qs.k.GetOneOutForecasterInfererNetworkRegret(ctx, req.TopicId, req.OneOutForecaster, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutForecasterInfererNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutForecasterForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutForecasterForecasterNetworkRegretRequest) (_ *emissionstypes.GetOneOutForecasterForecasterNetworkRegretResponse, err error) {
	labels := map[string]string{
		"topic_id":           strconv.FormatUint(req.TopicId, 10),
		"one_out_forecaster": req.OneOutForecaster,
		"forecaster":         req.Forecaster,
	}
	defer metrics.RecordMetrics("GetOneOutForecasterForecasterNetworkRegret", time.Now(), &err, labels)
	regret, _, err := qs.k.GetOneOutForecasterForecasterNetworkRegret(ctx, req.TopicId, req.OneOutForecaster, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutForecasterForecasterNetworkRegretResponse{Regret: &regret}, nil
}
