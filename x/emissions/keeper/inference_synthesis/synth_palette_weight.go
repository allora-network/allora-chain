package inference_synthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Given the current set of inferers and forecasters in the palette, calculate their
// weights using the current regrets
func (p *SynthPalette) CalcWeightsGivenWorkers() (RegretInformedWeights, error) {
	var regrets []alloraMath.Dec
	// Just regrets with a prior value associated will be considered
	infererRegrets := p.GetInfererRegretsSlice()
	forecasterRegrets := p.GetForecasterRegretsSlice()

	if len(infererRegrets) > 0 {
		regrets = append(regrets, infererRegrets...)
	}
	if len(forecasterRegrets) > 0 {
		regrets = append(regrets, forecasterRegrets...)
	}
	if len(regrets) == 0 {
		return RegretInformedWeights{}, errorsmod.Wrapf(emissionstypes.ErrEmptyArray, "No regrets to calculate weights")
	}

	// Calc std dev of regrets + epsilon
	// σ(R_ijk) + ε
	stdDevRegrets, err := alloraMath.StdDev(regrets)
	if err != nil {
		return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating standard deviation of regrets")
	}
	// Add epsilon to standard deviation
	stdDevRegretsPlusEpsilon, err := stdDevRegrets.Abs().Add(p.Epsilon)
	if err != nil {
		return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error adding epsilon to standard deviation")
	}

	// Normalize the regrets and find the max normalized regret among them
	normalizedInfererRegrets := make(map[Worker]Regret)
	maxRegret := alloraMath.ZeroDec()
	maxRegretInitialized := false
	for _, worker := range p.Inferers {
		regretInfo, ok := p.InfererRegrets[worker]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find worker in InfererRegrets in CalcWeightsGivenWorkers %v", worker))
			continue
		}
		regretFrac, err := regretInfo.regret.Quo(stdDevRegretsPlusEpsilon)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
		}
		normalizedInfererRegrets[worker] = regretFrac
		if !maxRegretInitialized {
			maxRegretInitialized = true
			maxRegret = regretFrac
		} else if regretFrac.Gt(maxRegret) {
			maxRegret = regretFrac
		}
	}

	normalizedForecasterRegrets := make(map[Worker]Regret)
	if len(p.ForecasterRegrets) > 0 {
		for _, worker := range p.Forecasters {
			regretInfo, ok := p.ForecasterRegrets[worker]
			if !ok {
				p.Logger.Debug(fmt.Sprintf("Cannot find worker in ForecasterRegrets in CalcWeightsGivenWorkers %v", worker))
				continue
			}
			regretFrac, err := regretInfo.regret.Quo(stdDevRegretsPlusEpsilon)
			if err != nil {
				return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
			}
			normalizedForecasterRegrets[worker] = regretFrac
			if !maxRegretInitialized {
				maxRegretInitialized = true
				maxRegret = regretFrac
			} else if regretFrac.Gt(maxRegret) {
				maxRegret = regretFrac
			}
		}
	}

	infererWeights := make(map[Worker]Weight)
	forecasterWeights := make(map[Worker]Weight)
	if p.InferersNewStatus != InferersAllNewExceptOne {
		// Calculate the weights from the normalized regrets
		for _, worker := range p.Inferers {
			regretInfo, ok := p.InfererRegrets[worker]
			if !ok {
				p.Logger.Debug(fmt.Sprintf("Cannot find worker in InfererRegrets in CalcWeightsGivenWorkers %v", worker))
				continue
			}
			// If there is more than one not-new inferer, calculate the weight for the ones that are not new
			var infererWeight = alloraMath.ZeroDec()
			if !regretInfo.noPriorRegret {
				infererWeight, err = CalcWeightFromNormalizedRegret(normalizedInfererRegrets[worker], maxRegret, p.PNorm, p.CNorm)
				if err != nil {
					return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating inferer weight")
				}
			}
			infererWeights[worker] = infererWeight
		}

		if len(p.ForecasterRegrets) > 0 {
			for _, worker := range p.Forecasters {
				regretInfo, ok := p.ForecasterRegrets[worker]
				if !ok {
					p.Logger.Debug(fmt.Sprintf("Cannot find worker in ForecasterRegrets in CalcWeightsGivenWorkers %v", worker))
					continue
				}
				var forecasterWeight = alloraMath.ZeroDec()
				if !regretInfo.noPriorRegret {
					forecasterWeight, err = CalcWeightFromNormalizedRegret(normalizedForecasterRegrets[worker], maxRegret, p.PNorm, p.CNorm)
					if err != nil {
						return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating forecaster weight")
					}
				}
				forecasterWeights[worker] = forecasterWeight
			}
		}
	}

	return RegretInformedWeights{
		inferers:    infererWeights,
		forecasters: forecasterWeights,
	}, nil
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

	// If all inferers are new, then the weight is 1 for all inferers
	if p.InferersNewStatus == InferersAllNew {
		for _, inferer := range p.Inferers {
			runningUnnormalizedI_i, err = runningUnnormalizedI_i.Add(p.InferenceByWorker[inferer].Value)
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error adding weight by worker value")
			}
			sumWeights, err = sumWeights.Add(alloraMath.OneDec())
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error adding weight")
			}
		}

		// If there is only one not-new inferer, then just consider that inferer
	} else if p.InferersNewStatus == InferersAllNewExceptOne {
		singleInferer := p.SingleNotNewInferer

		runningUnnormalizedI_i, err = runningUnnormalizedI_i.Add(p.InferenceByWorker[singleInferer].Value)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error adding weight by worker value")
		}
		sumWeights, err = sumWeights.Add(alloraMath.OneDec())
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error adding weight")
		}
	} else {
		for _, inferer := range p.Inferers {
			if _, ok := p.InferenceByWorker[inferer]; !ok {
				p.Logger.Debug(fmt.Sprintf("Cannot find inferer in InferenceByWorker in CalcWeightedInference %v", inferer))
				continue
			}
			if _, ok := weights.inferers[inferer]; !ok {
				p.Logger.Debug(fmt.Sprintf("Cannot find inferer in weights.inferers in CalcWeightedInference %v", inferer))
				continue
			}
			if _, ok := p.InfererRegrets[inferer]; !ok {
				p.Logger.Debug(fmt.Sprintf("Cannot find inferer in InfererRegrets in CalcWeightedInference %v", inferer))
				continue
			}
			runningUnnormalizedI_i, sumWeights, err = AccumulateWeights(
				p.InferenceByWorker[inferer],
				weights.inferers[inferer],
				p.InfererRegrets[inferer].noPriorRegret,
				p.InferersNewStatus == InferersAllNew,
				runningUnnormalizedI_i,
				sumWeights,
			)
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating weight of inferer")
			}
		}

		// If all inferers are new, forecasters are not considered
		if p.InferersNewStatus != InferersAllNew && p.InferersNewStatus != InferersAllNewExceptOne {
			for _, forecaster := range p.Forecasters {
				if _, ok := p.ForecastImpliedInferenceByWorker[forecaster]; !ok {
					p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecastImpliedInferenceByWorker in CalcWeightedInference %v", forecaster))
					continue
				}
				if _, ok := weights.forecasters[forecaster]; !ok {
					p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in weights.forecasters in CalcWeightedInference %v", forecaster))
					continue
				}
				if _, ok := p.ForecasterRegrets[forecaster]; !ok {
					p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in CalcWeightedInference %v", forecaster))
					continue
				}
				runningUnnormalizedI_i, sumWeights, err = AccumulateWeights(
					p.ForecastImpliedInferenceByWorker[forecaster],
					weights.forecasters[forecaster],
					p.ForecasterRegrets[forecaster].noPriorRegret,
					false,
					runningUnnormalizedI_i,
					sumWeights,
				)
				if err != nil {
					return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating weight of forecaster")
				}
			}
		}
	}

	// Normalize the running unnormalized network inference to yield output
	if sumWeights.Lt(p.Epsilon) {
		sumWeights = p.Epsilon
	}
	ret, err := runningUnnormalizedI_i.Quo(sumWeights)
	if err != nil {
		return InferenceValue{}, errorsmod.Wrapf(err, "Error normalizing network inference")
	}
	return ret, nil
}

func (p *SynthPalette) GetInfererRegretsSlice() []alloraMath.Dec {
	var regrets []alloraMath.Dec
	if len(p.InfererRegrets) == 0 {
		return regrets
	}
	regrets = make([]alloraMath.Dec, 0, len(p.InfererRegrets))
	for _, inferer := range p.Inferers {
		regretInfo, ok := p.InfererRegrets[inferer]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in InfererRegrets in GetInfererRegretsSlice %v", inferer))
			continue
		}
		if !regretInfo.noPriorRegret {
			regrets = append(regrets, regretInfo.regret)
		}
	}
	return regrets
}

func (p *SynthPalette) GetForecasterRegretsSlice() []alloraMath.Dec {
	var regrets []alloraMath.Dec
	if len(p.ForecasterRegrets) == 0 {
		return regrets
	}
	regrets = make([]alloraMath.Dec, 0, len(p.ForecasterRegrets))
	for _, forecaster := range p.Forecasters {
		regretInfo, ok := p.ForecasterRegrets[forecaster]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in GetForecasterRegretsSlice %v", forecaster))
			continue
		}
		if !regretInfo.noPriorRegret {
			regrets = append(regrets, regretInfo.regret)
		}
	}
	return regrets
}

func (p *SynthPalette) UpdateInferersInfo(newInferers []Worker) error {
	p.Inferers = newInferers
	inferenceByWorker := make(map[Worker]*emissionstypes.Inference)
	infererRegrets := make(map[Worker]*StatefulRegret)

	p.InferersNewStatus = InferersAllNew
	for _, inferer := range p.Inferers {
		regretInfo, ok := p.InfererRegrets[inferer]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find inferer in InfererRegrets in UpdateInferersInfo %v", inferer))
			continue
		}

		if !regretInfo.noPriorRegret {
			if p.InferersNewStatus == InferersAllNew {
				p.InferersNewStatus = InferersAllNewExceptOne
				p.SingleNotNewInferer = inferer
			} else {
				p.InferersNewStatus = InferersNotNew
				p.SingleNotNewInferer = ""
			}
		}

		inference, ok := p.InferenceByWorker[inferer]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find inferer in InferenceByWorker in UpdateInferersInfo %v", inferer))
			continue
		}
		infererRegrets[inferer] = regretInfo
		inferenceByWorker[inferer] = inference
	}
	p.InferenceByWorker = inferenceByWorker
	p.InfererRegrets = infererRegrets

	return nil
}

// UpdateForecasters updates the forecasters and their related fields in the SynthPalette.
func (p *SynthPalette) UpdateForecastersInfo(newForecasters []Worker) error {
	p.Forecasters = newForecasters
	forecastByWorker := make(map[Worker]*emissionstypes.Forecast)
	forecasterRegrets := make(map[Worker]*StatefulRegret)

	for _, forecaster := range p.Forecasters {
		regretInfo, ok := p.ForecasterRegrets[forecaster]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in UpdateForecastersInfo %v", forecaster))
			continue
		}

		if !regretInfo.noPriorRegret {
			if p.ForecastersNewStatus == ForecastersAllNew {
				p.ForecastersNewStatus = ForecastersAllNewExceptOne
			} else {
				p.ForecastersNewStatus = ForecastersNotNew
			}
		}

		forecast, ok := p.ForecastByWorker[forecaster]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecastByWorker in UpdateForecastersInfo %v", forecaster))
			continue
		}
		forecasterRegrets[forecaster] = regretInfo
		forecastByWorker[forecaster] = forecast
	}

	p.ForecastByWorker = forecastByWorker
	p.ForecasterRegrets = forecasterRegrets

	return nil
}

func AccumulateWeights(
	inference *emissionstypes.Inference,
	weight alloraMath.Dec,
	noPriorRegret bool,
	allPeersAreNew bool,
	runningUnnormalizedI_i alloraMath.Dec,
	sumWeights alloraMath.Dec,
) (alloraMath.Dec, alloraMath.Dec, error) {
	err := error(nil)

	// If there is no prior regret and there is at least 1 worker of the same type (inferer/forecaster)
	// already with a reget => skip this worker (set weight=0)
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

func CalcWeightFromNormalizedRegret(
	normalizedRegret alloraMath.Dec,
	maxNormalizedRegret alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (alloraMath.Dec, error) {
	// upper bound: c + 6.75 / p
	v6Point75OverP, err := alloraMath.MustNewDecFromString("6.75").Quo(pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating upper bound for regret normalization")
	}
	cPlus6Point75OverP, err := cNorm.Add(v6Point75OverP)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating upper bound for regret normalization")
	}

	// lower bound: c - 8.25 / p
	v8Point25OverP, err := alloraMath.MustNewDecFromString("8.25").Quo(pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower bound for regret normalization")
	}
	cMinus8Point25OverP, err := cNorm.Sub(v8Point25OverP)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower bound for regret normalization")
	}

	// threshold for zero weight: c - 17.25 / p
	v17Point25OverP, err := alloraMath.MustNewDecFromString("17.25").Quo(pNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower bound for regret normalization")
	}
	cMinus17Point25OverP, err := cNorm.Sub(v17Point25OverP)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating lower threshold for zero weight")
	}

	// Cap the normalized regrets at an upper value
	// regretFrac = min(regretFrac, c + 6.75 / p)
	if normalizedRegret.Gt(cPlus6Point75OverP) {
		normalizedRegret = cPlus6Point75OverP
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

	// Set weight to zero for low regrets
	// if regretFrac < c - 17.25 / p, then weight = 0
	if normalizedRegret.Lt(cMinus17Point25OverP) {
		return alloraMath.ZeroDec(), nil
	}

	weight, err := alloraMath.Gradient(pNorm, cNorm, normalizedRegret) // w_ijk = φ'_p(\hatR_ijk)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "error calculating gradient")
	}

	return weight, nil
}
