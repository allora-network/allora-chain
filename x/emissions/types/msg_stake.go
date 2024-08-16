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

func (msg *MsgAddStake) Validate() error {
	return validateHelper([]string{msg.Sender}, msg.Amount)
}

func (msg *MsgRemoveStake) Validate() error {
	return validateHelper([]string{msg.Sender}, msg.Amount)
}

func (msg *MsgDelegateStake) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, msg.Amount)
}

func (msg *MsgRemoveDelegateStake) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, msg.Amount)
}

func (msg *MsgCancelRemoveDelegateStake) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, cosmosMath.ZeroInt())
}

func (msg *MsgRewardDelegateStake) Validate() error {
	return validateHelper([]string{msg.Sender, msg.Reputer}, cosmosMath.ZeroInt())
}

func (msg *MsgCancelRemoveStake) Validate() error {
	return validateHelper([]string{msg.Sender}, cosmosMath.ZeroInt())
}
