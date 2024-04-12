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
	validatorsVsAlloraPercentReward, err := k.GetParamsValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		fmt.Println("percent error")
		return err
	}
	f_v, err := alloraMath.NewDecFromSdkLegacyDec(validatorsVsAlloraPercentReward)
	if err != nil {
		fmt.Println("legacyDec conversion error")
		return err
	}
	topicRewards := make([]alloraMath.Dec, len(activeTopics))
	for i := range weights {
		topicWeight := weights[i]
		topicRewardFraction, err := GetTopicRewardFraction(f_v, topicWeight, sumWeight)
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

		// Get Entropy for each task
		reputerEntropy, err := GetReputerTaskEntropy(
			ctx,
			k,
			topic.Id,
			moduleParams.TaskRewardAlpha,
			moduleParams.PRewardSpread,
			moduleParams.BetaEntropy,
		)
		forecastingEntropy, err := GetForecastingTaskEntropy(
			ctx,
			k,
			topic.Id,
		)
		inferenceEntropy, err := GetInferenceTaskEntropy(
			ctx,
			k,
			topic.Id,
		)

		// Get Total Rewards for Reputation task
		taskReputerReward, err := GetRewardForReputerTaskInTopic(
			inferenceEntropy,
			forecastingEntropy,
			reputerEntropy,
			topicRewards,
		)
		taskForecastingReward, err := GetRewardForForecastingTaskInTopic(
			chi,
			gamma,
			inferenceEntropy,
			forecastingEntropy,
			reputerEntropy,
			topicRewards,
		)
		taskInferenceReward, err := GetRewardForInferenceTaskInTopic(
			chi,
			gamma,
			inferenceEntropy,
			forecastingEntropy,
			reputerEntropy,
			topicRewards,
		)

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
