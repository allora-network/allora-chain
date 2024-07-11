package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetPreviousReputerRewardFraction(
	ctx context.Context,
	req *types.QueryPreviousReputerRewardFractionRequest,
) (
	*types.QueryPreviousReputerRewardFractionResponse,
	error,
) {
	rewardFraction, notFound, err := qs.k.GetPreviousReputerRewardFraction(ctx, req.TopicId, req.Reputer)
	if err != nil {
		return nil, err
	}

	return &types.QueryPreviousReputerRewardFractionResponse{RewardFraction: rewardFraction, NotFound: notFound}, nil
}
