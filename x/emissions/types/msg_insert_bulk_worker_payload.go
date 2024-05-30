package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgInsertBulkWorkerPayload) ValidateTopLevel() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if msg.Nonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "nonce cannot be nil")
	}

	if len(msg.WorkerDataBundles) == 0 {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"at least one worker data bundle must be provided",
		)
	}

	return nil
}
