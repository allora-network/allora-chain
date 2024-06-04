package inference_synthesis

import (
	alloraMath "github.com/allora-network/allora-chain/math"
)

// Could use Builder pattern in the future to make this cleaner
func (f *SynthPaletteFactory) BuildPaletteFromRequest(req SynthRequest) (SynthPalette, error) {
	inferenceByWorker := MakeMapFromInfererToTheirInference(req.Inferences.Inferences)
	forecastByWorker := MakeMapFromForecasterToTheirForecast(req.Forecasts.Forecasts)
	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)
	sortedForecasters := alloraMath.GetSortedKeys(forecastByWorker)

	// Those values not from req are to be considered defaults
	palette := SynthPalette{
		Ctx:                              req.Ctx,
		K:                                req.K,
		TopicId:                          req.TopicId,
		Inferers:                         sortedInferers,
		InferenceByWorker:                inferenceByWorker,
		InfererRegrets:                   make(map[string]StatefulRegret), // Populated below
		Forecasters:                      sortedForecasters,
		ForecastByWorker:                 forecastByWorker,
		ForecastImpliedInferenceByWorker: nil,                             // Populated below
		ForecasterRegrets:                make(map[string]StatefulRegret), // Populated below
		AllInferersAreNew:                true,                            // Populated below
		AllForecastersAreNew:             true,                            // Populated below
		AllWorkersAreNew:                 true,                            // Populated below
		NetworkCombinedLoss:              req.NetworkCombinedLoss,
		Epsilon:                          req.Epsilon,
		FTolerance:                       req.FTolerance,
		PNorm:                            req.PNorm,
		CNorm:                            req.CNorm,
	}

	// Populates: infererRegrets, forecasterRegrets, allInferersAreNew, allForecastersAreNew
	palette.BootstrapRegretData()
	palette.AllWorkersAreNew = palette.AllInferersAreNew && palette.AllForecastersAreNew

	// Populates: forecastImpliedInferenceByWorker,
	err := palette.UpdateForecastImpliedInferences()
	if err != nil {
		return SynthPalette{}, err
	}

	return palette, nil
}
