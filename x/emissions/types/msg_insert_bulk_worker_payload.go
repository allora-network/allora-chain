package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgInsertBulkWorkerPayload) Validate() error {
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
	for _, bundle := range msg.WorkerDataBundles {
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

		inference := bundle.InferenceForecastsBundle.Inference
		if inference == nil {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference cannot be nil")
		}
		_, err := sdk.AccAddressFromBech32(inference.Inferer)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid inferer address (%s)", err)
		}
		if inference.TopicId != msg.TopicId {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference topic ID must match the message topic ID")
		}
		if inference.BlockHeight < 0 {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference block height cannot be negative")
		}
		if len(inference.Proof) == 0 {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference proof cannot be empty")
		}

		forecast := bundle.InferenceForecastsBundle.Forecast
		if forecast == nil {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast cannot be nil")
		}
		if forecast.TopicId != msg.TopicId {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "forecast topic ID must match the message topic ID")
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
		}
	}

	return nil
}
