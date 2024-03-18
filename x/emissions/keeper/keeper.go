package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cosmosMath "cosmossdk.io/math"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Uint = cosmosMath.Uint
type Int = cosmosMath.Int

type TOPIC_ID = uint64
type LIB_P2P_KEY = string
type DELEGATOR = sdk.AccAddress
type WORKER = sdk.AccAddress
type REPUTER = sdk.AccAddress
type ACC_ADDRESS = string
type WORKERS = string
type REPUTERS = string
type BLOCK_NUMBER = int64
type UNIX_TIMESTAMP = uint64
type REQUEST_ID = string

type Keeper struct {
	cdc          codec.BinaryCodec
	addressCodec address.Codec

	// types management
	schema     collections.Schema
	params     collections.Item[types.Params]
	authKeeper AccountKeeper
	bankKeeper BankKeeper

	// ############################################
	// #             TOPIC types:                 #
	// ############################################

	// the next topic id to be used, equal to the number of topics that have been created
	nextTopicId collections.Sequence
	// every topic that has been created indexed by their topicId starting from 1 (0 is reserved for the root network)
	topics collections.Map[TOPIC_ID, types.Topic]
	// every topics that has been churned and ready to get inferences in the block
	churnReadyTopics collections.Item[types.TopicList]
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TOPIC_ID, sdk.AccAddress]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TOPIC_ID, sdk.AccAddress]]
	// for an address, what are all the topics that it's registered for?
	addressTopics collections.Map[sdk.AccAddress, []uint64]

	// ############################################
	// #                STAKING                   #
	// ############################################

	// total sum stake of all stakers on the network
	totalStake collections.Item[Uint]
	// for every topic, how much total stake does that topic have accumulated?
	topicStake collections.Map[TOPIC_ID, Uint]
	// amount of stake a reputer has placed in a topic, signalling their authority on the topic
	stakeByReputerAndTopicId collections.Map[collections.Pair[TOPIC_ID, REPUTER], Uint]
	// map of (reputer) -> removal information for that reputer
	stakeRemovalQueue collections.Map[REPUTER, types.StakeRemoval]
	// map of (delegator) -> removal information for that delegator
	delegatedStakeRemovalQueue collections.Map[DELEGATOR, types.DelegatedStakeRemoval]
	// map of (delegator) -> amount of stake that has been placed by that delegator
	stakeFromDelegator collections.Map[collections.Pair[TOPIC_ID, DELEGATOR], Uint]
	// map of (delegator, target) -> amount of stake that has been placed by that delegator on that target
	delegatedStakePlacement collections.Map[collections.Triple[TOPIC_ID, REPUTER, DELEGATOR], Uint]
	// map of (target) -> amount of stake that has been placed on that target
	stakeUponReputer collections.Map[collections.Pair[TOPIC_ID, REPUTER], Uint]

	// ############################################
	// #        INFERENCE REQUEST MEMPOOL         #
	// ############################################

	// map of (topic, request_id) -> full InferenceRequest information for that request
	mempool collections.Map[collections.Pair[TOPIC_ID, REQUEST_ID], types.InferenceRequest]
	// amount of money available for an inference request id that has been placed in the mempool but has not yet been fully satisfied
	requestUnmetDemand collections.Map[REQUEST_ID, Uint]
	// total amount of demand for a topic that has been placed in the mempool as a request for inference but has not yet been satisfied
	topicUnmetDemand collections.Map[TOPIC_ID, Uint]

	// ############################################
	// #            MISC GLOBAL STATE:            #
	// ############################################

	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TOPIC_ID, WORKER], types.Inference]

	// map of (topic, worker) -> forecast[]
	forecasts collections.Map[collections.Pair[TOPIC_ID, WORKER], types.Forecast]

	// map of (topic, worker) -> num_inferences_in_reward_epoch
	numInferencesInRewardEpoch collections.Map[collections.Pair[TOPIC_ID, WORKER], Uint]

	// map of worker id to node data about that worker
	workers collections.Map[LIB_P2P_KEY, types.OffchainNode]

	// map of reputer id to node data about that reputer
	reputers collections.Map[LIB_P2P_KEY, types.OffchainNode]

	// the last block the token inflation rewards were updated: int64 same as BlockHeight()
	lastRewardsUpdate collections.Item[BLOCK_NUMBER]

	// map of (topic, timestamp, index) -> Inference
	allInferences collections.Map[collections.Pair[TOPIC_ID, UNIX_TIMESTAMP], types.Inferences]

	// map of (topic, timestamp, index) -> Forecast
	allForecasts collections.Map[collections.Pair[TOPIC_ID, UNIX_TIMESTAMP], types.Forecasts]

	// map of (topic, timestamp, index) -> LossBundle
	allLossBundles collections.Map[collections.Pair[TOPIC_ID, UNIX_TIMESTAMP], types.LossBundles]

	// map of (topic, timestamp, index) -> LossBundle (1 network wide bundle per timestep)
	networkLossBundles collections.Map[collections.Pair[TOPIC_ID, UNIX_TIMESTAMP], types.LossBundle]

	accumulatedMetDemand collections.Map[TOPIC_ID, Uint]

	whitelistAdmins collections.KeySet[sdk.AccAddress]

	topicCreationWhitelist collections.KeySet[sdk.AccAddress]

	reputerWhitelist collections.KeySet[sdk.AccAddress]

	foundationWhitelist collections.KeySet[sdk.AccAddress]
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
		params:                     collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		authKeeper:                 ak,
		bankKeeper:                 bk,
		totalStake:                 collections.NewItem(sb, types.TotalStakeKey, "total_stake", UintValue),
		topicStake:                 collections.NewMap(sb, types.TopicStakeKey, "topic_stake", collections.Uint64Key, UintValue),
		lastRewardsUpdate:          collections.NewItem(sb, types.LastRewardsUpdateKey, "last_rewards_update", collections.Int64Value),
		nextTopicId:                collections.NewSequence(sb, types.NextTopicIdKey, "next_topic_id"),
		topics:                     collections.NewMap(sb, types.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[types.Topic](cdc)),
		churnReadyTopics:           collections.NewItem(sb, types.ChurnReadyTopicsKey, "churn_ready_topics", codec.CollValue[types.TopicList](cdc)),
		topicWorkers:               collections.NewKeySet(sb, types.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		addressTopics:              collections.NewMap(sb, types.AddressTopicsKey, "address_topics", sdk.AccAddressKey, TopicIdListValue),
		topicReputers:              collections.NewKeySet(sb, types.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		stakeByReputerAndTopicId:   collections.NewMap(sb, types.StakeRemovalQueueKey, "stake_by_reputer_and_topic_id", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		stakeRemovalQueue:          collections.NewMap(sb, types.StakeRemovalQueueKey, "stake_removal_queue", sdk.AccAddressKey, codec.CollValue[types.StakeRemoval](cdc)),
		delegatedStakeRemovalQueue: collections.NewMap(sb, types.DelegatedStakeRemovalQueueKey, "delegated_stake_removal_queue", sdk.AccAddressKey, codec.CollValue[types.DelegatedStakeRemoval](cdc)),
		stakeFromDelegator:         collections.NewMap(sb, types.DelegatorStakeKey, "stake_from_delegator", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		delegatedStakePlacement:    collections.NewMap(sb, types.BondsKey, "delegated_stake_placement", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), UintValue),
		stakeUponReputer:           collections.NewMap(sb, types.TargetStakeKey, "stake_upon_reputer", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		mempool:                    collections.NewMap(sb, types.MempoolKey, "mempool", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.InferenceRequest](cdc)),
		requestUnmetDemand:         collections.NewMap(sb, types.RequestUnmetDemandKey, "request_unmet_demand", collections.StringKey, UintValue),
		topicUnmetDemand:           collections.NewMap(sb, types.TopicUnmetDemandKey, "topic_unmet_demand", collections.Uint64Key, UintValue),
		inferences:                 collections.NewMap(sb, types.InferencesKey, "inferences", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Inference](cdc)),
		forecasts:                  collections.NewMap(sb, types.ForecastsKey, "forecasts", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Forecast](cdc)),
		workers:                    collections.NewMap(sb, types.WorkerNodesKey, "worker_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		reputers:                   collections.NewMap(sb, types.ReputerNodesKey, "reputer_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		allInferences:              collections.NewMap(sb, types.AllInferencesKey, "inferences_all", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), codec.CollValue[types.Inferences](cdc)),
		allForecasts:               collections.NewMap(sb, types.AllForecastsKey, "forecasts_all", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), codec.CollValue[types.Forecasts](cdc)),
		allLossBundles:             collections.NewMap(sb, types.AllLossBundlesKey, "loss_bundles_all", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), codec.CollValue[types.LossBundles](cdc)),
		networkLossBundles:         collections.NewMap(sb, types.NetworkLossBundlesKey, "loss_bundles_network", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), codec.CollValue[types.LossBundle](cdc)),
		accumulatedMetDemand:       collections.NewMap(sb, types.AccumulatedMetDemandKey, "accumulated_met_demand", collections.Uint64Key, UintValue),
		numInferencesInRewardEpoch: collections.NewMap(sb, types.NumInferencesInRewardEpochKey, "num_inferences_in_reward_epoch", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		whitelistAdmins:            collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", sdk.AccAddressKey),
		topicCreationWhitelist:     collections.NewKeySet(sb, types.TopicCreationWhitelistKey, "topic_creation_whitelist", sdk.AccAddressKey),
		reputerWhitelist:           collections.NewKeySet(sb, types.ReputerWhitelistKey, "weight_setting_whitelist", sdk.AccAddressKey),
		foundationWhitelist:        collections.NewKeySet(sb, types.FoundationWhitelistKey, "foundation_whitelist", sdk.AccAddressKey),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.schema = schema

	return k
}

///
/// PARAMETERS
///

func (k *Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.params.Set(ctx, params)
}

func (k *Keeper) GetParams(ctx context.Context) (types.Params, error) {
	ret, err := k.params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.DefaultParams(), nil
		}
		return types.Params{}, err
	}
	return ret, nil
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

func (k *Keeper) GetParamsMinLossCadence(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MinLossCadence, nil
}

///
/// INFERENCES
///

func (k *Keeper) GetAllInferences(ctx context.Context, topicId TOPIC_ID, timestamp uint64) (*types.Inferences, error) {
	// pair := collections.Join(topicId, timestamp, index)
	key := collections.Join(topicId, timestamp)
	inferences, err := k.allInferences.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &inferences, nil
}

// Insert a complete set of inferences for a topic/timestamp. Overwrites previous ones.
func (k *Keeper) InsertInferences(ctx context.Context, topicId TOPIC_ID, timestamp uint64, inferences types.Inferences) error {
	for _, inference := range inferences.Inferences {
		// Update latests inferences for each worker
		workerAcc, err := sdk.AccAddressFromBech32(inference.Worker)
		if err != nil {
			return err
		}
		key := collections.Join(topicId, workerAcc)
		err = k.inferences.Set(ctx, key, *inference)
		if err != nil {
			return err
		}
		// Update the number of inferences in the reward epoch for each worker
		err = k.IncrementNumInferencesInRewardEpoch(ctx, topicId, workerAcc)
		if err != nil {
			return err
		}
	}

	key := collections.Join(topicId, timestamp)
	return k.allInferences.Set(ctx, key, inferences)
}

// Insert a complete set of inferences for a topic/timestamp. Overwrites previous ones.
func (k *Keeper) InsertForecasts(ctx context.Context, topicId TOPIC_ID, timestamp uint64, forecasts types.Forecasts) error {
	for _, forecast := range forecasts.Forecasts {
		// Update latests forecasts for each worker
		workerAcc, err := sdk.AccAddressFromBech32(forecast.Forecaster)
		if err != nil {
			return err
		}
		key := collections.Join(topicId, workerAcc)
		err = k.forecasts.Set(ctx, key, *forecast)
		if err != nil {
			return err
		}
		// // Update the number of forecasts in the reward epoch for each forecaster
		// err = k.IncrementNumForecastsInRewardEpoch(ctx, topicId, workerAcc)
		// if err != nil {
		// 	return err
		// }
	}

	key := collections.Join(topicId, timestamp)
	return k.allForecasts.Set(ctx, key, forecasts)
}

// Insert a loss bundle for a topic/timestamp. Overwrites previous ones.
func (k *Keeper) InsertLossBudles(ctx context.Context, topicId TOPIC_ID, timestamp uint64, lossBundles types.LossBundles) error {
	key := collections.Join(topicId, timestamp)
	return k.allLossBundles.Set(ctx, key, lossBundles)
}

func (k *Keeper) GetWorkerLatestInferenceByTopicId(
	ctx context.Context,
	topicId TOPIC_ID,
	worker sdk.AccAddress) (types.Inference, error) {
	key := collections.Join(topicId, worker)
	return k.inferences.Get(ctx, key)
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
		return types.ErrBlockHeightNegative
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
		return types.ErrBlockHeightLessThanPrevious
	}
	return k.lastRewardsUpdate.Set(ctx, blockHeight)
}

// return epoch length
func (k *Keeper) GetParamsRewardCadence(ctx context.Context) (int64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.RewardCadence, nil
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
	epochsPassed := cosmosMath.NewInt(blocksSinceLastUpdate / params.RewardCadence)
	// get emission amount
	return epochsPassed.Mul(params.EmissionsPerEpoch), nil
}

// mint new rewards coins to this module account
func (k *Keeper) MintRewardsCoins(ctx context.Context, amount cosmosMath.Int) error {
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))
	return k.bankKeeper.MintCoins(ctx, types.AlloraStakingModuleName, coins)
}

// A function that accepts a topicId and returns list of Inferences or error
func (k *Keeper) GetLatestInferencesFromTopic(ctx context.Context, topicId TOPIC_ID) ([]*types.InferenceSetForScoring, error) {
	var inferences []*types.InferenceSetForScoring
	var latestTimestamp, err = k.GetTopicLossCalcLastRan(ctx, topicId)
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
		inferenceSet := &types.InferenceSetForScoring{
			TopicId:    key.K1(),
			Timestamp:  key.K2(),
			Inferences: &value,
		}
		inferences = append(inferences, inferenceSet)
	}
	return inferences, nil
}

// A function that accepts a topicId and returns list of Forecasts or error
func (k *Keeper) GetLatestForecastsFromTopic(ctx context.Context, topicId TOPIC_ID) ([]*types.ForecastSetForScoring, error) {
	var forecasts []*types.ForecastSetForScoring
	var latestTimestamp, err = k.GetTopicLossCalcLastRan(ctx, topicId)
	if err != nil {
		latestTimestamp = 0
	}
	rng := collections.
		NewPrefixedPairRange[TOPIC_ID, UNIX_TIMESTAMP](topicId).
		StartInclusive(latestTimestamp).
		Descending()

	iter, err := k.allForecasts.Iterate(ctx, rng)
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
		forecastSet := &types.ForecastSetForScoring{
			TopicId:   key.K1(),
			Timestamp: key.K2(),
			Forecasts: &value,
		}
		forecasts = append(forecasts, forecastSet)
	}
	return forecasts, nil
}

// A function that accepts a topicId and returns list of LossBu or error

// GetTopicsByCreator returns a slice of all topics created by a given creator.
func (k *Keeper) GetTopicsByCreator(ctx context.Context, creator string) ([]*types.Topic, error) {
	var topicsByCreator []*types.Topic

	err := k.topics.Walk(ctx, nil, func(id TOPIC_ID, topic types.Topic) (bool, error) {
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

///
/// STAKING
///

// Adds stake to the system for a given topic and reputer
func (k *Keeper) AddStake(ctx context.Context, topicId TOPIC_ID, reputer sdk.AccAddress, stake Uint) error {
	// Run checks to ensure that the stake can be added, and then update the types all at once, applying rollbacks if necessary
	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	// Get new reputer stake in topic
	topicReputerKey := collections.Join(topicId, reputer)
	reputerStakeInTopic, err := k.stakeByReputerAndTopicId.Get(ctx, topicReputerKey)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			reputerStakeInTopic = cosmosMath.NewUint(0)
		} else {
			return err
		}
	}
	reputerStakeNew := reputerStakeInTopic.Add(stake)

	// Get new sum topic stake for all topics
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return err
	}
	topicStakeNew := topicStake.Add(stake)

	// Get new sum topic stake for all topics
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return err
	}
	totalStakeNew := totalStake.Add(stake)

	// State updates -- done all at once after all checks

	// Set new reputer stake in topic
	if err := k.stakeByReputerAndTopicId.Set(ctx, topicReputerKey, reputerStakeNew); err != nil {
		return err
	}

	// Set new sum topic stake for all topics
	if err := k.topicStake.Set(ctx, topicId, topicStakeNew); err != nil {
		fmt.Println("Setting topic stake failed -- rolling back reputer stake")
		// Rollback reputer stake in topic
		err2 := k.stakeByReputerAndTopicId.Set(ctx, topicReputerKey, reputerStakeInTopic)
		if err2 != nil {
			return err2
		}
		return err
	}

	if err := k.totalStake.Set(ctx, totalStakeNew); err != nil {
		fmt.Println("Setting total stake failed -- rolling back reputer and topic stake")
		// Rollback reputer stake in topic
		err2 := k.stakeByReputerAndTopicId.Set(ctx, topicReputerKey, reputerStakeInTopic)
		if err2 != nil {
			return err2
		}
		// Rollback topic stake
		err2 = k.topicStake.Set(ctx, topicId, topicStake)
		if err2 != nil {
			return err2
		}
		return err
	}

	return nil
}

func (k *Keeper) AddDelegatedStake(ctx context.Context, topicId TOPIC_ID, delegator sdk.AccAddress, reputer sdk.AccAddress, stake Uint) error {
	// Run checks to ensure that delegate stake can be added, and then update the types all at once, applying rollbacks if necessary
	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	stakeFromDelegator, err := k.GetStakeFromDelegator(ctx, topicId, delegator)
	if err != nil {
		return err
	}
	stakeFromDelegatorNew := stakeFromDelegator.Add(stake)

	delegatedStakePlacement, err := k.GetDelegatedStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return err
	}
	stakePlacementNew := delegatedStakePlacement.Add(stake)

	stakeUponReputer, err := k.GetDelegatedStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	stakeUponReputerNew := stakeUponReputer.Add(stake)

	// types updates -- done all at once after all checks

	// Set new reputer stake in topic
	if err := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegatorNew); err != nil {
		return err
	}

	// Set new sum topic stake for all topics
	if err := k.SetDelegatedStakePlacement(ctx, topicId, delegator, reputer, stakePlacementNew); err != nil {
		fmt.Println("Setting topic stake failed -- rolling back stake from delegator")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		return err
	}

	if err := k.SetDelegatedStakeUponReputer(ctx, topicId, reputer, stakeUponReputerNew); err != nil {
		fmt.Println("Setting total stake failed -- rolling back stake from delegator and delegated stake placement")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		err2 = k.SetDelegatedStakePlacement(ctx, topicId, delegator, reputer, delegatedStakePlacement)
		if err2 != nil {
			return err2
		}
		return err
	}

	return nil
}

// Removes stake to the system for a given topic and reputer
func (k *Keeper) RemoveStake(
	ctx context.Context,
	topicId TOPIC_ID,
	reputer sdk.AccAddress,
	stake Uint) error {
	// Run checks to ensure that the stake can be removed, and then update the types all at once, applying rollbacks if necessary

	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	// Check reputerStakeInTopic >= stake
	topicReputerKey := collections.Join(topicId, reputer)
	reputerStakeInTopic, err := k.stakeByReputerAndTopicId.Get(ctx, topicReputerKey)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ErrTopicReputerStakeDoesNotExist
		} else {
			return err
		}
	}
	delegatedStakeUponReputerInTopic, err := k.GetDelegatedStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	reputerStakeInTopicWithoutDelegatedStake := reputerStakeInTopic.Sub(delegatedStakeUponReputerInTopic)
	// TODO Maybe we should check if reputerStakeInTopicWithoutDelegatedStake is zero and remove the key from the map
	if stake.GT(reputerStakeInTopicWithoutDelegatedStake) {
		return types.ErrIntegerUnderflowTopicReputerStake
	}
	reputerStakeNew := reputerStakeInTopic.Sub(stake)

	// Check topicStake >= stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return err
	}
	if stake.GT(topicStake) {
		return types.ErrIntegerUnderflowTopicStake
	}
	topicStakeNew := topicStake.Sub(stake)

	// Check totalStake >= stake
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return err
	}
	if stake.GT(totalStake) {
		return types.ErrIntegerUnderflowTotalStake
	}

	// types updates -- done all at once after all checks

	// Set topic-reputer stake
	if reputerStakeNew.IsZero() {
		err = k.stakeByReputerAndTopicId.Remove(ctx, topicReputerKey)
	} else {
		err = k.stakeByReputerAndTopicId.Set(ctx, topicReputerKey, reputerStakeNew)
	}
	if err != nil {
		return err
	}

	// Set topic stake
	if topicStakeNew.IsZero() {
		err = k.topicStake.Remove(ctx, topicId)
	} else {
		err = k.topicStake.Set(ctx, topicId, topicStakeNew)
	}
	if err != nil {
		fmt.Println("Setting topic stake failed -- rolling back topic-reputer stake")
		err2 := k.stakeByReputerAndTopicId.Set(ctx, topicReputerKey, reputerStakeInTopic)
		if err2 != nil {
			fmt.Println("Error rolling back topic-reputer stake")
		}
		return err
	}

	// Set total stake
	err = k.SetTotalStake(ctx, totalStake.Sub(stake))
	if err != nil {
		fmt.Println("Setting total stake failed -- rolling back topic-reputer and topic stake")
		err2 := k.stakeByReputerAndTopicId.Set(ctx, topicReputerKey, reputerStakeInTopic)
		if err2 != nil {
			fmt.Println("Error rolling back topic-reputer stake")
		}
		err2 = k.topicStake.Set(ctx, topicId, topicStake)
		if err2 != nil {
			fmt.Println("Error rolling back topic stake")
		}
		return err
	}

	return nil
}

// Removes delegated stake from the system for a given topic, delegator, and reputer
func (k *Keeper) RemoveDelegatedStake(
	ctx context.Context,
	topicId TOPIC_ID,
	delegator sdk.AccAddress,
	reputer sdk.AccAddress,
	stake Uint) error {
	// Run checks to ensure that the delegated stake can be removed, and then update the types all at once, applying rollbacks if necessary

	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	// Check stakeFromDelegator >= stake
	stakeFromDelegator, err := k.GetStakeFromDelegator(ctx, topicId, delegator)
	if err != nil {
		return err
	}
	if stake.GT(stakeFromDelegator) {
		return types.ErrIntegerUnderflowStakeFromDelegator
	}
	stakeFromDelegatorNew := stakeFromDelegator.Sub(stake)

	// Check stakePlacement >= stake
	stakePlacement, err := k.GetDelegatedStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return err
	}
	if stake.GT(stakePlacement) {
		return types.ErrIntegerUnderflowDelegatedStakePlacement
	}
	stakePlacementNew := stakePlacement.Sub(stake)

	// Check stakeUponReputer >= stake
	stakeUponReputer, err := k.GetDelegatedStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if stake.GT(stakeUponReputer) {
		return types.ErrIntegerUnderflowDelegatedStakeUponReputer
	}
	stakeUponReputerNew := stakeUponReputer.Sub(stake)

	// types updates -- done all at once after all checks

	// Set new stake from delegator
	if err := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegatorNew); err != nil {
		return err
	}

	// Set new delegated stake placement
	if err := k.SetDelegatedStakePlacement(ctx, topicId, delegator, reputer, stakePlacementNew); err != nil {
		fmt.Println("Setting delegated stake placement failed -- rolling back stake from delegator")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		return err
	}

	// Set new delegated stake upon reputer
	if err := k.SetDelegatedStakeUponReputer(ctx, topicId, reputer, stakeUponReputerNew); err != nil {
		fmt.Println("Setting delegated stake upon reputer failed -- rolling back stake from delegator and delegated stake placement")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		err2 = k.SetDelegatedStakePlacement(ctx, topicId, delegator, reputer, stakePlacement)
		if err2 != nil {
			return err2
		}
		return err
	}

	return nil
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
func (k *Keeper) GetStakePlacementsForReputer(ctx context.Context, reputer sdk.AccAddress) ([]types.StakePlacement, error) {
	topicIds := make([]TOPIC_ID, 0)
	amounts := make([]cosmosMath.Uint, 0)
	stakes := make([]types.StakePlacement, 0)
	iter, err := k.stakeByReputerAndTopicId.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	// iterate over all keys in stakePlacements
	kvs, err := iter.Keys()
	if err != nil {
		return nil, err
	}
	for _, kv := range kvs {
		reputerKey := kv.K2()
		// if the reputer key matches the reputer we're looking for
		if reputerKey.Equals(reputer) {
			amount, err := k.stakeByReputerAndTopicId.Get(ctx, kv)
			if err != nil {
				return nil, err
			}
			stakeInfo := types.StakePlacement{
				TopicId: kv.K1(),
				Amount:  amount,
			}
			stakes = append(stakes, stakeInfo)
			topicIds = append(topicIds, kv.K1())
			amounts = append(amounts, amount)
		}
	}
	if len(topicIds) != len(amounts) {
		return nil, types.ErrIterationLengthDoesNotMatch
	}

	return stakes, nil
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

func (k *Keeper) GetStakeOnTopicFromReputer(ctx context.Context, topicId TOPIC_ID, reputer sdk.AccAddress) (Uint, error) {
	key := collections.Join(topicId, reputer)
	stake, err := k.stakeByReputerAndTopicId.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return stake, nil
}

// Sets the total sum of all stake in the network across all topics
// Should just be used at genesis
func (k *Keeper) SetTotalStake(ctx context.Context, totalStake Uint) error {
	// Total stake does not have a zero guard because totalStake is allowed to be zero
	// It is initialized to zero at genesis anyways
	return k.totalStake.Set(ctx, totalStake)
}

// Returns the amount of stake placed by a specific delegator.
func (k *Keeper) GetStakeFromDelegator(ctx context.Context, topicId TOPIC_ID, delegator DELEGATOR) (Uint, error) {
	key := collections.Join(topicId, delegator)
	stake, err := k.stakeFromDelegator.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator.
func (k *Keeper) SetStakeFromDelegator(ctx context.Context, topicId TOPIC_ID, delegator DELEGATOR, stake Uint) error {
	key := collections.Join(topicId, delegator)
	if stake.IsZero() {
		return k.stakeFromDelegator.Remove(ctx, key)
	}
	return k.stakeFromDelegator.Set(ctx, key, stake)
}

// Returns the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) GetDelegatedStakePlacement(ctx context.Context, topicId TOPIC_ID, delegator DELEGATOR, target REPUTER) (Uint, error) {
	key := collections.Join3(topicId, delegator, target)
	stake, err := k.delegatedStakePlacement.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) SetDelegatedStakePlacement(ctx context.Context, topicId TOPIC_ID, delegator DELEGATOR, target REPUTER, stake Uint) error {
	key := collections.Join3(topicId, delegator, target)
	if stake.IsZero() {
		return k.delegatedStakePlacement.Remove(ctx, key)
	}
	return k.delegatedStakePlacement.Set(ctx, key, stake)
}

// Returns the amount of stake placed on a specific target.
func (k *Keeper) GetDelegatedStakeUponReputer(ctx context.Context, topicId TOPIC_ID, target REPUTER) (Uint, error) {
	key := collections.Join(topicId, target)
	stake, err := k.stakeUponReputer.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed on a specific target.
func (k *Keeper) SetDelegatedStakeUponReputer(ctx context.Context, topicId TOPIC_ID, target REPUTER, stake Uint) error {
	key := collections.Join(topicId, target)
	if stake.IsZero() {
		return k.stakeUponReputer.Remove(ctx, key)
	}
	return k.stakeUponReputer.Set(ctx, key, stake)
}

///
/// TOPICS
///

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

// Gets next topic id
func (k *Keeper) IncrementTopicId(ctx context.Context) (TOPIC_ID, error) {
	return k.nextTopicId.Next(ctx)
}

// Gets topic by topicId
func (k *Keeper) GetTopic(ctx context.Context, topicId TOPIC_ID) (types.Topic, error) {
	return k.topics.Get(ctx, topicId)
}

// Sets a topic config on a topicId
func (k *Keeper) SetTopic(ctx context.Context, topicId TOPIC_ID, topic types.Topic) error {
	return k.topics.Set(ctx, topicId, topic)
}

// Gets every topic
func (k *Keeper) GetAllTopics(ctx context.Context) ([]*types.Topic, error) {
	var allTopics []*types.Topic
	err := k.topics.Walk(ctx, nil, func(topicId TOPIC_ID, topic types.Topic) (bool, error) {
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
func (k *Keeper) GetActiveTopics(ctx context.Context) ([]*types.Topic, error) {
	var activeTopics []*types.Topic
	if err := k.topics.Walk(ctx, nil, func(topicId TOPIC_ID, topic types.Topic) (bool, error) {
		if topic.Active { // Check if the topic is marked as active
			activeTopics = append(activeTopics, &topic)
		}
		return false, nil // Continue the iteration
	}); err != nil {
		return nil, err
	}
	return activeTopics, nil
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

func (k *Keeper) GetTopicLossCalcLastRan(ctx context.Context, topicId TOPIC_ID) (uint64, error) {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return 0, err
	}
	ret := topic.LossLastRan
	return ret, nil
}

// UpdateTopicInferenceLastRan updates the InferenceLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicInferenceLastRan(ctx context.Context, topicId TOPIC_ID, lastRanTime uint64) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	var newTopic types.Topic = types.Topic{
		Id:               topic.Id,
		Creator:          topic.Creator,
		Metadata:         topic.Metadata,
		LossLogic:        topic.LossLogic,
		LossMethod:       topic.LossMethod,
		LossCadence:      topic.LossCadence,
		LossLastRan:      topic.LossLastRan,
		InferenceLogic:   topic.InferenceLogic,
		InferenceMethod:  topic.InferenceMethod,
		InferenceCadence: topic.InferenceCadence,
		InferenceLastRan: lastRanTime,
		Active:           topic.Active,
		DefaultArg:       topic.DefaultArg,
	}
	return k.topics.Set(ctx, topicId, newTopic)
}

// UpdateTopicLossUpdateLastRan updates the WeightLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicLossUpdateLastRan(ctx context.Context, topicId TOPIC_ID, lastRanTime uint64) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.LossLastRan = lastRanTime
	return k.topics.Set(ctx, topicId, topic)
}

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *Keeper) InsertReputer(ctx context.Context, TopicIds []TOPIC_ID, reputer sdk.AccAddress, reputerInfo types.OffchainNode) error {
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
func (k *Keeper) InsertWorker(ctx context.Context, TopicIds []TOPIC_ID, worker sdk.AccAddress, workerInfo types.OffchainNode) error {
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

func (k *Keeper) FindWorkerNodesByOwner(ctx sdk.Context, nodeId string) ([]*types.OffchainNode, error) {
	var nodes []*types.OffchainNode
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

// For a given address, get their stake removal information
func (k *Keeper) GetStakeRemovalQueueByAddress(ctx context.Context, address sdk.AccAddress) (types.StakeRemoval, error) {
	return k.stakeRemovalQueue.Get(ctx, address)
}

// For a given address, adds their stake removal information to the removal queue for delay waiting
func (k *Keeper) SetStakeRemovalQueueForAddress(ctx context.Context, address sdk.AccAddress, removalInfo types.StakeRemoval) error {
	return k.stakeRemovalQueue.Set(ctx, address, removalInfo)
}

// For a given address, get their stake removal information
func (k *Keeper) GetDelegatedStakeRemovalQueueByAddress(ctx context.Context, address sdk.AccAddress) (types.DelegatedStakeRemoval, error) {
	return k.delegatedStakeRemovalQueue.Get(ctx, address)
}

// For a given address, adds their stake removal information to the removal queue for delay waiting
func (k *Keeper) SetDelegatedStakeRemovalQueueForAddress(ctx context.Context, address sdk.AccAddress, removalInfo types.DelegatedStakeRemoval) error {
	return k.delegatedStakeRemovalQueue.Set(ctx, address, removalInfo)
}

///
/// MEMPOOL & INFERENCE REQUESTS
///

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
		return types.ErrIntegerUnderflowUnmetDemand
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

func (k *Keeper) AddToMempool(ctx context.Context, request types.InferenceRequest) error {
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

func (k *Keeper) RemoveFromMempool(ctx context.Context, request types.InferenceRequest) error {
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

func (k *Keeper) GetMempoolInferenceRequestById(ctx context.Context, topicId TOPIC_ID, requestId string) (types.InferenceRequest, error) {
	return k.mempool.Get(ctx, collections.Join(topicId, requestId))
}

func (k *Keeper) GetMempoolInferenceRequestsForTopic(ctx context.Context, topicId TOPIC_ID) ([]types.InferenceRequest, error) {
	var ret []types.InferenceRequest = make([]types.InferenceRequest, 0)
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

func (k *Keeper) GetParamsMaxRequestCadence(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxRequestCadence, nil
}

func (k *Keeper) GetMempool(ctx context.Context) ([]types.InferenceRequest, error) {
	var ret []types.InferenceRequest = make([]types.InferenceRequest, 0)
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
func (k *Keeper) SetChurnReadyTopics(ctx context.Context, topicList types.TopicList) error {
	return k.churnReadyTopics.Set(ctx, topicList)
}

// Get all churn ready topics
func (k *Keeper) GetChurnReadyTopics(ctx context.Context) (types.TopicList, error) {
	topicList, err := k.churnReadyTopics.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TopicList{}, nil
		}
		return types.TopicList{}, err
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

///
/// WHITELISTS
///

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

func (k *Keeper) IsInReputerWhitelist(ctx context.Context, address sdk.AccAddress) (bool, error) {
	return k.reputerWhitelist.Has(ctx, address)
}

func (k *Keeper) AddToReputerWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.reputerWhitelist.Set(ctx, address)
}

func (k *Keeper) RemoveFromReputerWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.reputerWhitelist.Remove(ctx, address)
}

func (k *Keeper) IsInFoundationWhitelist(ctx context.Context, address sdk.AccAddress) (bool, error) {
	return k.foundationWhitelist.Has(ctx, address)
}

func (k *Keeper) AddToFoundationWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.foundationWhitelist.Set(ctx, address)
}

func (k *Keeper) RemoveFromFoundationWhitelist(ctx context.Context, address sdk.AccAddress) error {
	return k.foundationWhitelist.Remove(ctx, address)
}

///
/// TREASURY
///

// Sets the subsidy for the topic within the topic struct
// Should only be called by a member of the foundation whitelist
func (k *Keeper) SetTopicSubsidy(ctx context.Context, topicId TOPIC_ID, subsidy uint64) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.Subsidy = subsidy
	return k.topics.Set(ctx, topicId, topic)
}

// Sets the number of reward for the topic within the topic struct
// Should only be called by a member of the foundation whitelist
func (k *Keeper) SetTopicSubsidizedRewardEpochs(ctx context.Context, topicId TOPIC_ID, subsidizedRewardEpochs float32) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.SubsidizedRewardEpochs = subsidizedRewardEpochs
	return k.topics.Set(ctx, topicId, topic)
}

// Sets the number of reward for the topic within the topic struct
// Should only be called by a member of the foundation whitelist
func (k *Keeper) SetTopicFTreasury(ctx context.Context, topicId TOPIC_ID, fTreasury float32) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.FTreasury = fTreasury
	return k.topics.Set(ctx, topicId, topic)
}

///
/// BANK KEEPER WRAPPERS
///

// SendCoinsFromModuleToModule
func (k *Keeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt)
}

// SendCoinsFromModuleToAccount
func (k *Keeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// SendCoinsFromAccountToModule
func (k *Keeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}
