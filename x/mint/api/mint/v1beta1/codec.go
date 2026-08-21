package mintv1beta1

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

func init() {
	// Historical tx queries walk nested (non-Any) fields via gogo proto.MessageType.
	// types/ now registers Params as mint.v5.Params, so v1beta1 UpdateParams txs
	// fail unless the pulsar type is also in the gogo registry.
	gogoproto.RegisterType((*Params)(nil), "mint.v1beta1.Params")
}

//nolint:exhaustruct
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
}
