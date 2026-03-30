package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewStakingKeeper(
	cdc codec.BinaryCodec,
	sb *collections.SchemaBuilder,
	topicKeeper *TopicKeeper,
	bankKeeper *BankingKeeper,
) *StakingKeeper {
	return &StakingKeeper{
		totalStake:                     collections.NewItem(sb, types.TotalStakeKey, "total_stake", sdk.IntValue),
		topicStake:                     collections.NewMap(sb, types.TopicStakeKey, "topic_stake", collections.Uint64Key, sdk.IntValue),
		stakeReputerAuthority:          collections.NewMap(sb, types.StakeByReputerAndTopicIdKey, "stake_by_reputer_and_TopicId", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), sdk.IntValue),
		stakeSumFromDelegator:          collections.NewMap(sb, types.DelegatorStakeKey, "stake_from_delegator", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), sdk.IntValue),
		delegatedStakes:                collections.NewMap(sb, types.DelegateStakePlacementKey, "delegate_stake_placement", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.DelegatorInfo](cdc)),
		stakeFromDelegatorsUponReputer: collections.NewMap(sb, types.TargetStakeKey, "stake_upon_reputer", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), sdk.IntValue),
		delegateRewardPerShare:         collections.NewMap(sb, types.DelegateRewardPerShare, "delegate_reward_per_share", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), alloraMath.DecValue),
		stakeRemovalsByBlock:           collections.NewMap(sb, types.StakeRemovalsByBlockKey, "stake_removals_by_block", collections.TripleKeyCodec(collections.Int64Key, collections.Uint64Key, collections.StringKey), codec.CollValue[types.StakeRemovalInfo](cdc)),
		stakeRemovalsByActor:           collections.NewKeySet(sb, types.StakeRemovalsByActorKey, "stake_removals_by_actor", collections.TripleKeyCodec(collections.StringKey, collections.Uint64Key, collections.Int64Key)),
		delegateStakeRemovalsByBlock:   collections.NewMap(sb, types.DelegateStakeRemovalsByBlockKey, "delegate_stake_removals_by_block", QuadrupleKeyCodec(collections.Int64Key, collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.DelegateStakeRemovalInfo](cdc)),
		delegateStakeRemovalsByActor:   collections.NewKeySet(sb, types.DelegateStakeRemovalsByActorKey, "delegate_stake_removals_by_actor", QuadrupleKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key, collections.Int64Key)),
		topicKeeper:                    topicKeeper,
		bankKeeper:                     bankKeeper,
	}
}

type StakingKeeper struct {
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
	// topic keeper
	topicKeeper *TopicKeeper
	// bank keeper
	bankKeeper *BankingKeeper
}

// Gets the total sum of all stake in the network across all topics
func (k *StakingKeeper) GetTotalStake(ctx context.Context) (cosmosMath.Int, error) {
	ret, err := k.totalStake.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.NewInt(0), nil
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting total stake")
	}
	return ret, nil
}

// Sets the total sum of all stake in the network across all topics
func (k *StakingKeeper) SetTotalStake(ctx context.Context, totalStake cosmosMath.Int) error {
	// big int pointer inside the cosmos int must be non nil
	if err := types.ValidateSdkIntRepresentingMonetaryValue(totalStake); err != nil {
		return errorsmod.Wrap(err, "totalStake cosmos Int is not valid")
	}
	// total stake does not have a zero guard because totalStake is allowed to be zero
	// it is initialized to zero at genesis anyways.
	return k.totalStake.Set(ctx, totalStake)
}

// Gets the stake in the network for a given topic
func (k *StakingKeeper) GetTopicStake(ctx context.Context, topicId TopicId) (cosmosMath.Int, error) {
	ret, err := k.topicStake.Get(ctx, topicId)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.NewInt(0), nil
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting topic stake")
	}
	return ret, nil
}

// sets the cumulative amount of stake in a topic
func (k *StakingKeeper) SetTopicStake(ctx context.Context, topicId TopicId, stake cosmosMath.Int) error {
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
func (k *StakingKeeper) GetStakeReputerAuthority(ctx context.Context, topicId TopicId, reputer ActorId) (cosmosMath.Int, error) {
	key := collections.Join(topicId, reputer)
	stake, err := k.stakeReputerAuthority.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.NewInt(0), nil
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting stake reputer authority")
	}
	return stake, nil
}

// Sets the amount of stake placed upon a reputer in addition to their personal stake on a specific topic
// Includes the stake placed by delegators on the reputer in that topic.
func (k *StakingKeeper) SetStakeReputerAuthority(ctx context.Context, topicId TopicId, reputer ActorId, amount cosmosMath.Int) error {
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
func (k *StakingKeeper) GetStakeFromDelegatorInTopic(ctx context.Context, topicId TopicId, delegator ActorId) (cosmosMath.Int, error) {
	key := collections.Join(topicId, delegator)
	stake, err := k.stakeSumFromDelegator.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.NewInt(0), nil
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting stake from delegator in topic")
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator.
func (k *StakingKeeper) SetStakeFromDelegator(ctx context.Context, topicId TopicId, delegator ActorId, stake cosmosMath.Int) error {
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
func (k *StakingKeeper) GetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator ActorId, target ActorId) (types.DelegatorInfo, error) {
	key := collections.Join3(topicId, delegator, target)
	stake, err := k.delegatedStakes.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return types.DelegatorInfo{Amount: alloraMath.NewDecFromInt64(0), RewardDebt: alloraMath.NewDecFromInt64(0)}, nil
	} else if err != nil {
		return types.DelegatorInfo{}, errorsmod.Wrap(err, "error getting delegate stake placement")
	}
	return stake, nil
}

// Sets the amount of stake placed by a specific delegator on a specific target.
func (k *StakingKeeper) SetDelegateStakePlacement(ctx context.Context, topicId TopicId, delegator ActorId, target ActorId, stake types.DelegatorInfo) error {
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
func (k *StakingKeeper) GetDelegateRewardPerShare(ctx context.Context, topicId TopicId, reputer ActorId) (alloraMath.Dec, error) {
	key := collections.Join(topicId, reputer)
	share, err := k.delegateRewardPerShare.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return alloraMath.NewDecFromInt64(0), nil
	} else if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting delegate reward per share")
	}
	return share, nil
}

// Set the share on specific reputer and topicId
func (k *StakingKeeper) SetDelegateRewardPerShare(ctx context.Context, topicId TopicId, reputer ActorId, share alloraMath.Dec) error {
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
	return k.delegateRewardPerShare.Set(ctx, key, share)
}

// Returns the amount of stake placed upon a reputer by delegators within that topic
func (k *StakingKeeper) GetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target ActorId) (cosmosMath.Int, error) {
	key := collections.Join(topicId, target)
	stake, err := k.stakeFromDelegatorsUponReputer.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return cosmosMath.NewInt(0), nil
	} else if err != nil {
		return cosmosMath.Int{}, errorsmod.Wrap(err, "error getting delegate stake upon reputer")
	}
	return stake, nil
}

// Sets the amount of stake placed on a specific target.
func (k *StakingKeeper) SetDelegateStakeUponReputer(ctx context.Context, topicId TopicId, target ActorId, stake cosmosMath.Int) error {
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
func (k *StakingKeeper) SetStakeRemoval(ctx context.Context, removalInfo types.StakeRemovalInfo) error {
	if err := removalInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "removalInfo is not valid")
	}
	byBlockKey := collections.Join3(removalInfo.BlockRemovalCompleted, removalInfo.TopicId, removalInfo.Reputer)
	err := k.stakeRemovalsByBlock.Set(ctx, byBlockKey, removalInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting stake removal by block")
	}
	byActorKey := collections.Join3(removalInfo.Reputer, removalInfo.TopicId, removalInfo.BlockRemovalCompleted)
	err = k.stakeRemovalsByActor.Set(ctx, byActorKey)
	if err != nil {
		return err
	}
	types.EmitNewRequestStakeRemovalEvent(ctx, removalInfo.TopicId, removalInfo.Reputer, removalInfo.Reputer, removalInfo.Amount, removalInfo.BlockRemovalCompleted)
	return nil
}

// remove a stake removal from the queue
func (k *StakingKeeper) DeleteStakeRemoval(
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
	err = k.stakeRemovalsByActor.Remove(ctx, byActorKey)
	if err != nil {
		return err
	}
	types.EmitNewCancelStakeRemovalEvent(ctx, topicId, address, address)
	return nil
}

// get info about a removal
func (k *StakingKeeper) GetStakeRemoval(
	ctx context.Context,
	BlockHeight int64,
	topicId TopicId,
	reputer ActorId,
) (types.StakeRemovalInfo, error) {
	return k.stakeRemovalsByBlock.Get(ctx, collections.Join3(BlockHeight, topicId, reputer))
}

// get a list of stake removals that are valid for removal
// before and including this block.
func (k *StakingKeeper) GetStakeRemovalsUpUntilBlock(
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
func (k *StakingKeeper) GetStakeRemovalForReputerAndTopicId(
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
func (k *StakingKeeper) SetDelegateStakeRemoval(ctx context.Context, removalInfo types.DelegateStakeRemovalInfo) error {
	if err := removalInfo.Validate(); err != nil {
		return errorsmod.Wrap(err, "removalInfo is not valid")
	}
	byBlockKey := Join4(removalInfo.BlockRemovalCompleted, removalInfo.TopicId, removalInfo.Delegator, removalInfo.Reputer)
	err := k.delegateStakeRemovalsByBlock.Set(ctx, byBlockKey, removalInfo)
	if err != nil {
		return errorsmod.Wrap(err, "error setting delegate stake removal by block")
	}
	byActorKey := Join4(removalInfo.Delegator, removalInfo.Reputer, removalInfo.TopicId, removalInfo.BlockRemovalCompleted)
	err = k.delegateStakeRemovalsByActor.Set(ctx, byActorKey)
	if err != nil {
		return err
	}
	types.EmitNewRequestStakeRemovalEvent(ctx, removalInfo.TopicId, removalInfo.Reputer, removalInfo.Delegator, removalInfo.Amount, removalInfo.BlockRemovalCompleted)
	return nil
}

// remove a stake removal from the queue
func (k *StakingKeeper) DeleteDelegateStakeRemoval(
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
	err = k.delegateStakeRemovalsByActor.Remove(ctx, byActorKey)
	if err != nil {
		return err
	}
	types.EmitNewCancelStakeRemovalEvent(ctx, topicId, reputer, delegator)
	return nil
}

// get info about a removal
func (k *StakingKeeper) GetDelegateStakeRemoval(
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
func (k *StakingKeeper) GetDelegateStakeRemovalsUpUntilBlock(
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
func (k *StakingKeeper) GetDelegateStakeRemovalForDelegatorReputerAndTopicId(
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
	defer iter.Close()
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

// Adds stake to the system for a given topic and reputer
// Adds to: totalStake, topicStake, stakeReputerAuthority,
func (k *StakingKeeper) AddReputerStake(
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

	types.EmitNewAddStakeEvent(ctx, topicId, reputer, reputer, stakeToAdd, reputerAuthorityNew)
	return nil
}

// adds stake to the system from a delegator
// adds to: totalStake, topicStake, stakeReputerAuthority,
//
//	stakeSumFromDelegator, delegatedStakes, stakeFromDelegatorsUponReputer
func (k *StakingKeeper) AddDelegateStake(
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
			err = k.bankKeeper.SendCoinsFromModuleToAccount(
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

	types.EmitNewAddStakeEvent(ctx, topicId, reputer, delegator, stakeToAdd, stakeReputerAuthorityNew)
	return nil
}

// Removes stake from the system for a given topic and reputer
// subtracts from: totalStake, topicStake, stakeReputerAuthority
func (k *StakingKeeper) RemoveReputerStake(
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

	// Calculate new weight and update totalSumPreviousTopicWeights
	if err := k.topicKeeper.UpdateTopicWeightAfterStakeChange(ctx, topicId); err != nil {
		return err
	}

	// remove stake withdrawal information
	err = k.DeleteStakeRemoval(ctx, blockHeight, topicId, reputer)
	if err != nil {
		return errorsmod.Wrapf(err, "Deleting stake removal from queue failed")
	}

	types.EmitNewRemoveStakeEvent(
		sdk.UnwrapSDKContext(ctx),
		topicId,
		reputer,
		reputer,
		stakeToRemove,
		reputerStakeNew,
	)

	return nil
}

// Removes delegate stake from the system for a given topic, delegator, and reputer
// subtracts from: totalStake, topicStake, stakeReputerAuthority
//
//	stakeSumFromDelegator, delegatedStakes, stakeFromDelegatorsUponReputer
func (k *StakingKeeper) RemoveDelegateStake(
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
		err = k.bankKeeper.SendCoinsFromModuleToAccount(
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

	// Calculate new weight and update totalSumPreviousTopicWeights
	if err := k.topicKeeper.UpdateTopicWeightAfterStakeChange(ctx, topicId); err != nil {
		return err
	}

	if err := k.DeleteDelegateStakeRemoval(ctx, stakeRemovalBlockHeight, topicId, reputer, delegator); err != nil {
		return errorsmod.Wrapf(err, "Deleting delegate stake removal from queue failed")
	}

	types.EmitNewRemoveStakeEvent(
		sdk.UnwrapSDKContext(ctx),
		topicId,
		reputer,
		delegator,
		stakeToRemove,
		stakeReputerAuthorityNew,
	)

	return nil
}
