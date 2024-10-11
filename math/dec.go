// This code is forked from Regen Ledger.
// Their code is under the Apache2.0 License
// https://github.com/regen-network/regen-ledger/blob/3d818cf6e01af92eed25de5c17728a79070f56a3/types/math/dec.go

package math

import (
	"encoding/json"
	"fmt"
	goMath "math"
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
	ErrNonIntegral        = errorsmod.Register(mathCodespace, 3, "value is non-integral")
	ErrInfiniteString     = errorsmod.Register(mathCodespace, 4, "value is infinite")
	ErrOverflow           = errorsmod.Register(mathCodespace, 5, "overflow")
	ErrNaN                = errorsmod.Register(mathCodespace, 6, "NaN not permitted in this context")
	ErrNotMatchingLength  = errorsmod.Register(mathCodespace, 7, "slices are not of the same length")
	ErrOutOfRange         = errorsmod.Register(mathCodespace, 8, "value is out of range")
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
	Rounding:    apd.RoundDown,
}

// create a new Dec that represents NaN
func NewNaN() Dec {
	return Dec{
		apd.Decimal{
			Form:     apd.Finite,
			Negative: false,
			Exponent: 0,
			Coeff:    *apd.NewBigInt(0),
		},
		true,
	}
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
		return Dec{}, errorsmod.Wrapf(ErrInvalidDecString, "error trying to parse %s %s", s, err.Error())
	}
	if d.IsNegative() {
		return Dec{}, errorsmod.Wrapf(ErrInvalidDecString, "expected a non-negative decimal, got %s", s)
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
		return Dec{}, errorsmod.Wrapf(ErrInvalidDecString, "%s exceeds maximum decimal places: %d", s, max)
	}
	return d, nil
}

// NewPositiveDecFromString returns a new Dec from a string,
// returning an error if the string cannot be parsed or if the
// decimal is not positive. The string should be in the format of `123.456`.
func NewPositiveDecFromString(s string) (Dec, error) {
	d, err := NewDecFromString(s)
	if err != nil {
		return Dec{}, errorsmod.Wrapf(ErrInvalidDecString, "error trying to parse %s %s", s, err.Error())
	}
	if !d.IsPositive() || !d.IsFinite() {
		return Dec{}, errorsmod.Wrapf(ErrInvalidDecString, "expected a positive decimal, got %s", s)
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
		return Dec{}, errorsmod.Wrapf(ErrInvalidDecString, "%s exceeds maximum decimal places: %d", s, max)
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
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot add with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := apd.BaseContext.Add(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Add result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal addition error %s %s", x.String(), y.String())
}

// Sub returns a new Dec with value `x-y` without mutating any argument and error if
// there is an overflow.
func (x Dec) Sub(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot subtract with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := apd.BaseContext.Sub(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Sub result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal subtraction error %s %s", x.String(), y.String())
}

// Quo returns a new Dec with value `x/y` (formatted as decimal128, 34 digit precision) without mutating any
// argument and error if there is an overflow.
func (x Dec) Quo(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot divide with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := dec128Context.Quo(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Quo result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal quotient error %s %s", x.String(), y.String())
}

// MulExact returns a new dec with value x * y. The product must not be rounded or
// ErrUnexpectedRounding will be returned.
func (x Dec) MulExact(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot MulExact with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	condition, err := dec128Context.Mul(&z.dec, &x.dec, &y.dec)
	if err != nil {
		return z, err
	}
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "MulExact result is NaN %s %s", x.String(), y.String())
	}
	if condition.Rounded() {
		return z, errorsmod.Wrapf(ErrUnexpectedRounding, "MulExact has unexpected rounding %s %s", x.String(), y.String())
	}
	return z, nil
}

// QuoExact is a version of Quo that returns ErrUnexpectedRounding if any rounding occurred.
func (x Dec) QuoExact(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot QuoExact with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	condition, err := dec128Context.Quo(&z.dec, &x.dec, &y.dec)
	if err != nil {
		return z, err
	}
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "QuoExact result is NaN %s %s", x.String(), y.String())
	}
	if condition.Rounded() {
		return z, errorsmod.Wrapf(ErrUnexpectedRounding, "QuoExact has unexpected rounding %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal quotient error %s %s", x.String(), y.String())
}

// QuoInteger returns a new integral Dec with value `x/y` (formatted as decimal128, with 34 digit precision)
// without mutating any argument and error if there is an overflow.
func (x Dec) QuoInteger(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot QuoInteger with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := dec128Context.QuoInteger(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "QuoInteger result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal quotient error %s %s", x.String(), y.String())
}

// Rem returns the integral remainder from `x/y` (formatted as decimal128, with 34 digit precision) without
// mutating any argument and error if the integer part of x/y cannot fit in 34 digit precision
func (x Dec) Rem(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Rem with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := dec128Context.Rem(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Rem result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal remainder error %s %s", x.String(), y.String())
}

// Mul returns a new Dec with value `x*y` (formatted as decimal128, with 34 digit precision) without
// mutating any argument and error if there is an overflow.
func (x Dec) Mul(y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Mul with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := dec128Context.Mul(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Mul result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal multiplication error %s %s", x.String(), y.String())
}

// Neg negates the decimal and returns a new Dec with value `-x` without
// mutating any argument and error if there is an overflow.
func (x Dec) Neg() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Neg a NaN %s", x.String())
	}
	var z Dec
	_, err := dec128Context.Neg(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Neg result is NaN %s", x.String())
	}
	return z, errorsmod.Wrapf(err, "decimal negation error %s", x.String())
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
	return z, errorsmod.Wrapf(err, "decimal base 10 logarithm error %s", x.String())
}

// Ln returns a new Dec with the value of the natural logarithm of x, without mutating x.
func Ln(x Dec) (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Ln a NaN %s", x.String())
	}
	var z Dec
	_, err := dec128Context.Ln(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Ln result is NaN %s", x.String())
	}
	return z, errorsmod.Wrapf(err, "decimal natural logarithm error %s", x.String())
}

// Exp returns a new Dec with the value of e^x, without mutating x.
func Exp(x Dec) (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Exp a NaN %s", x.String())
	}
	var z Dec
	_, err := dec128Context.Exp(&z.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Exp result is NaN %s", x.String())
	}
	return z, errorsmod.Wrapf(err, "decimal e to the x exponentiation error %s", x.String())
}

// Exp10 returns a new Dec with the value of 10^x, without mutating x.
func Exp10(x Dec) (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Exp10 a NaN %s", x.String())
	}
	var ten = NewDecFromInt64(10)
	var z Dec
	_, err := dec128Context.Pow(&z.dec, &ten.dec, &x.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Exp10 result is NaN %s", x.String())
	}
	return z, errorsmod.Wrapf(err, "decimal 10 to the x exponentiation error %s", x.String())
}

// Pow returns a new Dec with the value of x**y, without mutating x or y.
func Pow(x Dec, y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Pow with a NaN argument %s %s", x.String(), y.String())
	}
	var z Dec
	_, err := dec128Context.Pow(&z.dec, &x.dec, &y.dec)
	if z.IsNaN() {
		return z, errorsmod.Wrapf(ErrNaN, "Pow result is NaN %s %s", x.String(), y.String())
	}
	return z, errorsmod.Wrapf(err, "decimal exponentiation error %s %s", x.String(), y.String())
}

// returns the max of x and y without mutating x or y.
func Max(x Dec, y Dec) (Dec, error) {
	if x.IsNaN() || y.IsNaN() {
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Max with a NaN argument %s %s", x.String(), y.String())
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
		return Dec{}, errorsmod.Wrapf(ErrNaN, "cannot Min with a NaN argument %s %s", x.String(), y.String())
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
		return Dec{}, errorsmod.Wrap(ErrNaN, "cannot Sqrt a NaN")
	}
	var z Dec
	_, err := dec128Context.Sqrt(&z.dec, &x.dec)
	return z, errorsmod.Wrap(err, "decimal square root error")
}

// Abs returns a new Dec with the absolute value of x, without mutating x.
func (x Dec) Abs() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "cannot Abs a NaN")
	}
	var z Dec
	z.dec.Abs(&x.dec)
	return z, nil
}

// Ceil returns a new Dec with the value of x rounded up to the nearest integer, without mutating x.
func (x Dec) Ceil() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "cannot Ceil a NaN")
	}
	var z Dec
	_, err := dec128Context.Ceil(&z.dec, &x.dec)
	if err != nil {
		return z, errorsmod.Wrapf(err, "decimal ceiling error %s", x.String())
	}
	return z, nil
}

// Floor returns a new Dec with the value of x rounded down to the nearest integer, without mutating x.
func (x Dec) Floor() (Dec, error) {
	if x.IsNaN() {
		return Dec{}, errorsmod.Wrap(ErrNaN, "cannot Floor a NaN")
	}
	var z Dec
	_, err := dec128Context.Floor(&z.dec, &x.dec)
	if err != nil {
		return z, errorsmod.Wrapf(err, "decimal floor error %s", x.String())
	}
	return z, nil
}

// Int64 converts x to an int64 or returns an error if x cannot
// fit precisely into an int64.
func (x Dec) Int64() (int64, error) {
	if x.IsNaN() {
		return 0, errorsmod.Wrap(ErrNaN, "cannot convert NaN to int64")
	}
	return x.dec.Int64()
}

// Int64 converts x to an int64 or returns an error if x cannot
// fit precisely into an int64.
func (x Dec) UInt64() (uint64, error) {
	if x.IsNaN() {
		return 0, errorsmod.Wrap(ErrNaN, "cannot convert NaN to uint64")
	}
	bigInt, err := x.BigInt()
	if err != nil {
		return 0, errorsmod.Wrapf(err, "cannot convert to uint64 %s", x.String())
	}
	bigMaxUint := big.Int{}
	bigMaxUint.SetUint64(goMath.MaxUint64)
	if bigInt.Cmp(&bigMaxUint) == GreaterThan {
		return 0, errorsmod.Wrapf(ErrOverflow, "decimal is not representable as an uint64 %s", x.String())
	}
	if bigInt.Sign() == -1 {
		return 0, errorsmod.Wrapf(ErrOverflow, "negative number cannot be represented in uint64 %s", x.String())
	}
	return bigInt.Uint64(), nil
}

// BigInt converts x to a *big.Int or returns an error if x cannot
// fit precisely into an *big.Int.
func (x Dec) BigInt() (*big.Int, error) {
	if x.IsNaN() {
		return nil, errorsmod.Wrap(ErrNaN, "cannot convert NaN to big.Int")
	}
	y, _ := x.Reduce()
	z := &big.Int{}
	z, ok := z.SetString(y.String(), 10)
	if !ok {
		return nil, ErrNonIntegral
	}
	return z, nil
}

// Coeff copies x into a big int while minimizing trailing zeroes
func (x Dec) Coeff() (big.Int, error) {
	if x.IsNaN() {
		return big.Int{}, errorsmod.Wrap(ErrNaN, "cannot convert NaN to big.Int")
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
	if x.IsNaN() {
		return sdkmath.Int{}, errorsmod.Wrap(ErrNaN, "cannot trim NaN to sdkmath.Int")
	}
	r, err := x.Coeff()
	if err != nil {
		return sdkmath.Int{}, errorsmod.Wrapf(err, "cannot trim to sdkmath.Int %s", x.String())
	}
	if bigIntOverflows(&r) {
		return sdkmath.Int{}, errorsmod.Wrapf(ErrOverflow, "decimal is not representable as an sdkmath.Int %s", x.String())
	}
	return sdkmath.NewIntFromBigInt(&r), nil
}

// SdkLegacyDec converts Dec to `sdkmath.LegacyDec`
// can return nil if the value is not representable in a LegacyDec
func (x Dec) SdkLegacyDec() (sdkmath.LegacyDec, error) {
	if x.IsNaN() {
		return sdkmath.LegacyDec{}, errorsmod.Wrap(ErrNaN, "cannot convert NaN to sdkmath.LegacyDec")
	}
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
	y := ZeroDec()
	_, n := y.dec.Reduce(&x.dec)
	return y, n
}
