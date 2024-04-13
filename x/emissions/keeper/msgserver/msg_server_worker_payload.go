package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ms msgServer) VerifyAndInsertInferencesFromTopInferers(
	ctx context.Context,
	topicId uint64,
	nonce types.Nonce,
	inferences []*types.Inference,
	maxTopWorkersToReward uint64,
) error {
	latestInfererScores := make(map[string]types.Score)
	for _, inference := range inferences {
		// TODO check signatures! throw if invalid!

		if inference.TopicId != topicId {
			return types.ErrInvalidTopicId
		}

		// Get the latest score for each inferer => only take top few by score descending
		latestScore, err := ms.k.GetLatestInfererScore(ctx, topicId, sdk.AccAddress(inference.Worker))
		if err != nil {
			return err
		}
		latestInfererScores[inference.Worker] = latestScore

		// If we do PoX-like anti-sybil procedure, would go here
	}

	// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topInferers := FindTopNByScoreDesc(maxTopWorkersToReward, latestInfererScores, nonce.Nonce)

	inferencesFromTopInferers := make([]*types.Inference, 0)
	for _, inference := range inferences {
		if _, ok := topInferers[inference.Worker]; !ok {
			continue
		}

		//
		// TODO check signatures! throw if invalid!
		//

		inferencesFromTopInferers = append(inferencesFromTopInferers, inference)
	}

	inferencesToInsert := types.Inferences{
		Inferences: inferencesFromTopInferers,
	}
	err := ms.k.InsertInferences(ctx, topicId, nonce, inferencesToInsert)
	if err != nil {
		return err
	}

	return nil
}

func (ms msgServer) VerifyAndInsertForecastsFromTopForecasters(
	ctx context.Context,
	topicId uint64,
	nonce types.Nonce,
	forecasts []*types.Forecast,
	maxTopWorkersToReward uint64,
) error {
	latestForecasterScores := make(map[string]types.Score)
	for _, forecast := range forecasts {
		if forecast.TopicId != topicId {
			return types.ErrInvalidTopicId
		}

		// Get the latest score for each inferer => only take top few by score descending
		latestScore, err := ms.k.GetLatestForecasterScore(ctx, topicId, sdk.AccAddress(forecast.Forecaster))
		if err != nil {
			return err
		}
		latestForecasterScores[forecast.Forecaster] = latestScore

		// If we do PoX-like anti-sybil procedure, would go here
	}

	// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topForecasters := FindTopNByScoreDesc(maxTopWorkersToReward, latestForecasterScores, nonce.Nonce)

	forecastsFromTopForecasters := make([]*types.Forecast, 0)
	for _, forecast := range forecasts {
		if _, ok := topForecasters[forecast.Forecaster]; !ok {
			continue
		}

		//
		// TODO check signatures! throw if invalid!
		//

		forecastsFromTopForecasters = append(forecastsFromTopForecasters, forecast)
	}

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

	err = ms.VerifyAndInsertInferencesFromTopInferers(ctx, msg.TopicId, *msg.Nonce, msg.Inferences, maxTopWorkersToReward)
	if err != nil {
		return nil, err
	}

	err = ms.VerifyAndInsertForecastsFromTopForecasters(ctx, msg.TopicId, *msg.Nonce, msg.Forecasts, maxTopWorkersToReward)
	if err != nil {
		return nil, err
	}

	// Return an empty response as the operation was successful
	return &types.MsgInsertBulkWorkerPayloadResponse{}, nil
}
