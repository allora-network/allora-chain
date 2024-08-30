package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (bundle *WorkerDataBundle) Validate() error {
	if bundle == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle cannot be nil")
	}
	if bundle.Nonce == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker data bundle nonce cannot be nil")
	}
	if len(bundle.Worker) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "worker cannot be empty")
	}
	if len(bundle.Pubkey) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "public key cannot be empty")
	}
	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid pubkey")
	}
	pubkey := secp256k1.PubKey(pk)
	pubKeyConvertedToAddress := sdk.AccAddress(pubkey.Address().Bytes()).String()

	if len(bundle.InferencesForecastsBundleSignature) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	if bundle.InferenceForecastsBundle == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference forecasts bundle cannot be nil")
	}

	// Validate the inference and forecast of the bundle
	if bundle.InferenceForecastsBundle.Inference == nil && bundle.InferenceForecastsBundle.Forecast == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "inference and forecast cannot both be nil")
	}
	if bundle.InferenceForecastsBundle.Inference != nil {
		if err := bundle.InferenceForecastsBundle.Inference.Validate(); err != nil {
			return err
		}
		if bundle.InferenceForecastsBundle.Inference.Inferer != pubKeyConvertedToAddress {
			return errorsmod.Wrapf(sdkerrors.ErrUnauthorized,
				"Inference.Inferer %s does not match pubkey %s",
				bundle.InferenceForecastsBundle.Inference.Inferer, pubKeyConvertedToAddress)
		}
		if bundle.Worker != bundle.InferenceForecastsBundle.Inference.Inferer {
			return errorsmod.Wrapf(sdkerrors.ErrUnauthorized,
				"Inference.Inferer %s does not match worker address %s",
				bundle.InferenceForecastsBundle.Inference.Inferer, bundle.Worker)
		}
	}
	if bundle.InferenceForecastsBundle.Forecast != nil {
		if err := bundle.InferenceForecastsBundle.Forecast.Validate(); err != nil {
			return err
		}
		if bundle.InferenceForecastsBundle.Forecast.Forecaster != pubKeyConvertedToAddress {
			return errorsmod.Wrapf(sdkerrors.ErrUnauthorized,
				"Forecast.Forecaster %s does not match pubkey %s",
				bundle.InferenceForecastsBundle.Forecast.Forecaster, pubKeyConvertedToAddress)
		}
		if bundle.Worker != bundle.InferenceForecastsBundle.Forecast.Forecaster {
			return errorsmod.Wrapf(sdkerrors.ErrUnauthorized,
				"Forecast.Forecaster %s does not match worker address %s",
				bundle.InferenceForecastsBundle.Forecast.Forecaster, bundle.Worker)
		}
	}

	// Check signature from the bundle, throw if invalid!
	src := make([]byte, 0)
	src, _ = bundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.InferencesForecastsBundleSignature) {
		return errorsmod.Wrap(sdkerrors.ErrUnauthorized, "signature verification failed")
	}
	// Source: https://docs.cosmos.network/v0.46/basics/accounts.html#addresses
	if pubKeyConvertedToAddress != bundle.Worker {
		return errorsmod.Wrap(sdkerrors.ErrUnauthorized, "worker address does not match signature")
	}

	return nil
}
