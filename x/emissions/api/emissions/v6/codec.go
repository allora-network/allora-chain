package emissionsv6

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

func init() {
	// Nested worker/reputer fields on v6 txs are emissions.v3.* (registered in the
	// v3 package). OptionalParams lives in this package and is no longer in gogo.
	gogoproto.RegisterType((*OptionalParams)(nil), "emissions.v6.OptionalParams")
}

//nolint:exhaustruct
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&UpdateParamsRequest{},
		&CreateNewTopicRequest{},
		&RegisterRequest{},
		&RemoveRegistrationRequest{},
		&AddStakeRequest{},
		&RemoveStakeRequest{},
		&CancelRemoveStakeRequest{},
		&DelegateStakeRequest{},
		&RewardDelegateStakeRequest{},
		&RemoveDelegateStakeRequest{},
		&CancelRemoveDelegateStakeRequest{},
		&FundTopicRequest{},
		&AddToWhitelistAdminRequest{},
		&RemoveFromWhitelistAdminRequest{},
		&InsertWorkerPayloadRequest{},
		&InsertReputerPayloadRequest{},
		&AddToGlobalWhitelistRequest{},
		&RemoveFromGlobalWhitelistRequest{},
		&EnableTopicWorkerWhitelistRequest{},
		&DisableTopicWorkerWhitelistRequest{},
		&EnableTopicReputerWhitelistRequest{},
		&DisableTopicReputerWhitelistRequest{},
		&AddToTopicCreatorWhitelistRequest{},
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
	registry.RegisterConcrete(&OptionalParams{}, "emissions/v6/OptionalParams", nil) //nolint:exhaustruct
}
