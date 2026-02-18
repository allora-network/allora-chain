package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// ensure the module account exists
	stakingModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraStakingAccountName)
	k.authKeeper.SetModuleAccount(ctx, stakingModuleAccount)
	alloraRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraRewardsAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraRewardsModuleAccount)
	alloraPendingRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraPendingRewardsModuleAccount)

	// go through the genesis state object

	// params Params
	if err := k.paramsKeeper.SetParams(ctx, data.Params); err != nil {
		return errors.Wrap(err, "error setting params")
	}
	// nextTopicId uint64
	if data.NextTopicId == 0 {
		// reserve topic ID 0 for future use
		if _, err := k.topicKeeper.IncrementTopicId(ctx); err != nil {
			return errors.Wrap(err, "error incrementing topic ID")
		}
	} else {
		if err := k.topicKeeper.SetNextTopicId(ctx, data.NextTopicId); err != nil {
			return errors.Wrap(err, "error setting next topic ID")
		}
	}
	// Topics       []*TopicIdAndTopic
	for _, topic := range data.Topics {
		if topic != nil {
			if err := k.topicKeeper.SetTopic(ctx, topic.TopicId, *topic.Topic); err != nil {
				return errors.Wrap(err, "error setting topic")
			}
		}
	}
	// ActiveTopics []uint64
	for _, topicId := range data.ActiveTopics {
		if err := types.ValidateTopicId(topicId); err != nil {
			return errors.Wrapf(err, "error setting activeTopics %v", data.ActiveTopics)
		}
		if err := k.topicKeeper.activeTopics.Set(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting activeTopics")
		}
	}
	// RewardableTopics []uint64
	for _, topicId := range data.RewardableTopics {
		if err := k.topicKeeper.SetRewardableTopic(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting rewardableTopics")
		}
	}
	// TopicWorkers []*TopicAndActorId
	for _, topicAndActorId := range data.TopicWorkers {
		if topicAndActorId != nil {
			if err := types.ValidateTopicId(topicAndActorId.TopicId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
			if err := types.ValidateBech32(topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
			if err := k.workerKeeper.SetTopicWorker(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
		}
	}
	// TopicReputers []*TopicAndActorId
	for _, topicAndActorId := range data.TopicReputers {
		if topicAndActorId != nil {
			if err := types.ValidateTopicId(topicAndActorId.TopicId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
			if err := types.ValidateBech32(topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
			if err := k.reputerLossKeeper.SetTopicReputer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
		}
	}
	// TopicRewardNonce []*TopicIdAndBlockHeight
	for _, topicIdAndBlockHeight := range data.TopicRewardNonce {
		if topicIdAndBlockHeight != nil {
			if err := k.topicKeeper.SetTopicRewardNonce(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting topicRewardNonce")
			}
		}
	}

	// InfererScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.InfererScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := k.scoresKeeper.SetInfererScoresByBlock(ctx,
				topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight,
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting infererScoresByBlock")
			}
		}
	}
	// ForecasterScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.ForecasterScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := k.scoresKeeper.SetForecasterScoresByBlock(
				ctx, topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight,
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting forecasterScoresByBlock")
			}
		}
	}

	// ReputerScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.ReputerScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := types.ValidateTopicId(topicIdBlockHeightScores.TopicId); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
			if err := types.ValidateBlockHeight(topicIdBlockHeightScores.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
			if err := topicIdBlockHeightScores.Scores.Validate(); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
			if err := k.scoresKeeper.reputerScoresByBlock.Set(
				ctx,
				collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
		}
	}

	// LatestInfererScoresByWorker []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.InfererScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.scoresKeeper.SetInfererScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestInfererScoresByWorker")
			}
		}
	}
	// LatestForecasterScoresByWorker []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.ForecasterScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.scoresKeeper.SetForecasterScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestForecasterScoresByWorker")
			}
		}
	}
	// LatestReputerScoresByReputer []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.ReputerScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.scoresKeeper.SetReputerScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestReputerScoresByReputer")
			}
		}
	}
	// ReputerListeningCoefficient []*TopicIdActorIdListeningCoefficient
	for _, topicIdActorIdListeningCoefficient := range data.ReputerListeningCoefficient {
		if topicIdActorIdListeningCoefficient != nil {
			if err := k.scoresKeeper.SetListeningCoefficient(ctx,
				topicIdActorIdListeningCoefficient.TopicId, topicIdActorIdListeningCoefficient.ActorId,
				*topicIdActorIdListeningCoefficient.ListeningCoefficient); err != nil {
				return errors.Wrap(err, "error setting reputerListeningCoefficient")
			}
		}
	}
	// PreviousReputerRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousReputerRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.scoresKeeper.SetPreviousReputerRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousReputerRewardFraction")
			}
		}
	}
	// PreviousInferenceRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousInferenceRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.scoresKeeper.SetPreviousInferenceRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousInferenceRewardFraction")
			}
		}
	}
	// PreviousForecastRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousForecastRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.scoresKeeper.SetPreviousForecastRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousForecastRewardFraction")
			}
		}
	}
	// TotalStake cosmossdk_io_math.Int
	if data.TotalStake.GT(cosmosMath.ZeroInt()) {
		if err := k.stakingKeeper.SetTotalStake(ctx, data.TotalStake); err != nil {
			return errors.Wrap(err, "error setting totalStake")
		}
	} else {
		if err := k.stakingKeeper.SetTotalStake(ctx, cosmosMath.ZeroInt()); err != nil {
			return errors.Wrap(err, "error setting totalStake to zero int")
		}
	}
	// TopicStake []*TopicIdAndInt
	for _, topicIdAndInt := range data.TopicStake {
		if topicIdAndInt != nil {
			if err := k.stakingKeeper.SetTopicStake(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "error setting topicStake")
			}
		}
	}
	// StakeReputerAuthority []*TopicIdActorIdInt
	for _, topicIdActorIdInt := range data.StakeReputerAuthority {
		if topicIdActorIdInt != nil {
			if err := k.stakingKeeper.SetStakeReputerAuthority(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeReputerAuthority")
			}
		}
	}
	// StakeSumFromDelegator []*TopicIdActorIdInt
	for _, topicIdActorIdInt := range data.StakeSumFromDelegator {
		if topicIdActorIdInt != nil {
			if err := k.stakingKeeper.SetStakeFromDelegator(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeSumFromDelegator")
			}
		}
	}
	// DelegatedStakes []*TopicIdDelegatorReputerDelegatorInfo
	for _, topicIdDelegatorReputerDelegatorInfo := range data.DelegatedStakes {
		if topicIdDelegatorReputerDelegatorInfo != nil {
			if err := k.stakingKeeper.SetDelegateStakePlacement(ctx,
				topicIdDelegatorReputerDelegatorInfo.TopicId,
				topicIdDelegatorReputerDelegatorInfo.Delegator,
				topicIdDelegatorReputerDelegatorInfo.Reputer,
				*topicIdDelegatorReputerDelegatorInfo.DelegatorInfo); err != nil {
				return errors.Wrap(err, "error setting delegatedStakes")
			}
		}
	}
	// StakeFromDelegatorsUponReputer []*TopicIdActorIdInt
	for _, topicIdActorIdInt := range data.StakeFromDelegatorsUponReputer {
		if topicIdActorIdInt != nil {
			if err := k.stakingKeeper.SetDelegateStakeUponReputer(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeFromDelegatorsUponReputer")
			}
		}
	}
	// DelegateRewardPerShare []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.DelegateRewardPerShare {
		if topicIdActorIdDec != nil {
			if err := k.stakingKeeper.SetDelegateRewardPerShare(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting delegateRewardPerShare")
			}
		}
	}
	// StakeRemovalsByBlock []*BlockHeightTopicIdReputerStakeRemovalInfo
	// StakeRemovalsByActor []*ActorIdTopicIdBlockHeight
	for _, blockHeightTopicIdReputerStakeRemovalInfo := range data.StakeRemovalsByBlock {
		if blockHeightTopicIdReputerStakeRemovalInfo != nil {
			if err := k.stakingKeeper.SetStakeRemoval(ctx,
				*blockHeightTopicIdReputerStakeRemovalInfo.StakeRemovalInfo); err != nil {
				return errors.Wrapf(err, "error setting stakeRemovalsByBlock %v",
					*blockHeightTopicIdReputerStakeRemovalInfo.StakeRemovalInfo,
				)
			}
		}
	}
	// DelegateStakeRemovalsByBlock []*BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo
	// DelegateStakeRemovalsByActor []*DelegatorReputerTopicIdBlockHeight
	for _, blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo := range data.DelegateStakeRemovalsByBlock {
		if blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo != nil {
			if err := k.stakingKeeper.SetDelegateStakeRemoval(ctx,
				*blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.DelegateStakeRemovalInfo); err != nil {
				return errors.Wrap(err, "error setting delegateStakeRemovalsByBlock")
			}
		}
	}
	// Inferences []*TopicIdActorIdInference
	for _, topicIdActorIdInference := range data.Inferences {
		if topicIdActorIdInference != nil {
			if err := topicIdActorIdInference.Inference.Validate(); err != nil {
				return errors.Wrap(err, "inference in list is invalid")
			}
			if err := k.workerKeeper.inferences.Set(ctx,
				collections.Join(
					topicIdActorIdInference.TopicId,
					topicIdActorIdInference.ActorId),
				*topicIdActorIdInference.Inference); err != nil {
				return errors.Wrap(err, "error setting inferences")
			}
		}
	}

	// Forecasts []*TopicIdActorIdForecast
	for _, topicIdActorIdForecast := range data.Forecasts {
		if topicIdActorIdForecast != nil {
			if err := topicIdActorIdForecast.Forecast.Validate(); err != nil {
				return errors.Wrap(err, "forecast in list is invalid")
			}
			if err := k.workerKeeper.forecasts.Set(ctx,
				collections.Join(
					topicIdActorIdForecast.TopicId,
					topicIdActorIdForecast.ActorId),
				*topicIdActorIdForecast.Forecast); err != nil {
				return errors.Wrap(err, "error setting forecasts")
			}
		}
	}

	// Workers []*LibP2PKeyAndOffchainNode
	for _, libP2PKeyAndOffchainNode := range data.Workers {
		if libP2PKeyAndOffchainNode != nil {
			if err := libP2PKeyAndOffchainNode.OffchainNode.Validate(); err != nil {
				return errors.Wrap(err, "worker info validation failed")
			}
			if err := k.workerKeeper.workers.Set(
				ctx,
				libP2PKeyAndOffchainNode.LibP2PKey,
				*libP2PKeyAndOffchainNode.OffchainNode); err != nil {
				return errors.Wrap(err, "error setting workers")
			}
		}
	}

	// Reputers []*LibP2PKeyAndOffchainNode
	for _, libP2PKeyAndOffchainNode := range data.Reputers {
		if libP2PKeyAndOffchainNode != nil {
			if err := libP2PKeyAndOffchainNode.OffchainNode.Validate(); err != nil {
				return errors.Wrap(err, "reputer info validation failed")
			}
			if err := k.reputerLossKeeper.reputers.Set(
				ctx,
				libP2PKeyAndOffchainNode.LibP2PKey,
				*libP2PKeyAndOffchainNode.OffchainNode); err != nil {
				return errors.Wrap(err, "error setting reputers")
			}
		}
	}

	// TopicFeeRevenue []*TopicIdAndInt
	for _, topicIdAndInt := range data.TopicFeeRevenue {
		if topicIdAndInt != nil {
			if err := types.ValidateTopicId(topicIdAndInt.TopicId); err != nil {
				return errors.Wrap(err, "topic id validation failed")
			}
			if err := types.ValidateSdkIntRepresentingMonetaryValue(topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "topic fee revenue validation failed")
			}
			if err := k.topicKeeper.topicFeeRevenue.Set(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "error setting topicFeeRevenue")
			}
		}
	}

	// PreviousTopicWeight []*TopicIdAndDec
	for _, topicIdAndDec := range data.PreviousTopicWeight {
		if topicIdAndDec != nil {
			if err := k.topicKeeper.SetPreviousTopicWeight(
				ctx,
				topicIdAndDec.TopicId,
				topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicWeight")
			}
		}
	}

	// AllInferences []*TopicIdBlockHeightInferences
	for _, topicIdBlockHeightInferences := range data.AllInferences {
		if topicIdBlockHeightInferences != nil {
			for _, inference := range topicIdBlockHeightInferences.Inferences.Inferences {
				if inference != nil {
					if err := inference.Validate(); err != nil {
						return errors.Wrap(err, "inference validation failed")
					}
				}
			}
			if err := k.workerKeeper.allInferences.Set(ctx,
				collections.Join(topicIdBlockHeightInferences.TopicId, topicIdBlockHeightInferences.BlockHeight),
				*topicIdBlockHeightInferences.Inferences); err != nil {
				return errors.Wrap(err, "error setting allInferences")
			}
		}
	}
	// AllForecasts []*TopicIdBlockHeightForecasts
	for _, topicIdBlockHeightForecasts := range data.AllForecasts {
		if topicIdBlockHeightForecasts != nil {
			for _, forecast := range topicIdBlockHeightForecasts.Forecasts.Forecasts {
				if forecast != nil {
					if err := forecast.Validate(); err != nil {
						return errors.Wrap(err, "forecast validation failed")
					}
				}
			}
			if err := k.workerKeeper.allForecasts.Set(ctx,
				collections.Join(topicIdBlockHeightForecasts.TopicId, topicIdBlockHeightForecasts.BlockHeight),
				*topicIdBlockHeightForecasts.Forecasts); err != nil {
				return errors.Wrap(err, "error setting allForecasts")
			}
		}
	}

	// AllLossBundles []*TopicIdBlockHeightReputerValueBundles
	for _, topicIdBlockHeightReputerValueBundles := range data.AllLossBundles {
		lossBundles := types.LossBundles(topicIdBlockHeightReputerValueBundles.GetReputerValueBundles())
		if topicIdBlockHeightReputerValueBundles != nil {
			if err := lossBundles.Validate(); err != nil {
				return errors.Wrap(err, "reputer value bundles validation failed")
			}
			reputerValueBundles := make([]*types.ReputerValueBundle, len(lossBundles))
			for i := range lossBundles {
				reputerValueBundles[i] = &types.ReputerValueBundle{
					ValueBundle: lossBundles[i],
				}
			}
			if err := k.reputerLossKeeper.allLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightReputerValueBundles.TopicId, topicIdBlockHeightReputerValueBundles.BlockHeight),
				types.ReputerValueBundles{ReputerValueBundles: reputerValueBundles}); err != nil {
				return errors.Wrap(err, "error setting allLossBundles")
			}
		}
	}

	// NetworkLossBundles []*TopicIdBlockHeightValueBundles
	for _, topicIdBlockHeightValueBundles := range data.NetworkLossBundles {
		if topicIdBlockHeightValueBundles != nil {
			if err := topicIdBlockHeightValueBundles.ValueBundle.Validate(); err != nil {
				return errors.Wrap(err, "value bundle validation failed")
			}
			if err := k.reputerLossKeeper.networkLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightValueBundles.TopicId, topicIdBlockHeightValueBundles.BlockHeight),
				*topicIdBlockHeightValueBundles.ValueBundle); err != nil {
				return errors.Wrap(err, "error setting networkLossBundles")
			}
		}
	}

	// PreviousPercentageRewardToStakedReputers github_com_allora_network_allora_chain_math.Dec
	if data.PreviousPercentageRewardToStakedReputers != alloraMath.ZeroDec() {
		if err := k.scoresKeeper.SetPreviousPercentageRewardToStakedReputers(ctx, data.PreviousPercentageRewardToStakedReputers); err != nil {
			return errors.Wrap(err, "error setting previousPercentageRewardToStakedReputers")
		}
	} else {
		// For mint module inflation rate calculation set the initial
		// "previous percentage of rewards that went to staked reputers" to 30%
		if err := k.scoresKeeper.SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.3")); err != nil {
			return errors.Wrap(err, "error setting previousPercentageRewardToStakedReputers to 0.3")
		}
	}
	// openWorkerWindows []*BlockHeightAndListOfTopicIds
	for _, blockHeightAndListOfTopicIds := range data.OpenWorkerWindows {
		if blockHeightAndListOfTopicIds != nil {
			topicIds := types.TopicIds{TopicIds: blockHeightAndListOfTopicIds.TopicIds}
			for _, topicId := range topicIds.TopicIds {
				if err := types.ValidateTopicId(topicId); err != nil {
					return errors.Wrap(err, "error validating topic id")
				}
			}
			if err := types.ValidateBlockHeight(blockHeightAndListOfTopicIds.BlockHeight); err != nil {
				return errors.Wrap(err, "error validating block height")
			}
			if err := k.nonceKeeper.openWorkerWindows.Set(
				ctx,
				blockHeightAndListOfTopicIds.BlockHeight,
				topicIds,
			); err != nil {
				return errors.Wrap(err, "error setting openWorkerWindows")
			}
		}
	}

	// UnfulfilledWorkerNonces []*TopicIdAndNonces

	for _, topicIdAndNonces := range data.UnfulfilledWorkerNonces {
		if topicIdAndNonces != nil {
			if err := topicIdAndNonces.Nonces.Validate(); err != nil {
				return errors.Wrap(err, "error validating unfulfilled worker nonces")
			}
			if err := k.nonceKeeper.unfulfilledWorkerNonces.Set(ctx, topicIdAndNonces.TopicId, *topicIdAndNonces.Nonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledWorkerNonces")
			}
		}
	}
	// UnfulfilledReputerNonces []*TopicIdAndReputerRequestNonces

	for _, topicIdAndReputerRequestNonces := range data.UnfulfilledReputerNonces {
		if topicIdAndReputerRequestNonces != nil {
			if err := topicIdAndReputerRequestNonces.ReputerRequestNonces.Validate(); err != nil {
				return errors.Wrap(err, "error validating unfulfilled reputer nonces")
			}
			if err := k.nonceKeeper.unfulfilledReputerNonces.Set(ctx, topicIdAndReputerRequestNonces.TopicId, *topicIdAndReputerRequestNonces.ReputerRequestNonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledReputerNonces")
			}
		}
	}

	// lastDripBlock []*TopicIdAndBlockHeight
	for _, topicIdAndBlockHeight := range data.LastDripBlock {
		if topicIdAndBlockHeight != nil {
			if err := k.topicKeeper.SetLastDripBlock(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting lastDripBlock")
			}
		}
	}

	// LatestInfererNetworkRegrets []*TopicIdActorIdTimeStampedValue
	for _, topicIdActorIdTimeStampedValue := range data.LatestInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestInfererNetworkRegrets")
			}
		}
	}
	// LatestNaiveInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestNaiveInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetNaiveInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestNaiveInfererNetworkRegrets")
			}
		}
	}
	// LatestForecasterNetworkRegrets []*TopicIdActorIdTimeStampedValue
	for _, topicIdActorIdTimeStampedValue := range data.LatestForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetForecasterNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestForecasterNetworkRegrets")
			}
		}
	}
	// LatestOneOutInfererInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutInfererInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetOneOutInfererInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutInfererInfererNetworkRegrets")
			}
		}
	}
	// LatestOneOutInfererForecasterNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutInfererForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.latestOneOutInfererForecasterNetworkRegrets.Set(ctx,
				collections.Join3(
					topicIdActorIdTimeStampedValue.TopicId,
					topicIdActorIdTimeStampedValue.ActorId1,
					topicIdActorIdTimeStampedValue.ActorId2,
				),
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutInfererForecasterNetworkRegrets")
			}
		}
	}
	// LatestOneOutForecasterInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutForecasterInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetOneOutForecasterInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutForecasterInfererNetworkRegrets")
			}
		}
	}
	// LatestOneOutForecasterForecasterNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutForecasterForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetOneOutForecasterForecasterNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutForecasterForecasterNetworkRegrets")
			}
		}
	}
	// LatestOneInForecasterNetworkRegrets []*TopicIdActorIdActorIdTimeStampedValue
	for _, topicIdActorIdActorIdTimeStampedValue := range data.LatestOneInForecasterNetworkRegrets {
		if topicIdActorIdActorIdTimeStampedValue != nil {
			if err := k.regretsKeeper.SetOneInForecasterNetworkRegret(ctx,
				topicIdActorIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneInForecasterNetworkRegrets")
			}
		}
	}
	// PreviousForecasterScoreRatio
	for _, topicIdDec := range data.PreviousForecasterScoreRatio {
		if topicIdDec != nil {
			if err := k.scoresKeeper.SetPreviousForecasterScoreRatio(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousForecasterScoreRatio")
			}
		}
	}
	// CoreTeamAddresses, WhitelistAdmins []string
	// This allows us to add core team addresses to the whitelist during a genesis import
	// while still keeping the original core team addresses in the genesis file
	if len(data.CoreTeamAddresses) != 0 || len(data.WhitelistAdmins) != 0 {
		// make sure what we are storing is not garbage
		for _, address := range append(data.CoreTeamAddresses, data.WhitelistAdmins...) {
			_, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				return errors.Wrap(err, "error converting core team address from bech32")
			}
			err = k.whitelistsKeeper.AddWhitelistAdmin(ctx, address)
			if err != nil {
				return errors.Wrap(err, "error adding core team addresses to whitelists")
			}
		}
	}
	// TopicLastWorkerCommit   []*TopicIdTimestampedActorNonce
	for _, topicIdTimestampedActorNonce := range data.TopicLastWorkerCommit {
		if topicIdTimestampedActorNonce != nil {
			if err := k.topicKeeper.SetWorkerTopicLastCommit(ctx,
				topicIdTimestampedActorNonce.TopicId,
				topicIdTimestampedActorNonce.TimestampedActorNonce.BlockHeight,
				topicIdTimestampedActorNonce.TimestampedActorNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastWorkerCommit")
			}
		}
	}
	// TopicLastReputerCommit  []*TopicIdTimestampedActorNonce
	for _, topicIdTimestampedActorNonce := range data.TopicLastReputerCommit {
		if topicIdTimestampedActorNonce != nil {
			if err := k.topicKeeper.SetReputerTopicLastCommit(ctx,
				topicIdTimestampedActorNonce.TopicId,
				topicIdTimestampedActorNonce.TimestampedActorNonce.BlockHeight,
				topicIdTimestampedActorNonce.TimestampedActorNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastReputerCommit")
			}
		}
	}

	// TopicToNextPossibleChurningBlock []*topicBlock
	for _, topicBlock := range data.TopicToNextPossibleChurningBlock {
		if topicBlock != nil {
			if err := k.topicKeeper.SetTopicToNextPossibleChurningBlock(ctx,
				topicBlock.TopicId,
				topicBlock.BlockHeight); err != nil {
				return errors.Wrapf(err, "error setting topicToNextPossibleChurningBlock %v", topicBlock)
			}
		}
	}

	// BlockToActiveTopics []*blockToActiveTopics
	for _, blockToActiveTopics := range data.BlockToActiveTopics {
		if blockToActiveTopics != nil {
			if err := k.topicKeeper.blockToActiveTopics.Set(ctx,
				blockToActiveTopics.BlockHeight,
				*blockToActiveTopics.TopicIds); err != nil {
				return errors.Wrap(err, "error setting blockToActiveTopics")
			}
		}
	}

	// BlockToLowestActiveTopicWeight []*blockToLowestActiveTopicWeight
	for _, lowestActiveTopicWeight := range data.BlockToLowestActiveTopicWeight {
		if lowestActiveTopicWeight != nil {
			if err := k.topicKeeper.blockToLowestActiveTopicWeight.Set(ctx,
				lowestActiveTopicWeight.BlockHeight,
				*lowestActiveTopicWeight.TopicWeight); err != nil {
				return errors.Wrap(err, "error setting blockToLowestActiveTopicWeight")
			}
		}
	}

	// PreviousTopicQuantileInfererScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileInfererScoreEma {
		if topicIdDec != nil {
			if err := k.scoresKeeper.SetPreviousTopicQuantileInfererScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileInfererScoreEma")
			}
		}
	}

	// PreviousTopicQuantileForecasterScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileForecasterScoreEma {
		if topicIdDec != nil {
			if err := k.scoresKeeper.SetPreviousTopicQuantileForecasterScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileForecasterScoreEma")
			}
		}
	}

	// PreviousTopicQuantileReputerScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileReputerScoreEma {
		if topicIdDec != nil {
			if err := k.scoresKeeper.SetPreviousTopicQuantileReputerScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileReputerScoreEma")
			}
		}
	}

	// InitialInfererEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialInfererEmaScore {
		if topicIdAndDec != nil {
			if err := k.scoresKeeper.initialInfererEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialInfererEmaScore")
			}
		}
	}

	// InitialForecasterEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialForecasterEmaScore {
		if topicIdAndDec != nil {
			if err := k.scoresKeeper.initialForecasterEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialForecasterEmaScore")
			}
		}
	}

	// InitialReputerEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialReputerEmaScore {
		if topicIdAndDec != nil {
			if err := k.scoresKeeper.initialReputerEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialReputerEmaScore")
			}
		}
	}

	// ActiveInferers []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveInferers {
		if topicAndActorId != nil {
			if err := k.workerKeeper.AddActiveInferer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeInferers")
			}
		}
	}

	// ActiveForecasters []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveForecasters {
		if topicAndActorId != nil {
			if err := k.workerKeeper.AddActiveForecaster(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeForecasters")
			}
		}
	}

	// LowestInfererScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestInfererScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.scoresKeeper.SetLowestInfererScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestInfererScoreEma")
			}
		}
	}

	// LowestForecasterScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestForecasterScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.scoresKeeper.SetLowestForecasterScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestForecasterScoreEma")
			}
		}
	}

	// ActiveReputers []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveReputers {
		if topicAndActorId != nil {
			if err := k.reputerLossKeeper.AddActiveReputer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeReputers")
			}
		}
	}

	// LowestReputerScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestReputerScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.scoresKeeper.SetLowestReputerScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestReputerScoreEma")
			}
		}
	}

	// LossBundles
	for _, bundle := range data.LossBundles {
		if bundle != nil {
			key := collections.Join(bundle.TopicId, bundle.Reputer)
			if err := k.reputerLossKeeper.lossBundles.Set(ctx, key, types.ReputerValueBundle{
				ValueBundle: bundle.ReputerValueBundle,
			}); err != nil {
				return errors.Wrap(err, "error setting loss bundle")
			}
		}
	}

	// CountInfererInclusionsInTopicActiveSet
	for _, topicIdInfererCount := range data.CountInfererInclusionsInTopicActiveSet {
		if topicIdInfererCount != nil {
			if err := k.countInfererInclusionsInTopicActiveSet.Set(ctx, collections.Join(topicIdInfererCount.TopicId, topicIdInfererCount.ActorId), topicIdInfererCount.Uint64); err != nil {
				return errors.Wrap(err, "error setting countInfererInclusionsInTopicActiveSet")
			}
		}
	}

	// CountForecasterInclusionsInTopicActiveSet
	for _, topicIdForecasterCount := range data.CountForecasterInclusionsInTopicActiveSet {
		if topicIdForecasterCount != nil {
			if err := k.countForecasterInclusionsInTopicActiveSet.Set(ctx, collections.Join(topicIdForecasterCount.TopicId, topicIdForecasterCount.ActorId), topicIdForecasterCount.Uint64); err != nil {
				return errors.Wrap(err, "error setting countForecasterInclusionsInTopicActiveSet")
			}
		}
	}

	// TotalSumPreviousTopicWeights
	if data.TotalSumPreviousTopicWeights.Gt(alloraMath.ZeroDec()) {
		if err := k.topicKeeper.SetTotalSumPreviousTopicWeights(ctx, data.TotalSumPreviousTopicWeights); err != nil {
			return errors.Wrap(err, "error setting TotalSumPreviousTopicWeights")
		}
	} else {
		if err := k.topicKeeper.SetTotalSumPreviousTopicWeights(ctx, alloraMath.ZeroDec()); err != nil {
			return errors.Wrap(err, "error setting TotalSumPreviousTopicWeights to zero int")
		}
	}

	// RewardsCurrentBlockEmission cosmossdk_io_math.Int
	if data.RewardCurrentBlockEmission.GT(cosmosMath.ZeroInt()) {
		if err := k.SetRewardCurrentBlockEmission(ctx, data.RewardCurrentBlockEmission); err != nil {
			return errors.Wrap(err, "error setting RewardCurrentBlockEmission")
		}
	} else {
		if err := k.SetRewardCurrentBlockEmission(ctx, cosmosMath.ZeroInt()); err != nil {
			return errors.Wrap(err, "error setting RewardCurrentBlockEmission to zero int")
		}
	}

	// globalWhitelist
	for _, address := range data.GlobalWhitelist {
		if err := k.whitelistsKeeper.AddToGlobalWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalWhitelist")
		}
	}

	// globalWorkerWhitelist
	for _, address := range data.GlobalWorkerWhitelist {
		if err := k.whitelistsKeeper.AddToGlobalWorkerWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalWorkerWhitelist")
		}
	}

	// globalReputerWhitelist
	for _, address := range data.GlobalReputerWhitelist {
		if err := k.whitelistsKeeper.AddToGlobalReputerWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalReputerWhitelist")
		}
	}

	// globalAdminWhitelist
	for _, address := range data.GlobalAdminWhitelist {
		if err := k.whitelistsKeeper.AddToGlobalAdminWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalAdminWhitelist")
		}
	}

	// topicCreatorWhitelist
	for _, address := range data.TopicCreatorWhitelist {
		if err := k.whitelistsKeeper.AddToTopicCreatorWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting topicCreatorWhitelist")
		}
	}

	// topicWorkerWhitelist
	for _, topicAndActorId := range data.TopicWorkerWhitelist {
		if topicAndActorId != nil {
			if err := k.whitelistsKeeper.AddToTopicWorkerWhitelist(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkerWhitelist")
			}
		}
	}

	// topicReputerWhitelist
	for _, topicAndActorId := range data.TopicReputerWhitelist {
		if topicAndActorId != nil {
			if err := k.whitelistsKeeper.AddToTopicReputerWhitelist(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputerWhitelist")
			}
		}
	}

	// topicWorkerWhitelistEnabled
	for _, topicId := range data.TopicWorkerWhitelistEnabled {
		if err := k.whitelistsKeeper.EnableTopicWorkerWhitelist(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting topicWorkerWhitelistEnabled")
		}
	}

	// topicReputerWhitelistEnabled
	for _, topicId := range data.TopicReputerWhitelistEnabled {
		if err := k.whitelistsKeeper.EnableTopicReputerWhitelist(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting topicReputerWhitelistEnabled")
		}
	}

	// LastMedianInferences
	for _, topicIdAndDec := range data.LastMedianInferences {
		if topicIdAndDec != nil {
			if err := k.topicKeeper.SetLastMedianInferences(
				ctx,
				topicIdAndDec.TopicId,
				topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting lastMedianInferences")
			}
		}
	}

	// madInferences
	for _, topicIdDec := range data.MadInferences {
		if topicIdDec != nil {
			if err := k.topicKeeper.SetMadInferences(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting madInferences")
			}
		}
	}

	// Initialize latest regret stdnorm
	for _, stdnorm := range data.LatestRegretStdNorm {
		if stdnorm != nil {
			if err := k.weightsKeeper.SetLatestRegretStdNorm(ctx, stdnorm.TopicId, stdnorm.Dec); err != nil {
				return errors.Wrap(err, "error setting latest regret stdnorm")
			}
		}
	}

	// Initialize latest inferer weights
	for _, weight := range data.LatestInfererWeights {
		if weight != nil {
			if err := k.weightsKeeper.SetLatestInfererWeight(ctx, weight.TopicId, weight.ActorId, weight.Dec); err != nil {
				return errors.Wrap(err, "error setting latest inferer weight")
			}
		}
	}

	// Initialize latest forecaster weights
	for _, weight := range data.LatestForecasterWeights {
		if weight != nil {
			if err := k.weightsKeeper.SetLatestForecasterWeight(ctx, weight.TopicId, weight.ActorId, weight.Dec); err != nil {
				return errors.Wrap(err, "error setting latest forecaster weight")
			}
		}
	}

	// NetworkInferences
	for _, networkInference := range data.NetworkInferences {
		if networkInference != nil {
			if err := k.InsertNetworkInferences(ctx, networkInference.TopicId, networkInference.BlockHeight, *networkInference.ValueBundle); err != nil {
				return errors.Wrap(err, "error setting network inference")
			}
		}
	}

	// OutlierResistantNetworkInferences
	for _, outlierResistantNetworkInference := range data.OutlierResistantNetworkInferences {
		if outlierResistantNetworkInference != nil {
			if err := k.InsertOutlierResistantNetworkInferences(ctx, outlierResistantNetworkInference.TopicId, outlierResistantNetworkInference.BlockHeight, *outlierResistantNetworkInference.ValueBundle); err != nil {
				return errors.Wrap(err, "error setting outlier resistant network inference")
			}
		}
	}

	// MonthlyReputerRewards
	if err := types.ValidateSdkIntRepresentingMonetaryValue(data.MonthlyReputerRewards); err != nil {
		return errors.Wrap(err, "monthly reputer rewards validation failed")
	}
	if err := types.ValidateSdkIntRepresentingMonetaryValue(data.MonthlyTopicRewards); err != nil {
		return errors.Wrap(err, "monthly topic rewards validation failed")
	}
	// Will set to zero both monthlyReputerRewards and monthlyTopicRewards
	if err := k.weightsKeeper.ResetMonthlyRewards(ctx); err != nil {
		return errors.Wrap(err, "error setting monthlyTopicRewards")
	}

	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	moduleParams, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get module params")
	}

	nextTopicId, err := k.topicKeeper.nextTopicId.Peek(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get next topic ID")
	}

	topicsIter, err := k.topicKeeper.topics.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topics")
	}
	topics := make([]*types.TopicIdAndTopic, 0)
	for ; topicsIter.Valid(); topicsIter.Next() {
		keyValue, err := topicsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicsIter")
		}
		value := keyValue.Value
		topic := types.TopicIdAndTopic{
			TopicId: keyValue.Key,
			Topic:   &value,
		}
		topics = append(topics, &topic)
	}

	activeTopics := make([]uint64, 0)
	activeTopicsIter, err := k.topicKeeper.activeTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate active topics")
	}
	for ; activeTopicsIter.Valid(); activeTopicsIter.Next() {
		key, err := activeTopicsIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: activeTopicsIter")
		}
		activeTopics = append(activeTopics, key)
	}

	topicNextChurningBlock, err := k.topicKeeper.topicToNextPossibleChurningBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topicToNextPossibleChurningBlock")
	}
	topicToNextPossibleChurningBlock := make([]*types.TopicIdAndBlockHeight, 0)
	for ; topicNextChurningBlock.Valid(); topicNextChurningBlock.Next() {
		keyValue, err := topicNextChurningBlock.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicToNextPossibleChurningBlock")
		}
		value := keyValue.Value
		topic := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: value,
		}
		topicToNextPossibleChurningBlock = append(topicToNextPossibleChurningBlock, &topic)
	}

	blockActiveTopics, err := k.topicKeeper.blockToActiveTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate blockActiveTopics")
	}
	blockHeightTopicIds := make([]*types.BlockHeightTopicIds, 0)
	for ; blockActiveTopics.Valid(); blockActiveTopics.Next() {
		keyValue, err := blockActiveTopics.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: blockActiveTopics")
		}
		value := keyValue.Value
		topic := types.BlockHeightTopicIds{
			BlockHeight: keyValue.Key,
			TopicIds:    &value,
		}
		blockHeightTopicIds = append(blockHeightTopicIds, &topic)
	}

	lowestActiveTopic, err := k.topicKeeper.blockToLowestActiveTopicWeight.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate blockActiveTopics")
	}
	blockHeightTopicIdWeight := make([]*types.BlockHeightTopicIdWeightPair, 0)
	for ; lowestActiveTopic.Valid(); lowestActiveTopic.Next() {
		keyValue, err := lowestActiveTopic.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: blockActiveTopics")
		}
		value := keyValue.Value
		topic := types.BlockHeightTopicIdWeightPair{
			BlockHeight: keyValue.Key,
			TopicWeight: &value,
		}
		blockHeightTopicIdWeight = append(blockHeightTopicIdWeight, &topic)
	}

	rewardableTopics := make([]uint64, 0)
	rewardableTopicsIter, err := k.topicKeeper.rewardableTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate rewardable topics")
	}
	for ; rewardableTopicsIter.Valid(); rewardableTopicsIter.Next() {
		key, err := rewardableTopicsIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: rewardableTopicsIter")
		}
		rewardableTopics = append(rewardableTopics, key)
	}

	topicWorkers := make([]*types.TopicAndActorId, 0)
	topicWorkersIter, err := k.workerKeeper.topicWorkers.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic workers")
	}
	for ; topicWorkersIter.Valid(); topicWorkersIter.Next() {
		key, err := topicWorkersIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: topicWorkersIter")
		}
		topicIdAndActorId := types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		}
		topicWorkers = append(topicWorkers, &topicIdAndActorId)
	}

	topicReputers := make([]*types.TopicAndActorId, 0)
	topicReputersIter, err := k.reputerLossKeeper.topicReputers.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic reputers")
	}
	for ; topicReputersIter.Valid(); topicReputersIter.Next() {
		key, err := topicReputersIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: topicReputersIter")
		}
		topicIdAndActorId := types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		}
		topicReputers = append(topicReputers, &topicIdAndActorId)
	}

	topicRewardNonce := make([]*types.TopicIdAndBlockHeight, 0)
	topicRewardNonceIter, err := k.topicKeeper.topicRewardNonce.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic reward nonce")
	}
	for ; topicRewardNonceIter.Valid(); topicRewardNonceIter.Next() {
		keyValue, err := topicRewardNonceIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicRewardNonceIter")
		}
		topicIdAndBlockHeight := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: keyValue.Value,
		}
		topicRewardNonce = append(topicRewardNonce, &topicIdAndBlockHeight)
	}

	var initialInfererEmaScore []*types.TopicIdAndDec
	if err := k.scoresKeeper.initialInfererEmaScore.Walk(
		ctx,
		nil,
		func(key TopicId, value alloraMath.Dec) (stop bool, err error) {
			initialInfererEmaScore = append(initialInfererEmaScore, &types.TopicIdAndDec{
				TopicId: key,
				Dec:     value,
			})
			return false, nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "failed to walk inferer initial EMA score per topic")
	}

	var initialForecasterEmaScore []*types.TopicIdAndDec
	if err := k.scoresKeeper.initialForecasterEmaScore.Walk(
		ctx,
		nil,
		func(key TopicId, value alloraMath.Dec) (stop bool, err error) {
			initialForecasterEmaScore = append(initialForecasterEmaScore, &types.TopicIdAndDec{
				TopicId: key,
				Dec:     value,
			})
			return false, nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "failed to walk forecaster initial EMA score per topic")
	}

	var initialReputerEmaScore []*types.TopicIdAndDec
	if err := k.scoresKeeper.initialReputerEmaScore.Walk(
		ctx,
		nil,
		func(key TopicId, value alloraMath.Dec) (stop bool, err error) {
			initialReputerEmaScore = append(initialReputerEmaScore, &types.TopicIdAndDec{
				TopicId: key,
				Dec:     value,
			})
			return false, nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "failed to walk reputer initial EMA score per topic")
	}

	infererScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	infererScoresByBlockIter, err := k.scoresKeeper.infererScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate inferer scores by block")
	}
	for ; infererScoresByBlockIter.Valid(); infererScoresByBlockIter.Next() {
		keyValue, err := infererScoresByBlockIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: infererScoresByBlockIter")
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		infererScoresByBlock = append(infererScoresByBlock, &topicIdBlockHeightScores)
	}

	forecasterScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	forecasterScoresByBlockIter, err := k.scoresKeeper.forecasterScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate forecaster scores by block")
	}
	for ; forecasterScoresByBlockIter.Valid(); forecasterScoresByBlockIter.Next() {
		keyValue, err := forecasterScoresByBlockIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: forecasterScoresByBlockIter")
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		forecasterScoresByBlock = append(forecasterScoresByBlock, &topicIdBlockHeightScores)
	}

	reputerScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	reputerScoresByBlockIter, err := k.scoresKeeper.reputerScoresByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate reputer scores by block")
	}
	for ; reputerScoresByBlockIter.Valid(); reputerScoresByBlockIter.Next() {
		keyValue, err := reputerScoresByBlockIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: reputerScoresByBlockIter")
		}
		value := keyValue.Value
		topicIdBlockHeightScores := types.TopicIdBlockHeightScores{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Scores:      &value,
		}
		reputerScoresByBlock = append(reputerScoresByBlock, &topicIdBlockHeightScores)
	}

	innfererScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	infererScoreEmasIter, err := k.scoresKeeper.infererScoreEmas.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest inferer scores by worker")
	}
	for ; infererScoreEmasIter.Valid(); infererScoreEmasIter.Next() {
		keyValue, err := infererScoreEmasIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestInfererScoresByWorkerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		innfererScoreEmas = append(innfererScoreEmas, &topicIdActorIdScore)
	}

	forecasterScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	forecasterScoreEmaIter, err := k.scoresKeeper.forecasterScoreEmas.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest forecaster scores by worker")
	}
	for ; forecasterScoreEmaIter.Valid(); forecasterScoreEmaIter.Next() {
		keyValue, err := forecasterScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestForecasterScoresByWorkerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		forecasterScoreEmas = append(forecasterScoreEmas, &topicIdActorIdScore)
	}

	reputerScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	reputerScoreEmasIter, err := k.scoresKeeper.reputerScoreEmas.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest reputer scores by reputer")
	}
	for ; reputerScoreEmasIter.Valid(); reputerScoreEmasIter.Next() {
		keyValue, err := reputerScoreEmasIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestReputerScoresByReputerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		reputerScoreEmas = append(reputerScoreEmas, &topicIdActorIdScore)
	}

	reputerListeningCoefficient := make([]*types.TopicIdActorIdListeningCoefficient, 0)
	reputerListeningCoefficientIter, err := k.scoresKeeper.reputerListeningCoefficient.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate reputer listening coefficient")
	}
	for ; reputerListeningCoefficientIter.Valid(); reputerListeningCoefficientIter.Next() {
		keyValue, err := reputerListeningCoefficientIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: reputerListeningCoefficientIter")
		}
		value := keyValue.Value
		topicIdActorIdListeningCoefficient := types.TopicIdActorIdListeningCoefficient{
			TopicId:              keyValue.Key.K1(),
			ActorId:              keyValue.Key.K2(),
			ListeningCoefficient: &value,
		}
		reputerListeningCoefficient = append(reputerListeningCoefficient, &topicIdActorIdListeningCoefficient)
	}

	previousReputerRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousReputerRewardFractionIter, err := k.scoresKeeper.previousReputerRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous reputer reward fraction")
	}
	for ; previousReputerRewardFractionIter.Valid(); previousReputerRewardFractionIter.Next() {
		keyValue, err := previousReputerRewardFractionIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousReputerRewardFractionIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousReputerRewardFraction = append(previousReputerRewardFraction, &topicIdActorIdDec)
	}

	previousInferenceRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousInferenceRewardFractionIter, err := k.scoresKeeper.previousInferenceRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous inference reward fraction")
	}
	for ; previousInferenceRewardFractionIter.Valid(); previousInferenceRewardFractionIter.Next() {
		keyValue, err := previousInferenceRewardFractionIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousInferenceRewardFractionIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousInferenceRewardFraction = append(previousInferenceRewardFraction, &topicIdActorIdDec)
	}

	previousForecastRewardFraction := make([]*types.TopicIdActorIdDec, 0)
	previousForecastRewardFractionIter, err := k.scoresKeeper.previousForecastRewardFraction.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous forecast reward fraction")
	}
	for ; previousForecastRewardFractionIter.Valid(); previousForecastRewardFractionIter.Next() {
		keyValue, err := previousForecastRewardFractionIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousForecastRewardFractionIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		previousForecastRewardFraction = append(previousForecastRewardFraction, &topicIdActorIdDec)
	}

	/* bug in genesis export, previousForecasterScoreRatio is not correct type in genesis.proto
	previousForecasterScoreRatio := make([]*types.TopicIdAndDec, 0)
	previousForecasterScoreRatioIter, err := k.previousForecasterScoreRatio.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous forecaster score ratio")
	}
	for ; previousForecasterScoreRatioIter.Valid(); previousForecasterScoreRatioIter.Next() {
		keyValue, err := previousForecasterScoreRatioIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousForecasterScoreRatioIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousForecasterScoreRatio = append(previousForecasterScoreRatio, &topicIdAndDec)
	}
	*/

	totalStake, err := k.stakingKeeper.totalStake.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total stake")
	}

	// Fill in the values from keeper.go

	// topicStake
	topicStake := make([]*types.TopicIdAndInt, 0)
	var i uint64
	for i = 1; i < nextTopicId; i++ {
		stake, err := k.stakingKeeper.topicStake.Get(ctx, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get topic stake %d", i)
		}
		topicStake = append(topicStake, &types.TopicIdAndInt{
			TopicId: i,
			Int:     stake,
		})
	}

	// stakeReputerAuthority
	stakeReputerAuthority := make([]*types.TopicIdActorIdInt, 0)
	stakeReputerAuthorityIter, err := k.stakingKeeper.stakeReputerAuthority.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate stake reputer authority")
	}
	for ; stakeReputerAuthorityIter.Valid(); stakeReputerAuthorityIter.Next() {
		keyValue, err := stakeReputerAuthorityIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: stakeReputerAuthorityIter")
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeReputerAuthority = append(stakeReputerAuthority, &topicIdActorIdInt)
	}

	// stakeSumFromDelegator
	stakeSumFromDelegator := make([]*types.TopicIdActorIdInt, 0)
	stakeSumFromDelegatorIter, err := k.stakingKeeper.stakeSumFromDelegator.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate stake sum from delegator")
	}
	for ; stakeSumFromDelegatorIter.Valid(); stakeSumFromDelegatorIter.Next() {
		keyValue, err := stakeSumFromDelegatorIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: stakeSumFromDelegatorIter")
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeSumFromDelegator = append(stakeSumFromDelegator, &topicIdActorIdInt)
	}

	// delegatedStakes
	delegatedStakes := make([]*types.TopicIdDelegatorReputerDelegatorInfo, 0)
	delegatedStakesIter, err := k.stakingKeeper.delegatedStakes.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate delegated stakes")
	}
	for ; delegatedStakesIter.Valid(); delegatedStakesIter.Next() {
		keyValue, err := delegatedStakesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: delegatedStakesIter")
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

	// stakeFromDelegatorsUponReputer
	stakeFromDelegatorsUponReputer := make([]*types.TopicIdActorIdInt, 0)
	stakeFromDelegatorsUponReputerIter, err := k.stakingKeeper.stakeFromDelegatorsUponReputer.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate stake from delegators upon reputer")
	}
	for ; stakeFromDelegatorsUponReputerIter.Valid(); stakeFromDelegatorsUponReputerIter.Next() {
		keyValue, err := stakeFromDelegatorsUponReputerIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: stakeFromDelegatorsUponReputerIter")
		}
		topicIdActorIdInt := types.TopicIdActorIdInt{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Int:     keyValue.Value,
		}
		stakeFromDelegatorsUponReputer = append(stakeFromDelegatorsUponReputer, &topicIdActorIdInt)
	}

	// delegateRewardPerShare
	delegateRewardPerShare := make([]*types.TopicIdActorIdDec, 0)
	delegateRewardPerShareIter, err := k.stakingKeeper.delegateRewardPerShare.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate delegate reward per share")
	}
	for ; delegateRewardPerShareIter.Valid(); delegateRewardPerShareIter.Next() {
		keyValue, err := delegateRewardPerShareIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: delegateRewardPerShareIter")
		}
		topicIdActorIdDec := types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		}
		delegateRewardPerShare = append(delegateRewardPerShare, &topicIdActorIdDec)
	}

	// stakeRemovalsByBlock
	stakeRemovalsByBlock := make([]*types.BlockHeightTopicIdReputerStakeRemovalInfo, 0)
	stakeRemovalsByBlockIter, err := k.stakingKeeper.stakeRemovalsByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate stake removals by block")
	}
	for ; stakeRemovalsByBlockIter.Valid(); stakeRemovalsByBlockIter.Next() {
		keyValue, err := stakeRemovalsByBlockIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: stakeRemovalsByBlockIter")
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

	// stakeRemovalsByActor
	stakeRemovalsByActor := make([]*types.ActorIdTopicIdBlockHeight, 0)
	stakeRemovalsByActorIter, err := k.stakingKeeper.stakeRemovalsByActor.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate stake removals by actor")
	}
	for ; stakeRemovalsByActorIter.Valid(); stakeRemovalsByActorIter.Next() {
		key, err := stakeRemovalsByActorIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: stakeRemovalsByActorIter")
		}
		actorIdTopicIdBlockHeight := types.ActorIdTopicIdBlockHeight{
			ActorId:     key.K1(),
			TopicId:     key.K2(),
			BlockHeight: key.K3(),
		}
		stakeRemovalsByActor = append(stakeRemovalsByActor, &actorIdTopicIdBlockHeight)
	}

	// delegateStakeRemovalsByBlock
	delegateStakeRemovalsByBlock := make([]*types.BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo, 0)
	delegateStakeRemovalsByBlockIter, err := k.stakingKeeper.delegateStakeRemovalsByBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate delegate stake removals by block")
	}
	for ; delegateStakeRemovalsByBlockIter.Valid(); delegateStakeRemovalsByBlockIter.Next() {
		keyValue, err := delegateStakeRemovalsByBlockIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: delegateStakeRemovalsByBlockIter")
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

	// delegateStakeRemovalsByActor
	delegateStakeRemovalsByActor := make([]*types.DelegatorReputerTopicIdBlockHeight, 0)
	delegateStakeRemovalsByActorIter, err := k.stakingKeeper.delegateStakeRemovalsByActor.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate delegate stake removals by actor")
	}
	for ; delegateStakeRemovalsByActorIter.Valid(); delegateStakeRemovalsByActorIter.Next() {
		key, err := delegateStakeRemovalsByActorIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: delegateStakeRemovalsByActorIter")
		}
		delegatorReputerTopicIdBlockHeight := types.DelegatorReputerTopicIdBlockHeight{
			Delegator:   key.K1(),
			Reputer:     key.K2(),
			TopicId:     key.K3(),
			BlockHeight: key.K4(),
		}
		delegateStakeRemovalsByActor = append(delegateStakeRemovalsByActor, &delegatorReputerTopicIdBlockHeight)
	}

	// inferences
	inferences := make([]*types.TopicIdActorIdInference, 0)
	inferencesIter, err := k.workerKeeper.inferences.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate inferences")
	}
	for ; inferencesIter.Valid(); inferencesIter.Next() {
		keyValue, err := inferencesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: inferencesIter")
		}
		value := keyValue.Value
		topicIdActorIdInference := types.TopicIdActorIdInference{
			TopicId:   keyValue.Key.K1(),
			ActorId:   keyValue.Key.K2(),
			Inference: &value,
		}
		inferences = append(inferences, &topicIdActorIdInference)
	}

	// forecasts
	forecasts := make([]*types.TopicIdActorIdForecast, 0)
	forecastsIter, err := k.workerKeeper.forecasts.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate forecasts")
	}
	for ; forecastsIter.Valid(); forecastsIter.Next() {
		keyValue, err := forecastsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: forecastsIter")
		}
		value := keyValue.Value
		topicIdActorIdForecast := types.TopicIdActorIdForecast{
			TopicId:  keyValue.Key.K1(),
			ActorId:  keyValue.Key.K2(),
			Forecast: &value,
		}
		forecasts = append(forecasts, &topicIdActorIdForecast)
	}

	// workers
	workers := make([]*types.LibP2PKeyAndOffchainNode, 0)
	workersIter, err := k.workerKeeper.workers.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate workers")
	}
	for ; workersIter.Valid(); workersIter.Next() {
		keyValue, err := workersIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: workersIter")
		}
		value := keyValue.Value
		libP2PKeyAndOffchainNode := types.LibP2PKeyAndOffchainNode{
			LibP2PKey:    keyValue.Key,
			OffchainNode: &value,
		}
		workers = append(workers, &libP2PKeyAndOffchainNode)
	}

	// reputers
	reputers := make([]*types.LibP2PKeyAndOffchainNode, 0)
	reputersIter, err := k.reputerLossKeeper.reputers.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate reputers")
	}
	for ; reputersIter.Valid(); reputersIter.Next() {
		keyValue, err := reputersIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: reputersIter")
		}
		libP2PKeyAndOffchainNode := types.LibP2PKeyAndOffchainNode{
			LibP2PKey:    keyValue.Key,
			OffchainNode: &keyValue.Value,
		}
		reputers = append(reputers, &libP2PKeyAndOffchainNode)
	}

	// topicFeeRevenue
	topicFeeRevenue := make([]*types.TopicIdAndInt, 0)
	topicFeeRevenueIter, err := k.topicKeeper.topicFeeRevenue.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic fee revenue")
	}
	for ; topicFeeRevenueIter.Valid(); topicFeeRevenueIter.Next() {
		keyValue, err := topicFeeRevenueIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicFeeRevenueIter")
		}
		topicIdAndInt := types.TopicIdAndInt{
			TopicId: keyValue.Key,
			Int:     keyValue.Value,
		}
		topicFeeRevenue = append(topicFeeRevenue, &topicIdAndInt)
	}

	// previousTopicWeight
	previousTopicWeight := make([]*types.TopicIdAndDec, 0)
	previousTopicWeightIter, err := k.topicKeeper.previousTopicWeight.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous topic weight")
	}
	for ; previousTopicWeightIter.Valid(); previousTopicWeightIter.Next() {
		keyValue, err := previousTopicWeightIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousTopicWeightIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicWeight = append(previousTopicWeight, &topicIdAndDec)
	}

	// allInferences
	allInferences := make([]*types.TopicIdBlockHeightInferences, 0)
	allInferencesIter, err := k.workerKeeper.allInferences.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate all inferences")
	}
	for ; allInferencesIter.Valid(); allInferencesIter.Next() {
		keyValue, err := allInferencesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: allInferencesIter")
		}
		value := keyValue.Value
		topicIdBlockHeightInferences := types.TopicIdBlockHeightInferences{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Inferences:  &value,
		}
		allInferences = append(allInferences, &topicIdBlockHeightInferences)
	}

	// allForecasts
	allForecasts := make([]*types.TopicIdBlockHeightForecasts, 0)
	allForecastsIter, err := k.workerKeeper.allForecasts.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate all forecasts")
	}
	for ; allForecastsIter.Valid(); allForecastsIter.Next() {
		keyValue, err := allForecastsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: allForecastsIter")
		}
		value := keyValue.Value
		topicIdBlockHeightForecasts := types.TopicIdBlockHeightForecasts{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			Forecasts:   &value,
		}
		allForecasts = append(allForecasts, &topicIdBlockHeightForecasts)
	}

	// allLossBundles
	allLossBundles := make([]*types.TopicIdBlockHeightReputerValueBundles, 0)
	allLossBundlesIter, err := k.reputerLossKeeper.allLossBundles.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate all loss bundles")
	}
	for ; allLossBundlesIter.Valid(); allLossBundlesIter.Next() {
		keyValue, err := allLossBundlesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: allLossBundlesIter")
		}
		value := keyValue.Value
		reputerValueBundles := make(types.LossBundles, len(value.ReputerValueBundles))
		for i := range value.ReputerValueBundles {
			reputerValueBundles[i] = value.ReputerValueBundles[i].GetValueBundle()
		}
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightReputerValueBundles{
			TopicId:             keyValue.Key.K1(),
			BlockHeight:         keyValue.Key.K2(),
			ReputerValueBundles: reputerValueBundles,
		}
		allLossBundles = append(allLossBundles, &topicIdBlockHeightValueBundles)
	}

	// networkLossBundles
	networkLossBundles := make([]*types.TopicIdBlockHeightValueBundles, 0)
	networkLossBundlesIter, err := k.reputerLossKeeper.networkLossBundles.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate network loss bundles")
	}
	for ; networkLossBundlesIter.Valid(); networkLossBundlesIter.Next() {
		keyValue, err := networkLossBundlesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: networkLossBundlesIter")
		}
		value := keyValue.Value
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightValueBundles{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			ValueBundle: &value,
		}
		networkLossBundles = append(networkLossBundles, &topicIdBlockHeightValueBundles)
	}

	// previousPercentageRewardToStakedReputers
	previousPercentageRewardToStakedReputers, err := k.scoresKeeper.previousPercentageRewardToStakedReputers.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous percentage reward to staked reputers")
	}

	// openWorkerWindows
	openWorkerWindows := make([]*types.BlockHeightAndTopicIds, 0)
	openWorkerWindowsIter, err := k.nonceKeeper.openWorkerWindows.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate open worker windows")
	}
	for ; openWorkerWindowsIter.Valid(); openWorkerWindowsIter.Next() {
		keyValue, err := openWorkerWindowsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: openWorkerWindowsIter")
		}
		blockHeight := keyValue.Key
		topicIds := keyValue.Value.TopicIds
		openWorkerWindows = append(openWorkerWindows, &types.BlockHeightAndTopicIds{
			BlockHeight: blockHeight,
			TopicIds:    topicIds,
		})
	}

	// unfulfilledWorkerNonces
	unfulfilledWorkerNonces := make([]*types.TopicIdAndNonces, 0)
	unfulfilledWorkerNoncesIter, err := k.nonceKeeper.unfulfilledWorkerNonces.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate unfulfilled worker nonces")
	}
	for ; unfulfilledWorkerNoncesIter.Valid(); unfulfilledWorkerNoncesIter.Next() {
		keyValue, err := unfulfilledWorkerNoncesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: unfulfilledWorkerNoncesIter")
		}
		topicIdAndNonces := types.TopicIdAndNonces{
			TopicId: keyValue.Key,
			Nonces:  &keyValue.Value,
		}
		unfulfilledWorkerNonces = append(unfulfilledWorkerNonces, &topicIdAndNonces)
	}

	// unfulfilledReputerNonces
	unfulfilledReputerNonces := make([]*types.TopicIdAndReputerRequestNonces, 0)
	unfulfilledReputerNoncesIter, err := k.nonceKeeper.unfulfilledReputerNonces.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate unfulfilled reputer nonces")
	}
	for ; unfulfilledReputerNoncesIter.Valid(); unfulfilledReputerNoncesIter.Next() {
		keyValue, err := unfulfilledReputerNoncesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: unfulfilledReputerNoncesIter")
		}
		value := keyValue.Value
		topicIdAndReputerRequestNonces := types.TopicIdAndReputerRequestNonces{
			TopicId:              keyValue.Key,
			ReputerRequestNonces: &value,
		}
		unfulfilledReputerNonces = append(unfulfilledReputerNonces, &topicIdAndReputerRequestNonces)
	}

	// lastDripBlock
	lastDripBlock := make([]*types.TopicIdAndBlockHeight, 0)
	lastDripBlockIter, err := k.topicKeeper.lastDripBlock.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate last drip block")
	}
	for ; lastDripBlockIter.Valid(); lastDripBlockIter.Next() {
		keyValue, err := lastDripBlockIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: lastDripBlockIter")
		}
		topicIdAndBlockHeight := types.TopicIdAndBlockHeight{
			TopicId:     keyValue.Key,
			BlockHeight: keyValue.Value,
		}
		lastDripBlock = append(lastDripBlock, &topicIdAndBlockHeight)
	}

	latestInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestInfererNetworkRegretsIter, err := k.regretsKeeper.latestInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest inferer network regrets")
	}
	for ; latestInfererNetworkRegretsIter.Valid(); latestInfererNetworkRegretsIter.Next() {
		keyValue, err := latestInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestInfererNetworkRegrets = append(latestInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestNaiveInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestNaiveInfererNetworkRegretsIter, err := k.regretsKeeper.latestNaiveInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest naive inferer network regrets")
	}
	for ; latestNaiveInfererNetworkRegretsIter.Valid(); latestNaiveInfererNetworkRegretsIter.Next() {
		keyValue, err := latestNaiveInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestNaiveInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestNaiveInfererNetworkRegrets = append(latestNaiveInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestForecasterNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestForecasterNetworkRegretsIter, err := k.regretsKeeper.latestForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest forecaster network regrets")
	}
	for ; latestForecasterNetworkRegretsIter.Valid(); latestForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestForecasterNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestForecasterNetworkRegrets = append(latestForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestOneOutInfererInfererNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutInfererInfererNetworkRegretsIter, err := k.regretsKeeper.latestOneOutInfererInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest one out inferer inferer network regrets")
	}
	for ; latestOneOutInfererInfererNetworkRegretsIter.Valid(); latestOneOutInfererInfererNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutInfererInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestOneOutInfererInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutInfererInfererNetworkRegrets = append(latestOneOutInfererInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestOneOutInfererForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutInfererForecasterNetworkRegretsIter, err := k.regretsKeeper.latestOneOutInfererForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest one out inferer forecaster network regrets")
	}
	for ; latestOneOutInfererForecasterNetworkRegretsIter.Valid(); latestOneOutInfererForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutInfererForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestOneOutInfererForecasterNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutInfererForecasterNetworkRegrets = append(latestOneOutInfererForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestOneOutForecasterInfererNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutForecasterInfererNetworkRegretsIter, err := k.regretsKeeper.latestOneOutForecasterInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest one out forecaster inferer network regrets")
	}
	for ; latestOneOutForecasterInfererNetworkRegretsIter.Valid(); latestOneOutForecasterInfererNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutForecasterInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestOneOutForecasterInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutForecasterInfererNetworkRegrets = append(latestOneOutForecasterInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestOneOutForecasterForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutForecasterForecasterNetworkRegretsIter, err := k.regretsKeeper.latestOneOutForecasterForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest one out forecaster forecaster network regrets")
	}
	for ; latestOneOutForecasterForecasterNetworkRegretsIter.Valid(); latestOneOutForecasterForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutForecasterForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestOneOutForecasterForecasterNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutForecasterForecasterNetworkRegrets = append(latestOneOutForecasterForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}

	latestOneInForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneInForecasterNetworkRegretsIter, err := k.regretsKeeper.latestOneInForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest one in forecaster network regrets")
	}
	for ; latestOneInForecasterNetworkRegretsIter.Valid(); latestOneInForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneInForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestOneInForecasterNetworkRegretsIter")
		}
		topicIdActorIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneInForecasterNetworkRegrets = append(latestOneInForecasterNetworkRegrets, &topicIdActorIdActorIdTimeStampedValue)
	}

	previousForecasterScoreRatio := make([]*types.TopicIdAndDec, 0)
	previousForecasterScoreRatioIter, err := k.scoresKeeper.previousForecasterScoreRatio.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous forecaster score ratio")
	}
	for ; previousForecasterScoreRatioIter.Valid(); previousForecasterScoreRatioIter.Next() {
		keyValue, err := previousForecasterScoreRatioIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousForecasterScoreRatioIter")
		}
		previousForecasterScoreRatio = append(previousForecasterScoreRatio, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
	}

	coreTeamAddresses := make([]string, 0)
	coreTeamAddressesIter, err := k.whitelistsKeeper.whitelistAdmins.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate core team addresses")
	}
	for ; coreTeamAddressesIter.Valid(); coreTeamAddressesIter.Next() {
		key, err := coreTeamAddressesIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: coreTeamAddressesIter")
		}
		coreTeamAddresses = append(coreTeamAddresses, key)
	}

	topicLastWorkerCommit := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastWorkerCommitIter, err := k.topicKeeper.topicLastWorkerCommit.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic last worker commit")
	}
	for ; topicLastWorkerCommitIter.Valid(); topicLastWorkerCommitIter.Next() {
		keyValue, err := topicLastWorkerCommitIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicLastWorkerCommitIter")
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastWorkerCommit = append(topicLastWorkerCommit, &topicIdTimestampedActorNonce)
	}

	topicLastReputerCommit := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastReputerCommitIter, err := k.topicKeeper.topicLastReputerCommit.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic last reputer commit")
	}
	for ; topicLastReputerCommitIter.Valid(); topicLastReputerCommitIter.Next() {
		keyValue, err := topicLastReputerCommitIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicLastReputerCommitIter")
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastReputerCommit = append(topicLastReputerCommit, &topicIdTimestampedActorNonce)
	}

	previousTopicQuantileInfererScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileInfererScoreEmaIter, err := k.scoresKeeper.previousTopicQuantileInfererScoreEma.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous topic quantile inferer score ema")
	}
	for ; previousTopicQuantileInfererScoreEmaIter.Valid(); previousTopicQuantileInfererScoreEmaIter.Next() {
		keyValue, err := previousTopicQuantileInfererScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousTopicQuantileInfererScoreEmaIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicQuantileInfererScoreEma = append(previousTopicQuantileInfererScoreEma, &topicIdAndDec)
	}

	previousTopicQuantileForecasterScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileForecasterScoreEmaIter, err := k.scoresKeeper.previousTopicQuantileForecasterScoreEma.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous topic quantile forecaster score ema")
	}
	for ; previousTopicQuantileForecasterScoreEmaIter.Valid(); previousTopicQuantileForecasterScoreEmaIter.Next() {
		keyValue, err := previousTopicQuantileForecasterScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousTopicQuantileForecasterScoreEmaIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicQuantileForecasterScoreEma = append(previousTopicQuantileForecasterScoreEma, &topicIdAndDec)
	}

	previousTopicQuantileReputerScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileReputerScoreEmaIter, err := k.scoresKeeper.previousTopicQuantileReputerScoreEma.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate previous topic quantile reputer score ema")
	}
	for ; previousTopicQuantileReputerScoreEmaIter.Valid(); previousTopicQuantileReputerScoreEmaIter.Next() {
		keyValue, err := previousTopicQuantileReputerScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: previousTopicQuantileReputerScoreEmaIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		previousTopicQuantileReputerScoreEma = append(previousTopicQuantileReputerScoreEma, &topicIdAndDec)
	}

	activeInferers := make([]*types.TopicAndActorId, 0)
	activeInferersIter, err := k.workerKeeper.activeInferers.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate active inferers")
	}
	for ; activeInferersIter.Valid(); activeInferersIter.Next() {
		key, err := activeInferersIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: activeInferersIter")
		}
		activeInferers = append(activeInferers, &types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		})
	}

	activeForecasters := make([]*types.TopicAndActorId, 0)
	activeForecasterIter, err := k.workerKeeper.activeForecasters.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate active forecasters")
	}
	for ; activeForecasterIter.Valid(); activeForecasterIter.Next() {
		key, err := activeForecasterIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: activeForecasterIter")
		}
		activeForecasters = append(activeForecasters, &types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		})
	}

	lowestInfererScoreEma := make([]*types.TopicIdActorIdScore, 0)
	lowestInfererScoreEmaIter, err := k.scoresKeeper.lowestInfererScoreEma.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate lowest inferer score emas")
	}
	for ; lowestInfererScoreEmaIter.Valid(); lowestInfererScoreEmaIter.Next() {
		keyValue, err := lowestInfererScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: lowestInfererScoreEmaIter")
		}
		lowestInfererScoreEma = append(lowestInfererScoreEma, &types.TopicIdActorIdScore{
			TopicId: keyValue.Key,
			ActorId: keyValue.Value.Address,
			Score:   &keyValue.Value,
		})
	}

	lowestForecasterScoreEma := make([]*types.TopicIdActorIdScore, 0)
	lowestForecasterScoreEmaIter, err := k.scoresKeeper.lowestForecasterScoreEma.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate lowest forecaster score emas")
	}
	for ; lowestForecasterScoreEmaIter.Valid(); lowestForecasterScoreEmaIter.Next() {
		keyValue, err := lowestForecasterScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: lowestForecasterScoreEmaIter")
		}
		lowestForecasterScoreEma = append(lowestForecasterScoreEma, &types.TopicIdActorIdScore{
			TopicId: keyValue.Key,
			ActorId: keyValue.Value.Address,
			Score:   &keyValue.Value,
		})
	}

	activeReputers := make([]*types.TopicAndActorId, 0)
	activeReputersIter, err := k.reputerLossKeeper.activeReputers.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate active reputers")
	}
	for ; activeReputersIter.Valid(); activeReputersIter.Next() {
		key, err := activeReputersIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: activeReputersIter")
		}
		activeReputers = append(activeReputers, &types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		})
	}

	lowestReputerScoreEma := make([]*types.TopicIdActorIdScore, 0)
	lowestReputerScoreEmaIter, err := k.scoresKeeper.lowestReputerScoreEma.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate lowest reputer score emas")
	}
	for ; lowestReputerScoreEmaIter.Valid(); lowestReputerScoreEmaIter.Next() {
		keyValue, err := lowestReputerScoreEmaIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: lowestReputerScoreEmaIter")
		}
		lowestReputerScoreEma = append(lowestReputerScoreEma, &types.TopicIdActorIdScore{
			TopicId: keyValue.Key,
			ActorId: keyValue.Value.Address,
			Score:   &keyValue.Value,
		})
	}

	lossBundles := make([]*types.TopicIdReputerReputerValueBundle, 0)
	lossBundlesIter, err := k.reputerLossKeeper.lossBundles.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate loss bundles")
	}
	for ; lossBundlesIter.Valid(); lossBundlesIter.Next() {
		keyValue, err := lossBundlesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key-value: lossBundlesIter")
		}
		lossBundles = append(lossBundles, &types.TopicIdReputerReputerValueBundle{
			TopicId:            keyValue.Key.K1(),
			Reputer:            keyValue.Key.K2(),
			ReputerValueBundle: keyValue.Value.GetValueBundle(),
		})
	}

	countInfererInclusionsInTopicActiveSet := make([]*types.TopicIdActorIdUint64, 0)
	countInfererInclusionsInTopicIter, err := k.countInfererInclusionsInTopicActiveSet.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate count inferer inclusions in topic")
	}
	for ; countInfererInclusionsInTopicIter.Valid(); countInfererInclusionsInTopicIter.Next() {
		keyValue, err := countInfererInclusionsInTopicIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: countInfererInclusionsInTopicIter")
		}
		topicIdAndUint64 := types.TopicIdActorIdUint64{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Uint64:  keyValue.Value,
		}
		countInfererInclusionsInTopicActiveSet = append(countInfererInclusionsInTopicActiveSet, &topicIdAndUint64)
	}

	countForecasterInclusionsInTopicActiveSet := make([]*types.TopicIdActorIdUint64, 0)
	countForecasterInclusionsInTopicIter, err := k.countForecasterInclusionsInTopicActiveSet.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate count forecaster inclusions in topic")
	}
	for ; countForecasterInclusionsInTopicIter.Valid(); countForecasterInclusionsInTopicIter.Next() {
		keyValue, err := countForecasterInclusionsInTopicIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: countForecasterInclusionsInTopicIter")
		}
		topicIdAndUint64 := types.TopicIdActorIdUint64{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Uint64:  keyValue.Value,
		}
		countForecasterInclusionsInTopicActiveSet = append(countForecasterInclusionsInTopicActiveSet, &topicIdAndUint64)
	}

	rewardCurrentBlockEmission, err := k.GetRewardCurrentBlockEmission(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get reward current block emission")
	}

	totalSumPreviousTopicWeights, err := k.topicKeeper.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total sum previous topic weights")
	}

	whitelistAdmins := make([]string, 0)
	whitelistAdminsIter, err := k.whitelistsKeeper.whitelistAdmins.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate whitelist admins")
	}
	for ; whitelistAdminsIter.Valid(); whitelistAdminsIter.Next() {
		key, err := whitelistAdminsIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: whitelistAdminsIter")
		}
		whitelistAdmins = append(whitelistAdmins, key)
	}

	globalWhitelist := make([]string, 0)
	globalWhitelistIter, err := k.whitelistsKeeper.globalWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate global whitelist")
	}
	for ; globalWhitelistIter.Valid(); globalWhitelistIter.Next() {
		key, err := globalWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: globalWhitelistIter")
		}
		globalWhitelist = append(globalWhitelist, key)
	}

	globalWorkerWhitelist := make([]string, 0)
	globalWorkerWhitelistIter, err := k.whitelistsKeeper.globalWorkerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate global worker whitelist")
	}
	for ; globalWorkerWhitelistIter.Valid(); globalWorkerWhitelistIter.Next() {
		key, err := globalWorkerWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: globalWorkerWhitelistIter")
		}
		globalWorkerWhitelist = append(globalWorkerWhitelist, key)
	}

	globalReputerWhitelist := make([]string, 0)
	globalReputerWhitelistIter, err := k.whitelistsKeeper.globalReputerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate global reputer whitelist")
	}
	for ; globalReputerWhitelistIter.Valid(); globalReputerWhitelistIter.Next() {
		key, err := globalReputerWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: globalReputerWhitelistIter")
		}
		globalReputerWhitelist = append(globalReputerWhitelist, key)
	}

	globalAdminWhitelist := make([]string, 0)
	globalAdminWhitelistIter, err := k.whitelistsKeeper.globalAdminWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate global admin whitelist")
	}
	for ; globalAdminWhitelistIter.Valid(); globalAdminWhitelistIter.Next() {
		key, err := globalAdminWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: globalAdminWhitelistIter")
		}
		globalAdminWhitelist = append(globalAdminWhitelist, key)
	}

	topicCreatorWhitelist := make([]string, 0)
	topicCreatorWhitelistIter, err := k.whitelistsKeeper.topicCreatorWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic creator whitelist")
	}
	for ; topicCreatorWhitelistIter.Valid(); topicCreatorWhitelistIter.Next() {
		key, err := topicCreatorWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: topicCreatorWhitelistIter")
		}
		topicCreatorWhitelist = append(topicCreatorWhitelist, key)
	}

	topicWorkerWhitelist := make([]*types.TopicAndActorId, 0)
	topicWorkerWhitelistIter, err := k.whitelistsKeeper.topicWorkerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic worker whitelist")
	}
	for ; topicWorkerWhitelistIter.Valid(); topicWorkerWhitelistIter.Next() {
		keyValue, err := topicWorkerWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicWorkerWhitelistIter")
		}
		topicWorkerWhitelist = append(topicWorkerWhitelist, &types.TopicAndActorId{
			TopicId: keyValue.K1(),
			ActorId: keyValue.K2(),
		})
	}

	topicReputerWhitelist := make([]*types.TopicAndActorId, 0)
	topicReputerWhitelistIter, err := k.whitelistsKeeper.topicReputerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic reputer whitelist")
	}
	for ; topicReputerWhitelistIter.Valid(); topicReputerWhitelistIter.Next() {
		keyValue, err := topicReputerWhitelistIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicReputerWhitelistIter")
		}
		topicReputerWhitelist = append(topicReputerWhitelist, &types.TopicAndActorId{
			TopicId: keyValue.K1(),
			ActorId: keyValue.K2(),
		})
	}

	topicWorkerWhitelistEnabled := make([]uint64, 0)
	topicWorkerWhitelistEnabledIter, err := k.whitelistsKeeper.topicWorkerWhitelistEnabled.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic whitelist enabled")
	}
	for ; topicWorkerWhitelistEnabledIter.Valid(); topicWorkerWhitelistEnabledIter.Next() {
		key, err := topicWorkerWhitelistEnabledIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: topicWhitelistEnabledIter")
		}
		topicWorkerWhitelistEnabled = append(topicWorkerWhitelistEnabled, key)
	}

	topicReputerWhitelistEnabled := make([]uint64, 0)
	topicReputerWhitelistEnabledIter, err := k.whitelistsKeeper.topicReputerWhitelistEnabled.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic reputer whitelist enabled")
	}
	for ; topicReputerWhitelistEnabledIter.Valid(); topicReputerWhitelistEnabledIter.Next() {
		key, err := topicReputerWhitelistEnabledIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: topicReputerWhitelistEnabledIter")
		}
		topicReputerWhitelistEnabled = append(topicReputerWhitelistEnabled, key)
	}

	lastMedianInferences := make([]*types.TopicIdAndDec, 0)
	lastMedianInferencesIter, err := k.topicKeeper.lastMedianInferences.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate last median inferences")
	}
	for ; lastMedianInferencesIter.Valid(); lastMedianInferencesIter.Next() {
		keyValue, err := lastMedianInferencesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: lastMedianInferencesIter")
		}
		topicIdAndDec := types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		}
		lastMedianInferences = append(lastMedianInferences, &topicIdAndDec)
	}

	madInferences := make([]*types.TopicIdAndDec, 0)
	madInferencesIter, err := k.topicKeeper.madInferences.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate last mad inferences")
	}
	for ; madInferencesIter.Valid(); madInferencesIter.Next() {
		keyValue, err := madInferencesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: MadInferencesIter")
		}
		madInferences = append(madInferences, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
	}

	// Export latest regret stdnorm
	latestRegretStdNorm := make([]*types.TopicIdAndDec, 0)
	stdnormIter, err := k.weightsKeeper.latestRegretStdNorm.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest regret stdnorm")
	}
	for ; stdnormIter.Valid(); stdnormIter.Next() {
		keyValue, err := stdnormIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: stdnormIter")
		}
		latestRegretStdNorm = append(latestRegretStdNorm, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
	}

	// Export latest inferer weights
	latestInfererWeights := make([]*types.TopicIdActorIdDec, 0)
	infererWeightsIter, err := k.weightsKeeper.latestInfererWeights.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest inferer weights")
	}
	for ; infererWeightsIter.Valid(); infererWeightsIter.Next() {
		keyValue, err := infererWeightsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: infererWeightsIter")
		}
		latestInfererWeights = append(latestInfererWeights, &types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		})
	}

	// Export latest forecaster weights
	latestForecasterWeights := make([]*types.TopicIdActorIdDec, 0)
	forecasterWeightsIter, err := k.weightsKeeper.latestForecasterWeights.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest forecaster weights")
	}
	for ; forecasterWeightsIter.Valid(); forecasterWeightsIter.Next() {
		keyValue, err := forecasterWeightsIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: forecasterWeightsIter")
		}
		latestForecasterWeights = append(latestForecasterWeights, &types.TopicIdActorIdDec{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Dec:     keyValue.Value,
		})
	}

	// Export network inferences
	networkInferences := make([]*types.TopicIdBlockHeightValueBundles, 0)
	networkInferencesIter, err := k.networkInferences.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate network inferences")
	}
	for ; networkInferencesIter.Valid(); networkInferencesIter.Next() {
		keyValue, err := networkInferencesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: networkInferencesIter")
		}
		networkInferences = append(networkInferences, &types.TopicIdBlockHeightValueBundles{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			ValueBundle: &keyValue.Value,
		})
	}

	// Outlier resistant network inferences
	outlierResistantNetworkInferences := make([]*types.TopicIdBlockHeightValueBundles, 0)
	outlierResistantNetworkInferencesIter, err := k.outlierResistantNetworkInferences.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate outlier resistant network inferences")
	}
	for ; outlierResistantNetworkInferencesIter.Valid(); outlierResistantNetworkInferencesIter.Next() {
		keyValue, err := outlierResistantNetworkInferencesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: outlierResistantNetworkInferencesIter")
		}
		outlierResistantNetworkInferences = append(outlierResistantNetworkInferences, &types.TopicIdBlockHeightValueBundles{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			ValueBundle: &keyValue.Value,
		})
	}

	// Get Monthly Reputer Rewards
	monthlyReputerRewards, err := k.weightsKeeper.GetMonthlyReputerRewards(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get monthly reputer rewards")
	}

	// Get Monthly Topic Rewards
	monthlyTopicRewards, err := k.weightsKeeper.GetMonthlyTopicRewards(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get monthly topic rewards")
	}

	return &types.GenesisState{
		Params:                                         moduleParams,
		NextTopicId:                                    nextTopicId,
		Topics:                                         topics,
		ActiveTopics:                                   activeTopics,
		RewardableTopics:                               rewardableTopics,
		TopicWorkers:                                   topicWorkers,
		TopicReputers:                                  topicReputers,
		TopicRewardNonce:                               topicRewardNonce,
		InitialInfererEmaScore:                         initialInfererEmaScore,
		InitialForecasterEmaScore:                      initialForecasterEmaScore,
		InitialReputerEmaScore:                         initialReputerEmaScore,
		InfererScoresByBlock:                           infererScoresByBlock,
		ForecasterScoresByBlock:                        forecasterScoresByBlock,
		ReputerScoresByBlock:                           reputerScoresByBlock,
		InfererScoreEmas:                               innfererScoreEmas,
		ForecasterScoreEmas:                            forecasterScoreEmas,
		ReputerScoreEmas:                               reputerScoreEmas,
		ReputerListeningCoefficient:                    reputerListeningCoefficient,
		PreviousReputerRewardFraction:                  previousReputerRewardFraction,
		PreviousInferenceRewardFraction:                previousInferenceRewardFraction,
		PreviousForecastRewardFraction:                 previousForecastRewardFraction,
		TotalStake:                                     totalStake,
		TopicStake:                                     topicStake,
		StakeReputerAuthority:                          stakeReputerAuthority,
		StakeSumFromDelegator:                          stakeSumFromDelegator,
		DelegatedStakes:                                delegatedStakes,
		StakeFromDelegatorsUponReputer:                 stakeFromDelegatorsUponReputer,
		DelegateRewardPerShare:                         delegateRewardPerShare,
		StakeRemovalsByBlock:                           stakeRemovalsByBlock,
		StakeRemovalsByActor:                           stakeRemovalsByActor,
		DelegateStakeRemovalsByBlock:                   delegateStakeRemovalsByBlock,
		DelegateStakeRemovalsByActor:                   delegateStakeRemovalsByActor,
		Inferences:                                     inferences,
		Forecasts:                                      forecasts,
		Workers:                                        workers,
		Reputers:                                       reputers,
		TopicFeeRevenue:                                topicFeeRevenue,
		PreviousTopicWeight:                            previousTopicWeight,
		AllInferences:                                  allInferences,
		AllForecasts:                                   allForecasts,
		AllLossBundles:                                 allLossBundles,
		NetworkLossBundles:                             networkLossBundles,
		PreviousPercentageRewardToStakedReputers:       previousPercentageRewardToStakedReputers,
		OpenWorkerWindows:                              openWorkerWindows,
		UnfulfilledWorkerNonces:                        unfulfilledWorkerNonces,
		UnfulfilledReputerNonces:                       unfulfilledReputerNonces,
		LastDripBlock:                                  lastDripBlock,
		LatestInfererNetworkRegrets:                    latestInfererNetworkRegrets,
		LatestForecasterNetworkRegrets:                 latestForecasterNetworkRegrets,
		LatestOneInForecasterNetworkRegrets:            latestOneInForecasterNetworkRegrets,
		PreviousForecasterScoreRatio:                   previousForecasterScoreRatio,
		CoreTeamAddresses:                              coreTeamAddresses,
		TopicLastWorkerCommit:                          topicLastWorkerCommit,
		TopicLastReputerCommit:                         topicLastReputerCommit,
		LatestNaiveInfererNetworkRegrets:               latestNaiveInfererNetworkRegrets,
		LatestOneOutInfererInfererNetworkRegrets:       latestOneOutInfererInfererNetworkRegrets,
		LatestOneOutForecasterInfererNetworkRegrets:    latestOneOutForecasterInfererNetworkRegrets,
		LatestOneOutInfererForecasterNetworkRegrets:    latestOneOutInfererForecasterNetworkRegrets,
		LatestOneOutForecasterForecasterNetworkRegrets: latestOneOutForecasterForecasterNetworkRegrets,
		TopicToNextPossibleChurningBlock:               topicToNextPossibleChurningBlock,
		BlockToActiveTopics:                            blockHeightTopicIds,
		BlockToLowestActiveTopicWeight:                 blockHeightTopicIdWeight,
		PreviousTopicQuantileInfererScoreEma:           previousTopicQuantileInfererScoreEma,
		PreviousTopicQuantileForecasterScoreEma:        previousTopicQuantileForecasterScoreEma,
		PreviousTopicQuantileReputerScoreEma:           previousTopicQuantileReputerScoreEma,
		ActiveInferers:                                 activeInferers,
		ActiveForecasters:                              activeForecasters,
		ActiveReputers:                                 activeReputers,
		LowestInfererScoreEma:                          lowestInfererScoreEma,
		LowestForecasterScoreEma:                       lowestForecasterScoreEma,
		LowestReputerScoreEma:                          lowestReputerScoreEma,
		LossBundles:                                    lossBundles,
		CountInfererInclusionsInTopicActiveSet:         countInfererInclusionsInTopicActiveSet,
		CountForecasterInclusionsInTopicActiveSet:      countForecasterInclusionsInTopicActiveSet,
		TotalSumPreviousTopicWeights:                   totalSumPreviousTopicWeights,
		RewardCurrentBlockEmission:                     rewardCurrentBlockEmission,
		WhitelistAdmins:                                whitelistAdmins,
		GlobalWhitelist:                                globalWhitelist,
		GlobalWorkerWhitelist:                          globalWorkerWhitelist,
		GlobalReputerWhitelist:                         globalReputerWhitelist,
		GlobalAdminWhitelist:                           globalAdminWhitelist,
		TopicCreatorWhitelist:                          topicCreatorWhitelist,
		TopicWorkerWhitelist:                           topicWorkerWhitelist,
		TopicReputerWhitelist:                          topicReputerWhitelist,
		TopicWorkerWhitelistEnabled:                    topicWorkerWhitelistEnabled,
		TopicReputerWhitelistEnabled:                   topicReputerWhitelistEnabled,
		LastMedianInferences:                           lastMedianInferences,
		MadInferences:                                  madInferences,
		LatestRegretStdNorm:                            latestRegretStdNorm,
		LatestInfererWeights:                           latestInfererWeights,
		LatestForecasterWeights:                        latestForecasterWeights,
		NetworkInferences:                              networkInferences,
		OutlierResistantNetworkInferences:              outlierResistantNetworkInferences,
		MonthlyReputerRewards:                          monthlyReputerRewards,
		MonthlyTopicRewards:                            monthlyTopicRewards,
	}, nil
}
