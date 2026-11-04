package cmd

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

// Compile-time check that simulateKeyring satisfies keyring.Keyring.
var _ keyring.Keyring = simulateKeyring{} //nolint:exhaustruct // creating an empty simulateKeyring is valid

// simulateKeyring wraps a keyring.Keyring to gracefully handle empty key name
// lookups during transaction simulation.
//
// When --dry-run is combined with --gas auto, the Cosmos SDK clears the key
// name (FromName) in simulate mode (client.GetFromFields returns "" for the
// name when ctx.Simulate is true). However, the gas estimation code path
// (Factory.getSimPK) still attempts keybase.Key(fromName) when
// simulateAndExecute is true. With an empty fromName, the underlying keystore
// appends the ".info" suffix to an empty string and looks up ".info", which
// does not exist — producing the error: ".info: key not found".
//
// This wrapper intercepts Key("") calls and returns a default offline record
// with a placeholder secp256k1 public key, which is the same default that
// getSimPK would use if it skipped the keyring lookup. The chain does not
// verify signatures during gas simulation, so using a placeholder key is safe.
//
// Ref: https://github.com/cosmos/cosmos-sdk/issues/11283
type simulateKeyring struct {
	keyring.Keyring
}

func (sk simulateKeyring) Key(uid string) (*keyring.Record, error) {
	if uid == "" {
		pk := &secp256k1.PubKey{Key: make([]byte, secp256k1.PubKeySize)}
		return keyring.NewOfflineRecord("", pk)
	}
	return sk.Keyring.Key(uid)
}
