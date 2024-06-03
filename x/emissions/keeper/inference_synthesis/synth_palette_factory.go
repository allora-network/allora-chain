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
		Ctx:                              req.Ctx,
		K:                                req.K,
		TopicId:                          req.TopicId,
		Inferers:                         sortedInferers,
		InferenceByWorker:                inferenceByWorker,
		InfererRegrets:                   make(map[string]*StatefulRegret), // Populated below
		Forecasters:                      nil,                              // Populated below
		ForecastByWorker:                 forecastByWorker,
		ForecastImpliedInferenceByWorker: nil,                              // Populated below
		ForecasterRegrets:                make(map[string]*StatefulRegret), // Populated below
		AllInferersAreNew:                true,                             // Populated below
		AllForecastersAreNew:             true,                             // Populated below
		AllWorkersAreNew:                 true,                             // Populated below
		NetworkCombinedLoss:              req.NetworkCombinedLoss,
		Epsilon:                          req.Epsilon,
		FTolerance:                       req.FTolerance,
		PNorm:                            req.PNorm,
		CNorm:                            req.CNorm,
	}

	// Populates: forecastImpliedInferenceByWorker, forecasters
	palette.UpdateForecastImpliedInferences()
	palette.Forecasters = alloraMath.GetSortedKeys(palette.ForecastImpliedInferenceByWorker)

	// Populates: infererRegrets, forecasterRegrets, allInferersAreNew, allForecastersAreNew
	palette.BootstrapRegretData()
	palette.AllWorkersAreNew = palette.AllInferersAreNew && palette.AllForecastersAreNew

	return palette
}
