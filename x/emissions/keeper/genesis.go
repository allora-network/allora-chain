package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"cosmossdk.io/collections"
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	if err := k.SetParams(ctx, data.Params); err != nil {
		return errors.Wrap(err, "error setting params")
	}
	// nextTopicId uint64
	if data.NextTopicId == 0 {
		// reserve topic ID 0 for future use
		if _, err := k.IncrementTopicId(ctx); err != nil {
			return errors.Wrap(err, "error incrementing topic ID")
		}
	} else {
		if err := k.nextTopicId.Set(ctx, data.NextTopicId); err != nil {
			return errors.Wrap(err, "error setting next topic ID")
		}
	}
	//Topics       []*TopicIdAndTopic
	if len(data.Topics) != 0 {
		for _, topic := range data.Topics {
			if topic != nil {
				if err := k.topics.Set(ctx, topic.TopicId, *topic.Topic); err != nil {
					return errors.Wrap(err, "error setting topic")
				}
			}
		}
	}
	//ActiveTopics []uint64
	if len(data.ActiveTopics) != 0 {
		for _, topicId := range data.ActiveTopics {
			if err := k.activeTopics.Set(ctx, topicId); err != nil {
				return errors.Wrap(err, "error setting activeTopics")
			}
		}
	}
	//ChurnableTopics []uint64
	if len(data.ChurnableTopics) != 0 {
		for _, topicId := range data.ChurnableTopics {
			if err := k.churnableTopics.Set(ctx, topicId); err != nil {
				return errors.Wrap(err, "error setting churnableTopics")
			}
		}
	}
	//RewardableTopics []uint64
	if len(data.RewardableTopics) != 0 {
		for _, topicId := range data.RewardableTopics {
			if err := k.rewardableTopics.Set(ctx, topicId); err != nil {
				return errors.Wrap(err, "error setting rewardableTopics")
			}
		}
	}
	//TopicWorkers []*TopicAndActorId
	if len(data.TopicWorkers) != 0 {
		for _, topicAndActorId := range data.TopicWorkers {
			if topicAndActorId != nil {
				if err := k.topicWorkers.Set(ctx, collections.Join(topicAndActorId.TopicId, topicAndActorId.ActorId)); err != nil {
					return errors.Wrap(err, "error setting topicWorkers")
				}
			}
		}

	}
	//TopicReputers []*TopicAndActorId
	if len(data.TopicReputers) != 0 {
		for _, topicAndActorId := range data.TopicReputers {
			if topicAndActorId != nil {
				if err := k.topicReputers.Set(ctx, collections.Join(topicAndActorId.TopicId, topicAndActorId.ActorId)); err != nil {
					return errors.Wrap(err, "error setting topicReputers")
				}
			}
		}

	}
	//TopicRewardNonce []*TopicIdAndBlockHeight
	if len(data.TopicRewardNonce) != 0 {
		for _, topicIdAndBlockHeight := range data.TopicRewardNonce {
			if topicIdAndBlockHeight != nil {
				if err := k.topicRewardNonce.Set(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
					return errors.Wrap(err, "error setting topicRewardNonce")
				}
			}
		}
	}
	//InfererScoresByBlock []*TopicIdBlockHeightScores
	if len(data.InfererScoresByBlock) != 0 {
		for _, topicIdBlockHeightScores := range data.InfererScoresByBlock {
			if topicIdBlockHeightScores != nil {
				if err := k.infererScoresByBlock.Set(ctx,
					collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
					*topicIdBlockHeightScores.Scores); err != nil {
					return errors.Wrap(err, "error setting infererScoresByBlock")
				}
			}
		}

	}
	//ForecasterScoresByBlock []*TopicIdBlockHeightScores
	if len(data.ForecasterScoresByBlock) != 0 {
		for _, topicIdBlockHeightScores := range data.ForecasterScoresByBlock {
			if topicIdBlockHeightScores != nil {
				if err := k.forecasterScoresByBlock.Set(
					ctx,
					collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
					*topicIdBlockHeightScores.Scores); err != nil {
					return errors.Wrap(err, "error setting forecasterScoresByBlock")
				}
			}
		}

	}
	//ReputerScoresByBlock []*TopicIdBlockHeightScores
	if len(data.ReputerScoresByBlock) != 0 {
		for _, topicIdBlockHeightScores := range data.ReputerScoresByBlock {
			if topicIdBlockHeightScores != nil {
				if err := k.reputerScoresByBlock.Set(
					ctx,
					collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
					*topicIdBlockHeightScores.Scores); err != nil {
					return errors.Wrap(err, "error setting reputerScoresByBlock")
				}
			}
		}
	}
	//LatestInfererScoresByWorker []*TopicIdActorIdScore
	if len(data.LatestInfererScoresByWorker) != 0 {
		for _, topicIdActorIdScore := range data.LatestInfererScoresByWorker {
			if topicIdActorIdScore != nil {
				if err := k.latestInfererScoresByWorker.Set(ctx,
					collections.Join(topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId),
					*topicIdActorIdScore.Score); err != nil {
					return errors.Wrap(err, "error setting latestInfererScoresByWorker")
				}
			}
		}
	}
	//LatestForecasterScoresByWorker []*TopicIdActorIdScore
	if len(data.LatestForecasterScoresByWorker) != 0 {
		for _, topicIdActorIdScore := range data.LatestForecasterScoresByWorker {
			if topicIdActorIdScore != nil {
				if err := k.latestForecasterScoresByWorker.Set(ctx,
					collections.Join(topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId),
					*topicIdActorIdScore.Score); err != nil {
					return errors.Wrap(err, "error setting latestForecasterScoresByWorker")
				}
			}
		}
	}
	//LatestReputerScoresByReputer []*TopicIdActorIdScore
	if len(data.LatestReputerScoresByReputer) != 0 {
		for _, topicIdActorIdScore := range data.LatestReputerScoresByReputer {
			if topicIdActorIdScore != nil {
				if err := k.latestReputerScoresByReputer.Set(ctx,
					collections.Join(topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId),
					*topicIdActorIdScore.Score); err != nil {
					return errors.Wrap(err, "error setting latestReputerScoresByReputer")
				}
			}
		}
	}
	//ReputerListeningCoefficient []*TopicIdActorIdListeningCoefficient
	if len(data.ReputerListeningCoefficient) != 0 {
		for _, topicIdActorIdListeningCoefficient := range data.ReputerListeningCoefficient {
			if topicIdActorIdListeningCoefficient != nil {
				if err := k.reputerListeningCoefficient.Set(ctx,
					collections.Join(topicIdActorIdListeningCoefficient.TopicId, topicIdActorIdListeningCoefficient.ActorId),
					*topicIdActorIdListeningCoefficient.ListeningCoefficient); err != nil {
					return errors.Wrap(err, "error setting reputerListeningCoefficient")
				}
			}
		}
	}
	//PreviousReputerRewardFraction []*TopicIdActorIdDec
	if len(data.PreviousReputerRewardFraction) != 0 {
		for _, topicIdActorIdDec := range data.PreviousReputerRewardFraction {
			if topicIdActorIdDec != nil {
				if err := k.previousReputerRewardFraction.Set(ctx,
					collections.Join(topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId),
					topicIdActorIdDec.Dec); err != nil {
					return errors.Wrap(err, "error setting previousReputerRewardFraction")
				}
			}
		}
	}
	//PreviousInferenceRewardFraction []*TopicIdActorIdDec
	if len(data.PreviousInferenceRewardFraction) != 0 {
		for _, topicIdActorIdDec := range data.PreviousInferenceRewardFraction {
			if topicIdActorIdDec != nil {
				if err := k.previousInferenceRewardFraction.Set(ctx,
					collections.Join(topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId),
					topicIdActorIdDec.Dec); err != nil {
					return errors.Wrap(err, "error setting previousInferenceRewardFraction")
				}
			}
		}
	}
	//PreviousForecastRewardFraction []*TopicIdActorIdDec
	if len(data.PreviousForecastRewardFraction) != 0 {
		for _, topicIdActorIdDec := range data.PreviousForecastRewardFraction {
			if topicIdActorIdDec != nil {
				if err := k.previousForecastRewardFraction.Set(ctx,
					collections.Join(topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId),
					topicIdActorIdDec.Dec); err != nil {
					return errors.Wrap(err, "error setting previousForecastRewardFraction")
				}
			}
		}
	}
	// TotalStake cosmossdk_io_math.Int
	if data.TotalStake.GT(cosmosMath.ZeroInt()) {
		if err := k.totalStake.Set(ctx, data.TotalStake); err != nil {
			return errors.Wrap(err, "error setting totalStake")
		}
	} else {
		if err := k.totalStake.Set(ctx, cosmosMath.ZeroInt()); err != nil {
			return errors.Wrap(err, "error setting totalStake to zero int")
		}
	}
	//TopicStake []*TopicIdAndInt
	if len(data.TopicStake) != 0 {
		for _, topicIdAndInt := range data.TopicStake {
			if topicIdAndInt != nil {
				if err := k.topicStake.Set(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
					return errors.Wrap(err, "error setting topicStake")
				}
			}
		}
	}
	//StakeReputerAuthority []*TopicIdActorIdInt
	if len(data.StakeReputerAuthority) != 0 {
		for _, topicIdActorIdInt := range data.StakeReputerAuthority {
			if topicIdActorIdInt != nil {
				if err := k.stakeReputerAuthority.Set(ctx,
					collections.Join(topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId),
					topicIdActorIdInt.Int); err != nil {
					return errors.Wrap(err, "error setting stakeReputerAuthority")
				}
			}
		}
	}
	//StakeSumFromDelegator []*TopicIdActorIdInt
	if len(data.StakeSumFromDelegator) != 0 {
		for _, topicIdActorIdInt := range data.StakeSumFromDelegator {
			if topicIdActorIdInt != nil {
				if err := k.stakeSumFromDelegator.Set(ctx,
					collections.Join(topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId),
					topicIdActorIdInt.Int); err != nil {
					return errors.Wrap(err, "error setting stakeSumFromDelegator")
				}
			}
		}
	}
	//DelegatedStakes []*TopicIdDelegatorReputerDelegatorInfo
	if len(data.DelegatedStakes) != 0 {
		for _, topicIdDelegatorReputerDelegatorInfo := range data.DelegatedStakes {
			if topicIdDelegatorReputerDelegatorInfo != nil {
				if err := k.delegatedStakes.Set(ctx,
					collections.Join3(
						topicIdDelegatorReputerDelegatorInfo.TopicId,
						topicIdDelegatorReputerDelegatorInfo.Delegator,
						topicIdDelegatorReputerDelegatorInfo.Reputer,
					),
					*topicIdDelegatorReputerDelegatorInfo.DelegatorInfo); err != nil {
					return errors.Wrap(err, "error setting delegatedStakes")
				}
			}
		}
	}
	//StakeFromDelegatorsUponReputer []*TopicIdActorIdInt
	if len(data.StakeFromDelegatorsUponReputer) != 0 {
		for _, topicIdActorIdInt := range data.StakeFromDelegatorsUponReputer {
			if topicIdActorIdInt != nil {
				if err := k.stakeFromDelegatorsUponReputer.Set(ctx,
					collections.Join(topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId),
					topicIdActorIdInt.Int); err != nil {
					return errors.Wrap(err, "error setting stakeFromDelegatorsUponReputer")
				}
			}
		}
	}
	//DelegateRewardPerShare []*TopicIdActorIdDec
	if len(data.DelegateRewardPerShare) != 0 {
		for _, topicIdActorIdDec := range data.DelegateRewardPerShare {
			if topicIdActorIdDec != nil {
				if err := k.delegateRewardPerShare.Set(ctx,
					collections.Join(topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId),
					topicIdActorIdDec.Dec); err != nil {
					return errors.Wrap(err, "error setting delegateRewardPerShare")
				}
			}
		}
	}
	//StakeRemovalsByBlock []*BlockHeightTopicIdReputerStakeRemovalInfo
	if len(data.StakeRemovalsByBlock) != 0 {
		for _, blockHeightTopicIdReputerStakeRemovalInfo := range data.StakeRemovalsByBlock {
			if blockHeightTopicIdReputerStakeRemovalInfo != nil {
				if err := k.stakeRemovalsByBlock.Set(ctx,
					collections.Join3(
						blockHeightTopicIdReputerStakeRemovalInfo.BlockHeight,
						blockHeightTopicIdReputerStakeRemovalInfo.TopicId,
						blockHeightTopicIdReputerStakeRemovalInfo.Reputer),
					*blockHeightTopicIdReputerStakeRemovalInfo.StakeRemovalInfo); err != nil {
					return errors.Wrap(err, "error setting stakeRemovalsByBlock")
				}
			}
		}
	}
	//StakeRemovalsByActor []*ActorIdTopicIdBlockHeight
	if len(data.StakeRemovalsByActor) != 0 {
		for _, actorIdTopicIdBlockHeight := range data.StakeRemovalsByActor {
			if actorIdTopicIdBlockHeight != nil {
				if err := k.stakeRemovalsByActor.Set(ctx,
					collections.Join3(
						actorIdTopicIdBlockHeight.ActorId,
						actorIdTopicIdBlockHeight.TopicId,
						actorIdTopicIdBlockHeight.BlockHeight)); err != nil {
					return errors.Wrap(err, "error setting stakeRemovalsByActor")
				}
			}
		}
	}
	//DelegateStakeRemovalsByBlock []*BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo
	if len(data.DelegateStakeRemovalsByBlock) != 0 {
		for _, blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo := range data.DelegateStakeRemovalsByBlock {
			if blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo != nil {
				if err := k.delegateStakeRemovalsByBlock.Set(ctx,
					Join4(
						blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.BlockHeight,
						blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.TopicId,
						blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.Delegator,
						blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.Reputer,
					),
					*blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.DelegateStakeRemovalInfo); err != nil {
					return errors.Wrap(err, "error setting delegateStakeRemovalsByBlock")
				}
			}
		}
	}
	//DelegateStakeRemovalsByActor []*DelegatorReputerTopicIdBlockHeight
	if len(data.DelegateStakeRemovalsByActor) != 0 {
		for _, delegatorReputerTopicIdBlockHeight := range data.DelegateStakeRemovalsByActor {
			if delegatorReputerTopicIdBlockHeight != nil {
				if err := k.delegateStakeRemovalsByActor.Set(ctx,
					Join4(
						delegatorReputerTopicIdBlockHeight.Delegator,
						delegatorReputerTopicIdBlockHeight.Reputer,
						delegatorReputerTopicIdBlockHeight.TopicId,
						delegatorReputerTopicIdBlockHeight.BlockHeight)); err != nil {
					return errors.Wrap(err, "error setting delegateStakeRemovalsByActor")
				}
			}
		}
	}
	//Inferences []*TopicIdActorIdInference
	if len(data.Inferences) != 0 {
		for _, topicIdActorIdInference := range data.Inferences {
			if topicIdActorIdInference != nil {
				if err := k.inferences.Set(ctx,
					collections.Join(
						topicIdActorIdInference.TopicId,
						topicIdActorIdInference.ActorId),
					*topicIdActorIdInference.Inference); err != nil {
					return errors.Wrap(err, "error setting inferences")
				}
			}
		}
	}

	// Forecasts []*TopicIdActorIdForecast
	if len(data.Forecasts) != 0 {
		for _, topicIdActorIdForecast := range data.Forecasts {
			if topicIdActorIdForecast != nil {
				if err := k.forecasts.Set(ctx,
					collections.Join(
						topicIdActorIdForecast.TopicId,
						topicIdActorIdForecast.ActorId),
					*topicIdActorIdForecast.Forecast); err != nil {
					return errors.Wrap(err, "error setting forecasts")
				}
			}
		}
	}

	// Workers []*LibP2PKeyAndOffchainNode
	if len(data.Workers) != 0 {
		for _, libP2PKeyAndOffchainNode := range data.Workers {
			if libP2PKeyAndOffchainNode != nil {
				if err := k.workers.Set(
					ctx,
					libP2PKeyAndOffchainNode.LibP2PKey,
					*libP2PKeyAndOffchainNode.OffchainNode); err != nil {
					return errors.Wrap(err, "error setting workers")
				}
			}
		}
	}

	// Reputers []*LibP2PKeyAndOffchainNode
	if len(data.Reputers) != 0 {
		for _, libP2PKeyAndOffchainNode := range data.Reputers {
			if libP2PKeyAndOffchainNode != nil {
				if err := k.reputers.Set(
					ctx,
					libP2PKeyAndOffchainNode.LibP2PKey,
					*libP2PKeyAndOffchainNode.OffchainNode); err != nil {
					return errors.Wrap(err, "error setting reputers")
				}
			}
		}

	}

	// TopicFeeRevenue []*TopicIdAndInt
	if len(data.TopicFeeRevenue) != 0 {
		for _, topicIdAndInt := range data.TopicFeeRevenue {
			if topicIdAndInt != nil {
				if err := k.topicFeeRevenue.Set(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
					return errors.Wrap(err, "error setting topicFeeRevenue")
				}
			}
		}
	}
	// PreviousTopicWeight []*TopicIdAndDec
	if len(data.PreviousTopicWeight) != 0 {
		for _, topicIdAndDec := range data.PreviousTopicWeight {
			if topicIdAndDec != nil {
				if err := k.previousTopicWeight.Set(
					ctx,
					topicIdAndDec.TopicId,
					topicIdAndDec.Dec); err != nil {
					return errors.Wrap(err, "error setting previousTopicWeight")
				}
			}
		}
	}

	//AllInferences []*TopicIdBlockHeightInferences
	if len(data.AllInferences) != 0 {
		for _, topicIdBlockHeightInferences := range data.AllInferences {
			if err := k.allInferences.Set(ctx,
				collections.Join(topicIdBlockHeightInferences.TopicId, topicIdBlockHeightInferences.BlockHeight),
				*topicIdBlockHeightInferences.Inferences); err != nil {
				return errors.Wrap(err, "error setting allInferences")
			}
		}
	}
	//AllForecasts []*TopicIdBlockHeightForecasts
	if len(data.AllForecasts) != 0 {
		for _, topicIdBlockHeightForecasts := range data.AllForecasts {
			if err := k.allForecasts.Set(ctx,
				collections.Join(topicIdBlockHeightForecasts.TopicId, topicIdBlockHeightForecasts.BlockHeight),
				*topicIdBlockHeightForecasts.Forecasts); err != nil {
				return errors.Wrap(err, "error setting allForecasts")
			}
		}
	}
	//AllLossBundles []*TopicIdBlockHeightReputerValueBundles
	if len(data.AllLossBundles) != 0 {
		for _, topicIdBlockHeightReputerValueBundles := range data.AllLossBundles {
			if err := k.allLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightReputerValueBundles.TopicId, topicIdBlockHeightReputerValueBundles.BlockHeight),
				*topicIdBlockHeightReputerValueBundles.ReputerValueBundles); err != nil {
				return errors.Wrap(err, "error setting allLossBundles")
			}
		}
	}
	//NetworkLossBundles []*TopicIdBlockHeightValueBundles
	if len(data.NetworkLossBundles) != 0 {
		for _, topicIdBlockHeightValueBundles := range data.NetworkLossBundles {
			if err := k.networkLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightValueBundles.TopicId, topicIdBlockHeightValueBundles.BlockHeight),
				*topicIdBlockHeightValueBundles.ValueBundle); err != nil {
				return errors.Wrap(err, "error setting networkLossBundles")
			}
		}
	}
	//PreviousPercentageRewardToStakedReputers github_com_allora_network_allora_chain_math.Dec
	if data.PreviousPercentageRewardToStakedReputers != alloraMath.ZeroDec() {
		if err := k.SetPreviousPercentageRewardToStakedReputers(ctx, data.PreviousPercentageRewardToStakedReputers); err != nil {
			return errors.Wrap(err, "error setting previousPercentageRewardToStakedReputers")
		}
	} else {
		// For mint module inflation rate calculation set the initial
		// "previous percentage of rewards that went to staked reputers" to 30%
		if err := k.SetPreviousPercentageRewardToStakedReputers(ctx, alloraMath.MustNewDecFromString("0.3")); err != nil {
			return errors.Wrap(err, "error setting previousPercentageRewardToStakedReputers to 0.3")
		}
	}
	//UnfulfilledWorkerNonces []*TopicIdAndNonces
	if len(data.UnfulfilledWorkerNonces) != 0 {
		for _, topicIdAndNonces := range data.UnfulfilledWorkerNonces {
			if err := k.unfulfilledWorkerNonces.Set(ctx, topicIdAndNonces.TopicId, *topicIdAndNonces.Nonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledWorkerNonces")
			}
		}
	}
	//UnfulfilledReputerNonces []*TopicIdAndReputerRequestNonces
	if len(data.UnfulfilledReputerNonces) != 0 {
		for _, topicIdAndReputerRequestNonces := range data.UnfulfilledReputerNonces {
			if err := k.unfulfilledReputerNonces.Set(ctx, topicIdAndReputerRequestNonces.TopicId, *topicIdAndReputerRequestNonces.ReputerRequestNonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledReputerNonces")
			}
		}
	}
	//LatestInfererNetworkRegrets []*TopicIdActorIdTimeStampedValue
	if len(data.LatestInfererNetworkRegrets) != 0 {
		for _, topicIdActorIdTimeStampedValue := range data.LatestInfererNetworkRegrets {
			if err := k.latestInfererNetworkRegrets.Set(ctx,
				collections.Join(topicIdActorIdTimeStampedValue.TopicId, topicIdActorIdTimeStampedValue.ActorId),
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestInfererNetworkRegrets")
			}
		}
	}
	//LatestForecasterNetworkRegrets []*TopicIdActorIdTimeStampedValue
	if len(data.LatestForecasterNetworkRegrets) != 0 {
		for _, topicIdActorIdTimeStampedValue := range data.LatestForecasterNetworkRegrets {
			if err := k.latestForecasterNetworkRegrets.Set(ctx,
				collections.Join(topicIdActorIdTimeStampedValue.TopicId, topicIdActorIdTimeStampedValue.ActorId),
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestForecasterNetworkRegrets")
			}
		}
	}
	//LatestOneInForecasterNetworkRegrets []*TopicIdActorIdActorIdTimeStampedValue
	if len(data.LatestOneInForecasterNetworkRegrets) != 0 {
		for _, topicIdActorIdActorIdTimeStampedValue := range data.LatestOneInForecasterNetworkRegrets {
			if err := k.latestOneInForecasterNetworkRegrets.Set(ctx,
				collections.Join3(
					topicIdActorIdActorIdTimeStampedValue.TopicId,
					topicIdActorIdActorIdTimeStampedValue.ActorId1,
					topicIdActorIdActorIdTimeStampedValue.ActorId2),
				*topicIdActorIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneInForecasterNetworkRegrets")
			}
		}
	}
	//CoreTeamAddresses []string
	if len(data.CoreTeamAddresses) != 0 {
		// make sure what we are storage isn't garbage
		for _, address := range data.CoreTeamAddresses {
			_, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				return errors.Wrap(err, "error converting core team address from bech32")
			}
		}
		if err := k.addCoreTeamToWhitelists(ctx, data.CoreTeamAddresses); err != nil {
			return errors.Wrap(err, "error adding core team addresses to whitelists")
		}
	}
	//TopicLastWorkerCommit   []*TopicIdTimestampedActorNonce
	if len(data.TopicLastWorkerCommit) != 0 {
		for _, topicIdTimestampedActorNonce := range data.TopicLastWorkerCommit {
			if err := k.topicLastWorkerCommit.Set(ctx,
				topicIdTimestampedActorNonce.TopicId,
				*topicIdTimestampedActorNonce.TimestampedActorNonce); err != nil {
				return errors.Wrap(err, "error setting topicLastWorkerCommit")
			}
		}
	}
	//TopicLastReputerCommit  []*TopicIdTimestampedActorNonce
	if len(data.TopicLastReputerCommit) != 0 {
		for _, topicIdTimestampedActorNonce := range data.TopicLastReputerCommit {
			if err := k.topicLastReputerCommit.Set(ctx,
				topicIdTimestampedActorNonce.TopicId,
				*topicIdTimestampedActorNonce.TimestampedActorNonce); err != nil {
				return errors.Wrap(err, "error setting topicLastReputerCommit")
			}
		}
	}
	//TopicLastWorkerPayload  []*TopicIdTimestampedActorNonce
	if len(data.TopicLastWorkerPayload) != 0 {
		for _, topicIdTimestampedActorNonce := range data.TopicLastWorkerPayload {
			if err := k.topicLastWorkerPayload.Set(ctx,
				topicIdTimestampedActorNonce.TopicId,
				*topicIdTimestampedActorNonce.TimestampedActorNonce,
			); err != nil {
				return errors.Wrap(err, "error setting topicLastWorkerPayload")
			}
		}
	}
	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get module params")
	}

	nextTopicId, err := k.nextTopicId.Peek(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get next topic ID")
	}

	topicsIter, err := k.topics.Iterate(ctx, nil)
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
	activeTopicsIter, err := k.activeTopics.Iterate(ctx, nil)
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

	churnableTopics := make([]uint64, 0)
	churnableTopicsIter, err := k.churnableTopics.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate churnable topics")
	}
	for ; churnableTopicsIter.Valid(); churnableTopicsIter.Next() {
		key, err := churnableTopicsIter.Key()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key: churnableTopicsIter")
		}
		churnableTopics = append(churnableTopics, key)
	}

	rewardableTopics := make([]uint64, 0)
	rewardableTopicsIter, err := k.rewardableTopics.Iterate(ctx, nil)
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
	topicWorkersIter, err := k.topicWorkers.Iterate(ctx, nil)
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
	topicReputersIter, err := k.topicReputers.Iterate(ctx, nil)
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
	topicRewardNonceIter, err := k.topicRewardNonce.Iterate(ctx, nil)
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

	infererScoresByBlock := make([]*types.TopicIdBlockHeightScores, 0)
	infererScoresByBlockIter, err := k.infererScoresByBlock.Iterate(ctx, nil)
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
	forecasterScoresByBlockIter, err := k.forecasterScoresByBlock.Iterate(ctx, nil)
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
	reputerScoresByBlockIter, err := k.reputerScoresByBlock.Iterate(ctx, nil)
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

	latestInfererScoresByWorker := make([]*types.TopicIdActorIdScore, 0)
	latestInfererScoresByWorkerIter, err := k.latestInfererScoresByWorker.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest inferer scores by worker")
	}
	for ; latestInfererScoresByWorkerIter.Valid(); latestInfererScoresByWorkerIter.Next() {
		keyValue, err := latestInfererScoresByWorkerIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestInfererScoresByWorkerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		latestInfererScoresByWorker = append(latestInfererScoresByWorker, &topicIdActorIdScore)
	}

	latestForecasterScoresByWorker := make([]*types.TopicIdActorIdScore, 0)
	latestForecasterScoresByWorkerIter, err := k.latestForecasterScoresByWorker.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest forecaster scores by worker")
	}
	for ; latestForecasterScoresByWorkerIter.Valid(); latestForecasterScoresByWorkerIter.Next() {
		keyValue, err := latestForecasterScoresByWorkerIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestForecasterScoresByWorkerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		latestForecasterScoresByWorker = append(latestForecasterScoresByWorker, &topicIdActorIdScore)
	}

	latestReputerScoresByReputer := make([]*types.TopicIdActorIdScore, 0)
	latestReputerScoresByReputerIter, err := k.latestReputerScoresByReputer.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate latest reputer scores by reputer")
	}
	for ; latestReputerScoresByReputerIter.Valid(); latestReputerScoresByReputerIter.Next() {
		keyValue, err := latestReputerScoresByReputerIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: latestReputerScoresByReputerIter")
		}
		value := keyValue.Value
		topicIdActorIdScore := types.TopicIdActorIdScore{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Score:   &value,
		}
		latestReputerScoresByReputer = append(latestReputerScoresByReputer, &topicIdActorIdScore)
	}

	reputerListeningCoefficient := make([]*types.TopicIdActorIdListeningCoefficient, 0)
	reputerListeningCoefficientIter, err := k.reputerListeningCoefficient.Iterate(ctx, nil)
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
	previousReputerRewardFractionIter, err := k.previousReputerRewardFraction.Iterate(ctx, nil)
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
	previousInferenceRewardFractionIter, err := k.previousInferenceRewardFraction.Iterate(ctx, nil)
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
	previousForecastRewardFractionIter, err := k.previousForecastRewardFraction.Iterate(ctx, nil)
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

	totalStake, err := k.totalStake.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total stake")
	}

	// Fill in the values from keeper.go

	// topicStake
	topicStake := make([]*types.TopicIdAndInt, 0)
	var i uint64
	for i = 1; i < nextTopicId; i++ {
		stake, err := k.topicStake.Get(ctx, i)
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
	stakeReputerAuthorityIter, err := k.stakeReputerAuthority.Iterate(ctx, nil)
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
	stakeSumFromDelegatorIter, err := k.stakeSumFromDelegator.Iterate(ctx, nil)
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
	delegatedStakesIter, err := k.delegatedStakes.Iterate(ctx, nil)
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
	stakeFromDelegatorsUponReputerIter, err := k.stakeFromDelegatorsUponReputer.Iterate(ctx, nil)
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
	delegateRewardPerShareIter, err := k.delegateRewardPerShare.Iterate(ctx, nil)
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
	stakeRemovalsByBlockIter, err := k.stakeRemovalsByBlock.Iterate(ctx, nil)
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
			StakeRemovalInfo: &value,
		}
		stakeRemovalsByBlock = append(stakeRemovalsByBlock, &blockHeightTopicIdReputerStakeRemovalInfo)
	}

	// stakeRemovalsByActor
	stakeRemovalsByActor := make([]*types.ActorIdTopicIdBlockHeight, 0)
	stakeRemovalsByActorIter, err := k.stakeRemovalsByActor.Iterate(ctx, nil)
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
	delegateStakeRemovalsByBlockIter, err := k.delegateStakeRemovalsByBlock.Iterate(ctx, nil)
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
			DelegateStakeRemovalInfo: &value,
		}
		delegateStakeRemovalsByBlock = append(delegateStakeRemovalsByBlock, &blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo)
	}

	// delegateStakeRemovalsByActor
	delegateStakeRemovalsByActor := make([]*types.DelegatorReputerTopicIdBlockHeight, 0)
	delegateStakeRemovalsByActorIter, err := k.delegateStakeRemovalsByActor.Iterate(ctx, nil)
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
	inferencesIter, err := k.inferences.Iterate(ctx, nil)
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
	forecastsIter, err := k.forecasts.Iterate(ctx, nil)
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
	workersIter, err := k.workers.Iterate(ctx, nil)
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
	reputersIter, err := k.reputers.Iterate(ctx, nil)
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
	topicFeeRevenueIter, err := k.topicFeeRevenue.Iterate(ctx, nil)
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
	previousTopicWeightIter, err := k.previousTopicWeight.Iterate(ctx, nil)
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
	allInferencesIter, err := k.allInferences.Iterate(ctx, nil)
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
	allForecastsIter, err := k.allForecasts.Iterate(ctx, nil)
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
	allLossBundlesIter, err := k.allLossBundles.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate all loss bundles")
	}
	for ; allLossBundlesIter.Valid(); allLossBundlesIter.Next() {
		keyValue, err := allLossBundlesIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: allLossBundlesIter")
		}
		value := keyValue.Value
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightReputerValueBundles{
			TopicId:             keyValue.Key.K1(),
			BlockHeight:         keyValue.Key.K2(),
			ReputerValueBundles: &value,
		}
		allLossBundles = append(allLossBundles, &topicIdBlockHeightValueBundles)
	}

	// networkLossBundles
	networkLossBundles := make([]*types.TopicIdBlockHeightValueBundles, 0)
	networkLossBundlesIter, err := k.networkLossBundles.Iterate(ctx, nil)
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
	previousPercentageRewardToStakedReputers, err := k.previousPercentageRewardToStakedReputers.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous percentage reward to staked reputers")
	}

	// unfulfilledWorkerNonces
	unfulfilledWorkerNonces := make([]*types.TopicIdAndNonces, 0)
	unfulfilledWorkerNoncesIter, err := k.unfulfilledWorkerNonces.Iterate(ctx, nil)
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
	unfulfilledReputerNoncesIter, err := k.unfulfilledReputerNonces.Iterate(ctx, nil)
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

	latestInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestInfererNetworkRegretsIter, err := k.latestInfererNetworkRegrets.Iterate(ctx, nil)
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

	latestForecasterNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestForecasterNetworkRegretsIter, err := k.latestForecasterNetworkRegrets.Iterate(ctx, nil)
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

	latestOneInForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneInForecasterNetworkRegretsIter, err := k.latestOneInForecasterNetworkRegrets.Iterate(ctx, nil)
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

	coreTeamAddresses := make([]string, 0)
	coreTeamAddressesIter, err := k.whitelistAdmins.Iterate(ctx, nil)
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
	topicLastWorkerCommitIter, err := k.topicLastWorkerCommit.Iterate(ctx, nil)
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
	topicLastReputerCommitIter, err := k.topicLastReputerCommit.Iterate(ctx, nil)
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

	topicLastWorkerPayload := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastWorkerPayloadIter, err := k.topicLastWorkerPayload.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic last worker payload")
	}
	for ; topicLastWorkerPayloadIter.Valid(); topicLastWorkerPayloadIter.Next() {
		keyValue, err := topicLastWorkerPayloadIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicLastWorkerPayloadIter")
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastWorkerPayload = append(topicLastWorkerPayload, &topicIdTimestampedActorNonce)
	}

	topicLastReputerPayload := make([]*types.TopicIdTimestampedActorNonce, 0)
	topicLastReputerPayloadIter, err := k.topicLastReputerPayload.Iterate(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate topic last reputer payload")
	}
	for ; topicLastReputerPayloadIter.Valid(); topicLastReputerPayloadIter.Next() {
		keyValue, err := topicLastReputerPayloadIter.KeyValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get key value: topicLastReputerPayloadIter")
		}
		topicIdTimestampedActorNonce := types.TopicIdTimestampedActorNonce{
			TopicId:               keyValue.Key,
			TimestampedActorNonce: &keyValue.Value,
		}
		topicLastReputerPayload = append(topicLastReputerPayload, &topicIdTimestampedActorNonce)
	}

	return &types.GenesisState{
		Params:                                   moduleParams,
		NextTopicId:                              nextTopicId,
		Topics:                                   topics,
		ActiveTopics:                             activeTopics,
		ChurnableTopics:                          churnableTopics,
		RewardableTopics:                         rewardableTopics,
		TopicWorkers:                             topicWorkers,
		TopicReputers:                            topicReputers,
		TopicRewardNonce:                         topicRewardNonce,
		InfererScoresByBlock:                     infererScoresByBlock,
		ForecasterScoresByBlock:                  forecasterScoresByBlock,
		ReputerScoresByBlock:                     reputerScoresByBlock,
		LatestInfererScoresByWorker:              latestInfererScoresByWorker,
		LatestForecasterScoresByWorker:           latestForecasterScoresByWorker,
		LatestReputerScoresByReputer:             latestReputerScoresByReputer,
		ReputerListeningCoefficient:              reputerListeningCoefficient,
		PreviousReputerRewardFraction:            previousReputerRewardFraction,
		PreviousInferenceRewardFraction:          previousInferenceRewardFraction,
		PreviousForecastRewardFraction:           previousForecastRewardFraction,
		TotalStake:                               totalStake,
		TopicStake:                               topicStake,
		StakeReputerAuthority:                    stakeReputerAuthority,
		StakeSumFromDelegator:                    stakeSumFromDelegator,
		DelegatedStakes:                          delegatedStakes,
		StakeFromDelegatorsUponReputer:           stakeFromDelegatorsUponReputer,
		DelegateRewardPerShare:                   delegateRewardPerShare,
		StakeRemovalsByBlock:                     stakeRemovalsByBlock,
		StakeRemovalsByActor:                     stakeRemovalsByActor,
		DelegateStakeRemovalsByBlock:             delegateStakeRemovalsByBlock,
		DelegateStakeRemovalsByActor:             delegateStakeRemovalsByActor,
		Inferences:                               inferences,
		Forecasts:                                forecasts,
		Workers:                                  workers,
		Reputers:                                 reputers,
		TopicFeeRevenue:                          topicFeeRevenue,
		PreviousTopicWeight:                      previousTopicWeight,
		AllInferences:                            allInferences,
		AllForecasts:                             allForecasts,
		AllLossBundles:                           allLossBundles,
		NetworkLossBundles:                       networkLossBundles,
		PreviousPercentageRewardToStakedReputers: previousPercentageRewardToStakedReputers,
		UnfulfilledWorkerNonces:                  unfulfilledWorkerNonces,
		UnfulfilledReputerNonces:                 unfulfilledReputerNonces,
		LatestInfererNetworkRegrets:              latestInfererNetworkRegrets,
		LatestForecasterNetworkRegrets:           latestForecasterNetworkRegrets,
		LatestOneInForecasterNetworkRegrets:      latestOneInForecasterNetworkRegrets,
		CoreTeamAddresses:                        coreTeamAddresses,
		TopicLastWorkerCommit:                    topicLastWorkerCommit,
		TopicLastReputerCommit:                   topicLastReputerCommit,
		TopicLastWorkerPayload:                   topicLastWorkerPayload,
		TopicLastReputerPayload:                  topicLastReputerPayload,
	}, nil
}

func (k *Keeper) addCoreTeamToWhitelists(ctx context.Context, coreTeamAddresses []string) error {
	for _, addr := range coreTeamAddresses {
		k.AddWhitelistAdmin(ctx, addr)
	}
	return nil
}
