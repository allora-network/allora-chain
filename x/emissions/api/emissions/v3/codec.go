package emissionsv3

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// nolint: exhaustruct
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateNewTopic{},
		&MsgRegister{},
		&MsgRemoveRegistration{},
		&MsgAddStake{},
		&MsgRemoveStake{},
		&MsgCancelRemoveStake{},
		&MsgDelegateStake{},
		&MsgRewardDelegateStake{},
		&MsgRemoveDelegateStake{},
		&MsgCancelRemoveDelegateStake{},
		&MsgFundTopic{},
		&MsgAddToWhitelistAdmin{},
		&MsgRemoveFromWhitelistAdmin{},
		&MsgInsertWorkerPayload{},
		&MsgInsertReputerPayload{},
	)
}
