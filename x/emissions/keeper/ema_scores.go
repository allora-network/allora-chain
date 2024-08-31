package keeper

import (
	"context"

	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Calculates and saves the EMA scores for a given worker and topic
// Does nothing if the last update of the score was topic.WorkerSubmissionWindow blocks ago or less
// This is useful to ensure workers cannot game the system by spamming submissions to unfairly up their score
func (k *Keeper) CalcAndSaveInfererScoreEmaIfNewUpdate(
	ctx context.Context,
	topic types.Topic,
	block types.BlockHeight,
	worker ActorId,
	newScore types.Score,
) error {
	previousScore, err := k.GetInfererScoreEma(ctx, topic.Id, worker)
	if err != nil {
		return errors.Wrapf(err, "Error getting inferer score ema")
	}
	// Only calc and save if there's a new update
	if newScore.BlockHeight-previousScore.BlockHeight <= topic.WorkerSubmissionWindow {
		return nil
	}
	firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
	emaScoreDec, err := alloraMath.CalcEma(
		topic.MeritSortitionAlpha,
		newScore.Score,
		previousScore.Score,
		firstTime,
	)
	if err != nil {
		return errors.Wrapf(err, "Error calculating ema")
	}
	emaScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       emaScoreDec,
	}
	err = k.SetInfererScoreEma(ctx, topic.Id, worker, emaScore)
	if err != nil {
		return errors.Wrapf(err, "error setting latest inferer score")
	}
	return nil
}

// Calculates and saves the EMA scores for a given worker and topic
// Does nothing if the last update of the score was topic.WorkerSubmissionWindow blocks ago or less
// This is useful to ensure workers cannot game the system by spamming submissions to unfairly up their score
func (k *Keeper) CalcAndSaveForecasterScoreEmaIfNewUpdate(
	ctx context.Context,
	topic types.Topic,
	block types.BlockHeight,
	worker ActorId,
	newScore types.Score,
) error {
	previousScore, err := k.GetForecasterScoreEma(ctx, topic.Id, worker)
	if err != nil {
		return errors.Wrapf(err, "Error getting forecaster score ema")
	}
	// Only calc and save if there's a new update
	if newScore.BlockHeight-previousScore.BlockHeight <= topic.WorkerSubmissionWindow {
		return nil
	}
	firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
	emaScoreDec, err := alloraMath.CalcEma(
		topic.MeritSortitionAlpha,
		newScore.Score,
		previousScore.Score,
		firstTime,
	)
	if err != nil {
		return errors.Wrapf(err, "Error calculating ema")
	}
	emaScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       emaScoreDec,
	}
	err = k.SetForecasterScoreEma(ctx, topic.Id, worker, emaScore)
	if err != nil {
		return errors.Wrapf(err, "error setting latest forecaster score")
	}
	return nil
}

// Calculates and saves the EMA scores for a given reputer and topic
// Does nothing if the last update of the score was topic.EpochLength blocks ago or less
// This is useful to ensure reputers cannot game the system by spamming submissions to unfairly up their score
func (k *Keeper) CalcAndSaveReputerScoreEmaIfNewUpdate(
	ctx context.Context,
	topic types.Topic,
	block types.BlockHeight,
	worker ActorId,
	newScore types.Score,
) error {
	previousScore, err := k.GetReputerScoreEma(ctx, topic.Id, worker)
	if err != nil {
		return errors.Wrapf(err, "Error getting reputer score ema")
	}
	// Only calc and save if there's a new update
	if newScore.BlockHeight-previousScore.BlockHeight <= topic.EpochLength {
		return nil
	}
	firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
	emaScoreDec, err := alloraMath.CalcEma(
		topic.MeritSortitionAlpha,
		newScore.Score,
		previousScore.Score,
		firstTime,
	)
	if err != nil {
		return errors.Wrapf(err, "Error calculating ema")
	}
	emaScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       emaScoreDec,
	}
	err = k.SetReputerScoreEma(ctx, topic.Id, worker, emaScore)
	if err != nil {
		return errors.Wrapf(err, "error setting latest reputer score")
	}
	return nil
}
