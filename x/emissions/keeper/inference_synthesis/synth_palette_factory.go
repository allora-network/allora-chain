package inference_synthesis

import (
	alloraMath "github.com/allora-network/allora-chain/math"
)

// Could use Builder pattern in the future to make this cleaner
func (f *SynthPaletteFactory) BuildPaletteFromRequest(req SynthRequest) SynthPalette {
	inferenceByWorker := MakeMapFromInfererToTheirInference(req.Inferences.Inferences)
	forecastByWorker := MakeMapFromForecasterToTheirForecast(req.Forecasts.Forecasts)
	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	// Those values not from req are to be considered defaults
	palette := SynthPalette{
		ctx:                              req.Ctx,
		k:                                req.K,
		topicId:                          req.TopicId,
		inferers:                         sortedInferers,
		inferenceByWorker:                inferenceByWorker,
		infererRegrets:                   make(map[string]StatefulRegret), // Populated below
		forecasters:                      nil,                             // Populated below
		forecastByWorker:                 forecastByWorker,
		forecastImpliedInferenceByWorker: nil,                             // Populated below
		forecasterRegrets:                make(map[string]StatefulRegret), // Populated below
		allInferersAreNew:                true,                            // Populated below
		allForecastersAreNew:             true,                            // Populated below
		allWorkersAreNew:                 true,                            // Populated below
		networkCombinedLoss:              req.NetworkCombinedLoss,
		epsilon:                          req.Epsilon,
		pNorm:                            req.PNorm,
		cNorm:                            req.CNorm,
	}

	// Populates: forecastImpliedInferenceByWorker, forecasters
	palette.UpdateForecastImpliedInferences()
	palette.forecasters = alloraMath.GetSortedKeys(palette.forecastImpliedInferenceByWorker)

	// Populates: infererRegrets, forecasterRegrets, allInferersAreNew, allForecastersAreNew
	palette.BootstrapRegretData()
	palette.allWorkersAreNew = palette.allInferersAreNew && palette.allForecastersAreNew

	return palette
}
