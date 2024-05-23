package rewards

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"

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

// Active topics have more than a globally-set minimum weight, a function of revenue and stake
// "Safe" because bounded by max number of pages and apply running, online operations
func SafeApplyFuncOnAllActiveTopics(
	ctx context.Context,
	k keeper.Keeper,
	block BlockHeight,
	fn func(ctx context.Context, topic *types.Topic) error,
	topicPageLimit uint64,
	maxTopicPages uint64,
) error {
	topicPageKey := make([]byte, 0)
	i := uint64(0)
	for {
		topicPageRequest := &types.SimpleCursorPaginationRequest{Limit: topicPageLimit, Key: topicPageKey}
		topicsActive, topicPageResponse, err := k.GetIdsOfActiveTopics(ctx, topicPageRequest)
		if err != nil {
			fmt.Println("Error getting ids of active topics: ", err)
			continue
		}

		for _, topicId := range topicsActive {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				fmt.Println("Error getting topic: ", err)
				continue
			}

			if keeper.CheckCadence(block, topic) {
				// All checks passed => Apply function on the topic
				err = fn(ctx, &topic)
				if err != nil {
					fmt.Println("Error applying function on topic: ", err)
					continue
				}
			}
		}

		// if pageResponse.NextKey is empty then we have reached the end of the list
		if topicsActive == nil || i > maxTopicPages {
			break
		}
		topicPageKey = topicPageResponse.NextKey
		i++
	}
	return nil
}

// "Churn-ready topic" is active, has an epoch that ended, and is in top N by weights, has non-zero weight.
// We iterate through active topics, fetch their weight, skim the top N by weight (these are the churnable topics)
// then finally apply a function on each of these churnable topics.
func IdentifyChurnableAmongActiveTopicsAndApplyFn(
	ctx context.Context,
	k keeper.Keeper,
	block BlockHeight,
	fn func(ctx context.Context, topic *types.Topic) error,
	weights map[TopicId]*alloraMath.Dec,
) error {
	maxTopicsPerBlock, err := k.GetParamsMaxTopicsPerBlock(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get max topics per block")
	}
	weightsOfTopActiveTopics, sortedTopActiveTopics := SkimTopTopicsByWeightDesc(weights, maxTopicsPerBlock, block)

	for _, topicId := range sortedTopActiveTopics {
		weight := weightsOfTopActiveTopics[topicId]
		if weight.Equal(alloraMath.ZeroDec()) {
			fmt.Println("Skipping Topic ID: ", topicId, " Weight: ", weight)
			continue
		}
		// Get the topic
		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			fmt.Println("Error getting topic: ", err)
			continue
		}
		// Execute the function
		err = fn(ctx, &topic)
		if err != nil {
			fmt.Println("Error applying function on topic: ", err)
			continue
		}
	}
	return nil
}

// Iterates through every active topic, computes its target weight, then exponential moving average to get weight.
// Returns the total sum of weight, topic revenue, map of all of the weights by topic.
// Note that the outputted weights are not normalized => not dependent on pan-topic data.
// updatePrevious is a flag to perform update of previous weight of the topic
func GetAndOptionallyUpdateActiveTopicWeights(
	ctx context.Context,
	k keeper.Keeper,
	block BlockHeight,
	updatePreviousWeights bool,
) (
	weights map[TopicId]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalRevenue cosmosMath.Int,
	err error,
) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get alpha")
	}

	totalRevenue = cosmosMath.ZeroInt()
	sumWeight = alloraMath.ZeroDec()
	weights = make(map[TopicId]*alloraMath.Dec)
	fn := func(ctx context.Context, topic *types.Topic) error {
		// Calc weight and related data per topic
		weight, topicFeeRevenue, err := k.GetCurrentTopicWeight(
			ctx,
			topic.Id,
			topic.EpochLength,
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			cosmosMath.ZeroInt(),
		)
		if err != nil {
			return errors.Wrapf(err, "failed to get current topic weight")
		}

		totalRevenue = totalRevenue.Add(topicFeeRevenue)

		if updatePreviousWeights {
			err = k.SetPreviousTopicWeight(ctx, topic.Id, weight)
			if err != nil {
				return errors.Wrapf(err, "failed to set previous topic weight")
			}
		}
		weights[topic.Id] = &weight
		sumWeight, err = sumWeight.Add(weight)
		if err != nil {
			return errors.Wrapf(err, "failed to add weight to sum")
		}
		return nil
	}

	err = SafeApplyFuncOnAllActiveTopics(ctx, k, block, fn, params.TopicPageLimit, params.MaxTopicPages)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to apply function on all reward ready topics to get weights")
	}

	return weights, sumWeight, totalRevenue, nil
}
