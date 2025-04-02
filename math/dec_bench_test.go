package math_test

import (
	"testing"

	alloramath "github.com/allora-network/allora-chain/math"
)

func BenchmarkLn(b *testing.B) {
	tests := []alloramath.Dec{
		alloramath.MustNewDecFromString("1.2"),
		alloramath.MustNewDecFromString("1.234"),
		alloramath.MustNewDecFromString("1024"),
		alloramath.NewDecFromInt64(2048 * 2048 * 2048 * 2048 * 2048),
		alloramath.MustNewDecFromString("999999999999999999999999999999999999999999999999999999.9122181273612911"),
		alloramath.MustNewDecFromString("0.5632892391219024912482190471290471"),
		alloramath.MustNewDecFromString("0.0000000391219024912482190471290471"),
	}

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		for _, test := range tests {
			alloramath.Ln(test)
		}
		b.StopTimer()
	}
}
