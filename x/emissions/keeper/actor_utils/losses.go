package actor_utils

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// REPUTER NONCES CLOSING

// Closes an open reputer nonce.
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

	/// Do filters upon the leader (the sender) first, then do checks on each reputer in the payload
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
	// Check if the window time has passed: if blockheight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	blockHeight := ctx.BlockHeight()
	if blockHeight < nonce.BlockHeight+topic.GroundTruthLag {
		return types.ErrReputerNonceWindowNotAvailable
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	reputerLossBundles, err := k.GetReputerLossBundlesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return types.ErrNoValidBundles
	}

	/// Do checks on each reputer in the payload
	// Iterate through the array to ensure each reputer is in the whitelist
	// and get get score for each reputer => later we can skim only the top few by score descending
	lossBundlesByReputer := make([]*types.ReputerValueBundle, 0)
	stakesByReputer := make(map[string]cosmosMath.Int)
	for _, bundle := range reputerLossBundles.ReputerValueBundles {
		if err := bundle.Validate(); err != nil {
			continue
		}

		reputer := bundle.ValueBundle.Reputer

		// Check that the reputer's value bundle is for a topic matching the leader's given topic
		if bundle.ValueBundle.TopicId != topicId {
			continue
		}
		// Check that the reputer's value bundle is for a nonce matching the leader's given nonce
		if bundle.ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight != nonce.BlockHeight {
			continue
		}

		// Check that the reputer is registered in the topic
		isReputerRegistered, err := k.IsReputerRegisteredInTopic(ctx, bundle.ValueBundle.TopicId, reputer)
		if err != nil {
			continue
		}
		// We'll keep what we can get from the payload, but we'll ignore the rest
		if !isReputerRegistered {
			continue
		}

		// Check that the reputer enough stake in the topic
		stake, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
		if err != nil {
			continue
		}
		if stake.LT(params.RequiredMinimumStake) {
			continue
		}

		// Examine forecast elements to verify that they're for registered inferers in the current set.
		// A check of their registration and other filters have already been applied when their inferences were inserted.
		// We keep what we can, ignoring the reputer and their contribution (losses) entirely
		// if they're left with no valid losses.

		filteredBundle, err := filterUnacceptedWorkersFromReputerValueBundle(k, ctx, topicId, *bundle.ValueBundle.ReputerRequestNonce, bundle)
		if err != nil {
			continue
		}

		/// If we do PoX-like anti-sybil procedure, would go here

		/// Filtering done now, now write what we must for inclusion

		if err != nil {
			continue
		}
		lossBundlesByReputer = append(lossBundlesByReputer, filteredBundle)

		stake, err = k.GetStakeReputerAuthority(ctx, topicId, reputer)
		if err != nil {
			continue
		}
		stakesByReputer[bundle.ValueBundle.Reputer] = stake
	}

	// sort by reputer score descending
	sort.Slice(lossBundlesByReputer, func(i, j int) bool {
		return lossBundlesByReputer[i].ValueBundle.Reputer < lossBundlesByReputer[j].ValueBundle.Reputer
	})

	bundles := types.ReputerValueBundles{
		ReputerValueBundles: lossBundlesByReputer,
	}
	err = k.InsertReputerLossBundlesAtBlock(ctx, topicId, nonce.BlockHeight, bundles)
	if err != nil {
		return err
	}

	networkLossBundle, err := synth.CalcNetworkLosses(stakesByReputer, bundles, topic.Epsilon)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Debug(fmt.Sprintf("Reputer Nonce %d Network Loss Bundle %v", &nonce.BlockHeight, networkLossBundle))

	networkLossBundle.ReputerRequestNonce = &types.ReputerRequestNonce{
		ReputerNonce: &nonce,
	}

	err = k.InsertNetworkLossBundleAtBlock(ctx, topicId, nonce.BlockHeight, networkLossBundle)
	if err != nil {
		return err
	}

	types.EmitNewNetworkLossSetEvent(sdkCtx, topicId, nonce.BlockHeight, networkLossBundle)

	err = synth.GetCalcSetNetworkRegrets(
		sdkCtx,
		*k,
		topicId,
		networkLossBundle,
		*&nonce,
		topic.AlphaRegret,
		params.CNorm,
		topic.PNorm,
		topic.Epsilon)
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

	err = k.AddRewardableTopic(ctx, topicId)
	if err != nil {
		return err
	}

	err = k.SetTopicLastCommit(ctx, topic.Id, blockHeight, &nonce, types.ActorType_REPUTER)
	if err != nil {
		return err
	}

	err = k.SetTopicLastReputerPayload(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}
	sdkCtx.Logger().Info(fmt.Sprintf("Closed reputer nonce for topic: %d, nonce: %v", topicId, nonce))
	return nil
}

// Filter out values of unaccepted workers.
// It is assumed that the work of inferers and forecasters stored at the nonce is already filtered for acceptance.
// This also removes duplicate values of the same worker.
func filterUnacceptedWorkersFromReputerValueBundle(
	k *keeper.Keeper,
	ctx context.Context,
	topicId uint64,
	reputerRequestNonce types.ReputerRequestNonce,
	reputerValueBundle *types.ReputerValueBundle,
) (*types.ReputerValueBundle, error) {
	// Get the accepted inferers of the associated worker response payload
	inferences, err := k.GetInferencesAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no inferences found at block height")
		} else {
			return nil, err
		}
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

	return acceptedReputerValueBundle, nil
}
