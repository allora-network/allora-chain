package queryserver

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetWorkerLatestInferenceByTopicId handles the query for the latest inference by a specific worker for a given topic.
func (qs queryServer) GetWorkerLatestInferenceByTopicId(ctx context.Context, req *state.QueryWorkerLatestInferenceRequest) (*state.QueryWorkerLatestInferenceResponse, error) {
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

	return &state.QueryWorkerLatestInferenceResponse{LatestInference: &inference}, nil
}

func (qs queryServer) GetInferencesToScore(ctx context.Context, req *state.QueryInferencesToScoreRequest) (*state.QueryInferencesToScoreResponse, error) {
	// Defers implementation to the function in the Keeper
	topicId := req.TopicId
	inferences, err := qs.k.GetLatestInferencesFromTopic(ctx, topicId)
	if err != nil {
		return nil, err
	}

	response := &state.QueryInferencesToScoreResponse{Inferences: inferences}
	return response, nil
}

func (qs queryServer) GetAllInferences(ctx context.Context, req *state.QueryAllInferencesRequest) (*state.QueryAllInferencesResponse, error) {
	// Defers implementation to the function in the Keeper
	topicId := req.TopicId
	timestamp := req.Timestamp
	inferences, err := qs.k.GetAllInferences(ctx, topicId, timestamp)
	if err != nil {
		return nil, err
	}

	return &state.QueryAllInferencesResponse{Inferences: inferences}, nil
}
