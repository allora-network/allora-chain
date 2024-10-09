package rewards

import (
	"fmt"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	allorautils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Update unfullfilled reputer nonces for topic
func UpdateReputerNonce(ctx sdk.Context, k keeper.Keeper, topic types.Topic, block BlockHeight) error {
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topic.Id)
	if err != nil {
		ctx.Logger().Warn(fmt.Sprintf("Error getting unfulfilled worker nonces: %s", err.Error()))
		return err
	}
	for _, nonce := range nonces.Nonces {
		// Check if current blockheight has reached the blockheight of the nonce + groundTruthLag + epochLength
		// This means one epochLength is allowed for reputation responses to be sent since ground truth is revealed.
		closingReputerNonceMinBlockHeight := nonce.ReputerNonce.BlockHeight + topic.GroundTruthLag + topic.EpochLength
		if block >= closingReputerNonceMinBlockHeight {
			ctx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Closing reputer nonce for topic: %v nonce: %v, min: %d. \n",
				topic.Id, nonce, closingReputerNonceMinBlockHeight))
			err = allorautils.CloseReputerNonce(&k, ctx, topic, *nonce.ReputerNonce)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error closing reputer nonce: %s", err.Error()))
				// Proactively close the nonce to avoid
				_, err = k.FulfillReputerNonce(ctx, topic.Id, nonce.ReputerNonce)
				if err != nil {
					ctx.Logger().Warn(fmt.Sprintf("Error fulfilling reputer nonce: %s", err.Error()))
				}
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
		ctx.Logger().Warn(fmt.Sprintf("Error getting max retries to fulfil nonces for worker requests (using default), err: %s", err.Error()))
		return err
	} else {
		maxUnfulfilledReputerRequests = moduleParams.MaxUnfulfilledReputerRequests
	}
	// Adding one to cover for one extra epochLength
	reputerPruningBlock := block - (int64(maxUnfulfilledReputerRequests+1)*topic.EpochLength + topic.GroundTruthLag) //nolint:gosec // G115: integer overflow conversion uint64 -> int64 (gosec)
	if reputerPruningBlock > 0 {
		ctx.Logger().Debug(fmt.Sprintf("Pruning reputer nonces before block: %v for topic: %d on block: %v", reputerPruningBlock, topic.Id, block))
		err = k.PruneReputerNonces(ctx, topic.Id, reputerPruningBlock)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error pruning reputer nonces: %s", err.Error()))
		}

		// Reputer nonces need to check worker nonces from one epoch before
		workerPruningBlock := reputerPruningBlock - topic.EpochLength
		if workerPruningBlock > 0 {
			ctx.Logger().Debug(fmt.Sprintf("Pruning worker nonces before block: %d  for topic: %d", workerPruningBlock, topic.Id))
			// Prune old worker nonces previous to current block to avoid inserting inferences after its time has passed
			err = k.PruneWorkerNonces(ctx, topic.Id, workerPruningBlock)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error pruning worker nonces: %s", err.Error()))
			}
		}
	}
	return err
}
