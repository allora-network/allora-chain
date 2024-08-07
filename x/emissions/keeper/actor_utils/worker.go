package actor_utils

import (
	"fmt"
	"sort"

	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WORKER NONCES CLOSING

// Closes an open worker nonce.
func CloseWorkerNonce(k *keeper.Keeper, ctx sdk.Context, topicId keeper.TopicId, nonce types.Nonce) error {
	// Check if the topic exists
	topicExists, err := k.TopicExists(ctx, topicId)
	if err != nil {
		return err
	}
	if !topicExists {
		return types.ErrInvalidTopicId
	}

	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topicId, &nonce)
	if err != nil {
		return err
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return types.ErrUnfulfilledNonceNotFound
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return types.ErrInvalidTopicId
	}

	// Check if the window time has passed: if blockheight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	blockHeight := ctx.BlockHeight()
	if blockHeight <= topic.EpochLastEnded ||
		blockHeight > topic.EpochLastEnded+topic.GroundTruthLag {
		return types.ErrWorkerNonceWindowNotAvailable
	}

	// Get all inferences from this topic, nonce
	inferences, err := k.GetInferencesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return err
	}
	if len(inferences.Inferences) == 0 {
		return types.ErrNoValidInferences
	}

	acceptedInferers, err := insertInferencesFromTopInferers(
		ctx,
		k,
		topicId,
		nonce,
		inferences.Inferences,
	)
	if err != nil {
		return err
	}

	// Get all forecasts from this topicId, nonce
	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return err
	}

	err = insertForecastsFromTopForecasters(
		ctx,
		k,
		topicId,
		nonce,
		forecasts.Forecasts,
		acceptedInferers,
	)
	if err != nil {
		return err
	}
	// Update the unfulfilled worker nonce
	_, err = k.FulfillWorkerNonce(ctx, topicId, &nonce)
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

	ctx.Logger().Info(fmt.Sprintf("Closed worker nonce for topic: %d, nonce: %v", topicId, nonce))
	// Return an empty response as the operation was successful
	return nil
}

// Output a new set of inferences where only 1 inference per registerd inferer is kept,
// ignore the rest. In particular, take the first inference from each registered inferer
// and none from any unregistered inferer.
// Signatures, anti-synil procedures, and "skimming of only the top few workers by score
// descending" should be done here.
func insertInferencesFromTopInferers(
	ctx sdk.Context,
	k *keeper.Keeper,
	topicId uint64,
	nonce types.Nonce,
	inferences []*types.Inference,
) (map[string]bool, error) {
	acceptedInferers := make(map[string]bool, 0)
	if len(inferences) == 0 {
		ctx.Logger().Warn(fmt.Sprintf("No inferences to process for topic: %d, nonce: %v", topicId, nonce))
		return nil, types.ErrNoValidInferences // TODO Change err name - No inferences to process
	}
	for _, inference := range inferences {
		// Check that the forecast exist, is for the correct topic, and is for the correct nonce
		if inference.TopicId != topicId {
			ctx.Logger().Warn("Inference does not match topic: ", topicId, ", nonce: ", nonce, "for inferer: ", inference.Inferer)
			continue
		}
		if inference.BlockHeight != nonce.BlockHeight {
			ctx.Logger().Warn("Inference does not match blockHeight: ", topicId, ", nonce: ", nonce, "for inferer: ", inference.Inferer)
			continue
		}
		acceptedInferers[inference.Inferer] = true
	}

	// Ensure deterministic ordering of inferences
	sort.Slice(inferences, func(i, j int) bool {
		return inferences[i].Inferer < inferences[j].Inferer
	})

	// Store the final list of inferences
	inferencesToInsert := types.Inferences{
		Inferences: inferences,
	}
	err := k.InsertInferences(ctx, topicId, nonce, inferencesToInsert)
	if err != nil {
		return nil, err
	}

	return acceptedInferers, nil
}

// Output a new set of forecasts where only 1 forecast per registerd forecaster is kept,
// ignore the rest. In particular, take the first forecast from each registered forecaster
// and none from any unregistered forecaster.
// Signatures, anti-synil procedures, and "skimming of only the top few workers by score
// descending" should be done here.
func insertForecastsFromTopForecasters(
	ctx sdk.Context,
	k *keeper.Keeper,
	topicId uint64,
	nonce types.Nonce,
	forecasts []*types.Forecast,
	acceptedInferersOfBatch map[string]bool,
) error {
	forecastsByForecaster := make(map[string]*types.Forecast)
	latestForecaster := make([]*types.Forecast, 0)
	for _, forecast := range forecasts {
		// Examine forecast elements to verify that they're for inferers in the current set.
		// We assume that set of inferers has been verified above.
		// We keep what we can, ignoring the forecaster and their contribution (forecast) entirely
		// if they're left with no valid forecast elements.
		acceptedForecastElements := make([]*types.ForecastElement, 0)
		seenInferers := make(map[string]bool)
		for _, el := range forecast.ForecastElements {
			if _, ok := acceptedInferersOfBatch[el.Inferer]; ok && !seenInferers[el.Inferer] {
				acceptedForecastElements = append(acceptedForecastElements, el)
				seenInferers[el.Inferer] = true
			}
		}

		// Discard if empty
		if len(acceptedForecastElements) == 0 {
			continue
		}

		// Update the forecast with the filtered elements
		forecast.ForecastElements = acceptedForecastElements

		if forecast == nil {
			ctx.Logger().Warn("Forecast was added that is nil, ignoring")
			continue
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
			latestForecaster = append(latestForecaster, forecast)
			forecastsByForecaster[forecast.Forecaster] = forecast
		}
	}

	// Though less than ideal because it produces less-acurate network inferences,
	// it is fine if no forecasts are accepted
	// => no need to check len(forecastsFromTopForecasters) == 0

	// Ensure deterministic ordering
	sort.Slice(latestForecaster, func(i, j int) bool {
		return latestForecaster[i].Forecaster < latestForecaster[j].Forecaster
	})
	// Store the final list of forecasts
	forecastsToInsert := types.Forecasts{
		Forecasts: latestForecaster,
	}
	err := k.InsertForecasts(ctx, topicId, nonce, forecastsToInsert)
	if err != nil {
		return err
	}

	return nil
}
