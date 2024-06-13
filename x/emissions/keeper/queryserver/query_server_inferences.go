package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetWorkerLatestInferenceByTopicId handles the query for the latest inference by a specific worker for a given topic.
func (qs queryServer) GetWorkerLatestInferenceByTopicId(ctx context.Context, req *types.QueryWorkerLatestInferenceRequest) (*types.QueryWorkerLatestInferenceResponse, error) {
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

	inferences, err := qs.k.GetInferencesAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryInferencesAtBlockResponse{Inferences: inferences}, nil
}

// Return full set of inferences in I_i from the chain
func (qs queryServer) GetNetworkInferencesAtBlock(ctx context.Context, req *types.QueryNetworkInferencesAtBlockRequest) (*types.QueryNetworkInferencesAtBlockResponse, error) {
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
	req *types.QueryLatestNetworkInferencesAtBlockRequest,
) (
	*types.QueryLatestNetworkInferencesAtBlockResponse,
	error,
) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	}
	if topic.EpochLastEnded == 0 {
		return nil, status.Errorf(codes.NotFound, "network inference not available for topic %v", req.TopicId)
	}

	currentBlockHeight := sdkCtx.BlockHeight()
	reputerNonces, err := qs.k.GetUnfulfilledReputerNonces(ctx, topic.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no reputer nonces found for topic %v", topic.Id)
	}

	// No filtering - reputation of previous rounds can still be retried if work has been done.
	emissionsParams, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting max num of retries to fulfil nonces: %s", err.Error())
	}

	maxRetriesToFulfilNoncesReputer := emissionsParams.MaxRetriesToFulfilNoncesReputer

	topNReputerNonces := synth.SelectTopNReputerNonces(&reputerNonces, int(maxRetriesToFulfilNoncesReputer), currentBlockHeight, topic.GroundTruthLag)

	if len(topNReputerNonces) == 0 {
		return nil, status.Errorf(codes.NotFound, "no reputer nonces found for topic %v", topic.Id)
	}

	maxReputerNonceBlockHeight := int64(0)
	workerNonceBlockHeight := int64(0)

	for _, nonce := range topNReputerNonces {
		nonceCopy := nonce
		if nonceCopy.ReputerNonce.BlockHeight > maxReputerNonceBlockHeight {
			maxReputerNonceBlockHeight = nonceCopy.ReputerNonce.BlockHeight
			workerNonceBlockHeight = nonceCopy.WorkerNonce.BlockHeight
		}
	}

	networkInferences, forecastImpliedInferenceByWorker, infererWeights, forecasterWeights, err := synth.GetNetworkInferencesAtBlock(
		sdk.UnwrapSDKContext(ctx),
		qs.k,
		req.TopicId,
		maxReputerNonceBlockHeight,
		workerNonceBlockHeight,
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestNetworkInferencesAtBlockResponse{
		NetworkInferences:         networkInferences,
		InfererWeights:            qs.ConvertWeightsToArrays(infererWeights),
		ForecasterWeights:         qs.ConvertWeightsToArrays(forecasterWeights),
		ForecastImpliedInferences: qs.ConvertForecastImpliedInferencesToArrays(forecastImpliedInferenceByWorker),
	}, nil
}

func (qs queryServer) ConvertWeightsToArrays(weights map[inference_synthesis.Worker]inference_synthesis.Weight) []*types.RegretInformedWeight {
	weightsArray := make([]*types.RegretInformedWeight, 0)
	for worker, weight := range weights {
		weightsArray = append(weightsArray, &types.RegretInformedWeight{Worker: worker, Weight: weight})
	}
	return weightsArray
}

func (qs queryServer) ConvertForecastImpliedInferencesToArrays(
	forecastImpliedInferenceByWorker map[string]*types.Inference,
) []*types.WorkerAttributedValue {
	forecastImpliedInferences := make([]*types.WorkerAttributedValue, 0)
	for worker, inference := range forecastImpliedInferenceByWorker {
		forecastImpliedInferences = append(forecastImpliedInferences, &types.WorkerAttributedValue{Worker: worker, Value: inference.Value})
	}
	return forecastImpliedInferences
}
