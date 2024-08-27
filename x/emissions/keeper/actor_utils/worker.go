package actorutils

import (
	"fmt"

	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WORKER NONCES CLOSING

// Closes an open worker nonce.
// this function does no processing of the actual worker data.
// all processing is done at time of reputer nonce closure.
func CloseWorkerNonce(k *keeper.Keeper, ctx sdk.Context, topicId keeper.TopicId, nonce types.Nonce) error {
	// Check if the topic exists
	topicExists, err := k.TopicExists(ctx, topicId)
	if err != nil {
		return err
	}
	if !topicExists {
		return types.ErrInvalidTopicId
	}

	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := k.IsWorkerNonceUnfulfilled(ctx, topicId, &nonce)
	if err != nil {
		return err
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return types.ErrUnfulfilledNonceNotFound
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return types.ErrInvalidTopicId
	}

	// Check if the window time has passed: if blockheight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	blockHeight := ctx.BlockHeight()
	if blockHeight <= topic.EpochLastEnded ||
		blockHeight > topic.EpochLastEnded+topic.GroundTruthLag {
		return types.ErrWorkerNonceWindowNotAvailable
	}

	// Update the unfulfilled worker nonce
	_, err = k.FulfillWorkerNonce(ctx, topicId, &nonce)
	if err != nil {
		return err
	}

	err = k.AddReputerNonce(ctx, topic.Id, &nonce)
	if err != nil {
		return err
	}

	err = k.SetWorkerTopicLastCommit(ctx, topic.Id, blockHeight, &nonce)
	if err != nil {
		return err
	}

	ctx.Logger().Info(fmt.Sprintf("Closed worker nonce for topic: %d, nonce: %v", topicId, nonce))
	// Return an empty response as the operation was successful
	return nil
}
