package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Output a new set of inferences where only 1 inference per registerd inferer is kept,
// ignore the rest. In particular, take the first inference from each registered inferer
// and none from any unregistered inferer.
// Signatures, anti-synil procedures, and "skimming of only the top few workers by score
// descending" should be done here.
func (ms msgServer) VerifyAndInsertInferencesFromTopInferers(
	ctx context.Context,
	topicId uint64,
	nonce types.Nonce,
	inferences []*types.Inference,
	maxTopWorkersToReward uint64,
) (map[string]bool, error) {
	inferencesByInferer := make(map[string]*types.Inference)
	latestInfererScores := make(map[string]types.Score)
	for _, inference := range inferences {
		/// Do filters first, then consider the inferenes for inclusion
		/// Do filters on the per payload first, then on each inferer
		/// All filters should be done in order of increasing computational complexity

		// Check that the inference is for the correct topic
		if inference.TopicId != topicId {
			continue
		}

		// Check that the inference is for the correct nonce
		if inference.Nonce.Nonce != nonce.Nonce {
			continue
		}

		/// Now do filters on each inferer
		// Ensure that we only have one inference per inferer. If not, we just take the first one
		if _, ok := inferencesByInferer[inference.Inferer]; !ok {
			// Check if the inferer is registered
			isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, sdk.AccAddress(inference.Inferer))
			if err != nil {
				return nil, err
			}
			if !isInfererRegistered {
				continue
			}

			/// TODO check signatures! throw if invalid!

			/// If we do PoX-like anti-sybil procedure, would go here

			/// Filtering done now, now write what we must for inclusion

			inferencesByInferer[inference.Inferer] = inference

			// Get the latest score for each inferer => only take top few by score descending
			latestScore, err := ms.k.GetLatestInfererScore(ctx, topicId, sdk.AccAddress(inference.Inferer))
			if err != nil {
				return nil, err
			}
			latestInfererScores[inference.Inferer] = latestScore
		}
	}

	/// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topInferers := FindTopNByScoreDesc(maxTopWorkersToReward, latestInfererScores, nonce.Nonce)

	// Build list of inferences that pass all filters
	// AND are from top performing inferers among those who have submitted inferences in this batch
	inferencesFromTopInferers := make([]*types.Inference, 0)
	acceptedInferers := make(map[string]bool, 0)
	for worker, inference := range inferencesByInferer {
		if _, ok := topInferers[worker]; !ok {
			continue
		}

		acceptedInferers[worker] = true
		inferencesFromTopInferers = append(inferencesFromTopInferers, inference)
	}

	// Store the final list of inferences
	inferencesToInsert := types.Inferences{
		Inferences: inferencesFromTopInferers,
	}
	err := ms.k.InsertInferences(ctx, topicId, nonce, inferencesToInsert)
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
func (ms msgServer) VerifyAndInsertForecastsFromTopForecasters(
	ctx context.Context,
	topicId uint64,
	nonce types.Nonce,
	forecasts []*types.Forecast,
	// Inferers in the current batch, assumed to have passed VerifyAndInsertInferencesFromTopInferers() filters
	acceptedInferersOfBatch map[string]bool,
	maxTopWorkersToReward uint64,
) error {
	forecastsByForecaster := make(map[string]*types.Forecast)
	latestForecasterScores := make(map[string]types.Score)
	for _, forecast := range forecasts {
		/// Do filters first, then consider the inferenes for inclusion
		/// Do filters on the per payload first, then on each forecaster
		/// All filters should be done in order of increasing computational complexity

		// Check that the forecast is for the correct topic
		if forecast.TopicId != topicId {
			continue
		}

		// Check that the forecast is for the correct nonce
		if forecast.Nonce.Nonce != nonce.Nonce {
			continue
		}

		/// Now do filters on each forecaster
		// Ensure that we only have one forecast per forecaster. If not, we just take the first one
		if _, ok := forecastsByForecaster[forecast.Forecaster]; !ok {
			// Check if the forecaster is registered
			isForecasterRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, sdk.AccAddress(forecast.Forecaster))
			if err != nil {
				return err
			}
			if !isForecasterRegistered {
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

			/// TODO check signatures! throw if invalid!

			/// If we do PoX-like anti-sybil procedure, would go here

			/// Filtering done now, now write what we must for inclusion

			forecastsByForecaster[forecast.Forecaster] = forecast

			// Get the latest score for each forecaster => only take top few by score descending
			latestScore, err := ms.k.GetLatestForecasterScore(ctx, topicId, sdk.AccAddress(forecast.Forecaster))
			if err != nil {
				return err
			}
			latestForecasterScores[forecast.Forecaster] = latestScore
		}
	}

	/// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topForecasters := FindTopNByScoreDesc(maxTopWorkersToReward, latestForecasterScores, nonce.Nonce)

	// Build list of forecasts that pass all filters
	// AND are from top performing forecasters among those who have submitted forecasts in this batch
	forecastsFromTopForecasters := make([]*types.Forecast, 0)
	for worker, forecast := range forecastsByForecaster {
		if _, ok := topForecasters[worker]; !ok {
			continue
		}

		forecastsFromTopForecasters = append(forecastsFromTopForecasters, forecast)
	}

	// Store the final list of forecasts
	forecastsToInsert := types.Forecasts{
		Forecasts: forecastsFromTopForecasters,
	}
	err := ms.k.InsertForecasts(ctx, topicId, nonce, forecastsToInsert)
	if err != nil {
		return err
	}

	return nil
}

// A tx function that accepts a list of forecasts and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) InsertBulkWorkerPayload(ctx context.Context, msg *types.MsgInsertBulkWorkerPayload) (*types.MsgInsertBulkWorkerPayloadResponse, error) {
	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, msg.TopicId, msg.Nonce)
	if err != nil {
		return nil, err
	}
	if nonceUnfulfilled {
		return nil, types.ErrNonceNotUnfulfilled
	}

	maxTopWorkersToReward, err := ms.k.GetParamsMaxTopWorkersToReward(ctx)
	if err != nil {
		return nil, err
	}

	acceptedInferers, err := ms.VerifyAndInsertInferencesFromTopInferers(ctx, msg.TopicId, *msg.Nonce, msg.Inferences, maxTopWorkersToReward)
	if err != nil {
		return nil, err
	}

	err = ms.VerifyAndInsertForecastsFromTopForecasters(ctx, msg.TopicId, *msg.Nonce, msg.Forecasts, acceptedInferers, maxTopWorkersToReward)
	if err != nil {
		return nil, err
	}

	// Return an empty response as the operation was successful
	return &types.MsgInsertBulkWorkerPayloadResponse{}, nil
}
