package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	synth "github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetWorkerLatestInferenceByTopicId handles the query for the latest inference by a specific worker for a given topic.
func (qs queryServer) GetWorkerLatestInferenceByTopicId(ctx context.Context, req *types.QueryWorkerLatestInferenceRequest) (*types.QueryWorkerLatestInferenceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	workerAddr, err := sdk.AccAddressFromBech32(req.WorkerAddress)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "worker with address %s not found", req.WorkerAddress)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	inference, err := qs.k.GetWorkerLatestInferenceByTopicId(ctx, req.TopicId, workerAddr)
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

func (qs queryServer) GetNetworkInferencesAtBlock(ctx context.Context, req *types.QueryNetworkInferencesAtBlockRequest) (*types.QueryNetworkInferencesAtBlockResponse, error) {
	epsilon, err := qs.k.GetParamsEpsilon(ctx)
	if err != nil {
		return nil, err
	}

	pInferenceSynthesis, err := qs.k.GetParamsPInferenceSynthesis(ctx)
	if err != nil {
		return nil, err
	}

	stakesOnTopic, err := qs.k.GetStakePlacementsByTopic(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	// Map list of stakesOnTopic to map of stakesByReputer
	stakesByReputer := make(map[string]types.StakePlacement)
	for _, stake := range stakesOnTopic {
		stakesByReputer[stake.Reputer] = stake
	}

	reputerReportedLosses, _, err := qs.k.GetReputerReportedLossesAtOrBeforeBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	networkCombinedLoss, err := synth.CalcCombinedNetworkLoss(stakesByReputer, reputerReportedLosses, epsilon)
	if err != nil {
		return nil, err
	}

	inferences, blockHeight, err := qs.k.GetInferencesAtOrAfterBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}
	forecasts, _, err := qs.k.GetForecastsAtOrAfterBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	networkInferences, err := synth.CalcNetworkInferences(ctx.(sdk.Context), qs.k, req.TopicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	if err != nil {
		return nil, err
	}

	return &types.QueryNetworkInferencesAtBlockResponse{NetworkInferences: networkInferences, BlockHeight: blockHeight}, nil
}
