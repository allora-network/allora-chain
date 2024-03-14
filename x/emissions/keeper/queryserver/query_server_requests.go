package queryserver

import (
	"context"

	state "github.com/allora-network/allora-chain/x/emissions"
)

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
		inferenceRequestCopy := inferenceRequest
		ret = append(ret, &state.InferenceRequestAndDemandLeft{InferenceRequest: &inferenceRequestCopy, DemandLeft: demandLeft})
	}
	return &state.QueryAllExistingInferenceResponse{InferenceRequests: ret}, nil
}

