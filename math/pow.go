package math

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/cockroachdb/apd/v3"
)

func Pow(x, y Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) || y.IsNaN() || shouldBeNaN(&y.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot pow with NaN values")
	}
	if x.dec.Form == apd.Infinite || y.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot pow with infinite values")
	}
	if x.dec.Cmp(zeroDec) <= 0 {
		return Dec{}, fmt.Errorf("cannot pow a 0 or negative base")
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
	if nd := uint32(x.dec.NumDigits()); p < nd {
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
	var zf apd.Decimal
	ed.Abs(&zf, &x.dec)
	lnAbsX, err := Ln(Dec{dec: zf, isNaN: false})
	if err != nil {
		return Dec{}, errorsmod.Wrap(err, "ln error computing pow")
	}
	ed.Mul(&zf, &lnAbsX.dec, &frac)
	zfd, err := Exp(Dec{dec: zf, isNaN: false})
	if err != nil {
		return Dec{}, errorsmod.Wrap(err, "ln error computing pow")
	}

	// Join integer and frac parts back.
	ed.Mul(&z, &z, &zfd.dec)
	if err := ed.Err(); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing pow")
	}
	if _, err := dec128Context.Round(&z, &z); err != nil {
		return Dec{}, errorsmod.Wrap(err, "round error computing pow")
	}

	return Dec{dec: z, isNaN: false}, nil
}
