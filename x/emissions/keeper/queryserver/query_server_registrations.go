package queryserver

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (qs queryServer) GetWorkerNodeInfo(ctx context.Context, req *types.QueryWorkerNodeInfoRequest) (*types.QueryWorkerNodeInfoResponse, error) {
	node, err := qs.k.GetWorkerByLibp2pKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerNodeInfoResponse{NodeInfo: &node}, nil
}

func (qs queryServer) GetReputerNodeInfo(ctx context.Context, req *types.QueryReputerNodeInfoRequest) (*types.QueryReputerNodeInfoResponse, error) {
	node, err := qs.k.GetReputerByLibp2pKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputerNodeInfoResponse{NodeInfo: &node}, nil
}

func (qs queryServer) GetWorkerAddressByP2PKey(ctx context.Context, req *types.QueryWorkerAddressByP2PKeyRequest) (*types.QueryWorkerAddressByP2PKeyResponse, error) {
	workerAddr, err := qs.k.GetWorkerAddressByP2PKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryWorkerAddressByP2PKeyResponse{Address: workerAddr.String()}, nil
}

func (qs queryServer) GetReputerAddressByP2PKey(ctx context.Context, req *types.QueryReputerAddressByP2PKeyRequest) (*types.QueryReputerAddressByP2PKeyResponse, error) {
	address, err := qs.k.GetReputerAddressByP2PKey(sdk.UnwrapSDKContext(ctx), req.Libp2PKey)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputerAddressByP2PKeyResponse{Address: address.String()}, nil
}

func (qs queryServer) IsWorkerRegisteredInTopicId(ctx context.Context, req *types.QueryIsWorkerRegisteredInTopicIdRequest) (*types.QueryIsWorkerRegisteredInTopicIdResponse, error) {
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	isRegistered, err := qs.k.IsWorkerRegisteredInTopic(sdk.UnwrapSDKContext(ctx), req.TopicId, req.Address)
	if err != nil {
		return nil, err
	}

	return &types.QueryIsWorkerRegisteredInTopicIdResponse{IsRegistered: isRegistered}, nil
}

func (qs queryServer) IsReputerRegisteredInTopicId(ctx context.Context, req *types.QueryIsReputerRegisteredInTopicIdRequest) (*types.QueryIsReputerRegisteredInTopicIdResponse, error) {
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, err
	}
	isRegistered, err := qs.k.IsReputerRegisteredInTopic(sdk.UnwrapSDKContext(ctx), req.TopicId, req.Address)
	if err != nil {
		return nil, err
	}

	return &types.QueryIsReputerRegisteredInTopicIdResponse{IsRegistered: isRegistered}, nil
}
