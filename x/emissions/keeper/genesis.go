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
	for _, topic := range data.Topics {
		if topic != nil {
			if err := k.SetTopic(ctx, topic.TopicId, *topic.Topic); err != nil {
				return errors.Wrap(err, "error setting topic")
			}
		}
	}
	//ActiveTopics []uint64
	for _, topicId := range data.ActiveTopics {
		if err := types.ValidateTopicId(topicId); err != nil {
			return errors.Wrapf(err, "error setting activeTopics %v", data.ActiveTopics)
		}
		if err := k.activeTopics.Set(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting activeTopics")
		}
	}
	//RewardableTopics []uint64
	for _, topicId := range data.RewardableTopics {
		if err := k.rewardableTopics.Set(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting rewardableTopics")
		}
	}
	//TopicWorkers []*TopicAndActorId
	for _, topicAndActorId := range data.TopicWorkers {
		if topicAndActorId != nil {
			if err := types.ValidateTopicId(topicAndActorId.TopicId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
			if err := types.ValidateBech32(topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
			if err := k.topicWorkers.Set(ctx, collections.Join(topicAndActorId.TopicId, topicAndActorId.ActorId)); err != nil {
				return errors.Wrap(err, "error setting topicWorkers")
			}
		}
	}
	//TopicReputers []*TopicAndActorId
	for _, topicAndActorId := range data.TopicReputers {
		if topicAndActorId != nil {
			if err := types.ValidateTopicId(topicAndActorId.TopicId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
			if err := types.ValidateBech32(topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
			if err := k.topicReputers.Set(ctx, collections.Join(topicAndActorId.TopicId, topicAndActorId.ActorId)); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
		}
	}
	//TopicRewardNonce []*TopicIdAndBlockHeight
	for _, topicIdAndBlockHeight := range data.TopicRewardNonce {
		if topicIdAndBlockHeight != nil {
			if err := k.SetTopicRewardNonce(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting topicRewardNonce")
			}
		}
	}

	//InfererScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.InfererScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := types.ValidateTopicId(topicIdBlockHeightScores.TopicId); err != nil {
				return errors.Wrap(err, "error setting infererScoresByBlock")
			}
			if err := types.ValidateBlockHeight(topicIdBlockHeightScores.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting infererScoresByBlock")
			}
			if err := topicIdBlockHeightScores.Scores.Validate(); err != nil {
				return errors.Wrap(err, "error setting infererScoresByBlock")
			}
			if err := k.infererScoresByBlock.Set(ctx,
				collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting infererScoresByBlock")
			}
		}
	}
	//ForecasterScoresByBlock []*TopicIdBlockHeightScores
	for _, topicIdBlockHeightScores := range data.ForecasterScoresByBlock {
		if topicIdBlockHeightScores != nil {
			if err := types.ValidateTopicId(topicIdBlockHeightScores.TopicId); err != nil {
				return errors.Wrap(err, "error setting forecasterScoresByBlock")
			}
			if err := types.ValidateBlockHeight(topicIdBlockHeightScores.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting forecasterScoresByBlock")
			}
			if err := topicIdBlockHeightScores.Scores.Validate(); err != nil {
				return errors.Wrap(err, "error setting forecasterScoresByBlock")
			}
			if err := k.forecasterScoresByBlock.Set(
				ctx,
				collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting forecasterScoresByBlock")
			}
		}
	}

	//ReputerScoresByBlock []*TopicIdBlockHeightScores
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
			if err := k.reputerScoresByBlock.Set(
				ctx,
				collections.Join(topicIdBlockHeightScores.TopicId, topicIdBlockHeightScores.BlockHeight),
				*topicIdBlockHeightScores.Scores); err != nil {
				return errors.Wrap(err, "error setting reputerScoresByBlock")
			}
		}
	}

	//LatestInfererScoresByWorker []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.InfererScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.SetInfererScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestInfererScoresByWorker")
			}
		}
	}
	//LatestForecasterScoresByWorker []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.ForecasterScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.SetForecasterScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestForecasterScoresByWorker")
			}
		}
	}
	//LatestReputerScoresByReputer []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.ReputerScoreEmas {
		if topicIdActorIdScore != nil {
			if err := k.SetReputerScoreEma(ctx,
				topicIdActorIdScore.TopicId, topicIdActorIdScore.ActorId,
				*topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting latestReputerScoresByReputer")
			}
		}
	}
	//ReputerListeningCoefficient []*TopicIdActorIdListeningCoefficient
	for _, topicIdActorIdListeningCoefficient := range data.ReputerListeningCoefficient {
		if topicIdActorIdListeningCoefficient != nil {
			if err := k.SetListeningCoefficient(ctx,
				topicIdActorIdListeningCoefficient.TopicId, topicIdActorIdListeningCoefficient.ActorId,
				*topicIdActorIdListeningCoefficient.ListeningCoefficient); err != nil {
				return errors.Wrap(err, "error setting reputerListeningCoefficient")
			}
		}
	}
	//PreviousReputerRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousReputerRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.SetPreviousReputerRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousReputerRewardFraction")
			}
		}
	}
	//PreviousInferenceRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousInferenceRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.SetPreviousInferenceRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousInferenceRewardFraction")
			}
		}
	}
	//PreviousForecastRewardFraction []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.PreviousForecastRewardFraction {
		if topicIdActorIdDec != nil {
			if err := k.SetPreviousForecastRewardFraction(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousForecastRewardFraction")
			}
		}
	}
	// TotalStake cosmossdk_io_math.Int
	if data.TotalStake.GT(cosmosMath.ZeroInt()) {
		if err := k.SetTotalStake(ctx, data.TotalStake); err != nil {
			return errors.Wrap(err, "error setting totalStake")
		}
	} else {
		if err := k.SetTotalStake(ctx, cosmosMath.ZeroInt()); err != nil {
			return errors.Wrap(err, "error setting totalStake to zero int")
		}
	}
	//TopicStake []*TopicIdAndInt
	for _, topicIdAndInt := range data.TopicStake {
		if topicIdAndInt != nil {
			if err := k.SetTopicStake(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "error setting topicStake")
			}
		}
	}
	//StakeReputerAuthority []*TopicIdActorIdInt
	for _, topicIdActorIdInt := range data.StakeReputerAuthority {
		if topicIdActorIdInt != nil {
			if err := k.SetStakeReputerAuthority(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeReputerAuthority")
			}
		}
	}
	//StakeSumFromDelegator []*TopicIdActorIdInt
	for _, topicIdActorIdInt := range data.StakeSumFromDelegator {
		if topicIdActorIdInt != nil {
			if err := k.SetStakeFromDelegator(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeSumFromDelegator")
			}
		}
	}
	//DelegatedStakes []*TopicIdDelegatorReputerDelegatorInfo
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
	//StakeFromDelegatorsUponReputer []*TopicIdActorIdInt
	for _, topicIdActorIdInt := range data.StakeFromDelegatorsUponReputer {
		if topicIdActorIdInt != nil {
			if err := k.SetDelegateStakeUponReputer(ctx,
				topicIdActorIdInt.TopicId, topicIdActorIdInt.ActorId,
				topicIdActorIdInt.Int); err != nil {
				return errors.Wrap(err, "error setting stakeFromDelegatorsUponReputer")
			}
		}
	}
	//DelegateRewardPerShare []*TopicIdActorIdDec
	for _, topicIdActorIdDec := range data.DelegateRewardPerShare {
		if topicIdActorIdDec != nil {
			if err := k.SetDelegateRewardPerShare(ctx,
				topicIdActorIdDec.TopicId, topicIdActorIdDec.ActorId,
				topicIdActorIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting delegateRewardPerShare")
			}
		}
	}
	//StakeRemovalsByBlock []*BlockHeightTopicIdReputerStakeRemovalInfo
	//StakeRemovalsByActor []*ActorIdTopicIdBlockHeight
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
	//DelegateStakeRemovalsByBlock []*BlockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo
	//DelegateStakeRemovalsByActor []*DelegatorReputerTopicIdBlockHeight
	for _, blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo := range data.DelegateStakeRemovalsByBlock {
		if blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo != nil {
			if err := k.SetDelegateStakeRemoval(ctx,
				*blockHeightTopicIdDelegatorReputerDelegateStakeRemovalInfo.DelegateStakeRemovalInfo); err != nil {
				return errors.Wrap(err, "error setting delegateStakeRemovalsByBlock")
			}
		}
	}
	//Inferences []*TopicIdActorIdInference
	for _, topicIdActorIdInference := range data.Inferences {
		if topicIdActorIdInference != nil {
			if err := topicIdActorIdInference.Inference.Validate(); err != nil {
				return errors.Wrap(err, "inference in list is invalid")
			}
			if err := k.inferences.Set(ctx,
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
			if err := k.forecasts.Set(ctx,
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
			if err := k.workers.Set(
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
			if err := k.reputers.Set(
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
			if err := k.topicFeeRevenue.Set(ctx, topicIdAndInt.TopicId, topicIdAndInt.Int); err != nil {
				return errors.Wrap(err, "error setting topicFeeRevenue")
			}
		}
	}

	// PreviousTopicWeight []*TopicIdAndDec
	for _, topicIdAndDec := range data.PreviousTopicWeight {
		if topicIdAndDec != nil {
			if err := k.SetPreviousTopicWeight(
				ctx,
				topicIdAndDec.TopicId,
				topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicWeight")
			}
		}
	}

	//AllInferences []*TopicIdBlockHeightInferences
	for _, topicIdBlockHeightInferences := range data.AllInferences {
		if topicIdBlockHeightInferences != nil {
			for _, inference := range topicIdBlockHeightInferences.Inferences.Inferences {
				if inference != nil {
					if err := inference.Validate(); err != nil {
						return errors.Wrap(err, "inference validation failed")
					}
				}
			}
			if err := k.allInferences.Set(ctx,
				collections.Join(topicIdBlockHeightInferences.TopicId, topicIdBlockHeightInferences.BlockHeight),
				*topicIdBlockHeightInferences.Inferences); err != nil {
				return errors.Wrap(err, "error setting allInferences")
			}
		}
	}
	//AllForecasts []*TopicIdBlockHeightForecasts
	for _, topicIdBlockHeightForecasts := range data.AllForecasts {
		if topicIdBlockHeightForecasts != nil {
			for _, forecast := range topicIdBlockHeightForecasts.Forecasts.Forecasts {
				if forecast != nil {
					if err := forecast.Validate(); err != nil {
						return errors.Wrap(err, "forecast validation failed")
					}
				}
			}
			if err := k.allForecasts.Set(ctx,
				collections.Join(topicIdBlockHeightForecasts.TopicId, topicIdBlockHeightForecasts.BlockHeight),
				*topicIdBlockHeightForecasts.Forecasts); err != nil {
				return errors.Wrap(err, "error setting allForecasts")
			}
		}
	}

	//AllLossBundles []*TopicIdBlockHeightReputerValueBundles
	for _, topicIdBlockHeightReputerValueBundles := range data.AllLossBundles {
		if topicIdBlockHeightReputerValueBundles != nil {
			if err := topicIdBlockHeightReputerValueBundles.ReputerValueBundles.Validate(); err != nil {
				return errors.Wrap(err, "reputer value bundles validation failed")
			}
			if err := k.allLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightReputerValueBundles.TopicId, topicIdBlockHeightReputerValueBundles.BlockHeight),
				*topicIdBlockHeightReputerValueBundles.ReputerValueBundles); err != nil {
				return errors.Wrap(err, "error setting allLossBundles")
			}
		}
	}

	//NetworkLossBundles []*TopicIdBlockHeightValueBundles
	for _, topicIdBlockHeightValueBundles := range data.NetworkLossBundles {
		if topicIdBlockHeightValueBundles != nil {
			if err := topicIdBlockHeightValueBundles.ValueBundle.Validate(); err != nil {
				return errors.Wrap(err, "value bundle validation failed")
			}
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
	//openWorkerWindows []*BlockHeightAndListOfTopicIds
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
			if err := k.openWorkerWindows.Set(
				ctx,
				blockHeightAndListOfTopicIds.BlockHeight,
				topicIds,
			); err != nil {
				return errors.Wrap(err, "error setting openWorkerWindows")
			}
		}
	}

	//UnfulfilledWorkerNonces []*TopicIdAndNonces

	for _, topicIdAndNonces := range data.UnfulfilledWorkerNonces {
		if topicIdAndNonces != nil {
			if err := topicIdAndNonces.Nonces.Validate(); err != nil {
				return errors.Wrap(err, "error validating unfulfilled worker nonces")
			}
			if err := k.unfulfilledWorkerNonces.Set(ctx, topicIdAndNonces.TopicId, *topicIdAndNonces.Nonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledWorkerNonces")
			}
		}
	}
	//UnfulfilledReputerNonces []*TopicIdAndReputerRequestNonces

	for _, topicIdAndReputerRequestNonces := range data.UnfulfilledReputerNonces {
		if topicIdAndReputerRequestNonces != nil {
			if err := topicIdAndReputerRequestNonces.ReputerRequestNonces.Validate(); err != nil {
				return errors.Wrap(err, "error validating unfulfilled reputer nonces")
			}
			if err := k.unfulfilledReputerNonces.Set(ctx, topicIdAndReputerRequestNonces.TopicId, *topicIdAndReputerRequestNonces.ReputerRequestNonces); err != nil {
				return errors.Wrap(err, "error setting unfulfilledReputerNonces")
			}
		}
	}

	//lastDripBlock []*TopicIdAndBlockHeight
	for _, topicIdAndBlockHeight := range data.LastDripBlock {
		if topicIdAndBlockHeight != nil {
			if err := k.SetLastDripBlock(ctx, topicIdAndBlockHeight.TopicId, topicIdAndBlockHeight.BlockHeight); err != nil {
				return errors.Wrap(err, "error setting lastDripBlock")
			}
		}
	}

	//LatestInfererNetworkRegrets []*TopicIdActorIdTimeStampedValue
	for _, topicIdActorIdTimeStampedValue := range data.LatestInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetInfererNetworkRegret(ctx,
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
			if err := k.SetNaiveInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestNaiveInfererNetworkRegrets")
			}
		}
	}
	//LatestForecasterNetworkRegrets []*TopicIdActorIdTimeStampedValue
	for _, topicIdActorIdTimeStampedValue := range data.LatestForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetForecasterNetworkRegret(ctx,
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
			if err := k.SetOneOutInfererInfererNetworkRegret(ctx,
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
			if err := k.latestOneOutInfererForecasterNetworkRegrets.Set(ctx,
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
			if err := k.SetOneOutForecasterInfererNetworkRegret(ctx,
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
			if err := k.SetOneOutForecasterForecasterNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutForecasterForecasterNetworkRegrets")
			}
		}
	}
	//LatestOneInForecasterNetworkRegrets []*TopicIdActorIdActorIdTimeStampedValue
	for _, topicIdActorIdActorIdTimeStampedValue := range data.LatestOneInForecasterNetworkRegrets {
		if topicIdActorIdActorIdTimeStampedValue != nil {
			if err := k.SetOneInForecasterNetworkRegret(ctx,
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
			if err := k.SetPreviousForecasterScoreRatio(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
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
			err = k.AddWhitelistAdmin(ctx, address)
			if err != nil {
				return errors.Wrap(err, "error adding core team addresses to whitelists")
			}
		}
	}
	//TopicLastWorkerCommit   []*TopicIdTimestampedActorNonce
	for _, topicIdTimestampedActorNonce := range data.TopicLastWorkerCommit {
		if topicIdTimestampedActorNonce != nil {
			if err := k.SetWorkerTopicLastCommit(ctx,
				topicIdTimestampedActorNonce.TopicId,
				topicIdTimestampedActorNonce.TimestampedActorNonce.BlockHeight,
				topicIdTimestampedActorNonce.TimestampedActorNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastWorkerCommit")
			}
		}
	}
	//TopicLastReputerCommit  []*TopicIdTimestampedActorNonce
	for _, topicIdTimestampedActorNonce := range data.TopicLastReputerCommit {
		if topicIdTimestampedActorNonce != nil {
			if err := k.SetReputerTopicLastCommit(ctx,
				topicIdTimestampedActorNonce.TopicId,
				topicIdTimestampedActorNonce.TimestampedActorNonce.BlockHeight,
				topicIdTimestampedActorNonce.TimestampedActorNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastReputerCommit")
			}
		}
	}

	//TopicToNextPossibleChurningBlock []*topicBlock
	for _, topicBlock := range data.TopicToNextPossibleChurningBlock {
		if topicBlock != nil {
			if err := k.SetTopicToNextPossibleChurningBlock(ctx,
				topicBlock.TopicId,
				topicBlock.BlockHeight); err != nil {
				return errors.Wrapf(err, "error setting topicToNextPossibleChurningBlock %v", topicBlock)
			}
		}
	}

	//BlockToActiveTopics []*blockToActiveTopics
	for _, blockToActiveTopics := range data.BlockToActiveTopics {
		if blockToActiveTopics != nil {
			if err := k.blockToActiveTopics.Set(ctx,
				blockToActiveTopics.BlockHeight,
				*blockToActiveTopics.TopicIds); err != nil {
				return errors.Wrap(err, "error setting blockToActiveTopics")
			}
		}
	}

	//BlockToLowestActiveTopicWeight []*blockToLowestActiveTopicWeight
	for _, lowestActiveTopicWeight := range data.BlockToLowestActiveTopicWeight {
		if lowestActiveTopicWeight != nil {
			if err := k.blockToLowestActiveTopicWeight.Set(ctx,
				lowestActiveTopicWeight.BlockHeight,
				*lowestActiveTopicWeight.TopicWeight); err != nil {
				return errors.Wrap(err, "error setting blockToLowestActiveTopicWeight")
			}
		}
	}

	// PreviousTopicQuantileInfererScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileInfererScoreEma {
		if topicIdDec != nil {
			if err := k.SetPreviousTopicQuantileInfererScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileInfererScoreEma")
			}
		}
	}

	// PreviousTopicQuantileForecasterScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileForecasterScoreEma {
		if topicIdDec != nil {
			if err := k.SetPreviousTopicQuantileForecasterScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileForecasterScoreEma")
			}
		}
	}

	// PreviousTopicQuantileReputerScoreEma
	for _, topicIdDec := range data.PreviousTopicQuantileReputerScoreEma {
		if topicIdDec != nil {
			if err := k.SetPreviousTopicQuantileReputerScoreEma(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting previousTopicQuantileReputerScoreEma")
			}
		}
	}

	//InitialInfererEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialInfererEmaScore {
		if topicIdAndDec != nil {
			if err := k.initialInfererEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialInfererEmaScore")
			}
		}
	}

	//InitialForecasterEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialForecasterEmaScore {
		if topicIdAndDec != nil {
			if err := k.initialForecasterEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialForecasterEmaScore")
			}
		}
	}

	//InitialReputerEmaScore []*TopicIdAndDec
	for _, topicIdAndDec := range data.InitialReputerEmaScore {
		if topicIdAndDec != nil {
			if err := k.initialReputerEmaScore.Set(ctx, topicIdAndDec.TopicId, topicIdAndDec.Dec); err != nil {
				return errors.Wrap(err, "error setting initialReputerEmaScore")
			}
		}
	}

	// ActiveInferers []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveInferers {
		if topicAndActorId != nil {
			if err := k.AddActiveInferer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeInferers")
			}
		}
	}

	// ActiveForecasters []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveForecasters {
		if topicAndActorId != nil {
			if err := k.AddActiveForecaster(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeForecasters")
			}
		}
	}

	// LowestInfererScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestInfererScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.SetLowestInfererScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestInfererScoreEma")
			}
		}
	}

	// LowestForecasterScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestForecasterScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.SetLowestForecasterScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestForecasterScoreEma")
			}
		}
	}

	// ActiveReputers []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveReputers {
		if topicAndActorId != nil {
			if err := k.AddActiveReputer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeReputers")
			}
		}
	}

	// LowestReputerScoreEmas []*TopicIdActorIdScore
	for _, topicIdActorIdScore := range data.LowestReputerScoreEma {
		if topicIdActorIdScore != nil {
			if err := k.SetLowestReputerScoreEma(ctx, topicIdActorIdScore.TopicId, *topicIdActorIdScore.Score); err != nil {
				return errors.Wrap(err, "error setting lowestReputerScoreEma")
			}
		}
	}

	// LossBundles
	for _, bundle := range data.LossBundles {
		if bundle != nil {
			key := collections.Join(bundle.TopicId, bundle.Reputer)
			if err := k.lossBundles.Set(ctx, key, *bundle.ReputerValueBundle); err != nil {
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
		if err := k.SetTotalSumPreviousTopicWeights(ctx, data.TotalSumPreviousTopicWeights); err != nil {
			return errors.Wrap(err, "error setting TotalSumPreviousTopicWeights")
		}
	} else {
		if err := k.SetTotalSumPreviousTopicWeights(ctx, alloraMath.ZeroDec()); err != nil {
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
		if err := k.AddToGlobalWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalWhitelist")
		}
	}

	// globalWorkerWhitelist
	for _, address := range data.GlobalWorkerWhitelist {
		if err := k.AddToGlobalWorkerWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalWorkerWhitelist")
		}
	}

	// globalReputerWhitelist
	for _, address := range data.GlobalReputerWhitelist {
		if err := k.AddToGlobalReputerWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalReputerWhitelist")
		}
	}

	// globalAdminWhitelist
	for _, address := range data.GlobalAdminWhitelist {
		if err := k.AddToGlobalAdminWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalAdminWhitelist")
		}
	}

	// topicCreatorWhitelist
	for _, address := range data.TopicCreatorWhitelist {
		if err := k.AddToTopicCreatorWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting topicCreatorWhitelist")
		}
	}

	// topicWorkerWhitelist
	for _, topicAndActorId := range data.TopicWorkerWhitelist {
		if topicAndActorId != nil {
			if err := k.AddToTopicWorkerWhitelist(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkerWhitelist")
			}
		}
	}

	// topicReputerWhitelist
	for _, topicAndActorId := range data.TopicReputerWhitelist {
		if topicAndActorId != nil {
			if err := k.AddToTopicReputerWhitelist(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputerWhitelist")
			}
		}
	}

	// topicWorkerWhitelistEnabled
	for _, topicId := range data.TopicWorkerWhitelistEnabled {
		if err := k.EnableTopicWorkerWhitelist(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting topicWorkerWhitelistEnabled")
		}
	}

	// topicReputerWhitelistEnabled
	for _, topicId := range data.TopicReputerWhitelistEnabled {
		if err := k.EnableTopicReputerWhitelist(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting topicReputerWhitelistEnabled")
		}
	}

	// LastMedianInferences
	for _, topicIdAndDec := range data.LastMedianInferences {
		if topicIdAndDec != nil {
			if err := k.SetLastMedianInferences(
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
			if err := k.SetMadInferences(ctx, topicIdDec.TopicId, topicIdDec.Dec); err != nil {
				return errors.Wrap(err, "error setting madInferences")
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

	topicNextChurningBlock, err := k.topicToNextPossibleChurningBlock.Iterate(ctx, nil)
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

	blockActiveTopics, err := k.blockToActiveTopics.Iterate(ctx, nil)
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

	lowestActiveTopic, err := k.blockToLowestActiveTopicWeight.Iterate(ctx, nil)
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

	var initialInfererEmaScore []*types.TopicIdAndDec
	if err := k.initialInfererEmaScore.Walk(
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
	if err := k.initialForecasterEmaScore.Walk(
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
	if err := k.initialReputerEmaScore.Walk(
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

	innfererScoreEmas := make([]*types.TopicIdActorIdScore, 0)
	infererScoreEmasIter, err := k.infererScoreEmas.Iterate(ctx, nil)
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
	forecasterScoreEmaIter, err := k.forecasterScoreEmas.Iterate(ctx, nil)
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
	reputerScoreEmasIter, err := k.reputerScoreEmas.Iterate(ctx, nil)
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
			Reputer:          value.Reputer,
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
			Reputer:                  value.Reputer,
			Delegator:                value.Delegator,
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

	// openWorkerWindows
	openWorkerWindows := make([]*types.BlockHeightAndTopicIds, 0)
	openWorkerWindowsIter, err := k.openWorkerWindows.Iterate(ctx, nil)
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

	// lastDripBlock
	lastDripBlock := make([]*types.TopicIdAndBlockHeight, 0)
	lastDripBlockIter, err := k.lastDripBlock.Iterate(ctx, nil)
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

	latestNaiveInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestNaiveInfererNetworkRegretsIter, err := k.latestNaiveInfererNetworkRegrets.Iterate(ctx, nil)
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

	latestOneOutInfererInfererNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutInfererInfererNetworkRegretsIter, err := k.latestOneOutInfererInfererNetworkRegrets.Iterate(ctx, nil)
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
	latestOneOutInfererForecasterNetworkRegretsIter, err := k.latestOneOutInfererForecasterNetworkRegrets.Iterate(ctx, nil)
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
	latestOneOutForecasterInfererNetworkRegretsIter, err := k.latestOneOutForecasterInfererNetworkRegrets.Iterate(ctx, nil)
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
	latestOneOutForecasterForecasterNetworkRegretsIter, err := k.latestOneOutForecasterForecasterNetworkRegrets.Iterate(ctx, nil)
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
		previousForecasterScoreRatio = append(previousForecasterScoreRatio, &types.TopicIdAndDec{
			TopicId: keyValue.Key,
			Dec:     keyValue.Value,
		})
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

	previousTopicQuantileInfererScoreEma := make([]*types.TopicIdAndDec, 0)
	previousTopicQuantileInfererScoreEmaIter, err := k.previousTopicQuantileInfererScoreEma.Iterate(ctx, nil)
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
	previousTopicQuantileForecasterScoreEmaIter, err := k.previousTopicQuantileForecasterScoreEma.Iterate(ctx, nil)
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
	previousTopicQuantileReputerScoreEmaIter, err := k.previousTopicQuantileReputerScoreEma.Iterate(ctx, nil)
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
	activeInferersIter, err := k.activeInferers.Iterate(ctx, nil)
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
	activeForecasterIter, err := k.activeForecasters.Iterate(ctx, nil)
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
	lowestInfererScoreEmaIter, err := k.lowestInfererScoreEma.Iterate(ctx, nil)
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
	lowestForecasterScoreEmaIter, err := k.lowestForecasterScoreEma.Iterate(ctx, nil)
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
	activeReputersIter, err := k.activeReputers.Iterate(ctx, nil)
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
	lowestReputerScoreEmaIter, err := k.lowestReputerScoreEma.Iterate(ctx, nil)
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
	lossBundlesIter, err := k.lossBundles.Iterate(ctx, nil)
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
			ReputerValueBundle: &keyValue.Value,
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

	totalSumPreviousTopicWeights, err := k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total sum previous topic weights")
	}

	whitelistAdmins := make([]string, 0)
	whitelistAdminsIter, err := k.whitelistAdmins.Iterate(ctx, nil)
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
	globalWhitelistIter, err := k.globalWhitelist.Iterate(ctx, nil)
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
	globalWorkerWhitelistIter, err := k.globalWorkerWhitelist.Iterate(ctx, nil)
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
	globalReputerWhitelistIter, err := k.globalReputerWhitelist.Iterate(ctx, nil)
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
	globalAdminWhitelistIter, err := k.globalAdminWhitelist.Iterate(ctx, nil)
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
	topicCreatorWhitelistIter, err := k.topicCreatorWhitelist.Iterate(ctx, nil)
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
	topicWorkerWhitelistIter, err := k.topicWorkerWhitelist.Iterate(ctx, nil)
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
	topicReputerWhitelistIter, err := k.topicReputerWhitelist.Iterate(ctx, nil)
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
	topicWorkerWhitelistEnabledIter, err := k.topicWorkerWhitelistEnabled.Iterate(ctx, nil)
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
	topicReputerWhitelistEnabledIter, err := k.topicReputerWhitelistEnabled.Iterate(ctx, nil)
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
	lastMedianInferencesIter, err := k.lastMedianInferences.Iterate(ctx, nil)
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
	madInferencesIter, err := k.madInferences.Iterate(ctx, nil)
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
	}, nil
}
