package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
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
	storeService     coreStore.KVStoreService
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
	// for a topic, what is every worker node that has registered to it?
	topicWorkers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// for a topic, what is every reputer node that has registered to it?
	topicReputers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// map of (topic) -> nonce/block height
	topicRewardNonce collections.Map[TopicId, BlockHeight]
	// active topic inferers for a topic
	activeInferers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// active topic forecasters for a topic
	activeForecasters collections.KeySet[collections.Pair[TopicId, ActorId]]
	// lowest topic inferer score ema for a topic
	lowestInfererScoreEma collections.Map[TopicId, types.Score]
	// lowest topic forecaster score ema for a topic
	lowestForecasterScoreEma collections.Map[TopicId, types.Score]

	/// SCORES

	// map of (topic, block_height, worker) -> score
	infererScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, worker) -> score
	forecasterScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, reputer) -> score
	reputerScoresByBlock collections.Map[collections.Pair[TopicId, BlockHeight], types.Scores]
	// map of (topic, block_height, worker) -> score
	infererScoreEmas collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, block_height, worker) -> score
	forecasterScoreEmas collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, block_height, reputer) -> score
	reputerScoreEmas collections.Map[collections.Pair[TopicId, ActorId], types.Score]
	// map of (topic, reputer) -> listening coefficient
	reputerListeningCoefficient collections.Map[collections.Pair[TopicId, ActorId], types.ListeningCoefficient]
	// map of (topic, reputer) -> previous reward (used for EMA)
	previousReputerRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for inference (used for EMA)
	previousInferenceRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of (topic, worker) -> previous reward for forecast (used for EMA)
	previousForecastRewardFraction collections.Map[collections.Pair[TopicId, ActorId], alloraMath.Dec]
	// map of topic -> previous forecaster score ratio
	previousForecasterScoreRatio collections.Map[TopicId, alloraMath.Dec]
	// previous topic inferer ema score at topic quantile
	previousTopicQuantileInfererScoreEma collections.Map[TopicId, alloraMath.Dec]
	// previous topic forecaster ema score at topic quantile
	previousTopicQuantileForecasterScoreEma collections.Map[TopicId, alloraMath.Dec]
	// previous topic reputer ema score at topic quantile
	previousTopicQuantileReputerScoreEma collections.Map[TopicId, alloraMath.Dec]

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

	// map of (topic, reputer) -> reputer loss
	reputerLosses collections.Map[collections.Pair[TopicId, Reputer], types.ReputerValueBundle]

	// map of worker id to node data about that worker
	workers collections.Map[ActorId, types.OffchainNode]

	// map of reputer id to node data about that reputer
	reputers collections.Map[ActorId, types.OffchainNode]

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

	// map of open worker nonce windows for topics on particular block heights
	openWorkerWindows collections.Map[BlockHeight, types.TopicIds]

	// map of (topic) -> unfulfilled nonces
	unfulfilledWorkerNonces collections.Map[TopicId, types.Nonces]

	// map of (topic) -> unfulfilled nonces
	unfulfilledReputerNonces collections.Map[TopicId, types.ReputerRequestNonces]

	// map of (topic) -> last dripped block
	lastDripBlock collections.Map[TopicId, BlockHeight]

	/// REGRETS

	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other inferers
	latestInfererNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other forecasters
	latestForecasterNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic, forecaster, inferer) -> R^+_{ij_kk} regret of forecaster loss from comparing one-in loss with
	// all network inferer (3rd index) regrets L_ij made under the regime of the one-in forecaster (2nd index)
	latestOneInForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topic id, inferer) -> regret
	latestNaiveInfererNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic id , one out inferer, inferer)-> regret
	latestOneOutInfererInfererNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topicId, oneOutInferer, forecaster) -> regret
	latestOneOutInfererForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topicId, oneOutInferer, inferer) -> regret
	latestOneOutForecasterInfererNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topicId, oneOutForecaster, forecaster) -> regret
	latestOneOutForecasterForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]

	/// INCLUSIONS

	countInfererInclusionsInTopicActiveSet    collections.Map[collections.Pair[TopicId, ActorId], uint64]
	countForecasterInclusionsInTopicActiveSet collections.Map[collections.Pair[TopicId, ActorId], uint64]

	/// WHITELISTS

	whitelistAdmins collections.KeySet[ActorId]

	/// RECORD COMMITS

	topicLastWorkerCommit  collections.Map[TopicId, types.TimestampedActorNonce]
	topicLastReputerCommit collections.Map[TopicId, types.TimestampedActorNonce]

	// active reputers for a topic
	activeReputers collections.KeySet[collections.Pair[TopicId, ActorId]]
	// lowest reputer score ema for a topic
	lowestReputerScoreEma collections.Map[TopicId, types.Score]
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
		schema:                                   collections.Schema{},
		cdc:                                      cdc,
		storeService:                             storeService,
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
		topicToNextPossibleChurningBlock:         collections.NewMap(sb, types.TopicToNextPossibleChurningBlockKey, "topic_to_next_possible_churning_block", collections.Uint64Key, collections.Int64Value),
		blockToActiveTopics:                      collections.NewMap(sb, types.BlockToActiveTopicsKey, "block_to_active_topics", collections.Int64Key, codec.CollValue[types.TopicIds](cdc)),
		blockToLowestActiveTopicWeight:           collections.NewMap(sb, types.BlockToLowestActiveTopicWeightKey, "block_to_lowest_active_topic_weight", collections.Int64Key, codec.CollValue[types.TopicIdWeightPair](cdc)),
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
		latestNaiveInfererNetworkRegrets:         collections.NewMap(sb, types.LatestNaiveInfererNetworkRegretsKey, "latest_naive_inferer_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutInfererInfererNetworkRegrets: collections.NewMap(sb, types.LatestOneOutInfererInfererNetworkRegretsKey, "latest_one_out_inferer_inferer_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutInfererForecasterNetworkRegrets:    collections.NewMap(sb, types.LatestOneOutInfererForecasterNetworkRegretsKey, "latest_one_out_inferer_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutForecasterInfererNetworkRegrets:    collections.NewMap(sb, types.LatestOneOutForecasterInfererNetworkRegretsKey, "latest_one_out_forecaster_inferer_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutForecasterForecasterNetworkRegrets: collections.NewMap(sb, types.LatestOneOutForecasterForecasterNetworkRegretsKey, "latest_one_out_forecaster_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		whitelistAdmins:                           collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", collections.StringKey),
		infererScoresByBlock:                      collections.NewMap(sb, types.InferenceScoresKey, "inferer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		forecasterScoresByBlock:                   collections.NewMap(sb, types.ForecastScoresKey, "forecaster_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		infererScoreEmas:                          collections.NewMap(sb, types.InfererScoreEmasKey, "latest_inferer_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		forecasterScoreEmas:                       collections.NewMap(sb, types.ForecasterScoreEmasKey, "latest_forecaster_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		reputerScoreEmas:                          collections.NewMap(sb, types.ReputerScoreEmasKey, "latest_reputer_scores_by_reputer", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		previousReputerRewardFraction:             collections.NewMap(sb, types.PreviousReputerRewardFractionKey, "previous_reputer_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousInferenceRewardFraction:           collections.NewMap(sb, types.PreviousInferenceRewardFractionKey, "previous_inference_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecastRewardFraction:            collections.NewMap(sb, types.PreviousForecastRewardFractionKey, "previous_forecast_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecasterScoreRatio:              collections.NewMap(sb, types.PreviousForecasterScoreRatioKey, "previous_forecaster_score_ratio", collections.Uint64Key, alloraMath.DecValue),
		reputerScoresByBlock:                      collections.NewMap(sb, types.ReputerScoresKey, "reputer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerListeningCoefficient:               collections.NewMap(sb, types.ReputerListeningCoefficientKey, "reputer_listening_coefficient", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.ListeningCoefficient](cdc)),
		unfulfilledWorkerNonces:                   collections.NewMap(sb, types.UnfulfilledWorkerNoncesKey, "unfulfilled_worker_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
		unfulfilledReputerNonces:                  collections.NewMap(sb, types.UnfulfilledReputerNoncesKey, "unfulfilled_reputer_nonces", collections.Uint64Key, codec.CollValue[types.ReputerRequestNonces](cdc)),
		lastDripBlock:                             collections.NewMap(sb, types.LastDripBlockKey, "last_drip_block", collections.Uint64Key, collections.Int64Value),
		topicRewardNonce:                          collections.NewMap(sb, types.TopicRewardNonceKey, "topic_reward_nonce", collections.Uint64Key, collections.Int64Value),
		topicLastWorkerCommit:                     collections.NewMap(sb, types.TopicLastWorkerCommitKey, "topic_last_worker_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		topicLastReputerCommit:                    collections.NewMap(sb, types.TopicLastReputerCommitKey, "topic_last_reputer_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		openWorkerWindows:                         collections.NewMap(sb, types.OpenWorkerWindowsKey, "open_worker_windows", collections.Int64Key, codec.CollValue[types.TopicIds](cdc)),
		previousTopicQuantileInfererScoreEma:      collections.NewMap(sb, types.PreviousTopicQuantileInfererScoreEmaKey, "previous_topic_quantile_inferer_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileForecasterScoreEma:   collections.NewMap(sb, types.PreviousTopicQuantileForecasterScoreEmaKey, "previous_topic_quantile_forecaster_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileReputerScoreEma:      collections.NewMap(sb, types.PreviousTopicQuantileReputerScoreEmaKey, "previous_topic_quantile_reputer_score_ema", collections.Uint64Key, alloraMath.DecValue),
		countInfererInclusionsInTopicActiveSet:    collections.NewMap(sb, types.CountInfererInclusionsInTopicKey, "count_inferer_inclusions_in_topic", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), collections.Uint64Value),
		countForecasterInclusionsInTopicActiveSet: collections.NewMap(sb, types.CountForecasterInclusionsInTopicKey, "count_forecaster_inclusions_in_topic", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), collections.Uint64Value),
		qualifiedInferers:                       collections.NewKeySet(sb, types.QualifiedInferersKey, "qualified_inferers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		qualifiedForecasters:                    collections.NewKeySet(sb, types.QualifiedForecastersKey, "qualified_forecasters", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		lowestInfererScoreEmas:                  collections.NewMap(sb, types.LowestInfererScoreEmasKey, "lowest_inferer_score_emas", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		lowestForecasterScoreEmas:               collections.NewMap(sb, types.LowestForecasterScoreEmasKey, "lowest_forecaster_score_emas", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		whitelistAdmins:                         collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", collections.StringKey),
		infererScoresByBlock:                    collections.NewMap(sb, types.InferenceScoresKey, "inferer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		forecasterScoresByBlock:                 collections.NewMap(sb, types.ForecastScoresKey, "forecaster_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		infererScoreEmas:                        collections.NewMap(sb, types.InfererScoreEmasKey, "latest_inferer_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		forecasterScoreEmas:                     collections.NewMap(sb, types.ForecasterScoreEmasKey, "latest_forecaster_scores_by_worker", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		reputerScoreEmas:                        collections.NewMap(sb, types.ReputerScoreEmasKey, "latest_reputer_scores_by_reputer", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.Score](cdc)),
		previousReputerRewardFraction:           collections.NewMap(sb, types.PreviousReputerRewardFractionKey, "previous_reputer_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousInferenceRewardFraction:         collections.NewMap(sb, types.PreviousInferenceRewardFractionKey, "previous_inference_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecastRewardFraction:          collections.NewMap(sb, types.PreviousForecastRewardFractionKey, "previous_forecast_reward_fraction", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		previousForecasterScoreRatio:            collections.NewMap(sb, types.PreviousForecasterScoreRatioKey, "previous_forecaster_score_ratio", collections.Uint64Key, alloraMath.DecValue),
		reputerScoresByBlock:                    collections.NewMap(sb, types.ReputerScoresKey, "reputer_scores_by_block", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.Scores](cdc)),
		reputerListeningCoefficient:             collections.NewMap(sb, types.ReputerListeningCoefficientKey, "reputer_listening_coefficient", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.ListeningCoefficient](cdc)),
		unfulfilledWorkerNonces:                 collections.NewMap(sb, types.UnfulfilledWorkerNoncesKey, "unfulfilled_worker_nonces", collections.Uint64Key, codec.CollValue[types.Nonces](cdc)),
		unfulfilledReputerNonces:                collections.NewMap(sb, types.UnfulfilledReputerNoncesKey, "unfulfilled_reputer_nonces", collections.Uint64Key, codec.CollValue[types.ReputerRequestNonces](cdc)),
		lastDripBlock:                           collections.NewMap(sb, types.LastDripBlockKey, "last_drip_block", collections.Uint64Key, collections.Int64Value),
		topicRewardNonce:                        collections.NewMap(sb, types.TopicRewardNonceKey, "topic_reward_nonce", collections.Uint64Key, collections.Int64Value),
		topicLastWorkerCommit:                   collections.NewMap(sb, types.TopicLastWorkerCommitKey, "topic_last_worker_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		topicLastReputerCommit:                  collections.NewMap(sb, types.TopicLastReputerCommitKey, "topic_last_reputer_commit", collections.Uint64Key, codec.CollValue[types.TimestampedActorNonce](cdc)),
		openWorkerWindows:                       collections.NewMap(sb, types.OpenWorkerWindowsKey, "open_worker_windows", collections.Int64Key, codec.CollValue[types.TopicIds](cdc)),
		previousTopicQuantileInfererScoreEma:    collections.NewMap(sb, types.PreviousTopicQuantileInfererScoreEmaKey, "previous_topic_quantile_inferer_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileForecasterScoreEma: collections.NewMap(sb, types.PreviousTopicQuantileForecasterScoreEmaKey, "previous_topic_quantile_forecaster_score_ema", collections.Uint64Key, alloraMath.DecValue),
		previousTopicQuantileReputerScoreEma:    collections.NewMap(sb, types.PreviousTopicQuantileReputerScoreEmaKey, "previous_topic_quantile_reputer_score_ema", collections.Uint64Key, alloraMath.DecValue),
		activeInferers:                          collections.NewKeySet(sb, types.ActiveInferersKey, "active_inferers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		activeForecasters:                       collections.NewKeySet(sb, types.ActiveForecastersKey, "active_forecasters", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		lowestInfererScoreEma:                   collections.NewMap(sb, types.LowestInfererScoreEmaKey, "lowest_inferer_score_ema", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		lowestForecasterScoreEma:                collections.NewMap(sb, types.LowestForecasterScoreEmaKey, "lowest_forecaster_score_ema", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		activeReputers:                          collections.NewKeySet(sb, types.ActiveReputersKey, "active_reputers", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		lowestReputerScoreEma:                   collections.NewMap(sb, types.LowestReputerScoreEmaKey, "lowest_reputer_score_ema", collections.Uint64Key, codec.CollValue[types.Score](cdc)),
		reputerLosses:                           collections.NewMap(sb, types.ReputerLossesKey, "reputer_losses", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.ReputerValueBundle](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.schema = schema

	return k
}

func (k *Keeper) GetStorageService() coreStore.KVStoreService {
	return k.storeService
}

func (k *Keeper) GetBinaryCodec() codec.BinaryCodec {
	return k.cdc
}

/// NONCES

// GetTopicIds returns the TopicIds for a given BlockHeight.
// If no TopicIds are found for the BlockHeight, it returns an empty slice.
func (k *Keeper) GetWorkerWindowTopicIds(ctx sdk.Context, height BlockHeight) types.TopicIds {
	topicIds, err := k.openWorkerWindows.Get(ctx, height)
	if err != nil {
		return types.TopicIds{TopicIds: []TopicId{}}
	}
	return topicIds
}

// SetTopicId appends a new TopicId to the list of TopicIds for a given BlockHeight.
// If no entry exists for the BlockHeight, it creates a new entry with the TopicId.
func (k *Keeper) AddWorkerWindowTopicId(ctx sdk.Context, height BlockHeight, topicId TopicId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBlockHeight(height); err != nil {
		return errorsmod.Wrap(err, "error validating block height")
	}
	var topicIds types.TopicIds
	topicIds, err := k.openWorkerWindows.Get(ctx, height)
	if err != nil {
		topicIds = types.TopicIds{
			TopicIds: []types.TopicId{},
		}
	}
	topicIds.TopicIds = append(topicIds.TopicIds, topicId)
	err = k.openWorkerWindows.Set(ctx, height, topicIds)
	return errorsmod.Wrap(err, "error setting open worker windows")
}

func (k *Keeper) DeleteWorkerWindowBlockHeight(ctx sdk.Context, height BlockHeight) error {
	return k.openWorkerWindows.Remove(ctx, height)
}

// Attempts to fulfill an unfulfilled nonce.
// If the nonce is present, then it is removed from the unfulfilled nonces and this function returns true.
// If the nonce is not present, then the function returns false.
func (k *Keeper) FulfillWorkerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) (bool, error) {
	if err := types.ValidateTopicId(topicId); err != nil {
		return false, errorsmod.Wrap(err, "error validating topic id")
	}
	if err := nonce.Validate(); err != nil {
		return false, errorsmod.Wrap(err, "error validating nonce")
	}
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n.BlockHeight == nonce.BlockHeight {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			if err := unfulfilledNonces.Validate(); err != nil {
				return false, errorsmod.Wrap(err, "error validating unfulfilled nonces")
			}
			err := k.unfulfilledWorkerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return false, errorsmod.Wrap(err, "error setting unfulfilled worker nonces")
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
	if err := types.ValidateTopicId(topicId); err != nil {
		return false, errorsmod.Wrap(err, "error validating topic id")
	}
	if err := nonce.Validate(); err != nil {
		return false, errorsmod.Wrap(err, "error validating nonce")
	}
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Check if the nonce is present in the unfulfilled nonces
	for i, n := range unfulfilledNonces.Nonces {
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			// Remove the nonce from the unfulfilled nonces
			unfulfilledNonces.Nonces = append(unfulfilledNonces.Nonces[:i], unfulfilledNonces.Nonces[i+1:]...)
			if err := unfulfilledNonces.Validate(); err != nil {
				return false, errorsmod.Wrap(err, "error validating unfulfilled reputer nonces")
			}
			err := k.unfulfilledReputerNonces.Set(ctx, topicId, unfulfilledNonces)
			if err != nil {
				return false, errorsmod.Wrap(err, "error setting unfulfilled reputer nonces")
			}
			return true, nil
		}
	}

	// If the nonce is not present in the unfulfilled nonces
	return false, nil
}

// True if nonce is unfulfilled, false otherwise.
func (k *Keeper) IsWorkerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (isUnfulfilled bool, err error) {
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
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
func (k *Keeper) IsReputerNonceUnfulfilled(ctx context.Context, topicId TopicId, nonce *types.Nonce) (isUnfulfilled bool, err error) {
	// Get the latest unfulfilled nonces
	unfulfilledNonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
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
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := nonce.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating nonce")
	}
	nonces, err := k.GetUnfulfilledWorkerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
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
		return errorsmod.Wrap(err, "error getting module params")
	}
	maxUnfulfilledRequests := moduleParams.MaxUnfulfilledWorkerRequests

	lenNonces := uint64(len(nonces.Nonces))
	if lenNonces > maxUnfulfilledRequests {
		diff := uint64(len(nonces.Nonces)) - maxUnfulfilledRequests
		if diff > 0 {
			nonces.Nonces = nonces.Nonces[:maxUnfulfilledRequests]
		}
	}

	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled worker nonces")
	}
	return k.unfulfilledWorkerNonces.Set(ctx, topicId, nonces)
}

// Adds a nonce to the unfulfilled nonces for the topic if it is not yet added (idempotent).
// If the max number of nonces is reached, then the function removes the oldest nonce and adds the new nonce.
func (k *Keeper) AddReputerNonce(ctx context.Context, topicId TopicId, nonce *types.Nonce) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := nonce.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating nonce")
	}
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}
	if nonce == nil {
		return errors.New("nil reputer's nonce provided")
	}

	// Check that input nonce is not already contained in the nonces of this topic
	// nor that the `associatedWorkerNonce` is already associated with a worker request
	for _, n := range nonces.Nonces {
		// Do nothing if nonce is already in the list
		if n.ReputerNonce.BlockHeight == nonce.BlockHeight {
			return nil
		}
	}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: nonce,
	}
	nonces.Nonces = append([]*types.ReputerRequestNonce{reputerRequestNonce}, nonces.Nonces...)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting module params")
	}
	maxUnfulfilledRequests := moduleParams.MaxUnfulfilledReputerRequests
	lenNonces := uint64(len(nonces.Nonces))
	if lenNonces > maxUnfulfilledRequests {
		diff := uint64(len(nonces.Nonces)) - maxUnfulfilledRequests
		if diff > 0 {
			nonces.Nonces = nonces.Nonces[:maxUnfulfilledRequests]
		}
	}
	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled reputer nonces")
	}
	return k.unfulfilledReputerNonces.Set(ctx, topicId, nonces)
}

func (k *Keeper) GetUnfulfilledWorkerNonces(ctx context.Context, topicId TopicId) (types.Nonces, error) {
	nonces, err := k.unfulfilledWorkerNonces.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Nonces{Nonces: []*types.Nonce{}}, nil
		}
		return types.Nonces{Nonces: []*types.Nonce{}}, errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
	}
	return nonces, nil
}

func (k *Keeper) GetUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) (types.ReputerRequestNonces, error) {
	nonces, err := k.unfulfilledReputerNonces.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ReputerRequestNonces{Nonces: []*types.ReputerRequestNonce{}}, nil
		}
		return types.ReputerRequestNonces{Nonces: []*types.ReputerRequestNonce{}}, errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}
	return nonces, nil
}

func (k *Keeper) DeleteUnfulfilledWorkerNonces(ctx context.Context, topicId TopicId) error {
	return k.unfulfilledWorkerNonces.Remove(ctx, topicId)
}

func (k *Keeper) DeleteUnfulfilledReputerNonces(ctx context.Context, topicId TopicId) error {
	return k.unfulfilledReputerNonces.Remove(ctx, topicId)
}

/// INCLUSIONS

// Get the count of inferer inclusions in topic active set
func (k *Keeper) GetCountInfererInclusionsInTopic(ctx context.Context, topicId TopicId, inferer ActorId) (uint64, error) {
	key := collections.Join(topicId, inferer)
	count, err := k.countInfererInclusionsInTopicActiveSet.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return count, nil
}

// Get the count of inferer inclusions in topic active set
func (k *Keeper) IncrementCountInfererInclusionsInTopic(ctx context.Context, topicId TopicId, inferer ActorId) error {
	key := collections.Join(topicId, inferer)
	count, err := k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer)
	if err != nil {
		return err
	}
	count++
	return k.countInfererInclusionsInTopicActiveSet.Set(ctx, key, count)
}

// Get the count of forecaster inclusions in topic active set
func (k *Keeper) GetCountForecasterInclusionsInTopic(ctx context.Context, topicId TopicId, forecaster ActorId) (uint64, error) {
	key := collections.Join(topicId, forecaster)
	count, err := k.countForecasterInclusionsInTopicActiveSet.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return count, nil
}

// Increase the count of forecaster inclusions in topic active set
func (k *Keeper) IncrementCountForecasterInclusionsInTopic(ctx context.Context, topicId TopicId, forecaster ActorId) error {
	key := collections.Join(topicId, forecaster)
	count, err := k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster)
	if err != nil {
		return err
	}
	count++
	return k.countForecasterInclusionsInTopicActiveSet.Set(ctx, key, count)
}

/// REGRETS

func (k *Keeper) SetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker ActorId, regret types.TimestampedValue) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "error validating worker id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join(topicId, worker)
	return k.latestInfererNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a inferer from comparing loss of inferer relative to loss of other inferers
// Returns (0, true) if no regret is found
func (k *Keeper) GetInfererNetworkRegret(
	ctx context.Context, topicId TopicId, worker ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	regret, err = k.latestInfererNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{
					BlockHeight: 0,
					Value:       alloraMath.ZeroDec(),
				}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       alloraMath.ZeroDec(),
		}, false, errorsmod.Wrap(err, "error getting inferer network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "error validating worker id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join(topicId, worker)
	return k.latestForecasterNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a forecaster from comparing loss of forecaster relative to loss of other forecasters
// Returns (0, true) if no regret is found
func (k *Keeper) GetForecasterNetworkRegret(
	ctx context.Context, topicId TopicId, worker ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	regret, err = k.latestForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting forecaster network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetOneInForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneInForecaster ActorId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneInForecaster); err != nil {
		return errorsmod.Wrap(err, "error validating one in forecaster id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneInForecaster, inferer)
	return k.latestOneInForecasterNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a forecaster from comparing loss of forecaster relative to loss of other forecasters
// Returns (0, true) if no regret is found
func (k *Keeper) GetOneInForecasterNetworkRegret(
	ctx context.Context, topicId TopicId, oneInForecaster ActorId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneInForecaster, inferer)
	regret, err = k.latestOneInForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one in forecaster network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetNaiveInfererNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join(topicId, inferer)
	return k.latestNaiveInfererNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) GetNaiveInfererNetworkRegret(ctx context.Context, topicId TopicId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join(topicId, inferer)
	regret, err = k.latestNaiveInfererNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting naive inferer network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetOneOutInfererInfererNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutInferer ActorId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutInferer); err != nil {
		return errorsmod.Wrap(err, "error validating one out inferer id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutInferer, inferer)
	return k.latestOneOutInfererInfererNetworkRegrets.Set(ctx, key, regret)
}

// return the one out inferer regret, how much the one out inferer affects the network loss
// if no prior is found, return the initial regret of the topic
func (k *Keeper) GetOneOutInfererInfererNetworkRegret(
	ctx context.Context, topicId TopicId, oneOutInferer ActorId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutInferer, inferer)
	regret, err = k.latestOneOutInfererInfererNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out inferer inferer network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetOneOutInfererForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutInferer ActorId,
	forecaster ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutInferer); err != nil {
		return errorsmod.Wrap(err, "error validating one out inferer id")
	}
	if err := types.ValidateBech32(forecaster); err != nil {
		return errorsmod.Wrap(err, "error validating forecaster id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutInferer, forecaster)
	return k.latestOneOutInfererForecasterNetworkRegrets.Set(ctx, key, regret)
}

// return the one out inferer forecaster regret, how much that inferer affects the forecast loss
// if no prior is found, return the initial regret of the topic
func (k *Keeper) GetOneOutInfererForecasterNetworkRegret(
	ctx context.Context, topicId TopicId, oneOutInferer ActorId, forecaster ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutInferer, forecaster)
	regret, err = k.latestOneOutInfererForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out inferer forecaster network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetOneOutForecasterInfererNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutForecaster ActorId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutForecaster); err != nil {
		return errorsmod.Wrap(err, "error validating one out forecaster id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutForecaster, inferer)
	return k.latestOneOutForecasterInfererNetworkRegrets.Set(ctx, key, regret)
}

// return the one out forecaster inferer regret, how much that forecaster affects the inferer network loss
// if no prior is found, return the initial regret of the topic
func (k *Keeper) GetOneOutForecasterInfererNetworkRegret(
	ctx context.Context, topicId TopicId, oneOutForecaster ActorId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutForecaster, inferer)
	regret, err = k.latestOneOutForecasterInfererNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out forecaster inferer network regret")
	}
	return regret, false, nil
}

func (k *Keeper) SetOneOutForecasterForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutForecaster ActorId,
	forecaster ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutForecaster); err != nil {
		return errorsmod.Wrap(err, "error validating one out forecaster id")
	}
	if err := types.ValidateBech32(forecaster); err != nil {
		return errorsmod.Wrap(err, "error validating forecaster id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutForecaster, forecaster)
	return k.latestOneOutForecasterForecasterNetworkRegrets.Set(ctx, key, regret)
}

func (k *Keeper) GetOneOutForecasterForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutForecaster ActorId,
	forecaster ActorId,
) (regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutForecaster, forecaster)
	regret, err = k.latestOneOutForecasterForecasterNetworkRegrets.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			topic, err := k.GetTopic(ctx, topicId)
			if err != nil {
				return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
			}
			return types.TimestampedValue{
				BlockHeight: 0,
				Value:       topic.InitialRegret,
			}, true, nil
		}
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out forecaster forecaster network regret")
	}
	return regret, false, nil
}

/// PARAMETERS

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return errorsmod.Wrap(err, "failed to set params")
	}
	return k.params.Set(ctx, params)
}

func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	ret, err := k.params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.DefaultParams(), nil
		}
		return types.Params{}, errorsmod.Wrap(err, "error getting params")
	}
	return ret, nil
}

/// INFERENCES, FORECASTS

func (k *Keeper) GetInferencesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.Inferences, error) {
	key := collections.Join(topicId, block)
	inferences, err := k.allInferences.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.Inferences{Inferences: []*types.Inference{}}, nil
		}
		return nil, errorsmod.Wrap(err, "error getting inferences at block")
	}
	return &inferences, nil
}

// GetLatestTopicInferences retrieves the latest topic inferences and its block height.
func (k *Keeper) GetLatestTopicInferences(ctx context.Context, topicId TopicId) (*types.Inferences, BlockHeight, error) {
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).Descending()

	iter, err := k.allInferences.Iterate(ctx, rng)
	if err != nil {
		return nil, 0, errorsmod.Wrap(err, "error iterating over inferences")
	}
	defer iter.Close()

	if iter.Valid() {
		keyValue, err := iter.KeyValue()
		if err != nil {
			return nil, 0, errorsmod.Wrap(err, "error getting key value")
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
		if errors.Is(err, collections.ErrNotFound) {
			return &types.Forecasts{Forecasts: []*types.Forecast{}}, nil
		}
		return nil, errorsmod.Wrap(err, "error getting forecasts at block")
	}
	return &forecasts, nil
}

// Append individual inference for a topic/block
func (k *Keeper) AppendInference(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	inference *types.Inference,
	maxTopInferersToReward uint64,
) error {
	// Check if the inferers already submited the inference
	isActive, err := k.IsActiveInferer(ctx, topic.Id, inference.Inferer)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if worker already submitted inference")
	} else if isActive {
		return errors.New("inference already submitted")
	}

	workerAddresses, err := k.GetActiveInferersForTopic(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting active inferers for topic")
	}
	// If there are less than maxTopInferersToReward, add the current inferer
	if uint64(len(workerAddresses)) < maxTopInferersToReward {
		err := k.AddActiveInferer(ctx, topic.Id, inference.Inferer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active inferer")
		}
		return k.InsertInference(ctx, topic.Id, *inference)
	}

	previousEmaScore, err := k.GetInfererScoreEma(ctx, topic.Id, inference.Inferer)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting inferer score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	lowestEmaScore, found, err := k.GetLowestInfererScoreEma(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting lowest inferer score ema")
	} else if !found {
		lowestEmaScore, err = GetLowestScoreFromAllInferers(ctx, k, topic.Id, workerAddresses)
		if err != nil {
			return errorsmod.Wrap(err, "error getting lowest score from all inferers")
		}
		err = k.SetLowestInfererScoreEma(ctx, topic.Id, lowestEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting lowest inferer score ema")
		}
	}

	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		// Update EMA score for the lowest score inferer, who is not the current inferer
		err = k.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(
			ctx,
			topic,
			nonceBlockHeight,
			lowestEmaScore,
		)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving inferer score ema with last saved topic quantile")
		}
		// Remove inferer with lowest score
		err = k.RemoveActiveInferer(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing active inferer")
		}
		// Add new active inferer
		err = k.AddActiveInferer(ctx, topic.Id, inference.Inferer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active inferer")
		}
		// Calculate new lowest score with updated infererAddresses
		err = UpdateLowestScoreFromInfererAddresses(ctx, k, topic.Id, workerAddresses, inference.Inferer, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error getting low score from all inferences")
		}
		return k.InsertInference(ctx, topic.Id, *inference)
	} else {
		// Update EMA score for the current inferer, who is the lowest score inferer
		err = k.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving inferer score ema with last saved topic quantile")
		}
	}
	return nil
}

// Insert an inference for a specific topic
func (k *Keeper) InsertInference(
	ctx context.Context,
	topicId TopicId,
	inference types.Inference,
) error {
	err := inference.Validate()
	if err != nil {
		return errorsmod.Wrap(err, "inference in list is invalid")
	}
	key := collections.Join(topicId, inference.Inferer)
	return k.inferences.Set(ctx, key, inference)
}

// Insert a complete set of inferences for a topic/block.
func (k *Keeper) InsertActiveInferences(
	ctx context.Context,
	topicId TopicId,
	nonceBlockHeight BlockHeight,
	inferences types.Inferences,
) error {
	key := collections.Join(topicId, nonceBlockHeight)
	return k.allInferences.Set(ctx, key, inferences)
}

// Append individual forecast for a topic/block
func (k *Keeper) AppendForecast(
	ctx sdk.Context,
	topic types.Topic,
	nonceBlockHeight BlockHeight,
	forecast *types.Forecast,
	maxTopForecastersToReward uint64,
) error {
	// Check if the forecaster already submitted the forecast
	isActive, err := k.IsActiveForecaster(ctx, topic.Id, forecast.Forecaster)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if forecaster already submitted forecast")
	} else if isActive {
		return errors.New("forecast already submitted")
	}

	forecasterAddresses, err := k.GetActiveForecastersForTopic(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting active forecasters for topic")
	}
	// If there are less than maxTopForecastersToReward, add the current forecaster
	if uint64(len(forecasterAddresses)) < maxTopForecastersToReward {
		err := k.AddActiveForecaster(ctx, topic.Id, forecast.Forecaster)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active forecaster")
		}
		return k.InsertForecast(ctx, topic.Id, *forecast)
	}

	previousEmaScore, err := k.GetForecasterScoreEma(ctx, topic.Id, forecast.Forecaster)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting forecaster score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	lowestEmaScore, found, err := k.GetLowestForecasterScoreEma(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting lowest forecaster score ema")
	} else if !found {
		lowestEmaScore, err = GetLowestScoreFromAllForecasters(ctx, k, topic.Id, forecasterAddresses)
		if err != nil {
			return errorsmod.Wrap(err, "error getting lowest score from all forecasters")
		}
		err = k.SetLowestForecasterScoreEma(ctx, topic.Id, lowestEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting lowest forecaster score ema")
		}
	}

	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		// Update EMA score for the lowest score forecaster, who is not the current forecaster
		err = k.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(
			ctx,
			topic,
			nonceBlockHeight,
			lowestEmaScore,
		)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving forecaster score ema with last saved topic quantile")
		}
		// Remove forecaster with lowest score
		err = k.RemoveActiveForecaster(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing active forecaster")
		}
		// Add new active forecaster
		err = k.AddActiveForecaster(ctx, topic.Id, forecast.Forecaster)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active forecaster")
		}
		// Calculate new lowest score with updated forecasterAddresses
		err = UpdateLowestScoreFromForecasterAddresses(ctx, k, topic.Id, forecasterAddresses, forecast.Forecaster, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error getting low score from all forecasts")
		}
		return k.InsertForecast(ctx, topic.Id, *forecast)
	} else {
		// Update EMA score for the current forecaster, who is the lowest score forecaster
		err = k.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving forecaster score ema with last saved topic quantile")
		}
	}
	return nil
}

// InsertForecast inserts a forecast for a specific topic
func (k *Keeper) InsertForecast(
	ctx context.Context,
	topicId TopicId,
	forecast types.Forecast,
) error {
	err := forecast.Validate()
	if err != nil {
		return errorsmod.Wrap(err, "forecast is invalid")
	}
	key := collections.Join(topicId, forecast.Forecaster)
	return k.forecasts.Set(ctx, key, forecast)
}

// Insert a complete set of forecasts for a topic/block.
func (k *Keeper) InsertActiveForecasts(
	ctx context.Context,
	topicId TopicId,
	nonceBlockHeight BlockHeight,
	forecasts types.Forecasts,
) error {
	key := collections.Join(topicId, nonceBlockHeight)
	return k.allForecasts.Set(ctx, key, forecasts)
}

func (k *Keeper) GetWorkerLatestInferenceByTopicId(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
) (types.Inference, error) {
	key := collections.Join(topicId, worker)
	return k.inferences.Get(ctx, key)
}

func (k *Keeper) GetWorkerLatestForecastByTopicId(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
) (types.Forecast, error) {
	key := collections.Join(topicId, worker)
	return k.forecasts.Get(ctx, key)
}

/// TOPIC REWARD NONCE

// GetTopicRewardNonce retrieves the reward nonce for a given topic ID.
func (k *Keeper) GetTopicRewardNonce(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	nonce, err := k.topicRewardNonce.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil // Return 0 if not found
		}
		return 0, errorsmod.Wrap(err, "error getting topic reward nonce")
	}
	return nonce, nil
}

// SetTopicRewardNonce sets the reward nonce for a given topic ID.
func (k *Keeper) SetTopicRewardNonce(ctx context.Context, topicId TopicId, nonce BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(nonce); err != nil {
		return errorsmod.Wrap(err, "nonce validation failed")
	}
	return k.topicRewardNonce.Set(ctx, topicId, nonce)
}

// DeleteTopicRewardNonce removes the reward nonce entry for a given topic ID.
func (k *Keeper) DeleteTopicRewardNonce(ctx context.Context, topicId TopicId) error {
	return k.topicRewardNonce.Remove(ctx, topicId)
}

/// LOSS BUNDLES

// Append loss bundle for a topic and blockHeight
func (k *Keeper) AppendReputerLoss(
	ctx sdk.Context,
	topic types.Topic,
	moduleParams types.Params,
	nonceBlockHeight BlockHeight,
	reputerLoss *types.ReputerValueBundle,
) error {
	if reputerLoss == nil || reputerLoss.ValueBundle == nil {
		return errors.New("invalid reputerLoss bundle: reputer is empty or nil")
	}

	// Check if the reputer already submitted the loss
	isActive, err := k.IsActiveReputer(ctx, topic.Id, reputerLoss.ValueBundle.Reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error checking if reputer already submitted loss")
	} else if isActive {
		return errors.New("reputer loss already submitted")
	}

	reputerAddresses, err := k.GetActiveReputersForTopic(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting active reputers for topic")
	}
	// If there are less than maxTopReputersToReward, add the current reputer
	if uint64(len(reputerAddresses)) < moduleParams.MaxTopReputersToReward {
		err := k.AddActiveReputer(ctx, topic.Id, reputerLoss.ValueBundle.Reputer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active reputer")
		}
		return k.InsertReputerLoss(ctx, topic.Id, *reputerLoss)
	}

	previousEmaScore, err := k.GetReputerScoreEma(ctx, topic.Id, reputerLoss.ValueBundle.Reputer)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting reputer score ema")
	}
	// Only calc and save if there's a new update
	if previousEmaScore.BlockHeight >= nonceBlockHeight {
		return types.ErrCantUpdateEmaMoreThanOncePerWindow
	}

	lowestEmaScore, found, err := k.GetLowestReputerScoreEma(ctx, topic.Id)
	if err != nil {
		return errorsmod.Wrap(err, "error getting lowest reputer score ema")
	} else if !found {
		lowestEmaScore, err = GetLowestScoreFromAllReputers(ctx, k, topic.Id, reputerAddresses)
		if err != nil {
			return errorsmod.Wrap(err, "error getting lowest score from all reputers")
		}
		err = k.SetLowestReputerScoreEma(ctx, topic.Id, lowestEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error setting lowest reputer score ema")
		}
	}

	if previousEmaScore.Score.Gt(lowestEmaScore.Score) {
		// Update EMA score for the lowest score reputer, who is not the current reputer
		err = k.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(
			ctx,
			topic,
			nonceBlockHeight,
			lowestEmaScore,
		)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving reputer score ema with last saved topic quantile")
		}
		// Remove reputer with lowest score
		err = k.RemoveActiveReputer(ctx, topic.Id, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error removing active reputer")
		}
		// Add new active reputer
		err = k.AddActiveReputer(ctx, topic.Id, reputerLoss.ValueBundle.Reputer)
		if err != nil {
			return errorsmod.Wrap(err, "error adding active reputer")
		}
		// Calculate new lowest score with updated reputerAddresses
		err = UpdateLowestScoreFromReputerAddresses(ctx, k, topic.Id, reputerAddresses, reputerLoss.ValueBundle.Reputer, lowestEmaScore.Address)
		if err != nil {
			return errorsmod.Wrap(err, "error getting low score from all reputer losses")
		}
		return k.InsertReputerLoss(ctx, topic.Id, *reputerLoss)
	} else {
		// Update EMA score for the current reputer, who is the lowest score reputer
		err = k.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(ctx, topic, nonceBlockHeight, previousEmaScore)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating and saving reputer score ema with last saved topic quantile")
		}
	}
	return nil
}

// GetReputerLatestLossByTopicId
func (k *Keeper) GetReputerLatestLossByTopicId(
	ctx context.Context,
	topicId TopicId,
	reputer ActorId,
) (types.ReputerValueBundle, error) {
	key := collections.Join(topicId, reputer)
	return k.reputerLosses.Get(ctx, key)
}

// InsertReputerLoss inserts a reputer loss for a specific topic
func (k *Keeper) InsertReputerLoss(
	ctx context.Context,
	topicId TopicId,
	reputerLoss types.ReputerValueBundle,
) error {
	err := reputerLoss.Validate()
	if err != nil {
		return errorsmod.Wrap(err, "reputer loss is invalid")
	}
	key := collections.Join(topicId, reputerLoss.ValueBundle.Reputer)
	return k.reputerLosses.Set(ctx, key, reputerLoss)
}

// InsertActiveReputerLosses inserts a complete set of reputer losses for a topic/block
func (k *Keeper) InsertActiveReputerLosses(
	ctx context.Context,
	topicId TopicId,
	nonceBlockHeight BlockHeight,
	reputerLosses types.ReputerValueBundles,
) error {
	key := collections.Join(topicId, nonceBlockHeight)
	return k.allLossBundles.Set(ctx, key, reputerLosses)
}

// Insert a loss bundle for a topic and timestamp but do it in the CloseReputerNonce, so reputer loss bundles are known to be validated
// and not validated again (this is due to poor data type design choices, see PROTO-2369)
func (k *Keeper) InsertKnownGoodReputerLossBundlesAtBlock(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
	reputerLossBundles types.ReputerValueBundles,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := reputerLossBundles.Validate(); err != nil {
		// in the singular case of a bundle that has been filtered, we allow
		// the signature validation to fail. This is secure because we already do
		// bundle validation in CloseReputerNonce before calling this function
		// however this should be well and truly fixed by PROTO-2369
		if !strings.Contains(err.Error(), "signature verification failed") {
			return errorsmod.Wrap(err, "reputer loss bundles validation failed")
		}
	}
	key := collections.Join(topicId, block)
	return k.allLossBundles.Set(ctx, key, reputerLossBundles)
}

// Get loss bundles for a topic/timestamp
func (k *Keeper) GetReputerLossBundlesAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ReputerValueBundles, error) {
	key := collections.Join(topicId, block)
	reputerLossBundles, err := k.allLossBundles.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.ReputerValueBundles{ReputerValueBundles: []*types.ReputerValueBundle{}}, nil
		}
		return nil, errorsmod.Wrap(err, "error getting reputer loss bundles at block")
	}
	return &reputerLossBundles, nil
}

// Insert a network loss bundle for a topic and block.
func (k *Keeper) InsertNetworkLossBundleAtBlock(
	ctx context.Context,
	topicId TopicId,
	block BlockHeight,
	lossBundle types.ValueBundle,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := lossBundle.Validate(); err != nil {
		return errorsmod.Wrap(err, "loss bundle validation failed")
	}
	key := collections.Join(topicId, block)
	return k.networkLossBundles.Set(ctx, key, lossBundle)
}

// A function that accepts a topicId and returns the network LossBundle at the block or error
func (k *Keeper) GetNetworkLossBundleAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (*types.ValueBundle, error) {
	key := collections.Join(topicId, block)
	lossBundle, err := k.networkLossBundles.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.ValueBundle{
				TopicId: topicId,
				ReputerRequestNonce: &types.ReputerRequestNonce{
					ReputerNonce: &types.Nonce{
						BlockHeight: 0,
					},
				},
				Reputer:                       "",
				ExtraData:                     nil,
				CombinedValue:                 alloraMath.ZeroDec(),
				InfererValues:                 nil,
				ForecasterValues:              nil,
				NaiveValue:                    alloraMath.ZeroDec(),
				OneOutInfererValues:           nil,
				OneOutForecasterValues:        nil,
				OneInForecasterValues:         nil,
				OneOutInfererForecasterValues: nil,
			}, nil
		}
		return nil, errorsmod.Wrap(err, "error getting network loss bundle at block")
	}
	return &lossBundle, nil
}

// Returns the latest network loss bundle for a given topic id.
func (k *Keeper) GetLatestNetworkLossBundle(ctx context.Context, topicId TopicId) (*types.ValueBundle, error) {
	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).Descending()
	iter, err := k.networkLossBundles.Iterate(ctx, rng)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error iterating over network loss bundles")
	}
	defer iter.Close()

	if iter.Valid() {
		keyValue, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}
		return &keyValue.Value, nil
	}

	return nil, types.ErrNotFound
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
		return errorsmod.Wrap(err, "error getting reputer authority")
	}
	reputerAuthorityNew := reputerAuthority.Add(stakeToAdd)
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic stake")
	}
	topicStakeNew := topicStake.Add(stakeToAdd)
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting total stake")
	}
	totalStakeNew := totalStake.Add(stakeToAdd)

	// SET NEW VALUES
	if err := k.SetStakeReputerAuthority(ctx, topicId, reputer, reputerAuthorityNew); err != nil {
		return errorsmod.Wrap(err, "error setting reputer authority")
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
		return errorsmod.Wrap(err, "error getting total stake")
	}
	totalStakeNew := totalStake.Add(stakeToAdd)
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic stake")
	}
	topicStakeNew := topicStake.Add(stakeToAdd)
	stakeReputerAuthority, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting reputer authority")
	}
	stakeReputerAuthorityNew := stakeReputerAuthority.Add(stakeToAdd)
	stakeSumFromDelegator, err := k.GetStakeFromDelegatorInTopic(ctx, topicId, delegator)
	if err != nil {
		return errorsmod.Wrap(err, "error getting stake from delegator in topic")
	}
	stakeSumFromDelegatorNew := stakeSumFromDelegator.Add(stakeToAdd)
	delegateStakePlacement, err := k.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate stake placement")
	}
	share, err := k.GetDelegateRewardPerShare(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate reward per share")
	}
	if delegateStakePlacement.Amount.Gt(alloraMath.NewDecFromInt64(0)) {
		// Calculate pending reward and send to delegator
		pendingReward, err := delegateStakePlacement.Amount.Mul(share)
		if err != nil {
			return errorsmod.Wrap(err, "error calculating pending reward")
		}
		pendingReward, err = pendingReward.Sub(delegateStakePlacement.RewardDebt)
		if err != nil {
			return errorsmod.Wrap(err, "error subtracting reward debt from pending reward")
		}
		if pendingReward.Gt(alloraMath.NewDecFromInt64(0)) {
			pendingRewardInt, err := pendingReward.SdkIntTrim()
			if err != nil {
				return errorsmod.Wrap(err, "error trimming pending reward")
			}
			err = k.SendCoinsFromModuleToAccount(
				ctx,
				types.AlloraPendingRewardForDelegatorAccountName,
				delegator,
				sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, pendingRewardInt)),
			)
			if err != nil {
				return errorsmod.Wrap(err, "error sending pending reward to delegator")
			}
		}
	}
	stakeToAddDec, err := alloraMath.NewDecFromSdkInt(stakeToAdd)
	if err != nil {
		return errorsmod.Wrap(err, "error creating new amount from stake to add")
	}
	newAmount, err := delegateStakePlacement.Amount.Add(stakeToAddDec)
	if err != nil {
		return errorsmod.Wrap(err, "error adding stake to add to delegate stake placement amount")
	}
	newDebt, err := newAmount.Mul(share)
	if err != nil {
		return errorsmod.Wrap(err, "error multiplying new amount by share")
	}
	stakePlacementNew := types.DelegatorInfo{
		Amount:     newAmount,
		RewardDebt: newDebt,
	}
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate stake upon reputer")
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
		return errorsmod.Wrap(err, "error getting reputer authority")
	}
	delegateStakeUponReputerInTopic, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate stake upon reputer in topic")
	}
	reputerStakeInTopicWithoutDelegateStake := reputerAuthority.Sub(delegateStakeUponReputerInTopic)
	if stakeToRemove.GT(reputerStakeInTopicWithoutDelegateStake) {
		return types.ErrIntegerUnderflowTopicReputerStake
	}
	reputerStakeNew := reputerAuthority.Sub(stakeToRemove)

	// Check topicStake >= stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic stake")
	}
	if stakeToRemove.GT(topicStake) {
		return types.ErrIntegerUnderflowTopicStake
	}
	topicStakeNew := topicStake.Sub(stakeToRemove)

	// Check totalStake >= stake
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting total stake")
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
	stakeRemovalBlockHeight BlockHeight,
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
		return errorsmod.Wrap(err, "error getting stake from delegator in topic")
	}
	if stakeToRemove.GT(stakeSumFromDelegator) {
		return types.ErrIntegerUnderflowStakeFromDelegator
	}
	stakeFromDelegatorNew := stakeSumFromDelegator.Sub(stakeToRemove)

	// delegatedStakePlacement >= stake
	delegatedStakePlacement, err := k.GetDelegateStakePlacement(ctx, topicId, delegator, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate stake placement")
	}
	unStakeDec, err := alloraMath.NewDecFromSdkInt(stakeToRemove)
	if err != nil {
		return errorsmod.Wrap(err, "error creating new amount from stake to remove")
	}
	if delegatedStakePlacement.Amount.Lt(unStakeDec) {
		return types.ErrIntegerUnderflowDelegateStakePlacement
	}

	// Get share for this topicId and reputer
	share, err := k.GetDelegateRewardPerShare(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate reward per share")
	}

	// Calculate pending reward and send to delegator
	pendingReward, err := delegatedStakePlacement.Amount.Mul(share)
	if err != nil {
		return errorsmod.Wrap(err, "error multiplying delegated stake placement amount by share")
	}
	pendingReward, err = pendingReward.Sub(delegatedStakePlacement.RewardDebt)
	if err != nil {
		return errorsmod.Wrap(err, "error subtracting reward debt from pending reward")
	}
	if pendingReward.Gt(alloraMath.NewDecFromInt64(0)) {
		pendingRewardInt, err := pendingReward.SdkIntTrim()
		if err != nil {
			return errorsmod.Wrap(err, "error trimming pending reward")
		}
		err = k.SendCoinsFromModuleToAccount(
			ctx,
			types.AlloraPendingRewardForDelegatorAccountName,
			delegator,
			sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, pendingRewardInt)),
		)
		if err != nil {
			return errorsmod.Wrapf(err, "Sending pending reward to delegator failed")
		}
	}

	newAmount, err := delegatedStakePlacement.Amount.Sub(unStakeDec)
	if err != nil {
		return errorsmod.Wrap(err, "error subtracting stake to remove from delegated stake placement amount")
	}
	newRewardDebt, err := newAmount.Mul(share)
	if err != nil {
		return errorsmod.Wrap(err, "error multiplying new amount by share")
	}
	stakePlacementNew := types.DelegatorInfo{
		Amount:     newAmount,
		RewardDebt: newRewardDebt,
	}

	// stakeUponReputer >= stake
	stakeUponReputer, err := k.GetDelegateStakeUponReputer(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting delegate stake upon reputer")
	}
	if stakeToRemove.GT(stakeUponReputer) {
		return types.ErrIntegerUnderflowDelegateStakeUponReputer
	}
	stakeUponReputerNew := stakeUponReputer.Sub(stakeToRemove)

	// stakeReputerAuthority >= stake
	stakeReputerAuthority, err := k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return errorsmod.Wrap(err, "error getting reputer authority")
	}
	if stakeToRemove.GT(stakeReputerAuthority) {
		return types.ErrIntegerUnderflowReputerStakeAuthority
	}
	stakeReputerAuthorityNew := stakeReputerAuthority.Sub(stakeToRemove)

	// topicStake >= stake
	topicStake, err := k.GetTopicStake(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic stake")
	}
	if stakeToRemove.GT(topicStake) {
		return types.ErrIntegerUnderflowTopicStake
	}
	topicStakeNew := topicStake.Sub(stakeToRemove)

	// totalStake >= stake
	totalStake, err := k.GetTotalStake(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting total stake")
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
	if err := k.DeleteDelegateStakeRemoval(ctx, stakeRemovalBlockHeight, topicId, reputer, delegator); err != nil {
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
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting total stake")
	}
	return ret, nil
}

// Sets the total sum of all stake in the network across all topics
func (k *Keeper) SetTotalStake(ctx context.Context, totalStake cosmosMath.Int) error {
	// big int pointer inside the cosmos int must be non nil
	if err := types.ValidateSdkIntRepresentingMonetaryValue(totalStake); err != nil {
		return errorsmod.Wrap(err, "totalStake cosmos Int is not valid")
	}
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
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting topic stake")
	}
	return ret, nil
}

// sets the cumulative amount of stake in a topic
func (k *Keeper) SetTopicStake(ctx context.Context, topicId TopicId, stake cosmosMath.Int) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topicId is not valid")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(stake); err != nil {
		return errorsmod.Wrap(err, "stake cosmos Int is not valid")
	}
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
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting stake reputer authority")
	}
	return stake, nil
}

// Sets the amount of stake placed upon a reputer in addition to their personal stake on a specific topic
// Includes the stake placed by delegators on the reputer in that topic.
func (k *Keeper) SetStakeReputerAuthority(ctx context.Context, topicId TopicId, reputer ActorId, amount cosmosMath.Int) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topicId is not valid")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrap(err, "reputer is not valid")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(amount); err != nil {
		return errorsmod.Wrap(err, "amount is not valid")
	}
	key := collections.Join(topicId, reputer)
	if amount.IsNil() || amount.IsZero() {
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
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting stake from delegator in topic")
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator.
func (k *Keeper) SetStakeFromDelegator(ctx context.Context, topicId TopicId, delegator ActorId, stake cosmosMath.Int) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topicId is not valid")
	}
	if err := types.ValidateBech32(delegator); err != nil {
		return errorsmod.Wrap(err, "delegator is not valid")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(stake); err != nil {
		return errorsmod.Wrap(err, "stake is not valid")
	}
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
		return types.DelegatorInfo{}, errorsmod.Wrap(err, "error getting delegate stake placement")
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator on a specific target.
func (k *Keeper) SetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator ActorId, target ActorId, stake types.DelegatorInfo) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topicId is not valid")
	}
	if err := types.ValidateBech32(delegator); err != nil {
		return errorsmod.Wrap(err, "delegator is not valid")
	}
	if err := types.ValidateBech32(target); err != nil {
		return errorsmod.Wrap(err, "target is not valid")
	}
	if err := stake.Validate(); err != nil {
		return errorsmod.Wrap(err, "stake information is not valid")
	}
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
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting delegate reward per share")
	}
	return share, nil
}

// Set the share on specific reputer and topicId
func (k *Keeper) SetDelegateRewardPerShare(ctx context.Context, topicId TopicId, reputer ActorId, share alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topicId is not valid")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrap(err, "reputer is not valid")
	}
	if err := types.ValidateDec(share); err != nil {
		return errorsmod.Wrap(err, "share is not valid")
	}
	key := collections.Join(topicId, reputer)
	if err := types.ValidateDec(share); err != nil { // Added error check
		return errorsmod.Wrapf(err, "SetDelegateRewardPerShare: invalid share")
	}
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
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting delegate stake upon reputer")
	}
	return stake, nil
}

// Sets the amount of stake placed on a specific target.
func (k *Keeper) SetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target ActorId, stake cosmosMath.Int) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topicId is not valid")
	}
	if err := types.ValidateBech32(target); err != nil {
		return errorsmod.Wrap(err, "target is not valid")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(stake); err != nil {
		return errorsmod.Wrap(err, "stake is not valid")
	}
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
	if err := removalInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "removalInfo is not valid")
	}
	byBlockKey := collections.Join3(removalInfo.BlockRemovalCompleted, removalInfo.TopicId, removalInfo.Reputer)
	err := k.stakeRemovalsByBlock.Set(ctx, byBlockKey, removalInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting stake removal by block")
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
		return errorsmod.Wrap(err, "error checking if stake removal by block exists")
	}
	if !has {
		return types.ErrStakeRemovalNotFound
	}
	err = k.stakeRemovalsByBlock.Remove(ctx, byBlockKey)
	if err != nil {
		return errorsmod.Wrap(err, "error removing stake removal by block")
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

// get a list of stake removals that are valid for removal
// before and including this block.
func (k *Keeper) GetStakeRemovalsUpUntilBlock(
	ctx context.Context,
	blockHeight BlockHeight,
	limit uint64,
) (ret []types.StakeRemovalInfo, anyLeft bool, err error) {
	ret = make([]types.StakeRemovalInfo, 0)
	// make a range that has everything less than the block height, inclusive
	startKey := collections.TriplePrefix[BlockHeight, TopicId, ActorId](0)
	rng := &collections.Range[collections.Triple[BlockHeight, TopicId, ActorId]]{}
	rng = rng.Prefix(startKey)
	// +1 for end exclusive. Don't know why end inclusive is being buggy but it is
	endKey := collections.TriplePrefix[BlockHeight, TopicId, ActorId](blockHeight + 1)
	rng = rng.EndExclusive(endKey)

	iter, err := k.stakeRemovalsByBlock.Iterate(ctx, rng)
	if err != nil {
		return ret, false, errorsmod.Wrap(err, "error iterating over stake removals by block")
	}
	defer iter.Close()
	count := uint64(0)
	for ; iter.Valid(); iter.Next() {
		if count >= limit {
			return ret, true, nil
		}
		val, err := iter.Value()
		if err != nil {
			return ret, true, errorsmod.Wrap(err, "error getting stake removal by block")
		}
		ret = append(ret, val)
		count += 1
	}
	return ret, false, nil
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
		return types.StakeRemovalInfo{}, false, errorsmod.Wrap(err, "error iterating over stake removals by actor")
	}
	defer iter.Close()
	keys, err := iter.Keys()
	if err != nil {
		return types.StakeRemovalInfo{}, false, errorsmod.Wrap(err, "error getting keys")
	}
	keysLen := len(keys)
	if keysLen == 0 {
		return types.StakeRemovalInfo{
			BlockRemovalStarted:   0,
			TopicId:               0,
			Reputer:               "",
			Amount:                cosmosMath.ZeroInt(),
			BlockRemovalCompleted: 0,
		}, false, nil
	}
	if keysLen < 0 {
		return types.StakeRemovalInfo{}, false, errorsmod.Wrapf(types.ErrInvariantFailure, "Why is golang len function returning negative values?")
	}
	key := keys[0]
	byBlockKey := collections.Join3(key.K3(), topicId, reputer)
	ret, err := k.stakeRemovalsByBlock.Get(ctx, byBlockKey)
	if err != nil {
		return types.StakeRemovalInfo{}, false, errorsmod.Wrap(err, "error getting stake removal by block")
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
	if err := removalInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "removalInfo is not valid")
	}
	byBlockKey := Join4(removalInfo.BlockRemovalCompleted, removalInfo.TopicId, removalInfo.Delegator, removalInfo.Reputer)
	err := k.delegateStakeRemovalsByBlock.Set(ctx, byBlockKey, removalInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting delegate stake removal by block")
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
		return errorsmod.Wrap(err, "error checking if delegate stake removal by block exists")
	}
	if !has {
		return types.ErrStakeRemovalNotFound
	}
	err = k.delegateStakeRemovalsByBlock.Remove(ctx, byBlockKey)
	if err != nil {
		return errorsmod.Wrap(err, "error removing delegate stake removal by block")
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

// get a list of stake removals that are valid for removal
// before and including this block.
func (k *Keeper) GetDelegateStakeRemovalsUpUntilBlock(
	ctx context.Context,
	blockHeight BlockHeight,
	limit uint64,
) (ret []types.DelegateStakeRemovalInfo, limitHit bool, err error) {
	ret = make([]types.DelegateStakeRemovalInfo, 0)

	// make a range that has everything less than the block height, inclusive
	startKey := QuadrupleSinglePrefix[BlockHeight, TopicId, ActorId, ActorId](0)
	rng := &collections.Range[Quadruple[BlockHeight, TopicId, ActorId, ActorId]]{}
	rng = rng.Prefix(startKey)
	endKey := QuadrupleSinglePrefix[BlockHeight, TopicId, ActorId, ActorId](blockHeight + 1)
	rng = rng.EndExclusive(endKey)

	iter, err := k.delegateStakeRemovalsByBlock.Iterate(ctx, rng)
	if err != nil {
		return ret, false, errorsmod.Wrap(err, "error iterating over delegate stake removals by block")
	}
	defer iter.Close()
	count := uint64(0)
	for ; iter.Valid(); iter.Next() {
		if count >= limit {
			return ret, true, nil
		}
		val, err := iter.Value()
		if err != nil {
			return ret, true, errorsmod.Wrap(err, "error getting delegate stake removal by block")
		}
		ret = append(ret, val)
		count += 1
	}
	return ret, false, nil
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
		return types.DelegateStakeRemovalInfo{}, false, errorsmod.Wrap(err, "error iterating over delegate stake removals by actor")
	}
	keys, err := iter.Keys()
	if err != nil {
		return types.DelegateStakeRemovalInfo{}, false, errorsmod.Wrap(err, "error getting keys")
	}
	keysLen := len(keys)
	if keysLen == 0 {
		return types.DelegateStakeRemovalInfo{
			BlockRemovalStarted:   0,
			TopicId:               0,
			Delegator:             "",
			Reputer:               "",
			Amount:                cosmosMath.ZeroInt(),
			BlockRemovalCompleted: 0,
		}, false, nil
	}
	if keysLen < 0 {
		return types.DelegateStakeRemovalInfo{}, false, errorsmod.Wrapf(types.ErrInvariantFailure, "Why is golang len function returning negative values?")
	}
	key := keys[0]
	byBlockKey := Join4(key.K4(), topicId, delegator, reputer)
	ret, err := k.delegateStakeRemovalsByBlock.Get(ctx, byBlockKey)
	if err != nil {
		return types.DelegateStakeRemovalInfo{}, false, errorsmod.Wrap(err, "error getting delegate stake removal by block")
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
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrap(err, "reputer validation failed")
	}
	if err := reputerInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "reputer info validation failed")
	}
	topicKey := collections.Join(topicId, reputer)
	err := k.topicReputers.Set(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error setting topic reputer")
	}
	err = k.reputers.Set(ctx, reputer, reputerInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting reputer")
	}
	return nil
}

// Remove a reputer to the reputer tracking data structures and topicReputers
func (k *Keeper) RemoveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	topicKey := collections.Join(topicId, reputer)
	err := k.topicReputers.Remove(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error removing topic reputer")
	}
	return nil
}

func (k *Keeper) GetReputerInfo(ctx sdk.Context, reputerKey ActorId) (types.OffchainNode, error) {
	return k.reputers.Get(ctx, reputerKey)
}

/// WORKERS

// Adds a new worker to the worker tracking data structures, workers and topicWorkers
func (k *Keeper) InsertWorker(ctx context.Context, topicId TopicId, worker ActorId, workerInfo types.OffchainNode) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "worker validation failed")
	}
	if err := workerInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "worker info validation failed")
	}
	topicKey := collections.Join(topicId, worker)
	err := k.topicWorkers.Set(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error setting topic worker")
	}
	err = k.workers.Set(ctx, worker, workerInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting worker")
	}
	return nil
}

// Remove a worker to the worker tracking data structures and topicWorkers
func (k *Keeper) RemoveWorker(ctx context.Context, topicId TopicId, worker ActorId) error {
	topicKey := collections.Join(topicId, worker)
	err := k.topicWorkers.Remove(ctx, topicKey)
	if err != nil {
		return errorsmod.Wrap(err, "error removing topic worker")
	}
	return nil
}

func (k *Keeper) GetWorkerInfo(ctx sdk.Context, workerKey ActorId) (types.OffchainNode, error) {
	return k.workers.Get(ctx, workerKey)
}

/// TOPICS

// Get the previous weight during rewards calculation for a topic
// Returns ((0,0), true) if there was no prior topic weight set, else ((x,y), false) where x,y!=0
func (k *Keeper) GetPreviousTopicWeight(ctx context.Context, topicId TopicId) (topicWeight alloraMath.Dec, noPrior bool, err error) {
	topicWeight, err = k.previousTopicWeight.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.ZeroDec(), false, errorsmod.Wrap(err, "error getting previous topic weight")
	}
	return topicWeight, false, nil
}

// Set the previous weight during rewards calculation for a topic
func (k *Keeper) SetPreviousTopicWeight(ctx context.Context, topicId TopicId, weight alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateDec(weight); err != nil {
		return errorsmod.Wrap(err, "weight validation failed")
	}
	return k.previousTopicWeight.Set(ctx, topicId, weight)
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
	params, err := k.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}
	if err := topic.Validate(params); err != nil {
		return errorsmod.Wrap(err, "set topic validation failure")
	}
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

// Check if the topic is activated or not
func (k *Keeper) IsTopicActive(ctx context.Context, topicId TopicId) (bool, error) {
	_, active, err := k.GetNextPossibleChurningBlockByTopicId(ctx, topicId)
	if err != nil {
		return false, errorsmod.Wrap(err, "error getting next possible churning block by topic id")
	}

	return active, nil
}

// UpdateTopicInitialRegret updates the InitialRegret for a given topic.
func (k *Keeper) UpdateTopicInitialRegret(ctx context.Context, topicId TopicId, initialRegret alloraMath.Dec) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic")
	}
	topic.InitialRegret = initialRegret
	return k.SetTopic(ctx, topicId, topic)
}

// UpdateTopicInferenceLastRan updates the InferenceLastRan timestamp for a given topic.
func (k *Keeper) UpdateTopicEpochLastEnded(ctx context.Context, topicId TopicId, epochLastEnded BlockHeight) error {
	topic, err := k.topics.Get(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic")
	}
	topic.EpochLastEnded = epochLastEnded
	return k.SetTopic(ctx, topicId, topic)
}

// True if worker is registered in topic, else False
func (k *Keeper) IsWorkerRegisteredInTopic(ctx context.Context, topicId TopicId, worker ActorId) (bool, error) {
	topicKey := collections.Join(topicId, worker)
	return k.topicWorkers.Has(ctx, topicKey)
}

// True if reputer is registered in topic, else False
func (k *Keeper) IsReputerRegisteredInTopic(ctx context.Context, topicId TopicId, reputer ActorId) (bool, error) {
	topicKey := collections.Join(topicId, reputer)
	return k.topicReputers.Has(ctx, topicKey)
}

// wrapper for set operation around activeTopics
func (k *Keeper) SetActiveTopics(ctx context.Context, topicId TopicId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	return k.activeTopics.Set(ctx, topicId)
}

// wrapper for set operation around blockToActiveTopics
func (k *Keeper) SetBlockToActiveTopics(ctx context.Context, block BlockHeight, topicIds types.TopicIds) error {
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	for _, topicId := range topicIds.TopicIds {
		if err := types.ValidateTopicId(topicId); err != nil {
			return errorsmod.Wrap(err, "topic id validation failed")
		}
	}
	return k.blockToActiveTopics.Set(ctx, block, topicIds)
}

// wrapper for set operation around topicToNextPossibleChurningBlock
func (k *Keeper) SetTopicToNextPossibleChurningBlock(ctx context.Context, topicId TopicId, block BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	return k.topicToNextPossibleChurningBlock.Set(ctx, topicId, block)
}

// wrapper for set operation around blockToLowestActiveTopicWeight
func (k *Keeper) SetBlockToLowestActiveTopicWeight(
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

/// TOPIC FEE REVENUE

// Get the amount of fee revenue collected by a topic
func (k *Keeper) GetTopicFeeRevenue(ctx context.Context, topicId TopicId) (cosmosMath.Int, error) {
	feeRev, err := k.topicFeeRevenue.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return cosmosMath.ZeroInt(), nil
		}
		return cosmosMath.ZeroInt(), errorsmod.Wrap(err, "error getting topic fee revenue")
	}
	return feeRev, nil
}

// Add to the fee revenue collected by a topic
func (k *Keeper) AddTopicFeeRevenue(ctx context.Context, topicId TopicId, amount cosmosMath.Int) error {
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
	return k.topicFeeRevenue.Set(ctx, topicId, topicFeeRevenue)
}

// return the blocks per week
// defined as the blocks per month divided by 4.345
func calculateBlocksPerWeek(ctx sdk.Context, k Keeper) (alloraMath.Dec, error) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting params")
	}
	blocksPerMonth, err := alloraMath.NewDecFromUint64(moduleParams.BlocksPerMonth)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error creating blocks per month")
	}
	// 4.345 weeks per month on average
	weeksPerMonth, err := alloraMath.NewDecFromString("4.345")
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error creating weeks per month")
	}
	blocksPerWeek, err := blocksPerMonth.Quo(weeksPerMonth)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error calculating blocks per week")
	}
	return blocksPerWeek, nil
}

// return the last time we dripped the fee revenue for a topic
func (k *Keeper) GetLastDripBlock(ctx context.Context, topicId TopicId) (BlockHeight, error) {
	bh, err := k.lastDripBlock.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil
		}
		return 0, errorsmod.Wrap(err, "error getting last drip block")
	}
	return bh, nil
}

// set the last time we dripped the fee revenue for a topic
func (k *Keeper) SetLastDripBlock(ctx context.Context, topicId TopicId, block BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	return k.lastDripBlock.Set(ctx, topicId, block)
}

// Drop the fee revenue by the global Ecosystem bucket drip amount
// in the paper we say that
// ∆ C_{t,i} = N_{epochs,w} * C_{t,i}
// where C_{t,i} is the topic fee revenue
// and N_{epochs,w} is the number of epochs per week
// and this decay or drip happens each epoch
func (k *Keeper) DripTopicFeeRevenue(ctx sdk.Context, topicId TopicId, block BlockHeight) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(block); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	topicFeeRevenue, err := k.GetTopicFeeRevenue(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic fee revenue")
	}
	topicFeeRevenueDec, err := alloraMath.NewDecFromSdkInt(topicFeeRevenue)
	if err != nil {
		return errorsmod.Wrap(err, "error creating decimal from sdk int")
	}
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting topic")
	}
	blocksPerEpoch := alloraMath.NewDecFromInt64(topic.EpochLength)
	if blocksPerEpoch.IsZero() {
		ctx.Logger().Warn(fmt.Sprintf("Blocks per epoch is zero for topic %d. Skipping fee revenue drip.", topicId))
		return nil
	}
	blocksPerWeek, err := calculateBlocksPerWeek(ctx, *k)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating blocks per week")
	}
	epochsPerWeek, err := blocksPerWeek.Quo(blocksPerEpoch)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating epochs per week")
	}
	if epochsPerWeek.IsZero() {
		// Log a warning
		ctx.Logger().Warn(fmt.Sprintf("Epochs per week is zero for topic %d. Skipping fee revenue drip.", topicId))
		return nil
	}
	// this delta is the drip per epoch
	dripPerEpoch, err := topicFeeRevenueDec.Quo(epochsPerWeek)
	if err != nil {
		return errorsmod.Wrap(err, "error calculating drip per epoch")
	}
	lastDripBlock, err := k.GetLastDripBlock(ctx, topicId)
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

		if err = k.SetLastDripBlock(ctx, topicId, topic.EpochLastEnded); err != nil {
			return errorsmod.Wrap(err, "error setting last drip block")
		}
		ctx.Logger().Debug("Dripping topic fee revenue",
			"block", ctx.BlockHeight(),
			"topicId", topicId,
			"oldRevenue", topicFeeRevenue,
			"newRevenue", newTopicFeeRevenue)
		if err = types.ValidateSdkIntRepresentingMonetaryValue(newTopicFeeRevenue); err != nil {
			return errorsmod.Wrap(err, "error validating new topic fee revenue")
		}
		return k.topicFeeRevenue.Set(ctx, topicId, newTopicFeeRevenue)
	}
	return nil
}

/// SCORES

// If the new score is older than the current score, don't update
func (k *Keeper) SetInfererScoreEma(ctx context.Context, topicId TopicId, worker ActorId, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetInfererScoreEma: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetInfererScoreEma: Error validating worker")
	}
	if err := score.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetInfererScoreEma: Error validating inferer score")
	}
	key := collections.Join(topicId, worker)
	return k.infererScoreEmas.Set(ctx, key, score)
}

func (k *Keeper) GetInfererScoreEma(ctx context.Context, topicId TopicId, worker ActorId) (types.Score, error) {
	key := collections.Join(topicId, worker)
	score, err := k.infererScoreEmas.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{
				BlockHeight: 0,
				Address:     worker,
				TopicId:     topicId,
				Score:       alloraMath.ZeroDec(),
			}, nil
		}
		return types.Score{}, errorsmod.Wrap(err, "error getting inferer score ema")
	}
	return score, nil
}

func (k *Keeper) SetForecasterScoreEma(ctx context.Context, topicId TopicId, worker ActorId, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetForecasterScoreEma: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetForecasterScoreEma: Error validating worker")
	}
	if err := score.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetForecasterScoreEma: Error validating forecaster score")
	}
	key := collections.Join(topicId, worker)
	return k.forecasterScoreEmas.Set(ctx, key, score)
}

func (k *Keeper) GetForecasterScoreEma(ctx context.Context, topicId TopicId, worker ActorId) (types.Score, error) {
	key := collections.Join(topicId, worker)
	score, err := k.forecasterScoreEmas.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{
				BlockHeight: 0,
				Address:     worker,
				TopicId:     topicId,
				Score:       alloraMath.ZeroDec(),
			}, nil
		}
		return types.Score{}, errorsmod.Wrap(err, "error getting forecaster score ema")
	}
	return score, nil
}

// If the new score is older than the current score, don't update
func (k *Keeper) SetReputerScoreEma(ctx context.Context, topicId TopicId, reputer ActorId, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetReputerScoreEma: Error validating topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrapf(err, "SetReputerScoreEma: Error validating reputer")
	}
	if err := score.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetReputerScoreEma: Error validating reputer score")
	}
	key := collections.Join(topicId, reputer)
	return k.reputerScoreEmas.Set(ctx, key, score)
}

func (k *Keeper) GetReputerScoreEma(ctx context.Context, topicId TopicId, reputer ActorId) (types.Score, error) {
	key := collections.Join(topicId, reputer)
	score, err := k.reputerScoreEmas.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{
				BlockHeight: 0,
				Address:     reputer,
				TopicId:     topicId,
				Score:       alloraMath.ZeroDec(),
			}, nil
		}
		return types.Score{
			BlockHeight: 0,
			Address:     reputer,
			TopicId:     topicId,
			Score:       alloraMath.ZeroDec(),
		}, errorsmod.Wrap(err, "error getting reputer score ema")
	}
	return score, nil
}

func (k *Keeper) InsertWorkerInferenceScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerInferenceScore: Error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerInferenceScore: Error validating block height")
	}
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting worker inference scores at block")
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "Error getting params")
	}
	maxNumScores := moduleParams.MaxSamplesToScaleScores

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockHeight)
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerInferenceScore: Error validating worker inference scores")
	}
	return k.infererScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetInferenceScoresUntilBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight).
		Descending()

	iter, err := k.infererScoresByBlock.Iterate(ctx, rng)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error iterating inferer scores by block")
	}
	defer iter.Close()

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting params")
	}
	maxNumTimeSteps := moduleParams.MaxSamplesToScaleScores

	scores := make([]*types.Score, 0, maxNumTimeSteps)

	for iter.Valid() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}

		for _, score := range existingScores.Value.Scores {
			if uint64(len(scores)) < maxNumTimeSteps {
				scores = append(scores, score)
			} else {
				break
			}
		}
		if uint64(len(scores)) >= maxNumTimeSteps {
			break
		}
		iter.Next()
	}

	return scores, nil
}

func (k *Keeper) GetWorkerInferenceScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.infererScoresByBlock.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Scores{
				Scores: []*types.Score{},
			}, nil
		}
		return types.Scores{}, errorsmod.Wrap(err, "error getting worker inference scores at block")
	}
	return scores, nil
}

func (k *Keeper) InsertWorkerForecastScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerForecastScore: Error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerForecastScore: Error validating block height")
	}
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrap(err, "error getting worker forecast scores at block")
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
	}
	maxNumScores := moduleParams.MaxSamplesToScaleScores

	lenScores := uint64(len(scores.Scores))
	if lenScores > maxNumScores {
		diff := lenScores - maxNumScores
		scores.Scores = scores.Scores[diff:]
	}

	key := collections.Join(topicId, blockHeight)
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrapf(err, "InsertWorkerForecastScore: Error validating worker forecast scores")
	}
	return k.forecasterScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetForecastScoresUntilBlock(ctx context.Context, topicId TopicId, blockHeight BlockHeight) ([]*types.Score, error) {
	rng := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight).
		Descending()

	iter, err := k.forecasterScoresByBlock.Iterate(ctx, rng)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error iterating forecaster scores by block")
	}
	defer iter.Close()

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error getting params")
	}
	maxNumTimeSteps := moduleParams.MaxSamplesToScaleScores

	scores := make([]*types.Score, 0, maxNumTimeSteps)

	for iter.Valid() {
		existingScores, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}

		for _, score := range existingScores.Value.Scores {
			if uint64(len(scores)) < maxNumTimeSteps {
				scores = append(scores, score)
			} else {
				break
			}
		}
		if uint64(len(scores)) >= maxNumTimeSteps {
			break
		}
		iter.Next()
	}

	return scores, nil
}

func (k *Keeper) GetWorkerForecastScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.forecasterScoresByBlock.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Scores{Scores: []*types.Score{}}, nil
		}
		return types.Scores{}, errorsmod.Wrap(err, "error getting worker forecast scores at block")
	}
	return scores, nil
}

func (k *Keeper) InsertReputerScore(ctx context.Context, topicId TopicId, blockHeight BlockHeight, score types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "InsertReputerScore: Error validating topic id")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrapf(err, "InsertReputerScore: Error validating block height")
	}
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrap(err, "error getting reputers scores at block")
	}
	scores.Scores = append(scores.Scores, &score)

	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error getting params")
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
	if err := scores.Validate(); err != nil {
		return errorsmod.Wrapf(err, "InsertReputerScore: Error validating reputer scores")
	}
	return k.reputerScoresByBlock.Set(ctx, key, scores)
}

func (k *Keeper) GetReputersScoresAtBlock(ctx context.Context, topicId TopicId, block BlockHeight) (types.Scores, error) {
	key := collections.Join(topicId, block)
	scores, err := k.reputerScoresByBlock.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Scores{Scores: []*types.Score{}}, nil
		}
		return types.Scores{}, errorsmod.Wrap(err, "error getting reputers scores at block")
	}
	return scores, nil
}

func (k *Keeper) SetListeningCoefficient(ctx context.Context, topicId TopicId, reputer ActorId, coefficient types.ListeningCoefficient) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetListeningCoefficient: Error validating topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrapf(err, "SetListeningCoefficient: Error validating reputer")
	}
	if err := coefficient.Validate(); err != nil {
		return errorsmod.Wrapf(err, "SetListeningCoefficient: Error validating listening coefficient")
	}
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
		return types.ListeningCoefficient{}, errorsmod.Wrap(err, "error getting listening coefficient")
	}
	return coef, nil
}

func (k *Keeper) SetPreviousTopicQuantileInfererScoreEma(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileInfererScoreEma: Error validating topic id")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileInfererScoreEma: Error validating score")
	}
	return k.previousTopicQuantileInfererScoreEma.Set(ctx, topicId, score)
}

// Gets the previous Inferer Score Ema at Topic quantile
// Returns previous inferer score ema at topic quantile, or 0 if not yet seen
func (k *Keeper) GetPreviousTopicQuantileInfererScoreEma(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.previousTopicQuantileInfererScoreEma.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), nil
		}
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous topic quantile inferer score ema")
	}
	return score, nil
}

func (k *Keeper) SetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileForecasterScoreEma: Error validating topic id")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileForecasterScoreEma: Error validating score")
	}
	return k.previousTopicQuantileForecasterScoreEma.Set(ctx, topicId, score)
}

// Gets the previous Forecaster Score Ema at Topic quantile
// Returns previous forecaster score ema at topic quantile, or 0 if not yet seen
func (k *Keeper) GetPreviousTopicQuantileForecasterScoreEma(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.previousTopicQuantileForecasterScoreEma.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), nil
		}
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous topic quantile forecaster score ema")
	}
	return score, nil
}

func (k *Keeper) SetPreviousTopicQuantileReputerScoreEma(ctx context.Context, topicId TopicId, score alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileReputerScoreEma: Error validating topic id")
	}
	if err := types.ValidateDec(score); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousTopicQuantileReputerScoreEma: Error validating score")
	}
	return k.previousTopicQuantileReputerScoreEma.Set(ctx, topicId, score)
}

// Gets the previous Reputer Score Ema at Topic quantile
// Returns previous reputer score ema at topic quantile, or 0 if not yet seen
func (k *Keeper) GetPreviousTopicQuantileReputerScoreEma(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	score, err := k.previousTopicQuantileReputerScoreEma.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), nil
		}
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous topic quantile reputer score ema")
	}
	return score, nil
}

/// REWARD FRACTION

// Gets the previous W_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousReputerRewardFraction(
	ctx context.Context, topicId TopicId, reputer ActorId) (
	previousReputerRewardFraction alloraMath.Dec, noPrior bool, err error) {
	key := collections.Join(topicId, reputer)
	previousReputerRewardFraction, err = k.previousReputerRewardFraction.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.Dec{}, false, errorsmod.Wrap(err, "error getting previous reputer reward fraction")
	}
	return previousReputerRewardFraction, false, nil
}

// Sets the previous W_{i-1,m}
func (k *Keeper) SetPreviousReputerRewardFraction(ctx context.Context, topicId TopicId, reputer ActorId, reward alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousReputerRewardFraction: Error validating topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousReputerRewardFraction: Error validating reputer")
	}
	if err := types.ValidateDec(reward); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousReputerRewardFraction: Error validating reward")
	}
	key := collections.Join(topicId, reputer)
	return k.previousReputerRewardFraction.Set(ctx, key, reward)
}

// Gets the previous U_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker ActorId) (
	previousInferenceRewardFraction alloraMath.Dec, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	previousInferenceRewardFraction, err = k.previousInferenceRewardFraction.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.Dec{}, false, errorsmod.Wrap(err, "error getting previous inference reward fraction")
	}
	return previousInferenceRewardFraction, false, nil
}

// Sets the previous U_{i-1,m}
func (k *Keeper) SetPreviousInferenceRewardFraction(ctx context.Context, topicId TopicId, worker ActorId, reward alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousInferenceRewardFraction: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousInferenceRewardFraction: Error validating worker")
	}
	if err := types.ValidateDec(reward); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousInferenceRewardFraction: Error validating reward")
	}
	key := collections.Join(topicId, worker)
	return k.previousInferenceRewardFraction.Set(ctx, key, reward)
}

// Gets the previous V_{i-1,m}
// Returns previous reward fraction, and true if it has yet to be set for the first time (else false)
func (k *Keeper) GetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker ActorId) (
	previousForecastRewardFraction alloraMath.Dec, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	previousForecastRewardFraction, err = k.previousForecastRewardFraction.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), true, nil
		}
		return alloraMath.Dec{}, false, errorsmod.Wrap(err, "error getting previous forecast reward fraction")
	}
	return previousForecastRewardFraction, false, nil
}

// Sets the previous V_{i-1,m}
func (k *Keeper) SetPreviousForecastRewardFraction(ctx context.Context, topicId TopicId, worker ActorId, reward alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecastRewardFraction: Error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecastRewardFraction: Error validating worker")
	}
	if err := types.ValidateDec(reward); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecastRewardFraction: Error validating reward")
	}
	key := collections.Join(topicId, worker)
	return k.previousForecastRewardFraction.Set(ctx, key, reward)
}

func (k *Keeper) SetPreviousPercentageRewardToStakedReputers(
	ctx context.Context,
	percentageRewardToStakedReputers alloraMath.Dec,
) error {
	if err := types.ValidateDec(percentageRewardToStakedReputers); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousPercentageRewardToStakedReputers: Error validating percentage reward to staked reputers")
	}
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
	if err := types.ValidateBech32(admin); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
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
		return errorsmod.Wrap(err, "error getting recipient address")
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// wrapper around bank keeper SendCoinsFromAccountToModule
func (k *Keeper) SendCoinsFromAccountToModule(ctx context.Context, sender ActorId, recipientModule string, amt sdk.Coins) error {
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return errorsmod.Wrap(err, "error getting sender address")
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
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting total reward to distribute")
	}
	return totalRewardDec, nil
}

/// UTILS

// Convert pagination.key from []bytes to uint64, if pagination is nil or [], len = 0
// Get the limit from the pagination request, within acceptable bounds and defaulting as necessary
func (k Keeper) CalcAppropriatePaginationForUint64Cursor(ctx context.Context, pagination *types.SimpleCursorPaginationRequest) (uint64, uint64, error) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return uint64(0), uint64(0), errorsmod.Wrap(err, "error getting params")
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
		return errorsmod.Wrap(err, "error pruning inferences")
	}
	err = k.pruneForecasts(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning forecasts")
	}
	err = k.pruneLossBundles(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning loss bundles")
	}
	err = k.pruneNetworkLosses(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning network losses")
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
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	nonces, err := k.unfulfilledWorkerNonces.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return errorsmod.Wrapf(err, "no nonces found to prune for topic %d", topicId)
		}
		return errorsmod.Wrap(err, "error getting unfulfilled worker nonces")
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
	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled worker nonces")
	}
	if err := k.unfulfilledWorkerNonces.Set(ctx, topicId, nonces); err != nil {
		return errorsmod.Wrap(err, "error setting unfulfilled worker nonces")
	}

	return nil
}

func (k *Keeper) PruneReputerNonces(ctx context.Context, topicId uint64, blockHeightThreshold int64) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	nonces, err := k.GetUnfulfilledReputerNonces(ctx, topicId)
	if err != nil {
		return errorsmod.Wrap(err, "error getting unfulfilled reputer nonces")
	}

	// Filter Nonces based on block_height
	filteredNonces := make([]*types.ReputerRequestNonce, 0)
	for _, nonce := range nonces.Nonces {
		if nonce.ReputerNonce.BlockHeight >= blockHeightThreshold {
			filteredNonces = append(filteredNonces, nonce)
		}
	}

	if len(filteredNonces) == 0 {
		return k.unfulfilledReputerNonces.Remove(ctx, topicId)
	}

	// Update nonces in the map
	nonces.Nonces = filteredNonces
	if err := nonces.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating unfulfilled reputer nonces")
	}
	if err := k.unfulfilledReputerNonces.Set(ctx, topicId, nonces); err != nil {
		return errorsmod.Wrap(err, "error setting unfulfilled reputer nonces")
	}

	return nil
}

// Return true if the nonce is within the worker submission window for the topic
func (k *Keeper) BlockWithinWorkerSubmissionWindowOfNonce(topic types.Topic, nonce types.Nonce, blockHeight int64) bool {
	return nonce.BlockHeight <= blockHeight && blockHeight < topic.WorkerSubmissionWindow+nonce.BlockHeight
}

// Return true if the nonce is within the worker submission window for the topic
func (k *Keeper) BlockWithinReputerSubmissionWindowOfNonce(topic types.Topic, nonce types.ReputerRequestNonce, blockHeight int64) bool {
	return nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag <= blockHeight &&
		blockHeight <= nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag*2
}

func (k *Keeper) ValidateStringIsBech32(actor ActorId) error {
	_, err := sdk.AccAddressFromBech32(actor)
	if err != nil {
		return errorsmod.Wrap(err, "error validating actor id")
	}
	return nil
}

func (k *Keeper) SetWorkerTopicLastCommit(
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

func (k *Keeper) SetReputerTopicLastCommit(
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

func (k *Keeper) GetWorkerTopicLastCommit(ctx context.Context, topic TopicId) (types.TimestampedActorNonce, error) {
	return k.topicLastWorkerCommit.Get(ctx, topic)
}

func (k *Keeper) GetReputerTopicLastCommit(ctx context.Context, topic TopicId) (types.TimestampedActorNonce, error) {
	return k.topicLastReputerCommit.Get(ctx, topic)
}

func (k *Keeper) SetPreviousForecasterScoreRatio(ctx context.Context, topicId TopicId, forecasterScoreRatio alloraMath.Dec) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecasterScoreRatio: Error validating topic id")
	}
	if err := types.ValidateDec(forecasterScoreRatio); err != nil {
		return errorsmod.Wrapf(err, "SetPreviousForecasterScoreRatio: Error validating forecaster score ratio")
	}
	return k.previousForecasterScoreRatio.Set(ctx, topicId, forecasterScoreRatio)
}

func (k *Keeper) GetPreviousForecasterScoreRatio(ctx context.Context, topicId TopicId) (alloraMath.Dec, error) {
	forecastTau, err := k.previousForecasterScoreRatio.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return alloraMath.ZeroDec(), nil
		}
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting previous forecaster score ratio")
	}
	return forecastTau, nil
}

// AddActiveInferer adds an inferer to the active inferers set for a topic
func (k *Keeper) AddActiveInferer(ctx context.Context, topicId TopicId, inferer ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "invalid inferer address")
	}
	key := collections.Join(topicId, inferer)
	return k.activeInferers.Set(ctx, key)
}

// IsActiveInferer checks if an inferer is in the active inferers set for a topic
func (k *Keeper) IsActiveInferer(ctx context.Context, topicId TopicId, inferer ActorId) (bool, error) {
	key := collections.Join(topicId, inferer)
	return k.activeInferers.Has(ctx, key)
}

// RemoveActiveInferer removes an inferer from the active inferers set for a topic
func (k *Keeper) RemoveActiveInferer(ctx context.Context, topicId TopicId, inferer ActorId) error {
	key := collections.Join(topicId, inferer)
	return k.activeInferers.Remove(ctx, key)
}

// GetActiveInferersForTopic returns all active inferers for a specific topic
func (k *Keeper) GetActiveInferersForTopic(ctx context.Context, topicId TopicId) ([]ActorId, error) {
	var inferers []ActorId
	rng := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	err := k.activeInferers.Walk(ctx, rng, func(key collections.Pair[TopicId, ActorId]) (bool, error) {
		inferers = append(inferers, key.K2())
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "error walking active inferers")
	}
	return inferers, nil
}

// AddActiveForecaster adds a forecaster to the active forecasters set for a topic
func (k *Keeper) AddActiveForecaster(ctx context.Context, topicId TopicId, forecaster ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(forecaster); err != nil {
		return errorsmod.Wrap(err, "invalid forecaster address")
	}
	key := collections.Join(topicId, forecaster)
	return k.activeForecasters.Set(ctx, key)
}

// IsActiveForecaster checks if a forecaster is in the active forecasters set for a topic
func (k *Keeper) IsActiveForecaster(ctx context.Context, topicId TopicId, forecaster ActorId) (bool, error) {
	key := collections.Join(topicId, forecaster)
	return k.activeForecasters.Has(ctx, key)
}

// RemoveActiveForecaster removes a forecaster from the active forecasters set for a topic
func (k *Keeper) RemoveActiveForecaster(ctx context.Context, topicId TopicId, forecaster ActorId) error {
	key := collections.Join(topicId, forecaster)
	return k.activeForecasters.Remove(ctx, key)
}

// GetActiveForecastersForTopic returns all active forecasters for a specific topic
func (k *Keeper) GetActiveForecastersForTopic(ctx context.Context, topicId TopicId) ([]ActorId, error) {
	var forecasters []ActorId
	rng := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	err := k.activeForecasters.Walk(ctx, rng, func(key collections.Pair[TopicId, ActorId]) (bool, error) {
		forecasters = append(forecasters, key.K2())
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "error walking active forecasters")
	}
	return forecasters, nil
}

// SetLowestInfererScoreEma sets the lowest inferer score EMA for a topic
func (k *Keeper) SetLowestInfererScoreEma(ctx context.Context, topicId TopicId, lowestScore types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(lowestScore.Address); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}
	return k.lowestInfererScoreEma.Set(ctx, topicId, lowestScore)
}

// GetLowestInfererScoreEma gets the lowest inferer score EMA for a topic
func (k *Keeper) GetLowestInfererScoreEma(ctx context.Context, topicId TopicId) (types.Score, bool, error) {
	lowestScore, err := k.lowestInfererScoreEma.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{}, false, nil
		}
		return types.Score{}, false, errorsmod.Wrap(err, "error getting lowest inferer score EMA")
	}
	return lowestScore, true, nil
}

// SetLowestForecasterScoreEma sets the lowest forecaster score EMA for a topic
func (k *Keeper) SetLowestForecasterScoreEma(ctx context.Context, topicId TopicId, lowestScore types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(lowestScore.Address); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}
	return k.lowestForecasterScoreEma.Set(ctx, topicId, lowestScore)
}

// GetLowestForecasterScoreEma gets the lowest forecaster score EMA for a topic
func (k *Keeper) GetLowestForecasterScoreEma(ctx context.Context, topicId TopicId) (types.Score, bool, error) {
	lowestScore, err := k.lowestForecasterScoreEma.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{}, false, nil
		}
		return types.Score{}, false, errorsmod.Wrap(err, "error getting lowest forecaster score EMA")
	}
	return lowestScore, true, nil
}

// AddActiveReputer adds a reputer to the active reputers set for a topic
func (k *Keeper) AddActiveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(reputer); err != nil {
		return errorsmod.Wrap(err, "invalid reputer address")
	}
	key := collections.Join(topicId, reputer)
	return k.activeReputers.Set(ctx, key)
}

// IsActiveReputer checks if a reputer is in the active reputers set for a topic
func (k *Keeper) IsActiveReputer(ctx context.Context, topicId TopicId, reputer ActorId) (bool, error) {
	key := collections.Join(topicId, reputer)
	return k.activeReputers.Has(ctx, key)
}

// RemoveActiveReputer removes a reputer from the active reputers set for a topic
func (k *Keeper) RemoveActiveReputer(ctx context.Context, topicId TopicId, reputer ActorId) error {
	key := collections.Join(topicId, reputer)
	return k.activeReputers.Remove(ctx, key)
}

// GetActiveReputersForTopic returns all active reputers for a specific topic
func (k *Keeper) GetActiveReputersForTopic(ctx context.Context, topicId TopicId) ([]ActorId, error) {
	var reputers []ActorId
	rng := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
	err := k.activeReputers.Walk(ctx, rng, func(key collections.Pair[TopicId, ActorId]) (bool, error) {
		reputers = append(reputers, key.K2())
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "error walking active reputers")
	}
	return reputers, nil
}

// ResetActiveActorsForTopic resets the active actors for a topic
func (k *Keeper) ResetActiveActorsForTopic(ctx context.Context, topicId TopicId) error {
    // Clear active inferers for the topic
    infererRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
    if err := k.activeInferers.Clear(ctx, infererRange); err != nil {
        return errorsmod.Wrap(err, "error clearing active inferers")
    }

    // Clear active forecasters for the topic
    forecasterRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
    if err := k.activeForecasters.Clear(ctx, forecasterRange); err != nil {
        return errorsmod.Wrap(err, "error clearing active forecasters")
    }

    // Clear active reputers for the topic
    reputerRange := collections.NewPrefixedPairRange[TopicId, ActorId](topicId)
    if err := k.activeReputers.Clear(ctx, reputerRange); err != nil {
        return errorsmod.Wrap(err, "error clearing active reputers")
    }

    return nil
}

// SetLowestReputerScoreEma sets the lowest reputer score EMA for a topic
func (k *Keeper) SetLowestReputerScoreEma(ctx context.Context, topicId TopicId, lowestScore types.Score) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "invalid topic id")
	}
	if err := types.ValidateBech32(lowestScore.Address); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}
	return k.lowestReputerScoreEma.Set(ctx, topicId, lowestScore)
}

// GetLowestReputerScoreEma gets the lowest reputer score EMA for a topic
func (k *Keeper) GetLowestReputerScoreEma(ctx context.Context, topicId TopicId) (types.Score, bool, error) {
	lowestScore, err := k.lowestReputerScoreEma.Get(ctx, topicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Score{}, false, nil
		}
		return types.Score{}, false, errorsmod.Wrap(err, "error getting lowest reputer score EMA")
	}
	return lowestScore, true, nil
}
