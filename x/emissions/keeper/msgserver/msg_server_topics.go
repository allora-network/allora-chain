package msgserver

import (
	"context"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"math/big"
	"strconv"

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

	amountInt := cosmosMath.NewIntFromBigInt(big.NewInt(int64(types.DefaultParamsCreateTopicFee())))
	feeAmount := sdk.NewCoin(params.DefaultBondDenom, amountInt)

	hasEnoughBal, err := ms.k.CheckEnoughDenom(ctx, creator, feeAmount)
	if err != nil {
		return nil, err
	}
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

	alphaRegret, err := strconv.ParseFloat(msg.AlphaRegret, 32)
	if err != nil {
		return nil, err
	}
	prewardReputer, err := strconv.ParseFloat(msg.PrewardReputer, 32)
	if err != nil {
		return nil, err
	}
	prewardInference, err := strconv.ParseFloat(msg.PrewardInference, 32)
	if err != nil {
		return nil, err
	}
	prewardForecast, err := strconv.ParseFloat(msg.PrewardForecast, 32)
	if err != nil {
		return nil, err
	}
	fTolerance, err := strconv.ParseFloat(msg.FTolerance, 32)
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
		AlphaRegret:      alphaRegret,
		PrewardReputer:   prewardReputer,
		PrewardInference: prewardInference,
		PrewardForecast:  prewardForecast,
		FTolerance:       fTolerance,
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

	// Transfer fee amount from creator to ecosystem bucket
	_ = ms.k.SendCoinsFromAccountToModule(ctx, creator, types.EcosystemModuleName, sdk.NewCoins(feeAmount))
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
