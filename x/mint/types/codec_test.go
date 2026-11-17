package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/x/mint/types"
)

func TestRegisterLegacyAminoCodecDoesNotPanic(t *testing.T) {
	cdc := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		types.RegisterLegacyAminoCodec(cdc)
	})
}
