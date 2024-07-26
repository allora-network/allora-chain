package actor_utils

import (
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

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Get all inferences from this topic, nonce
	inferences, err := k.GetInferencesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return err
	}

	acceptedInferers, err := insertInferencesFromTopInferers(
		ctx,
		k,
		topicId,
		nonce,
		inferences.Inferences,
		moduleParams.MaxTopInferersToReward,
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
		moduleParams.MaxTopForecastersToReward,
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

	err = k.SetTopicLastCommit(ctx, topic.Id, blockHeight, &nonce, types.ActorType_INFERER)
	if err != nil {
		return err
	}

	err = k.SetTopicLastWorkerPayload(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}

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
	maxTopWorkersToReward uint64,
) (map[string]bool, error) {
	inferencesByInferer := make(map[string]*types.Inference)
	latestInfererScores := make(map[string]types.Score)
	if len(inferences) == 0 {
		ctx.Logger().Warn("No inferences to process for topic: ", topicId, ", nonce: ", nonce)
		return nil, types.ErrNoValidBundles // TODO Change err name - No inferences to process
	}
	for _, inference := range inferences {
		if inference == nil {
			ctx.Logger().Warn("Inference was added that is nil, ignoring")
			continue
		}

		if inference.Inferer == "" {
			ctx.Logger().Warn("Inference was added that has no inferer, ignoring")
			continue
		}

		// Check if the topic and nonce are correct
		if inference.TopicId != topicId ||
			inference.BlockHeight != nonce.BlockHeight {
			ctx.Logger().Warn("Inference does not match topic: ", topicId, ", nonce: ", nonce, "for inferer: ", inference.Inferer)
			continue
		}

		/// Now do filters on each inferer
		// Ensure that we only have one inference per inferer. If not, we just take the first one
		if _, ok := inferencesByInferer[inference.Inferer]; !ok {
			// Check if the inferer is registered
			isInfererRegistered, err := k.IsWorkerRegisteredInTopic(ctx, topicId, inference.Inferer)
			if err != nil {
				ctx.Logger().Warn("Err checking inferer registration, topic: ", topicId, ", nonce: ", nonce, "for inferer: ", inference.Inferer)
				continue
			}
			if !isInfererRegistered {
				ctx.Logger().Warn("Inferer not registered, topic: ", topicId, ", nonce: ", nonce, "for inferer: ", inference.Inferer)
				continue
			}

			// Get the latest score for each inferer => only take top few by score descending
			latestScore, err := k.GetLatestInfererScore(ctx, topicId, inference.Inferer)
			if err != nil {
				ctx.Logger().Warn("Latest score not found, topic: ", topicId, ", nonce: ", nonce, "for inferer: ", inference.Inferer)
				continue
			}
			/// Filtering done now, now write what we must for inclusion
			latestInfererScores[inference.Inferer] = latestScore
			inferencesByInferer[inference.Inferer] = inference
		}
	}

	/// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topInferers := FindTopNByScoreDesc(maxTopWorkersToReward, latestInfererScores, nonce.BlockHeight)

	// Build list of inferences that pass all filters
	// AND are from top performing inferers among those who have submitted inferences in this batch
	inferencesFromTopInferers := make([]*types.Inference, 0)
	acceptedInferers := make(map[string]bool, 0)
	for _, worker := range topInferers {
		acceptedInferers[worker] = true
		inferencesFromTopInferers = append(inferencesFromTopInferers, inferencesByInferer[worker])
	}

	if len(inferencesFromTopInferers) == 0 {
		return nil, types.ErrNoValidInferences
	}

	// Ensure deterministic ordering of inferences
	sort.Slice(inferencesFromTopInferers, func(i, j int) bool {
		return inferencesFromTopInferers[i].Inferer < inferencesFromTopInferers[j].Inferer
	})

	// Store the final list of inferences
	inferencesToInsert := types.Inferences{
		Inferences: inferencesFromTopInferers,
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
	maxTopWorkersToReward uint64,
) error {
	forecastsByForecaster := make(map[string]*types.Forecast)
	latestForecasterScores := make(map[string]types.Score)
	for _, forecast := range forecasts {
		if forecast == nil {
			ctx.Logger().Warn("Forecast was added that is nil, ignoring")
			continue
		}

		if forecast.Forecaster == "" {
			ctx.Logger().Warn("Forecast was added that has no forecaster, ignoring")
			continue
		}
		// Check that the forecast exist, is for the correct topic, and is for the correct nonce
		if forecast.TopicId != topicId ||
			forecast.BlockHeight != nonce.BlockHeight {
			ctx.Logger().Warn("Forecast does not match topic: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
			continue
		}

		/// Now do filters on each forecaster
		// Ensure that we only have one forecast per forecaster. If not, we just take the first one
		if _, ok := forecastsByForecaster[forecast.Forecaster]; !ok {
			// Check if the forecaster is registered
			isForecasterRegistered, err := k.IsWorkerRegisteredInTopic(ctx, topicId, forecast.Forecaster)
			if err != nil {
				ctx.Logger().Warn("Error checking forecaster registration: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
				continue
			}
			if !isForecasterRegistered {
				ctx.Logger().Warn("Forecaster not registered, topic: ", topicId, ", nonce: ", nonce, "for forecaster: ", forecast.Forecaster)
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

			// Discard if empty
			if len(acceptedForecastElements) == 0 {
				continue
			}

			/// Filtering done now, now write what we must for inclusion

			// Get the latest score for each forecaster => only take top few by score descending
			latestScore, err := k.GetLatestForecasterScore(ctx, topicId, forecast.Forecaster)
			if err != nil {
				continue
			}
			latestForecasterScores[forecast.Forecaster] = latestScore
			forecastsByForecaster[forecast.Forecaster] = forecast
		}
	}

	/// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topForecasters := FindTopNByScoreDesc(maxTopWorkersToReward, latestForecasterScores, nonce.BlockHeight)

	// Build list of forecasts that pass all filters
	// AND are from top performing forecasters among those who have submitted forecasts in this batch
	forecastsFromTopForecasters := make([]*types.Forecast, 0)
	for _, worker := range topForecasters {
		forecastsFromTopForecasters = append(forecastsFromTopForecasters, forecastsByForecaster[worker])
	}

	// Though less than ideal because it produces less-acurate network inferences,
	// it is fine if no forecasts are accepted
	// => no need to check len(forecastsFromTopForecasters) == 0

	// Ensure deterministic ordering
	sort.Slice(forecastsFromTopForecasters, func(i, j int) bool {
		return forecastsFromTopForecasters[i].Forecaster < forecastsFromTopForecasters[j].Forecaster
	})
	// Store the final list of forecasts
	forecastsToInsert := types.Forecasts{
		Forecasts: forecastsFromTopForecasters,
	}
	err := k.InsertForecasts(ctx, topicId, nonce, forecastsToInsert)
	if err != nil {
		return err
	}

	return nil
}
