package inference_synthesis

import (
	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (p *SynthPalette) calcWeightFromNormalizedRegret(
	normalizedRegret alloraMath.Dec,
	maxNormalizedRegret alloraMath.Dec,
) (alloraMath.Dec, error) {
	// upper bound: c + 6.75 / p
	v6Point75OverP, err := alloraMath.MustNewDecFromString("6.75").Quo(p.pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating upper bound for regret normalization")
	}
	cPlus6Point75OverP, err := p.cNorm.Add(v6Point75OverP)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating upper bound for regret normalization")
	}

	// Cap the normalized regrets at an upper value
	// regretFrac = min(regretFrac, c + 6.75 / p)
	if normalizedRegret.Gt(cPlus6Point75OverP) {
		normalizedRegret = cPlus6Point75OverP
	}

	// lower bound: c - 8.25 / p
	v8Point25OverP, err := alloraMath.MustNewDecFromString("8.25").Quo(p.pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower bound for regret normalization")
	}
	cMinus8Point25OverP, err := p.cNorm.Sub(v8Point25OverP)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower bound for regret normalization")
	}

	// if max(regretFrac) < c - 8.25 / p, then regretFrac = regretFrac - max(regretFrac) + (c - 8.25 / p)
	if maxNormalizedRegret.Lt(cMinus8Point25OverP) {
		normalizedRegret, err = normalizedRegret.Sub(maxNormalizedRegret)
		if err != nil {
			return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error anchoring normalized regrets at zero")
		}
		normalizedRegret, err = normalizedRegret.Add(cMinus8Point25OverP)
		if err != nil {
			return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adjusting anchored normalized regrets")
		}
	}

	v17Point25OverP, err := alloraMath.MustNewDecFromString("17.25").Quo(p.pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower bound for regret normalization")
	}
	cMinus17Point25OverP, err := p.cNorm.Sub(v17Point25OverP)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower threshold for zero weight")
	}

	// if regretFrac < c - 17.25 / p, then weight = 0
	if normalizedRegret.Lt(cMinus17Point25OverP) {
		return alloraMath.ZeroDec(), nil
	}

	weight, err := alloraMath.Gradient(p.pNorm, p.cNorm, normalizedRegret) // w_ijk = φ'_p(\hatR_ijk)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "error calculating gradient")
	}

	return weight, nil
}

func (p *SynthPalette) GetInfererRegretsSlice() []alloraMath.Dec {
	regrets := make([]alloraMath.Dec, len(p.inferers))
	for i, worker := range p.inferers {
		regrets[i] = p.infererRegrets[worker].regret
	}
	return regrets
}

func (p *SynthPalette) GetForecasterRegretsSlice() []alloraMath.Dec {
	regrets := make([]alloraMath.Dec, len(p.forecasters))
	for i, worker := range p.forecasters {
		regrets[i] = p.forecasterRegrets[worker].regret
	}
	return regrets
}

// Given the current set of inferers and forecasters in the palette, calculate their
// weights using the current regrets
func (p *SynthPalette) CalcWeightsGivenWorkers() (RegretInformedWeights, error) {
	// Calc std dev of regrets + epsilon
	// σ(R_ijk) + ε
	stdDevRegrets, err := alloraMath.StdDev(append(p.GetInfererRegretsSlice(), p.GetForecasterRegretsSlice()...))
	if err != nil {
		return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating standard deviation of regrets")
	}
	// Add epsilon to standard deviation
	stdDevRegretsPlusEpsilon, err := stdDevRegrets.Abs().Add(p.epsilon)
	if err != nil {
		return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error adding epsilon to standard deviation of regrets")
	}

	// Normalize the regrets and find the max normalized regret among them

	normalizedInfererRegrets := make(map[Worker]Regret)
	maxRegret := alloraMath.ZeroDec()
	for i, worker := range p.inferers {
		regretFrac, err := p.infererRegrets[worker].regret.Quo(stdDevRegretsPlusEpsilon)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
		}
		normalizedInfererRegrets[worker] = regretFrac
		if i == 0 || regretFrac.Gt(maxRegret) {
			maxRegret = regretFrac
		}
	}

	normalizedForecasterRegrets := make(map[Worker]Regret)
	for i, worker := range p.forecasters {
		regretFrac, err := p.forecasterRegrets[worker].regret.Quo(stdDevRegretsPlusEpsilon)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
		}
		normalizedForecasterRegrets[worker] = regretFrac
		if i == 0 || regretFrac.Gt(maxRegret) {
			maxRegret = regretFrac
		}
	}

	// Calculate the weights from the normalized regrets

	infererWeights := make(map[Worker]Weight)
	for _, worker := range p.inferers {
		infererWeight, err := p.calcWeightFromNormalizedRegret(normalizedInfererRegrets[worker], maxRegret)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating inferer weight")
		}
		infererWeights[worker] = infererWeight
	}

	forecasterWeights := make(map[Worker]Weight)
	for _, worker := range p.forecasters {
		forecasterWeight, err := p.calcWeightFromNormalizedRegret(normalizedForecasterRegrets[worker], maxRegret)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating forecaster weight")
		}
		forecasterWeights[worker] = forecasterWeight
	}

	return RegretInformedWeights{
		inferers:    infererWeights,
		forecasters: forecasterWeights,
	}, nil
}

// Given the current set of inferers and forecasters in the palette, calculate their
// weights using the provided forecasted regrets
func (p SynthPalette) CalcWeightsWithForecastedRegretOverride(
	inferers []Worker,
	forecastedRegrets map[string]StatefulRegret,
) (RegretInformedWeights, error) {
	if inferers != nil && len(inferers) > 0 && forecastedRegrets != nil {
		p.infererRegrets = forecastedRegrets
		p.forecasterRegrets = map[Worker]StatefulRegret{}
	}
	return p.CalcWeightsGivenWorkers()
}

func (p *SynthPalette) weightAccumulator(
	inference *emissions.Inference,
	weight alloraMath.Dec,
	noPriorRegret bool,
	allPeersAreNew bool,
	runningUnnormalizedI_i alloraMath.Dec,
	sumWeights alloraMath.Dec,
) (alloraMath.Dec, alloraMath.Dec, error) {
	err := error(nil)

	//
	// !!!! AHHHHH see below !!!!!
	//
	// If there is no prior regret and there is at least 1 non-new forecaster => skip this forecaster (set weight=0)
	if noPriorRegret && !allPeersAreNew {
		return runningUnnormalizedI_i, sumWeights, nil
	}

	// Avoid needless computation if the weight is 0 or if there is no inference
	if weight.IsNaN() || weight.Equal(alloraMath.ZeroDec()) || inference == nil {
		return runningUnnormalizedI_i, sumWeights, nil
	}

	// If all workers are new, then the weight is 1 for all workers
	// Otherwise, calculate the weight based on the regret of the worker
	if allPeersAreNew {
		// If all workers are new, then the weight is 1 for all workers; take regular average of inferences
		runningUnnormalizedI_i, err = runningUnnormalizedI_i.Add(inference.Value)
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight by worker value")
		}
		sumWeights, err = sumWeights.Add(alloraMath.OneDec())
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight")
		}
	} else {
		weightTimesInference, err := weight.Mul(inference.Value) // numerator of network combined inference calculation
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating weight by worker value")
		}
		runningUnnormalizedI_i, err = runningUnnormalizedI_i.Add(weightTimesInference)
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight by worker value")
		}
		sumWeights, err = sumWeights.Add(weight)
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight")
		}
	}

	return runningUnnormalizedI_i, sumWeights, nil
}

// Calculates network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper:
// I_i = Σ_l w_il I_il / Σ_l w_il
// w_il = φ'_p(\hatR_i-1,l)
// \hatR_i-1,l = R_i-1,l / |max_{l'}(R_i-1,l')|
// given inferences, forecast-implied inferences, and network regrets
func (p *SynthPalette) CalcWeightedInference(weights RegretInformedWeights) (InferenceValue, error) {
	runningUnnormalizedI_i := alloraMath.ZeroDec()
	sumWeights := alloraMath.ZeroDec()
	err := error(nil)

	for _, inferer := range p.inferers {
		runningUnnormalizedI_i, sumWeights, err = p.weightAccumulator(
			p.inferenceByWorker[inferer],
			weights.inferers[inferer],
			p.infererRegrets[inferer].noPriorRegret,
			p.allInferersAreNew,
			runningUnnormalizedI_i,
			sumWeights,
		)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating weight of inferer")
		}
	}

	for _, forecaster := range p.forecasters {
		runningUnnormalizedI_i, sumWeights, err = p.weightAccumulator(
			p.inferenceByWorker[forecaster],
			weights.forecasters[forecaster],
			p.forecasterRegrets[forecaster].noPriorRegret,
			p.allForecastersAreNew,
			runningUnnormalizedI_i,
			sumWeights,
		)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating weight of forecaster")
		}
	}

	// Normalize the running unnormalized network inference to yield output
	if sumWeights.Lt(p.epsilon) {
		sumWeights = p.epsilon
	}
	ret, err := runningUnnormalizedI_i.Quo(sumWeights)
	if err != nil {
		return InferenceValue{}, errorsmod.Wrapf(err, "Error normalizing network inference")
	}
	return ret, nil
}

// func GetRegretsThenMapToWeights(
// 	inferers []Worker,
// 	forecasters []Worker,
// 	epsilon alloraMath.Dec,
// 	pNorm alloraMath.Dec,
// 	cNorm alloraMath.Dec,
// ) (RegretInformedWeights, error) {
// 	// Calc std dev of regrets + epsilon
// 	// σ(R_ijk) + ε
// 	stdDevRegrets, err := alloraMath.StdDev(append(infererRegrets, forecasterRegrets...))
// 	if err != nil {
// 		return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating standard deviation of regrets")
// 	}
// 	// Add epsilon to standard deviation
// 	stdDevRegretsPlusEpsilon, err := stdDevRegrets.Abs().Add(epsilon)
// 	if err != nil {
// 		return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error adding epsilon to standard deviation of regrets")
// 	}

// 	// Normalize the regrets and find the max normalized regret among them

// 	normalizedInfererRegrets := make(map[Worker]Regret)
// 	maxRegret := alloraMath.ZeroDec()
// 	for i, worker := range inferers {
// 		regretFrac, err := infererRegrets[i].Quo(stdDevRegretsPlusEpsilon)
// 		if err != nil {
// 			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
// 		}
// 		normalizedInfererRegrets[worker] = regretFrac
// 		if i == 0 || regretFrac.Gt(maxRegret) {
// 			maxRegret = regretFrac
// 		}
// 	}

// 	normalizedForecasterRegrets := make(map[Worker]Regret)
// 	for i, worker := range forecasters {
// 		regretFrac, err := forecasterRegrets[i].Quo(stdDevRegretsPlusEpsilon)
// 		if err != nil {
// 			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
// 		}
// 		normalizedForecasterRegrets[worker] = regretFrac
// 		if i == 0 || regretFrac.Gt(maxRegret) {
// 			maxRegret = regretFrac
// 		}
// 	}

// 	// Calculate the weights from the normalized regrets

// 	infererWeights := make(map[Worker]Weight)
// 	for _, worker := range inferers {
// 		infererWeight, err := CalcWeightFromNormalizedRegret(
// 			normalizedInfererRegrets[worker],
// 			maxRegret,
// 			pNorm,
// 			cNorm,
// 		)
// 		if err != nil {
// 			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating inferer weight")
// 		}
// 		infererWeights[worker] = infererWeight
// 	}

// 	forecasterWeights := make(map[Worker]Weight)
// 	for _, worker := range forecasters {
// 		forecasterWeight, err := CalcWeightFromNormalizedRegret(
// 			normalizedForecasterRegrets[worker],
// 			maxRegret,
// 			pNorm,
// 			cNorm,
// 		)
// 		if err != nil {
// 			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating forecaster weight")
// 		}
// 		forecasterWeights[worker] = forecasterWeight
// 	}

// 	return RegretInformedWeights{
// 		inferers:    infererWeights,
// 		forecasters: forecasterWeights,
// 	}, nil
// }
