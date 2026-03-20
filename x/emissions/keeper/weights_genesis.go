package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *WeightsKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// Initialize latest regret scale from genesis data.
	for _, regretScale := range data.LatestRegretStdNorm {
		if regretScale != nil {
			if err := k.SetLatestRegretScale(ctx, regretScale.TopicId, regretScale.Dec); err != nil {
				return errors.Wrap(err, "error setting latest regret scale")
			}
		}
	}

	// Initialize latest inferer weights
	for _, weight := range data.LatestInfererWeights {
		if weight != nil {
			if err := k.SetLatestInfererWeight(ctx, weight.TopicId, weight.ActorId, weight.Dec); err != nil {
				return errors.Wrap(err, "error setting latest inferer weight")
			}
		}
	}

	// Initialize latest forecaster weights
	for _, weight := range data.LatestForecasterWeights {
		if weight != nil {
			if err := k.SetLatestForecasterWeight(ctx, weight.TopicId, weight.ActorId, weight.Dec); err != nil {
				return errors.Wrap(err, "error setting latest forecaster weight")
			}
		}
	}

	// MonthlyReputerRewards / MonthlyTopicRewards
	// Validate genesis values, then reset to zero. Monthly rewards are epoch-scoped
	// accumulators that should not carry over across genesis restarts.
	if err := types.ValidateSdkIntRepresentingMonetaryValue(data.MonthlyReputerRewards); err != nil {
		return errors.Wrap(err, "monthly reputer rewards validation failed")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(data.MonthlyTopicRewards); err != nil {
		return errors.Wrap(err, "monthly topic rewards validation failed")
	}
	if err := k.ResetMonthlyRewards(ctx); err != nil {
		return errors.Wrap(err, "error setting monthlyTopicRewards")
	}

	return nil
}

func (k *WeightsKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// Export latest regret scale.
	latestRegretScale := make([]*types.TopicIdAndDec, 0)
	regretScaleIter, err := k.latestRegretScale.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest regret scale")
	}
	for ; regretScaleIter.Valid(); regretScaleIter.Next() {
		keyValue, err := regretScaleIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: regretScaleIter")
		}
		latestRegretScale = append(latestRegretScale, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
	}
	data.LatestRegretStdNorm = latestRegretScale

	// Export latest inferer weights
	latestInfererWeights := make([]*types.TopicIdActorIdDec, 0)
	infererWeightsIter, err := k.latestInfererWeights.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest inferer weights")
	}
	for ; infererWeightsIter.Valid(); infererWeightsIter.Next() {
		keyValue, err := infererWeightsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: infererWeightsIter")
		}
		latestInfererWeights = append(latestInfererWeights, &types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		})
	}
	data.LatestInfererWeights = latestInfererWeights

	// Export latest forecaster weights
	latestForecasterWeights := make([]*types.TopicIdActorIdDec, 0)
	forecasterWeightsIter, err := k.latestForecasterWeights.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest forecaster weights")
	}
	for ; forecasterWeightsIter.Valid(); forecasterWeightsIter.Next() {
		keyValue, err := forecasterWeightsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: forecasterWeightsIter")
		}
		latestForecasterWeights = append(latestForecasterWeights, &types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		})
	}
	data.LatestForecasterWeights = latestForecasterWeights

	// Get Monthly Reputer Rewards
	monthlyReputerRewards, err := k.GetMonthlyReputerRewards(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get monthly reputer rewards")
	}
	data.MonthlyReputerRewards = monthlyReputerRewards

	// Get Monthly Topic Rewards
	monthlyTopicRewards, err := k.GetMonthlyTopicRewards(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get monthly topic rewards")
	}
	data.MonthlyTopicRewards = monthlyTopicRewards

	return nil
}
