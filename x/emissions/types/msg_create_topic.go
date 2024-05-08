package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgCreateNewTopic) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.LossLogic) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "loss logic cannot be empty")
	}
	if len(msg.LossMethod) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "loss method cannot be empty")
	}
	if len(msg.InferenceLogic) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference logic cannot be empty")
	}
	if len(msg.InferenceMethod) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference method cannot be empty")
	}
	if len(msg.DefaultArg) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "default argument cannot be empty")
	}
	if msg.EpochLength <= 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "epoch length must be greater than zero")
	}
	if msg.GroundTruthLag < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "ground truth lag cannot be negative")
	}
	if msg.AlphaRegret.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "alpha regret cannot negative")
	}
	if msg.PrewardReputer.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "preward for reputer cannot negative")
	}
	if msg.PrewardInference.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "preward for inference cannot negative")
	}
	if msg.PrewardForecast.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "preward for forecast cannot negative")
	}
	if msg.FTolerance.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "f tolerance cannot negative")
	}

	return nil
}
