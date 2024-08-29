package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func validateNewTopic(msg *emissionstypes.MsgCreateNewTopic) error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.LossMethod) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "loss method cannot be empty")
	}
	if msg.EpochLength <= 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "epoch length must be greater than zero")
	}
	if msg.WorkerSubmissionWindow == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker submission window must be greater than zero")
	}
	if msg.GroundTruthLag < msg.EpochLength {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "ground truth lag cannot be lower than epoch length")
	}
	if msg.WorkerSubmissionWindow > msg.EpochLength {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker submission window cannot be higher than epoch length")
	}
	if msg.AlphaRegret.Lte(alloraMath.ZeroDec()) ||
		msg.AlphaRegret.Gt(alloraMath.OneDec()) ||
		validateDec(msg.AlphaRegret) != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "alpha regret must be greater than 0 and less than or equal to 1")
	}
	if msg.PNorm.Lt(alloraMath.MustNewDecFromString("2.5")) ||
		msg.PNorm.Gt(alloraMath.MustNewDecFromString("4.5")) ||
		validateDec(msg.PNorm) != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "p-norm must be between 2.5 and 4.5")
	}
	if msg.Epsilon.Lte(alloraMath.ZeroDec()) || msg.Epsilon.IsNaN() || !msg.Epsilon.IsFinite() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "epsilon must be greater than 0")
	}
	if !msg.MeritSortitionAlpha.IsBetweenZeroAndOneInclusive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "max active inferer quantile must be between 0 and 1")
	}
	if !msg.ActiveInfererQuantile.IsBetweenZeroAndOneInclusive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "max active inferer quantile must be between 0 and 1")
	}
	if !msg.ActiveForecasterQuantile.IsBetweenZeroAndOneInclusive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "max active forecaster quantile must be between 0 and 1")
	}
	if !msg.ActiveReputerQuantile.IsBetweenZeroAndOneInclusive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "max active reputer quantile must be between 0 and 1")
	}

	return nil
}

func (ms msgServer) CreateNewTopic(ctx context.Context, msg *emissionstypes.MsgCreateNewTopic) (*emissionstypes.MsgCreateNewTopicResponse, error) {
	if err := validateNewTopic(msg); err != nil {
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
		return nil, emissionstypes.ErrTopicCadenceBelowMinimum
	}
	if uint64(msg.GroundTruthLag) > params.MaxUnfulfilledReputerRequests*uint64(msg.EpochLength) {
		return nil, emissionstypes.ErrGroundTruthLagTooBig
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = checkBalanceAndSendFee(ctx, ms, msg.Creator, params.CreateTopicFee)
	if err != nil {
		return nil, err
	}

	topic := emissionstypes.Topic{
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
	// Rather than set latest weight-adjustment timestamp of a topic to 0
	// we do nothing, since no value in the map means zero

	err = ms.k.AddTopicFeeRevenue(ctx, topicId, params.CreateTopicFee)
	return &emissionstypes.MsgCreateNewTopicResponse{TopicId: topicId}, err
}
