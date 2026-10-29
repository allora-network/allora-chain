package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNonceFromUint64(t *testing.T) {
	n := NewNonceFromUint64(0)
	require.Equal(t, NonceVersionV1, n.Version())
	require.Equal(t, uint64(0), n.Payload())

	n = NewNonceFromUint64(19345)
	require.Equal(t, NonceVersionV1, n.Version())
	require.Equal(t, uint64(19345), n.Payload())

	require.NotPanics(t, func() { NewNonceFromUint64(noncePayloadMask) })
	require.Panics(t, func() { NewNonceFromUint64(noncePayloadMask + 1) })
	require.Panics(t, func() { NewNonceFromUint64(noncePayloadMask + 100) })
}

func TestZeroNonce(t *testing.T) {
	n := ZeroNonce()
	require.Equal(t, NonceVersionV1, n.Version())
	require.Equal(t, uint64(0), n.Payload())
}

func TestNextNonce(t *testing.T) {
	n := NextNonce(ZeroNonce())
	require.Equal(t, NonceVersionV1, n.Version())
	require.Equal(t, uint64(1), n.Payload())

	n = NextNonce(NewNonceFromUint64(19345))
	require.Equal(t, NonceVersionV1, n.Version())
	require.Equal(t, NewNonceFromUint64(19346), n)

	// A V0 nonce is incremented and re-tagged as V1.
	v0 := NonceV2(0x0000000000000005)
	n = v0.NextNonce()
	require.Equal(t, NonceVersionV1, n.Version())
	require.Equal(t, uint64(6), n.Payload())
}

func TestVersion(t *testing.T) {
	n := NonceV2(0x0100000000000001)
	require.Equal(t, NonceVersionV1, n.Version())

	n = NonceV2(0x0000000000000001)
	require.Equal(t, NonceVersionV0, n.Version())

	n = NonceV2(0x0110001000101001)
	require.Equal(t, NonceVersionV1, n.Version())
}

func TestPayload(t *testing.T) {
	n := NonceV2(0x0100000000000001)
	require.Equal(t, uint64(1), n.Payload())

	n = NonceV2(0x0000000000000001)
	require.Equal(t, uint64(1), n.Payload())

	n = NonceV2(0x0110001000101001)
	require.Equal(t, uint64(0x10001000101001), n.Payload())
}
