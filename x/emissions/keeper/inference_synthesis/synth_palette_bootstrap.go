package inferencesynthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Bootstraps xRegrets, allxsAreNew (x="inferer"|"forecasters") for the inferers and forecasters in the palette
// Just requires these props:: ctx, k, topicId, inferers, forecasts
func (p *SynthPalette) BootstrapRegretData() error {
	p.Logger.Debug(fmt.Sprintf("Bootstrapping regret data for topic %v", p.TopicId))

	for _, inferer := range p.Inferers {
		regret, _, err := p.K.GetInfererNetworkRegret(p.Ctx, p.TopicId, inferer)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting inferer regret")
		}

		p.Logger.Debug(fmt.Sprintf("Inferer %v has regret %v", inferer, regret.Value))
		p.InfererRegrets[inferer] = &regret.Value
	}

	for _, forecaster := range p.Forecasters {
		regret, _, err := p.K.GetForecasterNetworkRegret(p.Ctx, p.TopicId, forecaster)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting forecaster regret")
		}

		p.Logger.Debug(fmt.Sprintf("Forecaster %v has regret %v", forecaster, regret.Value))
		p.ForecasterRegrets[forecaster] = &regret.Value
	}

	return nil
}

// Clone creates a deep copy of the SynthPalette.
func (p SynthPalette) Clone() SynthPalette {
	inferenceByWorker := make(map[Worker]*emissionstypes.Inference, len(p.InferenceByWorker))
	for _, worker := range p.Inferers {
		data, ok := p.InferenceByWorker[worker]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in InferenceByWorker in palette.Clone %v", worker))
			continue
		}
		inferenceCopy := *data
		inferenceByWorker[worker] = &inferenceCopy
	}
	forecastByWorker := make(map[Worker]*emissionstypes.Forecast, len(p.ForecastByWorker))
	for _, worker := range p.Forecasters {
		data, ok := p.ForecastByWorker[worker]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecastByWorker in palette.Clone %v", worker))
			continue
		}
		forecastCopy := *data
		forecastByWorker[worker] = &forecastCopy
	}
	forecastImpliedInferenceByWorker := make(map[Worker]*emissionstypes.Inference, len(p.ForecastImpliedInferenceByWorker))
	for _, worker := range p.Forecasters {
		data, ok := p.ForecastImpliedInferenceByWorker[worker]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecastImpliedInferenceByWorker in palette.Clone %v", worker))
			continue
		}
		inferenceCopy := *data
		forecastImpliedInferenceByWorker[worker] = &inferenceCopy
	}
	infererRegrets := make(map[Worker]*alloraMath.Dec, len(p.InfererRegrets))
	for _, worker := range p.Inferers {
		data, ok := p.InfererRegrets[worker]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in InfererRegrets in palette.Clone %v", worker))
			continue
		}
		regretCopy := *data
		infererRegrets[worker] = &regretCopy
	}
	forecasterRegrets := make(map[Worker]*alloraMath.Dec, len(p.ForecasterRegrets))
	for _, worker := range p.Forecasters {
		data, ok := p.ForecasterRegrets[worker]
		if !ok {
			p.Logger.Debug(fmt.Sprintf("Cannot find forecaster in ForecasterRegrets in palette.Clone %v", worker))
			continue
		}
		regretCopy := *data
		forecasterRegrets[worker] = &regretCopy
	}

	return SynthPalette{
		Ctx:                              p.Ctx,
		K:                                p.K,
		Logger:                           p.Logger,
		Nonce:                            p.Nonce,
		TopicId:                          p.TopicId,
		Inferers:                         append([]Worker(nil), p.Inferers...),
		InferenceByWorker:                inferenceByWorker,
		InfererRegrets:                   infererRegrets,
		Forecasters:                      append([]Worker(nil), p.Forecasters...),
		ForecastByWorker:                 forecastByWorker,
		ForecastImpliedInferenceByWorker: forecastImpliedInferenceByWorker,
		ForecasterRegrets:                forecasterRegrets,
		NetworkCombinedLoss:              p.NetworkCombinedLoss,
		EpsilonTopic:                     p.EpsilonTopic,
		EpsilonSafeDiv:                   p.EpsilonSafeDiv,
		PNorm:                            p.PNorm,
		CNorm:                            p.CNorm,
		AllInferersAreNew:                p.AllInferersAreNew,
	}
}
