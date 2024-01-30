package keeper

import (
	"context"
	"errors"

	cosmosMath "cosmossdk.io/math"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	state "github.com/upshot-tech/protocol-state-machine-module"
)

type Uint = cosmosMath.Uint
type Int = cosmosMath.Int
type Coin = sdk.Coin

type TOPIC_ID = uint64
type ACC_ADDRESS = string
type TARGET = sdk.AccAddress
type TARGET_STR = string
type DELEGATOR = sdk.AccAddress
type DELEGATOR_STR = string
type WORKERS = string
type REPUTERS = string
type BLOCK_NUMBER = int64
type UNIX_TIMESTAMP = uint64

var ErrIntegerUnderflowDelegator = errors.New("integer underflow subtracting from delegator stake")
var ErrIntegerUnderflowBonds = errors.New("integer underflow subtracting from bonds stake")
var ErrIntegerUnderflowTarget = errors.New("integer underflow subtracting from target stake")
var ErrIntegerUnderflowTopicStake = errors.New("integer underflow subtracting from topic stake")
var ErrIntegerUnderflowTotalStake = errors.New("integer underflow subtracting from total system stake")
var ErrIterationLengthDoesNotMatch = errors.New("iteration length does not match")

type Keeper struct {
	cdc          codec.BinaryCodec
	addressCodec address.Codec

	// State management
	schema     collections.Schema
	params     collections.Item[state.Params]
	authKeeper AccountKeeper
	bankKeeper BankKeeper

	// ############################################
	// #             TOPIC STATE:                 #
	// ############################################

	// the next topic id to be used, equal to the number of topics that have been created
	nextTopicId collections.Sequence
	// every topic that has been created indexed by their topicId starting from 1 (0 is reserved for the root network)
	topics collections.Map[TOPIC_ID, state.Topic]
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TOPIC_ID, sdk.AccAddress]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TOPIC_ID, sdk.AccAddress]]

	// ############################################
	// #                STAKING                   #
	// ############################################

	// total sum stake of all stakers on the network
	totalStake collections.Item[Uint]
	// for every topic, how much total stake does that topic have accumulated?
	topicStake collections.Map[TOPIC_ID, Uint]
	// For staking information, we have 3 views of the data we need to record:
	// 1. How many tokens did each delegator (e.g. alice) stake?
	//    How many tokens did alice put into the staking system?
	// 2. For each target (e.g. bob), then for each delegator (e.g. alice),
	//    how many tokens did that delegator stake upon that target? (e.g. alice -> bob)
	// 3. For each target, how much accumulated stake from everybody do they have?
	//    How many tokens have been staked upon bob total?
	// Explanation by Example:
	// Let there be Alice and Bob, existing as nodes in topic0
	//
	// At time t_0, Alice stakes 4 tokens upon herself
	// stakeOwnedByDelegator(Alice) -> +4
	// stakePlacement(Alice, Alice) -> +4
	// stakePlacedUponTarget(Alice) -> +4
	// TopicStake(topic0) -> +4
	// TotalStake across everybody: +4
	//
	// Later, at time t_1, Alice stakes 3 tokens upon Bob
	// stakeOwnedByDelegator(Alice) -> +3
	// stakePlacement(Alice, Bob) -> +3
	// stakePlacedUponTarget(Bob) -> +3
	// TopicStake(topic0) -> +3
	// TotalStake across everybody: +3
	//
	// At time t_2, Alice withdraws Alice's stake in herself and Bob
	// stakeOwnedByDelegator(Alice) -> -4 for Alice, -3 for Bob = -7 total
	// stakePlacement(Alice,Alice) -> -4
	// stakePlacement(Alice, Bob) -> -3
	// stakePlacedUponTarget(Alice) -> -4
	// stakePlacedUponTarget(Bob) -> -3
	// TopicStake(topic0) -> -7
	// TotalStake across everybody: -7
	//
	// TODO: enforcement and unit testing of various invariants from this data structure
	// Invariant 1: stakeOwnedByDelegator = sum of all stakes placed by that delegator
	// Invariant 2: targetStake = sum of all bonds placed upon that target
	// Invariant 3: sum of all targetStake = topicStake for that topic
	//
	// map of (delegator) -> total amount staked by delegator on everyone they've staked upon (including potentially themselves)
	stakeOwnedByDelegator collections.Map[DELEGATOR, Uint]
	// map of (target, delegator) -> amount staked by delegator upon target
	stakePlacement collections.Map[collections.Pair[TARGET, DELEGATOR], Uint]
	// map of (target) -> total amount staked upon target by themselves and all other delegators
	stakePlacedUponTarget collections.Map[TARGET, Uint]

	// ############################################
	// #            MISC GLOBAL STATE:            #
	// ############################################

	// map of Reputer, worker -> weight judged by reputer upon worker node
	weights collections.Map[collections.Triple[TOPIC_ID, sdk.AccAddress, sdk.AccAddress], Uint]

	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TOPIC_ID, sdk.AccAddress], state.Inference]

	// map of worker id to node data about that worker
	workers collections.Map[sdk.AccAddress, state.InferenceNode]

	// map of reputer id to node data about that reputer
	reputers collections.Map[sdk.AccAddress, state.InferenceNode]

	// the last block the token inflation rewards were updated: int64 same as BlockHeight()
	lastRewardsUpdate collections.Item[BLOCK_NUMBER]

	// map of topic -> latestInferenceTimestamp (M1)
	latestInferencesTimestamps collections.Map[TOPIC_ID, UNIX_TIMESTAMP]

	// map of (topic, timestamp, index) -> Inference
	allInferences collections.Map[collections.Pair[TOPIC_ID, UNIX_TIMESTAMP], state.Inferences]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	addressCodec address.Codec,
	storeService storetypes.KVStoreService,
	ak AccountKeeper,
	bk BankKeeper) Keeper {

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                        cdc,
		addressCodec:               addressCodec,
		params:                     collections.NewItem(sb, state.ParamsKey, "params", codec.CollValue[state.Params](cdc)),
		authKeeper:                 ak,
		bankKeeper:                 bk,
		totalStake:                 collections.NewItem(sb, state.TotalStakeKey, "total_stake", UintValue),
		topicStake:                 collections.NewMap(sb, state.TopicStakeKey, "topic_stake", collections.Uint64Key, UintValue),
		lastRewardsUpdate:          collections.NewItem(sb, state.LastRewardsUpdateKey, "last_rewards_update", collections.Int64Value),
		nextTopicId:                collections.NewSequence(sb, state.NextTopicIdKey, "next_topic_id"),
		topics:                     collections.NewMap(sb, state.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[state.Topic](cdc)),
		topicWorkers:               collections.NewKeySet(sb, state.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		topicReputers:              collections.NewKeySet(sb, state.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		stakeOwnedByDelegator:      collections.NewMap(sb, state.DelegatorStakeKey, "delegator_stake", sdk.AccAddressKey, UintValue),
		stakePlacement:             collections.NewMap(sb, state.BondsKey, "bonds", collections.PairKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey), UintValue),
		stakePlacedUponTarget:      collections.NewMap(sb, state.TargetStakeKey, "target_stake", sdk.AccAddressKey, UintValue),
		weights:                    collections.NewMap(sb, state.WeightsKey, "weights", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), UintValue),
		inferences:                 collections.NewMap(sb, state.InferencesKey, "inferences", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[state.Inference](cdc)),
		workers:                    collections.NewMap(sb, state.WorkerNodesKey, "worker_nodes", sdk.AccAddressKey, codec.CollValue[state.InferenceNode](cdc)),
		reputers:                   collections.NewMap(sb, state.ReputerNodesKey, "reputer_nodes", sdk.AccAddressKey, codec.CollValue[state.InferenceNode](cdc)),
		latestInferencesTimestamps: collections.NewMap(sb, state.LatestInferencesTsKey, "inferences_latest_ts", collections.Uint64Key, collections.Uint64Value),
		allInferences:              collections.NewMap(sb, state.AllInferencesKey, "inferences_all", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), codec.CollValue[state.Inferences](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.schema = schema

	return k
}

func (k *Keeper) GetLatestInferenceTimestamp(ctx context.Context, topicId TOPIC_ID) (uint64, error) {
	return k.latestInferencesTimestamps.Get(ctx, topicId)
}

func (k *Keeper) SetLatestInferenceTimestamp(ctx context.Context, topicId TOPIC_ID, timestamp uint64) error {
	return k.latestInferencesTimestamps.Set(ctx, topicId, timestamp)
}

func (k *Keeper) GetAllInferences(ctx context.Context, topicId TOPIC_ID, timestamp uint64) (*state.Inferences, error) {
	// pair := collections.Join(topicId, timestamp, index)
	key := collections.Join(topicId, timestamp)
	inferences, err := k.allInferences.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &inferences, nil
}

func appendToInferences(inferencesSet *state.Inferences, newInference *state.Inference) state.Inferences {
	if inferencesSet == nil || len(inferencesSet.Inferences) == 0 {
		// If Inferences is nil or empty, create a new one with the new Inference
		return state.Inferences{Inferences: []*state.Inference{newInference}}
	}
	// If Inferences is not empty, append the new Inference to the existing ones
	return state.Inferences{Inferences: append(inferencesSet.Inferences, newInference)}
}

func (k *Keeper) InsertInference(ctx context.Context, topicId TOPIC_ID, timestamp uint64, inference state.Inference) error {
	key := collections.Join(topicId, timestamp)
	inferences_set, err := k.allInferences.Get(ctx, key)
	if err != nil {
		inferences_set = state.Inferences{
			Inferences: []*state.Inference{},
		}
	}
	// inferences_new_set := append(inferences_set.Inferences, &inference)
	inferences_new_set := appendToInferences(&inferences_set, &inference)
	return k.allInferences.Set(ctx, key, inferences_new_set)
}

// Insert a complete set of inferences for a topic/timestamp. Overwrites previous ones.
func (k *Keeper) InsertInferences(ctx context.Context, topicId TOPIC_ID, timestamp uint64, inferences state.Inferences) error {
	key := collections.Join(topicId, timestamp)
	err := k.allInferences.Set(ctx, key, inferences)

	return err
}

// A function that accepts a topicId and returns list of Inferences or error
func (k *Keeper) GetLatestInferencesFromTopic(ctx context.Context, topicId TOPIC_ID) ([]*state.InferenceSetForScoring, error) {
	var inferences []*state.InferenceSetForScoring

	latest_timestamp, err := k.latestInferencesTimestamps.Get(ctx, topicId)
	if err != nil {
		return nil, err
	}

	rng := collections.
		NewPrefixedPairRange[TOPIC_ID, UNIX_TIMESTAMP](topicId).
		StartInclusive(latest_timestamp).
		Descending()

	iter, err := k.allInferences.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}
	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return nil, err
		}
		key := kv.Key
		value := kv.Value
		inferenceSet := &state.InferenceSetForScoring{
			TopicId:    key.K1(),
			Timestamp:  key.K2(),
			Inferences: &value,
		}
		inferences = append(inferences, inferenceSet)
	}
	return inferences, nil
}

// Gets the total sum of all stake in the network across all topics
func (k *Keeper) GetTotalStake(ctx context.Context) (Uint, error) {
	return k.totalStake.Get(ctx)
}

// Sets the total sum of all stake in the network across all topics
func (k *Keeper) SetTotalStake(ctx context.Context, totalStake Uint) error {
	return k.totalStake.Set(ctx, totalStake)
}

// Gets the stake in the network for a given topic
func (k *Keeper) GetTopicStake(ctx context.Context, topicId TOPIC_ID) (Uint, error) {
	return k.topicStake.Get(ctx, topicId)
}

func (k *Keeper) SetTopicStake(ctx context.Context, topicId TOPIC_ID, stake Uint) error {
	return k.topicStake.Set(ctx, topicId, stake)
}

// Gets a map of each topic in the network and how much stake it has
func (k *Keeper) GetAllTopicStake(ctx context.Context) (map[TOPIC_ID]Uint, error) {
	topicStake := make(map[TOPIC_ID]Uint)
	if err := k.topicStake.Walk(ctx, nil, func(topicId TOPIC_ID, stake Uint) (bool, error) {
		topicStake[topicId] = stake
		return false, nil
	}); err != nil {
		return nil, err
	}

	return topicStake, nil
}

// Returns the last block height at which rewards emissions were updated
func (k *Keeper) GetLastRewardsUpdate(ctx context.Context) (int64, error) {
	return k.lastRewardsUpdate.Get(ctx)
}

// Set the last block height at which rewards emissions were updated
func (k *Keeper) SetLastRewardsUpdate(ctx context.Context, blockHeight int64) error {
	return k.lastRewardsUpdate.Set(ctx, blockHeight)
}

// Get the amount of tokens that will be paid as rewards to network participants
// the next time the rewards calculation function runs
func (k *Keeper) GetAccumulatedEpochRewards(ctx context.Context) sdk.Coin {
	// the coin denom should match upshot-appchain config.go, i.e. upt
	moduleAccount := k.authKeeper.GetModuleAccount(ctx, state.ModuleName)
	moduleAccountAddress := moduleAccount.GetAddress()
	return k.bankKeeper.GetBalance(ctx, moduleAccountAddress, "upt")
}

// Returns the number of topics that are active in the network
func (k *Keeper) GetNumTopics(ctx context.Context) (TOPIC_ID, error) {
	return k.nextTopicId.Peek(ctx)
}

// Gets next topic id
func (k *Keeper) GetNextTopicId(ctx context.Context) (TOPIC_ID, error) {
	return k.nextTopicId.Next(ctx)
}

// Sets a topic config on a topicId
func (k *Keeper) SetTopic(ctx context.Context, topicId TOPIC_ID, topic state.Topic) error {
	return k.topics.Set(ctx, topicId, topic)
}

func (k *Keeper) GetWorker(ctx context.Context, worker sdk.AccAddress) (state.InferenceNode, error) {
	return k.workers.Get(ctx, worker)
}

func (k *Keeper) GetReputer(ctx context.Context, reputer sdk.AccAddress) (state.InferenceNode, error) {
	return k.reputers.Get(ctx, reputer)
}

// GetActiveTopics returns a slice of all active topics.
func (k *Keeper) GetActiveTopics(ctx context.Context) ([]state.Topic, error) {
	var activeTopics []state.Topic
	if err := k.topics.Walk(ctx, nil, func(topicId TOPIC_ID, topic state.Topic) (bool, error) {
		if topic.Active { // Check if the topic is marked as active
			activeTopics = append(activeTopics, topic)
		}
		return false, nil // Continue the iteration
	}); err != nil {
		return nil, err
	}
	return activeTopics, nil
}

// for a given topic, returns every reputer node registered to it
func (k *Keeper) GetTopicReputers(ctx context.Context, topicId TOPIC_ID) ([]sdk.AccAddress, error) {
	var reputers []sdk.AccAddress
	rng := collections.NewPrefixedPairRange[TOPIC_ID, sdk.AccAddress](topicId)
	iter, err := k.topicReputers.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}

	kvs, err := iter.Keys()
	if err != nil {
		return nil, err
	}
	for _, kv := range kvs {
		reputers = append(reputers, kv.K2())
	}
	return reputers, nil
}

// Add stake adds stake to the system for a given delegator and target
// it adds to existing holdings.
// it places the stake upon target, from delegator, in amount.
// it also updates the total stake for the subnet in question and the total global stake.
// see comments in keeper.go data structures for examples of how the data structure tracking works
func (k *Keeper) AddStake(ctx context.Context, topic TOPIC_ID, delegator string, target string, stake Uint) error {
	// update the stake array that tracks how much each delegator has invested in the system total
	delegatorAcc, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return err
	}
	delegatorStake, err := k.stakeOwnedByDelegator.Get(ctx, delegatorAcc)
	if err != nil {
		return err
	}
	delegatorStakeNew := delegatorStake.Add(stake)
	if err := k.stakeOwnedByDelegator.Set(ctx, delegatorAcc, delegatorStakeNew); err != nil {
		return err
	}

	// Update the bonds amount, which tracks each individual place
	// each delegator has placed their stake.
	// the sum of all bonds for a delegator should equal the delegatorStake
	// and the sum of all bonds on a target should equal the targetStake
	// set Bond(delegator -> target) = Bond(delegator -> target) + stake
	targetAcc, err := sdk.AccAddressFromBech32(target)
	if err != nil {
		return err
	}
	bondIndex := collections.Join(delegatorAcc, targetAcc)
	bond, err := k.stakePlacement.Get(ctx, bondIndex)
	if err != nil {
		return err
	}
	bondNew := bond.Add(stake)
	if err := k.stakePlacement.Set(ctx, bondIndex, bondNew); err != nil {
		return err
	}

	// set the targetStake for this target target
	// this is the sum total of all bonds placed upon this target
	// from all different people who have placed stake upon this target
	targetStake, err := k.stakePlacedUponTarget.Get(ctx, targetAcc)
	if err != nil {
		return err
	}
	targetStakeNew := targetStake.Add(stake)
	if err := k.stakePlacedUponTarget.Set(ctx, targetAcc, targetStakeNew); err != nil {
		return err
	}

	// Update the sum topic stake for this topic
	topicStake, err := k.topicStake.Get(ctx, topic)
	if err != nil {
		return err
	}
	topicStakeNew := topicStake.Add(stake)
	if err := k.topicStake.Set(ctx, topic, topicStakeNew); err != nil {
		return err
	}

	// Update the total stake across the entire system
	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		return err
	}
	totalStakeNew := totalStake.Add(stake)
	if err := k.totalStake.Set(ctx, totalStakeNew); err != nil {
		return err
	}

	return nil
}

// Remove stake from bond updates the various data structures associated
// with removing stake from the system for a given delegator and target
// it removes the stake upon target, from delegator, in amount.
// it also updates the total stake for the subnet in question and the total global stake.
// see comments in keeper.go data structures for examples of how the data structure tracking works
func (k *Keeper) RemoveStakeFromBond(
	ctx context.Context,
	topic TOPIC_ID,
	delegator sdk.AccAddress,
	target sdk.AccAddress,
	stake Uint) error {

	// 1. 2. and 3. make checks and update state for
	// delegatorStake, bonds, and targetStake
	err := k.RemoveStakeFromBondMissingTotalOrTopicStake(ctx, topic, delegator, target, stake)
	if err != nil {
		return err
	}

	// 4. Check: topicStake(topic) >= stake
	topicStake, err := k.topicStake.Get(ctx, topic)
	if err != nil {
		return err
	}
	if stake.GT(topicStake) {
		return ErrIntegerUnderflowTopicStake
	}

	// 5. Check: totalStake >= stake
	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		return err
	}
	if stake.GT(totalStake) {
		return ErrIntegerUnderflowTotalStake
	}

	// Perform State Updates
	// TODO: make this function prevent partial state updates / do rollbacks if any of the set statements fail
	// not necessary as long as callers are responsible, but it would be nice to have

	// topicStake(topic) = topicStake(topic) - stake
	err = k.topicStake.Set(ctx, topic, topicStake.Sub(stake))
	if err != nil {
		return err
	}

	// totalStake = totalStake - stake
	err = k.totalStake.Set(ctx, totalStake.Sub(stake))
	if err != nil {
		return err
	}

	return nil
}

// Remove stake from bond updates the various data structures associated
// with removing stake from the system for a given delegator and target
// it removes the stake upon target, from delegator, in amount.
// it *DOES NOT* update the total stake for the subnet in question and the total global stake.
// this is used by RemoveAllStake to avoid double counting the topic/total stake removal
func (k *Keeper) RemoveStakeFromBondMissingTotalOrTopicStake(
	ctx context.Context,
	topic TOPIC_ID,
	delegator sdk.AccAddress,
	target sdk.AccAddress,
	stake Uint) error {
	// 1. Check: delegatorStake(delegator) >= stake
	delegatorStake, err := k.stakeOwnedByDelegator.Get(ctx, delegator)
	if err != nil {
		return err
	}
	if stake.GT(delegatorStake) {
		return ErrIntegerUnderflowDelegator
	}

	// 2. Check: bonds(target, delegator) >= stake
	bond, err := k.stakePlacement.Get(ctx, collections.Join(delegator, target))
	if err != nil {
		return err
	}
	if stake.GT(bond) {
		return ErrIntegerUnderflowBonds
	}

	// 3. Check: targetStake(target) >= stake
	targetStake, err := k.stakePlacedUponTarget.Get(ctx, target)
	if err != nil {
		return err
	}
	if stake.GT(targetStake) {
		return ErrIntegerUnderflowTarget
	}

	// Perform State Updates
	// TODO: make this function prevent partial state updates / do rollbacks if any of the set statements fail
	// not necessary as long as callers are responsible, but it would be nice to have

	// delegatorStake(delegator) = delegatorStake(delegator) - stake
	err = k.stakeOwnedByDelegator.Set(ctx, delegator, delegatorStake.Sub(stake))
	if err != nil {
		return err
	}
	// bonds(target, delegator) = bonds(target, delegator) - stake
	err = k.stakePlacement.Set(ctx, collections.Join(delegator, target), bond.Sub(stake))
	if err != nil {
		return err
	}

	// targetStake(target) = targetStake(target) - stake
	err = k.stakePlacedUponTarget.Set(ctx, target, targetStake.Sub(stake))
	if err != nil {
		return err
	}

	return nil
}

// for a given address, find out how much stake they've put into the system
func (k *Keeper) GetDelegatorStake(ctx context.Context, delegator sdk.AccAddress) (Uint, error) {
	return k.stakeOwnedByDelegator.Get(ctx, delegator)
}

// For a given delegator and target, find out how much stake the delegator has placed upon the target
func (k *Keeper) GetBond(ctx context.Context, delegator sdk.AccAddress, target sdk.AccAddress) (Uint, error) {
	return k.stakePlacement.Get(ctx, collections.Join(delegator, target))
}

// For a given delegator, return a map of every target they've placed stake upon, and how much stake they've placed upon them
// O(n) over the number of targets registered.
// since maps of byte array types aren't supported in golang, we instead return two equal length arrays
// where the first array is the targets, and the second array is the amount of stake placed upon them
// indexes in the two arrays correspond to each other
// invariant that len(targets) == len(stakes)
func (k *Keeper) GetAllBondsForDelegator(ctx context.Context, delegator sdk.AccAddress) ([]sdk.AccAddress, []Uint, error) {
	targets := make([]sdk.AccAddress, 0)
	amounts := make([]Uint, 0)
	iter, err := k.stakePlacement.Iterate(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	// iterate over all keys in stakePlacements
	kvs, err := iter.Keys()
	if err != nil {
		return nil, nil, err
	}
	for _, kv := range kvs {
		d := kv.K2()
		// if the delegator key matches the delegator we're looking for
		if d.Equals(delegator) {
			target := kv.K1()
			amount, err := k.stakePlacement.Get(ctx, kv)
			if err != nil {
				return nil, nil, err
			}
			targets = append(targets, target)
			amounts = append(amounts, amount)
		}
	}
	if len(targets) != len(amounts) {
		return nil, nil, ErrIterationLengthDoesNotMatch
	}

	return targets, amounts, nil
}

// For a given target, find out how much stake has been placed upon them
func (k *Keeper) GetTargetStake(ctx context.Context, target sdk.AccAddress) (Uint, error) {
	return k.stakePlacedUponTarget.Get(ctx, target)
}

// For a given topic return the matrix (double map) of all (reputers, workers) -> weight of reputer upon worker
func (k *Keeper) GetWeightsFromTopic(ctx context.Context, topicId TOPIC_ID) (map[REPUTERS]map[WORKERS]*Uint, error) {
	weights := make(map[ACC_ADDRESS]map[ACC_ADDRESS]*Uint)
	rng := collections.NewPrefixedTripleRange[TOPIC_ID, sdk.AccAddress, sdk.AccAddress](topicId)
	iter, err := k.weights.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}

	kvs, err := iter.Keys()
	if err != nil {
		return nil, err
	}
	for _, kv := range kvs {
		reputer := kv.K2()
		worker := kv.K3()
		reputerWeights := weights[reputer.String()]
		if reputerWeights == nil {
			reputerWeights = make(map[ACC_ADDRESS]*Uint)
			weights[reputer.String()] = reputerWeights
		}
		weight, err := k.weights.Get(ctx, kv)
		if err != nil {
			return nil, err
		}
		weights[reputer.String()][worker.String()] = &weight
	}
	return weights, nil
}

// UpdateTopicInferenceLastRan updates the InferenceLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicInferenceLastRan(ctx context.Context, topicId TOPIC_ID, lastRanTime uint64) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.InferenceLastRan = lastRanTime
	return k.topics.Set(ctx, topicId, topic)
}

// UpdateTopicWeightLastRan updates the WeightLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicWeightLastRan(ctx context.Context, topicId TOPIC_ID, lastRanTime uint64) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.WeightLastRan = lastRanTime
	return k.topics.Set(ctx, topicId, topic)
}

// check a reputer node is registered
func (k *Keeper) IsReputerRegistered(ctx context.Context, reputer sdk.AccAddress) (bool, error) {
	return k.reputers.Has(ctx, reputer)
}

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *Keeper) InsertReputer(ctx context.Context, topicId uint64, reputer sdk.AccAddress, reputerInfo state.InferenceNode) error {
	topickey := collections.Join[uint64, sdk.AccAddress](topicId, reputer)
	err := k.topicReputers.Set(ctx, topickey)
	if err != nil {
		return err
	}
	err = k.reputers.Set(ctx, reputer, reputerInfo)
	if err != nil {
		return err
	}
	return nil
}

// check a worker node is registered
func (k *Keeper) IsWorkerRegistered(ctx context.Context, worker sdk.AccAddress) (bool, error) {
	return k.workers.Has(ctx, worker)
}

// Adds a new worker to the worker tracking data structures, workers and topicWorkers
func (k *Keeper) InsertWorker(ctx context.Context, topicId uint64, worker sdk.AccAddress, workerInfo state.InferenceNode) error {
	topickey := collections.Join[uint64, sdk.AccAddress](topicId, worker)
	err := k.topicWorkers.Set(ctx, topickey)
	if err != nil {
		return err
	}
	err = k.workers.Set(ctx, worker, workerInfo)
	if err != nil {
		return err
	}
	return nil
}
