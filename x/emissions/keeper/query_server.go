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

func (qs queryServer) GetAllTopics(ctx context.Context, req *state.QueryAllTopicsRequest) (*state.QueryAllTopicsResponse, error) {
	topics, err := qs.k.GetAllTopics(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryAllTopicsResponse{Topics: topics}, nil
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
	reputerAddr, err := sdk.AccAddressFromBech32(req.Reputer)
	if err != nil {
		return nil, err
	}
	workerAddr, err := sdk.AccAddressFromBech32(req.Worker)
	if err != nil {
		return nil, err
	}

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
func (qs queryServer) GetInference(ctx context.Context, req *state.QueryAllInferencesRequest) (*state.QueryAllInferencesResponse, error) {
	// TODO: Implement
	return &state.QueryAllInferencesResponse{}, nil
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

func (qs queryServer) GetRegisteredTopicIds(ctx context.Context, req *state.QueryRegisteredTopicIdsRequest) (*state.QueryRegisteredTopicIdsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	var TopicIds []uint64
	if req.IsReputer {
		TopicIds, err = qs.k.GetRegisteredTopicIdByReputerAddress(ctx.(sdk.Context), address)
		if err != nil {
			return nil, err
		}
	} else {
		TopicIds, err = qs.k.GetRegisteredTopicIdsByWorkerAddress(ctx.(sdk.Context), address)
		if err != nil {
			return nil, err
		}
	}

	return &state.QueryRegisteredTopicIdsResponse{TopicIds: TopicIds}, nil
}

func (qs queryServer) GetExistingInferenceRequest(ctx context.Context, req *state.QueryExistingInferenceRequest) (*state.QueryExistingInferenceResponse, error) {
	valid := state.IsValidRequestId(req.RequestId)
	if !valid {
		return nil, state.ErrInvalidRequestId
	}
	inMempool, err := qs.k.IsRequestInMempool(ctx, req.TopicId, req.RequestId)
	if err != nil {
		return nil, err
	}
	if !inMempool {
		return nil, state.ErrInferenceRequestNotInMempool
	}
	inferenceRequest, err := qs.k.GetMempoolInferenceRequestById(ctx, req.TopicId, req.RequestId)
	if err != nil {
		return nil, err
	}
	demandLeft, err := qs.k.GetRequestDemand(ctx, req.RequestId)
	if err != nil {
		return nil, err
	}
	return &state.QueryExistingInferenceResponse{InferenceRequest: &inferenceRequest, DemandLeft: demandLeft}, nil
}

func (qs queryServer) GetAllExistingInferenceRequests(ctx context.Context, req *state.QueryAllExistingInferenceRequest) (*state.QueryAllExistingInferenceResponse, error) {
	ret := make([]*state.InferenceRequestAndDemandLeft, 0)
	mempool, err := qs.k.GetMempool(ctx)
	if err != nil {
		return nil, err
	}
	for _, inferenceRequest := range mempool {
		reqId, err := inferenceRequest.GetRequestId()
		if err != nil {
			return nil, err
		}
		demandLeft, err := qs.k.GetRequestDemand(ctx, reqId)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &state.InferenceRequestAndDemandLeft{InferenceRequest: &inferenceRequest, DemandLeft: demandLeft})
	}
	return &state.QueryAllExistingInferenceResponse{InferenceRequests: ret}, nil
}

func (qs queryServer) GetTopicUnmetDemand(ctx context.Context, req *state.QueryTopicUnmetDemandRequest) (*state.QueryTopicUnmetDemandResponse, error) {
	unmetDemand, err := qs.k.GetTopicUnmetDemand(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &state.QueryTopicUnmetDemandResponse{DemandLeft: unmetDemand}, nil
}
