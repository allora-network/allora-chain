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

	// NONCE MGMT with Churnable weights
	var wg sync.WaitGroup
	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	fn := func(sdkCtx sdk.Context, topic *types.Topic) error {
		// Parallelize nonce management and update of topic to be in a churn ready state
		wg.Add(1)
		localTopic := *topic
		go func(topic types.Topic) {
			defer wg.Done()
			// Check the cadence of inferences, and just in case also check multiples of epoch lengths
			// to avoid potential situations where the block is missed
			if am.keeper.CheckWorkerOpenCadence(blockHeight, topic) {
				sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Inference cadence met for topic: %v metadata: %s . \n",
					topic.Id,
					topic.Metadata))

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockHeight)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error updating last inference ran: %s", err.Error()))
				}
				// Add Worker Nonces
				nextNonce := types.Nonce{BlockHeight: blockHeight}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error adding worker nonce: %s", err.Error()))
					return
				}
				sdkCtx.Logger().Debug(fmt.Sprintf("Added worker nonce for topic %d: %v \n", topic.Id, nextNonce.BlockHeight))

				MaxUnfulfilledReputerRequests := types.DefaultParams().MaxUnfulfilledReputerRequests
				moduleParams, err := am.keeper.GetParams(ctx)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error getting max retries to fulfil nonces for worker requests (using default), err: %s", err.Error()))
				} else {
					MaxUnfulfilledReputerRequests = moduleParams.MaxUnfulfilledReputerRequests
				}
				// Adding one to cover for one extra epochLength
				reputerPruningBlock := blockHeight - (int64(MaxUnfulfilledReputerRequests+1)*topic.EpochLength + topic.GroundTruthLag)
				if reputerPruningBlock > 0 {
					sdkCtx.Logger().Warn(fmt.Sprintf("Pruning reputer nonces before block: %v for topic: %d on block: %v", reputerPruningBlock, topic.Id, blockHeight))
					am.keeper.PruneReputerNonces(sdkCtx, topic.Id, reputerPruningBlock)

					// Reputer nonces need to check worker nonces from one epoch before
					workerPruningBlock := reputerPruningBlock - topic.EpochLength
					if workerPruningBlock > 0 {
						sdkCtx.Logger().Debug("Pruning worker nonces before block: ", workerPruningBlock, " for topic: ", topic.Id)
						// Prune old worker nonces previous to current blockHeight to avoid inserting inferences after its time has passed
						am.keeper.PruneWorkerNonces(sdkCtx, topic.Id, workerPruningBlock)
					}
				}
			}
			// Check Worker Close Cadence
			if am.keeper.CheckWorkerCloseCadence(blockHeight, topic) {
				sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Worker close cadence met for topic: %v metadata: %s . \n",
					topic.Id,
					topic.Metadata))
				// Check if there is an unfulfilled nonce
				nonces, err := am.keeper.GetUnfulfilledWorkerNonces(sdkCtx, topic.Id)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error getting unfulfilled worker nonces: %s", err.Error()))
					return
				}
				for _, nonce := range nonces.Nonces {
					// Check if current blockheight exists as an open nonce
					if nonce.BlockHeight == blockHeight {
						am.keeper.CloseWorkerNonce(sdkCtx, topic.Id, *nonce)
					}
				}
			}

		}(localTopic)
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
