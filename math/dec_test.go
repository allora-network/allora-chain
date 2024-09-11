// This code is forked from Regen Ledger.
// Their code is under the Apache2.0 License
// https://github.com/regen-network/regen-ledger/blob/3d818cf6e01af92eed25de5c17728a79070f56a3/types/math/dec_test.go

package math_test

import (
	"fmt"
	goMath "math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDec(t *testing.T) {
	// Property tests
	t.Run("TestNewDecFromInt64", rapid.MakeCheck(testDecInt64))

	// Properties about *FromString functions
	t.Run("TestInvalidNewDecFromString", rapid.MakeCheck(testInvalidNewDecFromString))
	t.Run("TestInvalidNewNonNegativeDecFromString", rapid.MakeCheck(testInvalidNewNonNegativeDecFromString))
	t.Run("TestInvalidNewNonNegativeFixedDecFromString", rapid.MakeCheck(testInvalidNewNonNegativeFixedDecFromString))
	t.Run("TestInvalidNewPositiveDecFromString", rapid.MakeCheck(testInvalidNewPositiveDecFromString))
	t.Run("TestInvalidNewPositiveFixedDecFromString", rapid.MakeCheck(testInvalidNewPositiveFixedDecFromString))

	// Properties about addition
	t.Run("TestAddLeftIdentity", rapid.MakeCheck(testAddLeftIdentity))
	t.Run("TestAddRightIdentity", rapid.MakeCheck(testAddRightIdentity))
	t.Run("TestAddCommutative", rapid.MakeCheck(testAddCommutative))
	t.Run("TestAddAssociative", rapid.MakeCheck(testAddAssociative))

	// Properties about subtraction
	t.Run("TestSubRightIdentity", rapid.MakeCheck(testSubRightIdentity))
	t.Run("TestSubZero", rapid.MakeCheck(testSubZero))

	// Properties about multiplication
	t.Run("TestMulLeftIdentity", rapid.MakeCheck(testMulLeftIdentity))
	t.Run("TestMulRightIdentity", rapid.MakeCheck(testMulRightIdentity))
	t.Run("TestMulCommutative", rapid.MakeCheck(testMulCommutative))
	t.Run("TestMulAssociative", rapid.MakeCheck(testMulAssociative))
	t.Run("TestZeroIdentity", rapid.MakeCheck(testMulZero))

	// Properties about division
	t.Run("TestDivisionBySelf", rapid.MakeCheck(testSelfQuo))
	t.Run("TestDivisionByOne", rapid.MakeCheck(testQuoByOne))

	// Properties combining operations
	t.Run("TestSubAdd", rapid.MakeCheck(testSubAdd))
	t.Run("TestAddSub", rapid.MakeCheck(testAddSub))
	t.Run("TestMulQuoA", rapid.MakeCheck(testMulQuoA))
	t.Run("TestMulQuoB", rapid.MakeCheck(testMulQuoB))
	t.Run("TestMulQuoExact", rapid.MakeCheck(testMulQuoExact))
	t.Run("TestQuoMulExact", rapid.MakeCheck(testQuoMulExact))

	// Properties about comparison and equality
	t.Run("TestCmpInverse", rapid.MakeCheck(testCmpInverse))
	t.Run("TestEqualCommutative", rapid.MakeCheck(testEqualCommutative))

	// Properties about tests on a single Dec
	t.Run("TestIsZero", rapid.MakeCheck(testIsZero))
	t.Run("TestIsNegative", rapid.MakeCheck(testIsNegative))
	t.Run("TestIsPositive", rapid.MakeCheck(testIsPositive))
	t.Run("TestNumDecimalPlaces", rapid.MakeCheck(testNumDecimalPlaces))

	// Unit tests
	zero := alloraMath.Dec{}
	one := alloraMath.NewDecFromInt64(1)
	two := alloraMath.NewDecFromInt64(2)
	three := alloraMath.NewDecFromInt64(3)
	four := alloraMath.NewDecFromInt64(4)
	five := alloraMath.NewDecFromInt64(5)
	minusOne := alloraMath.NewDecFromInt64(-1)

	onePointOneFive, err := alloraMath.NewDecFromString("1.15")
	require.NoError(t, err)
	twoPointThreeFour, err := alloraMath.NewDecFromString("2.34")
	require.NoError(t, err)
	threePointFourNine, err := alloraMath.NewDecFromString("3.49")
	require.NoError(t, err)
	onePointFourNine, err := alloraMath.NewDecFromString("1.49")
	require.NoError(t, err)
	minusFivePointZero, err := alloraMath.NewDecFromString("-5.0")
	require.NoError(t, err)

	twoThousand := alloraMath.NewDecFinite(2, 3)
	require.True(t, twoThousand.Equal(alloraMath.NewDecFromInt64(2000)))

	res, err := two.Add(zero)
	require.NoError(t, err)
	require.True(t, res.Equal(two))

	res, err = five.Sub(two)
	require.NoError(t, err)
	require.True(t, res.Equal(three))

	/*
		res, err = SafeSubBalance(five, two)
		require.NoError(t, err)
		require.True(t, res.Equal(three))

		_, err = SafeSubBalance(two, five)
		require.Error(t, err, "Expected insufficient funds error")

		res, err = SafeAddBalance(three, two)
		require.NoError(t, err)
		require.True(t, res.Equal(five))

		_, err = SafeAddBalance(minusFivePointZero, five)
		require.Error(t, err, "Expected ErrInvalidRequest")
	*/

	res, err = four.Quo(two)
	require.NoError(t, err)
	require.True(t, res.Equal(two))

	res, err = five.QuoInteger(two)
	require.NoError(t, err)
	require.True(t, res.Equal(two))

	res, err = five.Rem(two)
	require.NoError(t, err)
	require.True(t, res.Equal(one))

	x, err := four.Int64()
	require.NoError(t, err)
	require.Equal(t, int64(4), x)

	require.Equal(t, "5", five.String())

	res, err = onePointOneFive.Add(twoPointThreeFour)
	require.NoError(t, err)
	require.True(t, res.Equal(threePointFourNine))

	res, err = threePointFourNine.Sub(two)
	require.NoError(t, err)
	require.True(t, res.Equal(onePointFourNine))

	res, err = minusOne.Sub(four)
	require.NoError(t, err)
	require.True(t, res.Equal(minusFivePointZero))

	require.True(t, zero.IsZero())
	require.False(t, zero.IsPositive())
	require.False(t, zero.IsNegative())

	require.False(t, one.IsZero())
	require.True(t, one.IsPositive())
	require.False(t, one.IsNegative())

	require.False(t, minusOne.IsZero())
	require.False(t, minusOne.IsPositive())
	require.True(t, minusOne.IsNegative())

	res, err = one.MulExact(two)
	require.NoError(t, err)
	require.True(t, res.Equal(two))

	ten := alloraMath.NewDecFromInt64(10)
	oneHundred := alloraMath.NewDecFromInt64(100)
	logTenOneHundred, err := alloraMath.Log10(oneHundred)
	require.NoError(t, err)
	require.True(t, two.Equal(logTenOneHundred))
	logTenTen, err := alloraMath.Log10(ten)
	require.NoError(t, err)
	require.True(t, one.Equal(logTenTen))
	logTenOne, err := alloraMath.Log10(one)
	require.NoError(t, err)
	require.True(t, zero.Equal(logTenOne))

	logEOne, err := alloraMath.Ln(one)
	require.NoError(t, err)
	require.True(t, zero.Equal(logEOne))

	eight := alloraMath.NewDecFromInt64(8)
	twoCubed, err := alloraMath.Pow(two, three)
	require.NoError(t, err)
	require.True(t, eight.Equal(twoCubed))

	oneThousand := alloraMath.NewDecFromInt64(1000)
	tenSquared, err := alloraMath.Exp10(two)
	require.NoError(t, err)
	require.True(t, oneHundred.Equal(tenSquared))
	tenCubed, err := alloraMath.Exp10(three)
	require.NoError(t, err)
	require.True(t, oneThousand.Equal(tenCubed))

	cielOnePointFourNine, err := onePointFourNine.Ceil()
	require.NoError(t, err)
	require.True(t, two.Equal(cielOnePointFourNine))

	onePointFiveOne, err := alloraMath.NewDecFromString("1.51")
	require.NoError(t, err)
	floorOnePointFiveOne, err := onePointFiveOne.Floor()
	require.NoError(t, err)
	require.True(t, one.Equal(floorOnePointFiveOne))
}

// generate a dec value on the fly
var genDec *rapid.Generator[alloraMath.Dec] = rapid.Custom(func(t *rapid.T) alloraMath.Dec {
	f := rapid.Float64().Draw(t, "f")
	dec, err := alloraMath.NewDecFromString(fmt.Sprintf("%g", f))
	require.NoError(t, err)
	return dec
})

// A Dec value and the float used to create it
type floatAndDec struct {
	float float64
	dec   alloraMath.Dec
}

// Generate a Dec value along with the float used to create it
var genFloatAndDec *rapid.Generator[floatAndDec] = rapid.Custom(func(t *rapid.T) floatAndDec {
	f := rapid.Float64().Draw(t, "f")
	dec, err := alloraMath.NewDecFromString(fmt.Sprintf("%g", f))
	require.NoError(t, err)
	return floatAndDec{f, dec}
})

// Property: n == alloraMath.NewDecFromInt64(n).Int64()
func testDecInt64(t *rapid.T) {
	nIn := rapid.Int64().Draw(t, "n")
	nOut, err := alloraMath.NewDecFromInt64(nIn).Int64()

	require.NoError(t, err)
	require.Equal(t, nIn, nOut)
}

// Property: invalid_number_string(s) => alloraMath.NewDecFromString(s) == err
func testInvalidNewDecFromString(t *rapid.T) {
	s := rapid.StringMatching("[[:alpha:]]+").Draw(t, "s")
	_, err := alloraMath.NewDecFromString(s)
	require.Error(t, err)
}

// Property: invalid_number_string(s) || IsNegative(s)
// => NewNonNegativeDecFromString(s) == err
func testInvalidNewNonNegativeDecFromString(t *rapid.T) {
	s := rapid.OneOf(
		rapid.StringMatching("[[:alpha:]]+"),
		rapid.StringMatching(`^-\d*\.?\d+$`).Filter(
			func(s string) bool { return !strings.HasPrefix(s, "-0") && !strings.HasPrefix(s, "-.0") },
		),
	).Draw(t, "s")
	_, err := alloraMath.NewNonNegativeDecFromString(s)
	require.Error(t, err)
}

// Property: invalid_number_string(s) || IsNegative(s) || NumDecimals(s) > n
// => NewNonNegativeFixedDecFromString(s, n) == err
func testInvalidNewNonNegativeFixedDecFromString(t *rapid.T) {
	n := rapid.Uint32Range(0, 999).Draw(t, "n")
	s := rapid.OneOf(
		rapid.StringMatching("[[:alpha:]]+"),
		rapid.StringMatching(`^-\d*\.?\d+$`).Filter(
			func(s string) bool { return !strings.HasPrefix(s, "-0") && !strings.HasPrefix(s, "-.0") },
		),
		rapid.StringMatching(fmt.Sprintf(`\d*\.\d{%d,}`, n+1)),
	).Draw(t, "s")
	_, err := alloraMath.NewNonNegativeFixedDecFromString(s, n)
	require.Error(t, err)
}

// Property: invalid_number_string(s) || IsNegative(s) || IsZero(s)
// => NewPositiveDecFromString(s) == err
func testInvalidNewPositiveDecFromString(t *rapid.T) {
	s := rapid.OneOf(
		rapid.StringMatching("[[:alpha:]]+"),
		rapid.StringMatching(`^-\d*\.?\d+|0$`),
	).Draw(t, "s")
	_, err := alloraMath.NewPositiveDecFromString(s)
	require.Error(t, err)
}

// Property: invalid_number_string(s) || IsNegative(s) || IsZero(s) || NumDecimals(s) > n
// => NewPositiveFixedDecFromString(s) == err
func testInvalidNewPositiveFixedDecFromString(t *rapid.T) {
	n := rapid.Uint32Range(0, 999).Draw(t, "n")
	s := rapid.OneOf(
		rapid.StringMatching("[[:alpha:]]+"),
		rapid.StringMatching(`^-\d*\.?\d+|0$`),
		rapid.StringMatching(fmt.Sprintf(`\d*\.\d{%d,}`, n+1)),
	).Draw(t, "s")
	_, err := alloraMath.NewPositiveFixedDecFromString(s, n)
	require.Error(t, err)
}

// Property: 0 + a == a
func testAddLeftIdentity(t *rapid.T) {
	a := genDec.Draw(t, "a")
	zero := alloraMath.NewDecFromInt64(0)

	b, err := zero.Add(a)
	require.NoError(t, err)

	require.True(t, a.Equal(b))
}

// Property: a + 0 == a
func testAddRightIdentity(t *rapid.T) {
	a := genDec.Draw(t, "a")
	zero := alloraMath.NewDecFromInt64(0)

	b, err := a.Add(zero)
	require.NoError(t, err)

	require.True(t, a.Equal(b))
}

// Property: a + b == b + a
func testAddCommutative(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")

	c, err := a.Add(b)
	require.NoError(t, err)

	d, err := b.Add(a)
	require.NoError(t, err)

	require.True(t, c.Equal(d))
}

// Property: (a + b) + c == a + (b + c)
func testAddAssociative(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")
	c := genDec.Draw(t, "c")

	// (a + b) + c
	d, err := a.Add(b)
	require.NoError(t, err)

	e, err := d.Add(c)
	require.NoError(t, err)

	// a + (b + c)
	f, err := b.Add(c)
	require.NoError(t, err)

	g, err := a.Add(f)
	require.NoError(t, err)

	require.True(t, e.Equal(g))
}

// Property: a - 0 == a
func testSubRightIdentity(t *rapid.T) {
	a := genDec.Draw(t, "a")
	zero := alloraMath.NewDecFromInt64(0)

	b, err := a.Sub(zero)
	require.NoError(t, err)

	require.True(t, a.Equal(b))
}

// Property: a - a == 0
func testSubZero(t *rapid.T) {
	a := genDec.Draw(t, "a")
	zero := alloraMath.NewDecFromInt64(0)

	b, err := a.Sub(a)
	require.NoError(t, err)

	require.True(t, b.Equal(zero))
}

// Property: 1 * a == a
func testMulLeftIdentity(t *rapid.T) {
	a := genDec.Draw(t, "a")
	one := alloraMath.NewDecFromInt64(1)

	b, err := one.Mul(a)
	require.NoError(t, err)

	require.True(t, a.Equal(b))
}

// Property: a * 1 == a
func testMulRightIdentity(t *rapid.T) {
	a := genDec.Draw(t, "a")
	one := alloraMath.NewDecFromInt64(1)

	b, err := a.Mul(one)
	require.NoError(t, err)

	require.True(t, a.Equal(b))
}

// Property: a * b == b * a
func testMulCommutative(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")

	c, err := a.Mul(b)
	require.NoError(t, err)

	d, err := b.Mul(a)
	require.NoError(t, err)

	require.True(t, c.Equal(d))
}

// Property: (a * b) * c == a * (b * c)
func testMulAssociative(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")
	c := genDec.Draw(t, "c")

	// (a * b) * c
	d, err := a.Mul(b)
	require.NoError(t, err)

	e, err := d.Mul(c)
	require.NoError(t, err)

	// a * (b * c)
	f, err := b.Mul(c)
	require.NoError(t, err)

	g, err := a.Mul(f)
	require.NoError(t, err)

	require.True(t, e.Equal(g))
}

// Property: (a - b) + b == a
func testSubAdd(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")

	c, err := a.Sub(b)
	require.NoError(t, err)

	d, err := c.Add(b)
	require.NoError(t, err)

	require.True(t, a.Equal(d))
}

// Property: (a + b) - b == a
func testAddSub(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")

	c, err := a.Add(b)
	require.NoError(t, err)

	d, err := c.Sub(b)
	require.NoError(t, err)

	require.True(t, a.Equal(d))
}

// Property: a * 0 = 0
func testMulZero(t *rapid.T) {
	a := genDec.Draw(t, "a")
	zero := alloraMath.Dec{}

	c, err := a.Mul(zero)
	require.NoError(t, err)
	require.True(t, c.IsZero())
}

// Property: a/a = 1
func testSelfQuo(t *rapid.T) {
	decNotZero := func(d alloraMath.Dec) bool { return !d.IsZero() }
	a := genDec.Filter(decNotZero).Draw(t, "a")
	one := alloraMath.NewDecFromInt64(1)

	b, err := a.Quo(a)
	require.NoError(t, err)
	require.True(t, one.Equal(b))
}

// Property: a/1 = a
func testQuoByOne(t *rapid.T) {
	a := genDec.Draw(t, "a")
	one := alloraMath.NewDecFromInt64(1)

	b, err := a.Quo(one)
	require.NoError(t, err)
	require.True(t, a.Equal(b))
}

// Property: (a * b) / a == b
func testMulQuoA(t *rapid.T) {
	decNotZero := func(d alloraMath.Dec) bool { return !d.IsZero() }
	a := genDec.Filter(decNotZero).Draw(t, "a")
	b := genDec.Draw(t, "b")

	c, err := a.Mul(b)
	require.NoError(t, err)

	d, err := c.Quo(a)
	require.NoError(t, err)

	require.True(t, b.Equal(d))
}

// Property: (a * b) / b == a
func testMulQuoB(t *rapid.T) {
	decNotZero := func(d alloraMath.Dec) bool { return !d.IsZero() }
	a := genDec.Draw(t, "a")
	b := genDec.Filter(decNotZero).Draw(t, "b")

	c, err := a.Mul(b)
	require.NoError(t, err)

	d, err := c.Quo(b)
	require.NoError(t, err)

	require.True(t, a.Equal(d))
}

// Property: (a * 10^b) / 10^b == a using MulExact and QuoExact
// and a with no more than b decimal places (b <= 32).
func testMulQuoExact(t *rapid.T) {
	b := rapid.Int32Range(0, 32).Draw(t, "b")
	decPrec := func(d alloraMath.Dec) bool { return d.NumDecimalPlaces() <= uint32(b) }
	a := genDec.Filter(decPrec).Draw(t, "a")

	c := alloraMath.NewDecFinite(1, b)

	d, err := a.MulExact(c)
	require.NoError(t, err)

	e, err := d.QuoExact(c)
	require.NoError(t, err)

	require.True(t, a.Equal(e))
}

// Property: (a / b) * b == a using QuoExact and MulExact and
// a as an integer.
func testQuoMulExact(t *rapid.T) {
	a := rapid.Uint64().Draw(t, "a")
	aDec, err := alloraMath.NewDecFromString(fmt.Sprintf("%d", a))
	require.NoError(t, err)
	b := rapid.Int32Range(0, 32).Draw(t, "b")
	c := alloraMath.NewDecFinite(1, b)

	require.NoError(t, err)

	d, err := aDec.QuoExact(c)
	require.NoError(t, err)

	e, err := d.MulExact(c)
	require.NoError(t, err)

	require.True(t, aDec.Equal(e))
}

// Property: Cmp(a, b) == -Cmp(b, a)
func testCmpInverse(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")

	require.Equal(t, a.Cmp(b), -b.Cmp(a))
}

// Property: Equal(a, b) == Equal(b, a)
func testEqualCommutative(t *rapid.T) {
	a := genDec.Draw(t, "a")
	b := genDec.Draw(t, "b")

	require.Equal(t, a.Equal(b), b.Equal(a))
}

// Property: isZero(f) == isZero(NewDecFromString(f.String()))
func testIsZero(t *rapid.T) {
	floatAndDec := genFloatAndDec.Draw(t, "floatAndDec")
	f, dec := floatAndDec.float, floatAndDec.dec

	require.Equal(t, f == 0, dec.IsZero())
}

// Property: isNegative(f) == isNegative(NewDecFromString(f.String()))
func testIsNegative(t *rapid.T) {
	floatAndDec := genFloatAndDec.Draw(t, "floatAndDec")
	f, dec := floatAndDec.float, floatAndDec.dec

	require.Equal(t, f < 0, dec.IsNegative())
}

// Property: isPositive(f) == isPositive(NewDecFromString(f.String()))
func testIsPositive(t *rapid.T) {
	floatAndDec := genFloatAndDec.Draw(t, "floatAndDec")
	f, dec := floatAndDec.float, floatAndDec.dec

	require.Equal(t, f > 0, dec.IsPositive())
}

// Property: floatDecimalPlaces(f) == NumDecimalPlaces(NewDecFromString(f.String()))
func testNumDecimalPlaces(t *rapid.T) {
	floatAndDec := genFloatAndDec.Draw(t, "floatAndDec")
	f, dec := floatAndDec.float, floatAndDec.dec

	require.Equal(t, floatDecimalPlaces(t, f), dec.NumDecimalPlaces())
}

func floatDecimalPlaces(t *rapid.T, f float64) uint32 {
	reScientific := regexp.MustCompile(`^\-?(?:[[:digit:]]+(?:\.([[:digit:]]+))?|\.([[:digit:]]+))(?:e?(?:\+?([[:digit:]]+)|(-[[:digit:]]+)))?$`)
	fStr := fmt.Sprintf("%g", f)
	matches := reScientific.FindAllStringSubmatch(fStr, 1)
	if len(matches) != 1 {
		t.Fatalf("Didn't match float: %g", f)
	}

	// basePlaces is the number of decimal places in the decimal part of the
	// string
	basePlaces := 0
	if matches[0][1] != "" {
		basePlaces = len(matches[0][1])
	} else if matches[0][2] != "" {
		basePlaces = len(matches[0][2])
	}
	t.Logf("Base places: %d", basePlaces)

	// exp is the exponent
	exp := 0
	if matches[0][3] != "" {
		var err error
		exp, err = strconv.Atoi(matches[0][3])
		require.NoError(t, err)
	} else if matches[0][4] != "" {
		var err error
		exp, err = strconv.Atoi(matches[0][4])
		require.NoError(t, err)
	}

	// Subtract exponent from base and check if negative
	res := basePlaces - exp
	if res <= 0 {
		return 0
	}

	return uint32(res) //nolint:gosec //G115: integer overflow conversion int -> uint32
}

func TestIsFinite(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1.5")
	require.NoError(t, err)

	require.True(t, a.IsFinite())

	b, err := alloraMath.NewDecFromString("NaN")
	require.NoError(t, err)

	require.False(t, b.IsFinite())
}

func TestReduce(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1.30000")
	require.NoError(t, err)
	b, n := a.Reduce()
	require.Equal(t, 4, n)
	require.True(t, a.Equal(b))
	require.Equal(t, "1.3", b.String())
}

func TestMulExactGood(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1.000001")
	require.NoError(t, err)
	b := alloraMath.NewDecFinite(1, 6)
	c, err := a.MulExact(b)
	require.NoError(t, err)
	d, err := c.Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1000001), d)
}

func TestMulExactBad(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1.000000000000000000000000000000000000123456789")
	require.NoError(t, err)
	b := alloraMath.NewDecFinite(1, 10)
	_, err = a.MulExact(b)
	require.ErrorIs(t, err, alloraMath.ErrUnexpectedRounding)
}

func TestQuoExactGood(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1000001")
	require.NoError(t, err)
	b := alloraMath.NewDecFinite(1, 6)
	c, err := a.QuoExact(b)
	require.NoError(t, err)
	t.Logf("%s\n", c.String())
	require.Equal(t, "1.000001000000000000000000000000000", c.String())
}

func TestQuoExactBad(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1000000000000000000000000000000000000123456789")
	require.NoError(t, err)
	b := alloraMath.NewDecFinite(1, 10)
	_, err = a.QuoExact(b)
	require.ErrorIs(t, err, alloraMath.ErrUnexpectedRounding)
}

func TestToBigInt(t *testing.T) {
	i1 := "1000000000000000000000000000000000000123456789"
	tcs := []struct {
		intStr  string
		out     string
		isError error
	}{
		{i1, i1, nil},
		{"1000000000000000000000000000000000000123456789.00000000", i1, nil},
		{"123.456e6", "123456000", nil},
		{"12345.6", "", alloraMath.ErrNonIntegral},
	}
	for idx, tc := range tcs {
		a, err := alloraMath.NewDecFromString(tc.intStr)
		require.NoError(t, err)
		b, err := a.BigInt()
		if tc.isError == nil {
			require.NoError(t, err, "test_%d", idx)
			require.Equal(t, tc.out, b.String(), "test_%d", idx)
		} else {
			require.ErrorIs(t, err, tc.isError, "test_%d", idx)
		}
	}
}

func TestToSdkInt(t *testing.T) {
	i1 := "1000000000000000000000000000000000000123456789"
	tcs := []struct {
		intStr string
		out    string
	}{
		{i1, i1},
		{"1000000000000000000000000000000000000123456789.00000000", i1},
		{"123.456e6", "123456000"},
		{"123.456e1", "1234"},
		{"123.456", "123"},
		{"123.956", "123"},
		{"-123.456", "-123"},
		{"-123.956", "-123"},
		{"-0.956", "0"},
		{"-0.9", "0"},
	}
	for idx, tc := range tcs {
		a, err := alloraMath.NewDecFromString(tc.intStr)
		require.NoError(t, err)
		b, err := a.SdkIntTrim()
		require.NoError(t, err)
		require.Equal(t, tc.out, b.String(), "test_%d", idx)
	}
}

func TestToSdkIntFail(t *testing.T) {
	// 2 **256 - 1 (~1.15e77) should be the largest
	// representable integer in the cosmos SDK Int
	a, err := alloraMath.NewDecFromString("1.2e77")
	require.NoError(t, err)
	b, err := a.SdkIntTrim()
	require.Error(t, err)
	require.Equal(t, cosmosMath.Int{}, b)
}

func TestToSdkLegacyDec(t *testing.T) {
	i1 := "1000000000000000000000000000000000000123456789.321000000000000000"
	tcs := []struct {
		intStr string
		out    string
	}{
		{i1, i1},
		{"1000000000000000000000000000000000000123456789.321000000000000000", i1},
		{"123.456e6", "123456000.000000000000000000"},
		{"123.456e1", "1234.560000000000000000"},
		{"123.456", "123.456000000000000000"},
		{"123.956", "123.956000000000000000"},
		{"-123.456", "-123.456000000000000000"},
		{"-123.956", "-123.956000000000000000"},
		{"-0.956", "-0.956000000000000000"},
		{"-0.9", "-0.900000000000000000"},
	}
	for idx, tc := range tcs {
		a, err := alloraMath.NewDecFromString(tc.intStr)
		require.NoError(t, err)
		b, err := a.SdkLegacyDec()
		require.NoError(t, err)
		require.Equal(t, tc.out, b.String(), "test_%d", idx)
	}
}

func TestToSdkLegacyDecFail(t *testing.T) {
	a, err := alloraMath.NewDecFromString("1.2e77")
	require.NoError(t, err)
	_, err = a.SdkLegacyDec()
	require.Error(t, err)
}

func TestInfDecString(t *testing.T) {
	_, err := alloraMath.NewDecFromString("iNf")
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrInfiniteString)
}

func TestInDelta(t *testing.T) {
	// Test cases
	testCases := []struct {
		expected       alloraMath.Dec
		result         alloraMath.Dec
		epsilon        alloraMath.Dec
		expectedResult bool
	}{
		{alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(1), true},   // expected == result, within epsilon
		{alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(9), alloraMath.NewDecFromInt64(1), true},    // expected != result, but within epsilon
		{alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(8), alloraMath.NewDecFromInt64(1), false},   // expected != result, outside epsilon
		{alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(0), true},   // expected == result, epsilon is zero
		{alloraMath.NewDecFromInt64(10), alloraMath.NewDecFromInt64(-10), alloraMath.NewDecFromInt64(1), false}, // expected != result, outside epsilon
		{alloraMath.NewDecFromInt64(-10), alloraMath.NewDecFromInt64(-10), alloraMath.NewDecFromInt64(0), true}, // expected == result, epsilon is zero
	}

	// Run test cases
	for _, tc := range testCases {
		result, err := alloraMath.InDelta(tc.expected, tc.result, tc.epsilon)
		require.NoError(t, err)
		require.Equal(t, tc.expectedResult, result)
	}
}

func TestSlicesInDelta(t *testing.T) {
	// Test cases
	testCases := []struct {
		name      string
		a         []alloraMath.Dec
		b         []alloraMath.Dec
		epsilon   alloraMath.Dec
		expected  bool
		expectErr bool
	}{
		{
			name:      "Equal slices within epsilon",
			a:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)},
			b:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)},
			epsilon:   alloraMath.NewDecFromInt64(0),
			expected:  true,
			expectErr: false,
		},
		{
			name:      "Equal slices within epsilon",
			a:         []alloraMath.Dec{alloraMath.NewDecFromInt64(0), alloraMath.NewDecFromInt64(-1), alloraMath.NewDecFromInt64(4)},
			b:         []alloraMath.Dec{alloraMath.NewDecFromInt64(-1), alloraMath.NewDecFromInt64(-2), alloraMath.NewDecFromInt64(3)},
			epsilon:   alloraMath.NewDecFromInt64(1),
			expected:  true,
			expectErr: false,
		},
		{
			name:      "Equal slices NOT within epsilon",
			a:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(5)},
			b:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)},
			epsilon:   alloraMath.NewDecFromInt64(1),
			expected:  false,
			expectErr: false,
		},
		{
			name:      "Different slices within epsilon",
			a:         []alloraMath.Dec{alloraMath.NewDecFromInt64(-1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)},
			b:         []alloraMath.Dec{alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(5), alloraMath.NewDecFromInt64(6)},
			epsilon:   alloraMath.NewDecFromInt64(3),
			expected:  true,
			expectErr: false,
		},
		{
			name:      "Different slices outside epsilon",
			a:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)},
			b:         []alloraMath.Dec{alloraMath.NewDecFromInt64(4), alloraMath.NewDecFromInt64(5), alloraMath.NewDecFromInt64(6)},
			epsilon:   alloraMath.NewDecFromInt64(1),
			expected:  false,
			expectErr: false,
		},
		{
			name:      "Different slice lengths",
			a:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)},
			b:         []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2)},
			epsilon:   alloraMath.NewDecFromInt64(0),
			expected:  false,
			expectErr: true,
		},
		{
			name:      "Empty slice",
			a:         []alloraMath.Dec{},
			b:         []alloraMath.Dec{},
			epsilon:   alloraMath.NewDecFromInt64(0),
			expected:  true,
			expectErr: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := alloraMath.SlicesInDelta(tc.a, tc.b, tc.epsilon)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestSumDecSlice(t *testing.T) {
	// Test case 1: Empty slice
	x := []alloraMath.Dec{}
	expectedSum := alloraMath.ZeroDec()

	sum, err := alloraMath.SumDecSlice(x)
	require.NoError(t, err)
	require.True(t, sum.Equal(expectedSum), "Expected sum to be zero")

	// Test case 2: Slice with positive values
	x = []alloraMath.Dec{
		alloraMath.NewDecFromInt64(1),
		alloraMath.NewDecFromInt64(2),
		alloraMath.NewDecFromInt64(3),
	}
	expectedSum = alloraMath.NewDecFromInt64(6)

	sum, err = alloraMath.SumDecSlice(x)
	require.NoError(t, err)
	require.True(t, sum.Equal(expectedSum), "Expected sum to be 6")

	// Test case 3: Slice with negative values
	x = []alloraMath.Dec{
		alloraMath.NewDecFromInt64(-1),
		alloraMath.NewDecFromInt64(-2),
		alloraMath.NewDecFromInt64(-3),
	}
	expectedSum = alloraMath.NewDecFromInt64(-6)

	sum, err = alloraMath.SumDecSlice(x)
	require.NoError(t, err)
	require.True(t, sum.Equal(expectedSum), "Expected sum to be -6")

	// Test case 4: Slice with mixed positive and negative values
	x = []alloraMath.Dec{
		alloraMath.NewDecFromInt64(1),
		alloraMath.NewDecFromInt64(-2),
		alloraMath.NewDecFromInt64(3),
	}
	expectedSum = alloraMath.NewDecFromInt64(2)

	sum, err = alloraMath.SumDecSlice(x)
	require.NoError(t, err)
	require.True(t, sum.Equal(expectedSum), "Expected sum to be 2")
}

func TestNaNCreation(t *testing.T) {
	nan := alloraMath.NewNaN()
	require.True(t, nan.IsNaN())
	require.Equal(t, "NaN", nan.String())
}

func TestNewDecFromStringEmptyString(t *testing.T) {
	emptyString, err := alloraMath.NewDecFromString("")
	require.NoError(t, err)
	require.Equal(t, alloraMath.ZeroDec(), emptyString)
}

// only covers happy paths
func TestNewDecFromUint64(t *testing.T) {
	aDec, err := alloraMath.NewDecFromUint64(uint64(0))
	require.NoError(t, err)
	require.True(
		t,
		alloraMath.ZeroDec().Equal(aDec),
		"%s != %s",
		alloraMath.ZeroDec().String(),
		aDec.String(),
	)

	aDec, err = alloraMath.NewDecFromUint64(uint64(1))
	require.NoError(t, err)
	require.True(
		t, alloraMath.OneDec().Equal(aDec),
		"%s != %s",
		alloraMath.OneDec().String(),
		aDec.String(),
	)

	aDec, err = alloraMath.NewDecFromUint64(uint64(1337))
	require.NoError(t, err)
	require.True(
		t,
		alloraMath.MustNewDecFromString("1337").Equal(aDec),
		"%s != %s",
		alloraMath.MustNewDecFromString("1337").String(),
		aDec.String(),
	)

	maxUint := uint64(goMath.MaxUint64)
	aDec, err = alloraMath.NewDecFromUint64(maxUint)
	require.NoError(t, err)
	maxUintString := strconv.FormatUint(maxUint, 10)
	require.Equal(t, maxUintString, aDec.String())
}

// only covers happy path for now
func TestNewDecFromSdkInt(t *testing.T) {
	anInt := cosmosMath.NewInt(1337)
	aDec, err := alloraMath.NewDecFromSdkInt(anInt)
	require.NoError(t, err)
	require.True(t,
		aDec.Equal(alloraMath.MustNewDecFromString("1337")),
		"%s != %s",
		aDec.String(),
		alloraMath.MustNewDecFromString("1337").String(),
	)
}

// only covers happy path for now
func TestNewDecFromSdkLegacyDec(t *testing.T) {
	aLegacyDec := cosmosMath.LegacyMustNewDecFromStr("1337.123456789")
	aDec, err := alloraMath.NewDecFromSdkLegacyDec(aLegacyDec)
	require.NoError(t, err)
	aDecFromString, err := alloraMath.NewDecFromString("1337.123456789")
	require.True(t,
		aDec.Equal(aDecFromString),
		"%s != %s",
		aDec.String(),
		aDecFromString.String(),
	)
}

func TestAddFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.Add(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestSubFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.Sub(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestMulFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.Mul(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestQuoFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.Quo(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestMulExactFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.MulExact(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestQuoExactFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.QuoExact(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestQuoIntegerFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.QuoInteger(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestRemFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := dec.Rem(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestNegFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Neg()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestLog10FailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Log10(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestLnFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Ln(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestExpFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Exp(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestExp10FailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Exp10(nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestPowFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Pow(dec, nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestMaxFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Max(dec, nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestMinFailNaN(t *testing.T) {
	dec := alloraMath.OneDec()
	nan := alloraMath.NewNaN()
	_, err := alloraMath.Min(dec, nan)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestSqrtFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Sqrt()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestAbsFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Abs()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestCeilFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Ceil()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestFloorFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Floor()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestInt64FailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Int64()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestUInt64FailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.UInt64()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestBigIntFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.BigInt()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestCoeffFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.Coeff()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestSdkIntTrimFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.SdkIntTrim()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestSdkLegacyDecFailNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	_, err := nan.SdkLegacyDec()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestNeg(t *testing.T) {
	number := alloraMath.MustNewDecFromString("123")
	negated, err := number.Neg()
	require.NoError(t, err)
	require.Equal(t, "-123", negated.String())

	negatedNumber := alloraMath.MustNewDecFromString("-123")
	negated, err = negatedNumber.Neg()
	require.NoError(t, err)
	require.Equal(t, "123", negated.String())

	aDecimal := alloraMath.MustNewDecFromString("123.5")
	negated, err = aDecimal.Neg()
	require.NoError(t, err)
	require.Equal(t, "-123.5", negated.String())

	aNegativeDecimal := alloraMath.MustNewDecFromString("-123.5")
	negated, err = aNegativeDecimal.Neg()
	require.NoError(t, err)
	require.Equal(t, "123.5", negated.String())
}

func TestMax(t *testing.T) {
	max, err := alloraMath.Max(alloraMath.MustNewDecFromString("123"), alloraMath.MustNewDecFromString("124"))
	require.NoError(t, err)
	require.Equal(t, "124", max.String())

	max, err = alloraMath.Max(alloraMath.MustNewDecFromString("-123"), alloraMath.MustNewDecFromString("-124"))
	require.NoError(t, err)
	require.Equal(t, "-123", max.String())
}

func TestMin(t *testing.T) {
	min, err := alloraMath.Min(alloraMath.MustNewDecFromString("123"), alloraMath.MustNewDecFromString("124"))
	require.NoError(t, err)
	require.Equal(t, "123", min.String())

	min, err = alloraMath.Min(alloraMath.MustNewDecFromString("-123"), alloraMath.MustNewDecFromString("-124"))
	require.NoError(t, err)
	require.Equal(t, "-124", min.String())
}

func TestUint64(t *testing.T) {
	num, err := alloraMath.NewDecFromUint64(123)
	require.NoError(t, err)
	result, err := num.UInt64()
	require.NoError(t, err)
	require.Equal(t, uint64(123), result)

	num, err = alloraMath.NewDecFromUint64(goMath.MaxUint64)
	require.NoError(t, err)
	result, err = num.UInt64()
	require.NoError(t, err)
	require.Equal(t, uint64(goMath.MaxUint64), result)
}

func TestUint64FailOverflow(t *testing.T) {
	num, err := alloraMath.NewDecFromUint64(goMath.MaxUint64)
	require.NoError(t, err)
	num, err = num.Add(alloraMath.OneDec())
	require.NoError(t, err)
	_, err = num.UInt64()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrOverflow)
}

func TestUint64FailNegative(t *testing.T) {
	num := alloraMath.NewDecFromInt64(-1)
	_, err := num.UInt64()
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrOverflow)
}

func TestMarshal(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("123.456")
	bytes, err := dec.Marshal()
	require.NoError(t, err)
	require.Equal(t, []byte("123.456"), bytes)
}

func TestMarshalTo(t *testing.T) {
	testNum := "123.456"
	dec := alloraMath.MustNewDecFromString(testNum)
	buf := make([]byte, 10)
	n, err := dec.MarshalTo(buf)
	require.NoError(t, err)
	require.NotZero(t, n)
	require.Len(t, testNum, n)
	require.Equal(t, []byte(testNum), buf[:n])
}

func TestSize(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("123.456")
	require.Equal(t, 7, dec.Size())
}

func TestMarshalNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	bytes, err := nan.Marshal()
	require.NoError(t, err)
	require.Equal(t, []byte("NaN"), bytes)
}

func TestUnmarshalNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	bytes, err := nan.Marshal()
	require.NoError(t, err)
	require.Equal(t, []byte("NaN"), bytes)
}

func TestUnmarshal(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("123.456")
	bytes, err := dec.Marshal()
	require.NoError(t, err)

	unmarshaled := alloraMath.Dec{}
	require.NoError(t, unmarshaled.Unmarshal(bytes))
	require.Equal(t, dec, unmarshaled)
}

func TestMarshalJSON(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("123.456")
	json, err := dec.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "\"123.456\"", string(json))
}

func TestUnmarshalJSON(t *testing.T) {
	dec := alloraMath.MustNewDecFromString("123.456")
	json, err := dec.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "\"123.456\"", string(json))

	unmarshaled := alloraMath.Dec{}
	require.NoError(t, unmarshaled.UnmarshalJSON(json))
	require.Equal(t, dec, unmarshaled)
}

func TestMarshalJSONNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	json, err := nan.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "\"NaN\"", string(json))
}

func TestUnmarshalJSONNaN(t *testing.T) {
	nan := alloraMath.NewNaN()
	json, err := nan.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "\"NaN\"", string(json))

	unmarshaled := alloraMath.Dec{}
	require.NoError(t, unmarshaled.UnmarshalJSON(json))
	require.Equal(t, nan, unmarshaled)
}
