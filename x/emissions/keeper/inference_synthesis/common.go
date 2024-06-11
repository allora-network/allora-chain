package inference_synthesis

import (
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
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

func Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "inference_synthesis")
}
