package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *TopicKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// NextTopicId uint64
	if data.NextTopicId == 0 {
		// reserve topic ID 0 for future use
		if _, err := k.IncrementTopicId(ctx); err != nil {
			return errors.Wrap(err, "error incrementing topic ID")
		}
	} else {
		if err := k.SetNextTopicId(ctx, data.NextTopicId); err != nil {
			return errors.Wrap(err, "error setting next topic ID")
		}
	}
	// Topics []*TopicIdAndTopic
	for _, topic := range data.Topics {
		if topic != nil {
			if topic.Topic == nil {
				return errors.Wrap(types.ErrInvalidValue, "topic cannot be nil")
			}
			if err := k.SetTopic(ctx, topic.TopicId, *topic.Topic); err != nil {
				return errors.Wrap(err, "error setting topic")
			}
		}
	}
	// ActiveTopics []uint64
	for _, topicId := range data.ActiveTopics {
		if err := types.ValidateTopicId(topicId); err != nil {
			return errors.Wrapf(err, "error setting activeTopics %v", data.ActiveTopics)
		}
		if err := k.activeTopics.Set(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting activeTopics")
		}
	}
	// RewardableTopics []uint64
	for _, topicId := range data.RewardableTopics {
		if err := k.SetRewardableTopic(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting rewardableTopics")
		}
	}
	// TopicRewardNonce []*TopicIdAndBlockHeight
	for _, topicIdAndBlockHeight := range data.TopicRewardNonce {
		if topicIdAndBlockHeight != nil {
			if err := k.SetTopicRewardNonce(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting topicRewardNonce")
			}
		}
	}

	// TopicFeeRevenue []*TopicIdAndInt
	for _, topicIdAndInt := range data.TopicFeeRevenue {
		if topicIdAndInt != nil {
			if err := types.ValidateTopicId(topicIdAndInt.TopicId); err != nil {
				return errors.Wrap(err, "topic id validation failed")
			}
			if err := types.ValidateSdkIntRepresentingMonetaryValue(topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "topic fee revenue validation failed")
			}
			if err := k.topicFeeRevenue.Set(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "error setting topicFeeRevenue")
			}
		}
	}

	// PreviousTopicWeight []*TopicIdAndDec
	for _, topicIdAndDec := range data.PreviousTopicWeight {
		if topicIdAndDec != nil {
			if err := k.SetPreviousTopicWeight(
				ctx,
				topicIdAndDec.TopicId,
				topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicWeight")
			}
		}
	}

	// LastDripBlock []*TopicIdAndBlockHeight
	for _, topicIdAndBlockHeight := range data.LastDripBlock {
		if topicIdAndBlockHeight != nil {
			if err := k.SetLastDripBlock(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting lastDripBlock")
			}
		}
	}

	// TopicLastWorkerCommit []*TopicIdTimestampedActorNonce
	for _, topicIdTimestampedActorNonce := range data.TopicLastWorkerCommit {
		if topicIdTimestampedActorNonce != nil {
			if topicIdTimestampedActorNonce.TimestampedActorNonce == nil {
				return errors.Wrap(types.ErrInvalidValue, "topic last worker commit cannot be nil")
			}
			if err := k.SetWorkerTopicLastCommit(ctx,
				topicIdTimestampedActorNonce.TopicId,
				topicIdTimestampedActorNonce.TimestampedActorNonce.BlockHeight,
				topicIdTimestampedActorNonce.TimestampedActorNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastWorkerCommit")
			}
		}
	}
	// TopicLastReputerCommit []*TopicIdTimestampedActorNonce
	for _, topicIdTimestampedActorNonce := range data.TopicLastReputerCommit {
		if topicIdTimestampedActorNonce != nil {
			if topicIdTimestampedActorNonce.TimestampedActorNonce == nil {
				return errors.Wrap(types.ErrInvalidValue, "topic last reputer commit cannot be nil")
			}
			if err := k.SetReputerTopicLastCommit(ctx,
				topicIdTimestampedActorNonce.TopicId,
				topicIdTimestampedActorNonce.TimestampedActorNonce.BlockHeight,
				topicIdTimestampedActorNonce.TimestampedActorNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastReputerCommit")
			}
		}
	}

	// TopicToNextPossibleChurningBlock []*TopicIdAndBlockHeight
	for _, topicBlock := range data.TopicToNextPossibleChurningBlock {
		if topicBlock != nil {
			if err := k.SetTopicToNextPossibleChurningBlock(ctx,
				topicBlock.TopicId,
				topicBlock.BlockHeight); err != nil {
				return errors.Wrapf(err, "error setting topicToNextPossibleChurningBlock %v", topicBlock)
			}
		}
	}

	// BlockToActiveTopics []*BlockHeightTopicIds
	for _, blockToActiveTopics := range data.BlockToActiveTopics {
		if blockToActiveTopics != nil {
			if blockToActiveTopics.TopicIds == nil {
				return errors.Wrap(types.ErrInvalidValue, "block to active topics cannot be nil")
			}
			if err := k.blockToActiveTopics.Set(ctx,
				blockToActiveTopics.BlockHeight,
				*blockToActiveTopics.TopicIds); err != nil {
				return errors.Wrap(err, "error setting blockToActiveTopics")
			}
		}
	}

	// BlockToLowestActiveTopicWeight []*BlockHeightTopicIdWeightPair
	for _, lowestActiveTopicWeight := range data.BlockToLowestActiveTopicWeight {
		if lowestActiveTopicWeight != nil {
			if lowestActiveTopicWeight.TopicWeight == nil {
				return errors.Wrap(types.ErrInvalidValue, "block to lowest active topic weight cannot be nil")
			}
			if err := k.blockToLowestActiveTopicWeight.Set(ctx,
				lowestActiveTopicWeight.BlockHeight,
				*lowestActiveTopicWeight.TopicWeight); err != nil {
				return errors.Wrap(err, "error setting blockToLowestActiveTopicWeight")
			}
		}
	}

	// TotalSumPreviousTopicWeights
	if data.TotalSumPreviousTopicWeights.Gt(alloraMath.ZeroDec()) {
		if err := k.SetTotalSumPreviousTopicWeights(ctx, data.TotalSumPreviousTopicWeights); err != nil {
			return errors.Wrap(err, "error setting TotalSumPreviousTopicWeights")
		}
	} else {
		if err := k.SetTotalSumPreviousTopicWeights(ctx, alloraMath.ZeroDec()); err != nil {
			return errors.Wrap(err, "error setting TotalSumPreviousTopicWeights to zero int")
		}
	}

	// LastMedianInferences []*TopicIdAndDec
	for _, topicIdAndDec := range data.LastMedianInferences {
		if topicIdAndDec != nil {
			if err := k.SetLastMedianInferences(
				ctx,
				topicIdAndDec.TopicId,
				topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting lastMedianInferences")
			}
		}
	}

	// MadInferences []*TopicIdAndDec
	for _, topicIdDec := range data.MadInferences {
		if topicIdDec != nil {
			if err := k.SetMadInferences(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting madInferences")
			}
		}
	}

	// topicLabelRegistry []EpochLabelRegistry
	for _, registry := range data.EpochLabelRegistries {
		if registry != nil {
			key := collections.Join(registry.TopicId, BlockHeight(registry.EpochId))
			if err := k.topicLabelRegistry.Set(ctx, key, *registry); err != nil {
				return errors.Wrap(err, "error setting topicLabelRegistry")
			}
		}
	}

	return nil
}

func (k *TopicKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// nextTopicId
	nextTopicId, err := k.nextTopicId.Peek(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get next topic ID")
	}
	data.NextTopicId = nextTopicId

	// topics
	topicsIter, err := k.topics.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topics")
	}
	topics := make([]*types.TopicIdAndTopic, 0)
	for ; topicsIter.Valid(); topicsIter.Next() {
		keyValue, err := topicsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicsIter")
		}
		value := keyValue.Value
		topic := types.TopicIdAndTopic{
			TopicId: keyValue.Key,
			Topic:   &value,
		}
		topics = append(topics, &topic)
	}
	data.Topics = topics

	// activeTopics
	activeTopics := make([]uint64, 0)
	activeTopicsIter, err := k.activeTopics.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate active topics")
	}
	for ; activeTopicsIter.Valid(); activeTopicsIter.Next() {
		key, err := activeTopicsIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: activeTopicsIter")
		}
		activeTopics = append(activeTopics, key)
	}
	data.ActiveTopics = activeTopics

	// topicToNextPossibleChurningBlock
	topicNextChurningBlock, err := k.topicToNextPossibleChurningBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topicToNextPossibleChurningBlock")
	}
	topicToNextPossibleChurningBlock := make([]*types.TopicIdAndBlockHeight, 0)
	for ; topicNextChurningBlock.Valid(); topicNextChurningBlock.Next() {
		keyValue, err := topicNextChurningBlock.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicToNextPossibleChurningBlock")
		}
		value := keyValue.Value
		topic := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: value,
		}
		topicToNextPossibleChurningBlock = append(topicToNextPossibleChurningBlock, &topic)
	}
	data.TopicToNextPossibleChurningBlock = topicToNextPossibleChurningBlock

	// blockToActiveTopics
	blockActiveTopics, err := k.blockToActiveTopics.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate blockActiveTopics")
	}
	blockHeightTopicIds := make([]*types.BlockHeightTopicIds, 0)
	for ; blockActiveTopics.Valid(); blockActiveTopics.Next() {
		keyValue, err := blockActiveTopics.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: blockActiveTopics")
		}
		value := keyValue.Value
		topic := types.BlockHeightTopicIds{
			BlockHeight: keyValue.Key,
			TopicIds:    &value,
		}
		blockHeightTopicIds = append(blockHeightTopicIds, &topic)
	}
	data.BlockToActiveTopics = blockHeightTopicIds

	// blockToLowestActiveTopicWeight
	lowestActiveTopic, err := k.blockToLowestActiveTopicWeight.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate blockActiveTopics")
	}
	blockHeightTopicIdWeight := make([]*types.BlockHeightTopicIdWeightPair, 0)
	for ; lowestActiveTopic.Valid(); lowestActiveTopic.Next() {
		keyValue, err := lowestActiveTopic.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: blockActiveTopics")
		}
		value := keyValue.Value
		topic := types.BlockHeightTopicIdWeightPair{
			BlockHeight: keyValue.Key,
			TopicWeight: &value,
		}
		blockHeightTopicIdWeight = append(blockHeightTopicIdWeight, &topic)
	}
	data.BlockToLowestActiveTopicWeight = blockHeightTopicIdWeight

	// rewardableTopics
	rewardableTopics := make([]uint64, 0)
	rewardableTopicsIter, err := k.rewardableTopics.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate rewardable topics")
	}
	for ; rewardableTopicsIter.Valid(); rewardableTopicsIter.Next() {
		key, err := rewardableTopicsIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: rewardableTopicsIter")
		}
		rewardableTopics = append(rewardableTopics, key)
	}
	data.RewardableTopics = rewardableTopics

	// topicRewardNonce
	topicRewardNonce := make([]*types.TopicIdAndBlockHeight, 0)
	topicRewardNonceIter, err := k.topicRewardNonce.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic reward nonce")
	}
	for ; topicRewardNonceIter.Valid(); topicRewardNonceIter.Next() {
		keyValue, err := topicRewardNonceIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicRewardNonceIter")
		}
		topicIdAndBlockHeight := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: keyValue.Value,
		}
		topicRewardNonce = append(topicRewardNonce, &topicIdAndBlockHeight)
	}
	data.TopicRewardNonce = topicRewardNonce

	// topicFeeRevenue
	topicFeeRevenue := make([]*types.TopicIdAndInt, 0)
	topicFeeRevenueIter, err := k.topicFeeRevenue.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic fee revenue")
	}
	for ; topicFeeRevenueIter.Valid(); topicFeeRevenueIter.Next() {
		keyValue, err := topicFeeRevenueIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicFeeRevenueIter")
		}
		topicIdAndInt := types.TopicIdAndInt{
			TopicId: keyValue.Key,
			Int:     keyValue.Value,
		}
		topicFeeRevenue = append(topicFeeRevenue, &topicIdAndInt)
	}
	data.TopicFeeRevenue = topicFeeRevenue

	// previousTopicWeight
	previousTopicWeight := make([]*types.TopicIdAndDec, 0)
	previousTopicWeightIter, err := k.previousTopicWeight.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate previous topic weight")
	}
	for ; previousTopicWeightIter.Valid(); previousTopicWeightIter.Next() {
		keyValue, err := previousTopicWeightIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: previousTopicWeightIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicWeight = append(previousTopicWeight, &topicIdAndDec)
	}
	data.PreviousTopicWeight = previousTopicWeight

	// lastDripBlock
	lastDripBlock := make([]*types.TopicIdAndBlockHeight, 0)
	lastDripBlockIter, err := k.lastDripBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate last drip block")
	}
	for ; lastDripBlockIter.Valid(); lastDripBlockIter.Next() {
		keyValue, err := lastDripBlockIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: lastDripBlockIter")
		}
		topicIdAndBlockHeight := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: keyValue.Value,
		}
		lastDripBlock = append(lastDripBlock, &topicIdAndBlockHeight)
	}
	data.LastDripBlock = lastDripBlock

	// topicLastWorkerCommit
	topicLastWorkerCommit := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastWorkerCommitIter, err := k.topicLastWorkerCommit.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic last worker commit")
	}
	for ; topicLastWorkerCommitIter.Valid(); topicLastWorkerCommitIter.Next() {
		keyValue, err := topicLastWorkerCommitIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicLastWorkerCommitIter")
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastWorkerCommit = append(topicLastWorkerCommit, &topicIdTimestampedActorNonce)
	}
	data.TopicLastWorkerCommit = topicLastWorkerCommit

	// topicLastReputerCommit
	topicLastReputerCommit := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastReputerCommitIter, err := k.topicLastReputerCommit.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic last reputer commit")
	}
	for ; topicLastReputerCommitIter.Valid(); topicLastReputerCommitIter.Next() {
		keyValue, err := topicLastReputerCommitIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicLastReputerCommitIter")
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastReputerCommit = append(topicLastReputerCommit, &topicIdTimestampedActorNonce)
	}
	data.TopicLastReputerCommit = topicLastReputerCommit

	// totalSumPreviousTopicWeights
	totalSumPreviousTopicWeights, err := k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get total sum previous topic weights")
	}
	data.TotalSumPreviousTopicWeights = totalSumPreviousTopicWeights

	// lastMedianInferences
	lastMedianInferences := make([]*types.TopicIdAndDec, 0)
	lastMedianInferencesIter, err := k.lastMedianInferences.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate last median inferences")
	}
	for ; lastMedianInferencesIter.Valid(); lastMedianInferencesIter.Next() {
		keyValue, err := lastMedianInferencesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: lastMedianInferencesIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		lastMedianInferences = append(lastMedianInferences, &topicIdAndDec)
	}
	data.LastMedianInferences = lastMedianInferences

	// madInferences
	madInferences := make([]*types.TopicIdAndDec, 0)
	madInferencesIter, err := k.madInferences.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate last mad inferences")
	}
	for ; madInferencesIter.Valid(); madInferencesIter.Next() {
		keyValue, err := madInferencesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: MadInferencesIter")
		}
		madInferences = append(madInferences, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
	}
	data.MadInferences = madInferences

	// labelRegistries
	labelRegistries := make([]*types.EpochLabelRegistry, 0)
	labelRegistriesIter, err := k.topicLabelRegistry.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic label registries")
	}
	for ; labelRegistriesIter.Valid(); labelRegistriesIter.Next() {
		keyValue, err := labelRegistriesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: LabelRegistriesIter")
		}
		labelRegistries = append(labelRegistries, &types.EpochLabelRegistry{
			TopicId: keyValue.Key.K1(),
			EpochId: uint64(keyValue.Key.K2()),
			Labels:  keyValue.Value.GetLabels(),
		})
	}
	data.EpochLabelRegistries = labelRegistries

	return nil
}
