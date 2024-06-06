package inference_synthesis

import (
	errorsmod "cosmossdk.io/errors"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Bootstraps xRegrets, allxsAreNew (x="inferer"|"forecasters") for the inferers and forecasters in the palette
// Just requires these props:: ctx, k, topicId, inferers, forecasts
func (p *SynthPalette) BootstrapRegretData() error {
	p.InferersNewStatus = InferersAllNew

	for _, inferer := range p.Inferers {
		regret, noPriorRegret, err := p.K.GetInfererNetworkRegret(p.Ctx, p.TopicId, inferer)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting inferer regret")
		}

		if !noPriorRegret {
			if p.InferersNewStatus == InferersAllNew {
				p.InferersNewStatus = InferersAllNewExceptOne
				p.SingleNotNewInferer = inferer
			} else {
				p.InferersNewStatus = InferersNotNew
				p.SingleNotNewInferer = ""
			}
		}

		p.InfererRegrets[inferer] = &StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}

	for _, forecaster := range p.Forecasters {
		regret, noPriorRegret, err := p.K.GetForecasterNetworkRegret(p.Ctx, p.TopicId, forecaster)
		if err != nil {
			return errorsmod.Wrapf(err, "Error getting forecaster regret")
		}

		p.ForecasterRegrets[forecaster] = &StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}

	return nil
}

// Clone creates a deep copy of the SynthPalette.
func (p SynthPalette) Clone() SynthPalette {
	inferenceByWorker := make(map[Worker]*emissionstypes.Inference, len(p.InferenceByWorker))
	for k, v := range p.InferenceByWorker {
		inferenceCopy := *v
		inferenceByWorker[k] = &inferenceCopy
	}
	forecastByWorker := make(map[Worker]*emissionstypes.Forecast, len(p.ForecastByWorker))
	for k, v := range p.ForecastByWorker {
		forecastCopy := *v
		forecastByWorker[k] = &forecastCopy
	}
	forecastImpliedInferenceByWorker := make(map[Worker]*emissionstypes.Inference, len(p.ForecastImpliedInferenceByWorker))
	for k, v := range p.ForecastImpliedInferenceByWorker {
		inferenceCopy := *v
		forecastImpliedInferenceByWorker[k] = &inferenceCopy
	}
	infererRegrets := make(map[Worker]*StatefulRegret, len(p.InfererRegrets))
	for k, v := range p.InfererRegrets {
		regretCopy := *v
		infererRegrets[k] = &regretCopy
	}
	forecasterRegrets := make(map[Worker]*StatefulRegret, len(p.ForecasterRegrets))
	for k, v := range p.ForecasterRegrets {
		regretCopy := *v
		forecasterRegrets[k] = &regretCopy
	}

	return SynthPalette{
		Ctx:                              p.Ctx,
		K:                                p.K,
		TopicId:                          p.TopicId,
		Inferers:                         append([]Worker(nil), p.Inferers...),
		InferenceByWorker:                inferenceByWorker,
		InfererRegrets:                   infererRegrets,
		Forecasters:                      append([]Worker(nil), p.Forecasters...),
		ForecastByWorker:                 forecastByWorker,
		ForecastImpliedInferenceByWorker: forecastImpliedInferenceByWorker,
		ForecasterRegrets:                forecasterRegrets,
		InferersNewStatus:                p.InferersNewStatus,
		SingleNotNewInferer:              p.SingleNotNewInferer,
		NetworkCombinedLoss:              p.NetworkCombinedLoss,
		Epsilon:                          p.Epsilon,
		FTolerance:                       p.FTolerance,
		PNorm:                            p.PNorm,
		CNorm:                            p.CNorm,
	}
}
