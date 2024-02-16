package keeper_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	state "github.com/allora-network/allora-chain/x/emissions"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
)

func TestTopicIdListValueCodec_Encode(t *testing.T) {
	cases := []struct {
		name  string
		value []uint64
		want  []byte
	}{
		{
			name:  "EmptyValue",
			value: []uint64{},
			want:  []byte{},
		},
		{
			name:  "SingleValue",
			value: []uint64{123},
			want:  []byte{0, 0, 0, 0, 0, 0, 0, 123},
		},
		{
			name:  "MultipleValues",
			value: []uint64{1, 2, 3, 4, 5},
			want:  []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 5},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := keeper.TopicIdListValue.Encode(tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Errorf("unexpected result, got: %v, want: %v", got, tc.want)
			}
		})
	}
}
func TestTopicIdListValueCodec_Decode(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  []uint64
		err   error
	}{
		{
			name:  "ValidInput",
			input: []byte{0, 0, 0, 0, 0, 0, 0, 123},
			want:  []uint64{123},
			err:   nil,
		},
		{
			name:  "InvalidLength",
			input: []byte{0, 0, 0, 0, 0, 0, 0},
			want:  nil,
			err:   state.ErrTopicIdListValueDecodeInvalidLength,
		},
		{
			name:  "MultipleValues",
			input: []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 5},
			want:  []uint64{1, 2, 3, 4, 5},
			err:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := keeper.TopicIdListValue.Decode(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unexpected result, got: %v, want: %v", got, tc.want)
			}
			if !errors.Is(err, tc.err) {
				t.Errorf("unexpected error, got: %v, want: %v", err, tc.err)
			}
		})
	}
}

func TestTopicIdListValueCodec_Stringify(t *testing.T) {
	cases := []struct {
		name  string
		value []uint64
		want  string
	}{
		{
			name:  "EmptyValue",
			value: []uint64{},
			want:  "[]",
		},
		{
			name:  "SingleValue",
			value: []uint64{123},
			want:  "[123]",
		},
		{
			name:  "MultipleValues",
			value: []uint64{1, 2, 3, 4, 5},
			want:  "[1,2,3,4,5]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := keeper.TopicIdListValue.Stringify(tc.value)
			if got != tc.want {
				t.Errorf("unexpected result, got: %v, want: %v", got, tc.want)
			}
		})
	}
}
func TestTopicIdListValueCodec_EncodeJSON(t *testing.T) {
	cases := []struct {
		name   string
		value  []uint64
		want   []byte
		errMsg string
	}{
		{
			name:   "EmptyValue",
			value:  []uint64{},
			want:   []byte("[]"),
			errMsg: "",
		},
		{
			name:   "SingleValue",
			value:  []uint64{123},
			want:   []byte("[123]"),
			errMsg: "",
		},
		{
			name:   "MultipleValues",
			value:  []uint64{1, 2, 3, 4, 5},
			want:   []byte("[1,2,3,4,5]"),
			errMsg: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := keeper.TopicIdListValue.EncodeJSON(tc.value)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unexpected result, got: %v, want: %v", got, tc.want)
			}
			if tc.errMsg != "" {
				if err == nil {
					t.Errorf("expected error, got: nil, want: %v", tc.errMsg)
				} else if err.Error() != tc.errMsg {
					t.Errorf("unexpected error message, got: %v, want: %v", err.Error(), tc.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error, got: %v, want: nil", err)
				}
			}
		})
	}
}
func TestTopicIdListValueCodec_DecodeJSON(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  []uint64
		err   error
	}{
		{
			name:  "EmptyValue",
			input: []byte("[]"),
			want:  []uint64{},
			err:   nil,
		},
		{
			name:  "SingleValue",
			input: []byte("[123]"),
			want:  []uint64{123},
			err:   nil,
		},
		{
			name:  "MultipleValues",
			input: []byte("[1,2,3,4,5]"),
			want:  []uint64{1, 2, 3, 4, 5},
			err:   nil,
		},
		{
			name:  "InvalidLength",
			input: []byte(""),
			want:  nil,
			err:   state.ErrTopicIdListValueDecodeJsonInvalidLength,
		},
		{
			name:  "InvalidValue",
			input: []byte("[1,2,3,abc,5]"),
			want:  nil,
			err:   fmt.Errorf("strconv.ParseUint: parsing \"abc\": invalid syntax"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := keeper.TopicIdListValue.DecodeJSON(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unexpected result, got: %v, want: %v", got, tc.want)
			}
			if !errors.Is(err, tc.err) {
				if err.Error() != tc.err.Error() {
					t.Errorf("unexpected error, got: %v, want: %v", err, tc.err)
				}
			}
		})
	}
}
