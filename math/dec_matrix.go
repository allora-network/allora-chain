package math

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DecMatrix []DecArray

var _ sdk.CustomProtobufType = &DecMatrix{}

func (m DecMatrix) Equal(m2 DecMatrix) bool {
	if len(m) != len(m2) {
		return false
	}
	for i := range m {
		if len(m[i]) != len(m2[i]) {
			return false
		}
		for j := range m[i] {
			if !m[i][j].Equal(m2[i][j]) {
				return false
			}
		}
	}
	return true
}

func (m DecMatrix) Marshal() ([]byte, error)  { return m.MarshalJSON() }
func (m *DecMatrix) Unmarshal(b []byte) error { return m.UnmarshalJSON(b) }
func (m DecMatrix) Size() int                 { bz, _ := m.Marshal(); return len(bz) }
func (m DecMatrix) MarshalTo(data []byte) (int, error) {
	bz, err := m.Marshal()
	if err != nil {
		return 0, err
	}
	return copy(data, bz), nil
}

func (m DecMatrix) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, row := range m {
		if i > 0 {
			b.WriteByte(',')
		}
		// row is []Dec => reuse DecArray marshalling logic
		rowBz, err := (DecArray(row)).MarshalJSON()
		if err != nil {
			return nil, err
		}
		b.Write(rowBz)
	}
	b.WriteByte(']')
	return b.Bytes(), nil
}

func (m *DecMatrix) UnmarshalJSON(data []byte) error {
	*m = (*m)[:0]
	dec := json.NewDecoder(bytes.NewReader(data))

	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("read first token: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("expected JSON array")
	}

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return fmt.Errorf("decode row: %w", err)
		}
		var row DecArray
		if err := row.UnmarshalJSON(raw); err != nil {
			return fmt.Errorf("unmarshal row: %w", err)
		}
		*m = append(*m, []Dec(row))
	}

	tok, err = dec.Token()
	if err != nil {
		return fmt.Errorf("read closing token: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != ']' {
		return fmt.Errorf("expected closing ']'")
	}

	tok, err = dec.Token()
	if tok != nil || err != io.EOF {
		return fmt.Errorf("unexpected data after closing token")
	}
	return nil
}
