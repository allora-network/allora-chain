package actorutils

import (
	"sort"

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

		lossBundlesByReputer = append(lossBundlesByReputer, &bundle)
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

	types.EmitNewNetworkLossSetEvent(ctx, topic.Id, nonce.BlockHeight, networkLossBundle)

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
