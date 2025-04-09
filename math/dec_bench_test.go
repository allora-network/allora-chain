package math_test

import (
	"fmt"
	"testing"

	alloramath "github.com/allora-network/allora-chain/math"
)

// Carries testing dec for logarithms benchmarks
var benchLogData = []alloramath.Dec{
	alloramath.MustNewDecFromString("1.2"),
	alloramath.MustNewDecFromString("1024"),
	alloramath.NewDecFromInt64(2048 * 2048 * 2048 * 2048 * 2048),
	alloramath.MustNewDecFromString("999999999999999999999999999999999999999999999999999999.9122181273612911"),
	alloramath.MustNewDecFromString("0.5632892391219024912482190471290471"),
	alloramath.MustNewDecFromString("0.0000000000000024912482190471290471"),
}

// Carries testing dec for exponential benchmarks
var benchExpData = []alloramath.Dec{
	alloramath.MustNewDecFromString("1.2"),
	alloramath.MustNewDecFromString("1024"),
	alloramath.MustNewDecFromString("0.5632892391219024912482190471290471"),
	alloramath.MustNewDecFromString("0.0000000000000024912482190471290471"),
}

func BenchmarkLn(b *testing.B) {
	for _, test := range benchLogData {
		b.Run(fmt.Sprintf("input_%v", test), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StartTimer()
				_, err := alloramath.Ln(test)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
			}
		})
	}
}

func BenchmarkLog10(b *testing.B) {
	for _, test := range benchLogData {
		b.Run(fmt.Sprintf("input_%v", test), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StartTimer()
				_, err := alloramath.Log10(test)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
			}
		})
	}
}

func BenchmarkExp(b *testing.B) {
	for _, test := range benchExpData {
		b.Run(fmt.Sprintf("input_%v", test), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StartTimer()
				_, err := alloramath.Exp(test)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
			}
		})
	}
}

func BenchmarkPow(b *testing.B) {
	for _, d := range benchExpData {
		for _, e := range benchExpData {
			b.Run(fmt.Sprintf("input_%v_%v", d, e), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StartTimer()
					_, err := alloramath.Pow(d, e)
					if err != nil {
						b.Fatal(err)
					}
					b.StopTimer()
				}
			})
		}
	}
}
