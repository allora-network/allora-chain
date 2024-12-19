package emissionsv3

import (
	"github.com/cosmos/cosmos-sdk/codec"
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

// So we need to register types like:
func RegisterTypes(registry *codec.LegacyAmino) {
	// Internal types used by requests
	registry.RegisterConcrete(&OptionalParams{}, "emissions/v3/OptionalParams", nil)           //nolint:exhaustruct
	registry.RegisterConcrete(&WorkerDataBundle{}, "emissions/v3/WorkerDataBundle", nil)       //nolint:exhaustruct
	registry.RegisterConcrete(&ReputerValueBundle{}, "emissions/v3/ReputerValueBundle", nil)   //nolint:exhaustruct
	registry.RegisterConcrete(&WorkerDataBundles{}, "emissions/v3/WorkerDataBundles", nil)     //nolint:exhaustruct
	registry.RegisterConcrete(&ReputerValueBundles{}, "emissions/v3/ReputerValueBundles", nil) //nolint:exhaustruct
}
