package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewTopicKeeper(
	cdc codec.BinaryCodec,
	sb *collections.SchemaBuilder,
	paramsKeeper *ParamsKeeper,
	nonceKeeper *NonceKeeper,
	stakingKeeper *StakingKeeper,
) *TopicKeeper {
	return &TopicKeeper{
		nextTopicId:                      collections.NewSequence(sb, types.NextTopicIdKey, "next_TopicId"),
		topics:                           collections.NewMap(sb, types.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[types.Topic](cdc)),
		activeTopics:                     collections.NewKeySet(sb, types.ActiveTopicsKey, "active_topics", collections.Uint64Key),
		topicToNextPossibleChurningBlock: collections.NewMap(sb, types.TopicToNextPossibleChurningBlockKey, "topic_to_next_possible_churning_block", collections.Uint64Key, collections.Int64Value),
		blockToActiveTopics:              collections.NewMap(sb, types.BlockToActiveTopicsKey, "block_to_active_topics", collections.Int64Key, codec.CollValue[types.TopicIds](cdc)),
		blockToLowestActiveTopicWeight:   collections.NewMap(sb, types.BlockToLowestActiveTopicWeightKey, "block_to_lowest_active_topic_weight", collections.Int64Key, codec.CollValue[types.TopicIdWeightPair](cdc)),
		rewardableTopics:                 collections.NewKeySet(sb, types.RewardableTopicsKey, "rewardable_topics", collections.Uint64Key),
		topicRewardNonce:                 collections.NewMap(sb, types.TopicRewardNonceKey, "topic_reward_nonce", collections.Uint64Key, collections.Int64Value),
		previousTopicWeight:              collections.NewMap(sb, types.PreviousTopicWeightKey, "previous_topic_weight", collections.Uint64Key, alloraMath.DecValue),
		lastMedianInferences:             collections.NewMap(sb, types.LastMedianInferencesKey, "last_median_inferences", collections.Uint64Key, alloraMath.DecValue),
		madInferences:                    collections.NewMap(sb, types.MadInferencesKey, "mad_inferences", collections.Uint64Key, alloraMath.DecValue),
		totalSumPreviousTopicWeights:     collections.NewItem(sb, types.TotalSumPreviousTopicWeightsKey, "total_sum_previous_topic_weights", alloraMath.DecValue),
		lastDripBlock:                    collections.NewMap(sb, types.LastDripBlockKey, "last_drip_block", collections.Uint64Key, collections.Int64Value),
		topicFeeRevenue:                  collections.NewMap(sb, types.TopicFeeRevenueKey, "topic_fee_revenue", collections.Uint64Key, sdk.IntValue),
		topicLastWorkerCommit:            collections.NewMap(sb, types.TopicLastWorkerCommitKey, "topic_last_worker_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		topicLastReputerCommit:           collections.NewMap(sb, types.TopicLastReputerCommitKey, "topic_last_reputer_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		paramsKeeper:                     paramsKeeper,
		nonceKeeper:                      nonceKeeper,
		stakingKeeper:                    stakingKeeper,
	}
}

const RESERVED_BLOCK = 0

type TopicKeeper struct {
	// the next topic id to be used, equal to the number of topics that have been created
	nextTopicId collections.Sequence
	// every topic that has been created indexed by their topicId starting from 1 (0 is reserved for the root network)
	topics collections.Map[TopicId, types.Topic]
	// which topics are active at the current block height
	activeTopics collections.KeySet[TopicId]
	// topic to next possible churning block
	// if topic not included, then the topic is not active
	topicToNextPossibleChurningBlock collections.Map[TopicId, BlockHeight]
	// block height to active topics whose epochs reset at that block
	blockToActiveTopics collections.Map[BlockHeight, types.TopicIds]
	// block to lowest weight of any topic that is active and whose epoch expires in that block
	blockToLowestActiveTopicWeight collections.Map[BlockHeight, types.TopicIdWeightPair]
	// every topic that has been churned and ready to be rewarded i.e. reputer losses have been committed
	rewardableTopics collections.KeySet[TopicId]
	// map of (topic) -> nonce/block height
	topicRewardNonce collections.Map[TopicId, BlockHeight]
	// store previous weights for exponential moving average in rewards calc
	previousTopicWeight collections.Map[TopicId, alloraMath.Dec]
	// Last median per topic
	lastMedianInferences collections.Map[TopicId, alloraMath.Dec]
	// MAD (Median Absolute Deviation) per topic
	madInferences collections.Map[TopicId, alloraMath.Dec]
	// total weights of all active topics on the network
	totalSumPreviousTopicWeights collections.Item[alloraMath.Dec]
	// map of (topic) -> last dripped block
	lastDripBlock collections.Map[TopicId, BlockHeight]
	// fee revenue collected by a topic over the course of the last reward cadence
	topicFeeRevenue        collections.Map[TopicId, cosmosMath.Int]
	topicLastWorkerCommit  collections.Map[TopicId, types.TimestampedActorNonce]
	topicLastReputerCommit collections.Map[TopicId, types.TimestampedActorNonce]
	// params keeper
	paramsKeeper *ParamsKeeper
	// nonce keeper
	nonceKeeper *NonceKeeper
	// staking keeper
	stakingKeeper *StakingKeeper
}

// Get the previous weight during rewards calculation for a topic
// Returns ((0,0), true) if there was no prior topic weight set, else ((x,y), false) where x,y!=0
func (k *TopicKeeper) GetPreviousTopicWeight(ctx context.Context, topicId TopicId) (topicWeight alloraMath.Dec, noPrior bool, err error) {
	topicWeight, err = k.previousTopicWeight.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), true, nil
	} else if err != nil {
		return alloraMath.ZeroDec(), false, errorsmod.Wrap(err, "error getting previous topic weight")
	}
	return topicWeight, false, nil
}

// Set the previous weight during rewards calculation for a topic
// This function also updates the total sum of previous topic weights
func (k *TopicKeeper) SetPreviousTopicWeight(ctx context.Context, topicId TopicId, weight alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(weight); err != nil {
		return errorsmod.Wrap(err, "weight validation failed")
	}

	// First update total because it uses previous weight value
	err := k.UpdateTotalSumPreviousTopicWeights(ctx, topicId, weight)
	if err != nil {
		return errorsmod.Wrap(err, "error updating total sum of previous topic weights")
	}

	// Then update the previous weight
	err = k.previousTopicWeight.Set(ctx, topicId, weight)
	if err != nil {
		return err
	}

	return nil
}

// Getter and setter for lastMedianInferences
func (k *TopicKeeper) GetLastMedianInferences(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	medianInferences, err := k.lastMedianInferences.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrap(err, "error getting last median inferences")
	}
	return medianInferences, nil
}

func (k *TopicKeeper) SetLastMedianInferences(ctx context.Context, topicId TopicId, medianInferences alloraMath.Dec) error {
	if err := types.ValidateDec(medianInferences); err != nil {
		return errorsmod.Wrap(err, "median inferences validation failed")
	}
	return k.lastMedianInferences.Set(ctx, topicId, medianInferences)
}

// Getter and setter for madInferences
func (k *TopicKeeper) GetMadInferences(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	madInferences, err := k.madInferences.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.ZeroDec(), errorsmod.Wrap(err, "error getting last mad inferences")
	}
	return madInferences, nil
}

func (k *TopicKeeper) SetMadInferences(ctx context.Context, topicId TopicId, madInferences alloraMath.Dec) error {
	if err := types.ValidateDec(madInferences); err != nil {
		return errorsmod.Wrap(err, "mad inferences validation failed")
	}
	return k.madInferences.Set(ctx, topicId, madInferences)
}

// UpdateTotalSumPreviousTopicWeights updates the total sum of previous topic weights
// by subtracting the old weight if any, and adding the new weight for the given topicId.
func (k *TopicKeeper) UpdateTotalSumPreviousTopicWeights(ctx context.Context, topicId TopicId, newWeight alloraMath.Dec) error {
	// Get the current total sum, set to zero if not existent.
	totalSumPreviousTopicWeights, err := k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return errorsmod.Wrap(err, fmt.Sprintf("error getting total sum of previous topic weights while updating topic: %d", topicId))
	}

	// Subtract old weight
	oldTopicWeight, _, err := k.GetPreviousTopicWeight(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, fmt.Sprintf("error getting previous weight of topic: %d", topicId))
	}
	totalSumPreviousTopicWeights, err = totalSumPreviousTopicWeights.Sub(oldTopicWeight)
	if err != nil {
		return errorsmod.Wrap(err, "error subtracting old weight from total sum of previous topic weights")
	}

	// Add new weight
	totalSumPreviousTopicWeights, err = totalSumPreviousTopicWeights.Add(newWeight)
	if err != nil {
		return errorsmod.Wrap(err, "error adding new weight to total sum of previous topic weights")
	}

	// Set the new total sum
	return k.SetTotalSumPreviousTopicWeights(ctx, totalSumPreviousTopicWeights)
}

// Get the total sum of previous topic weights
func (k *TopicKeeper) GetTotalSumPreviousTopicWeights(ctx context.Context) (alloraMath.Dec, error) {
	sumPreviousWeights, err := k.totalSumPreviousTopicWeights.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.ZeroDec(), nil
	} else if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting topic fee revenue")
	}

	return sumPreviousWeights, nil
}

// Set the total sum of previous topic weights
func (k *TopicKeeper) SetTotalSumPreviousTopicWeights(ctx context.Context, weight alloraMath.Dec) error {
	if err := types.ValidateDec(weight); err != nil {
		return errorsmod.Wrap(err, "previous total topic weight validation failed")
	}
	if weight.Lt(alloraMath.ZeroDec()) {
		return errorsmod.Wrap(types.ErrInvalidValue, "total sum of previous topic weights cannot be negative")
	}
	return k.totalSumPreviousTopicWeights.Set(ctx, weight)
}

// Gets next topic id
func (k *TopicKeeper) IncrementTopicId(ctx context.Context) (TopicId, error) {
	return k.nextTopicId.Next(ctx)
}

// Gets topic by topicId
func (k *TopicKeeper) GetTopic(ctx context.Context, topicId TopicId) (types.Topic, error) {
	topic, err := k.topics.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return types.Topic{}, types.ErrTopicDoesNotExist
	} else if err != nil {
		return types.Topic{}, err
	}
	return topic, nil
}

// Sets a topic config on a topicId
func (k *TopicKeeper) SetTopic(ctx context.Context, topicId TopicId, topic types.Topic) error {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}
	if err := topic.Validate(params); err != nil {
		return errorsmod.Wrap(err, "set topic validation failure")
	}
	return k.topics.Set(ctx, topicId, topic)
}

// Checks if a topic exists
func (k *TopicKeeper) TopicExists(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topics.Has(ctx, topicId)
}

// UpdateTopic applies allowed changes to a topic.
func (k *TopicKeeper) UpdateTopic(ctx context.Context, topic types.Topic, updatedTopic types.Topic) (types.Topic, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "error getting params")
	}

	if err := updatedTopic.Validate(params); err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "updated topic validation failed")
	}

	// Check merit_sortition_alpha update restriction when topic is active and worker window is open
	if !topic.MeritSortitionAlpha.Equal(updatedTopic.MeritSortitionAlpha) {
		isActive, err := k.IsTopicActive(ctx, topic.Id)
		if err != nil {
			return types.Topic{}, errorsmod.Wrap(err, "failed to check topic active status")
		}
		if isActive {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			blockHeight := sdkCtx.BlockHeight()
			nonces, err := k.nonceKeeper.GetUnfulfilledWorkerNonces(ctx, topic.Id)
			if err != nil {
				return types.Topic{}, errorsmod.Wrap(err, "failed to get unfulfilled worker nonces")
			}
			if len(nonces.Nonces) > 0 {
				lastNonce := nonces.Nonces[0]
				withinWindow, err := BlockWithinWorkerSubmissionWindowOfNonce(topic, *lastNonce, blockHeight)
				if err != nil {
					return types.Topic{}, errorsmod.Wrap(err, "failed to check worker submission window")
				}
				if withinWindow {
					return types.Topic{}, errorsmod.Wrap(types.ErrWorkerNonceWindowNotAvailable, "cannot update merit_sortition_alpha while worker window is open")
				}
			}
		}
	}

	if err := k.SetTopic(ctx, topic.Id, updatedTopic); err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "failed to apply topic update")
	}

	return updatedTopic, nil
}

// Returns the number of topics that are active in the network
func (k *TopicKeeper) GetNextTopicId(ctx context.Context) (TopicId, error) {
	return k.nextTopicId.Peek(ctx)
}

// Check if the topic is activated or not
func (k *TopicKeeper) IsTopicActive(ctx context.Context, topicId TopicId) (bool, error) {
	_, active, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting next possible churning block by topic id")
	}

	return active, nil
}

// UpdateTopicInitialRegret updates the InitialRegret for a given topic.
func (k *TopicKeeper) UpdateTopicInitialRegret(ctx context.Context, topicId TopicId, initialRegret alloraMath.Dec) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic")
	}
	topic.InitialRegret = initialRegret
	return k.SetTopic(ctx, topicId, topic)
}

// UpdateTopicInferenceLastRan updates the InferenceLastRan timestamp for a given topic.
func (k *TopicKeeper) UpdateTopicEpochLastEnded(ctx context.Context, topicId TopicId, epochLastEnded BlockHeight) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic")
	}
	topic.EpochLastEnded = epochLastEnded
	return k.SetTopic(ctx, topicId, topic)
}

// wrapper for set operation around activeTopics
// TODO: Evaluate removing this KV store (activeTopics) - not being used
func (k *TopicKeeper) SetActiveTopics(ctx context.Context, topicId TopicId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	return k.activeTopics.Set(ctx, topicId)
}

// wrapper for set operation around blockToActiveTopics
func (k *TopicKeeper) SetBlockToActiveTopics(ctx context.Context, block BlockHeight, topicIds types.TopicIds) error {
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	for _, topicId := range topicIds.TopicIds {
		if err := types.ValidateTopicId(topicId); err != nil {
			return errorsmod.Wrap(err, "topic id validation failed")
		}
	}
	err := k.blockToActiveTopics.Set(ctx, block, topicIds)
	if err != nil {
		return err
	}
	return nil
}

// wrapper for set operation around topicToNextPossibleChurningBlock
func (k *TopicKeeper) SetTopicToNextPossibleChurningBlock(ctx context.Context, topicId TopicId, block BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	return k.topicToNextPossibleChurningBlock.Set(ctx, topicId, block)
}

// wrapper for set operation around blockToLowestActiveTopicWeight
func (k *TopicKeeper) SetBlockToLowestActiveTopicWeight(
	ctx context.Context,
	block BlockHeight,
	weightPair types.TopicIdWeightPair,
) error {
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := types.ValidateTopicId(weightPair.TopicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(weightPair.Weight); err != nil {
		return errorsmod.Wrap(err, "weight validation failed")
	}
	return k.blockToLowestActiveTopicWeight.Set(ctx, block, weightPair)
}

// Boolean true if topic is active, else false
func (k *TopicKeeper) GetNextPossibleChurningBlockByTopicId(ctx context.Context, topicId TopicId) (BlockHeight, bool, error) {
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	block, err := k.topicToNextPossibleChurningBlock.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return RESERVED_BLOCK, false, nil
	} else if err != nil {
		return RESERVED_BLOCK, false, err
	}
	return block, block >= currentBlock, nil
}

// UpdateTopicWeightAfterStakeChange updates the topic weight and total sum of previous topic weights
// after a stake change occurs, if there is no prior topic weight. This is used by both RemoveReputerStake and RemoveDelegateStake.
func (k *TopicKeeper) UpdateTopicWeightAfterStakeChange(
	ctx context.Context,
	topicId TopicId,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get the current topic weight before updating stake
	_, noPrior, err := k.GetPreviousTopicWeight(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting previous topic weight")
	}

	// If there is no prior topic weight, do nothing
	if noPrior {
		return nil
	}

	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic")
	}

	// Get params for weight calculation
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}

	// Calculate the new weight based on updated stake
	newWeight, topicFeeRevenue, topicStake, err := k.GetCurrentTopicWeight(
		ctx,
		topicId,
		topic.EpochLength,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating new topic weight")
	}

	// Update previous topic weight and total sum
	if err := k.SetPreviousTopicWeight(ctx, topicId, newWeight); err != nil {
		return errorsmod.Wrapf(err, "Setting previous topic weight failed")
	}

	// Emit topic weight updated event
	types.EmitNewTopicWeightUpdatedEvent(ctx, topicId, newWeight, topicStake, topicFeeRevenue)

	sdkCtx.Logger().Debug("Updated topic weight after stake change", "topicId", topicId, "newWeight", newWeight.String())

	return nil
}

// Remove the inferences that are outliers
func (k *TopicKeeper) FilterOutlierResistantInferences(ctx context.Context, topicId TopicId, inferences types.Inferences) (types.Inferences, error) {
	lastMedian, err := k.GetLastMedianInferences(ctx, topicId)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting last median inferences")
	}
	if lastMedian.IsZero() {
		return inferences, nil
	}
	mad, err := k.GetMadInferences(ctx, topicId)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting mad inferences")
	}
	if mad.IsZero() {
		return inferences, nil
	}

	filteredInferences := types.Inferences{
		Inferences: []*types.Inference{},
	}

	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting params")
	}
	outlierThresholdMultiplier := params.InferenceOutlierDetectionThreshold
	for _, inf := range inferences.Inferences {
		// Calculate absolute difference from median
		diff, err := inf.Value.Sub(lastMedian)
		if err != nil {
			return types.Inferences{}, errorsmod.Wrap(err, "error getting difference from median")
		}
		absDiff, err := diff.Abs()
		if err != nil {
			return types.Inferences{}, errorsmod.Wrap(err, "error getting absolute difference")
		}

		thresholdMad, err := outlierThresholdMultiplier.Mul(mad)
		if err != nil {
			return types.Inferences{}, errorsmod.Wrap(err, "error getting threshold mad")
		}
		// Check if within threshold
		if absDiff.Lte(thresholdMad) {
			filteredInferences.Inferences = append(filteredInferences.Inferences, inf)
		}
	}

	return filteredInferences, nil
}

// TOPIC REWARD NONCE

// GetTopicRewardNonce retrieves the reward nonce for a given topic ID.
func (k *TopicKeeper) GetTopicRewardNonce(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	nonce, err := k.topicRewardNonce.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return 0, nil // Return 0 if not found
	} else if err != nil {
		return 0, errorsmod.Wrap(err, "error getting topic reward nonce")
	}
	return nonce, nil
}

// SetTopicRewardNonce sets the reward nonce for a given topic ID.
func (k *TopicKeeper) SetTopicRewardNonce(ctx context.Context, topicId TopicId, nonce BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce validation failed")
	}
	return k.topicRewardNonce.Set(ctx, topicId, nonce)
}

// DeleteTopicRewardNonce removes the reward nonce entry for a given topic ID.
func (k *TopicKeeper) DeleteTopicRewardNonce(ctx context.Context, topicId TopicId) error {
	return k.topicRewardNonce.Remove(ctx, topicId)
}

// return the last time we dripped the fee revenue for a topic
func (k *TopicKeeper) GetLastDripBlock(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	bh, err := k.lastDripBlock.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, errorsmod.Wrap(err, "error getting last drip block")
	}
	return bh, nil
}

// set the last time we dripped the fee revenue for a topic
func (k *TopicKeeper) SetLastDripBlock(ctx context.Context, topicId TopicId, block BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	return k.lastDripBlock.Set(ctx, topicId, block)
}

// / TOPIC FEE REVENUE

// Get the amount of fee revenue collected by a topic
func (k *TopicKeeper) GetTopicFeeRevenue(ctx context.Context, topicId TopicId) (cosmosMath.Int, error) {
	feeRev, err := k.topicFeeRevenue.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.ZeroInt(), nil
	} else if err != nil {
		return cosmosMath.ZeroInt(), errorsmod.Wrap(err, "error getting topic fee revenue")
	}
	return feeRev, nil
}

// Add to the fee revenue collected by a topic
func (k *TopicKeeper) AddTopicFeeRevenue(ctx context.Context, topicId TopicId, amount cosmosMath.Int) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(amount); err != nil {
		return errorsmod.Wrap(err, "amount validation failed")
	}
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic fee revenue")
	}
	topicFeeRevenue = topicFeeRevenue.Add(amount)
	if err = k.topicFeeRevenue.Set(ctx, topicId, topicFeeRevenue); err != nil {
		return errorsmod.Wrap(err, "error setting topic fee revenue")
	}

	return nil
}

// Drop the fee revenue by the global Ecosystem bucket drip amount
// in the paper we say that
// ∆ C_{t,i} = N_{epochs,w} * C_{t,i}
// where C_{t,i} is the topic fee revenue
// and N_{epochs,w} is the number of epochs per week
// and this decay or drip happens each epoch
func (k *TopicKeeper) DripTopicFeeRevenue(ctx sdk.Context, topic types.Topic, blocksPerWeek alloraMath.Dec, block BlockHeight) error {
	if err := types.ValidateTopicId(topic.Id); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic fee revenue")
	}
	topicFeeRevenueDec, err := alloraMath.NewDecFromSdkInt(topicFeeRevenue)
	if err != nil {
		return errorsmod.Wrap(err, "error creating decimal from sdk int")
	}
	blocksPerEpoch := alloraMath.NewDecFromInt64(topic.EpochLength)
	if blocksPerEpoch.IsZero() {
		ctx.Logger().Warn("Blocks per epoch is zero for topic", "topicId", topic.Id)
		return nil
	}
	epochsPerWeek, err := blocksPerWeek.Quo(blocksPerEpoch)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating epochs per week")
	}
	if epochsPerWeek.IsZero() {
		// Log a warning
		ctx.Logger().Warn("Epochs per week is zero for topic", "topicId", topic.Id)
		return nil
	}
	// this delta is the drip per epoch
	dripPerEpoch, err := topicFeeRevenueDec.Quo(epochsPerWeek)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating drip per epoch")
	}
	lastDripBlock, err := k.GetLastDripBlock(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting last drip block")
	}
	// if we have not yet decayed this epoch, decay and set to decayed
	// if we have decayed this epoch already, do nothing and continue
	if lastDripBlock <= topic.EpochLastEnded {
		newTopicFeeRevenueDec, err := topicFeeRevenueDec.Sub(dripPerEpoch)
		if err != nil {
			return errorsmod.Wrap(err, "error subtracting drip per epoch")
		}
		if newTopicFeeRevenueDec.IsNegative() {
			newTopicFeeRevenueDec = alloraMath.ZeroDec()
		}

		newTopicFeeRevenue, err := newTopicFeeRevenueDec.SdkIntTrim()
		if err != nil {
			return errorsmod.Wrap(err, "error converting decimal to sdk int")
		}

		if err = k.SetLastDripBlock(ctx, topic.Id, topic.EpochLastEnded); err != nil {
			return errorsmod.Wrap(err, "error setting last drip block")
		}
		ctx.Logger().Debug("Dripping topic fee revenue",
			"block", ctx.BlockHeight(),
			"topicId", topic.Id,
			"oldRevenue", topicFeeRevenue,
			"newRevenue", newTopicFeeRevenue)
		if err = types.ValidateSdkIntRepresentingMonetaryValue(newTopicFeeRevenue); err != nil {
			return errorsmod.Wrap(err, "error validating new topic fee revenue")
		}

		// Emit event using the calculated dripPerEpoch
		dripAmountInt, err := dripPerEpoch.SdkIntTrim()
		if err != nil {
			ctx.Logger().Warn("Error converting drip per epoch to sdk int to emit topic fee revenue dripped event", "error", err)
		}
		types.EmitNewTopicFeeRevenueDrippedEvent(ctx, topic.Id, topicFeeRevenue, newTopicFeeRevenue, dripAmountInt)

		return k.topicFeeRevenue.Set(ctx, topic.Id, newTopicFeeRevenue)
	}
	return nil
}

func (k *TopicKeeper) SetWorkerTopicLastCommit(
	ctx context.Context,
	topic types.TopicId,
	blockHeight int64,
	nonce *types.Nonce,
) error {
	if err := types.ValidateTopicId(topic); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrap(err, "error validating block height")
	}
	if err := nonce.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating nonce")
	}
	return k.topicLastWorkerCommit.Set(ctx, topic, types.TimestampedActorNonce{
		BlockHeight: blockHeight,
		Nonce:       nonce,
	})
}

func (k *TopicKeeper) SetReputerTopicLastCommit(
	ctx context.Context,
	topic types.TopicId,
	blockHeight int64,
	nonce *types.Nonce,
) error {
	if err := types.ValidateTopicId(topic); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrap(err, "error validating block height")
	}
	if err := nonce.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating nonce")
	}
	return k.topicLastReputerCommit.Set(ctx, topic, types.TimestampedActorNonce{
		BlockHeight: blockHeight,
		Nonce:       nonce,
	})
}

func (k *TopicKeeper) GetWorkerTopicLastCommit(ctx context.Context, topic TopicId) (types.TimestampedActorNonce, error) {
	return k.topicLastWorkerCommit.Get(ctx, topic)
}

func (k *TopicKeeper) GetReputerTopicLastCommit(ctx context.Context, topic TopicId) (types.TimestampedActorNonce, error) {
	return k.topicLastReputerCommit.Get(ctx, topic)
}
