package rewards

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EmitRewards(
	ctx sdk.Context,
	k keeper.Keeper,
	blockHeight BlockHeight,
	weights map[uint64]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalRevenue cosmosMath.Int,
) error {
	totalReward, err := k.GetTotalRewardToDistribute(ctx)
	Logger(ctx).Debug(fmt.Sprintf("Reward to distribute this epoch: %s", totalReward.String()))
	if err != nil {
		return errors.Wrapf(err, "failed to get total reward to distribute")
	}
	if totalReward.IsZero() {
		Logger(ctx).Warn("The total scheduled rewards to distribute this epoch are zero!")
		return nil
	}

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get module params")
	}

	rewardableTopics, err := k.GetRewardableTopics(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get rewardable topics")
	}
	// Sorted, active topics by weight descending. Still need skim top N to truly be the churnable topics
	sortedChurnableTopics := alloraMath.GetSortedElementsByDecWeightDesc(rewardableTopics, weights)

	if len(sortedChurnableTopics) == 0 {
		Logger(ctx).Warn("No churnable topics found")
		return nil
	}

	// Top `N=MaxTopicsPerBlock` active topics of this block => the *actually* churnable topics
	if uint64(len(sortedChurnableTopics)) > moduleParams.MaxTopicsPerBlock {
		sortedChurnableTopics = sortedChurnableTopics[:moduleParams.MaxTopicsPerBlock]
	}

	// Get total weight of churnable topics
	sumWeightOfChurnableTopics := alloraMath.ZeroDec()
	for _, topicId := range sortedChurnableTopics {
		sumWeightOfChurnableTopics, err = sumWeightOfChurnableTopics.Add(*weights[topicId])
		if err != nil {
			return errors.Wrapf(err, "failed to add weight of top topics")
		}
	}

	// Revenue (above) is what was earned by topics in this timestep. Rewards are what are actually paid to topics => participants
	// The reward and revenue calculations are coupled here to minimize excessive compute
	topicRewards, err := CalcTopicRewards(ctx, k, weights, sortedChurnableTopics, sumWeightOfChurnableTopics, totalReward)
	if err != nil {
		return errors.Wrapf(err, "failed to calculate topic rewards")
	}

	// Calculate then pay out topic rewards to topic participants
	totalRewardToStakedReputers := alloraMath.ZeroDec() // This is used to communicate with the mint module
	for _, topicId := range sortedChurnableTopics {
		topicReward := topicRewards[topicId]
		if topicReward == nil {
			Logger(ctx).Warn(fmt.Sprintf("Topic %d has no reward, skipping", topicId))
			continue
		}
		// Get topic reward nonce/block height
		topicRewardNonce, err := k.GetTopicRewardNonce(ctx, topicId)
		// If the topic has no reward nonce, skip it
		if err != nil || topicRewardNonce == 0 {
			continue
		}

		// Distribute rewards between topic participants
		totalRewardsDistribution, rewardInTopicToReputers, err := GenerateRewardsDistributionByTopicParticipant(ctx, k, topicId, topicReward, topicRewardNonce, moduleParams)
		if err != nil {
			topicRewardString := "nil"
			Logger(ctx).Warn(
				fmt.Sprintf(
					"Failed to Generate Rewards for Topic, Skipping:\nTopic Id %d\nTopic Reward Amount %s\nError:\n%s\n\n",
					topicId,
					topicRewardString,
					err.Error(),
				),
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
				Logger(ctx).Warn(
					fmt.Sprintf(
						"Failed to pay out rewards to participant in Topic:\nTopic Id %d\nTopic Reward Amount %s\nError:\n%s\n\n",
						topicId,
						topicReward.String(),
						err.Error(),
					),
				)
			}
			continue
		}

		// Prune records after rewards have been paid out
		err = pruneRecordsAfterRewards(ctx, k, moduleParams.MinEpochLengthRecordLimit, topicId, topicRewardNonce)
		if err != nil {
			Logger(ctx).Warn(
				fmt.Sprintf(
					"Failed to prune records after rewards for Topic, Skipping:\nTopic Id %d\nTopic Reward Amount %s\nError:\n%s\n\n",
					topicId,
					topicReward.String(),
					err.Error(),
				),
			)
			continue
		}

		err = k.RemoveRewardableTopic(ctx, topicId)
		if err != nil {
			Logger(ctx).Warn(
				fmt.Sprintf(
					"Failed to remove rewardable topic:\nTopic Id %d\nError:\n%s\n\n",
					topicId,
					err.Error(),
				),
			)
			continue
		}
	}
	Logger(ctx).Debug(
		fmt.Sprintf("Paid out %s to staked reputers over %d topics",
			totalRewardToStakedReputers.String(),
			len(topicRewards)))
	if !totalReward.IsZero() && uint64(blockHeight)%moduleParams.BlocksPerMonth == 0 {
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
	}

	return nil
}

func CalcTopicRewards(
	ctx sdk.Context,
	k keeper.Keeper,
	weights map[uint64]*alloraMath.Dec,
	sortedTopics []uint64,
	sumWeight alloraMath.Dec,
	totalReward alloraMath.Dec,
) (
	map[uint64]*alloraMath.Dec,
	error,
) {
	topicRewards := make(map[TopicId]*alloraMath.Dec)
	for _, topicId := range sortedTopics {
		topicRewardFraction, err := GetTopicRewardFraction(weights[topicId], sumWeight)
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
	totalRewardsDistribution []types.TaskReward,
	taskReputerReward alloraMath.Dec,
	err error,
) {
	if topicReward == nil {
		return nil, alloraMath.Dec{}, types.ErrInvalidReward
	}
	bundles, err := k.GetReputerLossBundlesAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get network loss bundle at block %d", blockHeight)
	}

	lossBundles, err := k.GetNetworkLossBundleAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get network loss bundle at block %d", blockHeight)
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
	reputers, reputersRewardFractions, err := GetReputersRewardFractions(ctx, k, topicId, moduleParams.PRewardReputer, reputerScores)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer reward round data")
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
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer task entropy")
	}

	// Get inferer reward fractions
	inferers, inferersRewardFractions, err := GetInferenceTaskRewardFractions(
		ctx,
		k,
		topicId,
		blockHeight,
		moduleParams.PRewardInference,
		moduleParams.CRewardInference,
		infererScores,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inferer reward fractions")
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
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inference task entropy")
	}

	// Get forecaster reward fractions
	forecasters, forecastersRewardFractions, err := GetForecastingTaskRewardFractions(
		ctx,
		k,
		topicId,
		blockHeight,
		moduleParams.PRewardForecast,
		moduleParams.CRewardForecast,
		forecasterScores,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get forecaster reward fractions")
	}

	var forecastingEntropy alloraMath.Dec
	if len(forecasters) > 0 && len(inferers) > 1 {
		// Get forecasting entropy
		forecastingEntropy, err = GetForecastTaskEntropy(
			ctx,
			k,
			topicId,
			moduleParams.TaskRewardAlpha,
			moduleParams.BetaEntropy,
			forecasters,
			forecastersRewardFractions,
		)
		if err != nil {
			return []types.TaskReward{}, alloraMath.Dec{}, err
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
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for reputer task in topic")
	}

	// Get previous forecaster score ratio for topic
	previousForecasterScoreRatio, err := k.GetPreviousForecasterScoreRatio(ctx, topicId)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get previous forecast score ratio")
	}

	// Get chi (Forecasting Utility) and gamma (Normalization Factor)
	chi, gamma, err := GetChiAndGamma(
		lossBundles.NaiveValue,
		lossBundles.CombinedValue,
		inferenceEntropy,
		forecastingEntropy,
		infererScores,
		previousForecasterScoreRatio,
		moduleParams.TaskRewardAlpha,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get chi and gamma")
	}

	// Get Total Rewards for Inference task
	taskInferenceReward, err := GetRewardForInferenceTaskInTopic(
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicReward,
		chi,
		gamma,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for inference task in topic")
	}

	// Get Total Rewards for Forecasting task
	taskForecastingReward, err := GetRewardForForecastingTaskInTopic(
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		topicReward,
		chi,
		gamma,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for forecasting task in topic")
	}

	totalRewardsDistribution = make([]types.TaskReward, 0)

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
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer rewards")
	}
	totalRewardsDistribution = append(totalRewardsDistribution, reputerRewards...)

	// Get Distribution of Rewards per Worker - Inference Task
	inferenceRewards, err := GetRewardPerWorker(
		topicId,
		types.WorkerInferenceRewardType,
		taskInferenceReward,
		inferers,
		inferersRewardFractions,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inference rewards")
	}
	totalRewardsDistribution = append(totalRewardsDistribution, inferenceRewards...)

	// Get Distribution of Rewards per Worker - Forecast Task
	forecastRewards, err := GetRewardPerWorker(
		topicId,
		types.WorkerForecastRewardType,
		taskForecastingReward,
		forecasters,
		forecastersRewardFractions,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get forecast rewards")
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
	rewards []types.TaskReward,
) []error {
	ret := make([]error, 0) // errors to return from paying individual actors
	reputerAndDelegatorRewards := make([]types.TaskReward, 0)
	infererRewards := make([]types.TaskReward, 0)
	forecasterRewards := make([]types.TaskReward, 0)
	blockHeight := ctx.BlockHeight()
	for _, reward := range rewards {
		if reward.Reward.IsZero() {
			continue
		}
		if reward.Reward.IsNegative() {
			ret = append(ret, errors.Wrapf(
				types.ErrInvalidReward,
				"reward cannot be negative: { address: %s, amount: %s, topicId: %d, type: %s }",
				reward.Address,
				reward.Reward.String(),
				reward.TopicId,
				reward.Type,
			))
			continue
		}

		rewardInt := reward.Reward.SdkIntTrim()
		coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, rewardInt))

		if reward.Type == types.ReputerAndDelegatorRewardType {
			err := k.SendCoinsFromModuleToModule(ctx, types.AlloraRewardsAccountName, types.AlloraStakingAccountName, coins)
			if err != nil {
				ret = append(ret, errors.Wrapf(
					err,
					"failed to send coins from rewards module to staking module: %s",
					coins.String(),
				))
				continue
			}
			err = k.AddReputerStake(ctx, reward.TopicId, reward.Address, cosmosMath.Int(rewardInt))
			if err != nil {
				ret = append(ret, errors.Wrapf(err, "failed to add stake %s: %s", reward.Address, rewardInt.String()))
				continue
			}

			reputerAndDelegatorRewards = append(reputerAndDelegatorRewards, reward)
		} else {
			_, err := sdk.AccAddressFromBech32(reward.Address)
			if err != nil {
				ret = append(ret, errors.Wrapf(err, "failed to decode payout address: %s", reward.Address))
				continue
			}
			err = k.SendCoinsFromModuleToAccount(
				ctx,
				types.AlloraRewardsAccountName,
				reward.Address,
				coins,
			)
			if err != nil {
				ret = append(ret, errors.Wrapf(
					err,
					"failed to send coins from rewards module to payout address %s, %s",
					types.AlloraRewardsAccountName,
					reward.Address,
				))
				continue
			}

			if reward.Type == types.WorkerInferenceRewardType {
				infererRewards = append(infererRewards, reward)
			} else if reward.Type == types.WorkerForecastRewardType {
				forecasterRewards = append(forecasterRewards, reward)
			}
		}
	}

	types.EmitNewInfererRewardsSettledEvent(ctx, blockHeight, infererRewards)
	types.EmitNewForecasterRewardsSettledEvent(ctx, blockHeight, forecasterRewards)
	types.EmitNewReputerAndDelegatorRewardsSettledEvent(ctx, blockHeight, reputerAndDelegatorRewards)
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

func Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "rewards")
}
