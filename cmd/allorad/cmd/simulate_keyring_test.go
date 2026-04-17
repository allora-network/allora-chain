package cmd

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func testCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

func TestSimulateKeyring_EmptyUID(t *testing.T) {
	inner := keyring.NewInMemory(testCodec())
	sk := simulateKeyring{inner}

	rec, err := sk.Key("")
	require.NoError(t, err)
	require.NotNil(t, rec)

	pk, ok := rec.PubKey.GetCachedValue().(cryptotypes.PubKey)
	require.True(t, ok)
	require.IsType(t, &secp256k1.PubKey{}, pk) //nolint:exhaustruct // creating an empty Pubkey for the test check only
	require.Len(t, pk.Bytes(), secp256k1.PubKeySize)
}

func TestSimulateKeyring_ExistingKey_Passthrough(t *testing.T) {
	inner := keyring.NewInMemory(testCodec())
	expected, _, err := inner.NewMnemonic("alice", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	require.NoError(t, err)

	sk := simulateKeyring{inner}
	got, err := sk.Key("alice")
	require.NoError(t, err)
	require.Equal(t, expected.Name, got.Name)
}

func TestSimulateKeyring_NonEmptyUID_Passthrough(t *testing.T) {
	inner := keyring.NewInMemory(testCodec())
	sk := simulateKeyring{inner}

	_, err := sk.Key("nonexistent")
	require.Error(t, err, "non-empty UID should delegate to inner keyring and fail for missing key")
}

func TestSimulateKeyring_DelegatesOtherMethods(t *testing.T) {
	inner := keyring.NewInMemory(testCodec())
	sk := simulateKeyring{inner}

	require.Equal(t, inner.Backend(), sk.Backend())

	recs, err := sk.List()
	require.NoError(t, err)
	require.Empty(t, recs)
}
