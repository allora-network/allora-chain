package actorutils

import (
	"context"
	"errors"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// REPUTER NONCES CLOSING

// Closes an open reputer nonce.
func CloseReputerNonce(
	k *keeper.Keeper,
	ctx sdk.Context,
	topic types.Topic,
	nonce types.Nonce,
) (err error) {
	blockHeight := ctx.BlockHeight()

	// All filters should be done in order of increasing computational complexity
	// Check if the worker nonce is unfulfilled
	workerNonceUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topic.Id, &nonce)
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
	reputerNonceUnfulfilled, err := k.IsReputerNonceUnfulfilled(ctx, topic.Id, &nonce)
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
	// Check if the window time has passed
	if blockHeight < nonce.BlockHeight+topic.GroundTruthLag {
		return types.ErrReputerNonceWindowNotAvailable
	}

	defer func() {
		if err != nil {
			ctx.Logger().Error(
				"Error occurred before finalization in CloseReputerNonce, attempting cleanup anyway",
				"topicId", topic.Id,
				"nonce", nonce,
				"error", err,
			)
		}

		_, fulfillErr := k.FulfillReputerNonce(ctx, topic.Id, &nonce)
		if fulfillErr != nil {
			ctx.Logger().Error(
				"Error fulfilling reputer nonce during deferred cleanup",
				"topicId", topic.Id,
				"nonce", nonce,
				"error", fulfillErr,
			)
		}

		resetActiveErr := k.ResetActiveReputersForTopic(ctx, topic.Id)
		if resetActiveErr != nil {
			ctx.Logger().Error(
				"Error resetting active reputers during deferred cleanup",
				"topicId", topic.Id,
				"error", resetActiveErr,
			)
		}

		resetSubmissionsErr := k.ResetReputersIndividualSubmissionsForTopic(ctx, topic.Id)
		if resetSubmissionsErr != nil {
			ctx.Logger().Error(
				"Error resetting reputer individual submissions during deferred cleanup",
				"topicId", topic.Id,
				"error", resetSubmissionsErr,
			)
		}

		ctx.Logger().Info("Closed reputer nonce", "topicId", topic.Id, "nonce", nonce)
	}()

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Get active reputers for the topic
	activeReputerAddresses, err := k.GetActiveReputersForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}

	lossBundlesByReputer := make([]*types.ReputerValueBundle, 0)
	stakesByReputer := make(map[string]cosmosMath.Int)
	for _, address := range activeReputerAddresses {
		bundle, err := k.GetReputerLatestLossByTopicId(ctx, topic.Id, address)
		if err != nil {
			ctx.Logger().Warn("Could not get latest loss bundle for reputer, skipping", "reputer", address, "topicId", topic.Id, "error", err)
			continue // Skip this reputer
		}

		// Check that the reputer enough stake in the topic
		stake, err := k.GetStakeReputerAuthority(ctx, topic.Id, bundle.ValueBundle.Reputer)
		if err != nil {
			ctx.Logger().Warn("Could not get stake for reputer, skipping", "reputer", bundle.ValueBundle.Reputer, "topicId", topic.Id, "error", err)
			continue // Skip this reputer
		}
		if stake.LT(params.RequiredMinimumStake) {
			ctx.Logger().Debug("Reputer stake below minimum, skipping", "reputer", bundle.ValueBundle.Reputer, "topicId", topic.Id, "stake", stake)
			continue // Skip this reputer
		}

		// Examine forecast elements to verify that they're for registered inferers in the current set.
		// A check of their registration and other filters have already been applied when their inferences were inserted.
		// We keep what we can, ignoring the reputer and their contribution (losses) entirely
		// if they're left with no valid losses.
		filteredBundle, err := FilterUnacceptedWorkersFromReputerValueBundle(k, ctx, topic.Id, *bundle.ValueBundle.ReputerRequestNonce, &bundle)
		if err != nil {
			ctx.Logger().Warn("Could not filter bundle for reputer, skipping", "reputer", bundle.ValueBundle.Reputer, "topicId", topic.Id, "error", err)
			continue // Skip this reputer
		}

		// / Filtering done now, now write what we must for inclusion
		lossBundlesByReputer = append(lossBundlesByReputer, filteredBundle)
		stakesByReputer[bundle.ValueBundle.Reputer] = stake
	}

	if len(lossBundlesByReputer) == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrNotFound, "no valid losses found for reputers")
	}

	// sort by reputer score descending
	sort.Slice(lossBundlesByReputer, func(i, j int) bool {
		return lossBundlesByReputer[i].ValueBundle.Reputer < lossBundlesByReputer[j].ValueBundle.Reputer
	})

	bundles := types.ReputerValueBundles{
		ReputerValueBundles: lossBundlesByReputer,
	}
	err = k.InsertActiveReputerLosses(ctx, topic.Id, nonce.BlockHeight, bundles)
	if err != nil {
		return err
	}

	// Check that all network bundles correspond to the nonce requested before calling CalcNetworkLosses.
	// In case of a mismatch, we should remove that

	networkLossBundle, err := synth.CalcNetworkLosses(ctx, topic.Id, nonce.BlockHeight, stakesByReputer, bundles)
	if err != nil {
		return err
	}

	ctx.Logger().Debug("Reputer Nonce", "blockHeight", &nonce.BlockHeight, "networkLossBundle", networkLossBundle)

	err = k.InsertNetworkLossBundleAtBlock(ctx, topic.Id, nonce.BlockHeight, networkLossBundle)
	if err != nil {
		return err
	}

	types.EmitNewNetworkLossSetEvent(ctx, networkLossBundle)

	regrets, err := synth.GetCalcSetNetworkRegrets(
		synth.GetCalcSetNetworkRegretsArgs{
			Ctx:                   ctx,
			K:                     *k,
			TopicId:               topic.Id,
			NetworkLosses:         networkLossBundle,
			Nonce:                 nonce,
			AlphaRegret:           topic.AlphaRegret,
			CNorm:                 params.CNorm,
			PNorm:                 topic.PNorm,
			EpsilonTopic:          topic.Epsilon,
			InitialRegretQuantile: params.InitialRegretQuantile,
			PNormSafeDiv:          params.PNormSafeDiv,
		})
	if err != nil {
		return err
	}

	// Calculate the regret_stdnorm and the weights (multistep process).
	// 0. Get inferer and forecaster regrets
	infererRegrets := regrets.InfererRegrets
	inferers := alloraMath.GetSortedKeys(infererRegrets)
	forecasterRegrets := regrets.ForecasterRegrets
	forecasters := alloraMath.GetSortedKeys(forecasterRegrets)

	// 2. Calculate the regret_stdnorm to be used in
	// 2.a Calculate the regret_stdnorm filtered by ∫the previous weights. If not, apply stddev.
	stdDevPlusEpsilon, err := synth.CalcRegretStdDevFilteredByWeights(
		synth.CalcRegretStdDevFilteredByWeightsArgs{
			Ctx:                 ctx,
			K:                   k,
			Logger:              ctx.Logger(),
			TopicId:             topic.Id,
			Inferers:            inferers,
			Forecasters:         forecasters,
			InfererToRegret:     infererRegrets,
			ForecasterToRegret:  forecasterRegrets,
			NegligibleThreshold: params.MinWeightThresholdForStdnorm,
			EpsilonTopic:        topic.Epsilon,
		},
	)
	if err != nil {
		return err
	}
	// 2.b ... and store it.
	err = k.SetLatestRegretStdNorm(ctx, topic.Id, stdDevPlusEpsilon)
	if err != nil {
		return err
	}

	// 3. Calculate the new weights
	newWeights, err := synth.CalcWeightsGivenWorkers(
		synth.CalcWeightsGivenWorkersArgs{
			Logger:             ctx.Logger(),
			Inferers:           inferers,
			Forecasters:        forecasters,
			InfererToRegret:    infererRegrets,
			ForecasterToRegret: forecasterRegrets,
			EpsilonTopic:       topic.Epsilon,
			PNorm:              topic.PNorm,
			CNorm:              params.CNorm,
			StdDevPlusEpsilon:  stdDevPlusEpsilon,
		},
	)
	if err != nil {
		return err
	}

	// 4. Normalize weights! This was not done before, but it is needed for the filter of non-negligible weights.
	err = newWeights.NormalizeWeights()
	if err != nil {
		return err
	}

	// 5. Store the new weights
	err = synth.StoreLatestNormalizedWeights(ctx, *k, topic.Id, newWeights)
	if err != nil {
		return err
	}

	// Emit events: the regret stdnorm set event
	types.EmitNewRegretStdNormSetEvent(ctx, topic.Id, nonce.BlockHeight, stdDevPlusEpsilon)
	for _, inferer := range inferers {
		types.EmitNewInfererWeightSetEvent(ctx, topic.Id, nonce.BlockHeight, inferer, newWeights.Inferers[inferer])
	}
	for _, forecaster := range forecasters {
		types.EmitNewForecasterWeightSetEvent(ctx, topic.Id, nonce.BlockHeight, forecaster, newWeights.Forecasters[forecaster])
	}
	// -- end of regrets_stdnorm and weights multistep process

	err = k.SetTopicRewardNonce(ctx, topic.Id, nonce.BlockHeight)
	if err != nil {
		ctx.Logger().Error(
			"Error setting topic reward nonce during deferred cleanup",
			"topicId", topic.Id,
			"nonceBlockHeight", nonce.BlockHeight,
			"error", err,
		)
		return err
	}

	err = k.SetReputerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		ctx.Logger().Error(
			"Error setting reputer topic last commit during deferred cleanup",
			"topicId", topic.Id,
			"blockHeight", blockHeight,
			"nonce", nonce,
			"error", err,
		)
		return err
	}
	types.EmitNewReputerLastCommitSetEvent(ctx, topic.Id, blockHeight, &nonce)

	return nil
}

// Filter out values of unaccepted workers.
// It is assumed that the work of inferers and forecasters stored at the nonce is already filtered for acceptance.
// This also removes duplicate values of the same worker.
func FilterUnacceptedWorkersFromReputerValueBundle(
	k *keeper.Keeper,
	ctx context.Context,
	topicId uint64,
	reputerRequestNonce types.ReputerRequestNonce,
	reputerValueBundle *types.ReputerValueBundle,
) (*types.ReputerValueBundle, error) {
	// Get the accepted inferers of the associated worker response payload
	inferences, err := k.GetInferencesAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight, false)
	if errors.Is(err, collections.ErrNotFound) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrNotFound, "no inferences found at block height %d for topic %d", reputerRequestNonce.ReputerNonce.BlockHeight, topicId)
	} else if err != nil {
		return nil, err
	}

	acceptedInferersOfBatch := make(map[string]bool)
	for _, inference := range inferences.Inferences {
		acceptedInferersOfBatch[inference.Inferer] = true
	}

	// Get the accepted forecasters of the associated worker response payload
	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	acceptedForecastersOfBatch := make(map[string]bool)
	for _, forecast := range forecasts.Forecasts {
		acceptedForecastersOfBatch[forecast.Forecaster] = true
	}

	// Filter out values submitted by unaccepted workers
	acceptedInfererValues := make([]*types.WorkerAttributedValue, 0)
	infererAlreadySeen := make(map[string]bool)
	for _, workerVal := range reputerValueBundle.ValueBundle.InfererValues {
		if _, ok := acceptedInferersOfBatch[workerVal.Worker]; ok {
			if _, ok := infererAlreadySeen[workerVal.Worker]; !ok {
				acceptedInfererValues = append(acceptedInfererValues, workerVal)
				infererAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
			}
		}
	}

	acceptedForecasterValues := make([]*types.WorkerAttributedValue, 0)
	forecasterAlreadySeen := make(map[string]bool)
	for _, workerVal := range reputerValueBundle.ValueBundle.ForecasterValues {
		if _, ok := acceptedForecastersOfBatch[workerVal.Worker]; ok {
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
			if _, ok := acceptedInferersOfBatch[workerVal.Worker]; ok {
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
		if _, ok := acceptedForecastersOfBatch[workerVal.Worker]; ok {
			if _, ok := oneOutForecasterAlreadySeen[workerVal.Worker]; !ok {
				acceptedOneOutForecasterValues = append(acceptedOneOutForecasterValues, workerVal)
				oneOutForecasterAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
			}
		}
	}

	acceptedOneOutInfererForecasterValues := make([]*types.OneOutInfererForecasterValues, 0)
	for _, forecasterVal := range reputerValueBundle.ValueBundle.OneOutInfererForecasterValues {
		if _, ok := acceptedForecastersOfBatch[forecasterVal.Forecaster]; ok {
			// Filter out unaccepted workers for this forecaster
			acceptedWorkers := make([]*types.WithheldWorkerAttributedValue, 0)
			workerAlreadySeen := make(map[string]bool)
			for _, workerVal := range forecasterVal.OneOutInfererValues {
				if _, ok := acceptedInferersOfBatch[workerVal.Worker]; ok {
					if _, ok := workerAlreadySeen[workerVal.Worker]; !ok {
						acceptedWorkers = append(acceptedWorkers, workerVal)
						workerAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
					}
				}
			}
			// Only add forecaster if it has at least one accepted worker
			if len(acceptedWorkers) > 0 {
				acceptedOneOutInfererForecasterValues = append(acceptedOneOutInfererForecasterValues, &types.OneOutInfererForecasterValues{
					Forecaster:          forecasterVal.Forecaster,
					OneOutInfererValues: acceptedWorkers,
				})
			}
		}
	}

	acceptedOneInForecasterValues := make([]*types.WorkerAttributedValue, 0)
	oneInForecasterAlreadySeen := make(map[string]bool)
	for _, workerVal := range reputerValueBundle.ValueBundle.OneInForecasterValues {
		if _, ok := acceptedForecastersOfBatch[workerVal.Worker]; ok {
			if _, ok := oneInForecasterAlreadySeen[workerVal.Worker]; !ok {
				acceptedOneInForecasterValues = append(acceptedOneInForecasterValues, workerVal)
				oneInForecasterAlreadySeen[workerVal.Worker] = true // Mark as seen => no duplicates
			}
		}
	}

	// Construct the filtered bundle
	filteredValueBundle := &types.ValueBundle{
		TopicId:                       reputerValueBundle.ValueBundle.TopicId,
		ReputerRequestNonce:           reputerValueBundle.ValueBundle.ReputerRequestNonce,
		Reputer:                       reputerValueBundle.ValueBundle.Reputer,
		ExtraData:                     reputerValueBundle.ValueBundle.ExtraData,
		InfererValues:                 acceptedInfererValues,
		ForecasterValues:              acceptedForecasterValues,
		OneOutInfererValues:           acceptedOneOutInfererValues,
		OneOutForecasterValues:        acceptedOneOutForecasterValues,
		OneInForecasterValues:         acceptedOneInForecasterValues,
		OneOutInfererForecasterValues: acceptedOneOutInfererForecasterValues,
		NaiveValue:                    reputerValueBundle.ValueBundle.NaiveValue,
		CombinedValue:                 reputerValueBundle.ValueBundle.CombinedValue,
	}

	// Check if the filtering resulted in an effectively empty bundle that might be invalid downstream
	// e.g., if CalcNetworkLosses requires at least one value. Add checks if necessary.

	if len(acceptedInfererValues) == 0 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrNotFound, "no valid values found after filtering")
	}

	acceptedReputerValueBundle := &types.ReputerValueBundle{
		Pubkey:      reputerValueBundle.Pubkey,
		ValueBundle: filteredValueBundle,
		Signature:   reputerValueBundle.Signature,
	}

	return acceptedReputerValueBundle, nil
}
