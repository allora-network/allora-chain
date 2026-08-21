package emissionsv2_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	v2 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v2"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func TestV2NestedTypesRegisteredWithGogo(t *testing.T) {
	names := []string{
		"emissions.v2.Nonce",
		"emissions.v2.WorkerDataBundle",
		"emissions.v2.Inference",
		"emissions.v2.Forecast",
		"emissions.v2.ForecastElement",
		"emissions.v2.InferenceForecastBundle",
		"emissions.v2.ReputerValueBundle",
		"emissions.v2.ValueBundle",
		"emissions.v2.OptionalParams",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, gogoproto.MessageType(name), "gogo registry missing %s", name)
		})
	}

	require.Equal(t, "emissions.v10.WorkerDataBundle", gogoproto.MessageName(&emissionstypes.WorkerDataBundle{}))
	require.Equal(t, "emissions.v3.ReputerValueBundle", gogoproto.MessageName(&emissionstypes.ReputerValueBundle{}))
}

func TestUnknownProtoWalksV2InsertWorkerPayload(t *testing.T) {
	msg := &v2.MsgInsertWorkerPayload{
		Sender: "allo1sender",
		WorkerDataBundle: &v2.WorkerDataBundle{
			Worker:  "allo1worker",
			Nonce:   &v2.Nonce{BlockHeight: 1},
			TopicId: 1,
			InferenceForecastsBundle: &v2.InferenceForecastBundle{
				Inference: &v2.Inference{
					TopicId:     1,
					BlockHeight: 1,
					Inferer:     "allo1worker",
					Value:       "1.0",
				},
			},
			InferencesForecastsBundleSignature: []byte{0x01},
			Pubkey:                             "pubkey",
		},
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)
	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v2.MsgInsertWorkerPayload)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}

func TestUnknownProtoWalksV2InsertReputerPayload(t *testing.T) {
	//nolint:exhaustruct // historical walk only needs the nested message graph present
	msg := &v2.MsgInsertReputerPayload{
		Sender: "allo1sender",
		ReputerValueBundle: &v2.ReputerValueBundle{
			ValueBundle: &v2.ValueBundle{
				TopicId:       1,
				Reputer:       "allo1reputer",
				CombinedValue: "1.0",
				NaiveValue:    "1.0",
			},
			Signature: []byte{0x01},
			Pubkey:    "pubkey",
		},
	}

	bz, err := protov2.Marshal(msg)
	require.NoError(t, err)
	err = unknownproto.RejectUnknownFieldsStrict(bz, (*v2.MsgInsertReputerPayload)(nil), codectypes.NewInterfaceRegistry())
	require.NoError(t, err)
}
