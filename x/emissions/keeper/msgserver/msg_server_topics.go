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
	if err := msg.Validate(params.MaxStringLength); err != nil {
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
		// Label registry v2 fields. Canonicalization of LabelWhitelist is
		// applied by SetTopic; zero MaxLabelsPerSubmission means "fall back
		// to Params.MaxLabelsPerSubmission" at worker-payload submission.
		MaxLabelsPerSubmission: msg.MaxLabelsPerSubmission,
		LabelWhitelist:         msg.LabelWhitelist,
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

	if err := types.ValidateStringIsBech32(msg.Sender); err != nil {
		return nil, err
	}

	topic, err := ms.tk.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}

	if topic.Creator != msg.Sender {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "not permitted to modify topic")
	}

	updatedTopic := topic
	updatedTopic.Metadata = msg.Metadata
	updatedTopic.LossMethod = msg.LossMethod
	updatedTopic.AlphaRegret = msg.AlphaRegret
	updatedTopic.MeritSortitionAlpha = msg.MeritSortitionAlpha
	updatedTopic.PNorm = msg.PNorm
	updatedTopic.CNorm = msg.CNorm
	// Label registry v2: always apply the requested value for
	// max_labels_per_submission and label_whitelist. The keeper's UpdateTopic
	// rejects mutations that change these fields while any worker submission
	// window for the topic is open and canonicalizes the whitelist so the
	// persisted form is exact byte-equality with submitted labels.
	updatedTopic.MaxLabelsPerSubmission = msg.MaxLabelsPerSubmission
	updatedTopic.LabelWhitelist = msg.LabelWhitelist

	updatedTopic, err = ms.tk.UpdateTopic(ctx, topic, updatedTopic)
	if err != nil {
		return nil, err
	}

	types.EmitNewTopicUpdatedEvent(ctx, updatedTopic)

	return &types.UpdateTopicResponse{}, nil
}
