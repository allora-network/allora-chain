package emissionsv3_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	v3 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v3"
	v4 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v4"
	_ "github.com/allora-network/allora-chain/x/emissions/api/emissions/v5" // register v5 OptionalParams with gogo
	_ "github.com/allora-network/allora-chain/x/emissions/api/emissions/v6" // register v6 OptionalParams with gogo
	_ "github.com/allora-network/allora-chain/x/emissions/api/emissions/v7" // register v7 OptionalParams with gogo
	v8 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v8"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func TestV3WorkerNestedTypesRegisteredWithGogo(t *testing.T) {
	names := []string{
		"emissions.v3.WorkerDataBundle",
		"emissions.v3.Inference",
		"emissions.v3.Forecast",
		"emissions.v3.ForecastElement",
		"emissions.v3.InferenceForecastBundle",
		"emissions.v3.OptionalParams",
		"emissions.v4.OptionalParams",
		"emissions.v5.OptionalParams",
		"emissions.v6.OptionalParams",
		"emissions.v7.OptionalParams",
		"emissions.v8.OptionalParams",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, gogoproto.MessageType(name), "gogo registry missing %s", name)
		})
	}

	require.Equal(t, "emissions.v10.WorkerDataBundle", gogoproto.MessageName(&emissionstypes.WorkerDataBundle{}))
	require.Equal(t, "emissions.v10.Inference", gogoproto.MessageName(&emissionstypes.Inference{}))
	require.Equal(t, "emissions.v3.ReputerValueBundle", gogoproto.MessageName(&emissionstypes.ReputerValueBundle{}))
	require.Equal(t, "emissions.v3.Nonce", gogoproto.MessageName(&emissionstypes.Nonce{}))
}

func TestUnknownProtoWalksV3InsertWorkerPayload(t *testing.T) {
	msg := &v3.MsgInsertWorkerPayload{
		Sender:           "allo1sender",
		WorkerDataBundle: testV3WorkerBundle(),
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)
	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v3.MsgInsertWorkerPayload)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}

func TestUnknownProtoWalksV4InsertWorkerPayload(t *testing.T) {
	msg := &v4.InsertWorkerPayloadRequest{
		Sender:           "allo1sender",
		WorkerDataBundle: testV3WorkerBundle(),
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)
	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v4.InsertWorkerPayloadRequest)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}

func TestUnknownProtoWalksV8UpdateParams(t *testing.T) {
	//nolint:exhaustruct // empty OptionalParams is enough to force the nested type lookup
	msg := &v8.UpdateParamsRequest{
		Sender: "allo1sender",
		Params: &v8.OptionalParams{},
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)
	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v8.UpdateParamsRequest)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}

func testV3WorkerBundle() *v3.WorkerDataBundle {
	return &v3.WorkerDataBundle{
		Worker:  "allo1worker",
		Nonce:   &v3.Nonce{BlockHeight: 1},
		TopicId: 1,
		InferenceForecastsBundle: &v3.InferenceForecastBundle{
			Inference: &v3.Inference{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     "allo1worker",
				Value:       "1.0",
			},
		},
		InferencesForecastsBundleSignature: []byte{0x01},
		Pubkey:                             "pubkey",
	}
}
