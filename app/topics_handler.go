package app

import (
	"fmt"
	"sync"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TopicsHandler struct {
	emissionsKeeper emissionskeeper.Keeper
}

func NewTopicsHandler(emissionsKeeper emissionskeeper.Keeper) *TopicsHandler {
	return &TopicsHandler{
		emissionsKeeper: emissionsKeeper,
	}
}

func (th *TopicsHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		fmt.Printf("\n ---------------- TopicsHandler ------------------- \n")
		currentTime := uint64(ctx.BlockTime().Unix())
		churnReadyTopics, err := th.emissionsKeeper.GetChurnReadyTopics(ctx)
		if err != nil {
			fmt.Println("Error getting active topics and met demand: ", err)
			return nil, err
		}

		var wg sync.WaitGroup
		// Loop over and run epochs on topics whose inferences are demanded enough to be served
		// Within each loop, execute the inference and weight cadence checks and trigger the inference and weight generation
		for _, topic := range churnReadyTopics.Topics {
			// Parallelize the inference and loss cadence checks
			wg.Add(1)
			go func(topic *emissionstypes.Topic) {
				defer wg.Done()
				// Check the cadence of inferences
				if currentTime-topic.InferenceLastRan >= topic.InferenceCadence {
					fmt.Printf("Triggering inference generation for topic: %v metadata: %s default arg: %s. \n",
						topic.Id,
						topic.Metadata,
						topic.DefaultArg)

					go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id)
				}

				// Check the cadence of weight calculations
				if currentTime-topic.LossLastRan >= topic.LossCadence {
					fmt.Printf("Triggering Weight cadence met for topic: %v metadata: %s default arg: %s \n",
						topic.Id,
						topic.Metadata,
						topic.DefaultArg)

					// Get Latest Weights
					weights, err := th.emissionsKeeper.GetWeightsFromTopic(ctx, topic.Id)
					fmt.Println("Error getting latest weights: ", err)
					if err != nil {
						return
					}

					inferences, err := th.emissionsKeeper.GetLatestInferencesFromTopic(ctx, topic.Id)
					// Get Lastest Inference
					fmt.Println("Error getting latest inferences: ", err)
					if err != nil {
						return
					}

					go generateWeights(weights, inferences, topic.LossLogic, topic.LossMethod, topic.Id)
				}
			}(topic)
		}
		wg.Wait()
		// Return the transactions as they came
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}
