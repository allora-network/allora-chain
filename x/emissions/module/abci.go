package module

import (
	"context"

	"cosmossdk.io/errors"
	allorautils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, am AppModule) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	sdkCtx.Logger().Debug("---------------- Emissions EndBlock -------------------", "blockHeight", blockHeight)

	moduleParams, err := am.keeper.GetParams(sdkCtx)
	if err != nil {
		sdkCtx.Logger().Error("Error Getting module params", err)
		return err
	}

	// Remove Stakers that have been wanting to unstake this block. They no longer get paid rewards
	err = RemoveStakes(sdkCtx, blockHeight, &am.keeper, moduleParams.HalfMaxProcessStakeRemovalsEndBlock)
	if err != nil {
		sdkCtx.Logger().Error("Error removing stakes: ", err)
	}
	err = RemoveDelegateStakes(sdkCtx, blockHeight, &am.keeper, moduleParams.HalfMaxProcessStakeRemovalsEndBlock)
	if err != nil {
		sdkCtx.Logger().Error("Error removing delegate stakes: ", err)
	}

	// Get unnormalized weights of active topics and the sum weight and revenue they have generated
	weights, sumWeight, totalRevenue, err := rewards.GetAndUpdateActiveTopicWeights(sdkCtx, am.keeper, blockHeight)
	if err != nil {
		return errors.Wrapf(err, "Weights error")
	}

	sdkCtx.Logger().Debug("ABCI EndBlocker", "blockHeight", blockHeight, "totalRevenue", totalRevenue, "sumWeight", sumWeight)

	err = rewards.UpdateNoncesOfActiveTopics(
		sdkCtx,
		am.keeper,
		blockHeight,
		weights,
	)
	if err != nil {
		sdkCtx.Logger().Error("Error applying function on all rewardable topics: ", err)
		return err
	}

	// REWARDS (will internally filter any non-RewardReady topics)
	err = rewards.EmitRewards(rewards.EmitRewardsArgs{
		Ctx:          sdkCtx,
		K:            am.keeper,
		ModuleParams: moduleParams,
		BlockHeight:  blockHeight,
		Weights:      weights,
		SumWeight:    sumWeight,
		TotalRevenue: totalRevenue,
	})
	if err != nil {
		sdkCtx.Logger().Error("Error calculating global emission per topic: ", err)
		return errors.Wrapf(err, "Rewards error")
	}

	// Close any open windows due this blockHeight
	workerWindowsToClose := am.keeper.GetWorkerWindowTopicIds(sdkCtx, blockHeight)
	if len(workerWindowsToClose.TopicIds) > 0 {
		for _, topicId := range workerWindowsToClose.TopicIds {
			sdkCtx.Logger().Info("ABCI EndBlocker: Worker close cadence met for topic", "topicId", topicId)
			// Check if there is an unfulfilled nonce
			nonces, err := am.keeper.GetUnfulfilledWorkerNonces(sdkCtx, topicId)
			if err != nil {
				sdkCtx.Logger().Warn("Error getting unfulfilled worker nonces", "error", err)
				continue
			} else if len(nonces.Nonces) == 0 {
				// No nonces to fulfill
				continue
			} else {
				topic, err := am.keeper.GetTopic(sdkCtx, topicId)
				if err != nil {
					sdkCtx.Logger().Warn("Error getting topic", "error", err)
					continue
				}
				for _, nonce := range nonces.Nonces {
					// No need to validate blockHeight boundaries - we accept submissions until this block.
					sdkCtx.Logger().Debug("ABCI EndBlocker", "blockHeight", blockHeight, "closing worker window for topic", "topicId", topicId, "nonce", nonce)
					err = allorautils.CloseWorkerNonce(&am.keeper, sdkCtx, topic, *nonce)
					if err != nil {
						sdkCtx.Logger().Info("Error closing worker nonce", "error", err)
					}
				}
			}
		}
		err = am.keeper.DeleteWorkerWindowBlockHeight(sdkCtx, blockHeight)
		if err != nil {
			sdkCtx.Logger().Warn("Error deleting worker window blockheight", "error", err)
		}
	}

	return nil
}
