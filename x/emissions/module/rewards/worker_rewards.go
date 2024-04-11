package rewards

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TaskRewards struct {
	Address sdk.AccAddress
	Reward  alloraMath.Dec
}

func GetWorkersRewardsInferenceTask(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	preward alloraMath.Dec,
	totalInferenceRewards alloraMath.Dec,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, _, err := keeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get last score for each worker
	var scoresDec [][]alloraMath.Dec
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
		var workerLastScoresDec []alloraMath.Dec
		for _, score := range workerLastScores {
			workerLastScoresDec = append(workerLastScoresDec, score.Score)
		}
		scoresDec = append(scoresDec, workerLastScoresDec)
	}

	// Get worker portion of rewards
	rewards, err := GetWorkerPortionOfRewards(scoresDec, preward, totalInferenceRewards, workerAddresses)

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
	preward alloraMath.Dec,
	totalForecastRewards alloraMath.Dec,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, _, err := keeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get new score for each worker
	var scoresDec [][]alloraMath.Dec
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

		// Convert scores to alloraMath.Dec
		var workerLastScoresDec []alloraMath.Dec
		for _, score := range workerLastScores {
			workerLastScoresDec = append(workerLastScoresDec, score.Score)
		}
		scoresDec = append(scoresDec, workerLastScoresDec)
	}

	// Get worker portion of rewards
	rewards, err := GetWorkerPortionOfRewards(scoresDec, preward, totalForecastRewards, workerAddresses)

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
		avg, err := keeper.GetAverageWorkerReward(ctx, topicId, reward.Address)
		if err != nil {
			continue
		}
		avgValueTimesCount, err := avg.Value.Mul(alloraMath.NewDecFromInt64(int64(avg.Count)))
		if err != nil {
			continue
		}
		totalRewards, err := avgValueTimesCount.Add(reward.Reward)
		if err != nil {
			continue
		}
		avg.Count += 1
		avg.Value, err = totalRewards.Quo(alloraMath.NewDecFromInt64(int64(avg.Count)))
		if err != nil {
			continue
		}
		_ = keeper.SetAverageWorkerReward(ctx, topicId, reward.Address, avg)
		fee, err := CalculateWorkerTax(avg.Value)
		if err != nil {
			continue
		}
		reward.Reward, err = reward.Reward.Sub(fee)
		if err != nil {
			continue
		}
		if reward.Reward.Lt(alloraMath.ZeroDec()) {
			reward.Reward = alloraMath.ZeroDec()
		}
		result = append(result, TaskRewards{
			Address: reward.Address,
			Reward:  reward.Reward,
		})
	}

	return result, nil
}
