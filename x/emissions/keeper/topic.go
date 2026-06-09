package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
		topicLabelRegistry:               collections.NewMap(sb, types.TopicLabelRegistryKey, "topic_label_registry", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.EpochLabelRegistry](cdc)),
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
	// topic epoch label registry - keyed by [topic_id, nonce]
	topicLabelRegistry collections.Map[collections.Pair[TopicId, BlockHeight], types.EpochLabelRegistry]
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

func (k *TopicKeeper) SetRewardableTopic(ctx context.Context, topicId TopicId) error {
	return k.rewardableTopics.Set(ctx, topicId)
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
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting total sum of previous topic weights")
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

// Sets next topic id
func (k *TopicKeeper) SetNextTopicId(ctx context.Context, topicId TopicId) error {
	return k.nextTopicId.Set(ctx, topicId)
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

// Sets a topic config on a topicId. Canonicalizes LabelWhitelist in place
// (dedup + NFC + trim) so downstream whitelist lookups are exact byte-equality.
func (k *TopicKeeper) SetTopic(ctx context.Context, topicId TopicId, topic types.Topic) error {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}
	if err := types.ValidateTopicLabelWhitelistSize(topic.LabelWhitelist, params.MaxTopicLabelWhitelistSize); err != nil {
		return errorsmod.Wrap(err, "topic label_whitelist size validation failed")
	}
	if len(topic.LabelWhitelist) > 0 {
		canonical, err := types.CanonicalizeLabelList(
			topic.LabelWhitelist,
			true,
			params.MaxCanonicalLabelByteLength,
			topic.LabelCaseSensitive,
		)
		if err != nil {
			return errorsmod.Wrap(err, "topic label_whitelist canonicalization failed")
		}
		topic.LabelWhitelist = canonical
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

// isAnyUnfulfilledWorkerNonceWithinWindow reports true iff the topic is
// active and at least one of its unfulfilled worker nonces is currently
// inside a worker submission window. It is the shared guard for topic
// parameter mutations that must not race worker payload submission for an
// open epoch (merit_sortition_alpha, max_labels_per_submission, label
// whitelist, label default value). Unlike the original v14-era check this
// iterates every unfulfilled nonce rather than just the newest, because workers
// can submit against any currently-open nonce.
func (k *TopicKeeper) isAnyUnfulfilledWorkerNonceWithinWindow(
	ctx context.Context,
	topic types.Topic,
) (bool, error) {
	isActive, err := k.IsTopicActive(ctx, topic.Id)
	if err != nil {
		return false, errorsmod.Wrap(err, "failed to check topic active status")
	}
	if !isActive {
		return false, nil
	}
	nonces, err := k.nonceKeeper.GetUnfulfilledWorkerNonces(ctx, topic.Id)
	if err != nil {
		return false, errorsmod.Wrap(err, "failed to get unfulfilled worker nonces")
	}
	if len(nonces.Nonces) == 0 {
		return false, nil
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	for _, nonce := range nonces.Nonces {
		if nonce == nil {
			continue
		}
		withinWindow, err := BlockWithinWorkerSubmissionWindowOfNonce(topic, *nonce, blockHeight)
		if err != nil {
			return false, errorsmod.Wrap(err, "failed to check worker submission window")
		}
		if withinWindow {
			return true, nil
		}
	}
	return false, nil
}

// labelWhitelistChanged compares two whitelists for semantic inequality in
// their canonicalized forms. The caller is expected to have already run
// CanonicalizeLabelList over the updatedTopic slice (via SetTopic) if it is
// going to be persisted; at the keeper layer equal-length-and-contents means
// "no change" because UpdateTopic receives a full replacement topic.
func labelWhitelistChanged(a, b []string) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}

// UpdateTopic applies allowed changes to a topic. Fields that are unsafe to
// mutate while a worker submission window is open for ANY currently
// unfulfilled nonce are guarded with isAnyUnfulfilledWorkerNonceWithinWindow
// and rejected with ErrWorkerNonceWindowNotAvailable. Those fields are:
//   - merit_sortition_alpha (reward-math continuity)
//   - max_labels_per_submission (per-submission label cap)
//   - label_whitelist (per-topic label allowlist)
//   - label_default_value (implicit missing label semantics)
//
// This is stricter than the v14 behavior (which only checked the newest
// unfulfilled nonce) to match multilabel registry semantics, where label-
// related topic parameters must be stable across the entire lifetime of every
// open WSW.
func (k *TopicKeeper) UpdateTopic(ctx context.Context, topic types.Topic, updatedTopic types.Topic) (types.Topic, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "error getting params")
	}

	// LabelCaseSensitive is immutable after topic creation: flipping it would
	// change the canonical form of every persisted label (whitelist entries and
	// epoch registry names). Reject any divergence here so
	// the msgserver does not have to know about this invariant on its own.
	if topic.LabelCaseSensitive != updatedTopic.LabelCaseSensitive {
		return types.Topic{}, errorsmod.Wrap(
			types.ErrInvalidTopicUpdate,
			"label_case_sensitive is immutable after topic creation",
		)
	}
	if err := types.ValidateMaxLabelsPerSubmission(updatedTopic.MaxLabelsPerSubmission); err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "updated topic max_labels_per_submission validation failed")
	}

	// Canonicalize the whitelist on the proposed update before any
	// comparison or validation so the WSW guard and persisted state both
	// agree on the canonical form.
	if err := types.ValidateTopicLabelWhitelistSize(updatedTopic.LabelWhitelist, params.MaxTopicLabelWhitelistSize); err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "updated topic label_whitelist size validation failed")
	}
	if len(updatedTopic.LabelWhitelist) > 0 {
		canonical, err := types.CanonicalizeLabelList(
			updatedTopic.LabelWhitelist,
			true,
			params.MaxCanonicalLabelByteLength,
			updatedTopic.LabelCaseSensitive,
		)
		if err != nil {
			return types.Topic{}, errorsmod.Wrap(err, "updated topic label_whitelist canonicalization failed")
		}
		updatedTopic.LabelWhitelist = canonical
	}

	if err := updatedTopic.Validate(params); err != nil {
		return types.Topic{}, errorsmod.Wrap(err, "updated topic validation failed")
	}

	meritChanged := !topic.MeritSortitionAlpha.Equal(updatedTopic.MeritSortitionAlpha)
	maxLabelsChanged := topic.MaxLabelsPerSubmission != updatedTopic.MaxLabelsPerSubmission
	whitelistChanged := labelWhitelistChanged(topic.LabelWhitelist, updatedTopic.LabelWhitelist)
	labelDefaultChanged := !topic.LabelDefaultValue.Equal(updatedTopic.LabelDefaultValue)

	if meritChanged || maxLabelsChanged || whitelistChanged || labelDefaultChanged {
		withinWindow, err := k.isAnyUnfulfilledWorkerNonceWithinWindow(ctx, topic)
		if err != nil {
			return types.Topic{}, err
		}
		if withinWindow {
			var changedFields []string
			if meritChanged {
				changedFields = append(changedFields, "merit_sortition_alpha")
			}
			if maxLabelsChanged {
				changedFields = append(changedFields, "max_labels_per_submission")
			}
			if whitelistChanged {
				changedFields = append(changedFields, "label_whitelist")
			}
			if labelDefaultChanged {
				changedFields = append(changedFields, "label_default_value")
			}
			return types.Topic{}, errorsmod.Wrapf(
				types.ErrWorkerNonceWindowNotAvailable,
				"cannot update %s while a worker submission window is open",
				strings.Join(changedFields, ", "),
			)
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
func (k *TopicKeeper) FilterOutlierResistantInferences(ctx context.Context, topic types.Topic, inferences types.Inferences) (types.Inferences, error) {
	if topic.OutputArity == types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI {
		return types.Inferences{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "outlier resistant inferences are not supported for multi-label topics")
	}

	lastMedian, err := k.GetLastMedianInferences(ctx, topic.Id)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting last median inferences")
	}
	if lastMedian.IsZero() {
		return inferences, nil
	}
	mad, err := k.GetMadInferences(ctx, topic.Id)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting mad inferences")
	}
	if mad.IsZero() {
		return inferences, nil
	}

	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting params")
	}
	outlierThresholdMultiplier := params.InferenceOutlierDetectionThreshold

	filteredInferences := types.Inferences{
		Inferences: []*types.Inference{},
	}

	thresholdMad, err := outlierThresholdMultiplier.Mul(mad)
	if err != nil {
		return types.Inferences{}, errorsmod.Wrap(err, "error getting threshold mad")
	}

	for _, inf := range inferences.Inferences {
		// Calculate absolute difference from median
		score, err := inferenceOutlierScore(inf.Values)
		if err != nil {
			return types.Inferences{}, errorsmod.Wrap(err, "error calculating inference outlier score")
		}

		diff, err := score.Sub(lastMedian)
		if err != nil {
			return types.Inferences{}, errorsmod.Wrap(err, "error getting difference from median")
		}
		absDiff, err := diff.Abs()
		if err != nil {
			return types.Inferences{}, errorsmod.Wrap(err, "error getting absolute difference")
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

// GetEpochLabelRegistry returns the registry for (topicId, nonce).
// If none exists yet, it returns an empty registry (not an error).
func (k *TopicKeeper) GetEpochLabelRegistry(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
) (types.EpochLabelRegistry, error) {
	registry, err := k.topicLabelRegistry.Get(ctx, collections.Join(topicId, nonce))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.EpochLabelRegistry{
				TopicId: topicId,
				EpochId: uint64(nonce), //nolint:gosec // nonce is a non-negative block height; cast is safe
				Labels:  nil,
			}, nil
		}
		return types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error getting topic label registry")
	}
	registry.TopicId = topicId
	registry.EpochId = uint64(nonce) //nolint:gosec // nonce is a non-negative block height; cast is safe
	return registry, nil
}

// RegisterEpochLabels registers canonical labels for one inference with a
// single registry read/write. It preserves first-seen 1-based ids and returns
// ids in the same order as labelNames.
func (k *TopicKeeper) RegisterEpochLabels(
	ctx context.Context,
	topicID types.TopicId,
	labelCaseSensitive bool,
	nonce types.BlockHeight,
	labelNames []string,
	maxLabelBytes uint64,
	maxRegistrySize uint64,
) ([]LabelId, types.EpochLabelRegistry, error) {
	if err := types.ValidateTopicId(topicID); err != nil {
		return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(err, "nonce block height validation failed")
	}
	if err := types.ValidateMaxEpochLabelRegistrySize(maxRegistrySize); err != nil {
		return nil, types.EpochLabelRegistry{}, err
	}
	registry, err := k.GetEpochLabelRegistry(ctx, topicID, nonce)
	if err != nil {
		return nil, types.EpochLabelRegistry{}, err
	}
	if uint64(len(registry.Labels)) > maxRegistrySize {
		return nil, types.EpochLabelRegistry{}, errorsmod.Wrapf(
			types.ErrEpochLabelRegistrySaturated,
			"topic %d nonce %d registry size %d exceeds max %d",
			topicID,
			nonce,
			len(registry.Labels),
			maxRegistrySize,
		)
	}
	idsByName := make(map[string]LabelId, len(registry.Labels)+len(labelNames))
	for _, lbl := range registry.Labels {
		if lbl == nil {
			return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(sdkerrors.ErrLogic, "registry label cannot be nil")
		}
		if lbl.Id == 0 {
			return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(sdkerrors.ErrLogic, "label id cannot be zero")
		}
		idsByName[lbl.Name] = lbl.Id
	}
	ids := make([]LabelId, len(labelNames))
	changed := false
	for i, labelName := range labelNames {
		if err := types.EnsureCanonicalLabelName(labelName, maxLabelBytes, labelCaseSensitive); err != nil {
			return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(err, "label name validation failed")
		}
		if id, ok := idsByName[labelName]; ok {
			ids[i] = id
			continue
		}
		if uint64(len(registry.Labels)) >= maxRegistrySize {
			return nil, types.EpochLabelRegistry{}, errorsmod.Wrapf(
				types.ErrEpochLabelRegistrySaturated,
				"topic %d nonce %d registry size %d reached max %d",
				topicID,
				nonce,
				len(registry.Labels),
				maxRegistrySize,
			)
		}
		//nolint:gosec // maxRegistrySize is validated and bounds the registry length.
		nextID := LabelId(len(registry.Labels) + 1)
		registry.Labels = append(registry.Labels, &types.TopicLabel{Id: nextID, Name: labelName})
		idsByName[labelName] = nextID
		ids[i] = nextID
		changed = true
	}
	if changed {
		if err := k.topicLabelRegistry.Set(ctx, collections.Join(topicID, nonce), registry); err != nil {
			return nil, types.EpochLabelRegistry{}, errorsmod.Wrap(err, "error setting topic label registry")
		}
	}
	return ids, registry, nil
}

// GetEpochLabelId returns the label id for labelName, if present.
func (k *TopicKeeper) GetEpochLabelId(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
	labelName string,
) (LabelId, bool, error) {
	labelName = strings.TrimSpace(labelName)
	if labelName == "" {
		return 0, false, nil
	}

	registry, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
	if err != nil {
		return 0, false, err
	}

	for _, lbl := range registry.Labels {
		if lbl != nil && lbl.Name == labelName {
			return lbl.Id, true, nil
		}
	}
	return 0, false, nil
}

// GetEpochLabelName returns the label name for id, if present.
func (k *TopicKeeper) GetEpochLabelName(
	ctx context.Context,
	topicId types.TopicId,
	nonce types.BlockHeight,
	id LabelId,
) (string, bool, error) {
	if id == 0 {
		return "", false, nil
	}

	registry, err := k.GetEpochLabelRegistry(ctx, topicId, nonce)
	if err != nil {
		return "", false, err
	}

	idx := int(id) - 1
	if idx < 0 || idx >= len(registry.Labels) {
		return "", false, nil
	}
	lbl := registry.Labels[idx]
	if lbl == nil || lbl.Id != id {
		return "", false, nil
	}
	return lbl.Name, true, nil
}

func (k *TopicKeeper) PruneTopicLabelRegistry(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.topicLabelRegistry.Clear(ctx, blockRange)
}
