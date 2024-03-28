package module

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TaskRewards struct {
	Address sdk.AccAddress
	Reward  float64
}

// TODO: Add litepaper references
// TODO: Make sure to avoid duplicate rewarding/scoring
func GetWorkersRewardsInferenceTask(
	ctx sdk.Context,
	am AppModule,
	topicId uint64,
	block int64,
	preward float64,
	totalRewardsInferenceTask float64,
) ([]TaskRewards, error) {
	// Get network loss
	networkLosses, err := am.keeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get new score for each worker
	// TODO: Sort by worker (?)
	var scores [][]types.Score
	var workerAddresses []sdk.AccAddress
	for _, oneOutLoss := range networkLosses.OneOutLosses {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := am.keeper.GetLastWorkerInferenceScores(ctx, topicId, block, workerAddr)
		if err != nil {
			return nil, err
		}

		// Calculate new score
		workerNewScore := GetWorkerScore(float64(networkLosses.CombinedLoss.BigInt().Int64()), float64(oneOutLoss.Value.BigInt().Int64()))

		scoreToAdd := types.Score{
			TopicId:     topicId,
			BlockNumber: block,
			Address:     workerAddr.String(),
			Score:       workerNewScore,
		}

		// Persist worker score
		am.keeper.InsertWorkerInferenceScore(ctx, topicId, block, scoreToAdd)

		// Add new score in the last scores array
		workerLastScores = append(workerLastScores, scoreToAdd)

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
	networkLosses, err := am.keeper.GetNetworkLossBundleAtOrBeforeBlock(ctx, topicId, block)
	if err != nil {
		return nil, err
	}

	// Get worker scores for one out loss
	var workersScoresOneOut []float64
	for _, oneOutLoss := range networkLosses.OneOutLosses {
		workerScore := GetWorkerScore(float64(networkLosses.CombinedLoss.BigInt().Int64()), float64(oneOutLoss.Value.BigInt().Int64()))
		workersScoresOneOut = append(workersScoresOneOut, workerScore)
	}

	// Get worker scores for one in naive loss
	var workersScoresOneIn []float64
	for _, oneInNaiveLoss := range networkLosses.OneInNaiveLosses {
		workerScore := GetWorkerScore(float64(networkLosses.NaiveLoss.BigInt().Int64()), float64(oneInNaiveLoss.Value.BigInt().Int64()))
		workersScoresOneIn = append(workersScoresOneIn, workerScore)
	}

	// Get new score for each worker
	numForecasters := len(workersScoresOneOut)
	fUniqueAgg := GetfUniqueAgg(float64(numForecasters))
	var scores [][]types.Score
	var workerAddresses []sdk.AccAddress
	for i, workerScoreOneOut := range workersScoresOneOut {
		workerAddr, err := sdk.AccAddressFromBech32(networkLosses.OneInNaiveLosses[i].Worker)
		if err != nil {
			return nil, err
		}

		// Get worker last scores
		workerLastScores, err := am.keeper.GetLastWorkerForecastScores(ctx, topicId, block, workerAddr)
		if err != nil {
			return nil, err
		}

		// Calculate new score
		workerFinalScore := GetFinalWorkerScoreForecastTask(workersScoresOneIn[i], workerScoreOneOut, fUniqueAgg)

		scoreToAdd := types.Score{
			TopicId:     topicId,
			BlockNumber: block,
			Address:     workerAddr.String(),
			Score:       workerFinalScore,
		}

		// Persist worker score
		am.keeper.InsertWorkerForecastScore(ctx, topicId, block, scoreToAdd)

		// Add new score in the last scores array
		workerLastScores = append(workerLastScores, scoreToAdd)

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
