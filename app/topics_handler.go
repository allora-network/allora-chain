package app

import (
	"fmt"
	"sync"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions"
	emissionsmodule "github.com/allora-network/allora-chain/x/emissions/module"
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
		currentTime := uint64(ctx.BlockTime().Unix())
		topTopicsActiveWithDemand, _, err := emissionsmodule.ChurnRequestsGetActiveTopicsAndDemand(ctx, th.emissionsKeeper, currentTime)
		if err != nil {
			fmt.Println("Error getting active topics and met demand: ", err)
			return nil, err
		}

		var wg sync.WaitGroup
		// Loop over and run epochs on topics whose inferences are demanded enough to be served
		// Within each loop, execute the inference and weight cadence checks
		for _, topic := range topTopicsActiveWithDemand {
			// Parallelize the inference and weight cadence checks
			wg.Add(1)
			go func(topic emissionstypes.Topic) {
				defer wg.Done()
				// Check the cadence of inferences
				if currentTime-topic.InferenceLastRan >= topic.InferenceCadence {
					fmt.Printf("Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
						topic.Id,
						topic.Metadata,
						topic.DefaultArg)

					go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id)

					// Update the last inference ran
					err := th.emissionsKeeper.UpdateTopicInferenceLastRan(ctx, topic.Id, currentTime)
					if err != nil {
						fmt.Println("Error updating last inference ran: ", err)
					}
				}

				// Check the cadence of weight calculations
				if currentTime-topic.WeightLastRan >= topic.WeightCadence {
					fmt.Printf("Weight cadence met for topic: %v metadata: %s default arg: %s \n", topic.Id, topic.Metadata, topic.DefaultArg)
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

					go generateWeights(weights, inferences, topic.WeightLogic, topic.WeightMethod, topic.Id)

					// Update the last weight ran
					err = th.emissionsKeeper.UpdateTopicWeightLastRan(ctx, topic.Id, currentTime)
					if err != nil {
						fmt.Println("Error updating last weight ran: ", err)
					}
				}
			}(topic)
		}
		wg.Wait()
		return &abci.ResponsePrepareProposal{Txs: [][]byte{}}, nil
	}
}
