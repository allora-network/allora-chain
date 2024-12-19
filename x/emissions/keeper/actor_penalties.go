package keeper

import (
	"cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ApplyLivenessPenaltyToInferer penalises an inferer for missing previous epochs. It saves and returns the new EMA score.
// If the inferer didn't miss any epochs this is a no-op, the EMA score is returned as is.
func (k *Keeper) ApplyLivenessPenaltyToInferer(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight types.BlockHeight,
	emaScore types.Score,
) (types.Score, error) {
	return ApplyLivenessPenaltyToActor(
		ctx,
		CountWorkerContiguousMissedEpochs,
		func(topicId TopicId) (alloraMath.Dec, error) {
			return k.GetTopicInitialInfererEmaScore(ctx, topicId)
		},
		func(topicId TopicId, score types.Score) error {
			return k.SetInfererScoreEma(ctx, topicId, score.Address, score)
		},
		topic,
		nonceBlockHeight,
		emaScore,
	)
}

// ApplyLivenessPenaltyToForecaster penalises a forecaster for missing previous epochs. It saves and returns the new EMA score.
// If the forecaster didn't miss any epochs this is a no-op, the EMA score is returned as is.
func (k *Keeper) ApplyLivenessPenaltyToForecaster(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight types.BlockHeight,
	emaScore types.Score,
) (types.Score, error) {
	return ApplyLivenessPenaltyToActor(
		ctx,
		CountWorkerContiguousMissedEpochs,
		func(topicId TopicId) (alloraMath.Dec, error) {
			return k.GetTopicInitialForecasterEmaScore(ctx, topicId)
		},
		func(topicId TopicId, score types.Score) error {
			return k.SetForecasterScoreEma(ctx, topicId, score.Address, score)
		},
		topic,
		nonceBlockHeight,
		emaScore,
	)
}

// ApplyLivenessPenaltyToReputer penalises a reputer for missing previous epochs. It saves and returns the new EMA score.
// If the reputer didn't miss any epochs this is a no-op, the EMA score is returned as is.
func (k *Keeper) ApplyLivenessPenaltyToReputer(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight types.BlockHeight,
	emaScore types.Score,
) (types.Score, error) {
	return ApplyLivenessPenaltyToActor(
		ctx,
		CountReputerContiguousMissedEpochs,
		func(topicId TopicId) (alloraMath.Dec, error) {
			return k.GetTopicInitialReputerEmaScore(ctx, topicId)
		},
		func(topicId TopicId, score types.Score) error {
			return k.SetReputerScoreEma(ctx, topicId, score.Address, score)
		},
		topic,
		nonceBlockHeight,
		emaScore,
	)
}

func ApplyLivenessPenaltyToActor(
	ctx sdk.Context,
	missedEpochsFn func(topic types.Topic, lastSubmittedNonce int64) int64,
	getAsymptoteFn func(topicId TopicId) (alloraMath.Dec, error),
	setScoreFn func(topicId TopicId, score types.Score) error,
	topic types.Topic,
	nonceBlockHeight types.BlockHeight,
	emaScore types.Score,
) (types.Score, error) {
	missedEpochs := missedEpochsFn(topic, emaScore.BlockHeight)
	// No missed epochs == no penalty
	if missedEpochs == 0 {
		return emaScore, nil
	}

	penalty, err := getAsymptoteFn(topic.Id)
	if err != nil {
		return types.Score{}, err
	}
	emaScore.BlockHeight = nonceBlockHeight

	beforePenalty := emaScore
	emaScore.Score, err = applyPenalty(topic, penalty, emaScore.Score, missedEpochs)
	if err != nil {
		return types.Score{}, err
	}

	ctx.Logger().Debug("apply liveness penalty on actor",
		"nonce", nonceBlockHeight,
		"penalty", penalty,
		"before", beforePenalty,
		"after", emaScore,
	)

	// Save the penalised EMA score
	return emaScore, setScoreFn(topic.Id, emaScore)
}

// applyPenalty applies the penalty to the EMA score for the given number of missed epochs while staying above provided limit.
func applyPenalty(topic types.Topic, penalty, emaScore alloraMath.Dec, missedEpochs int64) (alloraMath.Dec, error) {
	return alloraMath.NCalcEma(topic.MeritSortitionAlpha, penalty, emaScore, uint64(missedEpochs))
}

// CountWorkerContiguousMissedEpochs counts the number of contiguous missed epochs prior to the given nonce, given the
// actor last submission.
func CountWorkerContiguousMissedEpochs(topic types.Topic, lastSubmittedNonce int64) int64 {
	prevEpochStart := topic.EpochLastEnded - topic.EpochLength
	return countContiguousMissedEpochs(prevEpochStart, topic.EpochLength, lastSubmittedNonce)
}

// CountReputerContiguousMissedEpochs counts the number of contiguous missed epochs prior to the given nonce, given the
// actor last submission.
func CountReputerContiguousMissedEpochs(topic types.Topic, lastSubmittedNonce int64) int64 {
	prevEpochStart := topic.EpochLastEnded - topic.EpochLength - topic.GroundTruthLag
	return countContiguousMissedEpochs(prevEpochStart, topic.EpochLength, lastSubmittedNonce)
}

func countContiguousMissedEpochs(prevEpochStart, epochLength, lastSubmittedNonce int64) int64 {
	lastSubmittedNonce = math.Max(lastSubmittedNonce, 0)
	prevEpochStart = math.Max(prevEpochStart, 0)
	epochLength = math.Max(epochLength, 0)

	if lastSubmittedNonce >= prevEpochStart {
		return 0
	}

	return (prevEpochStart-1-lastSubmittedNonce)/epochLength + 1
}
