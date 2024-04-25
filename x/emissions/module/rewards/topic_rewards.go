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
	return (*topicWeight).Quo(totalWeight)
}

// "Reward-ready topic" is active, has an epoch that ended, has a reputer nonce in need of reward
// "Safe" because bounded by max number of pages and apply running, online operations
func SafeApplyFuncOnAllRewardReadyTopics(
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
			// Get the topic
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				fmt.Println("Error getting topic: ", err)
				continue
			}

			// Check the cadence of inferences
			if block == topic.EpochLastEnded+topic.EpochLength || block-topic.EpochLastEnded >= 2*topic.EpochLength {
				// Check topic has an unfulfilled reward nonce
				rewardNonce, err := k.GetTopicRewardNonce(ctx, topicId)
				if err != nil {
					fmt.Println("Error getting reputer request nonces: ", err)
					continue
				}
				if rewardNonce == 0 {
					fmt.Println("Reputer request nonces is nil")
					continue
				}

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

// Iterates through every reward-ready topic, computes its target weight, then exponential moving average to get weight.
// Returns the total sum of weight, topic revenue, map of all of the weights by topic.
// Note that the outputted weights are not normalized => not dependent on pan-topic data.
func GetRewardReadyTopicWeights(
	ctx context.Context,
	k keeper.Keeper,
	block BlockHeight,
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
	// for i, topic := range activeTopics {
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

		// Update revenue data
		totalRevenue = totalRevenue.Add(topicFeeRevenue)

		// Update weight data
		err = k.SetPreviousTopicWeight(ctx, topic.Id, weight)
		if err != nil {
			return errors.Wrapf(err, "failed to set previous topic weight")
		}
		weights[topic.Id] = &weight
		sumWeight, err = sumWeight.Add(weight)
		if err != nil {
			return errors.Wrapf(err, "failed to add weight to sum")
		}
		return nil
	}

	err = SafeApplyFuncOnAllRewardReadyTopics(ctx, k, block, fn, params.TopicPageLimit, params.MaxTopicPages)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to apply function on all reward ready topics to get weights")
	}

	return weights, sumWeight, totalRevenue, nil
}
