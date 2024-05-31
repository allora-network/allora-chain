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
