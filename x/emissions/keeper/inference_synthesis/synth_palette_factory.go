package inference_synthesis

import (
	alloraMath "github.com/allora-network/allora-chain/math"
)

// Could use Builder pattern in the future to make this cleaner
func (f *SynthPaletteFactory) BuildPaletteFromRequest(req SynthRequest) SynthPalette {
	inferenceByWorker := MakeMapFromInfererToTheirInference(req.inferences.Inferences)
	forecastByWorker := MakeMapFromForecasterToTheirForecast(req.forecasts.Forecasts)
	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)

	// Those values not from req are to be considered defaults
	palette := SynthPalette{
		ctx:                              req.ctx,
		k:                                req.k,
		topicId:                          req.topicId,
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
		networkCombinedLoss:              req.networkCombinedLoss,
		epsilon:                          req.epsilon,
		pNorm:                            req.pNorm,
		cNorm:                            req.cNorm,
	}

	// Populates: infererRegrets, forecasterRegrets, allInferersAreNew, allForecastersAreNew
	palette.BootstrapRegretData()
	palette.allWorkersAreNew = palette.allInferersAreNew && palette.allForecastersAreNew

	// Populates: forecastImpliedInferenceByWorker, forecasters
	palette.UpdateForecastImpliedInferences()
	palette.forecasters = alloraMath.GetSortedKeys(palette.forecastImpliedInferenceByWorker)

	return palette
}
