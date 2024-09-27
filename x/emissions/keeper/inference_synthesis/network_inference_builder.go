package inferencesynthesis

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Calculates the network combined inference I_i, Equation 9
func GetCombinedInference(
	logger log.Logger,
	topicId uint64,
	inferers []Worker,
	infererToInference map[Worker]*emissions.Inference,
	infererToRegret map[Worker]*alloraMath.Dec,
	allInferersAreNew bool,
	forecasters []Worker,
	forecasterToRegret map[Worker]*alloraMath.Dec,
	forecasterToForecastImpliedInference map[Worker]*emissions.Inference,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (
	weights RegretInformedWeights, combinedInference InferenceValue, err error) {
	logger.Debug(fmt.Sprintf("Calculating combined inference for topic %v", topicId))

	weights, err = calcWeightsGivenWorkers(
		logger,
		inferers,
		forecasters,
		infererToRegret,
		forecasterToRegret,
		epsilonTopic,
		pNorm,
		cNorm,
	)
	if err != nil {
		return RegretInformedWeights{}, InferenceValue{}, errorsmod.Wrap(err, "Error calculating weights for combined inference")
	}

	combinedInference, err = calcWeightedInference(
		logger,
		allInferersAreNew,
		inferers,
		infererToInference,
		infererToRegret,
		forecasters,
		forecasterToRegret,
		forecasterToForecastImpliedInference,
		weights,
		epsilonSafeDiv,
	)
	if err != nil {
		return RegretInformedWeights{}, InferenceValue{}, errorsmod.Wrap(err, "Error calculating combined inference")
	}

	logger.Debug(fmt.Sprintf("Combined inference calculated for topic %v is %v", topicId, combinedInference))
	return weights, combinedInference, nil
}

// Map inferences to a WorkerAttributedValue array and set
func getInferences(inferers []Worker, infererToInference map[Worker]*emissions.Inference) (infererValues []*emissions.WorkerAttributedValue) {
	infererValues = make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range inferers {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer,
			Value:  infererToInference[inferer].Value,
		})
	}
	return infererValues
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func getForecastImpliedInferences(
	logger log.Logger,
	forecasters []Worker,
	forecasterToForecastImpliedInference map[Worker]*emissions.Inference,
) (forecastImpliedInferences []*emissions.WorkerAttributedValue) {
	forecastImpliedInferences = make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range forecasters {
		if forecasterToForecastImpliedInference[forecaster] == nil {
			logger.Warn(fmt.Sprintf("No forecast-implied inference for forecaster %s", forecaster))
			continue
		}
		forecastImpliedInferences = append(forecastImpliedInferences, &emissions.WorkerAttributedValue{
			Worker: forecasterToForecastImpliedInference[forecaster].Inferer,
			Value:  forecasterToForecastImpliedInference[forecaster].Value,
		})
	}
	return forecastImpliedInferences
}

// Calculates the network naive inference I^-_i
func GetNaiveInference(
	ctx context.Context,
	logger log.Logger,
	topicId uint64,
	emissionsKeeper emissionskeeper.Keeper,
	inferers []Worker,
	infererToInference map[Worker]*emissions.Inference,
	allInferersAreNew bool,
	forecasters []Worker,
	forecasterToRegret map[Worker]*alloraMath.Dec,
	forecasterToForecastImpliedInference map[Worker]*emissions.Inference,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (naiveInference alloraMath.Dec, err error) {
	logger.Debug(fmt.Sprintf("Calculating naive inference for topic %v", topicId))

	// Get inferer naive regrets
	infererToRegret := make(map[string]*alloraMath.Dec)
	for _, inferer := range inferers {
		regret, _, err := emissionsKeeper.GetNaiveInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting naive regret for inferer %s", inferer)
		}
		infererToRegret[inferer] = &regret.Value
	}

	weights, err := calcWeightsGivenWorkers(
		logger,
		inferers,
		forecasters,
		infererToRegret,
		make(map[Worker]*alloraMath.Dec, 0),
		epsilonTopic,
		pNorm,
		cNorm,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "Error calculating weights for naive inference")
	}

	naiveInference, err = calcWeightedInference(
		logger,
		allInferersAreNew,
		inferers,
		infererToInference,
		infererToRegret,
		forecasters,
		forecasterToRegret,
		forecasterToForecastImpliedInference,
		weights,
		epsilonSafeDiv,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "Error calculating naive inference")
	}

	logger.Debug(fmt.Sprintf("Naive inference calculated for topic %v is %v", topicId, naiveInference))
	return naiveInference, nil
}

// Calculate the one-out inference given a withheld inferer
func calcOneOutInfererInference(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	infererToRegret map[Inferer]*Regret,
	allInferersAreNew bool,
	forecasters []Forecaster,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToRegret map[Forecaster]*Regret,
	networkCombinedLoss alloraMath.Dec,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
	withheldInferer Inferer,
) (
	oneOutNetworkInferenceWithoutInferer alloraMath.Dec,
	err error,
) {
	logger.Debug(fmt.Sprintf(
		"Calculating one-out inference for topic %v withheld inferer %s", topicId, withheldInferer))

	// Remove the inferer from the palette's inferers
	remainingInferers := make([]Worker, 0)
	remainingInfererToInference := make(map[Worker]*emissions.Inference)
	remainingInfererRegrets := make(map[string]*alloraMath.Dec)
	for _, inferer := range inferers {
		// over just the remaining inferers
		if inferer != withheldInferer {
			remainingInferers = append(remainingInferers, inferer)
			inference, ok := infererToInference[inferer]
			if !ok {
				logger.Debug(fmt.Sprintf("Cannot find inferer in InferenceByWorker in UpdateInferersInfo %v", inferer))
				continue
			}
			remainingInfererToInference[inferer] = inference
		}

		//over every inferer
		regret, _, err := k.GetOneOutInfererInfererNetworkRegret(ctx, topicId, withheldInferer, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out inferer regret")
		}
		remainingInfererRegrets[inferer] = &regret.Value
	}

	// Recalculate the forecast-implied inferences without the worker's inference
	// This is necessary because the forecast-implied inferences are calculated based on the inferences of the inferers
	forecasterToForecastImpliedInference, _, _, err := CalcForecastImpliedInferences(
		logger,
		topicId,
		allInferersAreNew,
		remainingInferers,
		infererToInference,
		infererToRegret,
		forecasters,
		forecasterToForecast,
		forecasterToRegret,
		networkCombinedLoss,
		epsilonTopic,
		pNorm,
		cNorm,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error recalculating forecast-implied inferences")
	}

	// Get regrets for the forecasters
	remainingForecasterRegrets := make(map[string]*alloraMath.Dec)
	for _, forecaster := range forecasters {
		regret, _, err := k.GetOneOutInfererForecasterNetworkRegret(ctx, topicId, withheldInferer, forecaster)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out forecaster regret")
		}
		remainingForecasterRegrets[forecaster] = &regret.Value
	}

	weights, err := calcWeightsGivenWorkers(
		logger,
		remainingInferers,
		forecasters,
		remainingInfererRegrets,
		remainingForecasterRegrets,
		epsilonTopic,
		pNorm,
		cNorm,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err = calcWeightedInference(
		logger,
		allInferersAreNew,
		remainingInferers,
		remainingInfererToInference,
		remainingInfererRegrets,
		forecasters,
		remainingForecasterRegrets,
		forecasterToForecastImpliedInference,
		weights,
		epsilonSafeDiv,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	logger.Debug(fmt.Sprintf("One-out inference calculated for topic %v withheld inferer %s is %v", topicId, withheldInferer, oneOutNetworkInferenceWithoutInferer))
	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withhold one, then calculate the network inference less that withheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func GetOneOutInfererInferences(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	infererToRegret map[Inferer]*Regret,
	allInferersAreNew bool,
	forecasters []Forecaster,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToRegret map[Forecaster]*Regret,
	networkCombinedLoss alloraMath.Dec,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (
	oneOutInfererInferences []*emissions.WithheldWorkerAttributedValue,
	err error,
) {
	logger.Debug(fmt.Sprintf(
		"Calculating one-out inferer inferences for topic %v with %v inferers",
		topicId,
		len(inferers),
	))
	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range inferers {
		oneOutInference, err := calcOneOutInfererInference(
			ctx,
			k,
			logger,
			topicId,
			inferers,
			infererToInference,
			infererToRegret,
			allInferersAreNew,
			forecasters,
			forecasterToForecast,
			forecasterToRegret,
			networkCombinedLoss,
			epsilonTopic,
			epsilonSafeDiv,
			pNorm,
			cNorm,
			worker,
		)
		if err != nil {
			return []*emissions.WithheldWorkerAttributedValue{}, errorsmod.Wrapf(err, "Error calculating one-out inferer inferences")
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	logger.Debug(fmt.Sprintf("One-out inferer inferences calculated for topic %v", topicId))
	return oneOutInferences, nil
}

// Calculate the one-out inference given a withheld forecaster
func calcOneOutForecasterInference(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	infererToRegret map[Inferer]*Regret,
	allInferersAreNew bool,
	forecasters []Forecaster,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToForecastImpliedInference map[Forecaster]*emissions.Inference,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
	withheldForecaster Forecaster,
) (
	oneOutNetworkInferenceWithoutInferer alloraMath.Dec,
	err error,
) {
	logger.Debug(fmt.Sprintf("Calculating one-out inference for topic %v withheld forecaster %s", topicId, withheldForecaster))

	// Remove the withheldForecaster from the palette's forecasters
	remainingForecasters := make([]Forecaster, 0)
	remainingForecasterToForecast := make(map[Forecaster]*emissions.Forecast)
	remainingForecasterRegrets := make(map[Forecaster]*Regret)
	for _, forecaster := range forecasters {
		if forecaster != withheldForecaster {
			remainingForecasters = append(remainingForecasters, forecaster)

			regret, _, err := k.GetOneOutForecasterForecasterNetworkRegret(ctx, topicId, withheldForecaster, forecaster)
			if err != nil {
				return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out forecaster regret")
			}
			remainingForecasterRegrets[forecaster] = &regret.Value

			forecast, ok := forecasterToForecast[forecaster]
			if !ok {
				logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in UpdateForecastersInfo %v", forecaster))
				continue
			}
			remainingForecasterToForecast[forecaster] = forecast
		}
	}

	// Get regrets for the remaining inferers
	remainingInfererRegrets := make(map[Inferer]*Regret)
	for _, inferer := range inferers {
		regret, _, err := k.GetOneOutForecasterInfererNetworkRegret(ctx, topicId, withheldForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out inferer regret")
		}
		remainingInfererRegrets[inferer] = &regret.Value
	}

	weights, err := calcWeightsGivenWorkers(
		logger,
		inferers,
		remainingForecasters,
		remainingInfererRegrets,
		remainingForecasterRegrets,
		epsilonTopic,
		pNorm,
		cNorm,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err = calcWeightedInference(
		logger,
		allInferersAreNew,
		inferers,
		infererToInference,
		infererToRegret,
		remainingForecasters,
		remainingForecasterRegrets,
		forecasterToForecastImpliedInference,
		weights,
		epsilonSafeDiv,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	logger.Debug(fmt.Sprintf("One-out inference calculated for topic %v withheld forecaster %s is %v", topicId, withheldForecaster, oneOutNetworkInferenceWithoutInferer))
	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withhold one, then calculate the network inference less that withheld value
func GetOneOutForecasterInferences(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	infererToRegret map[Inferer]*Regret,
	allInferersAreNew bool,
	forecasters []Forecaster,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToRegret map[Forecaster]*Regret,
	forecasterToForecastImpliedInference map[Forecaster]*emissions.Inference,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (
	oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue, err error) {
	logger.Debug(fmt.Sprintf("Calculating one-out forecaster inferences for topic %v with %v forecasters", topicId, len(forecasters)))
	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutForecasterInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
	// If there is only one forecaster, there's no need to calculate one-out inferences
	if len(forecasters) > 1 {
		for _, worker := range forecasters {
			oneOutInference, err := calcOneOutForecasterInference(
				ctx,
				k,
				logger,
				topicId,
				inferers,
				infererToInference,
				infererToRegret,
				allInferersAreNew,
				forecasters,
				forecasterToForecast,
				forecasterToForecastImpliedInference,
				epsilonTopic,
				epsilonSafeDiv,
				pNorm,
				cNorm,
				worker,
			)
			if err != nil {
				return []*emissions.WithheldWorkerAttributedValue{}, errorsmod.Wrapf(err, "Error calculating one-out forecaster inferences")
			}
			oneOutForecasterInferences = append(oneOutForecasterInferences, &emissions.WithheldWorkerAttributedValue{
				Worker: worker,
				Value:  oneOutInference,
			})
		}
		logger.Debug(fmt.Sprintf("One-out forecaster inferences calculated for topic %v", topicId))
	}
	return oneOutForecasterInferences, nil
}

// Calculate the one-in inference given a withheld forecaster
func calcOneInValue(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	allInferersAreNew bool,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToForecastImpliedInference map[Forecaster]*emissions.Inference,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
	oneInForecaster Forecaster,
) (
	oneInInference alloraMath.Dec,
	err error,
) {
	logger.Debug(fmt.Sprintf("Calculating one-in inference for forecaster: %s", oneInForecaster))

	// In each loop, remove all forecast-implied inferences except one
	singleForecastImpliedInference := make(map[Worker]*emissions.Inference, 1)
	singleForecastImpliedInference[oneInForecaster] = forecasterToForecastImpliedInference[oneInForecaster]

	// Get self regret for the forecaster
	singleForecasterRegret := make(map[Worker]*Regret, 1)
	regret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, oneInForecaster, oneInForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
	}
	singleForecasterRegret[oneInForecaster] = &regret.Value

	// get self forecast list
	singleForecaster := []Worker{oneInForecaster}

	// get map of Forecaster to their forecast for the single forecaster
	singleForecasterToForecast := make(map[Forecaster]*emissions.Forecast, 1)
	singleForecasterToForecast[oneInForecaster] = forecasterToForecast[oneInForecaster]

	// Get one-in regrets for the forecaster and the inferers they provided forecasts for
	infererToRegretForSingleForecaster := make(map[Inferer]*Regret)
	infererToInferenceForSingleForecaster := make(map[Inferer]*emissions.Inference)
	for _, inferer := range inferers {
		regret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, oneInForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Calc one in value error getting one-in forecaster regret")
		}
		infererToRegretForSingleForecaster[inferer] = &regret.Value

		inference, ok := infererToInference[inferer]
		if !ok {
			logger.Debug(fmt.Sprintf("Calc one in value cannot find inferer in InferenceByWorker %v", inferer))
			continue
		}
		infererToInferenceForSingleForecaster[inferer] = inference
	}

	weights, err := calcWeightsGivenWorkers(
		logger,
		inferers,
		singleForecaster,
		infererToRegretForSingleForecaster,
		singleForecasterRegret,
		epsilonTopic,
		pNorm,
		cNorm,
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating weights for one-in inferences")
	}
	// Calculate the network inference with just this forecaster's forecast-implied inference
	oneInInference, err = calcWeightedInference(
		logger,
		allInferersAreNew,
		inferers,
		infererToInferenceForSingleForecaster,
		infererToRegretForSingleForecaster,
		singleForecaster,
		singleForecasterRegret,
		forecasterToForecastImpliedInference,
		weights,
		epsilonSafeDiv)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-in inference")
	}

	return oneInInference, nil
}

// Set all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker.
// Also assume that there is at most 1 forecast-implied inference per worker.
func GetOneInForecasterInferences(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	allInferersAreNew bool,
	forecasters []Forecaster,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToForecastImpliedInference map[Forecaster]*emissions.Inference,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (
	oneInInferences []*emissions.WorkerAttributedValue,
	err error,
) {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference
	// one at a time, then calculate the network inference given that one held out
	oneInInferences = make([]*emissions.WorkerAttributedValue, 0)
	// If there is only one forecaster, thre's no need to calculate one-in inferences
	if len(forecasters) > 1 {
		for _, oneInForecaster := range forecasters {
			oneInValue, err := calcOneInValue(
				ctx,
				k,
				logger,
				topicId,
				allInferersAreNew,
				inferers,
				infererToInference,
				forecasterToForecast,
				forecasterToForecastImpliedInference,
				epsilonTopic,
				epsilonSafeDiv,
				pNorm,
				cNorm,
				oneInForecaster,
			)
			if err != nil {
				return []*emissions.WorkerAttributedValue{}, errorsmod.Wrapf(err, "Error calculating one-in inferences")
			}
			oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
				Worker: oneInForecaster,
				Value:  oneInValue,
			})
		}
	}
	return oneInInferences, err
}

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences).
// Could improve this with Builder pattern, as for other instances of generated ValueBundles.
func CalcNetworkInferences(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferers []Inferer,
	infererToInference map[Inferer]*emissions.Inference,
	infererToRegret map[Inferer]*Regret,
	allInferersAreNew bool,
	forecasters []Forecaster,
	forecasterToForecast map[Forecaster]*emissions.Forecast,
	forecasterToRegret map[Forecaster]*Regret,
	forecasterToForecastImpliedInference map[Forecaster]*emissions.Inference,
	networkCombinedLoss alloraMath.Dec,
	epsilonTopic alloraMath.Dec,
	epsilonSafeDiv alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (
	inferenceBundle *emissions.ValueBundle,
	weights RegretInformedWeights,
	err error,
) {
	// first get the network combined inference I_i
	// which is the end result of all this work, the actual combined
	// inference from all of the inferers put together
	weights, combinedInference, err := GetCombinedInference(
		logger,
		topicId,
		inferers,
		infererToInference,
		infererToRegret,
		allInferersAreNew,
		forecasters,
		forecasterToRegret,
		forecasterToForecastImpliedInference,
		epsilonTopic,
		epsilonSafeDiv,
		pNorm,
		cNorm,
	)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating combined inference")
	}
	// get all the inferences which is all I_ij
	inferences := getInferences(inferers, infererToInference)
	// get all the forecast-implied inferences which is all I_ik
	forecastImpliedInferences := getForecastImpliedInferences(
		logger,
		forecasters,
		forecasterToForecastImpliedInference,
	)
	// get the naive network inference I^-_i
	// The naive network inference is used to quantify the contribution of the
	// forecasting task to the network accuracy, which in turn sets the reward
	// distribution between the inference and forecasting tasks
	naiveInference, err := GetNaiveInference(
		ctx,
		logger,
		topicId,
		k,
		inferers,
		infererToInference,
		allInferersAreNew,
		forecasters,
		forecasterToRegret,
		forecasterToForecastImpliedInference,
		epsilonTopic,
		epsilonSafeDiv,
		pNorm,
		cNorm,
	)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating naive inference")
	}
	// Get the one-out inferer inferences I^-_li over I_ij
	// The one-out network inferer inferences represent an approximation of
	// Shapley (1953) values and are used to quantify the individual
	// contributions of inferers to the network accuracy,
	// which in turn sets the reward distribution between inferers.
	// The one-out network inferer inferences are also used to
	// calculate confidence intervals on the network inference I_i
	var oneOutInfererInferences []*emissions.WithheldWorkerAttributedValue
	oneOutInfererInferences, err = GetOneOutInfererInferences(
		ctx,
		k,
		logger,
		topicId,
		inferers,
		infererToInference,
		infererToRegret,
		allInferersAreNew,
		forecasters,
		forecasterToForecast,
		forecasterToRegret,
		networkCombinedLoss,
		epsilonTopic,
		epsilonSafeDiv,
		pNorm,
		cNorm,
	)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating one-out inferer inferences")
	}
	// get the one-out forecaster inferences I^-_li over I_ik
	// The one-out network forecaster inferences represent an approximation of
	// Shapley (1953) values and are used to quantify the individual
	// contributions of forecasters to the network accuracy,
	// which in turn sets the reward distribution between forecasters.
	// The one-out network forecaster inferences are also used to
	// calculate confidence intervals on the network inference I_i
	oneOutForecasterInferences, err := GetOneOutForecasterInferences(
		ctx,
		k,
		logger,
		topicId,
		inferers,
		infererToInference,
		infererToRegret,
		allInferersAreNew,
		forecasters,
		forecasterToForecast,
		forecasterToRegret,
		forecasterToForecastImpliedInference,
		epsilonTopic,
		epsilonSafeDiv,
		pNorm,
		cNorm,
	)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating one-out forecaster inferences")
	}
	// get the one-in forecaster inferences I^+_ki
	// which adds only a single forecast-implied inference I_ik to the inferences
	// from the inference task I_ij . As such, it is used to quantify how the naive
	// network inference I^-_i changes with the addition of a single
	// forecast-implied inference, which in turn is used for setting the
	// reward distribution between workers for their forecasting tasks.
	oneInForecasterInferences, err := GetOneInForecasterInferences(
		ctx,
		k,
		logger,
		topicId,
		inferers,
		infererToInference,
		allInferersAreNew,
		forecasters,
		forecasterToForecast,
		forecasterToForecastImpliedInference,
		epsilonTopic,
		epsilonSafeDiv,
		pNorm,
		cNorm,
	)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating one-in inferences")
	}

	// Build value bundle to return all the calculated inferences
	return &emissions.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          combinedInference,
		InfererValues:          inferences,
		ForecasterValues:       forecastImpliedInferences,
		NaiveValue:             naiveInference,
		OneOutInfererValues:    oneOutInfererInferences,
		OneOutForecasterValues: oneOutForecasterInferences,
		OneInForecasterValues:  oneInForecasterInferences,
	}, weights, err
}
