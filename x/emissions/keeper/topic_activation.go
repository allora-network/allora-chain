package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const RESERVED_BLOCK = 0

/**
topics collections.Map[TopicId, types.Topic]

topicToNextPossibleChurningBlock collections.Map[TopicId, BlockHeight]

blockToActiveTopics collections.Map[BlockHeight, types.TopicIds]

blockToLowestActiveTopicWeight collections.Map[BlockHeight, alloraMath.Dec]
*/

// TODO
// * Implement GetTopicWeight() + cascade its usage + possibly should couple with topic weight-related corrections from Sherlock
// * Finish propagating per global param checklist
// * Error msgs + logging
// * Migration + genesis update + test

// Boolean true if topic is active, else false
func (k *Keeper) GetNextPossibleChurningBlockByTopicId(ctx context.Context, topicId TopicId) (BlockHeight, bool, error) {
	block, err := k.topicToNextPossibleChurningBlock.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return RESERVED_BLOCK, false, nil
		}
		return RESERVED_BLOCK, false, err
	}
	return block, block > RESERVED_BLOCK, nil
}

// It is assumed the size of the outputted array has been bounded as it was constructed
// => can be safely handled in memory.
func (k *Keeper) GetActiveTopicsAtBlock(ctx context.Context, block BlockHeight) (types.TopicIds, error) {
	idsOfActiveTopics, err := k.blockToActiveTopics.Get(ctx, block)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
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
		if errors.Is(err, collections.ErrNotFound) {
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
		return err
	}

	err = k.blockToLowestActiveTopicWeight.Remove(ctx, block)
	if err != nil {
		return err
	}

	return nil
}

func (k *Keeper) ResetLowestActiveTopicWeightAtBlock(ctx context.Context, block BlockHeight) error {
	activeTopicIds, err := k.GetActiveTopicsAtBlock(ctx, block)
	if err != nil {
		return err
	}

	if len(activeTopicIds.TopicIds) == 0 {
		return k.PruneTopicActivationDataAtBlock(ctx, block)
	}

	firstIter := true
	lowestWeight := alloraMath.NewDecFromInt64(0)
	idOfLowestWeightTopic := uint64(0)
	for _, topicId := range activeTopicIds.TopicIds {
		weight, err := k.GetTopicWeight(ctx, topicId)
		if err != nil {
			return err
		}

		if weight.LT(lowestWeight) || firstIter {
			lowestWeight = weight
			idOfLowestWeightTopic = topicId
			firstIter = false
		}
	}

	data := types.TopicIdWeightPair{Weight: lowestWeight, TopicId: idOfLowestWeightTopic}
	return k.blockToLowestActiveTopicWeight.Set(ctx, block, data)
}

// Set a topic to inactive if the topic exists and is active, else does nothing
func (k *Keeper) inactivateTopicWithoutMinWeightReset(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	topicExists, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return RESERVED_BLOCK, err
	}
	if !topicExists {
		return RESERVED_BLOCK, nil
	}

	block, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return RESERVED_BLOCK, err
	}
	if !topicIsActive {
		return block, nil
	}

	topicIdsActiveAtBlock, err := k.GetActiveTopicsAtBlock(ctx, block)
	if err != nil {
		return RESERVED_BLOCK, err
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
	err = k.blockToActiveTopics.Set(ctx, block, types.TopicIds{TopicIds: newActiveTopicIds})
	if err != nil {
		return RESERVED_BLOCK, err
	}

	err = k.topicToNextPossibleChurningBlock.Remove(ctx, topicId)
	if err != nil {
		return RESERVED_BLOCK, err
	}

	return block, nil
}

// Avoids O(num_topics^2) complexity otherwise due to resetting lowest weight for each topic
func (k *Keeper) InactivateManyTopics(ctx context.Context, topicIds []TopicId) error {
	uniqueBlocksOfNextEpochEnding := map[BlockHeight]bool{}
	blocksOfNextEpochEnding := []BlockHeight{}
	for _, id := range topicIds {
		block, err := k.inactivateTopicWithoutMinWeightReset(ctx, id)
		if err != nil {
			return err
		}

		if _, ok := uniqueBlocksOfNextEpochEnding[block]; !ok && block != RESERVED_BLOCK {
			uniqueBlocksOfNextEpochEnding[block] = true
			blocksOfNextEpochEnding = append(blocksOfNextEpochEnding, block)
		}
	}

	for _, block := range blocksOfNextEpochEnding {
		err := k.ResetLowestActiveTopicWeightAtBlock(ctx, block)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k *Keeper) addTopicToActiveSetRespectingLimitsWithoutMinWeightReset(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	topicIdsActiveAtBlock, err := k.GetActiveTopicsAtBlock(ctx, block)
	if err != nil {
		return err
	}
	existingActiveTopics := topicIdsActiveAtBlock.TopicIds

	// If the topic is already active at the block, no op
	for _, id := range existingActiveTopics {
		if id == topicId {
			return nil
		}
	}

	// If the number of active topics at the block is at the limit, remove the topic with the lowest weight
	if int32(len(existingActiveTopics)) >= int32(params.MaxActiveTopicsPerBlock) {
		// Remove the topic with the lowest weight
		lowestWeight, _, err := k.GetLowestActiveTopicWeightAtBlock(ctx, block)
		if err != nil {
			return err
		}

		_, err = k.inactivateTopicWithoutMinWeightReset(ctx, lowestWeight.TopicId)
		if err != nil {
			return err
		}

		// Remove the lowest weight topic from the active topics at the block
		for i, id := range existingActiveTopics {
			if id == lowestWeight.TopicId {
				existingActiveTopics = append(existingActiveTopics[:i], existingActiveTopics[i:]...)
				break
			}
		}
	}

	// Add newly active topic to the active topics at the block
	newActiveTopicIds := types.TopicIds{TopicIds: append(existingActiveTopics, topicId)}
	err = k.blockToActiveTopics.Set(ctx, block, newActiveTopicIds)
	if err != nil {
		return err
	}

	return nil
}

// Set a topic to active if the topic exists, else does nothing
func (k *Keeper) ActivateTopic(ctx context.Context, topicId TopicId) error {
	topicExists, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return err
	}
	if !topicExists {
		return nil
	}

	block, topicIsActive, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return err
	}
	if topicIsActive {
		return nil
	}

	err = k.addTopicToActiveSetRespectingLimitsWithoutMinWeightReset(ctx, topicId, block)
	if err != nil {
		return err
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return err
	}
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	err = k.topicToNextPossibleChurningBlock.Set(ctx, topicId, currentBlock+topic.EpochLength)
	if err != nil {
		return err
	}

	err = k.ResetLowestActiveTopicWeightAtBlock(ctx, block)
	if err != nil {
		return err
	}

	return nil
}

// func (k *Keeper) ActivateManyTopics(ctx context.Context, topicIds []TopicId) error {
// 	uniqueBlocksOfNextEpochEnding := map[BlockHeight]bool{}
// 	blocksOfNextEpochEnding := []BlockHeight{}
// 	for _, id := range topicIds {
// 		block, err := k.activateTopicWithoutMinWeightReset(ctx, id)
// 		if err != nil {
// 			return err
// 		}

//		 if _, ok := uniqueBlocksOfNextEpochEnding[block]; !ok && block != RESERVED_BLOCK {
// 			uniqueBlocksOfNextEpochEnding[block] = true
// 			blocksOfNextEpochEnding = append(blocksOfNextEpochEnding, block)
// 		}
// 	}

// 	for _, block := range blocksOfNextEpochEnding {
// 		err := k.ResetLowestActiveTopicWeightAtBlock(ctx, block)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

func (k *Keeper) IsTopicActive(ctx context.Context, topicId TopicId) (bool, error) {
	_, active, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return false, err
	}

	return active, nil
}
