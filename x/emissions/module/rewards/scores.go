package rewards

import (
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
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
	reportedLosses = EnsureWorkerPresence(reportedLosses)
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
		if reputerStakeDec.IsNaN() {
			return []types.Score{}, errors.Wrap(types.ErrInvalidReputerData, "Error invalid reputer Stake: NaN")
		}
		// Get reputer listening coefficient
		res, err := keeper.GetListeningCoefficient(ctx, topicId, reportedLoss.ValueBundle.Reputer)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting GetListeningCoefficient")
		}
		if res.Coefficient.IsNaN() {
			return []types.Score{}, errors.Wrap(types.ErrInvalidReputerData, "Error invalid reputer Stake: NaN")
		}

		// defer addition until all errors are checked to ensure no partial data is added
		reputerStakes = append(reputerStakes, reputerStakeDec)
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
	var emaScores []types.Score
	activeArr := make(map[string]bool)
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

		instantScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     reputer,
			Score:       scores[i],
		}
		err = keeper.InsertReputerScore(ctx, topicId, block, instantScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting reputer score")
		}

		emaScore, err := keeper.CalcAndSaveReputerScoreEmaForActiveSet(ctx, topic, block, reputer, instantScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error calculating and saving reputer score ema")
		}

		activeArr[reputer] = true
		newScores = append(newScores, instantScore)
		emaScores = append(emaScores, emaScore)
	}

	// Update topic quantile of instant score
	topicInstantScoreQuantile, err := actorutils.GetQuantileOfScores(newScores, topic.ActiveReputerQuantile)
	if err != nil {
		return nil, err
	}
	err = keeper.SetPreviousTopicQuantileReputerScoreEma(ctx, topicId, topicInstantScoreQuantile)
	if err != nil {
		return nil, err
	}

	types.EmitNewReputerScoresSetEvent(ctx, newScores)
	types.EmitNewActorEMAScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_REPUTER, emaScores, activeArr)
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
	var emaScores []types.Score
	activeArr := make(map[string]bool)
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
		types.EmitNewInfererScoresSetEvent(ctx, newScores)
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

		instantScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     oneOutLoss.Worker,
			Score:       workerNewScore,
		}
		err = keeper.InsertWorkerInferenceScore(ctx, topicId, block, instantScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker inference score")
		}

		emaScore, err := keeper.CalcAndSaveInfererScoreEmaForActiveSet(ctx, topic, block, oneOutLoss.Worker, instantScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error calculating and saving inferer score ema")
		}
		activeArr[oneOutLoss.Worker] = true
		newScores = append(newScores, instantScore)
		emaScores = append(emaScores, emaScore)
	}

	// Update topic quantile of instant score
	topicInstantScoreQuantile, err := actorutils.GetQuantileOfScores(newScores, topic.ActiveInfererQuantile)
	if err != nil {
		return nil, errors.Wrapf(err, "Error getting quantile of scores")
	}
	err = keeper.SetPreviousTopicQuantileInfererScoreEma(ctx, topicId, topicInstantScoreQuantile)
	if err != nil {
		return nil, errors.Wrapf(err, "Error setting previous topic quantile inferer score ema")
	}

	types.EmitNewInfererScoresSetEvent(ctx, newScores)
	types.EmitNewActorEMAScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, emaScores, activeArr)
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
	var emaScores []types.Score
	activeArr := make(map[string]bool)
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
			Address:     networkLosses.ForecasterValues[0].Worker,
			Score:       alloraMath.ZeroDec(),
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, block, newScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker inference score")
		}
		newScores = append(newScores, newScore)
		types.EmitNewForecasterScoresSetEvent(ctx, newScores)
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

		// Calculate worker performance score
		workerPerformanceScore, err := GetFinalWorkerPerformanceScore(workerScoreOneIn, workersScoresOneOut[i], fUniqueAgg)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error getting final worker score forecast task")
		}

		instantScore := types.Score{
			TopicId:     topicId,
			BlockHeight: block,
			Address:     oneInNaiveLoss.Worker,
			Score:       workerPerformanceScore,
		}
		err = keeper.InsertWorkerForecastScore(ctx, topicId, block, instantScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error inserting worker forecast score")
		}

		emaScore, err := keeper.CalcAndSaveForecasterScoreEmaForActiveSet(ctx, topic, block, oneInNaiveLoss.Worker, instantScore)
		if err != nil {
			return []types.Score{}, errors.Wrapf(err, "Error calculating and saving forecaster score ema")
		}

		activeArr[oneInNaiveLoss.Worker] = true
		newScores = append(newScores, instantScore)
		emaScores = append(emaScores, emaScore)
	}

	// Update topic quantile of instant score
	topicInstantScoreQuantile, err := actorutils.GetQuantileOfScores(newScores, topic.ActiveForecasterQuantile)
	if err != nil {
		return nil, err
	}
	err = keeper.SetPreviousTopicQuantileForecasterScoreEma(ctx, topicId, topicInstantScoreQuantile)
	if err != nil {
		return nil, err
	}

	// Emit forecaster performance scores
	types.EmitNewForecasterScoresSetEvent(ctx, newScores)
	types.EmitNewActorEMAScoresSetEvent(ctx, types.ActorType_ACTOR_TYPE_FORECASTER, emaScores, activeArr)
	return newScores, nil
}

// Check if all workers are present in the reported losses and add NaN values for missing workers
// Returns the reported losses adding NaN values for missing workers in uncompleted reported losses
func EnsureWorkerPresence(reportedLosses types.ReputerValueBundles) types.ReputerValueBundles {
	// Consolidate all unique worker addresses and forecaster addresses
	allWorkersInferer := make(map[string]struct{})
	allWorkersForecaster := make(map[string]struct{})
	allWorkersOneOutInferer := make(map[string]struct{})
	allWorkersOneOutForecaster := make(map[string]struct{})
	allWorkersOneInForecaster := make(map[string]struct{})
	allForecastersOneOutInferer := make(map[string]map[string]struct{})

	for _, bundle := range reportedLosses.ReputerValueBundles {
		// Collect unique workers for each type
		for _, workerValue := range bundle.ValueBundle.InfererValues {
			allWorkersInferer[workerValue.Worker] = struct{}{}
		}
		for _, workerValue := range bundle.ValueBundle.ForecasterValues {
			allWorkersForecaster[workerValue.Worker] = struct{}{}
		}
		for _, workerValue := range bundle.ValueBundle.OneOutInfererValues {
			allWorkersOneOutInferer[workerValue.Worker] = struct{}{}
		}
		for _, workerValue := range bundle.ValueBundle.OneOutForecasterValues {
			allWorkersOneOutForecaster[workerValue.Worker] = struct{}{}
		}
		for _, workerValue := range bundle.ValueBundle.OneInForecasterValues {
			allWorkersOneInForecaster[workerValue.Worker] = struct{}{}
		}
		// Collect unique forecasters and their workers
		for _, forecasterValue := range bundle.ValueBundle.OneOutInfererForecasterValues {
			// Ensure a map exists for this forecaster
			if _, exists := allForecastersOneOutInferer[forecasterValue.Forecaster]; !exists {
				allForecastersOneOutInferer[forecasterValue.Forecaster] = make(map[string]struct{})
			}
			// Collect all workers associated with this forecaster
			for _, workerValue := range forecasterValue.OneOutInfererValues {
				allForecastersOneOutInferer[forecasterValue.Forecaster][workerValue.Worker] = struct{}{}
			}
		}
	}

	// Ensure each set has all workers, add NaN value for missing workers
	for _, bundle := range reportedLosses.ReputerValueBundles {
		bundle.ValueBundle.InfererValues = EnsureAllWorkersPresent(bundle.ValueBundle.InfererValues, allWorkersInferer)
		bundle.ValueBundle.ForecasterValues = EnsureAllWorkersPresent(bundle.ValueBundle.ForecasterValues, allWorkersForecaster)
		bundle.ValueBundle.OneOutInfererValues = EnsureAllWorkersPresentWithheld(bundle.ValueBundle.OneOutInfererValues, allWorkersOneOutInferer)
		bundle.ValueBundle.OneOutForecasterValues = EnsureAllWorkersPresentWithheld(bundle.ValueBundle.OneOutForecasterValues, allWorkersOneOutForecaster)
		bundle.ValueBundle.OneInForecasterValues = EnsureAllWorkersPresent(bundle.ValueBundle.OneInForecasterValues, allWorkersOneInForecaster)

		// Ensure all forecasters and their associated workers are present
		for forecaster, workers := range allForecastersOneOutInferer {
			found := false
			for _, forecasterValue := range bundle.ValueBundle.OneOutInfererForecasterValues {
				if forecasterValue.Forecaster == forecaster {
					forecasterValue.OneOutInfererValues = EnsureAllWorkersPresentWithheld(forecasterValue.OneOutInfererValues, workers)
					found = true
					break
				}
			}
			// If forecaster is missing, add it with NaN values for all workers
			if !found {
				newForecasterValue := types.OneOutInfererForecasterValues{
					Forecaster:          forecaster,
					OneOutInfererValues: createNaNWithheldValues(workers),
				}
				bundle.ValueBundle.OneOutInfererForecasterValues = append(bundle.ValueBundle.OneOutInfererForecasterValues, &newForecasterValue)
			}
		}
	}

	return reportedLosses
}

// Helper function to create NaN values for missing workers
func createNaNWithheldValues(workers map[string]struct{}) []*types.WithheldWorkerAttributedValue {
	var values []*types.WithheldWorkerAttributedValue
	for worker := range workers {
		values = append(values, &types.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  alloraMath.NewNaN(),
		})
	}
	return values
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
