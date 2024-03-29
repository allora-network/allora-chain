package module

import (
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
	totalRewards float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := am.keeper.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get last score for each worker
	var scoresFloat64 [][]float64
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := am.keeper.GetWorkerInferenceScoresUntilBlock(ctx, topicId, block, workerAddr)
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
	return GetWorkerPortionOfRewards(scoresFloat64, preward, totalRewards, workerAddresses)
}

func GetWorkersRewardsForecastTask(
	ctx sdk.Context,
	am AppModule,
	topicId uint64,
	block int64,
	preward float64,
	totalRewards float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := am.keeper.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get new score for each worker
	var scoresFloat64 [][]float64
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := am.keeper.GetWorkerForecastScoresUntilBlock(ctx, topicId, block, workerAddr)
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
	return GetWorkerPortionOfRewards(scoresFloat64, preward, totalRewards, workerAddresses)
}
