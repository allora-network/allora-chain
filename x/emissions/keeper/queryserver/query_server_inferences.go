package queryserver

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/metrics"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// GetWorkerLatestInferenceByTopicId handles the query for the latest inference by a specific worker for a given topic.
func (qs queryServer) GetWorkerLatestInferenceByTopicId(ctx context.Context, req *emissionstypes.GetWorkerLatestInferenceByTopicIdRequest) (_ *emissionstypes.GetWorkerLatestInferenceByTopicIdResponse, err error) {
	defer metrics.RecordMetrics("GetWorkerLatestInferenceByTopicId", time.Now(), &err)

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
	defer metrics.RecordMetrics("GetInferencesAtBlock", time.Now(), &err)

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

// Return full set of inferences in I_i from the chain
func (qs queryServer) GetNetworkInferencesAtBlock(ctx context.Context, req *emissionstypes.GetNetworkInferencesAtBlockRequest) (_ *emissionstypes.GetNetworkInferencesAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetNetworkInferencesAtBlock", time.Now(), &err)

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

	return &emissionstypes.GetNetworkInferencesAtBlockResponse{
		NetworkInferences: valueBundleToNetworkInferenceBundle(networkInferences),
	}, nil
}

// An outlier resistant version of GetNetworkInferencesAtBlock
func (qs queryServer) GetNetworkInferencesAtBlockOutlierResistant(
	ctx context.Context,
	req *emissionstypes.GetNetworkInferencesAtBlockOutlierResistantRequest) (_ *emissionstypes.GetNetworkInferencesAtBlockOutlierResistantResponse, err error) {
	defer metrics.RecordMetrics("GetNetworkInferencesAtBlockOutlierResistant", time.Now(), &err)

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

	return &emissionstypes.GetNetworkInferencesAtBlockOutlierResistantResponse{
		NetworkInferences: valueBundleToNetworkInferenceBundle(outlierResistantNetworkInferences),
	}, nil
}

// Return full set of inferences in I_i from the chain
func (qs queryServer) GetLatestNetworkInferences(ctx context.Context, req *emissionstypes.GetLatestNetworkInferencesRequest) (_ *emissionstypes.GetLatestNetworkInferencesResponse, err error) {
	defer metrics.RecordMetrics("GetLatestNetworkInferences", time.Now(), &err)

	result, err := qs.k.GetLatestNetworkInferences(ctx, req.TopicId, false)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	return &emissionstypes.GetLatestNetworkInferencesResponse{
		NetworkInferences:    valueBundleToNetworkInferenceBundle(result),
		InferenceBlockHeight: result.ReputerRequestNonce.ReputerNonce.BlockHeight,
	}, nil
}

// Gets latest network inference with outlier resistance
func (qs queryServer) GetLatestNetworkInferencesOutlierResistant(ctx context.Context,
	req *emissionstypes.GetLatestNetworkInferencesOutlierResistantRequest) (
	_ *emissionstypes.GetLatestNetworkInferencesOutlierResistantResponse, err error) {
	defer metrics.RecordMetrics("GetLatestNetworkInferencesOutlierResistant", time.Now(), &err)

	result, err := qs.k.GetLatestNetworkInferences(ctx, req.TopicId, true)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	return &emissionstypes.GetLatestNetworkInferencesOutlierResistantResponse{
		NetworkInferences:    valueBundleToNetworkInferenceBundle(result),
		InferenceBlockHeight: result.ReputerRequestNonce.ReputerNonce.BlockHeight,
	}, nil
}

// TODO: re-create network inferences store to remove this conversion logic
func valueBundleToNetworkInferenceBundle(vb *emissionstypes.ValueBundle) *emissionstypes.NetworkInferenceBundle {
	const label0 uint32 = 0

	out := &emissionstypes.NetworkInferenceBundle{
		TopicId: vb.TopicId,
		Nonce:   vb.ReputerRequestNonce.ReputerNonce.BlockHeight,

		CombinedValue: []*emissionstypes.LabeledValue{
			{LabelId: label0, Value: vb.CombinedValue},
		},
		NaiveValue: []*emissionstypes.LabeledValue{
			{LabelId: label0, Value: vb.NaiveValue},
		},
	}

	// InfererValues: []*WorkerAttributedValue -> []*WorkerInference
	if n := len(vb.InfererValues); n > 0 {
		out.InfererValues = make([]*emissionstypes.WorkerInference, n)
		for i, v := range vb.InfererValues {
			out.InfererValues[i] = &emissionstypes.WorkerInference{
				Worker: v.Worker,
				Values: []*emissionstypes.LabeledValue{
					{LabelId: label0, Value: v.Value},
				},
			}
		}
	}

	// ForecasterValues: []*WorkerAttributedValue -> []*WorkerInference
	if n := len(vb.ForecasterValues); n > 0 {
		out.ForecasterValues = make([]*emissionstypes.WorkerInference, n)
		for i, v := range vb.ForecasterValues {
			out.ForecasterValues[i] = &emissionstypes.WorkerInference{
				Worker: v.Worker,
				Values: []*emissionstypes.LabeledValue{
					{LabelId: label0, Value: v.Value},
				},
			}
		}
	}

	// OneOutInfererValues: []*WithheldWorkerAttributedValue -> []*OneOutInfererValue
	if n := len(vb.OneOutInfererValues); n > 0 {
		out.OneOutInfererValues = make([]*emissionstypes.OneOutInfererValue, n)
		for i, v := range vb.OneOutInfererValues {
			out.OneOutInfererValues[i] = &emissionstypes.OneOutInfererValue{
				WithheldInferer: v.Worker,
				CombinedInference: []*emissionstypes.LabeledValue{
					{LabelId: label0, Value: v.Value},
				},
			}
		}
	}

	// OneOutForecasterValues: []*WithheldWorkerAttributedValue -> []*OneOutForecasterValue
	if n := len(vb.OneOutForecasterValues); n > 0 {
		out.OneOutForecasterValues = make([]*emissionstypes.OneOutForecasterValue, n)
		for i, v := range vb.OneOutForecasterValues {
			out.OneOutForecasterValues[i] = &emissionstypes.OneOutForecasterValue{
				WithheldForecaster: v.Worker,
				CombinedInference: []*emissionstypes.LabeledValue{
					{LabelId: label0, Value: v.Value},
				},
			}
		}
	}

	// OneInForecasterValues: []*WorkerAttributedValue -> []*OneInForecasterValue
	if n := len(vb.OneInForecasterValues); n > 0 {
		out.OneInForecasterValues = make([]*emissionstypes.OneInForecasterValue, n)
		for i, v := range vb.OneInForecasterValues {
			out.OneInForecasterValues[i] = &emissionstypes.OneInForecasterValue{
				Forecaster: v.Worker,
				CombinedInference: []*emissionstypes.LabeledValue{
					{LabelId: label0, Value: v.Value},
				},
			}
		}
	}

	// OneOutInfererForecasterValues: []*OneOutInfererForecasterValues -> []*OneOutInfererForecasterValue
	// Old structure: per forecaster -> list of withheld-inferer values (but no explicit withheld-inferer in output).
	// Here we emit one record per (forecaster, withheldInferer) pair.
	if n := len(vb.OneOutInfererForecasterValues); n > 0 {
		// pre-size approximately: sum of per-forecaster rows
		total := 0
		for _, row := range vb.OneOutInfererForecasterValues {
			total += len(row.OneOutInfererValues)
		}
		if total > 0 {
			out.OneOutInfererForecasterValues = make([]*emissionstypes.OneOutInfererForecasterValue, 0, total)
			for _, row := range vb.OneOutInfererForecasterValues {
				fc := row.Forecaster
				for _, cell := range row.OneOutInfererValues {
					out.OneOutInfererForecasterValues = append(out.OneOutInfererForecasterValues,
						&emissionstypes.OneOutInfererForecasterValue{
							Forecaster:      fc,
							WithheldInferer: cell.Worker,
							CombinedInference: []*emissionstypes.LabeledValue{
								{LabelId: label0, Value: cell.Value},
							},
						},
					)
				}
			}
		}
	}

	return out
}

func (qs queryServer) GetLatestTopicInferences(ctx context.Context, req *emissionstypes.GetLatestTopicInferencesRequest) (_ *emissionstypes.GetLatestTopicInferencesResponse, err error) {
	defer metrics.RecordMetrics("GetLatestTopicInferences", time.Now(), &err)
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
	defer metrics.RecordMetrics("IsWorkerNonceUnfulfilled", time.Now(), &err)
	isWorkerNonceUnfulfilled, err :=
		qs.k.IsWorkerNonceUnfulfilled(ctx, req.TopicId, &emissionstypes.Nonce{BlockHeight: req.BlockHeight})

	return &emissionstypes.IsWorkerNonceUnfulfilledResponse{IsWorkerNonceUnfulfilled: isWorkerNonceUnfulfilled}, err
}

func (qs queryServer) GetUnfulfilledWorkerNonces(ctx context.Context, req *emissionstypes.GetUnfulfilledWorkerNoncesRequest) (_ *emissionstypes.GetUnfulfilledWorkerNoncesResponse, err error) {
	defer metrics.RecordMetrics("GetUnfulfilledWorkerNonces", time.Now(), &err)
	unfulfilledNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetUnfulfilledWorkerNoncesResponse{Nonces: &unfulfilledNonces}, nil
}

func (qs queryServer) GetInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetInfererNetworkRegretRequest) (_ *emissionstypes.GetInfererNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetInfererNetworkRegret", time.Now(), &err)
	infererNetworkRegret, _, err := qs.k.GetInfererNetworkRegret(ctx, req.TopicId, req.ActorId)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetInfererNetworkRegretResponse{Regret: &infererNetworkRegret}, nil
}

func (qs queryServer) GetForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetForecasterNetworkRegretRequest) (_ *emissionstypes.GetForecasterNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetForecasterNetworkRegret", time.Now(), &err)
	forecasterNetworkRegret, _, err := qs.k.GetForecasterNetworkRegret(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetForecasterNetworkRegretResponse{Regret: &forecasterNetworkRegret}, nil
}

func (qs queryServer) GetOneInForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetOneInForecasterNetworkRegretRequest) (_ *emissionstypes.GetOneInForecasterNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetOneInForecasterNetworkRegret", time.Now(), &err)
	oneInForecasterNetworkRegret, _, err := qs.k.GetOneInForecasterNetworkRegret(ctx, req.TopicId, req.Forecaster, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneInForecasterNetworkRegretResponse{Regret: &oneInForecasterNetworkRegret}, nil
}

func (qs queryServer) GetNaiveInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetNaiveInfererNetworkRegretRequest) (_ *emissionstypes.GetNaiveInfererNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetNaiveInfererNetworkRegret", time.Now(), &err)
	regret, _, err := qs.k.GetNaiveInfererNetworkRegret(ctx, req.TopicId, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetNaiveInfererNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutInfererInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutInfererInfererNetworkRegretRequest) (_ *emissionstypes.GetOneOutInfererInfererNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetOneOutInfererInfererNetworkRegret", time.Now(), &err)
	regret, _, err := qs.k.GetOneOutInfererInfererNetworkRegret(ctx, req.TopicId, req.OneOutInferer, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutInfererInfererNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutInfererForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutInfererForecasterNetworkRegretRequest) (_ *emissionstypes.GetOneOutInfererForecasterNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetOneOutInfererForecasterNetworkRegret", time.Now(), &err)
	regret, _, err := qs.k.GetOneOutInfererForecasterNetworkRegret(ctx, req.TopicId, req.OneOutInferer, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutInfererForecasterNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutForecasterInfererNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutForecasterInfererNetworkRegretRequest) (_ *emissionstypes.GetOneOutForecasterInfererNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetOneOutForecasterInfererNetworkRegret", time.Now(), &err)
	regret, _, err := qs.k.GetOneOutForecasterInfererNetworkRegret(ctx, req.TopicId, req.OneOutForecaster, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutForecasterInfererNetworkRegretResponse{Regret: &regret}, nil
}

func (qs queryServer) GetOneOutForecasterForecasterNetworkRegret(ctx context.Context, req *emissionstypes.GetOneOutForecasterForecasterNetworkRegretRequest) (_ *emissionstypes.GetOneOutForecasterForecasterNetworkRegretResponse, err error) {
	defer metrics.RecordMetrics("GetOneOutForecasterForecasterNetworkRegret", time.Now(), &err)
	regret, _, err := qs.k.GetOneOutForecasterForecasterNetworkRegret(ctx, req.TopicId, req.OneOutForecaster, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetOneOutForecasterForecasterNetworkRegretResponse{Regret: &regret}, nil
}
