package actorutils

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalcReputerScores calculates and persists scores for reputers based on their reported losses.
// this is called at time of reputer nonce closing
func CalcReputerScoresSetListeningCoefficients(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	moduleParams types.Params,
	block int64,
	reportedLosses types.ReputerValueBundles,
) (reputerScores []types.Score, err error) {
	// Ensure all workers are present in the reported losses
	// This is necessary to ensure that all workers are accounted for in the final scores
	// If a worker is missing from the reported losses, it will be added with a NaN value
	reportedLosses = ensureWorkerPresence(reportedLosses)

	// Fetch reputers data
	var reputers []string
	var reputerStakes []alloraMath.Dec
	var reputerListeningCoefficients []alloraMath.Dec
	var losses [][]alloraMath.Dec
	for _, reportedLoss := range reportedLosses.ReputerValueBundles {
		reputers = append(reputers, reportedLoss.ValueBundle.Reputer)

		// Get reputer topic stake
		reputerStake, err := keeper.GetStakeReputerAuthority(ctx, topicId, reportedLoss.ValueBundle.Reputer)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting GetStakeOnReputerInTopic")
		}
		reputerStakeDec, err := alloraMath.NewDecFromSdkInt(reputerStake)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error converting reputer stake to Dec")
		}
		reputerStakes = append(reputerStakes, reputerStakeDec)

		// Get reputer listening coefficient
		res, err := keeper.GetListeningCoefficient(ctx, topicId, reportedLoss.ValueBundle.Reputer)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting GetListeningCoefficient")
		}
		reputerListeningCoefficients = append(reputerListeningCoefficients, res.Coefficient)

		// Get all reported losses from bundle
		reputerLosses := extractValues(reportedLoss.ValueBundle)
		losses = append(losses, reputerLosses)
	}

	// Get reputer output
	scores, newCoefficients, err := GetAllReputersOutput(
		losses,
		reputerStakes,
		reputerListeningCoefficients,
		int64(len(reputerStakes)),
		moduleParams.LearningRate,
		moduleParams.GradientDescentMaxIters,
		moduleParams.EpsilonReputer,
		moduleParams.EpsilonSafeDiv,
		moduleParams.MinStakeFraction,
		moduleParams.MaxGradientThreshold,
	)
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting GetAllReputersOutput")
	}

	// Insert new coeffients and scores
	reputerScores = make([]types.Score, 0)
	for i, reputer := range reputers {
		err := keeper.SetListeningCoefficient(
			ctx,
			topicId,
			reputer,
			types.ListeningCoefficient{Coefficient: newCoefficients[i]},
		)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error setting listening coefficient")
		}

		newScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     reputer,
			Score:       scores[i],
		}
		reputerScores = append(reputerScores, newScore)
	}

	return reputerScores, nil
}

// CalcInferenceScores calculates scores for workers based on their inference task performance.
func CalcInferenceScores(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topic types.Topic,
	block int64,
	networkLosses types.ValueBundle,
) (inferenceScores []types.Score, err error) {
	inferenceScores = make([]types.Score, 0)

	// If there is only one inferer, set score to 0
	// More than one inferer is required to have one-out losses
	if len(networkLosses.InfererValues) == 1 {
		newScore := types.Score{
			TopicId:     topic.Id,
			BlockHeight: block,
			Address:     networkLosses.InfererValues[0].Worker,
			Score:       alloraMath.ZeroDec(),
		}
		inferenceScores = append(inferenceScores, newScore)
		return inferenceScores, nil
	}

	for _, oneOutLoss := range networkLosses.OneOutInfererValues {
		workerScore, err := oneOutLoss.Value.Sub(networkLosses.CombinedValue)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting worker score")
		}

		score := types.Score{
			TopicId:     topic.Id,
			BlockHeight: block,
			Address:     oneOutLoss.Worker,
			Score:       workerScore,
		}
		inferenceScores = append(inferenceScores, score)
	}

	return inferenceScores, nil
}

// CalcForecasterScores calculates scores for workers based on their forecast task performance.
func CalcForecasterScores(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topic types.Topic,
	block int64,
	networkLosses types.ValueBundle,
) (forecasterScores []types.Score, err error) {
	// If there is only one forecaster, set score to 0
	// More than one forecaster is required to have one-out losses
	if len(networkLosses.ForecasterValues) == 1 {
		newScore := types.Score{
			TopicId:     topic.Id,
			BlockHeight: block,
			Address:     networkLosses.ForecasterValues[0].Worker,
			Score:       alloraMath.ZeroDec(),
		}
		forecasterScores = append(forecasterScores, newScore)
		return forecasterScores, nil
	}

	// Get worker scores for one out loss
	var workersScoresOneOut []alloraMath.Dec
	for _, oneOutLoss := range networkLosses.OneOutForecasterValues {
		workerScore, err := oneOutLoss.Value.Sub(networkLosses.CombinedValue)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting worker score")
		}

		workersScoresOneOut = append(workersScoresOneOut, workerScore)
	}

	numForecasters := int64(len(workersScoresOneOut))
	fUniqueAgg, err := GetfUniqueAgg(alloraMath.NewDecFromInt64(numForecasters))
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting fUniqueAgg")
	}

	for i, oneInNaiveLoss := range networkLosses.OneInForecasterValues {
		// Get worker score for one in loss
		workerScoreOneIn, err := networkLosses.NaiveValue.Sub(oneInNaiveLoss.Value)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting worker score")
		}

		// Calculate forecast score
		workerFinalScore, err := GetFinalWorkerScoreForecastTask(workerScoreOneIn, workersScoresOneOut[i], fUniqueAgg)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting final worker score forecast task")
		}

		newScore := types.Score{
			TopicId:     topic.Id,
			BlockHeight: block,
			Address:     oneInNaiveLoss.Worker,
			Score:       workerFinalScore,
		}
		forecasterScores = append(forecasterScores, newScore)
	}

	return forecasterScores, nil
}

// Check if all workers are present in the reported losses and add NaN values for missing workers
// Returns the reported losses adding NaN values for missing workers in uncompleted reported losses
func ensureWorkerPresence(reportedLosses types.ReputerValueBundles) types.ReputerValueBundles {
	// Consolidate all unique worker addresses from the three slices
	allWorkersOneOutInferer := make(map[string]struct{})
	allWorkersOneOutForecaster := make(map[string]struct{})
	allWorkersOneInForecaster := make(map[string]struct{})

	for _, bundle := range reportedLosses.ReputerValueBundles {
		for _, workerValue := range bundle.ValueBundle.OneOutInfererValues {
			allWorkersOneOutInferer[workerValue.Worker] = struct{}{}
		}
		for _, workerValue := range bundle.ValueBundle.OneOutForecasterValues {
			allWorkersOneOutForecaster[workerValue.Worker] = struct{}{}
		}
		for _, workerValue := range bundle.ValueBundle.OneInForecasterValues {
			allWorkersOneInForecaster[workerValue.Worker] = struct{}{}
		}
	}

	// Ensure each set has all workers, add NaN value for missing workers
	for _, bundle := range reportedLosses.ReputerValueBundles {
		bundle.ValueBundle.OneOutInfererValues = EnsureAllWorkersPresentWithheld(bundle.ValueBundle.OneOutInfererValues, allWorkersOneOutInferer)
		bundle.ValueBundle.OneOutForecasterValues = EnsureAllWorkersPresentWithheld(bundle.ValueBundle.OneOutForecasterValues, allWorkersOneOutForecaster)
		bundle.ValueBundle.OneInForecasterValues = EnsureAllWorkersPresent(bundle.ValueBundle.OneInForecasterValues, allWorkersOneInForecaster)
	}

	return reportedLosses
}

// ensureAllWorkersPresent checks and adds missing
// workers with NaN values for a given slice of WorkerAttributedValue
func EnsureAllWorkersPresent(
	values []*types.WorkerAttributedValue,
	allWorkers map[string]struct{},
) []*types.WorkerAttributedValue {
	foundWorkers := make(map[string]bool)
	for _, value := range values {
		foundWorkers[value.Worker] = true
	}

	// Need to sort here and not in encapsulating scope because of edge cases e.g. if 1 forecaster => there's 1-in but not 1-out
	sortedWorkers := alloraMath.GetSortedKeys(allWorkers)

	for _, worker := range sortedWorkers {
		if !foundWorkers[worker] {
			values = append(values, &types.WorkerAttributedValue{
				Worker: worker,
				Value:  alloraMath.NewNaN(),
			})
		}
	}

	return values
}
