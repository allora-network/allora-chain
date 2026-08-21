package emissionsv9

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

func init() {
	// Historical tx queries walk nested (non-Any) fields via gogo proto.MessageType.
	// After v10, x/emissions/types registers these names as emissions.v10.*, so
	// v9 nested payloads fail to decode unless the pulsar types are also in the
	// gogo registry. These are distinct Go types from the v10 gogo structs, so
	// the reverse name map for new txs is unchanged.
	gogoproto.RegisterType((*InputInference)(nil), "emissions.v9.InputInference")
	gogoproto.RegisterType((*InputInferences)(nil), "emissions.v9.InputInferences")
	gogoproto.RegisterType((*InputForecastElement)(nil), "emissions.v9.InputForecastElement")
	gogoproto.RegisterType((*InputForecast)(nil), "emissions.v9.InputForecast")
	gogoproto.RegisterType((*InputForecasts)(nil), "emissions.v9.InputForecasts")
	gogoproto.RegisterType((*InputInferenceForecastBundle)(nil), "emissions.v9.InputInferenceForecastBundle")
	gogoproto.RegisterType((*InputWorkerDataBundle)(nil), "emissions.v9.InputWorkerDataBundle")
	gogoproto.RegisterType((*InputWorkerDataBundles)(nil), "emissions.v9.InputWorkerDataBundles")
	gogoproto.RegisterType((*OptionalParams)(nil), "emissions.v9.OptionalParams")
}

//nolint:exhaustruct
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&UpdateParamsRequest{},
		&CreateNewTopicRequest{},
		&UpdateTopicRequest{},
		&InsertReputerPayloadRequest{},
		&InsertWorkerPayloadRequest{},
		&TransferActorOwnershipRequest{},
		&RegisterRequest{},
		&RemoveRegistrationRequest{},
		&AddStakeRequest{},
		&RemoveStakeRequest{},
		&CancelRemoveStakeRequest{},
		&DelegateStakeRequest{},
		&RemoveDelegateStakeRequest{},
		&CancelRemoveDelegateStakeRequest{},
		&RewardDelegateStakeRequest{},
		&FundTopicRequest{},
		&AddToWhitelistAdminRequest{},
		&RemoveFromWhitelistAdminRequest{},
		&EnableTopicWorkerWhitelistRequest{},
		&DisableTopicWorkerWhitelistRequest{},
		&EnableTopicReputerWhitelistRequest{},
		&DisableTopicReputerWhitelistRequest{},
		&AddToGlobalWhitelistRequest{},
		&RemoveFromGlobalWhitelistRequest{},
		&AddToTopicCreatorWhitelistRequest{},
		&AddToGlobalWorkerWhitelistRequest{},
		&RemoveFromGlobalWorkerWhitelistRequest{},
		&AddToGlobalReputerWhitelistRequest{},
		&RemoveFromGlobalReputerWhitelistRequest{},
		&AddToGlobalAdminWhitelistRequest{},
		&RemoveFromGlobalAdminWhitelistRequest{},
		&BulkAddToGlobalWorkerWhitelistRequest{},
		&BulkRemoveFromGlobalWorkerWhitelistRequest{},
		&BulkAddToGlobalReputerWhitelistRequest{},
		&BulkRemoveFromGlobalReputerWhitelistRequest{},
		&BulkAddToTopicWorkerWhitelistRequest{},
		&BulkRemoveFromTopicWorkerWhitelistRequest{},
		&BulkAddToTopicReputerWhitelistRequest{},
		&BulkRemoveFromTopicReputerWhitelistRequest{},
		&RemoveFromTopicCreatorWhitelistRequest{},
		&AddToTopicWorkerWhitelistRequest{},
		&RemoveFromTopicWorkerWhitelistRequest{},
		&AddToTopicReputerWhitelistRequest{},
		&RemoveFromTopicReputerWhitelistRequest{},
	)
}

// So we need to register types like:
func RegisterTypes(registry *codec.LegacyAmino) {
	// Internal types used by requests
	registry.RegisterConcrete(&OptionalParams{}, "emissions/v9/OptionalParams", nil) //nolint:exhaustruct
}
