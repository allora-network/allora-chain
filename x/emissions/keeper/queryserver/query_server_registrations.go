package queryserver

import (
	"context"
	"fmt"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
