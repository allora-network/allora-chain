package inferencesynthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Args for the CalcForecastImpliedInferences function
type CalcForecastImpliedInferencesArgs struct {
	Logger               log.Logger
	TopicId              uint64
	AllInferersAreNew    bool
	Inferers             []Inferer
	InfererToInference   map[Inferer]*emissionstypes.Inference
	InfererToRegret      map[Inferer]*Regret
	Forecasters          []Forecaster
	ForecasterToForecast map[Forecaster]*emissionstypes.Forecast
	ForecasterToRegret   map[Forecaster]*Regret
	NetworkCombinedLoss  alloraMath.Dec
	EpsilonTopic         alloraMath.Dec
	PNorm                alloraMath.Dec
	CNorm                alloraMath.Dec
}

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forecast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
func CalcForecastImpliedInferences(args CalcForecastImpliedInferencesArgs) (
	forecasterToForecastImpliedInference map[Forecaster]*emissionstypes.Inference,
	infererToRegretOut map[Inferer]*Regret,
	forecasterToRegretOut map[Forecaster]*Regret,
	err error,
) {
	args.Logger.Debug(fmt.Sprintf("Calculating forecast-implied inferences for topic %v", args.TopicId))
	// "k" here is the forecaster's address
	// For each forecast, and for each forecast element, calculate forecast-implied inferences I_ik
	forecasterToForecastImpliedInference = make(map[Forecaster]*emissionstypes.Inference, len(args.Forecasters))
	infererToRegretOut = args.InfererToRegret
	forecasterToRegretOut = args.ForecasterToRegret
	for _, forecaster := range args.Forecasters {
		_, ok := args.ForecasterToForecast[forecaster]
		if ok && len(args.ForecasterToForecast[forecaster].ForecastElements) > 0 {
			// Filter away all forecast elements that do not have an associated inference (match by worker)
			// Will effectively set weight in formulas for forecast-implied inference I_ik and network inference I_i to 0 for forecasts without inferences
			// Map inferer -> forecast element => only one (latest in array) forecast element per inferer
			forecastElementsByInferer := make(map[Worker]*emissionstypes.ForecastElement, 0)
			sortedInferersInForecast := make([]Worker, 0)
			for _, el := range args.ForecasterToForecast[forecaster].ForecastElements {
				if _, ok := args.InfererToInference[el.Inferer]; ok {
					// Check that there is an inference for the worker forecasted before including the forecast element
					// otherwise the max value below will be incorrect.
					forecastElementsByInferer[el.Inferer] = el
					sortedInferersInForecast = append(sortedInferersInForecast, el.Inferer)
				}
			}

			weightSum := alloraMath.ZeroDec()                 // denominator in calculation of forecast-implied inferences
			weightInferenceDotProduct := alloraMath.ZeroDec() // numerator in calculation of forecast-implied inferences

			blockHeight := int64(0)

			// Calculate the forecast-implied inferences I_ik
			if args.AllInferersAreNew {
				// If all inferers are new, calculate the median of the inferences
				// This means that forecasters won't be able to influence the network inference when all inferers are new
				// However, this seeds losses for forecasters for future rounds
				inferenceValues := make([]alloraMath.Dec, 0, len(sortedInferersInForecast))
				for _, inferer := range sortedInferersInForecast {
					inference, ok := args.InfererToInference[inferer]
					if ok {
						inferenceValues = append(inferenceValues, inference.Value)
						if blockHeight == 0 {
							blockHeight = inference.BlockHeight
						}
					}
				}

				medianValue, err := alloraMath.Median(inferenceValues)
				if err != nil {
					return nil, nil, nil, errorsmod.Wrapf(err, "error calculating median of inference values")
				}

				forecastImpliedInference := emissionstypes.Inference{
					TopicId:     args.TopicId,
					BlockHeight: blockHeight,
					Inferer:     forecaster,
					Value:       medianValue,
					ExtraData:   nil,
					Proof:       "",
				}
				forecasterToForecastImpliedInference[forecaster] = &forecastImpliedInference
			} else {
				// If not all inferers are new, calculate forecast-implied inferences using the previous inferer regrets and previous network loss

				// Approximate forecast regrets of the network inference
				// Map inferer -> regret
				// this is R_ik in the paper
				infererRegretsForThisForecaster := make(map[Inferer]*Regret, len(forecastElementsByInferer))
				// Forecast-regret-informed weights dot product with inferences to yield forecast-implied inferences
				// Map inferer -> weight
				// this is w_ik in the paper
				infererWeightsForThisForecaster := make(map[Inferer]Weight, len(forecastElementsByInferer))

				// Define variable to store maximum regret for forecast k
				// infererInForecast corresponds to
				// `j` the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the paper
				for _, infererInForecast := range sortedInferersInForecast {
					// Calculate the approximate forecast regret of the network inference
					// this is R_ijk in the paper
					forecastRegretOfNetworkInference, err :=
						args.NetworkCombinedLoss.Sub(forecastElementsByInferer[infererInForecast].Value)
					if err != nil {
						return nil, nil, nil, errorsmod.Wrapf(err,
							"error calculating forecast-implied inferences: error calculating network loss per value")
					}
					infererRegretsForThisForecaster[infererInForecast] = &forecastRegretOfNetworkInference
				}

				if len(sortedInferersInForecast) > 1 {
					infererToRegretOut = infererRegretsForThisForecaster
					forecasterToRegretOut = make(map[Forecaster]*alloraMath.Dec, 0)

					weights, err := calcWeightsGivenWorkers(
						calcWeightsGivenWorkersArgs{
							logger:             args.Logger,
							inferers:           args.Inferers,
							forecasters:        args.Forecasters,
							infererToRegret:    infererToRegretOut,
							forecasterToRegret: forecasterToRegretOut,
							epsilonTopic:       args.EpsilonTopic,
							pNorm:              args.PNorm,
							cNorm:              args.CNorm,
						},
					)
					if err != nil {
						return nil, nil, nil, errorsmod.Wrapf(err,
							"error calculating forecast-implied inferences:error calculating normalized forecasted regrets")
					}
					infererWeightsForThisForecaster = weights.inferers
				} else if len(sortedInferersInForecast) == 1 {
					weights := make(map[Worker]Weight, 1)
					weights[sortedInferersInForecast[0]] = alloraMath.OneDec()
					infererWeightsForThisForecaster = weights
				}

				// Calculate the forecast-implied inferences I_ik
				for _, infererInForecast := range sortedInferersInForecast {
					// this is w_ijk in the paper
					weightIJK := infererWeightsForThisForecaster[infererInForecast]

					inference, ok := args.InfererToInference[infererInForecast]
					if ok && blockHeight == 0 {
						blockHeight = inference.BlockHeight
					}
					if ok && !(weightIJK.Equal(alloraMath.ZeroDec())) {
						thisDotProduct, err := weightIJK.Mul(args.InfererToInference[infererInForecast].Value)
						if err != nil {
							return nil, nil, nil, errorsmod.Wrapf(err,
								"error calculating forecast-implied inferences: error calculating dot product")
						}
						weightInferenceDotProduct, err = weightInferenceDotProduct.Add(thisDotProduct)
						if err != nil {
							return nil, nil, nil, errorsmod.Wrapf(err,
								"error calculating forecast-implied inferences: error adding dot product")
						}
						weightSum, err = weightSum.Add(weightIJK)
						if err != nil {
							return nil, nil, nil, errorsmod.Wrapf(err,
								"error calculating forecast-implied inferences: error adding weight")
						}
					}
				}

				if !weightSum.Equal(alloraMath.ZeroDec()) {
					forecastValue, err := weightInferenceDotProduct.Quo(weightSum)
					if err != nil {
						return nil, nil, nil, errorsmod.Wrapf(err, "error calculating forecast value")
					}
					forecastImpliedInference := emissionstypes.Inference{
						TopicId:     args.TopicId,
						BlockHeight: blockHeight,
						Inferer:     forecaster,
						Value:       forecastValue,
						ExtraData:   nil,
						Proof:       "",
					}
					forecasterToForecastImpliedInference[forecaster] = &forecastImpliedInference
				}
			}
		}
	}

	args.Logger.Debug(fmt.Sprintf("Forecast-implied inferences: %v", forecasterToForecastImpliedInference))
	return forecasterToForecastImpliedInference, infererToRegretOut, forecasterToRegretOut, nil
}
