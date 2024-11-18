package testcommon

import (
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GeneratePrivKey() (cryptotypes.PrivKey, cryptotypes.PubKey, string) {
	algo := hd.Secp256k1

	privKey := secp256k1.GenPrivKey()
	privKeyCrypto := algo.Generate()(privKey.Bytes())
	pubKey := privKeyCrypto.PubKey()

	addressbytes := sdk.AccAddress(pubKey.Address().Bytes())
	address, err := sdk.Bech32ifyAddressBytes("allo", addressbytes)
	if err != nil {
		panic(err)
	}

	return privKeyCrypto, pubKey, address
}
