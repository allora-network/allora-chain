package module

import (
	"context"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/errors"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	emistypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func EndBlocker(ctx context.Context, am AppModule) (err error) {
	defer telemetry.ModuleMeasureSince(emistypes.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)
	defer errors.Annotate(&err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Debug("---------------- Emissions EndBlock -------------------")
	blockHeight := sdkCtx.BlockHeight()

	moduleParams, err := am.keeper.GetParams(sdkCtx)
	if err != nil {
		return errors.Wrapf(err, "failed: fetch module params")
	}

	// Remove Stakers that have been wanting to unstake this block. They no longer get paid rewards
	err = RemoveStakes(sdkCtx, blockHeight, &am.keeper, moduleParams.HalfMaxProcessStakeRemovalsEndBlock)
	if err != nil {
		return errors.Wrapf(err, "failed: remove stakes")
	}

	if uint64(blockHeight)%moduleParams.BlocksPerMonth == 0 {
		err := rewards.HandleMonthlyRewardsReset(sdkCtx, am.keeper)
		if err != nil {
			return errors.Wrap(err, "failed: monthly rewards reset")
		}
	}

	err = RemoveDelegateStakes(sdkCtx, blockHeight, &am.keeper, moduleParams.HalfMaxProcessStakeRemovalsEndBlock)
	if err != nil {
		return errors.Wrapf(err, "failed: remove delegate stakes")
	}

	// Get unnormalized weights of active topics and the sum weight and revenue they have generated
	weights, sumWeight, totalRevenue, err := rewards.GetAndUpdateActiveTopicWeights(sdkCtx, am.keeper, blockHeight)
	if err != nil {
		return errors.Wrapf(err, "failed: get and update active topic weights")
	}

	sdkCtx.Logger().Debug("ABCI EndBlocker", "totalRevenue", totalRevenue, "sumWeight", sumWeight)

	err = rewards.UpdateNoncesOfActiveTopics(
		sdkCtx,
		am.keeper,
		blockHeight,
		weights,
	)
	if err != nil {
		return errors.Wrapf(err, "failed: update nonces of active topics")
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
		return errors.Wrapf(err, "failed: emit rewards")
	}

	// Close any open windows due this blockHeight
	workerWindowsToClose := am.keeper.GetWorkerWindowTopicIds(sdkCtx, blockHeight)
	if len(workerWindowsToClose.TopicIds) == 0 {
		return nil
	}
	for _, topicId := range workerWindowsToClose.TopicIds {
		sdkCtx.Logger().Info("ABCI EndBlocker: Worker close cadence met for topic", "topicId", topicId)

		// Check if there is an unfulfilled nonce
		nonces, err := am.keeper.GetUnfulfilledWorkerNonces(sdkCtx, topicId)
		if err != nil {
			return errors.WrapWithFields(err, "failed: get unfulfilled worker nonces for topic", "topic_id", topicId)
		} else if len(nonces.Nonces) == 0 {
			// No nonces to fulfill
			continue
		}

		topic, err := am.keeper.GetTopic(sdkCtx, topicId)
		if err != nil {
			return errors.WrapWithFields(err, "failed: fetch topic", "topic_id", topicId)
		}

		for _, nonce := range nonces.Nonces {
			// No need to validate blockHeight boundaries - we accept submissions until this block.
			sdkCtx.Logger().Debug("ABCI EndBlocker: closing worker window for topic", "topic_id", topicId, "nonce", nonce)
			err = actorutils.CloseWorkerNonce(&am.keeper, sdkCtx, topic, *nonce)
			if err != nil {
				return errors.WrapWithFields(err, "failed: close worker nonce", "topic_id", topicId)
			}
		}
	}

	err = am.keeper.DeleteWorkerWindowBlockHeight(sdkCtx, blockHeight)
	if err != nil {
		return errors.Wrapf(err, "failed: delete worker window blockheight")
	}
	return nil
}
