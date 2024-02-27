package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryParamsResponse{Params: params}, nil
}

// Get timestamp of the last rewards update
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

// GetActiveTopics retrieves a list of active topics.
func (qs queryServer) GetActiveTopics(ctx context.Context, req *state.QueryActiveTopicsRequest) (*state.QueryActiveTopicsResponse, error) {
	activeTopics, err := qs.k.GetActiveTopics(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryActiveTopicsResponse{Topics: activeTopics}, nil
}

// GetTopicsByCreator retrieves a list of topics created by a given address.
func (qs queryServer) GetTopicsByCreator(ctx context.Context, req *state.QueryGetTopicsByCreatorRequest) (*state.QueryGetTopicsByCreatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	topics, err := qs.k.GetTopicsByCreator(ctx, req.Creator)
	if err != nil {
		return nil, err
	}

	return &state.QueryGetTopicsByCreatorResponse{Topics: topics}, nil
}

// GetAccountStakeList retrieves a list of stakes for a given account address.
func (qs queryServer) GetAccountStakeList(ctx context.Context, req *state.QueryAccountStakeListRequest) (*state.QueryAccountStakeListResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	stakes, err := qs.k.GetStakesForAccount(ctx, address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryAccountStakeListResponse{Stakes: stakes}, nil
}

// GetWeight find out how much weight the reputer has placed upon the worker for a given topid ID, reputer and worker.
func (qs queryServer) GetWeight(ctx context.Context, req *state.QueryWeightRequest) (*state.QueryWeightResponse, error) {
	reputerAddr := sdk.AccAddress(req.Reputer)
	workerAddr := sdk.AccAddress(req.Worker)

	key := collections.Join3(req.TopicId, reputerAddr, workerAddr)
	weight, err := qs.k.weights.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &state.QueryWeightResponse{Amount: cosmosMath.ZeroUint()}, nil
		}
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

	return &state.QueryRegisteredWorkerNodesResponse{Nodes: nodes}, nil
}

func (qs queryServer) GetWorkerAddressByP2PKey(ctx context.Context, req *state.QueryWorkerAddressByP2PKeyRequest) (*state.QueryWorkerAddressByP2PKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	workerAddr, err := qs.k.GetWorkerAddressByP2PKey(ctx.(sdk.Context), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &state.QueryWorkerAddressByP2PKeyResponse{Address: workerAddr.String()}, nil
}

func (qs queryServer) GetRegisteredTopicsIds(ctx context.Context, req *state.QueryRegisteredTopicsIdsRequest) (*state.QueryRegisteredTopicsIdsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	var topicsIds []uint64
	if req.IsReputer {
		topicsIds, err = qs.k.GetRegisteredTopicsIdsByReputerAddress(ctx.(sdk.Context), address)
		if err != nil {
			return nil, err
		}
	} else {
		topicsIds, err = qs.k.GetRegisteredTopicsIdsByWorkerAddress(ctx.(sdk.Context), address)
		if err != nil {
			return nil, err
		}
	}

	return &state.QueryRegisteredTopicsIdsResponse{TopicsIds: topicsIds}, nil
}
