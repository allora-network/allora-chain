package keeper

import (
	"context"
	"encoding/binary"
	"errors"

	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/app/params"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"

	coreStore "cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TopicId = uint64
type LibP2pKey = string
type ActorId = string
type BlockHeight = int64
type Reputer = string
type Delegator = string

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
	// every topic that is ready to request inferences and possible also losses
	churnableTopics collections.KeySet[TopicId]
	// every topic that has been churned and ready to be rewarded i.e. reputer losses have been committed
	rewardableTopics collections.KeySet[TopicId]
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// map of (topic) -> nonce/block height
	topicRewardNonce collections.Map[TopicId, BlockHeight]

	/// SCORES

	// map of (topic, block_height, worker) -> score
	infererScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, worker) -> score
	forecasterScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, reputer) -> score
	reputerScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, worker) -> score
	latestInfererScoresByWorker collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, block_height, worker) -> score
	latestForecasterScoresByWorker collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, block_height, reputer) -> score
	latestReputerScoresByReputer collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, reputer) -> listening coefficient
	reputerListeningCoefficient collections.Map[collections.Pair[TopicId, ActorId], types.ListeningCoefficient]
	// map of (topic, reputer) -> previous reward (used for EMA)
	previousReputerRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for inference (used for EMA)
	previousInferenceRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for forecast (used for EMA)
	previousForecastRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]

	/// STAKING

	// total sum stake of all stakers on the network
	totalStake collections.Item[cosmosMath.Int]
	// for every topic, how much total stake does that topic have accumulated?
	topicStake collections.Map[TopicId, cosmosMath.Int]
	// stake reputer placed in topic + delegate stake placed in them,
	// signalling their total authority on the topic
	// (topic Id, reputer) -> stake from reputer on self + stakeFromDelegatorsUponReputer
	stakeReputerAuthority collections.Map[collections.Pair[TopicId, Reputer], cosmosMath.Int]
	// map of (topic id, delegator) -> total amount of stake in that topic placed by that delegator
	stakeSumFromDelegator collections.Map[collections.Pair[TopicId, Delegator], cosmosMath.Int]
	// map of (topic id, delegator, reputer) -> amount of stake that has been placed by that delegator on that target
	delegatedStakes collections.Map[collections.Triple[TopicId, Delegator, Reputer], types.DelegatorInfo]
	// map of (topic id, reputer) -> total amount of stake that has been placed on that reputer by delegators
	stakeFromDelegatorsUponReputer collections.Map[collections.Pair[TopicId, Reputer], cosmosMath.Int]
	// map of (topicId, reputer) -> share of delegate reward
	delegateRewardPerShare collections.Map[collections.Pair[TopicId, Reputer], alloraMath.Dec]
	// stake removals are double indexed to avoid O(n) lookups when removing stake
	// map of (blockHeight, topic, reputer) -> removal information for that reputer
	stakeRemovalsByBlock collections.Map[collections.Triple[BlockHeight, TopicId, Reputer], types.StakeRemovalInfo]
	// key set of (reputer, topic, blockHeight) to existence of a removal in the forwards map
	stakeRemovalsByActor collections.KeySet[collections.Triple[Reputer, TopicId, BlockHeight]]
	// delegate stake removals are double indexed to avoid O(n) lookups when removing stake
	// map of (blockHeight, topic, delegator, reputer staked upon) -> (list of reputers delegated upon and info) to have stake removed at that block
	delegateStakeRemovalsByBlock collections.Map[Quadruple[BlockHeight, TopicId, Delegator, Reputer], types.DelegateStakeRemovalInfo]
	// key set of (delegator, reputer, topicId, blockHeight) to existence of a removal in the forwards map
	delegateStakeRemovalsByActor collections.KeySet[Quadruple[Delegator, Reputer, TopicId, BlockHeight]]

	/// MISC GLOBAL STATE

	// map of (topic, worker) -> inference
	inferences collections.Map[collections.Pair[TopicId, ActorId], types.Inference]

	// map of (topic, worker) -> forecast[]
	forecasts collections.Map[collections.Pair[TopicId, ActorId], types.Forecast]

	// map of worker id to node data about that worker
	workers collections.Map[LibP2pKey, types.OffchainNode]

	// map of reputer id to node data about that reputer
	reputers collections.Map[LibP2pKey, types.OffchainNode]

	// fee revenue collected by a topic over the course of the last reward cadence
	topicFeeRevenue collections.Map[TopicId, cosmosMath.Int]

	// store previous weights for exponential moving average in rewards calc
	previousTopicWeight collections.Map[TopicId, alloraMath.Dec]

	// map of (topic, block_height) -> Inference
	allInferences collections.Map[collections.Pair[TopicId, BlockHeight], types.Inferences]

	// map of (topic, block_height) -> Forecast
	allForecasts collections.Map[collections.Pair[TopicId, BlockHeight], types.Forecasts]

	// map of (topic, block_height) -> ReputerValueBundles (1 per reputer active at that time)
	allLossBundles collections.Map[collections.Pair[TopicId, BlockHeight], types.ReputerValueBundles]

	// map of (topic, block_height) -> ValueBundle (1 network wide bundle per timestep)
	networkLossBundles collections.Map[collections.Pair[TopicId, BlockHeight], types.ValueBundle]

	// Percentage of all rewards, paid out to staked reputers, during the previous reward cadence. Used by mint module
	previousPercentageRewardToStakedReputers collections.Item[alloraMath.Dec]

	/// NONCES

	// map of (topic) -> unfulfilled nonces
	unfulfilledWorkerNonces collections.Map[TopicId, types.Nonces]

	// map of (topic) -> unfulfilled nonces
	unfulfilledReputerNonces collections.Map[TopicId, types.ReputerRequestNonces]

	/// REGRETS

	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other inferers
	latestInfererNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other forecasters
	latestForecasterNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic, forecaster, inferer) -> R^+_{ij_kk} regret of forecaster loss from comparing one-in loss with
	// all network inferer (3rd index) regrets L_ij made under the regime of the one-in forecaster (2nd index)
	latestOneInForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// the forecaster (2nd index) regrets made under the regime of the same forecaster as a one-in forecaster
	latestOneInForecasterSelfNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]

	/// WHITELISTS

	whitelistAdmins collections.KeySet[ActorId]

	/// RECORD COMMITS

	//
	topicLastWorkerCommit  collections.Map[TopicId, types.TimestampedActorNonce]
	topicLastReputerCommit collections.Map[TopicId, types.TimestampedActorNonce]
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
		cdc:                                      cdc,
		addressCodec:                             addressCodec,
		feeCollectorName:                         feeCollectorName,
		params:                                   collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		authKeeper:                               ak,
		bankKeeper:                               bk,
		totalStake:                               collections.NewItem(sb, types.TotalStakeKey, "total_stake", sdk.IntValue),
		topicStake:                               collections.NewMap(sb, types.TopicStakeKey, "topic_stake", collections.Uint64Key, sdk.IntValue),
		nextTopicId:                              collections.NewSequence(sb, types.NextTopicIdKey, "next_TopicId"),
		topics:                                   collections.NewMap(sb, types.TopicsKey, "topics", collections.Uint64Key, codec.CollValue[types.Topic](cdc)),
		activeTopics:                             collections.NewKeySet(sb, types.ActiveTopicsKey, "active_topics", collections.Uint64Key),
		churnableTopics:                          collections.NewKeySet(sb, types.ChurnableTopicsKey, "churnable_topics", collections.Uint64Key),
		rewardableTopics:                         collections.NewKeySet(sb, types.RewardableTopicsKey, "rewardable_topics", collections.Uint64Key),
		topicWorkers:                             collections.NewKeySet(sb, types.TopicWorkersKey, "topic_workers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		topicReputers:                            collections.NewKeySet(sb, types.TopicReputersKey, "topic_reputers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		stakeReputerAuthority:                    collections.NewMap(sb, types.StakeByReputerAndTopicIdKey, "stake_by_reputer_and_TopicId", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), sdk.IntValue),
		stakeRemovalsByBlock:                     collections.NewMap(sb, types.StakeRemovalsByBlockKey, "stake_removals_by_block", collections.TripleKeyCodec(collections.Int64Key, collections.Uint64Key, collections.StringKey), codec.CollValue[types.StakeRemovalInfo](cdc)),
		stakeRemovalsByActor:                     collections.NewKeySet(sb, types.StakeRemovalsByActorKey, "stake_removals_by_actor", collections.TripleKeyCodec(collections.StringKey, collections.Uint64Key, collections.Int64Key)),
		delegateStakeRemovalsByBlock:             collections.NewMap(sb, types.DelegateStakeRemovalsByBlockKey, "delegate_stake_removals_by_block", QuadrupleKeyCodec(collections.Int64Key, collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.DelegateStakeRemovalInfo](cdc)),
		delegateStakeRemovalsByActor:             collections.NewKeySet(sb, types.DelegateStakeRemovalsByActorKey, "delegate_stake_removals_by_actor", QuadrupleKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key, collections.Int64Key)),
		stakeSumFromDelegator:                    collections.NewMap(sb, types.DelegatorStakeKey, "stake_from_delegator", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), sdk.IntValue),
		delegatedStakes:                          collections.NewMap(sb, types.DelegateStakePlacementKey, "delegate_stake_placement", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.DelegatorInfo](cdc)),
		stakeFromDelegatorsUponReputer:           collections.NewMap(sb, types.TargetStakeKey, "stake_upon_reputer", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), sdk.IntValue),
		delegateRewardPerShare:                   collections.NewMap(sb, types.DelegateRewardPerShare, "delegate_reward_per_share", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		topicFeeRevenue:                          collections.NewMap(sb, types.TopicFeeRevenueKey, "topic_fee_revenue", collections.Uint64Key, sdk.IntValue),
		previousTopicWeight:                      collections.NewMap(sb, types.PreviousTopicWeightKey, "previous_topic_weight", collections.Uint64Key, alloraMath.DecValue),
		inferences:                               collections.NewMap(sb, types.InferencesKey, "inferences", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Inference](cdc)),
		forecasts:                                collections.NewMap(sb, types.ForecastsKey, "forecasts", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Forecast](cdc)),
		workers:                                  collections.NewMap(sb, types.WorkerNodesKey, "worker_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		reputers:                                 collections.NewMap(sb, types.ReputerNodesKey, "reputer_nodes", collections.StringKey, codec.CollValue[types.OffchainNode](cdc)),
		allInferences:                            collections.NewMap(sb, types.AllInferencesKey, "inferences_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Inferences](cdc)),
		allForecasts:                             collections.NewMap(sb, types.AllForecastsKey, "forecasts_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Forecasts](cdc)),
		allLossBundles:                           collections.NewMap(sb, types.AllLossBundlesKey, "value_bundles_all", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ReputerValueBundles](cdc)),
		networkLossBundles:                       collections.NewMap(sb, types.NetworkLossBundlesKey, "value_bundles_network", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ValueBundle](cdc)),
		previousPercentageRewardToStakedReputers: collections.NewItem(sb, types.PreviousPercentageRewardToStakedReputersKey, "previous_percentage_reward_to_staked_reputers", alloraMath.DecValue),
		latestInfererNetworkRegrets:              collections.NewMap(sb, types.InfererNetworkRegretsKey, "inferer_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestForecasterNetworkRegrets:           collections.NewMap(sb, types.ForecasterNetworkRegretsKey, "forecaster_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneInForecasterNetworkRegrets:      collections.NewMap(sb, types.OneInForecasterNetworkRegretsKey, "one_in_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneInForecasterSelfNetworkRegrets:  collections.NewMap(sb, types.OneInForecasterSelfNetworkRegretsKey, "one_in_forecaster_self_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		whitelistAdmins:                          collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", collections.StringKey),
		infererScoresByBlock:                     collections.NewMap(sb, types.InferenceScoresKey, "inferer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		forecasterScoresByBlock:                  collections.NewMap(sb, types.ForecastScoresKey, "forecaster_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		latestInfererScoresByWorker:              collections.NewMap(sb, types.LatestInfererScoresByWorkerKey, "latest_inferer_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		latestForecasterScoresByWorker:           collections.NewMap(sb, types.LatestForecasterScoresByWorkerKey, "latest_forecaster_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		latestReputerScoresByReputer:             collections.NewMap(sb, types.LatestReputerScoresByReputerKey, "latest_reputer_scores_by_reputer", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		previousReputerRewardFraction:            collections.NewMap(sb, types.PreviousReputerRewardFractionKey, "previous_reputer_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousInferenceRewardFraction:          collections.NewMap(sb, types.PreviousInferenceRewardFractionKey, "previous_inference_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecastRewardFraction:           collections.NewMap(sb, types.PreviousForecastRewardFractionKey, "previous_forecast_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		reputerScoresByBlock:                     collections.NewMap(sb, types.ReputerScoresKey, "reputer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerListeningCoefficient:              collections.NewMap(sb, types.ReputerListeningCoefficientKey, "reputer_listening_coefficient", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.ListeningCoefficient](cdc)),
		unfulfilledWorkerNonces:                  collections.NewMap(sb, types.UnfulfilledWorkerNoncesKey, "unfulfilled_worker_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
		unfulfilledReputerNonces:                 collections.NewMap(sb, types.UnfulfilledReputerNoncesKey, "unfulfilled_reputer_nonces", collections.Uint64Key, codec.CollValue[types.ReputerRequestNonces](cdc)),
		topicRewardNonce:                         collections.NewMap(sb, types.TopicRewardNonceKey, "topic_reward_nonce", collections.Uint64Key, collections.Int64Value),
		topicLastWorkerCommit:                    collections.NewMap(sb, types.TopicLastWorkerCommitKey, "topic_last_worker_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		topicLastReputerCommit:                   collections.NewMap(sb, types.TopicLastReputerCommitKey, "topic_last_reputer_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
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

	// Check if the nonce is present in the unfulfilled nonces
	for _, n := range unfulfilledNonces.Nonces {
		if n == nil {
			continue
		}
		if n.BlockHeight == nonce.BlockHeight {
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
		if n == nil {
			continue
		}
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
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

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	maxUnfulfilledRequests := moduleParams.MaxUnfulfilledWorkerRequests

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
	if nonce == nil {
		return errors.New("nil reputer's nonce provided")
	}
	if associatedWorkerNonce == nil {
		return errors.New("nil reputer's worker nonce provided")
	}

	// Check that input nonce is not already contained in the nonces of this topic
	// nor that the `associatedWorkerNonce` is already associated with a worker requeset
	for _, n := range nonces.Nonces {
		// Do nothing if nonce is already in the list
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			return nil
		}
	}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: nonce,
		WorkerNonce:  associatedWorkerNonce,
	}
	nonces.Nonces = append([]*types.ReputerRequestNonce{reputerRequestNonce}, nonces.Nonces...)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	maxUnfulfilledRequests := moduleParams.MaxUnfulfilledReputerRequests
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

func (k *Keeper) DeleteUnfulfilledWorkerNonces(ctx context.Context, topicId TopicId) error {
	return k.unfulfilledWorkerNonces.Set(ctx, topicId, types.Nonces{})
}

func (k *Keeper) DeleteUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) error {
	return k.unfulfilledReputerNonces.Set(ctx, topicId, types.ReputerRequestNonces{})
}

/// REGRETS

func (k *Keeper) SetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker ActorId, regret types.TimestampedValue) error {
	key := collections.Join(topicId, worker)
	return k.latestInfererNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) SetForecasterNetworkRegret(ctx context.Context, topicId TopicId, worker ActorId, regret types.TimestampedValue) error {
	key := collections.Join(topicId, worker)
	return k.latestForecasterNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) SetOneInForecasterNetworkRegret(ctx context.Context, topicId TopicId, forecaster ActorId, inferer ActorId, regret types.TimestampedValue) error {
	key := collections.Join3(topicId, forecaster, inferer)
	return k.latestOneInForecasterNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) SetOneInForecasterSelfNetworkRegret(ctx context.Context, topicId TopicId, forecaster ActorId, regret types.TimestampedValue) error {
	key := collections.Join(topicId, forecaster)
	return k.latestOneInForecasterSelfNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a worker from comparing loss of worker relative to loss of other inferers
// Returns (0, true) if no regret is found
func (k *Keeper) GetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker ActorId) (types.TimestampedValue, bool, error) {
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
func (k *Keeper) GetForecasterNetworkRegret(ctx context.Context, topicId TopicId, worker ActorId) (types.TimestampedValue, bool, error) {
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
func (k *Keeper) GetOneInForecasterNetworkRegret(ctx context.Context, topicId TopicId, forecaster ActorId, inferer ActorId) (types.TimestampedValue, bool, error) {
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

func (k *Keeper) GetOneInForecasterSelfNetworkRegret(ctx context.Context, topicId TopicId, forecaster ActorId) (types.TimestampedValue, bool, error) {
	key := collections.Join(topicId, forecaster)
	regret, err := k.latestOneInForecasterSelfNetworkRegrets.Get(ctx, key)
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

func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	ret, err := k.params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.DefaultParams(), nil
		}
		return types.Params{}, err
	}
	return ret, nil
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

// GetLatestTopicInferences retrieves the latest topic inferences and its block height.
func (k *Keeper) GetLatestTopicInferences(ctx context.Context, topicId TopicId) (*types.Inferences, BlockHeight, error) {
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).Descending()

	iter, err := k.allInferences.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	if iter.Valid() {
		keyValue, err := iter.KeyValue()
		if err != nil {
			return nil, 0, err
		}
		return &keyValue.Value, keyValue.Key.K2(), nil
	}

	return &types.Inferences{
		Inferences: make([]*types.Inference, 0),
	}, 0, nil
}

func (k *Keeper) GetForecastsAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Forecasts, error) {
	key := collections.Join(topicId, block)
	forecasts, err := k.allForecasts.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &forecasts, nil
}

// Insert a complete set of inferences for a topic/block. Overwrites previous ones.
func (k *Keeper) InsertInferences(ctx context.Context, topicId TopicId, nonce types.Nonce, inferences types.Inferences) error {
	block := nonce.BlockHeight

	for _, inference := range inferences.Inferences {
		// Update latests inferences for each worker
		key := collections.Join(topicId, inference.Inferer)
		if err := k.inferences.Set(ctx, key, *inference); err != nil {
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
		key := collections.Join(topicId, forecast.Forecaster)
		if err := k.forecasts.Set(ctx, key, *forecast); err != nil {
			return err
		}
	}

	key := collections.Join(topicId, block)
	return k.allForecasts.Set(ctx, key, forecasts)
}

func (k *Keeper) GetWorkerLatestInferenceByTopicId(
	ctx context.Context,
	topicId TopicId,
	worker ActorId) (types.Inference, error) {
	key := collections.Join(topicId, worker)
	return k.inferences.Get(ctx, key)
}

/// TOPIC REWARD NONCE

// GetTopicRewardNonce retrieves the reward nonce for a given topic ID.
func (k *Keeper) GetTopicRewardNonce(ctx context.Context, topicId TopicId) (BlockHeight, error) {
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
func (k *Keeper) SetTopicRewardNonce(ctx context.Context, topicId TopicId, nonce BlockHeight) error {
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

/// STAKING

// Adds stake to the system for a given topic and reputer
// Adds to: totalStake, topicStake, stakeReputerAuthority,
func (k *Keeper) AddReputerStake(
	ctx context.Context,
	topicId TopicId,
	reputer ActorId,
	stakeToAdd cosmosMath.Int,
) error {
	// CHECKS
	if stakeToAdd.IsZero() {
		return errorsmod.Wrapf(types.ErrInvalidValue, "reputer stake to add must be greater than zero")
	}
	// GET CURRENT VALUES
	reputerAuthority, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	reputerAuthorityNew := reputerAuthority.Add(stakeToAdd)
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return err
	}
	topicStakeNew := topicStake.Add(stakeToAdd)
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return err
	}
	totalStakeNew := totalStake.Add(stakeToAdd)

	// SET NEW VALUES
	if err := k.SetStakeReputerAuthority(ctx, topicId, reputer, reputerAuthorityNew); err != nil {
		return err
	}
	if err := k.SetTopicStake(ctx, topicId, topicStakeNew); err != nil {
		return errorsmod.Wrapf(err, "Setting topic stake failed -- rolling back reputer stake")
	}
	if err := k.SetTotalStake(ctx, totalStakeNew); err != nil {
		return errorsmod.Wrapf(err, "Setting total stake failed -- rolling back reputer and topic stake")
	}
	return nil
}

// adds stake to the system from a delegator
// adds to: totalStake, topicStake, stakeReputerAuthority,
//
//	stakeSumFromDelegator, delegatedStakes, stakeFromDelegatorsUponReputer
func (k *Keeper) AddDelegateStake(
	ctx context.Context,
	topicId TopicId,
	delegator ActorId,
	reputer ActorId,
	stakeToAdd cosmosMath.Int,
) error {
	// CHECKS
	if stakeToAdd.IsZero() {
		return errorsmod.Wrapf(types.ErrInvalidValue, "delegator stake to add must be greater than zero")
	}

	// GET CURRENT VALUES
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return err
	}
	totalStakeNew := totalStake.Add(stakeToAdd)
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return err
	}
	topicStakeNew := topicStake.Add(stakeToAdd)
	stakeReputerAuthority, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	stakeReputerAuthorityNew := stakeReputerAuthority.Add(stakeToAdd)
	stakeSumFromDelegator, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegator)
	if err != nil {
		return err
	}
	stakeSumFromDelegatorNew := stakeSumFromDelegator.Add(stakeToAdd)
	delegateStakePlacement, err := k.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return err
	}
	share, err := k.GetDelegateRewardPerShare(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if delegateStakePlacement.Amount.Gt(alloraMath.NewDecFromInt64(0)) {
		// Calculate pending reward and send to delegator
		pendingReward, err := delegateStakePlacement.Amount.Mul(share)
		if err != nil {
			return err
		}
		pendingReward, err = pendingReward.Sub(delegateStakePlacement.RewardDebt)
		if err != nil {
			return err
		}
		if pendingReward.Gt(alloraMath.NewDecFromInt64(0)) {
			err = k.SendCoinsFromModuleToAccount(
				ctx,
				types.AlloraPendingRewardForDelegatorAccountName,
				delegator,
				sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, pendingReward.SdkIntTrim())),
			)
			if err != nil {
				return err
			}
		}
	}
	stakeToAddDec, err := alloraMath.NewDecFromSdkInt(stakeToAdd)
	if err != nil {
		return err
	}
	newAmount, err := delegateStakePlacement.Amount.Add(stakeToAddDec)
	if err != nil {
		return err
	}
	newDebt, err := newAmount.Mul(share)
	if err != nil {
		return err
	}
	stakePlacementNew := types.DelegatorInfo{
		Amount:     newAmount,
		RewardDebt: newDebt,
	}
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	stakeUponReputerNew := stakeUponReputer.Add(stakeToAdd)

	// UPDATE STATE AFTER CHECKS
	if err = k.SetTotalStake(ctx, totalStakeNew); err != nil {
		return errorsmod.Wrapf(err, "AddDelegateStake Setting total stake failed")
	}
	if err := k.SetTopicStake(ctx, topicId, topicStakeNew); err != nil {
		return errorsmod.Wrapf(err, "AddDelegateStake Setting topic stake failed")
	}
	if err := k.SetStakeReputerAuthority(ctx, topicId, reputer, stakeReputerAuthorityNew); err != nil {
		return errorsmod.Wrapf(err, "AddDelegateStake Setting reputer stake authority failed")
	}
	if err := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeSumFromDelegatorNew); err != nil {
		return errorsmod.Wrapf(err, "AddDelegateStake Setting stake sum from delegator failed")
	}
	if err := k.SetDelegateStakePlacement(ctx, topicId, delegator, reputer, stakePlacementNew); err != nil {
		return errorsmod.Wrapf(err, "AddDelegateStake Setting delegate stake placement failed")
	}
	if err := k.SetDelegateStakeUponReputer(ctx, topicId, reputer, stakeUponReputerNew); err != nil {
		return errorsmod.Wrapf(err, "AddDelegateStake Setting stake from delegators upon reputer failed")
	}
	return nil
}

// Removes stake from the system for a given topic and reputer
// subtracts from: totalStake, topicStake, stakeReputerAuthority
func (k *Keeper) RemoveReputerStake(
	ctx context.Context,
	blockHeight BlockHeight,
	topicId TopicId,
	reputer ActorId,
	stakeToRemove cosmosMath.Int) error {
	// CHECKS
	if stakeToRemove.IsZero() {
		return nil
	}
	// Check reputerAuthority >= stake
	reputerAuthority, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	delegateStakeUponReputerInTopic, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	reputerStakeInTopicWithoutDelegateStake := reputerAuthority.Sub(delegateStakeUponReputerInTopic)
	if stakeToRemove.GT(reputerStakeInTopicWithoutDelegateStake) {
		return types.ErrIntegerUnderflowTopicReputerStake
	}
	reputerStakeNew := reputerAuthority.Sub(stakeToRemove)

	// Check topicStake >= stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(topicStake) {
		return types.ErrIntegerUnderflowTopicStake
	}
	topicStakeNew := topicStake.Sub(stakeToRemove)

	// Check totalStake >= stake
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(totalStake) {
		return types.ErrIntegerUnderflowTotalStake
	}

	// Set topic-reputer stake
	if err := k.SetStakeReputerAuthority(ctx, topicId, reputer, reputerStakeNew); err != nil {
		return errorsmod.Wrapf(err, "Setting removed reputer stake in topic failed")
	}

	// Set topic stake
	if err := k.SetTopicStake(ctx, topicId, topicStakeNew); err != nil {
		return errorsmod.Wrapf(err, "Setting removed topic stake failed")
	}

	// Set total stake
	err = k.SetTotalStake(ctx, totalStake.Sub(stakeToRemove))
	if err != nil {
		return errorsmod.Wrapf(err, "Setting total stake failed")
	}

	// remove stake withdrawal information
	err = k.DeleteStakeRemoval(ctx, blockHeight, topicId, reputer)
	if err != nil {
		return errorsmod.Wrapf(err, "Deleting stake removal from queue failed")
	}

	return nil
}

// Removes delegate stake from the system for a given topic, delegator, and reputer
// subtracts from: totalStake, topicStake, stakeReputerAuthority
//
//	stakeSumFromDelegator, delegatedStakes, stakeFromDelegatorsUponReputer
func (k *Keeper) RemoveDelegateStake(
	ctx context.Context,
	blockHeight BlockHeight,
	topicId TopicId,
	delegator ActorId,
	reputer ActorId,
	stakeToRemove cosmosMath.Int,
) error {
	// CHECKS
	if stakeToRemove.IsZero() {
		return nil
	}

	// stakeSumFromDelegator >= stake
	stakeSumFromDelegator, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegator)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(stakeSumFromDelegator) {
		return types.ErrIntegerUnderflowStakeFromDelegator
	}
	stakeFromDelegatorNew := stakeSumFromDelegator.Sub(stakeToRemove)

	// delegatedStakePlacement >= stake
	delegatedStakePlacement, err := k.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return err
	}
	unStakeDec, err := alloraMath.NewDecFromSdkInt(stakeToRemove)
	if err != nil {
		return err
	}
	if delegatedStakePlacement.Amount.Lt(unStakeDec) {
		return types.ErrIntegerUnderflowDelegateStakePlacement
	}

	// Get share for this topicId and reputer
	share, err := k.GetDelegateRewardPerShare(ctx, topicId, reputer)
	if err != nil {
		return err
	}

	// Calculate pending reward and send to delegator
	pendingReward, err := delegatedStakePlacement.Amount.Mul(share)
	if err != nil {
		return err
	}
	pendingReward, err = pendingReward.Sub(delegatedStakePlacement.RewardDebt)
	if err != nil {
		return err
	}
	if pendingReward.Gt(alloraMath.NewDecFromInt64(0)) {
		err = k.SendCoinsFromModuleToAccount(
			ctx,
			types.AlloraPendingRewardForDelegatorAccountName,
			delegator,
			sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, pendingReward.SdkIntTrim())),
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Sending pending reward to delegator failed")
		}
	}

	newAmount, err := delegatedStakePlacement.Amount.Sub(unStakeDec)
	if err != nil {
		return err
	}
	newRewardDebt, err := newAmount.Mul(share)
	if err != nil {
		return err
	}
	stakePlacementNew := types.DelegatorInfo{
		Amount:     newAmount,
		RewardDebt: newRewardDebt,
	}

	// stakeUponReputer >= stake
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(stakeUponReputer) {
		return types.ErrIntegerUnderflowDelegateStakeUponReputer
	}
	stakeUponReputerNew := stakeUponReputer.Sub(stakeToRemove)

	// stakeReputerAuthority >= stake
	stakeReputerAuthority, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(stakeReputerAuthority) {
		return types.ErrIntegerUnderflowReputerStakeAuthority
	}
	stakeReputerAuthorityNew := stakeReputerAuthority.Sub(stakeToRemove)

	// topicStake >= stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(topicStake) {
		return types.ErrIntegerUnderflowTopicStake
	}
	topicStakeNew := topicStake.Sub(stakeToRemove)

	// totalStake >= stake
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return err
	}
	if stakeToRemove.GT(totalStake) {
		return types.ErrIntegerUnderflowTotalStake
	}
	totalStakeNew := totalStake.Sub(stakeToRemove)

	// SET NEW VALUES AFTER CHECKS

	if err := k.SetStakeFromDelegator(ctx, topicId, delegator, stakeFromDelegatorNew); err != nil {
		return errorsmod.Wrapf(err, "Setting stake from delegator failed")
	}
	if err := k.SetDelegateStakePlacement(ctx, topicId, delegator, reputer, stakePlacementNew); err != nil {
		return errorsmod.Wrapf(err, "Setting delegate stake placement failed")
	}
	if err := k.SetDelegateStakeUponReputer(ctx, topicId, reputer, stakeUponReputerNew); err != nil {
		return errorsmod.Wrapf(err, "Setting delegate stake upon reputer failed")
	}
	if err := k.SetStakeReputerAuthority(ctx, topicId, reputer, stakeReputerAuthorityNew); err != nil {
		return errorsmod.Wrapf(err, "Setting reputer stake authority failed")
	}
	if err := k.SetTopicStake(ctx, topicId, topicStakeNew); err != nil {
		return errorsmod.Wrapf(err, "Setting topic stake failed")
	}
	if err := k.SetTotalStake(ctx, totalStakeNew); err != nil {
		return errorsmod.Wrapf(err, "Setting total stake failed")
	}
	if err := k.DeleteDelegateStakeRemoval(ctx, blockHeight, topicId, reputer, delegator); err != nil {
		return errorsmod.Wrapf(err, "Deleting delegate stake removal from queue failed")
	}

	return nil
}

// Gets the total sum of all stake in the network across all topics
func (k Keeper) GetTotalStake(ctx context.Context) (cosmosMath.Int, error) {
	ret, err := k.totalStake.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewInt(0), nil
		}
		return cosmosMath.Int{}, err
	}
	return ret, nil
}

// Sets the total sum of all stake in the network across all topics
func (k *Keeper) SetTotalStake(ctx context.Context, totalStake cosmosMath.Int) error {
	// total stake does not have a zero guard because totalStake is allowed to be zero
	// it is initialized to zero at genesis anyways.
	return k.totalStake.Set(ctx, totalStake)
}

// Gets the stake in the network for a given topic
func (k *Keeper) GetTopicStake(ctx context.Context, topicId TopicId) (cosmosMath.Int, error) {
	ret, err := k.topicStake.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewInt(0), nil
		}
		return cosmosMath.Int{}, err
	}
	return ret, nil
}

// sets the cumulative amount of stake in a topic
func (k *Keeper) SetTopicStake(ctx context.Context, topicId TopicId, stake cosmosMath.Int) error {
	if stake.IsZero() {
		return k.topicStake.Remove(ctx, topicId)
	}
	return k.topicStake.Set(ctx, topicId, stake)
}

// Returns the amount of stake placed by a specific reputer on a specific topic.
// Includes the stake placed by delegators on the reputer in that topic.
func (k *Keeper) GetStakeReputerAuthority(ctx context.Context, topicId TopicId, reputer ActorId) (cosmosMath.Int, error) {
	key := collections.Join(topicId, reputer)
	stake, err := k.stakeReputerAuthority.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewInt(0), nil
		}
		return cosmosMath.Int{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed upon a reputer in addition to their personal stake on a specific topic
// Includes the stake placed by delegators on the reputer in that topic.
func (k *Keeper) SetStakeReputerAuthority(ctx context.Context, topicId TopicId, reputer ActorId, amount cosmosMath.Int) error {
	key := collections.Join(topicId, reputer)
	if amount.IsZero() {
		return k.stakeReputerAuthority.Remove(ctx, key)
	}
	return k.stakeReputerAuthority.Set(ctx, key, amount)
}

// Returns the amount of stake placed by a specific delegator.
func (k *Keeper) GetStakeFromDelegatorInTopic(ctx context.Context, topicId TopicId, delegator ActorId) (cosmosMath.Int, error) {
	key := collections.Join(topicId, delegator)
	stake, err := k.stakeSumFromDelegator.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewInt(0), nil
		}
		return cosmosMath.Int{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator.
func (k *Keeper) SetStakeFromDelegator(ctx context.Context, topicId TopicId, delegator ActorId, stake cosmosMath.Int) error {
	key := collections.Join(topicId, delegator)
	if stake.IsZero() {
		return k.stakeSumFromDelegator.Remove(ctx, key)
	}
	return k.stakeSumFromDelegator.Set(ctx, key, stake)
}

// Returns the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) GetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator ActorId, target ActorId) (types.DelegatorInfo, error) {
	key := collections.Join3(topicId, delegator, target)
	stake, err := k.delegatedStakes.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.DelegatorInfo{Amount: alloraMath.NewDecFromInt64(0), RewardDebt: alloraMath.NewDecFromInt64(0)}, nil
		}
		return types.DelegatorInfo{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) SetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator ActorId, target ActorId, stake types.DelegatorInfo) error {
	key := collections.Join3(topicId, delegator, target)
	if stake.Amount.IsZero() {
		return k.delegatedStakes.Remove(ctx, key)
	}
	return k.delegatedStakes.Set(ctx, key, stake)
}

// Returns the share of reward by a specific topic and reputer
func (k *Keeper) GetDelegateRewardPerShare(ctx context.Context, topicId TopicId, reputer ActorId) (alloraMath.Dec, error) {
	key := collections.Join(topicId, reputer)
	share, err := k.delegateRewardPerShare.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.NewDecFromInt64(0), nil
		}
		return alloraMath.Dec{}, err
	}
	return share, nil
}

// Set the share on specific reputer and topicId
func (k *Keeper) SetDelegateRewardPerShare(ctx context.Context, topicId TopicId, reputer ActorId, share alloraMath.Dec) error {
	key := collections.Join(topicId, reputer)
	return k.delegateRewardPerShare.Set(ctx, key, share)
}

// Returns the amount of stake placed upon a reputer by delegators within that topic
func (k *Keeper) GetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target ActorId) (cosmosMath.Int, error) {
	key := collections.Join(topicId, target)
	stake, err := k.stakeFromDelegatorsUponReputer.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.NewInt(0), nil
		}
		return cosmosMath.Int{}, err
	}
	return stake, nil
}

// Sets the amount of stake placed on a specific target.
func (k *Keeper) SetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target ActorId, stake cosmosMath.Int) error {
	key := collections.Join(topicId, target)
	if stake.IsZero() {
		return k.stakeFromDelegatorsUponReputer.Remove(ctx, key)
	}
	return k.stakeFromDelegatorsUponReputer.Set(ctx, key, stake)
}

// For a given address, adds their stake removal information to the removal queue for delay waiting
// The topic used will be the topic set in the `removalInfo`
// This completely overrides the existing stake removal
func (k *Keeper) SetStakeRemoval(ctx context.Context, removalInfo types.StakeRemovalInfo) error {
	byBlockKey := collections.Join3(removalInfo.BlockRemovalCompleted, removalInfo.TopicId, removalInfo.Reputer)
	err := k.stakeRemovalsByBlock.Set(ctx, byBlockKey, removalInfo)
	if err != nil {
		return err
	}
	byActorKey := collections.Join3(removalInfo.Reputer, removalInfo.TopicId, removalInfo.BlockRemovalCompleted)
	return k.stakeRemovalsByActor.Set(ctx, byActorKey)
}

// remove a stake removal from the queue
func (k *Keeper) DeleteStakeRemoval(
	ctx context.Context,
	blockHeight BlockHeight,
	topicId TopicId,
	address ActorId,
) error {
	byBlockKey := collections.Join3(blockHeight, topicId, address)
	has, err := k.stakeRemovalsByBlock.Has(ctx, byBlockKey)
	if err != nil {
		return err
	}
	if !has {
		return types.ErrStakeRemovalNotFound
	}
	err = k.stakeRemovalsByBlock.Remove(ctx, byBlockKey)
	if err != nil {
		return err
	}
	byActorKey := collections.Join3(address, topicId, blockHeight)
	return k.stakeRemovalsByActor.Remove(ctx, byActorKey)
}

// get info about a removal
func (k Keeper) GetStakeRemoval(
	ctx context.Context,
	BlockHeight int64,
	topicId TopicId,
	reputer ActorId,
) (types.StakeRemovalInfo, error) {
	return k.stakeRemovalsByBlock.Get(ctx, collections.Join3(BlockHeight, topicId, reputer))
}

// get a list of stake removals for this block
func (k *Keeper) GetStakeRemovalsForBlock(
	ctx context.Context,
	blockHeight BlockHeight,
) ([]types.StakeRemovalInfo, error) {
	ret := make([]types.StakeRemovalInfo, 0)
	rng := collections.NewPrefixedTripleRange[BlockHeight, TopicId, ActorId](blockHeight)
	iter, err := k.stakeRemovalsByBlock.Iterate(ctx, rng)
	if err != nil {
		return ret, err
	}
	for ; iter.Valid(); iter.Next() {
		val, err := iter.Value()
		if err != nil {
			return ret, err
		}
		ret = append(ret, val)
	}
	return ret, nil
}

// get the first found stake removal for a reputer and topicId or err not found if not found
func (k *Keeper) GetStakeRemovalForReputerAndTopicId(
	ctx sdk.Context,
	reputer string,
	topicId uint64,
) (removal types.StakeRemovalInfo, found bool, err error) {
	rng := collections.NewSuperPrefixedTripleRange[ActorId, TopicId, BlockHeight](reputer, topicId)
	iter, err := k.stakeRemovalsByActor.Iterate(ctx, rng)
	if err != nil {
		return types.StakeRemovalInfo{}, false, err
	}
	keys, err := iter.Keys()
	if err != nil {
		return types.StakeRemovalInfo{}, false, err
	}
	keysLen := len(keys)
	if keysLen == 0 {
		return types.StakeRemovalInfo{}, false, nil
	}
	if keysLen < 0 {
		return types.StakeRemovalInfo{}, false, errorsmod.Wrapf(types.ErrInvariantFailure, "Why is golang len function returning negative values?")
	}
	key := keys[0]
	byBlockKey := collections.Join3(key.K3(), topicId, reputer)
	ret, err := k.stakeRemovalsByBlock.Get(ctx, byBlockKey)
	if err != nil {
		return types.StakeRemovalInfo{}, false, err
	}
	if keysLen > 1 {
		ctx.Logger().Warn("Invariant failure! More than one stake removal found for reputer and topicId")
		return ret, true, errorsmod.Wrapf(types.ErrInvariantFailure, "More than one stake removal found for reputer and topicId")
	}
	return ret, true, nil
}

// For a given address, adds their stake removal information to the removal queue for delay waiting
// The topic used will be the topic set in the `removalInfo`
// This completely overrides the existing stake removal
func (k *Keeper) SetDelegateStakeRemoval(ctx context.Context, removalInfo types.DelegateStakeRemovalInfo) error {
	byBlockKey := Join4(removalInfo.BlockRemovalCompleted, removalInfo.TopicId, removalInfo.Delegator, removalInfo.Reputer)
	err := k.delegateStakeRemovalsByBlock.Set(ctx, byBlockKey, removalInfo)
	if err != nil {
		return err
	}
	byActorKey := Join4(removalInfo.Delegator, removalInfo.Reputer, removalInfo.TopicId, removalInfo.BlockRemovalCompleted)
	return k.delegateStakeRemovalsByActor.Set(ctx, byActorKey)
}

// remove a stake removal from the queue
func (k *Keeper) DeleteDelegateStakeRemoval(
	ctx context.Context,
	blockHeight BlockHeight,
	topicId TopicId,
	reputer ActorId,
	delegator ActorId,
) error {
	byBlockKey := Join4(blockHeight, topicId, delegator, reputer)
	has, err := k.delegateStakeRemovalsByBlock.Has(ctx, byBlockKey)
	if err != nil {
		return err
	}
	if !has {
		return types.ErrStakeRemovalNotFound
	}
	err = k.delegateStakeRemovalsByBlock.Remove(ctx, byBlockKey)
	if err != nil {
		return err
	}
	byActorKey := Join4(delegator, reputer, topicId, blockHeight)
	return k.delegateStakeRemovalsByActor.Remove(ctx, byActorKey)
}

// get info about a removal
func (k Keeper) GetDelegateStakeRemoval(
	ctx context.Context,
	blockHeight BlockHeight,
	topicId TopicId,
	delegator ActorId,
	reputer ActorId,
) (types.DelegateStakeRemovalInfo, error) {
	return k.delegateStakeRemovalsByBlock.Get(ctx, Join4(blockHeight, topicId, delegator, reputer))
}

// get a list of stake removals for this block
func (k *Keeper) GetDelegateStakeRemovalsForBlock(
	ctx context.Context,
	blockHeight BlockHeight,
) ([]types.DelegateStakeRemovalInfo, error) {
	ret := make([]types.DelegateStakeRemovalInfo, 0)
	rng := NewSinglePrefixedQuadrupleRange[BlockHeight, TopicId, ActorId, ActorId](blockHeight)
	iter, err := k.delegateStakeRemovalsByBlock.Iterate(ctx, rng)
	if err != nil {
		return ret, err
	}
	for ; iter.Valid(); iter.Next() {
		val, err := iter.Value()
		if err != nil {
			return ret, err
		}
		ret = append(ret, val)
	}
	return ret, nil
}

// return the first found stake removal object for a delegator, reputer, and topicId
func (k *Keeper) GetDelegateStakeRemovalForDelegatorReputerAndTopicId(
	ctx sdk.Context,
	delegator string,
	reputer string,
	topicId uint64,
) (removal types.DelegateStakeRemovalInfo, found bool, err error) {
	rng := NewTriplePrefixedQuadrupleRange[ActorId, ActorId, TopicId, BlockHeight](delegator, reputer, topicId)
	iter, err := k.delegateStakeRemovalsByActor.Iterate(ctx, rng)
	if err != nil {
		return types.DelegateStakeRemovalInfo{}, false, err
	}
	keys, err := iter.Keys()
	if err != nil {
		return types.DelegateStakeRemovalInfo{}, false, err
	}
	keysLen := len(keys)
	if keysLen == 0 {
		return types.DelegateStakeRemovalInfo{}, false, nil
	}
	if keysLen < 0 {
		return types.DelegateStakeRemovalInfo{}, false, errorsmod.Wrapf(types.ErrInvariantFailure, "Why is golang len function returning negative values?")
	}
	key := keys[0]
	byBlockKey := Join4(key.K4(), topicId, delegator, reputer)
	ret, err := k.delegateStakeRemovalsByBlock.Get(ctx, byBlockKey)
	if err != nil {
		return types.DelegateStakeRemovalInfo{}, false, err
	}
	if keysLen > 1 {
		ctx.Logger().Warn("Invariant failure! More than one delegate stake removal found for delegator, reputer and topicId")
		return ret, true, errorsmod.Wrapf(types.ErrInvariantFailure, "More than one delegate stake removal found for delegator, reputer and topicId")
	}
	return ret, true, nil
}

/// REPUTERS

// Adds a new reputer to the reputer tracking data structures, reputers and topicReputers
func (k *Keeper) InsertReputer(ctx context.Context, topicId TopicId, reputer ActorId, reputerInfo types.OffchainNode) error {
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
func (k *Keeper) RemoveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	topicKey := collections.Join(topicId, reputer)
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
func (k *Keeper) InsertWorker(ctx context.Context, topicId TopicId, worker ActorId, workerInfo types.OffchainNode) error {
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
func (k *Keeper) RemoveWorker(ctx context.Context, topicId TopicId, worker ActorId) error {
	topicKey := collections.Join(topicId, worker)
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
func (k *Keeper) GetPreviousTopicWeight(ctx context.Context, topicId TopicId) (alloraMath.Dec, bool, error) {
	topicWeight, err := k.previousTopicWeight.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.ZeroDec(), false, err
	}
	return topicWeight, false, err
}

// Set the previous weight during rewards calculation for a topic
func (k *Keeper) SetPreviousTopicWeight(ctx context.Context, topicId TopicId, weight alloraMath.Dec) error {
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

func (k Keeper) GetIdsOfActiveTopics(ctx context.Context, pagination *types.SimpleCursorPaginationRequest) ([]TopicId, *types.SimpleCursorPaginationResponse, error) {
	limit, start, err := k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	if err != nil {
		return nil, nil, err
	}
	rng := new(collections.Range[uint64]).StartExclusive(start)

	iter, err := k.activeTopics.Iterate(ctx, rng)
	if err != nil {
		return nil, nil, err
	}
	defer iter.Close()

	activeTopicIds := make([]TopicId, 0)
	nextKey := make([]byte, binary.MaxVarintLen64)
	nextTopicId := uint64(0)
	for ; iter.Valid(); iter.Next() {
		topicId, err := iter.Key()
		if err != nil {
			return nil, nil, err
		}

		if uint64(len(activeTopicIds)) >= limit {
			break
		}

		activeTopicIds = append(activeTopicIds, topicId)
		nextTopicId = topicId
	}
	binary.BigEndian.PutUint64(nextKey, nextTopicId)

	// If there are no topics, we return the nil for next key
	if len(activeTopicIds) == 0 {
		nextKey = make([]byte, 0)
	}

	return activeTopicIds, &types.SimpleCursorPaginationResponse{
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
func (k *Keeper) IsWorkerRegisteredInTopic(ctx context.Context, topicId TopicId, worker ActorId) (bool, error) {
	topickey := collections.Join(topicId, worker)
	return k.topicWorkers.Has(ctx, topickey)
}

// True if reputer is registered in topic, else False
func (k *Keeper) IsReputerRegisteredInTopic(ctx context.Context, topicId TopicId, reputer ActorId) (bool, error) {
	topickey := collections.Join(topicId, reputer)
	return k.topicReputers.Has(ctx, topickey)
}

/// TOPIC FEE REVENUE

// Get the amount of fee revenue collected by a topic
func (k *Keeper) GetTopicFeeRevenue(ctx context.Context, topicId TopicId) (cosmosMath.Int, error) {
	feeRev, err := k.topicFeeRevenue.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.ZeroInt(), nil
		}
		return cosmosMath.ZeroInt(), err
	}
	return feeRev, nil
}

// Add to the fee revenue collected by a topic
func (k *Keeper) AddTopicFeeRevenue(ctx context.Context, topicId TopicId, amount cosmosMath.Int) error {
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return err
	}
	topicFeeRevenue = topicFeeRevenue.Add(amount)
	return k.topicFeeRevenue.Set(ctx, topicId, topicFeeRevenue)
}

// Drop the fee revenue by the global Ecosystem bucket drip amount
func (k *Keeper) DripTopicFeeRevenue(ctx context.Context, topicId TopicId, block BlockHeight) error {
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return err
	}

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	minEffectiveTopicRevenue := moduleParams.MinEffectiveTopicRevenue
	topicFeeRevenueDecayRate := moduleParams.TopicFeeRevenueDecayRate

	topicFeeRevenueDec, err := alloraMath.NewDecFromSdkInt(topicFeeRevenue)
	if err != nil {
		return err
	}

	newTopicFeeRevenue := cosmosMath.ZeroInt()
	if topicFeeRevenueDec.Gte(minEffectiveTopicRevenue) {
		val, err := alloraMath.CalcExpDecay(topicFeeRevenueDec, topicFeeRevenueDecayRate)
		if err != nil {
			return err
		}
		newTopicFeeRevenue = val.SdkIntTrim()
	}
	return k.topicFeeRevenue.Set(ctx, topicId, newTopicFeeRevenue)
}

/// CHURNABLE TOPICS

// Get the churnable topics
func (k *Keeper) GetChurnableTopics(ctx context.Context) ([]TopicId, error) {
	iter, err := k.churnableTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	topics := make([]TopicId, 0)
	for ; iter.Valid(); iter.Next() {
		topicId, err := iter.Key()
		if err != nil {
			return nil, err
		}
		topics = append(topics, topicId)
	}

	return topics, nil
}

// Add as topic as churnable
func (k *Keeper) AddChurnableTopic(ctx context.Context, topicId TopicId) error {
	return k.churnableTopics.Set(ctx, topicId)
}

// ResetChurnReadyTopics clears all topics from the churn-ready set and resets related states.
func (k *Keeper) ResetChurnableTopics(ctx context.Context) error {
	k.churnableTopics.Clear(ctx, nil)
	return nil
}

// REWARDABLE TOPICS

// Get the rewardable topics
func (k *Keeper) GetRewardableTopics(ctx context.Context) ([]TopicId, error) {
	iter, err := k.rewardableTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	topics := make([]TopicId, 0)
	for ; iter.Valid(); iter.Next() {
		topicId, err := iter.Key()
		if err != nil {
			return nil, err
		}
		topics = append(topics, topicId)
	}

	return topics, nil
}

// Add a topic as rewardable
func (k *Keeper) AddRewardableTopic(ctx context.Context, topicId TopicId) error {
	return k.rewardableTopics.Set(ctx, topicId)
}

func (k *Keeper) RemoveRewardableTopic(ctx context.Context, topicId TopicId) error {
	return k.rewardableTopics.Remove(ctx, topicId)
}

/// SCORES

// If the new score is older than the current score, don't update
func (k *Keeper) SetLatestInfererScore(ctx context.Context, topicId TopicId, worker ActorId, score types.Score) error {
	oldScore, err := k.GetLatestInfererScore(ctx, topicId, worker)
	if err != nil {
		return err
	}
	if oldScore.BlockHeight >= score.BlockHeight {
		return nil
	}
	key := collections.Join(topicId, worker)
	return k.latestInfererScoresByWorker.Set(ctx, key, score)
}

func (k *Keeper) GetLatestInfererScore(ctx context.Context, topicId TopicId, worker ActorId) (types.Score, error) {
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
func (k *Keeper) SetLatestForecasterScore(ctx context.Context, topicId TopicId, worker ActorId, score types.Score) error {
	oldScore, err := k.GetLatestForecasterScore(ctx, topicId, worker)
	if err != nil {
		return err
	}
	if oldScore.BlockHeight >= score.BlockHeight {
		return nil
	}
	key := collections.Join(topicId, worker)
	return k.latestForecasterScoresByWorker.Set(ctx, key, score)
}

func (k *Keeper) GetLatestForecasterScore(ctx context.Context, topicId TopicId, worker ActorId) (types.Score, error) {
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
func (k *Keeper) SetLatestReputerScore(ctx context.Context, topicId TopicId, reputer ActorId, score types.Score) error {
	oldScore, err := k.GetLatestReputerScore(ctx, topicId, reputer)
	if err != nil {
		return err
	}
	if oldScore.BlockHeight >= score.BlockHeight {
		return nil
	}
	key := collections.Join(topicId, reputer)
	return k.latestReputerScoresByReputer.Set(ctx, key, score)
}

func (k *Keeper) GetLatestReputerScore(ctx context.Context, topicId TopicId, reputer ActorId) (types.Score, error) {
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

func (k *Keeper) InsertWorkerInferenceScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return err
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	maxNumScores := moduleParams.MaxSamplesToScaleScores

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockHeight)
	return k.infererScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetInferenceScoresUntilBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight).
		Descending()

	scores := make([]*types.Score, 0)
	iter, err := k.infererScoresByBlock.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}

	// Get max number of time steps that should be retrieved
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxNumTimeSteps := moduleParams.MaxSamplesToScaleScores

	count := 0
	for ; iter.Valid() && count < int(maxNumTimeSteps); iter.Next() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, err
		}
		for _, score := range existingScores.Value.Scores {
			scores = append(scores, score)
			count++
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

func (k *Keeper) InsertWorkerForecastScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return err
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	maxNumScores := moduleParams.MaxSamplesToScaleScores

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockHeight)
	return k.forecasterScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetForecastScoresUntilBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight).
		Descending()

	scores := make([]*types.Score, 0)
	iter, err := k.forecasterScoresByBlock.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}

	// Get max number of time steps that should be retrieved
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	maxNumTimeSteps := moduleParams.MaxSamplesToScaleScores

	count := 0
	for ; iter.Valid() && count < int(maxNumTimeSteps); iter.Next() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, err
		}
		for _, score := range existingScores.Value.Scores {
			scores = append(scores, score)
			count++
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

func (k *Keeper) InsertReputerScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return err
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	maxNumScores := moduleParams.MaxSamplesToScaleScores
	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		if diff > 0 {
			scores.Scores = scores.Scores[diff:]
		}
	}
	key := collections.Join(topicId, blockHeight)
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

func (k *Keeper) SetListeningCoefficient(ctx context.Context, topicId TopicId, reputer ActorId, coefficient types.ListeningCoefficient) error {
	key := collections.Join(topicId, reputer)
	return k.reputerListeningCoefficient.Set(ctx, key, coefficient)
}

func (k *Keeper) GetListeningCoefficient(ctx context.Context, topicId TopicId, reputer ActorId) (types.ListeningCoefficient, error) {
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
func (k *Keeper) GetPreviousReputerRewardFraction(ctx context.Context, topicId TopicId, reputer ActorId) (alloraMath.Dec, bool, error) {
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
func (k *Keeper) SetPreviousReputerRewardFraction(ctx context.Context, topicId TopicId, reputer ActorId, reward alloraMath.Dec) error {
	key := collections.Join(topicId, reputer)
	return k.previousReputerRewardFraction.Set(ctx, key, reward)
}

// Gets the previous U_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker ActorId) (alloraMath.Dec, bool, error) {
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
func (k *Keeper) SetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker ActorId, reward alloraMath.Dec) error {
	key := collections.Join(topicId, worker)
	return k.previousInferenceRewardFraction.Set(ctx, key, reward)
}

// Gets the previous V_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker ActorId) (alloraMath.Dec, bool, error) {
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
func (k *Keeper) SetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker ActorId, reward alloraMath.Dec) error {
	key := collections.Join(topicId, worker)
	return k.previousForecastRewardFraction.Set(ctx, key, reward)
}

func (k *Keeper) SetPreviousPercentageRewardToStakedReputers(ctx context.Context, percentageRewardToStakedReputers alloraMath.Dec) error {
	return k.previousPercentageRewardToStakedReputers.Set(ctx, percentageRewardToStakedReputers)
}

func (k Keeper) GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (alloraMath.Dec, error) {
	return k.previousPercentageRewardToStakedReputers.Get(ctx)
}

/// WHITELISTS

func (k Keeper) IsWhitelistAdmin(ctx context.Context, admin ActorId) (bool, error) {
	return k.whitelistAdmins.Has(ctx, admin)
}

func (k *Keeper) AddWhitelistAdmin(ctx context.Context, admin ActorId) error {
	return k.whitelistAdmins.Set(ctx, admin)
}

func (k *Keeper) RemoveWhitelistAdmin(ctx context.Context, admin ActorId) error {
	return k.whitelistAdmins.Remove(ctx, admin)
}

/// BANK KEEPER WRAPPERS

// wrapper around bank keeper SendCoinsFromModuleToAccount
func (k *Keeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient ActorId, amt sdk.Coins) error {
	recipientAddr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		return err
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// wrapper around bank keeper SendCoinsFromAccountToModule
func (k *Keeper) SendCoinsFromAccountToModule(ctx context.Context, sender ActorId, recipientModule string, amt sdk.Coins) error {
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return err
	}
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}

// wrapper around bank keeper SendCoinsFromModuleToModule
func (k *Keeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt)
}

// wrapper around bank keeper GetBalance
func (k *Keeper) GetBankBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return k.bankKeeper.GetBalance(ctx, addr, denom)
}

// GetTotalRewardToDistribute
func (k *Keeper) GetTotalRewardToDistribute(ctx context.Context) (alloraMath.Dec, error) {
	// Get Allora Rewards Account
	alloraRewardsAccountAddr := k.authKeeper.GetModuleAccount(ctx, types.AlloraRewardsAccountName).GetAddress()
	// Get Total Allocation
	totalReward := k.GetBankBalance(
		ctx,
		alloraRewardsAccountAddr,
		params.DefaultBondDenom).Amount
	totalRewardDec, err := alloraMath.NewDecFromSdkInt(totalReward)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return totalRewardDec, nil
}

/// UTILS

// Convert pagination.key from []bytes to uint64, if pagination is nil or [], len = 0
// Get the limit from the pagination request, within acceptable bounds and defaulting as necessary
func (k Keeper) CalcAppropriatePaginationForUint64Cursor(ctx context.Context, pagination *types.SimpleCursorPaginationRequest) (uint64, uint64, error) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return uint64(0), uint64(0), err
	}
	limit := moduleParams.DefaultPageLimit
	cursor := uint64(0)

	if pagination != nil {
		if len(pagination.Key) > 0 {
			cursor = binary.BigEndian.Uint64(pagination.Key)
		}
		if pagination.Limit > 0 {
			limit = pagination.Limit
		}
		if limit > moduleParams.MaxPageLimit {
			limit = moduleParams.MaxPageLimit
		}
	}

	return limit, cursor, nil
}

/// STATE MANAGEMENT

// Iterate through topic state and prune records that are no longer needed
func (k *Keeper) PruneRecordsAfterRewards(ctx context.Context, topicId TopicId, blockHeight int64) error {
	// Delete records until the blockHeight
	blockRange := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight)

	err := k.pruneInferences(ctx, blockRange)
	if err != nil {
		return err
	}
	err = k.pruneForecasts(ctx, blockRange)
	if err != nil {
		return err
	}
	err = k.pruneLossBundles(ctx, blockRange)
	if err != nil {
		return err
	}
	err = k.pruneNetworkLosses(ctx, blockRange)
	if err != nil {
		return err
	}

	return nil
}

func (k *Keeper) pruneInferences(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.allInferences.Clear(ctx, blockRange)
}

func (k *Keeper) pruneForecasts(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.allForecasts.Clear(ctx, blockRange)
}

func (k *Keeper) pruneLossBundles(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.allLossBundles.Clear(ctx, blockRange)
}

func (k *Keeper) pruneNetworkLosses(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.networkLossBundles.Clear(ctx, blockRange)
}

func (k *Keeper) PruneWorkerNonces(ctx context.Context, topicId uint64, blockHeightThreshold int64) error {
	nonces, err := k.unfulfilledWorkerNonces.Get(ctx, topicId)
	if err != nil {
		return err
	}

	// Filter Nonces based on block_height
	filteredNonces := make([]*types.Nonce, 0)
	for _, nonce := range nonces.Nonces {
		if nonce.BlockHeight >= blockHeightThreshold {
			filteredNonces = append(filteredNonces, nonce)
		}
	}

	// Update nonces in the map
	nonces.Nonces = filteredNonces
	if err := k.unfulfilledWorkerNonces.Set(ctx, topicId, nonces); err != nil {
		return err
	}

	return nil
}

func (k *Keeper) PruneReputerNonces(ctx context.Context, topicId uint64, blockHeightThreshold int64) error {
	nonces, err := k.unfulfilledReputerNonces.Get(ctx, topicId)
	if err != nil {
		return err
	}

	// Filter Nonces based on block_height
	filteredNonces := make([]*types.ReputerRequestNonce, 0)
	for _, nonce := range nonces.Nonces {
		if nonce.ReputerNonce.BlockHeight >= blockHeightThreshold {
			filteredNonces = append(filteredNonces, nonce)
		}
	}

	// Update nonces in the map
	nonces.Nonces = filteredNonces
	if err := k.unfulfilledReputerNonces.Set(ctx, topicId, nonces); err != nil {
		return err
	}

	return nil
}

// Return true if the topic has met its cadence or is the first run
func (k *Keeper) CheckCadence(blockHeight int64, topic types.Topic) bool {
	return (blockHeight-topic.EpochLastEnded)%topic.EpochLength == 0 ||
		topic.EpochLastEnded == 0
}

func (k *Keeper) ValidateStringIsBech32(actor ActorId) error {
	_, err := sdk.AccAddressFromBech32(actor)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) SetTopicLastCommit(ctx context.Context, topic types.TopicId, blockHeight int64, nonce *types.Nonce, actor ActorId, actorType types.ActorType) error {
	if actorType == types.ActorType_REPUTER {
		return k.topicLastReputerCommit.Set(ctx, topic, types.TimestampedActorNonce{
			BlockHeight: blockHeight,
			Actor:       actor,
			Nonce:       nonce,
		})
	}
	return k.topicLastWorkerCommit.Set(ctx, topic, types.TimestampedActorNonce{
		BlockHeight: blockHeight,
		Actor:       actor,
		Nonce:       nonce,
	})
}

func (k *Keeper) GetTopicLastCommit(ctx context.Context, topic TopicId, actorType types.ActorType) (types.TimestampedActorNonce, error) {
	if actorType == types.ActorType_REPUTER {
		return k.topicLastReputerCommit.Get(ctx, topic)
	}
	return k.topicLastWorkerCommit.Get(ctx, topic)
}
