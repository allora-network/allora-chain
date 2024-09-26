package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetCountInfererInclusionsInTopic(
	ctx context.Context,
	req *types.GetCountInfererInclusionsInTopicRequest,
) (
	*types.GetCountInfererInclusionsInTopicResponse,
	error,
) {
	count, err := qs.k.GetCountInfererInclusionsInTopic(ctx, req.TopicId, req.Inferer)
	return &types.GetCountInfererInclusionsInTopicResponse{Count: count}, err
}

func (qs queryServer) GetCountForecasterInclusionsInTopic(
	ctx context.Context,
	req *types.GetCountForecasterInclusionsInTopicRequest,
) (
	*types.GetCountForecasterInclusionsInTopicResponse,
	error,
) {
	count, err := qs.k.GetCountForecasterInclusionsInTopic(ctx, req.TopicId, req.Forecaster)
	return &types.GetCountForecasterInclusionsInTopicResponse{Count: count}, err
}
