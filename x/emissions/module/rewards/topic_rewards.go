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

// Return the target weight of a topic
// ^w_{t,i} = S^{μ}_{t,i} * (P/C)^{ν}_{t,i}
// where S_{t,i} is the stake of of topic t in the last reward epoch i
// and (P/C)_{t,i} is the fee revenue collected for performing inference per topic epoch
// requests for topic t in the last reward epoch i
// μ, ν are global constants with fiduciary values of 0.5 and 0.5
func GetTargetWeight(
	topicStake alloraMath.Dec,
	topicEpochLength int64,
	topicFeeRevenue alloraMath.Dec,
	stakeImportance alloraMath.Dec,
	feeImportance alloraMath.Dec,
) (alloraMath.Dec, error) {
	s, err := alloraMath.Pow(topicStake, stakeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	c := alloraMath.NewDecFromInt64(topicEpochLength)
	feePerEpoch, err := topicFeeRevenue.Quo(c)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	p, err := alloraMath.Pow(feePerEpoch, feeImportance)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return s.Mul(p)
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
func GetRewardReadyTopicWeights(
	ctx context.Context,
	k keeper.Keeper,
	block BlockHeight,
) (
	weights map[TopicId]*alloraMath.Dec,
	sumWeight alloraMath.Dec,
	totalRevenue cosmosMath.Uint,
	numRewardReadyTopics uint32, err error,
) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Uint{}, uint32(0), errors.Wrapf(err, "failed to get alpha")
	}
	stakeImportance, feeImportance, err := k.GetParamsStakeAndFeeRevenueImportance(ctx)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Uint{}, uint32(0), errors.Wrapf(err, "failed to get stake and fee revenue importance")
	}

	totalRevenue = cosmosMath.ZeroUint()
	sumWeight = alloraMath.ZeroDec()
	weights = make(map[TopicId]*alloraMath.Dec)
	numRewardReadyTopics = uint32(0)
	// for i, topic := range activeTopics {
	fn := func(ctx context.Context, topic *types.Topic) error {
		topicStake, err := k.GetTopicStake(ctx, topic.Id)
		if err != nil {
			return errors.Wrapf(err, "failed to get topic stake")
		}
		topicStakeDec, err := alloraMath.NewDecFromSdkUint(topicStake)
		if err != nil {
			return errors.Wrapf(err, "failed to convert topic stake to dec")
		}

		// Get and total topic fee revenue
		topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topic.Id)
		if err != nil {
			return errors.Wrapf(err, "failed to get topic fee revenue")
		}
		topicFeeRevenueUint := cosmosMath.Uint(topicFeeRevenue.Revenue)
		totalRevenue = totalRevenue.Add(topicFeeRevenueUint)

		// Calc target weight using fees, epoch length, stake, and params
		feeRevenue, err := alloraMath.NewDecFromSdkInt(topicFeeRevenue.Revenue)
		if err != nil {
			return errors.Wrapf(err, "failed to convert topic fee revenue to dec")
		}
		targetWeight, err := GetTargetWeight(
			topicStakeDec,
			topic.EpochLength,
			feeRevenue,
			stakeImportance,
			feeImportance,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to get target weight")
		}

		// Take EMA of target weight with previous weight
		previousTopicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topic.Id)
		if err != nil {
			return errors.Wrapf(err, "failed to get previous topic weight")
		}
		weight, err := alloraMath.CalcEma(params.TopicRewardAlpha, targetWeight, previousTopicWeight, noPrior)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate EMA")
		}
		err = k.SetPreviousTopicWeight(ctx, topic.Id, weight)
		if err != nil {
			return errors.Wrapf(err, "failed to set previous topic weight")
		}
		weights[topic.Id] = &weight
		sumWeight, err = sumWeight.Add(weight)
		if err != nil {
			return errors.Wrapf(err, "failed to add weight to sum")
		}
		numRewardReadyTopics++
		return nil
	}

	err = SafeApplyFuncOnAllRewardReadyTopics(ctx, k, block, fn, params.TopicPageLimit, params.MaxTopicPages)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Uint{}, uint32(0), errors.Wrapf(err, "failed to apply function on all reward ready topics to get weights")
	}

	return weights, sumWeight, totalRevenue, numRewardReadyTopics, nil
}
