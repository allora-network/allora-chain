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
	moduleParams types.Params,
	blockHeight BlockHeight,
	weights map[uint64]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalRevenue cosmosMath.Int,
) error {
	// Get current total treasury, to confirm it covers the rewards to give
	totalReward, err := k.GetTotalRewardToDistribute(ctx)
	Logger(ctx).Debug(fmt.Sprintf("Max rewards to distribute this epoch: %s", totalReward.String()))
	if err != nil {
		return errors.Wrapf(err, "failed to get max rewards to distribute")
	}
	if totalReward.IsZero() {
		Logger(ctx).Warn("The rewards treasury account has a total value of zero on this epoch!")
		return nil
	}

	// Sorted, active topics by weight descending. Still need skim top N to truly be the rewardable topics
	sortedRewardableTopics := alloraMath.GetSortedElementsByDecWeightDesc(weights)
	Logger(ctx).Debug(fmt.Sprintf("Rewardable topics: %v", sortedRewardableTopics))

	if len(sortedRewardableTopics) == 0 {
		Logger(ctx).Warn("No rewardable topics found")
		return nil
	}

	// Top `N=MaxTopicsPerBlock` active topics of this block => the *actually* rewardable topics
	if uint64(len(sortedRewardableTopics)) > moduleParams.MaxActiveTopicsPerBlock {
		sortedRewardableTopics = sortedRewardableTopics[:moduleParams.MaxActiveTopicsPerBlock]
	}

	// Get the global total sum of previous topic weights
	totalSumPreviousTopicWeights, err := k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get total sum of previous topic weights")
	}
	// Get epoch lengths for sorted rewardable topics
	epochLengths := make(map[uint64]int64)
	for _, topicId := range sortedRewardableTopics {
		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			return errors.Wrapf(err, "failed to get epoch length for topic %d", topicId)
		}
		epochLengths[topicId] = topic.EpochLength
	}

	// Get current block emission, to be extrapolated to be used in rewards calculation
	currentBlockEmission, err := k.GetRewardCurrentBlockEmission(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get current block emission")
	}
	Logger(ctx).Debug(fmt.Sprintf("Current block emission: %s", currentBlockEmission.String()))

	currentBlockEmissionDec, err := alloraMath.NewDecFromSdkInt(currentBlockEmission)
	if err != nil {
		return errors.Wrapf(err, "failed to convert current block emission to decimal")
	}
	// Revenue (above) is what was earned by topics in this timestep. Rewards are what are actually paid to topics => participants
	// The reward and revenue calculations are coupled here to minimize excessive compute
	topicRewards, err := CalcTopicRewards(ctx, weights, sortedRewardableTopics,
		totalSumPreviousTopicWeights, totalReward, epochLengths, currentBlockEmissionDec)
	if err != nil {
		return errors.Wrapf(err, "failed to calculate topic rewards")
	}
	Logger(ctx).Debug(fmt.Sprintf("Topic rewards: %v", topicRewards))

	// Initialize totalRewardToStakedReputers
	totalRewardToStakedReputers := alloraMath.ZeroDec()

	// Process rewards for each topic, pruning at the end of epoch
	for _, topicId := range sortedRewardableTopics {
		topicRewardNonce, err := k.GetTopicRewardNonce(ctx, topicId)
		if err != nil || topicRewardNonce == 0 {
			Logger(ctx).Info(fmt.Sprintf("Topic %d has no valid reward nonce, skipping", topicId))
			continue
		}
		// Defer pruning records after rewards payout
		defer func(topicId uint64, topicRewardNonce int64) {
			if err := pruneRecordsAfterRewards(ctx, k, moduleParams.MinEpochLengthRecordLimit, topicId, topicRewardNonce); err != nil {
				Logger(ctx).Error(fmt.Sprintf("Failed to prune records after rewards for Topic %d, nonce: %d, err: %s", topicId, topicRewardNonce, err.Error()))
			}
		}(topicId, topicRewardNonce)

		topicReward := topicRewards[topicId]
		if topicReward == nil {
			Logger(ctx).Warn(fmt.Sprintf("Topic %d has no reward, skipping", topicId))
			continue
		}
		rewardInTopicToReputers, err := getDistributionAndPayoutRewardsToTopicActors(ctx, k, topicId, topicRewardNonce, topicReward, moduleParams)
		if err != nil {
			Logger(ctx).Error(fmt.Sprintf("Failed to process rewards for topic %d: %s", topicId, err.Error()))
			continue
		}

		// Add rewardInTopicToReputers to totalRewardToStakedReputers
		totalRewardToStakedReputers, err = totalRewardToStakedReputers.Add(rewardInTopicToReputers)
		if err != nil {
			return errors.Wrapf(
				err,
				"Error finding sum of rewards to Reputers: totalReward: %s , rewardInTopic: %s",
				totalRewardToStakedReputers.String(),
				rewardInTopicToReputers.String(),
			)
		}
	}

	// Log and handle the final totalRewardToStakedReputers
	Logger(ctx).Debug(
		fmt.Sprintf("Paid out %s to staked reputers over %d topics",
			totalRewardToStakedReputers.String(),
			len(topicRewards)))

	if !totalReward.IsZero() && uint64(blockHeight)%moduleParams.BlocksPerMonth == 0 {
		percentageToStakedReputers, err := totalRewardToStakedReputers.Quo(totalReward)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate percentage to staked reputers")
		}
		err = k.SetPreviousPercentageRewardToStakedReputers(ctx, percentageToStakedReputers)
		if err != nil {
			return errors.Wrapf(err, "failed to set previous percentage reward to staked reputers")
		}
	}

	// Emit reward of each topic
	types.EmitNewTopicRewardSetEvent(ctx, topicRewards)
	return nil
}

// This function distributes and pays out rewards to topic actors based on their participation.
// It returns the total reward distributed to reputers
func getDistributionAndPayoutRewardsToTopicActors(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	topicRewardNonce int64,
	topicReward *alloraMath.Dec,
	// topicRewards map[uint64]*alloraMath.Dec,
	moduleParams types.Params,
) (alloraMath.Dec, error) {
	Logger(ctx).Debug(fmt.Sprintf("Distributing rewards for topic %d", topicId))

	Logger(ctx).Debug(fmt.Sprintf("Generating rewards distribution for topic: %d, topicRewardNonce: %d, topicReward: %s", topicId, topicRewardNonce, topicReward))

	// Get the distribution of rewards across actor types and participants in this topic
	totalRewardsDistribution, rewardInTopicToActors, err := GenerateRewardsDistributionByTopicParticipant(ctx, k, topicId, topicReward, topicRewardNonce, moduleParams)
	if err != nil {
		return alloraMath.ZeroDec(), errors.Wrapf(err, "Failed to Generate Rewards for Topic %d", topicId)
	}

	// Pay out rewards to topic participants
	payoutErrors := payoutRewards(ctx, k, totalRewardsDistribution)
	if len(payoutErrors) > 0 {
		for _, payoutErr := range payoutErrors {
			Logger(ctx).Warn(fmt.Sprintf("Failed to pay out rewards to participant in Topic %d: %s", topicId, payoutErr.Error()))
		}
		return alloraMath.ZeroDec(), nil // continue to next topic
	}

	// Return rewardInTopicToReputers for summation in the main function
	return rewardInTopicToActors, nil
}

func CalcTopicRewards(
	ctx sdk.Context,
	weights map[uint64]*alloraMath.Dec,
	sortedTopics []uint64,
	sumWeight alloraMath.Dec, // sum of all topic weights
	totalReward alloraMath.Dec, // maxTotalReward in treasury
	epochLengths map[uint64]int64, // epoch lengths for each topic
	currentBlockEmissionDec alloraMath.Dec, // perBlockEmission
) (
	map[uint64]*alloraMath.Dec,
	error,
) {
	if sumWeight.IsZero() {
		return nil, errors.Wrapf(types.ErrInvalidReward, "total reward is zero")
	}
	totalTopicRewardsSum := alloraMath.ZeroDec()
	topicRewards := make(map[TopicId]*alloraMath.Dec)
	for _, topicId := range sortedTopics {
		topicWeight := weights[topicId]
		if topicWeight == nil {
			zero := alloraMath.ZeroDec()
			topicWeight = &zero
		}
		topicRewardFraction, err := GetTopicRewardFraction(topicWeight, sumWeight)
		if err != nil {
			return nil, errors.Wrapf(err, "topic reward fraction error")
		}
		topicRewardPerBlock, err := GetTopicReward(topicRewardFraction, currentBlockEmissionDec)
		if err != nil {
			return nil, errors.Wrapf(err, "topic reward error")
		}
		epochLength := epochLengths[topicId]
		if epochLength <= 0 {
			return nil, errors.Wrapf(types.ErrInvalidLengthTopic, "epoch length is nil or zero for topic %d", topicId)
		}
		epochLengthDec := alloraMath.NewDecFromInt64(epochLength)
		topicRewardPerEpoch, err := topicRewardPerBlock.Mul(epochLengthDec)
		if err != nil {
			return nil, errors.Wrapf(err, "calcTopicRewards:topic reward multiplication with epoch length fraction error")
		}
		topicRewards[topicId] = &topicRewardPerEpoch
		totalTopicRewardsSum, err = totalTopicRewardsSum.Add(topicRewardPerEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "calcTopicRewards: total topic rewards sum error")
		}
	}
	if totalTopicRewardsSum.Gt(totalReward) {
		return nil, errors.Wrapf(types.ErrInvalidReward, "total topic rewards sum %s is greater than total reward in treasury %s, cancelling rewards distribution", totalTopicRewardsSum.String(), totalReward.String())
	}
	return topicRewards, nil
}

// Calculates distribution of rewards to topic participants.
// Retrieves the reputer and network loss bundles.
// It then calculates and sets the reputer, inferer, and forecaster scores,
// then returning reward distributions
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
	ctx.Logger().Debug(fmt.Sprintf("Generating rewards distribution for topic: %d, block: %d, topicReward: %s", topicId, blockHeight, topicReward.String()))
	bundles, err := k.GetReputerLossBundlesAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer loss bundle at block %d", blockHeight)
	}
	if bundles != nil && len(bundles.ReputerValueBundles) == 0 {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(types.ErrInvalidReward, "empty reputer loss bundles")
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
	if len(reputerScores) == 0 {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(types.ErrInvalidReward, "empty reputer scores")
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
	chi, gamma, updatedForecasterScoreRatio, forecastingTaskUtilityScore, err := GetChiAndGamma(
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
	types.EmitNewForecastTaskUtilityScoreSetEvent(ctx, topicId, forecastingTaskUtilityScore)

	// Set updated forecaster score ratio
	err = k.SetPreviousForecasterScoreRatio(ctx, topicId, updatedForecasterScoreRatio)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to set previous forecast score ratio")
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

		rewardInt, err := reward.Reward.SdkIntTrim()
		if err != nil {
			ret = append(ret, errors.Wrapf(err, "failed to convert reward to sdk.Int: %s", reward.Reward.String()))
			continue
		}
		if rewardInt.IsZero() {
			continue
		}
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
			err = k.AddReputerStake(ctx, reward.TopicId, reward.Address, rewardInt)
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

				err := k.IncrementCountInfererInclusionsInTopic(ctx, reward.TopicId, reward.Address)
				if err != nil {
					ret = append(ret, errors.Wrapf(err, "failed to increment count inferer inclusions in topic"))
					continue
				}
			} else if reward.Type == types.WorkerForecastRewardType {
				forecasterRewards = append(forecasterRewards, reward)

				err := k.IncrementCountForecasterInclusionsInTopic(ctx, reward.TopicId, reward.Address)
				if err != nil {
					ret = append(ret, errors.Wrapf(err, "failed to increment count forecaster inclusions in topic"))
					continue
				}
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
	if oldestNonce < 0 {
		oldestNonce = 0
	}
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
