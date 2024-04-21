package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetMempoolInferenceRequest(ctx context.Context, req *types.QueryMempoolInferenceRequest) (*types.QueryExistingInferenceResponse, error) {
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

func (qs queryServer) GetMempoolInferenceRequestsByTopic(ctx context.Context, req *types.QueryMempoolInferenceRequestsByTopic) (*types.QueryMempoolInferenceRequestsByTopicResponse, error) {
	ret := make([]*types.InferenceRequestAndDemandLeft, 0)
	mempool, pageRes, err := qs.k.GetMempoolInferenceRequestsForTopic(ctx, req.TopicId, req.Pagination)
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
	return &types.QueryMempoolInferenceRequestsByTopicResponse{InferenceRequests: ret, Pagination: pageRes}, nil
}
