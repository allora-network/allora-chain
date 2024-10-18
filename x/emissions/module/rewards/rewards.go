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

type CalcTopicRewardsArgs struct {
	Ctx                             sdk.Context
	Weights                         map[uint64]*alloraMath.Dec // weights of all active topics in this block
	SortedTopics                    []uint64                   // topics sorted by weight in descending order
	SumTopicWeights                 alloraMath.Dec             // sum of all active topic weights
	TotalAvailableInRewardsTreasury alloraMath.Dec             // Maximum amount of rewards available in treasury
	EpochLengths                    map[uint64]int64           // epoch lengths for each topic
	CurrentRewardsEmissionPerBlock  alloraMath.Dec             // Rewards emission per block
}

type GenerateRewardsDistributionByTopicParticipantArgs struct {
	Ctx          sdk.Context
	K            keeper.Keeper
	TopicId      uint64
	TopicReward  *alloraMath.Dec
	BlockHeight  int64
	ModuleParams types.Params
}

type GetDistributionAndPayoutRewardsToTopicActorsArgs struct {
	Ctx              sdk.Context
	K                keeper.Keeper
	TopicId          uint64
	TopicRewardNonce int64
	TopicReward      *alloraMath.Dec
	ModuleParams     types.Params
}

type EmitRewardsArgs struct {
	Ctx          sdk.Context
	K            keeper.Keeper
	ModuleParams types.Params
	BlockHeight  BlockHeight
	Weights      map[uint64]*alloraMath.Dec
	SumWeight    alloraMath.Dec
	TotalRevenue cosmosMath.Int
}

func EmitRewards(args EmitRewardsArgs) error {
	// Get current total treasury, to confirm it covers the rewards to give
	totalRewardTreasury, err := args.K.GetTotalRewardToDistribute(args.Ctx)
	Logger(args.Ctx).Debug(fmt.Sprintf("Max rewards to distribute this epoch: %s", totalRewardTreasury.String()))
	if err != nil {
		return errors.Wrapf(err, "failed to get max rewards to distribute")
	}
	if totalRewardTreasury.IsZero() {
		Logger(args.Ctx).Warn("The rewards treasury account has a total value of zero on this epoch!")
		return nil
	}

	// Sorted, active topics by weight descending. Still need skim top N to truly be the rewardable topics
	sortedRewardableTopics := alloraMath.GetSortedElementsByDecWeightDesc(args.Weights)
	Logger(args.Ctx).Debug(fmt.Sprintf("Rewardable topics: %v", sortedRewardableTopics))

	if len(sortedRewardableTopics) == 0 {
		Logger(args.Ctx).Warn("No rewardable topics found")
		return nil
	}

	// Top `N=MaxTopicsPerBlock` active topics of this block => the *actually* rewardable topics
	if uint64(len(sortedRewardableTopics)) > args.ModuleParams.MaxActiveTopicsPerBlock {
		sortedRewardableTopics = sortedRewardableTopics[:args.ModuleParams.MaxActiveTopicsPerBlock]
	}

	// Get the global total sum of previous topic weights
	totalSumPreviousTopicWeights, err := args.K.GetTotalSumPreviousTopicWeights(args.Ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get total sum of previous topic weights")
	}
	if totalSumPreviousTopicWeights.IsZero() {
		return errors.Wrapf(types.ErrInvalidReward, "No total weights set, no rewards")
	}
	// Get epoch lengths for sorted rewardable topics
	epochLengths := make(map[uint64]int64)
	for _, topicId := range sortedRewardableTopics {
		topic, err := args.K.GetTopic(args.Ctx, topicId)
		if err != nil {
			return errors.Wrapf(err, "failed to get epoch length for topic %d", topicId)
		}
		epochLengths[topicId] = topic.EpochLength
	}

	// Get current block emission, to be extrapolated to be used in rewards calculation
	currentBlockEmission, err := args.K.GetRewardCurrentBlockEmission(args.Ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get current block emission")
	}
	Logger(args.Ctx).Debug(fmt.Sprintf("Current block emission: %s", currentBlockEmission.String()))

	currentBlockEmissionDec, err := alloraMath.NewDecFromSdkInt(currentBlockEmission)
	if err != nil {
		return errors.Wrapf(err, "failed to convert current block emission to decimal")
	}
	// Revenue (above) is what was earned by topics in this timestep. Rewards are what are actually paid to topics => participants
	// The reward and revenue calculations are coupled here to minimize excessive compute
	calcTopicRewardsArgs := CalcTopicRewardsArgs{
		Ctx:                             args.Ctx,
		Weights:                         args.Weights,
		SortedTopics:                    sortedRewardableTopics,
		SumTopicWeights:                 totalSumPreviousTopicWeights,
		TotalAvailableInRewardsTreasury: totalRewardTreasury,
		EpochLengths:                    epochLengths,
		CurrentRewardsEmissionPerBlock:  currentBlockEmissionDec,
	}
	topicRewards, err := CalcTopicRewards(calcTopicRewardsArgs)
	if err != nil {
		return errors.Wrapf(err, "failed to calculate topic rewards")
	}
	Logger(args.Ctx).Debug(fmt.Sprintf("Topic rewards: %v", topicRewards))

	// Initialize totalRewardToStakedReputers
	totalRewardToStakedReputers := alloraMath.ZeroDec()

	// Process rewards for each topic, pruning at the end of epoch
	for _, topicId := range sortedRewardableTopics {
		topicRewardNonce, err := args.K.GetTopicRewardNonce(args.Ctx, topicId)
		if err != nil || topicRewardNonce == 0 {
			Logger(args.Ctx).Info(fmt.Sprintf("Topic %d has no valid reward nonce, skipping", topicId))
			continue
		}
		// Defer pruning records after rewards payout
		defer func(topicId uint64, topicRewardNonce int64) {
			if err := pruneRecordsAfterRewards(args.Ctx, args.K, args.ModuleParams.MinEpochLengthRecordLimit, topicId, topicRewardNonce); err != nil {
				Logger(args.Ctx).Error(fmt.Sprintf("Failed to prune records after rewards for Topic %d, nonce: %d, err: %s", topicId, topicRewardNonce, err.Error()))
			}
		}(topicId, topicRewardNonce)

		topicReward := topicRewards[topicId]
		rewardInTopicToReputers, err := getDistributionAndPayoutRewardsToTopicActors(GetDistributionAndPayoutRewardsToTopicActorsArgs{
			Ctx:              args.Ctx,
			K:                args.K,
			TopicId:          topicId,
			TopicRewardNonce: topicRewardNonce,
			TopicReward:      topicReward,
			ModuleParams:     args.ModuleParams,
		})
		if err != nil {
			Logger(args.Ctx).Error(fmt.Sprintf("Failed to process rewards for topic %d: %s", topicId, err.Error()))
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
	Logger(args.Ctx).Debug(
		fmt.Sprintf("Paid out %s to staked reputers over %d topics",
			totalRewardToStakedReputers.String(),
			len(topicRewards)))

	if !totalRewardTreasury.IsZero() && uint64(args.BlockHeight)%args.ModuleParams.BlocksPerMonth == 0 {
		percentageToStakedReputers, err := totalRewardToStakedReputers.Quo(totalRewardTreasury)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate percentage to staked reputers")
		}
		err = args.K.SetPreviousPercentageRewardToStakedReputers(args.Ctx, percentageToStakedReputers)
		if err != nil {
			return errors.Wrapf(err, "failed to set previous percentage reward to staked reputers")
		}
	}

	// Emit reward of each topic
	types.EmitNewTopicRewardSetEvent(args.Ctx, topicRewards)
	return nil
}

// This function distributes and pays out rewards to topic actors based on their participation.
// It returns the total reward distributed to reputers
func getDistributionAndPayoutRewardsToTopicActors(args GetDistributionAndPayoutRewardsToTopicActorsArgs) (alloraMath.Dec, error) {
	Logger(args.Ctx).Debug(fmt.Sprintf("Generating rewards distribution for topic: %d, topicRewardNonce: %d, topicReward: %s", args.TopicId, args.TopicRewardNonce, args.TopicReward))

	// Get the distribution of rewards across actor types and participants in this topic
	totalRewardsDistribution, rewardInTopicToActors, err := GenerateRewardsDistributionByTopicParticipant(GenerateRewardsDistributionByTopicParticipantArgs{
		Ctx:          args.Ctx,
		K:            args.K,
		TopicId:      args.TopicId,
		TopicReward:  args.TopicReward,
		BlockHeight:  args.TopicRewardNonce,
		ModuleParams: args.ModuleParams,
	})
	if err != nil {
		return alloraMath.ZeroDec(), errors.Wrapf(err, "Failed to Generate Rewards for Topic %d", args.TopicId)
	}

	// Pay out rewards to topic participants
	payoutErrors := payoutRewards(args.Ctx, args.K, totalRewardsDistribution)
	if len(payoutErrors) > 0 {
		for _, payoutErr := range payoutErrors {
			Logger(args.Ctx).Warn(fmt.Sprintf("Failed to pay out rewards to participant in Topic %d: %s", args.TopicId, payoutErr.Error()))
		}
		return alloraMath.ZeroDec(), nil // continue to next topic
	}

	// Return rewardInTopicToReputers for summation in the main function
	return rewardInTopicToActors, nil
}

// Calculates the rewards for each topic.
//
// Calculates rewards per-block based on their weights vs all active topics.
// Rewards are then calculated per epoch, so topic's epochLength is used to calculate epoch rewards.
// Uses the current total amount rewardable in treasury as the max reward available to check against.
// If rewards treasury does not cover the computed rewards for the topics, the distribution is recalculated
// by distributing the rewards treasury fairly among all topics based on their weight and epoch length.
// Assumes SumTopicWeights to be nonzero
func CalcTopicRewards(args CalcTopicRewardsArgs) (
	topicRewards map[uint64]*alloraMath.Dec,
	err error,
) {
	// General case calculation
	topicRewards, totalTopicRewardsSum, err := calculateRewardsForCurrentTopics(args)
	if err != nil {
		return nil, errors.Wrapf(err, "calcTopicRewards: calculate rewards for current topics error")
	}
	if totalTopicRewardsSum.Gt(args.TotalAvailableInRewardsTreasury) {
		// Note this will happen when the treasury is not enough to cover the rewards for all topics
		// This will happen very rarely (cases where rewards emission changes dramatically), so optimization is done for the main case.
		Logger(args.Ctx).Warn("Treasury lower than calculated rewards. Distributing treasury equally among current topics.")
		if len(args.SortedTopics) == 1 {
			// most likely case, only one topic - so it gets the entire treasury directly
			topicId := args.SortedTopics[0]
			topicRewards[topicId] = &args.TotalAvailableInRewardsTreasury
		} else {
			topicRewards, err = calculateRewardsFromWholeTreasury(args)
			if err != nil {
				return nil, errors.Wrapf(err, "calcTopicRewards: distribute rewards treasury to current topics error")
			}
		}
	}
	return topicRewards, nil
}

// Calculates rewards per-block based on their weights vs all active topics.
// The notion emission per-block is used to calculate the rewards per-block.
// Rewards are then calculated per epoch, so topic's epochLength is used to calculate epoch rewards.
func calculateRewardsForCurrentTopics(args CalcTopicRewardsArgs) (topicRewards map[uint64]*alloraMath.Dec, totalTopicRewardsSum alloraMath.Dec, err error) {
	totalTopicRewardsSum = alloraMath.ZeroDec()
	topicRewards = make(map[uint64]*alloraMath.Dec)
	for _, topicId := range args.SortedTopics {
		topicWeight := args.Weights[topicId]
		topicRewardFraction, err := GetTopicRewardFraction(topicWeight, args.SumTopicWeights)
		if err != nil {
			return nil, alloraMath.Dec{}, errors.Wrapf(err, "topic reward fraction error")
		}
		if alloraMath.ZeroDec().Equal(topicRewardFraction) {
			args.Ctx.Logger().Warn(fmt.Sprintf("Skipping rewards for topic: %d, zero weights", topicId))
			continue
		}
		topicRewardPerBlock, err := GetTopicReward(topicRewardFraction, args.CurrentRewardsEmissionPerBlock)
		if err != nil {
			return nil, alloraMath.Dec{}, errors.Wrapf(err, "topic reward error")
		}
		epochLength := args.EpochLengths[topicId]
		if epochLength <= 0 {
			return nil, alloraMath.Dec{}, errors.Wrapf(types.ErrInvalidLengthTopic, "epoch length is nil or zero for topic %d", topicId)
		}
		epochLengthDec := alloraMath.NewDecFromInt64(epochLength)
		topicRewardPerEpoch, err := topicRewardPerBlock.Mul(epochLengthDec)
		if err != nil {
			return nil, alloraMath.Dec{}, errors.Wrapf(err, "calcTopicRewards:topic reward multiplication with epoch length fraction error")
		}
		topicRewards[topicId] = &topicRewardPerEpoch
		totalTopicRewardsSum, err = totalTopicRewardsSum.Add(topicRewardPerEpoch)
		if err != nil {
			return nil, alloraMath.Dec{}, errors.Wrapf(err, "calcTopicRewards: total topic rewards sum error")
		}
	}
	return topicRewards, totalTopicRewardsSum, nil
}

// Distributes the whole rewards treasury across the topics based on their factor
func calculateRewardsFromWholeTreasury(args CalcTopicRewardsArgs) (topicRewards map[uint64]*alloraMath.Dec, err error) {
	// Factors of each topic, relative to the total factor of topics. Not using weight because already used in this context.
	topicFactors := make(map[uint64]*alloraMath.Dec)
	totalTopicFactors := alloraMath.ZeroDec()
	// Rewards awarded to each topic
	topicRewards = make(map[uint64]*alloraMath.Dec)

	// Calculate topic factors
	for _, topicId := range args.SortedTopics {
		topicWeight := args.Weights[topicId]
		topicEpochLength := args.EpochLengths[topicId]
		topicFactor, err := topicWeight.Mul(alloraMath.NewDecFromInt64(topicEpochLength))
		if err != nil {
			return nil, errors.Wrapf(err, "distributeRewardsTreasuryToCurrentTopics: topic factor error")
		}
		topicFactors[topicId] = &topicFactor
		totalTopicFactors, err = totalTopicFactors.Add(topicFactor)
		if err != nil {
			return nil, errors.Wrapf(err, "distributeRewardsTreasuryToCurrentTopics: total topic factors sum error")
		}
	}

	// Distribute treasury across topics based on their factor
	for _, topicId := range args.SortedTopics {
		topicRewardPerEpoch, err := args.TotalAvailableInRewardsTreasury.Mul(*topicFactors[topicId])
		if err != nil {
			return nil, errors.Wrapf(err, "distributeRewardsTreasuryToCurrentTopics: topic reward per epoch error")
		}
		topicRewardPerEpoch, err = topicRewardPerEpoch.Quo(totalTopicFactors)
		if err != nil {
			return nil, errors.Wrapf(err, "distributeRewardsTreasuryToCurrentTopics: topic reward per epoch error")
		}
		topicRewards[topicId] = &topicRewardPerEpoch
	}

	return topicRewards, nil
}

// Calculates distribution of rewards to topic participants.
// Retrieves the reputer and network loss bundles.
// It then calculates and sets the reputer, inferer, and forecaster scores,
// then returning reward distributions
func GenerateRewardsDistributionByTopicParticipant(
	args GenerateRewardsDistributionByTopicParticipantArgs) (
	totalRewardsDistribution []types.TaskReward,
	taskReputerReward alloraMath.Dec,
	err error,
) {
	if args.TopicReward == nil {
		return nil, alloraMath.Dec{}, types.ErrInvalidReward
	}
	args.Ctx.Logger().Debug(fmt.Sprintf("Generating rewards distribution for topic: %d, block: %d, topicReward: %s", args.TopicId, args.BlockHeight, args.TopicReward.String()))
	bundles, err := args.K.GetReputerLossBundlesAtBlock(args.Ctx, args.TopicId, args.BlockHeight)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer loss bundle at block %d", args.BlockHeight)
	}
	if bundles != nil && len(bundles.ReputerValueBundles) == 0 {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(types.ErrInvalidReward, "empty reputer loss bundles")
	}

	lossBundles, err := args.K.GetNetworkLossBundleAtBlock(args.Ctx, args.TopicId, args.BlockHeight)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get network loss bundle at block %d", args.BlockHeight)
	}

	// Calculate and Set the reputer scores
	reputerScores, err := GenerateReputerScores(args.Ctx, args.K, args.TopicId, args.BlockHeight, *bundles)
	if err != nil {
		return nil, alloraMath.Dec{}, err
	}
	if len(reputerScores) == 0 {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(types.ErrInvalidReward, "empty reputer scores")
	}

	// Calculate and Set the worker scores for their inference work
	infererScores, err := GenerateInferenceScores(args.Ctx, args.K, args.TopicId, args.BlockHeight, *lossBundles)
	if err != nil {
		return nil, alloraMath.Dec{}, err
	}

	// Calculate and Set the worker scores for their forecast work
	forecasterScores, err := GenerateForecastScores(args.Ctx, args.K, args.TopicId, args.BlockHeight, *lossBundles)
	if err != nil {
		return nil, alloraMath.Dec{}, err
	}

	// Get reputer participants' addresses and reward fractions to be used in the reward round for topic
	reputers, reputersRewardFractions, err := GetReputersRewardFractions(args.Ctx, args.K, args.TopicId, args.ModuleParams.PRewardReputer, reputerScores)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer reward round data")
	}

	// Get reputer task entropy
	reputerEntropy, err := GetReputerTaskEntropy(
		args.Ctx,
		args.K,
		args.TopicId,
		args.ModuleParams.TaskRewardAlpha,
		args.ModuleParams.BetaEntropy,
		reputers,
		reputersRewardFractions,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reputer task entropy")
	}

	// Get inferer reward fractions
	inferers, inferersRewardFractions, err := GetInferenceTaskRewardFractions(
		args.Ctx,
		args.K,
		args.TopicId,
		args.BlockHeight,
		args.ModuleParams.PRewardInference,
		args.ModuleParams.CRewardInference,
		infererScores,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inferer reward fractions")
	}

	// Get inference entropy
	inferenceEntropy, err := GetInferenceTaskEntropy(
		args.Ctx,
		args.K,
		args.TopicId,
		args.ModuleParams.TaskRewardAlpha,
		args.ModuleParams.BetaEntropy,
		inferers,
		inferersRewardFractions,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get inference task entropy")
	}

	// Get forecaster reward fractions
	forecasters, forecastersRewardFractions, err := GetForecastingTaskRewardFractions(
		args.Ctx,
		args.K,
		args.TopicId,
		args.BlockHeight,
		args.ModuleParams.PRewardForecast,
		args.ModuleParams.CRewardForecast,
		forecasterScores,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get forecaster reward fractions")
	}

	var forecastingEntropy alloraMath.Dec
	if len(forecasters) > 0 && len(inferers) > 1 {
		// Get forecasting entropy
		forecastingEntropy, err = GetForecastTaskEntropy(
			args.Ctx,
			args.K,
			args.TopicId,
			args.ModuleParams.TaskRewardAlpha,
			args.ModuleParams.BetaEntropy,
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
		args.TopicReward,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for reputer task in topic")
	}

	// Get previous forecaster score ratio for topic
	previousForecasterScoreRatio, err := args.K.GetPreviousForecasterScoreRatio(args.Ctx, args.TopicId)
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
		args.ModuleParams.TaskRewardAlpha,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get chi and gamma")
	}
	types.EmitNewForecastTaskUtilityScoreSetEvent(args.Ctx, args.TopicId, forecastingTaskUtilityScore)

	// Set updated forecaster score ratio
	err = args.K.SetPreviousForecasterScoreRatio(args.Ctx, args.TopicId, updatedForecasterScoreRatio)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to set previous forecast score ratio")
	}

	// Get Total Rewards for Inference task
	taskInferenceReward, err := GetRewardForInferenceTaskInTopic(
		inferenceEntropy,
		forecastingEntropy,
		reputerEntropy,
		args.TopicReward,
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
		args.TopicReward,
		chi,
		gamma,
	)
	if err != nil {
		return []types.TaskReward{}, alloraMath.Dec{}, errors.Wrapf(err, "failed to get reward for forecasting task in topic")
	}

	totalRewardsDistribution = make([]types.TaskReward, 0)

	// Get Distribution of Rewards per Reputer
	reputerRewards, err := GetRewardPerReputer(
		args.Ctx,
		args.K,
		args.TopicId,
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
		args.TopicId,
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
		args.TopicId,
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
