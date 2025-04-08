package math

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cockroachdb/apd/v3"
)

var (
	logOfEbase2 = MustNewDecFromString("1.442695040888963407359924681001892137")
	// 0.5 with sufficient precision digit to allow right shifting keeping dec128 precision
	log2B apd.Decimal
)

func init() {
	b, _, err := apd.NewFromString("0.5")
	if err != nil {
		panic(err)
	}
	enforceDecimalPrecision(b, dec128Context.Precision+2)
	log2B = *b
}

// Log10 returns a new Dec with the value of the base 10 logarithm of x, without mutating x.
func Log10(x Dec) (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Log10 a NaN %s", x.String())
	}
	var z Dec
	_, err := dec128Context.Log10(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Log10 result is NaN %s", x.String())
	}
	if err != nil {
		return z, errorsmod.Wrapf(err, "decimal base 10 logarithm error %s", x.String())
	}
	return z, nil
}

// Ln returns a new Dec with the value of the natural logarithm of x, without mutating x.
func Ln(x Dec) (Dec, error) {
	log2x, err := Log2(x)
	if err != nil {
		return Dec{}, errorsmod.Wrap(ErrNaN, "decimal natural logarithm error")
	}

	return log2x.Quo(logOfEbase2)
}

// Log2 returns a new Dec with the value of the base 2 logarithm of x, without mutating x.
// It implements the algorithm described in http://www.claysturner.com/dsp/binarylogarithm.pdf
// And inspired from Osmosis implementation: https://github.com/osmosis-labs/osmosis/blob/927d940cf569e25e5a21911a77d5c81a0c5a0dc6/osmomath/decimal.go#L1070
func Log2(x Dec) (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Log2 a NaN %s", x.String())
	}
	if x.dec.Sign() <= 0 {
		return Dec{}, errorsmod.Wrap(ErrNaN, "cannot Log2 a non positive value")
	}

	// Increase precision for result accuracy.
	decCtx := dec128Context.WithPrecision(dec128Context.Precision + 2)

	var xCopy apd.Decimal
	xCopy.Set(&x.dec)

	// As we'll make divisions by 2 through bit right shift, we need to make sure we have enough decimal digit precision
	// in the coeff.
	// For example:
	// Taking a 2.12 dec represented by {coeff: 212, exp: -2}
	// It is here transformed to {coeff: 212000000000000000000000000000000000, exp: -35}
	enforceDecimalPrecision(&xCopy, decCtx.Precision)

	yBig := apd.NewBigInt(0)

	for xCopy.Cmp(oneDec) == -1 {
		xCopy.Coeff.Lsh(&xCopy.Coeff, 1)
		yBig.Sub(yBig, oneBigInt)
	}

	for xCopy.Cmp(twoDec) >= 0 {
		xCopy.Coeff.Rsh(&xCopy.Coeff, 1)
		yBig.Add(yBig, oneBigInt)
	}

	var b apd.Decimal
	b.Set(&log2B)
	y := apd.NewWithBigInt(yBig, 0)

	// The more iterations, the more accurate the result.
	// 200 iterations seems enough for 34 precision when not too close from log limits.
	for i := 0; i < 200; i++ {
		_, err := decCtx.Mul(&xCopy, &xCopy, &xCopy)
		if err != nil {
			return Dec{}, errorsmod.Wrap(ErrNaN, "decimal binary logarithm error")
		}
		if xCopy.Cmp(twoDec) >= 0 {
			xCopy.Coeff.Rsh(&xCopy.Coeff, 1)
			_, err = apd.BaseContext.Add(y, y, &b)
			if err != nil {
				return Dec{}, errorsmod.Wrap(ErrNaN, "decimal binary logarithm error")
			}
		}
		b.Coeff.Rsh(&b.Coeff, 1)
	}

	return Dec{dec: *y, isNaN: false}, nil
}
