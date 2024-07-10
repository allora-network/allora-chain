package msgserver

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Called by reputer to submit their assessment of the quality of workers' work compared to ground truth
func (ms msgServer) InsertBulkReputerPayload(
	ctx context.Context,
	msg *types.MsgInsertBulkReputerPayload,
) (*types.MsgInsertBulkReputerPayloadResponse, error) {
	err := checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	// Validate top level here. We avoid validating the full message here because that would allow for 1 bundle to fail the whole message.
	if err := msg.ValidateTopLevel(); err != nil {
		return nil, err
	}

	// Check if the topic exists
	topic, err := ms.k.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, sdkerrors.ErrNotFound
	}

	/// Do filters upon the leader (the sender) first, then do checks on each reputer in the payload
	/// All filters should be done in order of increasing computational complexity

	// Check if the worker nonce is unfulfilled
	workerNonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// Throw if worker nonce is unfulfilled -- can't report losses on something not yet committed
	if workerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(
			types.ErrNonceStillUnfulfilled,
			"Reputer's worker nonce not yet fulfilled for reputer block: %v",
			msg.ReputerRequestNonce.ReputerNonce.BlockHeight,
		)
	}

	// Check if the reputer nonce is unfulfilled
	reputerNonceUnfulfilled, err := ms.k.IsReputerNonceUnfulfilled(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// Throw if already fulfilled -- can't return a response twice
	if !reputerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(
			types.ErrNonceAlreadyFulfilled,
			"Reputer nonce already fulfilled: %v",
			msg.ReputerRequestNonce.ReputerNonce.BlockHeight,
		)
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	/// Do checks on each reputer in the payload
	// Iterate through the array to ensure each reputer is in the whitelist
	// and get get score for each reputer => later we can skim only the top few by score descending
	lossBundlesByReputer := make(map[string]*types.ReputerValueBundle)
	latestReputerScores := make(map[string]types.Score)
	for _, bundle := range msg.ReputerValueBundles {
		if err := bundle.Validate(); err != nil {
			continue
		}

		reputer := bundle.ValueBundle.Reputer

		// Check that the reputer's value bundle is for a topic matching the leader's given topic
		if bundle.ValueBundle.TopicId != msg.TopicId {
			continue
		}
		// Check that the reputer's value bundle is for a nonce matching the leader's given nonce
		if bundle.ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight != msg.ReputerRequestNonce.ReputerNonce.BlockHeight {
			continue
		}

		// Check if we've seen this reputer already in this bulk payload
		if _, ok := lossBundlesByReputer[bundle.ValueBundle.Reputer]; !ok {
			// Check that the reputer is registered in the topic
			isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, bundle.ValueBundle.TopicId, reputer)
			if err != nil {
				continue
			}
			// We'll keep what we can get from the payload, but we'll ignore the rest
			if !isReputerRegistered {
				continue
			}

			// Check that the reputer enough stake in the topic
			stake, err := ms.k.GetStakeReputerAuthority(ctx, msg.TopicId, reputer)
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
			filteredBundle, err := filterUnacceptedWorkersFromReputerValueBundle(ctx, ms, msg.TopicId, *msg.ReputerRequestNonce, bundle)
			if err != nil {
				continue
			}

			/// If we do PoX-like anti-sybil procedure, would go here

			/// Filtering done now, now write what we must for inclusion

			// Get the latest score for each reputer
			latestScore, err := ms.k.GetLatestReputerScore(ctx, bundle.ValueBundle.TopicId, reputer)
			if err != nil {
				continue
			}
			latestReputerScores[bundle.ValueBundle.Reputer] = latestScore
			lossBundlesByReputer[bundle.ValueBundle.Reputer] = filteredBundle
		}
	}

	// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topReputers := FindTopNByScoreDesc(params.MaxTopReputersToReward, latestReputerScores, msg.ReputerRequestNonce.ReputerNonce.BlockHeight)

	// Check that the reputer in the payload is a top reputer among those who have submitted losses
	stakesByReputer := make(map[string]cosmosMath.Int)
	lossBundlesFromTopReputers := make([]*types.ReputerValueBundle, 0)
	for _, reputer := range topReputers {
		stake, err := ms.k.GetStakeReputerAuthority(ctx, msg.TopicId, reputer)
		if err != nil {
			continue
		}

		lossBundlesFromTopReputers = append(lossBundlesFromTopReputers, lossBundlesByReputer[reputer])
		stakesByReputer[reputer] = stake
	}
	// sort by reputer score descending
	sort.Slice(lossBundlesFromTopReputers, func(i, j int) bool {
		return lossBundlesFromTopReputers[i].ValueBundle.Reputer < lossBundlesFromTopReputers[j].ValueBundle.Reputer
	})

	if len(lossBundlesFromTopReputers) == 0 {
		return nil, types.ErrNoValidBundles
	}

	bundles := types.ReputerValueBundles{
		ReputerValueBundles: lossBundlesFromTopReputers,
	}
	err = ms.k.InsertReputerLossBundlesAtBlock(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.BlockHeight, bundles)
	if err != nil {
		return nil, err
	}

	networkLossBundle, err := synth.CalcNetworkLosses(stakesByReputer, bundles, topic.Epsilon)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Debug(fmt.Sprintf("Reputer Nonce %d Network Loss Bundle %v", msg.ReputerRequestNonce.ReputerNonce.BlockHeight, networkLossBundle))

	networkLossBundle.ReputerRequestNonce = msg.ReputerRequestNonce

	err = ms.k.InsertNetworkLossBundleAtBlock(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.BlockHeight, networkLossBundle)
	if err != nil {
		return nil, err
	}

	types.EmitNewNetworkLossSetEvent(sdkCtx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.BlockHeight, networkLossBundle)

	err = synth.GetCalcSetNetworkRegrets(sdkCtx, ms.k, msg.TopicId, networkLossBundle, *msg.ReputerRequestNonce.ReputerNonce, topic.AlphaRegret)
	if err != nil {
		return nil, err
	}

	// Update the unfulfilled nonces
	_, err = ms.k.FulfillReputerNonce(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce)
	if err != nil {
		return nil, err
	}

	// Update topic reward nonce
	err = ms.k.SetTopicRewardNonce(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		return nil, err
	}

	err = ms.k.AddRewardableTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}

	blockHeight := sdkCtx.BlockHeight()
	err = ms.k.SetTopicLastCommit(ctx, topic.Id, blockHeight, msg.ReputerRequestNonce.ReputerNonce, msg.Sender, types.ActorType_REPUTER)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertBulkReputerPayloadResponse{}, nil
}

// Filter out values of unaccepted workers.
// It is assumed that the work of inferers and forecasters stored at the nonce is already filtered for acceptance.
// This also removes duplicate values of the same worker.
func filterUnacceptedWorkersFromReputerValueBundle(
	ctx context.Context,
	ms msgServer,
	topicId uint64,
	reputerRequestNonce types.ReputerRequestNonce,
	reputerValueBundle *types.ReputerValueBundle,
) (*types.ReputerValueBundle, error) {
	// Get the accepted inferers of the associated worker response payload
	inferences, err := ms.k.GetInferencesAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight)
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
	forecasts, err := ms.k.GetForecastsAtBlock(ctx, topicId, reputerRequestNonce.ReputerNonce.BlockHeight)
	if err != nil {
		// If no forecasts, we'll just assume there are 0 forecasters
		if errors.Is(err, collections.ErrNotFound) {
			forecasts = &types.Forecasts{Forecasts: make([]*types.Forecast, 0)}
		} else {
			return nil, err
		}
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
