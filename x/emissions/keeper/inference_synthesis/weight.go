package inferencesynthesis

import (
	"fmt"

	"cosmossdk.io/log"

	errorsmod "cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalcWeightsGivenWorkersArgs holds inputs for CalcWeightsGivenWorkers.
type CalcWeightsGivenWorkersArgs struct {
	Logger                 log.Logger
	Inferers               []Worker
	Forecasters            []Worker
	InfererToRegret        map[Worker]*alloraMath.Dec
	ForecasterToRegret     map[Worker]*alloraMath.Dec
	EpsilonTopic           alloraMath.Dec
	PNorm                  alloraMath.Dec
	CNorm                  alloraMath.Dec
	RegretScalePlusEpsilon alloraMath.Dec
}

// CalcRegretScaleFilteredByWeightsArgs holds inputs for CalcRegretScaleFilteredByWeights.
type CalcRegretScaleFilteredByWeightsArgs struct {
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

// Scale factor to convert MAD to an estimate of standard deviation under a normal distribution.
// 1 / Phi^-1(0.75) ~= 1.4826
var madToStdDevFactor = alloraMath.MustNewDecFromString("1.4826")

// Math helper cache implementation
// Used to memoize expensive helper computations like Gradient and Exp1DivExp1.
var (
	mathHelperCache  = make(map[string]alloraMath.Dec)
	mathCacheEnabled = false
)

func enableMathCache() {
	mathCacheEnabled = true
}

func disableMathCache() {
	mathCacheEnabled = false
}

func clearMathCache() {
	mathHelperCache = make(map[string]alloraMath.Dec)
}

func cachedExp1DivExp1(a, b alloraMath.Dec) (alloraMath.Dec, error) {
	if !mathCacheEnabled {
		return alloraMath.Exp1DivExp1(a, b)
	}

	key := fmt.Sprintf("exp1divexp1:%s:%s", a.String(), b.String())

	if cachedValue, exists := mathHelperCache[key]; exists {
		return cachedValue, nil
	}

	result, err := alloraMath.Exp1DivExp1(a, b)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "error calculating Exp1DivExp1")
	}

	mathHelperCache[key] = result
	return result, nil
}

// CalcRegretScaleFilteredByWeights calculates the MAD-based scale (scaled to match stddev under normality)
// of the regrets provided plus epsilon. It uses previous epoch's weights to filter the regrets of workers
// that had a negligible weight. If there are less than 2 non-negligible weights, it uses all regrets.
func CalcRegretScaleFilteredByWeights(args CalcRegretScaleFilteredByWeightsArgs) (alloraMath.Dec, error) {
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
		return CalcRegretScalePlusEpsilon(regrets, args.EpsilonTopic)
	}
	return CalcRegretScalePlusEpsilon(filteredRegrets, args.EpsilonTopic)
}

// CalcRegretScalePlusEpsilon calculates the MAD-based scale (scaled to match stddev under normality)
// of the regrets provided plus epsilon.
func CalcRegretScalePlusEpsilon(regrets []alloraMath.Dec, epsilonTopic alloraMath.Dec) (alloraMath.Dec, error) {
	// Calc MAD of regrets, scaled to match stddev under normality, + epsilon
	// madToStdDevFactor * MAD(R_ijk) + ε
	madRegrets, _, err := alloraMath.MedianAbsoluteDeviation(regrets)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating MAD of regrets")
	}
	scaledMadRegrets, err := madRegrets.Mul(madToStdDevFactor)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error scaling MAD of regrets")
	}
	// Add epsilon to scaled MAD
	absScaledMadRegrets, err := scaledMadRegrets.Abs()
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating absolute value of scaled MAD")
	}
	return absScaledMadRegrets.Add(epsilonTopic)
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

	var regretScalePlusEpsilon alloraMath.Dec
	if args.RegretScalePlusEpsilon.Gt(alloraMath.ZeroDec()) {
		regretScalePlusEpsilon = args.RegretScalePlusEpsilon
	} else {
		args.Logger.Debug("CalcWeightsGivenWorkers(): regretScalePlusEpsilon is not provided, calculating it")
		// Calc MAD-based scale of regrets + epsilon
		// madToStdDevFactor * MAD(R_ijk) + ε
		var err error
		regretScalePlusEpsilon, err = CalcRegretScalePlusEpsilon(regrets, args.EpsilonTopic)
		if err != nil {
			return RegretInformedWeights{}, errorsmod.Wrapf(err, "Error adding epsilon to regret scale")
		}
	}

	// Normalize the regrets and find the max normalized regret among them
	normalizedInfererRegrets := make(map[Worker]Regret)
	maxRegret := alloraMath.ZeroDec()
	maxRegretInitialized := false
	for _, worker := range args.Inferers {
		regret, ok := args.InfererToRegret[worker]
		if !ok {
			args.Logger.Debug("Cannot find worker in InfererRegrets in CalcWeightsGivenWorkers", "worker", worker)
			continue
		}
		regretFrac, err := regret.Quo(regretScalePlusEpsilon)
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
				args.Logger.Debug("Cannot find worker in ForecasterRegrets in CalcWeightsGivenWorkers", "worker", worker)
				continue
			}
			regretFrac, err := regret.Quo(regretScalePlusEpsilon)
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
				args.logger.Debug("Cannot find inferer in InferenceByWorker in CalcWeightedInference", "inferer", inferer)
				continue
			}
			infererWeight, exists := args.weights.Inferers[inferer]
			if !exists {
				args.logger.Debug("Cannot find inferer in weights.inferers in CalcWeightedInference", "inferer", inferer)
				continue
			}
			_, exists = args.infererToRegret[inferer]
			if !exists {
				args.logger.Debug("Cannot find inferer in InfererRegrets in CalcWeightedInference", "inferer", inferer)
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
				args.logger.Debug("Cannot find forecaster in ForecastImpliedInferenceByWorker in CalcWeightedInference", "forecaster", forecaster)
				continue
			}
			forecasterWeight, exists := args.weights.Forecasters[forecaster]
			if !exists {
				args.logger.Debug("Cannot find forecaster in weights.forecasters in CalcWeightedInference", "forecaster", forecaster)
				continue
			}
			_, exists = args.forecasterToRegret[forecaster]
			if !exists {
				args.logger.Debug("Cannot find forecaster in ForecasterRegrets in CalcWeightedInference", "forecaster", forecaster)
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
			logger.Debug("Cannot find inferer in InfererRegrets in GetInfererRegretsSlice", "inferer", inferer)
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
			logger.Debug("Cannot find forecaster in ForecasterRegrets in GetForecasterRegretsSlice", "forecaster", forecaster)
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

// Normalized Regret Variables - removed old bound constants as they're no longer needed

func CalcWeightFromNormalizedRegret(
	normalizedRegret alloraMath.Dec,
	maxNormalizedRegret alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (alloraMath.Dec, error) {
	// Calculate exponent_regret = -p_norm * (normalized_regret - c_norm)
	regretMinusC, err := normalizedRegret.Sub(cNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating normalized_regret - c_norm")
	}
	exponentRegret, err := pNorm.Mul(regretMinusC)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating p_norm * (normalized_regret - c_norm)")
	}
	exponentRegret, err = exponentRegret.Neg()
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error negating exponent_regret")
	}

	// Calculate exponent_max_regret = -p_norm * (max_normalized_regret - c_norm)
	maxRegretMinusC, err := maxNormalizedRegret.Sub(cNorm)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating max_normalized_regret - c_norm")
	}
	exponentMaxRegret, err := pNorm.Mul(maxRegretMinusC)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating p_norm * (max_normalized_regret - c_norm)")
	}
	exponentMaxRegret, err = exponentMaxRegret.Neg()
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error negating exponent_max_regret")
	}

	// Calculate weight = exp1_div_exp1(exponent_max_regret, exponent_regret)
	weight, err := cachedExp1DivExp1(exponentMaxRegret, exponentRegret)
	if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating Exp1DivExp1")
	}

	return weight, nil
}

// NormalizeWeights normalizes all weights so their sum equals 1.0 while preserving relative proportions
func (w *RegretInformedWeights) NormalizeWeights() error {
	// Calculate total sum of all weights
	sum := alloraMath.ZeroDec()
	var err error

	// Get sorted worker lists
	infererWorkers := alloraMath.GetSortedKeys(w.Inferers)
	forecasterWorkers := alloraMath.GetSortedKeys(w.Forecasters)

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
	infererWorkers := alloraMath.GetSortedKeys(weights.Inferers)
	for _, worker := range infererWorkers {
		err := k.SetLatestInfererWeight(ctx, topicId, worker, weights.Inferers[worker])
		if err != nil {
			return errorsmod.Wrapf(err, "error setting latest inferer weight for worker %s", worker)
		}
	}

	// Set forecaster weights
	forecasterWorkers := alloraMath.GetSortedKeys(weights.Forecasters)
	for _, worker := range forecasterWorkers {
		err := k.SetLatestForecasterWeight(ctx, topicId, worker, weights.Forecasters[worker])
		if err != nil {
			return errorsmod.Wrapf(err, "error setting latest forecaster weight for worker %s", worker)
		}
	}

	return nil
}
