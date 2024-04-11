package app

import (
	"fmt"
	"sync"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"

	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"

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
		currentBlockHeight := ctx.BlockHeight()
		currentNonce := emissionstypes.Nonce{Nonce: currentBlockHeight}

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
				// Get previous topic height to repute
				previousBlockHeight := currentBlockHeight - topic.EpochLength
				if previousBlockHeight <= 0 {
					return
				}
				// Check if the inference and loss cadence is met, then run inf and loss generation
				if currentBlockHeight >= topic.EpochLastEnded+topic.EpochLength {
					fmt.Printf("Triggering inference generation for topic: %v metadata: %s default arg: %s. \n",
						topic.Id, topic.Metadata, topic.DefaultArg)
					fmt.Printf("")
					go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id, currentNonce)

					fmt.Printf("Triggering Losses cadence met for topic: %v metadata: %s default arg: %s \n",
						topic.Id, topic.Metadata, topic.DefaultArg)
					// We don't want just the latest inferences but the ValueBundle (I_i) instead
					currentTime := uint64(ctx.BlockTime().Unix())
					// Get from previous blockHeight
					inferences, _, err := synth.GetNetworkInferencesAtBlock(ctx, th.emissionsKeeper, topic.Id, previousBlockHeight)
					fmt.Println("Error getting latest inferences: ", err)
					if err != nil {
						return
					}
					go generateLosses(inferences, topic.LossLogic, topic.LossMethod, topic.Id, currentNonce, currentTime)
				} else {
					fmt.Println("Inference and Losses cadence not met for topic: ", topic.Id, "block height: ", currentBlockHeight, "epoch length: ", topic.EpochLength, "last ended: ", topic.EpochLastEnded)
				}

			}(topic)
		}
		wg.Wait()
		// Return the transactions as they came
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}
