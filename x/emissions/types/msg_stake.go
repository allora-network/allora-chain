package types

import (
	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func validateHelper(addr []string, amount cosmosMath.Int) error {
	for _, ad := range addr {
		_, err := sdk.AccAddressFromBech32(ad)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", ad)
		}
	}
	if amount.LT(cosmosMath.ZeroInt()) {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount (%s)", amount.String())
	}
	return nil
}

func (msg *MsgServiceAddStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender}, msg.Amount)
}

func (msg *MsgServiceRemoveStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender}, msg.Amount)
}

func (msg *MsgServiceDelegateStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, msg.Amount)
}

func (msg *MsgServiceRemoveDelegateStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, msg.Amount)
}

func (msg *MsgServiceCancelRemoveDelegateStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, cosmosMath.ZeroInt())
}

func (msg *MsgServiceRewardDelegateStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, cosmosMath.ZeroInt())
}

func (msg *MsgServiceCancelRemoveStakeRequest) Validate() error {
	return validateHelper([]string{msg.Sender}, cosmosMath.ZeroInt())
}
