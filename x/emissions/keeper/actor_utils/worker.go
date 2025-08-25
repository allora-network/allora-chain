package actorutils

import (
	"sort"

	"github.com/allora-network/allora-chain/errors"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WORKER NONCES CLOSING

// Closes an open worker nonce.
func CloseWorkerNonce(k *keeper.Keeper, ctx sdk.Context, topic types.Topic, nonce types.Nonce) (err error) {
	defer errors.Annotate(&err, "topic", topic.Id, "nonce", nonce.BlockHeight)

	blockHeight := ctx.BlockHeight()

	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topic.Id, &nonce)
	if err != nil {
		return errors.Wrap(err, "failed: fetch is worker nonce unfulfilled")
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return types.ErrUnfulfilledNonceNotFound
	}

	// Check if the window time has passed: if blockHeight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	if blockHeight <= nonce.BlockHeight ||
		blockHeight > nonce.BlockHeight+topic.WorkerSubmissionWindow {
		return types.ErrWorkerNonceWindowNotAvailable
	}

	// Get all active inferers for this topic
	activeInfererAddresses, err := k.GetActiveInferersForTopic(ctx, topic.Id)
	if err != nil {
		return errors.Wrap(err, "failed: fetch active inferrers for topic")
	}

	// Insert set of active activeInferences for this topic/block and return a map
	// of the inferers with active inferers to be used in the forecasts processing
	activeInfererAddressesMap, activeInferences, err := insertActiveInferences(
		ctx,
		k,
		topic.Id,
		nonce,
		activeInfererAddresses,
	)
	if err != nil {
		return errors.Wrap(err, "failed: close active inference set for topic")
	}

	// Get all active forecasters for this topic
	activeForecastAddresses, err := k.GetActiveForecastersForTopic(ctx, topic.Id)
	if err != nil {
		return errors.Wrap(err, "failed: fetch active forecasters for topic")
	}

	// Insert set of active forecasts for this topic/block and return a map
	// of the forecasters with active forecasts to be used in the forecasts processing
	activeForecasts, err := insertActiveForecasts(
		ctx,
		k,
		topic.Id,
		nonce,
		activeForecastAddresses,
		activeInfererAddressesMap,
	)
	if err != nil {
		return errors.Wrap(err, "failed: close active forecast set for topic")
	}

	// Now that inferences are closed, update the network inferences outlier metrics
	// Computes and stores both regular and outlier-resistant network inferences
	err = ProcessAndStoreNetworkInferences(k, ctx, topic.Id, nonce.BlockHeight, activeInferences, activeForecasts)
	if err != nil {
		return errors.Wrap(err, "failed: process and store network inferences")
	}

	_, err = k.FulfillWorkerNonce(ctx, topic.Id, &nonce)
	if err != nil {
		return errors.Wrap(err, "failed: fulfill worker nonce")
	}

	err = k.ResetActiveWorkersForTopic(ctx, topic.Id)
	if err != nil {
		return errors.Wrap(err, "failed: reset active workers for topic")
	}

	err = k.ResetWorkersIndividualSubmissionsForTopic(ctx, topic.Id)
	if err != nil {
		return errors.Wrap(err, "failed: reset workers individual submissions for topic")
	}

	if len(activeInfererAddresses) > 0 {
		// This should only happen when there are inferences for this nonce
		err = k.SetWorkerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
		if err != nil {
			return errors.Wrap(err, "failed: set worker topic last commit")
		}

		// Now that the inference/forecast phase is complete, we open the corresponding reputer nonce
		err = k.AddReputerNonce(ctx, topic.Id, &nonce)
		if err != nil {
			return errors.Wrap(err, "failed: add reputer nonce for topic")
		}

		types.EmitNewWorkerLastCommitSetEvent(ctx, topic.Id, blockHeight, &nonce)
	}

	ctx.Logger().Info("Closed worker nonce", "topicId", topic.Id, "nonce", nonce)

	return nil
}

// ProcessAndStoreNetworkInferences calculates and stores both regular and outlier-resistant network inferences
// for a given topic and block height.
func ProcessAndStoreNetworkInferences(
	k *keeper.Keeper,
	ctx sdk.Context,
	topicId uint64,
	blockHeight int64,
	activeInferences *types.Inferences,
	activeForecasts *types.Forecasts,
) error {
	if activeInferences == nil || len(activeInferences.Inferences) == 0 {
		return nil
	}

	err := k.UpdateNetworkInferencesOutlierMetrics(ctx, topicId, blockHeight)
	if err != nil {
		return errors.Wrap(err, "failed: update network inferences outlier metrics")
	}

	// Calculate regular network inferences
	networkInferencesResult, err := synth.GetNetworkInferences(
		sdk.UnwrapSDKContext(ctx),
		*k,
		topicId,
		&blockHeight,
		activeInferences,
		activeForecasts,
		false,
	)
	if err != nil {
		return errors.Wrap(err, "failed: calculate network inferences")
	}

	// Store regular network inferences
	err = k.InsertNetworkInferences(ctx, topicId, blockHeight, *networkInferencesResult.NetworkInferences)
	if err != nil {
		return errors.Wrap(err, "failed: insert network inference")
	}

	types.EmitNewNetworkInferencesEvent(ctx, topicId, blockHeight, *networkInferencesResult.NetworkInferences)

	// Get outlier resistant inferences
	outlierResistantFilteredInferences, err := k.FilterOutlierResistantInferences(ctx, topicId, *activeInferences)
	if err != nil {
		return errors.Wrap(err, "failed: filter outlier resistant inferences")
	}

	// Initialize outlier resistant result with regular result
	outlierResistantNetworkInferencesResult := networkInferencesResult

	// Recalculate only if outlier filtering changed the inference set
	if len(outlierResistantFilteredInferences.Inferences) != len(activeInferences.Inferences) {
		outlierResistantNetworkInferencesResult, err = synth.GetNetworkInferences(
			sdk.UnwrapSDKContext(ctx),
			*k,
			topicId,
			&blockHeight,
			&outlierResistantFilteredInferences,
			activeForecasts,
			true,
		)
		if err != nil {
			return errors.Wrap(err, "failed: calculate outlier resistant network inferences")
		}
	}

	// Store outlier resistant network inferences
	err = k.InsertOutlierResistantNetworkInferences(ctx, topicId, blockHeight, *outlierResistantNetworkInferencesResult.NetworkInferences)
	if err != nil {
		return errors.Wrap(err, "failed: insert outlier resistant network inference")
	}

	types.EmitNewOutlierResistantNetworkInferencesEvent(ctx, topicId, blockHeight, *outlierResistantNetworkInferencesResult.NetworkInferences)

	return nil
}

// Returns a map of active inferer addresses to their latest inference and the inferences themselves
func insertActiveInferences(
	ctx sdk.Context,
	k *keeper.Keeper,
	topicId uint64,
	nonce types.Nonce,
	activeInfererAddresses []string,
) (activeInfererAddressesMap map[string]bool, inferences *types.Inferences, err error) {
	activeInferences := make([]*types.Inference, 0)
	activeInfererAddressesMap = make(map[string]bool, 0)

	for _, address := range activeInfererAddresses {
		inference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, address)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed: get worker latest inference by topic id")
		}
		activeInferences = append(activeInferences, &inference)
		activeInfererAddressesMap[inference.Inferer] = true
	}

	// Ensure deterministic ordering
	sort.Slice(activeInferences, func(i, j int) bool {
		return activeInferences[i].Inferer < activeInferences[j].Inferer
	})

	inferences = &types.Inferences{
		Inferences: activeInferences,
	}

	err = k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, *inferences)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed: insert active inferences")
	}
	return activeInfererAddressesMap, inferences, nil
}

// insert forecasts from top forecasters
// check forecast elements to ensure they are forecasts made about
// the active list of inferers.
func insertActiveForecasts(
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
		forecast, err := k.GetWorkerLatestForecastByTopicId(ctx, topicId, address)
		if err != nil {
			return nil, errors.Wrap(err, "failed: get worker latest forecast by topic id")
		}

		// Forecast validations
		if forecast.TopicId != topicId {
			return nil, errors.NewWithFields("forecast does not match topic", "forecaster", forecast.Forecaster)
		} else if forecast.BlockHeight != nonce.BlockHeight {
			return nil, errors.NewWithFields("forecast does not match block height", "forecaster", forecast.Forecaster)
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

		/// Now do filters on each forecaster
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

	err = k.InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, *forecasts)
	if err != nil {
		return nil, errors.Wrap(err, "failed: insert active forecasts")
	}
	return forecasts, nil
}
