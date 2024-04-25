package rewards

import (
	"fmt"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	chainParams "github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EmitRewards(ctx sdk.Context, k keeper.Keeper, block BlockHeight) error {
	//
	/// Here, flush inactivation keyset
	//

	// Get Allora Rewards Account
	alloraRewardsAccountAddr := k.AccountKeeper().GetModuleAccount(ctx, types.AlloraRewardsAccountName).GetAddress()

	// Get Total Allocation
	totalReward := k.BankKeeper().GetBalance(
		ctx,
		alloraRewardsAccountAddr,
		params.DefaultBondDenom).Amount
	totalRewardDec, err := alloraMath.NewDecFromSdkInt(totalReward)
	if err != nil {
		return errors.Wrapf(err, "failed to convert total reward to decimal")
	}

	// Get Distribution of Rewards per Topic
	weights, sumWeight, sumRevenue, numRewardReeadyTopics, err := GetRewardReadyTopicWeights(ctx, k, block)
	if err != nil {
		return errors.Wrapf(err, "weights error")
	}
	if sumWeight.IsZero() {
		fmt.Println("No weights, no rewards!")
		return nil
	}

	// Send collected inference request fees to the Ecosystem module account
	// They will be paid out to reputers, workers, and cosmos validators
	// according to the formulas in the beginblock of the mint module
	err = k.BankKeeper().SendCoinsFromModuleToModule(
		ctx,
		types.AlloraRequestsAccountName,
		mintTypes.EcosystemModuleName,
		sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, cosmosMath.NewInt(sumRevenue.BigInt().Int64()))))
	if err != nil {
		fmt.Println("Error sending coins from module to module: ", err)
		return err
	}

	// minTopicWeight, err := k.GetParamsMinTopicWeight(ctx)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to get min topic weight")
	// }

	topicRewards := make(map[TopicId]alloraMath.Dec, numRewardReeadyTopics)
	for topicId, weight := range weights {
		// Consider sorting by weight desc and skimming the top here per SortTopicsByReturnDescWithRandomTiebreaker() and param MaxTopicsPerBlock
		// If we do that though, we should be sure to return the revenue to those topics that didn't make the cut

		// //
		// /// TODO add to inactivation keyset (pop before EmitRewards called, or at onset of it)
		// //
		// inactivated, err := InactivateTopicIfWeightBelowMin(ctx, k, minTopicWeight, topicId, weight)
		// if err != nil {
		// 	return errors.Wrapf(err, "failed to decrement topic unmet demand and inactivate")
		// }
		// if inactivated {
		// 	continue
		// }

		topicRewardFraction, err := GetTopicRewardFraction(weight, sumWeight)
		if err != nil {
			return errors.Wrapf(err, "reward fraction error")
		}
		topicReward, err := GetTopicReward(topicRewardFraction, totalRewardDec)
		if err != nil {
			return errors.Wrapf(err, "reward error")
		}
		topicRewards[topicId] = topicReward
	}

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get module params")
	}
	// for every topic
	for topicId, topicReward := range topicRewards {
		// To notify topic handler that the topic is ready for churn i.e. requests to be sent to workers and reputers
		err = k.AddChurnReadyTopic(ctx, topicId)
		if err != nil {
			fmt.Println("Error setting churn ready topic: ", err)
			return err
		}

		// Get topic reward nonce/block height
		topicRewardNonce, err := k.GetTopicRewardNonce(ctx, topicId)
		// If the topic has no reward nonce, skip it
		if err != nil || topicRewardNonce == 0 {
			continue
		}

		taskReputerReward, taskInferenceReward, taskForecastingReward, err := GenerateTasksRewards(ctx, k, topicId, topicReward, topicRewardNonce, moduleParams)
		if err != nil {
			return errors.Wrapf(err, "failed to generate task rewards")
		}

		totalRewardsDistribution := make([]TaskRewards, 0)

		// Get Distribution of Rewards per Reputer
		reputerRewards, err := GetReputerRewards(
			ctx,
			k,
			topicId,
			topicRewardNonce,
			moduleParams.PRewardSpread,
			taskReputerReward,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to get reputer rewards")
		}
		totalRewardsDistribution = append(totalRewardsDistribution, reputerRewards...)

		// Get Distribution of Rewards per Worker - Inference Task
		inferenceRewards, err := GetWorkersRewardsInferenceTask(
			ctx,
			k,
			topicId,
			topicRewardNonce,
			moduleParams.PRewardSpread,
			taskInferenceReward,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to get inference rewards")
		}
		totalRewardsDistribution = append(totalRewardsDistribution, inferenceRewards...)

		// Get Distribution of Rewards per Worker - Forecast Task
		forecastRewards, err := GetWorkersRewardsForecastTask(
			ctx,
			k,
			topicId,
			topicRewardNonce,
			moduleParams.PRewardSpread,
			taskForecastingReward,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to get forecast rewards")
		}
		totalRewardsDistribution = append(totalRewardsDistribution, forecastRewards...)

		// Pay out rewards
		err = payoutRewards(ctx, k, totalRewardsDistribution)
		if err != nil {
			return errors.Wrapf(err, "failed to pay out rewards")
		}

		// Delete topic reward nonce
		err = k.DeleteTopicRewardNonce(ctx, topicId)
		if err != nil {
			return errors.Wrapf(err, "failed to delete topic reward nonce")
		}
	}
	return nil
}

func GenerateTasksRewards(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	topicRewards alloraMath.Dec,
	block int64,
	moduleParams types.Params,
) (
	alloraMath.Dec,
	alloraMath.Dec,
	alloraMath.Dec,
	error,
) {
	lossBundles, err := k.GetNetworkLossBundleAtBlock(ctx, topicId, block)
	if err != nil {
		return alloraMath.Dec{},
			alloraMath.Dec{},
			alloraMath.Dec{},
			errors.Wrapf(err, "failed to get network loss bundle at block %d", block)
	}

	reputerEntropy, reputerFractions, reputers, err := GetReputerTaskEntropy(
		ctx,
		k,
		topicId,
		moduleParams.TaskRewardAlpha,
		moduleParams.PRewardSpread,
		moduleParams.BetaEntropy,
		block,
	)
	if err != nil {
		return alloraMath.Dec{},
			alloraMath.Dec{},
			alloraMath.Dec{},
			errors.Wrapf(err, "failed to get reputer task entropy")
	}
	inferenceEntropy, inferenceFractions, workersInference, err := GetInferenceTaskEntropy(
		ctx,
		k,
		topicId,
		moduleParams.TaskRewardAlpha,
		moduleParams.PRewardSpread,
		moduleParams.BetaEntropy,
		block,
	)
	if err != nil {
		return alloraMath.Dec{},
			alloraMath.Dec{},
			alloraMath.Dec{},
			errors.Wrapf(err, "failed to get inference task entropy")
	}
	forecastingEntropy, forecastFractions, workersForecast, err := GetForecastingTaskEntropy(
		ctx,
		k,
		topicId,
		moduleParams.TaskRewardAlpha,
		moduleParams.PRewardSpread,
		moduleParams.BetaEntropy,
		block,
	)
	if err != nil {
		return alloraMath.Dec{}, alloraMath.Dec{}, alloraMath.Dec{}, err
	}

	// Get Total Rewards for Reputation task
	taskReputerReward, err := GetRewardForReputerTaskInTopic(
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicRewards,
	)
	if err != nil {
		return alloraMath.Dec{},
			alloraMath.Dec{},
			alloraMath.Dec{},
			errors.Wrapf(err, "failed to get reward for reputer task in topic")
	}
	taskInferenceReward, err := GetRewardForInferenceTaskInTopic(
		lossBundles.NaiveValue,
		lossBundles.CombinedValue,
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicRewards,
		moduleParams.SigmoidA,
		moduleParams.SigmoidB,
	)
	if err != nil {
		return alloraMath.Dec{},
			alloraMath.Dec{},
			alloraMath.Dec{},
			errors.Wrapf(err, "failed to get reward for inference task in topic")
	}
	taskForecastingReward, err := GetRewardForForecastingTaskInTopic(
		lossBundles.NaiveValue,
		lossBundles.CombinedValue,
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicRewards,
		moduleParams.SigmoidA,
		moduleParams.SigmoidB,
	)
	if err != nil {
		return alloraMath.Dec{},
			alloraMath.Dec{},
			alloraMath.Dec{},
			errors.Wrapf(err, "failed to get reward for forecasting task in topic")
	}

	SetPreviousRewardFractions(
		ctx,
		k,
		topicId,
		reputers,
		reputerFractions,
		workersInference,
		inferenceFractions,
		workersForecast,
		forecastFractions,
	)

	return taskReputerReward, taskInferenceReward, taskForecastingReward, nil
}

func SetPreviousRewardFractions(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	reputers []sdk.AccAddress,
	reputerRewardFractions []alloraMath.Dec,
	workersInference []sdk.AccAddress,
	inferenceRewardFractions []alloraMath.Dec,
	workersForecast []sdk.AccAddress,
	forecastRewardFractions []alloraMath.Dec,
) error {
	for i, reputer := range reputers {
		err := k.SetPreviousReputerRewardFraction(ctx, topicId, reputer, reputerRewardFractions[i])
		if err != nil {
			return errors.Wrapf(err, "failed to set previous reputer reward fraction")
		}
	}
	for i, worker := range workersInference {
		err := k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, inferenceRewardFractions[i])
		if err != nil {
			return errors.Wrapf(err, "failed to set previous inference reward fraction")
		}
	}
	for i, worker := range workersForecast {
		err := k.SetPreviousForecastRewardFraction(ctx, topicId, worker, forecastRewardFractions[i])
		if err != nil {
			return errors.Wrapf(err, "failed to set previous forecast reward fraction")
		}
	}
	return nil
}

func payoutRewards(ctx sdk.Context, k keeper.Keeper, rewards []TaskRewards) error {
	for _, reward := range rewards {
		address, err := sdk.AccAddressFromBech32(reward.Address.String())
		if err != nil {
			return errors.Wrapf(err, "failed to decode payout address")
		}

		err = k.BankKeeper().SendCoinsFromModuleToAccount(
			ctx,
			types.AlloraRewardsAccountName,
			address,
			sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, reward.Reward.SdkIntTrim())),
		)
		if err != nil {
			return errors.Wrapf(err, "failed to send coins from rewards module to payout address")
		}
	}

	return nil
}
