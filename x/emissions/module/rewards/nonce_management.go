package rewards

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	allorautils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Update unfullfilled reputer nonces for topic
func UpdateReputerNonce(ctx sdk.Context, k keeper.Keeper, topic types.Topic, block BlockHeight) error {
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topic.Id)
	if err != nil {
		ctx.Logger().Warn("Error getting unfulfilled worker nonces", "error", err)
		return err
	}
	for _, nonce := range nonces.Nonces {
		if block >= nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag {
			windowStartBlock := block
			windowEndBlock := block + topic.EpochLength
			types.EmitReputerSubmissionWindowOpenedEvent(ctx, topic.Id, nonce.ReputerNonce.BlockHeight, windowStartBlock, windowEndBlock)
		}
		// Check if current blockheight has reached the blockheight of the nonce + groundTruthLag + epochLength
		// This means one epochLength is allowed for reputation responses to be sent since ground truth is revealed.
		closingReputerNonceMinBlockHeight := nonce.ReputerNonce.BlockHeight + topic.GroundTruthLag + topic.EpochLength
		if block >= closingReputerNonceMinBlockHeight {
			ctx.Logger().Debug("ABCI EndBlocker: Closing reputer nonce", "topic", topic.Id, "nonce", nonce, "min", closingReputerNonceMinBlockHeight)
			err = allorautils.CloseReputerNonce(&k, ctx, topic, *nonce.ReputerNonce)
			if err != nil {
				ctx.Logger().Error("Error closing reputer nonce", "error", err)
			}
		}
	}
	return err
}

// Prune reputer and worker nonces
func PruneReputerAndWorkerNonces(ctx sdk.Context, k keeper.Keeper, topic types.Topic, block BlockHeight) error {
	var maxUnfulfilledReputerRequests uint64
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		ctx.Logger().Warn("Error getting max retries to fulfil nonces for worker requests (using default)", "error", err)
		return err
	} else {
		maxUnfulfilledReputerRequests = moduleParams.MaxUnfulfilledReputerRequests
	}
	// Adding one to cover for one extra epochLength
	reputerPruningBlock := block - (int64(maxUnfulfilledReputerRequests+1)*topic.EpochLength + topic.GroundTruthLag) //nolint:gosec // G115: integer overflow conversion uint64 -> int64 (gosec)
	if reputerPruningBlock > 0 {
		ctx.Logger().Debug("Pruning reputer nonces before block", "reputerPruningBlock", reputerPruningBlock, "topicId", topic.Id, "block", block)
		err = k.PruneReputerNonces(ctx, topic.Id, reputerPruningBlock)
		if err != nil {
			ctx.Logger().Warn("Error pruning reputer nonces", "error", err)
		}

		// Reputer nonces need to check worker nonces from one epoch before
		workerPruningBlock := reputerPruningBlock - topic.EpochLength
		if workerPruningBlock > 0 {
			ctx.Logger().Debug("Pruning worker nonces before block", "workerPruningBlock", workerPruningBlock, "topicId", topic.Id)
			// Prune old worker nonces previous to current block to avoid inserting inferences after its time has passed
			err = k.PruneWorkerNonces(ctx, topic.Id, workerPruningBlock)
			if err != nil {
				ctx.Logger().Warn("Error pruning worker nonces", "error", err)
			}
		}
	}
	return err
}
