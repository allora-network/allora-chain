package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	allorautils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Update unfullfilled reputer nonces for topic
func UpdateReputerNonce(ctx sdk.Context, k keeper.Keeper, topic types.Topic, block BlockHeight) error {
	nonces, err := k.GetNonceKeeper().GetUnfulfilledReputerNonces(ctx, topic.Id)
	if err != nil {
		ctx.Logger().Warn("Error getting unfulfilled worker nonces", "error", err)
		return err
	}
	var updateErr error
	for _, nonce := range nonces.Nonces {
		windowStart, windowEnd, boundsErr := keeper.ReputerSubmissionWindowBounds(topic, *nonce)
		if boundsErr != nil {
			ctx.Logger().Warn("Error computing reputer submission window bounds", "error", boundsErr)
			updateErr = boundsErr
			continue
		}
		// Active topics are processed once per epoch, so this half-open interval
		// emits at most once and attributes a shared boundary to the newer nonce.
		if block >= windowStart && block < windowEnd {
			types.EmitNewReputerSubmissionWindowOpenedEvent(ctx, topic.Id, nonce.ReputerNonce.BlockHeight, windowEnd)
		}
		if block >= windowEnd {
			ctx.Logger().Debug("ABCI EndBlocker: Closing reputer nonce", "topic", topic.Id, "nonce", nonce, "min", windowEnd)
			closeErr := allorautils.CloseReputerNonce(&k, ctx, topic, *nonce.ReputerNonce)
			if closeErr != nil {
				ctx.Logger().Error("Error closing reputer nonce", "error", closeErr)
				updateErr = closeErr
			}
		}
	}
	return updateErr
}

// Prune reputer and worker nonces
func PruneReputerAndWorkerNonces(ctx sdk.Context, k keeper.Keeper, topic types.Topic, block BlockHeight) error {
	var maxUnfulfilledReputerRequests uint64
	moduleParams, err := k.GetParamsKeeper().GetParams(ctx)
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
		err = k.GetNonceKeeper().PruneReputerNonces(ctx, topic.Id, reputerPruningBlock)
		if err != nil {
			ctx.Logger().Warn("Error pruning reputer nonces", "error", err)
		}

		// Reputer nonces need to check worker nonces from one epoch before
		workerPruningBlock := reputerPruningBlock - topic.EpochLength
		if workerPruningBlock > 0 {
			ctx.Logger().Debug("Pruning worker nonces before block", "workerPruningBlock", workerPruningBlock, "topicId", topic.Id)
			// Prune old worker nonces previous to current block to avoid inserting inferences after its time has passed
			err = k.GetNonceKeeper().PruneWorkerNonces(ctx, topic.Id, workerPruningBlock)
			if err != nil {
				ctx.Logger().Warn("Error pruning worker nonces", "error", err)
			}
		}
	}
	return err
}
