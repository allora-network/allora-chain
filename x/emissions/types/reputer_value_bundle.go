package types

import (
	"encoding/hex"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (bundle *ReputerValueBundle) Validate() error {
	if bundle.ValueBundle == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "value bundle cannot be nil")
	}

	_, err := sdk.AccAddressFromBech32(bundle.ValueBundle.Reputer)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reputer address (%s)", err)
	}

	if bundle.ValueBundle.ReputerRequestNonce == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "value bundle's reputer request nonce cannot be nil")
	}

	if err := ValidateDec(bundle.ValueBundle.CombinedValue); err != nil {
		return err
	}

	if err := ValidateDec(bundle.ValueBundle.NaiveValue); err != nil {
		return err
	}

	for _, infererValue := range bundle.ValueBundle.InfererValues {
		if err := infererValue.Validate(); err != nil {
			return err
		}
	}

	for _, forecasterValue := range bundle.ValueBundle.ForecasterValues {
		if err := forecasterValue.Validate(); err != nil {
			return err
		}
	}

	for _, oneOutInfererValue := range bundle.ValueBundle.OneOutInfererValues {
		if err := oneOutInfererValue.Validate(); err != nil {
			return err
		}
	}

	for _, oneOutForecasterValue := range bundle.ValueBundle.OneOutForecasterValues {
		if err := oneOutForecasterValue.Validate(); err != nil {
			return err
		}
	}

	for _, oneInForecasterValue := range bundle.ValueBundle.OneInForecasterValues {
		if err := oneInForecasterValue.Validate(); err != nil {
			return err
		}
	}

	for _, oneOutInfererForecaster := range bundle.ValueBundle.OneOutInfererForecasterValues {
		_, err := sdk.AccAddressFromBech32(oneOutInfererForecaster.Forecaster)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid forecaster address in OneOutInfererForecasterValues (%s)", err)
		}
		for _, oneOutInfererValue := range oneOutInfererForecaster.OneOutInfererValues {
			if err := oneOutInfererValue.Validate(); err != nil {
				return err
			}
		}
	}

	pk, err := hex.DecodeString(bundle.Pubkey)
	if err != nil || len(pk) != secp256k1.PubKeySize {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}
	pubkey := secp256k1.PubKey(pk)

	src := make([]byte, 0)
	src, _ = bundle.ValueBundle.XXX_Marshal(src, true)
	if !pubkey.VerifySignature(src, bundle.Signature) {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "signature verification failed")
	}

	return nil
}
