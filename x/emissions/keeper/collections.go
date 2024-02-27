package keeper

import (
	collcodec "cosmossdk.io/collections/codec"
	cosmosMath "cosmossdk.io/math"
)

// This code is a placeholder until https://github.com/cosmos/cosmos-sdk/pull/19164
// is merged and we move to the appropriate new cosmos-sdk version

var UintValue collcodec.ValueCodec[cosmosMath.Uint] = uintValueCodec{}

type uintValueCodec struct{}

func (i uintValueCodec) Encode(value cosmosMath.Uint) ([]byte, error) {
	return value.Marshal()
}

func (i uintValueCodec) Decode(b []byte) (cosmosMath.Uint, error) {
	v := new(cosmosMath.Uint)
	err := v.Unmarshal(b)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return *v, nil
}

func (i uintValueCodec) EncodeJSON(value cosmosMath.Uint) ([]byte, error) {
	return value.MarshalJSON()
}

func (i uintValueCodec) DecodeJSON(b []byte) (cosmosMath.Uint, error) {
	v := new(cosmosMath.Uint)
	err := v.UnmarshalJSON(b)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return *v, nil
}

func (i uintValueCodec) Stringify(value cosmosMath.Uint) string {
	return value.String()
}

func (i uintValueCodec) ValueType() string {
	return "math.Uint"
}
