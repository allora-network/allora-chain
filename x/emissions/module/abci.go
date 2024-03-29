package module

import (
	"context"
	"fmt"
	"sync"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BeginBlocker(ctx context.Context, am AppModule) error {
	percentRewardsToReputersAndWorkers, err := am.keeper.GetParamsPercentRewardsReputersWorkers(ctx)
	if err != nil {
		return err
	}
	feeCollectorAddress := am.keeper.AccountKeeper().GetModuleAddress(am.keeper.GetFeeCollectorName())
	feesCollectedAndEmissionsMintedLastBlock := am.keeper.BankKeeper().GetBalance(ctx, feeCollectorAddress, params.DefaultBondDenom)
	reputerWorkerCut := cosmosMath.
		NewInt(int64(percentRewardsToReputersAndWorkers * Precision)).
		Mul(feesCollectedAndEmissionsMintedLastBlock.Amount).
		Quo(cosmosMath.NewInt(Precision))
	am.keeper.BankKeeper().SendCoinsFromModuleToModule(
		ctx,
		am.keeper.GetFeeCollectorName(),
		types.AlloraRewardsAccountName,
		sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, reputerWorkerCut)),
	)

	return nil
}

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
	// send collected inference request fees to the fee collector account
	// they will be paid out to reputers, workers, and cosmos validators
	// in the following BeginBlock of the next block
	err = am.keeper.BankKeeper().SendCoinsFromModuleToModule(
		ctx,
		types.AlloraRequestsAccountName,
		am.keeper.GetFeeCollectorName(),
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
		// err = emitRewards(sdkCtx, am)
		// the following code does NOT halt the chain in case of an error in rewards payments
		// if an error occurs and rewards payments are not made, globally they will still accumulate
		// and we can retroactively pay them out
		if err != nil {
			fmt.Println("Error calculating global emission per topic: ", err)
			panic(err)
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
			if blockNumber-topic.EpochLastEnded >= topic.EpochLength {
				fmt.Printf("Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
					topic.Id,
					topic.Metadata,
					topic.DefaultArg)

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockNumber)
				if err != nil {
					fmt.Println("Error updating last inference ran: ", err)
				}
			}
		}(topic)
	}
	wg.Wait()

	return nil
}
