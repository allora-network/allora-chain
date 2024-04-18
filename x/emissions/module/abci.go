package module

import (
	"context"
	"fmt"
	"sync"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, am AppModule) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Ensure that enough blocks have passed to hit an epoch.
	// If not, skip rewards calculation
	blockNumber := sdkCtx.BlockHeight()
	lastRewardsUpdate, err := am.keeper.GetLastRewardsUpdate(sdkCtx)
	if err != nil {
		return err
	}

	topTopicsActiveWithDemand, metDemand, err := ChurnRequestsGetActiveTopicsAndDemand(sdkCtx, am.keeper, blockNumber)
	if err != nil {
		fmt.Println("Error getting active topics and met demand: ", err)
		return err
	}
	// send collected inference request fees to the Ecosystem module account
	// they will be paid out to reputers, workers, and cosmos validators
	// according to the formulas in the beginblock of the mint module
	err = am.keeper.BankKeeper().SendCoinsFromModuleToModule(
		ctx,
		types.AlloraRequestsAccountName,
		mintTypes.EcosystemModuleName,
		sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(metDemand.BigInt().Int64()))))
	if err != nil {
		fmt.Println("Error sending coins from module to module: ", err)
		return err
	}

	blocksSinceLastUpdate := blockNumber - lastRewardsUpdate
	if blocksSinceLastUpdate < 0 {
		panic("Block number is less than last rewards update block number")
	}
	rewardCadence, err := am.keeper.GetParamsRewardCadence(ctx)
	if err != nil {
		return err
	}
	if blocksSinceLastUpdate >= rewardCadence {
		err = rewards.EmitRewards(sdkCtx, am.keeper, topTopicsActiveWithDemand)
		// the following code does NOT halt the chain in case of an error in rewards payments
		// if an error occurs and rewards payments are not made, globally they will still accumulate
		// and we can retroactively pay them out
		if err != nil {
			fmt.Println("Error calculating global emission per topic: ", err)
			panic(err)
		}
		err := am.keeper.IncrementFeeRevenueEpoch(sdkCtx)
		if err != nil {
			fmt.Println("Error incrementing fee revenue epoch: ", err)
			return err
		}
	}

	var wg sync.WaitGroup
	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	// Within each loop, execute the inference and weight cadence checks
	for _, topic := range topTopicsActiveWithDemand {
		// Parallelize the inference and weight cadence checks
		wg.Add(1)
		go func(topic types.Topic) {
			defer wg.Done()
			// Check the cadence of inferences
			if blockNumber == topic.EpochLastEnded+topic.EpochLength ||
				blockNumber-topic.EpochLastEnded >= 2*topic.EpochLength {
				fmt.Printf("Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
					topic.Id,
					topic.Metadata,
					topic.DefaultArg)

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockNumber)
				if err != nil {
					fmt.Println("Error updating last inference ran: ", err)
				}
				// Add Worker Nonces
				nextNonce := emissionstypes.Nonce{BlockHeight: blockNumber + topic.EpochLength}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					fmt.Println("Error adding worker nonce: ", err)
					return
				}
				// Add Reputer Nonces
				previousNonce := emissionstypes.Nonce{BlockHeight: blockNumber}
				err = am.keeper.AddReputerNonce(sdkCtx, topic.Id, &nextNonce, &previousNonce)
			}
		}(topic)
	}
	wg.Wait()

	return nil
}
