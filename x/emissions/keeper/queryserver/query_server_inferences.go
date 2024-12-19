package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	alloraMath "github.com/allora-network/allora-chain/math"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

func (qs queryServer) GetActiveInferersForTopic(ctx context.Context, req *emissionstypes.GetActiveInferersForTopicRequest) (_ *emissionstypes.GetActiveInferersForTopicResponse, err error) {
	defer metrics.RecordMetrics("GetActiveInferersForTopic", time.Now(), &err)

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
	defer metrics.RecordMetrics("GetNetworkInferencesAtBlock", time.Now(), &err)

	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	}
	if topic.EpochLastEnded == 0 {
		return nil, status.Errorf(codes.NotFound, "network inference not available for topic %v", req.TopicId)
	}

	result, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(ctx),
		qs.k,
		req.TopicId,
		&req.BlockHeightLastInference,
		false,
	)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetNetworkInferencesAtBlockResponse{NetworkInferences: result.NetworkInferences}, nil
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

	result, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(ctx),
		qs.k,
		req.TopicId,
		&req.BlockHeightLastInference,
		true,
	)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetNetworkInferencesAtBlockOutlierResistantResponse{NetworkInferences: result.NetworkInferences}, nil
}

// Input parameters type for both outlier resistant and non-outlier resistant functions
type NetworkInferencesParams struct {
	ctx              context.Context
	topicId          uint64
	outlierResistant bool
}

// Output result type for both outlier resistant and non-outlier resistant functions
type NetworkInferencesResult struct {
	networkInferences                *emissionstypes.ValueBundle
	infererWeights                   []*emissionstypes.RegretInformedWeight
	forecasterWeights                []*emissionstypes.RegretInformedWeight
	inferenceBlockHeight             int64
	lossBlockHeight                  int64
	confidenceIntervalRawPercentiles []alloraMath.Dec
	confidenceIntervalValues         []alloraMath.Dec
}

// Base function to get latest network inferences for both outlier resistant and non-outlier resistant functions
func (qs queryServer) getLatestNetworkInferencesBase(
	params NetworkInferencesParams,
) (out *NetworkInferencesResult, err error) {
	defer metrics.RecordMetrics("GetLatestNetworkInferences", time.Now(), &err)

	result, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(params.ctx),
		qs.k,
		params.topicId,
		nil,
		params.outlierResistant,
	)
	if err != nil {
		return nil, err
	}

	ciRawPercentiles, ciValues, err :=
		qs.GetConfidenceIntervalsForInferenceData(
			result.NetworkInferences,
			result.InfererToWeight,
			result.ForecasterToWeight,
		)
	if err != nil {
		return nil, err
	}

	if ciRawPercentiles == nil {
		ciRawPercentiles = []alloraMath.Dec{}
	}

	if ciValues == nil {
		ciValues = []alloraMath.Dec{}
	}

	inferers := alloraMath.GetSortedKeys(result.InfererToWeight)
	forecasters := alloraMath.GetSortedKeys(result.ForecasterToWeight)

	return &NetworkInferencesResult{
		networkInferences:                result.NetworkInferences,
		infererWeights:                   synth.ConvertWeightsToArrays(inferers, result.InfererToWeight),
		forecasterWeights:                synth.ConvertWeightsToArrays(forecasters, result.ForecasterToWeight),
		inferenceBlockHeight:             result.InferenceBlockHeight,
		lossBlockHeight:                  result.LossBlockHeight,
		confidenceIntervalRawPercentiles: ciRawPercentiles,
		confidenceIntervalValues:         ciValues,
	}, nil
}

// Return full set of inferences in I_i from the chain, as well as weights and forecast implied inferences
func (qs queryServer) GetLatestNetworkInferences(ctx context.Context, req *emissionstypes.GetLatestNetworkInferencesRequest) (_ *emissionstypes.GetLatestNetworkInferencesResponse, err error) {
	defer metrics.RecordMetrics("GetLatestNetworkInferences", time.Now(), &err)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	// Create params for base function
	params := NetworkInferencesParams{
		ctx:              ctx,
		topicId:          req.TopicId,
		outlierResistant: false,
	}

	// Call base function
	result, err := qs.getLatestNetworkInferencesBase(params)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	return &emissionstypes.GetLatestNetworkInferencesResponse{
		NetworkInferences:                result.networkInferences,
		InfererWeights:                   result.infererWeights,
		ForecasterWeights:                result.forecasterWeights,
		InferenceBlockHeight:             result.inferenceBlockHeight,
		LossBlockHeight:                  result.lossBlockHeight,
		ConfidenceIntervalRawPercentiles: result.confidenceIntervalRawPercentiles,
		ConfidenceIntervalValues:         result.confidenceIntervalValues,
	}, nil
}

// Gets latest network inference with outlier resistance
func (qs queryServer) GetLatestNetworkInferencesOutlierResistant(ctx context.Context,
	req *emissionstypes.GetLatestNetworkInferencesOutlierResistantRequest) (
	_ *emissionstypes.GetLatestNetworkInferencesOutlierResistantResponse, err error) {
	defer metrics.RecordMetrics("GetLatestNetworkInferencesOutlierResistant", time.Now(), &err)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	// Create params for base function
	params := NetworkInferencesParams{
		ctx:              ctx,
		topicId:          req.TopicId,
		outlierResistant: true,
	}

	// Call base function
	result, err := qs.getLatestNetworkInferencesBase(params)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	return &emissionstypes.GetLatestNetworkInferencesOutlierResistantResponse{
		NetworkInferences:                result.networkInferences,
		InfererWeights:                   result.infererWeights,
		ForecasterWeights:                result.forecasterWeights,
		InferenceBlockHeight:             result.inferenceBlockHeight,
		LossBlockHeight:                  result.lossBlockHeight,
		ConfidenceIntervalRawPercentiles: result.confidenceIntervalRawPercentiles,
		ConfidenceIntervalValues:         result.confidenceIntervalValues,
	}, nil
}

// Base function to get latest available network inferences for both outlier resistant and non-outlier resistant functions
func (qs queryServer) getLatestAvailableNetworkInferencesBase(
	params NetworkInferencesParams,
) (out *NetworkInferencesResult, err error) {
	defer metrics.RecordMetrics("GetLatestAvailableNetworkInferences", time.Now(), &err)

	lastWorkerCommit, err := qs.k.GetWorkerTopicLastCommit(params.ctx, params.topicId)
	if err != nil {
		return nil, err
	}

	lastReputerCommit, err := qs.k.GetReputerTopicLastCommit(params.ctx, params.topicId)
	if err != nil {
		return nil, err
	}

	result, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(params.ctx),
		qs.k,
		params.topicId,
		&lastWorkerCommit.Nonce.BlockHeight,
		params.outlierResistant,
	)
	if err != nil {
		return nil, err
	}

	ciRawPercentiles, ciValues, err :=
		qs.GetConfidenceIntervalsForInferenceData(
			result.NetworkInferences,
			result.InfererToWeight,
			result.ForecasterToWeight,
		)
	if err != nil {
		return nil, err
	}

	if ciRawPercentiles == nil {
		ciRawPercentiles = []alloraMath.Dec{}
	}

	if ciValues == nil {
		ciValues = []alloraMath.Dec{}
	}

	inferers := alloraMath.GetSortedKeys(result.InfererToWeight)
	forecasters := alloraMath.GetSortedKeys(result.ForecasterToWeight)

	return &NetworkInferencesResult{
		networkInferences:                result.NetworkInferences,
		infererWeights:                   synth.ConvertWeightsToArrays(inferers, result.InfererToWeight),
		forecasterWeights:                synth.ConvertWeightsToArrays(forecasters, result.ForecasterToWeight),
		inferenceBlockHeight:             lastWorkerCommit.Nonce.BlockHeight,
		lossBlockHeight:                  lastReputerCommit.Nonce.BlockHeight,
		confidenceIntervalRawPercentiles: ciRawPercentiles,
		confidenceIntervalValues:         ciValues,
	}, nil
}

// Original function converts result to response
func (qs queryServer) GetLatestAvailableNetworkInferences(
	ctx context.Context,
	req *emissionstypes.GetLatestAvailableNetworkInferencesRequest,
) (*emissionstypes.GetLatestAvailableNetworkInferencesResponse, error) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	params := NetworkInferencesParams{
		ctx:              ctx,
		topicId:          req.TopicId,
		outlierResistant: false,
	}

	result, err := qs.getLatestAvailableNetworkInferencesBase(params)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetLatestAvailableNetworkInferencesResponse{
		NetworkInferences:                result.networkInferences,
		InfererWeights:                   result.infererWeights,
		ForecasterWeights:                result.forecasterWeights,
		InferenceBlockHeight:             result.inferenceBlockHeight,
		LossBlockHeight:                  result.lossBlockHeight,
		ConfidenceIntervalRawPercentiles: result.confidenceIntervalRawPercentiles,
		ConfidenceIntervalValues:         result.confidenceIntervalValues,
	}, nil
}

func (qs queryServer) GetLatestAvailableNetworkInferencesOutlierResistant(
	ctx context.Context,
	req *emissionstypes.GetLatestAvailableNetworkInferencesOutlierResistantRequest,
) (*emissionstypes.GetLatestAvailableNetworkInferencesOutlierResistantResponse, error) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	params := NetworkInferencesParams{
		ctx:              ctx,
		topicId:          req.TopicId,
		outlierResistant: true,
	}

	result, err := qs.getLatestAvailableNetworkInferencesBase(params)
	if err != nil {
		return nil, err
	}

	return &emissionstypes.GetLatestAvailableNetworkInferencesOutlierResistantResponse{
		NetworkInferences:                result.networkInferences,
		InfererWeights:                   result.infererWeights,
		ForecasterWeights:                result.forecasterWeights,
		InferenceBlockHeight:             result.inferenceBlockHeight,
		LossBlockHeight:                  result.lossBlockHeight,
		ConfidenceIntervalRawPercentiles: result.confidenceIntervalRawPercentiles,
		ConfidenceIntervalValues:         result.confidenceIntervalValues,
	}, nil
}

func (qs queryServer) GetConfidenceIntervalsForInferenceData(
	networkInferences *emissionstypes.ValueBundle,
	infererWeights map[string]alloraMath.Dec,
	forecasterWeights map[string]alloraMath.Dec,
) (_ []alloraMath.Dec, _ []alloraMath.Dec, err error) {
	defer metrics.RecordMetrics("GetConfidenceIntervalsForInferenceData", time.Now(), &err)
	var inferences []alloraMath.Dec // from inferers + forecast-implied inferences
	var weights []alloraMath.Dec    // weights of all workers

	for _, inference := range networkInferences.InfererValues {
		weight, exists := infererWeights[inference.Worker]
		if exists {
			inferences = append(inferences, inference.Value)
			weights = append(weights, weight)
		}
	}

	for _, forecast := range networkInferences.ForecasterValues {
		weight, exists := forecasterWeights[forecast.Worker]
		if exists {
			inferences = append(inferences, forecast.Value)
			weights = append(weights, weight)
		}
	}

	ciRawPercentiles := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("2.28"),
		alloraMath.MustNewDecFromString("15.87"),
		alloraMath.MustNewDecFromString("50"),
		alloraMath.MustNewDecFromString("84.13"),
		alloraMath.MustNewDecFromString("97.72"),
	}

	var ciValues []alloraMath.Dec
	if len(inferences) == 0 {
		ciRawPercentiles = []alloraMath.Dec{}
		ciValues = []alloraMath.Dec{}
	} else {
		ciValues, err = alloraMath.WeightedPercentile(inferences, weights, ciRawPercentiles)
		if err != nil {
			return nil, nil, err
		}
	}

	return ciRawPercentiles, ciValues, nil
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
