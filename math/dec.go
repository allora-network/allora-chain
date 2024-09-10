// This code is forked from Regen Ledger.
// Their code is under the Apache2.0 License
// https://github.com/regen-network/regen-ledger/blob/3d818cf6e01af92eed25de5c17728a79070f56a3/types/math/dec.go

package math

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/bits"

	errorsmod "cosmossdk.io/errors"
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
const NaNStr = "NaN"

var (
	ErrInvalidDecString   = errorsmod.Register(mathCodespace, 1, "invalid decimal string")
	ErrUnexpectedRounding = errorsmod.Register(mathCodespace, 2, "unexpected rounding")
	ErrNonIntegeral       = errorsmod.Register(mathCodespace, 3, "value is non-integral")
	ErrInfiniteString     = errorsmod.Register(mathCodespace, 4, "value is infinite")
	ErrOverflow           = errorsmod.Register(mathCodespace, 5, "overflow")
	ErrNaN                = errorsmod.Register(mathCodespace, 6, "Not a Number (NaN) is not permitted in this context")
	ErrNotMatchingLength  = errorsmod.Register(mathCodespace, 7, "slices are not of the same length")
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

// create a new Dec that represents NaN
func NewNaN() Dec {
	return Dec{apd.Decimal{}, true}
}

// NewDecFromString returns a new Dec from a given string. It returns an error if the string
// cannot be parsed. The string should be in the format of `123.456`.
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
		return d1, ErrInfiniteString.Wrap(s)
	}

	return d1, nil
}

// MustNewDecFromString returns a new Dec from a given string. It panics if the string
// cannot be parsed. The string should be in the format of `123.456`.
func MustNewDecFromString(s string) Dec {
	ret, err := NewDecFromString(s)
	if err != nil {
		panic(err)
	}
	return ret
}

// NewNonNegativeDecFromString returns a new Dec from a given string.
// It returns an error if the string cannot be parsed or if the
// decimal is negative. The string should be in the format of `123.456`.
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

// NewNonNegativeFixedDecFromString returns a new Dec from a given string and
// an upper limit on the number of decimal places. It returns an error if the
// string cannot be parsed, if the decimal is negative, or if the number of
// decimal places exceeds the maximum. The string should be in the format of
// `123.456`.
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

// NewPositiveDecFromString returns a new Dec from a string,
// returning an error if the string cannot be parsed or if the
// decimal is not positive. The string should be in the format of `123.456`.
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

// NewPositiveFixedDecFromString takes a string
// and an upper limit on the number of decimal places, returning
// a Dec or error if the string cannot be parsed, if the decimal
// is not positive, or if the number of decimal places exceeds the
// maximum. The string should be in the format of `123.456`.
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

// create a new dec from an int64 value
func NewDecFromInt64(x int64) Dec {
	var res Dec
	res.dec.SetInt64(x)
	return res
}

// create a new dec from a uint64 value
// it converts via strings and throws an error if the string
// is unable to be parsed
func NewDecFromUint64(x uint64) (Dec, error) {
	strRep := fmt.Sprintf("%d", x)
	return NewDecFromString(strRep)
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

// NewDec takes a cosmos `sdkmath.LegacyDec` and turns it into a Dec
// it converts via strings and throws an error if the string
// is unable to be parsed
func NewDecFromSdkLegacyDec(x sdkmath.LegacyDec) (Dec, error) {
	strRep := x.String()
	return NewDecFromString(strRep)
}

// Add returns a new Dec with value `x+y` without mutating any argument and error if
// there is an overflow or resultant NaN.
func (x Dec) Add(y Dec) (Dec, error) {
	var z Dec
	_, err := apd.BaseContext.Add(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Add result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal addition error")
}

// Sub returns a new Dec with value `x-y` without mutating any argument and error if
// there is an overflow.
func (x Dec) Sub(y Dec) (Dec, error) {
	var z Dec
	_, err := apd.BaseContext.Sub(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Sub result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal subtraction error")
}

// Quo returns a new Dec with value `x/y` (formatted as decimal128, 34 digit precision) without mutating any
// argument and error if there is an overflow.
func (x Dec) Quo(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Quo(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Quo result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal quotient error")
}

// MulExact returns a new dec with value x * y. The product must not be rounded or
// ErrUnexpectedRounding will be returned.
func (x Dec) MulExact(y Dec) (Dec, error) {
	var z Dec
	condition, err := dec128Context.Mul(&z.dec, &x.dec, &y.dec)
	if err != nil {
		return z, err
	}
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "MulExact result is NaN")
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
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "QuoExact result is NaN")
	}
	if condition.Rounded() {
		return z, ErrUnexpectedRounding
	}
	return z, errorsmod.Wrap(err, "decimal quotient error")
}

// QuoInteger returns a new integral Dec with value `x/y` (formatted as decimal128, with 34 digit precision)
// without mutating any argument and error if there is an overflow.
func (x Dec) QuoInteger(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.QuoInteger(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "QuoInteger result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal quotient error")
}

// Rem returns the integral remainder from `x/y` (formatted as decimal128, with 34 digit precision) without
// mutating any argument and error if the integer part of x/y cannot fit in 34 digit precision
func (x Dec) Rem(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Rem(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Rem result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal remainder error")
}

// Mul returns a new Dec with value `x*y` (formatted as decimal128, with 34 digit precision) without
// mutating any argument and error if there is an overflow.
func (x Dec) Mul(y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Mul(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Mul result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal multiplication error")
}

// Neg negates the decimal and returns a new Dec with value `-x` without
// mutating any argument and error if there is an overflow.
func (x Dec) Neg() (Dec, error) {
	var z Dec
	_, err := dec128Context.Neg(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Neg result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal negation error")
}

// Log10 returns a new Dec with the value of the base 10 logarithm of x, without mutating x.
func Log10(x Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Log10(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Log10 result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal base 10 logarithm error")
}

// Ln returns a new Dec with the value of the natural logarithm of x, without mutating x.
func Ln(x Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Ln(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Ln result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal natural logarithm error")
}

// Exp returns a new Dec with the value of e^x, without mutating x.
func Exp(x Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Exp(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Exp result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal e to the x exponentiation error")
}

// Exp10 returns a new Dec with the value of 10^x, without mutating x.
func Exp10(x Dec) (Dec, error) {
	var ten = NewDecFromInt64(10)
	var z Dec
	_, err := dec128Context.Pow(&z.dec, &ten.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Exp10 result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal 10 to the x exponentiation error")
}

// Pow returns a new Dec with the value of x**y, without mutating x or y.
func Pow(x Dec, y Dec) (Dec, error) {
	var z Dec
	_, err := dec128Context.Pow(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrap(ErrNaN, "Pow result is NaN")
	}
	return z, errorsmod.Wrap(err, "decimal exponentiation error")
}

// returns the max of x and y without mutating x or y.
func Max(x Dec, y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "Max result is NaN")
	}
	var z Dec
	if x.Cmp(y) == GreaterThan {
		z.dec.Set(&x.dec)
	} else {
		z.dec.Set(&y.dec)
	}
	return z, nil
}

// returns the min of x and y without mutating x or y.
func Min(x Dec, y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "Min result is NaN")
	}
	var z Dec
	if x.Cmp(y) == LessThan {
		z.dec.Set(&x.dec)
	} else {
		z.dec.Set(&y.dec)
	}
	return z, nil
}

// Sqrt returns a new Dec with the value of the square root of x, without mutating x.
func (x Dec) Sqrt() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "Cannot sqrt a NaN")
	}
	var z Dec
	_, err := dec128Context.Sqrt(&z.dec, &x.dec)
	return z, errorsmod.Wrap(err, "decimal square root error")
}

// Abs returns a new Dec with the absolute value of x, without mutating x.
func (x Dec) Abs() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "Cannot abs a NaN")
	}
	var z Dec
	z.dec.Abs(&x.dec)
	return z, nil
}

// Ceil returns a new Dec with the value of x rounded up to the nearest integer, without mutating x.
func (x Dec) Ceil() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "Cannot ceil a NaN")
	}
	var z Dec
	_, err := dec128Context.Ceil(&z.dec, &x.dec)
	return z, errorsmod.Wrap(err, "decimal ceiling error")
}

// Floor returns a new Dec with the value of x rounded down to the nearest integer, without mutating x.
func (x Dec) Floor() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "Cannot floor a NaN")
	}
	var z Dec
	_, err := dec128Context.Floor(&z.dec, &x.dec)
	return z, errorsmod.Wrap(err, "decimal floor error")
}

// Int64 converts x to an int64 or returns an error if x cannot
// fit precisely into an int64.
func (x Dec) Int64() (int64, error) {
	if x.IsNaN() {
		return 0, errorsmod.Wrap(ErrNaN, "Cannot convert NaN to int64")
	}
	return x.dec.Int64()
}

// Int64 converts x to an int64 or returns an error if x cannot
// fit precisely into an int64.
func (x Dec) UInt64() (uint64, error) {
	if x.IsNaN() {
		return 0, errorsmod.Wrap(ErrNaN, "Cannot convert NaN to uint64")
	}
	val, err := x.dec.Int64()
	res := uint64(val)
	return res, err
}

// BigInt converts x to a *big.Int or returns an error if x cannot
// fit precisely into an *big.Int.
func (x Dec) BigInt() (*big.Int, error) {
	if x.IsNaN() {
		return nil, errorsmod.Wrap(ErrNaN, "Cannot convert NaN to big.Int")
	}
	y, _ := x.Reduce()
	z := &big.Int{}
	z, ok := z.SetString(y.String(), 10)
	if !ok {
		return nil, ErrNonIntegeral
	}
	return z, nil
}

// Coeff copies x into a big int while minimizing trailing zeroes
func (x Dec) Coeff() (big.Int, error) {
	if x.IsNaN() {
		return big.Int{}, errorsmod.Wrap(ErrNaN, "Cannot convert NaN to big.Int")
	}
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
	return *r.MathBigInt(), nil
}

// MaxBitLen defines the maximum bit length supported bit Int and Uint types.
const MaxBitLen = 256

// maxWordLen defines the maximum word length supported by Int and Uint types.
// We check overflow, by first doing a fast check if the word length is below maxWordLen
// and if not then do the slower full bitlen check.
// NOTE: If MaxBitLen is not a multiple of bits.UintSize, then we need to edit the used logic slightly.
const maxWordLen = MaxBitLen / bits.UintSize

// check if the big int is greater than the maximum word length
// of a sdk int (256 bits)
func bigIntOverflows(i *big.Int) bool {
	// overflow is defined as i.BitLen() > MaxBitLen
	// however this check can be expensive when doing many operations.
	// So we first check if the word length is greater than maxWordLen.
	// However the most significant word could be zero, hence we still do the bitlen check.
	if len(i.Bits()) > maxWordLen {
		return i.BitLen() > MaxBitLen
	}
	return false
}

// SdkIntTrim rounds decimal number to the integer towards zero and converts it to `sdkmath.Int`.
// returns error if the Dec is not representable in an sdkmath.Int
func (x Dec) SdkIntTrim() (sdkmath.Int, error) {
	r, err := x.Coeff()
	if err != nil {
		return sdkmath.Int{}, errorsmod.Wrap(err, "Unable to trim to sdkmath.Int")
	}
	if bigIntOverflows(&r) {
		return sdkmath.Int{}, errorsmod.Wrap(ErrOverflow, "decimal is not representable as an sdkmath.Int")
	}
	return sdkmath.NewIntFromBigInt(&r), nil
}

// SdkLegacyDec converts Dec to `sdkmath.LegacyDec`
// can return nil if the value is not representable in a LegacyDec
func (x Dec) SdkLegacyDec() (sdkmath.LegacyDec, error) {
	stringRep := x.dec.Text('f')
	return sdkmath.LegacyNewDecFromStr(stringRep)
}

func (x Dec) String() string {
	if x.IsNaN() {
		return NaNStr
	}
	return x.dec.Text('f')
}

// Marshal implements the gogo proto custom type interface.
func (x Dec) Marshal() ([]byte, error) {
	if x.IsNaN() {
		return []byte(NaNStr), nil
	}
	return x.dec.MarshalText()
}

// Unmarshal implements the gogo proto custom type interface.
func (x *Dec) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if string(data) == NaNStr {
		*x = NewNaN()
		return nil
	}

	if err := x.dec.UnmarshalText(data); err != nil {
		return err
	}

	return nil
}

// Size returns the size of the marshalled Dec type in bytes
func (x Dec) Size() int {
	bz, _ := x.Marshal()
	return len(bz)
}

// MarshalTo implements the gogo proto custom type interface.
func (x *Dec) MarshalTo(data []byte) (n int, err error) {
	bz, err := x.Marshal()
	if err != nil {
		return 0, err
	}

	copy(data, bz)
	return len(bz), nil
}

// MarshalJSON marshals the decimal
func (x Dec) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

// UnmarshalJSON defines custom decoding scheme
func (x *Dec) UnmarshalJSON(bz []byte) error {
	var text string
	err := json.Unmarshal(bz, &text)
	if err != nil {
		return err
	}
	if text == NaNStr {
		*x = NewNaN()
		return nil
	}

	newDec, err := NewDecFromString(text)
	if err != nil {
		return err
	}

	*x = newDec

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

// is x greater than y
func (x Dec) Gt(y Dec) bool {
	return x.dec.Cmp(&y.dec) == 1
}

// is x greater than or equal to y
func (x Dec) Gte(y Dec) bool {
	return x.dec.Cmp(&y.dec) == 1 || x.dec.Cmp(&y.dec) == 0
}

// is x less than y
func (x Dec) Lt(y Dec) bool {
	return x.dec.Cmp(&y.dec) == -1
}

// is x less than or equal to y
func (x Dec) Lte(y Dec) bool {
	return x.dec.Cmp(&y.dec) == -1 || x.dec.Cmp(&y.dec) == 0
}

// Equal returns true if x and y are equal.
func (x Dec) Equal(y Dec) bool {
	return x.dec.Cmp(&y.dec) == 0
}

// IsNaN returns true if the decimal is not a number.
func (x Dec) IsNaN() bool {
	return x.isNaN
}

// IsZero returns true if the decimal is zero.
func (x Dec) IsZero() bool {
	if x.IsNaN() {
		return false
	}
	return x.dec.IsZero()
}

// IsNegative returns true if the decimal is negative.
func (x Dec) IsNegative() bool {
	if x.IsNaN() {
		return false
	}
	return x.dec.Negative && !x.dec.IsZero()
}

// IsPositive returns true if the decimal is positive.
func (x Dec) IsPositive() bool {
	if x.IsNaN() {
		return false
	}
	return !x.dec.Negative && !x.dec.IsZero()
}

// IsFinite returns true if the decimal is finite.
func (x Dec) IsFinite() bool {
	if x.IsNaN() {
		return false
	}
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
func InDelta(expected, result Dec, epsilon Dec) (bool, error) {
	if expected.IsNaN() || result.IsNaN() {
		return false, errorsmod.Wrap(ErrNaN, "Cannot compare NaN")
	}
	delta, err := expected.Sub(result)
	if err != nil {
		return false, nil
	}
	deltaAbs, err := delta.Abs()
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting absolute value")
	}
	compare := deltaAbs.Cmp(epsilon)
	if compare == LessThan || compare == EqualTo {
		return true, nil
	}
	return false, nil
}

// Helper function to compare two slices of alloraMath.Dec within a delta
func SlicesInDelta(a, b []Dec, epsilon Dec) (bool, error) {
	lenA := len(a)
	if lenA != len(b) {
		return false, errorsmod.Wrap(ErrNotMatchingLength, "Unable to check if slices are within delta")
	}
	for i := 0; i < lenA; i++ {
		// for performance reasons we do not call InDelta
		// pass by copy causes this to run slow af for large slices
		delta, err := a[i].Sub(b[i])
		if err != nil {
			return false, errorsmod.Wrapf(err, "error subtracting %v from %v", b[i], a[i])
		}
		deltaAbs, err := delta.Abs()
		if err != nil {
			return false, errorsmod.Wrap(err, "error getting absolute value")
		}
		if deltaAbs.Cmp(epsilon) == GreaterThan {
			return false, nil
		}
	}
	return true, nil
}

// Generic Sum function, given an array of values returns its sum
func SumDecSlice(x []Dec) (Dec, error) {
	sum := ZeroDec()
	var err error
	for _, v := range x {
		sum, err = sum.Add(v)
		if err != nil {
			return Dec{}, errorsmod.Wrapf(err, "error adding %v + %v", v, sum)
		}
	}
	return sum, nil
}
