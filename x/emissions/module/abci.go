package module

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
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
	sdkCtx.Logger().Debug(fmt.Sprintf("EndBlocker %d: Total Revenue: %v, Sum Weight: %v", blockHeight, totalRevenue, sumWeight))

	// REWARDS (will internally filter any non-RewardReady topics)
	err = rewards.EmitRewards(sdkCtx, am.keeper, blockHeight, weights, sumWeight, totalRevenue)
	if err != nil {
		sdkCtx.Logger().Error("Error calculating global emission per topic: ", err)
		return errors.Wrapf(err, "Rewards error")
	}

	// Reset the churn ready topics
	err = am.keeper.ResetChurnableTopics(sdkCtx)
	if err != nil {
		sdkCtx.Logger().Error("Error resetting churn ready topics: ", err)
		return errors.Wrapf(err, "Resetting churn ready topics error")
	}

	// identify which topics are churnable from the list of active topics
	err = rewards.PickChurnableActiveTopics(
		sdkCtx,
		am.keeper,
		blockHeight,
		weights,
	)
	if err != nil {
		sdkCtx.Logger().Error("Error applying function on all rewardable topics: ", err)
		return err
	}

	return nil
}
