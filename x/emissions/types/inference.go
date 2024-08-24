package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (inference *Inference) Validate() error {
	if inference == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
	}
	_, err := sdk.AccAddressFromBech32(inference.Inferer)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
	}
	if inference.BlockHeight < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference block height cannot be negative")
	}
	if err := ValidateDec(inference.Value); err != nil {
		return err
	}
	return nil
}
