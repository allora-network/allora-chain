package rewards

import (
	"fmt"

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
	weightsOfTopActiveTopics, sortedTopActiveTopics, err := SkimTopTopicsByWeightDesc(
		ctx,
		weights,
		moduleParams.MaxActiveTopicsPerBlock,
		block,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to skim top topics by weight")
	}

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

		// Update the last inference ran
		err = k.UpdateTopicEpochLastEnded(ctx, topic.Id, block)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error updating last inference ran: %s", err.Error()))
			continue
		}

		// Add Worker Nonces
		nextNonce := types.Nonce{BlockHeight: block}
		err = k.AddWorkerNonce(ctx, topic.Id, &nextNonce)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error adding worker nonce: %s", err.Error()))
			continue
		}
		ctx.Logger().Debug(fmt.Sprintf("Added worker nonce for topic %d: %v \n", topic.Id, nextNonce.BlockHeight))

		err = k.AddWorkerWindowTopicId(ctx, block+topic.WorkerSubmissionWindow, topic.Id)
		if err != nil {
			ctx.Logger().Warn(fmt.Sprintf("Error adding worker window topic id: %s", err.Error()))
			continue
		}

		err = PruneReputerAndWorkerNonces(ctx, k, topic, block)
		if err != nil {
			Logger(ctx).Warn("Error pruning reputer and worker nonces: ", err)
			continue
		}

		err = UpdateReputerNonce(ctx, k, topic, block)
		if err != nil {
			Logger(ctx).Warn("Error updating reputer nonce: ", err)
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
	topicids, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get active topics")
	}
	totalRevenue = cosmosMath.ZeroInt()
	sumWeight = alloraMath.ZeroDec()
	weights = make(map[TopicId]*alloraMath.Dec)

	// Apply the function on all sorted topics
	for _, topicId := range topicids.TopicIds {
		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			continue
		}
		// Calc weight and related data per topic
		weight, topicFeeRevenue, err := k.GetCurrentTopicWeight(
			ctx,
			topic.Id,
			topic.EpochLength,
			moduleParams.TopicRewardAlpha,
			moduleParams.TopicRewardStakeImportance,
			moduleParams.TopicRewardFeeRevenueImportance,
		)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get current topic weight")
		}

		err = k.SetPreviousTopicWeight(ctx, topic.Id, weight)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to set previous topic weight")
		}

		// This revenue will be paid to top active topics of this block (the churnable topics).
		// This happens regardless of this topic's fate (inactivation or not)
		// => the influence of this topic's revenue needs to be appropriately diminished.
		err = k.DripTopicFeeRevenue(ctx, topic.Id, block)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to reset topic fee revenue")
		}

		// If the topic is inactive, inactivate it
		if weight.Lt(moduleParams.MinTopicWeight) {
			err := k.InactivateTopic(ctx, topic.Id)
			if err != nil {
				return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to inactivate topic")
			}
			ctx.Logger().Debug(fmt.Sprintf("Topic %d inactivated at block %d", topic.Id, block))
			continue
		}

		// Update topic active status
		err = k.AttemptTopicReactivation(ctx, topicId)
		if err != nil {
			ctx.Logger().Error("Error on attempt topic reactivation")
			continue
		}
		totalRevenue = totalRevenue.Add(topicFeeRevenue)
		weights[topic.Id] = &weight
		sumWeight, err = sumWeight.Add(weight)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to add weight to sum")
		}
	}

	return weights, sumWeight, totalRevenue, nil
}
