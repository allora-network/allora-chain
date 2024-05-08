package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgInsertBulkReputerPayload) Validate() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if msg.ReputerRequestNonce == nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "reputer request nonce cannot be nil")
	}

	if len(msg.ReputerValueBundles) == 0 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "at least one reputer value bundle needs to be provided")
	}

	for _, bundle := range msg.ReputerValueBundles {
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

		// Check each InfererValues entry
		for _, infererValue := range bundle.ValueBundle.InfererValues {
			_, err := sdk.AccAddressFromBech32(infererValue.Worker)
			if err != nil {
				return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s) - inferer values", err)
			}
		}

		// Check each ForecasterValues entry
		for _, forecasterValue := range bundle.ValueBundle.ForecasterValues {
			_, err := sdk.AccAddressFromBech32(forecasterValue.Worker)
			if err != nil {
				return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s) - forecaster values", err)
			}
		}

		// Check each OneOutInfererValues entry
		for _, oneOutInfererValue := range bundle.ValueBundle.OneOutInfererValues {
			_, err := sdk.AccAddressFromBech32(oneOutInfererValue.Worker)
			if err != nil {
				return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s) - one-out inferer values", err)
			}
		}

		// Check each OneOutForecasterValues entry
		for _, oneOutForecasterValue := range bundle.ValueBundle.OneOutForecasterValues {
			_, err := sdk.AccAddressFromBech32(oneOutForecasterValue.Worker)
			if err != nil {
				return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s) - one-out forecaster values", err)
			}
		}

		// Check each OneInForecasterValues entry
		for _, oneInForecasterValue := range bundle.ValueBundle.OneInForecasterValues {
			_, err := sdk.AccAddressFromBech32(oneInForecasterValue.Worker)
			if err != nil {
				return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid worker address (%s) - one-in forecaster values", err)
			}
		}
	}

	return nil
}
