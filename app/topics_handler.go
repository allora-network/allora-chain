package app

import (
	// "fmt"
	// "sync"

	"fmt"
	"sync"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"

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
		blockHeight := ctx.BlockHeight()
		nonce := emissionstypes.Nonce{Nonce: blockHeight}
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

				//
				// TODO  Add check whether the cadence is right - unclear about these new value changes
				//
				if topic.EpochLastEnded >= blockHeight {
					// Check the cadence of inferences
					fmt.Printf("Triggering inference generation for topic: %v metadata: %s default arg: %s. \n",
						topic.Id,
						topic.Metadata,
						topic.DefaultArg)
					go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id, nonce)
				}

				// TODO Check - is this the right cadence?
				if topic.EpochLastEnded >= blockHeight {
					// Check the cadence of weight calculations
					fmt.Printf("Triggering Weight cadence met for topic: %v metadata: %s default arg: %s \n",
						topic.Id,
						topic.Metadata,
						topic.DefaultArg)

					// TODO: We don't want just the latest inferences but the ValueBundle (I_i) instead
					inferences, err := th.emissionsKeeper.GetLatestInferencesFromTopic(ctx, topic.Id)
					// Get Lastest Inference
					fmt.Println("Error getting latest inferences: ", err)
					if err != nil {
						return
					}
					go generateLosses(inferences, topic.LossLogic, topic.LossMethod, topic.Id, nonce, currentTime)
				}

			}(topic)
		}
		wg.Wait()
		// Return the transactions as they came
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}
