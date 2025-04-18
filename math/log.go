package math

import (
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/cockroachdb/apd/v3"
)

var (
	ln10 = mustNewApdDecFromString("2.302585092994045684017991454684364207")
	// 1/ln(10) with 34 precision
	invLn10 apd.Decimal

	threeDec = apd.New(3, 0)
	nineDec  = apd.New(9, 0)
)

func init() {
	iLn10, _, err := apd.NewFromString("0.434294481903251827651128918916605082")
	if err != nil {
		panic(err)
	}
	invLn10 = *iLn10
}

// Log10 returns a new Dec with the value of the base 10 logarithm of x, without mutating x.
func Log10(x Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot ln base 10 a NaN")
	}
	if x.dec.Sign() < 0 || x.dec.Cmp(zeroDec) == 0 {
		return Dec{}, fmt.Errorf("cannot ln base 10 a non positive value")
	}
	if x.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot ln base 10 an infinite value")
	}

	var d apd.Decimal
	if err := ln(&d, &x.dec); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing base 10 logarithm")
	}

	if _, err := fnCalcCtx.Mul(&d, &d, &invLn10); err != nil {
		return Dec{}, errorsmod.Wrap(err, "decimal base 10 logarithm error")
	}

	if _, err := fnRoundCtx.Round(&d, &d); err != nil {
		return Dec{}, errorsmod.Wrap(err, "decimal base 10 logarithm error")
	}

	return Dec{dec: d, isNaN: false}, nil
}

// Ln returns a new Dec with the value of the natural logarithm of x, without mutating x.
func Ln(x Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot ln a NaN")
	}
	if x.dec.Sign() < 0 || x.dec.Cmp(zeroDec) == 0 {
		return Dec{}, fmt.Errorf("cannot ln a non positive value")
	}
	if x.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot ln an infinite value")
	}

	var d apd.Decimal
	if err := ln(&d, &x.dec); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing natural logarithm")
	}

	if _, err := fnRoundCtx.Round(&d, &d); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing natural logarithm")
	}

	return Dec{dec: d, isNaN: false}, nil
}

func ln(d, x *apd.Decimal) error {
	if x.Cmp(oneDec) == 0 {
		d.Set(zeroDec)
		return nil
	}

	ed := apd.MakeErrDecimal(&fnCalcCtx)

	// set a and b as x = a * 10^b
	var a apd.Decimal
	a.Set(x)
	b := x.NumDigits() + int64(x.Exponent) - 1
	if b > math.MaxInt32 || b < math.MinInt32 {
		return ErrOverflow
	}
	a.Exponent -= int32(b)

	// d = ln(a) + b * ln(10)
	var lna, bDec, bln10 apd.Decimal

	bDec.SetInt64(b)
	ed.Mul(&bln10, &bDec, ln10)

	// lna = 3 - 9 / (a + 2) # rough initial estimate
	ed.Add(&lna, &a, twoDec)
	ed.Quo(&lna, nineDec, &lna)
	ed.Sub(&lna, threeDec, &lna)

	var e, tmp1, tmp2 apd.Decimal
	for range 3 {
		//e = math.exp(lna)
		if err := exp(&e, &lna); err != nil {
			return err
		}

		// lna = lna - 2 * (e - a) / (e + a)
		ed.Sub(&tmp1, &e, &a)
		ed.Add(&tmp2, &e, &a)
		ed.Quo(&tmp1, &tmp1, &tmp2)
		ed.Mul(&tmp1, twoDec, &tmp1)
		ed.Sub(&lna, &lna, &tmp1)
	}

	ed.Add(d, &lna, &bln10)

	if err := ed.Err(); err != nil {
		return err
	}

	return nil
}
