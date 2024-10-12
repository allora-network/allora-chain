package actorutils

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
	topic types.Topic,
	nonce types.Nonce) error {
	/// All filters should be done in order of increasing computational complexity
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
	blockHeight := ctx.BlockHeight()
	if blockHeight < nonce.BlockHeight+topic.GroundTruthLag {
		return types.ErrReputerNonceWindowNotAvailable
	}

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
			return types.ErrNoValidBundles
		}

		// Check that the reputer enough stake in the topic
		stake, err := k.GetStakeReputerAuthority(ctx, topic.Id, bundle.ValueBundle.Reputer)
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
		filteredBundle, err := FilterUnacceptedWorkersFromReputerValueBundle(k, ctx, topic.Id, *bundle.ValueBundle.ReputerRequestNonce, &bundle)
		if err != nil {
			continue
		}

		/// Filtering done now, now write what we must for inclusion
		lossBundlesByReputer = append(lossBundlesByReputer, filteredBundle)
		stakesByReputer[bundle.ValueBundle.Reputer] = stake
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

	networkLossBundle, err := synth.CalcNetworkLosses(topic.Id, nonce.BlockHeight, stakesByReputer, bundles)
	if err != nil {
		return err
	}

	ctx.Logger().Debug(fmt.Sprintf("Reputer Nonce %d Network Loss Bundle %v", &nonce.BlockHeight, networkLossBundle))

	err = k.InsertNetworkLossBundleAtBlock(ctx, topic.Id, nonce.BlockHeight, networkLossBundle)
	if err != nil {
		return err
	}

	types.EmitNewNetworkLossSetEvent(ctx, topic.Id, nonce.BlockHeight, networkLossBundle)

	err = synth.GetCalcSetNetworkRegrets(
		synth.GetCalcSetNetworkRegretsArgs{
			Ctx:           ctx,
			K:             *k,
			TopicId:       topic.Id,
			NetworkLosses: networkLossBundle,
			Nonce:         nonce,
			AlphaRegret:   topic.AlphaRegret,
			CNorm:         params.CNorm,
			PNorm:         topic.PNorm,
			EpsilonTopic:  topic.Epsilon,
		})
	if err != nil {
		return err
	}

	_, err = k.FulfillReputerNonce(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}

	err = k.SetTopicRewardNonce(ctx, topic.Id, nonce.BlockHeight)
	if err != nil {
		return err
	}

	err = k.SetReputerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}

	err = k.ResetActiveReputersForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}

	err = k.ResetReputersIndividualSubmissionsForTopic(ctx, topic.Id)
	if err != nil {
		return err
	}

	types.EmitNewReputerLastCommitSetEvent(ctx, topic.Id, blockHeight, &nonce)
	ctx.Logger().Info(fmt.Sprintf("Closed reputer nonce for topic: %d, nonce: %v", topic.Id, nonce))
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

	acceptedReputerValueBundle := &types.ReputerValueBundle{
		Pubkey: reputerValueBundle.Pubkey,
		ValueBundle: &types.ValueBundle{
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
		},
		Signature: reputerValueBundle.Signature,
	}

	return acceptedReputerValueBundle, nil
}
