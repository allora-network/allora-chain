package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// validateMaxTopInferersToReward rejects a create/update request outside the
// global [min, max] range.
func validateMaxTopInferersToReward(requested, globalMin, globalMax uint64) error {
	if requested > globalMax {
		return errorsmod.Wrapf(
			types.ErrTopicMaxTopInferersToRewardTooBig,
			"requested %d exceeds global maximum %d", requested, globalMax,
		)
	}
	if requested < globalMin {
		return errorsmod.Wrapf(
			types.ErrTopicMaxTopInferersToRewardTooSmall,
			"requested %d is below global minimum %d", requested, globalMin,
		)
	}
	return nil
}

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *types.CreateNewTopicRequest) (_ *types.CreateNewTopicResponse, err error) {
	defer metrics.RecordMetrics("CreateNewTopic", time.Now(), &err)

	// Validate the address
	if err := types.ValidateStringIsBech32(msg.Creator); err != nil {
		return nil, err
	}
	canCreate, err := ms.wlk.CanCreateTopic(ctx, msg.Creator)
	if err != nil {
		return nil, err
	} else if !canCreate {
		return nil, types.ErrNotPermittedToCreateTopic
	}

	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Creator)
	}
	if err := msg.Validate(params.MaxStringLength, params.MaxTopicLabelWhitelistSize); err != nil {
		return nil, err
	}

	topicId, err := ms.tk.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}

	if msg.EpochLength < params.MinEpochLength {
		return nil, types.ErrTopicCadenceBelowMinimum
	}
	if uint64(msg.GroundTruthLag) > params.MaxUnfulfilledReputerRequests*uint64(msg.EpochLength) {
		return nil, types.ErrGroundTruthLagTooBig
	}
	if err := validateMaxTopInferersToReward(
		msg.MaxTopInferersToReward, params.MinTopInferersToReward, params.MaxTopInferersToReward); err != nil {
		return nil, err
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
		CNorm:                    msg.CNorm,
		TopicType:                msg.TopicType,
		OutputArity:              msg.OutputArity,
		RequireUnity:             msg.RequireUnity,
		UnityTolerance:           msg.UnityTolerance,
		// Label registry fields. Canonicalization of LabelWhitelist is
		// applied by SetTopic.
		MaxLabelsPerSubmission: msg.MaxLabelsPerSubmission,
		LabelWhitelist:         msg.LabelWhitelist,
		LabelDefaultValue:      msg.LabelDefaultValue,
		// LabelCaseSensitive is immutable after creation (UpdateTopic never
		// changes it because updatedTopic is derived from the existing topic).
		LabelCaseSensitive:     msg.LabelCaseSensitive,
		MaxTopInferersToReward: msg.MaxTopInferersToReward,
	}
	_, err = ms.tk.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.tk.SetTopic(ctx, topicId, topic); err != nil {
		return nil, err
	}

	// Turn topic whitelist on by default so no one can squeeze in payloads before an admin notices or can act
	if msg.EnableWorkerWhitelist {
		err = ms.wlk.EnableTopicWorkerWhitelist(ctx, topicId)
		if err != nil {
			return nil, err
		}
	}
	if msg.EnableReputerWhitelist {
		err = ms.wlk.EnableTopicReputerWhitelist(ctx, topicId)
		if err != nil {
			return nil, err
		}
	}

	err = ms.tk.AddTopicFeeRevenue(ctx, topicId, params.CreateTopicFee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error adding topic fee revenue")
	}

	types.EmitNewCreateNewTopicEvent(ctx, &topic)
	return &types.CreateNewTopicResponse{TopicId: topicId}, nil
}

func (ms msgServer) UpdateTopic(ctx context.Context, msg *types.UpdateTopicRequest) (_ *types.UpdateTopicResponse, err error) {
	defer metrics.RecordMetrics("UpdateTopic", time.Now(), &err)

	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}
	if err := msg.Validate(params.MaxStringLength, params.MaxTopicLabelWhitelistSize); err != nil {
		return nil, err
	}

	topic, err := ms.tk.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}

	if topic.Creator != msg.Sender {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "not permitted to modify topic")
	}

	if err := validateMaxTopInferersToReward(
		msg.MaxTopInferersToReward, params.MinTopInferersToReward, params.MaxTopInferersToReward); err != nil {
		return nil, err
	}

	updatedTopic := topic
	updatedTopic.Metadata = msg.Metadata
	updatedTopic.LossMethod = msg.LossMethod
	updatedTopic.AlphaRegret = msg.AlphaRegret
	updatedTopic.MeritSortitionAlpha = msg.MeritSortitionAlpha
	updatedTopic.PNorm = msg.PNorm
	updatedTopic.CNorm = msg.CNorm
	// Full replacement; the keeper's UpdateTopic rejects a change while a worker
	// submission window is open.
	updatedTopic.MaxTopInferersToReward = msg.MaxTopInferersToReward
	// Label registry settings: always apply the requested value. The keeper
	// rejects unsafe mutations while a worker submission window is open and
	// canonicalizes the whitelist before persistence.
	updatedTopic.MaxLabelsPerSubmission = msg.MaxLabelsPerSubmission
	updatedTopic.LabelWhitelist = msg.LabelWhitelist
	updatedTopic.LabelDefaultValue = msg.LabelDefaultValue

	updatedTopic, err = ms.tk.UpdateTopic(ctx, topic, updatedTopic)
	if err != nil {
		return nil, err
	}

	types.EmitNewTopicUpdatedEvent(ctx, updatedTopic)

	return &types.UpdateTopicResponse{}, nil
}
