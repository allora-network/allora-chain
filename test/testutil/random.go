package testutil

import (
	cryptorand "crypto/rand"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

func MustRandomString(n int) string {
	return string(MustRandomBytes(n))
}

func MustRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := cryptorand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func RandomValueOfType[T any](t *testing.T) T {
	t.Helper()

	var result T
	switch any(result).(type) {
	case string:
		return any(MustRandomString(10)).(T) //nolint:forcetypeassert
	case []byte:
		return any(MustRandomBytes(10)).(T) //nolint:forcetypeassert
	case int:
		return any(rand.Int()).(T) //nolint:gosec,forcetypeassert
	case int16:
		return any(int16(rand.IntN(math.MaxInt16))).(T) //nolint:gosec,forcetypeassert
	case int32:
		return any(rand.Int32()).(T) //nolint:gosec,forcetypeassert
	case int64:
		return any(rand.Int64()).(T) //nolint:gosec,forcetypeassert
	case uint:
		return any(uint(rand.Uint64())).(T) //nolint:gosec,forcetypeassert
	case uint16:
		return any(uint16(rand.UintN(math.MaxUint16))).(T) //nolint:gosec,forcetypeassert
	case uint32:
		return any(rand.Uint32()).(T) //nolint:gosec,forcetypeassert
	case uint64:
		return any(rand.Uint64()).(T) //nolint:gosec,forcetypeassert
	case float64:
		return any(rand.Float64()).(T) //nolint:gosec,forcetypeassert
	case bool:
		return any(rand.IntN(2) == 1).(T) //nolint:gosec,forcetypeassert
	default:
		t.Fatalf("Unsupported type: %v", reflect.TypeOf(result))
		return result // This line will never be reached, but is needed for compilation
	}
}
