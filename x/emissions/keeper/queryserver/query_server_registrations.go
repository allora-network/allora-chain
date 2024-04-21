package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (qs queryServer) GetWorkerNodeInfo(ctx context.Context, req *types.QueryWorkerNodeInfoRequest) (*types.QueryWorkerNodeInfoResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	node, err := qs.k.GetWorkerByLibp2pKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerNodeInfoResponse{NodeInfo: &node}, nil
}

func (qs queryServer) GetReputerNodeInfo(ctx context.Context, req *types.QueryReputerNodeInfoRequest) (*types.QueryReputerNodeInfoResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	node, err := qs.k.GetReputerByLibp2pKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputerNodeInfoResponse{NodeInfo: &node}, nil
}

func (qs queryServer) GetWorkerAddressByP2PKey(ctx context.Context, req *types.QueryWorkerAddressByP2PKeyRequest) (*types.QueryWorkerAddressByP2PKeyResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	workerAddr, err := qs.k.GetWorkerAddressByP2PKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerAddressByP2PKeyResponse{Address: workerAddr.String()}, nil
}

func (qs queryServer) GetReputerAddressByP2PKey(ctx context.Context, req *types.QueryReputerAddressByP2PKeyRequest) (*types.QueryReputerAddressByP2PKeyResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	address, err := qs.k.GetReputerAddressByP2PKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputerAddressByP2PKeyResponse{Address: address.String()}, nil
}

func (qs queryServer) IsWorkerRegisteredInTopicId(ctx context.Context, req *types.QueryIsWorkerRegisteredInTopicIdRequest) (*types.QueryIsWorkerRegisteredInTopicIdResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	isRegistered, err := qs.k.IsWorkerRegisteredInTopic(sdk.UnwrapSDKContext(ctx), req.TopicId, address)
	if err != nil {
		return nil, err
	}

	return &types.QueryIsWorkerRegisteredInTopicIdResponse{IsRegistered: isRegistered}, nil
}

func (qs queryServer) IsReputerRegisteredInTopicId(ctx context.Context, req *types.QueryIsReputerRegisteredInTopicIdRequest) (*types.QueryIsReputerRegisteredInTopicIdResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	isRegistered, err := qs.k.IsReputerRegisteredInTopic(sdk.UnwrapSDKContext(ctx), req.TopicId, address)
	if err != nil {
		return nil, err
	}

	return &types.QueryIsReputerRegisteredInTopicIdResponse{IsRegistered: isRegistered}, nil
}
