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
	groupedBundles := make(map[uint64][]*types.LossBundle)
	for _, lossBundle := range msg.LossBundles {
		reputer, err := sdk.AccAddressFromBech32(lossBundle.Reputer)
		if err != nil {
			return nil, err
		}
		isLossSetter, err := ms.k.IsInReputerWhitelist(ctx, reputer)
		if err != nil {
			return nil, err
		}
		if isLossSetter {
			groupedBundles[lossBundle.TopicId] = append(groupedBundles[lossBundle.TopicId], lossBundle)
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	actualTimestamp := uint64(sdkCtx.BlockTime().Unix())

	for topicId, lossBundles := range groupedBundles {
		bundles := &types.LossBundles{
			LossBundles: lossBundles,
		}
		err = ms.k.InsertLossBundles(ctx, topicId, actualTimestamp, *bundles)
		if err != nil {
			return nil, err
		}
	}

	/**
	 * TODO calculate eq14,15, and possibly ep9-11
	 */

	return &types.MsgSetLossesResponse{}, nil
}
