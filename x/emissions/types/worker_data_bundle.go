package types

import (
	"encoding/hex"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

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
	if bundle.InferenceForecastsBundle.Inference == nil &&
		bundle.InferenceForecastsBundle.Forecast == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "inference and forecast cannot both be nil")
	}
	if bundle.InferenceForecastsBundle.Inference != nil {
		if err := bundle.InferenceForecastsBundle.Inference.Validate(); err != nil {
			return err
		}
	}
	if bundle.InferenceForecastsBundle.Forecast != nil {
		if err := bundle.InferenceForecastsBundle.Forecast.Validate(); err != nil {
			return err
		}
	}

	// Check signature from the bundle, throw if invalid!
	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}
	pubkey := secp256k1.PubKey(pk)

	src := make([]byte, 0)
	src, _ = bundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.InferencesForecastsBundleSignature) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}
