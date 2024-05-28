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

const secondsInAMonth uint64 = 2592000

type TopicsHandler struct {
	emissionsKeeper emissionskeeper.Keeper
}

type TopicId = uint64

func NewTopicsHandler(emissionsKeeper emissionskeeper.Keeper) *TopicsHandler {
	return &TopicsHandler{
		emissionsKeeper: emissionsKeeper,
	}
}

// Calculate approximate time for the previous block as epoch timestamp
func (th *TopicsHandler) calculatePreviousBlockApproxTime(ctx sdk.Context, inferenceBlockHeight int64, groundTruthLag int64) (uint64, error) {
	blocksPerMonth, err := th.emissionsKeeper.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return 0, err
	}
	approximateTimePerBlockSeconds := secondsInAMonth / blocksPerMonth
	timeDifferenceInBlocks := ctx.BlockHeight() - inferenceBlockHeight
	// Ensure no time in the future is calculated because of ground truth lag
	if groundTruthLag > timeDifferenceInBlocks {
		timeDifferenceInBlocks = 0
	} else {
		timeDifferenceInBlocks -= groundTruthLag
	}

	timeDifferenceInSeconds := uint64(timeDifferenceInBlocks) * approximateTimePerBlockSeconds
	previousBlockApproxTime := uint64(ctx.BlockTime().Unix()) - timeDifferenceInSeconds
	return previousBlockApproxTime, nil
}

func (th *TopicsHandler) requestTopicWorkers(ctx sdk.Context, topic emissionstypes.Topic) {
	ctx.Logger().Debug(fmt.Sprintf("Triggering inference generation for topic: %v metadata: %s default arg: %s. \n",
		topic.Id, topic.Metadata, topic.DefaultArg))

	workerNonces, err := th.emissionsKeeper.GetUnfulfilledWorkerNonces(ctx, topic.Id)
	if err != nil {
		ctx.Logger().Error("Error getting worker nonces: ", err)
		return
	}
	// Filter workerNonces to only include those that are within the epoch length
	// This is to avoid requesting inferences for epochs that have already ended
	workerNonces = synth.FilterNoncesWithinEpochLength(workerNonces, ctx.BlockHeight(), topic.EpochLength)

	maxRetriesToFulfilNoncesWorker, err := th.emissionsKeeper.GetParamsMaxRetriesToFulfilNoncesWorker(ctx)
	if err != nil {
		maxRetriesToFulfilNoncesWorker = emissionstypes.DefaultParams().MaxRetriesToFulfilNoncesWorker
		ctx.Logger().Warn("Error getting max retries to fulfil nonces for worker requests (using default), err:", err)
	}
	sortedWorkerNonces := synth.SelectTopNWorkerNonces(workerNonces, int(maxRetriesToFulfilNoncesWorker))
	ctx.Logger().Debug("Iterating Top N Worker Nonces: ", len(sortedWorkerNonces))
	// iterate over all the worker nonces to find if this is unfulfilled
	for _, nonce := range sortedWorkerNonces {
		nonceCopy := nonce
		ctx.Logger().Debug("Current Worker block height has been found unfulfilled, requesting inferences ", nonceCopy)
		go generateInferencesRequest(ctx, topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id, topic.AllowNegative, *nonceCopy)
	}
}

func (th *TopicsHandler) requestTopicReputers(ctx sdk.Context, topic emissionstypes.Topic) {
	currentBlockHeight := ctx.BlockHeight()
	ctx.Logger().Debug(fmt.Sprintf("Triggering Losses cadence met for topic: %v metadata: %s default arg: %s \n",
		topic.Id, topic.Metadata, topic.DefaultArg))
	reputerNonces, err := th.emissionsKeeper.GetUnfulfilledReputerNonces(ctx, topic.Id)
	if err != nil {
		ctx.Logger().Error("Error getting reputer nonces: ", err)
		return
	}
	// No filtering - reputation of previous rounds can still be retried if work has been done.
	maxRetriesToFulfilNoncesReputer, err := th.emissionsKeeper.GetParamsMaxRetriesToFulfilNoncesReputer(ctx)
	if err != nil {
		ctx.Logger().Warn("Error getting max num of retries to fulfil nonces for worker requests (using default), err: ", err)
		maxRetriesToFulfilNoncesReputer = emissionstypes.DefaultParams().MaxRetriesToFulfilNoncesReputer
	}
	topNReputerNonces := synth.SelectTopNReputerNonces(&reputerNonces, int(maxRetriesToFulfilNoncesReputer), currentBlockHeight, topic.GroundTruthLag)
	ctx.Logger().Debug("Iterating Top N Reputer Nonces: ", len(topNReputerNonces))
	// iterate over all the reputer nonces to find if this is unfulfilled
	for _, nonce := range topNReputerNonces {
		nonceCopy := nonce
		ctx.Logger().Debug("Reputer block height found unfulfilled, requesting reputers for block ", nonceCopy.ReputerNonce.BlockHeight, ", worker:", nonceCopy.WorkerNonce.BlockHeight)
		reputerValueBundle, err := synth.GetNetworkInferencesAtBlock(
			ctx,
			th.emissionsKeeper,
			topic.Id,
			nonceCopy.ReputerNonce.BlockHeight,
			nonceCopy.WorkerNonce.BlockHeight,
		)
		if err != nil {
			ctx.Logger().Error("Error getting latest inferences at block: ", nonceCopy.ReputerNonce.BlockHeight, ", error: ", err)
			continue
		}
		if len(reputerValueBundle.InfererValues) == 0 {
			ctx.Logger().Error("ReputerValueBundle cannot be nil")
			continue
		}

		previousBlockApproxTime, err := th.calculatePreviousBlockApproxTime(ctx, nonceCopy.ReputerNonce.BlockHeight, topic.GroundTruthLag)
		if err != nil {
			ctx.Logger().Error("Error calculating previous block approx time: ", err)
			continue
		}
		ctx.Logger().Debug("Requesting losses for topic: ", topic.Id, "reputer nonce: ", nonceCopy.ReputerNonce, "worker nonce: ", nonceCopy.WorkerNonce, "previous block approx time: ", previousBlockApproxTime)
		go generateLossesRequest(ctx, reputerValueBundle, topic.LossLogic, topic.LossMethod, topic.Id, topic.AllowNegative, *nonceCopy.ReputerNonce, *nonceCopy.WorkerNonce, previousBlockApproxTime)
	}
}

func (th *TopicsHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		ctx.Logger().Debug("\n ---------------- TopicsHandler ------------------- \n")
		churnReadyTopics, err := th.emissionsKeeper.GetChurnReadyTopics(ctx)
		if err != nil {
			ctx.Logger().Error("Error getting max number of topics per block: ", err)
			return nil, err
		}

		var wg sync.WaitGroup
		// Loop over and run epochs on topics whose inferences are demanded enough to be served
		// Within each loop, execute the inference and weight cadence checks and trigger the inference and weight generation
		for _, churnReadyTopicId := range churnReadyTopics {
			wg.Add(1)
			go func(topicId TopicId) {
				defer wg.Done()
				topic, err := th.emissionsKeeper.GetTopic(ctx, topicId)
				if err != nil {
					ctx.Logger().Error("Error getting topic: ", err)
					return
				}
				th.requestTopicWorkers(ctx, topic)
				th.requestTopicReputers(ctx, topic)
			}(churnReadyTopicId)
		}
		wg.Wait()
		// Return the transactions as they came
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}
