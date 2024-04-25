package app

import (
	"fmt"
	"sync"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"

	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const secondsInAMonth uint64 = 2592000

type TopicsHandler struct {
	emissionsKeeper emissionskeeper.Keeper
	mintKeeper      mintkeeper.Keeper
}

type TopicId = uint64

func NewTopicsHandler(emissionsKeeper emissionskeeper.Keeper, mintKeeper mintkeeper.Keeper) *TopicsHandler {
	return &TopicsHandler{
		emissionsKeeper: emissionsKeeper,
		mintKeeper:      mintKeeper,
	}
}

// Calculate approximate time for the previous block as epoch timestamp
func (th *TopicsHandler) calculatePreviousBlockApproxTime(ctx sdk.Context, blockDifference int64) (uint64, error) {
	mintParams, err := th.mintKeeper.GetParams(ctx)
	if err != nil {
		fmt.Println("Error getting mint params: ", err)
		return 0, err
	}
	BlocksPerMonth := mintParams.GetBlocksPerMonth()
	var approximateTimePerBlockSeconds float64 = float64(secondsInAMonth) / float64(BlocksPerMonth)
	var diffFloat = (float64(blockDifference) * approximateTimePerBlockSeconds)
	var previousBlockApproxTime = uint64(ctx.BlockTime().Unix() - int64(diffFloat))
	return previousBlockApproxTime, nil
}

func selectTopNReputerNonces(reputerRequestNonces *emissionstypes.ReputerRequestNonces, N int) []*emissionstypes.ReputerRequestNonce {
	// Select the top N latest elements
	var topN []*emissionstypes.ReputerRequestNonce
	if len(reputerRequestNonces.Nonces) <= N {
		topN = reputerRequestNonces.Nonces
	} else {
		topN = reputerRequestNonces.Nonces[:N]
	}
	return topN
}

func selectTopNWorkerNonces(workerNonces emissionstypes.Nonces, N int) []*emissionstypes.Nonce {
	// Select the top N latest elements
	var topN []*emissionstypes.Nonce
	if len(workerNonces.Nonces) <= N {
		topN = workerNonces.Nonces
	} else {
		topN = workerNonces.Nonces[:N]
	}
	return topN
}

func (th *TopicsHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		fmt.Printf("\n ---------------- TopicsHandler ------------------- \n")
		currentBlockHeight := ctx.BlockHeight()

		maxNumberTopics, err := th.emissionsKeeper.GetParamsMaxTopicsPerBlock(ctx)
		if err != nil {
			fmt.Println("Error getting max number of topics per block: ", err)
			return nil, err
		}

		var wg sync.WaitGroup
		// Loop over and run epochs on topics whose inferences are demanded enough to be served
		// Within each loop, execute the inference and weight cadence checks and trigger the inference and weight generation
		for i := uint64(0); i < maxNumberTopics; i++ {
			// Pop churn ready topic
			churnReadyTopicId, err := th.emissionsKeeper.PopChurnReadyTopic(ctx)
			if err != nil {
				fmt.Println("Error popping churn ready topic: ", err)
				continue
			}

			// Parallelize the inference and loss cadence checks
			wg.Add(1)
			go func(topicId TopicId) {
				defer wg.Done()

				topic, err := th.emissionsKeeper.GetTopic(ctx, topicId)
				if err != nil {
					fmt.Println("Error getting topic: ", err)
					return
				}

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
					maxRetriesToFulfilNoncesWorker, err := th.emissionsKeeper.GetParamsMaxRetriesToFulfilNoncesWorker(ctx)
					if err != nil {
						maxRetriesToFulfilNoncesWorker = emissionstypes.DefaultParams().MaxRetriesToFulfilNoncesWorker
						fmt.Println("Error getting max retries to fulfil nonces for worker requests (using default), err:", err)
					}
					sortedWorkerNonces := selectTopNWorkerNonces(workerNonces, int(maxRetriesToFulfilNoncesWorker))
					fmt.Println("Iterating Top N Worker Nonces: ", len(sortedWorkerNonces))
					// iterate over all the worker nonces to find if this is unfulfilled
					for _, nonce := range sortedWorkerNonces {
						nonceCopy := nonce
						fmt.Println("Current Worker block height has been found unfulfilled, requesting inferences ", nonceCopy)
						go generateInferences(topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id, *nonceCopy)
					}

					// REPUTER
					// Get previous topic height to repute
					fmt.Printf("Triggering Losses cadence met for topic: %v metadata: %s default arg: %s \n",
						topic.Id, topic.Metadata, topic.DefaultArg)
					reputerNonces, err := th.emissionsKeeper.GetUnfulfilledReputerNonces(ctx, topic.Id)
					if err != nil {
						fmt.Println("Error getting reputer nonces: ", err)
						return
					}
					maxRetriesToFulfilNoncesReputer, err := th.emissionsKeeper.GetParamsMaxRetriesToFulfilNoncesReputer(ctx)
					if err != nil {
						fmt.Println("Error getting max num of retries to fulfil nonces for worker requests (using default), err: ", err)
						maxRetriesToFulfilNoncesReputer = emissionstypes.DefaultParams().MaxRetriesToFulfilNoncesReputer
					}
					topNReputerNonces := selectTopNReputerNonces(&reputerNonces, int(maxRetriesToFulfilNoncesReputer))
					fmt.Println("Iterating Top N Reputer Nonces: ", len(topNReputerNonces))
					// iterate over all the reputer nonces to find if this is unfulfilled
					for _, nonce := range topNReputerNonces {
						nonceCopy := nonce
						fmt.Println("Reputer block height found unfulfilled, requesting reputers for block ", nonceCopy.ReputerNonce.BlockHeight, ", worker:", nonceCopy.WorkerNonce.BlockHeight)
						reputerValueBundle, inferencesBlockHeight, err := synth.GetNetworkInferencesAtBlock(ctx, th.emissionsKeeper, topic.Id, nonceCopy.ReputerNonce.BlockHeight)
						if err != nil {
							fmt.Println("Error getting latest inferences at block: ", nonceCopy.ReputerNonce.BlockHeight, ", error: ", err)
						}

						blockDifference := currentBlockHeight - inferencesBlockHeight
						previousBlockApproxTime, err := th.calculatePreviousBlockApproxTime(ctx, blockDifference)
						if err != nil {
							fmt.Println("Error calculating previous block approx time: ", err)
							continue
						}
						fmt.Println("Requesting losses for topic: ", topic.Id, "reputer nonce: ", nonceCopy.ReputerNonce, "worker nonce: ", nonceCopy.WorkerNonce, "previous block approx time: ", previousBlockApproxTime)
						go generateLosses(reputerValueBundle, topic.LossLogic, topic.LossMethod, topic.Id, *nonceCopy.ReputerNonce, *nonceCopy.WorkerNonce, previousBlockApproxTime)
					}
				} else {
					fmt.Println("Inference and Losses cadence not met for topic: ", topic.Id, "block height: ", currentBlockHeight, "epoch length: ", topic.EpochLength, "last ended: ", topic.EpochLastEnded)
				}
			}(churnReadyTopicId)
		}
		wg.Wait()
		// Return the transactions as they came
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}
