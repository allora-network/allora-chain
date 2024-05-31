package inference_synthesis

import (
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Create a map from worker address to their inference or forecast-implied inference
func MakeMapFromWorkerToTheirWork(inferences []*emissions.Inference) map[Worker]*emissions.Inference {
	inferencesByWorker := make(map[Worker]*emissions.Inference)
	for _, inference := range inferences {
		inferencesByWorker[inference.Inferer] = inference
	}
	return inferencesByWorker
}

type AllWorkersAreNew struct {
	AllInferersAreNew    bool
	AllForecastersAreNew bool
}

func AreAllWorkersNew(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	sortedInferers []Worker,
	forecasts *emissions.Forecasts,
) (AllWorkersAreNew, error) {
	allInferersAreNew := true
	for _, inferer := range sortedInferers {
		_, noPriorRegret, err := k.GetInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return AllWorkersAreNew{}, errorsmod.Wrapf(err, "Error getting inferer regret")
		}

		allInferersAreNew = allInferersAreNew && noPriorRegret
	}

	allForecastersAreNew := true
	for _, forecast := range forecasts.Forecasts {
		_, noPriorRegret, err := k.GetForecasterNetworkRegret(ctx, topicId, forecast.Forecaster)
		if err != nil {
			return AllWorkersAreNew{}, errorsmod.Wrapf(err, "Error getting forecaster regret")
		}

		allForecastersAreNew = allForecastersAreNew && noPriorRegret
	}

	return AllWorkersAreNew{
		AllInferersAreNew:    allInferersAreNew,
		AllForecastersAreNew: allForecastersAreNew,
	}, nil
}

type NormalizedRegrets struct {
	Regrets   map[string]Regret
	MaxRegret Regret
}

func GetInfererNormalizedRegretsWithMax(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	sortedInferers []Worker,
	fTolerance alloraMath.Dec,
) (NormalizedRegrets, error) {
	infererRegrets := make([]Regret, 0)
	for _, inferer := range sortedInferers {
		regret, _, err := k.GetInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error getting inferer regret")
		}
		infererRegrets = append(infererRegrets, regret.Value)
	}

	// Calc std dev of regrets + f_tolerance
	// σ(R_ijk) + ε
	stdDevRegrets, err := alloraMath.StdDev(infererRegrets)
	if err != nil {
		return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error calculating standard deviation of inferer regrets")
	}
	// Add f_tolerance to standard deviation
	stdDevRegretsPlusEpsilon, err := stdDevRegrets.Add(fTolerance)
	if err != nil {
		return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error adding epsilon to standard deviation of inferer regrets")
	}

	// Normalize the regrets
	normalizedRegrets := make(map[string]Regret)
	maxRegret := alloraMath.ZeroDec()
	for i, inferer := range sortedInferers {
		regretFrac, err := infererRegrets[i].Quo(stdDevRegretsPlusEpsilon.Abs())
		if err != nil {
			return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
		}
		normalizedRegrets[inferer] = regretFrac
		if i == 0 || regretFrac.Gt(maxRegret) {
			maxRegret = regretFrac
		}
	}

	return NormalizedRegrets{
		Regrets:   normalizedRegrets,
		MaxRegret: maxRegret,
	}, nil
}

func GetForecasterNormalizedRegretsWithMax(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	sortedForecasters []Worker,
	fTolerance alloraMath.Dec,
) (NormalizedRegrets, error) {
	forecasterRegrets := make([]Regret, 0)
	for _, forecaster := range sortedForecasters {
		regret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, forecaster)
		if err != nil {
			return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error getting forecaster regret")
		}
		forecasterRegrets = append(forecasterRegrets, regret.Value)
	}

	// Calc std dev of regrets + f_tolerance
	stdDevRegrets, err := alloraMath.StdDev(forecasterRegrets)
	if err != nil {
		return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error calculating standard deviation of forecaster regrets")
	}
	// Add f_tolerance to standard deviation
	stdDevRegretsPlusEpsilon, err := stdDevRegrets.Add(fTolerance)
	if err != nil {
		return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error adding epsilon to standard deviation of forecaster regrets")
	}

	// Normalize the regrets
	normalizedRegrets := make(map[string]Regret)
	maxRegret := alloraMath.ZeroDec()
	for i, forecaster := range sortedForecasters {
		regretFrac, err := forecasterRegrets[i].Quo(stdDevRegretsPlusEpsilon.Abs())
		if err != nil {
			return NormalizedRegrets{}, errorsmod.Wrapf(err, "Error calculating regret fraction")
		}
		normalizedRegrets[forecaster] = regretFrac
		if i == 0 || regretFrac.Gt(maxRegret) {
			maxRegret = regretFrac
		}
	}

	return NormalizedRegrets{
		Regrets:   normalizedRegrets,
		MaxRegret: maxRegret,
	}, nil
}

func accumulateNormalizedI_iAndSumWeights(
	inference *emissions.Inference,
	normalizedRegret alloraMath.Dec,
	noPriorRegret bool,
	allWorkersAreNew bool,
	maxRegret Regret,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
	unnormalizedI_i alloraMath.Dec,
	sumWeights alloraMath.Dec,
) (alloraMath.Dec, alloraMath.Dec, error) {
	err := error(nil)

	// If there is no prior regret and there is at least 1 non-new forecaster => skip this forecaster, i.e. set their weight to 0
	if noPriorRegret && !allWorkersAreNew {
		return unnormalizedI_i, sumWeights, nil
	}

	// If all workers are new, then the weight is 1 for all workers
	// Otherwise, calculate the weight based on the regret of the worker
	if allWorkersAreNew {
		// If all workers are new, then the weight is 1 for all workers; take regular average of inferences
		unnormalizedI_i, err = unnormalizedI_i.Add(inference.Value)
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight by worker value")
		}
		sumWeights, err = sumWeights.Add(alloraMath.OneDec())
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight")
		}
	} else {
		weight, err := calcWeightFromRegret(normalizedRegret, maxRegret, pNorm, cNorm)
		if err != nil {
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating weight")
		}

		if !weight.Equal(alloraMath.ZeroDec()) && inference != nil {
			weightTimesInference, err := weight.Mul(inference.Value) // numerator of network combined inference calculation
			if err != nil {
				return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error calculating weight by worker value")
			}
			unnormalizedI_i, err = unnormalizedI_i.Add(weightTimesInference)
			if err != nil {
				return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight by worker value")
			}
			sumWeights, err = sumWeights.Add(weight)
			if err != nil {
				return alloraMath.ZeroDec(), alloraMath.ZeroDec(), errorsmod.Wrapf(err, "Error adding weight")
			}
		}
	}

	return unnormalizedI_i, sumWeights, nil
}

// Calculates network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper:
// I_i = Σ_l w_il I_il / Σ_l w_il
// w_il = φ'_p(\hatR_i-1,l)
// \hatR_i-1,l = R_i-1,l / |max_{l'}(R_i-1,l')|
// given inferences, forecast-implied inferences, and network regrets
func CalcWeightedInference(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferenceByWorker map[Worker]*emissions.Inference,
	sortedInferers []Worker,
	forecastImpliedInferenceByWorker map[Worker]*emissions.Inference,
	sortedForecasters []Worker,
	infererNormalizedRegrets NormalizedRegrets,
	forecasterNormalizedRegrets NormalizedRegrets,
	allWorkersAreNew AllWorkersAreNew,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (InferenceValue, error) {
	// Calculate the network combined inference and network worker regrets
	unnormalizedNetworkInferece := alloraMath.ZeroDec()
	sumWeights := alloraMath.ZeroDec()

	for _, inferer := range sortedInferers {
		// Get the regret of the inferer
		_, noPriorRegret, err := k.GetInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error getting inferer regret")
		}
		// Get normalized regret of the inferer
		normalizedRegret := infererNormalizedRegrets.Regrets[inferer]

		unnormalizedNetworkInferece, sumWeights, err = accumulateNormalizedI_iAndSumWeights(
			inferenceByWorker[inferer],
			normalizedRegret,
			noPriorRegret,
			allWorkersAreNew.AllInferersAreNew,
			infererNormalizedRegrets.MaxRegret,
			pNorm,
			cNorm,
			unnormalizedNetworkInferece,
			sumWeights,
		)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating i_i and weights for inferer")
		}
	}

	for _, forecaster := range sortedForecasters {
		// Get the regret of the forecaster
		_, noPriorRegret, err := k.GetForecasterNetworkRegret(ctx, topicId, forecaster)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error getting forecaster regret")
		}
		// Get normalized regret of the forecaster
		normalizedRegret := forecasterNormalizedRegrets.Regrets[forecaster]

		unnormalizedNetworkInferece, sumWeights, err = accumulateNormalizedI_iAndSumWeights(
			forecastImpliedInferenceByWorker[forecaster],
			normalizedRegret,
			noPriorRegret,
			allWorkersAreNew.AllForecastersAreNew,
			forecasterNormalizedRegrets.MaxRegret,
			pNorm,
			cNorm,
			unnormalizedNetworkInferece,
			sumWeights,
		)
		if err != nil {
			return InferenceValue{}, errorsmod.Wrapf(err, "Error accumulating i_i and weights for forecaster")
		}
	}

	// Normalize the network combined inference
	if sumWeights.Lt(epsilon) {
		return InferenceValue{}, emissions.ErrSumWeightsLessThanEta
	}
	ret, err := unnormalizedNetworkInferece.Quo(sumWeights)
	if err != nil {
		return InferenceValue{}, errorsmod.Wrapf(err, "Error calculating network combined inference")
	}
	return ret, nil // divide numerator by denominator to get network combined inference
}

// Returns all one-out inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker. Also that there is at most 1 forecast-implied inference per worker.
// Loop over all inferences and forecast-implied inferences and withold one inference. Then calculate the network inference less that witheld inference
// If an inference is held out => recalculate the forecast-implied inferences before calculating the network inference
func CalcOneOutInferences(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferenceByWorker map[Worker]*emissions.Inference,
	sortedInferers []Worker,
	forecastImpliedInferenceByWorker map[Worker]*emissions.Inference,
	sortedForecasters []Worker,
	forecasts *emissions.Forecasts,
	allWorkersAreNew AllWorkersAreNew,
	infererNormalizedRegrets NormalizedRegrets,
	forecasterNormalizedRegrets NormalizedRegrets,
	networkCombinedLoss Loss,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) ([]*emissions.WithheldWorkerAttributedValue, []*emissions.WithheldWorkerAttributedValue, error) {
	// Loop over inferences and reclculate forecast-implied inferences before calculating the network inference
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range sortedInferers {
		// Remove the inference of the worker from the inferences
		inferencesWithoutWorker := make(map[Worker]*emissions.Inference)

		for _, workerOfInference := range sortedInferers {
			if workerOfInference != worker {
				inferencesWithoutWorker[workerOfInference] = inferenceByWorker[workerOfInference]
			}
		}

		sortedInferersWithoutWorker := alloraMath.GetSortedKeys(inferencesWithoutWorker)

		// Recalculate the forecast-implied inferences without the worker's inference
		forecastImpliedInferencesWithoutWorkerByWorker, err := CalcForecastImpliedInferences(
			inferencesWithoutWorker,
			sortedInferersWithoutWorker,
			forecasts,
			networkCombinedLoss,
			allWorkersAreNew.AllInferersAreNew,
			epsilon,
			pNorm,
			cNorm,
		)

		if err != nil {
			return nil, nil, errorsmod.Wrapf(err, "Error calculating forecast-implied inferences for held-out inference")
		}

		oneOutNetworkInferenceWithoutInferer, err := CalcWeightedInference(
			ctx,
			k,
			topicId,
			inferencesWithoutWorker,
			sortedInferersWithoutWorker,
			forecastImpliedInferencesWithoutWorkerByWorker,
			sortedForecasters,
			infererNormalizedRegrets,
			forecasterNormalizedRegrets,
			allWorkersAreNew,
			epsilon,
			pNorm,
			cNorm,
		)
		if err != nil {
			return nil, nil, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutNetworkInferenceWithoutInferer,
		})
	}

	// Loop over forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneOutImpliedInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range sortedForecasters {
		// Remove the inference of the worker from the inferences
		impliedInferenceWithoutWorker := make(map[Worker]*emissions.Inference)

		for _, workerOfImpliedInference := range sortedForecasters {
			if workerOfImpliedInference != worker {
				impliedInferenceWithoutWorker[workerOfImpliedInference] = forecastImpliedInferenceByWorker[workerOfImpliedInference]
			}
		}

		sortedForecastersWithoutWorker := alloraMath.GetSortedKeys(impliedInferenceWithoutWorker)

		// Calculate the network inference without the worker's inference
		oneOutInference, err := CalcWeightedInference(
			ctx,
			k,
			topicId,
			inferenceByWorker,
			sortedInferers,
			impliedInferenceWithoutWorker,
			sortedForecastersWithoutWorker,
			infererNormalizedRegrets,
			forecasterNormalizedRegrets,
			allWorkersAreNew,
			epsilon,
			pNorm,
			cNorm,
		)
		if err != nil {
			return nil, nil, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
		}
		oneOutImpliedInferences = append(oneOutImpliedInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	return oneOutInferences, oneOutImpliedInferences, nil
}

// Returns all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker. Also that there is at most 1 forecast-implied inference per worker.
func CalcOneInInferences(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferencesByWorker map[Worker]*emissions.Inference,
	sortedInferers []Worker,
	forecastImpliedInferences map[Worker]*emissions.Inference,
	sortedForecasters []Worker,
	allWorkersAreNew AllWorkersAreNew,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) ([]*emissions.WorkerAttributedValue, error) {
	// Get Inferer normalized regrets and max regret
	infererNormalizedRegrets, err := GetInfererNormalizedRegretsWithMax(ctx, k, topicId, sortedInferers, epsilon)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error getting inferer normalized regrets: %s", err.Error()))
		return make([]*emissions.WorkerAttributedValue, 0), errorsmod.Wrapf(err, "Error calculating infererNormalizedRegrets in calc one-in inference")
	}

	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneInInferences := make([]*emissions.WorkerAttributedValue, 0)
	for _, oneInForecaster := range sortedForecasters {
		// In each loop, remove all forecast-implied inferences except one
		forecastImpliedInferencesWithForecaster := make(map[Worker]*emissions.Inference)
		forecastImpliedInferencesWithForecaster[oneInForecaster] = forecastImpliedInferences[oneInForecaster]
		// Calculate the network inference without the worker's forecast-implied inference

		sortedForecastersWithForecaster := alloraMath.GetSortedKeys(forecastImpliedInferencesWithForecaster)

		// Get Forecaster normalized regrets and max regret
		forecastNormalizedRegrets, err := GetForecasterNormalizedRegretsWithMax(ctx, k, topicId, sortedForecastersWithForecaster, epsilon)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error getting forecaster normalized regrets: %s", err.Error()))
			return make([]*emissions.WorkerAttributedValue, 0), errorsmod.Wrapf(err, "Error calculating forecastNormalizedRegrets in calc one-in inference")
		}

		oneInInference, err := CalcWeightedInference(
			ctx,
			k,
			topicId,
			inferencesByWorker,
			sortedInferers,
			forecastImpliedInferencesWithForecaster,
			sortedForecastersWithForecaster,
			infererNormalizedRegrets,
			forecastNormalizedRegrets,
			allWorkersAreNew,
			epsilon,
			pNorm,
			cNorm,
		)
		if err != nil {
			return make([]*emissions.WorkerAttributedValue, 0), errorsmod.Wrapf(err, "Error calculating one-in inference")
		}
		oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
			Worker: oneInForecaster,
			Value:  oneInInference,
		})
	}
	return oneInInferences, nil
}

// Calculates all network inferences in I_i given inferences, forecast implied inferences, and network combined inference.
// I_ij are the inferences of worker j and already given as an argument.
func CalcNetworkInferences(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	networkCombinedLoss Loss,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (*emissions.ValueBundle, error) {
	// Map each worker to their inference
	inferenceByWorker := MakeMapFromWorkerToTheirWork(inferences.Inferences)
	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	allWorkersAreNew, err := AreAllWorkersNew(ctx, k, topicId, sortedInferers, forecasts)
	if err != nil {
		return &emissions.ValueBundle{}, errorsmod.Wrapf(err, fmt.Sprintf("Error calculating forecast-implied inferences: %s", err.Error()))
	}

	// Calculate forecast-implied inferences I_ik
	forecastImpliedInferenceByWorker, err := CalcForecastImpliedInferences(
		inferenceByWorker,
		sortedInferers,
		forecasts,
		networkCombinedLoss,
		allWorkersAreNew.AllInferersAreNew,
		epsilon,
		pNorm,
		cNorm,
	)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error calculating forecast-implied inferences: %s", err.Error()))
	}
	sortedForecasters := alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker)

	// Get Inferer normalized regrets and max regret
	infererNormalizedRegrets, err := GetInfererNormalizedRegretsWithMax(ctx, k, topicId, sortedInferers, epsilon)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error getting inferer normalized regrets: %s", err.Error()))
	}

	// Get Forecaster normalized regrets and max regret
	forecastNormalizedRegrets, err := GetForecasterNormalizedRegretsWithMax(ctx, k, topicId, sortedForecasters, epsilon)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error getting forecaster normalized regrets: %s", err.Error()))
	}

	// Calculate the combined network inference I_i
	combinedNetworkInference, err := CalcWeightedInference(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		allWorkersAreNew,
		epsilon,
		pNorm,
		cNorm,
	)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error calculating network combined inference: %s", err.Error()))
	}

	// Calculate the naive inference I^-_i
	naiveInference, err := CalcWeightedInference(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		sortedInferers,
		nil,
		nil,
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		allWorkersAreNew,
		epsilon,
		pNorm,
		cNorm,
	)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error calculating naive inference: %s", err.Error()))
	}

	// Calculate the one-out inference I^-_li
	oneOutInferences, oneOutImpliedInferences, err := CalcOneOutInferences(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		forecasts,
		allWorkersAreNew,
		infererNormalizedRegrets,
		forecastNormalizedRegrets,
		networkCombinedLoss,
		epsilon,
		pNorm,
		cNorm,
	)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error calculating one-out inferences: %s", err.Error()))
		oneOutInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
		oneOutImpliedInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
	}

	// Calculate the one-in inference I^+_ki
	oneInInferences, err := CalcOneInInferences(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		sortedInferers,
		forecastImpliedInferenceByWorker,
		sortedForecasters,
		allWorkersAreNew,
		epsilon,
		pNorm,
		cNorm,
	)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error calculating one-in inferences: %s", err.Error()))
		oneInInferences = make([]*emissions.WorkerAttributedValue, 0)
	}

	// For completeness, send the inferences and forecastImpliedInferences in the bundle
	// Turn the forecast-implied inferences into a WorkerAttributedValue array
	infererValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, infererence := range inferences.Inferences {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: infererence.Inferer,
			Value:  infererence.Value,
		})
	}

	forecastImpliedValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range sortedForecasters {
		forecastImpliedValues = append(forecastImpliedValues, &emissions.WorkerAttributedValue{
			Worker: forecastImpliedInferenceByWorker[forecaster].Inferer,
			Value:  forecastImpliedInferenceByWorker[forecaster].Value,
		})
	}

	// Build value bundle to return all the calculated inferences
	return &emissions.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          combinedNetworkInference,
		InfererValues:          infererValues,
		ForecasterValues:       forecastImpliedValues,
		NaiveValue:             naiveInference,
		OneOutInfererValues:    oneOutInferences,
		OneOutForecasterValues: oneOutImpliedInferences,
		OneInForecasterValues:  oneInInferences,
	}, nil
}

func GetNetworkInferencesAtBlock(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferencesNonce BlockHeight,
	previousLossNonce BlockHeight,
) (*emissions.ValueBundle, error) {
	networkInferences := &emissions.ValueBundle{
		TopicId:          topicId,
		InfererValues:    make([]*emissions.WorkerAttributedValue, 0),
		ForecasterValues: make([]*emissions.WorkerAttributedValue, 0),
	}

	inferences, err := k.GetInferencesAtBlock(ctx, topicId, inferencesNonce)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("no inferences found for topic %v at block %v, %v", topicId, inferencesNonce, err.Error()))
		return networkInferences, nil
	}
	// Add inferences in the bundle -> this bundle will be used as a fallback in case of error
	for _, infererence := range inferences.Inferences {
		networkInferences.InfererValues = append(networkInferences.InfererValues, &emissions.WorkerAttributedValue{
			Worker: infererence.Inferer,
			Value:  infererence.Value,
		})
	}

	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, inferencesNonce)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			forecasts = &emissions.Forecasts{
				Forecasts: make([]*emissions.Forecast, 0),
			}
		} else {
			return nil, err
		}
	}

	if len(inferences.Inferences) > 1 {
		moduleParams, err := k.GetParams(ctx)
		if err != nil {
			return nil, err
		}

		reputerReportedLosses, err := k.GetReputerLossBundlesAtBlock(ctx, topicId, previousLossNonce)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error getting reputer losses: %s", err.Error()))
			return networkInferences, nil
		}

		// Map list of stakesOnTopic to map of stakesByReputer
		stakesByReputer := make(map[string]cosmosMath.Int)
		for _, bundle := range reputerReportedLosses.ReputerValueBundles {
			stakeAmount, err := k.GetStakeOnReputerInTopic(ctx, topicId, bundle.ValueBundle.Reputer)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error getting stake on reputer: %s", err.Error()))
				return networkInferences, nil
			}
			stakesByReputer[bundle.ValueBundle.Reputer] = stakeAmount
		}

		networkCombinedLoss, err := CalcCombinedNetworkLoss(
			stakesByReputer,
			reputerReportedLosses,
			moduleParams.Epsilon,
		)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error calculating network combined loss: %s", err.Error()))
			return networkInferences, nil
		}
		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error getting topic: %s", err.Error()))
			return networkInferences, nil
		}
		networkInferences, err = CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, moduleParams.Epsilon, topic.PNorm, moduleParams.CNorm)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error calculating network inferences: %s", err.Error()))
			return networkInferences, nil
		}
	} else {
		// If there is only one valid inference, then the network inference is the same as the single inference
		// For the forecasts to be meaningful, there should be at least 2 inferences
		singleInference := inferences.Inferences[0]

		networkInferences = &emissions.ValueBundle{
			TopicId:       topicId,
			CombinedValue: singleInference.Value,
			InfererValues: []*emissions.WorkerAttributedValue{
				{
					Worker: singleInference.Inferer,
					Value:  singleInference.Value,
				},
			},
			ForecasterValues:       []*emissions.WorkerAttributedValue{},
			NaiveValue:             singleInference.Value,
			OneOutInfererValues:    []*emissions.WithheldWorkerAttributedValue{},
			OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{},
			OneInForecasterValues:  []*emissions.WorkerAttributedValue{},
		}
	}

	return networkInferences, nil
}

// Filter nonces that are within the epoch length of the current block height
func FilterNoncesWithinEpochLength(n emissions.Nonces, blockHeight, epochLength int64) emissions.Nonces {
	var filtered emissions.Nonces
	for _, nonce := range n.Nonces {
		if blockHeight-nonce.BlockHeight <= epochLength {
			filtered.Nonces = append(filtered.Nonces, nonce)
		}
	}
	return filtered
}

func SortByBlockHeight(r []*emissions.ReputerRequestNonce) {
	sort.Slice(r, func(i, j int) bool {
		// Sorting in descending order (bigger values first)
		return r[i].ReputerNonce.BlockHeight > r[j].ReputerNonce.BlockHeight
	})
}

// Select the top N latest reputer nonces
func SelectTopNReputerNonces(reputerRequestNonces *emissions.ReputerRequestNonces, N int, currentBlockHeight int64, groundTruthLag int64) []*emissions.ReputerRequestNonce {
	topN := make([]*emissions.ReputerRequestNonce, 0)
	// sort reputerRequestNonces by reputer block height

	// Create a copy of the original slice to avoid modifying chain state
	sortedSlice := make([]*emissions.ReputerRequestNonce, len(reputerRequestNonces.Nonces))
	copy(sortedSlice, reputerRequestNonces.Nonces)
	SortByBlockHeight(sortedSlice)

	// loop reputerRequestNonces
	for _, nonce := range sortedSlice {
		nonceCopy := nonce
		if currentBlockHeight >= nonceCopy.ReputerNonce.BlockHeight+groundTruthLag {
			topN = append(topN, nonceCopy)
		}
		if len(topN) >= N {
			break
		}
	}
	return topN
}

// Select the top N latest worker nonces
func SelectTopNWorkerNonces(workerNonces emissions.Nonces, N int) []*emissions.Nonce {
	if len(workerNonces.Nonces) <= N {
		return workerNonces.Nonces
	}
	return workerNonces.Nonces[:N]
}
