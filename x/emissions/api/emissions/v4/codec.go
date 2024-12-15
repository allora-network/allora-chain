package emissionsv4

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// nolint: exhaustruct
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
	)
}
