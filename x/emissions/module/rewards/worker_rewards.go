package rewards

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TaskRewards struct {
	Address sdk.AccAddress
	Reward  float64
}

func GetWorkersRewardsInferenceTask(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	preward float64,
	totalInferenceRewards float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := keeper.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get last score for each worker
	var scoresFloat64 [][]float64
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutInfererValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := keeper.GetWorkerInferenceScoresUntilBlock(ctx, topicId, block, workerAddr)
		if err != nil {
			return nil, err
		}

		// Add worker address in the worker addresses array
		workerAddresses = append(workerAddresses, workerAddr)

		// Convert scores to float64
		var workerLastScoresFloat64 []float64
		for _, score := range workerLastScores {
			workerLastScoresFloat64 = append(workerLastScoresFloat64, score.Score)
		}
		scoresFloat64 = append(scoresFloat64, workerLastScoresFloat64)
	}

	// Get worker portion of rewards
	rewards, err := GetWorkerPortionOfRewards(scoresFloat64, preward, totalInferenceRewards, workerAddresses)

	if err != nil {
		return nil, err
	}
	return GetRewardsWithOutTax(ctx, keeper, rewards, topicId)
}

func GetWorkersRewardsForecastTask(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	preward float64,
	totalForecastRewards float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := keeper.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get new score for each worker
	var scoresFloat64 [][]float64
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutForecasterValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := keeper.GetWorkerForecastScoresUntilBlock(ctx, topicId, block, workerAddr)
		if err != nil {
			return nil, err
		}

		// Add worker address in the worker addresses array
		workerAddresses = append(workerAddresses, workerAddr)

		// Convert scores to float64
		var workerLastScoresFloat64 []float64
		for _, score := range workerLastScores {
			workerLastScoresFloat64 = append(workerLastScoresFloat64, score.Score)
		}
		scoresFloat64 = append(scoresFloat64, workerLastScoresFloat64)
	}

	// Get worker portion of rewards
	rewards, err := GetWorkerPortionOfRewards(scoresFloat64, preward, totalForecastRewards, workerAddresses)

	if err != nil {
		return nil, err
	}

	return GetRewardsWithOutTax(ctx, keeper, rewards, topicId)
}

func GetRewardsWithOutTax(
	ctx sdk.Context,
	keeper keeper.Keeper,
	rewards []TaskRewards,
	topicId uint64,
) ([]TaskRewards, error) {

	var result []TaskRewards
	// Get average reward for this worker
	for _, reward := range rewards {
		avg, e := keeper.GetAverageWorkerReward(ctx, topicId, reward.Address)
		if e != nil {
			continue
		}
		totalRewards := avg.Value*float64(avg.Count) + reward.Reward
		avg.Count += 1
		avg.Value = totalRewards / float64(avg.Count)
		_ = keeper.SetAverageWorkerReward(ctx, topicId, reward.Address, avg)
		fee := CalculateWorkerTax(avg.Value)
		reward.Reward -= fee
		if reward.Reward < 0 {
			reward.Reward = 0
		}
		result = append(result, TaskRewards{
			Address: reward.Address,
			Reward:  reward.Reward,
		})
	}

	return result, nil
}
