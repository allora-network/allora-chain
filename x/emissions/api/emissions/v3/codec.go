package emissionsv3

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

func init() {
	// Historical tx queries walk nested (non-Any) fields via gogo proto.MessageType.
	// v3–v8 worker payloads nest emissions.v3.WorkerDataBundle; types/ now registers
	// those worker/inference names as v10. Skip Nonce and ReputerValueBundle — they
	// are still registered as v3 in types/.
	gogoproto.RegisterType((*TimestampedValue)(nil), "emissions.v3.TimestampedValue")
	gogoproto.RegisterType((*Inference)(nil), "emissions.v3.Inference")
	gogoproto.RegisterType((*Inferences)(nil), "emissions.v3.Inferences")
	gogoproto.RegisterType((*ForecastElement)(nil), "emissions.v3.ForecastElement")
	gogoproto.RegisterType((*Forecast)(nil), "emissions.v3.Forecast")
	gogoproto.RegisterType((*Forecasts)(nil), "emissions.v3.Forecasts")
	gogoproto.RegisterType((*InferenceForecastBundle)(nil), "emissions.v3.InferenceForecastBundle")
	gogoproto.RegisterType((*WorkerDataBundle)(nil), "emissions.v3.WorkerDataBundle")
	gogoproto.RegisterType((*WorkerDataBundles)(nil), "emissions.v3.WorkerDataBundles")
	gogoproto.RegisterType((*OptionalParams)(nil), "emissions.v3.OptionalParams")
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
	registry.RegisterConcrete(&OptionalParams{}, "emissions/v3/OptionalParams", nil)           //nolint:exhaustruct
	registry.RegisterConcrete(&WorkerDataBundle{}, "emissions/v3/WorkerDataBundle", nil)       //nolint:exhaustruct
	registry.RegisterConcrete(&ReputerValueBundle{}, "emissions/v3/ReputerValueBundle", nil)   //nolint:exhaustruct
	registry.RegisterConcrete(&WorkerDataBundles{}, "emissions/v3/WorkerDataBundles", nil)     //nolint:exhaustruct
	registry.RegisterConcrete(&ReputerValueBundles{}, "emissions/v3/ReputerValueBundles", nil) //nolint:exhaustruct
}
