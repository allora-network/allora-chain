package module

import (
	"context"
	"fmt"
	"sync"

	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, am AppModule) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	params, err := am.keeper.GetParams(ctx)
	if err != nil {
		return err
	}

	err = rewards.EmitRewards(sdkCtx, am.keeper, blockHeight)
	if err != nil {
		fmt.Println("Error calculating global emission per topic: ", err)
		panic(err)
	}

	var wg sync.WaitGroup
	// Loop over and run epochs on topics whose inferences are demanded enough to be served
	// Within each loop, execute the inference and weight cadence checks
	fn := func(ctx context.Context, topic *types.Topic) error {
		// Parallelize the inference and weight cadence checks
		wg.Add(1)
		go func(topic types.Topic) {
			defer wg.Done()
			// Check the cadence of inferences
			if blockHeight == topic.EpochLastEnded+topic.EpochLength ||
				blockHeight-topic.EpochLastEnded >= 2*topic.EpochLength {
				fmt.Printf("Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
					topic.Id,
					topic.Metadata,
					topic.DefaultArg)

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockHeight)
				if err != nil {
					fmt.Println("Error updating last inference ran: ", err)
				}
				// Add Worker Nonces
				nextNonce := types.Nonce{BlockHeight: blockHeight + topic.EpochLength}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					fmt.Println("Error adding worker nonce: ", err)
					return
				}
				// Add Reputer Nonces
				if blockHeight-topic.EpochLength > 0 {
					ReputerReputerNonce := types.Nonce{BlockHeight: blockHeight}
					ReputerWorkerNonce := types.Nonce{BlockHeight: blockHeight - topic.EpochLength}
					err = am.keeper.AddReputerNonce(sdkCtx, topic.Id, &ReputerReputerNonce, &ReputerWorkerNonce)
					if err != nil {
						fmt.Println("Error adding reputer nonce: ", err)
						return
					}
				} else {
					fmt.Println("Not adding reputer nonce, too early in topic history", blockHeight, topic.EpochLength)
				}

			}
		}(*topic)
		return nil
	}
	err = rewards.SafeApplyFuncOnAllRewardReadyTopics(
		sdkCtx,
		am.keeper,
		blockHeight,
		fn,
		params.TopicPageLimit,
		params.MaxTopicPages,
	)
	if err != nil {
		fmt.Println("Error applying function on all reward ready topics: ", err)
		return err
	}
	wg.Wait()

	return nil
}
