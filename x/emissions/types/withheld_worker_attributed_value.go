package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (withheldWorkerValue *WithheldWorkerAttributedValue) Validate() error {
	_, err := sdk.AccAddressFromBech32(withheldWorkerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid withheld worker address (%s)", err)
	}

	if err := validateDec(withheldWorkerValue.Value); err != nil {
		return err
	}

	return nil
}
