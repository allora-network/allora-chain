package msgserver

import (
	"context"
	"strconv"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

///
/// TOPICS
///

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
		Id:                     id,
		Creator:                creator.String(),
		Metadata:               msg.Metadata,
		LossLogic:              msg.LossLogic,
		LossMethod:             msg.LossMethod,
		InferenceLogic:         msg.InferenceLogic,
		InferenceMethod:        msg.InferenceMethod,
		EpochLastEnded:         0,
		EpochLength:            msg.EpochLength,
		GroundTruthLag:         msg.GroundTruthLag,
		Active:                 true,
		DefaultArg:             msg.DefaultArg,
		Pnorm:                  msg.Pnorm,
		AlphaRegret:            float32(alphaRegret),
		PrewardReputer:         float32(prewardReputer),
		PrewardInference:       float32(prewardInference),
		PrewardForecast:        float32(prewardForecast),
		FTolerance:             float32(fTolerance),
		Subsidy:                0,   // Can later be updated by a Foundation member
		SubsidizedRewardEpochs: 0,   // Can later be updated by a Foundation member
		FTreasury:              0.5, // Can later be updated by a Foundation member
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

///
/// FOUNDATION TOPIC MANAGEMENT
///

// Modfies topic subsidy and subsidized_reward_epochs properties
// Can only be called by a whitelisted foundation member
func (ms msgServer) ModifyTopicSubsidy(ctx context.Context, msg *types.MsgModifyTopicSubsidyAndSubsidizedRewardEpochs) (*types.MsgModifyTopicSubsidyAndSubsidizedRewardEpochsResponse, error) {
	// Check that sender is in the foundation whitelist
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsInFoundationWhitelist(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	// Modify the topic subsidy + F_treasury
	err = ms.k.SetTopicSubsidy(ctx, msg.TopicId, msg.Subsidy)
	if err != nil {
		return nil, err
	}
	err = ms.k.SetTopicSubsidizedRewardEpochs(ctx, msg.TopicId, msg.SubsidizedRewardEpochs)
	if err != nil {
		return nil, err
	}
	return &types.MsgModifyTopicSubsidyAndSubsidizedRewardEpochsResponse{Success: true}, nil
}

func (ms msgServer) ModifyTopicFTreasury(ctx context.Context, msg *types.MsgModifyTopicFTreasury) (*types.MsgModifyTopicFTreasuryResponse, error) {
	// Check that sender is in the foundation whitelist
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsInFoundationWhitelist(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	// Modify the topic subsidy + F_treasury
	err = ms.k.SetTopicFTreasury(ctx, msg.TopicId, float32(msg.FTreasury))
	if err != nil {
		return nil, err
	}
	return &types.MsgModifyTopicFTreasuryResponse{Success: true}, nil
}
