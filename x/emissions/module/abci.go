package module

import (
	"context"
	"sort"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	allorautils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
)

func EndBlocker(ctx context.Context, am AppModule) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	sdkCtx.Logger().Debug("---------------- Emissions EndBlock -------------------", "blockHeight", blockHeight)

	moduleParams, err := am.keeper.GetParamsKeeper().GetParams(sdkCtx)
	if err != nil {
		sdkCtx.Logger().Error("Error Getting module params", err)
		return err
	}

	// Remove Stakers that have been wanting to unstake this block. They no longer get paid rewards
	err = RemoveStakes(
		sdkCtx,
		blockHeight,
		am.keeper.GetStakingKeeper().GetStakeRemovalsUpUntilBlock,
		am.keeper.GetStakingKeeper().RemoveReputerStake,
		am.keeper.GetBankingKeeper().SendCoinsFromModuleToAccount,
		moduleParams.HalfMaxProcessStakeRemovalsEndBlock,
	)
	if err != nil {
		sdkCtx.Logger().Error("Error removing stakes: ", err)
	}
	err = RemoveDelegateStakes(
		sdkCtx,
		blockHeight,
		am.keeper.GetStakingKeeper().GetDelegateStakeRemovalsUpUntilBlock,
		am.keeper.GetStakingKeeper().RemoveDelegateStake,
		am.keeper.GetBankingKeeper().SendCoinsFromModuleToAccount,
		moduleParams.HalfMaxProcessStakeRemovalsEndBlock,
	)
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

	// Close worker windows before rewards so a rewards error cannot skip closes.
	// Unfulfilled nonces are stored newest-first; close oldest first so a leftover
	// nonce cannot reset the active set before the nonce that still has submissions.
	workerWindowsToClose := am.keeper.GetNonceKeeper().GetWorkerWindowTopicIds(sdkCtx, blockHeight)
	if len(workerWindowsToClose.TopicIds) > 0 {
		for _, topicId := range workerWindowsToClose.TopicIds {
			sdkCtx.Logger().Info("ABCI EndBlocker: Worker close cadence met for topic", "topicId", topicId)
			nonces, err := am.keeper.GetNonceKeeper().GetUnfulfilledWorkerNonces(sdkCtx, topicId)
			if err != nil {
				sdkCtx.Logger().Warn("Error getting unfulfilled worker nonces", "error", err)
				continue
			} else if len(nonces.Nonces) == 0 {
				continue
			} else {
				topic, err := am.keeper.GetTopicKeeper().GetTopic(sdkCtx, topicId)
				if err != nil {
					sdkCtx.Logger().Warn("Error getting topic", "error", err)
					continue
				}
				sort.Slice(nonces.Nonces, func(i, j int) bool {
					if nonces.Nonces[i] == nil || nonces.Nonces[j] == nil {
						return nonces.Nonces[j] == nil
					}
					return nonces.Nonces[i].BlockHeight < nonces.Nonces[j].BlockHeight
				})
				for _, nonce := range nonces.Nonces {
					sdkCtx.Logger().Debug("ABCI EndBlocker", "blockHeight", blockHeight, "closing worker window for topic", "topicId", topicId, "nonce", nonce)
					err = allorautils.CloseWorkerNonce(&am.keeper, sdkCtx, topic, *nonce)
					if err != nil {
						sdkCtx.Logger().Info("Error closing worker nonce", "error", err)
					}
				}
			}
		}
		err = am.keeper.GetNonceKeeper().DeleteWorkerWindowBlockHeight(sdkCtx, blockHeight)
		if err != nil {
			sdkCtx.Logger().Warn("Error deleting worker window blockheight", "error", err)
		}
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

	return nil
}
