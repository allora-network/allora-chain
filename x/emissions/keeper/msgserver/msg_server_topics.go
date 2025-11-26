package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *types.CreateNewTopicRequest) (_ *types.CreateNewTopicResponse, err error) {
	defer metrics.RecordMetrics("CreateNewTopic", time.Now(), &err)

	// Validate the address
	if err := ms.k.ValidateStringIsBech32(msg.Creator); err != nil {
		return nil, err
	}
	canCreate, err := ms.k.CanCreateTopic(ctx, msg.Creator)
	if err != nil {
		return nil, err
	} else if !canCreate {
		return nil, types.ErrNotPermittedToCreateTopic
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Creator)
	}
	if err := msg.Validate(params.MaxStringLength); err != nil {
		return nil, err
	}

	topicId, err := ms.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}

	if msg.EpochLength < params.MinEpochLength {
		return nil, types.ErrTopicCadenceBelowMinimum
	}
	if uint64(msg.GroundTruthLag) > params.MaxUnfulfilledReputerRequests*uint64(msg.EpochLength) {
		return nil, types.ErrGroundTruthLagTooBig
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = checkBalanceAndSendFee(ctx, ms, msg.Creator, params.CreateTopicFee)
	if err != nil {
		return nil, err
	}

	topic := types.Topic{
		Id:                       topicId,
		Creator:                  msg.Creator,
		Metadata:                 msg.Metadata,
		LossMethod:               msg.LossMethod,
		EpochLastEnded:           0,
		EpochLength:              msg.EpochLength,
		GroundTruthLag:           msg.GroundTruthLag,
		WorkerSubmissionWindow:   msg.WorkerSubmissionWindow,
		PNorm:                    msg.PNorm,
		AlphaRegret:              msg.AlphaRegret,
		AllowNegative:            msg.AllowNegative,
		Epsilon:                  msg.Epsilon,
		InitialRegret:            alloraMath.ZeroDec(),
		MeritSortitionAlpha:      msg.MeritSortitionAlpha,
		ActiveInfererQuantile:    msg.ActiveInfererQuantile,
		ActiveForecasterQuantile: msg.ActiveForecasterQuantile,
		ActiveReputerQuantile:    msg.ActiveReputerQuantile,
	}
	_, err = ms.k.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.k.SetTopic(ctx, topicId, topic); err != nil {
		return nil, err
	}

	// Turn topic whitelist on by default so no one can squeeze in payloads before an admin notices or can act
	if msg.EnableWorkerWhitelist {
		err = ms.k.EnableTopicWorkerWhitelist(ctx, topicId)
		if err != nil {
			return nil, err
		}
	}
	if msg.EnableReputerWhitelist {
		err = ms.k.EnableTopicReputerWhitelist(ctx, topicId)
		if err != nil {
			return nil, err
		}
	}

	err = ms.k.AddTopicFeeRevenue(ctx, topicId, params.CreateTopicFee)

	types.EmitNewCreateNewTopicEvent(ctx, &topic)
	return &types.CreateNewTopicResponse{TopicId: topicId}, err
}

func (ms msgServer) UpdateTopic(ctx context.Context, msg *types.UpdateTopicRequest) (_ *types.UpdateTopicResponse, err error) {
	defer metrics.RecordMetrics("UpdateTopic", time.Now(), &err)

	// Validate the sender address
	if err := ms.k.ValidateStringIsBech32(msg.Sender); err != nil {
		return nil, err
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}

	// Check if topic exists
	topic, err := ms.k.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting topic: %v", msg.TopicId)
	}

	// Check if sender is the topic creator
	if topic.Creator != msg.Sender {
		return nil, types.ErrNotPermittedToModifyTopic
	}

	// Validate the request now that we have the existing topic's epoch length
	if err := msg.Validate(params.MaxStringLength); err != nil {
		return nil, err
	}

	// Create updated topic by copying existing topic and applying ONLY allowed changes
	// Only LossMethod and Metadata can be updated
	updatedTopic := topic
	hasChanges := false

	if len(msg.Metadata) > 0 {
		updatedTopic.Metadata = msg.Metadata[0]
		hasChanges = true
	}

	if len(msg.LossMethod) > 0 {
		updatedTopic.LossMethod = msg.LossMethod[0]
		hasChanges = true
	}

	if len(msg.AlphaRegret) > 0 {
		alphaRegret, err := alloraMath.NewDecFromString(msg.AlphaRegret[0])
		if err != nil {
			return nil, errorsmod.Wrap(err, "Failed to parse alpha_regret")
		}
		updatedTopic.AlphaRegret = alphaRegret
		hasChanges = true
	}

	if len(msg.MeritSortitionAlpha) > 0 {
		meritSortitionAlpha, err := alloraMath.NewDecFromString(msg.MeritSortitionAlpha[0])
		if err != nil {
			return nil, errorsmod.Wrap(err, "Failed to parse merit_sortition_alpha")
		}

		// If the topic is active and the worker submission window is open, disallow updating merit_sortition_alpha
		isActive, err := ms.k.IsTopicActive(ctx, msg.TopicId)
		if err != nil {
			return nil, errorsmod.Wrap(err, "Failed to check topic active status")
		}
		if isActive {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			blockHeight := sdkCtx.BlockHeight()
			nonces, err := ms.k.GetUnfulfilledWorkerNonces(ctx, msg.TopicId)
			if err != nil {
				return nil, errorsmod.Wrap(err, "Failed to get unfulfilled worker nonces")
			}
			if len(nonces.Nonces) > 0 {
				lastNonce := nonces.Nonces[0]
				withinWindow, err := keeper.BlockWithinWorkerSubmissionWindowOfNonce(updatedTopic, *lastNonce, blockHeight)
				if err != nil {
					return nil, errorsmod.Wrap(err, "Failed to check worker submission window")
				}
				if withinWindow {
					return nil, errorsmod.Wrap(types.ErrWorkerNonceWindowNotAvailable, "cannot update merit_sortition_alpha while worker window is open")
				}
			}
		}

		updatedTopic.MeritSortitionAlpha = meritSortitionAlpha
		hasChanges = true
	}

	if len(msg.PNorm) > 0 {
		pNorm, err := alloraMath.NewDecFromString(msg.PNorm[0])
		if err != nil {
			return nil, errorsmod.Wrap(err, "Failed to parse p_norm")
		}
		updatedTopic.PNorm = pNorm
		hasChanges = true
	}

	if !hasChanges {
		return nil, errorsmod.Wrap(types.ErrNoUpdateFields, "No fields to update")
	}

	// Validate the updated topic
	if err := updatedTopic.Validate(params); err != nil {
		return nil, errorsmod.Wrap(err, "Updated topic validation failed")
	}

	// Apply update
	if err := ms.k.SetTopic(ctx, msg.TopicId, updatedTopic); err != nil {
		return nil, errorsmod.Wrap(err, "Failed to apply topic update")
	}

	types.EmitNewTopicUpdatedEvent(ctx, msg.TopicId)

	return &types.UpdateTopicResponse{}, nil
}
