package emissions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions"
)

func TestInferenceRequest_GetRequestId(t *testing.T) {
	inferenceRequest := &emissions.InferenceRequest{
		Sender:               "A",
		Nonce:                0x31,
		TopicId:              0x32,
		Cadence:              0x33,
		MaxPricePerInference: cosmosMath.NewUint(0x34),
		BidAmount:            cosmosMath.NewUint(0x35),
		TimestampValidUntil:  0x36,
		ExtraData:            []byte("B"),
	}

	// expected data representation of inferenceRequest is
	// 0a 01 [41] 10 [31] 18 [32] 20 [33] 2a 02 35 32 32 02 35 33 38 [36] 42 01 [42]
	//      Sender  Nonce  TopicId  Cadence     big int stuff      Timestamp   ExtraData
	// the first byte is the tag, the second byte is the length of the data
	// the rest of the bytes are the data
	// for Uint, the big number is encoded some confusing way with length and size and stuff
	expectedDataRep := []byte("\x0a\x01\x41\x10\x31\x18\x32\x20\x33\x2a\x02\x35\x32\x32\x02\x35\x33\x38\x36\x42\x01\x42")
	dataRep, err := inferenceRequest.Marshal()
	require.NoError(t, err)
	require.Equal(t, expectedDataRep, dataRep)

	// using online sha256 calculator, the hash of the data representation is
	// 0x2babfffe58eb0fdad2830d9b1c4c4a97db9f0682c5530230305daa5967857f37
	expectedRequestId := "0x2babfffe58eb0fdad2830d9b1c4c4a97db9f0682c5530230305daa5967857f37"

	requestId, err := inferenceRequest.GetRequestId()
	require.NoError(t, err)

	require.Equal(t, expectedRequestId, requestId)
}

func TestInferenceRequest_GetRequestIdDifferentHash(t *testing.T) {
	inferenceRequest := &emissions.InferenceRequest{
		Sender:               "A",
		Nonce:                0x31,
		TopicId:              0x32,
		Cadence:              0x33,
		MaxPricePerInference: cosmosMath.NewUint(0x34),
		BidAmount:            cosmosMath.NewUint(0x35),
		TimestampValidUntil:  0x37,
		ExtraData:            []byte("B"),
	}
	expectedRequestId := "0x2babfffe58eb0fdad2830d9b1c4c4a97db9f0682c5530230305daa5967857f37"

	requestId, err := inferenceRequest.GetRequestId()
	require.NoError(t, err)
	require.NotEqual(t, expectedRequestId, requestId)
}
