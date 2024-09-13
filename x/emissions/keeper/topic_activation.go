package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const RESERVED_BLOCK = 0

// Boolean true if topic is active, else false
func (k *Keeper) GetNextPossibleChurningBlockByTopicId(ctx context.Context, topicId TopicId) (BlockHeight, bool, error) {
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	block, err := k.topicToNextPossibleChurningBlock.Get(ctx, topicId)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return RESERVED_BLOCK, false, nil
		}
		return RESERVED_BLOCK, false, err
	}
	return block, block >= currentBlock, nil
}

// It is assumed the size of the outputted array has been bounded as it was constructed
// => can be safely handled in memory.
func (k *Keeper) GetActiveTopicIdsAtBlock(ctx context.Context, block BlockHeight) (types.TopicIds, error) {
	idsOfActiveTopics, err := k.blockToActiveTopics.Get(ctx, block)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			topicIds := []TopicId{}
			return types.TopicIds{TopicIds: topicIds}, nil
		}
		return types.TopicIds{}, err
	}
	return idsOfActiveTopics, nil
}

// Boolean is true if the block is not found (true if no prior value), else false
func (k *Keeper) GetLowestActiveTopicWeightAtBlock(ctx context.Context, block BlockHeight) (types.TopicIdWeightPair, bool, error) {
	weight, err := k.blockToLowestActiveTopicWeight.Get(ctx, block)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.TopicIdWeightPair{}, true, nil
		}
		return types.TopicIdWeightPair{}, false, err
	}
	return weight, false, nil
}

// Removes data for a block if it exists in the maps:
// - blockToActiveTopics
// - blockToLowestActiveTopicWeight
// No op if the block does not exist in the maps
func (k *Keeper) PruneTopicActivationDataAtBlock(ctx context.Context, block BlockHeight) error {
	err := k.blockToActiveTopics.Remove(ctx, block)
	if err != nil {
		return errors.Wrap(err, "failed to remove block to active topics")
	}

	err = k.blockToLowestActiveTopicWeight.Remove(ctx, block)
	if err != nil {
		return errors.Wrap(err, "failed to remove block to lowest active topic weight")
	}

	return nil
}

func (k *Keeper) ResetLowestActiveTopicWeightAtBlock(ctx context.Context, block BlockHeight) error {
	activeTopicIds, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err, "failed to get active topic ids at block")
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
		return errors.Wrap(err, "failed to set block to lowest active topic weight")
	}
	return nil
}

// Set a topic to inactive if the topic exists and is active, else does nothing
func (k *Keeper) inactivateTopicWithoutMinWeightReset(ctx context.Context, topicId TopicId) error {
	topicExists, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to check if topic exists")
	}
	if !topicExists {
		return nil
	}

	// Check if this topic is activated or not
	block, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to get next possible churning block by topic id")
	}
	if !topicIsActive {
		return nil
	}

	topicIdsActiveAtBlock, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err, "failed to get active topic ids at block")
	}
	// Remove the topic from the active topics at the block
	// If the topic is not found in the active topics at the block, no op
	newActiveTopicIds := []TopicId{}
	for i, id := range topicIdsActiveAtBlock.TopicIds {
		if id == topicId {
			newActiveTopicIds = append(topicIdsActiveAtBlock.TopicIds[:i], topicIdsActiveAtBlock.TopicIds[i+1:]...)
			break
		}
	}
	err = k.SetBlockToActiveTopics(ctx, block, types.TopicIds{TopicIds: newActiveTopicIds})
	if err != nil {
		return errors.Wrap(err, "failed to set block to active topics")
	}

	err = k.topicToNextPossibleChurningBlock.Remove(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to remove topic to next possible churning block")
	}

	// Set inactive for this topic
	err = k.activeTopics.Remove(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to remove active topics")
	}

	return nil
}

func (k *Keeper) addTopicToActiveSetRespectingLimitsWithoutMinWeightReset(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}

	topicIdsActiveAtBlock, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return false, err
	}
	existingActiveTopics := topicIdsActiveAtBlock.TopicIds

	// If the topic is already active at the block, no op
	for _, id := range existingActiveTopics {
		if id == topicId {
			return false, nil
		}
	}

	// If the number of active topics at the block is at the limit, remove the topic with the lowest weight
	if uint64(len(existingActiveTopics)) >= params.MaxActiveTopicsPerBlock {
		// Remove the topic with the lowest weight
		lowestWeight, _, err := k.GetLowestActiveTopicWeightAtBlock(ctx, block)
		if err != nil {
			return false, err
		}

		weight, err := k.GetTopicWeightFromTopicId(ctx, topicId)
		if err != nil {
			return false, err
		}

		if weight.Lt(lowestWeight.Weight) {
			sdkCtx.Logger().Warn(fmt.Sprintf("Topic%d cannot be activated due to less than lowest weight at block %d", topicId, block))
			return false, nil
		}
		err = k.inactivateTopicWithoutMinWeightReset(ctx, lowestWeight.TopicId)
		if err != nil {
			return false, err
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
		return false, err
	}
	return true, nil
}

// Set a topic to active if the topic exists, else does nothing
func (k *Keeper) ActivateTopic(ctx context.Context, topicId TopicId) error {
	topicExists, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to check if topic exists")
	}
	if !topicExists {
		return nil
	}

	// Check topic activation with next possible churning block
	_, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to get next possible churning block by topic id")
	}
	if topicIsActive {
		return nil
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to get topic")
	}
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	epochEndBlock := currentBlock + topic.EpochLength

	err = k.activateTopicAndResetLowestWeightAtBlock(ctx, topicId, epochEndBlock)
	if err != nil {
		return errors.Wrap(err, "failed to activate topic and reset lowest weight at block")
	}

	// Set active for this topic
	err = k.SetActiveTopics(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to set active topics")
	}

	return nil
}

// Inactivate the topic
func (k *Keeper) InactivateTopic(ctx context.Context, topicId TopicId) error {
	err := k.inactivateTopicWithoutMinWeightReset(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to inactivate topic without min weight reset")
	}
	return nil
}

// If the topic weight is not less than lowest weight keep it as activated
func (k *Keeper) AttemptTopicReactivation(ctx context.Context, topicId TopicId) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to get topic")
	}
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	epochEndBlock := currentBlock + topic.EpochLength

	// Remove current active topic from block
	err = k.removeCurrentTopicFromBlock(ctx, topicId, currentBlock)
	if err != nil {
		sdkCtx.Logger().Warn(fmt.Sprintf("Failed to remove current active topic from block %d", topicId))
		return errors.Wrap(err, "failed to remove current active topic from block")
	}

	err = k.activateTopicAndResetLowestWeightAtBlock(ctx, topicId, epochEndBlock)
	if err != nil {
		return errors.Wrap(err, "failed to activate topic and reset lowest weight at block")
	}

	sdkCtx.Logger().Debug(fmt.Sprintf("Topic %d reactivated at next epoch %d", topicId, epochEndBlock))
	return nil
}

func (k *Keeper) removeCurrentTopicFromBlock(ctx context.Context, topicId TopicId, block BlockHeight) error {
	activeTopicIds, err := k.GetActiveTopicIdsAtBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err, "failed to get active topic ids at block")
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
		return errors.Wrap(err, "failed to set block to active topics")
	}
	err = k.topicToNextPossibleChurningBlock.Remove(ctx, topicId)
	if err != nil {
		return errors.Wrap(err, "failed to remove topic to next possible churning block")
	}
	return nil
}

func (k *Keeper) activateTopicAndResetLowestWeightAtBlock(ctx context.Context, topicId TopicId, epochEndBlock BlockHeight) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Add to next epoch end block if greater than lowest weight
	isAdded, err := k.addTopicToActiveSetRespectingLimitsWithoutMinWeightReset(ctx, topicId, epochEndBlock)
	if err != nil {
		sdkCtx.Logger().Warn(fmt.Sprintf("Failed to add topic at next epoch %d, %d", topicId, epochEndBlock))
		return errors.Wrap(err, "failed to add topic to active set respecting limits without min weight reset")
	}
	if !isAdded {
		return nil
	}

	err = k.SetTopicToNextPossibleChurningBlock(ctx, topicId, epochEndBlock)
	if err != nil {
		return errors.Wrap(err, "failed to set topic to next possible churning block")
	}

	// Reset lowest weight
	err = k.ResetLowestActiveTopicWeightAtBlock(ctx, epochEndBlock)
	if err != nil {
		sdkCtx.Logger().Warn(fmt.Sprintf("Failed to reset lowest weight at next epoch %d, %d", topicId, epochEndBlock))
		return errors.Wrap(err, "failed to reset lowest weight at block")
	}

	return nil
}
