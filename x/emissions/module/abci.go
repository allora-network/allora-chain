package module

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, am AppModule) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	sdkCtx.Logger().Debug(
		fmt.Sprintf("\n ---------------- Emissions EndBlock %d ------------------- \n",
			blockHeight))

	// Remove Stakers that have been wanting to unstake this block. They no longer get paid rewards
	RemoveStakes(sdkCtx, blockHeight, am.keeper)
	RemoveDelegateStakes(sdkCtx, blockHeight, am.keeper)

	// Get unnormalized weights of active topics and the sum weight and revenue they have generated
	weights, sumWeight, totalRevenue, err := rewards.GetAndUpdateActiveTopicWeights(sdkCtx, am.keeper, blockHeight)
	if err != nil {
		return errors.Wrapf(err, "Weights error")
	}
	sdkCtx.Logger().Debug(fmt.Sprintf("EndBlocker %d: Total Revenue: %v, Sum Weight: %v", blockHeight, totalRevenue, sumWeight))

	// REWARDS (will internally filter any non-RewardReady topics)
	err = rewards.EmitRewards(sdkCtx, am.keeper, blockHeight, weights, sumWeight, totalRevenue)
	if err != nil {
		sdkCtx.Logger().Error("Error calculating global emission per topic: ", err)
		return errors.Wrapf(err, "Rewards error")
	}

	// Reset the churn ready topics
	err = am.keeper.ResetChurnableTopics(ctx)
	if err != nil {
		sdkCtx.Logger().Error("Error resetting churn ready topics: ", err)
		return errors.Wrapf(err, "Resetting churn ready topics error")
	}

	// NONCE MGMT with Churnable weights
	var wg sync.WaitGroup
	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	fn := func(sdkCtx sdk.Context, topic *types.Topic) error {
		// Parallelize nonce management and update of topic to be in a churn ready state
		wg.Add(1)
		go func(topic types.Topic) {
			defer wg.Done()
			// Check the cadence of inferences, and just in case also check multiples of epoch lengths
			// to avoid potential situations where the block is missed
			if am.keeper.CheckCadence(blockHeight, topic) {
				sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
					topic.Id,
					topic.Metadata,
					topic.DefaultArg))

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockHeight)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error updating last inference ran: %s", err.Error()))
				}
				// Add Worker Nonces
				nextNonce := types.Nonce{BlockHeight: blockHeight + topic.EpochLength}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error adding worker nonce: %s", err.Error()))
					return
				}
				sdkCtx.Logger().Debug(fmt.Sprintf("Added worker nonce for topic %d: %v \n", topic.Id, nextNonce.BlockHeight))
				// To notify topic handler that the topic is ready for churn i.e. requests to be sent to workers and reputers
				err = am.keeper.AddChurnableTopic(ctx, topic.Id)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error setting churn ready topic: %s", err.Error()))
					return
				}

				MaxUnfulfilledReputerRequests := types.DefaultParams().MaxUnfulfilledReputerRequests
				moduleParams, err := am.keeper.GetParams(ctx)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error getting max retries to fulfil nonces for worker requests (using default), err: %s", err.Error()))
				} else {
					MaxUnfulfilledReputerRequests = moduleParams.MaxUnfulfilledReputerRequests
				}
				reputerPruningBlock := blockHeight - (int64(MaxUnfulfilledReputerRequests)*topic.EpochLength + topic.GroundTruthLag)
				if reputerPruningBlock > 0 {
					sdkCtx.Logger().Warn(fmt.Sprintf("Pruning reputer nonces before block: %v for topic: %d on block: %v", reputerPruningBlock, topic.Id, blockHeight))
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
		sdkCtx.Logger().Error("Error applying function on all rewardable topics: ", err)
		return err
	}
	wg.Wait()

	return nil
}
