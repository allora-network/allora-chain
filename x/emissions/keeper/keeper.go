package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"

	coreStore "cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
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
type RequestIndex = uint64 // A concept completely internal to the keeper. Each request in a topic is indexed by a unique request index.

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
	topics       collections.Map[TopicId, types.Topic]
	activeTopics collections.KeySet[TopicId]
	// every topics that has been churned and ready to get inferences in the block
	churnReadyTopics collections.Item[types.TopicList]
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TopicId, sdk.AccAddress]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TopicId, sdk.AccAddress]]
	// map of (topic) -> nonce/block height
	topicRewardNonce collections.Map[TopicId, int64]

	/// SCORES

	// map of (topic, block_number, worker) -> score
	infererScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_number, worker) -> score
	forecasterScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_number, reputer) -> score
	reputerScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_number, worker) -> score
	latestInfererScoresByWorker collections.Map[collections.Pair[TopicId, Worker], types.Score]
	// map of (topic, block_number, worker) -> score
	latestForecasterScoresByWorker collections.Map[collections.Pair[TopicId, Worker], types.Score]
	// map of (topic, block_number, reputer) -> score
	latestReputerScoresByReputer collections.Map[collections.Pair[TopicId, Reputer], types.Score]
	// map of (topic, reputer) -> listening coefficient
	reputerListeningCoefficient collections.Map[collections.Pair[TopicId, Reputer], types.ListeningCoefficient]
	// map of (topic, reputer) -> previous reward (used for EMA)
	previousReputerRewardFraction collections.Map[collections.Pair[TopicId, Reputer], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for inference (used for EMA)
	previousInferenceRewardFraction collections.Map[collections.Pair[TopicId, Worker], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for forecast (used for EMA)
	previousForecastRewardFraction collections.Map[collections.Pair[TopicId, Worker], alloraMath.Dec]

	/// STAKING

	// total sum stake of all stakers on the network
	totalStake collections.Item[Uint]
	// for every topic, how much total stake does that topic have accumulated?
	topicStake collections.Map[TopicId, Uint]
	// amount of stake a reputer has placed in a topic + delegate stake placed in them, signalling their authority on the topic
	stakeByReputerAndTopicId collections.Map[collections.Pair[TopicId, Reputer], Uint]
	// map of (reputer) -> removal information for that reputer
	stakeRemoval collections.Map[collections.Pair[TopicId, Reputer], types.StakeRemoval]
	// map of (delegator) -> removal information for that delegator
	delegateStakeRemoval collections.Map[collections.Triple[TopicId, Reputer, Delegator], types.DelegateStakeRemoval]
	// map of (delegator) -> amount of stake that has been placed by that delegator
	stakeFromDelegator collections.Map[collections.Pair[TopicId, Delegator], Uint]
	// map of (delegator, target) -> amount of stake that has been placed by that delegator on that target
	delegateStakePlacement collections.Map[collections.Triple[TopicId, Reputer, Delegator], Uint]
	// map of (target) -> amount of stake that has been placed on that target
	stakeUponReputer collections.Map[collections.Pair[TopicId, Reputer], Uint]

	/// INFERENCE REQUEST MEMPOOL

	// map of (request_id) -> full InferenceRequest information for that request
	requests collections.Map[RequestId, types.InferenceRequest]
	// map of (topic, request_index) -> bounded list of request_ids
	topicRequests collections.Map[collections.Pair[TopicId, uint64], RequestId]
	// map of (topic) -> number of active requests in a topic
	numRequestsPerTopic collections.Map[TopicId, uint64]
	// amount of money available for an inference request id that has been placed in the mempool but has not yet been fully satisfied
	requestUnmetDemand collections.Map[RequestId, Uint]
	// total amount of demand for a topic that has been placed in the mempool as a request for inference but has not yet been satisfied
	topicUnmetDemand collections.Map[TopicId, Uint]

	/// MISC GLOBAL STATE

	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TopicId, Worker], types.Inference]

	// map of (topic, worker) -> forecast[]
	forecasts collections.Map[collections.Pair[TopicId, Worker], types.Forecast]

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

	/// NONCES

	// map of (topic) -> unfulfilled nonces
	unfulfilledWorkerNonces collections.Map[TopicId, types.Nonces]

	// map of (topic) -> unfulfilled nonces
	unfulfilledReputerNonces collections.Map[TopicId, types.ReputerRequestNonces]

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
	storeService coreStore.KVStoreService,
	ak AccountKeeper,
	bk BankKeeper,
	feeCollectorName string,
) Keeper {

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                                 cdc,
		addressCodec:                        addressCodec,
		feeCollectorName:                    feeCollectorName,
		params:                              collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		authKeeper:                          ak,
		bankKeeper:                          bk,
		totalStake:                          collections.NewItem(sb, types.TotalStakeKey, "total_stake", alloraMath.UintValue),
		topicStake:                          collections.NewMap(sb, types.TopicStakeKey, "topic_stake", collections.Uint64Key, alloraMath.UintValue),
		lastRewardsUpdate:                   collections.NewItem(sb, types.LastRewardsUpdateKey, "last_rewards_update", collections.Int64Value),
		nextTopicId:                         collections.NewSequence(sb, types.NextTopicIdKey, "next_TopicId"),
		topics:                              collections.NewMap(sb, types.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[types.Topic](cdc)),
		activeTopics:                        collections.NewKeySet(sb, types.ActiveTopicsKey, "active_topics", collections.Uint64Key),
		churnReadyTopics:                    collections.NewItem(sb, types.ChurnReadyTopicsKey, "churn_ready_topics", codec.CollValue[types.TopicList](cdc)),
		topicWorkers:                        collections.NewKeySet(sb, types.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		topicReputers:                       collections.NewKeySet(sb, types.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey)),
		stakeByReputerAndTopicId:            collections.NewMap(sb, types.StakeByReputerAndTopicIdKey, "stake_by_reputer_and_TopicId", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), alloraMath.UintValue),
		stakeRemoval:                        collections.NewMap(sb, types.StakeRemovalKey, "stake_removal_queue", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.StakeRemoval](cdc)),
		delegateStakeRemoval:                collections.NewMap(sb, types.DelegateStakeRemovalKey, "delegate_stake_removal_queue", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), codec.CollValue[types.DelegateStakeRemoval](cdc)),
		stakeFromDelegator:                  collections.NewMap(sb, types.DelegatorStakeKey, "stake_from_delegator", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), alloraMath.UintValue),
		delegateStakePlacement:              collections.NewMap(sb, types.DelegateStakePlacementKey, "delegate_stake_placement", collections.TripleKeyCodec(collections.Uint64Key, sdk.AccAddressKey, sdk.AccAddressKey), alloraMath.UintValue),
		stakeUponReputer:                    collections.NewMap(sb, types.TargetStakeKey, "stake_upon_reputer", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), alloraMath.UintValue),
		requests:                            collections.NewMap(sb, types.RequestsKey, "requests", collections.StringKey, codec.CollValue[types.InferenceRequest](cdc)),
		topicRequests:                       collections.NewMap(sb, types.TopicRequestsKey, "topic_requests", collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), collections.StringValue),
		numRequestsPerTopic:                 collections.NewMap(sb, types.NumRequestsPerTopicKey, "num_requests_per_topic", collections.Uint64Key, collections.Uint64Value),
		requestUnmetDemand:                  collections.NewMap(sb, types.RequestUnmetDemandKey, "request_unmet_demand", collections.StringKey, alloraMath.UintValue),
		topicUnmetDemand:                    collections.NewMap(sb, types.TopicUnmetDemandKey, "topic_unmet_demand", collections.Uint64Key, alloraMath.UintValue),
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
		whitelistAdmins:                     collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", sdk.AccAddressKey),
		topicCreationWhitelist:              collections.NewKeySet(sb, types.TopicCreationWhitelistKey, "topic_creation_whitelist", sdk.AccAddressKey),
		reputerWhitelist:                    collections.NewKeySet(sb, types.ReputerWhitelistKey, "weight_setting_whitelist", sdk.AccAddressKey),
		infererScoresByBlock:                collections.NewMap(sb, types.InferenceScoresKey, "inferer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		forecasterScoresByBlock:             collections.NewMap(sb, types.ForecastScoresKey, "forecaster_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		latestInfererScoresByWorker:         collections.NewMap(sb, types.LatestInfererScoresByWorkerKey, "latest_inferer_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Score](cdc)),
		latestForecasterScoresByWorker:      collections.NewMap(sb, types.LatestForecasterScoresByWorkerKey, "latest_forecaster_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Score](cdc)),
		latestReputerScoresByReputer:        collections.NewMap(sb, types.LatestReputerScoresByReputerKey, "latest_reputer_scores_by_reputer", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Score](cdc)),
		previousReputerRewardFraction:       collections.NewMap(sb, types.PreviousReputerRewardFractionKey, "previous_reputer_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), alloraMath.DecValue),
		previousInferenceRewardFraction:     collections.NewMap(sb, types.PreviousInferenceRewardFractionKey, "previous_inference_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), alloraMath.DecValue),
		previousForecastRewardFraction:      collections.NewMap(sb, types.PreviousForecastRewardFractionKey, "previous_forecast_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), alloraMath.DecValue),
		reputerScoresByBlock:                collections.NewMap(sb, types.ReputerScoresKey, "reputer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerListeningCoefficient:         collections.NewMap(sb, types.ReputerListeningCoefficientKey, "reputer_listening_coefficient", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.ListeningCoefficient](cdc)),
		unfulfilledWorkerNonces:             collections.NewMap(sb, types.UnfulfilledWorkerNoncesKey, "unfulfilled_worker_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
		unfulfilledReputerNonces:            collections.NewMap(sb, types.UnfulfilledReputerNoncesKey, "unfulfilled_reputer_nonces", collections.Uint64Key, codec.CollValue[types.ReputerRequestNonces](cdc)),
		topicRewardNonce:                    collections.NewMap(sb, types.TopicRewardNonceKey, "topic_reward_nonce", collections.Uint64Key, collections.Int64Value),
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
		if n.BlockHeight == nonce.BlockHeight {
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
func (k *Keeper) FulfillReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return false, err
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			err := k.unfulfilledReputerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	// If the nonce is not present in the unfulfilled nonces
	return false, nil
}

// True if nonce is unfulfilled, false otherwise.
func (k *Keeper) IsWorkerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, err
	}

	if nonce == nil {
		return false, errors.New("nil worker nonce provided")
	}
	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n == nil {
			fmt.Println("warn: nil worker nonce stored")
			continue
		}
		if n.BlockHeight == nonce.BlockHeight {
			fmt.Println("Worker nonce found", nonce)
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
	if nonce == nil {
		return false, errors.New("nil reputer nonce provided")
	}
	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n == nil {
			fmt.Println("warn: nil reputer nonce stored")
			continue
		}
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			fmt.Println("Reputer nonce found", nonce)
			return true, nil
		}
	}
	return false, nil
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *Keeper) AddWorkerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	nonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return err
	}

	// Check that input nonce is not already contained in the nonces of this topic
	for _, n := range nonces.Nonces {
		if n.BlockHeight == nonce.BlockHeight {
			return nil
		}
	}
	nonces.Nonces = append([]*types.Nonce{nonce}, nonces.Nonces...)

	maxUnfulfilledRequests, err := k.GetParamsMaxUnfulfilledWorkerRequests(ctx)
	if err != nil {
		return err
	}

	lenNonces := uint64(len(nonces.Nonces))
	if lenNonces > maxUnfulfilledRequests {
		diff := uint64(len(nonces.Nonces)) - maxUnfulfilledRequests
		if diff > 0 {
			nonces.Nonces = nonces.Nonces[:maxUnfulfilledRequests]
		}
	}

	return k.unfulfilledWorkerNonces.Set(ctx, topicId, nonces)
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *Keeper) AddReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce, associatedWorkerNonce *types.Nonce) error {
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return err
	}

	// Check that input nonce is not already contained in the nonces of this topic
	// nor that the `associatedWorkerNonce` is already associated with a worker requeset
	for _, n := range nonces.Nonces {
		// Do nothing if nonce is already in the list
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			return nil
		}
		// Do nothing if the associated worker nonce is already in the list
		if n.WorkerNonce.BlockHeight == associatedWorkerNonce.BlockHeight {
			return nil
		}
	}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: nonce,
		WorkerNonce:  associatedWorkerNonce,
	}
	nonces.Nonces = append([]*types.ReputerRequestNonce{reputerRequestNonce}, nonces.Nonces...)

	maxUnfulfilledRequests, err := k.GetParamsMaxUnfulfilledReputerRequests(ctx)
	if err != nil {
		return err
	}
	lenNonces := uint64(len(nonces.Nonces))
	if lenNonces > maxUnfulfilledRequests {
		diff := uint64(len(nonces.Nonces)) - maxUnfulfilledRequests
		if diff > 0 {
			nonces.Nonces = nonces.Nonces[:maxUnfulfilledRequests]
		}
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

func (k *Keeper) GetUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) (types.ReputerRequestNonces, error) {
	nonces, err := k.unfulfilledReputerNonces.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ReputerRequestNonces{}, nil
		}
		return types.ReputerRequestNonces{}, err
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
	return k.latestForecasterNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) SetOneInForecasterNetworkRegret(ctx context.Context, topicId TopicId, forecaster Worker, inferer Worker, regret types.TimestampedValue) error {
	key := collections.Join3(topicId, forecaster, inferer)
	return k.latestOneInForecasterNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a worker from comparing loss of worker relative to loss of other inferers
// Returns (0, true) if no regret is found
func (k *Keeper) GetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker Worker) (types.TimestampedValue, bool, error) {
	key := collections.Join(topicId, worker)
	regret, err := k.latestInfererNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       alloraMath.NewDecFromInt64(0),
			}, true, nil
		}
		return types.TimestampedValue{}, false, err
	}
	return regret, false, nil
}

// Returns the regret of a worker from comparing loss of worker relative to loss of other inferers
// Returns (0, true) if no regret is found
func (k *Keeper) GetForecasterNetworkRegret(ctx context.Context, topicId TopicId, worker Worker) (types.TimestampedValue, bool, error) {
	key := collections.Join(topicId, worker)
	regret, err := k.latestForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       alloraMath.NewDecFromInt64(0),
			}, true, nil
		}
		return types.TimestampedValue{}, false, err
	}
	return regret, false, nil
}

// Returns the regret of a worker from comparing loss of worker relative to loss of other inferers
// Returns (0, true) if no regret is found
func (k *Keeper) GetOneInForecasterNetworkRegret(ctx context.Context, topicId TopicId, forecaster Worker, inferer Worker) (types.TimestampedValue, bool, error) {
	key := collections.Join3(topicId, forecaster, inferer)
	regret, err := k.latestOneInForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       alloraMath.NewDecFromInt64(0),
			}, true, nil
		}
		return types.TimestampedValue{}, false, err
	}
	return regret, false, nil
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

func (k *Keeper) GetParamsMaxTopicsPerBlock(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxTopicsPerBlock, nil
}

func (k *Keeper) GetParamsMinRequestUnmetDemand(ctx context.Context) (cosmosMath.Uint, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return params.MinRequestUnmetDemand, nil
}

func (k *Keeper) GetParamsMinTopicUnmetDemand(ctx context.Context) (cosmosMath.Uint, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	return params.MinTopicUnmetDemand, nil
}

func (k *Keeper) GetParamsRequiredMinimumStake(ctx context.Context) (cosmosMath.Uint, error) {
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

func (k *Keeper) GetParamsSharpness(ctx context.Context) (alloraMath.Dec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return params.Sharpness, nil
}

func (k *Keeper) GetParamsLearningRate(ctx context.Context) (alloraMath.Dec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return params.LearningRate, nil
}

func (k *Keeper) GetParamsGradientDescentMaxIters(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.GradientDescentMaxIters, nil
}

func (k *Keeper) GetParamsStakeAndFeeRevenueImportance(ctx context.Context) (alloraMath.Dec, alloraMath.Dec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, alloraMath.Dec{}, err
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

func (k *Keeper) GetParamsMaxUnfulfilledReputerRequests(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxUnfulfilledReputerRequests, nil
}

func (k *Keeper) GetParamsTopicRewardAlpha(ctx context.Context) (alloraMath.Dec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return params.TopicRewardAlpha, nil
}

func (k Keeper) GetParamsValidatorsVsAlloraPercentReward(ctx context.Context) (alloraMath.Dec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return params.ValidatorsVsAlloraPercentReward, nil
}

func (k *Keeper) GetParamsMaxSamplesToScaleScores(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxSamplesToScaleScores, nil
}

func (k *Keeper) GetParamsMaxTopWorkersToReward(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxTopWorkersToReward, nil
}

func (k *Keeper) GetParamsTopicCreationFee(ctx context.Context) (cosmosMath.Int, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Int{}, err
	}
	return params.CreateTopicFee, nil
}

func (k *Keeper) GetParamsMaxRetriesToFulfilNoncesWorker(ctx context.Context) (int64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxRetriesToFulfilNoncesWorker, nil
}

func (k *Keeper) GetParamsMaxRetriesToFulfilNoncesReputer(ctx context.Context) (int64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxRetriesToFulfilNoncesReputer, nil
}

func (k *Keeper) GetParamsRegistrationFee(ctx context.Context) (cosmosMath.Int, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return cosmosMath.Int{}, err
	}
	return params.RegistrationFee, nil
}

func (k *Keeper) GetParamsMaxRequestsPerTopic(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxRequestsPerTopic, nil
}

func (k *Keeper) GetParamsDefaultLimit(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return uint64(0), err
	}
	return params.DefaultLimit, nil
}

func (k *Keeper) GetParamsMaxLimit(ctx context.Context) (uint64, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return uint64(0), err
	}
	return params.MaxLimit, nil
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
	// Define the range query starting from the highest available block down to and including the specified block
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		StartInclusive(block) // Set the lower boundary as the specified block, inclusive

	var inferencesToReturn types.Inferences
	currentBlockHeight := BlockHeight(0)

	iter, err := k.allInferences.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close() // Ensure that resources are released

	// Iterate through entries in descending order and collect all inferences after the specified block
	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return nil, 0, err
		}
		currentBlockHeight = kv.Key.K2() // Current entry's block height
		inferencesToReturn.Inferences = kv.Value.Inferences
		break
	}

	// Return the collected inferences and the lowest block height at which they were found
	return &inferencesToReturn, currentBlockHeight, nil
}

func (k *Keeper) GetForecastsAtOrAfterBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Forecasts, BlockHeight, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		StartInclusive(block) // Set the lower boundary as the specified block, inclusive

	forecastsToReturn := types.Forecasts{}
	currentBlockHeight := BlockHeight(0)

	iter, err := k.allForecasts.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close() // Ensure that resources are released

	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return nil, 0, err
		}
		currentBlockHeight = kv.Key.K2() // Current entry's block height
		forecastsToReturn.Forecasts = kv.Value.Forecasts
		break
	}

	return &forecastsToReturn, currentBlockHeight, nil
}

// Insert a complete set of inferences for a topic/block. Overwrites previous ones.
func (k *Keeper) InsertInferences(ctx context.Context, topicId TopicId, nonce types.Nonce, inferences types.Inferences) error {
	block := nonce.BlockHeight

	for _, inference := range inferences.Inferences {
		inferenceCopy := *inference
		// Update latests inferences for each worker
		workerAcc, err := sdk.AccAddressFromBech32(inferenceCopy.Inferer)
		if err != nil {
			return err
		}
		key := collections.Join(topicId, workerAcc)
		err = k.inferences.Set(ctx, key, inferenceCopy)
		if err != nil {
			return err
		}
	}

	key := collections.Join(topicId, block)
	return k.allInferences.Set(ctx, key, inferences)
}

// Insert a complete set of inferences for a topic/block. Overwrites previous ones.
func (k *Keeper) InsertForecasts(ctx context.Context, topicId TopicId, nonce types.Nonce, forecasts types.Forecasts) error {
	block := nonce.BlockHeight

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

/// TOPIC REWARD NONCE

// GetTopicRewardNonce retrieves the reward nonce for a given topic ID.
func (k *Keeper) GetTopicRewardNonce(ctx context.Context, topicId TopicId) (int64, error) {
	nonce, err := k.topicRewardNonce.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil // Return 0 if not found
		}
		return 0, err
	}
	return nonce, nil
}

// SetTopicRewardNonce sets the reward nonce for a given topic ID.
func (k *Keeper) SetTopicRewardNonce(ctx context.Context, topicId TopicId, nonce int64) error {
	return k.topicRewardNonce.Set(ctx, topicId, nonce)
}

// DeleteTopicRewardNonce removes the reward nonce entry for a given topic ID.
func (k *Keeper) DeleteTopicRewardNonce(ctx context.Context, topicId TopicId) error {
	return k.topicRewardNonce.Remove(ctx, topicId)
}

/// LOSS BUNDLES

// Insert a loss bundle for a topic and timestamp. Overwrites previous ones stored at that composite index.
func (k *Keeper) InsertReputerLossBundlesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight, reputerLossBundles types.ReputerValueBundles) error {
	key := collections.Join(topicId, block)
	return k.allLossBundles.Set(ctx, key, reputerLossBundles)
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
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(block).
		Descending()

	iter, err := k.networkLossBundles.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close() // Ensure resources are released properly

	if !iter.Valid() {
		// Return empty loss bundle if no loss bundle is found
		return &types.ValueBundle{}, 0, nil
	}

	kv, err := iter.KeyValue()
	if err != nil {
		return nil, 0, err
	}

	// Return the found bundle and the associated block height
	return &kv.Value, kv.Key.K2(), nil
}

func (k *Keeper) GetReputerReportedLossesAtOrBeforeBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ReputerValueBundles, BlockHeight, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(block). // Correctly set the end of the range to include the block
		Descending()

	iter, err := k.allLossBundles.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close() // Properly manage resources by closing the iterator when done

	if !iter.Valid() {
		// Return empty loss bundle if no loss bundle is found
		return &types.ReputerValueBundles{}, 0, nil
	}

	kv, err := iter.KeyValue()
	if err != nil {
		return nil, 0, err
	}

	// Return the found bundle and the associated block height
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

func (k *Keeper) AddDelegateStake(ctx context.Context, topicId TopicId, delegator sdk.AccAddress, reputer sdk.AccAddress, stake Uint) error {
	// Run checks to ensure that delegate stake can be added, and then update the types all at once, applying rollbacks if necessary
	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	stakeFromDelegator, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegator)
	if err != nil {
		return err
	}
	stakeFromDelegatorNew := stakeFromDelegator.Add(stake)

	delegateStakePlacement, err := k.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return err
	}
	stakePlacementNew := delegateStakePlacement.Add(stake)

	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
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
	if err := k.SetDelegateStakePlacement(ctx, topicId, delegator, reputer, stakePlacementNew); err != nil {
		fmt.Println("Setting topic stake failed -- rolling back stake from delegator")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		return err
	}

	if err := k.SetDelegateStakeUponReputer(ctx, topicId, reputer, stakeUponReputerNew); err != nil {
		fmt.Println("Setting total stake failed -- rolling back stake from delegator and delegate stake placement")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		err2 = k.SetDelegateStakePlacement(ctx, topicId, delegator, reputer, delegateStakePlacement)
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
	delegateStakeUponReputerInTopic, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	reputerStakeInTopicWithoutDelegateStake := reputerStakeInTopic.Sub(delegateStakeUponReputerInTopic)
	// TODO Maybe we should check if reputerStakeInTopicWithoutDelegateStake is zero and remove the key from the map
	if stake.GT(reputerStakeInTopicWithoutDelegateStake) {
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

// Removes delegate stake from the system for a given topic, delegator, and reputer
func (k *Keeper) RemoveDelegateStake(
	ctx context.Context,
	topicId TopicId,
	delegator sdk.AccAddress,
	reputer sdk.AccAddress,
	stake Uint) error {
	// Run checks to ensure that the delegate stake can be removed, and then update the types all at once, applying rollbacks if necessary

	if stake.IsZero() {
		return errors.New("stake must be greater than zero")
	}

	// Check stakeFromDelegator >= stake
	stakeFromDelegator, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegator)
	if err != nil {
		return err
	}
	if stake.GT(stakeFromDelegator) {
		return types.ErrIntegerUnderflowStakeFromDelegator
	}
	stakeFromDelegatorNew := stakeFromDelegator.Sub(stake)

	// Check stakePlacement >= stake
	stakePlacement, err := k.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return err
	}
	if stake.GT(stakePlacement) {
		return types.ErrIntegerUnderflowDelegateStakePlacement
	}
	stakePlacementNew := stakePlacement.Sub(stake)

	// Check stakeUponReputer >= stake
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if stake.GT(stakeUponReputer) {
		return types.ErrIntegerUnderflowDelegateStakeUponReputer
	}
	stakeUponReputerNew := stakeUponReputer.Sub(stake)

	// types updates -- done all at once after all checks

	// Set new stake from delegator
	if err := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegatorNew); err != nil {
		return err
	}

	// Set new delegate stake placement
	if err := k.SetDelegateStakePlacement(ctx, topicId, delegator, reputer, stakePlacementNew); err != nil {
		fmt.Println("Setting delegate stake placement failed -- rolling back stake from delegator")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		return err
	}

	// Set new delegate stake upon reputer
	if err := k.SetDelegateStakeUponReputer(ctx, topicId, reputer, stakeUponReputerNew); err != nil {
		fmt.Println("Setting delegate stake upon reputer failed -- rolling back stake from delegator and delegate stake placement")
		err2 := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegator)
		if err2 != nil {
			return err2
		}
		err2 = k.SetDelegateStakePlacement(ctx, topicId, delegator, reputer, stakePlacement)
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
func (k *Keeper) GetStakeFromDelegatorInTopic(ctx context.Context, topicId TopicId, delegator Delegator) (Uint, error) {
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
func (k *Keeper) GetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator Delegator, target Reputer) (Uint, error) {
	key := collections.Join3(topicId, delegator, target)
	stake, err := k.delegateStakePlacement.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewUint(0), nil
		}
		return cosmosMath.Uint{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) SetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator Delegator, target Reputer, stake Uint) error {
	key := collections.Join3(topicId, delegator, target)
	if stake.IsZero() {
		return k.delegateStakePlacement.Remove(ctx, key)
	}
	return k.delegateStakePlacement.Set(ctx, key, stake)
}

// Returns the amount of stake placed on a specific target.
func (k *Keeper) GetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target Reputer) (Uint, error) {
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
func (k *Keeper) SetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target Reputer, stake Uint) error {
	key := collections.Join(topicId, target)
	if stake.IsZero() {
		return k.stakeUponReputer.Remove(ctx, key)
	}
	return k.stakeUponReputer.Set(ctx, key, stake)
}

// For a given topic id and reputer address, get their stake removal information
func (k *Keeper) GetStakeRemovalByTopicAndAddress(ctx context.Context, topicId TopicId, address sdk.AccAddress) (types.StakeRemoval, error) {
	key := collections.Join(topicId, address)
	return k.stakeRemoval.Get(ctx, key)
}

// For a given address, adds their stake removal information to the removal queue for delay waiting
// The topic used will be the topic set in the `removalInfo`
// This completely overrides the existing stake removal
func (k *Keeper) SetStakeRemoval(ctx context.Context, address sdk.AccAddress, removalInfo types.StakeRemoval) error {
	key := collections.Join(removalInfo.Placement.TopicId, address)
	return k.stakeRemoval.Set(ctx, key, removalInfo)
}

// For a given topic id and reputer address, get their stake removal information
func (k *Keeper) GetDelegateStakeRemovalByTopicAndAddress(ctx context.Context, topicId TopicId, reputer sdk.AccAddress, delegator sdk.AccAddress) (types.DelegateStakeRemoval, error) {
	key := collections.Join3(topicId, reputer, delegator)
	return k.delegateStakeRemoval.Get(ctx, key)
}

// For a given address, adds their stake removal information to the removal queue for delay waiting
// The topic used will be the topic set in the `removalInfo`
// This completely overrides the existing stake removal
func (k *Keeper) SetDelegateStakeRemoval(ctx context.Context, removalInfo types.DelegateStakeRemoval) error {
	reputerAddress, err := sdk.AccAddressFromBech32(removalInfo.Placement.Reputer)
	if err != nil {
		return err
	}
	delegatorAddress, err := sdk.AccAddressFromBech32(removalInfo.Placement.Delegator)
	if err != nil {
		return err
	}
	key := collections.Join3(removalInfo.Placement.TopicId, reputerAddress, delegatorAddress)
	return k.delegateStakeRemoval.Set(ctx, key, removalInfo)
}

/// REPUTERS

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *Keeper) InsertReputer(ctx context.Context, topicId TopicId, reputer sdk.AccAddress, reputerInfo types.OffchainNode) error {
	topicKey := collections.Join(topicId, reputer)
	err := k.topicReputers.Set(ctx, topicKey)
	if err != nil {
		return err
	}
	err = k.reputers.Set(ctx, reputerInfo.LibP2PKey, reputerInfo)
	if err != nil {
		return err
	}
	return nil
}

// Remove a reputer to the reputer tracking data structures and topicReputers
func (k *Keeper) RemoveReputer(ctx context.Context, topicId TopicId, reputerAddr sdk.AccAddress) error {
	topicKey := collections.Join(topicId, reputerAddr)
	err := k.topicReputers.Remove(ctx, topicKey)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) GetReputerByLibp2pKey(ctx sdk.Context, reputerKey string) (types.OffchainNode, error) {
	return k.reputers.Get(ctx, reputerKey)
}

/// WORKERS

// Adds a new worker to the worker tracking data structures, workers and topicWorkers
func (k *Keeper) InsertWorker(ctx context.Context, topicId TopicId, worker sdk.AccAddress, workerInfo types.OffchainNode) error {
	topickey := collections.Join(topicId, worker)
	err := k.topicWorkers.Set(ctx, topickey)
	if err != nil {
		return err
	}
	err = k.workers.Set(ctx, workerInfo.LibP2PKey, workerInfo)
	if err != nil {
		return err
	}
	return nil
}

// Remove a worker to the worker tracking data structures and topicWorkers
func (k *Keeper) RemoveWorker(ctx context.Context, topicId TopicId, workerAddr sdk.AccAddress) error {
	topicKey := collections.Join(topicId, workerAddr)
	err := k.topicWorkers.Remove(ctx, topicKey)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) GetWorkerByLibp2pKey(ctx sdk.Context, workerKey string) (types.OffchainNode, error) {
	return k.workers.Get(ctx, workerKey)
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

func (k *Keeper) GetReputerAddressByP2PKey(ctx context.Context, p2pKey string) (sdk.AccAddress, error) {
	reputer, err := k.reputers.Get(ctx, p2pKey)
	if err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(reputer.GetOwner())
	if err != nil {
		return nil, err
	}

	return address, nil
}

/// TOPICS

// Get the previous weight during rewards calculation for a topic
// Returns ((0,0), true) if there was no prior topic weight set, else ((x,y), false) where x,y!=0
func (k *Keeper) GetPreviousTopicWeight(ctx context.Context, topicId TopicId) (types.PreviousTopicWeight, bool, error) {
	topicWeight, err := k.previousTopicWeight.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		ret := types.PreviousTopicWeight{
			Weight: alloraMath.ZeroDec(),
			Epoch:  0,
		}
		return ret, true, nil
	}
	return topicWeight, false, err
}

// Set the previous weight during rewards calculation for a topic
func (k *Keeper) SetPreviousTopicWeight(ctx context.Context, topicId TopicId, weight types.PreviousTopicWeight) error {
	return k.previousTopicWeight.Set(ctx, topicId, weight)
}

// Set a topic to inactive if the topic exists and is active, else does nothing
func (k *Keeper) InactivateTopic(ctx context.Context, topicId TopicId) error {
	present, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return err
	}

	isActive, err := k.activeTopics.Has(ctx, topicId)
	if err != nil {
		return err
	}

	if present && isActive {
		err = k.activeTopics.Remove(ctx, topicId)
		if err != nil {
			return err
		}
	}

	return nil
}

// Set a topic to active if the topic exists, else does nothing
func (k *Keeper) ActivateTopic(ctx context.Context, topicId TopicId) error {
	present, err := k.topics.Has(ctx, topicId)
	if err != nil {
		return err
	}

	if !present {
		return nil
	}

	if err := k.activeTopics.Set(ctx, topicId); err != nil {
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

// Checks if a topic exists
func (k *Keeper) TopicExists(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topics.Has(ctx, topicId)
}

// Returns the number of topics that are active in the network
func (k *Keeper) GetNextTopicId(ctx context.Context) (TopicId, error) {
	return k.nextTopicId.Peek(ctx)
}

func (k *Keeper) IsTopicActive(ctx context.Context, topicId TopicId) (bool, error) {
	return k.activeTopics.Has(ctx, topicId)
}

func (k Keeper) GetActiveTopics(ctx context.Context, pagination *types.SimpleCursorPaginationRequest) ([]TopicId, *types.SimpleCursorPaginationResponse, error) {
	limit, startTopicId, err := k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	if err != nil {
		return nil, nil, err
	}

	rng := new(collections.Range[uint64]).StartInclusive(startTopicId)

	iter, err := k.activeTopics.Iterate(ctx, rng)
	if err != nil {
		return nil, nil, err
	}
	defer iter.Close()

	activeTopics := make([]TopicId, 0)
	count := uint64(0)
	for ; iter.Valid(); iter.Next() {
		newKey, err := iter.Key()
		if err != nil {
			return nil, nil, err
		}

		activeTopics = append(activeTopics, newKey)

		if count >= limit {
			break
		}
		count++
	}

	// If there are no more topics, we return the same cursor as the last one
	if !iter.Valid() {
		return activeTopics, &types.SimpleCursorPaginationResponse{
			NextKey: pagination.Key,
		}, nil
	}

	// If there are more topics, we return the next key
	nextKey := make([]byte, 8)
	binary.BigEndian.PutUint64(nextKey, startTopicId+limit)

	return activeTopics, &types.SimpleCursorPaginationResponse{
		NextKey: nextKey,
	}, nil
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

func (k *Keeper) GetTopicEpochLastEnded(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return 0, err
	}
	ret := topic.EpochLastEnded
	return ret, nil
}

// True if worker is registered in topic, else False
func (k *Keeper) IsWorkerRegisteredInTopic(ctx context.Context, topicId TopicId, worker sdk.AccAddress) (bool, error) {
	topickey := collections.Join(topicId, worker)
	return k.topicWorkers.Has(ctx, topickey)
}

// True if reputer is registered in topic, else False
func (k *Keeper) IsReputerRegisteredInTopic(ctx context.Context, topicId TopicId, reputer sdk.AccAddress) (bool, error) {
	topickey := collections.Join(topicId, reputer)
	return k.topicReputers.Has(ctx, topickey)
}

/// FEE REVENUE

// Get the amount of fee revenue collected by a topic
func (k *Keeper) GetTopicFeeRevenue(ctx context.Context, topicId TopicId) (types.TopicFeeRevenue, error) {
	feeRev, err := k.topicFeeRevenue.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return types.TopicFeeRevenue{
			Epoch:   0,
			Revenue: cosmosMath.ZeroInt(),
		}, nil
	}
	return feeRev, nil
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

func (k *Keeper) GetTopicMempoolRequestCount(ctx context.Context, topicId TopicId) (RequestIndex, error) {
	num, err := k.numRequestsPerTopic.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return uint64(0), nil
		} else {
			return uint64(0), err
		}
	}
	return num, nil
}

func (k *Keeper) IncrementTopicMempoolRequestCount(ctx context.Context, topicId TopicId) (RequestIndex, error) {
	num, err := k.GetTopicMempoolRequestCount(ctx, topicId)
	if err != nil {
		return uint64(0), err
	}
	num++
	err = k.numRequestsPerTopic.Set(ctx, topicId, num)
	if err != nil {
		return uint64(0), err
	}
	return num, nil
}

func (k *Keeper) DecrementTopicMempoolRequestCount(ctx context.Context, topicId TopicId) (RequestIndex, error) {
	num, err := k.GetTopicMempoolRequestCount(ctx, topicId)
	if err != nil {
		return uint64(0), err
	}
	num--
	if num < 0 {
		return uint64(0), nil
	}
	err = k.numRequestsPerTopic.Set(ctx, topicId, num)
	if err != nil {
		return uint64(0), err
	}
	return num, nil
}

func (k *Keeper) addToTopicMempool(ctx context.Context, topicId TopicId, requestIndex RequestIndex, requestId RequestId) error {
	key := collections.Join(topicId, requestIndex)
	return k.topicRequests.Set(ctx, key, requestId)
}

func (k *Keeper) removeFromTopicMempool(ctx context.Context, topicId TopicId, requestIndex RequestIndex) error {
	key := collections.Join(topicId, requestIndex)
	return k.topicRequests.Remove(ctx, key)
}

// Throws an error if we are already at capacity
// Does not check for pre-existing inclusion in the topic mempool before adding!
func (k *Keeper) AddToMempool(ctx context.Context, request types.InferenceRequest) (Uint, error) {
	hasRequest, err := k.requests.Has(ctx, request.Id)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	if hasRequest {
		return cosmosMath.Uint{}, types.ErrRequestAlreadyExists
	}

	count, err := k.GetTopicMempoolRequestCount(ctx, request.TopicId)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	maxCount, err := k.GetParamsMaxRequestsPerTopic(ctx)
	if err != nil {
		return cosmosMath.Uint{}, err
	}
	if count >= maxCount {
		return cosmosMath.Uint{}, types.ErrTopicMempoolAtCapacity
	}

	err = k.requests.Set(ctx, request.Id, request)
	if err != nil {
		return cosmosMath.Uint{}, err
	}

	newIndex, err := k.IncrementTopicMempoolRequestCount(ctx, request.TopicId)
	if err != nil {
		return cosmosMath.Uint{}, err
	}

	err = k.addToTopicMempool(ctx, request.TopicId, newIndex-1, request.Id)
	if err != nil {
		return cosmosMath.Uint{}, err
	}

	err = k.AddUnmetDemand(ctx, request.TopicId, request.BidAmount)
	if err != nil {
		return cosmosMath.Uint{}, err
	}

	return k.GetTopicUnmetDemand(ctx, request.TopicId)
}

// Does not check for pre-existing exclusion in the topic mempool before removing!
func (k *Keeper) RemoveFromMempool(ctx context.Context, requestId RequestId) error {
	request, err := k.GetMempoolInferenceRequestById(ctx, requestId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ErrRequestDoesNotExist
		}
		return err
	}

	err = k.requests.Remove(ctx, requestId)
	if err != nil {
		return err
	}

	newCount, err := k.DecrementTopicMempoolRequestCount(ctx, request.TopicId)
	if err != nil {
		return err
	}

	err = k.removeFromTopicMempool(ctx, request.TopicId, newCount)
	if err != nil {
		return err
	}

	return k.RemoveUnmetDemand(ctx, request.TopicId, request.BidAmount)
}

func (k *Keeper) IsRequestInMempool(ctx context.Context, requestId RequestId) (bool, error) {
	return k.requests.Has(ctx, requestId)
}

func (k *Keeper) GetMempoolInferenceRequestById(ctx context.Context, requestId RequestId) (types.InferenceRequest, error) {
	return k.requests.Get(ctx, requestId)
}

// Returns pages of inference requests in the mempool for a given topic
func (k *Keeper) GetMempoolInferenceRequestsForTopic(
	ctx context.Context,
	topicId TopicId,
	pagination *types.SimpleCursorPaginationRequest,
) ([]types.InferenceRequest, *types.SimpleCursorPaginationResponse, error) {
	limit, cursor, err := k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	if err != nil {
		return nil, nil, err
	}

	numRequestsInTopic, err := k.numRequestsPerTopic.Get(ctx, topicId)
	if err != nil {
		return nil, nil, err
	}

	// If the cursor is greater than the number of requests in the topic, return an empty list
	if numRequestsInTopic <= cursor {
		return []types.InferenceRequest{}, nil, nil
	}

	// Get the requests in the page
	requests := make([]types.InferenceRequest, 0)
	for i := cursor; i < k.Uint64Min(numRequestsInTopic, limit+numRequestsInTopic); i++ {
		key := collections.Join(topicId, i)
		requestId, err := k.topicRequests.Get(ctx, key)
		if err != nil {
			return nil, nil, err
		}
		request, err := k.GetMempoolInferenceRequestById(ctx, requestId)
		if err != nil {
			return nil, nil, err
		}
		requests = append(requests, request)
	}

	// If the number of requests in the topic is less than the limit, return an empty key
	var nextKey []byte
	if numRequestsInTopic <= cursor+limit {
		nextKey = nil
	} else {
		nextKey = make([]byte, 8)
		binary.BigEndian.PutUint64(nextKey, cursor+limit)
	}

	return requests, &types.SimpleCursorPaginationResponse{
		NextKey: nextKey,
	}, nil
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

// Reset the mapping entirely. Should be called at the end of every block
func (k *Keeper) ResetChurnReadyTopics(ctx context.Context) error {
	return k.churnReadyTopics.Remove(ctx)
}

// Set a topic as churn ready
func (k *Keeper) SetChurnReadyTopics(ctx context.Context, topicList types.TopicList) error {
	return k.churnReadyTopics.Set(ctx, topicList)
}

// Get all churn ready topics
// In practice, shouldn't be more than a max amount, enforced by other functionality
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

/// SCORES

// If the new score is older than the current score, don't update
func (k *Keeper) SetLatestInfererScore(ctx context.Context, topicId TopicId, worker Worker, score types.Score) error {
	oldScore, err := k.GetLatestInfererScore(ctx, topicId, worker)
	if err != nil {
		return err
	}
	if oldScore.BlockNumber >= score.BlockNumber {
		return nil
	}
	key := collections.Join(topicId, worker)
	return k.latestInfererScoresByWorker.Set(ctx, key, score)
}

func (k *Keeper) GetLatestInfererScore(ctx context.Context, topicId TopicId, worker Worker) (types.Score, error) {
	key := collections.Join(topicId, worker)
	score, err := k.latestInfererScoresByWorker.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{}, nil
		}
		return types.Score{}, err
	}
	return score, nil
}

// If the new score is older than the current score, don't update
func (k *Keeper) SetLatestForecasterScore(ctx context.Context, topicId TopicId, worker Worker, score types.Score) error {
	oldScore, err := k.GetLatestForecasterScore(ctx, topicId, worker)
	if err != nil {
		return err
	}
	if oldScore.BlockNumber >= score.BlockNumber {
		return nil
	}
	key := collections.Join(topicId, worker)
	return k.latestForecasterScoresByWorker.Set(ctx, key, score)
}

func (k *Keeper) GetLatestForecasterScore(ctx context.Context, topicId TopicId, worker Worker) (types.Score, error) {
	key := collections.Join(topicId, worker)
	score, err := k.latestForecasterScoresByWorker.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{}, nil
		}
		return types.Score{}, err
	}
	return score, nil
}

// If the new score is older than the current score, don't update
func (k *Keeper) SetLatestReputerScore(ctx context.Context, topicId TopicId, reputer Reputer, score types.Score) error {
	oldScore, err := k.GetLatestReputerScore(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if oldScore.BlockNumber >= score.BlockNumber {
		return nil
	}
	key := collections.Join(topicId, reputer)
	return k.latestReputerScoresByReputer.Set(ctx, key, score)
}

func (k *Keeper) GetLatestReputerScore(ctx context.Context, topicId TopicId, reputer Reputer) (types.Score, error) {
	key := collections.Join(topicId, reputer)
	score, err := k.latestReputerScoresByReputer.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{}, nil
		}
		return types.Score{}, err
	}
	return score, nil
}

func (k *Keeper) InsertWorkerInferenceScore(ctx context.Context, topicId TopicId, blockNumber BlockHeight, score types.Score) error {
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockNumber)
	if err != nil {
		return err
	}
	scores.Scores = append(scores.Scores, &score)

	maxNumScores, err := k.GetParamsMaxSamplesToScaleScores(ctx)
	if err != nil {
		return err
	}

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockNumber)
	return k.infererScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetWorkerInferenceScoresUntilBlock(ctx context.Context, topicId TopicId, blockNumber BlockHeight, worker Worker) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockNumber).
		Descending()

	scores := make([]*types.Score, 0)
	iter, err := k.infererScoresByBlock.Iterate(ctx, rng)
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

func (k *Keeper) GetWorkerInferenceScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.infererScoresByBlock.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Scores{}, nil
		}
		return types.Scores{}, err
	}
	return scores, nil
}

func (k *Keeper) InsertWorkerForecastScore(ctx context.Context, topicId TopicId, blockNumber BlockHeight, score types.Score) error {
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockNumber)
	if err != nil {
		return err
	}
	scores.Scores = append(scores.Scores, &score)

	maxNumScores, err := k.GetParamsMaxSamplesToScaleScores(ctx)
	if err != nil {
		return err
	}

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockNumber)
	return k.forecasterScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetWorkerForecastScoresUntilBlock(ctx context.Context, topicId TopicId, blockNumber BlockHeight, worker Worker) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockNumber).
		Descending()

	scores := make([]*types.Score, 0)
	iter, err := k.forecasterScoresByBlock.Iterate(ctx, rng)
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

func (k *Keeper) GetWorkerForecastScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.forecasterScoresByBlock.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Scores{}, nil
		}
		return types.Scores{}, err
	}
	return scores, nil
}

func (k *Keeper) InsertReputerScore(ctx context.Context, topicId TopicId, blockNumber BlockHeight, score types.Score) error {
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockNumber)
	if err != nil {
		return err
	}
	scores.Scores = append(scores.Scores, &score)

	maxNumScores, err := k.GetParamsMaxSamplesToScaleScores(ctx)
	if err != nil {
		return err
	}
	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		if diff > 0 {
			scores.Scores = scores.Scores[diff:]
		}
	}
	key := collections.Join(topicId, blockNumber)
	return k.reputerScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetReputersScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.reputerScoresByBlock.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Scores{}, nil
		}
		return types.Scores{}, err
	}
	return scores, nil
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
			return types.ListeningCoefficient{Coefficient: alloraMath.NewDecFromInt64(1)}, nil
		}
		return types.ListeningCoefficient{}, err
	}
	return coef, nil
}

/// REWARD FRACTION

// Gets the previous W_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousReputerRewardFraction(ctx context.Context, topicId TopicId, reputer sdk.AccAddress) (alloraMath.Dec, bool, error) {
	key := collections.Join(topicId, reputer)
	reward, err := k.previousReputerRewardFraction.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.Dec{}, false, err
	}
	return reward, false, nil
}

// Sets the previous W_{i-1,m}
func (k *Keeper) SetPreviousReputerRewardFraction(ctx context.Context, topicId TopicId, reputer sdk.AccAddress, reward alloraMath.Dec) error {
	key := collections.Join(topicId, reputer)
	return k.previousReputerRewardFraction.Set(ctx, key, reward)
}

// Gets the previous U_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker sdk.AccAddress) (alloraMath.Dec, bool, error) {
	key := collections.Join(topicId, worker)
	reward, err := k.previousInferenceRewardFraction.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.Dec{}, false, err
	}
	return reward, false, nil
}

// Sets the previous U_{i-1,m}
func (k *Keeper) SetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker sdk.AccAddress, reward alloraMath.Dec) error {
	key := collections.Join(topicId, worker)
	return k.previousInferenceRewardFraction.Set(ctx, key, reward)
}

// Gets the previous V_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker sdk.AccAddress) (alloraMath.Dec, bool, error) {
	key := collections.Join(topicId, worker)
	reward, err := k.previousForecastRewardFraction.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.Dec{}, false, err
	}
	return reward, false, nil
}

// Sets the previous V_{i-1,m}
func (k *Keeper) SetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker sdk.AccAddress, reward alloraMath.Dec) error {
	key := collections.Join(topicId, worker)
	return k.previousForecastRewardFraction.Set(ctx, key, reward)
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

/// UTILS

func (k *Keeper) Uint64Min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Convert pagination.key from []bytes to uint64, if pagination is nil or [], len = 0
// Get the limit from the pagination request, within acceptable bounds and defaulting as necessary
func (k Keeper) CalcAppropriatePaginationForUint64Cursor(ctx context.Context, pagination *types.SimpleCursorPaginationRequest) (uint64, uint64, error) {
	cursor := uint64(0)
	limit, err := k.GetParamsDefaultLimit(ctx)
	if err != nil {
		return uint64(0), uint64(0), err
	}
	if pagination != nil {
		if len(pagination.Key) > 0 {
			cursor = binary.BigEndian.Uint64(pagination.Key)
		}
		if pagination.Limit == 0 {
			return limit, cursor, nil
		} else {
			maxLimit, err := k.GetParamsMaxLimit(ctx)
			if err != nil {
				return uint64(0), uint64(0), err
			}
			if limit > maxLimit {
				limit = maxLimit
			}
			return limit, cursor, nil
		}
	}

	return limit, cursor, nil
}
