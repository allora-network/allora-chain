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
	reputer ActorId,
	newScore types.Score,
) error {
	previousScore, err := k.GetReputerScoreEma(ctx, topic.Id, reputer)
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
		Address:     reputer,
		Score:       emaScoreDec,
	}
	err = k.SetReputerScoreEma(ctx, topic.Id, reputer, emaScore)
	if err != nil {
		return errors.Wrapf(err, "error setting latest reputer score")
	}
	return nil
}

// Calculates and saves the EMA scores for a given worker and topic
// Uses the last saved topic quantile score to calculate the EMA
func (k *Keeper) CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(
	ctx context.Context,
	topic types.Topic,
	block types.BlockHeight,
	worker ActorId,
) error {
	previousScore, err := k.GetInfererScoreEma(ctx, topic.Id, worker)
	if err != nil {
		return errors.Wrapf(err, "Error getting inferer score ema")
	}
	previousTopicQuantileInfererScoreEma, err := k.GetPreviousTopicQuantileInfererScoreEma(ctx, topic.Id)
	if err != nil {
		return err
	}
	firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
	emaScoreDec, err := alloraMath.CalcEma(
		topic.MeritSortitionAlpha,
		previousTopicQuantileInfererScoreEma,
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
// Uses the last saved topic quantile score to calculate the EMA
func (k *Keeper) CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(
	ctx context.Context,
	topic types.Topic,
	block types.BlockHeight,
	worker ActorId,
) error {
	previousScore, err := k.GetForecasterScoreEma(ctx, topic.Id, worker)
	if err != nil {
		return errors.Wrapf(err, "Error getting forecaster score ema")
	}
	previousTopicQuantileForecasterScoreEma, err := k.GetPreviousTopicQuantileForecasterScoreEma(ctx, topic.Id)
	if err != nil {
		return err
	}
	firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
	emaScoreDec, err := alloraMath.CalcEma(
		topic.MeritSortitionAlpha,
		previousTopicQuantileForecasterScoreEma,
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
// Uses the last saved topic quantile score to calculate the EMA
func (k *Keeper) CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(
	ctx context.Context,
	topic types.Topic,
	block types.BlockHeight,
	reputer ActorId,
) error {
	previousScore, err := k.GetReputerScoreEma(ctx, topic.Id, reputer)
	if err != nil {
		return errors.Wrapf(err, "Error getting reputer score ema")
	}
	// Only calc and save if there's a new update
	previousTopicQuantileReputerScoreEma, err := k.GetPreviousTopicQuantileReputerScoreEma(ctx, topic.Id)
	if err != nil {
		return err
	}
	firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
	emaScoreDec, err := alloraMath.CalcEma(
		topic.MeritSortitionAlpha,
		previousTopicQuantileReputerScoreEma,
		previousScore.Score,
		firstTime,
	)
	if err != nil {
		return errors.Wrapf(err, "Error calculating ema")
	}
	emaScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     reputer,
		Score:       emaScoreDec,
	}
	err = k.SetReputerScoreEma(ctx, topic.Id, reputer, emaScore)
	if err != nil {
		return errors.Wrapf(err, "error setting latest reputer score")
	}
	return nil
}
