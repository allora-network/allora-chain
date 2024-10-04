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
) (lowScore types.Score, lowScoreIndex int, err error) {
	lowScoreIndex = 0
	lowScore, err = k.GetReputerScoreEma(ctx, topicId, lossBundles.ReputerValueBundles[0].ValueBundle.Reputer)
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

// Update lowest score from new inferer addresses set
func UpdateLowestScoreFromInfererAddresses(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	infererAddresses []string,
	addedInferer string,
	removedInfererAddress string,
) error {
	infererAddresses = append(infererAddresses, addedInferer)
	lowScore := types.Score{}
	for i, address := range infererAddresses {
		if address == removedInfererAddress {
			continue
		}
		score, err := k.GetInfererScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return k.SetLowestInfererScoreEma(ctx, topicId, lowScore)
}

// Get lowest score from all inferers
func GetLowestScoreFromAllInferers(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	infererAddresses []string,
) (lowScore types.Score, err error) {
	for i, address := range infererAddresses {
		score, err := k.GetInfererScoreEma(ctx, topicId, address)
		if err != nil {
			continue
		}
		if lowScore.Score.Gt(score.Score) || i == 0 {
			lowScore = score
		}
	}
	return lowScore, nil
}

// Return low score and index among all forecasts
func GetLowScoreFromAllForecasts(
	ctx context.Context,
	k *Keeper,
	topicId TopicId,
	forecasts types.Forecasts,
) (lowScore types.Score, lowScoreIndex int, err error) {
	lowScoreIndex = 0
	lowScore, err = k.GetForecasterScoreEma(ctx, topicId, forecasts.Forecasts[0].Forecaster)
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
