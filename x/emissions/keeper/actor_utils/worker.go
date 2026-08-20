package actorutils

import (
	"sort"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// WORKER NONCES CLOSING

// Closes an open worker nonce.
func CloseWorkerNonce(k *keeper.Keeper, ctx sdk.Context, topic types.Topic, nonce types.Nonce) (err error) {
	blockHeight := ctx.BlockHeight()

	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := k.GetNonceKeeper().IsWorkerNonceUnfulfilled(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return types.ErrUnfulfilledNonceNotFound
	}

	// Too early: the window has not opened yet. Overdue closes (after WorkerSubmissionWindow)
	// are allowed so a missed close-cadence block can still add the reputer nonce.
	if blockHeight <= nonce.BlockHeight {
		return types.ErrWorkerNonceWindowNotAvailable
	}

	defer func() {
		if err != nil {
			ctx.Logger().Error(
				"Error occurred before finalization in CloseWorkerNonce, attempting cleanup anyway",
				"topicId", topic.Id,
				"nonce", nonce,
				"error", err,
			)
		}

		_, fulfillErr := k.GetNonceKeeper().FulfillWorkerNonce(ctx, topic.Id, &nonce)
		if fulfillErr != nil {
			ctx.Logger().Error(
				"Error fulfilling worker nonce during deferred cleanup",
				"topicId", topic.Id,
				"nonce", nonce,
				"error", fulfillErr,
			)
		}

		resetActiveErr := k.GetWorkerKeeper().ResetActiveWorkersForTopic(ctx, topic.Id)
		if resetActiveErr != nil {
			ctx.Logger().Error(
				"Error resetting active workers during deferred cleanup",
				"topicId", topic.Id,
				"error", resetActiveErr,
			)
		}

		resetSubmissionsErr := k.GetWorkerKeeper().ResetWorkersIndividualSubmissionsForTopic(ctx, topic.Id)
		if resetSubmissionsErr != nil {
			ctx.Logger().Error(
				"Error resetting worker individual submissions during deferred cleanup",
				"topicId", topic.Id,
				"error", resetSubmissionsErr,
			)
		}

		ctx.Logger().Info("Closed worker nonce", "topicId", topic.Id, "nonce", nonce)

		// Emit worker submission window closed event
		types.EmitNewWorkerSubmissionWindowClosedEvent(ctx, topic.Id, nonce.BlockHeight)
	}()

	// Get all active inferers for this topic
	activeInfererAddresses, err := k.GetWorkerKeeper().GetActiveInferersForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}
	if len(activeInfererAddresses) == 0 {
		return types.ErrNoQualifiedInferers
	}

	// Insert set of active activeInferences for this topic/block and return a map
	// of the inferers with active inferers to be used in the forecasts processing
	activeInfererAddressesMap, activeInferences, err := closeActiveInferencesSet(
		ctx,
		k,
		topic,
		nonce,
		activeInfererAddresses,
	)
	if err != nil {
		return err
	}

	// Get all active forecasters for this topic
	activeForecastAddresses, err := k.GetWorkerKeeper().GetActiveForecastersForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}

	// Insert set of active forecasts for this topic/block and return a map
	// of the forecasters with active forecasts to be used in the forecasts processing
	activeForecasts, err := closeActiveForecastsSet(
		ctx,
		k,
		topic.Id,
		nonce,
		activeForecastAddresses,
		activeInfererAddressesMap,
	)
	if err != nil {
		return err
	}

	err = k.GetNonceKeeper().AddReputerNonce(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}

	err = k.GetTopicKeeper().SetWorkerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}

	if topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
		// Once inferences are closed, update the network inferences outlier metrics
		err = k.GetWorkerKeeper().UpdateNetworkInferencesOutlierMetrics(ctx, topic, nonce.BlockHeight)
		if err != nil {
			return err
		}
	}

	// Computes and stores both regular and outlier-resistant network inferences
	err = ProcessAndStoreNetworkInferences(k, ctx, topic, nonce.BlockHeight, activeInferences, activeForecasts)
	if err != nil {
		return err
	}

	types.EmitNewActiveInferersSetEvent(ctx, topic.Id, nonce.BlockHeight, activeInfererAddresses)
	types.EmitNewActiveForecastersSetEvent(ctx, topic.Id, nonce.BlockHeight, activeForecastAddresses)
	types.EmitNewWorkerLastCommitSetEvent(ctx, topic.Id, blockHeight, &nonce)
	return nil
}

// ProcessAndStoreNetworkInferences calculates and stores both regular and outlier-resistant network inferences
// for a given topic and block height.
func ProcessAndStoreNetworkInferences(
	k *keeper.Keeper,
	ctx sdk.Context,
	topic types.Topic,
	nonce int64,
	activeInferences *types.Inferences,
	activeForecasts *types.Forecasts,
) error {
	// Calculate regular network inferences
	networkInferencesResult, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(ctx),
		*k,
		topic.Id,
		&nonce,
		activeInferences,
		activeForecasts,
	)
	if err != nil {
		return errorsmod.Wrap(err, "failed to calculate network inferences")
	}

	// Store regular network inferences
	if err := k.InsertNetworkInferenceBundle(ctx, topic.Id, nonce, *networkInferencesResult.NetworkInferences); err != nil {
		return errorsmod.Wrap(err, "failed to insert network inference")
	}

	types.EmitNewNetworkInferencesEvent(ctx, *networkInferencesResult.NetworkInferences)

	// Emit packed network inference weight events
	infererAddresses, infererWeights := buildSortedAddressWeights(networkInferencesResult.InfererToWeight)
	if len(infererAddresses) > 0 {
		types.EmitNewNetworkInferenceInfererWeightsSetEvent(ctx, topic.Id, nonce, infererAddresses, infererWeights)
	}

	forecasterAddresses, forecasterWeights := buildSortedAddressWeights(networkInferencesResult.ForecasterToWeight)
	if len(forecasterAddresses) > 0 {
		types.EmitNewNetworkInferenceForecasterWeightsSetEvent(ctx, topic.Id, nonce, forecasterAddresses, forecasterWeights)
	}

	if topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE {
		// Get outlier resistant inferences
		outlierResistantFilteredInferences, err := k.GetTopicKeeper().FilterOutlierResistantInferences(ctx, topic, *activeInferences)
		if err != nil {
			return errorsmod.Wrap(err, "failed to filter outlier resistant inferences")
		}

		// Initialize outlier resistant result with regular result
		outlierResistantNetworkInferencesResult := networkInferencesResult

		// Recalculate only if outlier filtering changed the inference set
		if len(outlierResistantFilteredInferences.Inferences) != len(activeInferences.Inferences) {
			outlierResistantNetworkInferencesResult, err = synth.GetNetworkInferences(sdk.UnwrapSDKContext(ctx), *k, topic.Id, &nonce, &outlierResistantFilteredInferences, activeForecasts)
			if err != nil {
				return errorsmod.Wrap(err, "failed to calculate outlier resistant network inferences")
			}
		}

		// Store outlier resistant network inferences
		if err := k.InsertOutlierResistantNetworkInferenceBundle(ctx, topic.Id, nonce, *outlierResistantNetworkInferencesResult.NetworkInferences); err != nil {
			return errorsmod.Wrap(err, "failed to insert outlier resistant network inference")
		}

		types.EmitNewOutlierResistantNetworkInferencesEvent(ctx, *outlierResistantNetworkInferencesResult.NetworkInferences)
	}

	return nil
}

func buildSortedAddressWeights(weightsByAddress map[string]alloraMath.Dec) ([]string, []alloraMath.Dec) {
	if len(weightsByAddress) == 0 {
		return nil, nil
	}

	addresses := make([]string, 0, len(weightsByAddress))
	for address := range weightsByAddress { //nolint:maprange // it is later sorted below
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	weights := make([]alloraMath.Dec, 0, len(addresses))
	for _, address := range addresses {
		weights = append(weights, weightsByAddress[address])
	}
	return addresses, weights
}

// closeActiveInferencesSet freezes the active workers' surviving labels into a
// compact final EpochLabelRegistry, then remaps each vector into a committed
// types.Inference aligned to the final compact registry.
//
// This is where the classification design transitions from temporary label
// ids to final label ids: during the WSW, active inferences are dense vectors
// aligned to the temporary first-seen ELR; after this call every downstream
// consumer sees dense Values aligned to the final compact ELR.
//
// Returns (inferer address set, finalized inferences) and also emits
// EventEpochLabelRegistryFrozen so offchain indexers can track the committed
// registry size for this epoch.
func closeActiveInferencesSet(
	ctx sdk.Context,
	k *keeper.Keeper,
	topic types.Topic,
	nonce types.Nonce,
	activeInfererAddresses []string,
) (activeInfererAddressesMap map[string]bool, inferences *types.Inferences, err error) {
	activeInfererAddressesMap = make(map[string]bool, 0)

	inferences, registry, err := k.GetWorkerKeeper().FinalizeInferencesAndRegistryAtClose(
		ctx, topic, nonce.BlockHeight, activeInfererAddresses,
	)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "failed to finalize active inferences for topic %d nonce %d", topic.Id, nonce.BlockHeight)
	}

	err = k.GetTopicKeeper().SetEpochLabelRegistry(ctx, registry)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "error setting final epoch label registry")
	}

	//nolint:gosec // registry size is bounded by MaxLabelsPerSubmission (uint64), safe to cast
	registrySize := uint64(len(registry.Labels))
	types.EmitNewEpochLabelRegistryFrozenEvent(ctx, topic.Id, nonce.BlockHeight, registrySize)

	for _, inference := range inferences.Inferences {
		activeInfererAddressesMap[inference.Inferer] = true
	}

	err = k.GetWorkerKeeper().InsertActiveInferences(ctx, topic.Id, nonce.BlockHeight, *inferences)
	if err != nil {
		return nil, nil, err
	}

	return activeInfererAddressesMap, inferences, nil
}

// insert forecasts from top forecasters
// check forecast elements to ensure they are forecasts made about
// the active list of inferers.
func closeActiveForecastsSet(
	ctx sdk.Context,
	k *keeper.Keeper,
	topicId uint64,
	nonce types.Nonce,
	activeForecastAddresses []string,
	acceptedInferersOfBatch map[string]bool,
) (forecasts *types.Forecasts, err error) {
	forecastsByForecaster := make(map[string]*types.Forecast)
	activeForecasts := make([]*types.Forecast, 0)

	for _, address := range activeForecastAddresses {
		forecast, err := k.GetWorkerKeeper().GetWorkerLatestForecastByTopicId(ctx, topicId, address)
		if err != nil {
			return nil, err
		}

		// Forecast validations
		if forecast.TopicId != topicId {
			ctx.Logger().Warn("Forecast does not match topic: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
			continue
		}
		if forecast.BlockHeight != nonce.BlockHeight {
			ctx.Logger().Warn("Forecast does not match blockHeight: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
			continue
		}

		// Examine forecast elements to verify that they're for inferers in the current set.
		// We assume that set of inferers has been verified above.
		// We keep what we can, ignoring the forecaster and their contribution (forecast) entirely
		// if they're left with no valid forecast elements.
		acceptedForecastElements := make([]*types.ForecastElement, 0)
		for _, el := range forecast.ForecastElements {
			if _, ok := acceptedInferersOfBatch[el.Inferer]; ok {
				acceptedForecastElements = append(acceptedForecastElements, el)
			}
		}

		// Discard if no accepted forecasts elements found
		if len(acceptedForecastElements) == 0 {
			continue
		}

		// Update the forecast with the filtered elements
		if forecast.ForecastElements != nil {
			forecast.ForecastElements = acceptedForecastElements
		}

		// / Now do filters on each forecaster
		// Ensure that we only have one forecast per forecaster. If not, we just take the first one
		if _, ok := forecastsByForecaster[forecast.Forecaster]; !ok {
			activeForecasts = append(activeForecasts, &forecast)
			forecastsByForecaster[forecast.Forecaster] = &forecast
		}
	}

	// Ensure deterministic ordering
	sort.Slice(activeForecasts, func(i, j int) bool {
		return activeForecasts[i].Forecaster < activeForecasts[j].Forecaster
	})

	forecasts = &types.Forecasts{
		Forecasts: activeForecasts,
	}

	err = k.GetWorkerKeeper().InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, *forecasts)
	if err != nil {
		return nil, err
	}
	return forecasts, nil
}
