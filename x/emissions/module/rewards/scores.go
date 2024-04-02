package rewards

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GenerateReputerScores(ctx sdk.Context, keeper keeper.Keeper, topicId uint64, block int64, reportedLosses types.ReputerValueBundles) ([]types.Score, error) {
	// Get reputers informations
	var reputerAddresses []sdk.AccAddress
	var reputerStakes []float64
	var reputerListeningCoefficients []float64
	var losses [][]float64
	for _, reportedLoss := range reportedLosses.ReputerValueBundles {
		reputerAddr, err := sdk.AccAddressFromBech32(reportedLoss.Reputer)
		if err != nil {
			return []types.Score{}, err
		}
		reputerAddresses = append(reputerAddresses, reputerAddr)

		// Get reputer topic stake
		reputerStake, err := keeper.GetStakeOnTopicFromReputer(ctx, topicId, reputerAddr)
		if err != nil {
			return []types.Score{}, err
		}
		reputerStakes = append(reputerStakes, float64(reputerStake.BigInt().Int64()))

		// Get reputer listening coefficient
		res, err := keeper.GetListeningCoefficient(ctx, topicId, reputerAddr)
		if err != nil {
			return []types.Score{}, err
		}
		reputerListeningCoefficients = append(reputerListeningCoefficients, res.Coefficient)

		// Get all reported losses from bundle
		reputerLosses := ExtractValues(reportedLoss.ValueBundle)
		losses = append(losses, reputerLosses)
	}

	// Get reputer output
	scores, newCoefficients, err := GetAllReputersOutput(losses, reputerStakes, reputerListeningCoefficients, len(reputerStakes))
	if err != nil {
		return []types.Score{}, err
	}

	// Insert new coeffients and scores
	var newScores []types.Score
	for i, reputerAddr := range reputerAddresses {
		err := keeper.SetListeningCoefficient(ctx, topicId, reputerAddr, types.ListeningCoefficient{Coefficient: newCoefficients[i]})
		if err != nil {
			return []types.Score{}, err
		}

		newScore := types.Score{
			TopicId:     topicId,
			BlockNumber: block,
			Address:     reputerAddr.String(),
			Score:       scores[i],
		}
		err = keeper.InsertReputerScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, err
		}
		newScores = append(newScores, newScore)
	}

	return newScores, nil
}

func GenerateInferenceScores(ctx sdk.Context, keeper keeper.Keeper, topicId uint64, block int64, networkLosses types.ValueBundle) ([]types.Score, error) {
	var newScores []types.Score
	for _, oneOutLoss := range networkLosses.OneOutValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneOutLoss.Worker)
		if err != nil {
			return []types.Score{}, err
		}

		// Calculate new score
		workerNewScore := GetWorkerScore(networkLosses.CombinedValue, oneOutLoss.Value)

		newScore := types.Score{
			TopicId:     topicId,
			BlockNumber: block,
			Address:     workerAddr.String(),
			Score:       workerNewScore,
		}
		err = keeper.InsertWorkerInferenceScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, err
		}
		newScores = append(newScores, newScore)
	}
	return newScores, nil
}

func GenerateForecastScores(ctx sdk.Context, keeper keeper.Keeper, topicId uint64, block int64, networkLosses types.ValueBundle) ([]types.Score, error) {
	// Get worker scores for one out loss
	var workersScoresOneOut []float64
	for _, oneOutLoss := range networkLosses.OneOutValues {
		workerScore := GetWorkerScore(networkLosses.CombinedValue, oneOutLoss.Value)
		workersScoresOneOut = append(workersScoresOneOut, workerScore)
	}

	numForecasters := len(workersScoresOneOut)
	fUniqueAgg := GetfUniqueAgg(float64(numForecasters))
	var newScores []types.Score
	for i, oneInNaiveLoss := range networkLosses.OneInNaiveValues {
		workerAddr, err := sdk.AccAddressFromBech32(oneInNaiveLoss.Worker)
		if err != nil {
			return []types.Score{}, err
		}

		// Get worker score for one in loss
		workerScoreOneIn := GetWorkerScore(networkLosses.NaiveValue, oneInNaiveLoss.Value)

		// Calculate forecast score
		workerFinalScore := GetFinalWorkerScoreForecastTask(workerScoreOneIn, workersScoresOneOut[i], fUniqueAgg)

		newScore := types.Score{
			TopicId:     topicId,
			BlockNumber: block,
			Address:     workerAddr.String(),
			Score:       workerFinalScore,
		}
		err = keeper.InsertWorkerForecastScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, err
		}
	}

	return newScores, nil
}
