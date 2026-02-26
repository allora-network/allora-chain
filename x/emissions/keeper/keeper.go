package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/pkg/errors"

	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	coreStore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/types"
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

	// TYPES
	schema        collections.Schema
	authKeeper    AccountKeeper
	bankingKeeper *BankingKeeper

	// TOPIC
	topicKeeper *TopicKeeper

	// SCORES
	scoresKeeper *ScoresKeeper

	// STAKING
	stakingKeeper *StakingKeeper

	// PARAMETERS
	paramsKeeper *ParamsKeeper

	// REPUTERS AND LOSSES
	reputerLossKeeper *ReputerLossKeeper

	// ACTOR PENALTIES
	actorPenaltiesKeeper *ActorPenaltiesKeeper

	// WORKERS
	workerKeeper *WorkerKeeper

	// / MISC GLOBAL STATE

	// Current block emission, set by mint module
	rewardCurrentBlockEmission collections.Item[cosmosMath.Int]

	// / NONCES
	nonceKeeper *NonceKeeper

	// / REGRETS
	regretsKeeper *RegretsKeeper

	// WEIGHTS
	weightsKeeper *WeightsKeeper

	// / WHITELISTS
	whitelistsKeeper *WhitelistsKeeper

	// / INCLUSIONS

	countInfererInclusionsInTopicActiveSet    collections.Map[collections.Pair[TopicId, ActorId], uint64]
	countForecasterInclusionsInTopicActiveSet collections.Map[collections.Pair[TopicId, ActorId], uint64]

	// map of (topic, block_height) -> ValueBundle
	networkInferences collections.Map[collections.Pair[TopicId, BlockHeight], types.ValueBundle]
	// map of (topic, block_height) -> ValueBundle
	outlierResistantNetworkInferences collections.Map[collections.Pair[TopicId, BlockHeight], types.ValueBundle]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	addressCodec address.Codec,
	storeService coreStore.KVStoreService,
	ak AccountKeeper,
	bankKeeper BankKeeper,
	feeCollectorName string,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	bk := NewBankingKeeper(bankKeeper, ak)
	pk := NewParamsKeeper(cdc, sb)
	stk := NewStakingKeeper(cdc, sb, nil, bk) // set tk below
	nk := NewNonceKeeper(cdc, sb, nil, pk)    // set tk below
	tk := NewTopicKeeper(cdc, sb, pk, nk, stk)
	stk.topicKeeper = tk
	nk.topicKeeper = tk
	sk := NewScoresKeeper(cdc, sb, pk)
	apk := NewActorPenaltiesKeeper(sk)
	rlk := NewReputerLossKeeper(cdc, sb, sk, tk, apk)
	wk := NewWorkerKeeper(cdc, sb, tk, sk, pk, apk)
	rk := NewRegretsKeeper(cdc, sb, tk)
	wgk := NewWeightsKeeper(sb, sk)
	wlk := NewWhitelistsKeeper(sb, pk, tk)
	k := Keeper{
		cdc:                  cdc,
		schema:               collections.Schema{},
		storeService:         storeService,
		addressCodec:         addressCodec,
		feeCollectorName:     feeCollectorName,
		authKeeper:           ak,
		bankingKeeper:        bk,
		topicKeeper:          tk,
		scoresKeeper:         sk,
		stakingKeeper:        stk,
		paramsKeeper:         pk,
		reputerLossKeeper:    rlk,
		actorPenaltiesKeeper: apk,
		workerKeeper:         wk,
		nonceKeeper:          nk,
		regretsKeeper:        rk,
		weightsKeeper:        wgk,
		whitelistsKeeper:     wlk,
		// emissionsMintKeeper:                    NewEmissionsMintKeeper(), TODO
		rewardCurrentBlockEmission:                collections.NewItem(sb, types.RewardCurrentBlockEmissionKey, "reward_current_block_emission", sdk.IntValue),
		countInfererInclusionsInTopicActiveSet:    collections.NewMap(sb, types.CountInfererInclusionsInTopicKey, "count_inferer_inclusions_in_topic", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), collections.Uint64Value),
		countForecasterInclusionsInTopicActiveSet: collections.NewMap(sb, types.CountForecasterInclusionsInTopicKey, "count_forecaster_inclusions_in_topic", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), collections.Uint64Value),
		networkInferences:                         collections.NewMap(sb, types.NetworkInferencesKey, "network_inferences", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ValueBundle](cdc)),
		outlierResistantNetworkInferences:         collections.NewMap(sb, types.OutlierResistantNetworkInferencesKey, "outlier_resistant_network_inferences", collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key), codec.CollValue[types.ValueBundle](cdc)),
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

// Insert a network inference for a topic at a block
func (k *Keeper) InsertNetworkInferences(ctx context.Context, topicId TopicId, blockHeight BlockHeight, bundle types.ValueBundle) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := bundle.Validate(); err != nil {
		return errorsmod.Wrap(err, "loss bundle validation failed")
	}
	return k.networkInferences.Set(ctx, collections.Join(topicId, blockHeight), bundle)
}

// Get Network Inferences
func (k *Keeper) GetNetworkInferences(ctx context.Context, topicId TopicId, blockHeight BlockHeight) (*types.ValueBundle, error) {
	key := collections.Join(topicId, blockHeight)
	networkInferences, err := k.networkInferences.Get(ctx, key)
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
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting network inferences at block")
	}
	return &networkInferences, nil
}

func (k *Keeper) InsertOutlierResistantNetworkInferences(ctx context.Context, topicId TopicId, blockHeight BlockHeight, bundle types.ValueBundle) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "topic id validation failed")
	}
	if err := types.ValidateBlockHeight(blockHeight); err != nil {
		return errorsmod.Wrap(err, "block height validation failed")
	}
	if err := bundle.Validate(); err != nil {
		return errorsmod.Wrap(err, "loss bundle validation failed")
	}
	return k.outlierResistantNetworkInferences.Set(ctx, collections.Join(topicId, blockHeight), bundle)
}

// Get Outlier Resistant Network Inferences
func (k *Keeper) GetOutlierResistantNetworkInferences(ctx context.Context, topicId TopicId, blockHeight BlockHeight) (*types.ValueBundle, error) {
	key := collections.Join(topicId, blockHeight)
	networkInferences, err := k.outlierResistantNetworkInferences.Get(ctx, key)
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
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "error getting network inferences at block")
	}
	return &networkInferences, nil
}

// Gets Latest Network Inferences, outlier resistant or not
func (k *Keeper) GetLatestNetworkInferences(ctx context.Context, topicId TopicId, outlierResistant bool) (*types.ValueBundle, error) {
	if err := types.ValidateTopicId(topicId); err != nil {
		return nil, errorsmod.Wrap(err, "invalid topic id")
	}

	rng := collections.NewPrefixedPairRange[TopicId, BlockHeight](topicId).Descending()
	var err error
	var iter collections.Iterator[collections.Pair[TopicId, BlockHeight], types.ValueBundle]
	if outlierResistant {
		iter, err = k.outlierResistantNetworkInferences.Iterate(ctx, rng)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error iterating outlier resistant network inferences")
		}
	} else {
		iter, err = k.networkInferences.Iterate(ctx, rng)
		if err != nil {
			return nil, errorsmod.Wrap(err, "error iterating network inferences")
		}
	}
	defer iter.Close()

	// Get the first (latest) entry
	if iter.Valid() {
		keyValue, err := iter.KeyValue()
		if err != nil {
			return nil, errorsmod.Wrap(err, "error getting key value")
		}
		return &keyValue.Value, nil
	}
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

// / INCLUSIONS

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

// / STATE MANAGEMENT

// Iterate through topic state and prune records that are no longer needed
func (k *Keeper) PruneRecordsAfterRewards(ctx sdk.Context, topicId TopicId, blockHeight int64) error {
	defer types.EmitNewPruneRecordsEvent(ctx, blockHeight, topicId)
	// Delete records until the blockHeight
	blockRange := collections.
		NewPrefixedPairRange[TopicId, BlockHeight](topicId).
		EndInclusive(blockHeight)

	err := k.workerKeeper.PruneInferences(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning inferences")
	}
	err = k.workerKeeper.PruneForecasts(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning forecasts")
	}
	err = k.reputerLossKeeper.PruneLossBundles(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning loss bundles")
	}
	err = k.reputerLossKeeper.PruneNetworkLosses(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning network losses")
	}
	err = k.pruneNetworkInferences(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning network inferences")
	}
	err = k.pruneOutlierResistantNetworkInferences(ctx, blockRange)
	if err != nil {
		return errorsmod.Wrap(err, "error pruning outlier resistant network inferences")
	}
	return nil
}

func (k *Keeper) pruneNetworkInferences(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.networkInferences.Clear(ctx, blockRange)
}

func (k *Keeper) pruneOutlierResistantNetworkInferences(ctx context.Context, blockRange *collections.PairRange[uint64, int64]) error {
	return k.outlierResistantNetworkInferences.Clear(ctx, blockRange)
}

// GetRewardCurrentBlockEmission retrieves the current block emission reward.
func (k *Keeper) GetRewardCurrentBlockEmission(ctx context.Context) (cosmosMath.Int, error) {
	emission, err := k.rewardCurrentBlockEmission.Get(ctx)

	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.ZeroInt(), nil // Return zero if not found
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting current block emission reward")
	}
	return emission, nil
}

// SetRewardCurrentBlockEmission sets the current block emission reward.
func (k Keeper) SetRewardCurrentBlockEmission(ctx context.Context, emission cosmosMath.Int) error {
	if emission.IsNegative() {
		return errorsmod.Wrap(types.ErrInvalidValue, "current block emission reward cannot be negative")
	}
	return k.rewardCurrentBlockEmission.Set(ctx, emission)
}

func (k *Keeper) GetScoresKeeper() *ScoresKeeper {
	return k.scoresKeeper
}

func (k *Keeper) GetParamsKeeper() *ParamsKeeper {
	return k.paramsKeeper
}

func (k *Keeper) GetTopicKeeper() *TopicKeeper {
	return k.topicKeeper
}

func (k *Keeper) GetStakingKeeper() *StakingKeeper {
	return k.stakingKeeper
}

func (k *Keeper) GetBankingKeeper() *BankingKeeper {
	return k.bankingKeeper
}

func (k *Keeper) GetReputerLossKeeper() *ReputerLossKeeper {
	return k.reputerLossKeeper
}

func (k *Keeper) GetNonceKeeper() *NonceKeeper {
	return k.nonceKeeper
}

func (k *Keeper) GetWorkerKeeper() *WorkerKeeper {
	return k.workerKeeper
}

func (k *Keeper) GetWeightsKeeper() *WeightsKeeper {
	return k.weightsKeeper
}

func (k *Keeper) GetWhitelistsKeeper() *WhitelistsKeeper {
	return k.whitelistsKeeper
}

func (k *Keeper) GetActorPenaltiesKeeper() *ActorPenaltiesKeeper {
	return k.actorPenaltiesKeeper
}

func (k *Keeper) GetRegretsKeeper() *RegretsKeeper {
	return k.regretsKeeper
}

// TODO: move elsewhere
func ValidateStringIsBech32(actor ActorId) error {
	_, err := sdk.AccAddressFromBech32(actor)
	if err != nil {
		return errorsmod.Wrap(err, "error validating actor id")
	}
	return nil
}
