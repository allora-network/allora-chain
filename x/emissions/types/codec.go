package types

import (
	"cosmossdk.io/collections/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var (
	EpochStateKey = codec.NewInt32Key[EpochState]()
	NonceKey      = codec.NewUint64Key[NonceV2]()
)

// RegisterInterfaces registers the interfaces types with the interface registry.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	msgservice.RegisterMsgServiceDesc(registry, &_MsgService_serviceDesc)
}
