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
	if inference.Value.IsNaN() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference value cannot be NaN")
	}
	_, err := sdk.AccAddressFromBech32(inference.Inferer)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
	}
	if inference.BlockHeight < 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference block height cannot be negative")
	}
	if len(inference.Proof) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference proof cannot be empty")
	}

	return nil
}

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
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "at least one forecast element must be provided")
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

func (bundle *WorkerDataBundle) Validate() error {
	// Validate top level then elements of the bundle
	if bundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle cannot be nil")
	}
	if len(bundle.Worker) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "worker cannot be empty")
	}
	if len(bundle.Pubkey) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "public key cannot be empty")
	}
	if len(bundle.InferencesForecastsBundleSignature) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	if bundle.InferenceForecastsBundle == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference forecasts bundle cannot be nil")
	}

	// Validate the inference and forecast of the bundle
	if err := bundle.InferenceForecastsBundle.Inference.Validate(); err != nil {
		return err
	}
	if err := bundle.InferenceForecastsBundle.Forecast.Validate(); err != nil {
		return err
	}

	return nil
}

func (msg *MsgInsertBulkWorkerPayload) ValidateTopLevel() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if msg.Nonce == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "nonce cannot be nil")
	}

	if len(msg.WorkerDataBundles) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "at least one worker data bundle must be provided")
	}

	return nil
}
