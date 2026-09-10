package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// It is assumed the size of the outputted array has been bounded as it was constructed
// => can be safely handled in memory.
func (k *TopicKeeper) GetActiveTopicIdsAtBlock(ctx context.Context, block BlockHeight) (types.TopicIds, error) {
	idsOfActiveTopics, err := k.blockToActiveTopics.Get(ctx, block)
	if errors.Is(err, collections.ErrNotFound) {
		return types.TopicIds{TopicIds: nil}, nil
	} else if err != nil {
		return types.TopicIds{}, err
	}
	return idsOfActiveTopics, nil
}

// Boolean is true if the block is not found (true if no prior value), else false
func (k *TopicKeeper) GetLowestActiveTopicWeightAtBlock(
	ctx context.Context,
	block BlockHeight,
) (topicIdAndWeight types.TopicIdWeightPair, noPrior bool, err error) {
	topicIdAndWeight, err = k.blockToLowestActiveTopicWeight.Get(ctx, block)
	if errors.Is(err, collections.ErrNotFound) {
		return types.TopicIdWeightPair{
			TopicId: 0,
			Weight:  alloraMath.NewDecFromInt64(0),
		}, true, nil
	} else if err != nil {
		return types.TopicIdWeightPair{}, false, err
	}
	return topicIdAndWeight, false, nil
}

// Removes data for a block if it exists in the maps:
// - blockToActiveTopics
// - blockToLowestActiveTopicWeight
// No op if the block does not exist in the maps
func (k *TopicKeeper) PruneTopicActivationDataAtBlock(ctx context.Context, block BlockHeight) error {
	err := k.blockToActiveTopics.Remove(ctx, block)
	if err != nil {
		return errorsmod.Wrap(err, "failed to remove block to active topics")
	}

	err = k.blockToLowestActiveTopicWeight.Remove(ctx, block)
	if err != nil {
		return errorsmod.Wrap(err, "failed to remove block to lowest active topic weight")
	}

	return nil
}

func (k *TopicKeeper) ResetLowestActiveTopicWeightAtBlock(ctx context.Context, block BlockHeight) error {
	activeTopicIds, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get active topic ids at block")
	}

	if len(activeTopicIds.TopicIds) == 0 {
		return k.PruneTopicActivationDataAtBlock(ctx, block)
	}

	firstIter := true
	lowestWeight := alloraMath.NewDecFromInt64(0)
	idOfLowestWeightTopic := uint64(0)
	for _, topicId := range activeTopicIds.TopicIds {
		weight, err := k.GetTopicWeightFromTopicId(ctx, topicId)
		if err != nil {
			continue
		}
		if weight.Lt(lowestWeight) || firstIter {
			lowestWeight = weight
			idOfLowestWeightTopic = topicId
			firstIter = false
		}
	}

	data := types.TopicIdWeightPair{Weight: lowestWeight, TopicId: idOfLowestWeightTopic}
	err = k.SetBlockToLowestActiveTopicWeight(ctx, block, data)
	if err != nil {
		return errorsmod.Wrap(err, "failed to set block to lowest active topic weight")
	}
	return nil
}

// Set a topic to inactive if the topic exists and is active, else does nothing
func (k *TopicKeeper) inactivateTopicWithoutMinWeightReset(ctx context.Context, topicId TopicId) error {
	topicExists, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to check if topic exists")
	}
	if !topicExists {
		return types.ErrTopicDoesNotExist
	}

	// Check if this topic is activated or not
	block, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get next possible churning block by topic id")
	}
	if !topicIsActive {
		return nil
	}

	topicIdsActiveAtBlock, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get active topic ids at block")
	}
	topicFound := false
	// Remove the topic from the active topics at the block
	// If the topic is not found in the active topics at the block, no op
	for i, id := range topicIdsActiveAtBlock.TopicIds {
		if id == topicId {
			newActiveTopicIds := append(
				topicIdsActiveAtBlock.TopicIds[:i],
				topicIdsActiveAtBlock.TopicIds[i+1:]...,
			)
			err = k.SetBlockToActiveTopics(ctx, block, types.TopicIds{TopicIds: newActiveTopicIds})
			if err != nil {
				return errorsmod.Wrap(err, "failed to set block to active topics")
			}
			topicFound = true
			break
		}
	}

	if !topicFound {
		return errorsmod.Wrap(types.ErrNotFound, "active topic expected to be found in block's active topics list")
	}

	err = k.topicToNextPossibleChurningBlock.Remove(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to remove topic to next possible churning block")
	}

	// Set inactive for this topic
	err = k.activeTopics.Remove(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to remove active topics")
	}

	err = k.RemoveTopicFromPreviousTopicWeights(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to remove topic from previous topic weights")
	}

	// Emit topic deactivation event
	types.EmitNewTopicStatusChangedEvent(ctx, topicId, false)

	return nil
}

func (k *TopicKeeper) RemoveTopicFromPreviousTopicWeights(ctx context.Context, topicId TopicId) error {
	// Remove previous weight from total sum of previous topic weights, but keep on list

	previousTopicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get previous topic weight")
	}
	if noPrior {
		return nil
	}

	totalSumPreviousTopicWeights, err := k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get total sum of previous topic weights")
	}

	totalSumPreviousTopicWeights, err = totalSumPreviousTopicWeights.Sub(previousTopicWeight)
	if err != nil {
		return errorsmod.Wrap(err, "failed to subtract previous topic weight from total sum of previous topic weights")
	}
	err = k.SetTotalSumPreviousTopicWeights(ctx, totalSumPreviousTopicWeights)
	if err != nil {
		return errorsmod.Wrap(err, "failed to set total sum of previous topic weights")
	}

	return nil
}

func (k *TopicKeeper) addTopicToActiveSetRespectingLimitsWithoutMinWeightReset(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	topicIdsActiveAtBlock, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return err
	}
	existingActiveTopics := topicIdsActiveAtBlock.TopicIds

	// If the topic is already active at the block, no op
	for _, id := range existingActiveTopics {
		if id == topicId {
			return types.ErrTopicAlreadyActive
		}
	}

	// If the number of active topics at the block is at the limit, remove the topic with the lowest weight
	if uint64(len(existingActiveTopics)) >= params.MaxActiveTopicsPerBlock {
		// Remove the topic with the lowest weight
		lowestWeight, _, err := k.GetLowestActiveTopicWeightAtBlock(ctx, block)
		if err != nil {
			return err
		}

		weight, err := k.GetTopicWeightFromTopicId(ctx, topicId)
		if err != nil {
			return err
		}

		if weight.Lt(lowestWeight.Weight) {
			sdkCtx.Logger().Warn("Topic cannot be activated due to less than lowest weight at block", "topicId", topicId, "block", block)
			return types.ErrTopicCannotBeActivated
		}
		err = k.inactivateTopicWithoutMinWeightReset(ctx, lowestWeight.TopicId)
		if err != nil {
			return err
		}

		// Remove the lowest weight topic from the active topics at the block
		for i, id := range existingActiveTopics {
			if id == lowestWeight.TopicId {
				existingActiveTopics = append(existingActiveTopics[:i], existingActiveTopics[i+1:]...)
				break
			}
		}
	}

	existingActiveTopics = append(existingActiveTopics, topicId)
	// Add newly active topic to the active topics at the block
	newActiveTopicIds := types.TopicIds{TopicIds: existingActiveTopics}
	err = k.SetBlockToActiveTopics(ctx, block, newActiveTopicIds)
	if err != nil {
		return err
	}

	// Emit topic activation event
	types.EmitNewTopicStatusChangedEvent(ctx, topicId, true)

	return nil
}

// Set a topic to active if the topic exists, else does nothing
func (k *TopicKeeper) ActivateTopic(ctx context.Context, topicId TopicId) error {
	topicExists, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to check if topic exists")
	}
	if !topicExists {
		return nil
	}

	// Check topic activation with next possible churning block
	_, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get next possible churning block by topic id")
	}
	if topicIsActive {
		return nil
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get topic")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := sdkCtx.BlockHeight()
	epochEndBlock := currentBlock + topic.EpochLength

	err = k.activateTopicAndResetLowestWeightAtBlock(ctx, topicId, epochEndBlock)
	if errorsmod.IsOf(err, types.ErrTopicAlreadyActive, types.ErrTopicCannotBeActivated) {
		sdkCtx.Logger().Info("Failed to add topic at next epoch", "topicId", topicId, "epochEndBlock", epochEndBlock, "error", err)
		return nil
	}
	if err != nil {
		return errorsmod.Wrap(err, "failed to activate topic and reset lowest weight at block")
	}

	// A topic already in the active set still has its weight counted in the total;
	// only a topic entering the set contributes its stored weight.
	wasInActiveSet, err := k.IsTopicInActiveSet(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to check active set membership")
	}

	// Set active for this topic
	err = k.SetActiveTopics(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to set active topics")
	}

	sdkCtx.Logger().Info("Topic activated at block", "topicId", topicId, "block", currentBlock)
	// This topic was activated, so we need to add its previous weight if any to the total sum of previous topic weights
	topicWeight, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get topic weight from topic id")
	}
	if !wasInActiveSet && !noPrior && !topicWeight.IsZero() {
		totalSumPreviousTopicWeights, err := k.GetTotalSumPreviousTopicWeights(ctx)
		if err != nil {
			return errorsmod.Wrap(err, "failed to get total sum of previous topic weights")
		}
		totalSumPreviousTopicWeights, err = totalSumPreviousTopicWeights.Add(topicWeight)
		if err != nil {
			return errorsmod.Wrap(err, "failed to add weight to total sum of previous topic weights")
		}
		err = k.SetTotalSumPreviousTopicWeights(ctx, totalSumPreviousTopicWeights)
		if err != nil {
			return errorsmod.Wrap(err, "failed to set total sum of previous topic weights")
		}
	}
	return nil
}

// Inactivate the topic
func (k *TopicKeeper) InactivateTopic(ctx context.Context, topicId TopicId) error {
	err := k.inactivateTopicWithoutMinWeightReset(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to inactivate topic without min weight reset")
	}
	return nil
}

// If the topic weight is not less than lowest weight keep it as activated
func (k *TopicKeeper) AttemptTopicReactivation(ctx context.Context, topicId TopicId) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get topic")
	}
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	epochEndBlock := currentBlock + topic.EpochLength

	// Schedule the next epoch before leaving the current block: a topic that does not make
	// the cut is inactivated while it is still listed at its churning block, which is where
	// inactivation expects to find it. Leaving it in the active set with no schedule would
	// keep its weight in the total and let a later activation count that weight again.
	err = k.activateTopicAndResetLowestWeightAtBlock(ctx, topicId, epochEndBlock)
	if errorsmod.IsOf(err, types.ErrTopicCannotBeActivated) {
		sdkCtx.Logger().Info("Topic did not make the cut for its next epoch, inactivating", "topicId", topicId, "epochEndBlock", epochEndBlock)
		err = k.inactivateTopicWithoutMinWeightReset(ctx, topicId)
		if err != nil {
			return errorsmod.Wrap(err, "failed to inactivate topic that could not be rescheduled")
		}
		return nil
	} else if errorsmod.IsOf(err, types.ErrTopicAlreadyActive) {
		// Already listed at the next block; only the churning block has to move forward.
		err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId, epochEndBlock)
		if err != nil {
			return errorsmod.Wrap(err, "failed to set topic to next possible churning block")
		}
	} else if err != nil {
		return errorsmod.Wrap(err, "failed to activate topic and reset lowest weight at block")
	}

	err = k.removeTopicFromBlock(ctx, topicId, currentBlock)
	if err != nil {
		sdkCtx.Logger().Warn("Failed to remove current active topic", "topicId", topicId, "block", currentBlock, "error", err)
		return errorsmod.Wrap(err, "failed to remove current active topic from block")
	}

	sdkCtx.Logger().Debug("Topic reactivated at next epoch", "topicId", topicId, "epochEndBlock", epochEndBlock)
	return nil
}

// removeTopicFromBlock drops the topic from the list of topics whose epoch ends at block.
// The topic's churning block is left untouched: callers have already moved it forward.
func (k *TopicKeeper) removeTopicFromBlock(ctx context.Context, topicId TopicId, block BlockHeight) error {
	activeTopicIds, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get active topic ids at block")
	}
	existingActiveTopics := activeTopicIds.TopicIds
	// Remove the lowest weight topic from the active topics at the block
	for i, id := range existingActiveTopics {
		if id == topicId {
			existingActiveTopics = append(existingActiveTopics[:i], existingActiveTopics[i+1:]...)
			break
		}
	}
	newActiveTopicIds := types.TopicIds{TopicIds: existingActiveTopics}
	err = k.SetBlockToActiveTopics(ctx, block, newActiveTopicIds)
	if err != nil {
		return errorsmod.Wrap(err, "failed to set block to active topics")
	}
	return nil
}

func (k *TopicKeeper) activateTopicAndResetLowestWeightAtBlock(ctx context.Context, topicId TopicId, epochEndBlock BlockHeight) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Add to next epoch end block if greater than lowest weight
	err := k.addTopicToActiveSetRespectingLimitsWithoutMinWeightReset(ctx, topicId, epochEndBlock)
	if err != nil {
		return errorsmod.Wrap(err, "failed to add topic to active set respecting limits without min weight reset")
	}

	err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId, epochEndBlock)
	if err != nil {
		return errorsmod.Wrap(err, "failed to set topic to next possible churning block")
	}

	// Reset lowest weight
	err = k.ResetLowestActiveTopicWeightAtBlock(ctx, epochEndBlock)
	if err != nil {
		sdkCtx.Logger().Warn("Failed to reset lowest weight at next epoch", "topicId", topicId, "epochEndBlock", epochEndBlock, "error", err)
		return errorsmod.Wrap(err, "failed to reset lowest weight at block")
	}

	return nil
}
