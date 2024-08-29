package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Return low score and index among all inferences
func GetLowScoreFromAllLossBundles(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	lossBundles types.ReputerValueBundles,
) (types.Score, int, error) {
	lowScoreIndex := 0
	lowScore, err := k.GetReputerScoreEma(ctx, topicId, lossBundles.ReputerValueBundles[0].ValueBundle.Reputer)
	if err != nil {
		return types.Score{}, lowScoreIndex, err
	}
	for index, extLossBundle := range lossBundles.ReputerValueBundles {
		extScore, err := k.GetReputerScoreEma(ctx, topicId, extLossBundle.ValueBundle.Reputer)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(extScore.Score) {
			lowScore = extScore
			lowScoreIndex = index
		}
	}
	return lowScore, lowScoreIndex, nil
}

// Return low score and index among all inferences
func GetLowScoreFromAllInferences(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	inferences types.Inferences,
) (types.Score, int, error) {
	lowScoreIndex := 0
	lowScore, err := k.GetInfererScoreEma(ctx, topicId, inferences.Inferences[0].Inferer)
	if err != nil {
		return types.Score{}, lowScoreIndex, err
	}
	for index, extInference := range inferences.Inferences {
		extScore, err := k.GetInfererScoreEma(ctx, topicId, extInference.Inferer)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(extScore.Score) {
			lowScore = extScore
			lowScoreIndex = index
		}
	}
	return lowScore, lowScoreIndex, nil
}

// Return low score and index among all forecasts
func GetLowScoreFromAllForecasts(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	forecasts types.Forecasts,
) (types.Score, int, error) {
	lowScoreIndex := 0
	lowScore, err := k.GetForecasterScoreEma(ctx, topicId, forecasts.Forecasts[0].Forecaster)
	if err != nil {
		return types.Score{}, lowScoreIndex, err
	}
	for index, extForecast := range forecasts.Forecasts {
		extScore, err := k.GetForecasterScoreEma(ctx, topicId, extForecast.Forecaster)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(extScore.Score) {
			lowScore = extScore
			lowScoreIndex = index
		}
	}
	return lowScore, lowScoreIndex, nil
}
