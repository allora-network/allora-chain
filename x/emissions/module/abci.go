package module

import (
	"context"
	"fmt"
	"sync"

	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
			// Check the cadence of inferences, and just in case also check multiples of epoch lengths
			// to avoid potential situations where the block is missed
			if (blockHeight-topic.EpochLastEnded)%topic.EpochLength == 0 {
				fmt.Printf("ABCI EndBlocker: Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
					topic.Id,
					topic.Metadata,
					topic.DefaultArg)

				// Update the last inference ran
				err = am.keeper.UpdateTopicEpochLastEnded(sdkCtx, topic.Id, blockHeight)
				if err != nil {
					fmt.Println("Error updating last inference ran: ", err)
				}
				// Add Worker Nonces
				nextNonce := emissionstypes.Nonce{BlockHeight: blockHeight + topic.EpochLength}
				err = am.keeper.AddWorkerNonce(sdkCtx, topic.Id, &nextNonce)
				if err != nil {
					fmt.Println("Error adding worker nonce: ", err)
					return
				}
				// Add Reputer Nonces
				if blockHeight-topic.EpochLength > 0 {
					ReputerReputerNonce := emissionstypes.Nonce{BlockHeight: blockHeight}
					ReputerWorkerNonce := emissionstypes.Nonce{BlockHeight: blockHeight - topic.EpochLength}
					err = am.keeper.AddReputerNonce(sdkCtx, topic.Id, &ReputerReputerNonce, &ReputerWorkerNonce)
					if err != nil {
						fmt.Println("Error adding reputer nonce: ", err)
						return
					}
				} else {
					fmt.Println("Not adding reputer nonce, too early in topic history", blockHeight, topic.EpochLength)
				}

				// To notify topic handler that the topic is ready for churn i.e. requests to be sent to workers and reputers
				err = am.keeper.AddChurnReadyTopic(ctx, topic.Id)
				if err != nil {
					fmt.Println("Error setting churn ready topic: ", err)
					return
				}
			}
		}(*topic)
		return nil
	}
	err = rewards.ApplyFuncOnAllChurnReadyTopics(
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
