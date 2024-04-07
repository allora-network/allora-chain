package keeper

import (
	collcodec "cosmossdk.io/collections/codec"
	cosmosMath "cosmossdk.io/math"
)

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
