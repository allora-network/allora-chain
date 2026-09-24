package emissionsv2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

func init() {
	// Historical tx queries walk nested (non-Any) fields via gogo proto.MessageType.
	// These v2 pulsar types are distinct from later gogo structs, so live type URLs
	// are unchanged.
	gogoproto.RegisterType((*Nonce)(nil), "emissions.v2.Nonce")
	gogoproto.RegisterType((*Nonces)(nil), "emissions.v2.Nonces")
	gogoproto.RegisterType((*ReputerRequestNonce)(nil), "emissions.v2.ReputerRequestNonce")
	gogoproto.RegisterType((*ReputerRequestNonces)(nil), "emissions.v2.ReputerRequestNonces")
	gogoproto.RegisterType((*TimestampedValue)(nil), "emissions.v2.TimestampedValue")
	gogoproto.RegisterType((*Inference)(nil), "emissions.v2.Inference")
	gogoproto.RegisterType((*Inferences)(nil), "emissions.v2.Inferences")
	gogoproto.RegisterType((*ForecastElement)(nil), "emissions.v2.ForecastElement")
	gogoproto.RegisterType((*Forecast)(nil), "emissions.v2.Forecast")
	gogoproto.RegisterType((*Forecasts)(nil), "emissions.v2.Forecasts")
	gogoproto.RegisterType((*InferenceForecastBundle)(nil), "emissions.v2.InferenceForecastBundle")
	gogoproto.RegisterType((*WorkerDataBundle)(nil), "emissions.v2.WorkerDataBundle")
	gogoproto.RegisterType((*WorkerDataBundles)(nil), "emissions.v2.WorkerDataBundles")
	gogoproto.RegisterType((*WorkerAttributedValue)(nil), "emissions.v2.WorkerAttributedValue")
	gogoproto.RegisterType((*WithheldWorkerAttributedValue)(nil), "emissions.v2.WithheldWorkerAttributedValue")
	gogoproto.RegisterType((*OneOutInfererForecasterValues)(nil), "emissions.v2.OneOutInfererForecasterValues")
	gogoproto.RegisterType((*ValueBundle)(nil), "emissions.v2.ValueBundle")
	gogoproto.RegisterType((*ReputerValueBundle)(nil), "emissions.v2.ReputerValueBundle")
	gogoproto.RegisterType((*ReputerValueBundles)(nil), "emissions.v2.ReputerValueBundles")
	gogoproto.RegisterType((*OptionalParams)(nil), "emissions.v2.OptionalParams")
}

//nolint:exhaustruct
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
	registry.RegisterConcrete(&OptionalParams{}, "emissions/v2/OptionalParams", nil)           //nolint:exhaustruct
	registry.RegisterConcrete(&WorkerDataBundle{}, "emissions/v2/WorkerDataBundle", nil)       //nolint:exhaustruct
	registry.RegisterConcrete(&ReputerValueBundle{}, "emissions/v2/ReputerValueBundle", nil)   //nolint:exhaustruct
	registry.RegisterConcrete(&WorkerDataBundles{}, "emissions/v2/WorkerDataBundles", nil)     //nolint:exhaustruct
	registry.RegisterConcrete(&ReputerValueBundles{}, "emissions/v2/ReputerValueBundles", nil) //nolint:exhaustruct
}
