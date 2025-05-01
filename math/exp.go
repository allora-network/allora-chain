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
	ln10n                 apd.Decimal
	nln10                 apd.Decimal
	ln10nhi               = mustNewApdDecFromString("0.007675283643313485613393")
	ln10nlo               = mustNewApdDecFromString("3.048489478806920036716287625765868e-25")

	// Boundaries of the input for the exponential function.
	maxExpInput apd.Decimal
	minExpInput apd.Decimal

	// Terms to compute exponential.
	expP2 apd.Decimal // 1/6
	expP4 apd.Decimal // -1/360
	expP6 apd.Decimal // 1/15120
	expP8 apd.Decimal // -1/604800
)

// fnCalcCtx is the context used to compute functions like exponential, logarithm, etc.
var fnCalcCtx = apd.Context{
	Precision:   dec128Context.Precision + 2,
	MaxExponent: apd.MaxExponent,
	MinExponent: apd.MinExponent,
	Traps:       apd.DefaultTraps,
	Rounding:    apd.RoundHalfEven,
}

// fnRoundCtx is the context used to round final results of functions like exponential, logarithm, etc.
var fnRoundCtx = fnCalcCtx.WithPrecision(dec128Context.Precision)

func init() {
	ed := apd.MakeErrDecimal(&fnCalcCtx)

	ed.Mul(&maxExpInput, apd.New(int64(dec128Context.MaxExponent), 0), ln10)
	ed.Mul(&minExpInput, apd.New(int64(dec128Context.MinExponent), 0), ln10)

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

// Exp returns a new Dec with the value of the exponential of x, without mutating x.
func Exp(x Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot exp a NaN")
	}
	if x.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot exp an infinite value")
	}

	var d apd.Decimal
	if err := exp(&d, &x.dec); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing exponential")
	}

	if _, err := fnRoundCtx.Round(&d, &d); err != nil {
		return Dec{}, errorsmod.Wrap(err, "error computing exponential")
	}

	return Dec{dec: d, isNaN: false}, nil
}

func exp(d, x *apd.Decimal) error {
	if x.Cmp(zeroDec) == 0 {
		d.Set(oneDec)
		return nil
	}
	if x.Cmp(&minExpInput) < 0 {
		d.Set(zeroDec)
		return nil
	}
	if x.Cmp(&maxExpInput) > 0 {
		d.Form = apd.Infinite
		return nil
	}

	ed := apd.MakeErrDecimal(&fnCalcCtx)

	var tmp, chi, clo, c apd.Decimal
	ed.Mul(&tmp, x, &nln10)
	ed.RoundToIntegralValue(&tmp, &tmp)
	tmp64 := ed.Int64(&tmp) // true because of the boundaries
	a := tmp64 / expPowersTableSize

	b := tmp64 % expPowersTableSize
	if b < 0 {
		b += expPowersTableSize
		a -= 1
	}

	ed.Mul(&chi, &tmp, ln10nhi)
	ed.Sub(&chi, x, &chi)
	ed.Mul(&clo, &tmp, ln10nlo)
	ed.Sub(&c, &chi, &clo)

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
	pb := expPowers[b]
	ed.Mul(&pbc, &pb, &c)
	ed.Sub(&tmp, twoDec, &r)
	ed.Mul(&expbc, &pbc, &r)
	ed.Quo(&expbc, &expbc, &tmp)
	ed.Add(&expbc, &expbc, &pbc)
	ed.Add(&expbc, &expbc, &pb)

	if err := ed.Err(); err != nil {
		return err
	}

	exponent := int64(expbc.Exponent) + a
	if exponent > math.MaxInt32 || exponent < math.MinInt32 {
		return ErrOverflow
	}
	expbc.Exponent = int32(exponent) //nolint:gosec // ensured just above

	d.Set(&expbc)
	return nil
}
