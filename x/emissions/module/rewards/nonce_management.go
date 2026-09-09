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
	for _, nonce := range nonces.Nonces {
		// Must match keeper.ReputerSubmissionWindowBounds, the single source of
		// truth for when a submission is actually accepted. This used to be its
		// own formula ending at +extraLag instead of the window's real start
		// (+extraLag was folded into the acceptance lower bound to close a
		// permanent-trap bug -- see CHANGELOG). On an uninterrupted epoch grid the
		// two formulas coincide, but a topic that was inactivated and reactivated
		// gets a new churn schedule with no relation to an existing nonce's grid
		// (topic_activation.go), so a reactivated topic could hit a block inside
		// the old range and announce the window open while it was still rejecting
		// submissions until the true start.
		windowStart, windowEnd, err := keeper.ReputerSubmissionWindowBounds(topic, *nonce)
		if err != nil {
			ctx.Logger().Warn("Error computing reputer submission window bounds", "error", err)
			continue
		}
		if block == windowStart {
			types.EmitNewReputerSubmissionWindowOpenedEvent(ctx, topic.Id, nonce.ReputerNonce.BlockHeight, windowEnd)
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
