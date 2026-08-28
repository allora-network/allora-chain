package rewards

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type BlockHeight = int64
type TopicId = uint64

// At the end of epoch, we should update nonce status of active topics, skim the top N by weight.
// Update worker/reputer nonces, also add reward topic nonce to get reward for this nonce.
func UpdateNoncesOfActiveTopics(
	ctx sdk.Context,
	k keeper.Keeper,
	block BlockHeight,
	weights map[TopicId]*alloraMath.Dec,
) error {
	moduleParams, err := k.GetParamsKeeper().GetParams(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get max topics per block")
	}
	weightsOfTopActiveTopics, sortedTopActiveTopics, err := SkimTopTopicsByWeightDesc(
		ctx,
		weights,
		moduleParams.MaxActiveTopicsPerBlock,
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
		topic, err := k.GetTopicKeeper().GetTopic(ctx, topicId)
		if err != nil {
			Logger(ctx).Debug("Error getting topic: ", err)
			continue
		}

		// Update the last inference ran
		err = k.GetTopicKeeper().UpdateTopicEpochLastEnded(ctx, topic.Id, block)
		if err != nil {
			ctx.Logger().Warn("Error updating last inference ran", "error", err)
			continue
		}

		// Add Worker Nonces
		nextNonce := types.Nonce{BlockHeight: block}
		err = k.GetNonceKeeper().AddWorkerNonce(ctx, topic.Id, &nextNonce)
		if err != nil {
			ctx.Logger().Warn("Error adding worker nonce", "error", err)
			continue
		}
		ctx.Logger().Debug("Added worker nonce for topic", "topicId", topic.Id, "nonce", nextNonce.BlockHeight)

		err = k.GetNonceKeeper().AddWorkerWindowTopicId(ctx, block+topic.WorkerSubmissionWindow, topic.Id)
		if err != nil {
			ctx.Logger().Warn("Error adding worker window topic id", "error", err)
			continue
		}

		// Emit worker submission window opened event
		windowEndBlock := nextNonce.BlockHeight + topic.WorkerSubmissionWindow
		types.EmitNewWorkerSubmissionWindowOpenedEvent(ctx, topic.Id, nextNonce.BlockHeight, windowEndBlock)

		err = PruneReputerAndWorkerNonces(ctx, k, topic, block)
		if err != nil {
			Logger(ctx).Warn("Error pruning reputer and worker nonces", "error", err)
			continue
		}

		err = UpdateReputerNonce(ctx, k, topic, block)
		if err != nil {
			Logger(ctx).Warn("Error updating reputer nonce", "error", err)
		}
	}
	return nil
}

// Iterates through every active topic, computes its target weight, then exponential moving average to get weight.
// Returns the total sum of weight, topic revenue, map of all of the weights by topic.
// Note that the outputted weights are not normalized => not dependent on pan-topic data.
// The computed weights are NOT persisted here: they only take effect for the
// next epoch, so they are committed via CommitTopicWeights after this block's
// rewards have been distributed. Topic fee revenue dripping and topic
// (in)activation are still applied here.
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
	moduleParams, err := k.GetParamsKeeper().GetParams(ctx)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get alpha")
	}
	blocksPerWeek, err := alloraMath.CalculateBlocksPerWeek(moduleParams.BlocksPerMonth)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "error calculating blocks per week")
	}

	// Retrieve and sort all active topics with epoch ending at this block
	// default page limit for the max because default is 100 and max is 1000
	// 1000 is excessive for the topic query
	topicids, err := k.GetTopicKeeper().GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get active topics")
	}
	totalRevenue = cosmosMath.ZeroInt()
	sumWeight = alloraMath.ZeroDec()
	weights = make(map[TopicId]*alloraMath.Dec)

	// Apply the function on all sorted topics
	for _, topicId := range topicids.TopicIds {
		topic, err := k.GetTopicKeeper().GetTopic(ctx, topicId)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, err
		}

		// Calc weight and related data per topic
		weight, topicFeeRevenue, topicStake, err := k.GetTopicKeeper().GetCurrentTopicWeight(
			ctx,
			topic.Id,
			topic.EpochLength,
			moduleParams.TopicRewardAlpha,
			moduleParams.TopicRewardStakeImportance,
			moduleParams.TopicRewardFeeRevenueImportance,
			moduleParams.BlocksPerMonth,
		)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to get current topic weight")
		}
		// Emit topic weight updated event; the weight is committed to state by
		// CommitTopicWeights after this block's rewards are distributed
		types.EmitNewTopicWeightUpdatedEvent(ctx, topic.Id, weight, topicStake, topicFeeRevenue)

		// This revenue will be paid to top active topics of this block (the churnable topics).
		// This happens regardless of this topic's fate (inactivation or not)
		// => the influence of this topic's revenue needs to be appropriately diminished.
		err = k.GetTopicKeeper().DripTopicFeeRevenue(ctx, topic, blocksPerWeek, block)
		if err != nil {
			return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to reset topic fee revenue")
		}

		// If the topic is inactive, inactivate it
		if weight.Lt(moduleParams.MinTopicWeight) {
			err := k.GetTopicKeeper().InactivateTopic(ctx, topic.Id)
			if err != nil {
				return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to inactivate topic")
			}
			ctx.Logger().Debug("Topic inactivated at block", "topicId", topic.Id, "block", block)
			continue
		}

		// Update topic active status
		err = k.GetTopicKeeper().AttemptTopicReactivation(ctx, topicId)
		if err != nil {
			ctx.Logger().Error("Error on attempt topic reactivation, explicitly inactivating topic")
			err := k.GetTopicKeeper().InactivateTopic(ctx, topic.Id)
			if err != nil {
				return nil, alloraMath.Dec{}, cosmosMath.Int{}, errors.Wrapf(err, "failed to inactivate topic")
			}
			ctx.Logger().Debug("Topic inactivated at block", "topicId", topic.Id, "block", block)
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

// CommitTopicWeights persists the weights computed by GetAndUpdateActiveTopicWeights,
// updating the total sum of topic weights along the way. It is called after the
// block's rewards have been distributed, so that reward calculations see the
// weights that were in effect during the epochs being paid, and the newly
// computed weights only take effect from the next block on.
func CommitTopicWeights(
	ctx sdk.Context,
	k keeper.Keeper,
	weights map[TopicId]*alloraMath.Dec,
) error {
	for _, topicId := range alloraMath.GetSortedKeys(weights) {
		weight := weights[topicId]
		if weight == nil {
			return errors.Wrapf(types.ErrInvalidValue, "nil weight for topic %d", topicId)
		}
		Logger(ctx).Debug("Setting previous topic weight for topic", "topicId", topicId, "weight", weight.String())
		err := k.GetTopicKeeper().SetPreviousTopicWeight(ctx, topicId, *weight)
		if err != nil {
			return errors.Wrapf(err, "failed to set previous topic weight for topic %d", topicId)
		}
	}
	return nil
}
