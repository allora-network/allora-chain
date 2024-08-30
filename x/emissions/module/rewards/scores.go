package rewards

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
 These functions will be used immediately after the network loss for the relevant time step has been generated.
 Using the network loss and the sets of losses reported by each reputer, the scores are calculated. In the case
 of workers (who perform the forecast task and network task), the last 10 previous scores will also be taken into
 consideration to generate the score at the most recent time step.
*/

// GenerateReputerScores calculates and persists scores for reputers based on their reported losses.
func GenerateReputerScores(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	reportedLosses types.ReputerValueBundles,
) ([]types.Score, error) {
	// Ensure all workers are present in the reported losses
	// This is necessary to ensure that all workers are accounted for in the final scores
	// If a worker is missing from the reported losses, it will be added with a NaN value
	reportedLosses = ensureWorkerPresence(reportedLosses)
	topic, err := keeper.GetTopic(ctx, topicId)
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting topic")
	}

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
		reputerLosses := ExtractValues(reportedLoss.ValueBundle)
		losses = append(losses, reputerLosses)
	}

	params, err := keeper.GetParams(ctx)
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting GetParams")
	}

	// Get reputer output
	scores, newCoefficients, err := GetAllReputersOutput(
		losses,
		reputerStakes,
		reputerListeningCoefficients,
		int64(len(reputerStakes)),
		params.LearningRate,
		params.GradientDescentMaxIters,
		params.EpsilonReputer,
		params.EpsilonSafeDiv,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting GetAllReputersOutput")
	}

	// Insert new coeffients and scores
	var newScores []types.Score
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
		err = keeper.InsertReputerScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting reputer score")
		}
		previousScore, err := keeper.GetReputerScoreEma(ctx, topicId, reputer)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting reputer score ema")
		}
		firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
		emaScoreDec, err := alloraMath.CalcEma(
			topic.MeritSortitionAlpha,
			newScore.Score,
			previousScore.Score,
			firstTime,
		)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error calculating ema")
		}
		emaScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     reputer,
			Score:       emaScoreDec,
		}
		err = keeper.SetReputerScoreEma(ctx, topicId, reputer, emaScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error setting latest reputer score")
		}
		newScores = append(newScores, newScore)
	}

	types.EmitNewReputerScoresSetEvent(ctx, newScores)
	return newScores, nil
}

// GenerateInferenceScores calculates and persists scores for workers based on their inference task performance.
func GenerateInferenceScores(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	networkLosses types.ValueBundle,
) ([]types.Score, error) {
	var newScores []types.Score

	// If there is only one inferer, set score to 0
	// More than one inferer is required to have one-out losses
	if len(networkLosses.InfererValues) == 1 {
		newScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     networkLosses.InfererValues[0].Worker,
			Score:       alloraMath.ZeroDec(),
		}
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker inference score")
		}
		newScores = append(newScores, newScore)
		return newScores, nil
	}
	topic, err := keeper.GetTopic(ctx, topicId)
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting topic")
	}

	for _, oneOutLoss := range networkLosses.OneOutInfererValues {
		workerNewScore, err := oneOutLoss.Value.Sub(networkLosses.CombinedValue)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting worker score")
		}

		newScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     oneOutLoss.Worker,
			Score:       workerNewScore,
		}
		err = keeper.InsertWorkerInferenceScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker inference score")
		}
		previousScore, err := keeper.GetInfererScoreEma(ctx, topicId, oneOutLoss.Worker)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting inferer score ema")
		}
		firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
		emaScoreDec, err := alloraMath.CalcEma(
			topic.MeritSortitionAlpha,
			newScore.Score,
			previousScore.Score,
			firstTime,
		)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error calculating ema")
		}
		emaScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     oneOutLoss.Worker,
			Score:       emaScoreDec,
		}
		err = keeper.SetInfererScoreEma(ctx, topicId, oneOutLoss.Worker, emaScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "error setting latest inferer score")
		}
		newScores = append(newScores, newScore)
	}

	types.EmitNewInfererScoresSetEvent(ctx, newScores)
	return newScores, nil
}

// GenerateForecastScores calculates and persists scores for workers based on their forecast task performance.
func GenerateForecastScores(
	ctx sdk.Context,
	keeper keeper.Keeper,
	topicId uint64,
	block int64,
	networkLosses types.ValueBundle,
) ([]types.Score, error) {
	var newScores []types.Score
	topic, err := keeper.GetTopic(ctx, topicId)
	if err != nil {
		return []types.Score{}, errors.Wrapf(err, "Error getting topic")
	}

	// If there is only one forecaster, set score to 0
	// More than one forecaster is required to have one-out losses
	if len(networkLosses.ForecasterValues) == 1 {
		newScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     networkLosses.InfererValues[0].Worker,
			Score:       alloraMath.ZeroDec(),
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker inference score")
		}
		newScores = append(newScores, newScore)
		return newScores, nil
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
			TopicId:     topicId,
			BlockHeight: block,
			Address:     oneInNaiveLoss.Worker,
			Score:       workerFinalScore,
		}
		err = keeper.InsertWorkerForecastScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker forecast score")
		}
		previousScore, err := keeper.GetForecasterScoreEma(ctx, topicId, oneInNaiveLoss.Worker)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting inferer score ema")
		}
		firstTime := previousScore.BlockHeight == 0 && previousScore.Score.IsZero()
		emaScoreDec, err := alloraMath.CalcEma(
			topic.MeritSortitionAlpha,
			newScore.Score,
			previousScore.Score,
			firstTime,
		)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error calculating ema")
		}
		emaScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     oneInNaiveLoss.Worker,
			Score:       emaScoreDec,
		}
		err = keeper.SetForecasterScoreEma(ctx, topicId, oneInNaiveLoss.Worker, emaScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error setting latest forecaster score")
		}
		newScores = append(newScores, newScore)
	}

	types.EmitNewForecasterScoresSetEvent(ctx, newScores)
	return newScores, nil
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

// ensureAllWorkersPresentWithheld checks and adds missing
// workers with NaN values for a given slice of WithheldWorkerAttributedValue
func EnsureAllWorkersPresentWithheld(
	values []*types.WithheldWorkerAttributedValue,
	allWorkers map[string]struct{},
) []*types.WithheldWorkerAttributedValue {
	foundWorkers := make(map[string]bool)
	for _, value := range values {
		foundWorkers[value.Worker] = true
	}

	// Need to sort here and not in encapsulating scope because of edge cases e.g. if 1 forecaster => there's 1-in but not 1-out
	sortedWorkers := alloraMath.GetSortedKeys(allWorkers)

	for _, worker := range sortedWorkers {
		if !foundWorkers[worker] {
			values = append(values, &types.WithheldWorkerAttributedValue{
				Worker: worker,
				Value:  alloraMath.NewNaN(),
			})
		}
	}

	return values
}
