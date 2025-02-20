package inferencesynthesis

import (
	"fmt"
	"sort"

	"cosmossdk.io/log"

	errorsmod "cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// args for calcWeightsGivenWorkers function
type CalcWeightsGivenWorkersArgs struct {
	Logger             log.Logger
	Inferers           []Worker
	Forecasters        []Worker
	InfererToRegret    map[Worker]*alloraMath.Dec
	ForecasterToRegret map[Worker]*alloraMath.Dec
	EpsilonTopic       alloraMath.Dec
	PNorm              alloraMath.Dec
	CNorm              alloraMath.Dec
	StdDevPlusEpsilon  alloraMath.Dec
}

type CalcRegretStdDevFilteredByWeightsArgs struct {
	Ctx                 sdk.Context
	K                   *keeper.Keeper
	Logger              log.Logger
	TopicId             uint64
	Inferers            []Worker
	Forecasters         []Worker
	InfererToRegret     map[Worker]*alloraMath.Dec
	ForecasterToRegret  map[Worker]*alloraMath.Dec
	NegligibleThreshold alloraMath.Dec
	EpsilonTopic        alloraMath.Dec
}

// Calculates the standard deviation of the regrets provided plus epsilon
// It uses previous epoch's weights to filter the regrets of workers that had a negligible weight
// If there are less than 2 non-negligible weights, it uses all regrets.
func CalcRegretStdDevFilteredByWeights(args CalcRegretStdDevFilteredByWeightsArgs) (alloraMath.Dec, error) {
	// Combine all weights and regrets
	var filteredRegrets []alloraMath.Dec
	nonNegligibleCount := 0

	// Count non-negligible weights and gather corresponding regrets
	for _, worker := range args.Inferers {
		weight, err := args.K.GetLatestInfererWeight(args.Ctx, args.TopicId, worker)
		if err != nil {
			continue
		}
		if weight.Gt(args.NegligibleThreshold) {
			nonNegligibleCount++
			if regret, ok := args.InfererToRegret[worker]; ok {
				filteredRegrets = append(filteredRegrets, *regret)
			}
		}
	}
	for _, worker := range args.Forecasters {
		weight, err := args.K.GetLatestForecasterWeight(args.Ctx, args.TopicId, worker)
		if err != nil {
			continue
		}
		if weight.Gt(args.NegligibleThreshold) {
			nonNegligibleCount++
			if regret, ok := args.ForecasterToRegret[worker]; ok {
				filteredRegrets = append(filteredRegrets, *regret)
			}
		}
	}

	// If fewer than 2 non-negligible weights, use all regrets
	if nonNegligibleCount < 2 {
		regrets, _, _, err := GatherWorkerRegrets(
			args.Logger,
			args.Inferers,
			args.Forecasters,
			args.InfererToRegret,
			args.ForecasterToRegret,
		)
		if err != nil {
			return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error gathering worker regrets")
		}
		return CalcStdDevPlusEpsilon(regrets, args.EpsilonTopic)
	}
	return CalcStdDevPlusEpsilon(filteredRegrets, args.EpsilonTopic)
}

// Calculates the standard deviation of the regrets provided plus epsilon
func CalcStdDevPlusEpsilon(regrets []alloraMath.Dec, epsilonTopic alloraMath.Dec) (alloraMath.Dec, error) {
	// Calc std dev of regrets + epsilon
	// σ(R_ijk) + ε
	stdDevRegrets, err := alloraMath.StdDev(regrets)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating standard deviation of regrets")
	}
	// Add epsilon to standard deviation
	absStdDevRegrets, err := stdDevRegrets.Abs()
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating absolute value of standard deviation")
	}
	return absStdDevRegrets.Add(epsilonTopic)
}

// Gather regrets from workers and forecasters.
func GatherWorkerRegrets(
	logger log.Logger,
	inferers []Worker,
	forecasters []Worker,
	infererToRegret map[Worker]*alloraMath.Dec,
	forecasterToRegret map[Worker]*alloraMath.Dec,
) ([]alloraMath.Dec, []alloraMath.Dec, []alloraMath.Dec, error) {
	var regrets []alloraMath.Dec
	infererRegrets := getInfererRegretsSlice(logger, inferers, infererToRegret)
	forecasterRegrets := getForecasterRegretsSlice(logger, forecasters, forecasterToRegret)

	if len(infererRegrets) > 0 {
		regrets = append(regrets, infererRegrets...)
	}
	if len(forecasterRegrets) > 0 {
		regrets = append(regrets, forecasterRegrets...)
	}
	if len(regrets) == 0 {
		return nil, nil, nil, errorsmod.Wrapf(emissionstypes.ErrEmptyArray, "No regrets to calculate weights")
	}

	return regrets, infererRegrets, forecasterRegrets, nil
}

// Given the current set of inferers and forecasters, calculate their
// weights using the current regrets
func CalcWeightsGivenWorkers(args CalcWeightsGivenWorkersArgs) (RegretInformedWeights, error) {
	// Gather regrets from forecasters and inferers
	regrets, _, forecasterRegrets, err := GatherWorkerRegrets(
		args.Logger,
		args.Inferers,
		args.Forecasters,
		args.InfererToRegret,
		args.ForecasterToRegret,
	)
	if err != nil {
		return RegretInformedWeights{}, err
	}

	var stdDevRegretsPlusEpsilon alloraMath.Dec
	if args.StdDevPlusEpsilon.Gt(alloraMath.ZeroDec()) {
		stdDevRegretsPlusEpsilon = args.StdDevPlusEpsilon
	} else {
		args.Logger.Debug("CalcWeightsGivenWorkers(): stdDevRegretsPlusEpsilon is not provided, calculating it")
		// Calc std dev of regrets + epsilon
		// σ(R_ijk) + ε
		var err error
		stdDevRegretsPlusEpsilon, err = CalcStdDevPlusEpsilon(regrets, args.EpsilonTopic)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error adding epsilon to standard deviation")
		}
	}

	// Normalize the regrets and find the max normalized regret among them
	normalizedInfererRegrets := make(map[Worker]Regret)
	maxRegret := alloraMath.ZeroDec()
	maxRegretInitialized := false
	for _, worker := range args.Inferers {
		regret, ok := args.InfererToRegret[worker]
		if !ok {
			args.Logger.Debug(fmt.Sprintf("Cannot find worker in InfererRegrets in CalcWeightsGivenWorkers %v", worker))
			continue
		}
		regretFrac, err := regret.Quo(stdDevRegretsPlusEpsilon)
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
	if len(forecasterRegrets) > 0 {
		for _, worker := range args.Forecasters {
			regret, ok := args.ForecasterToRegret[worker]
			if !ok {
				args.Logger.Debug(fmt.Sprintf("Cannot find worker in ForecasterRegrets in CalcWeightsGivenWorkers %v", worker))
				continue
			}
			regretFrac, err := regret.Quo(stdDevRegretsPlusEpsilon)
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

	// Calculate the weights from the normalized regrets
	for _, worker := range args.Inferers {
		// If there is more than one not-new inferer, calculate the weight for the ones that are not new
		infererWeight, err := CalcWeightFromNormalizedRegret(normalizedInfererRegrets[worker], maxRegret, args.PNorm, args.CNorm)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating inferer weight")
		}

		infererWeights[worker] = infererWeight
	}

	if len(forecasterRegrets) > 0 {
		for _, worker := range args.Forecasters {
			forecasterWeight, err := CalcWeightFromNormalizedRegret(normalizedForecasterRegrets[worker], maxRegret, args.PNorm, args.CNorm)
			if err != nil {
				return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error calculating forecaster weight")
			}
			forecasterWeights[worker] = forecasterWeight
		}
	}

	return RegretInformedWeights{
		Inferers:    infererWeights,
		Forecasters: forecasterWeights,
	}, nil
}

type calcWeightedInferenceArgs struct {
	logger                               log.Logger
	allInferersAreNew                    bool
	inferers                             []Worker
	workerToInference                    map[Worker]*emissionstypes.Inference
	infererToRegret                      map[Worker]*alloraMath.Dec
	forecasters                          []Worker
	forecasterToRegret                   map[Worker]*alloraMath.Dec
	forecasterToForecastImpliedInference map[Worker]*emissionstypes.Inference
	weights                              RegretInformedWeights
	epsilonSafeDiv                       alloraMath.Dec
}

// Calculates network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper:
// I_i = Σ_l w_il I_il / Σ_l w_il
// w_il = φ'_p(\hatR_i-1,l)
// \hatR_i-1,l = R_i-1,l / |max_{l'}(R_i-1,l')|
// given inferences, forecast-implied inferences, and network regrets
func calcWeightedInference(args calcWeightedInferenceArgs) (InferenceValue, error) {
	runningUnnormalizedI_i := alloraMath.ZeroDec() //nolint:revive // var-naming: don't use underscores in Go names
	sumWeights := alloraMath.ZeroDec()
	err := error(nil)

	// If all inferers are new, then the weight is 1 for all inferers
	if args.allInferersAreNew {
		for _, inferer := range args.inferers {
			runningUnnormalizedI_i, err = runningUnnormalizedI_i.Add(args.workerToInference[inferer].Value)
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error adding weight by worker value")
			}
			sumWeights, err = sumWeights.Add(alloraMath.OneDec())
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error adding weight")
			}
		}
	} else {
		for _, inferer := range args.inferers {
			inferenceByWorker, exists := args.workerToInference[inferer]
			if !exists {
				args.logger.Debug(fmt.Sprintf("Cannot find inferer in InferenceByWorker in CalcWeightedInference %v", inferer))
				continue
			}
			infererWeight, exists := args.weights.Inferers[inferer]
			if !exists {
				args.logger.Debug(fmt.Sprintf("Cannot find inferer in weights.inferers in CalcWeightedInference %v", inferer))
				continue
			}
			_, exists = args.infererToRegret[inferer]
			if !exists {
				args.logger.Debug(fmt.Sprintf("Cannot find inferer in InfererRegrets in CalcWeightedInference %v", inferer))
				continue
			}
			runningUnnormalizedI_i, sumWeights, err = accumulateWeights(
				*inferenceByWorker,
				infererWeight,
				args.allInferersAreNew,
				runningUnnormalizedI_i,
				sumWeights,
			)
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating weight of inferer")
			}
		}
		for _, forecaster := range args.forecasters {
			workerForecastImpliedInference, exists := args.forecasterToForecastImpliedInference[forecaster]
			if !exists {
				args.logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecastImpliedInferenceByWorker in CalcWeightedInference %v", forecaster))
				continue
			}
			forecasterWeight, exists := args.weights.Forecasters[forecaster]
			if !exists {
				args.logger.Debug(fmt.Sprintf("Cannot find forecaster in weights.forecasters in CalcWeightedInference %v", forecaster))
				continue
			}
			_, exists = args.forecasterToRegret[forecaster]
			if !exists {
				args.logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in CalcWeightedInference %v", forecaster))
				continue
			}
			runningUnnormalizedI_i, sumWeights, err = accumulateWeights(
				*workerForecastImpliedInference,
				forecasterWeight,
				false,
				runningUnnormalizedI_i,
				sumWeights,
			)
			if err != nil {
				return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating weight of forecaster")
			}
		}
	}

	// Normalize the running unnormalized network inference to yield output
	if sumWeights.Lt(args.epsilonSafeDiv) {
		sumWeights = args.epsilonSafeDiv
	}
	ret, err := runningUnnormalizedI_i.Quo(sumWeights)
	if err != nil {
		return InferenceValue{}, errorsmod.Wrapf(err, "Error normalizing network inference")
	}
	return ret, nil
}

// getInfererRegretsSlice converts a map of inferer regrets into a slice, maintaining the order defined by the inferers array.
// Returns an empty slice if the regret map is empty or if no valid regrets are found for the provided inferers.
func getInfererRegretsSlice(
	logger log.Logger,
	inferers []Worker,
	infererToRegret map[Worker]*alloraMath.Dec,
) []alloraMath.Dec {
	var regrets []alloraMath.Dec
	if len(infererToRegret) == 0 {
		return regrets
	}
	regrets = make([]alloraMath.Dec, 0, len(inferers))
	for _, inferer := range inferers {
		regret, ok := infererToRegret[inferer]
		if !ok {
			logger.Debug(fmt.Sprintf("Cannot find inferer in InfererRegrets in GetInfererRegretsSlice %v", inferer))
			continue
		}
		regrets = append(regrets, *regret)
	}
	return regrets
}

// getForecasterRegretsSlice converts a map of forecaster regrets into a slice, maintaining the order defined by the forecasters array.
// Returns an empty slice if the regret map is empty or if no valid regrets are found for the provided forecasters.
func getForecasterRegretsSlice(
	logger log.Logger,
	forecasters []Worker,
	forecasterToRegret map[Worker]*alloraMath.Dec,
) []alloraMath.Dec {
	var regrets []alloraMath.Dec
	if len(forecasterToRegret) == 0 {
		return regrets
	}
	regrets = make([]alloraMath.Dec, 0, len(forecasters))
	for _, forecaster := range forecasters {
		regret, ok := forecasterToRegret[forecaster]
		if !ok {
			logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in GetForecasterRegretsSlice %v", forecaster))
			continue
		}
		regrets = append(regrets, *regret)
	}
	return regrets
}

// sum up all of the inference values into running network combined inference
// and sum up all of the weights of all of the inferers
func accumulateWeights(
	inference emissionstypes.Inference,
	weight alloraMath.Dec,
	allPeersAreNew bool,
	runningUnnormalizedI_i alloraMath.Dec, //nolint:revive // var-naming: don't use underscores in Go names
	sumWeights alloraMath.Dec,
) (alloraMath.Dec, alloraMath.Dec, error) {
	err := error(nil)

	// Avoid needless computation if the weight is 0 or if there is no inference
	if weight.IsNaN() || weight.Equal(alloraMath.ZeroDec()) {
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

// NormalizeWeights normalizes all weights so their sum equals 1.0 while preserving relative proportions
func (w *RegretInformedWeights) NormalizeWeights() error {
	// Calculate total sum of all weights
	sum := alloraMath.ZeroDec()
	var err error

	// Get sorted worker lists
	infererWorkers := make([]string, 0, len(w.Inferers))
	for worker := range w.Inferers {
		infererWorkers = append(infererWorkers, worker)
	}
	sort.Strings(infererWorkers)

	forecasterWorkers := make([]string, 0, len(w.Forecasters))
	for worker := range w.Forecasters {
		forecasterWorkers = append(forecasterWorkers, worker)
	}
	sort.Strings(forecasterWorkers)

	// Sum weights in deterministic order
	for _, worker := range infererWorkers {
		sum, err = sum.Add(w.Inferers[worker])
		if err != nil {
			return errorsmod.Wrapf(err, "error adding inferer weight")
		}
	}
	for _, worker := range forecasterWorkers {
		sum, err = sum.Add(w.Forecasters[worker])
		if err != nil {
			return errorsmod.Wrapf(err, "error adding forecaster weight")
		}
	}

	// If sum is zero, we can't normalize
	if sum.IsZero() {
		return errorsmod.Wrap(emissionstypes.ErrInvalidValue, "cannot normalize weights: sum is zero")
	}

	// Normalize each weight in deterministic order
	for _, worker := range infererWorkers {
		normalizedWeight, err := w.Inferers[worker].Quo(sum)
		if err != nil {
			return errorsmod.Wrapf(err, "error normalizing inferer weight for %s", worker)
		}
		w.Inferers[worker] = normalizedWeight
	}

	for _, worker := range forecasterWorkers {
		normalizedWeight, err := w.Forecasters[worker].Quo(sum)
		if err != nil {
			return errorsmod.Wrapf(err, "error normalizing forecaster weight for %s", worker)
		}
		w.Forecasters[worker] = normalizedWeight
	}

	return nil
}

// StoreLatestNormalizedWeights sets the latest weights for the given topic
func StoreLatestNormalizedWeights(ctx sdk.Context, k keeper.Keeper, topicId TopicId, weights RegretInformedWeights) error {
	// Set inferer weights
	for worker, weight := range weights.Inferers {
		err := k.SetLatestInfererWeight(ctx, topicId, worker, weight)
		if err != nil {
			return errorsmod.Wrapf(err, "error setting latest inferer weight for worker %s", worker)
		}
	}

	// Set forecaster weights
	for worker, weight := range weights.Forecasters {
		err := k.SetLatestForecasterWeight(ctx, topicId, worker, weight)
		if err != nil {
			return errorsmod.Wrapf(err, "error setting latest forecaster weight for worker %s", worker)
		}
	}

	return nil
}
