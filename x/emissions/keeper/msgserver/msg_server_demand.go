package msgserver

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (ms msgServer) FundTopic(ctx context.Context, msg *types.FundTopicRequest) (*types.FundTopicResponse, error) {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	// Check the topic is valid
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", msg.TopicId)
	} else if err != nil {
		return nil, err
	}

	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, msg.TopicId, msg.Amount)
	return &types.FundTopicResponse{}, err
}
