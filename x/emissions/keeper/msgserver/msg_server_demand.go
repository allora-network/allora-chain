package msgserver

import (
	"context"
	"time"

	"cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// FundTopic adds funds to a topic.
func (ms msgServer) FundTopic(ctx context.Context, msg *types.FundTopicRequest) (_ *types.FundTopicResponse, err error) {
	defer metrics.RecordMetrics("FundTopic", time.Now(), &err)

	if err := types.ValidateSdkIntRepresentingMonetaryValue(msg.Amount); err != nil {
		return nil, errors.Wrap(err, "amount is not valid")
	}

	err = types.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	topicExists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", msg.TopicId)
	} else if err != nil {
		return nil, err
	}

	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, msg.TopicId, msg.Amount)
	if err != nil {
		return nil, err
	}

	types.EmitNewFundTopicEvent(ctx, msg.TopicId, msg.Sender, msg.Amount)
	return &types.FundTopicResponse{}, err
}
