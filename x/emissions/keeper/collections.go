package keeper

import (
	collcodec "cosmossdk.io/collections/codec"
	math "cosmossdk.io/math"
)

// This code is a placeholder until https://github.com/cosmos/cosmos-sdk/pull/19164
// is merged and we move to the appropriate new cosmos-sdk version

var UintValue collcodec.ValueCodec[math.Uint] = uintValueCodec{}

type uintValueCodec struct{}

func (i uintValueCodec) Encode(value math.Uint) ([]byte, error) {
	return value.Marshal()
}

func (i uintValueCodec) Decode(b []byte) (math.Uint, error) {
	v := new(math.Uint)
	err := v.Unmarshal(b)
	if err != nil {
		return math.Uint{}, err
	}
	return *v, nil
}

func (i uintValueCodec) EncodeJSON(value math.Uint) ([]byte, error) {
	return value.MarshalJSON()
}

func (i uintValueCodec) DecodeJSON(b []byte) (math.Uint, error) {
	v := new(math.Uint)
	err := v.UnmarshalJSON(b)
	if err != nil {
		return math.Uint{}, err
	}
	return *v, nil
}

func (i uintValueCodec) Stringify(value math.Uint) string {
	return value.String()
}

func (i uintValueCodec) ValueType() string {
	return "math.Uint"
}
