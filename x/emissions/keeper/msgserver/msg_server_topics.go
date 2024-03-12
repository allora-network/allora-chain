package msgserver

import (
	"context"
	"fmt"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

///
/// TOPICS
///

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *state.MsgCreateNewTopic) (*state.MsgCreateNewTopicResponse, error) {
	fmt.Println("CreateNewTopic called with: ", msg)
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
		return nil, state.ErrNotInTopicCreationWhitelist
	}

	id, err := ms.k.GetNumTopics(ctx)
	if err != nil {
		return nil, err
	}

	fastestCadence, err := ms.k.GetParamsMinRequestCadence(ctx)
	if err != nil {
		return nil, err
	}
	if msg.InferenceCadence < fastestCadence {
		return nil, state.ErrInferenceCadenceBelowMinimum
	}

	weightFastestCadence, err := ms.k.GetParamsMinLossCadence(ctx)
	if err != nil {
		return nil, err
	}
	if msg.LossCadence < weightFastestCadence {
		return nil, state.ErrLossCadenceBelowMinimum
	}

	topic := state.Topic{
		Id:                     id,
		Creator:                creator.String(),
		Metadata:               msg.Metadata,
		LossLogic:              msg.LossLogic,
		LossMethod:             msg.LossMethod,
		LossCadence:            msg.LossCadence,
		LossLastRan:            0,
		InferenceLogic:         msg.InferenceLogic,
		InferenceMethod:        msg.InferenceMethod,
		InferenceCadence:       msg.InferenceCadence,
		InferenceLastRan:       0,
		Active:                 true,
		DefaultArg:             msg.DefaultArg,
		Pnorm:                  msg.Pnorm,
		AlphaRegret:            msg.AlphaRegret,
		PrewardReputer:         msg.PrewardReputer,
		PrewardInference:       msg.PrewardInference,
		PrewardForecast:        msg.PrewardForecast,
		FTolerance:             msg.FTolerance,
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

	return &state.MsgCreateNewTopicResponse{TopicId: id}, nil
}

func (ms msgServer) ReactivateTopic(ctx context.Context, msg *state.MsgReactivateTopic) (*state.MsgReactivateTopicResponse, error) {
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
		return nil, state.ErrTopicNotEnoughDemand
	}

	// If the topic has enough demand, reactivate it
	err = ms.k.ReactivateTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}
	return &state.MsgReactivateTopicResponse{Success: true}, nil
}

///
/// FOUNDATION TOPIC MANAGEMENT
///

// Modfies topic subsidy and subsidized_reward_epochs properties
// Can only be called by a whitelisted foundation member
func (ms msgServer) ModifyTopicSubsidy(ctx context.Context, msg *state.MsgModifyTopicSubsidyAndSubsidizedRewardEpochs) (*state.MsgModifyTopicSubsidyAndSubsidizedRewardEpochsResponse, error) {
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
		return nil, state.ErrNotWhitelistAdmin
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
	return &state.MsgModifyTopicSubsidyAndSubsidizedRewardEpochsResponse{Success: true}, nil
}

func (ms msgServer) ModifyTopicFTreasury(ctx context.Context, msg *state.MsgModifyTopicFTreasury) (*state.MsgModifyTopicFTreasuryResponse, error) {
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
		return nil, state.ErrNotWhitelistAdmin
	}
	// Modify the topic subsidy + F_treasury
	err = ms.k.SetTopicFTreasury(ctx, msg.TopicId, float32(msg.FTreasury))
	if err != nil {
		return nil, err
	}
	return &state.MsgModifyTopicFTreasuryResponse{Success: true}, nil
}
