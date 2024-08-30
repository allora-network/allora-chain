package types

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func validateDec(value alloraMath.Dec) error {
	if value.IsNaN() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be NaN")
	}

	if !value.IsFinite() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "value must be finite")
	}

	return nil
}
