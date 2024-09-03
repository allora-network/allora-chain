package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (workerValue *WorkerAttributedValue) Validate() error {
	_, err := sdk.AccAddressFromBech32(workerValue.Worker)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s)", err)
	}

	if err := validateDec(workerValue.Value); err != nil {
		return err
	}

	return nil
}
