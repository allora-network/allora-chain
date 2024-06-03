package inference_synthesis

import (
	errorsmod "cosmossdk.io/errors"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Bootstraps xRegrets, allxsAreNew (x="inferer"|"forecasters") for the inferers and forecasters in the palette
// Just requires these props:: ctx, k, topicId, inferers, forecasts
func (p *SynthPalette) BootstrapRegretData() error {
	p.allInferersAreNew = true
	for _, inferer := range p.inferers {
		regret, noPriorRegret, err := p.k.GetInfererNetworkRegret(p.ctx, p.topicId, inferer)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting inferer regret")
		}

		p.allInferersAreNew = p.allInferersAreNew && noPriorRegret
		p.infererRegrets[inferer] = StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}

	p.allForecastersAreNew = true
	for _, forecaster := range p.forecasters {
		regret, noPriorRegret, err := p.k.GetForecasterNetworkRegret(p.ctx, p.topicId, forecaster)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting forecaster regret")
		}

		p.allForecastersAreNew = p.allForecastersAreNew && noPriorRegret
		p.forecasterRegrets[forecaster] = StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}

	return nil
}

// Clone creates a deep copy of the SynthPalette.
func (p SynthPalette) Clone() SynthPalette {
	inferenceByWorker := make(map[Worker]*emissionstypes.Inference, len(p.inferenceByWorker))
	for k, v := range p.inferenceByWorker {
		inferenceByWorker[k] = v
	}
	forecastByWorker := make(map[Worker]*emissionstypes.Forecast, len(p.forecastByWorker))
	for k, v := range p.forecastByWorker {
		forecastByWorker[k] = v
	}
	forecastImpliedInferenceByWorker := make(map[Worker]*emissionstypes.Inference, len(p.forecastImpliedInferenceByWorker))
	for k, v := range p.forecastImpliedInferenceByWorker {
		inferenceCopy := *v
		forecastImpliedInferenceByWorker[k] = &inferenceCopy
	}
	infererRegrets := make(map[Worker]StatefulRegret, len(p.infererRegrets))
	for k, v := range p.infererRegrets {
		infererRegrets[k] = v
	}
	forecasterRegrets := make(map[Worker]StatefulRegret, len(p.forecasterRegrets))
	for k, v := range p.forecasterRegrets {
		forecasterRegrets[k] = v
	}

	return SynthPalette{
		ctx:                              p.ctx,
		k:                                p.k,
		topicId:                          p.topicId,
		inferers:                         append([]Worker(nil), p.inferers...),
		inferenceByWorker:                inferenceByWorker,
		infererRegrets:                   infererRegrets,
		forecasters:                      append([]Worker(nil), p.forecasters...),
		forecastByWorker:                 forecastByWorker,
		forecastImpliedInferenceByWorker: forecastImpliedInferenceByWorker,
		forecasterRegrets:                forecasterRegrets,
		allInferersAreNew:                p.allInferersAreNew,
		allForecastersAreNew:             p.allForecastersAreNew,
		allWorkersAreNew:                 p.allWorkersAreNew,
		networkCombinedLoss:              p.networkCombinedLoss,
		epsilon:                          p.epsilon,
		pNorm:                            p.pNorm,
		cNorm:                            p.cNorm,
	}
}
