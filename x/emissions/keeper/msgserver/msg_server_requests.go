package msgserver

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
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
	amountDec := alloraMath.NewDecFromInt64(msg.Amount.Int64())
	if amountDec.Lte(epsilon) {
		return nil, types.ErrFundAmountTooLow
	}
	// Check sender has funds to pay for the inference request
	// bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
	// Send funds
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	amountInt := cosmosMath.NewIntFromBigInt(msg.Amount.BigInt())
	coins := sdk.NewCoins(sdk.NewCoin(appParams.DefaultBondDenom, amountInt))
	err = ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraRequestsAccountName, coins)
	if err != nil {
		return nil, err
	}

	// Account for the revenue the topic has generated
	err = ms.k.AddTopicFeeRevenue(ctx, msg.TopicId, msg.Amount)
	if err != nil {
		return nil, err
	}

	// Activate topic if it exhibits minimum weight
	isActivated, err := ms.k.IsTopicActive(ctx, msg.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic activation status")
	}
	if !isActivated {
		params, err := ms.k.GetParams(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "error getting params")
		}
		topic, err := ms.k.GetTopic(ctx, msg.TopicId)
		if err != nil {
			return nil, errors.Wrapf(err, "error getting topic")
		}

		newTopicWeight, _, err := ms.k.GetCurrentTopicWeight(
			ctx,
			msg.TopicId,
			topic.EpochLength,
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			msg.Amount,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "error getting current topic weight")
		}

		if newTopicWeight.Gte(params.MinTopicWeight) {
			err = ms.k.ActivateTopic(ctx, msg.TopicId)
			if err != nil {
				return nil, err
			}
		}
	}
	return &types.MsgFundTopicResponse{}, nil
}
