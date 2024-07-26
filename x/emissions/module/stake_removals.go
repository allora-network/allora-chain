package module

import (
	"fmt"

	chainParams "github.com/allora-network/allora-chain/app/params"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Remove all stakes this block that have been marked for removal
func RemoveStakes(
	sdkCtx sdk.Context,
	currentBlock int64,
	k emissionskeeper.Keeper,
	limitToProcess uint64,
) {
	removals, limitHit, err := k.GetStakeRemovalsUpUntilBlock(sdkCtx, currentBlock, limitToProcess)
	if err != nil {
		sdkCtx.Logger().Error(fmt.Sprintf(
			"Unable to get stake removals for block %d, skipping removing stakes: %v",
			currentBlock,
			err,
		))
		return
	}
	if limitHit {
		sdkCtx.Logger().Info(fmt.Sprintf(
			"Hit limit of number of stake removals we can process up until block %d, only removing %d",
			currentBlock,
			limitToProcess,
		))
	}
	for _, stakeRemoval := range removals {
		// attempt writes in a cache context, only write finally if there are no errors
		cacheSdkCtx, write := sdkCtx.CacheContext()

		// Update the stake data structures
		err = k.RemoveReputerStake(
			cacheSdkCtx,
			currentBlock,
			stakeRemoval.TopicId,
			stakeRemoval.Reputer,
			stakeRemoval.Amount,
		)
		if err != nil {
			sdkCtx.Logger().Error(fmt.Sprintf(
				"Error removing stake data structures: %v | %v",
				stakeRemoval,
				err,
			))
			continue
		}

		// do no checking that the stake removal struct is valid. In order to have a stake removal
		// it would have had to be created in msgServer.RemoveStake which would have done
		// validation of validity up front before scheduling the delay

		// Check the module has enough funds to send back to the sender
		// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// Send the funds
		coins := sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, stakeRemoval.Amount))
		err = k.SendCoinsFromModuleToAccount(
			cacheSdkCtx,
			emissionstypes.AlloraStakingAccountName,
			stakeRemoval.Reputer,
			coins,
		)
		if err != nil {
			sdkCtx.Logger().Error(fmt.Sprintf(
				"Error removing stake funds: %v | %v",
				stakeRemoval,
				err,
			))
			continue
		}

		// if there were no errors up to this point, then the removal should be safe to do,
		// and therefore we can write the cache to the main state
		write()
	}
}

// remove all delegated stakes that have been marked for removal this block
func RemoveDelegateStakes(
	sdkCtx sdk.Context,
	currentBlock int64,
	k emissionskeeper.Keeper,
	limitToProcess uint64,
) {
	removals, limitHit, err := k.GetDelegateStakeRemovalsUpUntilBlock(sdkCtx, currentBlock, limitToProcess)
	if err != nil {
		sdkCtx.Logger().Error(
			fmt.Sprintf(
				"Unable to get stake removals for block %d, skipping removing stakes: %v",
				currentBlock,
				err,
			))
		return
	}
	if limitHit {
		sdkCtx.Logger().Info(fmt.Sprintf(
			"Hit limit of number of stake removals we can process up until block %d, only removing %d",
			currentBlock,
			limitToProcess,
		))
	}
	for _, stakeRemoval := range removals {
		// attempt writes in a cache context, only write finally if there are no errors
		cacheSdkCtx, write := sdkCtx.CacheContext()

		// Update the stake data structures
		err = k.RemoveDelegateStake(
			cacheSdkCtx,
			currentBlock,
			stakeRemoval.TopicId,
			stakeRemoval.Delegator,
			stakeRemoval.Reputer,
			stakeRemoval.Amount,
		)
		if err != nil {
			sdkCtx.Logger().Error(fmt.Sprintf(
				"Error removing delegate stake state: %v | %v",
				stakeRemoval,
				err,
			))
			continue
		}

		// do no checking that the stake removal struct is valid. In order to have a stake removal
		// it would have had to be created in msgServer.RemoveDelegateStake which would have done
		// validation of validity up front before scheduling the delay

		// Check the module has enough funds to send back to the sender
		// Bank module does this for us in module SendCoins / subUnlockedCoins so we don't need to check
		// Send the funds
		coins := sdk.NewCoins(sdk.NewCoin(chainParams.DefaultBondDenom, stakeRemoval.Amount))
		err = k.SendCoinsFromModuleToAccount(
			cacheSdkCtx,
			emissionstypes.AlloraStakingAccountName,
			stakeRemoval.Delegator, coins)
		if err != nil {
			sdkCtx.Logger().Error(fmt.Sprintf(
				"Error removing delegate stake send funds: %v | %v",
				stakeRemoval,
				err,
			))
			continue
		}

		write()
	}
}
