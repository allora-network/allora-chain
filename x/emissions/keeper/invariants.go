package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInvariants registers the emissions module invariants.
func RegisterInvariants(ir sdk.InvariantRegistry, k *Keeper) {
	ir.RegisterRoute(emissionstypes.ModuleName, "allora-staking-total-balance", StakingInvariantTotalStakeEqualAlloraStakingBankBalance(*k))
	ir.RegisterRoute(emissionstypes.ModuleName, "allora-staking-total-topic-stake-equal-reputer-authority", StakingInvariantSumStakeFromStakeReputerAuthorityEqualTotalStakeAndTopicStake(*k))
	ir.RegisterRoute(emissionstypes.ModuleName, "stake-removals-length-same", StakingInvariantLenStakeRemovalsSame(*k))
	ir.RegisterRoute(emissionstypes.ModuleName, "stake-sum-delegated-stakes", StakingInvariantDelegatedStakes(*k))
	ir.RegisterRoute(emissionstypes.ModuleName, "pending-reward-for-delegators-equal-reward-per-share-minus-reward-debt", StakingInvariantPendingRewardForDelegatorsEqualRewardPerShareMinusRewardDebt(*k))
}

// AllInvariants is a convience function to run all invariants in the emissions module.
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		if res, stop := StakingInvariantTotalStakeEqualAlloraStakingBankBalance(k)(ctx); stop {
			return res, stop
		}
		if res, stop := StakingInvariantSumStakeFromStakeReputerAuthorityEqualTotalStakeAndTopicStake(k)(ctx); stop {
			return res, stop
		}
		if res, stop := StakingInvariantLenStakeRemovalsSame(k)(ctx); stop {
			return res, stop
		}
		if res, stop := StakingInvariantDelegatedStakes(k)(ctx); stop {
			return res, stop
		}
		if res, stop := StakingInvariantPendingRewardForDelegatorsEqualRewardPerShareMinusRewardDebt(k)(ctx); stop {
			return res, stop
		}
		return "", false
	}
}

// StakingInvariantTotalStakeEqualAlloraStakingBankBalance checks that
// the total stake in the emissions module is equal to the balance
// of the Allora staking bank account.
func StakingInvariantTotalStakeEqualAlloraStakingBankBalance(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		totalStake, err := k.GetTotalStake(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to get total stake: %v", err))
		}
		alloraStakingAddr := k.authKeeper.GetModuleAccount(ctx, emissionstypes.AlloraStakingAccountName).GetAddress()
		alloraStakingBalance := k.bankKeeper.GetBalance(
			ctx,
			alloraStakingAddr,
			params.DefaultBondDenom).Amount
		broken := !totalStake.Equal(alloraStakingBalance)
		return sdk.FormatInvariant(
			emissionstypes.ModuleName,
			"emissions module total stake equal allora staking bank balance",
			fmt.Sprintf("TotalStake: %s | Allora Module Account Staking Balance: %s",
				totalStake.String(),
				alloraStakingBalance.String(),
			),
		), broken
	}
}

// the number of values in the stakeRemovalsByBlock map
// should always equal the number of values in the stakeRemovalsByActor map
func StakingInvariantLenStakeRemovalsSame(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		iterByBlock, err := k.stakeRemovalsByBlock.Iterate(ctx, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to get stake removals iterator: %v", err))
		}
		defer iterByBlock.Close()
		valuesByBlock, err := iterByBlock.Values()
		if err != nil {
			panic(fmt.Sprintf("failed to get stake removals values: %v", err))
		}
		lenByBlock := len(valuesByBlock)
		iterByActor, err := k.stakeRemovalsByActor.Iterate(ctx, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to get stake removals iterator: %v", err))
		}
		defer iterByActor.Close()
		valuesByActor, err := iterByActor.Keys()
		if err != nil {
			panic(fmt.Sprintf("failed to get stake removals values: %v", err))
		}
		lenByActor := len(valuesByActor)

		broken := lenByBlock != lenByActor
		if broken {
			return sdk.FormatInvariant(
				emissionstypes.ModuleName,
				"emissions module length of stake removals same",
				fmt.Sprintf("Length of stake removals: %d | Length of stake removals: %d\n%v\n%v",
					lenByBlock,
					lenByActor,
					valuesByBlock,
					valuesByActor,
				),
			), broken
		}
		totalSumRemove := cosmosMath.ZeroInt()
		topicSumsRemove := make(map[uint64]cosmosMath.Int)
		for _, value := range valuesByBlock {
			topicSumBefore, has := topicSumsRemove[value.TopicId]
			if !has {
				topicSumBefore = cosmosMath.ZeroInt()
			}
			topicSumsRemove[value.TopicId] = topicSumBefore.Add(value.Amount)
			totalSumRemove = totalSumRemove.Add(value.Amount)
		}
		totalStake, err := k.totalStake.Get(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to get total stake: %v", err))
		}
		broken = !totalStake.GTE(totalSumRemove)
		if broken {
			return sdk.FormatInvariant(
				emissionstypes.ModuleName,
				"emissions module total stake greater than or equal to total stake removals",
				fmt.Sprintf("TotalStake: %s | TotalStakeRemove: %s",
					totalStake.String(),
					totalSumRemove.String(),
				),
			), broken
		}
		for topicId, topicSumRemove := range topicSumsRemove {
			topicStake, err := k.GetTopicStake(ctx, topicId)
			if err != nil {
				panic(fmt.Sprintf("failed to get topic stake: %v", err))
			}
			broken = !topicStake.GTE(topicSumRemove)
			if broken {
				return sdk.FormatInvariant(
					emissionstypes.ModuleName,
					"emissions module topic stake greater than or equal to topic stake removals",
					fmt.Sprintf("TopicId: %d | TopicStake: %s | TopicStakeRemove: %s",
						topicId,
						topicStake.String(),
						topicSumRemove.String(),
					),
				), broken
			}
		}
		return "", false
	}
}

// stakeSumFromDelegator = Sum(delegatedStakes[topicid, delegator, all reputers])
// stakeFromDelegatorsUponReputer = Sum(delegatedStakes[topicid, all delegators, reputer])
func StakingInvariantDelegatedStakes(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		numTopics, err := k.GetNextTopicId(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to get next topic id: %v", err))
		}
		for i := uint64(0); i < numTopics; i++ {
			rng := collections.NewPrefixedTripleRange[uint64, string, string](i)
			topicIter, err := k.delegatedStakes.Iterate(ctx, rng)
			if err != nil {
				panic(fmt.Sprintf("failed to get delegated stakes iterator: %v", err))
			}
			defer topicIter.Close()
			type ExpectedSumToComputedSum struct {
				expected cosmosMath.Int
				computed cosmosMath.Int
			}
			delegatorsToSumsMap := make(map[string]ExpectedSumToComputedSum)
			reputersToSumsMap := make(map[string]ExpectedSumToComputedSum)
			for ; topicIter.Valid(); topicIter.Next() {
				delegatorInfo, err := topicIter.KeyValue()
				if err != nil {
					panic(fmt.Sprintf("failed to get delegator info: %v", err))
				}
				delegator := delegatorInfo.Key.K2()
				reputer := delegatorInfo.Key.K3()
				amount := delegatorInfo.Value.Amount.SdkIntTrim()
				existingSumsDelegator, presentDelegator := delegatorsToSumsMap[delegator]
				if !presentDelegator {
					stakeSumForDelegator, err := k.stakeSumFromDelegator.Get(ctx, collections.Join(i, delegator))
					if err != nil {
						panic(fmt.Sprintf("failed to get stake sum from delegator: %v", err))
					}
					delegatorsToSumsMap[delegator] = ExpectedSumToComputedSum{
						expected: stakeSumForDelegator,
						computed: amount,
					}
				} else {
					newSum := existingSumsDelegator.computed.Add(amount)
					delegatorsToSumsMap[delegator] = ExpectedSumToComputedSum{
						expected: existingSumsDelegator.expected,
						computed: newSum,
					}
				}
				existingSumsReputer, presentReputer := reputersToSumsMap[reputer]
				if !presentReputer {
					stakeSumForReputer, err := k.stakeFromDelegatorsUponReputer.Get(ctx, collections.Join(i, reputer))
					if err != nil {
						panic(fmt.Sprintf("failed to get delegator stake upon reputer: %v", err))
					}
					reputersToSumsMap[reputer] = ExpectedSumToComputedSum{
						expected: stakeSumForReputer,
						computed: amount,
					}
				} else {
					newSum := existingSumsReputer.computed.Add(amount)
					reputersToSumsMap[reputer] = ExpectedSumToComputedSum{
						expected: existingSumsReputer.expected,
						computed: newSum,
					}
				}
			}
			for delegator, sums := range delegatorsToSumsMap {
				broken := !sums.expected.Equal(sums.computed)
				if broken {
					return sdk.FormatInvariant(
						emissionstypes.ModuleName,
						"emissions module stake sum from delegator equal sum of delegated stakes for that delegator",
						fmt.Sprintf("Topic Id: %d | Delegator: %s | sum from stakeSumFromDelegator: %s | sum from delegatedStakes: %s",
							i,
							delegator,
							sums.expected.String(),
							sums.computed.String(),
						),
					), broken
				}
			}
			for reputer, sums := range reputersToSumsMap {
				broken := !sums.expected.Equal(sums.computed)
				if broken {
					return sdk.FormatInvariant(
						emissionstypes.ModuleName,
						"emissions module stake sum from delegator upon reputer equal sum delegatedStakes upon reputer",
						fmt.Sprintf("Topic Id: %d | Reputer: %s | sum from stakeFromDelegatorsUponReputer: %s | sum from delegatedStakes: %s",
							i,
							reputer,
							sums.expected.String(),
							sums.computed.String(),
						),
					), broken
				}
			}
		}
		return "", false
	}
}

func StakingInvariantSumStakeFromStakeReputerAuthorityEqualTotalStakeAndTopicStake(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		totalStake, err := k.GetTotalStake(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to get total stake: %v", err))
		}
		numTopics, err := k.GetNextTopicId(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to get next topic id: %v", err))
		}
		sumTopicStakes := cosmosMath.ZeroInt()
		for i := uint64(0); i < numTopics; i++ {
			topicStake, err := k.GetTopicStake(ctx, i)
			if err != nil {
				panic(fmt.Sprintf("failed to get topic stake: %v", err))
			}
			sumTopicStakes = sumTopicStakes.Add(topicStake)

			sumReputersThisTopic := cosmosMath.ZeroInt()
			rng := collections.NewPrefixedPairRange[uint64, string](i)
			reputerAuthoritiesForTopicIter, err := k.stakeReputerAuthority.Iterate(ctx, rng)
			if err != nil {
				panic(fmt.Sprintf("failed to get reputer authorities iterator: %v", err))
			}
			defer reputerAuthoritiesForTopicIter.Close()
			for ; reputerAuthoritiesForTopicIter.Valid(); reputerAuthoritiesForTopicIter.Next() {
				reputerAuthority, err := reputerAuthoritiesForTopicIter.Value()
				if err != nil {
					panic(fmt.Sprintf("failed to get reputer authority: %v", err))
				}
				sumReputersThisTopic = sumReputersThisTopic.Add(reputerAuthority)
			}

			broken := !sumReputersThisTopic.Equal(topicStake)
			if broken {
				return sdk.FormatInvariant(
					emissionstypes.ModuleName,
					"emissions module sum stake from stake reputer authority equal topic stake",
					fmt.Sprintf("Sum of Stake from Stake Reputer Authority: %s | Topic Stake: %s | Topic ID: %d",
						sumReputersThisTopic.String(),
						topicStake.String(),
						i,
					),
				), broken
			}
		}
		broken := !totalStake.Equal(sumTopicStakes)
		return sdk.FormatInvariant(
			emissionstypes.ModuleName,
			"emissions module total stake equal sum of all topic stakes",
			fmt.Sprintf("TotalStake: %s | Sum of all Topic Stakes: %s | num topics :%d",
				totalStake.String(),
				sumTopicStakes.String(),
				numTopics,
			),
		), broken
	}
}

func StakingInvariantPendingRewardForDelegatorsEqualRewardPerShareMinusRewardDebt(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		// get the sum of all accumulated debts that happend to be staked upon a reputer
		iter, err := k.delegatedStakes.Iterate(ctx, nil)
		if err != nil {
			panic("failed to get delegated stakes iterator")
		}
		defer iter.Close()
		type TopicAndReputer struct {
			topicId uint64
			reputer string
		}
		rewardDebtSums := make(map[TopicAndReputer]alloraMath.Dec)
		reputerStakeSums := make(map[TopicAndReputer]alloraMath.Dec)
		for ; iter.Valid(); iter.Next() {
			keyValue, err := iter.KeyValue()
			if err != nil {
				panic("failed to get key value from delegatedStakes iterator")
			}
			topicAndReputer := TopicAndReputer{topicId: keyValue.Key.K1(), reputer: keyValue.Key.K3()}
			rewardDebtSum, has := rewardDebtSums[topicAndReputer]
			if !has {
				rewardDebtSum = alloraMath.ZeroDec()
			}
			rewardDebtSumNew, err := rewardDebtSum.Add(keyValue.Value.RewardDebt)
			if err != nil {
				panic("failed to add reward debt sum")
			}
			rewardDebtSums[topicAndReputer] = rewardDebtSumNew
			delegatedStakeSum, has := reputerStakeSums[topicAndReputer]
			if !has {
				delegatedStakeSum = alloraMath.ZeroDec()
			}
			delegatedStakeSumNew, err := delegatedStakeSum.Add(keyValue.Value.Amount)
			if err != nil {
				panic("failed to add stake sum")
			}
			reputerStakeSums[topicAndReputer] = delegatedStakeSumNew
		}
		// now for each reputer and topic, get the accumulated reward per share
		// then multiply it by the stake sum of the reputer
		iter2, err := k.delegateRewardPerShare.Iterate(ctx, nil)
		if err != nil {
			panic("failed to get delegate reward per share iterator")
		}
		defer iter2.Close()
		accumulatedRewardsBeyondRewardDebt := alloraMath.ZeroDec()
		for ; iter2.Valid(); iter2.Next() {
			keyValue, err := iter2.KeyValue()
			if err != nil {
				panic("failed to get key value from delegateRewardPerShare iterator")
			}
			topicAndReputer := TopicAndReputer{topicId: keyValue.Key.K1(), reputer: keyValue.Key.K2()}
			stakeSum := reputerStakeSums[topicAndReputer]
			rewardDebt := rewardDebtSums[topicAndReputer]
			stakeTimesRewardPerShare, err := stakeSum.Mul(keyValue.Value)
			if err != nil {
				panic("failed to multiply stake sum by reward per share")
			}
			stakeTimeRewardPerShareMinusRewardDebt, err := stakeTimesRewardPerShare.Sub(rewardDebt)
			if err != nil {
				panic("failed to subtract reward debt from stake times reward per share")
			}
			if stakeTimeRewardPerShareMinusRewardDebt.IsNegative() {
				panic(fmt.Sprintf("reward per share minus reward debt negative: stakeSum: %s, rewardPerShare: %s, rewardDebt: %s",
					stakeSum.String(), keyValue.Value.String(), rewardDebt.String(),
				))
			}
			accumulatedRewardsBeyondRewardDebt, err = accumulatedRewardsBeyondRewardDebt.Add(stakeTimeRewardPerShareMinusRewardDebt)
			if err != nil {
				panic("failed to add stake times reward per share minus reward debt to accumulated rewards")
			}
		}

		// now check the total pending rewards bank balance is equal to the accumulated rewards beyond reward debt
		alloraPendingAddr := k.authKeeper.GetModuleAccount(ctx, emissionstypes.AlloraPendingRewardForDelegatorAccountName).GetAddress()
		bal := k.GetBankBalance(ctx, alloraPendingAddr, params.DefaultBondDenom).Amount
		rewards := accumulatedRewardsBeyondRewardDebt.SdkIntTrim()
		// we define the invariant as holding if the rewards we think we
		// have to pay people is equal to the balance we have earmarked to pay them
		// OR if the balance we have earmarked to pay them is greater than the rewards
		// we think we owe them + 1 (for rounding error)
		notbroken := rewards.Equal(bal) || rewards.AddRaw(1).Equal(bal)
		broken := !notbroken
		if broken {
			return sdk.FormatInvariant(
				emissionstypes.ModuleName,
				"emissions module pending reward for delegators equal reward per share minus reward debt",
				fmt.Sprintf("Pending Reward For Delegators Module Account Balance: %s | Accumulated Rewards Beyond Reward Debt: %s | Accumulated Rewards Beyond Reward Debt as Dec: %s",
					bal.String(),
					rewards.String(),
					accumulatedRewardsBeyondRewardDebt.String(),
				),
			), broken
		}
		return "", false
	}
}
