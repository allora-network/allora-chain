package msgserver

import (
	"context"

	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Called by reputer to submit their assessment of the quality of workers' work compared to ground truth
func (ms msgServer) InsertBulkReputerPayload(
	ctx context.Context,
	msg *types.MsgInsertBulkReputerPayload,
) (*types.MsgInsertBulkReputerPayloadResponse, error) {
	/// Do filters upon the leader (the sender) first, then do checks on each reputer in the payload
	/// All filters should be done in order of increasing computational complexity

	// Check if the sender is in the reputer whitelist
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	isLossSetter, err := ms.k.IsInReputerWhitelist(ctx, sender)
	if err != nil {
		return nil, err
	}
	if !isLossSetter {
		return nil, types.ErrNotInReputerWhitelist
	}

	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, msg.TopicId, msg.ReputerRequestNonce.WorkerNonce)
	if err != nil {
		return nil, err
	}
	if nonceUnfulfilled {
		return nil, types.ErrNonceNotUnfulfilled
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
		reputer, err := sdk.AccAddressFromBech32(bundle.Reputer)
		if err != nil {
			return nil, err
		}

		// Check that the reputer's value bundle is for a topic matching the leader's given topic
		if bundle.ValueBundle.TopicId != msg.TopicId {
			return nil, types.ErrInvalidTopicId
		}

		// Check that the reputer's value bundle is for a nonce matching the leader's given nonce
		if bundle.ReputerRequestNonce.WorkerNonce.Nonce != msg.ReputerRequestNonce.WorkerNonce.Nonce {
			return nil, types.ErrInvalidWorkerNonce
		}
		if bundle.ReputerRequestNonce.ReputerNonce.Nonce != msg.ReputerRequestNonce.ReputerNonce.Nonce {
			return nil, types.ErrInvalidReputerNonce
		}

		// Check if we've seen this reputer already in this bulk payload
		if _, ok := lossBundlesByReputer[bundle.Reputer]; !ok {
			// Check if the reputer is in the reputer whitelist
			isWhitelisted, err := ms.k.IsInReputerWhitelist(ctx, reputer)
			if err != nil {
				return nil, err
			}
			// We'll keep what we can get from the payload, but we'll ignore the rest
			if !isWhitelisted {
				continue
			}

			// Check that the reputer is registered in the topic
			isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, msg.TopicId, reputer)
			if err != nil {
				return nil, err
			}
			// We'll keep what we can get from the payload, but we'll ignore the rest
			if !isReputerRegistered {
				continue
			}

			// Examine forecast elements to verify that they're for registered inferers in the current set.
			// A check of their registration and other filters have already been applied when their inferences were inserted.
			// We keep what we can, ignoring the reputer and their contribution (losses) entirely
			// if they're left with no valid losses.
			filteredBundle, err := ms.FilterUnacceptedWorkersFromReputerValueBundle(ctx, msg.TopicId, *msg.ReputerRequestNonce, bundle)
			if err != nil {
				return nil, err
			}

			/// TODO check signatures! throw if invalid!

			/// If we do PoX-like anti-sybil procedure, would go here

			/// Filtering done now, now write what we must for inclusion

			lossBundlesByReputer[bundle.Reputer] = filteredBundle

			// Get the latest score for each reputer
			latestScore, err := ms.k.GetLatestReputerScore(ctx, msg.TopicId, reputer)
			if err != nil {
				return nil, err
			}
			latestReputerScores[bundle.Reputer] = latestScore
		}
	}

	// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topReputers := FindTopNByScoreDesc(params.MaxReputersPerTopicRequest, latestReputerScores, msg.ReputerRequestNonce.ReputerNonce.Nonce)

	// Check that the reputer in teh payload is a top reputer signatures
	stakesByReputer := make(map[string]types.StakePlacement)
	lossBundlesFromTopReputers := make([]*types.ReputerValueBundle, 0)
	for reputer, bundle := range lossBundlesByReputer {
		if _, ok := topReputers[reputer]; !ok {
			continue
		}

		lossBundlesFromTopReputers = append(lossBundlesFromTopReputers, bundle)

		stake, err := ms.k.GetStakeOnTopicFromReputer(ctx, msg.TopicId, sdk.AccAddress(bundle.Reputer))
		if err != nil {
			return nil, err
		}

		stakesByReputer[bundle.Reputer] = types.StakePlacement{
			TopicId: msg.TopicId,
			Reputer: bundle.Reputer,
			Amount:  stake,
		}
	}

	bundles := types.ReputerValueBundles{
		ReputerValueBundles: lossBundlesFromTopReputers,
	}
	err = ms.k.InsertReputerLossBundlesAtBlock(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.Nonce, bundles)
	if err != nil {
		return nil, err
	}

	networkLossBundle, err := synth.CalcNetworkLosses(stakesByReputer, bundles, params.Epsilon)
	if err != nil {
		return nil, err
	}

	err = ms.k.InsertNetworkLossBundleAtBlock(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.Nonce, networkLossBundle)
	if err != nil {
		return nil, err
	}

	err = synth.GetCalcSetNetworkRegrets(ctx.(sdk.Context), ms.k, msg.TopicId, networkLossBundle, *msg.ReputerRequestNonce.ReputerNonce, params.AlphaRegret)
	if err != nil {
		return nil, err
	}

	// Calculate and Set the reputer scores
	_, err = rewards.GenerateReputerScores(ctx.(sdk.Context), ms.k, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.Nonce, bundles)
	if err != nil {
		return nil, err
	}

	// Calculate and Set the worker scores for their inference work
	_, err = rewards.GenerateInferenceScores(ctx.(sdk.Context), ms.k, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.Nonce, networkLossBundle)
	if err != nil {
		return nil, err
	}

	// Calculate and Set the worker scores for their forecast work
	_, err = rewards.GenerateForecastScores(ctx.(sdk.Context), ms.k, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce.Nonce, networkLossBundle)
	if err != nil {
		return nil, err
	}

	// Update the unfulfilled nonces
	err = ms.k.FulfillReputerNonce(ctx, msg.TopicId, msg.ReputerRequestNonce.ReputerNonce)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertBulkReputerPayloadResponse{}, nil
}

// Filter out values of unaccepted workers.
// It is assumed that the work of inferers and forecasters stored at the nonce is already filtered for acceptance.
func (ms msgServer) FilterUnacceptedWorkersFromReputerValueBundle(
	ctx context.Context,
	topicId uint64,
	reputerRequestNonce types.ReputerRequestNonce,
	reputerValueBundle *types.ReputerValueBundle,
) (*types.ReputerValueBundle, error) {
	// Get the accepted inferers of the associated worker response payload
	inferences, err := ms.k.GetInferencesAtBlock(ctx, topicId, reputerRequestNonce.WorkerNonce.Nonce)
	if err != nil {
		return nil, err
	}
	acceptedInferersOfBatch := make(map[string]bool)
	for _, inference := range inferences.Inferences {
		acceptedInferersOfBatch[inference.Inferer] = true
	}

	// Get the accepted forecasters of the associated worker response payload
	forecasts, err := ms.k.GetForecastsAtBlock(ctx, topicId, reputerRequestNonce.WorkerNonce.Nonce)
	if err != nil {
		return nil, err
	}
	acceptedForecastersOfBatch := make(map[string]bool)
	for _, forecast := range forecasts.Forecasts {
		acceptedForecastersOfBatch[forecast.Forecaster] = true
	}

	// Filter out values of unaccepted workers

	acceptedInfererValues := make([]*types.WorkerAttributedValue, 0)
	for _, workerVal := range reputerValueBundle.ValueBundle.InfererValues {
		if _, ok := acceptedInferersOfBatch[workerVal.Worker]; ok {
			acceptedInfererValues = append(acceptedInfererValues, workerVal)
		}
	}

	acceptedForecasterValues := make([]*types.WorkerAttributedValue, 0)
	for _, workerVal := range reputerValueBundle.ValueBundle.ForecasterValues {
		if _, ok := acceptedForecastersOfBatch[workerVal.Worker]; ok {
			acceptedForecasterValues = append(acceptedForecasterValues, workerVal)
		}
	}

	acceptedOneOutInfererValues := make([]*types.WithheldWorkerAttributedValue, 0)
	for _, workerVal := range reputerValueBundle.ValueBundle.OneOutInfererValues {
		if _, ok := acceptedInferersOfBatch[workerVal.Worker]; ok {
			acceptedOneOutInfererValues = append(acceptedOneOutInfererValues, workerVal)
		}
	}

	acceptedOneOutForecasterValues := make([]*types.WithheldWorkerAttributedValue, 0)
	for _, workerVal := range reputerValueBundle.ValueBundle.OneOutForecasterValues {
		if _, ok := acceptedForecastersOfBatch[workerVal.Worker]; ok {
			acceptedOneOutForecasterValues = append(acceptedOneOutForecasterValues, workerVal)
		}
	}

	acceptedOneInForecasterValues := make([]*types.WorkerAttributedValue, 0)
	for _, workerVal := range reputerValueBundle.ValueBundle.OneInForecasterValues {
		if _, ok := acceptedForecastersOfBatch[workerVal.Worker]; ok {
			acceptedOneInForecasterValues = append(acceptedOneInForecasterValues, workerVal)
		}
	}

	acceptedReputerValueBundle := &types.ReputerValueBundle{
		Reputer: reputerValueBundle.Reputer,
		ValueBundle: &types.ValueBundle{
			TopicId:                reputerValueBundle.ValueBundle.TopicId,
			InfererValues:          acceptedInfererValues,
			ForecasterValues:       acceptedForecasterValues,
			OneOutInfererValues:    acceptedOneOutInfererValues,
			OneOutForecasterValues: acceptedOneOutForecasterValues,
			OneInForecasterValues:  acceptedOneInForecasterValues,
		},
	}
	return acceptedReputerValueBundle, nil
}
