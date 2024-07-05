package queryserver

import (
	"context"
	"strconv"

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

	return &types.QueryLatestNetworkInferencesResponse{
		NetworkInferences:                networkInferences,
		InfererWeights:                   synth.ConvertWeightsToArrays(infererWeights),
		ForecasterWeights:                synth.ConvertWeightsToArrays(forecasterWeights),
		ForecastImpliedInferences:        synth.ConvertForecastImpliedInferencesToArrays(forecastImpliedInferenceByWorker),
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

	// Find the latest inference block height
	_, inferenceBlockHeight, err := qs.k.GetLatestTopicInferences(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	if inferenceBlockHeight == 0 {
		return nil, status.Errorf(codes.NotFound, "inferences not yet available for topic %v", req.TopicId)
	}

	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// calculate the previous loss block height
	previousLossBlockHeight := inferenceBlockHeight - topic.EpochLength

	for i := 0; i < int(params.DefaultPageLimit); i++ {
		if previousLossBlockHeight <= 0 {
			return nil, status.Errorf(codes.NotFound, "losses not yet available for topic %v", req.TopicId)
		}

		// check for inferences
		inferences, err := qs.k.GetInferencesAtBlock(ctx, req.TopicId, inferenceBlockHeight)
		if err == nil && inferences != nil && len(inferences.Inferences) > 0 {
			// check for losses. If they exist, then break so we can query
			_, err = qs.k.GetNetworkLossBundleAtBlock(ctx, req.TopicId, previousLossBlockHeight)
			if err == nil {
				break
			}
		}

		inferenceBlockHeight -= topic.EpochLength
		previousLossBlockHeight -= topic.EpochLength
	}

	networkInferences, forecastImpliedInferenceByWorker, infererWeights, forecasterWeights, err :=
		synth.GetNetworkInferencesAtBlock(
			sdk.UnwrapSDKContext(ctx),
			qs.k,
			req.TopicId,
			inferenceBlockHeight,
			previousLossBlockHeight,
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

	return &types.QueryLatestNetworkInferencesResponse{
		NetworkInferences:                networkInferences,
		InfererWeights:                   synth.ConvertWeightsToArrays(infererWeights),
		ForecasterWeights:                synth.ConvertWeightsToArrays(forecasterWeights),
		ForecastImpliedInferences:        synth.ConvertForecastImpliedInferencesToArrays(forecastImpliedInferenceByWorker),
		InferenceBlockHeight:             inferenceBlockHeight,
		LossBlockHeight:                  previousLossBlockHeight,
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

	inferersToInference := make(map[string]float64)
	for _, infererValue := range networkInferences.InfererValues {
		floatValue, err := stringToFloat64(infererValue.Value.String())
		if err != nil {
			return nil, nil, err
		}
		inferersToInference[infererValue.Worker] = floatValue
	}

	forecastersToInference := make(map[string]float64)
	for _, forecasterValue := range networkInferences.ForecasterValues {
		floatValue, err := stringToFloat64(forecasterValue.Value.String())
		if err != nil {
			return nil, nil, err
		}
		forecastersToInference[forecasterValue.Worker] = floatValue
	}

	inferersToWeights := make(map[string]float64)
	for worker, weight := range infererWeights {
		floatValue, err := stringToFloat64(weight.String())
		if err != nil {
			return nil, nil, err
		}
		inferersToWeights[worker] = floatValue
	}

	forecastersToWeights := make(map[string]float64)
	for worker, weight := range forecasterWeights {
		floatValue, err := stringToFloat64(weight.String())
		if err != nil {
			return nil, nil, err
		}
		forecastersToWeights[worker] = floatValue
	}

	var inferences []float64
	var weights []float64

	for inferer, inference := range inferersToInference {
		weight, exists := inferersToWeights[inferer]
		if exists {
			inferences = append(inferences, inference)
			weights = append(weights, weight)
		}
	}

	for forecaster, inference := range forecastersToInference {
		weight, exists := forecastersToWeights[forecaster]
		if exists {
			inferences = append(inferences, inference)
			weights = append(weights, weight)
		}
	}

	ciRawPercentiles := []float64{2.28, 15.87, 50, 84.13, 97.72}

	var ciValues []float64
	var err error
	if len(inferences) == 0 {
		ciValues = []float64{}
		ciRawPercentiles = []float64{}
	} else {
		ciValues, err = synth.WeightedPercentile(inferences, weights, ciRawPercentiles)
		if err != nil {
			return nil, nil, err
		}
	}

	ciValuesAlloraMath := make([]alloraMath.Dec, len(ciValues))
	for i, value := range ciValues {
		ciValuesAlloraMath[i] = alloraMath.MustNewDecFromString(float64ToString(value))
	}

	ciRawPercentilesAlloraMath := make([]alloraMath.Dec, len(ciRawPercentiles))
	for i, value := range ciRawPercentiles {
		ciRawPercentilesAlloraMath[i] = alloraMath.MustNewDecFromString(float64ToString(value))
	}

	return ciRawPercentilesAlloraMath, ciValuesAlloraMath, nil
}

func stringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func float64ToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
