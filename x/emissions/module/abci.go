package module

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/errors"
	allorautils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
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
	moduleParams, err := am.keeper.GetParams(sdkCtx)
	if err != nil {
		sdkCtx.Logger().Error("Error Getting module params", err)
		return err
	}
	// Remove Stakers that have been wanting to unstake this block. They no longer get paid rewards
	RemoveStakes(sdkCtx, blockHeight, am.keeper, moduleParams.HalfMaxProcessStakeRemovalsEndBlock)
	RemoveDelegateStakes(sdkCtx, blockHeight, am.keeper, moduleParams.HalfMaxProcessStakeRemovalsEndBlock)

	// Get unnormalized weights of active topics and the sum weight and revenue they have generated
	weights, sumWeight, totalRevenue, err := rewards.GetAndUpdateActiveTopicWeights(sdkCtx, am.keeper, blockHeight)
	if err != nil {
		return errors.Wrapf(err, "Weights error")
	}
	sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker %d: Total Revenue: %v, Sum Weight: %v", blockHeight, totalRevenue, sumWeight))

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
				sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Worker open cadence met for topic: %v metadata: %s . \n",
					topic.Id,
					topic.Metadata))

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockHeight)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error updating last inference ran: %s", err.Error()))
					return
				}
				// Add Worker Nonces
				nextNonce := types.Nonce{BlockHeight: blockHeight}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error adding worker nonce: %s", err.Error()))
					return
				}
				sdkCtx.Logger().Debug(fmt.Sprintf("Added worker nonce for topic %d: %v \n", topic.Id, nextNonce.BlockHeight))

				err = am.keeper.AddWorkerWindowTopicId(sdkCtx, blockHeight+topic.WorkerSubmissionWindow, types.Topicid{TopicId: topic.Id})
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error adding worker window topic id: %s", err.Error()))
					return
				}

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
					sdkCtx.Logger().Debug(fmt.Sprintf("Pruning reputer nonces before block: %v for topic: %d on block: %v", reputerPruningBlock, topic.Id, blockHeight))
					err = am.keeper.PruneReputerNonces(sdkCtx, topic.Id, reputerPruningBlock)
					if err != nil {
						sdkCtx.Logger().Warn(fmt.Sprintf("Error pruning reputer nonces: %s", err.Error()))
					}

					// Reputer nonces need to check worker nonces from one epoch before
					workerPruningBlock := reputerPruningBlock - topic.EpochLength
					if workerPruningBlock > 0 {
						sdkCtx.Logger().Debug("Pruning worker nonces before block: ", workerPruningBlock, " for topic: ", topic.Id)
						// Prune old worker nonces previous to current blockHeight to avoid inserting inferences after its time has passed
						err = am.keeper.PruneWorkerNonces(sdkCtx, topic.Id, workerPruningBlock)
						if err != nil {
							sdkCtx.Logger().Warn(fmt.Sprintf("Error pruning worker nonces: %s", err.Error()))
						}
					}
				}
			}
			// Check Reputer Close Cadence
			if am.keeper.CheckReputerCloseCadence(blockHeight, topic) {
				// Check if there is an unfulfilled nonce
				nonces, err := am.keeper.GetUnfulfilledReputerNonces(sdkCtx, topic.Id)
				if err != nil {
					sdkCtx.Logger().Warn(fmt.Sprintf("Error getting unfulfilled worker nonces: %s", err.Error()))
					return
				}
				for _, nonce := range nonces.Nonces {
					// Check if current blockheight has reached the blockheight of the nonce + groundTruthLag + epochLength
					// This means one epochLength is allowed for reputation responses to be sent since ground truth is revealed.
					closingReputerNonceMinBlockHeight := nonce.ReputerNonce.BlockHeight + topic.GroundTruthLag + topic.EpochLength
					if blockHeight >= closingReputerNonceMinBlockHeight {
						sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Closing reputer nonce for topic: %v nonce: %v, min: %d. \n",
							topic.Id, nonce, closingReputerNonceMinBlockHeight))
						err = allorautils.CloseReputerNonce(&am.keeper, sdkCtx, topic.Id, *nonce.ReputerNonce)
						if err != nil {
							sdkCtx.Logger().Warn(fmt.Sprintf("Error closing reputer nonce: %s", err.Error()))
							// Proactively close the nonce to avoid
							am.keeper.FulfillReputerNonce(sdkCtx, topic.Id, nonce.ReputerNonce)
						}
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

	// Close any open windows due this blockHeight
	workerWindowsToClose := am.keeper.GetWorkerWindowTopicIds(sdkCtx, blockHeight)
	if len(workerWindowsToClose.TopicIds) > 0 {
		for _, topicId := range workerWindowsToClose.TopicIds {
			sdkCtx.Logger().Info(fmt.Sprintf("ABCI EndBlocker: Worker close cadence met for topic: %d", topicId))
			// Check if there is an unfulfilled nonce
			nonces, err := am.keeper.GetUnfulfilledWorkerNonces(sdkCtx, topicId.TopicId)
			if err != nil {
				sdkCtx.Logger().Warn(fmt.Sprintf("Error getting unfulfilled worker nonces: %s", err.Error()))
				continue
			} else if len(nonces.Nonces) == 0 {
				// No nonces to fulfill
				continue
			} else {
				for _, nonce := range nonces.Nonces {
					sdkCtx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker %d: Closing Worker window for topic: %d, nonce: %v", blockHeight, topicId.TopicId, nonce))
					err := allorautils.CloseWorkerNonce(&am.keeper, sdkCtx, topicId.TopicId, *nonce)
					if err != nil {
						sdkCtx.Logger().Info(fmt.Sprintf("Error closing worker nonce, proactively fulfilling: %s", err.Error()))
						// Proactively close the nonce
						fulfilledNonce, err := am.keeper.FulfillWorkerNonce(sdkCtx, topicId.TopicId, nonce)
						if err != nil {
							sdkCtx.Logger().Warn(fmt.Sprintf("Error fulfilling worker nonce: %s", err.Error()))
						} else {
							sdkCtx.Logger().Debug(fmt.Sprintf("Fulfilled: %t, worker nonce: %v", fulfilledNonce, nonce))
						}
					}
				}
			}
		}
		err = am.keeper.DeleteWorkerWindowBlockheight(sdkCtx, blockHeight)
		if err != nil {
			sdkCtx.Logger().Warn(fmt.Sprintf("Error deleting worker window blockheight: %s", err.Error()))
		}
	}
	return nil
}
