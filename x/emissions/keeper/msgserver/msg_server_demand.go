package msgserver

import (
	"context"

	appParams "github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ms msgServer) FundTopic(ctx context.Context, msg *types.MsgFundTopic) (*types.MsgFundTopicResponse, error) {
	// Check the topic is valid
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	if !topicExists {
		return nil, types.ErrInvalidTopicId
	}

	// Check that the request isn't spam by checking that the amount of funds it bids is greater than a global minimum demand per request
	epsilon, err := ms.k.GetParamsEpsilon(ctx)
	if err != nil {
		return nil, err
	}
	amountDec, err := alloraMath.NewDecFromSdkInt(msg.Amount)
	if err != nil {
		return nil, err
	}
	if amountDec.Lte(epsilon) {
		return nil, types.ErrFundAmountTooLow
	}
	// Check sender has funds to pay for the inference request
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send funds
	coins := sdk.NewCoins(sdk.NewCoin(appParams.DefaultBondDenom, msg.Amount))
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, types.AlloraRequestsAccountName, coins)
	if err != nil {
		return nil, err
	}

	// Account for the revenue the topic has generated
	err = ms.k.AddTopicFeeRevenue(ctx, msg.TopicId, msg.Amount)
	if err != nil {
		return nil, err
	}

	// Activate topic if it exhibits minimum weight
	err = ms.ActivateTopicIfWeightAtLeastGlobalMin(ctx, msg.TopicId, msg.Amount)
	return &types.MsgFundTopicResponse{}, err
}
