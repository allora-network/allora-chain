package inference_synthesis

import (
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
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
	inferenceByWorker map[Worker]*emissions.Inference,
	forecasts *emissions.Forecasts,
) (AllWorkersAreNew, error) {
	allInferersAreNew := true
	for inferer := range inferenceByWorker {
		_, noPriorRegret, err := k.GetInfererNetworkRegret(ctx, topicId, sdk.AccAddress(inferer))
		if err != nil {
			fmt.Println("Error getting inferer regret: ", err)
			return AllWorkersAreNew{}, err // TODO: THIS OR continue ??
		}

		allInferersAreNew = allInferersAreNew && noPriorRegret
	}

	allForecastersAreNew := true
	for _, forecast := range forecasts.Forecasts {
		_, noPriorRegret, err := k.GetForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(forecast.Forecaster))
		if err != nil {
			fmt.Println("Error getting forecaster regret: ", err)
			return AllWorkersAreNew{}, err // TODO: THIS OR continue ??
		}

		allForecastersAreNew = allForecastersAreNew && noPriorRegret
	}

	return AllWorkersAreNew{
		AllInferersAreNew:    allInferersAreNew,
		AllForecastersAreNew: allForecastersAreNew,
	}, nil
}

type MaximalRegrets struct {
	MaxInferenceRegret     Regret
	MaxForecastRegret      Regret
	MaxOneInForecastRegret map[Worker]Regret // max regret for each one-in forecaster
}

// Find the maximum regret admitted by any worker for an inference or forecast task; used in calculating the network combined inference
func FindMaxRegretAmongWorkersWithLosses(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferenceByWorker map[Worker]*emissions.Inference,
	forecastImpliedInferenceByWorker map[Worker]*emissions.Inference,
	epsilon alloraMath.Dec,
) (MaximalRegrets, error) {
	maxInfererRegret := epsilon // averts div by 0 error
	for inferer := range inferenceByWorker {
		infererRegret, _, err := k.GetInfererNetworkRegret(ctx, topicId, sdk.AccAddress(inferer))
		if err != nil {
			fmt.Println("Error getting inferer regret: ", err)
			return MaximalRegrets{}, err // TODO: THIS OR continue ??
		}

		if maxInfererRegret.Lt(infererRegret.Value) {
			maxInfererRegret = infererRegret.Value
		}
	}

	maxForecasterRegret := epsilon // averts div by 0 error
	for forecaster := range forecastImpliedInferenceByWorker {
		forecasterRegret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(forecaster))
		if err != nil {
			fmt.Println("Error getting forecaster regret: ", err)
			return MaximalRegrets{}, err // TODO: THIS OR continue ??
		}

		if maxForecasterRegret.Lt(forecasterRegret.Value) {
			maxForecasterRegret = forecasterRegret.Value
		}
	}

	maxOneInForecasterRegret := make(map[Worker]Regret) // averts div by 0 error
	for forecaster := range forecastImpliedInferenceByWorker {
		for inferer := range inferenceByWorker {
			oneInForecasterRegret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(forecaster), sdk.AccAddress(inferer))
			if err != nil {
				fmt.Println("Error getting forecaster regret: ", err)
				return MaximalRegrets{}, err // TODO: THIS OR continue ??
			}
			if maxOneInForecasterRegret[forecaster].Lt(oneInForecasterRegret.Value) {
				maxOneInForecasterRegret[forecaster] = oneInForecasterRegret.Value
			}
		}

		oneInForecasterSelfRegret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(forecaster), sdk.AccAddress(forecaster))
		if err != nil {
			fmt.Println("Error getting one-in forecaster self regret: ", err)
			return MaximalRegrets{}, err // TODO: THIS OR continue ??
		}
		if maxOneInForecasterRegret[forecaster].Lt(oneInForecasterSelfRegret.Value) {
			maxOneInForecasterRegret[forecaster] = oneInForecasterSelfRegret.Value
		}
	}

	return MaximalRegrets{
		MaxInferenceRegret:     maxInfererRegret,
		MaxForecastRegret:      maxForecasterRegret,
		MaxOneInForecastRegret: maxOneInForecasterRegret,
	}, nil
}

func accumulateNormalizedI_iAndSumWeights(
	inference *emissions.Inference,
	regret emissions.TimestampedValue,
	noPriorRegret bool,
	allWorkersAreNew bool,
	maxRegret Regret,
	pInferenceSynthesis alloraMath.Dec,
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
		// If at least one worker is not new, then we take a weighted average of all workers' inferences

		// Normalize forecaster regret then calculate gradient => weight per forecaster for network combined inference
		regretFrac, err := regret.Value.Quo(maxRegret.Abs())
		if err != nil {
			fmt.Println("Error calculating regret fraction: ", err)
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
		}
		weight, err := Gradient(pInferenceSynthesis, regretFrac)
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
		}
		if !weight.Equal(alloraMath.ZeroDec()) && inference != nil {
			weightTimesInference, err := weight.Mul(inference.Value) // numerator of network combined inference calculation
			if err != nil {
				fmt.Println("Error calculating weight by worker value: ", err)
				return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
			}
			unnormalizedI_i, err = unnormalizedI_i.Add(weightTimesInference)
			if err != nil {
				fmt.Println("Error adding weight by worker value: ", err)
				return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
			}
			sumWeights, err = sumWeights.Add(weight)
			if err != nil {
				fmt.Println("Error adding weight: ", err)
				return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
			}
		}
	} else {
		// If all workers are new, then the weight is 1 for all workers; take regular average if inferences
		unnormalizedI_i, err = unnormalizedI_i.Add(inference.Value)
		if err != nil {
			fmt.Println("Error adding weight by worker value: ", err)
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
		}
		sumWeights, err = sumWeights.Add(alloraMath.OneDec())
		if err != nil {
			fmt.Println("Error adding weight: ", err)
			return alloraMath.ZeroDec(), alloraMath.ZeroDec(), err
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
	forecastImpliedInferenceByWorker map[Worker]*emissions.Inference,
	allWorkersAreNew AllWorkersAreNew,
	maxRegret Regret,
	epsilon alloraMath.Dec,
	pInferenceSynthesis alloraMath.Dec,
) (InferenceValue, error) {
	if maxRegret.Lt(epsilon) {
		fmt.Println("Error maxRegret < epsilon: ", maxRegret, epsilon)
		return InferenceValue{}, emissions.ErrFractionDivideByZero
	}

	// Calculate the network combined inference and network worker regrets
	unnormalizedI_i := alloraMath.ZeroDec()
	sumWeights := alloraMath.ZeroDec()

	for inferer := range inferenceByWorker {
		// Get the regret of the inferer
		regret, noPriorRegret, err := k.GetInfererNetworkRegret(ctx, topicId, sdk.AccAddress(inferer))
		if err != nil {
			fmt.Println("Error getting inferer regret: ", err)
			return InferenceValue{}, err
		}
		unnormalizedI_i, sumWeights, err = accumulateNormalizedI_iAndSumWeights(
			inferenceByWorker[inferer],
			regret,
			noPriorRegret,
			allWorkersAreNew.AllInferersAreNew,
			maxRegret,
			pInferenceSynthesis,
			unnormalizedI_i,
			sumWeights,
		)
		if err != nil {
			fmt.Println("Error accumulating i_i and weights for inferer: ", err)
			return InferenceValue{}, err
		}
	}

	for forecaster := range forecastImpliedInferenceByWorker {
		// Get the regret of the forecaster
		regret, noPriorRegret, err := k.GetForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(forecaster))
		if err != nil {
			fmt.Println("Error getting forecaster regret: ", err)
			return InferenceValue{}, err
		}
		unnormalizedI_i, sumWeights, err = accumulateNormalizedI_iAndSumWeights(
			forecastImpliedInferenceByWorker[forecaster],
			regret,
			noPriorRegret,
			allWorkersAreNew.AllForecastersAreNew,
			maxRegret,
			pInferenceSynthesis,
			unnormalizedI_i,
			sumWeights,
		)
		if err != nil {
			fmt.Println("Error accumulating i_i and weights for forecaster: ", err)
			return InferenceValue{}, err
		}
	}

	// Normalize the network combined inference
	if sumWeights.Lt(epsilon) {
		return InferenceValue{}, emissions.ErrSumWeightsLessThanEta
	}
	ret, err := unnormalizedI_i.Quo(sumWeights)
	if err != nil {
		fmt.Println("Error calculating network combined inference: ", err)
		return InferenceValue{}, err
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
	forecastImpliedInferenceByWorker map[Worker]*emissions.Inference,
	forecasts *emissions.Forecasts,
	allWorkersAreNew AllWorkersAreNew,
	maxRegret Regret,
	networkCombinedLoss Loss,
	epsilon alloraMath.Dec,
	pInferenceSynthesis alloraMath.Dec,
) ([]*emissions.WithheldWorkerAttributedValue, []*emissions.WithheldWorkerAttributedValue, error) {
	// Loop over inferences and reclculate forecast-implied inferences before calculating the network inference
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for worker := range inferenceByWorker {
		// Remove the inference of the worker from the inferences
		inferencesWithoutWorker := make(map[Worker]*emissions.Inference)

		for workerOfInference, inference := range inferenceByWorker {
			if workerOfInference != worker {
				inferencesWithoutWorker[workerOfInference] = inference
			}
		}

		// Recalculate the forecast-implied inferences without the worker's inference
		forecastImpliedInferencesWithoutWorkerByWorker, err := CalcForcastImpliedInferences(
			inferencesWithoutWorker,
			forecasts,
			networkCombinedLoss,
			allWorkersAreNew.AllInferersAreNew,
			epsilon,
			pInferenceSynthesis,
		)

		if err != nil {
			fmt.Println("Error calculating forecast-implied inferences for held-out inference: ", err)
			return nil, nil, err
		}

		oneOutNetworkInferenceWithoutInferer, err := CalcWeightedInference(
			ctx,
			k,
			topicId,
			inferencesWithoutWorker,
			forecastImpliedInferencesWithoutWorkerByWorker,
			allWorkersAreNew,
			maxRegret,
			epsilon,
			pInferenceSynthesis,
		)
		if err != nil {
			fmt.Println("Error calculating one-out inference for inferer: ", err)
			return nil, nil, err
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutNetworkInferenceWithoutInferer,
		})
	}

	// Loop over forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneOutImpliedInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for worker := range forecastImpliedInferenceByWorker {
		// Remove the inference of the worker from the inferences
		impliedInferenceWithoutWorker := make(map[Worker]*emissions.Inference)

		for workerOfImpliedInference, inference := range inferenceByWorker {
			if workerOfImpliedInference != worker {
				impliedInferenceWithoutWorker[workerOfImpliedInference] = inference
			}
		}

		// Calculate the network inference without the worker's inference
		oneOutInference, err := CalcWeightedInference(
			ctx,
			k,
			topicId,
			inferenceByWorker,
			impliedInferenceWithoutWorker,
			allWorkersAreNew,
			maxRegret,
			epsilon,
			pInferenceSynthesis,
		)
		if err != nil {
			fmt.Println("Error calculating one-out inference for forecaster: ", err)
			return nil, nil, err
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
	inferences map[Worker]*emissions.Inference,
	forecastImpliedInferences map[Worker]*emissions.Inference,
	allWorkersAreNew AllWorkersAreNew,
	maxRegretsByOneInForecaster map[Worker]Regret,
	epsilon alloraMath.Dec,
	pInferenceSynthesis alloraMath.Dec,
) ([]*emissions.WorkerAttributedValue, error) {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneInInferences := make([]*emissions.WorkerAttributedValue, 0)
	for oneInForecaster := range forecastImpliedInferences {
		// In each loop, remove all forecast-implied inferences except one
		forecastImpliedInferencesWithForecaster := make(map[Worker]*emissions.Inference)
		forecastImpliedInferencesWithForecaster[oneInForecaster] = forecastImpliedInferences[oneInForecaster]
		// Calculate the network inference without the worker's forecast-implied inference
		oneInInference, err := CalcWeightedInference(
			ctx,
			k,
			topicId,
			inferences,
			forecastImpliedInferencesWithForecaster,
			allWorkersAreNew,
			maxRegretsByOneInForecaster[oneInForecaster],
			epsilon,
			pInferenceSynthesis,
		)
		if err != nil {
			fmt.Println("Error calculating one-in inference: ", err)
			return nil, err
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
	pInferenceSynthesis alloraMath.Dec,
) (*emissions.ValueBundle, error) {
	// Map each worker to their inference
	inferenceByWorker := MakeMapFromWorkerToTheirWork(inferences.Inferences)

	allWorkersAreNew, err := AreAllWorkersNew(ctx, k, topicId, inferenceByWorker, forecasts)
	if err != nil {
		fmt.Println("Error checking if all workers are new: ", err)
		return nil, err
	}

	// Calculate forecast-implied inferences I_ik
	forecastImpliedInferenceByWorker, err := CalcForcastImpliedInferences(
		inferenceByWorker,
		forecasts,
		networkCombinedLoss,
		allWorkersAreNew.AllInferersAreNew,
		epsilon,
		pInferenceSynthesis,
	)
	if err != nil {
		fmt.Println("Error calculating forecast-implied inferences: ", err)
		return nil, err
	}

	// Find the maximum regret admitted by any worker for an inference or forecast task; used to normalize regrets that are passed to the gradient function
	currentMaxRegrets, err := FindMaxRegretAmongWorkersWithLosses(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		forecastImpliedInferenceByWorker,
		epsilon,
	)
	if err != nil {
		fmt.Println("Error finding max regret among workers with losses: ", err)
		return nil, err
	}
	maxCombinedRegret := alloraMath.Max(currentMaxRegrets.MaxInferenceRegret, currentMaxRegrets.MaxForecastRegret)

	// Calculate the combined network inference I_i
	combinedNetworkInference, err := CalcWeightedInference(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		forecastImpliedInferenceByWorker,
		allWorkersAreNew,
		maxCombinedRegret,
		epsilon,
		pInferenceSynthesis,
	)
	if err != nil {
		fmt.Println("Error calculating network combined inference: ", err)
		return nil, err
	}

	// Calculate the naive inference I^-_i
	naiveInference, err := CalcWeightedInference(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		nil,
		allWorkersAreNew,
		currentMaxRegrets.MaxInferenceRegret,
		epsilon,
		pInferenceSynthesis,
	)
	if err != nil {
		fmt.Println("Error calculating naive inference: ", err)
		return nil, err
	}

	// Calculate the one-out inference I^-_li
	oneOutInferences, oneOutImpliedInferences, err := CalcOneOutInferences(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		forecastImpliedInferenceByWorker,
		forecasts,
		allWorkersAreNew,
		maxCombinedRegret,
		networkCombinedLoss,
		epsilon,
		pInferenceSynthesis,
	)
	if err != nil {
		fmt.Println("Error calculating one-out inferences: ", err)
		return nil, err
	}

	// Calculate the one-in inference I^+_ki
	oneInInferences, err := CalcOneInInferences(ctx, k,
		topicId,
		inferenceByWorker,
		forecastImpliedInferenceByWorker,
		allWorkersAreNew,
		currentMaxRegrets.MaxOneInForecastRegret,
		epsilon,
		pInferenceSynthesis,
	)
	if err != nil {
		fmt.Println("Error calculating one-in inferences: ", err)
		return nil, err
	}

	infererValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range inferences.Inferences {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer.Inferer,
			Value:  inferer.Value,
		})
	}

	forecasterValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range forecastImpliedInferenceByWorker {
		forecasterValues = append(forecasterValues, &emissions.WorkerAttributedValue{
			Worker: forecaster.Inferer,
			Value:  forecaster.Value,
		})
	}

	// Build value bundle to return all the calculated inferences
	// Shouldn't need inferences nor forecasts because given from context (input arguments)
	return &emissions.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          combinedNetworkInference,
		InfererValues:          infererValues,
		ForecasterValues:       forecasterValues,
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
	blockHeight BlockHeight,
) (*emissions.ValueBundle, BlockHeight, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, 0, err
	}

	stakesOnTopic, err := k.GetStakePlacementsByTopic(ctx, topicId)
	if err != nil {
		return nil, 0, err
	}

	// Map list of stakesOnTopic to map of stakesByReputer
	stakesByReputer := make(map[string]types.StakePlacement)
	for _, stake := range stakesOnTopic {
		stakesByReputer[stake.Reputer] = stake
	}

	reputerReportedLosses, _, err := k.GetReputerReportedLossesAtOrBeforeBlock(ctx, topicId, blockHeight)
	if err != nil {
		return nil, 0, err
	}

	networkCombinedLoss, err := CalcCombinedNetworkLoss(stakesByReputer, reputerReportedLosses, params.Epsilon)
	if err != nil {
		return nil, 0, err
	}

	inferences, blockHeight, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
	if err != nil {
		return nil, 0, err
	}
	forecasts, _, err := k.GetForecastsAtOrAfterBlock(ctx, topicId, blockHeight)
	if err != nil {
		return nil, 0, err
	}

	networkInferences, err := CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, params.Epsilon, params.PInferenceSynthesis)
	if err != nil {
		return nil, 0, err
	}

	return networkInferences, blockHeight, nil
}
