package keeper

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *StakingKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// TotalStake
	if data.TotalStake.GT(cosmosMath.ZeroInt()) {
		if err := k.SetTotalStake(ctx, data.TotalStake); err != nil {
			return errors.Wrap(err, "error setting totalStake")
		}
	} else {
		if err := k.SetTotalStake(ctx, cosmosMath.ZeroInt()); err != nil {
			return errors.Wrap(err, "error setting totalStake to zero int")
		}
	}
	// TopicStake
	for _, topicIdAndInt := range data.TopicStake {
		if topicIdAndInt != nil {
			if err := k.SetTopicStake(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "error setting topicStake")
			}
		}
	}
	// StakeReputerAuthority
	for _, topicIdActorIdInt := range data.StakeReputerAuthority {
		if topicIdActorIdInt != nil {
			if err := k.SetStakeReputerAuthority(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeReputerAuthority")
			}
		}
	}
	// StakeSumFromDelegator
	for _, topicIdActorIdInt := range data.StakeSumFromDelegator {
		if topicIdActorIdInt != nil {
			if err := k.SetStakeFromDelegator(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeSumFromDelegator")
			}
		}
	}
	// DelegatedStakes
	for _, topicIdDelegatorReputerDelegatorInfo := range data.DelegatedStakes {
		if topicIdDelegatorReputerDelegatorInfo != nil {
			if err := k.SetDelegateStakePlacement(ctx,
				topicIdDelegatorReputerDelegatorInfo.TopicId,
				topicIdDelegatorReputerDelegatorInfo.Delegator,
				topicIdDelegatorReputerDelegatorInfo.Reputer,
				*topicIdDelegatorReputerDelegatorInfo.DelegatorInfo); err != nil {
				return errors.Wrap(err, "error setting delegatedStakes")
			}
		}
	}
	// StakeFromDelegatorsUponReputer
	for _, topicIdActorIdInt := range data.StakeFromDelegatorsUponReputer {
		if topicIdActorIdInt != nil {
			if err := k.SetDelegateStakeUponReputer(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeFromDelegatorsUponReputer")
			}
		}
	}
	// DelegateRewardPerShare
	for _, topicIdActorIdDec := range data.DelegateRewardPerShare {
		if topicIdActorIdDec != nil {
			if err := k.SetDelegateRewardPerShare(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting delegateRewardPerShare")
			}
		}
	}
	// StakeRemovalsByBlock (also populates StakeRemovalsByActor)
	for _, blockHeightTopicIdReputerStakeRemovalInfo := range data.StakeRemovalsByBlock {
		if blockHeightTopicIdReputerStakeRemovalInfo != nil {
			if err := k.SetStakeRemoval(ctx,
				*blockHeightTopicIdReputerStakeRemovalInfo.StakeRemovalInfo); err != nil {
				return errors.Wrapf(err, "error setting stakeRemovalsByBlock %v",
					*blockHeightTopicIdReputerStakeRemovalInfo.StakeRemovalInfo,
				)
			}
		}
	}
	// DelegateStakeRemovalsByBlock (also populates DelegateStakeRemovalsByActor)
	for _, blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo := range data.DelegateStakeRemovalsByBlock {
		if blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo != nil {
			if err := k.SetDelegateStakeRemoval(ctx,
				*blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.DelegateStakeRemovalInfo); err != nil {
				return errors.Wrap(err, "error setting delegateStakeRemovalsByBlock")
			}
		}
	}

	return nil
}

func (k *StakingKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// totalStake
	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get total stake")
	}
	data.TotalStake = totalStake

	// topicStake
	topicStake := make([]*types.TopicIdAndInt, 0)
	topicStakeIter, err := k.topicStake.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic stake")
	}
	for ; topicStakeIter.Valid(); topicStakeIter.Next() {
		keyValue, err := topicStakeIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicStakeIter")
		}
		topicStake = append(topicStake, &types.TopicIdAndInt{
			TopicId: keyValue.Key,
			Int:     keyValue.Value,
		})
	}
	data.TopicStake = topicStake

	// stakeReputerAuthority
	stakeReputerAuthority := make([]*types.TopicIdActorIdInt, 0)
	stakeReputerAuthorityIter, err := k.stakeReputerAuthority.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate stake reputer authority")
	}
	for ; stakeReputerAuthorityIter.Valid(); stakeReputerAuthorityIter.Next() {
		keyValue, err := stakeReputerAuthorityIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: stakeReputerAuthorityIter")
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeReputerAuthority = append(stakeReputerAuthority, &topicIdActorIdInt)
	}
	data.StakeReputerAuthority = stakeReputerAuthority

	// stakeSumFromDelegator
	stakeSumFromDelegator := make([]*types.TopicIdActorIdInt, 0)
	stakeSumFromDelegatorIter, err := k.stakeSumFromDelegator.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate stake sum from delegator")
	}
	for ; stakeSumFromDelegatorIter.Valid(); stakeSumFromDelegatorIter.Next() {
		keyValue, err := stakeSumFromDelegatorIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: stakeSumFromDelegatorIter")
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeSumFromDelegator = append(stakeSumFromDelegator, &topicIdActorIdInt)
	}
	data.StakeSumFromDelegator = stakeSumFromDelegator

	// delegatedStakes
	delegatedStakes := make([]*types.TopicIdDelegatorReputerDelegatorInfo, 0)
	delegatedStakesIter, err := k.delegatedStakes.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate delegated stakes")
	}
	for ; delegatedStakesIter.Valid(); delegatedStakesIter.Next() {
		keyValue, err := delegatedStakesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: delegatedStakesIter")
		}
		value := keyValue.Value
		topicIdDelegatorReputerDelegatorInfo := types.TopicIdDelegatorReputerDelegatorInfo{
			TopicId:       keyValue.Key.K1(),
			Delegator:     keyValue.Key.K2(),
			Reputer:       keyValue.Key.K3(),
			DelegatorInfo: &value,
		}
		delegatedStakes = append(delegatedStakes, &topicIdDelegatorReputerDelegatorInfo)
	}
	data.DelegatedStakes = delegatedStakes

	// stakeFromDelegatorsUponReputer
	stakeFromDelegatorsUponReputer := make([]*types.TopicIdActorIdInt, 0)
	stakeFromDelegatorsUponReputerIter, err := k.stakeFromDelegatorsUponReputer.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate stake from delegators upon reputer")
	}
	for ; stakeFromDelegatorsUponReputerIter.Valid(); stakeFromDelegatorsUponReputerIter.Next() {
		keyValue, err := stakeFromDelegatorsUponReputerIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: stakeFromDelegatorsUponReputerIter")
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeFromDelegatorsUponReputer = append(stakeFromDelegatorsUponReputer, &topicIdActorIdInt)
	}
	data.StakeFromDelegatorsUponReputer = stakeFromDelegatorsUponReputer

	// delegateRewardPerShare
	delegateRewardPerShare := make([]*types.TopicIdActorIdDec, 0)
	delegateRewardPerShareIter, err := k.delegateRewardPerShare.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate delegate reward per share")
	}
	for ; delegateRewardPerShareIter.Valid(); delegateRewardPerShareIter.Next() {
		keyValue, err := delegateRewardPerShareIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: delegateRewardPerShareIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		delegateRewardPerShare = append(delegateRewardPerShare, &topicIdActorIdDec)
	}
	data.DelegateRewardPerShare = delegateRewardPerShare

	// stakeRemovalsByBlock
	stakeRemovalsByBlock := make([]*types.BlockHeightTopicIdReputerStakeRemovalInfo, 0)
	stakeRemovalsByBlockIter, err := k.stakeRemovalsByBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate stake removals by block")
	}
	for ; stakeRemovalsByBlockIter.Valid(); stakeRemovalsByBlockIter.Next() {
		keyValue, err := stakeRemovalsByBlockIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: stakeRemovalsByBlockIter")
		}
		value := keyValue.Value
		blockHeightTopicIdReputerStakeRemovalInfo := types.BlockHeightTopicIdReputerStakeRemovalInfo{
			BlockHeight:      keyValue.Key.K1(),
			TopicId:          keyValue.Key.K2(),
			Reputer:          value.Reputer,
			StakeRemovalInfo: &value,
		}
		stakeRemovalsByBlock = append(stakeRemovalsByBlock, &blockHeightTopicIdReputerStakeRemovalInfo)
	}
	data.StakeRemovalsByBlock = stakeRemovalsByBlock

	// stakeRemovalsByActor
	stakeRemovalsByActor := make([]*types.ActorIdTopicIdBlockHeight, 0)
	stakeRemovalsByActorIter, err := k.stakeRemovalsByActor.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate stake removals by actor")
	}
	for ; stakeRemovalsByActorIter.Valid(); stakeRemovalsByActorIter.Next() {
		key, err := stakeRemovalsByActorIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: stakeRemovalsByActorIter")
		}
		actorIdTopicIdBlockHeight := types.ActorIdTopicIdBlockHeight{
			ActorId:     key.K1(),
			TopicId:     key.K2(),
			BlockHeight: key.K3(),
		}
		stakeRemovalsByActor = append(stakeRemovalsByActor, &actorIdTopicIdBlockHeight)
	}
	data.StakeRemovalsByActor = stakeRemovalsByActor

	// delegateStakeRemovalsByBlock
	delegateStakeRemovalsByBlock := make([]*types.BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo, 0)
	delegateStakeRemovalsByBlockIter, err := k.delegateStakeRemovalsByBlock.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate delegate stake removals by block")
	}
	for ; delegateStakeRemovalsByBlockIter.Valid(); delegateStakeRemovalsByBlockIter.Next() {
		keyValue, err := delegateStakeRemovalsByBlockIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: delegateStakeRemovalsByBlockIter")
		}
		value := keyValue.Value
		blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo := types.BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo{
			BlockHeight:              keyValue.Key.K1(),
			TopicId:                  keyValue.Key.K2(),
			Reputer:                  value.Reputer,
			Delegator:                value.Delegator,
			DelegateStakeRemovalInfo: &value,
		}
		delegateStakeRemovalsByBlock = append(delegateStakeRemovalsByBlock, &blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo)
	}
	data.DelegateStakeRemovalsByBlock = delegateStakeRemovalsByBlock

	// delegateStakeRemovalsByActor
	delegateStakeRemovalsByActor := make([]*types.DelegatorReputerTopicIdBlockHeight, 0)
	delegateStakeRemovalsByActorIter, err := k.delegateStakeRemovalsByActor.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate delegate stake removals by actor")
	}
	for ; delegateStakeRemovalsByActorIter.Valid(); delegateStakeRemovalsByActorIter.Next() {
		key, err := delegateStakeRemovalsByActorIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: delegateStakeRemovalsByActorIter")
		}
		delegatorReputerTopicIdBlockHeight := types.DelegatorReputerTopicIdBlockHeight{
			Delegator:   key.K1(),
			Reputer:     key.K2(),
			TopicId:     key.K3(),
			BlockHeight: key.K4(),
		}
		delegateStakeRemovalsByActor = append(delegateStakeRemovalsByActor, &delegatorReputerTopicIdBlockHeight)
	}
	data.DelegateStakeRemovalsByActor = delegateStakeRemovalsByActor

	return nil
}
