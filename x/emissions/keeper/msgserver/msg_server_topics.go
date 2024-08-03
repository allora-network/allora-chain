package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *types.MsgCreateNewTopic) (*types.MsgCreateNewTopicResponse, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	topicId, err := ms.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Creator)
	}
	if msg.EpochLength < params.MinEpochLength {
		return nil, types.ErrTopicCadenceBelowMinimum
	}
	if msg.GroundTruthLag > int64(params.MaxUnfulfilledReputerRequests)*msg.EpochLength {
		return nil, types.ErrGroundTruthLagTooBig
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = checkBalanceAndSendFee(ctx, ms, msg.Creator, topicId, params.CreateTopicFee, "create topic")
	if err != nil {
		return nil, err
	}

	topic := types.Topic{
		Id:                     topicId,
		Creator:                msg.Creator,
		Metadata:               msg.Metadata,
		LossMethod:             msg.LossMethod,
		EpochLastEnded:         0,
		EpochLength:            msg.EpochLength,
		GroundTruthLag:         msg.GroundTruthLag,
		WorkerSubmissionWindow: msg.WorkerSubmissionWindow,
		PNorm:                  msg.PNorm,
		AlphaRegret:            msg.AlphaRegret,
		AllowNegative:          msg.AllowNegative,
		Epsilon:                msg.Epsilon,
		InitialRegret:          alloraMath.ZeroDec(),
	}
	_, err = ms.k.IncrementTopicId(ctx)
	if err != nil {
		return nil, err
	}
	if err := ms.k.SetTopic(ctx, topicId, topic); err != nil {
		return nil, err
	}
	// Rather than set latest weight-adjustment timestamp of a topic to 0
	// we do nothing, since no value in the map means zero

	err = ms.k.AddTopicFeeRevenue(ctx, topicId, params.CreateTopicFee)
	return &types.MsgCreateNewTopicResponse{TopicId: topicId}, err
}
