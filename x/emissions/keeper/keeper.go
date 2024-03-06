package keeper

import (
	"context"
	"errors"
	"math/big"
	"strings"

	cosmosMath "cosmossdk.io/math"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Uint = cosmosMath.Uint
type Int = cosmosMath.Int

type TOPIC_ID = uint64
type LIB_P2P_KEY = string
type ACC_ADDRESS = string
type TARGET = sdk.AccAddress
type TARGET_STR = string
type DELEGATOR = sdk.AccAddress
type DELEGATOR_STR = string
type WORKERS = string
type REPUTERS = string
type BLOCK_NUMBER = int64
type UNIX_TIMESTAMP = uint64
type REQUEST_ID = string

type Keeper struct {
	cdc              codec.BinaryCodec
	addressCodec     address.Codec
	feeCollectorName string

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
	// every topics that has been churned and ready to get inferences in the block
	churnReadyTopics collections.Item[state.TopicList]
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TOPIC_ID, sdk.AccAddress]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TOPIC_ID, sdk.AccAddress]]
	// for an address, what are all the topics that it's registered for?
	addressTopics collections.Map[sdk.AccAddress, []uint64]

	// ############################################
	// #                STAKING                   #
	// ############################################

	// total sum stake of all topics
	allTopicStakeSum collections.Item[Uint]
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
	// map of (delegator, target) -> amount staked by delegator upon target
	stakePlacement collections.Map[collections.Pair[DELEGATOR, TARGET], Uint]
	// map of (target) -> total amount staked upon target by themselves and all other delegators
	stakePlacedUponTarget collections.Map[TARGET, Uint]
	// map of (delegator) -> removal information for that delegator
	stakeRemovalQueue collections.Map[DELEGATOR, state.StakeRemoval]

	// ############################################
	// #        INFERENCE REQUEST MEMPOOL         #
	// ############################################
	// map of (topic, request_id) -> full InferenceRequest information for that request
	mempool collections.Map[collections.Pair[TOPIC_ID, REQUEST_ID], state.InferenceRequest]
	// amount of money available for an inference request id that has been placed in the mempool but has not yet been fully satisfied
	requestUnmetDemand collections.Map[REQUEST_ID, Uint]
	// total amount of demand for a topic that has been placed in the mempool as a request for inference but has not yet been satisfied
	topicUnmetDemand collections.Map[TOPIC_ID, Uint]

	// ############################################
	// #            MISC GLOBAL STATE:            #
	// ############################################

	// map of Reputer, worker -> weight judged by reputer upon worker node
	weights collections.Map[collections.Triple[TOPIC_ID, sdk.AccAddress, sdk.AccAddress], Uint]

	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TOPIC_ID, sdk.AccAddress], state.Inference]

	// map of (topic, worker) -> num_inferences_in_reward_epoch
	numInferencesInRewardEpoch collections.Map[collections.Pair[TOPIC_ID, sdk.AccAddress], Uint]

	// map of worker id to node data about that worker
	workers collections.Map[LIB_P2P_KEY, state.OffchainNode]

	// map of reputer id to node data about that reputer
	reputers collections.Map[LIB_P2P_KEY, state.OffchainNode]

	// the last block the token inflation rewards were updated: int64 same as BlockHeight()
	lastRewardsUpdate collections.Item[BLOCK_NUMBER]

	// map of (topic, timestamp, index) -> Inference
	allInferences collections.Map[collections.Pair[TOPIC_ID, UNIX_TIMESTAMP], state.Inferences]

	accumulatedMetDemand collections.Map[TOPIC_ID, Uint]

	whitelistAdmins collections.KeySet[sdk.AccAddress]

	topicCreationWhitelist collections.KeySet[sdk.AccAddress]

	weightSettingWhitelist collections.KeySet[sdk.AccAddress]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	addressCodec address.Codec,
	storeService storetypes.KVStoreService,
	ak AccountKeeper,
	bk BankKeeper,
	feeCollectorName string) Keeper {

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                        cdc,
		addressCodec:               addressCodec,
		feeCollectorName:           feeCollectorName,
		params:                     collections.NewItem(sb, state.ParamsKey, "params", codec.CollValue[state.Params](cdc)),
		authKeeper:                 ak,
		bankKeeper:                 bk,
		totalStake:                 collections.NewItem(sb, state.TotalStakeKey, "total_stake", UintValue),
		topicStake:                 collections.NewMap(sb, state.TopicStakeKey, "topic_stake", collections.Uint64Key, UintValue),
		lastRewardsUpdate:          collections.NewItem(sb, state.LastRewardsUpdateKey, "last_rewards_update", collections.Int64Value),
		nextTopicId:                collections.NewSequence(sb, state.NextTopicIdKey, "next_topic_id"),
		topics:                     collections.NewMap(sb, state.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[state.Topic](cdc)),
		churnReadyTopics:           collections.NewItem(sb, state.ChurnReadyTopicsKey, "churn_ready_topics", codec.CollValue[state.TopicList](cdc)),
		topicWorkers:               collections.NewKeySet(sb, state.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		addressTopics:              collections.NewMap(sb, state.AddressTopicsKey, "address_topics", sdk.AccAddressKey, TopicIdListValue),
		topicReputers:              collections.NewKeySet(sb, state.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		allTopicStakeSum:           collections.NewItem(sb, state.AllTopicStakeSumKey, "all_topic_stake_sum", UintValue),
		stakeOwnedByDelegator:      collections.NewMap(sb, state.DelegatorStakeKey, "delegator_stake", sdk.AccAddressKey, UintValue),
		stakePlacement:             collections.NewMap(sb, state.BondsKey, "bonds", collections.PairKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey), UintValue),
		stakePlacedUponTarget:      collections.NewMap(sb, state.TargetStakeKey, "target_stake", sdk.AccAddressKey, UintValue),
		stakeRemovalQueue:          collections.NewMap(sb, state.StakeRemovalQueueKey, "stake_removal_queue", sdk.AccAddressKey, codec.CollValue[state.StakeRemoval](cdc)),
		mempool:                    collections.NewMap(sb, state.MempoolKey, "mempool", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[state.InferenceRequest](cdc)),
		requestUnmetDemand:         collections.NewMap(sb, state.RequestUnmetDemandKey, "request_unmet_demand", collections.StringKey, UintValue),
		topicUnmetDemand:           collections.NewMap(sb, state.TopicUnmetDemandKey, "topic_unmet_demand", collections.Uint64Key, UintValue),
		weights:                    collections.NewMap(sb, state.WeightsKey, "weights", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), UintValue),
		inferences:                 collections.NewMap(sb, state.InferencesKey, "inferences", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[state.Inference](cdc)),
		workers:                    collections.NewMap(sb, state.WorkerNodesKey, "worker_nodes", collections.StringKey, codec.CollValue[state.OffchainNode](cdc)),
		reputers:                   collections.NewMap(sb, state.ReputerNodesKey, "reputer_nodes", collections.StringKey, codec.CollValue[state.OffchainNode](cdc)),
		allInferences:              collections.NewMap(sb, state.AllInferencesKey, "inferences_all", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), codec.CollValue[state.Inferences](cdc)),
		accumulatedMetDemand:       collections.NewMap(sb, state.AccumulatedMetDemandKey, "accumulated_met_demand", collections.Uint64Key, UintValue),
		numInferencesInRewardEpoch: collections.NewMap(sb, state.NumInferencesInRewardEpochKey, "num_inferences_in_reward_epoch", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		whitelistAdmins:            collections.NewKeySet(sb, state.WhitelistAdminsKey, "whitelist_admins", sdk.AccAddressKey),
		topicCreationWhitelist:     collections.NewKeySet(sb, state.TopicCreationWhitelistKey, "topic_creation_whitelist", sdk.AccAddressKey),
		weightSettingWhitelist:     collections.NewKeySet(sb, state.WeightSettingWhitelistKey, "weight_setting_whitelist", sdk.AccAddressKey),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.schema = schema

	return k
}

func (k *Keeper) SetParams(ctx context.Context, params state.Params) error {
	return k.params.Set(ctx, params)
}

func (k *Keeper) GetParams(ctx context.Context) (state.Params, error) {
	ret, err := k.params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return state.DefaultParams(), nil
		}
		return state.Params{}, err
	}
	return ret, nil
}

func (k *Keeper) FeeCollectorName() string {
	return k.feeCollectorName
}

func (k *Keeper) GetTopicWeightLastRan(ctx context.Context, topicId TOPIC_ID) (uint64, error) {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return 0, err
	}
	ret := topic.WeightLastRan
	return ret, nil
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

func (k *Keeper) GetStakePlacedUponTarget(ctx context.Context, target sdk.AccAddress) (Uint, error) {
	ret, err := k.stakePlacedUponTarget.Get(ctx, target)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return ret, nil
}

func (k *Keeper) SetStakePlacedUponTarget(ctx context.Context, target sdk.AccAddress, stake Uint) error {
	if stake.IsZero() {
		return k.stakePlacedUponTarget.Remove(ctx, target)
	}
	return k.stakePlacedUponTarget.Set(ctx, target, stake)
}

// Returns the last block height at which rewards emissions were updated
func (k *Keeper) GetLastRewardsUpdate(ctx context.Context) (int64, error) {
	lastRewardsUpdate, err := k.lastRewardsUpdate.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil
		} else {
			return 0, err
		}
	}
	return lastRewardsUpdate, nil
}

// Set the last block height at which rewards emissions were updated
func (k *Keeper) SetLastRewardsUpdate(ctx context.Context, blockHeight int64) error {
	if blockHeight < 0 {
		return state.ErrBlockHeightNegative
	}
	previousBlockHeight, err := k.lastRewardsUpdate.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			previousBlockHeight = 0
		} else {
			return err
		}
	}
	if blockHeight < previousBlockHeight {
		return state.ErrBlockHeightLessThanPrevious
	}
	return k.lastRewardsUpdate.Set(ctx, blockHeight)
}

// return epoch length
func (k *Keeper) GetParamsEpochLength(ctx context.Context) (int64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.EpochLength, nil
}

// return how many new coins should be minted for the next emission
func (k *Keeper) CalculateAccumulatedEmissions(ctx context.Context) (cosmosMath.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockNumber := sdkCtx.BlockHeight()
	lastRewardsUpdate, err := k.GetLastRewardsUpdate(sdkCtx)
	if err != nil {
		return cosmosMath.Int{}, err
	}
	blocksSinceLastUpdate := blockNumber - lastRewardsUpdate
	// number of epochs that have passed (if more than 1)
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Int{}, err
	}
	epochsPassed := cosmosMath.NewInt(blocksSinceLastUpdate / params.EpochLength)
	// get emission amount
	return epochsPassed.Mul(params.EmissionsPerEpoch), nil
}

// mint new rewards coins to this module account
func (k *Keeper) MintRewardsCoins(ctx context.Context, amount cosmosMath.Int) error {
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))
	return k.bankKeeper.MintCoins(ctx, state.AlloraStakingModuleName, coins)
}

// for a given topic, returns every reputer node registered to it and their normalized stake
func (k *Keeper) GetReputerNormalizedStake(
	ctx sdk.Context,
	topicId TOPIC_ID,
	topicStake *big.Float) (reputerNormalizedStakeMap map[ACC_ADDRESS]*big.Float, retErr error) {
	reputerNormalizedStakeMap = make(map[ACC_ADDRESS]*big.Float)
	rng := collections.NewPrefixedPairRange[TOPIC_ID, sdk.AccAddress](topicId)
	retErr = nil
	retErr = k.topicReputers.Walk(ctx, rng, func(key collections.Pair[TOPIC_ID, sdk.AccAddress]) (stop bool, err error) {
		reputer := key.K2()
		// Get Stake in each reputer
		reputerTargetStake, err := k.stakePlacedUponTarget.Get(ctx, reputer)
		if err != nil {
			return true, err
		}
		reputerTotalStake := big.NewFloat(0).SetInt(reputerTargetStake.BigInt())

		// How much stake does each reputer have as a percentage of the total stake in the topic?
		reputerNormalizedStake := big.NewFloat(0).Quo(reputerTotalStake, topicStake)
		reputerNormalizedStakeMap[reputer.String()] = reputerNormalizedStake
		return false, nil
	})
	return reputerNormalizedStakeMap, retErr
}

// Gets the total sum of all stake in the network across all topics
func (k *Keeper) GetTotalStake(ctx context.Context) (Uint, error) {
	ret, err := k.totalStake.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return ret, nil
}

// Sets the total sum of all stake in the network across all topics
func (k *Keeper) SetTotalStake(ctx context.Context, totalStake Uint) error {
	// total stake does not have a zero guard because totalStake is allowed to be zero
	// it is initialized to zero at genesis anyways.
	return k.totalStake.Set(ctx, totalStake)
}

// A function that accepts a topicId and returns list of Inferences or error
func (k *Keeper) GetLatestInferencesFromTopic(ctx context.Context, topicId TOPIC_ID) ([]*state.InferenceSetForScoring, error) {
	var inferences []*state.InferenceSetForScoring
	var latestTimestamp, err = k.GetTopicWeightLastRan(ctx, topicId)
	if err != nil {
		latestTimestamp = 0
	}
	rng := collections.
		NewPrefixedPairRange[TOPIC_ID, UNIX_TIMESTAMP](topicId).
		StartInclusive(latestTimestamp).
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

// Gets the stake in the network for a given topic
func (k *Keeper) GetTopicStake(ctx context.Context, topicId TOPIC_ID) (Uint, error) {
	ret, err := k.topicStake.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return ret, nil
}

// Sets the stake in the network for a given topic
func (k *Keeper) SetTopicStake(ctx context.Context, topicId TOPIC_ID, stake Uint) error {
	if stake.IsZero() {
		return k.topicStake.Remove(ctx, topicId)
	}
	return k.topicStake.Set(ctx, topicId, stake)
}

// GetTopicsByCreator returns a slice of all topics created by a given creator.
func (k *Keeper) GetTopicsByCreator(ctx context.Context, creator string) ([]*state.Topic, error) {
	var topicsByCreator []*state.Topic

	err := k.topics.Walk(ctx, nil, func(id TOPIC_ID, topic state.Topic) (bool, error) {
		if topic.Creator == creator {
			topicsByCreator = append(topicsByCreator, &topic)
		}
		return false, nil // Continue iterating
	})

	if err != nil {
		return nil, err
	}

	return topicsByCreator, nil
}

// AddAddressTopics adds new topics to the address's list of topics, avoiding duplicates.
func (k *Keeper) AddAddressTopics(ctx context.Context, address sdk.AccAddress, newTopics []uint64) error {
	// Get the current list of topics for the address
	currentTopics, err := k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return err
	}

	topicSet := make(map[uint64]bool)
	for _, topic := range currentTopics {
		topicSet[topic] = true
	}

	for _, newTopic := range newTopics {
		if _, exists := topicSet[newTopic]; !exists {
			currentTopics = append(currentTopics, newTopic)
		}
	}

	// Set the updated list of topics for the address
	return k.addressTopics.Set(ctx, address, currentTopics)
}

// RemoveAddressTopic removes a specified topic from the address's list of topics.
func (k *Keeper) RemoveAddressTopic(ctx context.Context, address sdk.AccAddress, topicToRemove uint64) error {
	// Get the current list of topics for the address
	currentTopics, err := k.GetRegisteredTopicIdsByAddress(ctx, address)
	if err != nil {
		return err
	}

	// Find and remove the specified topic
	filteredTopics := make([]uint64, 0)
	for _, topic := range currentTopics {
		if topic != topicToRemove {
			filteredTopics = append(filteredTopics, topic)
		}
	}

	// Set the updated list of topics for the address
	return k.addressTopics.Set(ctx, address, filteredTopics)
}

// GetRegisteredTopicsByAddress returns a slice of all topics ids registered by a given address.
func (k *Keeper) GetRegisteredTopicIdsByAddress(ctx context.Context, address sdk.AccAddress) ([]uint64, error) {
	topics, err := k.addressTopics.Get(ctx, address)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// Return an empty slice if the address is not found, or handle it differently if needed.
			return []uint64{}, nil
		}
		return nil, err
	}
	return topics, nil
}

// GetRegisteredTopicIdsByWorkerAddress returns a slice of all topics ids registered by a given worker address.
func (k *Keeper) GetRegisteredTopicIdsByWorkerAddress(ctx context.Context, address sdk.AccAddress) ([]uint64, error) {
	var topicsByAddress []uint64

	err := k.topicWorkers.Walk(ctx, nil, func(pair collections.Pair[TOPIC_ID, sdk.AccAddress]) (bool, error) {
		if pair.K2().String() == address.String() {
			topicsByAddress = append(topicsByAddress, pair.K1())
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return topicsByAddress, nil
}

// GetRegisteredTopicIdByReputerAddress returns a slice of all topics ids registered by a given reputer address.
func (k *Keeper) GetRegisteredTopicIdByReputerAddress(ctx context.Context, address sdk.AccAddress) ([]uint64, error) {
	var topicsByAddress []uint64

	err := k.topicReputers.Walk(ctx, nil, func(pair collections.Pair[TOPIC_ID, sdk.AccAddress]) (bool, error) {
		if pair.K2().String() == address.String() {
			topicsByAddress = append(topicsByAddress, pair.K1())
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return topicsByAddress, nil
}

func (k *Keeper) IterateAllTopicStake(ctx context.Context) (collections.Iterator[uint64, cosmosMath.Uint], error) {
	rng := collections.Range[uint64]{}
	rng.StartInclusive(0)
	end, err := k.nextTopicId.Peek(ctx)
	if err != nil {
		return collections.Iterator[uint64, cosmosMath.Uint]{}, err
	}
	rng.EndExclusive(end)
	return k.topicStake.Iterate(ctx, &rng)
}

// Runs an arbitrary function for every topic in the network
func (k *Keeper) WalkAllTopicStake(ctx context.Context, walkFunc func(topicId TOPIC_ID, stake Uint) (stop bool, err error)) error {
	rng := collections.Range[uint64]{}
	rng.StartInclusive(0)
	end, err := k.nextTopicId.Peek(ctx)
	if err != nil {
		return err
	}
	rng.EndExclusive(end)
	err = k.topicStake.Walk(ctx, &rng, walkFunc)
	return err
}

// GetStakesForAccount returns the list of stakes for a given account address.
func (k *Keeper) GetStakesForAccount(ctx context.Context, delegator sdk.AccAddress) ([]*state.StakeInfo, error) {
	targets, amounts, err := k.GetAllBondsForDelegator(ctx, delegator)
	if err != nil {
		return nil, err
	}

	stakeInfos := make([]*state.StakeInfo, len(targets))
	for i, target := range targets {
		stakeInfos[i] = &state.StakeInfo{
			Address: target.String(),
			Amount:  amounts[i].String(),
		}
	}

	return stakeInfos, nil
}

// Gets next topic id
func (k *Keeper) IncrementTopicId(ctx context.Context) (TOPIC_ID, error) {
	return k.nextTopicId.Next(ctx)
}

// Gets topic by topicId
func (k *Keeper) GetTopic(ctx context.Context, topicId TOPIC_ID) (state.Topic, error) {
	return k.topics.Get(ctx, topicId)
}

// Sets a topic config on a topicId
func (k *Keeper) SetTopic(ctx context.Context, topicId TOPIC_ID, topic state.Topic) error {
	return k.topics.Set(ctx, topicId, topic)
}

// Gets every topic
func (k *Keeper) GetAllTopics(ctx context.Context) ([]*state.Topic, error) {
	var allTopics []*state.Topic
	err := k.topics.Walk(ctx, nil, func(topicId TOPIC_ID, topic state.Topic) (bool, error) {
		allTopics = append(allTopics, &topic)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return allTopics, nil
}

// Checks if a topic exists
func (k *Keeper) TopicExists(ctx context.Context, topicId TOPIC_ID) (bool, error) {
	return k.topics.Has(ctx, topicId)
}

// Returns the number of topics that are active in the network
func (k *Keeper) GetNumTopics(ctx context.Context) (TOPIC_ID, error) {
	return k.nextTopicId.Peek(ctx)
}

// GetActiveTopics returns a slice of all active topics.
func (k *Keeper) GetActiveTopics(ctx context.Context) ([]*state.Topic, error) {
	var activeTopics []*state.Topic
	if err := k.topics.Walk(ctx, nil, func(topicId TOPIC_ID, topic state.Topic) (bool, error) {
		if topic.Active { // Check if the topic is marked as active
			activeTopics = append(activeTopics, &topic)
		}
		return false, nil // Continue the iteration
	}); err != nil {
		return nil, err
	}
	return activeTopics, nil
}

// Add stake adds stake to the system for a given delegator and target
// it adds to existing holdings.
// it places the stake upon target, from delegator, in amount.
// it also updates the total stake for the subnet in question and the total global stake.
// see comments in keeper.go data structures for examples of how the data structure tracking works
func (k *Keeper) AddStake(ctx context.Context, TopicIds []TOPIC_ID, delegator string, target string, stake Uint) error {

	// if stake is zero this function is a no-op
	if stake.IsZero() {
		return state.ErrDoNotSetMapValueToZero
	}

	// update the stake array that tracks how much each delegator has invested in the system total
	delegatorAcc, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return err
	}
	delegatorStake, err := k.stakeOwnedByDelegator.Get(ctx, delegatorAcc)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			delegatorStake = cosmosMath.NewUint(0)
		} else {
			return err
		}
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
		if errors.Is(err, collections.ErrNotFound) {
			bond = cosmosMath.NewUint(0)
		} else {
			return err
		}
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
		if errors.Is(err, collections.ErrNotFound) {
			targetStake = cosmosMath.NewUint(0)
		} else {
			return err
		}
	}
	targetStakeNew := targetStake.Add(stake)
	if err := k.stakePlacedUponTarget.Set(ctx, targetAcc, targetStakeNew); err != nil {
		return err
	}

	// Update the sum topic stake for all topics
	for _, topicId := range TopicIds {
		topicStake, err := k.topicStake.Get(ctx, topicId)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				topicStake = cosmosMath.NewUint(0)
			} else {
				return err
			}
		}
		topicStakeNew := topicStake.Add(stake)
		if err := k.topicStake.Set(ctx, topicId, topicStakeNew); err != nil {
			return err
		}

		// Update the total stake across all topics
		allTopicStakeSum, err := k.allTopicStakeSum.Get(ctx)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				allTopicStakeSum = cosmosMath.NewUint(0)
			} else {
				return err
			}
		}
		allTopicStakeSumNew := allTopicStakeSum.Add(stake)
		if err := k.allTopicStakeSum.Set(ctx, allTopicStakeSumNew); err != nil {
			return err
		}
	}

	// Update the total stake across the entire system
	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			totalStake = cosmosMath.NewUint(0)
		} else {
			return err
		}
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
// it also updates the total stake for the topic in question and the total global stake.
// see comments in keeper.go data structures for examples of how the data structure tracking works
func (k *Keeper) RemoveStakeFromBond(
	ctx context.Context,
	TopicIds []TOPIC_ID,
	delegator sdk.AccAddress,
	target sdk.AccAddress,
	stake Uint) error {

	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	// 1. 2. and 3. make checks and update state for
	// delegatorStake, bonds, and targetStake
	err := k.RemoveStakeFromBondMissingTotalOrTopicStake(ctx, delegator, target, stake)
	if err != nil {
		return err
	}

	// Perform State Updates
	// TODO: make this function prevent partial state updates / do rollbacks if any of the set statements fail
	// not necessary as long as callers are responsible, but it would be nice to have

	// topicStake(topic) = topicStake(topic) - stake
	for _, topic := range TopicIds {
		topicStake, err := k.topicStake.Get(ctx, topic)
		if err != nil {
			return err
		}
		if stake.GT(topicStake) {
			return state.ErrIntegerUnderflowTopicStake
		}

		topicStakeNew := topicStake.Sub(stake)
		if topicStakeNew.IsZero() {
			err = k.topicStake.Remove(ctx, topic)
		} else {
			err = k.topicStake.Set(ctx, topic, topicStakeNew)
		}
		if err != nil {
			return err
		}
	}

	// totalStake = totalStake - stake
	// 4. Check: totalStake >= stake
	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		return err
	}
	if stake.GT(totalStake) {
		return state.ErrIntegerUnderflowTotalStake
	}

	// we do write zero here, because totalStake is allowed to be zero
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
	delegator sdk.AccAddress,
	target sdk.AccAddress,
	stake Uint) error {
	// 1. Check: delegatorStake(delegator) >= stake
	delegatorStake, err := k.stakeOwnedByDelegator.Get(ctx, delegator)
	if err != nil {
		return err
	}
	if stake.GT(delegatorStake) {
		return state.ErrIntegerUnderflowDelegator
	}

	// 2. Check: bonds(target, delegator) >= stake
	bond, err := k.stakePlacement.Get(ctx, collections.Join(delegator, target))
	if err != nil {
		return err
	}
	if stake.GT(bond) {
		return state.ErrIntegerUnderflowBonds
	}

	// 3. Check: targetStake(target) >= stake
	targetStake, err := k.stakePlacedUponTarget.Get(ctx, target)
	if err != nil {
		return err
	}
	if stake.GT(targetStake) {
		return state.ErrIntegerUnderflowTarget
	}

	// Perform State Updates
	// TODO: make this function prevent partial state updates / do rollbacks if any of the set statements fail
	// not necessary as long as callers are responsible, but it would be nice to have

	// delegatorStake(delegator) = delegatorStake(delegator) - stake
	delegatorStakeNew := delegatorStake.Sub(stake)
	if delegatorStakeNew.IsZero() {
		err = k.stakeOwnedByDelegator.Remove(ctx, delegator)
	} else {
		err = k.stakeOwnedByDelegator.Set(ctx, delegator, delegatorStakeNew)
	}
	if err != nil {
		return err
	}
	// bonds(target, delegator) = bonds(target, delegator) - stake
	bondNew := bond.Sub(stake)
	if bondNew.IsZero() {
		err = k.stakePlacement.Remove(ctx, collections.Join(delegator, target))
	} else {
		err = k.stakePlacement.Set(ctx, collections.Join(delegator, target), bondNew)
	}
	if err != nil {
		return err
	}

	// targetStake(target) = targetStake(target) - stake
	targetStakeNew := targetStake.Sub(stake)
	if targetStakeNew.IsZero() {
		err = k.stakePlacedUponTarget.Remove(ctx, target)
	} else {
		err = k.stakePlacedUponTarget.Set(ctx, target, targetStakeNew)
	}
	if err != nil {
		return err
	}

	return nil
}

// Used by Modify functions to change stake placements. This function subtracts from the stakePlacement mapping ONLY
// and does not modify any of the other stake mappings e.g. delegatorStake totalStake or topicStake in a system.
func (k *Keeper) SubStakePlacement(ctx context.Context, delegator sdk.AccAddress, target sdk.AccAddress, amount Uint) error {
	bond, err := k.GetBond(ctx, delegator, target)
	if err != nil {
		return err
	}
	if amount.GT(bond) {
		return state.ErrIntegerUnderflowBonds
	}
	bondNew := bond.Sub(amount)
	return k.stakePlacement.Set(ctx, collections.Join(delegator, target), bondNew)
}

// Used by Modify functions to change stake placements. This function adds to the stakePlacement mapping ONLY
// and does not modify any of the other stake mappings e.g. delegatorStake totalStake or topicStake in a system.
func (k *Keeper) AddStakePlacement(ctx context.Context, delegator sdk.AccAddress, target sdk.AccAddress, amount Uint) error {
	bond, err := k.GetBond(ctx, delegator, target)
	if err != nil {
		return err
	}
	bondNew := bond.Add(amount)
	return k.stakePlacement.Set(ctx, collections.Join(delegator, target), bondNew)
}

// Used by Modify functions to change stake placements. This function subtracts from the stakePlacedUponTarget mapping ONLY
// and does not modify any of the other stake mappings e.g. delegatorStake totalStake or topicStake in a system.
func (k *Keeper) SubStakePlacedUponTarget(ctx context.Context, target sdk.AccAddress, amount Uint) error {
	targetStake, err := k.GetStakePlacedUponTarget(ctx, target)
	if err != nil {
		return err
	}
	if amount.GT(targetStake) {
		return state.ErrIntegerUnderflowTarget
	}
	targetStakeNew := targetStake.Sub(amount)
	return k.stakePlacedUponTarget.Set(ctx, target, targetStakeNew)
}

// Used by Modify functions to change stake placements. This function adds to the stakePlacedUponTarget mapping ONLY
// and does not modify any of the other stake mappings e.g. delegatorStake totalStake or topicStake in a system.
func (k *Keeper) AddStakePlacedUponTarget(ctx context.Context, target sdk.AccAddress, amount Uint) error {
	targetStake, err := k.GetStakePlacedUponTarget(ctx, target)
	if err != nil {
		return err
	}
	targetStakeNew := targetStake.Add(amount)
	return k.stakePlacedUponTarget.Set(ctx, target, targetStakeNew)
}

// Add stake into an array of topics
func (k *Keeper) AddStakeToTopics(ctx context.Context, TopicIds []TOPIC_ID, stake Uint) error {
	if stake.IsZero() {
		return state.ErrDoNotSetMapValueToZero
	}

	// Calculate the total stake to be added across all topics
	totalStakeToAdd := stake.Mul(cosmosMath.NewUint(uint64(len(TopicIds))))

	for _, topicId := range TopicIds {
		topicStake, err := k.topicStake.Get(ctx, topicId)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				topicStake = cosmosMath.NewUint(0)
			} else {
				return err
			}
		}

		topicStakeNew := topicStake.Add(stake)
		if err := k.topicStake.Set(ctx, topicId, topicStakeNew); err != nil {
			return err
		}
	}

	// Update the allTopicStakeSum
	allTopicStakeSum, err := k.allTopicStakeSum.Get(ctx)
	if err != nil {
		return err
	}

	newAllTopicStakeSum := allTopicStakeSum.Add(totalStakeToAdd)
	if err := k.allTopicStakeSum.Set(ctx, newAllTopicStakeSum); err != nil {
		return err
	}

	return nil
}

// Remove stake from an array of topics
func (k *Keeper) RemoveStakeFromTopics(ctx context.Context, TopicIds []TOPIC_ID, stake Uint) error {
	if stake.IsZero() {
		return state.ErrDoNotSetMapValueToZero
	}

	// Calculate the total stake to be removed across all topics
	totalStakeToRemove := stake.Mul(cosmosMath.NewUint(uint64(len(TopicIds))))

	for _, topicId := range TopicIds {
		topicStake, err := k.topicStake.Get(ctx, topicId)
		if err != nil {
			return err // If there's an error, it's not because the topic doesn't exist but some other reason
		}

		if topicStake.LT(stake) {
			return state.ErrCannotRemoveMoreStakeThanStakedInTopic
		}

		topicStakeNew := topicStake.Sub(stake)
		if err := k.topicStake.Set(ctx, topicId, topicStakeNew); err != nil {
			return err
		}
	}

	// Update the allTopicStakeSum
	allTopicStakeSum, err := k.allTopicStakeSum.Get(ctx)
	if err != nil {
		return err
	}

	newAllTopicStakeSum := allTopicStakeSum.Sub(totalStakeToRemove)
	if err := k.allTopicStakeSum.Set(ctx, newAllTopicStakeSum); err != nil {
		return err
	}

	return nil
}

// for a given address, find out how much stake they've put into the system
func (k *Keeper) GetDelegatorStake(ctx context.Context, delegator sdk.AccAddress) (Uint, error) {
	ret, err := k.stakeOwnedByDelegator.Get(ctx, delegator)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return ret, nil
}

// For a given delegator and target, find out how much stake the delegator has placed upon the target
func (k *Keeper) GetBond(ctx context.Context, delegator sdk.AccAddress, target sdk.AccAddress) (Uint, error) {
	ret, err := k.stakePlacement.Get(ctx, collections.Join(delegator, target))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return ret, nil
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
		d := kv.K1()
		// if the delegator key matches the delegator we're looking for
		if d.Equals(delegator) {
			target := kv.K2()
			amount, err := k.stakePlacement.Get(ctx, kv)
			if err != nil {
				return nil, nil, err
			}
			targets = append(targets, target)
			amounts = append(amounts, amount)
		}
	}
	if len(targets) != len(amounts) {
		return nil, nil, state.ErrIterationLengthDoesNotMatch
	}

	return targets, amounts, nil
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

// Get the last time an inference was ran for a given topic
func (k *Keeper) GetTopicInferenceLastRan(ctx context.Context, topicId TOPIC_ID) (lastRanTime uint64, err error) {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return topic.InferenceLastRan, nil
}

// UpdateTopicInferenceLastRan updates the InferenceLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicInferenceLastRan(ctx context.Context, topicId TOPIC_ID, lastRanTime uint64) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	var newTopic state.Topic = state.Topic{
		Id:               topic.Id,
		Creator:          topic.Creator,
		Metadata:         topic.Metadata,
		WeightLogic:      topic.WeightLogic,
		WeightMethod:     topic.WeightMethod,
		WeightCadence:    topic.WeightCadence,
		WeightLastRan:    topic.WeightLastRan,
		InferenceLogic:   topic.InferenceLogic,
		InferenceMethod:  topic.InferenceMethod,
		InferenceCadence: topic.InferenceCadence,
		InferenceLastRan: lastRanTime,
		Active:           topic.Active,
		DefaultArg:       topic.DefaultArg,
	}
	return k.topics.Set(ctx, topicId, newTopic)
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

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *Keeper) InsertReputer(ctx context.Context, TopicIds []TOPIC_ID, reputer sdk.AccAddress, reputerInfo state.OffchainNode) error {
	for _, topicId := range TopicIds {
		topicKey := collections.Join[uint64, sdk.AccAddress](topicId, reputer)
		err := k.topicReputers.Set(ctx, topicKey)
		if err != nil {
			return err
		}
	}
	err := k.reputers.Set(ctx, reputerInfo.LibP2PKey, reputerInfo)
	if err != nil {
		return err
	}
	err = k.AddAddressTopics(ctx, reputer, TopicIds)
	if err != nil {
		return err
	}
	return nil
}

// Remove a reputer to the reputer tracking data structures and topicReputers
func (k *Keeper) RemoveReputer(ctx context.Context, topicId TOPIC_ID, reputerAddr sdk.AccAddress) error {

	topicKey := collections.Join[uint64, sdk.AccAddress](topicId, reputerAddr)
	err := k.topicReputers.Remove(ctx, topicKey)
	if err != nil {
		return err
	}

	err = k.RemoveAddressTopic(ctx, reputerAddr, topicId)
	if err != nil {
		return err
	}
	return nil
}

// Remove a worker to the worker tracking data structures and topicWorkers
func (k *Keeper) RemoveWorker(ctx context.Context, topicId TOPIC_ID, workerAddr sdk.AccAddress) error {

	topicKey := collections.Join[uint64, sdk.AccAddress](topicId, workerAddr)
	err := k.topicWorkers.Remove(ctx, topicKey)
	if err != nil {
		return err
	}

	err = k.RemoveAddressTopic(ctx, workerAddr, topicId)
	if err != nil {
		return err
	}
	return nil
}

// Adds a new worker to the worker tracking data structures, workers and topicWorkers
func (k *Keeper) InsertWorker(ctx context.Context, TopicIds []TOPIC_ID, worker sdk.AccAddress, workerInfo state.OffchainNode) error {
	for _, topicId := range TopicIds {
		topickey := collections.Join[uint64, sdk.AccAddress](topicId, worker)
		err := k.topicWorkers.Set(ctx, topickey)
		if err != nil {
			return err
		}
	}
	err := k.workers.Set(ctx, workerInfo.LibP2PKey, workerInfo)
	if err != nil {
		return err
	}
	err = k.AddAddressTopics(ctx, worker, TopicIds)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) FindWorkerNodesByOwner(ctx sdk.Context, nodeId string) ([]*state.OffchainNode, error) {
	var nodes []*state.OffchainNode
	var nodeIdParts = strings.Split(nodeId, "|")

	if len(nodeIdParts) < 2 {
		nodeIdParts = append(nodeIdParts, "")
	}

	owner, libp2pkey := nodeIdParts[0], nodeIdParts[1]

	iterator, err := k.workers.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	for ; iterator.Valid(); iterator.Next() {
		node, _ := iterator.Value()
		if node.Owner == owner && len(libp2pkey) == 0 || node.Owner == owner && node.LibP2PKey == libp2pkey {
			nodes = append(nodes, &node)
		}
	}

	return nodes, nil
}

func (k *Keeper) GetWorkerAddressByP2PKey(ctx context.Context, p2pKey string) (sdk.AccAddress, error) {
	worker, err := k.workers.Get(ctx, p2pKey)
	if err != nil {
		return nil, err
	}

	workerAddress, err := sdk.AccAddressFromBech32(worker.GetOwner())
	if err != nil {
		return nil, err
	}

	return workerAddress, nil
}

func (k *Keeper) SetWeight(
	ctx context.Context,
	topicId TOPIC_ID,
	reputer sdk.AccAddress,
	worker sdk.AccAddress,
	weight Uint) error {
	key := collections.Join3(topicId, reputer, worker)
	if weight.IsZero() {
		return k.weights.Remove(ctx, key)
	}
	return k.weights.Set(ctx, key, weight)
}

func (k *Keeper) SetInference(
	ctx context.Context,
	topicId TOPIC_ID,
	worker sdk.AccAddress,
	inference state.Inference) error {
	key := collections.Join(topicId, worker)
	err := k.inferences.Set(ctx, key, inference)
	if err != nil {
		return err
	}
	return k.IncrementNumInferencesInRewardEpoch(ctx, topicId, worker)
}

// for a given delegator, get their stake removal information
func (k *Keeper) GetStakeRemovalQueueForDelegator(ctx context.Context, delegator sdk.AccAddress) (state.StakeRemoval, error) {
	return k.stakeRemovalQueue.Get(ctx, delegator)
}

// For a given delegator, adds their stake removal information to the removal queue for delay waiting
func (k *Keeper) SetStakeRemovalQueueForDelegator(ctx context.Context, delegator sdk.AccAddress, removalInfo state.StakeRemoval) error {
	return k.stakeRemovalQueue.Set(ctx, delegator, removalInfo)
}

func (k *Keeper) AddUnmetDemand(ctx context.Context, topicId TOPIC_ID, amt cosmosMath.Uint) error {
	topicUnmetDemand, err := k.GetTopicUnmetDemand(ctx, topicId)
	if err != nil {
		return err
	}
	topicUnmetDemand = topicUnmetDemand.Add(amt)
	return k.topicUnmetDemand.Set(ctx, topicId, topicUnmetDemand)
}

func (k *Keeper) RemoveUnmetDemand(ctx context.Context, topicId TOPIC_ID, amt cosmosMath.Uint) error {
	topicUnmetDemand, err := k.topicUnmetDemand.Get(ctx, topicId)
	if err != nil {
		return err
	}
	if amt.GT(topicUnmetDemand) {
		return state.ErrIntegerUnderflowUnmetDemand
	}
	topicUnmetDemand = topicUnmetDemand.Sub(amt)
	return k.SetTopicUnmetDemand(ctx, topicId, topicUnmetDemand)
}

func (k *Keeper) SetTopicUnmetDemand(ctx context.Context, topicId TOPIC_ID, amt cosmosMath.Uint) error {
	if amt.IsZero() {
		return k.topicUnmetDemand.Remove(ctx, topicId)
	}
	return k.topicUnmetDemand.Set(ctx, topicId, amt)
}

func (k *Keeper) GetTopicUnmetDemand(ctx context.Context, topicId TOPIC_ID) (Uint, error) {
	topicUnmetDemand, err := k.topicUnmetDemand.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		} else {
			return cosmosMath.Uint{}, err
		}
	}
	return topicUnmetDemand, nil
}

func (k *Keeper) AddToMempool(ctx context.Context, request state.InferenceRequest) error {
	requestId, err := request.GetRequestId()
	if err != nil {
		return err
	}
	key := collections.Join(request.TopicId, requestId)
	err = k.mempool.Set(ctx, key, request)
	if err != nil {
		return err
	}

	return k.AddUnmetDemand(ctx, request.TopicId, request.BidAmount)
}

func (k *Keeper) RemoveFromMempool(ctx context.Context, request state.InferenceRequest) error {
	requestId, err := request.GetRequestId()
	if err != nil {
		return err
	}
	key := collections.Join(request.TopicId, requestId)
	err = k.mempool.Remove(ctx, key)
	if err != nil {
		return err
	}

	return k.RemoveUnmetDemand(ctx, request.TopicId, request.BidAmount)
}

func (k *Keeper) IsRequestInMempool(ctx context.Context, topicId TOPIC_ID, requestId string) (bool, error) {
	return k.mempool.Has(ctx, collections.Join(topicId, requestId))
}

func (k *Keeper) GetMempoolInferenceRequestById(ctx context.Context, topicId TOPIC_ID, requestId string) (state.InferenceRequest, error) {
	return k.mempool.Get(ctx, collections.Join(topicId, requestId))
}

func (k *Keeper) GetMempoolInferenceRequestsForTopic(ctx context.Context, topicId TOPIC_ID) ([]state.InferenceRequest, error) {
	var ret []state.InferenceRequest = make([]state.InferenceRequest, 0)
	rng := collections.NewPrefixedPairRange[TOPIC_ID, REQUEST_ID](topicId)
	iter, err := k.mempool.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}
	for ; iter.Valid(); iter.Next() {
		value, err := iter.Value()
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}
	return ret, nil
}

func (k *Keeper) InactivateTopic(ctx context.Context, topicId TOPIC_ID) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}

	topic.Active = false

	err = k.topics.Set(ctx, topicId, topic)
	if err != nil {
		return err
	}

	return nil
}

func (k *Keeper) ReactivateTopic(ctx context.Context, topicId TOPIC_ID) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}

	topic.Active = true

	err = k.topics.Set(ctx, topicId, topic)
	if err != nil {
		return err
	}

	return nil
}

func (k *Keeper) GetParamsMaxMissingInferencePercent(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxMissingInferencePercent, nil
}

func (k *Keeper) GetParamsMaxTopicsPerBlock(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxTopicsPerBlock, nil
}

func (k *Keeper) GetParamsMinRequestUnmetDemand(ctx context.Context) (Uint, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return params.MinRequestUnmetDemand, nil
}

func (k *Keeper) GetParamsMinTopicUnmetDemand(ctx context.Context) (Uint, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return params.MinTopicUnmetDemand, nil
}

func (k *Keeper) GetParamsRequiredMinimumStake(ctx context.Context) (Uint, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return params.RequiredMinimumStake, nil
}

func (k *Keeper) GetParamsRemoveStakeDelayWindow(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.RemoveStakeDelayWindow, nil
}

func (k *Keeper) GetParamsMaxInferenceRequestValidity(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxInferenceRequestValidity, nil
}

func (k *Keeper) GetParamsMinRequestCadence(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MinRequestCadence, nil
}

func (k *Keeper) GetParamsMinWeightCadence(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MinWeightCadence, nil
}

func (k *Keeper) GetParamsMaxRequestCadence(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxRequestCadence, nil
}

func (k *Keeper) GetMempool(ctx context.Context) ([]state.InferenceRequest, error) {
	var ret []state.InferenceRequest = make([]state.InferenceRequest, 0)
	iter, err := k.mempool.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	for ; iter.Valid(); iter.Next() {
		value, err := iter.Value()
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}
	return ret, nil
}

func (k *Keeper) SetRequestDemand(ctx context.Context, requestId string, amount Uint) error {
	if amount.IsZero() {
		return k.requestUnmetDemand.Remove(ctx, requestId)
	}
	return k.requestUnmetDemand.Set(ctx, requestId, amount)
}

func (k *Keeper) GetRequestDemand(ctx context.Context, requestId string) (Uint, error) {
	return k.requestUnmetDemand.Get(ctx, requestId)
}

func (k *Keeper) GetTopicAccumulatedMetDemand(ctx context.Context, topicId TOPIC_ID) (Uint, error) {
	res, err := k.accumulatedMetDemand.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return res, nil
}

func (k *Keeper) AddTopicAccumulateMetDemand(ctx context.Context, topicId TOPIC_ID, metDemand Uint) error {
	currentMetDemand, err := k.accumulatedMetDemand.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	currentMetDemand = currentMetDemand.Add(metDemand)
	return k.SetTopicAccumulatedMetDemand(ctx, topicId, currentMetDemand)
}

func (k *Keeper) SetTopicAccumulatedMetDemand(ctx context.Context, topicId TOPIC_ID, metDemand Uint) error {
	if metDemand.IsZero() {
		return k.accumulatedMetDemand.Remove(ctx, topicId)
	}
	return k.accumulatedMetDemand.Set(ctx, topicId, metDemand)
}

func (k *Keeper) GetNumInferencesInRewardEpoch(ctx context.Context, topicId TOPIC_ID, worker sdk.AccAddress) (Uint, error) {
	key := collections.Join(topicId, worker)
	res, err := k.numInferencesInRewardEpoch.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return res, nil
}

func (k *Keeper) IncrementNumInferencesInRewardEpoch(ctx context.Context, topicId TOPIC_ID, worker sdk.AccAddress) error {
	key := collections.Join(topicId, worker)
	currentNumInferences, err := k.numInferencesInRewardEpoch.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			currentNumInferences = cosmosMath.NewUint(0)
		} else {
			return err
		}
	}
	newNumInferences := currentNumInferences.Add(cosmosMath.NewUint(1))
	return k.numInferencesInRewardEpoch.Set(ctx, key, newNumInferences)
}

// Reset the mapping entirely. Should be called at the end of every block
func (k *Keeper) ResetChurnReadyTopics(ctx context.Context) error {
	return k.churnReadyTopics.Remove(ctx)
}

// Set a topic as churn ready
func (k *Keeper) SetChurnReadyTopics(ctx context.Context, topicList state.TopicList) error {
	return k.churnReadyTopics.Set(ctx, topicList)
}

// Get all churn ready topics
func (k *Keeper) GetChurnReadyTopics(ctx context.Context) (state.TopicList, error) {
	topicList, err := k.churnReadyTopics.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return state.TopicList{}, nil
		}
		return state.TopicList{}, err
	}
	return topicList, nil
}

// Reset the mapping entirely. Should be called at the end of every reward epoch
func (k *Keeper) ResetNumInferencesInRewardEpoch(ctx context.Context) error {
	iter, err := k.numInferencesInRewardEpoch.Iterate(ctx, nil)
	if err != nil {
		return err
	}

	// Iterate over all keys
	kvs, err := iter.Keys()
	if err != nil {
		return err
	}
	for _, kv := range kvs {
		err := k.numInferencesInRewardEpoch.Remove(ctx, kv)
		if err != nil {
			return err
		}
	}

	return nil
}

//
// WHITELIST FUNCTIONS
//

func (k *Keeper) IsWhitelistAdmin(ctx context.Context, admin sdk.AccAddress) (bool, error) {
	return k.whitelistAdmins.Has(ctx, admin)
}

func (k *Keeper) AddWhitelistAdmin(ctx context.Context, admin sdk.AccAddress) error {
	return k.whitelistAdmins.Set(ctx, admin)
}

func (k *Keeper) RemoveWhitelistAdmin(ctx context.Context, admin sdk.AccAddress) error {
	return k.whitelistAdmins.Remove(ctx, admin)
}

func (k *Keeper) IsInTopicCreationWhitelist(ctx context.Context, address sdk.AccAddress) (bool, error) {
	return k.topicCreationWhitelist.Has(ctx, address)
}

func (k *Keeper) AddToTopicCreationWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.topicCreationWhitelist.Set(ctx, address)
}

func (k *Keeper) RemoveFromTopicCreationWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.topicCreationWhitelist.Remove(ctx, address)
}

func (k *Keeper) IsInWeightSettingWhitelist(ctx context.Context, address sdk.AccAddress) (bool, error) {
	return k.weightSettingWhitelist.Has(ctx, address)
}

func (k *Keeper) AddToWeightSettingWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.weightSettingWhitelist.Set(ctx, address)
}

func (k *Keeper) RemoveFromWeightSettingWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.weightSettingWhitelist.Remove(ctx, address)
}

//
// BANK KEEPER WRAPPERS
//

// SendCoinsFromModuleToModule
func (k *Keeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt)
}
