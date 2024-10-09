package actorutils

import (
	"fmt"
	"sort"

	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WORKER NONCES CLOSING

// Closes an open worker nonce.
func CloseWorkerNonce(k *keeper.Keeper, ctx sdk.Context, topic types.Topic, nonce types.Nonce) error {
	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return types.ErrUnfulfilledNonceNotFound
	}

	// Check if the window time has passed: if blockHeight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	blockHeight := ctx.BlockHeight()
	if blockHeight <= topic.EpochLastEnded ||
		blockHeight > topic.EpochLastEnded+topic.GroundTruthLag {
		return types.ErrWorkerNonceWindowNotAvailable
	}

	// Get all active inferers for this topic
	activeInfererAddresses, err := k.GetActiveInferersForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}
	if len(activeInfererAddresses) == 0 {
		return types.ErrNoQualifiedInferers
	}

	// Insert set of active inferences for this topic/block and return a map
	// of the inferers with active inferers to be used in the forecasts processing
	activeInfererAddressesMap, err := closeActiveInferencesSet(
		ctx,
		k,
		topic.Id,
		nonce,
		activeInfererAddresses,
	)
	if err != nil {
		return err
	}

	// Get all active forecasters for this topic
	activeForecastAddresses, err := k.GetActiveForecastersForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}

	// Insert set of active forecasts for this topic/block and return a map
	// of the forecasters with active forecasts to be used in the forecasts processing
	err = closeActiveForecastsSet(
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
	// Update the unfulfilled worker nonce
	_, err = k.FulfillWorkerNonce(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}

	err = k.AddReputerNonce(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}

	err = k.SetWorkerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}

	types.EmitNewWorkerLastCommitSetEvent(ctx, topic.Id, blockHeight, &nonce)
	ctx.Logger().Info(fmt.Sprintf("Closed worker nonce for topic: %d, nonce: %v", topic.Id, nonce))
	// Return an empty response as the operation was successful
	return nil
}

func closeActiveInferencesSet(
	ctx sdk.Context,
	k *keeper.Keeper,
	topicId uint64,
	nonce types.Nonce,
	activeInfererAddresses []string,
) (map[string]bool, error) {
	activeInferences := make([]*types.Inference, 0)
	activeInfererAddressesMap := make(map[string]bool, 0)
	for _, address := range activeInfererAddresses {
		inference, err := k.GetWorkerLatestInferenceByTopicId(ctx, topicId, address)
		if err != nil {
			return nil, err
		}
		activeInferences = append(activeInferences, &inference)
		activeInfererAddressesMap[inference.Inferer] = true
	}

	// Ensure deterministic ordering
	sort.Slice(activeInferences, func(i, j int) bool {
		return activeInferences[i].Inferer < activeInferences[j].Inferer
	})

	err := k.InsertActiveInferences(ctx, topicId, nonce.BlockHeight, types.Inferences{
		Inferences: activeInferences,
	})
	if err != nil {
		return nil, err
	}

	return activeInfererAddressesMap, nil
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
) error {
	forecastsByForecaster := make(map[string]*types.Forecast)
	activeForecasts := make([]*types.Forecast, 0)
	for _, address := range activeForecastAddresses {
		forecast, err := k.GetWorkerLatestForecastByTopicId(ctx, topicId, address)
		if err != nil {
			return err
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

		// Discard if empty
		if len(acceptedForecastElements) == 0 {
			continue
		}

		// Update the forecast with the filtered elements
		if forecast.ForecastElements != nil {
			forecast.ForecastElements = acceptedForecastElements
		}

		if forecast.Forecaster == "" {
			ctx.Logger().Warn("Forecast was added that has no forecaster, ignoring")
			continue
		}
		// Check that the forecast exist, is for the correct topic, and is for the correct nonce
		if forecast.TopicId != topicId {
			ctx.Logger().Warn("Forecast does not match topic: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
			continue
		}
		if forecast.BlockHeight != nonce.BlockHeight {
			ctx.Logger().Warn("Forecast does not match blockHeight: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
			continue
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

	return k.InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, types.Forecasts{
		Forecasts: activeForecasts,
	})
}
