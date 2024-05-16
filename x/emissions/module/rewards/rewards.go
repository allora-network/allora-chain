package rewards

import (
	"fmt"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EmitRewards(ctx sdk.Context, k keeper.Keeper, blockHeight BlockHeight, weights map[uint64]*alloraMath.Dec, sumWeight alloraMath.Dec, totalRevenue cosmosMath.Int) error {
	totalReward, err := k.GetTotalRewardToDistribute(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get total reward to distribute")
	}
	if totalReward.IsZero() {
		ctx.Logger().Warn("The total scheduled rewards to distribute this epoch are zero! Skipping rewards distribution.")
		return nil
	}
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get module params")
	}

	// Distribute rewards between topics
	topicRewards, err := GenerateRewardsDistributionByTopic(ctx, k, moduleParams.MaxTopicsPerBlock, blockHeight, totalReward, weights, sumWeight, totalRevenue)
	if err != nil {
		return errors.Wrapf(err, "failed to generate total reward by topic")
		// Will return nil if there are no topics to reward
	} else if topicRewards == nil {
		return nil
	}

	totalRewardToStakedReputers := alloraMath.ZeroDec()
	// for every topic
	for topicId, topicReward := range topicRewards {
		// Get topic reward nonce/block height
		topicRewardNonce, err := k.GetTopicRewardNonce(ctx, topicId)
		// If the topic has no reward nonce, skip it
		if err != nil || topicRewardNonce == 0 {
			continue
		}

		// Distribute rewards between topic participants
		totalRewardsDistribution, rewardInTopicToReputers, err := GenerateRewardsDistributionByTopicParticipant(ctx, k, topicId, topicReward, topicRewardNonce, moduleParams)
		if err != nil {
			fmt.Printf(
				"Failed to Generate Rewards for Topic, Skipping:\nTopic Id %d\nTopic Reward Amount %s\nError:\n%s\n\n",
				topicId,
				topicReward.String(),
				err,
			)
			continue
		}
		totalRewardToStakedReputers, err = totalRewardToStakedReputers.Add(rewardInTopicToReputers)
		if err != nil {
			return errors.Wrapf(
				err,
				"Error finding sum of rewards to Reputers:\n%s\n%s",
				totalRewardToStakedReputers.String(),
				rewardInTopicToReputers.String(),
			)
		}

		// Pay out rewards to topic participants
		payoutErrors := payoutRewards(ctx, k, totalRewardsDistribution)
		if len(payoutErrors) > 0 {
			for _, err := range payoutErrors {
				fmt.Printf(
					"Failed to pay out rewards to participant in Topic:\nTopic Id %d\nTopic Reward Amount %s\nError:\n%s\n\n",
					topicId,
					topicReward.String(),
					err,
				)
			}
			continue
		}

		// Prune records after rewards have been paid out
		err = pruneRecordsAfterRewards(ctx, k, moduleParams.MinEpochLengthRecordLimit, topicId, topicRewardNonce)
		if err != nil {
			fmt.Printf(
				"Failed to prune records after rewards for Topic, Skipping:\nTopic Id %d\nTopic Reward Amount %s\nError:\n%s\n\n",
				topicId,
				topicReward.String(),
				err,
			)
			continue
		}
	}
	// set the previous percentage reward to staked reputers
	// for the mint module to be able to control the inflation rate to that actor
	percentageToStakedReputers, err := totalRewardToStakedReputers.Quo(totalReward)
	if err != nil {
		return errors.Wrapf(err, "failed to calculate percentage to staked reputers")
	}
	err = k.SetPreviousPercentageRewardToStakedReputers(ctx, percentageToStakedReputers)
	if err != nil {
		return errors.Wrapf(err, "failed to set previous percentage reward to staked reputers")
	}

	return nil
}

func GenerateRewardsDistributionByTopic(
	ctx sdk.Context,
	k keeper.Keeper,
	maxTopicsPerBlock uint64,
	blockHeight BlockHeight,
	totalReward alloraMath.Dec,
	weights map[uint64]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalRevenue cosmosMath.Int,
) (map[uint64]*alloraMath.Dec, error) {
	if sumWeight.IsZero() {
		fmt.Println("No weights, no rewards!")
		return nil, nil
	}
	// Filter out topics that are not reward-ready, inactivate if needed
	weightsOfActiveTopics, sumWeight, sumRevenue, err := FilterAndInactivateTopicsUpdatingSums(
		ctx,
		k,
		weights,
		sumWeight,
		totalRevenue,
		totalReward,
		blockHeight,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to inactivate topics and update sums")
	}
	if sumWeight.IsZero() {
		fmt.Println("No filtered weights, no rewards!")
		return nil, nil
	}

	// Sort remaining active topics by weight desc and skim the top via SortTopicsByReturnDescWithRandomTiebreaker() and param MaxTopicsPerBlock
	weightsOfTopActiveTopics := SkimTopTopicsByWeightDesc(weightsOfActiveTopics, maxTopicsPerBlock, blockHeight)

	// Return the revenue to those topics that didn't make the cut
	// Loop though weightsOfActiveTopics and if the topic is not in weightsOfTopActiveTopics, add to running revenue sum
	sumRevenueOfBottomTopics := cosmosMath.ZeroInt()
	for topicId := range weightsOfActiveTopics {
		// If the topic is not in the top active topics, add its revenue to the running sum
		if _, ok := weightsOfTopActiveTopics[topicId]; !ok {
			topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get topic fee revenue")
			}
			sumRevenueOfBottomTopics = sumRevenueOfBottomTopics.Add(topicFeeRevenue.Revenue)
		}

		// This way we won't double count from this earlier epoch revenue the next epoch
		// This must come after GetTopicFeeRevenue() is last called per topic because otherwise the returned revenue will be zero
		err = k.ResetTopicFeeRevenue(ctx, topicId, blockHeight)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to reset topic fee revenue")
		}
	}

	// Send remaining collected inference request fees to the Ecosystem module account
	// They will be paid out to reputers, workers, and validators
	// according to the formulas in the beginblock of the mint module
	err = k.SendCoinsFromModuleToModule(
		ctx,
		types.AlloraRequestsAccountName,
		mintTypes.EcosystemModuleName,
		sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(sumRevenue.Sub(sumRevenueOfBottomTopics).BigInt().Int64()))))
	if err != nil {
		fmt.Println("Error sending coins from module to module: ", err)
		return nil, err
	}

	topicRewards, err := CalcTopicRewards(ctx, k, weightsOfTopActiveTopics, sumWeight, totalReward)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate topic rewards")
	}

	return topicRewards, nil
}

func removeFromSumWeightAndRevenue(ctx sdk.Context, k keeper.Keeper, sumWeight alloraMath.Dec, sumRevenue cosmosMath.Int,
	weight *alloraMath.Dec, topicId uint64) (alloraMath.Dec, cosmosMath.Int, error) {
	// Update sum weight and revenue -- We won't be deducting fees from inactive topics, as we won't be churning them
	// i.e. we'll neither emit their worker/reputer requests or calculate rewards for its participants this epoch
	sumWeight, err := sumWeight.Sub(*weight)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to subtract weight from sum")
	}
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get topic fee revenue")
	}
	sumRevenue = sumRevenue.Sub(topicFeeRevenue.Revenue)

	return sumWeight, sumRevenue, nil
}

func FilterAndInactivateTopicsUpdatingSums(
	ctx sdk.Context,
	k keeper.Keeper,
	weights map[uint64]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	sumRevenue cosmosMath.Int,
	totalReward alloraMath.Dec,
	blockHeight BlockHeight,
) (
	map[uint64]*alloraMath.Dec,
	alloraMath.Dec,
	cosmosMath.Int,
	error,
) {

	minTopicWeight, err := k.GetParamsMinTopicWeight(ctx)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get min topic weight")
	}

	weightsOfActiveTopics := make(map[TopicId]*alloraMath.Dec)
	for topicId, weight := range weights {
		// Filter out if not reward-ready
		// Check topic has an unfulfilled reward nonce
		rewardNonce, err := k.GetTopicRewardNonce(ctx, topicId)
		filterOutTopic := false
		filterOutErrorMessage := ""
		if err != nil {
			fmt.Println("Error getting reputer request nonces: ", err)
			filterOutTopic = true
			filterOutErrorMessage = "failed to remove from sum weight and revenue"
		}
		if rewardNonce == 0 {
			fmt.Println("Reputer request nonces is nil")
			filterOutTopic = true
			filterOutErrorMessage = "failed to remove nil-reputer-nonce topic from sum weight and revenue"
		}

		// Inactivate and skip the topic if its weight is below the globally-set minimum
		if weight.Lt(minTopicWeight) {
			err = k.InactivateTopic(ctx, topicId)
			if err != nil {
				return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to inactivate topic")
			}

			// This way we won't double count from this earlier epoch revenue the next time this topic is activated
			// This must come after GetTopicFeeRevenue() is last called per topic because otherwise the returned revenue will be zero
			err = k.ResetTopicFeeRevenue(ctx, topicId, blockHeight)
			if err != nil {
				return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to reset topic fee revenue")
			}

			// Update sum weight and revenue -- We won't be deducting fees from inactive topics, as we won't be churning them
			// i.e. we'll neither emit their worker/reputer requests or calculate rewards for its participants this epoch
			filterOutTopic = true
			filterOutErrorMessage = "failed to remove inactivated from sum weight and revenue"
		}
		if filterOutTopic {
			sumWeight, sumRevenue, err = removeFromSumWeightAndRevenue(ctx, k, sumWeight, sumRevenue, weight, topicId)
			if err != nil {
				return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, filterOutErrorMessage)
			}
		}
		weightsOfActiveTopics[topicId] = weight
	}
	return weightsOfActiveTopics, sumWeight, sumRevenue, nil
}

func CalcTopicRewards(
	ctx sdk.Context,
	k keeper.Keeper,
	weights map[uint64]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalReward alloraMath.Dec,
) (
	map[uint64]*alloraMath.Dec,
	error,
) {
	topicRewards := make(map[TopicId]*alloraMath.Dec)
	for topicId, weight := range weights {
		topicRewardFraction, err := GetTopicRewardFraction(weight, sumWeight)
		if err != nil {
			return nil, errors.Wrapf(err, "topic reward fraction error")
		}
		topicReward, err := GetTopicReward(topicRewardFraction, totalReward)
		if err != nil {
			return nil, errors.Wrapf(err, "topic reward error")
		}
		topicRewards[topicId] = &topicReward
	}
	return topicRewards, nil
}

func GenerateRewardsDistributionByTopicParticipant(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	topicReward *alloraMath.Dec,
	blockHeight int64,
	moduleParams types.Params,
) (
	totalRewardsDistribution []TaskRewards,
	taskReputerReward alloraMath.Dec,
	err error,
) {
	bundles, err := k.GetReputerLossBundlesAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get network loss bundle at block %d", blockHeight)
	}

	lossBundles, err := k.GetNetworkLossBundleAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get network loss bundle at block %d", blockHeight)
	}

	// Calculate and Set the reputer scores
	reputerScores, err := GenerateReputerScores(ctx, k, topicId, blockHeight, *bundles)
	if err != nil {
		return nil, alloraMath.Dec{}, err
	}

	// Calculate and Set the worker scores for their inference work
	infererScores, err := GenerateInferenceScores(ctx, k, topicId, blockHeight, *lossBundles)
	if err != nil {
		return nil, alloraMath.Dec{}, err
	}

	// Calculate and Set the worker scores for their forecast work
	forecasterScores, err := GenerateForecastScores(ctx, k, topicId, blockHeight, *lossBundles)
	if err != nil {
		return nil, alloraMath.Dec{}, err
	}

	// Get reputer participants' addresses and reward fractions to be used in the reward round for topic
	reputers, reputersRewardFractions, err := GetReputersRewardFractions(ctx, k, topicId, moduleParams.PRewardSpread, reputerScores)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer reward round data")
	}

	// Get reputer task entropy
	reputerEntropy, err := GetReputerTaskEntropy(
		ctx,
		k,
		topicId,
		moduleParams.TaskRewardAlpha,
		moduleParams.BetaEntropy,
		reputers,
		reputersRewardFractions,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer task entropy")
	}

	// Get inferer reward fractions
	inferers, inferersRewardFractions, err := GetInferenceTaskRewardFractions(
		ctx,
		k,
		topicId,
		blockHeight,
		moduleParams.PRewardSpread,
		infererScores,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inferer reward fractions")
	}

	// Get inference entropy
	inferenceEntropy, err := GetInferenceTaskEntropy(
		ctx,
		k,
		topicId,
		moduleParams.TaskRewardAlpha,
		moduleParams.BetaEntropy,
		inferers,
		inferersRewardFractions,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inference task entropy")
	}

	// Get forecaster reward fractions
	forecasters, forecastersRewardFractions, err := GetForecastingTaskRewardFractions(
		ctx,
		k,
		topicId,
		blockHeight,
		moduleParams.PRewardSpread,
		forecasterScores,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get forecaster reward fractions")
	}

	var forecastingEntropy alloraMath.Dec
	if len(forecasters) > 0 && len(inferers) > 1 {
		// Get forecasting entropy
		forecastingEntropy, err = GetForecastingTaskEntropy(
			ctx,
			k,
			topicId,
			moduleParams.TaskRewardAlpha,
			moduleParams.BetaEntropy,
			forecasters,
			forecastersRewardFractions,
		)
		if err != nil {
			return []TaskRewards{}, alloraMath.Dec{}, err
		}
	} else {
		// If there are no forecasters, set forecasting entropy to zero
		forecastingEntropy = alloraMath.ZeroDec()
	}

	// Get Total Rewards for Reputation task
	taskReputerReward, err = GetRewardForReputerTaskInTopic(
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicReward,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for reputer task in topic")
	}

	// Get Total Rewards for Inference task
	taskInferenceReward, err := GetRewardForInferenceTaskInTopic(
		lossBundles.NaiveValue,
		lossBundles.CombinedValue,
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicReward,
		moduleParams.SigmoidA,
		moduleParams.SigmoidB,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for inference task in topic")
	}

	// Get Total Rewards for Forecasting task
	taskForecastingReward, err := GetRewardForForecastingTaskInTopic(
		lossBundles.NaiveValue,
		lossBundles.CombinedValue,
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicReward,
		moduleParams.SigmoidA,
		moduleParams.SigmoidB,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for forecasting task in topic")
	}

	totalRewardsDistribution = make([]TaskRewards, 0)

	// Get Distribution of Rewards per Reputer
	reputerRewards, err := GetRewardPerReputer(
		ctx,
		k,
		topicId,
		taskReputerReward,
		reputers,
		reputersRewardFractions,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer rewards")
	}
	totalRewardsDistribution = append(totalRewardsDistribution, reputerRewards...)

	// Get Distribution of Rewards per Worker - Inference Task
	inferenceRewards, err := GetRewardPerWorker(
		topicId,
		WorkerInferenceRewardType,
		taskInferenceReward,
		inferers,
		inferersRewardFractions,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inference rewards")
	}
	totalRewardsDistribution = append(totalRewardsDistribution, inferenceRewards...)

	// Get Distribution of Rewards per Worker - Forecast Task
	forecastRewards, err := GetRewardPerWorker(
		topicId,
		WorkerForecastRewardType,
		taskForecastingReward,
		forecasters,
		forecastersRewardFractions,
	)
	if err != nil {
		return []TaskRewards{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get forecast rewards")
	}
	totalRewardsDistribution = append(totalRewardsDistribution, forecastRewards...)

	return totalRewardsDistribution, taskReputerReward, nil
}

// pay out the rewards to the participants
// this function moves tokens from the rewards module to the participants
// if it fails to pay a particular participant, it will continue to the next participant
func payoutRewards(
	ctx sdk.Context,
	k keeper.Keeper,
	rewards []TaskRewards,
) []error {
	ret := make([]error, 0)
	for _, reward := range rewards {
		address, err := sdk.AccAddressFromBech32(reward.Address.String())
		if err != nil {
			ret = append(ret, errors.Wrapf(err, "failed to decode payout address: %s", reward.Address.String()))
			continue
		}

		if reward.Reward.IsZero() {
			continue
		}

		rewardInt := reward.Reward.Abs().SdkIntTrim()
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, rewardInt))

		if reward.Type == ReputerRewardType {
			err = k.SendCoinsFromModuleToModule(ctx, types.AlloraRewardsAccountName, types.AlloraStakingAccountName, coins)
			if err != nil {
				ret = append(ret, errors.Wrapf(
					err,
					"failed to send coins from rewards module to staking module: %s",
					coins.String(),
				))
				continue
			}
			err = k.AddStake(ctx, reward.TopicId, reward.Address, cosmosMath.Uint(rewardInt))
			if err != nil {
				ret = append(ret, errors.Wrapf(err, "failed to add stake %s: %s", reward.Address.String(), rewardInt.String()))
				continue
			}
		} else {
			err = k.BankKeeper().SendCoinsFromModuleToAccount(
				ctx,
				types.AlloraRewardsAccountName,
				address,
				coins,
			)
			if err != nil {
				ret = append(ret, errors.Wrapf(
					err,
					"failed to send coins from rewards module to payout address %s, %s",
					types.AlloraRewardsAccountName,
					reward.Address.String(),
				))
				continue
			}
		}
	}

	return ret
}

func pruneRecordsAfterRewards(
	ctx sdk.Context,
	k keeper.Keeper,
	minEpochLengthRecordLimit int64,
	topicId uint64,
	topicRewardNonce int64,
) error {
	// Delete topic reward nonce
	err := k.DeleteTopicRewardNonce(ctx, topicId)
	if err != nil {
		return errors.Wrapf(err, "failed to delete topic reward nonce")
	}

	// Get oldest unfulfilled nonce - delete everything behind it
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return err
	}

	// Assume the oldest nonce is the topic reward nonce
	oldestNonce := topicRewardNonce
	// If there are unfulfilled nonces, find the oldest one
	if len(unfulfilledNonces.Nonces) > 0 {
		oldestNonce = unfulfilledNonces.Nonces[0].ReputerNonce.BlockHeight
		for _, nonce := range unfulfilledNonces.Nonces {
			if nonce.ReputerNonce.BlockHeight < oldestNonce {
				oldestNonce = nonce.ReputerNonce.BlockHeight
			}
		}
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errors.Wrapf(err, "failed to get topic")
	}

	// Prune records x EpochsLengths behind the oldest nonce
	// This is to leave the necessary data for the remaining
	// unfulfilled nonces to be fulfilled
	oldestNonce -= minEpochLengthRecordLimit * topic.EpochLength

	// Prune old records after rewards have been paid out
	err = k.PruneRecordsAfterRewards(ctx, topicId, oldestNonce)
	if err != nil {
		return errors.Wrapf(err, "failed to prune records after rewards")
	}

	return nil
}
