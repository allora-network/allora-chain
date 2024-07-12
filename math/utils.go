package math

import (
	"cmp"
	"sort"

	"errors"
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
	if firstTime || current.Equal(previous) {
		return current, nil
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

func CalcExpDecay(
	currentRev,
	decayFactor Dec,
) (Dec, error) {
	oneMinusDecayFactor, err := OneDec().Sub(decayFactor)
	if err != nil {
		return ZeroDec(), err
	}
	newRev, err := oneMinusDecayFactor.Mul(currentRev)
	if err != nil {
		return ZeroDec(), err
	}
	return newRev, nil
}

// Generic function that sorts the keys of a map
// Used for deterministic ranging of maps
func GetSortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// Generic function that sorts the keys of a map.
// Used for deterministic ranging of arrays with weights in a map
// whose keys may not include some values in the array.
// When an array element is not in the map, it is not included in the output array.
func GetSortedElementsByDecWeightDesc[K cmp.Ordered](l []K, m map[K]*Dec) []K {
	// Create a new array that only contains unique elements that are in the map
	newL := make([]K, 0)
	hasKeyBeenSeen := make(map[K]bool)
	for _, el := range l {
		if _, ok := m[el]; ok {
			if _, ok := hasKeyBeenSeen[el]; !ok {
				newL = append(newL, el)
				hasKeyBeenSeen[el] = true
			}
		}
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
	mean := ZeroDec()
	var err error = nil
	for _, v := range data {
		mean, err = mean.Add(v)
		if err != nil {
			return Dec{}, err
		}
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
	sdOverLen, err := sd.Quo(lenData)
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
	if len(xp) != len(fp) {
		return nil, errors.New("xp and fp must have the same length")
	}

	result := make([]Dec, len(x))
	for i, xi := range x {
		if xi.Lte(xp[0]) {
			result[i] = fp[0]
		} else if xi.Gte(xp[len(xp)-1]) {
			result[i] = fp[len(fp)-1]
		} else {
			// Find the interval xp[i] <= xi < xp[i + 1]
			j := 0
			for xi.Gte(xp[j+1]) {
				j++
			}
			// Linear interpolation formula
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
			fp_j_t, err := fpjMulOneMinusT.Add(fpjPlusOneMulT)
			if err != nil {
				return nil, err
			}
			result[i] = fp_j_t
		}
	}
	return result, nil
}

// WeightedPercentile calculates weighted percentiles of Dec values.
func WeightedPercentile(data, weights, percentiles []Dec) ([]Dec, error) {
	if len(weights) != len(data) {
		return nil, errors.New("the length of data and weights must be the same")
	}
	hundred := MustNewDecFromString("100")
	zero := ZeroDec()
	for _, p := range percentiles {
		if p.Gt(hundred) || p.Lt(zero) {
			return nil, errors.New("percentile must have a value between 0 and 100")
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
