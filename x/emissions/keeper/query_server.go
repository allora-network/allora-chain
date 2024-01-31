package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	state "github.com/upshot-tech/protocol-state-machine-module"
)

var _ state.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k Keeper) state.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) Params(ctx context.Context, req *state.QueryParamsRequest) (*state.QueryParamsResponse, error) {
	params, err := qs.k.params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &state.QueryParamsResponse{Params: state.Params{}}, nil
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryParamsResponse{Params: params}, nil
}

// last_rewards_calc_update
func (qs queryServer) GetLastRewardsUpdate(ctx context.Context, req *state.QueryLastRewardsUpdateRequest) (*state.QueryLastRewardsUpdateResponse, error) {
	lastRewardsUpdate, err := qs.k.lastRewardsUpdate.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryLastRewardsUpdateResponse{LastRewardsUpdate: lastRewardsUpdate}, nil
}

// TotalStake defines the handler for the Query/TotalStake RPC method.
func (qs queryServer) GetTotalStake(ctx context.Context, req *state.QueryTotalStakeRequest) (*state.QueryTotalStakeResponse, error) {
	totalStake, err := qs.k.totalStake.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryTotalStakeResponse{Amount: totalStake}, nil
}

// Get the amount of token rewards that have accumulated this epoch
func (qs queryServer) GetAccumulatedEpochRewards(ctx context.Context, req *state.QueryAccumulatedEpochRewardsRequest) (*state.QueryAccumulatedEpochRewardsResponse, error) {
	emissions, err := qs.k.CalculateAccumulatedEmissions(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryAccumulatedEpochRewardsResponse{Amount: emissions}, nil
}

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *state.QueryNextTopicIdRequest) (*state.QueryNextTopicIdResponse, error) {
	nextTopicId, err := qs.k.nextTopicId.Peek(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Query/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *state.QueryTopicRequest) (*state.QueryTopicResponse, error) {
	topic, err := qs.k.topics.Get(ctx, req.TopicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &state.QueryTopicResponse{Topic: &topic}, nil
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryTopicResponse{Topic: &topic}, nil
}

// GetWeight find out how much weight the reputer has placed upon the worker for a given topid ID, reputer and worker.
func (qs queryServer) GetWeight(ctx context.Context, req *state.QueryWeightRequest) (*state.QueryWeightResponse, error) {
	reputerAddr := sdk.AccAddress(req.Reputer)
	workerAddr := sdk.AccAddress(req.Worker)

	key := collections.Join3(req.TopicId, reputerAddr, workerAddr)
	weight, err := qs.k.weights.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return &state.QueryWeightResponse{Amount: weight}, nil
}

// GetInference retrieves the inference value for a given topic ID and worker address.
func (qs queryServer) GetInference(ctx context.Context, req *state.QueryInferenceRequest) (*state.QueryInferenceResponse, error) {
	// TODO: Implement
	return &state.QueryInferenceResponse{}, nil
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

func (qs queryServer) GetAllInferences(ctx context.Context, req *state.QueryInferenceRequest) (*state.QueryInferenceResponse, error) {
	// Defers implementation to the function in the Keeper
	topicId := req.TopicId
	timestamp := req.Timestamp
	inferences, err := qs.k.GetAllInferences(ctx, topicId, timestamp)
	if err != nil {
		return nil, err
	}

	return &state.QueryInferenceResponse{Inferences: inferences}, nil
}

func (qs queryServer) GetWorkerNodeRegistration(ctx context.Context, req *state.QueryRegisteredWorkerNodesRequest) (*state.QueryRegisteredWorkerNodesResponse, error) {
    if req == nil {
        return nil, fmt.Errorf("received nil request")
    }

    nodes, err := qs.k.FindWorkerNodesByOwner(ctx.(sdk.Context), req.NodeId)
    if err != nil {
        return nil, err
    }

    // Prepare and return the response
    return &state.QueryRegisteredWorkerNodesResponse{Nodes: nodes}, nil
}