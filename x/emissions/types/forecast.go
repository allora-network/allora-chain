package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (forecast *Forecast) Validate() error {
	if forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
	}
	if forecast.BlockHeight < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast block height cannot be negative")
	}
	if len(forecast.Forecaster) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecaster cannot be empty")
	}
	if len(forecast.ForecastElements) == 0 {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"at least one forecast element must be provided",
		)
	}
	for _, elem := range forecast.ForecastElements {
		_, err := sdk.AccAddressFromBech32(elem.Inferer)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
		}

		if elem.Value.IsNaN() {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast element value cannot be NaN")
		}
	}

	return nil
}
