package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Called by reputer to submit their assessment of the quality of workers' work compared to ground truth
func (ms msgServer) InsertLosses(ctx context.Context, msg *types.MsgSetLosses) (*types.MsgSetLossesResponse, error) {
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

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	for topicId, bundles := range groupedBundles {
		bundles := &types.ReputerValueBundles{
			ReputerValueBundles: bundles,
		}
		err = ms.k.InsertValueBundles(ctx, topicId, blockHeight, *bundles)
		if err != nil {
			return nil, err
		}
	}

	/**
	 * TODO calculate eq14,15, and possibly ep9-12
	 * TODO calc eq3-15\13 when reputer queries for the chain. Then, make caching tickets for the validators
	 */

	return &types.MsgSetLossesResponse{}, nil
}
