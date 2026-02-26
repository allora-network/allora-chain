package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewWeightsKeeper(sb *collections.SchemaBuilder, scoresKeeper *ScoresKeeper) *WeightsKeeper {
	return &WeightsKeeper{
		latestRegretScale:       collections.NewMap(sb, types.LatestRegretStdNormKey, "latest_regret_stdnorm", collections.Uint64Key, alloraMath.DecValue),
		latestInfererWeights:    collections.NewMap(sb, types.LatestInfererWeightsKey, "latest_inferer_weights", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		latestForecasterWeights: collections.NewMap(sb, types.LatestForecasterWeightsKey, "latest_forecaster_weights", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		monthlyReputerRewards:   collections.NewItem(sb, types.MonthlyReputerRewardsKey, "monthly_reputer_rewards", sdk.IntValue),
		monthlyTopicRewards:     collections.NewItem(sb, types.MonthlyTopicRewardsKey, "monthly_topic_rewards", sdk.IntValue),
		scoresKeeper:            scoresKeeper,
	}
}

type WeightsKeeper struct {
	// The latest regret normalization scale for a topic.
	// NOTE: legacy name "stdnorm" is preserved for state compatibility (MAD-based scale).
	latestRegretScale collections.Map[TopicId, alloraMath.Dec]
	// The latest weights for a topic
	latestInfererWeights    collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	latestForecasterWeights collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// total reward for the month going to reputers
	monthlyReputerRewards collections.Item[cosmosMath.Int]
	// total reward for the month going to all topic participants (reputers, inferers, forecasters)
	monthlyTopicRewards collections.Item[cosmosMath.Int]
	// scores keeper
	scoresKeeper *ScoresKeeper
}

// GetLatestRegretScale returns the latest regret normalization scale for a topic.
// NOTE: legacy name "stdnorm" is preserved for state compatibility (MAD-based scale).
func (k *WeightsKeeper) GetLatestRegretScale(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	regretScale, err := k.latestRegretScale.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	}
	return regretScale, err
}

// SetLatestRegretScale sets the latest regret normalization scale for a topic.
// NOTE: legacy name "stdnorm" is preserved for state compatibility (MAD-based scale).
func (k *WeightsKeeper) SetLatestRegretScale(ctx context.Context, topicId TopicId, regretScale alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(regretScale); err != nil {
		return errorsmod.Wrap(err, "regret scale validation failed")
	}
	if regretScale.IsZero() {
		return errorsmod.Wrap(types.ErrInvalidValue, "regret scale cannot be zero")
	}
	return k.latestRegretScale.Set(ctx, topicId, regretScale)
}

// GetLatestInfererWeight returns the latest inferer weight for a topic and worker
func (k *WeightsKeeper) GetLatestInfererWeight(ctx context.Context, topicId TopicId, worker ActorId) (alloraMath.Dec, error) {
	weight, err := k.latestInfererWeights.Get(ctx, collections.Join(topicId, worker))
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	}
	return weight, err
}

// SetLatestInfererWeight sets the latest inferer weight for a topic and worker
func (k *WeightsKeeper) SetLatestInfererWeight(ctx context.Context, topicId TopicId, worker ActorId, weight alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "worker address validation failed")
	}
	return k.latestInfererWeights.Set(ctx, collections.Join(topicId, worker), weight)
}

// GetLatestForecasterWeight returns the latest forecaster weight for a topic and worker
func (k *WeightsKeeper) GetLatestForecasterWeight(ctx context.Context, topicId TopicId, worker ActorId) (alloraMath.Dec, error) {
	weight, err := k.latestForecasterWeights.Get(ctx, collections.Join(topicId, worker))
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	}
	return weight, err
}

// SetLatestForecasterWeight sets the latest forecaster weight for a topic and worker
func (k *WeightsKeeper) SetLatestForecasterWeight(ctx context.Context, topicId TopicId, worker ActorId, weight alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "worker address validation failed")
	}
	return k.latestForecasterWeights.Set(ctx, collections.Join(topicId, worker), weight)
}

// GetMonthlyReputerRewards retrieves the total reward for the month going to reputers.
func (k *WeightsKeeper) GetMonthlyReputerRewards(ctx context.Context) (cosmosMath.Int, error) {
	rewards, err := k.monthlyReputerRewards.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.ZeroInt(), nil // Return zero if not found
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting monthly reputer rewards")
	}
	return rewards, nil
}

// GetMonthlyTopicRewards retrieves the total reward for the month going to all topic participants.
func (k *WeightsKeeper) GetMonthlyTopicRewards(ctx context.Context) (cosmosMath.Int, error) {
	rewards, err := k.monthlyTopicRewards.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.ZeroInt(), nil // Return zero if not found
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting monthly topic rewards")
	}
	return rewards, nil
}

// AddMonthlyRewards adds the specified amounts to the monthly reputer and topic reward counters.
func (k *WeightsKeeper) AddMonthlyRewards(ctx context.Context, reputerReward cosmosMath.Int, topicReward cosmosMath.Int) error {
	currentReputerRewards, err := k.GetMonthlyReputerRewards(ctx)
	if err != nil {
		return err
	}
	newReputerRewards := currentReputerRewards.Add(reputerReward)
	if err := k.monthlyReputerRewards.Set(ctx, newReputerRewards); err != nil {
		return err
	}

	currentTopicRewards, err := k.GetMonthlyTopicRewards(ctx)
	if err != nil {
		return err
	}
	newTopicRewards := currentTopicRewards.Add(topicReward)
	return k.monthlyTopicRewards.Set(ctx, newTopicRewards)
}

// ResetMonthlyRewards resets the monthly reputer and topic reward counters to zero.
func (k *WeightsKeeper) ResetMonthlyRewards(ctx context.Context) error {
	if err := k.monthlyReputerRewards.Set(ctx, cosmosMath.ZeroInt()); err != nil {
		return err
	}
	return k.monthlyTopicRewards.Set(ctx, cosmosMath.ZeroInt())
}

// HandleMonthlyRewardsReset calculates the percentage of rewards that went to staked reputers
// in the previous month, stores it, and resets the monthly counters.
// This method is called monthly (typically from the mint module's BeginBlocker) to:
// 1. Read accumulated monthly rewards for reputers and topic participants
// 2. Calculate what percentage of topic rewards went to staked reputers
// 3. Store this percentage for use in next month's emission calculations
// 4. Reset the monthly counters to zero for the next month
// Note: Uses value receiver to allow passing keeper by value to mint module (consistent with other keepers)
func (k *WeightsKeeper) HandleMonthlyRewardsReset(ctx context.Context) (err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	totalRewardToStakedReputers, err := k.GetMonthlyReputerRewards(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "Failed to get monthly reputer rewards")
	}

	totalRewardToTopicParticipants, err := k.GetMonthlyTopicRewards(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "Failed to get monthly topic rewards")
	}

	sdkCtx.Logger().Info("Monthly rewards", "totalRewardToStakedReputers", totalRewardToStakedReputers, "totalRewardToTopicParticipants", totalRewardToTopicParticipants)

	percentageToStakedReputersDec := alloraMath.ZeroDec()
	if !totalRewardToTopicParticipants.IsZero() {
		totalReputersDec, err := alloraMath.NewDecFromSdkInt(totalRewardToStakedReputers)
		if err != nil {
			return errorsmod.Wrap(err, "Failed to convert total reputer rewards to alloraMath.Dec")
		}
		totalTopicDec, err := alloraMath.NewDecFromSdkInt(totalRewardToTopicParticipants)
		if err != nil {
			return errorsmod.Wrap(err, "Failed to convert total topic rewards to alloraMath.Dec")
		}

		percentageToStakedReputersDec, err = totalReputersDec.Quo(totalTopicDec)
		if err != nil {
			return errorsmod.Wrap(err, "Failed to calculate percentage to staked reputers")
		}
	}

	err = k.scoresKeeper.SetPreviousPercentageRewardToStakedReputers(ctx, percentageToStakedReputersDec)
	if err != nil {
		return errorsmod.Wrap(err, "Failed to set previous percentage reward to staked reputers")
	}

	// Emit the event
	types.EmitNewPreviousPercentageRewardToStakedReputersSetEvent(sdkCtx, sdkCtx.BlockHeight(), percentageToStakedReputersDec)

	// Reset monthly reputer rewards
	err = k.ResetMonthlyRewards(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "Failed to reset monthly reputer rewards")
	}

	sdkCtx.Logger().Debug("Monthly rewards reset triggered for block height", "blockHeight", sdkCtx.BlockHeight(),
		"previousPercentageRewardToStakedReputers", percentageToStakedReputersDec.String())
	return nil
}
