package math

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/cockroachdb/apd/v3"
)

const powerTenTableSize = 128

var (
	twentyThreeDec          = apd.New(23, 0)
	thousandDec             = apd.New(1000, 0)
	expConvergenceSlopeDec  = apd.New(1435, -3)
	expConvergenceOffsetDec = apd.New(1182, -3)
	pow10LookupTable        [powerTenTableSize + 1]apd.BigInt
)

func init() {
	for i := int64(0); i <= powerTenTableSize; i++ {
		setBigWithPow(&pow10LookupTable[i], i)
	}
}

// Exp computes the exponential of x, returning a new Dec.
// It is based on apd's implementation but in a deterministic way.
func Exp(x Dec) (Dec, error) {
	if x.IsNaN() || shouldBeNaN(&x.dec) {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot exp a NaN")
	}
	if x.dec.Form == apd.Infinite {
		return Dec{}, fmt.Errorf("cannot exp an infinite value")
	}

	if x.IsZero() {
		return OneDec(), nil
	}

	// Stage 1
	cp := dec128Context.Precision
	var absX apd.Decimal
	absX.Abs(&x.dec)
	if _, err := absX.Float64(); err == nil {
		// This algorithm doesn't work if currentprecision*23 < |x|. Attempt to
		// increase the working precision if needed as long as it isn't too large. If
		// it is too large, don't bump the precision, causing an early overflow return.
		var ncp apd.Decimal
		if _, err := dec128Context.Quo(&ncp, &absX, twentyThreeDec); err != nil {
			return Dec{}, errorsmod.Wrapf(err, "quo error computing exp")
		}
		if ncp.Cmp(apd.New(int64(cp), 0)) > 0 && ncp.Cmp(thousandDec) < 0 {
			if _, err := dec128Context.Ceil(&ncp, &ncp); err != nil {
				return Dec{}, errorsmod.Wrapf(err, "ceil error computing exp")
			}
			ncpi64, err := ncp.Int64()
			if err != nil {
				return Dec{}, errorsmod.Wrapf(err, "cast to int64 error computing exp")
			}
			cp = uint32(ncpi64) // we are sure it's > 0 and < 1000
		}
	}
	var cpTimes23 apd.Decimal
	cpTimes23.SetInt64(int64(cp) * 23)
	// if abs(x) > 23*currentprecision; overflow
	if absX.Cmp(&cpTimes23) > 0 {
		return Dec{}, errorsmod.Wrapf(ErrOverflow, "exp overflow: %s", x.String())
	}
	// if abs(x) <= setexp(.9, -currentprecision); then result 1 as x -> 0
	var minValPrec apd.Decimal
	minValPrec.SetFinite(9, int32(-cp)-1)
	if absX.Cmp(&minValPrec) <= 0 {
		return OneDec(), nil
	}

	// Stage 2
	// Add x.NumDigits because the paper assumes that x.Coeff [0.1, 1).
	t := x.dec.Exponent + int32(x.dec.NumDigits())
	if t < 0 {
		t = 0
	}
	var k, r apd.Decimal
	k.SetFinite(1, t)
	nc := dec128Context.WithPrecision(cp)
	nc.Rounding = apd.RoundHalfEven
	if _, err := nc.Quo(&r, &x.dec, &k); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "quo error computing exp")
	}
	var ra apd.Decimal
	ra.Abs(&r)
	p := int64(cp) + int64(t) + 2
	var pDec apd.Decimal
	pDec.SetFinite(p, 0)

	// Stage 3
	if _, err := ra.Float64(); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "cast to float error computing exp")
	}
	var pfDivRf apd.Decimal
	if _, err := nc.Quo(&pfDivRf, &pDec, &ra); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "quo error computing exp")
	}
	logRatio, err := Log10(Dec{dec: pfDivRf, isNaN: false})
	if err != nil {
		return Dec{}, errorsmod.Wrapf(err, "log10 error computing exp")
	}
	var numerator apd.Decimal
	if _, err := nc.Mul(&numerator, expConvergenceSlopeDec, &pDec); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "mul error computing exp")
	}
	if _, err := nc.Quo(&numerator, &numerator, expConvergenceOffsetDec); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "quo error computing exp")
	}
	var termCountEstimate apd.Decimal
	if _, err := nc.Quo(&termCountEstimate, &numerator, &logRatio.dec); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "quo error computing exp")
	}

	if _, err := dec128Context.Ceil(&termCountEstimate, &termCountEstimate); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "ceil error computing exp")
	}
	if termCountEstimate.Cmp(thousandDec) > 0 {
		return Dec{}, fmt.Errorf("cannot exp, too many iterations: %s", x.String())
	}
	n, err := termCountEstimate.Int64()
	if err != nil {
		return Dec{}, errorsmod.Wrapf(err, "cast to int64 error computing exp")
	}

	// Stage 4
	nc.Precision = uint32(p)
	ed := apd.MakeErrDecimal(nc)
	var sum, it, rDivI apd.Decimal
	sum.SetInt64(1)
	it.SetFinite(0, 0)
	for i := n - 1; i > 0; i-- {
		it.Coeff.SetInt64(i)
		// tmp1 = r / i
		ed.Quo(&rDivI, &r, &it)
		// sum = sum * r / i
		ed.Mul(&sum, &rDivI, &sum)
		// sum = sum + 1
		ed.Add(&sum, &sum, oneDec)
	}
	if err := ed.Err(); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "stage 4 error computing exp")
	}

	// sum ** k
	var tmpE apd.BigInt
	ki, err := exp10(int64(t), &tmpE)
	if err != nil {
		return Dec{}, errorsmod.Wrapf(err, "exp10 error computing exp")
	}
	var z apd.Decimal
	if err := integerPower(nc, &z, &sum, ki); err != nil {
		return Dec{}, errorsmod.Wrapf(err, "integral power error computing exp")
	}

	nc.Precision = dec128Context.Precision
	_, err = nc.Round(&z, &z)
	if err != nil {
		return Dec{}, errorsmod.Wrapf(err, "round error computing exp")
	}

	return Dec{dec: z, isNaN: false}, nil
}

func exp10(x int64, tmp *apd.BigInt) (exp *apd.BigInt, err error) {
	if x > apd.MaxExponent || x < apd.MinExponent {
		return nil, errorsmod.Wrapf(ErrOutOfRange, "exp10 exponent out of range: %d", x)
	}
	return tableExp10(x, tmp), nil
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

func tableExp10(x int64, tmp *apd.BigInt) *apd.BigInt {
	if x <= powerTenTableSize {
		return &pow10LookupTable[x]
	}
	setBigWithPow(tmp, x)
	return tmp
}

func setBigWithPow(res *apd.BigInt, pow int64) {
	var tmp apd.BigInt
	tmp.SetInt64(pow)
	res.Exp(tenBigInt, &tmp, nil)
}
