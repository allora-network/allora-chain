package msgserver

import (
	"context"

	synth "github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Called by reputer to submit their assessment of the quality of workers' work compared to ground truth
func (ms msgServer) InsertLosses(ctx context.Context, msg *types.MsgInsertLosses) (*types.MsgInsertLossesResponse, error) {
	// Check if the sender is in the weight setting whitelist
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

	// Iterate through the array to ensure each reputer is in the whitelist
	// Group loss bundles by topicId - Create a map to store the grouped loss bundles
	groupedBundles := make(map[uint64][]*types.ReputerValueBundle)
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
			groupedBundles[bundle.ValueBundle.TopicId] = append(groupedBundles[bundle.ValueBundle.TopicId], bundle)
		}
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	for topicId, bundles := range groupedBundles {
		// Get the latest unfulfilled nonces
		unfulfilledNonces, err := ms.k.GetUnfulfilledReputerNonces(ctx, topicId)
		if err != nil {
			return nil, err
		}

		// Check if the nonce is among the latest unfulfilled nonces
		nonceFound := false
		for _, nonce := range unfulfilledNonces.Nonces {
			if nonce.Nonce == msg.Nonce.Nonce {
				nonceFound = true
				break
			}
		}
		if !nonceFound {
			continue
		}

		bundles := &types.ReputerValueBundles{
			ReputerValueBundles: bundles,
		}
		err = ms.k.InsertReputerLossBundlesAtBlock(ctx, topicId, msg.Nonce.Nonce, *bundles)
		if err != nil {
			return nil, err
		}

		stakesOnTopic, err := ms.k.GetStakePlacementsByTopic(ctx, topicId)
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

		err = ms.k.InsertNetworkLossBundleAtBlock(ctx, topicId, msg.Nonce.Nonce, networkLossBundle)
		if err != nil {
			return nil, err
		}

		err = synth.GetCalcSetNetworkRegrets(ctx.(sdk.Context), ms.k, topicId, networkLossBundle, *msg.Nonce, params.AlphaRegret)
		if err != nil {
			return nil, err
		}

		// Update the unfulfilled nonces
		err = ms.k.FulfillReputerNonce(ctx, topicId, msg.Nonce)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgInsertLossesResponse{}, nil
}
