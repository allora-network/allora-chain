package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers concrete types on the LegacyAmino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(Params{}, "allora-chain/x/mint/Params", nil)                               //nolint: exhaustruct
	legacy.RegisterAminoMsg(cdc, &UpdateParamsRequest{}, "allora-chain/x/mint/UpdateParamsRequest") //nolint: exhaustruct
}

// RegisterInterfaces registers the interfaces types with the interface registry.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&UpdateParamsRequest{}, //nolint: exhaustruct
	)

	msgservice.RegisterMsgServiceDesc(registry, &_MsgService_serviceDesc)
}
