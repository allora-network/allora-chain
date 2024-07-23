package rewards

import (
	"fmt"
	"sort"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type BlockHeight = int64
type TopicId = uint64

// The amount of emission rewards to be distributed to a topic
// E_{t,i} = f_{t,i}*E_i
// f_{t,i} is the reward fraction for that topic
// E_i is the reward emission total for that epoch
func GetTopicReward(
	topicRewardFraction alloraMath.Dec,
	totalReward alloraMath.Dec,
) (alloraMath.Dec, error) {
	return topicRewardFraction.Mul(totalReward)
}

// The reward fraction for a topic
// normalize the topic reward weight
// f{t,i} = (1 - f_v) * (w_{t,i}) / (∑_t w_{t,i})
// where f_v is a global parameter set that controls the
// fraction of total reward emissions for cosmos network validators
// we don't use f_v here, because by the time the emissions module runs
// the validator rewards have already been distributed to the fee_collector account
// (this is done in the mint and then distribution module)
// w_{t,i} is the weight of topic t
// and the sum is naturally the total of all the weights for all topics
func GetTopicRewardFraction(
	//	f_v alloraMath.Dec,
	topicWeight *alloraMath.Dec,
	totalWeight alloraMath.Dec,
) (alloraMath.Dec, error) {
	if topicWeight == nil {
		return alloraMath.ZeroDec(), types.ErrInvalidValue
	}
	return (*topicWeight).Quo(totalWeight)
}

// Get all active topics that have ended their epoch
func GetAllActiveEpochEndingTopics(
	ctx sdk.Context,
	k keeper.Keeper,
	block BlockHeight,
	topicPageLimit uint64,
	maxTopicPages uint64,
) []types.Topic {
	var epochEndingTopics []types.Topic
	topicPageKey := make([]byte, 0)
	pageIterationCounter := uint64(0)
	for {
		topicPageRequest := &types.SimpleCursorPaginationRequest{Limit: topicPageLimit, Key: topicPageKey}
		topicsActive, topicPageResponse, err := k.GetIdsOfActiveTopics(ctx, topicPageRequest)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting ids of active topics: %s", err.Error()))
			break
		}
		for _, topicId := range topicsActive {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				Logger(ctx).Warn(fmt.Sprintf("Error getting topic: %s", err.Error()))
				continue
			}

			if k.CheckCadence(block, topic) {
				epochEndingTopics = append(epochEndingTopics, topic)
			}
		}

		// if pageResponse.NextKey is empty then we have reached the end of the list
		if len(topicsActive) == 0 || pageIterationCounter > maxTopicPages {
			break
		}
		topicPageKey = topicPageResponse.NextKey
		pageIterationCounter++
	}

	// Sort topics by ID
	sort.Slice(epochEndingTopics, func(i, j int) bool {
		return epochEndingTopics[i].Id < epochEndingTopics[j].Id
	})

	return epochEndingTopics
}

// "Churn-ready topic" is active, has an epoch that ended, and is in top N by weights, has non-zero weight.
// We iterate through active topics, fetch their weight, skim the top N by weight (these are the churnable topics)
// then finally apply a function on each of these churnable topics.
func PickChurnableActiveTopics(
	ctx sdk.Context,
	k keeper.Keeper,
	block BlockHeight,
	weights map[TopicId]*alloraMath.Dec,
) error {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get max topics per block")
	}
	weightsOfTopActiveTopics, sortedTopActiveTopics := SkimTopTopicsByWeightDesc(
		ctx,
		weights,
		moduleParams.MaxTopicsPerBlock,
		block,
	)

	for _, topicId := range sortedTopActiveTopics {
		weight := weightsOfTopActiveTopics[topicId]
		if weight.Equal(alloraMath.ZeroDec()) {
			Logger(ctx).Debug("Skipping Topic ID: ", topicId, " Weight: ", weight)
			continue
		}
		// Get the topic
		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			Logger(ctx).Debug("Error getting topic: ", err)
			continue
		}
		// Loop over and run epochs on topics whose inferences are demanded enough to be served
		// Check the cadence of inferences, and just in case also check multiples of epoch lengths
		// to avoid potential situations where the block is missed
		if k.CheckCadence(block, topic) {
			ctx.Logger().Debug(fmt.Sprintf("ABCI EndBlocker: Inference cadence met for topic: %v metadata: %s default arg: %s. \n",
				topic.Id,
				topic.Metadata,
				topic.DefaultArg))

			// Update the last inference ran
			err = k.UpdateTopicEpochLastEnded(ctx, topic.Id, block)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error updating last inference ran: %s", err.Error()))
			}
			// Add Worker Nonces
			nextNonce := types.Nonce{BlockHeight: block}
			err = k.AddWorkerNonce(ctx, topic.Id, &nextNonce)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error adding worker nonce: %s", err.Error()))
				continue
			}
			ctx.Logger().Debug(fmt.Sprintf("Added worker nonce for topic %d: %v \n", topic.Id, nextNonce.BlockHeight))
			// To notify topic handler that the topic is ready for churn i.e. requests to be sent to workers and reputers
			err = k.AddChurnableTopic(ctx, topic.Id)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error setting churn ready topic: %s", err.Error()))
				continue
			}

			MaxUnfulfilledReputerRequests := types.DefaultParams().MaxUnfulfilledReputerRequests
			moduleParams, err := k.GetParams(ctx)
			if err != nil {
				ctx.Logger().Warn(fmt.Sprintf("Error getting max retries to fulfil nonces for worker requests (using default), err: %s", err.Error()))
			} else {
				MaxUnfulfilledReputerRequests = moduleParams.MaxUnfulfilledReputerRequests
			}
			// Adding one to cover for one extra epochLength
			reputerPruningBlock := block - (int64(MaxUnfulfilledReputerRequests+1)*topic.EpochLength + topic.GroundTruthLag)
			if reputerPruningBlock > 0 {
				ctx.Logger().Warn(fmt.Sprintf("Pruning reputer nonces before block: %v for topic: %d on block: %v", reputerPruningBlock, topic.Id, block))
				k.PruneReputerNonces(ctx, topic.Id, reputerPruningBlock)

				// Reputer nonces need to check worker nonces from one epoch before
				workerPruningBlock := reputerPruningBlock - topic.EpochLength
				if workerPruningBlock > 0 {
					ctx.Logger().Debug("Pruning worker nonces before block: ", workerPruningBlock, " for topic: ", topic.Id)
					// Prune old worker nonces previous to current blockHeight to avoid inserting inferences after its time has passed
					k.PruneWorkerNonces(ctx, topic.Id, workerPruningBlock)
				}
			}
		}
		if err != nil {
			Logger(ctx).Debug("Error applying function on topic: ", err)
			continue
		}
	}
	return nil
}

// Iterates through every active topic, computes its target weight, then exponential moving average to get weight.
// Returns the total sum of weight, topic revenue, map of all of the weights by topic.
// Note that the outputted weights are not normalized => not dependent on pan-topic data.
func GetAndUpdateActiveTopicWeights(
	ctx sdk.Context,
	k keeper.Keeper,
	block BlockHeight,
) (
	weights map[TopicId]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalRevenue cosmosMath.Int,
	err error,
) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get alpha")
	}

	// Retrieve and sort all active topics with epoch ending at this block
	// default page limit for the max because default is 100 and max is 1000
	// 1000 is excessive for the topic query
	sortedEpochEndingTopics := GetAllActiveEpochEndingTopics(ctx, k, block, moduleParams.DefaultPageLimit, moduleParams.DefaultPageLimit)

	totalRevenue = cosmosMath.ZeroInt()
	sumWeight = alloraMath.ZeroDec()
	weights = make(map[TopicId]*alloraMath.Dec)
	fn := func(ctx sdk.Context, topic *types.Topic) error {
		// Calc weight and related data per topic
		weight, topicFeeRevenue, err := k.GetCurrentTopicWeight(
			ctx,
			topic.Id,
			topic.EpochLength,
			moduleParams.TopicRewardAlpha,
			moduleParams.TopicRewardStakeImportance,
			moduleParams.TopicRewardFeeRevenueImportance,
			cosmosMath.ZeroInt(),
		)
		if err != nil {
			return errors.Wrapf(err, "failed to get current topic weight")
		}

		err = k.SetPreviousTopicWeight(ctx, topic.Id, weight)
		if err != nil {
			return errors.Wrapf(err, "failed to set previous topic weight")
		}

		// This revenue will be paid to top active topics of this block (the churnable topics).
		// This happens regardless of this topic's fate (inactivation or not)
		// => the influence of this topic's revenue needs to be appropriately diminished.
		err = k.DripTopicFeeRevenue(ctx, topic.Id, block)
		if err != nil {
			return errors.Wrapf(err, "failed to reset topic fee revenue")
		}

		// If the topic is inactive, inactivate it
		if weight.Lt(moduleParams.MinTopicWeight) {
			err := k.InactivateTopic(ctx, topic.Id)
			return errors.Wrapf(err, "failed to inactivate topic")
		}

		totalRevenue = totalRevenue.Add(topicFeeRevenue)
		weights[topic.Id] = &weight
		sumWeight, err = sumWeight.Add(weight)
		if err != nil {
			return errors.Wrapf(err, "failed to add weight to sum")
		}
		return nil
	}

	// Apply the function on all sorted topics
	for _, topic := range sortedEpochEndingTopics {
		err := fn(ctx, &topic)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, err
		}
	}

	return weights, sumWeight, totalRevenue, nil
}
