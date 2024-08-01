package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (ms msgServer) FundTopic(ctx context.Context, msg *types.MsgFundTopic) (*types.MsgFundTopicResponse, error) {
	// Check the topic is valid
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", msg.TopicId)
	} else if err != nil {
		return nil, err
	}

	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, msg.TopicId, msg.Amount, "fund topic")
	return &types.MsgFundTopicResponse{}, err
}
