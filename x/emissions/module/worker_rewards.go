package module

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TaskRewards struct {
	Address sdk.AccAddress
	Reward  float64
}

func GetWorkersRewardsInferenceTask(
	ctx sdk.Context,
	am AppModule,
	topicId uint64,
	block int64,
	preward float64,
	totalRewardsInferenceTask float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := am.keeper.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get last score for each worker
	var scores [][]*types.Score
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := am.keeper.GetLastWorkerInferenceScores(ctx, topicId, block, workerAddr)
		if err != nil {
			return nil, err
		}

		// Add worker score in the scores array
		scores = append(scores, workerLastScores)

		// Add worker address in the worker addresses array
		workerAddresses = append(workerAddresses, workerAddr)
	}

	// Convert scores to float64
	var scoresFloat64 [][]float64
	for _, workerScoreArray := range scores {
		var scoresOneOutFloat64 []float64
		for _, score := range workerScoreArray {
			scoresOneOutFloat64 = append(scoresOneOutFloat64, score.Score)
		}
		scoresFloat64 = append(scoresFloat64, scoresOneOutFloat64)
	}

	// Get Worker Reward Fractions
	workersRewardsFractions, err := GetWorkerRewardFractions(scoresFloat64, preward)
	if err != nil {
		return nil, err
	}

	// Get Worker Rewards
	var workerRewards []TaskRewards
	for i, rewardFraction := range workersRewardsFractions {
		workerReward := rewardFraction * totalRewardsInferenceTask
		workerRewards = append(workerRewards, TaskRewards{
			Address: workerAddresses[i],
			Reward:  workerReward,
		})
	}

	return workerRewards, nil
}

func GetWorkersRewardsForecastTask(
	ctx sdk.Context,
	am AppModule,
	topicId uint64,
	block int64,
	preward float64,
	totalRewardsForecastTask float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := am.keeper.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get new score for each worker
	var scores [][]*types.Score
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := am.keeper.GetLastWorkerForecastScores(ctx, topicId, block, workerAddr)
		if err != nil {
			return nil, err
		}

		// Add worker score in the scores array
		scores = append(scores, workerLastScores)

		// Add worker address in the worker addresses array
		workerAddresses = append(workerAddresses, workerAddr)
	}

	// Convert scores to float64
	var scoresFloat64 [][]float64
	for _, workerScoreArray := range scores {
		var scoresOneOutFloat64 []float64
		for _, score := range workerScoreArray {
			scoresOneOutFloat64 = append(scoresOneOutFloat64, score.Score)
		}
		scoresFloat64 = append(scoresFloat64, scoresOneOutFloat64)
	}

	// Get Worker Reward Fractions
	workersRewardsFractions, err := GetWorkerRewardFractions(scoresFloat64, preward)
	if err != nil {
		return nil, err
	}

	// Get Worker Rewards
	var workerRewards []TaskRewards
	for i, rewardFraction := range workersRewardsFractions {
		workerReward := rewardFraction * totalRewardsForecastTask
		workerRewards = append(workerRewards, TaskRewards{
			Address: workerAddresses[i],
			Reward:  workerReward,
		})
	}

	return workerRewards, nil
}
