package rewards

import (
	"fmt"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EmitRewards(ctx sdk.Context, k keeper.Keeper, activeTopics []types.Topic) error {
	// Get Total Emissions/ Fees Collected

	// Get Total Allocation
	totalReward := k.BankKeeper().GetBalance(
		ctx,
		sdk.AccAddress(types.AlloraRewardsAccountName),
		sdk.DefaultBondDenom).Amount
	totalRewardDec, err := alloraMath.NewDecFromSdkInt(totalReward)
	if err != nil {
		return err
	}

	// Get Distribution of Rewards per Topic
	weights, sumWeight, err := GetActiveTopicWeights(ctx, k, activeTopics)
	if err != nil {
		fmt.Println("weights error")
		return err
	}
	if sumWeight.IsZero() {
		fmt.Println("No weights, no rewards!")
		return nil
	}
	topicRewards := make([]alloraMath.Dec, len(activeTopics))
	for i := range weights {
		topicWeight := weights[i]
		topicRewardFraction, err := GetTopicRewardFraction(topicWeight, sumWeight)
		if err != nil {
			fmt.Println("reward fraction error")
			return err
		}
		topicReward, err := GetTopicReward(topicRewardFraction, totalRewardDec)
		if err != nil {
			fmt.Println("reward error")
			return err
		}
		topicRewards[i] = topicReward
	}

	blockHeight := ctx.BlockHeight()
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	// for every topic
	for i := 0; i < len(activeTopics); i++ {
		topic := activeTopics[i]
		topicRewards := topicRewards[i] // E_{t,i}
		lossBundles, err := k.GetNetworkLossBundleAtBlock(ctx, topic.Id, blockHeight)
		if err != nil {
			return err
		}

		// Get Entropy for each task
		reputerEntropy, reputerFractions, reputers, err := GetReputerTaskEntropy(
			ctx,
			k,
			topic.Id,
			moduleParams.TaskRewardAlpha,
			moduleParams.PRewardSpread,
			moduleParams.BetaEntropy,
		)
		if err != nil {
			return err
		}
		inferenceEntropy, inferenceFractions, workersInference, err := GetInferenceTaskEntropy(
			ctx,
			k,
			topic.Id,
			moduleParams.TaskRewardAlpha,
			moduleParams.PRewardSpread,
			moduleParams.BetaEntropy,
		)
		if err != nil {
			return err
		}
		forecastingEntropy, forecastFractions, workersForecast, err := GetForecastingTaskEntropy(
			ctx,
			k,
			topic.Id,
			moduleParams.TaskRewardAlpha,
			moduleParams.PRewardSpread,
			moduleParams.BetaEntropy,
		)
		if err != nil {
			return err
		}

		// Get Total Rewards for Reputation task
		taskReputerReward, err := GetRewardForReputerTaskInTopic(
			inferenceEntropy,
			forecastingEntropy,
			reputerEntropy,
			topicRewards,
		)
		if err != nil {
			return err
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
			return err
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
			return err
		}

		totalRewardsDistribution := make([]TaskRewards, 0)

		// Get Distribution of Rewards per Reputer
		reputerRewards, err := GetReputerRewards(
			ctx,
			k,
			topic.Id,
			blockHeight,
			moduleParams.PRewardSpread,
			taskReputerReward,
		)
		if err != nil {
			return err
		}
		totalRewardsDistribution = append(totalRewardsDistribution, reputerRewards...)

		// Get Distribution of Rewards per Worker - Inference Task
		inferenceRewards, err := GetWorkersRewardsInferenceTask(
			ctx,
			k,
			topic.Id,
			blockHeight,
			moduleParams.PRewardSpread,
			taskInferenceReward,
		)
		if err != nil {
			return err
		}
		totalRewardsDistribution = append(totalRewardsDistribution, inferenceRewards...)

		// Get Distribution of Rewards per Worker - Forecast Task
		forecastRewards, err := GetWorkersRewardsForecastTask(
			ctx,
			k,
			topic.Id,
			blockHeight,
			moduleParams.PRewardSpread,
			taskForecastingReward,
		)
		if err != nil {
			return err
		}
		totalRewardsDistribution = append(totalRewardsDistribution, forecastRewards...)

		// Pay out rewards
		err = payoutRewards(ctx, k, totalRewardsDistribution)
		if err != nil {
			return err
		}
		SetPreviousRewardFractions(
			ctx,
			k,
			topic.Id,
			reputers,
			reputerFractions,
			workersInference,
			inferenceFractions,
			workersForecast,
			forecastFractions,
		)
	}

	SetPreviousTopicWeights(ctx, k, activeTopics, weights)
	return nil
}

func payoutRewards(ctx sdk.Context, k keeper.Keeper, rewards []TaskRewards) error {
	for _, reward := range rewards {
		address, err := sdk.AccAddressFromBech32(string(reward.Address))
		if err != nil {
			return err
		}

		// TODO: Check precision of rewards
		err = k.BankKeeper().SendCoinsFromModuleToAccount(
			ctx,
			types.AlloraRewardsAccountName,
			address,
			sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, reward.Reward.SdkIntTrim())),
		)
		if err != nil {
			return err
		}
	}

	return nil
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
			return err
		}
	}
	for i, worker := range workersInference {
		err := k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, inferenceRewardFractions[i])
		if err != nil {
			return err
		}
	}
	for i, worker := range workersForecast {
		err := k.SetPreviousForecastRewardFraction(ctx, topicId, worker, forecastRewardFractions[i])
		if err != nil {
			return err
		}
	}
	return nil
}
