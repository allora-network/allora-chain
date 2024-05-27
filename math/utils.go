package math

import (
	"cmp"
	"sort"

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

// generic function that sorts the keys of a map
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
