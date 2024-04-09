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

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Uint = cosmosMath.Uint
type Int = cosmosMath.Int

type TopicId = uint64
type LibP2pKey = string
type Delegator = sdk.AccAddress
type Worker = sdk.AccAddress
type Reputer = sdk.AccAddress
type AccAddress = string
type Workers = string
type Reputers = string
type BlockHeight = int64
type RequestId = string

type Keeper struct {
	cdc              codec.BinaryCodec
	addressCodec     address.Codec
	feeCollectorName string

	/// TYPES

	schema     collections.Schema
	params     collections.Item[types.Params]
	authKeeper AccountKeeper
	bankKeeper BankKeeper

	/// TOPIC

	// the next topic id to be used, equal to the number of topics that have been created
	nextTopicId collections.Sequence
	// every topic that has been created indexed by their topicId starting from 1 (0 is reserved for the root network)
	topics collections.Map[TopicId, types.Topic]
	// every topics that has been churned and ready to get inferences in the block
	churnReadyTopics collections.Item[types.TopicList]
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TopicId, sdk.AccAddress]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TopicId, sdk.AccAddress]]
	// for an address, what are all the topics that it's registered for?
	addressTopics collections.Map[sdk.AccAddress, []uint64]

	/// SCORES

	// map of (topic, block_number, worker) -> score
	inferenceScores collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_number, worker) -> score
	forecastScores collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_number, reputer) -> score
	reputerScores collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, reputer) -> listening coefficient
	reputerListeningCoefficient collections.Map[collections.Pair[TopicId, Reputer], types.ListeningCoefficient]

	/// TAX for REWARD
	// map of (topic, block_number, worker) -> avg_worker_reward
	averageWorkerReward collections.Map[collections.Pair[TopicId, sdk.AccAddress], types.AverageWorkerReward]

	/// STAKING

	// total sum stake of all stakers on the network
	totalStake collections.Item[Uint]
	// for every topic, how much total stake does that topic have accumulated?
	topicStake collections.Map[TopicId, Uint]
	// amount of stake a reputer has placed in a topic, signalling their authority on the topic
	stakeByReputerAndTopicId collections.Map[collections.Pair[TopicId, Reputer], Uint]
	// map of (reputer) -> removal information for that reputer
	stakeRemovalQueue collections.Map[Reputer, types.StakeRemoval]
	// map of (delegator) -> removal information for that delegator
	delegatedStakeRemovalQueue collections.Map[Delegator, types.DelegatedStakeRemoval]
	// map of (delegator) -> amount of stake that has been placed by that delegator
	stakeFromDelegator collections.Map[collections.Pair[TopicId, Delegator], Uint]
	// map of (delegator, target) -> amount of stake that has been placed by that delegator on that target
	delegatedStakePlacement collections.Map[collections.Triple[TopicId, Reputer, Delegator], Uint]
	// map of (target) -> amount of stake that has been placed on that target
	stakeUponReputer collections.Map[collections.Pair[TopicId, Reputer], Uint]

	/// INFERENCE REQUEST MEMPOOL

	// map of (topic, request_id) -> full InferenceRequest information for that request
	mempool collections.Map[collections.Pair[TopicId, RequestId], types.InferenceRequest]
	// amount of money available for an inference request id that has been placed in the mempool but has not yet been fully satisfied
	requestUnmetDemand collections.Map[RequestId, Uint]
	// total amount of demand for a topic that has been placed in the mempool as a request for inference but has not yet been satisfied
	topicUnmetDemand collections.Map[TopicId, Uint]

	/// MISC GLOBAL STATE

	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TopicId, Worker], types.Inference]

	// map of (topic, worker) -> forecast[]
	forecasts collections.Map[collections.Pair[TopicId, Worker], types.Forecast]

	// map of (topic, worker) -> num_inferences_in_reward_epoch
	numInferencesInRewardEpoch collections.Map[collections.Pair[TopicId, Worker], Uint]

	// map of worker id to node data about that worker
	workers collections.Map[LibP2pKey, types.OffchainNode]

	// map of reputer id to node data about that reputer
	reputers collections.Map[LibP2pKey, types.OffchainNode]

	// the last block the token inflation rewards were updated: int64 same as BlockHeight()
	lastRewardsUpdate collections.Item[BlockHeight]

	// fee revenue collected by a topic over the course of the last reward cadence
	topicFeeRevenue collections.Map[TopicId, types.TopicFeeRevenue]

	// feeRevenueEpoch marks the current epoch for fee revenue
	feeRevenueEpoch collections.Sequence

	// store previous wieghts for exponential moving average in rewards calc
	previousTopicWeight collections.Map[TopicId, types.PreviousTopicWeight]

	// map of (topic, block_height) -> Inference
	allInferences collections.Map[collections.Pair[TopicId, BlockHeight], types.Inferences]

	// map of (topic, block_height) -> Forecast
	allForecasts collections.Map[collections.Pair[TopicId, BlockHeight], types.Forecasts]

	// map of (topic, block_height) -> ReputerValueBundles (1 per reputer active at that time)
	allLossBundles collections.Map[collections.Pair[TopicId, BlockHeight], types.ReputerValueBundles]

	// map of (topic, block_height) -> ValueBundle (1 network wide bundle per timestep)
	networkLossBundles collections.Map[collections.Pair[TopicId, BlockHeight], types.ValueBundle]

	accumulatedMetDemand collections.Map[TopicId, Uint]

	/// NONCES

	// map of (topic) -> unfulfilled nonces
	unfulfilledWorkerNonces collections.Map[TopicId, types.Nonces]

	// map of (topic) -> unfulfilled nonces
	unfulfilledReputerNonces collections.Map[TopicId, types.Nonces]

	/// REGRETS

	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other inferers
	latestInfererNetworkRegrets collections.Map[collections.Pair[TopicId, Worker], types.TimestampedValue]
	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other forecasters
	latestForecasterNetworkRegrets collections.Map[collections.Pair[TopicId, Worker], types.TimestampedValue]
	// map of (topic, forecaster, inferer) -> R^+_{ij_kk} regret of forecaster loss from comparing one-in loss with
	// all network inferer losses L_ij including the network forecast-implied inference L_ik^* of the forecaster
	latestOneInForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, Worker, Worker], types.TimestampedValue]

	/// WHITELISTS

	whitelistAdmins collections.KeySet[sdk.AccAddress]

	topicCreationWhitelist collections.KeySet[sdk.AccAddress]

	reputerWhitelist collections.KeySet[sdk.AccAddress]
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
		cdc:                                 cdc,
		addressCodec:                        addressCodec,
		feeCollectorName:                    feeCollectorName,
		params:                              collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		authKeeper:                          ak,
		bankKeeper:                          bk,
		totalStake:                          collections.NewItem(sb, types.TotalStakeKey, "total_stake", UintValue),
		topicStake:                          collections.NewMap(sb, types.TopicStakeKey, "topic_stake", collections.Uint64Key, UintValue),
		lastRewardsUpdate:                   collections.NewItem(sb, types.LastRewardsUpdateKey, "last_rewards_update", collections.Int64Value),
		nextTopicId:                         collections.NewSequence(sb, types.NextTopicIdKey, "next_TopicId"),
		topics:                              collections.NewMap(sb, types.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[types.Topic](cdc)),
		churnReadyTopics:                    collections.NewItem(sb, types.ChurnReadyTopicsKey, "churn_ready_topics", codec.CollValue[types.TopicList](cdc)),
		topicWorkers:                        collections.NewKeySet(sb, types.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		addressTopics:                       collections.NewMap(sb, types.AddressTopicsKey, "address_topics", sdk.AccAddressKey, TopicIdListValue),
		topicReputers:                       collections.NewKeySet(sb, types.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		stakeByReputerAndTopicId:            collections.NewMap(sb, types.StakeByReputerAndTopicIdKey, "stake_by_reputer_and_TopicId", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		stakeRemovalQueue:                   collections.NewMap(sb, types.StakeRemovalQueueKey, "stake_removal_queue", sdk.AccAddressKey, codec.CollValue[types.StakeRemoval](cdc)),
		delegatedStakeRemovalQueue:          collections.NewMap(sb, types.DelegatedStakeRemovalQueueKey, "delegated_stake_removal_queue", sdk.AccAddressKey, codec.CollValue[types.DelegatedStakeRemoval](cdc)),
		stakeFromDelegator:                  collections.NewMap(sb, types.DelegatorStakeKey, "stake_from_delegator", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		delegatedStakePlacement:             collections.NewMap(sb, types.DelegatedStakePlacementKey, "delegated_stake_placement", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), UintValue),
		stakeUponReputer:                    collections.NewMap(sb, types.TargetStakeKey, "stake_upon_reputer", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		mempool:                             collections.NewMap(sb, types.MempoolKey, "mempool", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.InferenceRequest](cdc)),
		requestUnmetDemand:                  collections.NewMap(sb, types.RequestUnmetDemandKey, "request_unmet_demand", collections.StringKey, UintValue),
		topicUnmetDemand:                    collections.NewMap(sb, types.TopicUnmetDemandKey, "topic_unmet_demand", collections.Uint64Key, UintValue),
		topicFeeRevenue:                     collections.NewMap(sb, types.TopicFeeRevenueKey, "topic_fee_revenue", collections.Uint64Key, codec.CollValue[types.TopicFeeRevenue](cdc)),
		feeRevenueEpoch:                     collections.NewSequence(sb, types.FeeRevenueEpochKey, "fee_revenue_epoch"),
		previousTopicWeight:                 collections.NewMap(sb, types.PreviousTopicWeightKey, "previous_topic_weight", collections.Uint64Key, codec.CollValue[types.PreviousTopicWeight](cdc)),
		inferences:                          collections.NewMap(sb, types.InferencesKey, "inferences", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Inference](cdc)),
		forecasts:                           collections.NewMap(sb, types.ForecastsKey, "forecasts", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Forecast](cdc)),
		workers:                             collections.NewMap(sb, types.WorkerNodesKey, "worker_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		reputers:                            collections.NewMap(sb, types.ReputerNodesKey, "reputer_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		allInferences:                       collections.NewMap(sb, types.AllInferencesKey, "inferences_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Inferences](cdc)),
		allForecasts:                        collections.NewMap(sb, types.AllForecastsKey, "forecasts_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Forecasts](cdc)),
		allLossBundles:                      collections.NewMap(sb, types.AllLossBundlesKey, "value_bundles_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ReputerValueBundles](cdc)),
		networkLossBundles:                  collections.NewMap(sb, types.NetworkLossBundlesKey, "value_bundles_network", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ValueBundle](cdc)),
		latestInfererNetworkRegrets:         collections.NewMap(sb, types.InfererNetworkRegretsKey, "inferer_network_regrets", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestForecasterNetworkRegrets:      collections.NewMap(sb, types.ForecasterNetworkRegretsKey, "forecaster_network_regrets", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneInForecasterNetworkRegrets: collections.NewMap(sb, types.OneInForecasterNetworkRegretsKey, "one_in_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), codec.CollValue[types.TimestampedValue](cdc)),
		accumulatedMetDemand:                collections.NewMap(sb, types.AccumulatedMetDemandKey, "accumulated_met_demand", collections.Uint64Key, UintValue),
		numInferencesInRewardEpoch:          collections.NewMap(sb, types.NumInferencesInRewardEpochKey, "num_inferences_in_reward_epoch", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), UintValue),
		whitelistAdmins:                     collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", sdk.AccAddressKey),
		topicCreationWhitelist:              collections.NewKeySet(sb, types.TopicCreationWhitelistKey, "topic_creation_whitelist", sdk.AccAddressKey),
		reputerWhitelist:                    collections.NewKeySet(sb, types.ReputerWhitelistKey, "weight_setting_whitelist", sdk.AccAddressKey),
		inferenceScores:                     collections.NewMap(sb, types.InferenceScoresKey, "worker_inference_scores", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		forecastScores:                      collections.NewMap(sb, types.ForecastScoresKey, "worker_forecast_scores", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerScores:                       collections.NewMap(sb, types.ReputerScoresKey, "reputer_scores", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerListeningCoefficient:         collections.NewMap(sb, types.ReputerListeningCoefficientKey, "reputer_listening_coefficient", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.ListeningCoefficient](cdc)),
		unfulfilledWorkerNonces:             collections.NewMap(sb, types.UnfulfilledWorkerNoncesKey, "unfulfilled_worker_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
		unfulfilledReputerNonces:            collections.NewMap(sb, types.UnfulfilledReputerNoncesKey, "unfulfilled_reputer_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.schema = schema

	return k
}

/// NONCES

// Attempts to fulfill an unfulfilled nonce.
// If the nonce is present, then it is removed from the unfulfilled nonces and this function returns true.
// If the nonce is not present, then the function returns false.
func (k *Keeper) FulfillWorkerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, err
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n.Nonce == nonce.Nonce {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			err := k.unfulfilledWorkerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	// If the nonce is not present in the unfulfilled nonces
	return false, nil
}

// Attempts to fulfill an unfulfilled nonce.
// If the nonce is present, then it is removed from the unfulfilled nonces and this function returns true.
// If the nonce is not present, then the function returns false.
func (k *Keeper) FulfillReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return err
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n.Nonce == nonce.Nonce {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			err := k.unfulfilledReputerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return err
			}
			return nil
		}
	}

	// If the nonce is not present in the unfulfilled nonces
	return nil
}

// True if nonce is unfulfilled, false otherwise.
func (k *Keeper) IsWorkerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, err
	}

	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n.Nonce == nonce.Nonce {
			return true, nil
		}
	}

	return false, nil
}

// True if nonce is unfulfilled, false otherwise.
func (k *Keeper) IsReputerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return false, err
	}

	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n.Nonce == nonce.Nonce {
			return true, nil
		}
	}

	return false, nil
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *Keeper) AddWorkerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return err
	}

	// Check that input nonce is not already contained in the nonces of this topic
	for _, n := range nonces.Nonces {
		if n.Nonce == nonce.Nonce {
			return nil
		}
	}
	nonces.Nonces = append(nonces.Nonces, nonce)

	maxUnfulfilledRequests, err := k.GetParamsMaxUnfulfilledReputerRequests(ctx)
	if err != nil {
		return err
	}

	if uint64(len(nonces.Nonces)) < maxUnfulfilledRequests {
		nonces.Nonces = nonces.Nonces[1:]
	}

	return k.unfulfilledReputerNonces.Set(ctx, topicId, nonces)
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *Keeper) AddReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return err
	}

	// Check that input nonce is not already contained in the nonces of this topic
	for _, n := range nonces.Nonces {
		if n.Nonce == nonce.Nonce {
			return nil
		}
	}
	nonces.Nonces = append(nonces.Nonces, nonce)

	maxUnfulfilledRequests, err := k.GetParamsMaxUnfulfilledWorkerRequests(ctx)
	if err != nil {
		return err
	}

	if uint64(len(nonces.Nonces)) < maxUnfulfilledRequests {
		nonces.Nonces = nonces.Nonces[1:]
	}

	return k.unfulfilledReputerNonces.Set(ctx, topicId, nonces)
}

func (k *Keeper) GetUnfulfilledWorkerNonces(ctx context.Context, topicId TopicId) (types.Nonces, error) {
	nonces, err := k.unfulfilledWorkerNonces.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Nonces{}, nil
		}
		return types.Nonces{}, err
	}
	return nonces, nil
}

func (k *Keeper) GetUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) (types.Nonces, error) {
	nonces, err := k.unfulfilledReputerNonces.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Nonces{}, nil
		}
		return types.Nonces{}, err
	}
	return nonces, nil
}

/// REGRETS

func (k *Keeper) SetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker Worker, regret types.TimestampedValue) error {
	key := collections.Join(topicId, worker)
	return k.latestInfererNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) SetForecasterNetworkRegret(ctx context.Context, topicId TopicId, worker Worker, regret types.TimestampedValue) error {
	key := collections.Join(topicId, worker)
	return k.latestInfererNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) SetOneInForecasterNetworkRegret(ctx context.Context, topicId TopicId, forecaster Worker, inferer Worker, regret types.TimestampedValue) error {
	key := collections.Join3(topicId, forecaster, inferer)
	return k.latestOneInForecasterNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) GetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker Worker) (types.TimestampedValue, error) {
	key := collections.Join(topicId, worker)
	regret, err := k.latestInfererNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       1,
			}, nil
		}
		return types.TimestampedValue{}, err
	}
	return regret, nil
}

func (k *Keeper) GetForecasterNetworkRegret(ctx context.Context, topicId TopicId, worker Worker) (types.TimestampedValue, error) {
	key := collections.Join(topicId, worker)
	regret, err := k.latestForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       1,
			}, nil
		}
		return types.TimestampedValue{}, err
	}
	return regret, nil
}

func (k *Keeper) GetOneInForecasterNetworkRegret(ctx context.Context, topicId TopicId, forecaster Worker, inferer Worker) (types.TimestampedValue, error) {
	key := collections.Join3(topicId, forecaster, inferer)
	regret, err := k.latestOneInForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       1,
			}, nil
		}
		return types.TimestampedValue{}, err
	}
	return regret, nil
}

/// PARAMETERS

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

func (k *Keeper) GetFeeCollectorName() string {
	return k.feeCollectorName
}

func (k *Keeper) GetParamsMaxMissingInferencePercent(ctx context.Context) (float64, error) {
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

func (k *Keeper) GetParamsRemoveStakeDelayWindow(ctx context.Context) (BlockHeight, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.RemoveStakeDelayWindow, nil
}

func (k *Keeper) GetParamsMaxInferenceRequestValidity(ctx context.Context) (BlockHeight, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxInferenceRequestValidity, nil
}

func (k *Keeper) GetParamsMinEpochLength(ctx context.Context) (BlockHeight, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MinEpochLength, nil
}

func (k *Keeper) GetParamsMaxRequestCadence(ctx context.Context) (BlockHeight, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxRequestCadence, nil
}

func (k *Keeper) GetParamsPercentRewardsReputersWorkers(ctx context.Context) (float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.PercentRewardsReputersWorkers, nil
}

func (k *Keeper) GetParamsEpsilon(ctx context.Context) (float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.Epsilon, nil
}

func (k *Keeper) GetParamsPInferenceSynthesis(ctx context.Context) (float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.PInferenceSynthesis, nil
}

func (k *Keeper) GetParamsStakeAndFeeRevenueImportance(ctx context.Context) (float64, float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, 0, err
	}
	return params.TopicRewardStakeImportance, params.TopicRewardFeeRevenueImportance, nil
}

func (k *Keeper) GetParamsMaxUnfulfilledWorkerRequests(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxUnfulfilledWorkerRequests, nil
}

func (k *Keeper) GetParamsTopicRewardAlpha(ctx context.Context) (float64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.TopicRewardAlpha, nil
}

func (k *Keeper) GetParamsMaxUnfulfilledReputerRequests(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxUnfulfilledReputerRequests, nil
}

func (k Keeper) GetParamsValidatorsVsAlloraPercentReward(ctx context.Context) (cosmosMath.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.LegacyDec{}, err
	}
	return params.ValidatorsVsAlloraPercentReward, nil
}

/// INFERENCES, FORECASTS

func (k *Keeper) GetInferencesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Inferences, error) {
	key := collections.Join(topicId, block)
	inferences, err := k.allInferences.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &inferences, nil
}

func (k *Keeper) GetForecastsAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Forecasts, error) {
	key := collections.Join(topicId, block)
	forecasts, err := k.allForecasts.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &forecasts, nil
}

func (k *Keeper) GetInferencesAtOrAfterBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Inferences, BlockHeight, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(block).
		Descending()

	inferencesToReturn := types.Inferences{}
	blockHeight := int64(0)
	iter, err := k.allInferences.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return nil, 0, err
		}
		inferencesToReturn = kv.Value
		blockHeight = kv.Key.K2()
	}

	return &inferencesToReturn, blockHeight, nil
}

func (k *Keeper) GetForecastsAtOrAfterBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Forecasts, BlockHeight, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(block).
		Descending()

	forecastsToReturn := types.Forecasts{}
	blockHeight := int64(0)
	iter, err := k.allForecasts.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return nil, 0, err
		}
		forecastsToReturn = kv.Value
		blockHeight = kv.Key.K2()
	}

	return &forecastsToReturn, blockHeight, nil
}

// Insert a complete set of inferences for a topic/block. Overwrites previous ones.
func (k *Keeper) InsertInferences(ctx context.Context, topicId TopicId, nonce types.Nonce, inferences types.Inferences) error {
	block := nonce.Nonce

	for _, inference := range inferences.Inferences {
		inferenceCopy := *inference
		// Update latests inferences for each worker
		workerAcc, err := sdk.AccAddressFromBech32(inferenceCopy.Worker)
		if err != nil {
			return err
		}
		key := collections.Join(topicId, workerAcc)
		err = k.inferences.Set(ctx, key, inferenceCopy)
		if err != nil {
			return err
		}
		// Update the number of inferences in the reward epoch for each worker
		err = k.IncrementNumInferencesInRewardEpoch(ctx, topicId, workerAcc)
		if err != nil {
			return err
		}
	}

	key := collections.Join(topicId, block)
	return k.allInferences.Set(ctx, key, inferences)
}

// Insert a complete set of inferences for a topic/block. Overwrites previous ones.
func (k *Keeper) InsertForecasts(ctx context.Context, topicId TopicId, nonce types.Nonce, forecasts types.Forecasts) error {
	block := nonce.Nonce

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

	key := collections.Join(topicId, block)
	return k.allForecasts.Set(ctx, key, forecasts)
}

func (k *Keeper) GetWorkerLatestInferenceByTopicId(
	ctx context.Context,
	topicId TopicId,
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

// A function that accepts a topicId and returns list of Inferences or error
func (k *Keeper) GetLatestInferencesFromTopic(ctx context.Context, topicId TopicId) ([]*types.InferenceSetForScoring, error) {
	var inferences []*types.InferenceSetForScoring
	var latestBlock, err = k.GetTopicEpochLastEnded(ctx, topicId)
	if err != nil {
		latestBlock = 0
	}
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		StartInclusive(latestBlock).
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
			TopicId:     key.K1(),
			BlockHeight: key.K2(),
			Inferences:  &value,
		}
		inferences = append(inferences, inferenceSet)
	}
	return inferences, nil
}

// A function that accepts a topicId and returns list of Forecasts or error
func (k *Keeper) GetLatestForecastsFromTopic(ctx context.Context, topicId TopicId) ([]*types.ForecastSetForScoring, error) {
	var forecasts []*types.ForecastSetForScoring
	var latestBlock, err = k.GetTopicEpochLastEnded(ctx, topicId)
	if err != nil {
		latestBlock = 0
	}
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		StartInclusive(latestBlock).
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
			TopicId:     key.K1(),
			BlockHeight: key.K2(),
			Forecasts:   &value,
		}
		forecasts = append(forecasts, forecastSet)
	}
	return forecasts, nil
}

/// LOSS BUNDLES

// Insert a loss bundle for a topic and timestamp. Overwrites previous ones stored at that composite index.
func (k *Keeper) InsertReputerLossBundlesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight, reptuerLossBundles types.ReputerValueBundles) error {
	key := collections.Join(topicId, block)
	return k.allLossBundles.Set(ctx, key, reptuerLossBundles)
}

// Get loss bundles for a topic/timestamp
func (k *Keeper) GetReputerLossBundlesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ReputerValueBundles, error) {
	key := collections.Join(topicId, block)
	reputerLossBundles, err := k.allLossBundles.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &reputerLossBundles, nil
}

// Insert a network loss bundle for a topic and block.
func (k *Keeper) InsertNetworkLossBundleAtBlock(ctx context.Context, topicId TopicId, block BlockHeight, lossBundle types.ValueBundle) error {
	key := collections.Join(topicId, block)
	return k.networkLossBundles.Set(ctx, key, lossBundle)
}

// A function that accepts a topicId and returns the network LossBundle at the block or error
func (k *Keeper) GetNetworkLossBundleAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ValueBundle, error) {
	key := collections.Join(topicId, block)
	lossBundle, err := k.networkLossBundles.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &lossBundle, nil
}

func (k *Keeper) GetNetworkLossBundleAtOrBeforeBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ValueBundle, BlockHeight, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		StartInclusive(block).
		Descending()

	iter, err := k.networkLossBundles.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	if !iter.Valid() {
		// Return empty loss bundle if no loss bundle is found
		return &types.ValueBundle{}, 0, nil
	}
	kv, err := iter.KeyValue()
	if err != nil {
		return nil, 0, err
	}
	return &kv.Value, kv.Key.K2(), nil
}

func (k *Keeper) GetReputerReportedLossesAtOrBeforeBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ReputerValueBundles, BlockHeight, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		StartInclusive(block).
		Descending()

	iter, err := k.allLossBundles.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	if !iter.Valid() {
		// Return empty loss bundle if no loss bundle is found
		return &types.ReputerValueBundles{}, 0, nil
	}
	kv, err := iter.KeyValue()
	if err != nil {
		return nil, 0, err
	}
	return &kv.Value, kv.Key.K2(), nil
}

/// STAKING

// Adds stake to the system for a given topic and reputer
func (k *Keeper) AddStake(ctx context.Context, topicId TopicId, reputer sdk.AccAddress, stake Uint) error {
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

func (k *Keeper) AddDelegatedStake(ctx context.Context, topicId TopicId, delegator sdk.AccAddress, reputer sdk.AccAddress, stake Uint) error {
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
	topicId TopicId,
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
	topicId TopicId,
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

// Gets the total sum of all stake in the network across all topics
func (k Keeper) GetTotalStake(ctx context.Context) (Uint, error) {
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
func (k *Keeper) WalkAllTopicStake(ctx context.Context, walkFunc func(topicId TopicId, stake Uint) (stop bool, err error)) error {
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
func (k *Keeper) GetStakePlacementsByReputer(ctx context.Context, reputer sdk.AccAddress) ([]types.StakePlacement, error) {
	topicIds := make([]TopicId, 0)
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
				Reputer: reputerKey.String(),
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

func (k *Keeper) GetStakePlacementsByTopic(ctx context.Context, topicId TopicId) ([]types.StakePlacement, error) {
	reputers := make([]Reputer, 0)
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
		topicKey := kv.K1()
		// if the topic key matches the topic we're looking for
		if topicKey == topicId {
			amount, err := k.stakeByReputerAndTopicId.Get(ctx, kv)
			if err != nil {
				return nil, err
			}
			stakeInfo := types.StakePlacement{
				TopicId: topicKey,
				Amount:  amount,
				Reputer: kv.K2().String(),
			}
			stakes = append(stakes, stakeInfo)
			reputers = append(reputers, kv.K2())
			amounts = append(amounts, amount)
		}
	}
	if len(reputers) != len(amounts) {
		return nil, types.ErrIterationLengthDoesNotMatch
	}

	return stakes, nil
}

// Gets the stake in the network for a given topic
func (k *Keeper) GetTopicStake(ctx context.Context, topicId TopicId) (Uint, error) {
	ret, err := k.topicStake.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return ret, nil
}

func (k *Keeper) GetStakeOnTopicFromReputer(ctx context.Context, topicId TopicId, reputer sdk.AccAddress) (Uint, error) {
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

// Returns the amount of stake placed by a specific delegator.
func (k *Keeper) GetStakeFromDelegator(ctx context.Context, topicId TopicId, delegator Delegator) (Uint, error) {
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
func (k *Keeper) SetStakeFromDelegator(ctx context.Context, topicId TopicId, delegator Delegator, stake Uint) error {
	key := collections.Join(topicId, delegator)
	if stake.IsZero() {
		return k.stakeFromDelegator.Remove(ctx, key)
	}
	return k.stakeFromDelegator.Set(ctx, key, stake)
}

// Returns the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) GetDelegatedStakePlacement(ctx context.Context, topicId TopicId, delegator Delegator, target Reputer) (Uint, error) {
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
func (k *Keeper) SetDelegatedStakePlacement(ctx context.Context, topicId TopicId, delegator Delegator, target Reputer, stake Uint) error {
	key := collections.Join3(topicId, delegator, target)
	if stake.IsZero() {
		return k.delegatedStakePlacement.Remove(ctx, key)
	}
	return k.delegatedStakePlacement.Set(ctx, key, stake)
}

// Returns the amount of stake placed on a specific target.
func (k *Keeper) GetDelegatedStakeUponReputer(ctx context.Context, topicId TopicId, target Reputer) (Uint, error) {
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
func (k *Keeper) SetDelegatedStakeUponReputer(ctx context.Context, topicId TopicId, target Reputer, stake Uint) error {
	key := collections.Join(topicId, target)
	if stake.IsZero() {
		return k.stakeUponReputer.Remove(ctx, key)
	}
	return k.stakeUponReputer.Set(ctx, key, stake)
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

/// TOPICS

// Get the previous weight during rewards calculation for a topic
func (k *Keeper) GetPreviousTopicWeight(ctx context.Context, topicId TopicId) (types.PreviousTopicWeight, error) {
	topicWeight, err := k.previousTopicWeight.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		ret := types.PreviousTopicWeight{
			Weight: 0.0,
			Epoch:  0,
		}
		return ret, nil
	}
	return topicWeight, err
}

// Set the previous weight during rewards calculation for a topic
func (k *Keeper) SetPreviousTopicWeight(ctx context.Context, topicId TopicId, weight types.PreviousTopicWeight) error {
	return k.previousTopicWeight.Set(ctx, topicId, weight)
}

func (k *Keeper) InactivateTopic(ctx context.Context, topicId TopicId) error {
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

func (k *Keeper) ReactivateTopic(ctx context.Context, topicId TopicId) error {
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
func (k *Keeper) IncrementTopicId(ctx context.Context) (TopicId, error) {
	return k.nextTopicId.Next(ctx)
}

// Gets topic by topicId
func (k *Keeper) GetTopic(ctx context.Context, topicId TopicId) (types.Topic, error) {
	return k.topics.Get(ctx, topicId)
}

// Sets a topic config on a topicId
func (k *Keeper) SetTopic(ctx context.Context, topicId TopicId, topic types.Topic) error {
	return k.topics.Set(ctx, topicId, topic)
}

// Gets every topic
func (k *Keeper) GetAllTopics(ctx context.Context) ([]*types.Topic, error) {
	var allTopics []*types.Topic
	err := k.topics.Walk(ctx, nil, func(topicId TopicId, topic types.Topic) (bool, error) {
		allTopics = append(allTopics, &topic)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return allTopics, nil
}

// Checks if a topic exists
func (k *Keeper) TopicExists(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topics.Has(ctx, topicId)
}

// Returns the number of topics that are active in the network
func (k *Keeper) GetNumTopics(ctx context.Context) (TopicId, error) {
	return k.nextTopicId.Peek(ctx)
}

// GetActiveTopics returns a slice of all active topics.
func (k *Keeper) GetActiveTopics(ctx context.Context) ([]*types.Topic, error) {
	var activeTopics []*types.Topic
	if err := k.topics.Walk(ctx, nil, func(topicId TopicId, topic types.Topic) (bool, error) {
		if topic.Active { // Check if the topic is marked as active
			activeTopics = append(activeTopics, &topic)
		}
		return false, nil // Continue the iteration
	}); err != nil {
		return nil, err
	}
	return activeTopics, nil
}

func (k *Keeper) GetTopicEpochLastEnded(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return 0, err
	}
	ret := topic.EpochLastEnded
	return ret, nil
}

// UpdateTopicInferenceLastRan updates the InferenceLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicEpochLastEnded(ctx context.Context, topicId TopicId, epochLastEnded BlockHeight) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return err
	}
	topic.EpochLastEnded = epochLastEnded
	return k.topics.Set(ctx, topicId, topic)
}

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *Keeper) InsertReputer(ctx context.Context, TopicIds []TopicId, reputer sdk.AccAddress, reputerInfo types.OffchainNode) error {
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
func (k *Keeper) RemoveReputer(ctx context.Context, topicId TopicId, reputerAddr sdk.AccAddress) error {

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
func (k *Keeper) RemoveWorker(ctx context.Context, topicId TopicId, workerAddr sdk.AccAddress) error {

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
func (k *Keeper) InsertWorker(ctx context.Context, TopicIds []TopicId, worker sdk.AccAddress, workerInfo types.OffchainNode) error {
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
			nodeCopy := node
			nodes = append(nodes, &nodeCopy)
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

// GetTopicsByCreator returns a slice of all topics created by a given creator.
func (k *Keeper) GetTopicsByCreator(ctx context.Context, creator string) ([]*types.Topic, error) {
	var topicsByCreator []*types.Topic

	err := k.topics.Walk(ctx, nil, func(id TopicId, topic types.Topic) (bool, error) {
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

	err := k.topicWorkers.Walk(ctx, nil, func(pair collections.Pair[TopicId, sdk.AccAddress]) (bool, error) {
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

	err := k.topicReputers.Walk(ctx, nil, func(pair collections.Pair[TopicId, sdk.AccAddress]) (bool, error) {
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

// Get the amount of fee revenue collected by a topic
func (k *Keeper) GetTopicFeeRevenue(ctx context.Context, topicId TopicId) (types.TopicFeeRevenue, error) {
	return k.topicFeeRevenue.Get(ctx, topicId)
}

// Add to the fee revenue collected by a topic for this reward epoch
func (k *Keeper) AddTopicFeeRevenue(ctx context.Context, topicId TopicId, amount Uint) error {
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return err
	}
	currEpoch, err := k.GetFeeRevenueEpoch(ctx)
	if err != nil {
		return err
	}
	newTopicFeeRevenue := types.TopicFeeRevenue{}
	if topicFeeRevenue.Epoch != currEpoch {
		newTopicFeeRevenue.Epoch = currEpoch
		newTopicFeeRevenue.Revenue = cosmosMath.NewIntFromBigInt(amount.BigInt())
	} else {
		newTopicFeeRevenue.Epoch = topicFeeRevenue.Epoch
		amountInt := cosmosMath.NewIntFromBigInt(amount.BigInt())
		newTopicFeeRevenue.Revenue = topicFeeRevenue.Revenue.Add(amountInt)
	}
	return k.topicFeeRevenue.Set(ctx, topicId, newTopicFeeRevenue)
}

// what is the current latest fee revenue epoch
func (k *Keeper) GetFeeRevenueEpoch(ctx context.Context) (uint64, error) {
	return k.feeRevenueEpoch.Peek(ctx)
}

// at the end of a rewards epoch, increment the fee revenue
func (k *Keeper) IncrementFeeRevenueEpoch(ctx context.Context) error {
	_, err := k.feeRevenueEpoch.Next(ctx)
	return err
}

/// MEMPOOL & INFERENCE REQUESTS

func (k *Keeper) AddUnmetDemand(ctx context.Context, topicId TopicId, amt cosmosMath.Uint) error {
	topicUnmetDemand, err := k.GetTopicUnmetDemand(ctx, topicId)
	if err != nil {
		return err
	}
	topicUnmetDemand = topicUnmetDemand.Add(amt)
	return k.topicUnmetDemand.Set(ctx, topicId, topicUnmetDemand)
}

func (k *Keeper) RemoveUnmetDemand(ctx context.Context, topicId TopicId, amt cosmosMath.Uint) error {
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

func (k *Keeper) SetTopicUnmetDemand(ctx context.Context, topicId TopicId, amt cosmosMath.Uint) error {
	if amt.IsZero() {
		return k.topicUnmetDemand.Remove(ctx, topicId)
	}
	return k.topicUnmetDemand.Set(ctx, topicId, amt)
}

func (k *Keeper) GetTopicUnmetDemand(ctx context.Context, topicId TopicId) (Uint, error) {
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

func (k *Keeper) IsRequestInMempool(ctx context.Context, topicId TopicId, requestId string) (bool, error) {
	return k.mempool.Has(ctx, collections.Join(topicId, requestId))
}

func (k *Keeper) GetMempoolInferenceRequestById(ctx context.Context, topicId TopicId, requestId string) (types.InferenceRequest, error) {
	return k.mempool.Get(ctx, collections.Join(topicId, requestId))
}

func (k *Keeper) GetMempoolInferenceRequestsForTopic(ctx context.Context, topicId TopicId) ([]types.InferenceRequest, error) {
	var ret []types.InferenceRequest = make([]types.InferenceRequest, 0)
	rng := collections.NewPrefixedPairRange[TopicId, RequestId](topicId)
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

func (k *Keeper) GetTopicAccumulatedMetDemand(ctx context.Context, topicId TopicId) (Uint, error) {
	res, err := k.accumulatedMetDemand.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return res, nil
}

func (k *Keeper) AddTopicAccumulateMetDemand(ctx context.Context, topicId TopicId, metDemand Uint) error {
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

func (k *Keeper) SetTopicAccumulatedMetDemand(ctx context.Context, topicId TopicId, metDemand Uint) error {
	if metDemand.IsZero() {
		return k.accumulatedMetDemand.Remove(ctx, topicId)
	}
	return k.accumulatedMetDemand.Set(ctx, topicId, metDemand)
}

func (k *Keeper) GetNumInferencesInRewardEpoch(ctx context.Context, topicId TopicId, worker sdk.AccAddress) (Uint, error) {
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

func (k *Keeper) IncrementNumInferencesInRewardEpoch(ctx context.Context, topicId TopicId, worker sdk.AccAddress) error {
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

/// SCORES

func (k *Keeper) InsertWorkerInferenceScore(ctx context.Context, topicId TopicId, blockNumber BlockHeight, score types.Score) error {
	key := collections.Join(topicId, blockNumber)
	var scores types.Scores

	scores, err := k.inferenceScores.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			scores = types.Scores{}
		} else {
			return err
		}
	}
	scores.Scores = append(scores.Scores, &score)

	return k.inferenceScores.Set(ctx, key, scores)
}

func (k *Keeper) InsertWorkerForecastScore(ctx context.Context, topicId TopicId, blockNumber BlockHeight, score types.Score) error {
	key := collections.Join(topicId, blockNumber)

	scores, err := k.forecastScores.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			scores = types.Scores{}
		} else {
			return err
		}
	}
	scores.Scores = append(scores.Scores, &score)

	return k.forecastScores.Set(ctx, key, scores)
}

func (k *Keeper) InsertReputerScore(ctx context.Context, topicId TopicId, blockNumber BlockHeight, score types.Score) error {
	key := collections.Join(topicId, blockNumber)

	scores, err := k.reputerScores.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			scores = types.Scores{}
		} else {
			return err
		}
	}
	scores.Scores = append(scores.Scores, &score)

	return k.reputerScores.Set(ctx, key, scores)
}

func (k *Keeper) GetWorkerInferenceScoresUntilBlock(ctx context.Context, topicId TopicId, blockNumber BlockHeight, worker Worker) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockNumber).
		Descending()

	scores := make([]*types.Score, 0)
	iter, err := k.inferenceScores.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}

	count := 0
	for ; iter.Valid() && count < 10; iter.Next() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, err
		}
		for _, score := range existingScores.Value.Scores {
			if score.Address == worker.String() {
				scores = append(scores, score)
				count++
			}
		}
	}

	return scores, nil
}

func (k *Keeper) GetWorkerForecastScoresUntilBlock(ctx context.Context, topicId TopicId, blockNumber BlockHeight, worker Worker) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockNumber).
		Descending()

	scores := make([]*types.Score, 0)
	iter, err := k.forecastScores.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}

	count := 0
	for ; iter.Valid() && count < 10; iter.Next() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, err
		}
		for _, score := range existingScores.Value.Scores {
			if score.Address == worker.String() {
				scores = append(scores, score)
				count++
			}
		}
	}

	return scores, nil
}

func (k *Keeper) GetReputersScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) ([]*types.Score, error) {
	key := collections.Join(topicId, block)
	scores, err := k.reputerScores.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return []*types.Score{}, nil
		}
		return nil, err
	}
	return scores.Scores, nil
}

func (k *Keeper) SetListeningCoefficient(ctx context.Context, topicId TopicId, reputer sdk.AccAddress, coefficient types.ListeningCoefficient) error {
	key := collections.Join(topicId, reputer)
	return k.reputerListeningCoefficient.Set(ctx, key, coefficient)
}

func (k *Keeper) GetListeningCoefficient(ctx context.Context, topicId TopicId, reputer sdk.AccAddress) (types.ListeningCoefficient, error) {
	key := collections.Join(topicId, reputer)
	coef, err := k.reputerListeningCoefficient.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// Return a default value
			return types.ListeningCoefficient{Coefficient: 1.0}, nil
		}
		return types.ListeningCoefficient{}, err
	}
	return coef, nil
}

/// TAX for REWARD

func (k *Keeper) SetAverageWorkerReward(ctx context.Context, topicId TopicId, worker sdk.AccAddress, value types.AverageWorkerReward) error {
	key := collections.Join(topicId, worker)
	return k.averageWorkerReward.Set(ctx, key, value)
}

func (k *Keeper) GetAverageWorkerReward(ctx context.Context, topicId TopicId, worker sdk.AccAddress) (types.AverageWorkerReward, error) {
	key := collections.Join(topicId, worker)
	val, err := k.averageWorkerReward.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// Return a default value
			return types.AverageWorkerReward{Count: 0, Value: 0.0}, nil
		}
		return types.AverageWorkerReward{}, err
	}
	return val, nil
}

/// WHITELISTS

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

/// BANK KEEPER WRAPPERS

// SendCoinsFromModuleToModule
func (k *Keeper) AccountKeeper() AccountKeeper {
	return k.authKeeper
}

func (k *Keeper) BankKeeper() BankKeeper {
	return k.bankKeeper
}

// SendCoinsFromModuleToAccount
func (k *Keeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// SendCoinsFromAccountToModule
func (k *Keeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}
