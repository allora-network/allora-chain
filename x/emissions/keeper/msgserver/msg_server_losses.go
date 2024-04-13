package msgserver

import (
	"context"

	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Called by reputer to submit their assessment of the quality of workers' work compared to ground truth
func (ms msgServer) InsertBulkReputerPayload(ctx context.Context, msg *types.MsgInsertBulkReputerPayload) (*types.MsgInsertBulkReputerPayloadResponse, error) {
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
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, msg.TopicId, msg.Nonce)
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

	// Iterate through the array to ensure each reputer is in the whitelist
	// and get get score for each reputer => later we can skim only the top few by score descending
	lossBundles := make([]*types.ReputerValueBundle, 0)
	latestReputerScores := make(map[string]types.Score)
	for _, bundle := range msg.ReputerValueBundles {
		reputer, err := sdk.AccAddressFromBech32(bundle.Reputer)
		if err != nil {
			return nil, err
		}
		isLossSetter, err := ms.k.IsInReputerWhitelist(ctx, reputer)
		if err != nil {
			return nil, err
		}
		if isLossSetter {
			lossBundles = append(lossBundles, bundle)
		}

		// Get the latest score for each reputer
		latestScore, err := ms.k.GetLatestReputerScore(ctx, msg.TopicId, reputer)
		if err != nil {
			return nil, err
		}
		latestReputerScores[bundle.Reputer] = latestScore

		// If we do PoX-like anti-sybil procedure, would go here
	}

	// If we pseudo-random sample from the non-sybil set of reputers, we would do it here
	topReputers := FindTopNByScoreDesc(params.MaxReputersPerTopicRequest, latestReputerScores, msg.Nonce.Nonce)

	// Check that the reputer in teh payload is a top reputer signatures
	stakesByReputer := make(map[string]types.StakePlacement)
	lossBundlesFromTopReputers := make([]*types.ReputerValueBundle, 0)
	for _, bundle := range lossBundles {
		if _, ok := topReputers[bundle.Reputer]; !ok {
			continue
		}

		//
		// TODO check signatures! throw if invalid!
		//

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
	err = ms.k.InsertReputerLossBundlesAtBlock(ctx, msg.TopicId, msg.Nonce.Nonce, bundles)
	if err != nil {
		return nil, err
	}

	networkLossBundle, err := synth.CalcNetworkLosses(stakesByReputer, bundles, params.Epsilon)
	if err != nil {
		return nil, err
	}

	err = ms.k.InsertNetworkLossBundleAtBlock(ctx, msg.TopicId, msg.Nonce.Nonce, networkLossBundle)
	if err != nil {
		return nil, err
	}

	err = synth.GetCalcSetNetworkRegrets(ctx.(sdk.Context), ms.k, msg.TopicId, networkLossBundle, *msg.Nonce, params.AlphaRegret)
	if err != nil {
		return nil, err
	}

	// Calculate and Set the reputer scores
	_, err = rewards.GenerateReputerScores(ctx.(sdk.Context), ms.k, msg.TopicId, msg.Nonce.Nonce, bundles)
	if err != nil {
		return nil, err
	}

	// Calculate and Set the worker scores for their inference work
	_, err = rewards.GenerateInferenceScores(ctx.(sdk.Context), ms.k, msg.TopicId, msg.Nonce.Nonce, networkLossBundle)
	if err != nil {
		return nil, err
	}

	// Calculate and Set the worker scores for their forecast work
	_, err = rewards.GenerateForecastScores(ctx.(sdk.Context), ms.k, msg.TopicId, msg.Nonce.Nonce, networkLossBundle)
	if err != nil {
		return nil, err
	}

	// Update the unfulfilled nonces
	err = ms.k.FulfillReputerNonce(ctx, msg.TopicId, msg.Nonce)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertBulkReputerPayloadResponse{}, nil
}
