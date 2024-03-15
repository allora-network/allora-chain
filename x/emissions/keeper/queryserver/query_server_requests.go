package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetExistingInferenceRequest(ctx context.Context, req *types.QueryExistingInferenceRequest) (*types.QueryExistingInferenceResponse, error) {
	valid := types.IsValidRequestId(req.RequestId)
	if !valid {
		return nil, types.ErrInvalidRequestId
	}
	inMempool, err := qs.k.IsRequestInMempool(ctx, req.TopicId, req.RequestId)
	if err != nil {
		return nil, err
	}
	if !inMempool {
		return nil, types.ErrInferenceRequestNotInMempool
	}
	inferenceRequest, err := qs.k.GetMempoolInferenceRequestById(ctx, req.TopicId, req.RequestId)
	if err != nil {
		return nil, err
	}
	demandLeft, err := qs.k.GetRequestDemand(ctx, req.RequestId)
	if err != nil {
		return nil, err
	}
	return &types.QueryExistingInferenceResponse{InferenceRequest: &inferenceRequest, DemandLeft: demandLeft}, nil
}

func (qs queryServer) GetAllExistingInferenceRequests(ctx context.Context, req *types.QueryAllExistingInferenceRequest) (*types.QueryAllExistingInferenceResponse, error) {
	ret := make([]*types.InferenceRequestAndDemandLeft, 0)
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
		inferenceRequestCopy := inferenceRequest
		ret = append(ret, &types.InferenceRequestAndDemandLeft{InferenceRequest: &inferenceRequestCopy, DemandLeft: demandLeft})
	}
	return &types.QueryAllExistingInferenceResponse{InferenceRequests: ret}, nil
}

