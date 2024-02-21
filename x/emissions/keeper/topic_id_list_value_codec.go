package keeper

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"strings"

	state "github.com/allora-network/allora-chain/x/emissions"

	collcodec "cosmossdk.io/collections/codec"
)

var TopicIdListValue collcodec.ValueCodec[[]uint64] = topicIdListValueCodec{}

type topicIdListValueCodec struct{}

func (i topicIdListValueCodec) Encode(value []uint64) ([]byte, error) {
	buf := make([]byte, len(value)*8)
	for i, v := range value {
		binary.BigEndian.PutUint64(buf[i*8:], v)
	}
	return buf, nil
}

func (i topicIdListValueCodec) Decode(b []byte) ([]uint64, error) {
	if len(b)%8 != 0 {
		return nil, state.ErrTopicIdListValueDecodeInvalidLength
	}
	value := make([]uint64, len(b)/8)
	for i := 0; i < len(b); i += 8 {
		value[i/8] = binary.BigEndian.Uint64(b[i : i+8])
	}
	return value, nil
}

func (i topicIdListValueCodec) EncodeJSON(value []uint64) ([]byte, error) {
	var retStr string
	retStr = "["
	valueLen := len(value)
	for i, v := range value {
		if i == valueLen-1 {
			retStr += fmt.Sprintf("%d", v)
		} else {
			retStr += fmt.Sprintf("%d,", v)
		}
	}
	retStr += "]"
	return []byte(retStr), nil
}

func (i topicIdListValueCodec) DecodeJSON(b []byte) ([]uint64, error) {
	stringRep := string(b)
	if len(stringRep) < 2 {
		return nil, state.ErrTopicIdListValueDecodeJsonInvalidLength
	}
	if stringRep[0] != '[' || stringRep[len(stringRep)-1] != ']' {
		return nil, state.ErrTopicIdListValueDecodeJsonInvalidFormat
	}
	if len(stringRep) == 2 {
		return []uint64{}, nil
	}
	stringRep = stringRep[1 : len(stringRep)-1]
	splitString := strings.Split(stringRep, ",")
	ret := make([]uint64, len(splitString))
	for i, v := range splitString {
		v = strings.TrimSpace(v)
		r, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		ret[i] = r
	}
	return ret, nil
}

func (i topicIdListValueCodec) Stringify(value []uint64) string {
	var retStr string
	retStr = "["
	valueLen := len(value)
	for i, v := range value {
		if i == valueLen-1 {
			retStr += fmt.Sprintf("%d", v)
		} else {
			retStr += fmt.Sprintf("%d,", v)
		}
	}
	retStr += "]"
	return retStr
}

func (i topicIdListValueCodec) ValueType() string {
	return "[]uint64"
}
