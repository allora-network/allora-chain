package keeper

import (
	"encoding/json"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
)

// This is a copy of the cosmos collections triple.go made into a quadruple.go
// cosmos collections.go: https://github.com/cosmos/cosmos-sdk/blob/main/collections/triple.go

// Quadruple defines a multipart key composed of four keys.
type Quadruple[K1, K2, K3, K4 any] struct {
	k1 *K1
	k2 *K2
	k3 *K3
	k4 *K4
}

// Join4 instantiates a new Quadruple instance composed of the three provided keys, in order.
func Join4[K1, K2, K3, K4 any](k1 K1, k2 K2, k3 K3, k4 K4) Quadruple[K1, K2, K3, K4] {
	return Quadruple[K1, K2, K3, K4]{&k1, &k2, &k3, &k4}
}

// K1 returns the first part of the key. If nil, the zero value is returned.
func (t Quadruple[K1, K2, K3, K4]) K1() (x K1) {
	if t.k1 != nil {
		return *t.k1
	}
	return x
}

// K2 returns the second part of the key. If nil, the zero value is returned.
func (t Quadruple[K1, K2, K3, K4]) K2() (x K2) {
	if t.k2 != nil {
		return *t.k2
	}
	return x
}

// K3 returns the third part of the key. If nil, the zero value is returned.
func (t Quadruple[K1, K2, K3, K4]) K3() (x K3) {
	if t.k3 != nil {
		return *t.k3
	}
	return x
}

// K4 returns the fourth part of the key. If nil, the zero value is returned.
func (t Quadruple[K1, K2, K3, K4]) K4() (x K4) {
	if t.k4 != nil {
		return *t.k4
	}
	return x
}

// QuadrupleSinglePrefix creates a new Quadruple instance composed only of the first part of the key.
func QuadrupleSinglePrefix[K1, K2, K3, K4 any](k1 K1) Quadruple[K1, K2, K3, K4] {
	return Quadruple[K1, K2, K3, K4]{k1: &k1} //nolint:exhaustruct
}

// QuadrupleDoublePrefix creates a new Quadruple instance composed only of the first two parts of the key.
func QuadrupleDoublePrefix[K1, K2, K3, K4 any](k1 K1, k2 K2) Quadruple[K1, K2, K3, K4] {
	return Quadruple[K1, K2, K3, K4]{k1: &k1, k2: &k2} //nolint:exhaustruct
}

// QuadrupleTriplePrefix creates a new Quadruple instance composed only of the first three parts of the key.
func QuadrupleTriplePrefix[K1, K2, K3, K4 any](k1 K1, k2 K2, k3 K3) Quadruple[K1, K2, K3, K4] {
	return Quadruple[K1, K2, K3, K4]{k1: &k1, k2: &k2, k3: &k3} //nolint:exhaustruct
}

// QuadrupleKeyCodec instantiates a new KeyCodec instance that can encode the Quadruple, given
// the KeyCodecs of the three parts of the key, in order.
func QuadrupleKeyCodec[K1, K2, K3, K4 any](
	keyCodec1 codec.KeyCodec[K1],
	keyCodec2 codec.KeyCodec[K2],
	keyCodec3 codec.KeyCodec[K3],
	keyCodec4 codec.KeyCodec[K4],
) codec.KeyCodec[Quadruple[K1, K2, K3, K4]] {
	return quadrupleKeyCodec[K1, K2, K3, K4]{
		keyCodec1: keyCodec1,
		keyCodec2: keyCodec2,
		keyCodec3: keyCodec3,
		keyCodec4: keyCodec4,
	}
}

type quadrupleKeyCodec[K1, K2, K3, K4 any] struct {
	keyCodec1 codec.KeyCodec[K1]
	keyCodec2 codec.KeyCodec[K2]
	keyCodec3 codec.KeyCodec[K3]
	keyCodec4 codec.KeyCodec[K4]
}

type jsonQuadrupleKey [4]json.RawMessage

func (t quadrupleKeyCodec[K1, K2, K3, K4]) EncodeJSON(value Quadruple[K1, K2, K3, K4]) ([]byte, error) {
	json1, err := t.keyCodec1.EncodeJSON(*value.k1)
	if err != nil {
		return nil, err
	}

	json2, err := t.keyCodec2.EncodeJSON(*value.k2)
	if err != nil {
		return nil, err
	}

	json3, err := t.keyCodec3.EncodeJSON(*value.k3)
	if err != nil {
		return nil, err
	}

	json4, err := t.keyCodec4.EncodeJSON(*value.k4)
	if err != nil {
		return nil, err
	}

	return json.Marshal(jsonQuadrupleKey{json1, json2, json3, json4})
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) DecodeJSON(b []byte) (Quadruple[K1, K2, K3, K4], error) {
	var jsonKey jsonQuadrupleKey
	err := json.Unmarshal(b, &jsonKey)
	if err != nil {
		return Quadruple[K1, K2, K3, K4]{}, err
	}

	key1, err := t.keyCodec1.DecodeJSON(jsonKey[0])
	if err != nil {
		return Quadruple[K1, K2, K3, K4]{}, err
	}

	key2, err := t.keyCodec2.DecodeJSON(jsonKey[1])
	if err != nil {
		return Quadruple[K1, K2, K3, K4]{}, err
	}

	key3, err := t.keyCodec3.DecodeJSON(jsonKey[2])
	if err != nil {
		return Quadruple[K1, K2, K3, K4]{}, err
	}

	key4, err := t.keyCodec4.DecodeJSON(jsonKey[3])
	if err != nil {
		return Quadruple[K1, K2, K3, K4]{}, err
	}

	return Join4(key1, key2, key3, key4), nil
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) Stringify(key Quadruple[K1, K2, K3, K4]) string {
	b := new(strings.Builder)
	b.WriteByte('(')
	if key.k1 != nil {
		b.WriteByte('"')
		b.WriteString(t.keyCodec1.Stringify(*key.k1))
		b.WriteByte('"')
	} else {
		b.WriteString("<nil>")
	}

	b.WriteString(", ")
	if key.k2 != nil {
		b.WriteByte('"')
		b.WriteString(t.keyCodec2.Stringify(*key.k2))
		b.WriteByte('"')
	} else {
		b.WriteString("<nil>")
	}

	b.WriteString(", ")
	if key.k3 != nil {
		b.WriteByte('"')
		b.WriteString(t.keyCodec3.Stringify(*key.k3))
		b.WriteByte('"')
	} else {
		b.WriteString("<nil>")
	}

	b.WriteString(", ")
	if key.k4 != nil {
		b.WriteByte('"')
		b.WriteString(t.keyCodec4.Stringify(*key.k4))
		b.WriteByte('"')
	} else {
		b.WriteString("<nil>")
	}

	b.WriteByte(')')
	return b.String()
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) KeyType() string {
	return fmt.Sprintf(
		"Quadruple[%s,%s,%s,%s]",
		t.keyCodec1.KeyType(),
		t.keyCodec2.KeyType(),
		t.keyCodec3.KeyType(),
		t.keyCodec4.KeyType(),
	)
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) Encode(buffer []byte, key Quadruple[K1, K2, K3, K4]) (int, error) {
	writtenTotal := 0
	if key.k1 != nil {
		written, err := t.keyCodec1.EncodeNonTerminal(buffer, *key.k1)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	if key.k2 != nil {
		written, err := t.keyCodec2.EncodeNonTerminal(buffer[writtenTotal:], *key.k2)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	if key.k3 != nil {
		written, err := t.keyCodec3.EncodeNonTerminal(buffer[writtenTotal:], *key.k3)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	if key.k4 != nil {
		written, err := t.keyCodec4.Encode(buffer[writtenTotal:], *key.k4)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	return writtenTotal, nil
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) Decode(buffer []byte) (int, Quadruple[K1, K2, K3, K4], error) {
	readTotal := 0
	read, key1, err := t.keyCodec1.DecodeNonTerminal(buffer)
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	read, key2, err := t.keyCodec2.DecodeNonTerminal(buffer[readTotal:])
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	read, key3, err := t.keyCodec3.DecodeNonTerminal(buffer[readTotal:])
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	read, key4, err := t.keyCodec4.Decode(buffer[readTotal:])
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	return readTotal, Join4(key1, key2, key3, key4), nil
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) Size(key Quadruple[K1, K2, K3, K4]) int {
	size := 0
	if key.k1 != nil {
		size += t.keyCodec1.SizeNonTerminal(*key.k1)
	}
	if key.k2 != nil {
		size += t.keyCodec2.SizeNonTerminal(*key.k2)
	}
	if key.k3 != nil {
		size += t.keyCodec3.SizeNonTerminal(*key.k3)
	}
	if key.k4 != nil {
		size += t.keyCodec4.Size(*key.k4)
	}
	return size
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) EncodeNonTerminal(buffer []byte, key Quadruple[K1, K2, K3, K4]) (int, error) {
	writtenTotal := 0
	if key.k1 != nil {
		written, err := t.keyCodec1.EncodeNonTerminal(buffer, *key.k1)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	if key.k2 != nil {
		written, err := t.keyCodec2.EncodeNonTerminal(buffer[writtenTotal:], *key.k2)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	if key.k3 != nil {
		written, err := t.keyCodec3.EncodeNonTerminal(buffer[writtenTotal:], *key.k3)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	if key.k4 != nil {
		written, err := t.keyCodec4.EncodeNonTerminal(buffer[writtenTotal:], *key.k4)
		if err != nil {
			return 0, err
		}
		writtenTotal += written
	}
	return writtenTotal, nil
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) DecodeNonTerminal(buffer []byte) (int, Quadruple[K1, K2, K3, K4], error) {
	readTotal := 0
	read, key1, err := t.keyCodec1.DecodeNonTerminal(buffer)
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	read, key2, err := t.keyCodec2.DecodeNonTerminal(buffer[readTotal:])
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	read, key3, err := t.keyCodec3.DecodeNonTerminal(buffer[readTotal:])
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	read, key4, err := t.keyCodec4.DecodeNonTerminal(buffer[readTotal:])
	if err != nil {
		return 0, Quadruple[K1, K2, K3, K4]{}, err
	}
	readTotal += read
	return readTotal, Join4(key1, key2, key3, key4), nil
}

func (t quadrupleKeyCodec[K1, K2, K3, K4]) SizeNonTerminal(key Quadruple[K1, K2, K3, K4]) int {
	size := 0
	if key.k1 != nil {
		size += t.keyCodec1.SizeNonTerminal(*key.k1)
	}
	if key.k2 != nil {
		size += t.keyCodec2.SizeNonTerminal(*key.k2)
	}
	if key.k3 != nil {
		size += t.keyCodec3.SizeNonTerminal(*key.k3)
	}
	if key.k4 != nil {
		size += t.keyCodec4.SizeNonTerminal(*key.k4)
	}
	return size
}

// NewSinglePrefixUntilQuadrupleRange defines a collection query which ranges until the provided Pair prefix.
// Unstable: this API might change in the future.
func NewSinglePrefixUntilQuadrupleRange[K1, K2, K3, K4 any](k1 K1) collections.Ranger[Quadruple[K1, K2, K3, K4]] {
	key := QuadrupleSinglePrefix[K1, K2, K3, K4](k1)
	r := &collections.Range[Quadruple[K1, K2, K3, K4]]{}
	r = r.EndExclusive(key)
	return r
}

// NewSinglePrefixedQuadrupleRange provides a Range for all keys prefixed with the given
// first part of the Quadruple key.
func NewSinglePrefixedQuadrupleRange[K1, K2, K3, K4 any](k1 K1) collections.Ranger[Quadruple[K1, K2, K3, K4]] {
	key := QuadrupleSinglePrefix[K1, K2, K3, K4](k1)
	r := &collections.Range[Quadruple[K1, K2, K3, K4]]{}
	r = r.Prefix(key)
	return r
}

// NewDoublePrefixedQuadrupleRange provides a Range for all keys prefixed with the given
// first and second parts of the Quadruple key.
func NewDoublePrefixedQuadrupleRange[K1, K2, K3, K4 any](k1 K1, k2 K2) collections.Ranger[Quadruple[K1, K2, K3, K4]] {
	key := QuadrupleDoublePrefix[K1, K2, K3, K4](k1, k2)
	r := &collections.Range[Quadruple[K1, K2, K3, K4]]{}
	r = r.Prefix(key)
	return r
}

// NewTriplePrefixedQuadrupleRange provides a Range for all keys prefixed with the given
// first, second, and third parts of the Quadruple key.
func NewTriplePrefixedQuadrupleRange[K1, K2, K3, K4 any](k1 K1, k2 K2, k3 K3) collections.Ranger[Quadruple[K1, K2, K3, K4]] {
	key := QuadrupleTriplePrefix[K1, K2, K3, K4](k1, k2, k3)
	r := &collections.Range[Quadruple[K1, K2, K3, K4]]{}
	r = r.Prefix(key)
	return r
}
