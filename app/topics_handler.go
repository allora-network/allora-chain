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

const approximateTimePerBlockSeconds = 5

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
		currentNonce := emissionstypes.Nonce{BlockHeight: currentBlockHeight}

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

				// Check if the inference and loss cadence is met, then run inf and loss generation
				if currentBlockHeight == topic.EpochLastEnded+topic.EpochLength ||
					currentBlockHeight-topic.EpochLastEnded > 2*topic.EpochLength {
					// WORKER
					fmt.Printf("Triggering inference generation for topic: %v metadata: %s default arg: %s. \n",
						topic.Id, topic.Metadata, topic.DefaultArg)

					workerNonces, err := th.emissionsKeeper.GetUnfulfilledWorkerNonces(ctx, topic.Id)
					if err != nil {
						fmt.Println("Error getting worker nonces: ", err)
						return
					}
					// iterate over all the worker nonces to find if this is unfulfilled
					for _, nonce := range workerNonces.Nonces {
						if nonce.BlockHeight == currentBlockHeight {
							fmt.Println("Current Worker block height has been found unfulfilled, requesting inferences ", currentNonce)
							go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id, currentNonce)
						}
					}

					// REPUTER
					// Get previous topic height to repute
					previousBlockHeight := topic.EpochLastEnded
					if previousBlockHeight < 0 {
						fmt.Println("Previous block height is less than 0, skipping")
						return
					}
					previousToPreviousBlockHeight := previousBlockHeight - topic.EpochLength
					if previousBlockHeight < 0 {
						fmt.Println("Previous to previous block height is less than 0, skipping")
						return
					} else {
						fmt.Println("Previous block height: ", previousBlockHeight, "Previous to previous block height: ", previousToPreviousBlockHeight)
					}
					fmt.Printf("Triggering Losses cadence met for topic: %v metadata: %s default arg: %s \n",
						topic.Id, topic.Metadata, topic.DefaultArg)
					reputerNonces, err := th.emissionsKeeper.GetUnfulfilledReputerNonces(ctx, topic.Id)
					if err != nil {
						fmt.Println("Error getting reputer nonces: ", err)
						return
					}

					// iterate over all the worker nonces to find if this is unfulfilled
					for _, nonce := range reputerNonces.Nonces {
						if nonce.ReputerNonce.BlockHeight == previousBlockHeight &&
							nonce.WorkerNonce.BlockHeight == previousToPreviousBlockHeight {
							fmt.Println("Current Reputer block height has been found unfulfilled, requesting reputers for block ", previousBlockHeight)
							reputerValueBundle, inferencesBlockHeight, err := synth.GetNetworkInferencesAtBlock(ctx, th.emissionsKeeper, topic.Id, previousBlockHeight)
							if err != nil {
								fmt.Println("Error getting latest inferences at block: ", previousBlockHeight, ", error: ", err)
								continue
							}
							if reputerValueBundle == nil {
								fmt.Println("Reputer value bundle is nil, skipping")
								continue
							}
							// Get approximated time of the previous block
							// Get difference of blocks from current to previous block
							blockDifference := currentBlockHeight - inferencesBlockHeight
							previousBlockApproxTime := uint64(ctx.BlockTime().Unix() - (blockDifference * approximateTimePerBlockSeconds))

							reputerNonce := emissionstypes.Nonce{BlockHeight: previousBlockHeight}
							workerNonce := emissionstypes.Nonce{BlockHeight: previousToPreviousBlockHeight}
							// print the request of loss generation
							fmt.Println("Requesting losses for topic: ", topic.Id, "reputer nonce: ", reputerNonce, "worker nonce: ", workerNonce, "previous block approx time: ", previousBlockApproxTime)
							go generateLosses(reputerValueBundle, topic.LossLogic, topic.LossMethod, topic.Id, reputerNonce, workerNonce, previousBlockApproxTime)
						} else {
							fmt.Println("Reputer nonce not met: (", nonce.ReputerNonce.BlockHeight, ",", nonce.WorkerNonce.BlockHeight, ") for topic: ", topic.Id, "block height: ", currentBlockHeight, "epoch length: ", topic.EpochLength)
						}
					}
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
