package msgserver

import (
	"context"

	appParams "github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	// Check sender has funds to pay for the inference request
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send funds
	coins := sdk.NewCoins(sdk.NewCoin(appParams.DefaultBondDenom, msg.Amount))
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, minttypes.EcosystemModuleName, coins)
	if err != nil {
		return nil, err
	}

	// Account for the revenue the topic has generated
	err = ms.k.AddTopicFeeRevenue(ctx, msg.TopicId, msg.Amount)
	if err != nil {
		return nil, err
	}

	// Activate topic if it exhibits minimum weight
	err = activateTopicIfWeightAtLeastGlobalMin(ctx, ms, msg.TopicId, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgFundTopicResponse{}, nil
}
