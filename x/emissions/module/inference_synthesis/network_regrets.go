package inference_synthesis

import (
	"fmt"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type networkLossesByWorker struct {
	CombinedLoss           Loss
	InfererLosses          map[Worker]Loss
	ForecasterLosses       map[Worker]Loss
	NaiveLoss              Loss
	OneOutInfererLosses    map[Worker]Loss
	OneOutForecasterLosses map[Worker]Loss
	OneInForecasterLosses  map[Worker]Loss
}

// Convert a ValueBundle to a networkLossesByWorker
func convertValueBundleToNetworkLossesByWorker(
	valueBundle *emissions.ValueBundle,
) *networkLossesByWorker {
	infererLosses := make(map[Worker]Loss)
	for _, inferer := range valueBundle.InfererValues {
		infererLosses[inferer.Worker] = inferer.Value
	}

	forecasterLosses := make(map[Worker]Loss)
	for _, forecaster := range valueBundle.ForecasterValues {
		forecasterLosses[forecaster.Worker] = forecaster.Value
	}

	oneOutInfererLosses := make(map[Worker]Loss)
	for _, oneOutInferer := range valueBundle.OneOutInfererValues {
		oneOutInfererLosses[oneOutInferer.Worker] = oneOutInferer.Value
	}

	oneOutForecasterLosses := make(map[Worker]Loss)
	for _, oneOutForecaster := range valueBundle.OneOutForecasterValues {
		oneOutForecasterLosses[oneOutForecaster.Worker] = oneOutForecaster.Value
	}

	oneInForecasterLosses := make(map[Worker]Loss)
	for _, oneInForecaster := range valueBundle.OneInForecasterValues {
		oneInForecasterLosses[oneInForecaster.Worker] = oneInForecaster.Value
	}

	return &networkLossesByWorker{
		CombinedLoss:           valueBundle.CombinedValue,
		InfererLosses:          infererLosses,
		ForecasterLosses:       forecasterLosses,
		NaiveLoss:              valueBundle.NaiveValue,
		OneOutInfererLosses:    oneOutInfererLosses,
		OneOutForecasterLosses: oneOutForecasterLosses,
		OneInForecasterLosses:  oneInForecasterLosses,
	}
}

func computeEMRegretFromLosses(
	lossA Loss,
	lossB Loss,
	currentRegret Regret,
	alpha float64,
) Regret {
	return (1.0-alpha)*currentRegret + alpha*(lossA-lossB)
}

func computeAndBuildEMRegret(
	lossA Loss,
	lossB Loss,
	currentRegret Regret,
	alpha float64,
	blockHeight BlockHeight,
) types.TimestampedValue {
	newRegret := computeEMRegretFromLosses(lossA, lossB, currentRegret, alpha)
	return types.TimestampedValue{
		BlockHeight: blockHeight,
		Value:       newRegret,
	}
}

// Calculate the new network regrets by taking EMAs between the previous network regrets
// and the new regrets admitted by the inputted network losses
func GetCalcSetNetworkRegrets(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	networkLosses *emissions.ValueBundle,
	blockHeight BlockHeight,
	alpha float64,
) error {
	// Convert the network losses to a networkLossesByWorker
	networkLossesByWorker := convertValueBundleToNetworkLossesByWorker(networkLosses)

	// Get old regret R_{i-1,j} and Calculate then Set the new regrets R_ij for inferers
	for _, infererLoss := range networkLosses.InfererValues {
		lastRegret, err := k.GetInfererNetworkRegret(ctx, topicId, sdk.AccAddress(infererLoss.Worker))
		if err != nil {
			fmt.Println("Error getting inferer regret: ", err)
			return err
		}
		newInfererRegret := computeAndBuildEMRegret(networkLosses.CombinedValue, networkLossesByWorker.InfererLosses[infererLoss.Worker], lastRegret.Value, alpha, blockHeight)
		k.SetInfererNetworkRegret(ctx, topicId, sdk.AccAddress(infererLoss.Worker), newInfererRegret)
	}

	// Get old regret R_{i-1,k} and Calculate then Set the new regrets R_ik for forecastsers
	for _, forecasterLoss := range networkLosses.ForecasterValues {
		lastRegret, err := k.GetInfererNetworkRegret(ctx, topicId, sdk.AccAddress(forecasterLoss.Worker))
		if err != nil {
			fmt.Println("Error getting forecaster regret: ", err)
			return err
		}
		newForecasterRegret := computeAndBuildEMRegret(networkLosses.CombinedValue, networkLossesByWorker.ForecasterLosses[forecasterLoss.Worker], lastRegret.Value, alpha, blockHeight)
		k.SetForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(forecasterLoss.Worker), newForecasterRegret)
	}

	// Calculate the new one-in regrets for the forecasters R^+_ij'k where j' includes all j and forecast implied inference from forecaster k
	for _, oneInForecasterLoss := range networkLosses.OneInForecasterValues {
		// Loop over the inferer losses so that their losses may be compared against the one-in forecaster's loss, for each forecaster
		for _, infererLoss := range networkLosses.InfererValues {
			lastRegret, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(oneInForecasterLoss.Worker), sdk.AccAddress(infererLoss.Worker))
			if err != nil {
				fmt.Println("Error getting one-in forecaster regret: ", err)
				return err
			}
			newOneInForecasterRegret := computeAndBuildEMRegret(networkLossesByWorker.OneInForecasterLosses[oneInForecasterLoss.Worker], networkLossesByWorker.InfererLosses[infererLoss.Worker], lastRegret.Value, alpha, blockHeight)
			k.SetOneInForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(oneInForecasterLoss.Worker), sdk.AccAddress(infererLoss.Worker), newOneInForecasterRegret)
		}
		// Self-regret for the forecaster given their own regret
		lastRegret, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(oneInForecasterLoss.Worker), sdk.AccAddress(oneInForecasterLoss.Worker))
		if err != nil {
			fmt.Println("Error getting one-in forecaster self regret: ", err)
			return err
		}
		oneInForecasterSelfRegret := computeAndBuildEMRegret(networkLossesByWorker.OneInForecasterLosses[oneInForecasterLoss.Worker], networkLossesByWorker.ForecasterLosses[oneInForecasterLoss.Worker], lastRegret.Value, alpha, blockHeight)
		k.SetOneInForecasterNetworkRegret(ctx, topicId, sdk.AccAddress(oneInForecasterLoss.Worker), sdk.AccAddress(oneInForecasterLoss.Worker), oneInForecasterSelfRegret)
	}

	return nil
}

// /// TODO ensure root functions check for uniqueness of inferer and forecaster workers in the input => Every sub function need not care

// /// !! TODO: Apply the following functions to the module + Revamp them as needed to exhibit the proper I/O interface !!

// // Gathers inferences and forecasts from the worker update at or just after the given time, and
// // losses from the loss update just before the given time.
// // Then invokes calculation of the forecast-implied inferences using the Inference Synthesis formula from the litepaper.
// func GetAndCalcForcastImpliedInferencesAtBlock(
// 	ctx sdk.Context,
// 	k keeper.Keeper,
// 	topicId TopicId,
// 	blockHeight BlockHeight,
// ) ([]*emissions.Inference, error) {
// 	// Get inferences from worker update at or just after the given time
// 	inferences, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting inferences: ", err)
// 		return nil, err
// 	}
// 	// Get forecasts from worker update at or just after the given time
// 	forecasts, err := k.GetForecastsAtOrAfterBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting forecasts: ", err)
// 		return nil, err
// 	}
// 	// Get losses from loss update just before the given time
// 	networkValueBundle, err := k.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting network losses: ", err)
// 		return nil, err
// 	}

// 	epsilon, err := k.GetParamsEpsilon(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return nil, err
// 	}

// 	pInferenceSynthesis, err := k.GetParamsPInferenceSynthesis(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return nil, err
// 	}

// 	return calcForcastImpliedInferences(inferences, forecasts, networkValueBundle, epsilon, pInferenceSynthesis)
// }

// // Gathers inferences, forecast-implied inferences as of the given block
// // and the network regrets admitted by workers at or just before the given time,
// // then invokes calculation of network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper.
// func GetAndCalcNetworkCombinedInference(
// 	ctx sdk.Context,
// 	k keeper.Keeper,
// 	topicId TopicId,
// 	blockHeight BlockHeight,
// ) (InferenceValue, error) {
// 	// Get inferences from worker update at or just after the given block
// 	inferences, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting inferences: ", err)
// 		return 0, err
// 	}
// 	// Map each worker to their inference
// 	inferenceByWorker := makeMapFromWorkerToTheirWork(inferences.Inferences)

// 	// Get regrets admitted by workers just before the given block
// 	regrets, err := k.GetNetworkRegretsAtOrBeforeBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting network regrets: ", err)
// 		return 0, err
// 	}
// 	// Get forecast-implied inferences at the given block
// 	forecastImpliedInferences, err := GetAndCalcForcastImpliedInferencesAtBlock(ctx, k, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting forecast implied inferences: ", err)
// 		return 0, err
// 	}
// 	// Map each worker to their forecast-implied inference
// 	forecastImpliedInferenceByWorker := makeMapFromWorkerToTheirWork(forecastImpliedInferences)

// 	epsilon, err := k.GetParamsEpsilon(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return 0, err
// 	}

// 	pInferenceSynthesis, err := k.GetParamsPInferenceSynthesis(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return 0, err
// 	}

// 	return calcWeightedInference(inferenceByWorker, forecastImpliedInferenceByWorker, regrets, epsilon, pInferenceSynthesis)
// }
