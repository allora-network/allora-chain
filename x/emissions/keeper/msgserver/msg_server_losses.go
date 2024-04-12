package msgserver

import (
	"context"
	"encoding/json"
	"strconv"

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

	// Verify nonce signature
	pk := ms.k.AccountKeeper().GetAccount(ctx, sender)
	stringNonce := strconv.FormatInt(msg.Nonce.Nonce, 10)
	nonceBytes := []byte(stringNonce)
	if !pk.GetPubKey().VerifySignature(nonceBytes, msg.Signature) {
		return nil, types.ErrSignatureVerificationFailed
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
		reputerAddr, err := sdk.AccAddressFromBech32(bundle.Reputer)
		if err != nil {
			return nil, err
		}
		pk := ms.k.AccountKeeper().GetAccount(ctx, reputerAddr)
		src, _ := json.Marshal(bundle.ValueBundle)
		if !pk.GetPubKey().VerifySignature(src, bundle.Signature) {
			return nil, types.ErrSignatureVerificationFailed
		}
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
