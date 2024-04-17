// This code is forked from Regen Ledger.
// Their code is under the Apache2.0 License
// https://github.com/regen-network/regen-ledger/blob/3d818cf6e01af92eed25de5c17728a79070f56a3/types/math/dec.go

package math

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cockroachdb/apd/v3"
)

// Dec is a wrapper struct around apd.Decimal that does no mutation of apd.Decimal's when performing
// arithmetic, instead creating a new apd.Decimal for every operation ensuring usage is safe.
//
// Using apd.Decimal directly can be unsafe because apd operations mutate the underlying Decimal,
// but when copying the big.Int structure can be shared between Decimal instances causing corruption.
// This was originally discovered in regen0-network/mainnet#15.
type Dec struct {
	dec   apd.Decimal
	isNaN bool
}

// constants for more convenient intent behind dec.Cmp values.
const (
	GreaterThan = 1
	LessThan    = -1
	EqualTo     = 0
)

const mathCodespace = "math"

var (
	ErrInvalidDecString   = errors.Register(mathCodespace, 1, "invalid decimal string")
	ErrUnexpectedRounding = errors.Register(mathCodespace, 2, "unexpected rounding")
	ErrNonIntegeral       = errors.Register(mathCodespace, 3, "value is non-integral")
	ErrInfiniteString     = errors.Register(mathCodespace, 4, "value is infinite")
)

// The number 0 encoded as Dec
func ZeroDec() Dec {
	return NewDecFromInt64(0)
}

// The number 1 encoded as Dec
func OneDec() Dec {
	return NewDecFromInt64(1)
}

// In cosmos-sdk#7773, decimal128 (with 34 digits of precision) was suggested for performing
// Quo/Mult arithmetic generically across the SDK. Even though the SDK
// has yet to support a GDA with decimal128 (34 digits), we choose to utilize it here.
// https://github.com/cosmos/cosmos-sdk/issues/7773#issuecomment-725006142
var dec128Context = apd.Context{
	Precision:   34,
	MaxExponent: apd.MaxExponent,
	MinExponent: apd.MinExponent,
	Traps:       apd.DefaultTraps,
}

func NewNaN() Dec {
	return Dec{apd.Decimal{}, true}
}

func NewDecFromString(s string) (Dec, error) {
	if s == "" {
		s = "0"
	}
	d, _, err := apd.NewFromString(s)
	if err != nil {
		return Dec{}, ErrInvalidDecString.Wrap(err.Error())
	}

	d1 := Dec{*d, false}
	if d1.dec.Form == apd.Infinite {
		return d1, ErrInfiniteString.Wrapf(s)
	}

	return d1, nil
}

func MustNewDecFromString(s string) Dec {
	ret, err := NewDecFromString(s)
	if err != nil {
		panic(err)
	}
	return ret
}

func NewNonNegativeDecFromString(s string) (Dec, error) {
	d, err := NewDecFromString(s)
	if err != nil {
		return Dec{}, ErrInvalidDecString.Wrap(err.Error())
	}
	if d.IsNegative() {
		return Dec{}, ErrInvalidDecString.Wrapf("expected a non-negative decimal, got %s", s)
	}
	return d, nil
}

func NewNonNegativeFixedDecFromString(s string, max uint32) (Dec, error) {
	d, err := NewNonNegativeDecFromString(s)
	if err != nil {
		return Dec{}, err
	}
	if d.NumDecimalPlaces() > max {
		return Dec{}, fmt.Errorf("%s exceeds maximum decimal places: %d", s, max)
	}
	return d, nil
}

func NewPositiveDecFromString(s string) (Dec, error) {
	d, err := NewDecFromString(s)
	if err != nil {
		return Dec{}, ErrInvalidDecString.Wrap(err.Error())
	}
	if !d.IsPositive() || !d.IsFinite() {
		return Dec{}, ErrInvalidDecString.Wrapf("expected a positive decimal, got %s", s)
	}
	return d, nil
}

func NewPositiveFixedDecFromString(s string, max uint32) (Dec, error) {
	d, err := NewPositiveDecFromString(s)
	if err != nil {
		return Dec{}, err
	}
	if d.NumDecimalPlaces() > max {
		return Dec{}, fmt.Errorf("%s exceeds maximum decimal places: %d", s, max)
	}
	return d, nil
}

func NewDecFromInt64(x int64) Dec {
	var res Dec
	res.dec.SetInt64(x)
	return res
}

// NewDecFinite returns a decimal with a value of coeff * 10^exp.
func NewDecFinite(coeff int64, exp int32) Dec {
	var res Dec
	res.dec.SetFinite(coeff, exp)
	return res
}

// NewDec takes a cosmos `sdkmath.Int` and turns it into a Dec
// it converts via strings and throws an error if the string
// is unable to be parsed
func NewDecFromSdkInt(x sdkmath.Int) (Dec, error) {
	strRep := x.String()
	return NewDecFromString(strRep)
}

// NewDec takes a cosmos `sdkmath.Uint` and turns it into a Dec
// it converts via strings and throws an error if the string
// is unable to be parsed
func NewDecFromSdkUint(x sdkmath.Uint) (Dec, error) {
	strRep := x.String()
	return NewDecFromString(strRep)
}

// NewDec takes a cosmos `sdkmath.LegacyDec` and turns it into a Dec
// it converts via strings and throws an error if the string
// is unable to be parsed
func NewDecFromSdkLegacyDec(x sdkmath.LegacyDec) (Dec, error) {
	strRep := x.String()
	return NewDecFromString(strRep)
}

// Add returns a new Dec with value `x+y` without mutating any argument and error if
// there is an overflow.
func (x Dec) Add(y Dec) (Dec, error) {
	var z Dec
	_, err := apd.BaseContext.Add(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal addition error")
}

// Sub returns a new Dec with value `x-y` without mutating any argument and error if
// there is an overflow.
func (x Dec) Sub(y Dec) (Dec, error) {
	var z Dec
	_, err := apd.BaseContext.Sub(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal subtraction error")
}

// Quo returns a new Dec with value `x/y` (formatted as decimal128, 34 digit precision) without mutating any
// argument and error if there is an overflow.
func (x Dec) Quo(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Quo(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal quotient error")
}

// MulExact returns a new dec with value x * y. The product must not round or
// ErrUnexpectedRounding will be returned.
func (x Dec) MulExact(y Dec) (Dec, error) {
	var z Dec
	condition, err := dec128Context.Mul(&z.dec, &x.dec, &y.dec)
	if err != nil {
		return z, err
	}
	if condition.Rounded() {
		return z, ErrUnexpectedRounding
	}
	return z, nil
}

// QuoExact is a version of Quo that returns ErrUnexpectedRounding if any rounding occurred.
func (x Dec) QuoExact(y Dec) (Dec, error) {
	var z Dec
	condition, err := dec128Context.Quo(&z.dec, &x.dec, &y.dec)
	if err != nil {
		return z, err
	}
	if condition.Rounded() {
		return z, ErrUnexpectedRounding
	}
	return z, errors.Wrap(err, "decimal quotient error")
}

// QuoInteger returns a new integral Dec with value `x/y` (formatted as decimal128, with 34 digit precision)
// without mutating any argument and error if there is an overflow.
func (x Dec) QuoInteger(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.QuoInteger(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal quotient error")
}

// Rem returns the integral remainder from `x/y` (formatted as decimal128, with 34 digit precision) without
// mutating any argument and error if the integer part of x/y cannot fit in 34 digit precision
func (x Dec) Rem(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Rem(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal remainder error")
}

// Mul returns a new Dec with value `x*y` (formatted as decimal128, with 34 digit precision) without
// mutating any argument and error if there is an overflow.
func (x Dec) Mul(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Mul(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal multiplication error")
}

// Neg negates the decimal and returns a new Dec with value `-x` without
// mutating any argument and error if there is an overflow.
func (x Dec) Neg() (Dec, error) {
	var z Dec
	_, err := dec128Context.Neg(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal negation error")
}

// Log10 returns a new Dec with the value of the base 10 logarithm of x, without mutating x.
func Log10(x Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Log10(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal base 10 logarithm error")
}

// Ln returns a new Dec with the value of the natural logarithm of x, without mutating x.
func Ln(x Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Ln(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal natural logarithm error")
}

// Exp returns a new Dec with the value of e^x, without mutating x.
func Exp(x Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Exp(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal e to the x exponentiation error")
}

// Exp10 returns a new Dec with the value of 10^x, without mutating x.
func Exp10(x Dec) (Dec, error) {
	var ten Dec = NewDecFromInt64(10)
	var z Dec
	_, err := dec128Context.Pow(&z.dec, &ten.dec, &x.dec)
	return z, errors.Wrap(err, "decimal 10 to the x exponentiation error")
}

// Pow returns a new Dec with the value of x**y, without mutating x or y.
func Pow(x Dec, y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Pow(&z.dec, &x.dec, &y.dec)
	return z, errors.Wrap(err, "decimal exponentiation error")
}

// returns the max of x and y without mutating x or y.
func Max(x Dec, y Dec) Dec {
	var z Dec
	if x.Cmp(y) == GreaterThan {
		z.dec.Set(&x.dec)
	} else {
		z.dec.Set(&y.dec)
	}
	return z
}

// returns the min of x and y without mutating x or y.
func Min(x Dec, y Dec) Dec {
	var z Dec
	if x.Cmp(y) == LessThan {
		z.dec.Set(&x.dec)
	} else {
		z.dec.Set(&y.dec)
	}
	return z
}

// Sqrt returns a new Dec with the value of the square root of x, without mutating x.
func (x Dec) Sqrt() (Dec, error) {
	var z Dec
	_, err := dec128Context.Sqrt(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal square root error")
}

// Abs returns a new Dec with the absolute value of x, without mutating x.
func (x Dec) Abs() Dec {
	var z Dec
	z.dec.Abs(&x.dec)
	return z
}

// Ceil returns a new Dec with the value of x rounded up to the nearest integer, without mutating x.
func (x Dec) Ceil() (Dec, error) {
	var z Dec
	_, err := dec128Context.Ceil(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal ceiling error")
}

// Floor returns a new Dec with the value of x rounded down to the nearest integer, without mutating x.
func (x Dec) Floor() (Dec, error) {
	var z Dec
	_, err := dec128Context.Floor(&z.dec, &x.dec)
	return z, errors.Wrap(err, "decimal floor error")
}

// Int64 converts x to an int64 or returns an error if x cannot
// fit precisely into an int64.
func (x Dec) Int64() (int64, error) {
	return x.dec.Int64()
}

// BigInt converts x to a *big.Int or returns an error if x cannot
// fit precisely into an *big.Int.
func (x Dec) BigInt() (*big.Int, error) {
	y, _ := x.Reduce()
	z := &big.Int{}
	z, ok := z.SetString(y.String(), 10)
	if !ok {
		return nil, ErrNonIntegeral
	}
	return z, nil
}

// SdkIntTrim rounds decimal number to the integer towards zero and converts it to `sdkmath.Int`.
// Panics if x is bigger the SDK Int max value
func (x Dec) SdkIntTrim() sdkmath.Int {
	y, _ := x.Reduce()
	var r = y.dec.Coeff
	if y.dec.Exponent != 0 {
		decs := apd.NewBigInt(10)
		if y.dec.Exponent > 0 {
			decs.Exp(decs, apd.NewBigInt(int64(y.dec.Exponent)), nil)
			r.Mul(&y.dec.Coeff, decs)
		} else {
			decs.Exp(decs, apd.NewBigInt(int64(-y.dec.Exponent)), nil)
			r.Quo(&y.dec.Coeff, decs)
		}
	}
	if x.dec.Negative {
		r.Neg(&r)
	}
	return sdkmath.NewIntFromBigInt(r.MathBigInt())
}

func (x Dec) SdkLegacyDec() sdkmath.LegacyDec {
	y, _ := sdkmath.LegacyNewDecFromStr(x.dec.Text('f'))
	return y
}

func (x Dec) String() string {
	return x.dec.Text('f')
}

// Marshal implements the gogo proto custom type interface.
func (d Dec) Marshal() ([]byte, error) {
	return d.dec.MarshalText()
}

// Unmarshal implements the gogo proto custom type interface.
func (d *Dec) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if err := d.dec.UnmarshalText(data); err != nil {
		return err
	}

	return nil
}

// Size returns the size of the marshalled Dec type in bytes
func (d Dec) Size() int {
	bz, _ := d.Marshal()
	return len(bz)
}

// MarshalTo implements the gogo proto custom type interface.
func (d *Dec) MarshalTo(data []byte) (n int, err error) {
	bz, err := d.Marshal()
	if err != nil {
		return 0, err
	}

	copy(data, bz)
	return len(bz), nil
}

// MarshalJSON marshals the decimal
func (d Dec) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON defines custom decoding scheme
func (d *Dec) UnmarshalJSON(bz []byte) error {

	var text string
	err := json.Unmarshal(bz, &text)
	if err != nil {
		return err
	}

	newDec, err := NewDecFromString(text)
	if err != nil {
		return err
	}

	*d = newDec

	return nil
}

// Cmp compares x and y and returns:
// -1 if x <  y
// 0 if x == y
// +1 if x >  y
// undefined if d or x are NaN
func (x Dec) Cmp(y Dec) int {
	return x.dec.Cmp(&y.dec)
}

func (x Dec) Gt(y Dec) bool {
	return x.dec.Cmp(&y.dec) == 1
}

func (x Dec) Gte(y Dec) bool {
	return x.dec.Cmp(&y.dec) == 1 || x.dec.Cmp(&y.dec) == 0
}

func (x Dec) Lt(y Dec) bool {
	return x.dec.Cmp(&y.dec) == -1
}

func (x Dec) Lte(y Dec) bool {
	return x.dec.Cmp(&y.dec) == -1 || x.dec.Cmp(&y.dec) == 0
}

func (x Dec) Equal(y Dec) bool {
	return x.dec.Cmp(&y.dec) == 0
}

func (x Dec) IsNaN() bool {
	return x.isNaN
}

// IsZero returns true if the decimal is zero.
func (x Dec) IsZero() bool {
	return x.dec.IsZero()
}

// IsNegative returns true if the decimal is negative.
func (x Dec) IsNegative() bool {
	return x.dec.Negative && !x.dec.IsZero()
}

// IsPositive returns true if the decimal is positive.
func (x Dec) IsPositive() bool {
	return !x.dec.Negative && !x.dec.IsZero()
}

// IsFinite returns true if the decimal is finite.
func (x Dec) IsFinite() bool {
	return x.dec.Form == apd.Finite
}

// NumDecimalPlaces returns the number of decimal places in x.
func (x Dec) NumDecimalPlaces() uint32 {
	exp := x.dec.Exponent
	if exp >= 0 {
		return 0
	}
	return uint32(-exp)
}

// Reduce returns a copy of x with all trailing zeros removed and the number
// of trailing zeros removed.
func (x Dec) Reduce() (Dec, int) {
	y := Dec{}
	_, n := y.dec.Reduce(&x.dec)
	return y, n
}

// helper function for test suites that want to check
// if some math is within a delta
func InDelta(expected, result Dec, epsilon Dec) bool {
	delta, err := expected.Sub(result)
	if err != nil {
		return false
	}
	deltaAbs := delta.Abs()
	compare := deltaAbs.Cmp(epsilon)
	if compare == LessThan || compare == EqualTo {
		return true
	}
	return false
}

// Helper function to compare two slices of alloraMath.Dec within a delta
func SlicesInDelta(a, b []Dec, epsilon Dec) bool {
	lenA := len(a)
	if lenA != len(b) {
		return false
	}
	for i := 0; i < lenA; i++ {
		// for performance reasons we do not call InDelta
		// pass by copy causes this to run slow af for large slices
		delta, err := a[i].Sub(b[i])
		if err != nil {
			return false
		}
		if delta.Abs().Cmp(epsilon) == GreaterThan {
			return false
		}
	}
	return true
}

// Generic Sum function, given an array of values returns its sum
func SumDecSlice(x []Dec) (Dec, error) {
	sum := ZeroDec()
	var err error = nil
	for _, v := range x {
		sum, err = sum.Add(v)
		if err != nil {
			return Dec{}, err
		}
	}
	return sum, nil
}
