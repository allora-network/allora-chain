package module

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: Add litepaper references
// TODO: Make sure to avoid duplicate rewarding/scoring
func GetWorkersRewardsInferenceTask(
	ctx sdk.Context,
	am AppModule,
	topicId uint64,
	block int64,
	preward float64,
	totalRewardsInferenceTask float64,
) ([]float64, error) {
	// Get Network Losses from the last block
	// TODO: Change this after merging with kenny's PR
	networkLosses, err := am.keeper.GetLossBundles(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get Score for each worker
	// TODO: Sort by block (?)
	var scores [][]float64
	for _, networkLosses := range networkLosses.LossBundles {

		// TODO: Sort by worker (?)
		var workersScores []float64
		for _, oneOutLoss := range networkLosses.OneOutLosses {
			workerScore := GetWorkerScore(float64(networkLosses.CombinedLoss.BigInt().Int64()), float64(oneOutLoss.Value.BigInt().Int64()))

			workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
			if err != nil {
				return nil, err
			}

			// Persist worker score
			am.keeper.InsertWorkerInferenceScore(ctx, topicId, block, types.Score{
				TopicId:     topicId,
				BlockNumber: block,
				Address:     workerAddr.String(),
				Score:       workerScore,
			})

			workersScores = append(workersScores, workerScore)
		}
		scores = append(scores, workersScores)
	}

	// Get Worker Reward Fractions
	workersRewardsFractions, err := GetWorkerRewardFractions(scores, preward)
	if err != nil {
		return nil, err
	}

	// Get Worker Rewards
	var workerRewards []float64
	for _, rewardFraction := range workersRewardsFractions {
		workerReward := rewardFraction * totalRewardsInferenceTask
		workerRewards = append(workerRewards, workerReward)
	}

	return workerRewards, nil
}

// TODO: Add litepaper references
func GetWorkersRewardsForecastTask(
	ctx sdk.Context,
	am AppModule,
	topicId uint64,
	block int64,
	preward float64,
	totalRewardsForecastTask float64,
) ([]float64, error) {
	// Get Network Losses from the last block
	// TODO: Change this after merging with kenny's PR
	networkLosses, err := am.keeper.GetLossBundles(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get Score for each worker
	// TODO: Sort by block (?)
	var scores [][]float64
	for _, networkLosses := range networkLosses.LossBundles {

		// TODO: Sort by worker (?)
		var workersScoresOneOut []float64
		for _, oneOutLoss := range networkLosses.OneOutLosses {
			workerScore := GetWorkerScore(float64(networkLosses.CombinedLoss.BigInt().Int64()), float64(oneOutLoss.Value.BigInt().Int64()))
			workersScoresOneOut = append(workersScoresOneOut, workerScore)
		}

		var workersScoresOneIn []float64
		for _, oneInNaiveLoss := range networkLosses.OneInNaiveLosses {
			workerScore := GetWorkerScore(float64(networkLosses.NaiveLoss.BigInt().Int64()), float64(oneInNaiveLoss.Value.BigInt().Int64()))
			workersScoresOneIn = append(workersScoresOneIn, workerScore)
		}

		var workersFinalScoresTimestep []float64
		numForecasters := len(workersScoresOneOut)
		fUniqueAgg := GetfUniqueAgg(float64(numForecasters))
		for i, workerScoreOneOut := range workersScoresOneOut {
			workerFinalScore := GetFinalWorkerScoreForecastTask(workersScoresOneIn[i], workerScoreOneOut, fUniqueAgg)

			workerAddr, err := sdk.AccAddressFromBech32(networkLosses.OneInNaiveLosses[i].Worker)
			if err != nil {
				return nil, err
			}

			// Persist worker score
			am.keeper.InsertWorkerForecastScore(ctx, topicId, block, types.Score{
				TopicId:     topicId,
				BlockNumber: block,
				Address:     workerAddr.String(),
				Score:       workerFinalScore,
			})

			workersFinalScoresTimestep = append(workersFinalScoresTimestep, workerFinalScore)
		}

		scores = append(scores, workersFinalScoresTimestep)
	}

	// Get Worker Reward Fractions
	workersRewardsFractions, err := GetWorkerRewardFractions(scores, preward)
	if err != nil {
		return nil, err
	}

	// Get Worker Rewards
	var workerRewards []float64
	for _, rewardFraction := range workersRewardsFractions {
		workerReward := rewardFraction * totalRewardsForecastTask
		workerRewards = append(workerRewards, workerReward)
	}

	return workerRewards, nil
}
