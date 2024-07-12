package types

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
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
	if msg.AlphaRegret.Lte(alloraMath.ZeroDec()) || msg.AlphaRegret.Gt(alloraMath.OneDec()) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "alpha regret must be greater than 0 and less than or equal to 1")
	}
	if msg.PNorm.Lt(alloraMath.MustNewDecFromString("2.5")) || msg.PNorm.Gt(alloraMath.MustNewDecFromString("4.5")) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "p-norm must be between 2.5 and 4.5")
	}
	if msg.Epsilon.Lte(alloraMath.ZeroDec()) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "epsilon must be greater than 0")
	}
	// Validation of actor quantiles is stateful and happens in the topic creation handler

	return nil
}
