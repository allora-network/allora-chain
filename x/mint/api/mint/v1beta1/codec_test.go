package mintv1beta1_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	v1beta1 "github.com/allora-network/allora-chain/x/mint/api/mint/v1beta1"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func TestV1Beta1ParamsRegisteredWithGogo(t *testing.T) {
	require.NotNil(t, gogoproto.MessageType("mint.v1beta1.Params"))
	require.Equal(t, "mint.v5.Params", gogoproto.MessageName(&minttypes.Params{}))
}

func TestUnknownProtoWalksV1Beta1UpdateParams(t *testing.T) {
	//nolint:exhaustruct // empty Params is enough to force the nested type lookup
	msg := &v1beta1.MsgUpdateParams{
		Sender: "allo1sender",
		Params: &v1beta1.Params{MintDenom: "uallo"},
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)
	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v1beta1.MsgUpdateParams)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}
