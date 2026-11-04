package module

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	chainParams "github.com/allora-network/allora-chain/app/params"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type GetStakeRemovalsUpUntilBlockFn func(
	ctx context.Context,
	blockHeight emissionstypes.BlockHeight,
	limit uint64,
) (ret []emissionstypes.StakeRemovalInfo, anyLeft bool, err error)

type RemoveReputerStakeFn func(
	ctx context.Context,
	blockHeight emissionstypes.BlockHeight,
	topicId emissionskeeper.TopicId,
	reputer emissionskeeper.ActorId,
	stakeToRemove cosmosMath.Int,
) error

type SendCoinsFromModuleToAccountFn func(
	ctx context.Context,
	senderModule string,
	recipient emissionskeeper.ActorId,
	amt sdk.Coins,
) error

// Remove all stakes this block that have been marked for removal
func RemoveStakes(
	sdkCtx sdk.Context,
	currentBlock int64,
	getStakeRemovalsUpUntilBlock GetStakeRemovalsUpUntilBlockFn,
	removeReputerStake RemoveReputerStakeFn,
	sendCoinsFromModuleToAccount SendCoinsFromModuleToAccountFn,
	limitToProcess uint64,
) error {
	removals, limitHit, err := getStakeRemovalsUpUntilBlock(sdkCtx, currentBlock, limitToProcess)
	if err != nil {
		return errors.Wrapf(err, "Unable to get stake removals for block %d", currentBlock)
	}
	if limitHit {
		sdkCtx.Logger().Warn(
			"Hit limit of number of stake removals we can process in this block",
			"blockHeight", currentBlock,
			"limitToProcess", limitToProcess,
		)
	}
	for _, stakeRemoval := range removals {
		// attempt writes in a cache context, only write finally if there are no errors
		cacheSdkCtx, write := sdkCtx.CacheContext()

		// Update the stake data structures
		err = removeReputerStake(
			cacheSdkCtx,
			stakeRemoval.BlockRemovalCompleted,
			stakeRemoval.TopicId,
			stakeRemoval.Reputer,
			stakeRemoval.Amount,
		)
		if err != nil {
			sdkCtx.Logger().Error("Error removing stake data structures", "stakeRemoval", stakeRemoval, "error", err)
			continue
		}

		// do no checking that the stake removal struct is valid. In order to have a stake removal
		// it would have had to be created in msgServer.RemoveStake which would have done
		// validation of validity up front before scheduling the delay

		// Check the module has enough funds to send back to the sender
		// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// Send the funds
		coins := sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, stakeRemoval.Amount))
		err = sendCoinsFromModuleToAccount(
			cacheSdkCtx,
			emissionstypes.AlloraStakingAccountName,
			stakeRemoval.Reputer,
			coins,
		)
		if err != nil {
			sdkCtx.Logger().Error("Error removing stake funds", "stakeRemoval", stakeRemoval, "error", err)
			continue
		}

		// if there were no errors up to this point, then the removal should be safe to do,
		// and therefore we can write the cache to the main state
		write()
	}

	return nil
}

type RemoveDelegateStakeFn func(
	ctx context.Context,
	stakeRemovalBlockHeight emissionstypes.BlockHeight,
	topicId emissionskeeper.TopicId,
	delegator, reputer emissionskeeper.ActorId,
	stakeToRemove cosmosMath.Int,
) error

type GetDelegateStakeRemovalsUpUntilBlockFn func(
	ctx context.Context,
	blockHeight emissionstypes.BlockHeight,
	limit uint64,
) (ret []emissionstypes.DelegateStakeRemovalInfo, limitHit bool, err error)

// remove all delegated stakes that have been marked for removal this block
func RemoveDelegateStakes(
	sdkCtx sdk.Context,
	currentBlock int64,
	getDelegateStakeRemovalsUpUntilBlock GetDelegateStakeRemovalsUpUntilBlockFn,
	removeDelegateStake RemoveDelegateStakeFn,
	sendCoinsFromModuleToAccount SendCoinsFromModuleToAccountFn,
	limitToProcess uint64,
) error {
	removals, limitHit, err := getDelegateStakeRemovalsUpUntilBlock(sdkCtx, currentBlock, limitToProcess)
	if err != nil {
		return errors.Wrapf(err, "Unable to get stake removals for block %d", currentBlock)
	}
	if limitHit {
		sdkCtx.Logger().Warn(
			"Hit limit of number of stake removals we can process in this block",
			"blockHeight", currentBlock,
			"limitToProcess", limitToProcess,
		)
	}
	for _, stakeRemoval := range removals {
		// attempt writes in a cache context, only write finally if there are no errors
		cacheSdkCtx, write := sdkCtx.CacheContext()

		// Update the stake data structures
		err = removeDelegateStake(
			cacheSdkCtx,
			stakeRemoval.BlockRemovalCompleted,
			stakeRemoval.TopicId,
			stakeRemoval.Delegator,
			stakeRemoval.Reputer,
			stakeRemoval.Amount,
		)
		if err != nil {
			sdkCtx.Logger().Error("Error removing delegate stake state", "stakeRemoval", stakeRemoval, "error", err)
			continue
		}

		// do no checking that the stake removal struct is valid. In order to have a stake removal
		// it would have had to be created in msgServer.RemoveDelegateStake which would have done
		// validation of validity up front before scheduling the delay

		// Check the module has enough funds to send back to the sender
		// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// Send the funds
		coins := sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, stakeRemoval.Amount))
		err = sendCoinsFromModuleToAccount(
			cacheSdkCtx,
			emissionstypes.AlloraStakingAccountName,
			stakeRemoval.Delegator, coins)
		if err != nil {
			sdkCtx.Logger().Error("Error removing delegate stake send funds", "stakeRemoval", stakeRemoval, "error", err)
			continue
		}

		write()
	}

	return nil
}
