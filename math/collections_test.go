package math_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestLegacyDecEncoding(t *testing.T) {
	legacyDec := sdkmath.LegacyMustNewDecFromStr("1.25")

	encoded, err := alloraMath.LegacyDecValue.Encode(legacyDec)
	require.NoError(t, err)
	require.Equal(t, []byte("1250000000000000000"), encoded)

	decoded, err := alloraMath.LegacyDecValue.Decode(encoded)
	require.NoError(t, err)

	require.Equal(t, legacyDec, decoded)
}

func TestLegacyDecEncodingJSON(t *testing.T) {
	legacyDec := sdkmath.LegacyMustNewDecFromStr("1.25")

	encoded, err := alloraMath.LegacyDecValue.EncodeJSON(legacyDec)
	require.NoError(t, err)
	require.Equal(t, []byte("\"1.250000000000000000\""), encoded)
	decoded, err := alloraMath.LegacyDecValue.DecodeJSON(encoded)
	require.NoError(t, err)

	require.Equal(t, legacyDec, decoded)
}

func TestLegacyDecEncodingStringify(t *testing.T) {
	legacyDec := sdkmath.LegacyMustNewDecFromStr("1.25")
	result := alloraMath.LegacyDecValue.Stringify(legacyDec)
	require.Equal(t, "1.250000000000000000", result)
}

func TestLegacyDecEncodingValueType(t *testing.T) {
	require.Equal(t, "math.LegacyDec", alloraMath.LegacyDecValue.ValueType())
}

func TestDecEncoding(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("1.25")

	encoded, err := alloraMath.DecValue.Encode(dec)
	require.NoError(t, err)
	require.Equal(t, []byte("1.25"), encoded)
}

func TestDecEncodingJSON(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("1.25")
	encoded, err := alloraMath.DecValue.EncodeJSON(dec)
	require.NoError(t, err)
	require.Equal(t, []byte("\"1.25\""), encoded)
}

func TestDecEncodingStringify(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("1.25")
	result := alloraMath.DecValue.Stringify(dec)
	require.Equal(t, "1.25", result)
}

func TestDecEncodingValueType(t *testing.T) {
	require.Equal(t, "AlloraDec", alloraMath.DecValue.ValueType())
}
