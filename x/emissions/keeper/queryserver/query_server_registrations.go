package queryserver

import (
	"context"
	"fmt"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO paginate
func (qs queryServer) GetWorkerNodeRegistration(ctx context.Context, req *types.QueryRegisteredWorkerNodesRequest) (*types.QueryRegisteredWorkerNodesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	nodes, err := qs.k.FindWorkerNodesByOwner(sdk.UnwrapSDKContext(ctx), req.NodeId)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegisteredWorkerNodesResponse{Nodes: nodes}, nil
}

func (qs queryServer) GetWorkerAddressByP2PKey(ctx context.Context, req *types.QueryWorkerAddressByP2PKeyRequest) (*types.QueryWorkerAddressByP2PKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	workerAddr, err := qs.k.GetWorkerAddressByP2PKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerAddressByP2PKeyResponse{Address: workerAddr.String()}, nil
}

func (qs queryServer) GetReputerAddressByP2PKey(ctx context.Context, req *types.QueryReputerAddressByP2PKeyRequest) (*types.QueryReputerAddressByP2PKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	address, err := qs.k.GetReputerAddressByP2PKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputerAddressByP2PKeyResponse{Address: address.String()}, nil
}

// TODO paginate
func (qs queryServer) GetRegisteredTopicIds(ctx context.Context, req *types.QueryRegisteredTopicIdsRequest) (*types.QueryRegisteredTopicIdsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("received nil request")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	var TopicIds []uint64
	if req.IsReputer {
		TopicIds, err = qs.k.GetRegisteredTopicIdByReputerAddress(sdk.UnwrapSDKContext(ctx), address)
		if err != nil {
			return nil, err
		}
	} else {
		TopicIds, err = qs.k.GetRegisteredTopicIdsByWorkerAddress(sdk.UnwrapSDKContext(ctx), address)
		if err != nil {
			return nil, err
		}
	}

	return &types.QueryRegisteredTopicIdsResponse{TopicIds: TopicIds}, nil
}
