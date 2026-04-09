package inferencesynthesis

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// CalcForecastImpliedInferencesArgs holds inputs for CalcForecastImpliedInferences.
type CalcForecastImpliedInferencesArgs struct {
	Logger                 log.Logger
	TopicId                emissionstypes.TopicId
	TopicArity             emissionstypes.TopicOutputArity
	AllInferersAreNew      bool
	Inferers               []Inferer
	InfererToInference     map[Inferer]*emissionstypes.Inference
	InfererToRegret        map[Inferer]*Regret
	Forecasters            []Forecaster
	ForecasterToForecast   map[Forecaster]*emissionstypes.Forecast
	ForecasterToRegret     map[Forecaster]*Regret
	NetworkCombinedLoss    *alloraMath.Dec
	EpsilonTopic           alloraMath.Dec
	PNorm                  alloraMath.Dec
	CNorm                  alloraMath.Dec
	RegretScalePlusEpsilon alloraMath.Dec
	LabelRegistry          *emissionstypes.EpochLabelRegistry
	NumLabels              int
}

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forecast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
func CalcForecastImpliedInferences(args CalcForecastImpliedInferencesArgs) (map[Forecaster]*emissionstypes.Inference, error) {
	args.Logger.Debug("Calculating forecast-implied inferences", "topicId", args.TopicId)

	// If NetworkCombinedLoss is nil, return empty maps immediately
	if args.NetworkCombinedLoss == nil {
		args.Logger.Debug("NetworkCombinedLoss is nil, returning empty forecast-implied inferences", "topicId", args.TopicId)
		return map[Forecaster]*emissionstypes.Inference{}, nil
	}

	if args.LabelRegistry == nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "topic id %d: LabelRegistry is nil", args.TopicId)
	}
	labels := args.LabelRegistry.GetLabels()
	regLen := args.NumLabels
	if regLen == 0 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "topic id %d has no labels", args.TopicId)
	}

	forecasterToForecastImpliedInference := make(map[Forecaster]*emissionstypes.Inference, len(args.Forecasters))

	for _, forecaster := range args.Forecasters {
		fc, ok := args.ForecasterToForecast[forecaster]
		if !ok || len(fc.ForecastElements) == 0 {
			continue
		}

		forecastElementsByInferer := make(map[Worker]*emissionstypes.ForecastElement)
		sortedInferersInForecast := make([]Worker, 0)

		for _, el := range fc.ForecastElements {
			if _, ok := args.InfererToInference[el.Inferer]; ok {
				forecastElementsByInferer[el.Inferer] = el
				sortedInferersInForecast = append(sortedInferersInForecast, el.Inferer)
			}
		}

		blockHeight := int64(0)

		if args.AllInferersAreNew {
			// ---------- MEDIAN ----------
			vecs := make([]emissionstypes.InferenceValues, 0, len(sortedInferersInForecast))

			for _, inferer := range sortedInferersInForecast {
				inf := args.InfererToInference[inferer]

				if blockHeight == 0 {
					blockHeight = inf.BlockHeight
				}

				iv, err := emissionstypes.ConvertInferenceValuesFromProto(args.TopicArity, labels, inf)
				if err != nil {
					return nil, err
				}

				vecs = append(vecs, iv)
			}

			var result emissionstypes.InferenceValues

			if args.TopicArity == emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {

				values := make([]alloraMath.Dec, 0, len(vecs))
				for _, v := range vecs {
					values = append(values, v[0])
				}

				m, err := alloraMath.Median(values)
				if err != nil {
					return nil, err
				}

				result = emissionstypes.InferenceValues{m}

			} else {

				out := make(alloraMath.DecArray, regLen)

				for i := 0; i < regLen; i++ {
					col := make([]alloraMath.Dec, 0, len(vecs))
					for _, v := range vecs {
						col = append(col, v[i])
					}

					m, err := alloraMath.Median(col)
					if err != nil {
						return nil, err
					}

					out[i] = m
				}

				result = out
			}

			//nolint:exhaustruct
			forecastImpliedInference := emissionstypes.Inference{
				TopicId:     args.TopicId,
				BlockHeight: blockHeight,
				Inferer:     forecaster,
				Values:      []alloraMath.Dec(result),
			}

			forecasterToForecastImpliedInference[forecaster] = &forecastImpliedInference
			continue
		}

		// ---------- WEIGHTED ----------
		infererRegretsForThisForecaster := make(map[Inferer]*Regret, len(forecastElementsByInferer))
		infererWeightsForThisForecaster := make(map[Inferer]Weight, len(forecastElementsByInferer))

		for _, infererInForecast := range sortedInferersInForecast {
			r, err := (*args.NetworkCombinedLoss).Sub(forecastElementsByInferer[infererInForecast].Value)
			if err != nil {
				return nil, err
			}

			infererRegretsForThisForecaster[infererInForecast] = &r
		}

		if len(sortedInferersInForecast) > 1 {
			infererToRegretOut := infererRegretsForThisForecaster
			forecasterToRegretOut := make(map[Forecaster]*Regret)

			weights, err := CalcWeightsGivenWorkers(
				CalcWeightsGivenWorkersArgs{
					Logger:                 args.Logger,
					Inferers:               args.Inferers,
					Forecasters:            args.Forecasters,
					InfererToRegret:        infererToRegretOut,
					ForecasterToRegret:     forecasterToRegretOut,
					EpsilonTopic:           args.EpsilonTopic,
					PNorm:                  args.PNorm,
					CNorm:                  args.CNorm,
					RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
				},
			)
			if err != nil {
				return nil, err
			}

			infererWeightsForThisForecaster = weights.Inferers

		} else if len(sortedInferersInForecast) > 0 {
			infererWeightsForThisForecaster[sortedInferersInForecast[0]] = alloraMath.OneDec()
		} else {
			continue
		}

		sumWeights := alloraMath.ZeroDec()
		running := make(alloraMath.DecArray, regLen)

		for _, inferer := range sortedInferersInForecast {
			w := infererWeightsForThisForecaster[inferer]
			if w.Equal(alloraMath.ZeroDec()) {
				continue
			}

			inf := args.InfererToInference[inferer]

			if blockHeight == 0 {
				blockHeight = inf.BlockHeight
			}

			iv, err := emissionstypes.ConvertInferenceValuesFromProto(args.TopicArity, labels, inf)
			if err != nil {
				return nil, err
			}

			for i := range iv {
				v, err := w.Mul(iv[i])
				if err != nil {
					return nil, err
				}

				running[i], err = running[i].Add(v)
				if err != nil {
					return nil, err
				}
			}

			sumWeights, err = sumWeights.Add(w)
			if err != nil {
				return nil, err
			}
		}

		if sumWeights.Equal(alloraMath.ZeroDec()) {
			continue
		}

		for i := range running {
			var err error
			running[i], err = running[i].Quo(sumWeights)
			if err != nil {
				return nil, err
			}
		}

		//nolint:exhaustruct
		forecastImpliedInference := emissionstypes.Inference{
			TopicId:     args.TopicId,
			BlockHeight: blockHeight,
			Inferer:     forecaster,
			Values:      []alloraMath.Dec(running),
		}

		forecasterToForecastImpliedInference[forecaster] = &forecastImpliedInference
	}

	return forecasterToForecastImpliedInference, nil
}
