package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgRegister) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address (%s)", err)
	}

	if len(msg.LibP2PKey) == 0 {
		return errors.Wrap(ErrLibP2PKeyRequired, "libP2PKey cannot be empty")
	}
	if len(msg.MultiAddress) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "multiAddress cannot be empty")
	}

	return nil
}
