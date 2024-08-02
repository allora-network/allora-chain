package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	alloraMath "github.com/allora-network/allora-chain/math"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetWorkerLatestInferenceByTopicId handles the query for the latest inference by a specific worker for a given topic.
func (qs queryServer) GetWorkerLatestInferenceByTopicId(ctx context.Context, req *types.QueryWorkerLatestInferenceRequest) (*types.QueryWorkerLatestInferenceResponse, error) {
	if err := qs.k.ValidateStringIsBech32(req.WorkerAddress); err != nil {
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

	return &types.QueryWorkerLatestInferenceResponse{LatestInference: &inference}, nil
}

func (qs queryServer) GetInferencesAtBlock(ctx context.Context, req *types.QueryInferencesAtBlockRequest) (*types.QueryInferencesAtBlockResponse, error) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inferences, err := qs.k.GetInferencesAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryInferencesAtBlockResponse{Inferences: inferences}, nil
}

// Return full set of inferences in I_i from the chain
func (qs queryServer) GetNetworkInferencesAtBlock(
	ctx context.Context,
	req *types.QueryNetworkInferencesAtBlockRequest,
) (*types.QueryNetworkInferencesAtBlockResponse, error) {
	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	}
	if topic.EpochLastEnded == 0 {
		return nil, status.Errorf(codes.NotFound, "network inference not available for topic %v", req.TopicId)
	}

	networkInferences, _, _, _, err := synth.GetNetworkInferencesAtBlock(
		sdk.UnwrapSDKContext(ctx),
		qs.k,
		req.TopicId,
		req.BlockHeightLastInference,
		req.BlockHeightLastReward,
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryNetworkInferencesAtBlockResponse{NetworkInferences: networkInferences}, nil
}

// Return full set of inferences in I_i from the chain, as well as weights and forecast implied inferences
func (qs queryServer) GetLatestNetworkInference(
	ctx context.Context,
	req *types.QueryLatestNetworkInferencesRequest,
) (
	*types.QueryLatestNetworkInferencesResponse,
	error,
) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	networkInferences, forecastImpliedInferenceByWorker, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, err := synth.GetLatestNetworkInference(
		sdk.UnwrapSDKContext(ctx),
		qs.k,
		req.TopicId,
	)
	if err != nil {
		return nil, err
	}

	ciRawPercentiles, ciValues, err :=
		qs.GetConfidenceIntervalsForInferenceData(
			networkInferences,
			forecastImpliedInferenceByWorker,
			infererWeights,
			forecasterWeights,
		)

	if ciRawPercentiles == nil {
		ciRawPercentiles = []alloraMath.Dec{}
	}

	if ciValues == nil {
		ciValues = []alloraMath.Dec{}
	}

	inferers := alloraMath.GetSortedKeys(infererWeights)
	forecasters := alloraMath.GetSortedKeys(forecasterWeights)

	return &types.QueryLatestNetworkInferencesResponse{
		NetworkInferences:                networkInferences,
		InfererWeights:                   synth.ConvertWeightsToArrays(inferers, infererWeights),
		ForecasterWeights:                synth.ConvertWeightsToArrays(forecasters, forecasterWeights),
		ForecastImpliedInferences:        synth.ConvertForecastImpliedInferencesToArrays(forecasters, forecastImpliedInferenceByWorker),
		InferenceBlockHeight:             inferenceBlockHeight,
		LossBlockHeight:                  lossBlockHeight,
		ConfidenceIntervalRawPercentiles: ciRawPercentiles,
		ConfidenceIntervalValues:         ciValues,
	}, nil
}

func (qs queryServer) GetLatestAvailableNetworkInference(
	ctx context.Context,
	req *types.QueryLatestNetworkInferencesRequest,
) (
	*types.QueryLatestNetworkInferencesResponse,
	error,
) {

	lastWorkerCommit, err := qs.k.GetTopicLastWorkerPayload(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	lastReputerCommit, err := qs.k.GetTopicLastReputerPayload(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	networkInferences, forecastImpliedInferenceByWorker, infererWeights, forecasterWeights, err :=
		synth.GetNetworkInferencesAtBlock(
			sdk.UnwrapSDKContext(ctx),
			qs.k,
			req.TopicId,
			lastWorkerCommit.Nonce.BlockHeight,
			lastReputerCommit.Nonce.BlockHeight,
		)
	if err != nil {
		return nil, err
	}

	ciRawPercentiles, ciValues, err :=
		qs.GetConfidenceIntervalsForInferenceData(
			networkInferences,
			forecastImpliedInferenceByWorker,
			infererWeights,
			forecasterWeights,
		)

	if ciRawPercentiles == nil {
		ciRawPercentiles = []alloraMath.Dec{}
	}

	if ciValues == nil {
		ciValues = []alloraMath.Dec{}
	}

	inferers := alloraMath.GetSortedKeys(infererWeights)
	forecasters := alloraMath.GetSortedKeys(forecasterWeights)

	return &types.QueryLatestNetworkInferencesResponse{
		NetworkInferences:                networkInferences,
		InfererWeights:                   synth.ConvertWeightsToArrays(inferers, infererWeights),
		ForecasterWeights:                synth.ConvertWeightsToArrays(forecasters, forecasterWeights),
		ForecastImpliedInferences:        synth.ConvertForecastImpliedInferencesToArrays(forecasters, forecastImpliedInferenceByWorker),
		InferenceBlockHeight:             lastWorkerCommit.Nonce.BlockHeight,
		LossBlockHeight:                  lastReputerCommit.Nonce.BlockHeight,
		ConfidenceIntervalRawPercentiles: ciRawPercentiles,
		ConfidenceIntervalValues:         ciValues,
	}, nil
}

func (qs queryServer) GetConfidenceIntervalsForInferenceData(
	networkInferences *emissions.ValueBundle,
	forecastImpliedInferenceByWorker map[string]*emissions.Inference,
	infererWeights map[string]alloraMath.Dec,
	forecasterWeights map[string]alloraMath.Dec,
) ([]alloraMath.Dec, []alloraMath.Dec, error) {
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
			inferences = append(inferences, forecastImpliedInferenceByWorker[forecast.Worker].Value)
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
	var err error
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

func (qs queryServer) GetLatestTopicInferences(
	ctx context.Context,
	req *types.QueryLatestTopicInferencesRequest,
) (
	*types.QueryLatestTopicInferencesResponse,
	error,
) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inferences, blockHeight, err := qs.k.GetLatestTopicInferences(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestTopicInferencesResponse{Inferences: inferences, BlockHeight: blockHeight}, nil
}

func (qs queryServer) IsWorkerNonceUnfulfilled(
	ctx context.Context,
	req *types.QueryIsWorkerNonceUnfulfilledRequest,
) (
	*types.QueryIsWorkerNonceUnfulfilledResponse,
	error,
) {
	isWorkerNonceUnfulfilled, err :=
		qs.k.IsWorkerNonceUnfulfilled(ctx, req.TopicId, &types.Nonce{BlockHeight: req.BlockHeight})

	return &types.QueryIsWorkerNonceUnfulfilledResponse{IsWorkerNonceUnfulfilled: isWorkerNonceUnfulfilled}, err
}

func (qs queryServer) GetUnfulfilledWorkerNonces(
	ctx context.Context,
	req *types.QueryUnfulfilledWorkerNoncesRequest,
) (
	*types.QueryUnfulfilledWorkerNoncesResponse,
	error,
) {
	unfulfilledNonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryUnfulfilledWorkerNoncesResponse{Nonces: &unfulfilledNonces}, nil
}

func (qs queryServer) GetInfererNetworkRegret(
	ctx context.Context,
	req *types.QueryInfererNetworkRegretRequest,
) (
	*types.QueryInfererNetworkRegretResponse,
	error,
) {
	infererNetworkRegret, _, err := qs.k.GetInfererNetworkRegret(ctx, req.TopicId, req.ActorId)
	if err != nil {
		return nil, err
	}

	return &types.QueryInfererNetworkRegretResponse{Regret: &infererNetworkRegret}, nil
}

func (qs queryServer) GetForecasterNetworkRegret(
	ctx context.Context,
	req *types.QueryForecasterNetworkRegretRequest,
) (
	*types.QueryForecasterNetworkRegretResponse,
	error,
) {
	forecasterNetworkRegret, _, err := qs.k.GetForecasterNetworkRegret(ctx, req.TopicId, req.Worker)
	if err != nil {
		return nil, err
	}

	return &types.QueryForecasterNetworkRegretResponse{Regret: &forecasterNetworkRegret}, nil
}

func (qs queryServer) GetOneInForecasterNetworkRegret(
	ctx context.Context,
	req *types.QueryOneInForecasterNetworkRegretRequest,
) (
	*types.QueryOneInForecasterNetworkRegretResponse,
	error,
) {
	oneInForecasterNetworkRegret, _, err := qs.k.GetOneInForecasterNetworkRegret(ctx, req.TopicId, req.Forecaster, req.Inferer)
	if err != nil {
		return nil, err
	}

	return &types.QueryOneInForecasterNetworkRegretResponse{Regret: &oneInForecasterNetworkRegret}, nil
}

func (qs queryServer) GetOneInForecasterSelfNetworkRegret(
	ctx context.Context,
	req *types.QueryOneInForecasterSelfNetworkRegretRequest,
) (
	*types.QueryOneInForecasterSelfNetworkRegretResponse,
	error,
) {
	oneInForecasterSelfNetworkRegret, _, err := qs.k.GetOneInForecasterNetworkRegret(ctx, req.TopicId, req.Forecaster, req.Forecaster)
	if err != nil {
		return nil, err
	}

	return &types.QueryOneInForecasterSelfNetworkRegretResponse{Regret: &oneInForecasterSelfNetworkRegret}, nil
}
