package types

// NonceVersionV0 is the V0 nonce version, not used anymore but was used when the block height was used as payload.
const NonceVersionV0 uint8 = 0

// NonceVersionV1 is the V1 nonce version, the currently used format in which the payload is incremental, and zero based.
const NonceVersionV1 uint8 = 1

const noncePayloadMask uint64 = 1<<56 - 1

// Nonce represents an opaque nonce with versionning support.
// The nonce is a 64-bit unsigned integer where the highest 8 bits represent
// the version and the remaining 56 bits contains the payload.
type NonceV2 uint64

func newNonce(version uint8, payload uint64) NonceV2 {
	if payload > noncePayloadMask {
		panic("nonce payload exceeds maximum size")
	}

	return NonceV2(uint64(version)<<56 | payload)
}

// NewNonceFromUint64 creates a new Nonce with version V1 and the given payload.
// It panics if the payload exceeds the maximum size of uint56.
func NewNonceFromUint64(n uint64) NonceV2 {
	return newNonce(NonceVersionV1, n)
}

// ZeroNonce returns a nonce with version V1 and payload 0.
func ZeroNonce() NonceV2 {
	return newNonce(NonceVersionV1, 0)
}

// NextNonce returns the next nonce by incrementing the payload of the given nonce.
func NextNonce(n NonceV2) NonceV2 {
	return newNonce(NonceVersionV1, n.Payload()+1)
}

// NextNonce returns the next nonce by incrementing the payload of the given nonce.
func (n NonceV2) NextNonce() NonceV2 {
	return NextNonce(n)
}

// Version returns the version of the nonce.
func (n NonceV2) Version() uint8 {
	// Version occupies the high 8 bits; the shift result always fits in uint8.
	return uint8((uint64(n) >> 56) & 0xFF) //nolint:gosec // G115: high 8 bits of a uint64
}

// Payload returns the payload of the nonce.
func (n NonceV2) Payload() uint64 {
	return uint64(n) & noncePayloadMask
}
