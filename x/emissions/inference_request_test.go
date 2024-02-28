package emissions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestInferenceRequest_GetRequestId(t *testing.T) {
	config := sdk.GetConfig()
	var Bech32PrefixAccAddr = "allo"
	var Bech32PrefixAccPub = Bech32PrefixAccAddr + "pub"
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	inferenceRequest := &emissions.InferenceRequest{
		Sender:               "allo1m4ssnux4kh5pfmjzzkpde0hvxfg0d37mla0pdf",
		Nonce:                0x31,
		TopicId:              0x32,
		Cadence:              0x33,
		MaxPricePerInference: cosmosMath.NewUint(0x34),
		BidAmount:            cosmosMath.NewUint(0x35),
		TimestampValidUntil:  0x36,
		ExtraData:            []byte("B"),
	}

	// expected Data representation is a concatentaion of the public address bytes, nonce bytes, and topicId bytes
	// 20 bytes address
	// 8 bytes nonce
	// 8 bytes topicId
	expectedDataRep := []byte("\xdd\x61\x09\xf0\xd5\xb5\xe8\x14\xee\x42\x15\x82\xdc\xbe\xec\x32\x50\xf6\xc7\xdb") //address
	expectedDataRep = append(expectedDataRep, []byte("\x31\x00\x00\x00\x00\x00\x00\x00")...)                      // nonce
	expectedDataRep = append(expectedDataRep, []byte("\x32\x00\x00\x00\x00\x00\x00\x00")...)                      // topicId
	dataRep, err := inferenceRequest.GetRequestBytes()
	require.NoError(t, err)
	require.Equal(t, expectedDataRep, dataRep)

	// using online sha256 calculator, the hash of the data representation is
	// 0x2babfffe58eb0fdad2830d9b1c4c4a97db9f0682c5530230305daa5967857f37
	expectedRequestId := "0x3ed3fbee9d2b2dd644621e4b681d5dd9675400049bb76baede120847be8c0430"

	requestId, err := inferenceRequest.GetRequestId()
	require.NoError(t, err)

	require.Equal(t, expectedRequestId, requestId)
}

func TestInferenceRequest_GetRequestIdDifferentHash(t *testing.T) {
	config := sdk.GetConfig()
	var Bech32PrefixAccAddr = "allo"
	var Bech32PrefixAccPub = Bech32PrefixAccAddr + "pub"
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	inferenceRequest := &emissions.InferenceRequest{
		Sender:               "allo1m4ssnux4kh5pfmjzzkpde0hvxfg0d37mla0pdf",
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

func TestIsValidRequestId(t *testing.T) {
	testCasesInvalid := [4]string{
		"",
		"0x3ed3fBee9d2b2dd644621e4b681d5dd9675400049bb76baede120847be8c0430",
		"0x3ed3fBee9d2b2dd644621e4b681d5dd9675400049bb76baede120847be8c043O",
		"3ed3fbee9d2b2dd644621e4b681d5dd9675400049bb76baede120847be8c0430",
	}
	testCasesValid := [4]string{
		"0x3ed3fbee9d2b2dd644621e4b681d5dd9675400049bb76baede120847be8c0430",
		"0x2babfffe58eb0fdad2830d9b1c4c4a97db9f0682c5530230305daa5967857f37",
		"0x0000000000000000000000000000000000000000000000000000000000000000",
		"0xfeedfacecafebabedeadbeefbadd1e5f00ba5c0ffee1baddecafbeefbadd1e59",
	}
	for _, testCase := range testCasesInvalid {
		require.False(t, emissions.IsValidRequestId(testCase))
	}
	for _, testCase := range testCasesValid {
		require.True(t, emissions.IsValidRequestId(testCase))
	}
}
