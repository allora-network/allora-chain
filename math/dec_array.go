package math

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DecArray is an array of Dec values.
// Its purpose is to allow two-dimensional Dec arrays to be properly marshaled to JSON, especially when used in events.
type DecArray []Dec

var _ sdk.CustomProtobufType = &DecArray{}

func (d DecArray) Equal(d2 DecArray) bool {
	if len(d) != len(d2) {
		return false
	}
	for i := range d {
		if !d[i].Equal(d2[i]) {
			return false
		}
	}
	return true
}

func (d DecArray) Marshal() ([]byte, error) {
	return d.MarshalJSON()
}

func (d DecArray) MarshalTo(data []byte) (n int, err error) {
	bz, err := d.Marshal()
	if err != nil {
		return 0, err
	}

	return copy(data, bz), nil
}

func (d *DecArray) Unmarshal(data []byte) error {
	return d.UnmarshalJSON(data)
}

func (d DecArray) Size() int {
	bz, _ := d.Marshal()
	return len(bz)
}

func (d DecArray) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	b.WriteByte('[')
	for i, dec := range d {
		if i > 0 {
			b.WriteByte(',')
		}
		decJson, err := dec.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if _, err := b.Write(decJson); err != nil {
			return nil, err
		}
	}
	b.WriteByte(']')
	return b.Bytes(), nil
}

func (d *DecArray) UnmarshalJSON(data []byte) error {
	*d = (*d)[:0]
	decoder := json.NewDecoder(bytes.NewReader(data))

	// starts with '['
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("read first token: %w", err)
	}
	if d, ok := token.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("expected JSON array")
	}

	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return fmt.Errorf("decode dec: %w", err)
		}

		var dec Dec
		if err := dec.UnmarshalJSON(raw); err != nil {
			return fmt.Errorf("unmarshal dec: %w", err)
		}
		*d = append(*d, dec)
	}

	// ends with ']'
	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("read closing token: %w", err)
	}
	if d, ok := token.(json.Delim); !ok || d != ']' {
		return fmt.Errorf("expected closing ']'")
	}

	token, err = decoder.Token()
	if token != nil || err != io.EOF {
		return fmt.Errorf("unexpected data after closing token")
	}

	return nil
}
