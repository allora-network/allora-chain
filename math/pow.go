package math

import (
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/cockroachdb/apd/v3"
)

// Pow returns a new Dec with the value x power y, without mutating x and y.
func Pow(x, y Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) || y.IsNaN() || shouldBeNaN(&y.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot pow with NaN values")
	}
	if x.dec.Form == apd.Infinite || y.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot pow with infinite values")
	}
	switch x.dec.Cmp(zeroDec) {
	case -1:
		return Dec{}, fmt.Errorf("cannot pow a negative base")
	case 0:
		return ZeroDec(), nil
	}

	if y.IsZero() {
		return OneDec(), nil
	}
	if y.dec.Cmp(oneDec) == 0 {
		return x, nil
	}

	var integ, frac apd.Decimal
	y.dec.Modf(&integ, &frac)

	p := dec128Context.Precision
	xNumDigits := x.dec.NumDigits()
	if xNumDigits > math.MaxUint32 {
		return Dec{}, errorsmod.Wrapf(ErrOverflow, "x has too many digits")
	}
	if nd := uint32(x.dec.NumDigits()); p < nd { //nolint:gosec // ensured just above (and cannot be negative)
		p = nd
	}
	p += 4 + 6
	nc := apd.BaseContext.WithPrecision(p)

	var z apd.Decimal

	// If integ.Exponent > 0, we need to add trailing 0s to integ.Coeff.
	if _, err := dec128Context.Quantize(&integ, &integ, 0); err != nil {
		return Dec{}, errorsmod.Wrap(err, "quantize error computing pow")
	}
	if err := integerPower(nc, &z, &x.dec, &integ.Coeff); err != nil {
		return Dec{}, errorsmod.Wrap(err, "integer power error computing pow")
	}

	// no fractional part, we can return the integer part
	if frac.IsZero() {
		if _, err := dec128Context.Round(&z, &z); err != nil {
			return Dec{}, errorsmod.Wrap(err, "round error computing pow")
		}
		return Dec{dec: z, isNaN: false}, nil
	}

	ed := apd.MakeErrDecimal(nc)

	// Compute x**frac(y)
	var zf, lnAbsX, zfd apd.Decimal
	ed.Abs(&zf, &x.dec)
	if err := ln(&lnAbsX, &zf); err != nil {
		return Dec{}, errorsmod.Wrap(err, "ln error computing pow")
	}
	ed.Mul(&zf, &lnAbsX, &frac)
	if err := exp(&zfd, &zf); err != nil {
		return Dec{}, errorsmod.Wrap(err, "ln error computing pow")
	}

	// Join integer and frac parts back.
	ed.Mul(&z, &z, &zfd)
	if err := ed.Err(); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing pow")
	}
	if _, err := dec128Context.Round(&z, &z); err != nil {
		return Dec{}, errorsmod.Wrap(err, "round error computing pow")
	}

	return Dec{dec: z, isNaN: false}, nil
}

// integerPower sets d = x**y. d and x must not point to the same Decimal.
func integerPower(c *apd.Context, d, x *apd.Decimal, y *apd.BigInt) error {
	// See: https://en.wikipedia.org/wiki/Exponentiation_by_squaring.

	var b apd.BigInt
	b.Set(y)
	neg := b.Sign() < 0
	if neg {
		b.Abs(&b)
	}

	var n apd.Decimal
	n.Set(x)
	z := d
	z.Set(oneDec)
	ed := apd.MakeErrDecimal(c)
	for b.Sign() > 0 {
		if b.Bit(0) == 1 {
			ed.Mul(z, z, &n)
		}
		b.Rsh(&b, 1)

		// Only compute the next n if we are going to use it. Otherwise n can overflow
		// on the last iteration causing this to error.
		if b.Sign() > 0 {
			ed.Mul(&n, &n, &n)
		}
		if err := ed.Err(); err != nil {
			return err
		}
	}

	if neg {
		ed.Quo(z, oneDec, z)
	}
	return ed.Err()
}
