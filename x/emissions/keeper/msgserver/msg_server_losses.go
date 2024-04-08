package msgserver

import (
	"context"

	synth "github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
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

	// Iterate through the array to ensure each reputer is in the whitelist
	// Group loss bundles by topicId - Create a map to store the grouped loss bundles
	lossBundles := make([]*types.ReputerValueBundle, 0)
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

		// TODO check signatures! throw if invalid!
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	bundles := types.ReputerValueBundles{
		ReputerValueBundles: lossBundles,
	}
	err = ms.k.InsertReputerLossBundlesAtBlock(ctx, msg.TopicId, msg.Nonce.Nonce, bundles)
	if err != nil {
		return nil, err
	}

	stakesOnTopic, err := ms.k.GetStakePlacementsByTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	}

	// Map list of stakesOnTopic to map of stakesByReputer
	stakesByReputer := make(map[string]types.StakePlacement)
	for _, stake := range stakesOnTopic {
		stakesByReputer[stake.Reputer] = stake
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

	// Update the unfulfilled nonces
	err = ms.k.FulfillReputerNonce(ctx, msg.TopicId, msg.Nonce)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertBulkReputerPayloadResponse{}, nil
}
