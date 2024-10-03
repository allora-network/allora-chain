package inferencesynthesis

import (
	alloraMath "github.com/allora-network/allora-chain/math"
)

// Could use Builder pattern in the future to make this cleaner
func (f *SynthPaletteFactory) BuildPaletteFromRequest(req SynthRequest) (SynthPalette, error) {
	inferenceByWorker := MakeMapFromInfererToTheirInference(req.Inferences.Inferences)
	forecastByWorker := MakeMapFromForecasterToTheirForecast(req.Forecasts.Forecasts)
	sortedInferers := alloraMath.GetSortedKeys(inferenceByWorker)
	sortedForecasters := alloraMath.GetSortedKeys(forecastByWorker)

	topic, err := req.K.GetTopic(req.Ctx, req.TopicId)
	if err != nil {
		return SynthPalette{}, err
	}

	// Those values not from req are to be considered defaults
	palette := SynthPalette{
		Ctx:                              req.Ctx,
		K:                                req.K,
		Logger:                           Logger(req.Ctx),
		TopicId:                          req.TopicId,
		Nonce:                            *req.Nonce,
		AllInferersAreNew:                topic.InitialRegret.Equal(alloraMath.ZeroDec()), // If initial regret is 0, all inferers are new
		Inferers:                         sortedInferers,
		InferenceByWorker:                inferenceByWorker,
		InfererRegrets:                   make(map[string]*alloraMath.Dec), // Populated below
		Forecasters:                      sortedForecasters,
		ForecastByWorker:                 forecastByWorker,
		ForecastImpliedInferenceByWorker: nil,                              // Populated below
		ForecasterRegrets:                make(map[string]*alloraMath.Dec), // Populated below
		NetworkCombinedLoss:              req.NetworkCombinedLoss,
		EpsilonTopic:                     req.EpsilonTopic,
		EpsilonSafeDiv:                   req.EpsilonSafeDiv,
		PNorm:                            req.PNorm,
		CNorm:                            req.CNorm,
	}

	// Populates: infererRegrets, forecasterRegrets, allInferersAreNew
	err = palette.BootstrapRegretData()
	if err != nil {
		return SynthPalette{}, err
	}

	paletteCopy := palette.Clone()
	// Populates: forecastImpliedInferenceByWorker,
	err = paletteCopy.UpdateForecastImpliedInferences()
	if err != nil {
		return SynthPalette{}, err
	}
	palette.ForecastImpliedInferenceByWorker = paletteCopy.ForecastImpliedInferenceByWorker

	return palette, nil
}
