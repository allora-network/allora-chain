package inference_synthesis

import errorsmod "cosmossdk.io/errors"

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
