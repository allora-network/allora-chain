package math

import (
	"cmp"
	"slices"
	"sort"

	errorsmod "cosmossdk.io/errors"
)

// all exponential moving average functions take the form
// x_average=α*x_current + (1-α)*x_previous
//
// this covers the equations
// Uij = αUij + (1 − α)Ui−1,j
// ̃Vik = αVik + (1 − α)Vi−1,k
// ̃Wim = αWim + (1 − α)Wi−1,m
func CalcEma(
	alpha,
	current,
	previous Dec,
	firstTime bool,
) (Dec, error) {
	// If first iteration, then return just the new value
	if current.isNaN {
		return ZeroDec(), errorsmod.Wrap(ErrNaN, "CalcEma current EMA operand should not be NaN")
	}
	if firstTime || current.Equal(previous) {
		return current, nil
	}
	if previous.isNaN {
		return ZeroDec(), errorsmod.Wrap(ErrNaN, "CalcEma previous EMA operand should not be NaN")
	}
	if alpha.isNaN {
		return ZeroDec(), errorsmod.Wrap(ErrNaN, "CalcEma alpha EMA operand should not be NaN")
	}
	alphaCurrent, err := alpha.Mul(current)
	if err != nil {
		return ZeroDec(), err
	}
	oneMinusAlpha, err := OneDec().Sub(alpha)
	if err != nil {
		return ZeroDec(), err
	}
	oneMinusAlphaTimesPrev, err := oneMinusAlpha.Mul(previous)
	if err != nil {
		return ZeroDec(), err
	}
	ret, err := alphaCurrent.Add(oneMinusAlphaTimesPrev)
	if err != nil {
		return ZeroDec(), err
	}
	return ret, nil
}

// Generic function that sorts the keys of a map
// Used for deterministic ranging of maps
func GetSortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// Generic function that sorts the keys of a map.
// Used for deterministic ranging of arrays with weights in a map
// the keys are sorted by the value they map to, in descending order
// whose keys may not include some values in the array.
// When an array element is not in the map, it is not included in the output array.
func GetSortedElementsByDecWeightDesc[K cmp.Ordered](m map[K]*Dec) []K {
	// Create a new array that only contains unique elements that are in the map
	newL := make([]K, 0)
	for id := range m {
		newL = append(newL, id)
	}

	sort.Slice(newL, func(i, j int) bool {
		if (*m[newL[i]]).Equal(*m[newL[j]]) {
			return newL[i] < newL[j]
		}
		return (*m[newL[i]]).Gt(*m[newL[j]])
	})
	return newL
}

// StdDev calculates the standard deviation of a slice of `Dec`
// stdDev = sqrt((Σ(x - μ))^2/ N)
// where μ is mean and N is number of elements
func StdDev(data []Dec) (Dec, error) {
	if len(data) == 1 {
		return ZeroDec(), nil
	}

	mean := ZeroDec()
	var count int64 // To count valid numbers
	var err error

	// Calculate the mean, excluding NaN values
	for _, v := range data {
		if v.isNaN { // Check if the value is NaN
			return Dec{}, errorsmod.Wrap(ErrNaN, "stddev input data contains NaN values")
		}
		mean, err = mean.Add(v)
		if err != nil {
			return Dec{}, err
		}
		count++ // Increment count of valid numbers
	}

	if count == 0 {
		return ZeroDec(), nil // Return ZeroDec if all values were NaN
	}
	lenData := NewDecFromInt64(int64(len(data)))
	mean, err = mean.Quo(lenData)
	if err != nil {
		return Dec{}, err
	}
	sd := ZeroDec()
	for _, v := range data {
		vMinusMean, err := v.Sub(mean)
		if err != nil {
			return Dec{}, err
		}
		vMinusMeanSquared, err := vMinusMean.Mul(vMinusMean)
		if err != nil {
			return Dec{}, err
		}
		sd, err = sd.Add(vMinusMeanSquared)
		if err != nil {
			return Dec{}, err
		}
	}

	// Apply Bessel's correction by dividing by (N - 1) instead of N
	lenDataMinusOne, err := lenData.Sub(OneDec())
	if err != nil {
		return Dec{}, err
	}

	sdOverLen, err := sd.Quo(lenDataMinusOne)
	if err != nil {
		return Dec{}, err
	}

	sqrtSdOverLen, err := sdOverLen.Sqrt()
	if err != nil {
		return Dec{}, err
	}
	return sqrtSdOverLen, nil
}

// Median calculates the median of a slice of `Dec`
func Median(data []Dec) (Dec, error) {
	for _, v := range data {
		if v.isNaN {
			return Dec{}, errorsmod.Wrap(ErrNaN, "median input data contains NaN values")
		}
	}

	n := len(data)
	if n == 0 {
		return ZeroDec(), nil
	}

	// Sort the data
	sort.Slice(data, func(i, j int) bool {
		return data[i].Lt(data[j])
	})

	if n%2 == 1 {
		// Odd number of elements, return the middle one
		return data[n/2], nil
	}

	// Even number of elements, return the average of the two middle ones
	mid1 := data[n/2-1]
	mid2 := data[n/2]
	sum, err := mid1.Add(mid2)
	if err != nil {
		return ZeroDec(), err
	}
	return sum.Quo(NewDecFromInt64(2))
}

// Implements the new gradient function phi prime
// φ'_p(x) = p / (exp(p * (c - x)) + 1)
func Gradient(p, c, x Dec) (Dec, error) {
	if p.isNaN || c.isNaN || x.isNaN {
		return Dec{}, errorsmod.Wrap(ErrNaN, "gradient input values must not be NaN")
	}

	// Calculate c - x
	cMinusX, err := c.Sub(x)
	if err != nil {
		return Dec{}, err
	}

	// Calculate p * (c - x)
	pTimesCMinusX, err := p.Mul(cMinusX)
	if err != nil {
		return Dec{}, err
	}

	// Calculate exp(p * (c - x))
	eToThePtimesCMinusX, err := Exp(pTimesCMinusX)
	if err != nil {
		return Dec{}, err
	}

	// Calculate exp(p * (c - x)) + 1
	onePlusEToThePtimesCMinusX, err := OneDec().Add(eToThePtimesCMinusX)
	if err != nil {
		return Dec{}, err
	}

	// Calculate p / (exp(p * (c - x)) + 1)
	ret, err := p.Quo(onePlusEToThePtimesCMinusX)
	if err != nil {
		return Dec{}, err
	}

	return ret, nil
}

// Implements the potential function phi for the module
// ϕ_p(x) = ln(1 + e^(p * (x - c)))
func Phi(p, c, x Dec) (Dec, error) {
	if p.isNaN || c.isNaN || x.isNaN {
		return Dec{}, errorsmod.Wrap(ErrNaN, "phi input values must not be NaN")
	}
	// Calculate p * (x - c)
	xMinusC, err := x.Sub(c)
	if err != nil {
		return Dec{}, err
	}
	pTimesXMinusC, err := p.Mul(xMinusC)
	if err != nil {
		return Dec{}, err
	}

	// Calculate e^(p * (x - c))
	eToThePtimesXminusC, err := Exp(pTimesXMinusC)
	if err != nil {
		return Dec{}, err
	}

	// Calculate 1 + e^(p * (x - c))
	onePlusEToThePtimesXminusC, err := OneDec().Add(eToThePtimesXminusC)
	if err != nil {
		return Dec{}, err
	}

	// Calculate ln(1 + e^(p * (x - c)))
	result, err := Ln(onePlusEToThePtimesXminusC)
	if err != nil {
		return Dec{}, err
	}

	return result, nil
}

// CumulativeSum calculates the cumulative sum of an array of Dec values.
func CumulativeSum(arr []Dec) ([]Dec, error) {
	for _, val := range arr {
		if val.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "cumulative sum input array contains NaN values")
		}
	}
	result := make([]Dec, len(arr))
	sum := ZeroDec()

	for i, val := range arr {
		var err error
		sum, err = sum.Add(val)
		if err != nil {
			return nil, err
		}
		result[i] = sum
	}
	return result, nil
}

// LinearInterpolation performs linear interpolation on Dec values.
func LinearInterpolation(x, xp, fp []Dec) ([]Dec, error) {
	for _, xi := range x {
		if xi.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "linear interpolation input x contains NaN values")
		}
	}
	for _, xpi := range xp {
		if xpi.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "linear interpolation input xp contains NaN values")
		}
	}
	for _, fpi := range fp {
		if fpi.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "linear interpolation input fp contains NaN values")
		}
	}
	if len(xp) != len(fp) {
		return nil, errorsmod.Wrap(ErrNotMatchingLength, "linear interpolation input xp and fp must have the same length")
	}
	result := make([]Dec, len(x))
	for i, xi := range x {
		if xi.Lte(xp[0]) {
			result[i] = fp[0]
		} else if xi.Gte(xp[len(xp)-1]) {
			result[i] = fp[len(fp)-1]
		} else {
			j := sort.Search(len(xp)-1, func(j int) bool { return xi.Lt(xp[j+1]) })
			denominator, err := xp[j+1].Sub(xp[j])
			if err != nil {
				return nil, err
			}
			numerator, err := xi.Sub(xp[j])
			if err != nil {
				return nil, err
			}
			t, err := numerator.Quo(denominator)
			if err != nil {
				return nil, err
			}
			oneMinusT, err := OneDec().Sub(t)
			if err != nil {
				return nil, err
			}
			fpjMulOneMinusT, err := fp[j].Mul(oneMinusT)
			if err != nil {
				return nil, err
			}
			fpjPlusOneMulT, err := fp[j+1].Mul(t)
			if err != nil {
				return nil, err
			}
			result[i], err = fpjMulOneMinusT.Add(fpjPlusOneMulT)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// WeightedPercentile calculates weighted percentiles of Dec values.
func WeightedPercentile(data, weights, percentiles []Dec) ([]Dec, error) {
	for _, d := range data {
		if d.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "weighted percentile input data contains NaN values")
		}
	}
	for _, w := range weights {
		if w.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "weighted percentile input weights contain NaN values")
		}
	}
	for _, p := range percentiles {
		if p.isNaN {
			return nil, errorsmod.Wrap(ErrNaN, "weighted percentile input percentiles contain NaN values")
		}
	}
	if len(weights) != len(data) {
		return nil, errorsmod.Wrap(ErrNotMatchingLength, "weighted percentile input data and weights must have the same length")
	}
	hundred := MustNewDecFromString("100")
	zero := ZeroDec()
	for _, p := range percentiles {
		if p.Gt(hundred) || p.Lt(zero) {
			return nil, errorsmod.Wrap(ErrOutOfRange, "percentile must have a value between 0 and 100")
		}
	}

	// Sort data and weights
	type pair struct {
		value  Dec
		weight Dec
	}
	pairs := make([]pair, len(data))
	for i := range data {
		pairs[i] = pair{data[i], weights[i]}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].value.Lt(pairs[j].value)
	})

	sortedData := make([]Dec, len(data))
	sortedWeights := make([]Dec, len(data))
	for i := range pairs {
		sortedData[i] = pairs[i].value
		sortedWeights[i] = pairs[i].weight
	}

	// Compute the cumulative sum of weights and normalize by the last value
	csw, err := CumulativeSum(sortedWeights)
	if err != nil {
		return nil, err
	}
	normalizedWeights := make([]Dec, len(csw))
	for i, value := range csw {
		halfWeight, err := sortedWeights[i].Quo(NewDecFromInt64(2))
		if err != nil {
			return nil, err
		}
		valueSubHalfWeight, err := value.Sub(halfWeight)
		if err != nil {
			return nil, err
		}
		normalizedWeights[i], err = valueSubHalfWeight.Quo(csw[len(csw)-1])
		if err != nil {
			return nil, err
		}
	}

	// Interpolate to compute the percentiles
	quantiles := make([]Dec, len(percentiles))
	for i, p := range percentiles {
		quantiles[i], err = p.Quo(hundred)
		if err != nil {
			return nil, err
		}
	}
	result, err := LinearInterpolation(quantiles, normalizedWeights, sortedData)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// helper function for test suites that want to check
// if some math is within a delta
func InDelta(expected, result Dec, epsilon Dec) (bool, error) {
	if expected.IsNaN() || result.IsNaN() {
		return false, errorsmod.Wrap(ErrNaN, "cannot compare NaN")
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
		return false, errorsmod.Wrap(ErrNotMatchingLength, "cannot check if slices are within delta")
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

// Get the quantile of decs array
func GetQuantileOfDecs(
	decs []Dec,
	quantile Dec,
) (Dec, error) {
	// If there are no decs then the quantile of scores is 0.
	// This better ensures chain continuity without consequence because in this situation
	// there is no meaningful quantile to calculate.
	if len(decs) == 0 {
		return ZeroDec(), nil
	}

	// Sort decs in descending order. Address is used to break ties.
	slices.SortStableFunc(decs, func(x, y Dec) int {
		if x.Lt(y) {
			return 1
		}
		return -1
	})

	// n elements, q quantile
	// position = (1 - q) * (n - 1)
	nLessOne, err := NewDecFromUint64(uint64(len(decs) - 1))
	if err != nil {
		return Dec{}, err
	}
	oneLessQ, err := OneDec().Sub(quantile)
	if err != nil {
		return Dec{}, err
	}
	position, err := oneLessQ.Mul(nLessOne)
	if err != nil {
		return Dec{}, err
	}

	lowerIndex, err := position.Floor()
	if err != nil {
		return Dec{}, err
	}
	lowerIndexInt, err := lowerIndex.Int64()
	if err != nil {
		return Dec{}, err
	}
	upperIndex, err := position.Ceil()
	if err != nil {
		return Dec{}, err
	}
	upperIndexInt, err := upperIndex.Int64()
	if err != nil {
		return Dec{}, err
	}

	if lowerIndex == upperIndex {
		return decs[lowerIndexInt], nil
	}

	// in cases where the quantile is between two values
	// return lowerValue + (upperValue-lowerValue)*(position-lowerIndex)
	lowerDec := decs[lowerIndexInt]
	upperDec := decs[upperIndexInt]
	positionMinusLowerIndex, err := position.Sub(lowerIndex)
	if err != nil {
		return Dec{}, err
	}
	upperMinusLower, err := upperDec.Sub(lowerDec)
	if err != nil {
		return Dec{}, err
	}
	product, err := positionMinusLowerIndex.Mul(upperMinusLower)
	if err != nil {
		return Dec{}, err
	}
	ret, err := lowerDec.Add(product)
	if err != nil {
		return Dec{}, err
	}
	return ret, nil
}
