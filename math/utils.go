package math

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
