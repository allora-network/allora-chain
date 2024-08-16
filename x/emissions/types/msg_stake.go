package types

import (
	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgAddStake) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if msg.Amount.LTE(cosmosMath.ZeroInt()) {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount (%s)", err)
	}

	return nil
}

func (msg *MsgRemoveStake) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address %s", msg.Sender)
	}
	if msg.Amount.LTE(cosmosMath.ZeroInt()) {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount %s", msg.Amount.String())
	}
	return nil
}
