package msgserver

import (
	"context"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"math/big"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *types.MsgCreateNewTopic) (*types.MsgCreateNewTopicResponse, error) {
	// Check if the sender is in the topic creation whitelist
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	isTopicCreator, err := ms.k.IsInTopicCreationWhitelist(ctx, creator)
	if err != nil {
		return nil, err
	}
	if !isTopicCreator {
		return nil, types.ErrNotInTopicCreationWhitelist
	}

	hasEnoughBal, fee, _ := ms.CheckBalanceForTopic(ctx, creator)
	if !hasEnoughBal {
		return nil, types.ErrTopicCreatorNotEnoughDenom
	}

	id, err := ms.k.GetNumTopics(ctx)
	if err != nil {
		return nil, err
	}

	fastestCadence, err := ms.k.GetParamsMinEpochLength(ctx)
	if err != nil {
		return nil, err
	}
	if msg.EpochLength < fastestCadence {
		return nil, types.ErrTopicCadenceBelowMinimum
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(ctx, creator, types.AlloraStakingAccountName, sdk.NewCoins(fee))
	if err != nil {
		return nil, err
	}

	topic := types.Topic{
		Id:               id,
		Creator:          creator.String(),
		Metadata:         msg.Metadata,
		LossLogic:        msg.LossLogic,
		LossMethod:       msg.LossMethod,
		InferenceLogic:   msg.InferenceLogic,
		InferenceMethod:  msg.InferenceMethod,
		EpochLastEnded:   0,
		EpochLength:      msg.EpochLength,
		GroundTruthLag:   msg.GroundTruthLag,
		Active:           true,
		DefaultArg:       msg.DefaultArg,
		Pnorm:            msg.Pnorm,
		AlphaRegret:      msg.AlphaRegret,
		PrewardReputer:   msg.PrewardReputer,
		PrewardInference: msg.PrewardInference,
		PrewardForecast:  msg.PrewardForecast,
		FTolerance:       msg.FTolerance,
	}
	_, err = ms.k.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.k.SetTopic(ctx, id, topic); err != nil {
		return nil, err
	}
	// Rather than set latest weight-adjustment timestamp of a topic to 0
	// we do nothing, since no value in the map means zero

	return &types.MsgCreateNewTopicResponse{TopicId: id}, nil
}

func (ms msgServer) ReactivateTopic(ctx context.Context, msg *types.MsgReactivateTopic) (*types.MsgReactivateTopicResponse, error) {
	// Check that the topic has enough demand to be reactivated
	unmetDemand, err := ms.k.GetTopicUnmetDemand(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}

	minTopicUnmentDemand, err := ms.k.GetParamsMinTopicUnmetDemand(ctx)
	if err != nil {
		return nil, err
	}
	// If the topic does not have enough demand, return an error
	if unmetDemand.LT(minTopicUnmentDemand) {
		return nil, types.ErrTopicNotEnoughDemand
	}

	// If the topic has enough demand, reactivate it
	err = ms.k.ReactivateTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.MsgReactivateTopicResponse{Success: true}, nil
}

func (ms msgServer) CheckBalanceForTopic(ctx context.Context, address sdk.AccAddress) (bool, sdk.Coin, error) {
	amountInt := cosmosMath.NewIntFromBigInt(big.NewInt(int64(types.DefaultParamsCreateTopicFee())))
	fee := sdk.NewCoin(params.DefaultBondDenom, amountInt)
	balance := ms.k.BankKeeper().GetBalance(ctx, address, fee.Denom)
	return balance.IsGTE(fee), fee, nil
}
