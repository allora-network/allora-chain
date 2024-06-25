package inference_synthesis

import (
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CosmosIntOneE18() cosmosMath.Int {
	ret, ok := cosmosMath.NewIntFromString("1000000000000000000")
	if !ok {
		panic("1*10^18 is not a valid cosmos int")
	}
	return ret
}

// Create a map from worker address to their inference or forecast-implied inference
func MakeMapFromInfererToTheirInference(inferences []*emissions.Inference) map[Worker]*emissions.Inference {
	inferencesByWorker := make(map[Worker]*emissions.Inference)
	for _, inference := range inferences {
		inferencesByWorker[inference.Inferer] = inference
	}
	return inferencesByWorker
}

// Create a map from worker address to their inference or forecast-implied inference
func MakeMapFromForecasterToTheirForecast(forecasts []*emissions.Forecast) map[Worker]*emissions.Forecast {
	forecastsByWorker := make(map[Worker]*emissions.Forecast)
	for _, forecast := range forecasts {
		forecastsByWorker[forecast.Forecaster] = forecast
	}
	return forecastsByWorker
}

func ConvertWeightsToArrays(weights map[Worker]Weight) []*types.RegretInformedWeight {
	weightsArray := make([]*types.RegretInformedWeight, 0)
	for worker, weight := range weights {
		weightsArray = append(weightsArray, &types.RegretInformedWeight{Worker: worker, Weight: weight})
	}
	return weightsArray
}

func ConvertForecastImpliedInferencesToArrays(
	forecastImpliedInferenceByWorker map[string]*types.Inference,
) []*types.WorkerAttributedValue {
	forecastImpliedInferences := make([]*types.WorkerAttributedValue, 0)
	for worker, inference := range forecastImpliedInferenceByWorker {
		forecastImpliedInferences = append(forecastImpliedInferences, &types.WorkerAttributedValue{Worker: worker, Value: inference.Value})
	}
	return forecastImpliedInferences
}

func Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "inference_synthesis")
}
