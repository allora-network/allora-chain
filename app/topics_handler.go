package app

import (
	"fmt"
	"sync"

	"cosmossdk.io/log"
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
func (th *TopicsHandler) calculatePreviousBlockApproxTime(ctx sdk.Context, inferenceBlockHeight, groundTruthLag, epochLength int64) (uint64, error) {
	emissionsParams, err := th.emissionsKeeper.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	approximateTimePerBlockSeconds := secondsInAMonth / emissionsParams.BlocksPerMonth
	timeDifferenceInBlocks := ctx.BlockHeight() - (inferenceBlockHeight + epochLength)
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
	Logger(ctx).Debug(fmt.Sprintf("Triggering inference generation for topic: %v metadata: %s default arg: %s. \n",
		topic.Id, topic.Metadata, topic.DefaultArg))

	workerNonces, err := th.emissionsKeeper.GetUnfulfilledWorkerNonces(ctx, topic.Id)
	if err != nil {
		Logger(ctx).Error("Error getting worker nonces: " + err.Error())
		return
	}
	// Filter workerNonces to only include those that are within the epoch length
	// This is to avoid requesting inferences for epochs that have already ended
	workerNonces = synth.FilterNoncesWithinEpochLength(workerNonces, ctx.BlockHeight(), topic.EpochLength)

	maxRetriesToFulfilNoncesWorker := emissionstypes.DefaultParams().MaxRetriesToFulfilNoncesWorker
	emissionsParams, err := th.emissionsKeeper.GetParams(ctx)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error getting max retries to fulfil nonces for worker requests (using default), err: %s", err.Error()))
	} else {
		maxRetriesToFulfilNoncesWorker = emissionsParams.MaxRetriesToFulfilNoncesWorker
	}
	sortedWorkerNonces := synth.SelectTopNWorkerNonces(workerNonces, int(maxRetriesToFulfilNoncesWorker))
	Logger(ctx).Debug(fmt.Sprintf("Iterating Top N Worker Nonces: %d", len(sortedWorkerNonces)))
	// iterate over all the worker nonces to find if this is unfulfilled
	for _, nonce := range sortedWorkerNonces {
		nonceCopy := nonce
		Logger(ctx).Debug(fmt.Sprintf("Current Worker block height has been found unfulfilled, requesting inferences %v", nonceCopy))
		go generateInferencesRequest(ctx, topic.InferenceLogic, topic.InferenceMethod, topic.DefaultArg, topic.Id, topic.AllowNegative, *nonceCopy)
	}
}

func (th *TopicsHandler) requestTopicReputers(ctx sdk.Context, topic emissionstypes.Topic) {
	currentBlockHeight := ctx.BlockHeight()
	Logger(ctx).Debug(fmt.Sprintf("Triggering Losses cadence met for topic: %v metadata: %s default arg: %s \n",
		topic.Id, topic.Metadata, topic.DefaultArg))
	reputerNonces, err := th.emissionsKeeper.GetUnfulfilledReputerNonces(ctx, topic.Id)
	if err != nil {
		Logger(ctx).Error("Error getting reputer nonces: " + err.Error())
		return
	}
	// No filtering - reputation of previous rounds can still be retried if work has been done.
	maxRetriesToFulfilNoncesReputer := emissionstypes.DefaultParams().MaxRetriesToFulfilNoncesReputer
	emissionsParams, err := th.emissionsKeeper.GetParams(ctx)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error getting max num of retries to fulfil nonces for worker requests (using default), err: %s", err.Error()))
	} else {
		maxRetriesToFulfilNoncesReputer = emissionsParams.MaxRetriesToFulfilNoncesReputer
	}
	topNReputerNonces := synth.SelectTopNReputerNonces(&reputerNonces, int(maxRetriesToFulfilNoncesReputer), currentBlockHeight, topic.GroundTruthLag, topic.EpochLength)
	Logger(ctx).Warn(fmt.Sprintf("Iterating Top N Reputer Nonces: %v", len(topNReputerNonces)))

	lastCommit, err := th.emissionsKeeper.GetTopicLastCommit(ctx, topic.Id, emissionstypes.ActorType_REPUTER)
	if err != nil {
		Logger(ctx).Warn("Error getting reputer last commit: "+err.Error(), ", first reputer commit?")
	}
	// iterate over all the reputer nonces to find if this is unfulfilled
	for _, nonce := range topNReputerNonces {
		nonceCopy := nonce
		// Get previous losses from keeper, default: reputer - epochLength
		lastReputerCommitBlockHeight := nonceCopy.ReputerNonce.BlockHeight - topic.EpochLength
		if lastCommit.Nonce != nil {
			Logger(ctx).Debug(fmt.Sprintf("Reputer last commit found, setting: %v", lastCommit))
			lastReputerCommitBlockHeight = lastCommit.Nonce.BlockHeight
		}

		Logger(ctx).Warn(fmt.Sprintf("Reputer block height found unfulfilled, requesting reputers for block: %v", nonceCopy.ReputerNonce.BlockHeight))
		reputerValueBundle, _, _, _, err := synth.GetNetworkInferencesAtBlock(
			ctx,
			th.emissionsKeeper,
			topic.Id,
			nonceCopy.ReputerNonce.BlockHeight,
			lastReputerCommitBlockHeight,
		)
		if err != nil {
			Logger(ctx).Error(fmt.Sprintf("Error getting latest inferences at block: %d  error: %s", nonceCopy.ReputerNonce.BlockHeight, err.Error()))
			continue
		}
		if reputerValueBundle == nil || len(reputerValueBundle.InfererValues) == 0 {
			Logger(ctx).Error("ReputerValueBundle cannot be nil")
			continue
		}

		previousBlockApproxTime, err := th.calculatePreviousBlockApproxTime(ctx, nonceCopy.ReputerNonce.BlockHeight, topic.GroundTruthLag, topic.EpochLength)
		if err != nil {
			Logger(ctx).Error("Error calculating previous block approx time: " + err.Error())
			continue
		}
		// Create "previous losses" nonce
		previousLossNonce := &emissionstypes.Nonce{
			BlockHeight: lastReputerCommitBlockHeight,
		}
		Logger(ctx).Debug(fmt.Sprintf("Requesting losses for topic: %d reputer nonce: %d worker nonce: %d previous block approx time: %d",
			topic.Id, nonceCopy.ReputerNonce, previousLossNonce, previousBlockApproxTime))
		go generateLossesRequest(ctx, reputerValueBundle, topic.LossLogic, topic.LossMethod, topic.Id, topic.AllowNegative, *nonceCopy.ReputerNonce, *previousLossNonce, previousBlockApproxTime)
	}
}

func (th *TopicsHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		Logger(ctx).Debug("\n ---------------- TopicsHandler ------------------- \n")
		churnableTopics, err := th.emissionsKeeper.GetChurnableTopics(ctx)
		if err != nil {
			Logger(ctx).Error("Error getting max number of topics per block: " + err.Error())
			return nil, err
		}

		var wg sync.WaitGroup
		// Loop over and run epochs on topics whose inferences are demanded enough to be served
		// Within each loop, execute the inference and weight cadence checks and trigger the inference and weight generation
		for _, churnableTopicId := range churnableTopics {
			wg.Add(1)
			go func(topicId TopicId) {
				defer wg.Done()
				topic, err := th.emissionsKeeper.GetTopic(ctx, topicId)
				if err != nil {
					Logger(ctx).Error("Error getting topic: " + err.Error())
					return
				}
				th.requestTopicWorkers(ctx, topic)
				th.requestTopicReputers(ctx, topic)
			}(churnableTopicId)
		}
		wg.Wait()
		// Return the transactions as they came
		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}

func Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "topic_handler")
}
