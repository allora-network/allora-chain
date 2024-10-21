package inferencesynthesis

import (
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "inference_synthesis")
}

func CosmosIntOneE18() cosmosMath.Int {
	ret, ok := cosmosMath.NewIntFromString("1000000000000000000")
	if !ok {
		panic("1*10^18 is not a valid cosmos int")
	}
	return ret
}

// Create a map from worker address to their inference or forecast-implied inference
func MakeMapFromInfererToTheirInference(inferences []*emissionstypes.Inference) map[Worker]*emissionstypes.Inference {
	inferencesByWorker := make(map[Worker]*emissionstypes.Inference)
	for _, inference := range inferences {
		inferencesByWorker[inference.Inferer] = inference
	}
	return inferencesByWorker
}

// Create a map from worker address to their inference or forecast-implied inference
func MakeMapFromForecasterToTheirForecast(forecasts []*emissionstypes.Forecast) map[Worker]*emissionstypes.Forecast {
	forecastsByWorker := make(map[Worker]*emissionstypes.Forecast)
	for _, forecast := range forecasts {
		forecastsByWorker[forecast.Forecaster] = forecast
	}
	return forecastsByWorker
}

// It is assumed every key of `weights` is contained within the `workers` slice
func ConvertWeightsToArrays(workers []Worker, weights map[Worker]Weight) []*emissionstypes.RegretInformedWeight {
	weightsArray := make([]*emissionstypes.RegretInformedWeight, 0)
	for _, worker := range workers {
		weightsArray = append(weightsArray, &emissionstypes.RegretInformedWeight{Worker: worker, Weight: weights[worker]})
	}
	return weightsArray
}
