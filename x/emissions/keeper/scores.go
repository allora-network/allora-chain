package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewScoresKeeper(cdc codec.BinaryCodec, sb *collections.SchemaBuilder, paramsKeeper *ParamsKeeper) *ScoresKeeper {
	return &ScoresKeeper{
		infererScoresByBlock:                     collections.NewMap(sb, types.InferenceScoresKey, "inferer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		forecasterScoresByBlock:                  collections.NewMap(sb, types.ForecastScoresKey, "forecaster_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerScoresByBlock:                     collections.NewMap(sb, types.ReputerScoresKey, "reputer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		infererScoreEmas:                         collections.NewMap(sb, types.InfererScoreEmasKey, "latest_inferer_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		forecasterScoreEmas:                      collections.NewMap(sb, types.ForecasterScoreEmasKey, "latest_forecaster_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		reputerScoreEmas:                         collections.NewMap(sb, types.ReputerScoreEmasKey, "latest_reputer_scores_by_reputer", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		reputerListeningCoefficient:              collections.NewMap(sb, types.ReputerListeningCoefficientKey, "reputer_listening_coefficient", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.ListeningCoefficient](cdc)),
		previousReputerRewardFraction:            collections.NewMap(sb, types.PreviousReputerRewardFractionKey, "previous_reputer_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousInferenceRewardFraction:          collections.NewMap(sb, types.PreviousInferenceRewardFractionKey, "previous_inference_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecastRewardFraction:           collections.NewMap(sb, types.PreviousForecastRewardFractionKey, "previous_forecast_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecasterScoreRatio:             collections.NewMap(sb, types.PreviousForecasterScoreRatioKey, "previous_forecaster_score_ratio", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileInfererScoreEma:     collections.NewMap(sb, types.PreviousTopicQuantileInfererScoreEmaKey, "previous_topic_quantile_inferer_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileForecasterScoreEma:  collections.NewMap(sb, types.PreviousTopicQuantileForecasterScoreEmaKey, "previous_topic_quantile_forecaster_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileReputerScoreEma:     collections.NewMap(sb, types.PreviousTopicQuantileReputerScoreEmaKey, "previous_topic_quantile_reputer_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousPercentageRewardToStakedReputers: collections.NewItem(sb, types.PreviousPercentageRewardToStakedReputersKey, "previous_percentage_reward_to_staked_reputers", alloraMath.DecValue),
		initialInfererEmaScore:                   collections.NewMap(sb, types.InitialInfererEmaScoreKey, "initial_inferer_ema_score", collections.Uint64Key, alloraMath.DecValue),
		initialForecasterEmaScore:                collections.NewMap(sb, types.InitialForecasterEmaScoreKey, "initial_forecaster_ema_score", collections.Uint64Key, alloraMath.DecValue),
		initialReputerEmaScore:                   collections.NewMap(sb, types.InitialReputerEmaScoreKey, "initial_reputer_ema_score", collections.Uint64Key, alloraMath.DecValue),
		lowestReputerScoreEma:                    collections.NewMap(sb, types.LowestReputerScoreEmaKey, "lowest_reputer_score_ema", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		lowestInfererScoreEma:                    collections.NewMap(sb, types.LowestInfererScoreEmaKey, "lowest_inferer_score_ema", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		lowestForecasterScoreEma:                 collections.NewMap(sb, types.LowestForecasterScoreEmaKey, "lowest_forecaster_score_ema", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		paramsKeeper:                             paramsKeeper,
	}
}

type ScoresKeeper struct {
	// map of (topic, block_height, worker) -> score
	infererScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, worker) -> score
	forecasterScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, reputer) -> score
	reputerScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, worker) -> score
	infererScoreEmas collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, worker) -> score
	forecasterScoreEmas collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, reputer) -> score
	reputerScoreEmas collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, reputer) -> listening coefficient
	reputerListeningCoefficient collections.Map[collections.Pair[TopicId, ActorId], types.ListeningCoefficient]
	// map of (topic, reputer) -> previous reward (used for EMA)
	previousReputerRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for inference (used for EMA)
	previousInferenceRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for forecast (used for EMA)
	previousForecastRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of topic -> previous forecaster score ratio
	previousForecasterScoreRatio collections.Map[TopicId, alloraMath.Dec]
	// previous topic inferer ema score at topic quantile
	previousTopicQuantileInfererScoreEma collections.Map[TopicId, alloraMath.Dec]
	// previous topic forecaster ema score at topic quantile
	previousTopicQuantileForecasterScoreEma collections.Map[TopicId, alloraMath.Dec]
	// previous topic reputer ema score at topic quantile
	previousTopicQuantileReputerScoreEma collections.Map[TopicId, alloraMath.Dec]
	// Percentage of all rewards, paid out to staked reputers, during the previous reward cadence. Used by mint module
	previousPercentageRewardToStakedReputers collections.Item[alloraMath.Dec]
	// Initial EMA scores for inferers, forecasters, and reputers
	initialInfererEmaScore    collections.Map[TopicId, alloraMath.Dec]
	initialForecasterEmaScore collections.Map[TopicId, alloraMath.Dec]
	initialReputerEmaScore    collections.Map[TopicId, alloraMath.Dec]
	// lowest reputer score ema for a topic
	lowestReputerScoreEma collections.Map[TopicId, types.Score]
	// lowest topic inferer score ema for a topic
	lowestInfererScoreEma collections.Map[TopicId, types.Score]
	// lowest topic forecaster score ema for a topic
	lowestForecasterScoreEma collections.Map[TopicId, types.Score]
	// params keeper
	paramsKeeper *ParamsKeeper
}

// If the new score is older than the current score, don't update
func (k *ScoresKeeper) SetInfererScoreEma(ctx context.Context, topicId TopicId, worker ActorId, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetInfererScoreEma: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetInfererScoreEma: Error validating worker")
	}
	if err := score.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetInfererScoreEma: Error validating inferer score")
	}
	key := collections.Join(topicId, worker)
	return k.infererScoreEmas.Set(ctx, key, score)
}

func (k *ScoresKeeper) GetInfererScoreEma(ctx context.Context, topicId TopicId, worker ActorId) (types.Score, error) {
	key := collections.Join(topicId, worker)
	score, err := k.infererScoreEmas.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return types.Score{
			BlockHeight: 0,
			Address:     worker,
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, nil
	} else if err != nil {
		return types.Score{}, errorsmod.Wrap(err, "error getting inferer score ema")
	}
	return score, nil
}

func (k *ScoresKeeper) SetForecasterScoreEma(ctx context.Context, topicId TopicId, worker ActorId, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetForecasterScoreEma: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetForecasterScoreEma: Error validating worker")
	}
	if err := score.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetForecasterScoreEma: Error validating forecaster score")
	}
	key := collections.Join(topicId, worker)
	return k.forecasterScoreEmas.Set(ctx, key, score)
}

func (k *ScoresKeeper) GetForecasterScoreEma(ctx context.Context, topicId TopicId, worker ActorId) (types.Score, error) {
	key := collections.Join(topicId, worker)
	score, err := k.forecasterScoreEmas.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return types.Score{
			BlockHeight: 0,
			Address:     worker,
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, nil
	} else if err != nil {
		return types.Score{}, errorsmod.Wrap(err, "error getting forecaster score ema")
	}
	return score, nil
}

// If the new score is older than the current score, don't update
func (k *ScoresKeeper) SetReputerScoreEma(ctx context.Context, topicId TopicId, reputer ActorId, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetReputerScoreEma: Error validating topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrapf(err, "SetReputerScoreEma: Error validating reputer")
	}
	if err := score.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetReputerScoreEma: Error validating reputer score")
	}
	key := collections.Join(topicId, reputer)
	return k.reputerScoreEmas.Set(ctx, key, score)
}

func (k *ScoresKeeper) GetReputerScoreEma(ctx context.Context, topicId TopicId, reputer ActorId) (types.Score, error) {
	key := collections.Join(topicId, reputer)
	score, err := k.reputerScoreEmas.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Score{
			BlockHeight: 0,
			Address:     reputer,
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, nil
	} else if err != nil {
		return types.Score{
			BlockHeight: 0,
			Address:     reputer,
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, errorsmod.Wrap(err, "error getting reputer score ema")
	}
	return score, nil
}

func (k *ScoresKeeper) InsertWorkerInferenceScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerInferenceScore: Error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerInferenceScore: Error validating block height")
	}
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting worker inference scores at block")
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting params")
	}

	maxNumScores := moduleParams.MaxSamplesToScaleScores * moduleParams.MaxTopInferersToReward

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockHeight)
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerInferenceScore: Error validating worker inference scores")
	}
	return k.infererScoresByBlock.Set(ctx, key, scores)
}

func (k *ScoresKeeper) SetInfererScoresByBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight, scores types.Scores) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error setting infererScoresByBlock")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrap(err, "error setting infererScoresByBlock")
	}
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrap(err, "error setting infererScoresByBlock")
	}
	if err := k.infererScoresByBlock.Set(ctx,
		collections.Join(topicId, blockHeight),
		scores); err != nil {
		return errorsmod.Wrap(err, "error setting infererScoresByBlock")
	}
	return nil
}

func (k *ScoresKeeper) GetInferenceScoresUntilBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight).
		Descending()

	iter, err := k.infererScoresByBlock.Iterate(ctx, rng)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error iterating inferer scores by block")
	}
	defer iter.Close()

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting params")
	}

	maxNumScores := moduleParams.MaxSamplesToScaleScores * moduleParams.MaxTopInferersToReward

	scores := make([]*types.Score, 0, maxNumScores)
	for iter.Valid() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}

		for _, score := range existingScores.Value.Scores {
			if uint64(len(scores)) < maxNumScores {
				scores = append(scores, score)
			} else {
				break
			}
		}
		if uint64(len(scores)) >= maxNumScores {
			break
		}
		iter.Next()
	}

	return scores, nil
}

func (k *ScoresKeeper) GetWorkerInferenceScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.infererScoresByBlock.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Scores{
			Scores: []*types.Score{},
		}, nil
	} else if err != nil {
		return types.Scores{}, errorsmod.Wrap(err, "error getting worker inference scores at block")
	}
	return scores, nil
}

func (k *ScoresKeeper) InsertWorkerForecastScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerForecastScore: Error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerForecastScore: Error validating block height")
	}
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrap(err, "error getting worker forecast scores at block")
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}

	maxNumScores := moduleParams.MaxSamplesToScaleScores * moduleParams.MaxTopForecastersToReward

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockHeight)
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerForecastScore: Error validating worker forecast scores")
	}
	return k.forecasterScoresByBlock.Set(ctx, key, scores)
}

func (k *ScoresKeeper) SetForecasterScoresByBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight, scores types.Scores) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error setting forecasterScoresByBlock")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrap(err, "error setting forecasterScoresByBlock")
	}
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrap(err, "error setting forecasterScoresByBlock")
	}
	if err := k.forecasterScoresByBlock.Set(ctx, collections.Join(topicId, blockHeight), scores); err != nil {
		return errorsmod.Wrap(err, "error setting forecasterScoresByBlock")
	}
	return nil
}

func (k *ScoresKeeper) GetForecastScoresUntilBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight).
		Descending()

	iter, err := k.forecasterScoresByBlock.Iterate(ctx, rng)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error iterating forecaster scores by block")
	}
	defer iter.Close()

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting params")
	}

	maxNumScores := moduleParams.MaxSamplesToScaleScores * moduleParams.MaxTopForecastersToReward

	scores := make([]*types.Score, 0, maxNumScores)
	for iter.Valid() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}

		for _, score := range existingScores.Value.Scores {
			if uint64(len(scores)) < maxNumScores {
				scores = append(scores, score)
			} else {
				break
			}
		}
		if uint64(len(scores)) >= maxNumScores {
			break
		}
		iter.Next()
	}

	return scores, nil
}

func (k *ScoresKeeper) GetWorkerForecastScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.forecasterScoresByBlock.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Scores{Scores: []*types.Score{}}, nil
	} else if err != nil {
		return types.Scores{}, errorsmod.Wrap(err, "error getting worker forecast scores at block")
	}
	return scores, nil
}

func (k *ScoresKeeper) InsertReputerScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "InsertReputerScore: Error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrapf(err, "InsertReputerScore: Error validating block height")
	}
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrap(err, "error getting reputers scores at block")
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}
	maxNumScores := moduleParams.MaxSamplesToScaleScores
	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		scores.Scores = scores.Scores[lenScores-maxNumScores:]
	}
	key := collections.Join(topicId, blockHeight)
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrapf(err, "InsertReputerScore: Error validating reputer scores")
	}
	return k.reputerScoresByBlock.Set(ctx, key, scores)
}

func (k *ScoresKeeper) GetReputersScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.reputerScoresByBlock.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		return types.Scores{Scores: []*types.Score{}}, nil
	} else if err != nil {
		return types.Scores{}, errorsmod.Wrap(err, "error getting reputers scores at block")
	}
	return scores, nil
}

func (k *ScoresKeeper) SetListeningCoefficient(ctx context.Context, topicId TopicId, reputer ActorId, coefficient types.ListeningCoefficient) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetListeningCoefficient: Error validating topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrapf(err, "SetListeningCoefficient: Error validating reputer")
	}
	if err := coefficient.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetListeningCoefficient: Error validating listening coefficient")
	}
	key := collections.Join(topicId, reputer)
	return k.reputerListeningCoefficient.Set(ctx, key, coefficient)
}

func (k *ScoresKeeper) GetListeningCoefficient(ctx context.Context, topicId TopicId, reputer ActorId) (types.ListeningCoefficient, error) {
	key := collections.Join(topicId, reputer)
	coef, err := k.reputerListeningCoefficient.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		// Return a default value
		return types.ListeningCoefficient{Coefficient: alloraMath.NewDecFromInt64(1)}, nil
	} else if err != nil {
		return types.ListeningCoefficient{}, errorsmod.Wrap(err, "error getting listening coefficient")
	}
	return coef, nil
}

func (k *ScoresKeeper) SetPreviousTopicQuantileInfererScoreEma(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileInfererScoreEma: Error validating topic id")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileInfererScoreEma: Error validating score")
	}
	return k.previousTopicQuantileInfererScoreEma.Set(ctx, topicId, score)
}

// Gets the previous Inferer Score Ema at Topic quantile
// Returns previous inferer score ema at topic quantile, or 0 if not yet seen
func (k *ScoresKeeper) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.previousTopicQuantileInfererScoreEma.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous topic quantile inferer score ema")
	}
	return score, nil
}

func (k *ScoresKeeper) SetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileForecasterScoreEma: Error validating topic id")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileForecasterScoreEma: Error validating score")
	}
	return k.previousTopicQuantileForecasterScoreEma.Set(ctx, topicId, score)
}

// Gets the previous Forecaster Score Ema at Topic quantile
// Returns previous forecaster score ema at topic quantile, or 0 if not yet seen
func (k *ScoresKeeper) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.previousTopicQuantileForecasterScoreEma.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous topic quantile forecaster score ema")
	}
	return score, nil
}

func (k *ScoresKeeper) SetPreviousTopicQuantileReputerScoreEma(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileReputerScoreEma: Error validating topic id")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileReputerScoreEma: Error validating score")
	}
	return k.previousTopicQuantileReputerScoreEma.Set(ctx, topicId, score)
}

// Gets the previous Reputer Score Ema at Topic quantile
// Returns previous reputer score ema at topic quantile, or 0 if not yet seen
func (k *ScoresKeeper) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.previousTopicQuantileReputerScoreEma.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous topic quantile reputer score ema")
	}
	return score, nil
}

// / REWARD FRACTION

// Gets the previous W_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *ScoresKeeper) GetPreviousReputerRewardFraction(
	ctx context.Context, topicId TopicId, reputer ActorId) (
	previousReputerRewardFraction alloraMath.Dec, noPrior bool, err error) {
	key := collections.Join(topicId, reputer)
	previousReputerRewardFraction, err = k.previousReputerRewardFraction.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), true, nil
	} else if err != nil {
		return alloraMath.Dec{}, false, errorsmod.Wrap(err, "error getting previous reputer reward fraction")
	}
	return previousReputerRewardFraction, false, nil
}

// Sets the previous W_{i-1,m}
func (k *ScoresKeeper) SetPreviousReputerRewardFraction(ctx context.Context, topicId TopicId, reputer ActorId, reward alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousReputerRewardFraction: Error validating topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousReputerRewardFraction: Error validating reputer")
	}
	if err := types.ValidateDec(reward); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousReputerRewardFraction: Error validating reward")
	}
	key := collections.Join(topicId, reputer)
	return k.previousReputerRewardFraction.Set(ctx, key, reward)
}

// Gets the previous U_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *ScoresKeeper) GetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker ActorId) (
	previousInferenceRewardFraction alloraMath.Dec, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	previousInferenceRewardFraction, err = k.previousInferenceRewardFraction.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), true, nil
	} else if err != nil {
		return alloraMath.Dec{}, false, errorsmod.Wrap(err, "error getting previous inference reward fraction")
	}
	return previousInferenceRewardFraction, false, nil
}

// Sets the previous U_{i-1,m}
func (k *ScoresKeeper) SetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker ActorId, reward alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousInferenceRewardFraction: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousInferenceRewardFraction: Error validating worker")
	}
	if err := types.ValidateDec(reward); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousInferenceRewardFraction: Error validating reward")
	}
	key := collections.Join(topicId, worker)
	return k.previousInferenceRewardFraction.Set(ctx, key, reward)
}

// Gets the previous V_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *ScoresKeeper) GetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker ActorId) (
	previousForecastRewardFraction alloraMath.Dec, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	previousForecastRewardFraction, err = k.previousForecastRewardFraction.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), true, nil
	} else if err != nil {
		return alloraMath.Dec{}, false, errorsmod.Wrap(err, "error getting previous forecast reward fraction")
	}
	return previousForecastRewardFraction, false, nil
}

// Sets the previous V_{i-1,m}
func (k *ScoresKeeper) SetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker ActorId, reward alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecastRewardFraction: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecastRewardFraction: Error validating worker")
	}
	if err := types.ValidateDec(reward); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecastRewardFraction: Error validating reward")
	}
	key := collections.Join(topicId, worker)
	return k.previousForecastRewardFraction.Set(ctx, key, reward)
}

func (k *ScoresKeeper) SetPreviousPercentageRewardToStakedReputers(
	ctx context.Context,
	percentageRewardToStakedReputers alloraMath.Dec,
) error {
	if err := types.ValidateDec(percentageRewardToStakedReputers); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousPercentageRewardToStakedReputers: Error validating percentage reward to staked reputers")
	}
	return k.previousPercentageRewardToStakedReputers.Set(ctx, percentageRewardToStakedReputers)
}

func (k *ScoresKeeper) GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (alloraMath.Dec, error) {
	return k.previousPercentageRewardToStakedReputers.Get(ctx)
}

func (k *ScoresKeeper) SetPreviousForecasterScoreRatio(ctx context.Context, topicId TopicId, forecasterScoreRatio alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecasterScoreRatio: Error validating topic id")
	}
	if err := types.ValidateDec(forecasterScoreRatio); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecasterScoreRatio: Error validating forecaster score ratio")
	}
	return k.previousForecasterScoreRatio.Set(ctx, topicId, forecasterScoreRatio)
}

func (k *ScoresKeeper) GetPreviousForecasterScoreRatio(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	forecastTau, err := k.previousForecasterScoreRatio.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous forecaster score ratio")
	}
	return forecastTau, nil
}

func (k *ScoresKeeper) GetTopicInitialInfererEmaScore(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.initialInfererEmaScore.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	}
	return score, err
}

func (k *ScoresKeeper) SetTopicInitialInfererEmaScore(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrap(err, "score validation failed")
	}
	return k.initialInfererEmaScore.Set(ctx, topicId, score)
}

func (k *ScoresKeeper) GetTopicInitialForecasterEmaScore(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.initialForecasterEmaScore.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	}
	return score, err
}

func (k *ScoresKeeper) SetTopicInitialForecasterEmaScore(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrap(err, "score validation failed")
	}
	return k.initialForecasterEmaScore.Set(ctx, topicId, score)
}

func (k *ScoresKeeper) GetTopicInitialReputerEmaScore(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.initialReputerEmaScore.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	}
	return score, err
}

func (k *ScoresKeeper) SetTopicInitialReputerEmaScore(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrap(err, "score validation failed")
	}
	return k.initialReputerEmaScore.Set(ctx, topicId, score)
}
