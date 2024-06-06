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

	if workerValue.Value.IsNaN() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be NaN")
	}

	return nil
}
