package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetInferenceScoresUntilBlock(ctx context.Context, req *types.QueryInferenceScoresUntilBlockRequest) (*types.QueryInferenceScoresUntilBlockResponse, error) {
	scores, err := qs.k.GetInferenceScoresUntilBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}
	return &types.QueryInferenceScoresUntilBlockRequest{Scores: scores}, nil
}
