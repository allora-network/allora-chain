package msgserver

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
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
	cosmosMathEpsilon := cosmosMath.NewUintFromString(epsilon.String())
	if msg.Amount.LT(cosmosMathEpsilon) {
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
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt))
	err = ms.k.SendCoinsFromAccountToModule(ctx, senderAddr, types.AlloraRequestsAccountName, coins)
	if err != nil {
		return nil, err
	}

	// Account for the revenue the topic has generated
	err = ms.k.AddTopicFeeRevenue(ctx, msg.TopicId, msg.Amount)
	if err != nil {
		return nil, err
	}

	// // Activate topic if it exhibits minimum unmet demand
	// isActivated, err := ms.k.IsTopicActive(ctx, msg.TopicId)
	// if err != nil {
	// 	return nil, err
	// }
	// if !isActivated {
	// 	minTopicUnmentDemand, err := ms.k.GetParamsMinTopicWeight(ctx)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	minTopicUnmetDemandUint := cosmosMath.NewUintFromString(minTopicUnmentDemand.String())
	// 	if newTopicUnmetDemand.GTE(minTopicUnmetDemandUint) {
	// 		err = ms.k.ActivateTopic(ctx, msg.TopicId)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }
	return &types.MsgFundTopicResponse{}, nil
}
