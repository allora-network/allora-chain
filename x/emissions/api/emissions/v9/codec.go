package emissionsv9

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
