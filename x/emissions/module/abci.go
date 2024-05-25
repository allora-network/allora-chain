package module

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, am AppModule) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Debug("\n ---------------- Emissions EndBlock ------------------- \n")
	blockHeight := sdkCtx.BlockHeight()

	// Get unnormalized weights of active topics
	weights, sumWeight, totalRevenue, err := rewards.GetAndOptionallyUpdateActiveTopicWeights(sdkCtx, am.keeper, blockHeight, true)
	if err != nil {
		return errors.Wrapf(err, "Weights error")
	}
	sdkCtx.Logger().Debug(fmt.Sprintf("EndBlocker: Total Revenue: %v, Sum Weight: %v", totalRevenue, sumWeight))

	// REWARDS (will internally filter any non-RewardReady topics)
	err = rewards.EmitRewards(sdkCtx, am.keeper, blockHeight, weights, sumWeight, totalRevenue)
	if err != nil {
		sdkCtx.Logger().Error("Error calculating global emission per topic: ", err)
		return errors.Wrapf(err, "Rewards error")
	}

	// Reset the churn ready topics
	err = am.keeper.ResetChurnReadyTopics(ctx)
	if err != nil {
		sdkCtx.Logger().Error("Error resetting churn ready topics: ", err)
		return errors.Wrapf(err, "Resetting churn ready topics error")
	}

	// NONCE MGMT with churnReady weights
	var wg sync.WaitGroup
	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	fn := func(ctx context.Context, topic *types.Topic) error {
		// Parallelize nonce management and update of topic to be in a churn ready state
		wg.Add(1)
		go func(topic types.Topic) {
			defer wg.Done()
			// Check the cadence of inferences, and just in case also check multiples of epoch lengths
			// to avoid potential situations where the block is missed
			if keeper.CheckCadence(blockHeight, topic) {
				sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
					topic.Id,
					topic.Metadata,
					topic.DefaultArg))

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockHeight)
				if err != nil {
					sdkCtx.Logger().Warn("Error updating last inference ran: ", err)
				}
				// Add Worker Nonces
				nextNonce := types.Nonce{BlockHeight: blockHeight + topic.EpochLength}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					sdkCtx.Logger().Warn("Error adding worker nonce: ", err)
					return
				}
				// To notify topic handler that the topic is ready for churn i.e. requests to be sent to workers and reputers
				err = am.keeper.AddChurnReadyTopic(ctx, topic.Id)
				if err != nil {
					sdkCtx.Logger().Warn("Error setting churn ready topic: ", err)
					return
				}

				MaxUnfulfilledReputerRequests, err := am.keeper.GetParamsMaxUnfulfilledReputerRequests(ctx)
				if err != nil {
					MaxUnfulfilledReputerRequests = types.DefaultParams().MaxUnfulfilledReputerRequests
					sdkCtx.Logger().Warn("Error getting max retries to fulfil nonces for worker requests (using default), err:", err)
				}
				reputerPruningBlock := blockHeight - (int64(MaxUnfulfilledReputerRequests)*topic.EpochLength + topic.GroundTruthLag)
				if reputerPruningBlock > 0 {
					sdkCtx.Logger().Warn("Pruning reputer nonces before block: ", reputerPruningBlock, " for topic: ", topic.Id, " on block: ", blockHeight)
					am.keeper.PruneReputerNonces(sdkCtx, topic.Id, reputerPruningBlock)

					workerPruningBlock := reputerPruningBlock - topic.EpochLength
					if workerPruningBlock > 0 {
						sdkCtx.Logger().Debug("Pruning worker nonces before block: ", workerPruningBlock, " for topic: ", topic.Id)
						// Prune old worker nonces previous to current blockHeight to avoid inserting inferences after its time has passed
						// Reputer nonces need to check worker nonces one epoch before the reputer nonces
						am.keeper.PruneWorkerNonces(sdkCtx, topic.Id, workerPruningBlock)
					}
				}
			}
		}(*topic)
		return nil
	}
	err = rewards.IdentifyChurnableAmongActiveTopicsAndApplyFn(
		sdkCtx,
		am.keeper,
		blockHeight,
		fn,
		weights,
	)
	if err != nil {
		sdkCtx.Logger().Error("Error applying function on all reward ready topics: ", err)
		return err
	}
	wg.Wait()

	return nil
}
