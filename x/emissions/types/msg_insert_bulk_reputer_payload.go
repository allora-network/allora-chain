package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgInsertBulkReputerPayload) ValidateTopLevel() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if msg.ReputerRequestNonce == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "reputer request nonce cannot be nil")
	}
	if msg.ReputerRequestNonce.ReputerNonce == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "reputer nonce cannot be nil")
	}
	if len(msg.ReputerValueBundles) == 0 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "at least one reputer value bundle needs to be provided")
	}

	return nil
}
