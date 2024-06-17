package types

import (
	"encoding/json"

	collcodec "cosmossdk.io/collections/codec"
)

type BlockHeight = int64
type TopicId = uint64

var StringSliceValue collcodec.ValueCodec[[]string] = stringSliceValue{}

type stringSliceValue struct{}

func (i stringSliceValue) Encode(value []string) ([]byte, error) {
	return i.EncodeJSON(value)
}

func (i stringSliceValue) Decode(bz []byte) ([]string, error) {
	return i.DecodeJSON(bz)
}

func (i stringSliceValue) EncodeJSON(value []string) ([]byte, error) {
	return json.Marshal(value)
}

func (i stringSliceValue) DecodeJSON(bz []byte) ([]string, error) {
	var value []string
	err := json.Unmarshal(bz, &value)
	return value, err
}

func (i stringSliceValue) Stringify(value []string) string {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return string(out)
}

func (i stringSliceValue) ValueType() string {
	return "[]string"
}
