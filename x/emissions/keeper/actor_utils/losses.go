package actorutils

import (
	"fmt"
	"slices"

	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// REPUTER NONCES CLOSING

// Closes an open reputer nonce.
// calculates the network loss of the topic,
// the scores for inferers, forecasters, and reputers
// filters the inferers, forecasters, and reputers for this epoch
// based on the Exponential Moving Average (EMA) of their scores
// and then closes the reputer nonce to finish the round.
func CloseReputerNonce(
	k *keeper.Keeper,
	ctx sdk.Context,
	topicId keeper.TopicId,
	nonce types.Nonce) error {
	// Check if the topic exists
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return sdkerrors.ErrNotFound
	}

	/// All filters should be done in order of increasing computational complexity
	// Check if the worker nonce is unfulfilled
	workerNonceUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topicId, &nonce)
	if err != nil {
		return err
	}
	// Throw if worker nonce is unfulfilled -- can't report losses on something not yet committed
	if workerNonceUnfulfilled {
		return errorsmod.Wrapf(
			types.ErrNonceStillUnfulfilled,
			"Reputer's worker nonce not yet fulfilled for reputer block: %v",
			&nonce.BlockHeight,
		)
	}

	// Check if the reputer nonce is unfulfilled
	reputerNonceUnfulfilled, err := k.IsReputerNonceUnfulfilled(ctx, topicId, &nonce)
	if err != nil {
		return err
	}
	// Throw if already fulfilled -- can't return a response twice
	if !reputerNonceUnfulfilled {
		return errorsmod.Wrapf(
			types.ErrUnfulfilledNonceNotFound,
			"Reputer nonce already fulfilled: %v",
			&nonce.BlockHeight,
		)
	}

	// Check if the time for ground truth to become apparent has passed
	blockHeight := ctx.BlockHeight()
	if blockHeight < nonce.BlockHeight+topic.GroundTruthLag {
		return types.ErrReputerNonceWindowNotAvailable
	}

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	reputerLossBundles, err := k.GetReputerLossBundlesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return types.ErrNoValidBundles
	}

	// Validation of bundles should occur at time of bundle insertion
	// Get score for each reputer => later we can skim only the top few by score descending
	reputerValueBundles := reputerLossBundles.ReputerValueBundles
	stakesByReputer := make(map[string]cosmosMath.Int)
	for _, bundle := range reputerValueBundles {
		stake, err := k.GetStakeReputerAuthority(ctx, topicId, bundle.ValueBundle.Reputer)
		if err != nil {
			continue
		}
		stakesByReputer[bundle.ValueBundle.Reputer] = stake
	}

	// sort by reputer score descending
	// we don't need to worry about sort stability here because the reputer addresses are unique
	// when a reputer uploads a bundle twice, the second upload will overwrite the first
	slices.SortFunc(reputerValueBundles, func(a, b *types.ReputerValueBundle) int {
		if a.ValueBundle.Reputer < b.ValueBundle.Reputer {
			return -1
		} else if a.ValueBundle.Reputer > b.ValueBundle.Reputer {
			return 1
		}
		return 0
	})

	networkLossBundle, err := synth.CalcNetworkLosses(stakesByReputer, *reputerLossBundles)
	if err != nil {
		return err
	}

	ctx.Logger().Debug(fmt.Sprintf("Reputer Nonce %d Network Loss Bundle %v", &nonce.BlockHeight, networkLossBundle))

	// insert the network loss bundle into the data store
	networkLossBundle.ReputerRequestNonce = &types.ReputerRequestNonce{
		ReputerNonce: &nonce,
	}
	err = k.InsertNetworkLossBundleAtBlock(ctx, topicId, nonce.BlockHeight, networkLossBundle)
	if err != nil {
		return err
	}
	types.EmitNewNetworkLossSetEvent(ctx, topicId, nonce.BlockHeight, networkLossBundle)

	// now that we have network losses, we can generate scores for inferers, forecasters

	// find the inferers that will be considered active this epoch
	// and update everybody's score EMA to what it should be for their work this epoch
	acceptedInferers, err := filterActiveInferersUpdateScoreEmas(
		ctx,
		*k,
		topic,
		nonce,
		moduleParams.MeritSortitionAlpha,
		moduleParams.MaxTopInferersToReward,
		networkLossBundle,
	)
	if err != nil {
		return err
	}

	// find the forecasters that will be considered active this epoch
	// and update everybody's score EMA to what it should be for their work this epoch
	acceptedForecasters, err := filterActiveForecastersUpdateScoreEmas(
		ctx,
		*k,
		topic,
		nonce,
		moduleParams.MeritSortitionAlpha,
		moduleParams.MaxTopForecastersToReward,
		networkLossBundle,
	)
	if err != nil {
		return err
	}

	// find the reputers that will be considered active this epoch
	// and update everybody's score EMA to what it should be for their work this epoch
	acceptedReputers, err := filterActiveReputersUpdateScoreEmas(
		ctx,
		*k,
		topic,
		nonce,
		moduleParams,
		*reputerLossBundles,
	)
	if err != nil {
		return err
	}

	// now clean up the reputer value bundle from all the unaccepted workers
	acceptedReputerValueBundles := filterUnacceptedWorkersFromReputerValueBundles(
		*reputerLossBundles,
		acceptedInferers,
		acceptedForecasters,
		acceptedReputers,
	)

	// recalculate the losses now over just the minimized data
	// todo: explore how to call GetCalcSetNetworkRegrets with the new data only
	// without having to recalculate all the losses again
	// should be possible changing some function parameters or maybe adding some
	// new functions that extract out just the needed calculations
	filteredNetworkLossBundle, err := synth.CalcNetworkLosses(stakesByReputer, *acceptedReputerValueBundles)
	if err != nil {
		return err
	}

	// with the filtered loss bundle, calculate the regrets for the topic
	err = synth.GetCalcSetNetworkRegrets(
		ctx,
		*k,
		topicId,
		filteredNetworkLossBundle,
		nonce,
		topic.AlphaRegret,
		moduleParams.CNorm,
		topic.PNorm,
		topic.Epsilon)
	if err != nil {
		return err
	}

	// clean out the unaccepted inferers, forecasters, and reputers from the data store
	err = purgeUnacceptedActorsFromKeeper(
		ctx,
		*k,
		topicId,
		nonce,
		acceptedInferers,
		acceptedForecasters,
		*acceptedReputerValueBundles,
	)
	if err != nil {
		return err
	}

	// Update the unfulfilled nonces
	_, err = k.FulfillReputerNonce(ctx, topicId, &nonce)
	if err != nil {
		return err
	}

	// Update topic reward nonce
	err = k.SetTopicRewardNonce(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return err
	}

	// Update this topic as being rewardable
	err = k.AddRewardableTopic(ctx, topicId)
	if err != nil {
		return err
	}

	// Update this reputer nonce as being committed
	err = k.SetReputerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}

	ctx.Logger().Info(fmt.Sprintf("Closed reputer nonce for topic: %d, nonce: %v", topicId, nonce))
	return nil
}

// Generate the new scores for all inferers at this nonce.
// 1. Get the EMA of their scores in the past
// 2. Update the EMA of their scores
// 3. Use this new EMA value to filter which inferences, forecasts we shall accept
// 4. After we've filtered, assign/store the passive set of actors the quantile score
// 5. For the active set, assign/store their real value ema scores
func filterActiveInferersUpdateScoreEmas(
	ctx sdk.Context,
	k keeper.Keeper,
	topic types.Topic,
	nonce types.Nonce,
	meritSortitionAlpha alloraMath.Dec,
	maxTopInferersToReward uint64,
	networkLossBundle types.ValueBundle,
) (acceptedInferers map[string]struct{}, err error) {
	// get all of the inferer scores based on the network losses
	allInfererScores, err := CalcInferenceScores(
		ctx,
		k,
		topic,
		nonce.BlockHeight,
		networkLossBundle,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "CloseWorkerNonce: Error generating inference scores")
	}

	// for every inferer, get their previous EMA value,
	// and with their current score, get their new EMA value
	allInfererEmaScores := make([]types.Score, 0, len(allInfererScores))
	for _, score := range allInfererScores {
		previousEma, err := k.GetInfererScoreEma(ctx, topic.Id, score.Address)
		if err != nil {
			return nil, errorsmod.Wrap(err, "CloseWorkerNonce: Error getting inferer EMA")
		}
		// if we have no historical EMA to work off of, then just assign the new score as the EMA
		newEmaScore, err := alloraMath.CalcEma(
			meritSortitionAlpha,
			score.Score,
			previousEma.Score,
			previousEma.BlockHeight == 0 && previousEma.Score.IsZero(), // first time or not
		)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf(
				"CloseWorkerNonce: Error calculating new inferer EMA for inferer %s: %v", score.Address, err))
			continue
		}
		allInfererEmaScores = append(allInfererEmaScores, types.Score{
			Address: score.Address,
			Score:   newEmaScore,
		})
	}

	// find the top maxTopInferersToReward inferers
	_, allInfererEmaScores, acceptedInferers = FindTopNByScoreDesc(
		ctx,
		maxTopInferersToReward,
		allInfererEmaScores,
		ctx.BlockHeight(),
	)

	// find the quantile of the inferer scores, based on the topic's ActiveInfererQuantile
	quantile, err := GetQuantileOfScores(allInfererEmaScores, topic.ActiveInfererQuantile)
	if err != nil {
		return acceptedInferers, errorsmod.Wrap(err, "CloseWorkerNonce: Error getting quantile of inferer scores")
	}

	// update the inferer scores in the data store
	// inferers that are in the top N active set are assigned their real value ema scores
	// inferers in the passive set not top are assigned the quantile score
	for i, score := range allInfererEmaScores {
		newScore := score
		if _, isTopInferer := acceptedInferers[score.Address]; !isTopInferer {
			newScore = types.Score{
				TopicId:     score.TopicId,
				BlockHeight: score.BlockHeight,
				Address:     score.Address,
				Score:       quantile,
			}
			// update the slice with the new score for the emit new inferer scores event
			allInfererEmaScores[i] = newScore
		}
		err := k.SetInfererScoreEma(ctx, score.TopicId, score.Address, newScore)
		if err != nil {
			return acceptedInferers, errorsmod.Wrap(err, "CloseWorkerNonce: Error setting inferer EMA")
		}
	}
	types.EmitNewInfererScoresSetEvent(ctx, allInfererEmaScores)

	return acceptedInferers, nil
}

// Generate the new scores for all forecasters at this nonce.
// 1. Get the EMA of their scores in the past
// 2. Update the EMA of their scores
// 3. Use this new EMA value to filter which inferences, forecasts we shall accept
// 4. After we've filtered, assign/store the passive set of actors the quantile score
// 5. For the active set, assign/store their real value ema scores
func filterActiveForecastersUpdateScoreEmas(
	ctx sdk.Context,
	k keeper.Keeper,
	topic types.Topic,
	nonce types.Nonce,
	meritSortitionAlpha alloraMath.Dec,
	maxTopForecastersToReward uint64,
	networkLossBundle types.ValueBundle,
) (acceptedForecasters map[string]struct{}, err error) {
	// get all of the inferer scores based on the network losses
	allForecasterScores, err := CalcForecasterScores(
		ctx,
		k,
		topic,
		nonce.BlockHeight,
		networkLossBundle,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "CloseWorkerNonce: Error generating forecaster scores")
	}

	// for every forecaster, get their previous EMA value,
	// and with their current score, get their new EMA value
	allForecasterEmaScores := make([]types.Score, 0, len(allForecasterScores))
	for _, score := range allForecasterScores {
		previousEma, err := k.GetForecasterScoreEma(ctx, topic.Id, score.Address)
		if err != nil {
			return nil, errorsmod.Wrap(err, "CloseWorkerNonce: Error getting inferer EMA")
		}
		// if we have no historical EMA to work off of, then just assign the new score as the EMA
		newEmaScore, err := alloraMath.CalcEma(
			meritSortitionAlpha,
			score.Score,
			previousEma.Score,
			previousEma.BlockHeight == 0 && previousEma.Score.IsZero(), // first time or not
		)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf(
				"CloseWorkerNonce: Error calculating new inferer EMA for inferer %s: %v", score.Address, err))
			continue
		}
		allForecasterEmaScores = append(allForecasterEmaScores, types.Score{
			TopicId:     score.TopicId,
			BlockHeight: score.BlockHeight,
			Address:     score.Address,
			Score:       newEmaScore,
		})
	}

	// find the top maxTopInferersToReward inferers
	_, allForecasterEmaScores, acceptedForecasters = FindTopNByScoreDesc(
		ctx,
		maxTopForecastersToReward,
		allForecasterEmaScores,
		ctx.BlockHeight(),
	)

	// find the quantile of the inferer scores, based on the topic's ActiveInfererQuantile
	quantile, err := GetQuantileOfScores(allForecasterEmaScores, topic.ActiveForecasterQuantile)
	if err != nil {
		return acceptedForecasters, errorsmod.Wrap(err, "CloseWorkerNonce: Error getting quantile of inferer scores")
	}

	// update the inferer scores in the data store
	// forecasters that are in the top N active set are assigned their real value ema scores
	// forecasters in the passive set not top are assigned the quantile score
	for i, score := range allForecasterEmaScores {
		newScore := score
		if _, isTopForecaster := acceptedForecasters[score.Address]; !isTopForecaster {
			newScore = types.Score{
				TopicId:     score.TopicId,
				BlockHeight: score.BlockHeight,
				Address:     score.Address,
				Score:       quantile,
			}
			// update the slice with the new score for the emit new inferer scores event
			allForecasterEmaScores[i] = newScore
		}
		err := k.SetForecasterScoreEma(ctx, topic.Id, score.Address, newScore)
		if err != nil {
			return acceptedForecasters, errorsmod.Wrap(err, "CloseWorkerNonce: Error setting forecaster EMA")
		}
	}
	types.EmitNewForecasterScoresSetEvent(ctx, allForecasterEmaScores)

	return acceptedForecasters, nil
}

// Generate the new scores for all reputers at this nonce.
// 1. Get the EMA of their scores in the past
// 2. Update the EMA of their scores
// 3. Use this new EMA value to filter which reputation bundles we shall accept
// 4. After we've filtered, assign/store the passive set of actors the quantile score
// 5. For the active set, assign/store their real value ema scores
func filterActiveReputersUpdateScoreEmas(
	ctx sdk.Context,
	k keeper.Keeper,
	topic types.Topic,
	nonce types.Nonce,
	moduleParams types.Params,
	reputerLossBundles types.ReputerValueBundles,
) (acceptedReputers map[string]struct{}, err error) {
	// get all the reputer scores
	allReputerScores, err := CalcReputerScoresSetListeningCoefficients(
		ctx,
		k,
		topic.Id,
		moduleParams,
		nonce.BlockHeight,
		reputerLossBundles,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "CloseWorkerNonce: Error generating reputer scores")
	}

	// for every reputer, get their previous EMA value,
	// and with their current score, get their new EMA value
	allReputerEmaScores := make([]types.Score, 0, len(allReputerScores))
	for _, score := range allReputerScores {
		previousEma, err := k.GetReputerScoreEma(ctx, topic.Id, score.Address)
		if err != nil {
			return nil, errorsmod.Wrap(err, "CloseWorkerNonce: Error getting inferer EMA")
		}
		// if we have no historical EMA to work off of, then just assign the new score as the EMA
		newEmaScore, err := alloraMath.CalcEma(
			moduleParams.MeritSortitionAlpha,
			score.Score,
			previousEma.Score,
			previousEma.BlockHeight == 0 && previousEma.Score.IsZero(), // first time or not
		)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf(
				"CloseWorkerNonce: Error calculating new inferer EMA for inferer %s: %v", score.Address, err))
			continue
		}
		allReputerEmaScores = append(allReputerEmaScores, types.Score{
			TopicId:     score.TopicId,
			BlockHeight: score.BlockHeight,
			Address:     score.Address,
			Score:       newEmaScore,
		})
	}

	// find the top maxTopInferersToReward inferers
	_, allReputerEmaScores, acceptedReputers = FindTopNByScoreDesc(
		ctx,
		moduleParams.MaxTopReputersToReward,
		allReputerEmaScores,
		ctx.BlockHeight(),
	)

	// find the quantile of the inferer scores, based on the topic's ActiveInfererQuantile
	quantile, err := GetQuantileOfScores(allReputerEmaScores, topic.ActiveReputerQuantile)
	if err != nil {
		return acceptedReputers, errorsmod.Wrap(err, "CloseWorkerNonce: Error getting quantile of inferer scores")
	}

	// update the inferer scores in the data store
	// forecasters that are in the top N active set are assigned their real value ema scores
	// forecasters in the passive set not top are assigned the quantile score
	for i, score := range allReputerEmaScores {
		newScore := score
		if _, isTopReputer := acceptedReputers[score.Address]; !isTopReputer {
			newScore = types.Score{
				TopicId:     score.TopicId,
				BlockHeight: score.BlockHeight,
				Address:     score.Address,
				Score:       quantile,
			}
			// update the slice with the new score for the emit new inferer scores event
			allReputerEmaScores[i] = newScore
		}
		err := k.SetReputerScoreEma(ctx, topic.Id, score.Address, newScore)
		if err != nil {
			return acceptedReputers, errorsmod.Wrap(err, "CloseWorkerNonce: Error setting forecaster EMA")
		}
	}
	types.EmitNewReputerScoresSetEvent(ctx, allReputerEmaScores)

	return acceptedReputers, nil
}

// Filter out values of unaccepted workers.
// It is assumed that the work of inferers and forecasters stored at the nonce is already filtered for acceptance.
// This also removes duplicate values of the same worker.
func filterUnacceptedWorkersFromReputerValueBundles(
	reputerValueBundles types.ReputerValueBundles,
	acceptedInferers map[string]struct{},
	acceptedForecasters map[string]struct{},
	acceptedReputers map[string]struct{},
) (filteredLossBundles *types.ReputerValueBundles) {
	filteredLossBundles = &types.ReputerValueBundles{
		ReputerValueBundles: make([]*types.ReputerValueBundle, 0),
	}
	for _, reputerValueBundle := range reputerValueBundles.ReputerValueBundles {
		if _, exists := acceptedReputers[reputerValueBundle.ValueBundle.Reputer]; exists {
			// Filter out values submitted by unaccepted workers
			acceptedInfererValues := make([]*types.WorkerAttributedValue, 0)
			infererAlreadySeen := make(map[string]bool)
			for _, workerVal := range reputerValueBundle.ValueBundle.InfererValues {
				if _, exists := acceptedInferers[workerVal.Worker]; exists {
					if _, ok := infererAlreadySeen[workerVal.Worker]; !ok {
						acceptedInfererValues = append(acceptedInfererValues, workerVal)
						infererAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
					}
				}
			}

			acceptedForecasterValues := make([]*types.WorkerAttributedValue, 0)
			forecasterAlreadySeen := make(map[string]bool)
			for _, workerVal := range reputerValueBundle.ValueBundle.ForecasterValues {
				if _, exists := acceptedForecasters[workerVal.Worker]; exists {
					if _, ok := forecasterAlreadySeen[workerVal.Worker]; !ok {
						acceptedForecasterValues = append(acceptedForecasterValues, workerVal)
						forecasterAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
					}
				}
			}

			acceptedOneOutInfererValues := make([]*types.WithheldWorkerAttributedValue, 0)
			// If 1 or fewer inferers, there's no one-out inferer data to receive
			if len(acceptedInfererValues) > 1 {
				oneOutInfererAlreadySeen := make(map[string]bool)
				for _, workerVal := range reputerValueBundle.ValueBundle.OneOutInfererValues {
					if _, exists := acceptedInferers[workerVal.Worker]; exists {
						if _, ok := oneOutInfererAlreadySeen[workerVal.Worker]; !ok {
							acceptedOneOutInfererValues = append(acceptedOneOutInfererValues, workerVal)
							oneOutInfererAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
						}
					}
				}
			}

			acceptedOneOutForecasterValues := make([]*types.WithheldWorkerAttributedValue, 0)
			oneOutForecasterAlreadySeen := make(map[string]bool)
			for _, workerVal := range reputerValueBundle.ValueBundle.OneOutForecasterValues {
				if _, exists := acceptedForecasters[workerVal.Worker]; exists {
					if _, ok := oneOutForecasterAlreadySeen[workerVal.Worker]; !ok {
						acceptedOneOutForecasterValues = append(acceptedOneOutForecasterValues, workerVal)
						oneOutForecasterAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
					}
				}
			}

			acceptedOneInForecasterValues := make([]*types.WorkerAttributedValue, 0)
			oneInForecasterAlreadySeen := make(map[string]bool)
			for _, workerVal := range reputerValueBundle.ValueBundle.OneInForecasterValues {
				if _, exists := acceptedForecasters[workerVal.Worker]; exists {
					if _, ok := oneInForecasterAlreadySeen[workerVal.Worker]; !ok {
						acceptedOneInForecasterValues = append(acceptedOneInForecasterValues, workerVal)
						oneInForecasterAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
					}
				}
			}

			acceptedReputerValueBundle := &types.ReputerValueBundle{
				ValueBundle: &types.ValueBundle{
					TopicId:                reputerValueBundle.ValueBundle.TopicId,
					ReputerRequestNonce:    reputerValueBundle.ValueBundle.ReputerRequestNonce,
					Reputer:                reputerValueBundle.ValueBundle.Reputer,
					ExtraData:              reputerValueBundle.ValueBundle.ExtraData,
					InfererValues:          acceptedInfererValues,
					ForecasterValues:       acceptedForecasterValues,
					OneOutInfererValues:    acceptedOneOutInfererValues,
					OneOutForecasterValues: acceptedOneOutForecasterValues,
					OneInForecasterValues:  acceptedOneInForecasterValues,
					NaiveValue:             reputerValueBundle.ValueBundle.NaiveValue,
					CombinedValue:          reputerValueBundle.ValueBundle.CombinedValue,
				},
				Signature: reputerValueBundle.Signature,
			}

			filteredLossBundles.ReputerValueBundles = append(
				filteredLossBundles.ReputerValueBundles,
				acceptedReputerValueBundle,
			)
		}
	}

	return filteredLossBundles
}

// purgeUnacceptedActorsFromKeeper purges the unaccepted actors from the data store
// note that it is not a perfect purge - e.g. if you have a forecaster that forecasts
// for a inferer that is not accepted, but the forecaster is accepted,
// then the forecasts will be kept, even though the inferences were purged
// and same logic with reputers
// todo improve this function so that the purge is complete
func purgeUnacceptedActorsFromKeeper(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId uint64,
	nonce types.Nonce,
	acceptedInferers map[string]struct{},
	acceptedForecasters map[string]struct{},
	acceptedReputerValueBundles types.ReputerValueBundles,
) error {
	// get all the inferences for this nonce
	// find the ones we're keeping
	// overwrite the inferences for this block with the ones we're keeping
	allInferences, err := k.GetInferencesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return errorsmod.Wrap(err, "purgeUnacceptedActorsFromKeeper: Error getting inferences at block height")
	}
	keepInferences := make([]*types.Inference, 0, len(allInferences.Inferences))
	for _, inference := range allInferences.Inferences {
		if _, accepted := acceptedInferers[inference.Inferer]; accepted {
			keepInferences = append(keepInferences, inference)
		}
	}
	keptInferences := types.Inferences{
		Inferences: keepInferences,
	}
	err = k.SetInferences(ctx, topicId, nonce, keptInferences)
	if err != nil {
		return errorsmod.Wrap(err, "purgeUnacceptedActorsFromKeeper: Error setting inferences at block height")
	}

	// get all the forecasts for this nonce
	// find the ones we're keeping
	// overwrite the forecasts for this block with the ones we're keeping
	allForecasts, err := k.GetForecastsAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return errorsmod.Wrap(err, "purgeUnacceptedActorsFromKeeper: Error getting forecasts at block height")
	}
	keepForecasts := make([]*types.Forecast, 0, len(allForecasts.Forecasts))
	for _, forecast := range allForecasts.Forecasts {
		if _, accepted := acceptedForecasters[forecast.Forecaster]; accepted {
			keepForecasts = append(keepForecasts, forecast)
		}
	}
	keptForecasts := types.Forecasts{
		Forecasts: keepForecasts,
	}
	err = k.SetForecasts(ctx, topicId, nonce, keptForecasts)
	if err != nil {
		return errorsmod.Wrap(err, "purgeUnacceptedActorsFromKeeper: Error setting forecasts at block height")
	}

	err = k.SetReputerLossBundlesAtBlock(ctx, topicId, nonce.BlockHeight, acceptedReputerValueBundles)
	if err != nil {
		return errorsmod.Wrap(err, "purgeUnacceptedActorsFromKeeper: Error setting reputer loss bundles at block height")
	}

	return nil
}
