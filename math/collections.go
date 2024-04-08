package math

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

var LegacyDecValue collcodec.ValueCodec[cosmosMath.LegacyDec] = legacyDecValueCodec{}

type legacyDecValueCodec struct{}

func (i legacyDecValueCodec) Encode(value cosmosMath.LegacyDec) ([]byte, error) {
	return value.Marshal()
}

func (i legacyDecValueCodec) Decode(b []byte) (cosmosMath.LegacyDec, error) {
	v := new(cosmosMath.LegacyDec)
	err := v.Unmarshal(b)
	if err != nil {
		return cosmosMath.LegacyDec{}, err
	}
	return *v, nil
}

func (i legacyDecValueCodec) EncodeJSON(value cosmosMath.LegacyDec) ([]byte, error) {
	return value.MarshalJSON()
}

func (i legacyDecValueCodec) DecodeJSON(b []byte) (cosmosMath.LegacyDec, error) {
	v := new(cosmosMath.LegacyDec)
	err := v.UnmarshalJSON(b)
	if err != nil {
		return cosmosMath.LegacyDec{}, err
	}
	return *v, nil
}

func (i legacyDecValueCodec) Stringify(value cosmosMath.LegacyDec) string {
	return value.String()
}

func (i legacyDecValueCodec) ValueType() string {
	return "math.LegacyDec"
}

var DecValue collcodec.ValueCodec[Dec] = decValueCodec{}

type decValueCodec struct{}

func (i decValueCodec) Encode(value Dec) ([]byte, error) {
	return value.Marshal()
}

func (i decValueCodec) Decode(b []byte) (Dec, error) {
	v := new(Dec)
	err := v.Unmarshal(b)
	if err != nil {
		return Dec{}, err
	}
	return *v, nil
}

func (i decValueCodec) EncodeJSON(value Dec) ([]byte, error) {
	return value.MarshalJSON()
}

func (i decValueCodec) DecodeJSON(b []byte) (Dec, error) {
	v := new(Dec)
	err := v.UnmarshalJSON(b)
	if err != nil {
		return Dec{}, err
	}
	return *v, nil
}

func (i decValueCodec) Stringify(value Dec) string {
	return value.String()
}

func (i decValueCodec) ValueType() string {
	return "AlloraDec"
}
