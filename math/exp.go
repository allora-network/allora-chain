package math

import (
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/cockroachdb/apd/v3"
)

const expPowersTableSize = 300

var (
	// expPowers contains 10**(i/n) precomputed values, this is used to compute exponential.
	expPowers             = [expPowersTableSize]apd.Decimal{}
	expPowersTableSizeDec = apd.New(expPowersTableSize, 0)
	ln10                  = mustNewApdDecFromString("2.3025850929940456840179914546843642")
	ln10n                 apd.Decimal
	nln10                 apd.Decimal

	// Terms to compute exponential.
	expP2 apd.Decimal // 1/6
	expP4 apd.Decimal // -1/360
	expP6 apd.Decimal // 1/15120
	expP8 apd.Decimal // -1/604800
)

func init() {
	ctx := apd.BaseContext.WithPrecision(dec128Context.Precision + 2)
	ctx.Rounding = apd.RoundHalfEven
	ed := apd.MakeErrDecimal(ctx)

	ed.Quo(&ln10n, ln10, expPowersTableSizeDec)
	ed.Quo(&nln10, expPowersTableSizeDec, ln10)

	var pq apd.Decimal
	pq.SetInt64(6)
	ed.Quo(&expP2, oneDec, &pq)
	pq.SetInt64(-360)
	ed.Quo(&expP4, oneDec, &pq)
	pq.SetInt64(15120)
	ed.Quo(&expP6, oneDec, &pq)
	pq.SetInt64(-604800)
	ed.Quo(&expP8, oneDec, &pq)

	for i := int64(0); i < expPowersTableSize; i++ {
		var pow, iDec apd.Decimal
		iDec.SetInt64(i)
		ed.Quo(&pow, &iDec, expPowersTableSizeDec)
		ed.Pow(&pow, tenDec, &pow)
		expPowers[i] = pow
	}

	if err := ed.Err(); err != nil {
		panic(err)
	}
}

func Exp(x Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot exp a NaN")
	}
	if x.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot exp an infinite value")
	}

	var d apd.Decimal
	if err := exp(&d, &x.dec); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "error computing exp")
	}

	return Dec{dec: d, isNaN: false}, nil
}

func exp(d, x *apd.Decimal) error {
	if x.Cmp(zeroDec) == 0 {
		d.SetInt64(1)
		return nil
	}

	ctx := apd.BaseContext.WithPrecision(dec128Context.Precision + 2)
	ctx.Rounding = apd.RoundHalfEven
	ed := apd.MakeErrDecimal(ctx)

	var tmp, a, b, c apd.Decimal
	ed.Mul(&tmp, x, &nln10)
	ed.RoundToIntegralValue(&tmp, &tmp)
	ed.QuoInteger(&a, &tmp, expPowersTableSizeDec)
	a64, err := a.Int64()
	if err != nil {
		return errorsmod.Wrap(err, "error computing exp")
	}

	ed.Rem(&b, &tmp, expPowersTableSizeDec)
	ed.Mul(&c, &tmp, &ln10n)
	ed.Sub(&c, x, &c)

	var c2, r, pbc, expbc apd.Decimal
	ed.Mul(&c2, &c, &c)
	ed.Mul(&r, &c2, &expP8)
	ed.Add(&r, &expP6, &r)
	ed.Mul(&r, &c2, &r)
	ed.Add(&r, &expP4, &r)
	ed.Mul(&r, &c2, &r)
	ed.Add(&r, &expP2, &r)
	ed.Mul(&r, &c2, &r)
	ed.Sub(&r, &c, &r)
	b64, err := b.Int64()
	if err != nil {
		return errorsmod.Wrap(err, "error computing exp")
	}
	pb := expPowers[b64]
	ed.Mul(&pbc, &pb, &c)
	ed.Sub(&tmp, twoDec, &r)
	ed.Mul(&expbc, &pbc, &r)
	ed.Quo(&expbc, &expbc, &tmp)
	ed.Add(&expbc, &expbc, &pbc)
	ed.Add(&expbc, &expbc, &pb)

	if err := ed.Err(); err != nil {
		return errorsmod.Wrap(err, "error computing exp")
	}

	exp := int64(expbc.Exponent) + a64
	if exp > math.MaxInt32 || exp < math.MinInt32 {
		return errorsmod.Wrap(err, "error computing exp")
	}
	expbc.Exponent = int32(exp)

	if _, err := ctx.WithPrecision(dec128Context.Precision).Round(&expbc, &expbc); err != nil {
		return errorsmod.Wrap(err, "error computing exp")
	}

	d.Set(&expbc)
	return nil
}
