package emissionsv9_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	v3 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v3"
	v9 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v9"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func TestV9NestedTypesRegisteredWithGogo(t *testing.T) {
	names := []string{
		"emissions.v9.InputInference",
		"emissions.v9.InputInferences",
		"emissions.v9.InputForecastElement",
		"emissions.v9.InputForecast",
		"emissions.v9.InputForecasts",
		"emissions.v9.InputInferenceForecastBundle",
		"emissions.v9.InputWorkerDataBundle",
		"emissions.v9.InputWorkerDataBundles",
		"emissions.v9.OptionalParams",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, gogoproto.MessageType(name), "gogo registry missing %s", name)
		})
	}

	// Registering pulsar types under v9 names must not rename the live v10 gogo types.
	require.Equal(t, "emissions.v10.InputWorkerDataBundle", gogoproto.MessageName(&emissionstypes.InputWorkerDataBundle{}))
	require.Equal(t, "emissions.v10.OptionalParams", gogoproto.MessageName(&emissionstypes.OptionalParams{}))
}

func TestUnknownProtoWalksV9InsertWorkerPayload(t *testing.T) {
	msg := &v9.InsertWorkerPayloadRequest{
		Sender: "allo1sender",
		WorkerDataBundle: &v9.InputWorkerDataBundle{
			Worker:  "allo1worker",
			Nonce:   &v3.Nonce{BlockHeight: 1},
			TopicId: 1,
			InferenceForecastsBundle: &v9.InputInferenceForecastBundle{
				Inference: &v9.InputInference{
					TopicId:     1,
					BlockHeight: 1,
					Inferer:     "allo1worker",
					Value:       "1.0",
					ExtraData:   nil,
					Proof:       "",
				},
				Forecast: nil,
			},
			InferencesForecastsBundleSignature: []byte{0x01},
			Pubkey:                             "pubkey",
		},
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)

	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v9.InsertWorkerPayloadRequest)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}
